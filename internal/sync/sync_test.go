package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/model"
	skillparser "github.com/klauern/skillsync/internal/parser/skills"
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

func TestSynchronizer_Sync_CopilotSourceToCodex(t *testing.T) {
	s := New()
	sourceDir := filepath.Join(t.TempDir(), ".github")
	targetDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(sourceDir, "prompts"), 0o750); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "prompts", "review.prompt.md"), []byte(`---
description: Review code
---

Review the current changes.`), 0o600); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	result, err := s.Sync(model.Copilot, model.Codex, Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 synced skill, got %d", len(result.Skills))
	}
	if result.Skills[0].Action != ActionCreated {
		t.Fatalf("expected created action, got %s", result.Skills[0].Action)
	}

	targetFile := filepath.Join(targetDir, "review", "SKILL.md")
	// #nosec G304 -- test helper reads only repo-controlled sync output.
	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read synced target file: %v", err)
	}
	if !strings.Contains(string(content), "type: prompt") {
		t.Fatalf("expected prompt transport metadata in %s", targetFile)
	}
}

func TestSynchronizer_SyncWithSkills_ToCopilotPaths(t *testing.T) {
	s := New()
	targetDir := t.TempDir()

	sourceFile := filepath.Join(t.TempDir(), "source.md")
	if err := os.WriteFile(sourceFile, []byte("source"), 0o600); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	skills := []model.Skill{
		{
			Name:     "copilot-instructions",
			Platform: model.ClaudeCode,
			Path:     filepath.Join(t.TempDir(), "CLAUDE.md"),
			Content:  "Always-on guidance",
			Metadata: map[string]string{
				model.MetadataKeyCopilotArtifact: model.CopilotArtifactRepositoryInstructions,
			},
		},
		{
			Name:     "go-style",
			Platform: model.ClaudeCode,
			Path:     sourceFile,
			Content:  "Go guidance",
			Metadata: map[string]string{
				model.MetadataKeyCopilotArtifact: model.CopilotArtifactInstructions,
				"applyTo":                        "**/*.go",
			},
		},
		{
			Name:     "review",
			Platform: model.ClaudeCode,
			Path:     sourceFile,
			Content:  "Review prompt",
			Type:     model.SkillTypePrompt,
			Trigger:  "/review",
		},
		{
			Name:     "reviewer",
			Platform: model.ClaudeCode,
			Path:     sourceFile,
			Content:  "Review agent",
		},
	}

	result, err := s.SyncWithSkills(skills, model.Copilot, Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		TargetPath: targetDir,
	})
	if err != nil {
		t.Fatalf("SyncWithSkills failed: %v", err)
	}

	if len(result.Skills) != 4 {
		t.Fatalf("expected 4 results, got %d", len(result.Skills))
	}

	for _, path := range []string{
		filepath.Join(targetDir, "copilot-instructions.md"),
		filepath.Join(targetDir, "instructions", "go-style.instructions.md"),
		filepath.Join(targetDir, "prompts", "review.prompt.md"),
		filepath.Join(targetDir, "agents", "reviewer.agent.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected Copilot artifact at %s: %v", path, err)
		}
	}
}

