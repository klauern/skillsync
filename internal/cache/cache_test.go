package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/model"
)

func TestNew(t *testing.T) {
	// Set up temp directory
	tmpDir := t.TempDir()
	t.Setenv("SKILLSYNC_HOME", tmpDir)

	cache, err := New("test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if cache.Version != cacheVersion {
		t.Errorf("cache.Version = %q, want %q", cache.Version, cacheVersion)
	}

	if cache.Entries == nil {
		t.Error("cache.Entries should not be nil")
	}

	if cache.Size() != 0 {
		t.Errorf("cache.Size() = %d, want 0", cache.Size())
	}
}

func TestCacheSetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILLSYNC_HOME", tmpDir)

	// Create a test file for the skill path
	skillFile := filepath.Join(tmpDir, "test-skill.md")
	if err := os.WriteFile(skillFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cache, err := New("test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	skill := model.Skill{
		Name:        "test-skill",
		Description: "A test skill",
		Platform:    model.ClaudeCode,
		Path:        skillFile,
		Content:     "test content",
		ModifiedAt:  time.Now(),
	}

	// Set skill
	cache.Set("test-skill", skill)

	if cache.Size() != 1 {
		t.Errorf("cache.Size() = %d, want 1", cache.Size())
	}

	// Get skill
	retrieved, ok := cache.Get("test-skill")
	if !ok {
		t.Error("cache.Get() should return true for existing key")
	}

	if retrieved.Name != skill.Name {
		t.Errorf("retrieved.Name = %q, want %q", retrieved.Name, skill.Name)
	}

	if retrieved.Description != skill.Description {
		t.Errorf("retrieved.Description = %q, want %q", retrieved.Description, skill.Description)
	}

	// Get non-existent key
	_, ok = cache.Get("non-existent")
	if ok {
		t.Error("cache.Get() should return false for non-existent key")
	}
}

func TestCacheSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILLSYNC_HOME", tmpDir)

	// Create a test file for the skill path
	skillFile := filepath.Join(tmpDir, "test-skill.md")
	if err := os.WriteFile(skillFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create and populate cache
	cache1, err := New("test-persist")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	skill := model.Skill{
		Name:        "persisted-skill",
		Description: "A persisted skill",
		Platform:    model.Cursor,
		Path:        skillFile,
		Content:     "test content",
	}

	cache1.Set("persisted-skill", skill)

	// Save cache
	if err := cache1.Save(); err != nil {
		t.Fatalf("cache.Save() error = %v", err)
	}

	// Load cache in new instance
	cache2, err := New("test-persist")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if cache2.Size() != 1 {
		t.Errorf("loaded cache.Size() = %d, want 1", cache2.Size())
	}

	retrieved, ok := cache2.Get("persisted-skill")
	if !ok {
		t.Error("loaded cache should contain persisted skill")
	}

	if retrieved.Name != skill.Name {
		t.Errorf("retrieved.Name = %q, want %q", retrieved.Name, skill.Name)
	}
}

func TestCacheStaleDetection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILLSYNC_HOME", tmpDir)

	// Create a test file for the skill path
	skillFile := filepath.Join(tmpDir, "test-skill.md")
	if err := os.WriteFile(skillFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cache, err := New("test-stale")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	skill := model.Skill{
		Name:     "stale-skill",
		Platform: model.ClaudeCode,
		Path:     skillFile,
	}

	cache.Set("stale-skill", skill)

	// Fresh cache should not be stale
	if cache.IsStale(time.Hour) {
		t.Error("fresh cache should not be stale with 1 hour TTL")
	}

	// Cache should be stale with 0 TTL
	if !cache.IsStale(0) {
		t.Error("cache should be stale with 0 TTL")
	}
}

func TestCachePrune(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILLSYNC_HOME", tmpDir)

	// Create a test file for the skill path
	skillFile := filepath.Join(tmpDir, "test-skill.md")
	if err := os.WriteFile(skillFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cache, err := New("test-prune")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	skill := model.Skill{
		Name:     "prune-skill",
		Platform: model.ClaudeCode,
		Path:     skillFile,
	}

	cache.Set("prune-skill", skill)

	// Prune with long TTL should not remove anything
	pruned := cache.Prune(time.Hour)
	if pruned != 0 {
		t.Errorf("Prune() with long TTL = %d, want 0", pruned)
	}

	if cache.Size() != 1 {
		t.Errorf("cache.Size() after prune = %d, want 1", cache.Size())
	}

	// Prune with 0 TTL should remove all
	pruned = cache.Prune(0)
	if pruned != 1 {
		t.Errorf("Prune() with 0 TTL = %d, want 1", pruned)
	}

	if cache.Size() != 0 {
		t.Errorf("cache.Size() after prune = %d, want 0", cache.Size())
	}
}

func TestCacheClear(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILLSYNC_HOME", tmpDir)

	// Create a test file for the skill path
	skillFile := filepath.Join(tmpDir, "test-skill.md")
	if err := os.WriteFile(skillFile, []byte("test content"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cache, err := New("test-clear")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	skill := model.Skill{
		Name:     "clear-skill",
		Platform: model.ClaudeCode,
		Path:     skillFile,
	}

	cache.Set("clear-skill", skill)

	// Save first
	if err := cache.Save(); err != nil {
		t.Fatalf("cache.Save() error = %v", err)
	}

	// Clear should remove all entries and cache file
	if err := cache.Clear(); err != nil && !os.IsNotExist(err) {
		t.Fatalf("cache.Clear() error = %v", err)
	}

	if cache.Size() != 0 {
		t.Errorf("cache.Size() after clear = %d, want 0", cache.Size())
	}
}

func TestCacheStaleSourceFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILLSYNC_HOME", tmpDir)

	// Create a test file for the skill path
	skillFile := filepath.Join(tmpDir, "test-skill.md")
	if err := os.WriteFile(skillFile, []byte("original content"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cache, err := New("test-source-stale")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	skill := model.Skill{
		Name:     "source-stale-skill",
		Platform: model.ClaudeCode,
		Path:     skillFile,
		Content:  "original content",
	}

	cache.Set("source-stale-skill", skill)

	// Skill should be retrievable
	_, ok := cache.Get("source-stale-skill")
	if !ok {
		t.Error("cache.Get() should return true for fresh entry")
	}

	// Wait a moment and modify the source file
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(skillFile, []byte("modified content"), 0o600); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	// Now the cache entry should be stale
	_, ok = cache.Get("source-stale-skill")
	if ok {
		t.Error("cache.Get() should return false when source file is modified")
	}
}
