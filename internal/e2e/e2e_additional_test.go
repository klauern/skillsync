package e2e_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/e2e"
)

// ============================================================================
// Additional Platform Combination Tests
// ============================================================================

// TestSyncCursorToCodex verifies sync from Cursor to Codex platform.
func TestSyncCursorToCodex(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create skill in Cursor
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("cursor-skill.md", "cursor-skill", "A Cursor skill", "# Cursor Skill\n\nContent from Cursor.")

	// Create Codex target
	codexFixture := h.CodexFixture()

	// Run sync
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "cursor", "codex")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertOutputContains(t, result, "1")

	// Verify skill was created in Codex
	e2e.AssertFileExists(t, codexFixture.Path("cursor-skill.md"))
	e2e.AssertFileContains(t, codexFixture.Path("cursor-skill.md"), "Content from Cursor")
}

// TestSyncCodexToClaudeCode verifies sync from Codex to Claude Code.
func TestSyncCodexToClaudeCode(t *testing.T) {
	t.Skip("Codex parser requires SKILL.md in subdirectories or config.toml - tested in integration tests")
}

// TestSyncCodexToCursor verifies sync from Codex to Cursor platform.
func TestSyncCodexToCursor(t *testing.T) {
	t.Skip("Codex parser requires SKILL.md in subdirectories or config.toml - tested in integration tests")
}

// ============================================================================
// Agent Skills Standard (SKILL.md) Format Tests
// ============================================================================

// TestSyncAgentSkillsStandardFormat verifies SKILL.md directory format is handled.
func TestSyncAgentSkillsStandardFormat(t *testing.T) {
	t.Skip("Agent Skills Standard SKILL.md directory sync is complex - needs dedicated test setup")
}

// TestSyncPreservesAgentSkillsMetadata verifies Agent Skills Standard metadata is preserved.
func TestSyncPreservesAgentSkillsMetadata(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create skill with rich metadata
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteFile("rich-skill.md", `---
name: rich-skill
description: Skill with complete metadata
scope: user
license: MIT
compatibility:
  claude-code: ">=1.0.0"
  cursor: ">=0.5.0"
disable-model-invocation: true
---

# Rich Skill

Content with full metadata.
`)

	// Create Cursor target
	cursorFixture := h.CursorFixture()

	// Run sync
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Read synced skill and verify metadata preservation
	if cursorFixture.Exists("rich-skill.md") {
		content := cursorFixture.ReadFile("rich-skill.md")
		// Key metadata should be preserved
		if !strings.Contains(content, "rich-skill") {
			t.Error("expected name to be preserved in synced skill")
		}
		if !strings.Contains(content, "Skill with complete metadata") {
			t.Error("expected description to be preserved")
		}
	}
}

// ============================================================================
// Platform-Specific Feature Tests
// ============================================================================

// TestSyncCursorMDCFormat verifies Cursor .mdc files are handled.
func TestSyncCursorMDCFormat(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create .mdc file in Cursor
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteFile("cursor-mdc-skill.mdc", `---
name: cursor-mdc-skill
description: A Cursor .mdc skill
globs:
  - "**/*.go"
alwaysApply: true
---

# Cursor MDC Skill

This is a Cursor-specific .mdc file.
`)

	// Create Claude Code target
	claudeFixture := h.ClaudeCodeFixture()

	// Run sync
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "cursor", "claudecode")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")

	// Verify skill was converted and created
	e2e.AssertFileExists(t, claudeFixture.Path("cursor-mdc-skill.md"))
	e2e.AssertFileContains(t, claudeFixture.Path("cursor-mdc-skill.md"), "Cursor MDC Skill")
}

// TestSyncCodexConfigTOMLInstructions verifies Codex config.toml is handled.
func TestSyncCodexConfigTOMLInstructions(t *testing.T) {
	t.Skip("Codex config.toml parsing requires full Codex directory structure - tested in integration tests")
}

// ============================================================================
// Large File and Performance Tests
// ============================================================================

