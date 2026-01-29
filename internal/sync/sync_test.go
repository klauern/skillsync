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

func TestProgressCallback_Success(t *testing.T) {
	s := New()
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create test skill
	skillContent := `---
name: progress-skill
description: Test progress callback
---
Test content.
`
	skillPath := filepath.Join(sourceDir, "progress-skill.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o600); err != nil {
		t.Fatalf("Failed to create skill file: %v", err)
	}

	// Track progress events
	var events []ProgressEvent
	progressCallback := func(event ProgressEvent) error {
		events = append(events, event)
		return nil
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
		Progress:   progressCallback,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.Skills) != 1 {
		t.Errorf("Expected 1 skill, got %d", len(result.Skills))
	}

	// Verify progress events were emitted
	if len(events) < 3 {
		t.Errorf("Expected at least 3 events (start, skill_start, skill_complete, complete), got %d", len(events))
	}

	// Check event types
	if events[0].Type != ProgressEventStart {
		t.Errorf("First event should be start, got %s", events[0].Type)
	}

	lastEvent := events[len(events)-1]
	if lastEvent.Type != ProgressEventComplete {
		t.Errorf("Last event should be complete, got %s", lastEvent.Type)
	}

	if lastEvent.PercentComplete != 100 {
		t.Errorf("Expected 100%% complete, got %d%%", lastEvent.PercentComplete)
	}
}

func TestProgressCallback_Cancellation(t *testing.T) {
	s := New()
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create test skill
	skillContent := `---
name: cancel-skill
description: Test cancellation
---
Test content.
`
	skillPath := filepath.Join(sourceDir, "cancel-skill.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o600); err != nil {
		t.Fatalf("Failed to create skill file: %v", err)
	}

	// Cancel on first skill start
	progressCallback := func(event ProgressEvent) error {
		if event.Type == ProgressEventSkillStart {
			return os.ErrClosed // Return an error to cancel
		}
		return nil
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
		Progress:   progressCallback,
	}

	_, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	if err == nil {
		t.Error("Expected error from cancelled sync")
	}
}

func TestSyncBidirectional_EmptyPlatforms(t *testing.T) {
	s := New()
	dirA := t.TempDir()
	dirB := t.TempDir()

	opts := Options{
		DryRun:     true,
		Strategy:   StrategyOverwrite,
		SourcePath: dirA,
		TargetPath: dirB,
	}

	result, err := s.SyncBidirectional(model.ClaudeCode, model.Cursor, opts)
	if err != nil {
		t.Fatalf("Bidirectional sync failed: %v", err)
	}

	if result.ResultAtoB != nil {
		t.Error("Expected no A->B sync for empty platforms")
	}

	if result.ResultBtoA != nil {
		t.Error("Expected no B->A sync for empty platforms")
	}

	if len(result.Conflicts) != 0 {
		t.Errorf("Expected 0 conflicts, got %d", len(result.Conflicts))
	}
}

func TestSyncBidirectional_SkillOnlyInA(t *testing.T) {
	s := New()
	dirA := t.TempDir()
	dirB := t.TempDir()

	// Create skill only in A
	skillContent := `---
name: only-in-a
description: Skill only in A
---
Content from A.
`
	skillPath := filepath.Join(dirA, "only-in-a.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o600); err != nil {
		t.Fatalf("Failed to create skill file: %v", err)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: dirA,
		TargetPath: dirB,
	}

	result, err := s.SyncBidirectional(model.ClaudeCode, model.Cursor, opts)
	if err != nil {
		t.Fatalf("Bidirectional sync failed: %v", err)
	}

	// Skill should be synced A -> B
	if result.ResultAtoB == nil {
		t.Fatal("Expected A->B sync result")
	}

	if len(result.ResultAtoB.Created()) != 1 {
		t.Errorf("Expected 1 created skill in A->B, got %d", len(result.ResultAtoB.Created()))
	}

	// No B -> A sync
	if result.ResultBtoA != nil {
		t.Error("Expected no B->A sync")
	}

	// No conflicts
	if len(result.Conflicts) != 0 {
		t.Errorf("Expected 0 conflicts, got %d", len(result.Conflicts))
	}
}