func TestSynchronizer_Sync_SkipsNestedSkillDuplicates(t *testing.T) {
	s := New()
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	parentDir := filepath.Join(sourceDir, "cursor")
	nestedDir := filepath.Join(parentDir, "skills", "cursor-hooks")
	if err := os.MkdirAll(nestedDir, 0o750); err != nil {
		t.Fatalf("failed to create nested skill directory: %v", err)
	}

	parentSkill := `---
name: cursor
description: parent skill
---
Parent content.`
	if err := os.WriteFile(filepath.Join(parentDir, "SKILL.md"), []byte(parentSkill), 0o600); err != nil {
		t.Fatalf("failed to write parent SKILL.md: %v", err)
	}

	nestedSkill := `---
name: cursor-hooks
description: nested skill
---
Nested content.`
	if err := os.WriteFile(filepath.Join(nestedDir, "SKILL.md"), []byte(nestedSkill), 0o600); err != nil {
		t.Fatalf("failed to write nested SKILL.md: %v", err)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Codex, opts)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	var nestedSkipped bool
	for _, skill := range result.Skills {
		if skill.Skill.Name == "cursor-hooks" && skill.Action == ActionSkipped {
			nestedSkipped = true
			break
		}
	}
	if !nestedSkipped {
		t.Fatalf("expected nested cursor-hooks skill to be skipped, got results: %+v", result.Skills)
	}

	copiedNestedPath := filepath.Join(targetDir, "cursor", "skills", "cursor-hooks", "SKILL.md")
	if _, err := os.Stat(copiedNestedPath); err != nil {
		t.Fatalf("expected nested skill to exist under parent copy, stat failed: %v", err)
	}

	duplicatedTopLevelPath := filepath.Join(targetDir, "cursor-hooks")
	if _, err := os.Stat(duplicatedTopLevelPath); !os.IsNotExist(err) {
		t.Fatalf("expected no duplicated top-level cursor-hooks directory, got err=%v", err)
	}
}