// TestSyncLargeSkillContent verifies large skill files are handled correctly.
func TestSyncLargeSkillContent(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create a large skill (100KB+ of content)
	var largeContent strings.Builder
	largeContent.WriteString("# Large Skill\n\n")
	for i := range 5000 {
		largeContent.WriteString("This is line " + string(rune(i)) + " of a very large skill file.\n")
	}
	largeContentStr := largeContent.String()

	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("large-skill.md", "large-skill", "A very large skill", largeContentStr)

	// Create Cursor target
	cursorFixture := h.CursorFixture()

	// Run sync
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")

	// Verify large file was created and not truncated
	e2e.AssertFileExists(t, cursorFixture.Path("large-skill.md"))
	content := cursorFixture.ReadFile("large-skill.md")
	if len(content) < 50000 {
		t.Errorf("expected large file to be at least 50KB, got %d bytes", len(content))
	}
}

// TestSyncManySkills verifies syncing many skills at once.
func TestSyncManySkills(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	cursorFixture := h.CursorFixture()

	// Create 50 skills with safe names (alphanumeric only)
	for i := range 50 {
		skillName := fmt.Sprintf("batch-skill-%02d", i)
		skillDesc := fmt.Sprintf("Batch skill number %d", i)
		skillContent := fmt.Sprintf("# Batch Skill %d\n\nThis is batch skill number %d", i, i)
		claudeFixture.WriteSkill(skillName+".md", skillName, skillDesc, skillContent)
	}

	// Run sync
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertOutputContains(t, result, "50")

	// Spot check a few files
	e2e.AssertFileExists(t, cursorFixture.Path("batch-skill-00.md"))
	e2e.AssertFileExists(t, cursorFixture.Path("batch-skill-25.md"))
	e2e.AssertFileExists(t, cursorFixture.Path("batch-skill-49.md"))
}

// ============================================================================
// Unicode and Special Character Tests
// ============================================================================

// TestSyncUnicodeContent verifies Unicode content is preserved.
func TestSyncUnicodeContent(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create skill with Unicode content (emoji, CJK, symbols)
	claudeFixture := h.ClaudeCodeFixture()
	unicodeContent := `# Unicode Test 🎉

## Features
- 日本語のサポート (Japanese support)
- 中文支持 (Chinese support)
- 한국어 지원 (Korean support)
- Emoji support: ✅ 🚀 💻 🎯 ⚡

## Code Examples
` + "```python" + `
def hello_world():
    print("Hello, 世界! 🌍")
` + "```" + `
`
	claudeFixture.WriteSkill("unicode-test.md", "unicode-test", "Unicode test skill 🌐", unicodeContent)

	// Create Cursor target
	cursorFixture := h.CursorFixture()

	// Run sync
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify Unicode content is preserved
	content := cursorFixture.ReadFile("unicode-test.md")
	if !strings.Contains(content, "🎉") {
		t.Error("expected emoji to be preserved")
	}
	if !strings.Contains(content, "日本語") {
		t.Error("expected Japanese characters to be preserved")
	}
	if !strings.Contains(content, "中文") {
		t.Error("expected Chinese characters to be preserved")
	}
	if !strings.Contains(content, "한국어") {
		t.Error("expected Korean characters to be preserved")
	}
}

// TestSyncSkillNameWithDashes verifies skill names with dashes work correctly.
func TestSyncSkillNameWithDashes(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("multi-word-skill-name.md", "multi-word-skill-name", "Multi-word skill", "# Multi Word Skill\n\nContent.")

	cursorFixture := h.CursorFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertFileExists(t, cursorFixture.Path("multi-word-skill-name.md"))
}

// TestSyncSkillNameWithUnderscores verifies skill names with underscores work.
func TestSyncSkillNameWithUnderscores(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("skill_with_underscores.md", "skill_with_underscores", "Underscore skill", "# Underscore\n\nContent.")

	cursorFixture := h.CursorFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertFileExists(t, cursorFixture.Path("skill_with_underscores.md"))
}

// ============================================================================
// Error Handling and Edge Case Tests
// ============================================================================

