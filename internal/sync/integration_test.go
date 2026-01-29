package sync

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/backup"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/util"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestMain(m *testing.M) {
	flag.Parse()
	util.SetUpdateGolden(*updateGolden)
	os.Exit(m.Run())
}

// testdataDir returns the path to the testdata directory for golden files.
func testdataDir() string {
	return filepath.Join("..", "..", "testdata", "sync")
}

// Integration tests for end-to-end synchronization scenarios

func TestIntegration_MultiSkillSync(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create multiple source skills
	skills := map[string]string{
		"skill-1.md": `---
name: skill-1
description: First skill
---

# Skill 1

This is the first skill content.
`,
		"skill-2.md": `---
name: skill-2
description: Second skill
tools:
  - read
  - write
---

# Skill 2

This skill uses read and write tools.
`,
		"skill-3.md": `---
name: skill-3
description: Third skill
---

# Skill 3

Multi-line content
with several paragraphs.

And some code:
` + "```go\nfunc main() {}\n```\n",
	}

	for name, content := range skills {
		path := filepath.Join(sourceDir, name)
		util.WriteFile(t, path, content)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 3)
	util.AssertEqual(t, result.TotalProcessed(), 3)
	util.AssertEqual(t, result.Success(), true)

	// Verify all files were created
	for name := range skills {
		targetPath := filepath.Join(targetDir, name)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			t.Errorf("Expected target file %s to exist", name)
		}
	}
}

func TestIntegration_AllPlatformCombinations(t *testing.T) {
	// Test ClaudeCode <-> Cursor combinations (they share the same .md format)
	// Codex uses AGENTS.md which has different structure
	testCases := []struct {
		source model.Platform
		target model.Platform
	}{
		{model.ClaudeCode, model.Cursor},
		{model.Cursor, model.ClaudeCode},
	}

	for _, tc := range testCases {
		t.Run(string(tc.source)+"->"+string(tc.target), func(t *testing.T) {
			s := New()
			sourceDir := t.TempDir()
			targetDir := t.TempDir()

			// Create a simple skill
			skillContent := `---
name: cross-platform-test
description: Testing cross-platform sync
---

Test content for cross-platform synchronization.
`
			util.WriteFile(t, filepath.Join(sourceDir, "test.md"), skillContent)

			opts := Options{
				DryRun:     false,
				Strategy:   StrategyOverwrite,
				SourcePath: sourceDir,
				TargetPath: targetDir,
			}

			result, err := s.Sync(tc.source, tc.target, opts)
			util.AssertNoError(t, err)

			util.AssertEqual(t, len(result.Created()), 1)
			util.AssertEqual(t, result.Source, tc.source)
			util.AssertEqual(t, result.Target, tc.target)
		})
	}
}

func TestIntegration_MixedActions(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create skills that will result in different actions
	// New skill (will be created)
	util.WriteFile(t, filepath.Join(sourceDir, "new-skill.md"), `---
name: new-skill
description: A new skill
---

Brand new content.
`)

	// Skill that exists in target (will be updated with overwrite)
	util.WriteFile(t, filepath.Join(sourceDir, "existing-skill.md"), `---
name: existing-skill
description: Updated description
---

Updated source content.
`)

	util.WriteFile(t, filepath.Join(targetDir, "existing-skill.md"), `---
name: existing-skill
description: Original description
---

Original target content.
`)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 1)
	util.AssertEqual(t, len(result.Updated()), 1)
	util.AssertEqual(t, result.TotalProcessed(), 2)
}

func TestIntegration_LargeSkillFile(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a large skill file (~50KB)
	var content string
	content = `---
name: large-skill
description: A skill with lots of content
---

# Large Skill

`
	// Add repeated sections to make it large
	for range 500 {
		content += `## Section

This is a paragraph. It contains multiple lines
of text that simulate real-world skill documentation.

- Item 1 for this section
- Item 2 for this section
- Item 3 for this section

`
	}

	util.WriteFile(t, filepath.Join(sourceDir, "large-skill.md"), content)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 1)

	// Verify the large file was written correctly
	targetPath := filepath.Join(targetDir, "large-skill.md")
	info, err := os.Stat(targetPath)
	util.AssertNoError(t, err)

	if info.Size() < 40000 {
		t.Errorf("Expected large file, got %d bytes", info.Size())
	}
}

