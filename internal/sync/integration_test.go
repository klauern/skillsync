package sync

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	for i := 0; i < 500; i++ {
		content += `## Section ` + string(rune('A'+i%26)) + `

This is paragraph number ` + string(rune('0'+i%10)) + `. It contains multiple lines
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

func TestIntegration_SyncWithSkills_PluginScope(t *testing.T) {
	s := New()

	targetDir := t.TempDir()

	// Create plugin-scope skills (simulating skills from plugin cache)
	pluginSkills := []model.Skill{
		{
			Name:        "conventional-commits",
			Description: "Create conventional commits",
			Platform:    model.ClaudeCode,
			Scope:       model.ScopePlugin,
			Content:     "# Conventional Commits\n\nHelp create conventional commits.",
			Path:        "/fake/path/conventional-commits/SKILL.md",
			PluginInfo: &model.PluginInfo{
				PluginName:  "commits@klauern-skills",
				Marketplace: "klauern-skills",
				Version:     "1.2.0",
				IsDev:       false,
			},
		},
		{
			Name:        "worktree-manager",
			Description: "Manage git worktrees",
			Platform:    model.ClaudeCode,
			Scope:       model.ScopePlugin,
			Content:     "# Worktree Manager\n\nManage git worktrees.",
			Path:        "/fake/path/worktree/SKILL.md",
			PluginInfo: &model.PluginInfo{
				PluginName:  "worktree@klauern-skills",
				Marketplace: "klauern-skills",
				Version:     "0.1.0",
				IsDev:       false,
			},
		},
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		TargetPath: targetDir,
	}

	result, err := s.SyncWithSkills(pluginSkills, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 2)
	util.AssertEqual(t, result.Success(), true)

	// Verify files were created
	for _, skill := range pluginSkills {
		targetPath := filepath.Join(targetDir, skill.Name+".md")
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			t.Errorf("Expected target file %s to exist", targetPath)
		}
	}

	// Verify skill results have correct info
	for _, sr := range result.Skills {
		if sr.Skill.Scope != model.ScopePlugin {
			t.Errorf("Expected skill %s to have plugin scope, got %s", sr.Skill.Name, sr.Skill.Scope)
		}
	}
}

func TestIntegration_SyncWithSkills_MixedScopes(t *testing.T) {
	s := New()

	targetDir := t.TempDir()

	// Create skills with different scopes
	mixedSkills := []model.Skill{
		{
			Name:     "user-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
			Content:  "# User Skill\n\nUser-defined skill.",
			Path:     "/fake/path/user-skill.md",
		},
		{
			Name:     "plugin-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopePlugin,
			Content:  "# Plugin Skill\n\nPlugin skill.",
			Path:     "/fake/path/plugin-skill/SKILL.md",
			PluginInfo: &model.PluginInfo{
				PluginName:  "test@marketplace",
				Marketplace: "marketplace",
				IsDev:       false,
			},
		},
		{
			Name:     "repo-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeRepo,
			Content:  "# Repo Skill\n\nRepository skill.",
			Path:     "/fake/path/.claude/skills/repo-skill/SKILL.md",
		},
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		TargetPath: targetDir,
	}

	result, err := s.SyncWithSkills(mixedSkills, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 3)

	// Verify each skill maintains its scope
	scopeCounts := make(map[model.SkillScope]int)
	for _, sr := range result.Skills {
		scopeCounts[sr.Skill.Scope]++
	}

	util.AssertEqual(t, scopeCounts[model.ScopeUser], 1)
	util.AssertEqual(t, scopeCounts[model.ScopePlugin], 1)
	util.AssertEqual(t, scopeCounts[model.ScopeRepo], 1)
}

func TestIntegration_SyncWithSkills_DevPluginSymlink(t *testing.T) {
	s := New()

	targetDir := t.TempDir()

	// Create a dev plugin skill (simulating symlinked development skill)
	devPluginSkill := []model.Skill{
		{
			Name:        "dev-commits",
			Description: "Development version of commits skill",
			Platform:    model.ClaudeCode,
			Scope:       model.ScopePlugin,
			Content:     "# Dev Commits\n\nDevelopment version.",
			Path:        "/Users/test/dev/klauern-skills/plugins/commits/SKILL.md",
			PluginInfo: &model.PluginInfo{
				PluginName:    "commits@klauern-skills",
				Marketplace:   "klauern-skills",
				IsDev:         true,
				SymlinkTarget: "../../../klauern-skills/plugins/commits",
				InstallPath:   "/Users/test/dev/klauern-skills/plugins/commits",
			},
		},
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		TargetPath: targetDir,
	}

	result, err := s.SyncWithSkills(devPluginSkill, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 1)

	// Verify the skill was synced with dev plugin info preserved
	sr := result.Skills[0]
	if sr.Skill.PluginInfo == nil {
		t.Fatal("PluginInfo should be preserved")
	}
	if !sr.Skill.PluginInfo.IsDev {
		t.Error("IsDev should be true for dev plugin")
	}
}
