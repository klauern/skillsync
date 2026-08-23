// Package agents provides opt-in synchronization of harness-native custom agents.
package agents

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/trust"
)

// Discoverer reads custom agents for one harness without invoking them.
type Discoverer interface {
	DiscoverAgents(context.Context, model.Platform) ([]model.CustomAgent, error)
}

// Writer writes one fully validated custom-agent batch.
type Writer interface {
	WriteAgents(context.Context, []model.CustomAgent) error
}

// Mapping permits one exact directional mapping between native harnesses.
type Mapping struct {
	Key                            string
	SourcePlatform, TargetPlatform model.Platform
}

// Registry stores explicit cross-harness mappings.
type Registry struct{ mappings []Mapping }

// Register adds one exact directional mapping.
func (r *Registry) Register(m Mapping) error {
	if strings.TrimSpace(m.Key) == "" || !supported(m.SourcePlatform) || !supported(m.TargetPlatform) || m.SourcePlatform == m.TargetPlatform {
		return fmt.Errorf("agent mapping requires a key and distinct supported platforms")
	}
	for _, existing := range r.mappings {
		if existing == m {
			return fmt.Errorf("agent mapping %q already exists", m.Key)
		}
	}
	r.mappings = append(r.mappings, m)
	return nil
}

// Options controls custom-agent synchronization. It is disabled by default.
type Options struct {
	Enabled     bool
	Registry    *Registry
	TrustPolicy trust.Policy
}

// Action describes one synchronization decision.
type Action string

const (
	// ActionWrite permits a validated native-agent write.
	ActionWrite Action = "write"
	// ActionDisabled records that synchronization was not enabled.
	ActionDisabled Action = "disabled"
	// ActionBlocked records that the trust policy denied native configuration.
	ActionBlocked Action = "blocked"
	// ActionUnmapped records that no exact directional mapping exists.
	ActionUnmapped Action = "unmapped"
	// ActionUnsupported records that a harness has no native-agent adapter.
	ActionUnsupported Action = "unsupported"
)

// PlanItem records one source and target decision plus any loss warning.
type PlanItem struct {
	Source, Target model.CustomAgent
	Action         Action
	Reason         string
	Warning        string
}

// Discover returns validated custom agents only when explicitly enabled.
func Discover(ctx context.Context, platform model.Platform, d Discoverer, enabled bool) ([]model.CustomAgent, error) {
	if !enabled {
		return nil, nil
	}
	if d == nil {
		return nil, fmt.Errorf("custom agent discoverer is required")
	}
	found, err := d.DiscoverAgents(ctx, platform)
	if err != nil {
		return nil, fmt.Errorf("discover custom agents for %s: %w", platform, err)
	}
	for i := range found {
		if err := found[i].Validate(); err != nil {
			return nil, fmt.Errorf("validate discovered custom agent %d: %w", i, err)
		}
		if found[i].Platform != platform {
			return nil, fmt.Errorf("custom agent %q has platform %s, want %s", found[i].Name, found[i].Platform, platform)
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Key() < found[j].Key() })
	return found, nil
}

// Plan validates the full batch and records unsupported and lossy boundaries.
func Plan(sourceAgents []model.CustomAgent, target model.Platform, opts Options) ([]PlanItem, error) {
	if !target.IsValid() {
		return nil, fmt.Errorf("unsupported target platform %q", target)
	}
	items := make([]PlanItem, 0, len(sourceAgents))
	seen := make(map[string]bool)
	for i, source := range sourceAgents {
		if err := source.Validate(); err != nil {
			return nil, fmt.Errorf("validate source custom agent %d: %w", i, err)
		}
		item := PlanItem{Source: source}
		switch {
		case !opts.Enabled:
			item.Action, item.Reason = ActionDisabled, "custom agent synchronization is disabled"
		case !supported(source.Platform) || !supported(target):
			item.Action, item.Reason = ActionUnsupported, "harness has no supported native custom-agent adapter"
		case !opts.TrustPolicy.Allowed[trust.RiskNativeConfig]:
			item.Action, item.Reason = ActionBlocked, "custom agent synchronization requires trust for native-config"
		case source.Platform == target:
			item.Target, item.Action = clone(source), ActionWrite
		default:
			if !opts.Registry.lookup(source, target) {
				item.Action, item.Reason = ActionUnmapped, "no exact cross-platform custom-agent mapping"
				break
			}
			item.Target, item.Action = clone(source), ActionWrite
			item.Target.Platform = target
			item.Target.Native = nil
			item.Warning = "lossy custom-agent mapping: verify routing, tools, model, and invocation behavior on the target"
		}
		if item.Action == ActionWrite {
			if err := item.Target.Validate(); err != nil {
				return nil, err
			}
			if seen[item.Target.Key()] {
				return nil, fmt.Errorf("duplicate target custom agent %q", item.Target.Name)
			}
			seen[item.Target.Key()] = true
		}
		items = append(items, item)
	}
	return items, nil
}

// Sync writes once and only after the full batch passes preflight.
func Sync(ctx context.Context, sourceAgents []model.CustomAgent, target model.Platform, writer Writer, opts Options) ([]PlanItem, error) {
	items, err := Plan(sourceAgents, target, opts)
	if err != nil {
		return nil, err
	}
	var writes []model.CustomAgent
	for _, item := range items {
		if item.Action == ActionWrite {
			writes = append(writes, item.Target)
		}
	}
	if len(writes) == 0 {
		return items, nil
	}
	if writer == nil {
		return items, fmt.Errorf("custom agent writer is required")
	}
	if err := writer.WriteAgents(ctx, writes); err != nil {
		return items, fmt.Errorf("write custom agents: %w", err)
	}
	return items, nil
}

func (r *Registry) lookup(a model.CustomAgent, target model.Platform) bool {
	if r == nil || a.MappingKey == "" {
		return false
	}
	for _, m := range r.mappings {
		if m.Key == a.MappingKey && m.SourcePlatform == a.Platform && m.TargetPlatform == target {
			return true
		}
	}
	return false
}

func supported(p model.Platform) bool {
	return p == model.ClaudeCode || p == model.Copilot || p == model.Gemini
}

func clone(a model.CustomAgent) model.CustomAgent {
	a.Tools = append([]string(nil), a.Tools...)
	if a.Native != nil {
		a.Native = make(map[string]any, len(a.Native))
		for k, v := range a.Native {
			a.Native[k] = v
		}
	}
	return a
}