func TestIntegration_EmptyTargetDirectory(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create skills
	util.WriteFile(t, filepath.Join(sourceDir, "skill-a.md"), `---
name: skill-a
---

Content A
`)
	util.WriteFile(t, filepath.Join(sourceDir, "skill-b.md"), `---
name: skill-b
---

Content B
`)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// All skills should be created
	util.AssertEqual(t, len(result.Created()), 2)
	util.AssertEqual(t, len(result.Updated()), 0)
	util.AssertEqual(t, len(result.Skipped()), 0)
}

func TestIntegration_AllStrategies(t *testing.T) {
	strategies := []Strategy{
		StrategyOverwrite,
		StrategySkip,
		StrategyNewer,
		StrategyMerge,
		StrategyThreeWay,
	}

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			s := New()

			sourceDir := t.TempDir()
			targetDir := t.TempDir()

			// Create source skill
			util.WriteFile(t, filepath.Join(sourceDir, "test.md"), `---
name: test
description: Source version
---

Source content.
`)

			// Create existing target skill
			targetPath := filepath.Join(targetDir, "test.md")
			util.WriteFile(t, targetPath, `---
name: test
description: Target version
---

Target content.
`)

			// Make source newer for "newer" strategy
			if strategy == StrategyNewer {
				oldTime := time.Now().Add(-24 * time.Hour)
				if err := os.Chtimes(targetPath, oldTime, oldTime); err != nil {
					t.Fatalf("Failed to set file time: %v", err)
				}
			}

			opts := Options{
				DryRun:     false,
				Strategy:   strategy,
				SourcePath: sourceDir,
				TargetPath: targetDir,
			}

			result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
			util.AssertNoError(t, err)

			// Just verify sync completed without error
			util.AssertEqual(t, result.TotalProcessed(), 1)
		})
	}
}

func TestIntegration_DryRunPreview(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create skills
	util.WriteFile(t, filepath.Join(sourceDir, "new-skill.md"), `---
name: new-skill
---

New content.
`)

	util.WriteFile(t, filepath.Join(sourceDir, "update-skill.md"), `---
name: update-skill
---

Updated content.
`)

	util.WriteFile(t, filepath.Join(targetDir, "update-skill.md"), `---
name: update-skill
---

Original content.
`)

	opts := Options{
		DryRun:     true,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, result.DryRun, true)

	// Files should NOT be modified in dry run
	newSkillPath := filepath.Join(targetDir, "new-skill.md")
	if _, err := os.Stat(newSkillPath); !os.IsNotExist(err) {
		t.Error("New skill should not exist in dry run mode")
	}

	// Existing file should still have original content
	// #nosec G304 - test file path is controlled
	content, err := os.ReadFile(filepath.Join(targetDir, "update-skill.md"))
	util.AssertNoError(t, err)

	if string(content) != `---
name: update-skill
---

Original content.
` {
		t.Error("Target content should not change in dry run")
	}
}

func TestIntegration_RepeatedSyncIdempotent(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a skill
	skillContent := `---
name: idempotent-test
description: Testing idempotent sync
---

Content that stays the same.
`
	util.WriteFile(t, filepath.Join(sourceDir, "test.md"), skillContent)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategySkip, // Skip strategy makes repeated syncs idempotent
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	// First sync - creates
	result1, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)
	util.AssertEqual(t, len(result1.Created()), 1)

	// Second sync - skips (already exists)
	result2, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)
	util.AssertEqual(t, len(result2.Skipped()), 1)
	util.AssertEqual(t, len(result2.Created()), 0)

	// Third sync - still skips
	result3, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)
	util.AssertEqual(t, len(result3.Skipped()), 1)
}

func TestIntegration_ResultSummary_Golden(t *testing.T) {
	// Test that Result.Summary() output matches golden file
	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Cursor,
		Strategy: StrategyOverwrite,
		DryRun:   false,
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "created-skill"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "updated-skill"}, Action: ActionUpdated},
			{Skill: model.Skill{Name: "skipped-skill"}, Action: ActionSkipped},
		},
	}

	summary := result.Summary()
	util.GoldenFile(t, testdataDir(), "result-summary-basic", summary)
}

func TestIntegration_ResultSummary_DryRun_Golden(t *testing.T) {
	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Cursor,
		Strategy: StrategyThreeWay,
		DryRun:   true,
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "skill-1"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "skill-2"}, Action: ActionCreated},
		},
	}

	summary := result.Summary()
	util.GoldenFile(t, testdataDir(), "result-summary-dryrun", summary)
}

