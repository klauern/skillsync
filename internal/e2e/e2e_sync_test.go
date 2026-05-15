package e2e_test

import (
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/e2e"
)

// TestSyncMissingArgs verifies sync command requires source and target.
func TestSyncMissingArgs(t *testing.T) {
	tests := map[string]struct {
		args []string
	}{
		"no arguments": {
			args: []string{"sync"},
		},
		"only source": {
			args: []string{"sync", "claudecode"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := e2e.NewHarness(t)

			result := h.Run(tt.args...)

			e2e.AssertError(t, result)
		})
	}
}

// TestSyncInvalidPlatform verifies sync command rejects invalid platforms.
func TestSyncInvalidPlatform(t *testing.T) {
	tests := map[string]struct {
		args []string
	}{
		"invalid source": {
			args: []string{"sync", "invalid", "cursor"},
		},
		"invalid target": {
			args: []string{"sync", "claudecode", "invalid"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := e2e.NewHarness(t)

			result := h.Run(tt.args...)

			e2e.AssertError(t, result)
		})
	}
}

// TestSyncDryRun verifies sync with dry-run flag doesn't modify files.
func TestSyncDryRun(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create a skill in Claude Code
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("test-skill.md", "test-skill", "A test skill", "# Test Skill\n\nThis is a test.")

	// Run sync with dry-run
	result := h.Run("sync", "--dry-run", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Dry run")
}

// TestSyncHelp verifies sync command help output includes all strategies.
func TestSyncHelp(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("sync", "--help")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "USAGE")
	e2e.AssertOutputContains(t, result, "--dry-run")
	e2e.AssertOutputContains(t, result, "--strategy")
	e2e.AssertOutputContains(t, result, "overwrite")
	e2e.AssertOutputContains(t, result, "skip")
	e2e.AssertOutputContains(t, result, "newer")
	e2e.AssertOutputContains(t, result, "merge")
	e2e.AssertOutputContains(t, result, "three-way")
	e2e.AssertOutputContains(t, result, "interactive")
}

// TestSyncInvalidStrategy verifies sync command rejects invalid strategies.
func TestSyncInvalidStrategy(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("sync", "--strategy", "invalid-strategy", "--yes", "--skip-validation", "claudecode", "cursor")

	e2e.AssertError(t, result)
	e2e.AssertErrorContains(t, result, "invalid strategy")
}

// TestSyncSamePlatform verifies sync fails when source and target are the same.
func TestSyncSamePlatform(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("sync", "--yes", "--skip-validation", "claudecode", "claudecode")

	e2e.AssertError(t, result)
}

// TestSyncCreatesNewSkill verifies sync creates a skill in empty target.
func TestSyncCreatesNewSkill(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create a skill in Claude Code source
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("new-skill.md", "new-skill", "A brand new skill", "# New Skill\n\nThis is a new skill content.")

	// Ensure Cursor target directory exists but is empty
	cursorFixture := h.CursorFixture()

	// Run sync with --yes to skip confirmation
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertOutputContains(t, result, "1")

	// Verify skill was created in target
	e2e.AssertFileExists(t, cursorFixture.Path("new-skill.md"))
	e2e.AssertFileContains(t, cursorFixture.Path("new-skill.md"), "new-skill")
}

// TestSyncMultipleSkills verifies sync handles multiple skills.
func TestSyncMultipleSkills(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create multiple skills in Claude Code
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("skill-one.md", "skill-one", "First skill", "# Skill One\n\nContent one.")
	claudeFixture.WriteSkill("skill-two.md", "skill-two", "Second skill", "# Skill Two\n\nContent two.")
	claudeFixture.WriteSkill("skill-three.md", "skill-three", "Third skill", "# Skill Three\n\nContent three.")

	// Ensure Cursor target exists
	cursorFixture := h.CursorFixture()

	// Run sync
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertOutputContains(t, result, "3")

	// Verify all skills were created
	e2e.AssertFileExists(t, cursorFixture.Path("skill-one.md"))
	e2e.AssertFileExists(t, cursorFixture.Path("skill-two.md"))
	e2e.AssertFileExists(t, cursorFixture.Path("skill-three.md"))
}

