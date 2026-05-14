package e2e_test

import (
	"testing"

	"github.com/klauern/skillsync/internal/e2e"
)

// TestSyncCopilotToCodex verifies sync from GitHub Copilot to Codex.
// A copilot prompt file (.prompt.md) syncs as a codex skill.
func TestSyncCopilotToCodex(t *testing.T) {
	h := e2e.NewHarness(t)

	copilotFixture := h.CopilotFixture()
	copilotFixture.WriteFile("prompts/review.prompt.md", "---\ndescription: Review code\n---\n\nReview the code at $ARGUMENTS.")

	codexFixture := h.CodexFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--include-prompts", "copilot", "codex")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Synced copilot -> codex using overwrite strategy")
	e2e.AssertOutputContains(t, result, "  Created:   1")
	e2e.AssertOutputContains(t, result, "  Failed:    0")
	e2e.AssertOutputContains(t, result, "  ✓ review: created")
	e2e.AssertFileExists(t, codexFixture.Path("review/SKILL.md"))
	e2e.AssertFileContains(t, codexFixture.Path("review/SKILL.md"), "Review the code at $ARGUMENTS.")
}

// TestSyncCodexToGemini verifies sync from Codex to Gemini CLI.
// A codex SKILL.md skill is synced as a directory-structured skill in gemini.
func TestSyncCodexToGemini(t *testing.T) {
	h := e2e.NewHarness(t)

	codexFixture := h.CodexFixture()
	codexFixture.WriteSkill("codex-gemini/SKILL.md", "codex-gemini", "Codex to Gemini", "# Codex Gemini\n\nShared content.")

	geminiFixture := h.GeminiFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "codex", "gemini")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	// Codex SKILL.md skills are synced as directories (directory-based source type)
	e2e.AssertFileExists(t, geminiFixture.Path("codex-gemini/SKILL.md"))
}

// TestSyncPiDevSystemMDToClaudeCode verifies that Pi.dev SYSTEM.md system-prompt
// artifacts are synced to Claude Code. SYSTEM.md lives in configRoot (parent of
// the skills basePath: ~/.pi/agent/ not ~/.pi/agent/skills/).
func TestSyncPiDevSystemMDToClaudeCode(t *testing.T) {
	h := e2e.NewHarness(t)

	piDevFixture := h.PiDevFixture()
	piDevFixture.WriteFile("../SYSTEM.md", "You are a helpful assistant focused on coding.")

	claudeFixture := h.ClaudeCodeFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "pi.dev", "claudecode")

	e2e.AssertSuccess(t, result)
	e2e.AssertFileExists(t, claudeFixture.Path("SYSTEM.md"))
}