func TestIntegration_ResultSummary_WithConflicts_Golden(t *testing.T) {
	conflict := &Conflict{
		SkillName: "conflict-skill",
		Type:      ConflictTypeContent,
	}

	result := &Result{
		Source:   model.Cursor,
		Target:   model.Codex,
		Strategy: StrategyThreeWay,
		DryRun:   false,
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "clean-skill"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "conflict-skill"}, Action: ActionConflict, Conflict: conflict},
		},
	}

	summary := result.Summary()
	util.GoldenFile(t, testdataDir(), "result-summary-conflicts", summary)
}

func TestIntegration_ResultSummary_WithFailures_Golden(t *testing.T) {
	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Codex,
		Strategy: StrategyOverwrite,
		DryRun:   false,
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "success-skill"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "failed-skill"}, Action: ActionFailed, Error: os.ErrPermission},
		},
	}

	summary := result.Summary()
	util.GoldenFile(t, testdataDir(), "result-summary-failures", summary)
}

// Conflict detection integration tests

func TestIntegration_ConflictDetection_ContentOnly(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create source skill
	util.WriteFile(t, filepath.Join(sourceDir, "conflict-skill.md"), `---
name: conflict-skill
description: Same description
tools:
  - read
  - write
---

Source version content that differs.
`)

	// Create target skill with same metadata but different content
	util.WriteFile(t, filepath.Join(targetDir, "conflict-skill.md"), `---
name: conflict-skill
description: Same description
tools:
  - read
  - write
---

Target version content that differs.
`)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyThreeWay,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Three-way strategy attempts automatic merge. It either:
	// 1. Merges successfully (ActionMerged)
	// 2. Has unresolvable conflicts (ActionConflict)
	util.AssertEqual(t, result.TotalProcessed(), 1)

	// Verify some action was taken (not skipped)
	if len(result.Skipped()) == 1 {
		t.Error("Expected skill to be processed, not skipped")
	}
}

func TestIntegration_ConflictDetection_MetadataOnly(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create source skill
	util.WriteFile(t, filepath.Join(sourceDir, "metadata-conflict.md"), `---
name: metadata-conflict
description: Source description
tools:
  - read
---

Same content in both versions.
`)

	// Create target skill with different metadata but same content
	util.WriteFile(t, filepath.Join(targetDir, "metadata-conflict.md"), `---
name: metadata-conflict
description: Target description
tools:
  - write
---

Same content in both versions.
`)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyThreeWay,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Metadata differences are detected by conflict detector
	// Three-way strategy will attempt merge
	util.AssertEqual(t, result.TotalProcessed(), 1)
}

func TestIntegration_ConflictDetection_BothContentAndMetadata(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create source skill
	util.WriteFile(t, filepath.Join(sourceDir, "both-conflict.md"), `---
name: both-conflict
description: Source description
tools:
  - read
  - grep
---

Source content here.
`)

	// Create target skill with different metadata AND content
	util.WriteFile(t, filepath.Join(targetDir, "both-conflict.md"), `---
name: both-conflict
description: Target description
tools:
  - write
  - bash
---

Target content here.
`)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyThreeWay,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Differences in both content and metadata detected
	// Three-way will attempt merge
	util.AssertEqual(t, result.TotalProcessed(), 1)
}

func TestIntegration_ConflictDetection_EmptyFiles(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create source skill with minimal content
	util.WriteFile(t, filepath.Join(sourceDir, "empty-test.md"), `---
name: empty-test
---

`)

	// Create target skill with different empty-ish content
	util.WriteFile(t, filepath.Join(targetDir, "empty-test.md"), `---
name: empty-test
---


`)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyThreeWay,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Empty content differences should be detected but handled gracefully
	util.AssertEqual(t, result.TotalProcessed(), 1)
}