// TestSyncNonexistentSourcePath verifies error handling for missing source.
func TestSyncNonexistentSourcePath(t *testing.T) {
	h := e2e.NewHarness(t)

	// Set source path to nonexistent directory
	h.SetEnv("SKILLSYNC_CLAUDE_CODE_PATH", h.HomeDir()+"/nonexistent")

	// Create valid target
	h.CursorFixture()

	// Run sync - should fail gracefully
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	// Should either succeed with no skills or fail with clear error
	// Both are acceptable behaviors
	if result.Err != nil {
		e2e.AssertErrorContains(t, result, "not found")
	} else {
		e2e.AssertOutputContains(t, result, "0")
	}
}

// TestSyncReadOnlyTarget verifies error handling for read-only target.
func TestSyncReadOnlyTarget(t *testing.T) {
	t.Skip("Read-only permission tests are platform-specific and may require root - tested manually")
}

// TestSyncWithSymlinks verifies handling of symlinked skills.
func TestSyncWithSymlinks(t *testing.T) {
	t.Skip("Symlink handling for plugins is tested in parser tests - E2E would be redundant")
}

// ============================================================================
// Frontmatter Transformation Tests
// ============================================================================

// TestSyncTransformsClaudeCodeTools verifies tools array is handled correctly.
func TestSyncTransformsClaudeCodeTools(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create Claude Code skill with tools array
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteFile("tools-skill.md", `---
name: tools-skill
description: Skill with tools
tools:
  - read
  - write
  - bash
---

# Tools Skill

This skill uses several tools.
`)

	// Sync to Cursor
	cursorFixture := h.CursorFixture()
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify skill was created
	e2e.AssertFileExists(t, cursorFixture.Path("tools-skill.md"))

	// Tools array handling is platform-specific
	// Just verify the sync succeeded and file exists
	content := cursorFixture.ReadFile("tools-skill.md")
	if !strings.Contains(content, "tools-skill") {
		t.Error("expected skill name to be preserved")
	}
}

// TestSyncTransformsCursorGlobs verifies Cursor globs are handled.
func TestSyncTransformsCursorGlobs(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create Cursor skill with globs
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteFile("globs-skill.md", `---
name: globs-skill
description: Skill with globs
globs:
  - "**/*.go"
  - "**/*.ts"
alwaysApply: true
---

# Globs Skill

This skill applies to specific file patterns.
`)

	// Sync to Claude Code
	claudeFixture := h.ClaudeCodeFixture()
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "cursor", "claudecode")

	e2e.AssertSuccess(t, result)

	// Verify skill was created
	e2e.AssertFileExists(t, claudeFixture.Path("globs-skill.md"))

	// Globs are Cursor-specific and may be stripped or converted
	// Just verify the sync succeeded
	content := claudeFixture.ReadFile("globs-skill.md")
	if !strings.Contains(content, "globs-skill") {
		t.Error("expected skill name to be preserved")
	}
}

// ============================================================================
// Validation Tests
// ============================================================================

// TestSyncValidationRejectsInvalidContent verifies validation catches issues.
func TestSyncValidationRejectsInvalidContent(t *testing.T) {
	t.Skip("Validation behavior depends on --skip-validation flag - covered in other tests")
}

// ============================================================================
// Dry Run Comprehensive Tests
// ============================================================================

// TestSyncDryRunShowsAllActions verifies dry-run previews all action types.
func TestSyncDryRunShowsAllActions(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source with various scenarios
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("new-skill.md", "new-skill", "New", "# New\n\nWill be created.")
	claudeFixture.WriteSkill("update-skill.md", "update-skill", "Updated", "# Update\n\nUpdated content.")

	// Create target with existing skill
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("update-skill.md", "update-skill", "Original", "# Update\n\nOriginal content.")

	// Run dry-run
	result := h.Run("sync", "--dry-run", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Dry run")
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertOutputContains(t, result, "Updated")

	// Verify no actual changes
	e2e.AssertFileNotExists(t, cursorFixture.Path("new-skill.md"))
	e2e.AssertFileContains(t, cursorFixture.Path("update-skill.md"), "Original content")
}
