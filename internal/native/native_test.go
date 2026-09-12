package native

import (
	"context"
	"reflect"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/trust"
)

type fakeDiscoverer struct {
	calls int
	items []model.NativePackage
}

func (f *fakeDiscoverer) DiscoverNativePackages(context.Context, model.Platform) ([]model.NativePackage, error) {
	f.calls++
	return f.items, nil
}

type fakeWriter struct {
	items []model.NativePackage
}

func (f *fakeWriter) WriteNativePackage(_ context.Context, pkg model.NativePackage) error {
	f.items = append(f.items, pkg)
	return nil
}

func nativePackage(platform model.Platform) model.NativePackage {
	return model.NativePackage{
		Name:       "commits@marketplace",
		Kind:       model.NativePackagePlugin,
		Platform:   platform,
		Provenance: model.NativeProvenanceMarketplace,
		Source:     "marketplace.example/commits",
		Version:    "1.2.3",
		Enabled:    true,
		Metadata:   map[string]string{"revision": "abc123"},
	}
}

func trustedOptions() Options {
	return Options{Enabled: true, TrustPolicy: trust.Policy{Allowed: map[trust.Risk]bool{trust.RiskNativeConfig: true}}}
}

func TestDiscoverIsOptIn(t *testing.T) {
	t.Parallel()
	discoverer := &fakeDiscoverer{items: []model.NativePackage{nativePackage(model.ClaudeCode)}}

	got, err := Discover(context.Background(), model.ClaudeCode, discoverer, DiscoveryOptions{})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if got != nil || discoverer.calls != 0 {
		t.Fatalf("Discover() = %#v, calls = %d; want nil and zero calls", got, discoverer.calls)
	}

	got, err = Discover(context.Background(), model.ClaudeCode, discoverer, DiscoveryOptions{Enabled: true})
	if err != nil {
		t.Fatalf("enabled Discover() error = %v", err)
	}
	if len(got) != 1 || discoverer.calls != 1 {
		t.Fatalf("enabled Discover() = %#v, calls = %d", got, discoverer.calls)
	}
}

func TestSyncDisabledAndUnmappedNeverWrite(t *testing.T) {
	t.Parallel()
	source := nativePackage(model.ClaudeCode)
	writer := &fakeWriter{}

	items, err := Sync(context.Background(), []model.NativePackage{source}, model.Codex, writer, Options{})
	if err != nil {
		t.Fatalf("disabled Sync() error = %v", err)
	}
	if items[0].Action != ActionDisabled || len(writer.items) != 0 {
		t.Fatalf("disabled action = %q, writes = %d", items[0].Action, len(writer.items))
	}

	blocked, err := Sync(context.Background(), []model.NativePackage{source}, model.Codex, writer, Options{Enabled: true})
	if err != nil {
		t.Fatalf("blocked Sync() error = %v", err)
	}
	if blocked[0].Action != ActionBlocked || len(writer.items) != 0 {
		t.Fatalf("blocked action = %q, writes = %d", blocked[0].Action, len(writer.items))
	}

	items, err = Sync(context.Background(), []model.NativePackage{source}, model.Codex, writer, trustedOptions())
	if err != nil {
		t.Fatalf("unmapped Sync() error = %v", err)
	}
	if items[0].Action != ActionUnmapped || len(writer.items) != 0 {
		t.Fatalf("unmapped action = %q, writes = %d", items[0].Action, len(writer.items))
	}
}

func TestSyncRequiresExactMappingAndPreservesProvenance(t *testing.T) {
	t.Parallel()
	source := nativePackage(model.ClaudeCode)
	source.MappingKey = "commits"
	registry := &Registry{}
	if err := registry.Register(Mapping{
		Key:            "commits",
		SourcePlatform: model.ClaudeCode,
		SourceKind:     model.NativePackagePlugin,
		SourceName:     source.Name,
		TargetPlatform: model.Codex,
		TargetKind:     model.NativePackagePackage,
		TargetName:     "marketplace/commits",
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	writer := &fakeWriter{}
	opts := trustedOptions()
	opts.Registry = registry
	items, err := Sync(context.Background(), []model.NativePackage{source}, model.Codex, writer, opts)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if items[0].Action != ActionWrite || len(writer.items) != 1 {
		t.Fatalf("action = %q, writes = %d", items[0].Action, len(writer.items))
	}
	got := writer.items[0]
	if got.Platform != model.Codex || got.Kind != model.NativePackagePackage || got.Name != "marketplace/commits" {
		t.Fatalf("mapped identity = %#v", got)
	}
	if got.Provenance != source.Provenance || got.Source != source.Source || got.Version != source.Version || !reflect.DeepEqual(got.Metadata, source.Metadata) {
		t.Fatalf("mapped provenance = %#v, want source provenance %#v", got, source)
	}

	got.Metadata["revision"] = "changed"
	if source.Metadata["revision"] != "abc123" {
		t.Fatal("Sync() aliased source metadata")
	}
}

func TestSamePlatformSyncPreservesPackage(t *testing.T) {
	t.Parallel()
	source := nativePackage(model.Gemini)
	writer := &fakeWriter{}

	_, err := Sync(context.Background(), []model.NativePackage{source}, model.Gemini, writer, trustedOptions())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if len(writer.items) != 1 || !reflect.DeepEqual(writer.items[0], source) {
		t.Fatalf("written package = %#v, want %#v", writer.items, source)
	}
}