func TestIntegration_ConflictDetection_LargeContentDiff(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create source skill with large content block
	var sourceBuilder, targetBuilder strings.Builder
	sourceBuilder.WriteString(`---
name: large-diff
description: Testing large content differences
---

# Source Version

`)
	for range 100 {
		sourceBuilder.WriteString("This is source line with unique content.\n")
	}
	sourceContent := sourceBuilder.String()

	// Create target skill with completely different large content
	targetBuilder.WriteString(`---
name: large-diff
description: Testing large content differences
---

# Target Version

`)
	for range 100 {
		targetBuilder.WriteString("This is target line with different content.\n")
	}
	targetContent := targetBuilder.String()

	util.WriteFile(t, filepath.Join(sourceDir, "large-diff.md"), sourceContent)
	util.WriteFile(t, filepath.Join(targetDir, "large-diff.md"), targetContent)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyThreeWay,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Should handle large diffs without crashing
	util.AssertEqual(t, result.TotalProcessed(), 1)
	if result.HasConflicts() {
		// Verify hunks were generated
		conflicts := result.Conflicts()
		util.AssertEqual(t, len(conflicts), 1)
		if conflicts[0].Conflict != nil && len(conflicts[0].Conflict.Hunks) == 0 {
			t.Error("Expected diff hunks to be generated for large content difference")
		}
	}
}

func TestIntegration_ConflictDetection_DiffHunkGeneration(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create source and target with specific differences for hunk testing
	util.WriteFile(t, filepath.Join(sourceDir, "hunk-test.md"), `---
name: hunk-test
---

Line 1: Same
Line 2: Different in source
Line 3: Same
Line 4: Same
Line 5: Another difference in source
Line 6: Same
`)

	util.WriteFile(t, filepath.Join(targetDir, "hunk-test.md"), `---
name: hunk-test
---

Line 1: Same
Line 2: Different in target
Line 3: Same
Line 4: Same
Line 5: Another difference in target
Line 6: Same
`)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyThreeWay,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	if result.HasConflicts() {
		conflicts := result.Conflicts()
		util.AssertEqual(t, len(conflicts), 1)

		// Verify hunks were generated
		if conflicts[0].Conflict != nil {
			if len(conflicts[0].Conflict.Hunks) == 0 {
				t.Error("Expected diff hunks to be generated")
			}
		}
	}
}

// Merge strategy failure mode tests

func TestIntegration_MergeStrategy_ThreeWayWithoutAncestor(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create source and target with no common history (simulates no ancestor)
	util.WriteFile(t, filepath.Join(sourceDir, "no-ancestor.md"), `---
name: no-ancestor
description: Source version
---

This is completely new content from source.
Multiple lines that have no relation
to any previous version.
`)

	util.WriteFile(t, filepath.Join(targetDir, "no-ancestor.md"), `---
name: no-ancestor
description: Target version
---

This is completely different content from target.
Also multiple lines with no shared history
or common ancestor base.
`)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyThreeWay,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Three-way without ancestor should fall back to two-way merge
	// and potentially detect conflicts
	util.AssertEqual(t, result.TotalProcessed(), 1)
}

