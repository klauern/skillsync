// Package hooks provides opt-in synchronization of lifecycle hook declarations.
package hooks

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/trust"
)

// Discoverer reads hooks for one harness without executing them.
type Discoverer interface {
	DiscoverHooks(context.Context, model.Platform) ([]model.HookConfig, error)
}

// Writer writes one fully validated batch.
type Writer interface {
	WriteHooks(context.Context, []model.HookConfig) error
}

// Mapping defines one exact event mapping between two harnesses.
type Mapping struct {
	Key, SourceEvent, TargetEvent  string
	SourcePlatform, TargetPlatform model.Platform
}

// Registry stores explicit cross-harness mappings.
type Registry struct{ mappings []Mapping }

// Register adds one exact mapping.
func (r *Registry) Register(m Mapping) error {
	if strings.TrimSpace(m.Key) == "" || strings.TrimSpace(m.SourceEvent) == "" || strings.TrimSpace(m.TargetEvent) == "" {
		return fmt.Errorf("hook mapping key and events are required")
	}
	if !supported(m.SourcePlatform) || !supported(m.TargetPlatform) || m.SourcePlatform == m.TargetPlatform {
		return fmt.Errorf("hook mapping requires distinct supported platforms")
	}
	for _, existing := range r.mappings {
		if existing.Key == m.Key && existing.SourcePlatform == m.SourcePlatform && existing.TargetPlatform == m.TargetPlatform {
			return fmt.Errorf("hook mapping %q already exists", m.Key)
		}
	}
	r.mappings = append(r.mappings, m)
	return nil
}

// Options controls hook synchronization. It is disabled by default.
type Options struct {
	Enabled     bool
	Registry    *Registry
	TrustPolicy trust.Policy
}

// Action describes one synchronization decision.
type Action string

const (
	// ActionWrite permits a validated batch write.
	ActionWrite Action = "write"
	// ActionDisabled records that hook synchronization was not enabled.
	ActionDisabled Action = "disabled"
	// ActionBlocked records that the trust policy denied behavior-bearing configuration.
	ActionBlocked Action = "blocked"
	// ActionUnmapped records that no exact event mapping exists.
	ActionUnmapped Action = "unmapped"
	// ActionUnsupported records that a harness has no supported hook adapter.
	ActionUnsupported Action = "unsupported"
)

// PlanItem records one source and target decision plus any portability warning.
type PlanItem struct {
	Source, Target model.HookConfig
	Action         Action
	Reason         string
	Warning        string
}

// Discover returns validated declarations only when explicitly enabled.
func Discover(ctx context.Context, platform model.Platform, d Discoverer, enabled bool) ([]model.HookConfig, error) {
	if !enabled {
		return nil, nil
	}
	if d == nil {
		return nil, fmt.Errorf("hook discoverer is required")
	}
	hooks, err := d.DiscoverHooks(ctx, platform)
	if err != nil {
		return nil, fmt.Errorf("discover hooks for %s: %w", platform, err)
	}
	for i := range hooks {
		if err := hooks[i].Validate(); err != nil {
			return nil, fmt.Errorf("validate discovered hook %d: %w", i, err)
		}
		if hooks[i].Platform != platform {
			return nil, fmt.Errorf("hook %q has platform %s, want %s", hooks[i].Name, hooks[i].Platform, platform)
		}
	}
	sort.Slice(hooks, func(i, j int) bool { return hooks[i].Key() < hooks[j].Key() })
	return hooks, nil
}

// Plan validates the full batch and records unsupported and lossy boundaries.
func Plan(sourceHooks []model.HookConfig, target model.Platform, opts Options) ([]PlanItem, error) {
	if !target.IsValid() {
		return nil, fmt.Errorf("unsupported target platform %q", target)
	}
	items := make([]PlanItem, 0, len(sourceHooks))
	seen := make(map[string]bool)
	for i, source := range sourceHooks {
		if err := source.Validate(); err != nil {
			return nil, fmt.Errorf("validate source hook %d: %w", i, err)
		}
		item := PlanItem{Source: source}
		switch {
		case !opts.Enabled:
			item.Action, item.Reason = ActionDisabled, "hook synchronization is disabled"
		case !supported(source.Platform) || !supported(target):
			item.Action, item.Reason = ActionUnsupported, "hook lifecycle surface is not supported for synchronization"
		case !opts.TrustPolicy.Allowed[trust.RiskNativeConfig] || !opts.TrustPolicy.Allowed[trust.RiskExecutable]:
			item.Action, item.Reason = ActionBlocked, "hook synchronization requires trust for native-config and executable"
		case source.Platform == target:
			item.Target, item.Action = source, ActionWrite
		default:
			mapping, ok := opts.Registry.lookup(source, target)
			if !ok {
				item.Action, item.Reason = ActionUnmapped, "no exact cross-platform hook mapping"
				break
			}
			item.Target, item.Action = source, ActionWrite
			item.Target.Platform, item.Target.Event = target, mapping.TargetEvent
			item.Warning = "lossy hook mapping: verify lifecycle scope and command behavior on the target"
		}
		if item.Action == ActionWrite {
			if err := validateTarget(item.Target); err != nil {
				return nil, err
			}
			if seen[item.Target.Key()] {
				return nil, fmt.Errorf("duplicate target hook %q", item.Target.Name)
			}
			seen[item.Target.Key()] = true
		}
		items = append(items, item)
	}
	return items, nil
}

// Sync writes once and only after the full batch passes preflight.
func Sync(ctx context.Context, sourceHooks []model.HookConfig, target model.Platform, writer Writer, opts Options) ([]PlanItem, error) {
	items, err := Plan(sourceHooks, target, opts)
	if err != nil {
		return nil, err
	}
	var writes []model.HookConfig
	for _, item := range items {
		if item.Action == ActionWrite {
			writes = append(writes, item.Target)
		}
	}
	if len(writes) == 0 {
		return items, nil
	}
	if writer == nil {
		return items, fmt.Errorf("hook writer is required")
	}
	if err := writer.WriteHooks(ctx, writes); err != nil {
		return items, fmt.Errorf("write hooks: %w", err)
	}
	return items, nil
}

func (r *Registry) lookup(h model.HookConfig, target model.Platform) (Mapping, bool) {
	if r == nil || h.MappingKey == "" {
		return Mapping{}, false
	}
	for _, m := range r.mappings {
		if m.Key == h.MappingKey && m.SourcePlatform == h.Platform && m.SourceEvent == h.Event && m.TargetPlatform == target {
			return m, true
		}
	}
	return Mapping{}, false
}

func supported(platform model.Platform) bool {
	return platform == model.Codex || platform == model.Gemini
}

func validateTarget(h model.HookConfig) error {
	if !supported(h.Platform) {
		return fmt.Errorf("hook codec does not support %s", h.Platform)
	}
	if err := h.Validate(); err != nil {
		return err
	}
	if !supportedEvents[h.Platform][h.Event] {
		return fmt.Errorf("%s hook %q uses unsupported event %q", h.Platform, h.Name, h.Event)
	}
	return nil
}

var supportedEvents = map[model.Platform]map[string]bool{
	model.Codex: {
		"SessionStart": true, "PreToolUse": true, "PostToolUse": true, "UserPromptSubmit": true, "Stop": true,
	},
	model.Gemini: {
		"SessionStart": true, "BeforeAgent": true, "BeforeToolSelection": true, "BeforeTool": true,
		"AfterTool": true, "BeforeModel": true, "AfterModel": true, "AfterAgent": true,
		"SessionEnd": true, "Notification": true, "PreCompress": true,
	},
}
