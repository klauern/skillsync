package util

import (
	"path/filepath"
	"testing"
)

func TestHomeDir(t *testing.T) {
	home := HomeDir()
	if home == "" {
		t.Error("HomeDir() returned empty string")
	}

	// Verify it's an absolute path
	if !filepath.IsAbs(home) {
		t.Errorf("HomeDir() returned relative path: %s", home)
	}
}

func TestClaudeCodeSkillsPath(t *testing.T) {
	path := ClaudeCodeSkillsPath()

	expected := filepath.Join(HomeDir(), ".claude", "skills")
	if path != expected {
		t.Errorf("ClaudeCodeSkillsPath() = %q, want %q", path, expected)
	}
}

func TestCursorRulesPath(t *testing.T) {
	projectDir := "/test/project"
	path := CursorRulesPath(projectDir)

	expected := "/test/project/.cursor/rules"
	if path != expected {
		t.Errorf("CursorRulesPath(%q) = %q, want %q", projectDir, path, expected)
	}
}

func TestCodexConfigPath(t *testing.T) {
	projectDir := "/test/project"
	path := CodexConfigPath(projectDir)

	expected := "/test/project/.codex"
	if path != expected {
		t.Errorf("CodexConfigPath(%q) = %q, want %q", projectDir, path, expected)
	}
}
