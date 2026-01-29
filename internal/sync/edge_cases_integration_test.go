package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/util"
)

// Integration tests for edge cases, error handling, and boundary conditions

func TestIntegration_SpecialCharactersInFilenames(t *testing.T) {
	// Test syncing files with special characters in filenames
	// Note: Skill names have validation rules (alphanumeric, dashes, underscores)
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create skills with valid special characters (filesystem-safe and name-valid)
	specialNames := []string{
		"skill-with-dashes.md",
		"skill_with_underscores.md",
	}

	for _, name := range specialNames {
		skillName := strings.TrimSuffix(name, ".md")
		content := `---
name: ` + skillName + `
description: Testing special characters
---

Content for ` + name + `
`
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

	// Should process the valid skills
	if result.TotalProcessed() < 2 {
		t.Errorf("Expected at least 2 skills processed, got %d", result.TotalProcessed())
	}

	// Verify valid files were created
	for _, name := range specialNames {
		targetPath := filepath.Join(targetDir, name)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s to be created", name)
		}
	}
}

func TestIntegration_SpecialCharactersInContent(t *testing.T) {
	// Test syncing skills with special characters in content
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Content with various special characters
	specialContent := `---
name: special-content
description: Testing "quotes" and \backslashes\
---

# Special Content Test

This content has:
- "Double quotes"
- 'Single quotes'
- \Backslashes\
- Unicode: émojis 🚀 ✨ 🎉
- Code: ` + "`" + `backticks` + "`" + `
- Tabs:	indented	content
- Newlines and special chars: & < > | ; $ *

\` + "`" + "`" + `go
func example() {
	fmt.Println("Code block with special chars: $VAR")
}
\` + "`" + "`" + `
`

	util.WriteFile(t, filepath.Join(sourceDir, "special-content.md"), specialContent)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 1)

	// Verify content is preserved exactly
	targetPath := filepath.Join(targetDir, "special-content.md")
	// #nosec G304 - test file path is controlled
	targetContent, err := os.ReadFile(targetPath)
	util.AssertNoError(t, err)

	// Content should match (modulo any normalization)
	if len(targetContent) == 0 {
		t.Error("Expected content to be preserved")
	}

	target := string(targetContent)
	if !strings.Contains(target, "émojis 🚀") {
		t.Error("Unicode characters not preserved")
	}
}

func TestIntegration_EmptySourceDirectory(t *testing.T) {
	// Test syncing from an empty source directory
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a skill in target but not in source
	util.WriteFile(t, filepath.Join(targetDir, "existing-skill.md"), `---
name: existing-skill
---

This exists in target only.
`)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// No skills to sync
	util.AssertEqual(t, result.TotalProcessed(), 0)

	// Target file should still exist (sync doesn't delete)
	targetPath := filepath.Join(targetDir, "existing-skill.md")
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Error("Target file should not be deleted when source is empty")
	}
}

func TestIntegration_EmptyTarget_AllCreated(t *testing.T) {
	// Test syncing to an empty target directory (all files created)
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create skills in source
	for i := 1; i <= 3; i++ {
		name := filepath.Join(sourceDir, "skill-"+string(rune('0'+i))+".md")
		content := `---
name: skill-` + string(rune('0'+i)) + `
---

Content ` + string(rune('0'+i)) + `
`
		util.WriteFile(t, name, content)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// All should be created
	util.AssertEqual(t, len(result.Created()), 3)
	util.AssertEqual(t, len(result.Updated()), 0)
}

func TestIntegration_InvalidYAMLFrontmatter(t *testing.T) {
	// Test handling of files with invalid YAML frontmatter
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create file with malformed YAML
	invalidYAML := `---
name: invalid-yaml
description: "Unclosed quote
tools:
  - read
---

Content here.
`

	util.WriteFile(t, filepath.Join(sourceDir, "invalid-yaml.md"), invalidYAML)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	// Should handle gracefully (either error or skip)
	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)

	// Either returns an error or creates a failed result
	if err == nil {
		// If no error, check for failed skills
		if len(result.Failed()) > 0 {
			t.Log("Invalid YAML handled as failed skill (expected)")
		}
	} else {
		t.Logf("Invalid YAML returned error (expected): %v", err)
	}
}

func TestIntegration_MissingRequiredFields(t *testing.T) {
	// Test syncing skills missing required fields
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create skill without name field
	noName := `---
description: Skill without name
---

Content without name.
`

	util.WriteFile(t, filepath.Join(sourceDir, "no-name.md"), noName)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	// Should handle missing name gracefully
	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)

	// Either errors or processes with default name
	if err != nil {
		t.Logf("Missing name field caused error: %v", err)
	} else if result.TotalProcessed() > 0 {
		t.Log("Missing name field was handled (possibly with default)")
	}
}

func TestIntegration_VeryLongFilename(t *testing.T) {
	// Test syncing with very long (but valid) filenames
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a long but valid filename (< 255 chars for most filesystems)
	longName := strings.Repeat("a", 200) + ".md"

	content := `---
name: long-filename-skill
description: Testing very long filenames
---

Content for long filename test.
`

	util.WriteFile(t, filepath.Join(sourceDir, longName), content)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 1)

	// Verify file exists
	targetPath := filepath.Join(targetDir, longName)
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Error("Long filename should be created successfully")
	}
}

func TestIntegration_ConcurrentSyncOperations(t *testing.T) {
	// Test multiple concurrent sync operations (thread safety)
	testCases := []struct {
		name string
		dir  string
	}{
		{"sync-1", t.TempDir()},
		{"sync-2", t.TempDir()},
		{"sync-3", t.TempDir()},
	}

	// Run syncs concurrently
	done := make(chan error, len(testCases))

	for _, tc := range testCases {
		go func(name, dir string) {
			s := New()
			sourceDir := t.TempDir()

			content := `---
name: ` + name + `
---

Content for ` + name + `
`
			util.WriteFile(t, filepath.Join(sourceDir, name+".md"), content)

			opts := Options{
				DryRun:     false,
				Strategy:   StrategyOverwrite,
				SourcePath: sourceDir,
				TargetPath: dir,
			}

			_, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
			done <- err
		}(tc.name, tc.dir)
	}

	// Wait for all to complete
	for range testCases {
		err := <-done
		if err != nil {
			t.Errorf("Concurrent sync failed: %v", err)
		}
	}
}

func TestIntegration_DryRunIdempotency(t *testing.T) {
	// Test that dry run can be called multiple times without side effects
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	util.WriteFile(t, filepath.Join(sourceDir, "test.md"), `---
name: test
---
Content
`)

	opts := Options{
		DryRun:     true,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	// Run dry run multiple times
	for i := 0; i < 3; i++ {
		result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
		util.AssertNoError(t, err)
		util.AssertEqual(t, result.DryRun, true)
		util.AssertEqual(t, result.TotalProcessed(), 1)

		// Target should never be created
		targetPath := filepath.Join(targetDir, "test.md")
		if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
			t.Error("Dry run should not create files")
		}
	}
}

func TestIntegration_ErrorRecovery_PartialSync(t *testing.T) {
	// Test that sync handles partial failures gracefully
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create multiple skills
	for i := 1; i <= 5; i++ {
		content := `---
name: skill-` + string(rune('0'+i)) + `
---
Content ` + string(rune('0'+i)) + `
`
		util.WriteFile(t, filepath.Join(sourceDir, "skill-"+string(rune('0'+i))+".md"), content)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Should process all skills
	util.AssertEqual(t, result.TotalProcessed(), 5)

	// All should succeed (in normal case)
	if len(result.Failed()) > 0 {
		t.Logf("Some skills failed (this tests error recovery): %d", len(result.Failed()))
	}
}

func TestIntegration_DirectoryWithSubdirectories(t *testing.T) {
	// Test syncing from directories with nested subdirectories
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create nested structure
	subdir := filepath.Join(sourceDir, "subdir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Skill in root
	util.WriteFile(t, filepath.Join(sourceDir, "root-skill.md"), `---
name: root-skill
---
Root content
`)

	// Skill in subdir
	util.WriteFile(t, filepath.Join(subdir, "sub-skill.md"), `---
name: sub-skill
---
Sub content
`)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Should discover skills in subdirectories (or just root depending on implementation)
	if result.TotalProcessed() == 0 {
		t.Error("Expected to process at least one skill")
	}
}

func TestIntegration_SymlinkHandling(t *testing.T) {
	// Test how sync handles symlinked skill files
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	externalDir := t.TempDir()

	// Create actual skill file
	actualSkill := filepath.Join(externalDir, "actual-skill.md")
	util.WriteFile(t, actualSkill, `---
name: symlinked-skill
---
Content from actual file
`)

	// Create symlink in source
	symlinkPath := filepath.Join(sourceDir, "symlinked-skill.md")
	if err := os.Symlink(actualSkill, symlinkPath); err != nil {
		t.Skipf("Skipping symlink test (symlinks not supported): %v", err)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)

	// Behavior depends on implementation - may follow symlink or skip
	if err != nil {
		t.Logf("Symlink caused error: %v", err)
	} else {
		t.Logf("Symlink handled: %d skills processed", result.TotalProcessed())
	}
}

func TestIntegration_ReadOnlySourceDirectory(t *testing.T) {
	// Test syncing from a read-only source directory
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create skill
	util.WriteFile(t, filepath.Join(sourceDir, "readonly-test.md"), `---
name: readonly-test
---
Content
`)

	// Make source read-only
	if err := os.Chmod(sourceDir, 0o555); err != nil {
		t.Skipf("Could not make directory read-only: %v", err)
	}
	defer os.Chmod(sourceDir, 0o755) // Restore for cleanup

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	// Should still be able to read from read-only source
	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 1)
}

func TestIntegration_ZeroByteFile(t *testing.T) {
	// Test handling of zero-byte files
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create empty file
	emptyFile := filepath.Join(sourceDir, "empty.md")
	util.WriteFile(t, emptyFile, "")

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	// Should handle empty file gracefully
	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)

	// May skip or fail empty files
	if err != nil {
		t.Logf("Empty file caused error (expected): %v", err)
	} else if len(result.Failed()) > 0 {
		t.Log("Empty file marked as failed (expected)")
	} else if len(result.Skipped()) > 0 {
		t.Log("Empty file skipped (expected)")
	}
}