func TestSynchronizer_SyncWithSkills_SkipsNestedSkillDuplicates(t *testing.T) {
	s := New()
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	parentDir := filepath.Join(sourceDir, "cursor")
	nestedDir := filepath.Join(parentDir, "skills", "cursor-hooks")
	if err := os.MkdirAll(nestedDir, 0o750); err != nil {
		t.Fatalf("failed to create nested skill directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(parentDir, "SKILL.md"), []byte(`---
name: cursor
description: parent skill
---
Parent content.`), 0o600); err != nil {
		t.Fatalf("failed to write parent SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "SKILL.md"), []byte(`---
name: cursor-hooks
description: nested skill
---
Nested content.`), 0o600); err != nil {
		t.Fatalf("failed to write nested SKILL.md: %v", err)
	}

	skills, err := skillparser.New(sourceDir, model.ClaudeCode).Parse()
	if err != nil {
		t.Fatalf("failed to parse skills: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 parsed skills (parent + nested), got %d", len(skills))
	}

	result, err := s.SyncWithSkills(skills, model.Codex, Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		TargetPath: targetDir,
	})
	if err != nil {
		t.Fatalf("SyncWithSkills failed: %v", err)
	}

	var nestedSkipped bool
	for _, skill := range result.Skills {
		if skill.Skill.Name == "cursor-hooks" && skill.Action == ActionSkipped {
			nestedSkipped = true
			break
		}
	}
	if !nestedSkipped {
		t.Fatalf("expected nested cursor-hooks skill to be skipped, got results: %+v", result.Skills)
	}

	copiedNestedPath := filepath.Join(targetDir, "cursor", "skills", "cursor-hooks", "SKILL.md")
	if _, err := os.Stat(copiedNestedPath); err != nil {
		t.Fatalf("expected nested skill to exist under parent copy, stat failed: %v", err)
	}

	duplicatedTopLevelPath := filepath.Join(targetDir, "cursor-hooks")
	if _, err := os.Stat(duplicatedTopLevelPath); !os.IsNotExist(err) {
		t.Fatalf("expected no duplicated top-level cursor-hooks directory, got err=%v", err)
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

func TestMappingWarning(t *testing.T) {
	tests := map[string]struct {
		skill      model.Skill
		target     model.Platform
		contains   []string
		notContain []string
	}{
		"non-prompt has no warning": {
			skill: model.Skill{
				Name: "plain-skill",
				Type: model.SkillTypeSkill,
			},
			target:   model.Codex,
			contains: nil,
		},
		"prompt to codex warns about trigger semantics": {
			skill: model.Skill{
				Name:    "review",
				Type:    model.SkillTypePrompt,
				Trigger: "/review",
			},
			target:   model.Codex,
			contains: []string{"lossy mapping: prompt trigger semantics"},
		},
		"prompt with argument hint warns for non-claude target": {
			skill: model.Skill{
				Name: "review",
				Type: model.SkillTypePrompt,
				Metadata: map[string]string{
					"argument-hint": "<path>",
				},
			},
			target:   model.Cursor,
			contains: []string{"lossy mapping: argument-hint preserved as metadata only"},
		},
		"prompt to claude has no non-portable warning": {
			skill: model.Skill{
				Name:    "review",
				Type:    model.SkillTypePrompt,
				Trigger: "/review",
			},
			target:     model.ClaudeCode,
			contains:   nil,
			notContain: []string{"lossy mapping"},
		},
		"copilot instruction applyTo preserved as metadata outside cursor": {
			skill: model.Skill{
				Name: "react-standards",
				Metadata: map[string]string{
					"applyTo": "**/*.tsx",
				},
			},
			target:   model.Codex,
			contains: []string{"lossy mapping: applyTo preserved as metadata only"},
		},
		"copilot per-skill model preserved as metadata outside claude": {
			skill: model.Skill{
				Name: "review",
				Metadata: map[string]string{
					"model": "GPT-4o",
				},
			},
			target:   model.Codex,
			contains: []string{"lossy mapping: model preserved as metadata only"},
		},
		"copilot agent-only fields warn when dropped": {
			skill: model.Skill{
				Name: "reviewer",
				Metadata: map[string]string{
					"handoffs":    "[{\"agent\":\"implementer\"}]",
					"target":      "vscode",
					"mcp-servers": "{\"github\":{\"command\":\"gh\"}}",
				},
			},
			target: model.Codex,
			contains: []string{
				"lossy mapping: handoffs dropped without target equivalent",
				"lossy mapping: target dropped without target equivalent",
				"lossy mapping: mcp-servers dropped without target equivalent",
			},
		},
		"tools preserved as metadata for pidev": {
			skill: model.Skill{
				Name:  "portable-skill",
				Tools: []string{"Read", "Write"},
			},
			target:   model.PiDev,
			contains: []string{"lossy mapping: allowed-tools preserved as metadata only"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			msg := mappingWarning(tt.skill, tt.target)
			for _, want := range tt.contains {
				if !strings.Contains(msg, want) {
					t.Errorf("mappingWarning() = %q, want substring %q", msg, want)
				}
			}
			for _, avoid := range tt.notContain {
				if strings.Contains(msg, avoid) {
					t.Errorf("mappingWarning() = %q, should not contain %q", msg, avoid)
				}
			}
			if len(tt.contains) == 0 && msg != "" {
				t.Errorf("mappingWarning() = %q, want empty", msg)
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

func TestSynchronizer_Sync_PiAgentToClaudeCode(t *testing.T) {
	s := New()
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	skillDir := filepath.Join(sourceDir, "example")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: example
description: A Pi Agent skill
---

# Example
`), 0o600); err != nil {
		t.Fatalf("failed to write skill: %v", err)
	}

	result, err := s.Sync(model.PiAgent, model.ClaudeCode, Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 synced skill, got %d", len(result.Skills))
	}
	if result.Skills[0].Action != ActionCreated {
		t.Fatalf("expected created action, got %s", result.Skills[0].Action)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "example", "SKILL.md")); err != nil {
		t.Fatalf("expected synced file at targetDir/example/SKILL.md: %v", err)
	}

	// #nosec G304 - targetDir is a test-controlled temp path
	got, err := os.ReadFile(filepath.Join(targetDir, "example", "SKILL.md"))
	if err != nil {
		t.Fatalf("failed to read synced SKILL.md: %v", err)
	}
	const wantContent = "---\nname: example\ndescription: A Pi Agent skill\n---\n\n# Example\n"
	if string(got) != wantContent {
		t.Fatalf("synced SKILL.md content mismatch:\ngot:  %q\nwant: %q", string(got), wantContent)
	}
}
