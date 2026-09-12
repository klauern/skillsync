// Package native provides opt-in discovery and synchronization of harness-owned
// plugins, packages, and extensions.
package native

import (
	"context"
	"fmt"
	"strings"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/trust"
)

// Discoverer reads native installations for one harness.
type Discoverer interface {
	DiscoverNativePackages(context.Context, model.Platform) ([]model.NativePackage, error)
}

// Writer installs one native package on its target harness.
type Writer interface {
	WriteNativePackage(context.Context, model.NativePackage) error
}

// DiscoveryOptions controls native package discovery. Discovery is disabled by
// default because native installations are separate from portable skills.
type DiscoveryOptions struct {
	Enabled bool
}

// Discover returns native packages only when explicitly enabled.
func Discover(ctx context.Context, platform model.Platform, discoverer Discoverer, opts DiscoveryOptions) ([]model.NativePackage, error) {
	if !opts.Enabled {
		return nil, nil
	}
	if discoverer == nil {
		return nil, fmt.Errorf("native package discoverer is required")
	}
	packages, err := discoverer.DiscoverNativePackages(ctx, platform)
	if err != nil {
		return nil, fmt.Errorf("discover native packages for %s: %w", platform, err)
	}
	for i := range packages {
		if err := packages[i].Validate(); err != nil {
			return nil, fmt.Errorf("validate discovered native package %d: %w", i, err)
		}
		if packages[i].Platform != platform {
			return nil, fmt.Errorf("discovered native package %q has platform %s, want %s", packages[i].Name, packages[i].Platform, platform)
		}
	}
	return packages, nil
}

// Mapping is an explicit conversion of one native identity to another harness.
type Mapping struct {
	Key            string
	SourcePlatform model.Platform
	SourceKind     model.NativePackageKind
	SourceName     string
	TargetPlatform model.Platform
	TargetKind     model.NativePackageKind
	TargetName     string
}

// Registry stores explicit cross-harness mappings.
type Registry struct {
	mappings []Mapping
}

// Register adds one exact cross-harness mapping.
func (r *Registry) Register(mapping Mapping) error {
	if strings.TrimSpace(mapping.Key) == "" || strings.TrimSpace(mapping.SourceName) == "" || strings.TrimSpace(mapping.TargetName) == "" {
		return fmt.Errorf("native package mapping key and names are required")
	}
	if !mapping.SourcePlatform.IsValid() || !mapping.TargetPlatform.IsValid() {
		return fmt.Errorf("native package mapping platforms must be valid")
	}
	if mapping.SourcePlatform == mapping.TargetPlatform {
		return fmt.Errorf("native package mapping must be cross-platform")
	}
	if !mapping.SourceKind.IsValid() || !mapping.TargetKind.IsValid() {
		return fmt.Errorf("native package mapping kinds must be valid")
	}
	for _, existing := range r.mappings {
		if existing.Key == mapping.Key && existing.SourcePlatform == mapping.SourcePlatform && existing.TargetPlatform == mapping.TargetPlatform {
			return fmt.Errorf("native package mapping %q already registered for %s to %s", mapping.Key, mapping.SourcePlatform, mapping.TargetPlatform)
		}
	}
	r.mappings = append(r.mappings, mapping)
	return nil
}

func (r *Registry) lookup(source model.NativePackage, target model.Platform) (Mapping, bool) {
	if r == nil || source.MappingKey == "" {
		return Mapping{}, false
	}
	for _, mapping := range r.mappings {
		if mapping.Key == source.MappingKey && mapping.SourcePlatform == source.Platform && mapping.SourceKind == source.Kind &&
			mapping.SourceName == source.Name && mapping.TargetPlatform == target {
			return mapping, true
		}
	}
	return Mapping{}, false
}

// Options controls native synchronization. Synchronization is disabled by default.
type Options struct {
	Enabled     bool
	Registry    *Registry
	TrustPolicy trust.Policy
}

// Action describes the disposition of one source package.
type Action string

const (
	// ActionWrite permits a native package write.
	ActionWrite Action = "write"
	// ActionDisabled records that native synchronization was not enabled.
	ActionDisabled Action = "disabled"
	// ActionUnmapped records that no exact cross-harness mapping exists.
	ActionUnmapped Action = "unmapped"
	// ActionBlocked records that the trust policy denied native configuration.
	ActionBlocked Action = "blocked"
)

// PlanItem describes one synchronization decision.
type PlanItem struct {
	Source model.NativePackage
	Target model.NativePackage
	Action Action
	Reason string
}

// Plan creates synchronization decisions without invoking a writer.
func Plan(packages []model.NativePackage, target model.Platform, opts Options) ([]PlanItem, error) {
	if !target.IsValid() {
		return nil, fmt.Errorf("unsupported target platform %q", target)
	}
	items := make([]PlanItem, 0, len(packages))
	for i, source := range packages {
		if err := source.Validate(); err != nil {
			return nil, fmt.Errorf("validate source native package %d: %w", i, err)
		}
		item := PlanItem{Source: clonePackage(source)}
		if !opts.Enabled {
			item.Action, item.Reason = ActionDisabled, "native package synchronization is disabled"
			items = append(items, item)
			continue
		}
		if !opts.TrustPolicy.Allowed[trust.RiskNativeConfig] {
			item.Action, item.Reason = ActionBlocked, "native package installation requires trust for native-config"
			items = append(items, item)
			continue
		}
		if source.Platform == target {
			item.Target, item.Action = clonePackage(source), ActionWrite
			items = append(items, item)
			continue
		}
		mapping, ok := opts.Registry.lookup(source, target)
		if !ok {
			item.Action, item.Reason = ActionUnmapped, "no exact cross-platform mapping"
			items = append(items, item)
			continue
		}
		mapped := clonePackage(source)
		mapped.Platform = mapping.TargetPlatform
		mapped.Kind = mapping.TargetKind
		mapped.Name = mapping.TargetName
		item.Target, item.Action = mapped, ActionWrite
		items = append(items, item)
	}
	return items, nil
}

// Sync executes writable plan items. Disabled and unmapped items never reach the writer.
func Sync(ctx context.Context, packages []model.NativePackage, target model.Platform, writer Writer, opts Options) ([]PlanItem, error) {
	items, err := Plan(packages, target, opts)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.Action != ActionWrite {
			continue
		}
		if writer == nil {
			return items, fmt.Errorf("native package writer is required")
		}
		if err := writer.WriteNativePackage(ctx, clonePackage(item.Target)); err != nil {
			return items, fmt.Errorf("write native package %q: %w", item.Target.Name, err)
		}
	}
	return items, nil
}

func clonePackage(pkg model.NativePackage) model.NativePackage {
	pkg.Metadata = cloneMetadata(pkg.Metadata)
	return pkg
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return nil
	}
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}
