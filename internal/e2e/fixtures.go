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

// NewFixture creates a new fixture helper rooted at a directory.
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
// It is a convenience helper for creating typical skill files.
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

// Exists returns true if a file or directory exists.
func (f *Fixture) Exists(relPath string) bool {
	f.t.Helper()
	fullPath := filepath.Join(f.baseDir, relPath)
	_, err := os.Stat(fullPath)
	return err == nil
}

// ReadFile reads and returns file content.
func (f *Fixture) ReadFile(relPath string) string {
	f.t.Helper()
	fullPath := filepath.Join(f.baseDir, relPath)

	// #nosec G304 - fullPath is constructed from a trusted fixture base and test-provided path.
	data, err := os.ReadFile(fullPath)
	if err != nil {
		f.t.Fatalf("failed to read file %s: %v", fullPath, err)
	}

	return string(data)
}

func (h *Harness) platformFixture(envKey, label string, fallbackParts ...string) *Fixture {
	h.t.Helper()

	skillsDir := h.env[envKey]
	if skillsDir == "" {
		skillsDir = filepath.Join(append([]string{h.homeDir}, fallbackParts...)...)
	}
	if err := os.MkdirAll(skillsDir, 0o750); err != nil {
		h.t.Fatalf("failed to create %s directory: %v", label, err)
	}

	return NewFixture(h.t, skillsDir)
}

func (h *Harness) addDiscoveryPath(envKey, path string) {
	h.t.Helper()
	paths := h.env[envKey]
	if paths == "" {
		h.SetEnv(envKey, path)
		return
	}
	h.SetEnv(envKey, paths+string(os.PathListSeparator)+path)
}

// ClaudeCodeFixture creates a fixture helper for the Claude Code skills directory.
// It sets up the expected directory structure for Claude Code.
// The path matches the SKILLSYNC_CLAUDE_CODE_PATH environment variable set by NewHarness.
func (h *Harness) ClaudeCodeFixture() *Fixture {
	h.t.Helper()
	return h.platformFixture("SKILLSYNC_CLAUDE_CODE_PATH", "Claude Code skills", ".claude", "skills")
}

// ClaudeCommandsFixture creates a fixture for Claude Code command artifacts.
func (h *Harness) ClaudeCommandsFixture() *Fixture {
	h.t.Helper()
	fixture := h.platformFixture("", "Claude Code commands", ".claude", "commands")
	h.addDiscoveryPath("SKILLSYNC_CLAUDE_CODE_SKILLS_PATHS", fixture.Path(""))
	return fixture
}

// CursorFixture creates a fixture helper for the Cursor skills directory.
// It sets up the expected directory structure for Cursor.
// The path matches the SKILLSYNC_CURSOR_PATH environment variable set by NewHarness.
func (h *Harness) CursorFixture() *Fixture {
	h.t.Helper()
	return h.platformFixture("SKILLSYNC_CURSOR_PATH", "Cursor skills", ".cursor", "skills")
}

// CursorRulesFixture creates a fixture for Cursor project rules.
func (h *Harness) CursorRulesFixture() *Fixture {
	h.t.Helper()
	fixture := h.platformFixture("", "Cursor rules", ".cursor", "rules")
	h.addDiscoveryPath("SKILLSYNC_CURSOR_SKILLS_PATHS", fixture.Path(""))
	return fixture
}

// CodexFixture creates a fixture helper for the Codex skills directory.
// It sets up the expected directory structure for Codex.
// The path matches the SKILLSYNC_CODEX_PATH environment variable set by NewHarness.
func (h *Harness) CodexFixture() *Fixture {
	h.t.Helper()
	return h.platformFixture("SKILLSYNC_CODEX_PATH", "Codex skills", ".agents", "skills")
}

// PiFixture creates a fixture helper for the Pi user skills directory.
func (h *Harness) PiFixture() *Fixture {
	h.t.Helper()
	return h.platformFixture("SKILLSYNC_PI_PATH", "Pi skills", ".pi", "agent", "skills")
}

// PiDevFixture is retained for compatibility with older E2E tests.
func (h *Harness) PiDevFixture() *Fixture {
	h.t.Helper()
	return h.PiFixture()
}

// CopilotFixture creates a fixture helper for the GitHub Copilot skills directory.
// It sets up the expected ~/.github structure in an isolated home.
func (h *Harness) CopilotFixture() *Fixture {
	h.t.Helper()
	return h.platformFixture("SKILLSYNC_COPILOT_PATH", "Copilot skills", ".copilot", "skills")
}

// CopilotRepoFixture creates a fixture for repository instruction, prompt, and agent artifacts.
func (h *Harness) CopilotRepoFixture() *Fixture {
	h.t.Helper()
	fixture := h.platformFixture("", "Copilot repository artifacts", ".github")
	h.addDiscoveryPath("SKILLSYNC_COPILOT_SKILLS_PATHS", fixture.Path(""))
	return fixture
}

// GeminiFixture creates a fixture helper for the Gemini CLI skills directory.
// It sets up the expected ~/.gemini structure in an isolated home.
func (h *Harness) GeminiFixture() *Fixture {
	h.t.Helper()
	return h.platformFixture("SKILLSYNC_GEMINI_PATH", "Gemini skills", ".gemini", "skills")
}

// GeminiConfigFixture creates a fixture for Gemini context and command artifacts.
func (h *Harness) GeminiConfigFixture() *Fixture {
	h.t.Helper()
	fixture := h.platformFixture("", "Gemini configuration", ".gemini")
	h.addDiscoveryPath("SKILLSYNC_GEMINI_SKILLS_PATHS", fixture.Path(""))
	return fixture
}

// TempFixture creates a fixture helper rooted at a new temporary directory.
func (h *Harness) TempFixture() *Fixture {
	h.t.Helper()
	tempDir := h.t.TempDir()
	return NewFixture(h.t, tempDir)
}
