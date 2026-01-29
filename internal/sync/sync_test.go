package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/model"
)

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if s.transformer == nil {
		t.Error("New() did not initialize transformer")
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.DryRun {
		t.Error("DefaultOptions should have DryRun=false")
	}
	if opts.Strategy != StrategyOverwrite {
		t.Errorf("DefaultOptions should have Strategy=overwrite, got %s", opts.Strategy)
	}
}

func TestSynchronizer_Sync_EmptySource(t *testing.T) {
	s := New()

	// Create temp directories
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	opts := Options{
		DryRun:     true,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.Skills) != 0 {
		t.Errorf("Expected 0 skills, got %d", len(result.Skills))
	}
}

func TestSynchronizer_Sync_SingleSkill(t *testing.T) {
	s := New()

	// Create temp directories
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a source skill file
	skillContent := `---
name: test-skill
description: A test skill
---

This is the skill content.
`
	skillPath := filepath.Join(sourceDir, "test-skill.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o600); err != nil {
		t.Fatalf("Failed to create skill file: %v", err)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.Skills) != 1 {
		t.Errorf("Expected 1 skill, got %d", len(result.Skills))
	}

	if result.Skills[0].Action != ActionCreated {
		t.Errorf("Expected action 'created', got %s", result.Skills[0].Action)
	}

	// Verify file was created in target
	targetFile := filepath.Join(targetDir, "test-skill.md")
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		t.Error("Target file was not created")
	}
}

func TestSynchronizer_Sync_DryRun(t *testing.T) {
	s := New()

	// Create temp directories
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a source skill file
	skillContent := `---
name: test-skill
---

Content here.
`
	skillPath := filepath.Join(sourceDir, "test-skill.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o600); err != nil {
		t.Fatalf("Failed to create skill file: %v", err)
	}

	opts := Options{
		DryRun:     true,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if !result.DryRun {
		t.Error("Result should indicate dry run")
	}

	// Verify file was NOT created in target (dry run)
	targetFile := filepath.Join(targetDir, "test-skill.md")
	if _, err := os.Stat(targetFile); !os.IsNotExist(err) {
		t.Error("Target file should not exist in dry run mode")
	}
}

func TestSynchronizer_Sync_SkipStrategy(t *testing.T) {
	s := New()

	// Create temp directories
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create source skill
	sourceContent := `---
name: test-skill
---

Source content.
`
	sourcePath := filepath.Join(sourceDir, "test-skill.md")
	if err := os.WriteFile(sourcePath, []byte(sourceContent), 0o600); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create existing target skill
	targetContent := `---
name: test-skill
---

Target content.
`
	targetPath := filepath.Join(targetDir, "test-skill.md")
	if err := os.WriteFile(targetPath, []byte(targetContent), 0o600); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategySkip,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.Skipped()) != 1 {
		t.Errorf("Expected 1 skipped skill, got %d", len(result.Skipped()))
	}

	// Verify target content was not changed
	// #nosec G304 - targetPath is constructed from test temp directory
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read target file: %v", err)
	}
	if string(content) != targetContent {
		t.Error("Target content should not have changed with skip strategy")
	}
}

func TestSynchronizer_Sync_NewerStrategy(t *testing.T) {
	s := New()

	// Create temp directories
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create older target skill first
	targetContent := `---
name: test-skill
---

Old content.
`
	targetPath := filepath.Join(targetDir, "test-skill.md")
	if err := os.WriteFile(targetPath, []byte(targetContent), 0o600); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Set older modification time on target
	oldTime := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(targetPath, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set target file time: %v", err)
	}

	// Create newer source skill
	sourceContent := `---
name: test-skill
---

New content.
`
	sourcePath := filepath.Join(sourceDir, "test-skill.md")
	if err := os.WriteFile(sourcePath, []byte(sourceContent), 0o600); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyNewer,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.Updated()) != 1 {
		t.Errorf("Expected 1 updated skill with newer strategy, got %d", len(result.Updated()))
	}
}

func TestSynchronizer_DetermineAction(t *testing.T) {
	s := New()
	now := time.Now()
	older := now.Add(-1 * time.Hour)

	tests := []struct {
		name     string
		source   model.Skill
		existing model.Skill
		exists   bool
		strategy Strategy
		expected Action
	}{
		{
			name:     "new skill with overwrite",
			source:   model.Skill{Name: "test", ModifiedAt: now},
			exists:   false,
			strategy: StrategyOverwrite,
			expected: ActionCreated,
		},
		{
			name:     "existing skill with overwrite",
			source:   model.Skill{Name: "test", ModifiedAt: now},
			existing: model.Skill{Name: "test", ModifiedAt: older},
			exists:   true,
			strategy: StrategyOverwrite,
			expected: ActionUpdated,
		},
		{
			name:     "existing skill with skip",
			source:   model.Skill{Name: "test", ModifiedAt: now},
			existing: model.Skill{Name: "test", ModifiedAt: older},
			exists:   true,
			strategy: StrategySkip,
			expected: ActionSkipped,
		},
		{
			name:     "newer source with newer strategy",
			source:   model.Skill{Name: "test", ModifiedAt: now},
			existing: model.Skill{Name: "test", ModifiedAt: older},
			exists:   true,
			strategy: StrategyNewer,
			expected: ActionUpdated,
		},
		{
			name:     "older source with newer strategy",
			source:   model.Skill{Name: "test", ModifiedAt: older},
			existing: model.Skill{Name: "test", ModifiedAt: now},
			exists:   true,
			strategy: StrategyNewer,
			expected: ActionSkipped,
		},
		{
			name:     "existing skill with merge",
			source:   model.Skill{Name: "test", ModifiedAt: now},
			existing: model.Skill{Name: "test", ModifiedAt: older},
			exists:   true,
			strategy: StrategyMerge,
			expected: ActionMerged,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, _, _ := s.determineAction(tt.source, tt.existing, tt.exists, tt.strategy)
			if action != tt.expected {
				t.Errorf("Expected action %s, got %s", tt.expected, action)
			}
		})
	}
}