// TestSyncDryRunNoChanges verifies dry-run doesn't modify files.
func TestSyncDryRunNoChanges(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create a skill in Claude Code
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("dry-test.md", "dry-test", "Dry run test skill", "# Dry Test\n\nThis should not be copied.")

	// Create Cursor fixture
	cursorFixture := h.CursorFixture()

	// Run sync with dry-run
	result := h.Run("sync", "--dry-run", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Dry run")
	e2e.AssertOutputContains(t, result, "Created")

	// Verify skill was NOT created in target
	e2e.AssertFileNotExists(t, cursorFixture.Path("dry-test.md"))
}

func TestSyncDeleteModeDeletes(t *testing.T) {
	h := e2e.NewHarness(t)

	src := h.ClaudeCodeFixture()
	src.WriteSkill("del.md", "del", "", "# del")

	tgt := h.CursorFixture()
	tgt.WriteSkill("del.md", "del", "", "# del")

	result := h.RunWithStdin("y\n", "delete", "--skip-backup", "claudecode", "cursor")
	e2e.AssertSuccess(t, result)
	e2e.AssertFileNotExists(t, tgt.Path("del.md"))
}

func TestSyncDeleteModeDryRun(t *testing.T) {
	h := e2e.NewHarness(t)

	src := h.ClaudeCodeFixture()
	src.WriteSkill("del.md", "del", "", "# del")

	tgt := h.CursorFixture()
	tgt.WriteSkill("del.md", "del", "", "# del")

	result := h.Run("delete", "--dry-run", "--skip-backup", "claudecode", "cursor")
	e2e.AssertSuccess(t, result)
	e2e.AssertFileExists(t, tgt.Path("del.md"))
}

func TestSyncValidatesSourceSkills(t *testing.T) {
	h := e2e.NewHarness(t)

	src := h.ClaudeCodeFixture()
	src.WriteSkill("valid.md", "valid", "", "# valid")
	h.CursorFixture()

	result := h.RunWithStdin("y\n", "sync", "claudecode", "cursor")
	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Validating source skills")
}

// ============================================================================
// Resolution Strategy E2E Tests
// ============================================================================

// TestSyncOverwriteStrategy verifies overwrite strategy replaces existing skills.
func TestSyncOverwriteStrategy(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skill with new content
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("overwrite-test.md", "overwrite-test", "Updated description", "# Overwrite Test\n\nNew content from source.")

	// Create existing target skill with different content
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("overwrite-test.md", "overwrite-test", "Old description", "# Overwrite Test\n\nOld content in target.")

	// Run sync with overwrite strategy (default)
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "overwrite", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Updated")
	e2e.AssertOutputContains(t, result, "1")

	// Verify target was overwritten with source content
	e2e.AssertFileContains(t, cursorFixture.Path("overwrite-test.md"), "New content from source")
}

// TestSyncSkipStrategy verifies skip strategy preserves existing skills.
func TestSyncSkipStrategy(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skill
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("skip-test.md", "skip-test", "Source description", "# Skip Test\n\nNew content from source.")

	// Create existing target skill with different content
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("skip-test.md", "skip-test", "Target description", "# Skip Test\n\nOriginal content in target.")

	// Run sync with skip strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "skip", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Skipped")
	e2e.AssertOutputContains(t, result, "1")

	// Verify target was NOT overwritten - still has original content
	e2e.AssertFileContains(t, cursorFixture.Path("skip-test.md"), "Original content in target")
}

// TestSyncSkipStrategyWithNewSkill verifies skip strategy still creates new skills.
func TestSyncSkipStrategyWithNewSkill(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skills - one existing, one new
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("existing-skill.md", "existing-skill", "Existing", "# Existing\n\nExisting content.")
	claudeFixture.WriteSkill("brand-new-skill.md", "brand-new-skill", "New", "# Brand New\n\nNew content.")

	// Create only the existing skill in target
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("existing-skill.md", "existing-skill", "Target existing", "# Existing\n\nTarget's existing content.")

	// Run sync with skip strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "skip", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertOutputContains(t, result, "Skipped")

	// Verify existing skill was not modified
	e2e.AssertFileContains(t, cursorFixture.Path("existing-skill.md"), "Target's existing content")

	// Verify new skill was created
	e2e.AssertFileExists(t, cursorFixture.Path("brand-new-skill.md"))
	e2e.AssertFileContains(t, cursorFixture.Path("brand-new-skill.md"), "New content")
}

// TestSyncMergeStrategy verifies merge strategy combines content.
func TestSyncMergeStrategy(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skill
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("merge-test.md", "merge-test", "Source description", "# Merge Test\n\nSource content to merge.")

	// Create existing target skill with different content
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("merge-test.md", "merge-test", "Target description", "# Merge Test\n\nTarget content to keep.")

	// Run sync with merge strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "merge", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Merged")
	e2e.AssertOutputContains(t, result, "1")

	// Verify target contains merged content (both source and target content present)
	content := cursorFixture.ReadFile("merge-test.md")
	if !strings.Contains(content, "Source content to merge") || !strings.Contains(content, "Target content to keep") {
		t.Errorf("expected merged content to contain both source and target content\ngot: %s", content)
	}
}

// ============================================================================
// Edge Cases and Error Handling E2E Tests
// ============================================================================

// TestSyncEmptySource verifies sync handles empty source directory.
func TestSyncEmptySource(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create empty source directory
	h.ClaudeCodeFixture()
	h.CursorFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	// Should complete successfully with no changes
	e2e.AssertOutputContains(t, result, "0")
}

// TestSyncWithSpecialCharactersInName verifies sync handles special characters.
func TestSyncWithSpecialCharactersInName(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	// Use a skill name with special characters (but valid for filenames)
	claudeFixture.WriteSkill("my-special_skill.md", "my-special_skill", "Special chars", "# Special\n\nContent with special chars.")

	cursorFixture := h.CursorFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertFileExists(t, cursorFixture.Path("my-special_skill.md"))
}

// TestSyncPreservesSkillMetadata verifies metadata is preserved during sync.
func TestSyncPreservesSkillMetadata(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("metadata-skill.md", "metadata-skill", "Preserve this description", "# Metadata Skill\n\nContent to sync.")

	cursorFixture := h.CursorFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify metadata was preserved
	content := cursorFixture.ReadFile("metadata-skill.md")
	if !strings.Contains(content, "name: metadata-skill") {
		t.Errorf("expected name metadata to be preserved, got: %s", content)
	}
	if !strings.Contains(content, "description: Preserve this description") && !strings.Contains(content, "Preserve this description") {
		t.Errorf("expected description metadata to be preserved, got: %s", content)
	}
}

// TestSyncMixedActions verifies sync handles mixed create/update/skip correctly.
func TestSyncMixedActions(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create 3 skills in source
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("new-skill.md", "new-skill", "New", "# New\n\nNew content.")
	claudeFixture.WriteSkill("update-skill.md", "update-skill", "Update", "# Update\n\nUpdated content.")
	claudeFixture.WriteSkill("skip-skill.md", "skip-skill", "Skip", "# Skip\n\nShould skip this.")

	// Create some existing skills in target
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("update-skill.md", "update-skill", "Old", "# Update\n\nOld content.")
	cursorFixture.WriteSkill("skip-skill.md", "skip-skill", "Skip", "# Skip\n\nShould skip this.")

	// Run with skip strategy so we can see all three actions
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "skip", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Should have 1 created (new-skill), 2 skipped (update-skill, skip-skill)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertOutputContains(t, result, "Skipped")

	// Verify new skill was created
	e2e.AssertFileExists(t, cursorFixture.Path("new-skill.md"))

	// Verify skipped skills weren't modified
	e2e.AssertFileContains(t, cursorFixture.Path("update-skill.md"), "Old content")
}
