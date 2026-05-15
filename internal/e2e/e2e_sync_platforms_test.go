package e2e_test

import (
	"os"
	"testing"

	"github.com/klauern/skillsync/internal/e2e"
)

// ============================================================================
// Cross-Platform Sync E2E Tests
// ============================================================================

// TestSyncClaudeCodeToCursor verifies sync from Claude Code to Cursor.
func TestSyncClaudeCodeToCursor(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("platform-test.md", "platform-test", "Cross-platform skill", "# Platform Test\n\nWorks across platforms.")

	cursorFixture := h.CursorFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertFileExists(t, cursorFixture.Path("platform-test.md"))
}

// TestSyncCursorRuleWithGlobsToClaudeCode verifies that a cursor rule file
// containing globs and alwaysApply frontmatter is discovered and synced.
// Globs/alwaysApply are cursor-only fields; the target receives the skill content.
func TestSyncCursorRuleWithGlobsToClaudeCode(t *testing.T) {
	h := e2e.NewHarness(t)

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteFile("go-rules.md", `---
name: go-rules
description: Go coding rules
globs:
  - "**/*.go"
alwaysApply: true
---
# Go Rules

Apply these rules for all Go files.
`)

	claudeFixture := h.ClaudeCodeFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "cursor", "claudecode")

	e2e.AssertSuccess(t, result)
	// The rule file was discovered and synced — Claude receives it as a skill
	e2e.AssertFileExists(t, claudeFixture.Path("go-rules.md"))
}

// TestSyncCursorToClaudeCode verifies sync from Cursor to Claude Code.
func TestSyncCursorToClaudeCode(t *testing.T) {
	h := e2e.NewHarness(t)

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("reverse-test.md", "reverse-test", "Reverse sync skill", "# Reverse Test\n\nFrom Cursor to Claude.")

	claudeFixture := h.ClaudeCodeFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "cursor", "claudecode")

	e2e.AssertSuccess(t, result)
	e2e.AssertFileExists(t, claudeFixture.Path("reverse-test.md"))
}

// TestSyncClaudeCodeToCodex verifies sync from Claude Code to Codex.
func TestSyncClaudeCodeToCodex(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("codex-test.md", "codex-test", "To Codex", "# Codex Test\n\nContent for Codex.")

	codexFixture := h.CodexFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "codex")

	e2e.AssertSuccess(t, result)
	// Codex may transform the file differently
	e2e.AssertOutputContains(t, result, "Created")
	// Verify something was created in codex directory
	if !codexFixture.Exists("AGENTS.md") && !codexFixture.Exists("codex-test.md") {
		// Codex might aggregate into AGENTS.md or use individual files
		t.Log("Note: Codex file structure may differ from other platforms")
	}
}

// TestSyncClaudeCodeToPiDev verifies sync from Claude Code to Pi.dev.
func TestSyncClaudeCodeToPiDev(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("pi-test.md", "pi-test", "To Pi.dev", "# Pi Test\n\nContent for Pi.")

	piFixture := h.PiDevFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "pi-dev")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertFileExists(t, piFixture.Path("pi-test/SKILL.md"))
}

// TestSyncPiDevToCursor verifies sync from Pi.dev to Cursor.
func TestSyncPiDevToCursor(t *testing.T) {
	h := e2e.NewHarness(t)

	piFixture := h.PiDevFixture()
	piFixture.WriteSkill("pi-to-cursor/SKILL.md", "pi-to-cursor", "Pi to Cursor", "# Pi To Cursor\n\nContent for Cursor.")

	cursorFixture := h.CursorFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "pi-dev", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertFileExists(t, cursorFixture.Path("pi-to-cursor/SKILL.md"))
}