func TestResult_Methods(t *testing.T) {
	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Cursor,
		Strategy: StrategyOverwrite,
		DryRun:   false,
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "created-skill"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "updated-skill"}, Action: ActionUpdated},
			{Skill: model.Skill{Name: "skipped-skill"}, Action: ActionSkipped},
			{Skill: model.Skill{Name: "merged-skill"}, Action: ActionMerged},
			{Skill: model.Skill{Name: "failed-skill"}, Action: ActionFailed},
		},
	}

	if len(result.Created()) != 1 {
		t.Errorf("Expected 1 created, got %d", len(result.Created()))
	}
	if len(result.Updated()) != 1 {
		t.Errorf("Expected 1 updated, got %d", len(result.Updated()))
	}
	if len(result.Skipped()) != 1 {
		t.Errorf("Expected 1 skipped, got %d", len(result.Skipped()))
	}
	if len(result.Merged()) != 1 {
		t.Errorf("Expected 1 merged, got %d", len(result.Merged()))
	}
	if len(result.Failed()) != 1 {
		t.Errorf("Expected 1 failed, got %d", len(result.Failed()))
	}
	if result.TotalProcessed() != 5 {
		t.Errorf("Expected 5 total processed, got %d", result.TotalProcessed())
	}
	if result.TotalChanged() != 3 {
		t.Errorf("Expected 3 total changed, got %d", result.TotalChanged())
	}
	if result.Success() {
		t.Error("Result with failed skill should not be success")
	}
}

func TestResult_Summary(t *testing.T) {
	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Cursor,
		Strategy: StrategyOverwrite,
		DryRun:   true,
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "test"}, Action: ActionCreated},
		},
	}

	summary := result.Summary()
	if summary == "" {
		t.Error("Summary should not be empty")
	}
	if !contains(summary, "Dry run") {
		t.Error("Summary should indicate dry run")
	}
	if !contains(summary, "claude-code") {
		t.Error("Summary should contain source platform")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests for sync operations

func BenchmarkSync(b *testing.B) {
	// Create realistic scenario with 50 skills
	sourceDir := b.TempDir()
	targetDir := b.TempDir()

	// Create 50 source skill files
	for i := range 50 {
		skillContent := `---
name: skill-` + string(rune('a'+i%26)) + `-` + string(rune('0'+(i/26)%10)) + `
description: Test skill for benchmarking sync operations
platforms: [claude-code, cursor]
---

# Skill Content

This is a test skill with realistic content.

## Usage

Instructions for using this skill go here.

## Examples

- Example 1
- Example 2
- Example 3
`
		skillPath := filepath.Join(sourceDir, "skill-"+string(rune('a'+i%26))+"-"+string(rune('0'+(i/26)%10))+".md")
		// #nosec G306 - benchmark files don't need restrictive permissions
		if err := os.WriteFile(skillPath, []byte(skillContent), 0o600); err != nil {
			b.Fatalf("Failed to create skill file: %v", err)
		}
	}

	s := New()
	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
		if err != nil {
			b.Fatalf("Sync failed: %v", err)
		}
	}
}

func BenchmarkProcessSkill(b *testing.B) {
	s := New()
	targetDir := b.TempDir()

	// Create a realistic skill
	skill := model.Skill{
		Name:        "benchmark-skill",
		Description: "A skill for benchmarking processSkill",
		Content:     "# Benchmark Skill\n\nThis is the skill content for benchmarking.",
		Platform:    model.ClaudeCode,
		Metadata: map[string]string{
			"author":  "Test",
			"version": "1.0.0",
		},
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: "",
		TargetPath: targetDir,
	}

	// Create empty target skills map
	targetSkills := make(map[string]model.Skill)

	b.ResetTimer()
	for b.Loop() {
		_ = s.processSkill(skill, model.Cursor, targetDir, targetSkills, opts)
	}
}

func BenchmarkDetermineAction(b *testing.B) {
	s := New()

	sourceSkill := model.Skill{
		Name:        "test-skill",
		Description: "Source skill",
		Content:     "# Source Content\n\nThis is the source.",
		Platform:    model.ClaudeCode,
	}

	targetSkill := model.Skill{
		Name:        "test-skill",
		Description: "Target skill (different content)",
		Content:     "# Target Content\n\nThis is the target.",
		Platform:    model.Cursor,
	}

	b.Run("overwrite strategy", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_, _, _ = s.determineAction(sourceSkill, targetSkill, true, StrategyOverwrite)
		}
	})

	b.Run("skip strategy", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_, _, _ = s.determineAction(sourceSkill, targetSkill, true, StrategySkip)
		}
	})

	b.Run("merge strategy", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_, _, _ = s.determineAction(sourceSkill, targetSkill, true, StrategyMerge)
		}
	})
}