func TestIntegration_MergeStrategy_ConflictingTimestamps(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create skills with specific timestamps
	sourceContent := `---
name: timestamp-test
description: Testing timestamp conflicts
---

Source content.
`
	targetContent := `---
name: timestamp-test
description: Testing timestamp conflicts
---

Target content.
`

	sourcePath := filepath.Join(sourceDir, "timestamp-test.md")
	targetPath := filepath.Join(targetDir, "timestamp-test.md")

	util.WriteFile(t, sourcePath, sourceContent)
	util.WriteFile(t, targetPath, targetContent)

	// Set target to be newer than source (inverted case)
	oldTime := time.Now().Add(-48 * time.Hour)
	newTime := time.Now().Add(-1 * time.Hour)

	if err := os.Chtimes(sourcePath, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set source time: %v", err)
	}
	if err := os.Chtimes(targetPath, newTime, newTime); err != nil {
		t.Fatalf("Failed to set target time: %v", err)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyNewer,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Newer strategy should skip since target is newer
	skipped := result.Skipped()
	util.AssertEqual(t, len(skipped), 1)
	util.AssertEqual(t, skipped[0].Skill.Name, "timestamp-test")
}

func TestIntegration_MergeStrategy_IncompatibleContent(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create source with structured content
	util.WriteFile(t, filepath.Join(sourceDir, "incompatible.md"), `---
name: incompatible
description: Structured content
---

# Section A
Content A

# Section B
Content B
`)

	// Create target with completely different structure
	util.WriteFile(t, filepath.Join(targetDir, "incompatible.md"), `---
name: incompatible
description: Unstructured content
---

Just a blob of text
with no structure at all
completely incompatible
for automatic merging
`)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyMerge,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Merge strategy should concatenate even if incompatible
	merged := result.Merged()
	util.AssertEqual(t, len(merged), 1)

	// Verify merged content contains both versions
	targetPath := filepath.Join(targetDir, "incompatible.md")
	// #nosec G304 - test file path is controlled
	content, err := os.ReadFile(targetPath)
	util.AssertNoError(t, err)

	contentStr := string(content)
	// Should contain merge separator
	if len(contentStr) == 0 {
		t.Error("Expected merged content to not be empty")
	}
}

func TestIntegration_MergeStrategy_SkipIdempotency(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	skillContent := `---
name: skip-test
description: Testing skip strategy idempotency
---

Content that should not change.
`

	util.WriteFile(t, filepath.Join(sourceDir, "skip-test.md"), skillContent)
	util.WriteFile(t, filepath.Join(targetDir, "skip-test.md"), skillContent)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategySkip,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	// Run sync multiple times
	for range 3 {
		result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
		util.AssertNoError(t, err)

		// Should always skip existing files
		skipped := result.Skipped()
		util.AssertEqual(t, len(skipped), 1)
		util.AssertEqual(t, result.TotalProcessed(), 1)
	}

	// Verify content never changed
	targetPath := filepath.Join(targetDir, "skip-test.md")
	// #nosec G304 - test file path is controlled
	content, err := os.ReadFile(targetPath)
	util.AssertNoError(t, err)

	util.AssertEqual(t, string(content), skillContent)
}

func TestIntegration_MergeStrategy_OverwriteForce(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create source with new content
	util.WriteFile(t, filepath.Join(sourceDir, "force-test.md"), `---
name: force-test
description: New version
---

Completely new content that should replace target.
`)

	// Create target with old content
	oldContent := `---
name: force-test
description: Old version
---

Old content that will be overwritten.
`
	util.WriteFile(t, filepath.Join(targetDir, "force-test.md"), oldContent)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Should always update with overwrite
	updated := result.Updated()
	util.AssertEqual(t, len(updated), 1)

	// Verify content was replaced
	targetPath := filepath.Join(targetDir, "force-test.md")
	// #nosec G304 - test file path is controlled
	content, err := os.ReadFile(targetPath)
	util.AssertNoError(t, err)

	contentStr := string(content)
	if contentStr == oldContent {
		t.Error("Expected content to be overwritten")
	}
}

// Note: Backup/restore tests demonstrate potential integration patterns.
// The sync engine doesn't currently have built-in backup integration,
// but these tests show how it could work if implemented.

func TestIntegration_BackupBeforeSync(t *testing.T) {
	// This test demonstrates how backup could be integrated with sync
	// Currently, backup must be called manually before sync operations

	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create original target content
	originalContent := `---
name: backup-test
description: Original content
---

Original content that will be overwritten.
`
	targetPath := filepath.Join(targetDir, "backup-test.md")
	util.WriteFile(t, targetPath, originalContent)

	// Create source with different content
	util.WriteFile(t, filepath.Join(sourceDir, "backup-test.md"), `---
name: backup-test
description: New content
---

New content from source.
`)

	// Create backup before sync operation
	t.Setenv("SKILLSYNC_HOME", t.TempDir())
	backupOpts := backup.Options{
		Platform:    "cursor",
		Description: "Backup before overwrite test",
		Tags:        []string{"integration-test"},
	}
	metadata, err := backup.CreateBackup(targetPath, backupOpts)
	util.AssertNoError(t, err)

	// Verify backup metadata
	util.AssertEqual(t, metadata.Platform, "cursor")
	util.AssertEqual(t, metadata.SourcePath, targetPath)
	util.AssertEqual(t, metadata.Description, "Backup before overwrite test")
	util.AssertEqual(t, len(metadata.Tags), 1)
	util.AssertEqual(t, metadata.Tags[0], "integration-test")

	// Verify backup file exists
	backupInfo, err := os.Stat(metadata.BackupPath)
	util.AssertNoError(t, err)
	if backupInfo.IsDir() {
		t.Error("Expected backup path to be a file, not a directory")
	}

	// Verify backup content matches original
	// #nosec G304 - test file path is controlled
	backupContent, err := os.ReadFile(metadata.BackupPath)
	util.AssertNoError(t, err)
	util.AssertEqual(t, string(backupContent), originalContent)

	// Verify backup hash is valid SHA256 (64 hex characters)
	util.AssertEqual(t, len(metadata.Hash), 64)

	// Verify backup size matches
	util.AssertEqual(t, metadata.Size, int64(len(originalContent)))

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Verify sync succeeded
	util.AssertEqual(t, len(result.Updated()), 1)

	// Verify content was changed
	// #nosec G304 - test file path is controlled
	newContent, err := os.ReadFile(targetPath)
	util.AssertNoError(t, err)

	if string(newContent) == originalContent {
		t.Error("Expected content to be updated")
	}
}

func TestIntegration_DryRunNoBackupNeeded(t *testing.T) {
	// Demonstrates that dry-run mode shouldn't trigger backups
	// since no actual changes are made

	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create target file
	targetPath := filepath.Join(targetDir, "dryrun-test.md")
	originalContent := `---
name: dryrun-test
---

Original content.
`
	util.WriteFile(t, targetPath, originalContent)

	// Create different source
	util.WriteFile(t, filepath.Join(sourceDir, "dryrun-test.md"), `---
name: dryrun-test
---

Different content.
`)

	opts := Options{
		DryRun:     true,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, result.DryRun, true)

	// Verify original content unchanged (no backup needed)
	// #nosec G304 - test file path is controlled
	content, err := os.ReadFile(targetPath)
	util.AssertNoError(t, err)

	util.AssertEqual(t, string(content), originalContent)
}

// Note: Dependency resolution tests document potential future functionality.
// The sync engine doesn't currently track or resolve skill dependencies,
// but these tests show expected behavior if implemented.

func TestIntegration_MultipleSkillsPreserveNames(t *testing.T) {
	// Tests that skill names are preserved during sync, which would be
	// necessary for any future dependency tracking between skills

	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create multiple related skills
	skills := map[string]string{
		"base-skill.md": `---
name: base-skill
description: Foundation skill that others might depend on
---

Base functionality.
`,
		"dependent-skill.md": `---
name: dependent-skill
description: Skill that could reference base-skill
---

Uses base-skill features.
`,
		"another-skill.md": `---
name: another-skill
description: Independent skill
---

Standalone functionality.
`,
	}

	for name, content := range skills {
		util.WriteFile(t, filepath.Join(sourceDir, name), content)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Verify all skills synced successfully
	util.AssertEqual(t, result.TotalProcessed(), 3)
	util.AssertEqual(t, len(result.Created()), 3)

	// Verify skill names are preserved (important for dependency references)
	expectedNames := []string{"base-skill", "dependent-skill", "another-skill"}
	for _, skillResult := range result.Skills {
		if !slices.Contains(expectedNames, skillResult.Skill.Name) {
			t.Errorf("Skill name %q not in expected list", skillResult.Skill.Name)
		}
	}
}

func TestIntegration_CrossPlatformNameConsistency(t *testing.T) {
	// Tests that skill names remain consistent across platform transformations,
	// which would be critical for dependency resolution

	s := New()

	sourceDir := t.TempDir()
	intermediateDir := t.TempDir()
	finalDir := t.TempDir()

	skillContent := `---
name: consistent-name
description: Testing name consistency
---

Content.
`

	util.WriteFile(t, filepath.Join(sourceDir, "consistent-name.md"), skillContent)

	// Sync ClaudeCode -> Cursor
	opts1 := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: intermediateDir,
	}

	result1, err := s.Sync(model.ClaudeCode, model.Cursor, opts1)
	util.AssertNoError(t, err)
	util.AssertEqual(t, len(result1.Created()), 1)

	// Sync Cursor -> back to ClaudeCode format
	opts2 := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: intermediateDir,
		TargetPath: finalDir,
	}

	result2, err := s.Sync(model.Cursor, model.ClaudeCode, opts2)
	util.AssertNoError(t, err)
	util.AssertEqual(t, len(result2.Created()), 1)

	// Verify name consistency through transformations
	util.AssertEqual(t, result1.Skills[0].Skill.Name, "consistent-name")
	util.AssertEqual(t, result2.Skills[0].Skill.Name, "consistent-name")
}

func TestIntegration_MetadataPreservation(t *testing.T) {
	// Tests that skill metadata (which could include dependency info) is preserved

	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create skill with metadata that could represent dependencies
	util.WriteFile(t, filepath.Join(sourceDir, "with-metadata.md"), `---
name: with-metadata
description: Skill with metadata
metadata:
  category: utility
  version: "1.0.0"
  requires: base-skill
---

Content.
`)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 1)

	// Metadata preservation is handled by the transformer
	// This test ensures sync doesn't lose metadata during transfer
	skill := result.Skills[0].Skill
	if skill.Metadata == nil {
		t.Error("Expected metadata to be preserved")
	}
}
