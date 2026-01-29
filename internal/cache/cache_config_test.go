package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestNewWithCustomCacheDir(t *testing.T) {
	tmpDir := t.TempDir()
	customCacheDir := filepath.Join(tmpDir, "custom", "cache")

	cache, err := New("test", customCacheDir)
	if err != nil {
		t.Fatalf("New() with custom cache dir error = %v", err)
	}

	// Verify cache directory was created
	if _, err := os.Stat(customCacheDir); os.IsNotExist(err) {
		t.Errorf("custom cache directory was not created at %s", customCacheDir)
	}

	// Verify cache file path uses custom directory
	expectedPath := filepath.Join(customCacheDir, "test.json")
	if cache.path != expectedPath {
		t.Errorf("cache.path = %q, want %q", cache.path, expectedPath)
	}
}

func TestNewWithEmptyCacheDirUsesDefault(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILLSYNC_HOME", tmpDir)

	cache, err := New("test", "")
	if err != nil {
		t.Fatalf("New() with empty cache dir error = %v", err)
	}

	// Verify default path is used
	expectedDir := filepath.Join(tmpDir, "cache")
	expectedPath := filepath.Join(expectedDir, "test.json")
	if cache.path != expectedPath {
		t.Errorf("cache.path = %q, want %q", cache.path, expectedPath)
	}
}

func TestCachePersistsToCustomLocation(t *testing.T) {
	tmpDir := t.TempDir()
	customCacheDir := filepath.Join(tmpDir, "custom", "cache")

	// Create cache with custom location
	cache1, err := New("persist-custom", customCacheDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Create test skill file
	skillFile := filepath.Join(tmpDir, "skill.md")
	if err := os.WriteFile(skillFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to create skill file: %v", err)
	}

	skill := model.Skill{
		Name:     "test-skill",
		Platform: model.ClaudeCode,
		Path:     skillFile,
	}
	cache1.Set("test-skill", skill)

	// Save to custom location
	if err := cache1.Save(); err != nil {
		t.Fatalf("cache.Save() error = %v", err)
	}

	// Verify file exists in custom location
	cachePath := filepath.Join(customCacheDir, "persist-custom.json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("cache file not found at custom location: %s", cachePath)
	}

	// Load from same custom location
	cache2, err := New("persist-custom", customCacheDir)
	if err != nil {
		t.Fatalf("New() second time error = %v", err)
	}

	if cache2.Size() != 1 {
		t.Errorf("loaded cache.Size() = %d, want 1", cache2.Size())
	}

	retrieved, ok := cache2.Get("test-skill")
	if !ok {
		t.Error("cache should contain persisted skill")
	}

	if retrieved.Name != skill.Name {
		t.Errorf("retrieved.Name = %q, want %q", retrieved.Name, skill.Name)
	}
}
