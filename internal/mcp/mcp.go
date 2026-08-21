// Package mcp provides opt-in, secret-safe synchronization of MCP server declarations.
package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/trust"
)

// Discoverer reads MCP declarations for one harness.
type Discoverer interface {
	DiscoverMCPServers(context.Context, model.Platform) ([]model.MCPServer, error)
}

// Writer writes a validated batch of MCP declarations.
type Writer interface {
	WriteMCPServers(context.Context, []model.MCPServer) error
}

// Mapping defines one exact cross-harness server name mapping.
type Mapping struct {
	Key, SourceName, TargetName    string
	SourcePlatform, TargetPlatform model.Platform
}

// Registry stores explicit cross-harness mappings.
type Registry struct{ mappings []Mapping }

// Register adds one exact cross-harness mapping.
func (r *Registry) Register(m Mapping) error {
	if strings.TrimSpace(m.Key) == "" || strings.TrimSpace(m.SourceName) == "" || strings.TrimSpace(m.TargetName) == "" {
		return fmt.Errorf("MCP mapping key and names are required")
	}
	if !m.SourcePlatform.IsValid() || !m.TargetPlatform.IsValid() || m.SourcePlatform == m.TargetPlatform {
		return fmt.Errorf("MCP mapping requires distinct valid platforms")
	}
	for _, existing := range r.mappings {
		if existing.Key == m.Key && existing.SourcePlatform == m.SourcePlatform && existing.TargetPlatform == m.TargetPlatform {
			return fmt.Errorf("MCP mapping %q already exists", m.Key)
		}
	}
	r.mappings = append(r.mappings, m)
	return nil
}

// Options controls MCP synchronization. It is disabled by default.
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
	// ActionDisabled records that MCP synchronization was not enabled.
	ActionDisabled Action = "disabled"
	// ActionUnmapped records that no exact mapping exists.
	ActionUnmapped Action = "unmapped"
	// ActionBlocked records that the trust policy denied native configuration.
	ActionBlocked Action = "blocked"
)

// PlanItem records one source and target decision.
type PlanItem struct {
	Source, Target model.MCPServer
	Action         Action
	Reason         string
}

// Discover returns validated declarations only when explicitly enabled.
func Discover(ctx context.Context, platform model.Platform, d Discoverer, enabled bool) ([]model.MCPServer, error) {
	if !enabled {
		return nil, nil
	}
	if d == nil {
		return nil, fmt.Errorf("MCP discoverer is required")
	}
	servers, err := d.DiscoverMCPServers(ctx, platform)
	if err != nil {
		return nil, fmt.Errorf("discover MCP servers for %s: %w", platform, err)
	}
	for i := range servers {
		if err := servers[i].Validate(); err != nil {
			return nil, fmt.Errorf("validate discovered MCP server %d: %w", i, err)
		}
		if servers[i].Platform != platform {
			return nil, fmt.Errorf("MCP server %q has platform %s, want %s", servers[i].Name, servers[i].Platform, platform)
		}
	}
	sort.Slice(servers, func(i, j int) bool { return servers[i].Key() < servers[j].Key() })
	return servers, nil
}

// Plan validates the full batch and creates decisions without invoking a writer.
func Plan(servers []model.MCPServer, target model.Platform, opts Options) ([]PlanItem, error) {
	if !target.IsValid() {
		return nil, fmt.Errorf("unsupported target platform %q", target)
	}
	items := make([]PlanItem, 0, len(servers))
	seen := make(map[string]bool)
	for i, source := range servers {
		if err := source.Validate(); err != nil {
			return nil, fmt.Errorf("validate source MCP server %d: %w", i, err)
		}
		item := PlanItem{Source: clone(source)}
		switch {
		case !opts.Enabled:
			item.Action, item.Reason = ActionDisabled, "MCP synchronization is disabled"
		case !opts.TrustPolicy.Allowed[trust.RiskNativeConfig]:
			item.Action, item.Reason = ActionBlocked, "MCP synchronization requires trust for native-config"
		case source.Platform == target:
			item.Target, item.Action = clone(source), ActionWrite
		default:
			mapping, ok := opts.Registry.lookup(source, target)
			if !ok {
				item.Action, item.Reason = ActionUnmapped, "no exact cross-platform mapping"
				break
			}
			item.Target, item.Action = clone(source), ActionWrite
			item.Target.Platform, item.Target.Name = target, mapping.TargetName
		}
		if item.Action == ActionWrite {
			if err := validateTarget(item.Target); err != nil {
				return nil, err
			}
			if seen[item.Target.Key()] {
				return nil, fmt.Errorf("duplicate target MCP server %q", item.Target.Name)
			}
			seen[item.Target.Key()] = true
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Registry) lookup(s model.MCPServer, target model.Platform) (Mapping, bool) {
	if r == nil || s.MappingKey == "" {
		return Mapping{}, false
	}
	for _, m := range r.mappings {
		if m.Key == s.MappingKey && m.SourcePlatform == s.Platform && m.SourceName == s.Name && m.TargetPlatform == target {
			return m, true
		}
	}
	return Mapping{}, false
}

func validateTarget(s model.MCPServer) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if s.Platform == model.Codex && len(s.Headers) != 0 {
		return fmt.Errorf("codex MCP server %q does not support headers", s.Name)
	}
	return nil
}

// Sync writes the batch only after every source and target passes preflight.
func Sync(ctx context.Context, servers []model.MCPServer, target model.Platform, writer Writer, opts Options) ([]PlanItem, error) {
	items, err := Plan(servers, target, opts)
	if err != nil {
		return nil, err
	}
	var writes []model.MCPServer
	for _, item := range items {
		if item.Action == ActionWrite {
			writes = append(writes, clone(item.Target))
		}
	}
	if len(writes) == 0 {
		return items, nil
	}
	if writer == nil {
		return items, fmt.Errorf("MCP writer is required")
	}
	if err := writer.WriteMCPServers(ctx, writes); err != nil {
		return items, fmt.Errorf("write MCP servers: %w", err)
	}
	return items, nil
}

func clone(s model.MCPServer) model.MCPServer {
	s.Args = append([]string(nil), s.Args...)
	s.Env = cloneMap(s.Env)
	s.Headers = cloneMap(s.Headers)
	return s
}

func cloneMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