func TestSyncBidirectional_SkillOnlyInB(t *testing.T) {
	s := New()
	dirA := t.TempDir()
	dirB := t.TempDir()

	// Create skill only in B
	skillContent := `---
name: only-in-b
description: Skill only in B
---
Content from B.
`
	skillPath := filepath.Join(dirB, "only-in-b.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o600); err != nil {
		t.Fatalf("Failed to create skill file: %v", err)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: dirA,
		TargetPath: dirB,
	}

	result, err := s.SyncBidirectional(model.ClaudeCode, model.Cursor, opts)
	if err != nil {
		t.Fatalf("Bidirectional sync failed: %v", err)
	}

	// No A -> B sync
	if result.ResultAtoB != nil {
		t.Error("Expected no A->B sync")
	}

	// Skill should be synced B -> A
	if result.ResultBtoA == nil {
		t.Fatal("Expected B->A sync result")
	}

	if len(result.ResultBtoA.Created()) != 1 {
		t.Errorf("Expected 1 created skill in B->A, got %d", len(result.ResultBtoA.Created()))
	}

	// No conflicts
	if len(result.Conflicts) != 0 {
		t.Errorf("Expected 0 conflicts, got %d", len(result.Conflicts))
	}
}

func TestSyncBidirectional_ConflictWithNewerStrategy(t *testing.T) {
	s := New()
	dirA := t.TempDir()
	dirB := t.TempDir()

	// Create skill in A (older)
	skillAContent := `---
name: conflict-skill
description: Skill with conflict
---
Content from A.
`
	skillAPath := filepath.Join(dirA, "conflict-skill.md")
	if err := os.WriteFile(skillAPath, []byte(skillAContent), 0o600); err != nil {
		t.Fatalf("Failed to create skill A: %v", err)
	}

	// Wait a moment to ensure different timestamps
	time.Sleep(10 * time.Millisecond)

	// Create skill in B (newer)
	skillBContent := `---
name: conflict-skill
description: Skill with conflict
---
Content from B.
`
	skillBPath := filepath.Join(dirB, "conflict-skill.md")
	if err := os.WriteFile(skillBPath, []byte(skillBContent), 0o600); err != nil {
		t.Fatalf("Failed to create skill B: %v", err)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyNewer,
		SourcePath: dirA,
		TargetPath: dirB,
	}

	result, err := s.SyncBidirectional(model.ClaudeCode, model.Cursor, opts)
	if err != nil {
		t.Fatalf("Bidirectional sync failed: %v", err)
	}

	// B is newer, so should sync B -> A
	if result.ResultBtoA == nil {
		t.Fatal("Expected B->A sync for newer skill")
	}

	if len(result.ResultBtoA.Updated()) != 1 {
		t.Errorf("Expected 1 updated skill in B->A, got %d", len(result.ResultBtoA.Updated()))
	}

	// No A -> B sync (A is older)
	if result.ResultAtoB != nil {
		t.Error("Expected no A->B sync for older skill")
	}
}

func TestSyncBidirectional_ConflictWithMergeStrategy(t *testing.T) {
	s := New()
	dirA := t.TempDir()
	dirB := t.TempDir()

	// Create conflicting skills
	skillAContent := `---
name: merge-conflict
description: Test merge conflict
---
Content from A.
`
	skillAPath := filepath.Join(dirA, "merge-conflict.md")
	if err := os.WriteFile(skillAPath, []byte(skillAContent), 0o600); err != nil {
		t.Fatalf("Failed to create skill A: %v", err)
	}

	skillBContent := `---
name: merge-conflict
description: Test merge conflict
---
Content from B.
`
	skillBPath := filepath.Join(dirB, "merge-conflict.md")
	if err := os.WriteFile(skillBPath, []byte(skillBContent), 0o600); err != nil {
		t.Fatalf("Failed to create skill B: %v", err)
	}

	opts := Options{
		DryRun:     true,
		Strategy:   StrategyMerge,
		SourcePath: dirA,
		TargetPath: dirB,
	}

	result, err := s.SyncBidirectional(model.ClaudeCode, model.Cursor, opts)
	if err != nil {
		t.Fatalf("Bidirectional sync failed: %v", err)
	}

	// Merge strategy should result in conflicts for bidirectional sync
	if len(result.Conflicts) != 1 {
		t.Errorf("Expected 1 conflict with merge strategy, got %d", len(result.Conflicts))
	}

	if result.Conflicts[0].Name != "merge-conflict" {
		t.Errorf("Expected conflict for 'merge-conflict', got %s", result.Conflicts[0].Name)
	}
}

func TestBidirectionalResult_Summary(t *testing.T) {
	result := &BidirectionalResult{
		PlatformA: model.ClaudeCode,
		PlatformB: model.Cursor,
		Strategy:  StrategyOverwrite,
		DryRun:    false,
		Conflicts: []BidirectionalConflict{
			{
				Name: "test-conflict",
			},
		},
	}

	summary := result.Summary()
	if summary == "" {
		t.Error("Expected non-empty summary")
	}

	if !result.HasConflicts() {
		t.Error("Expected HasConflicts to return true")
	}
}
