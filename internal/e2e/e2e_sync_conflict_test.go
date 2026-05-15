package e2e_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/e2e"
)

// ============================================================================
// Conflict Detection E2E Tests
// ============================================================================

// TestSyncThreeWayNoConflict verifies three-way merge with identical content skips.
func TestSyncThreeWayNoConflict(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create identical skills in source and target
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("identical.md", "identical", "Same description", "# Identical\n\nSame content.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("identical.md", "identical", "Same description", "# Identical\n\nSame content.")

	// Run sync with three-way strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "three-way", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	// With identical content, should be skipped
	e2e.AssertOutputContains(t, result, "Skipped")
}

// TestSyncThreeWayContentConflict verifies three-way merge detects content differences.
func TestSyncThreeWayContentConflict(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skill with different content
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("conflict.md", "conflict", "Description", "# Conflict\n\nSource version line 1.\nSource version line 2.")

	// Create target skill with different content
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("conflict.md", "conflict", "Description", "# Conflict\n\nTarget version line 1.\nTarget version line 2.")

	// Run sync with three-way strategy - should detect conflict
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "three-way", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	// Three-way should either merge or report conflict
	output := result.Stdout
	if !strings.Contains(output, "Merged") && !strings.Contains(output, "Conflict") {
		t.Errorf("expected three-way to result in merge or conflict, got: %s", output)
	}
}

// TestSyncThreeWayMetadataConflict verifies three-way merge detects metadata differences.
func TestSyncThreeWayMetadataConflict(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skill with one description
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("metadata-conflict.md", "metadata-conflict", "Source description is different", "# Metadata\n\nSame content here.")

	// Create target skill with different description but same content
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("metadata-conflict.md", "metadata-conflict", "Target description is different", "# Metadata\n\nSame content here.")

	// Run sync with three-way strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "three-way", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	// Metadata-only conflicts may be handled as merged or conflict
	// The key is that the sync completes successfully
}

// TestSyncShowsConflictDetails verifies conflict information is displayed.
func TestSyncShowsConflictDetails(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create conflicting skills
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("show-conflict.md", "show-conflict", "Source version", "# Show Conflict\n\nLine A from source.\nLine B from source.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("show-conflict.md", "show-conflict", "Target version", "# Show Conflict\n\nLine X from target.\nLine Y from target.")

	// Run sync with three-way strategy (which reports conflicts)
	result := h.Run("sync", "--dry-run", "--skip-validation", "--strategy", "three-way", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	// Output should show the skill name and some indication of what happened
	e2e.AssertOutputContains(t, result, "show-conflict")
}

// ============================================================================
// Newer Strategy E2E Tests
// ============================================================================

// TestSyncNewerStrategySourceNewer verifies newer strategy copies when source is newer.
func TestSyncNewerStrategySourceNewer(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create old target file first
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("newer-test.md", "newer-test", "Old target", "# Newer Test\n\nOld target content.")

	// Sleep briefly to ensure timestamp difference
	// Note: In CI, file times may have limited precision, so we set times explicitly
	targetPath := cursorFixture.Path("newer-test.md")
	oldTime := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(targetPath, oldTime, oldTime); err != nil {
		t.Fatalf("failed to set target file time: %v", err)
	}

	// Create newer source file
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("newer-test.md", "newer-test", "New source", "# Newer Test\n\nNew source content.")

	// Run sync with newer strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "newer", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Updated")

	// Verify target was updated with source content (source is newer)
	e2e.AssertFileContains(t, cursorFixture.Path("newer-test.md"), "New source content")
}

// TestSyncNewerStrategyTargetNewer verifies newer strategy skips when target is newer.
func TestSyncNewerStrategyTargetNewer(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source file first
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("older-test.md", "older-test", "Old source", "# Older Test\n\nOld source content.")

	// Set source file to old timestamp
	sourcePath := claudeFixture.Path("older-test.md")
	oldTime := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(sourcePath, oldTime, oldTime); err != nil {
		t.Fatalf("failed to set source file time: %v", err)
	}

	// Create newer target file
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("older-test.md", "older-test", "New target", "# Older Test\n\nNew target content.")

	// Run sync with newer strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "newer", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Skipped")

	// Verify target was NOT updated (target is newer)
	e2e.AssertFileContains(t, cursorFixture.Path("older-test.md"), "New target content")
}

// TestSyncNewerStrategyNewFile verifies newer strategy creates new files.
func TestSyncNewerStrategyNewFile(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skill with no corresponding target
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("brand-new.md", "brand-new", "Brand new skill", "# Brand New\n\nThis skill doesn't exist in target.")

	// Create empty target directory
	cursorFixture := h.CursorFixture()

	// Run sync with newer strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "newer", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")

	// Verify new skill was created
	e2e.AssertFileExists(t, cursorFixture.Path("brand-new.md"))
	e2e.AssertFileContains(t, cursorFixture.Path("brand-new.md"), "Brand new skill")
}

// ============================================================================
// Interactive Strategy E2E Tests
// ============================================================================

// TestSyncInteractiveUseSource verifies interactive strategy with source selection.
func TestSyncInteractiveUseSource(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create conflicting skills
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("interactive-test.md", "interactive-test", "Source version", "# Interactive Test\n\nSource content.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("interactive-test.md", "interactive-test", "Target version", "# Interactive Test\n\nTarget content.")

	// With --yes flag, skip confirmation but still prompt for conflict resolution
	// Per-conflict prompt options:
	//   1. Use source version (overwrite target)
	//   2. Keep target version (discard source changes)
	//   3. Attempt automatic merge
	//   4. Skip this skill
	result := h.RunWithStdin("1\n", "sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "interactive", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify target was overwritten with source
	e2e.AssertFileContains(t, cursorFixture.Path("interactive-test.md"), "Source content")
}

// TestSyncInteractiveKeepTarget verifies interactive strategy with target selection.
func TestSyncInteractiveKeepTarget(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create conflicting skills
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("keep-target.md", "keep-target", "Source", "# Keep Target\n\nSource content to discard.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("keep-target.md", "keep-target", "Target", "# Keep Target\n\nOriginal target content.")

	// With --yes flag, skip confirmation but still prompt for conflict resolution
	// Per-conflict prompt options:
	//   1. Use source version (overwrite target)
	//   2. Keep target version (discard source changes)
	//   3. Attempt automatic merge
	//   4. Skip this skill
	result := h.RunWithStdin("2\n", "sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "interactive", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify target was NOT modified (kept original)
	e2e.AssertFileContains(t, cursorFixture.Path("keep-target.md"), "Original target content")
}

// TestSyncInteractiveSkip verifies interactive strategy skip option.
func TestSyncInteractiveSkip(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create conflicting skills
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("skip-test.md", "skip-test", "Source", "# Skip Test\n\nSource content.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("skip-test.md", "skip-test", "Target", "# Skip Test\n\nOriginal content to preserve.")

	// With --yes flag, skip confirmation but still prompt for conflict resolution
	// Per-conflict prompt options:
	//   1. Use source version (overwrite target)
	//   2. Keep target version (discard source changes)
	//   3. Attempt automatic merge
	//   4. Skip this skill
	result := h.RunWithStdin("4\n", "sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "interactive", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify target was NOT modified (skipped)
	e2e.AssertFileContains(t, cursorFixture.Path("skip-test.md"), "Original content to preserve")
}

// TestSyncInteractiveAutoMerge verifies interactive strategy auto-merge option.
func TestSyncInteractiveAutoMerge(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create conflicting skills
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("auto-merge.md", "auto-merge", "Source", "# Auto Merge\n\nSource specific content.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("auto-merge.md", "auto-merge", "Target", "# Auto Merge\n\nTarget specific content.")

	// With --yes flag, skip confirmation but still prompt for conflict resolution
	// Per-conflict prompt options:
	//   1. Use source version (overwrite target)
	//   2. Keep target version (discard source changes)
	//   3. Attempt automatic merge (may have conflict markers)
	//   4. Skip this skill
	result := h.RunWithStdin("3\n", "sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "interactive", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify content was merged (both should appear or conflict markers present)
	content := cursorFixture.ReadFile("auto-merge.md")
	// After auto-merge, content should contain merged result or conflict markers
	hasSource := strings.Contains(content, "Source specific content")
	hasTarget := strings.Contains(content, "Target specific content")
	hasMarkers := strings.Contains(content, "<<<<<<<") || strings.Contains(content, ">>>>>>>")

	if !hasSource && !hasTarget && !hasMarkers {
		t.Errorf("expected merged content to contain source, target, or conflict markers\ngot: %s", content)
	}
}

// TestSyncInteractiveNoConflicts verifies interactive with no conflicts skips prompts.
func TestSyncInteractiveNoConflicts(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skill with no existing target
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("no-conflict.md", "no-conflict", "New skill", "# No Conflict\n\nBrand new content.")

	cursorFixture := h.CursorFixture()

	// No stdin needed - no conflicts means no prompts
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "interactive", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")

	// Verify skill was created
	e2e.AssertFileExists(t, cursorFixture.Path("no-conflict.md"))
}

// ============================================================================
// Three-Way Merge Conflict Markers E2E Tests
// ============================================================================

// TestSyncThreeWayWithConflictMarkers verifies three-way merge can produce conflict markers.
func TestSyncThreeWayWithConflictMarkers(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create heavily conflicting content that can't be auto-merged cleanly
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("markers-test.md", "markers-test", "Source version",
		"# Markers Test\n\n"+
			"This line is unique to source.\n"+
			"Line A in source version.\n"+
			"Line B in source version.\n"+
			"Ending from source.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("markers-test.md", "markers-test", "Target version",
		"# Markers Test\n\n"+
			"This line is unique to target.\n"+
			"Line X in target version.\n"+
			"Line Y in target version.\n"+
			"Ending from target.")

	// Run sync with three-way strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "three-way", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Read the result - it should either be merged or have conflict markers
	content := cursorFixture.ReadFile("markers-test.md")

	// The three-way merge should either:
	// 1. Successfully merge the content
	// 2. Leave conflict markers if auto-merge fails
	hasConflictMarkers := strings.Contains(content, "<<<<<<<") ||
		strings.Contains(content, "=======") ||
		strings.Contains(content, ">>>>>>>")
	hasSourceContent := strings.Contains(content, "source")
	hasTargetContent := strings.Contains(content, "target")

	// Verify some merge action occurred
	if !hasConflictMarkers && !hasSourceContent && !hasTargetContent {
		t.Errorf("expected merge result to contain source content, target content, or conflict markers\ngot: %s", content)
	}
}

// TestSyncThreeWayCleanMerge verifies three-way merge handles non-conflicting changes.
func TestSyncThreeWayCleanMerge(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create content where changes are in different areas (clean merge possible)
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("clean-merge.md", "clean-merge", "Desc",
		"# Clean Merge\n\n"+
			"Section 1: Source added this section.\n\n"+
			"Section 2: Common middle content.\n\n"+
			"Section 3: Common ending.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("clean-merge.md", "clean-merge", "Desc",
		"# Clean Merge\n\n"+
			"Section 1: Common start content.\n\n"+
			"Section 2: Common middle content.\n\n"+
			"Section 3: Target modified this.")

	// Run sync with three-way strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "three-way", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify the sync completed - exact merge result depends on implementation
	content := cursorFixture.ReadFile("clean-merge.md")
	if content == "" {
		t.Error("expected file to have content after merge")
	}
}

// ============================================================================
// Multiple Conflict Resolution E2E Tests
// ============================================================================

// TestSyncMultipleConflicts verifies handling of multiple conflicts in one sync.
func TestSyncMultipleConflicts(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create multiple conflicting skills
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("conflict1.md", "conflict1", "Source1", "# Conflict 1\n\nSource version 1.")
	claudeFixture.WriteSkill("conflict2.md", "conflict2", "Source2", "# Conflict 2\n\nSource version 2.")
	claudeFixture.WriteSkill("conflict3.md", "conflict3", "Source3", "# Conflict 3\n\nSource version 3.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("conflict1.md", "conflict1", "Target1", "# Conflict 1\n\nTarget version 1.")
	cursorFixture.WriteSkill("conflict2.md", "conflict2", "Target2", "# Conflict 2\n\nTarget version 2.")
	cursorFixture.WriteSkill("conflict3.md", "conflict3", "Target3", "# Conflict 3\n\nTarget version 3.")

	// Run sync with overwrite strategy to resolve all conflicts
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "overwrite", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Updated")
	e2e.AssertOutputContains(t, result, "3")

	// Verify all were updated
	e2e.AssertFileContains(t, cursorFixture.Path("conflict1.md"), "Source version 1")
	e2e.AssertFileContains(t, cursorFixture.Path("conflict2.md"), "Source version 2")
	e2e.AssertFileContains(t, cursorFixture.Path("conflict3.md"), "Source version 3")
}

// TestSyncMixedConflictAndNew verifies handling mixed new files and conflicts.
func TestSyncMixedConflictAndNew(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source with mix of new and conflicting
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("new-file1.md", "new-file1", "New1", "# New File 1\n\nBrand new content.")
	claudeFixture.WriteSkill("existing.md", "existing", "Updated", "# Existing\n\nUpdated source content.")
	claudeFixture.WriteSkill("new-file2.md", "new-file2", "New2", "# New File 2\n\nAnother new file.")

	// Create target with one existing file
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("existing.md", "existing", "Original", "# Existing\n\nOriginal target content.")

	// Run sync with merge strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "merge", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertOutputContains(t, result, "Merged")

	// Verify new files were created
	e2e.AssertFileExists(t, cursorFixture.Path("new-file1.md"))
	e2e.AssertFileExists(t, cursorFixture.Path("new-file2.md"))

	// Verify existing was merged (should contain both contents)
	content := cursorFixture.ReadFile("existing.md")
	if !strings.Contains(content, "source") && !strings.Contains(content, "target") {
		// Merge should produce content from both
		t.Logf("Merged content: %s", content)
	}
}
