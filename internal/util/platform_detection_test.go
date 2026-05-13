package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatalf("failed to create %s: %v", path, err)
	}
}

func TestDetectInstalledPlatformsWithConfig(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("HOME", home)

	// Full presence for Claude Code: repo and user paths exist.
	mustMkdirAll(t, filepath.Join(cwd, ".claude", "skills"))
	mustMkdirAll(t, filepath.Join(home, ".claude", "skills"))

	// Partial presence for Codex: only one of the checked locations exists.
	mustMkdirAll(t, filepath.Join(home, ".codex", "skills"))

	result := DetectInstalledPlatformsWithConfig(TieredPathConfig{WorkingDir: cwd})

	if !result.HasPlatform(model.ClaudeCode) {
		t.Fatal("expected Claude Code to be detected")
	}
	if !result.HasPlatform(model.Codex) {
		t.Fatal("expected Codex to be detected")
	}
	if result.HasPlatform(model.Gemini) {
		t.Fatal("did not expect Gemini to be detected")
	}

	claude, ok := result.Detail(model.ClaudeCode)
	if !ok {
		t.Fatal("missing Claude Code detail")
	}
	if claude.Status != PlatformDetectionPresent {
		t.Fatalf("Claude Code status = %q, want %q", claude.Status, PlatformDetectionPresent)
	}
	if claude.Reason == "" {
		t.Fatal("Claude Code reason should not be empty")
	}

	codex, ok := result.Detail(model.Codex)
	if !ok {
		t.Fatal("missing Codex detail")
	}
	if codex.Status != PlatformDetectionPartial {
		t.Fatalf("Codex status = %q, want %q", codex.Status, PlatformDetectionPartial)
	}
	if len(codex.MissingPaths) == 0 {
		t.Fatal("expected Codex missing paths to be reported")
	}
	if codex.Reason == "" {
		t.Fatal("Codex reason should not be empty")
	}
}

func TestDetectInstalledPlatformsWithConfig_MissingReportsReason(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("HOME", home)

	result := DetectInstalledPlatformsWithConfig(TieredPathConfig{WorkingDir: cwd})

	gemini, ok := result.Detail(model.Gemini)
	if !ok {
		t.Fatal("missing Gemini detail")
	}
	if gemini.Status != PlatformDetectionMissing {
		t.Fatalf("Gemini status = %q, want %q", gemini.Status, PlatformDetectionMissing)
	}
	if gemini.Reason == "" {
		t.Fatal("Gemini reason should not be empty")
	}
	if len(gemini.PresentPaths) != 0 {
		t.Fatalf("Gemini present paths = %d, want 0", len(gemini.PresentPaths))
	}
}
