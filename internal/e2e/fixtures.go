package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

// Fixture provides helpers for creating test fixtures in E2E tests.
type Fixture struct {
	t       *testing.T
	baseDir string
}

// NewFixture creates a new fixture helper rooted at the given directory.
func NewFixture(t *testing.T, baseDir string) *Fixture {
	t.Helper()
	return &Fixture{
		t:       t,
		baseDir: baseDir,
	}
}

// WriteFile writes content to a file relative to the fixture base directory.
// It creates parent directories as needed.
func (f *Fixture) WriteFile(relPath, content string) string {
	f.t.Helper()
	fullPath := filepath.Join(f.baseDir, relPath)

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		f.t.Fatalf("failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
		f.t.Fatalf("failed to write file %s: %v", fullPath, err)
	}

	return fullPath
}

// WriteSkill writes a skill file with frontmatter and content.
// This is a convenience helper for creating typical skill files.
func (f *Fixture) WriteSkill(relPath, name, description, content string) string {
	f.t.Helper()

	skillContent := "---\n"
	skillContent += "name: " + name + "\n"
	if description != "" {
		skillContent += "description: " + description + "\n"
	}
	skillContent += "---\n\n"
	skillContent += content

	return f.WriteFile(relPath, skillContent)
}

// MkdirAll creates a directory and all parent directories relative to the base.
func (f *Fixture) MkdirAll(relPath string) string {
	f.t.Helper()
	fullPath := filepath.Join(f.baseDir, relPath)

	if err := os.MkdirAll(fullPath, 0o750); err != nil {
		f.t.Fatalf("failed to create directory %s: %v", fullPath, err)
	}

	return fullPath
}

// Path returns the full path for a relative path.
func (f *Fixture) Path(relPath string) string {
	return filepath.Join(f.baseDir, relPath)
}

// Exists returns true if the file or directory exists.
func (f *Fixture) Exists(relPath string) bool {
	f.t.Helper()
	fullPath := filepath.Join(f.baseDir, relPath)
	_, err := os.Stat(fullPath)
	return err == nil
}

// ReadFile reads and returns the content of a file.
func (f *Fixture) ReadFile(relPath string) string {
	f.t.Helper()
	fullPath := filepath.Join(f.baseDir, relPath)

	// #nosec G304 - fullPath is constructed from trusted test fixture base and test-provided path
	data, err := os.ReadFile(fullPath)
	if err != nil {
		f.t.Fatalf("failed to read file %s: %v", fullPath, err)
	}

	return string(data)
}

// ClaudeCodeFixture creates a fixture helper for Claude Code skills directory.
// It sets up the expected directory structure for Claude Code.
// The path matches the SKILLSYNC_CLAUDE_CODE_PATH environment variable set by NewHarness.
func (h *Harness) ClaudeCodeFixture() *Fixture {
	h.t.Helper()

	// Claude Code stores skills in the path set by SKILLSYNC_CLAUDE_CODE_PATH
	skillsDir := h.env["SKILLSYNC_CLAUDE_CODE_PATH"]
	if skillsDir == "" {
		skillsDir = filepath.Join(h.homeDir, ".claude", "commands")
	}
	if err := os.MkdirAll(skillsDir, 0o750); err != nil {
		h.t.Fatalf("failed to create Claude Code skills directory: %v", err)
	}

	return NewFixture(h.t, skillsDir)
}

// CursorFixture creates a fixture helper for Cursor skills directory.
// It sets up the expected directory structure for Cursor.
// The path matches the SKILLSYNC_CURSOR_PATH environment variable set by NewHarness.
func (h *Harness) CursorFixture() *Fixture {
	h.t.Helper()

	// Cursor stores skills in the path set by SKILLSYNC_CURSOR_PATH
	skillsDir := h.env["SKILLSYNC_CURSOR_PATH"]
	if skillsDir == "" {
		skillsDir = filepath.Join(h.homeDir, ".cursor", "rules")
	}
	if err := os.MkdirAll(skillsDir, 0o750); err != nil {
		h.t.Fatalf("failed to create Cursor skills directory: %v", err)
	}

	return NewFixture(h.t, skillsDir)
}

// CodexFixture creates a fixture helper for Codex skills directory.
// It sets up the expected directory structure for Codex.
// The path matches the SKILLSYNC_CODEX_PATH environment variable set by NewHarness.
func (h *Harness) CodexFixture() *Fixture {
	h.t.Helper()

	// Codex stores skills in the path set by SKILLSYNC_CODEX_PATH
	skillsDir := h.env["SKILLSYNC_CODEX_PATH"]
	if skillsDir == "" {
		skillsDir = filepath.Join(h.homeDir, ".codex", "skills")
	}
	if err := os.MkdirAll(skillsDir, 0o750); err != nil {
		h.t.Fatalf("failed to create Codex skills directory: %v", err)
	}

	return NewFixture(h.t, skillsDir)
}

// TempFixture creates a fixture helper for a new temporary directory.
func (h *Harness) TempFixture() *Fixture {
	h.t.Helper()

	tempDir := h.t.TempDir()
	return NewFixture(h.t, tempDir)
}