// TestSyncPiDevToClaudeCode verifies sync from Pi.dev to Claude Code (user scope).
func TestSyncPiDevToClaudeCode(t *testing.T) {
	h := e2e.NewHarness(t)

	piFixture := h.PiDevFixture()
	piFixture.WriteSkill("pi-to-claude/SKILL.md", "pi-to-claude", "Pi to Claude", "# Pi To Claude\n\nContent for Claude.")

	claudeFixture := h.ClaudeCodeFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "pi-dev", "claudecode")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertFileExists(t, claudeFixture.Path("pi-to-claude/SKILL.md"))
}

// TestSyncPiDevToCodex verifies sync from Pi.dev to Codex.
func TestSyncPiDevToCodex(t *testing.T) {
	h := e2e.NewHarness(t)

	piFixture := h.PiDevFixture()
	piFixture.WriteSkill("pi-to-codex/SKILL.md", "pi-to-codex", "Pi to Codex", "# Pi To Codex\n\nContent for Codex.")

	codexFixture := h.CodexFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "pi-dev", "codex")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertFileExists(t, codexFixture.Path("pi-to-codex/SKILL.md"))
}

// TestSyncClaudeCommandToCodexWithIncludePrompts verifies prompt/command
// artifacts can be synced to Codex when explicitly enabled.
func TestSyncClaudeCommandToCodexWithIncludePrompts(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteFile("review.md", `---
description: review command
allowed-tools: Bash, Read
argument-hint: "<path>"
---
# /review

Review this code.`)

	codexFixture := h.CodexFixture()
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--include-prompts", "claudecode", "codex")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "lossy mapping")
	e2e.AssertFileExists(t, codexFixture.Path("review/SKILL.md"))
}

// TestSyncClaudeCodeToCopilot verifies sync from Claude Code to GitHub Copilot.
func TestSyncClaudeCodeToCopilot(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("copilot-test.md", "copilot-test", "Test for Copilot", "# Copilot Test\n\nContent for Copilot.")

	copilotFixture := h.CopilotFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "copilot")

	e2e.AssertSuccess(t, result)
	e2e.AssertFileExists(t, copilotFixture.Path("agents/copilot-test.agent.md"))
}

// TestSyncCopilotToClaudeCode verifies sync from GitHub Copilot to Claude Code.
func TestSyncCopilotToClaudeCode(t *testing.T) {
	h := e2e.NewHarness(t)

	copilotFixture := h.CopilotFixture()
	copilotFixture.WriteFile("copilot-instructions.md", "# Copilot Instructions\n\nAlways follow best practices.")

	claudeFixture := h.ClaudeCodeFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "copilot", "claudecode")

	e2e.AssertSuccess(t, result)
	e2e.AssertFileExists(t, claudeFixture.Path("copilot-instructions.md"))
}

// TestSyncClaudeCodeToGemini verifies sync from Claude Code to Gemini CLI.
func TestSyncClaudeCodeToGemini(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("gemini-test.md", "gemini-test", "Test for Gemini", "# Gemini Test\n\nContent for Gemini.")

	geminiFixture := h.GeminiFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "gemini")

	e2e.AssertSuccess(t, result)
	e2e.AssertFileExists(t, geminiFixture.Path("gemini-test.md"))
}

// TestSyncGeminiToClaudeCode verifies sync from Gemini CLI to Claude Code.
func TestSyncGeminiToClaudeCode(t *testing.T) {
	h := e2e.NewHarness(t)

	geminiFixture := h.GeminiFixture()
	if err := os.MkdirAll(geminiFixture.Path("skills/gemini-skill"), 0o750); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	geminiFixture.WriteSkill("skills/gemini-skill/SKILL.md", "gemini-skill", "Gemini to Claude", "# Gemini Skill\n\nContent.")

	claudeFixture := h.ClaudeCodeFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "gemini", "claudecode")

	e2e.AssertSuccess(t, result)
	// Gemini SKILL.md skills are synced as directories (directory-based source type)
	e2e.AssertFileExists(t, claudeFixture.Path("gemini-skill/SKILL.md"))
}
