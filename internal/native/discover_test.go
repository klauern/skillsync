package native

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestGeminiDiscovererPreservesManifestProvenance(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	extension := filepath.Join(root, "reviewer")
	if err := os.Mkdir(extension, 0o750); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"reviewer","version":"1.2.3","source":"https://example.com/reviewer.git"}`
	if err := os.WriteFile(filepath.Join(extension, "gemini-extension.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	discoverer := GeminiDiscoverer{Root: root}
	packages, err := Discover(context.Background(), model.Gemini, discoverer, DiscoveryOptions{Enabled: true})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("Discover() returned %d packages, want 1", len(packages))
	}
	pkg := packages[0]
	if pkg.Name != "reviewer" || pkg.Kind != model.NativePackageExtension || pkg.Version != "1.2.3" || pkg.Source == "" {
		t.Fatalf("discovered package = %#v", pkg)
	}
}

func TestGeminiDiscovererIgnoresDirectoriesWithoutManifest(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "incomplete"), 0o750); err != nil {
		t.Fatal(err)
	}

	packages, err := Discover(context.Background(), model.Gemini, GeminiDiscoverer{Root: root}, DiscoveryOptions{Enabled: true})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(packages) != 0 {
		t.Fatalf("Discover() returned %#v, want no packages", packages)
	}
}
