package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/sync"
	"github.com/klauern/skillsync/internal/ui/tui"
)

func TestSyncSkillsInteractive_SelectsSubset(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	setPlatformPaths(t, sourceDir, targetDir)

	writeSkill(t, sourceDir, "skill-one.md", "skill-one", "Skill one", "# One\n\nSource one.")
	writeSkill(t, sourceDir, "skill-two.md", "skill-two", "Skill two", "# Two\n\nSource two.")

	sourceSkills, err := parsePlatformSkillsWithScope(model.ClaudeCode, nil, false)
	if err != nil {
		t.Fatalf("failed to parse source skills: %v", err)
	}

	selected := findSkillByName(sourceSkills, "skill-one")
	if selected.Name == "" {
		t.Fatalf("failed to locate selected skill in source list")
	}

	originalRunSyncList := runSyncList
	t.Cleanup(func() { runSyncList = originalRunSyncList })
	runSyncList = func(_ []model.Skill, _ model.Platform, _ model.Platform) (tui.SyncListResult, error) {
		return tui.SyncListResult{
			Action:         tui.SyncActionSync,
			SelectedSkills: []model.Skill{selected},
		}, nil
	}

	sourceSpec, err := model.ParsePlatformSpec("claudecode")
	if err != nil {
		t.Fatalf("failed to parse source spec: %v", err)
	}
	targetSpec, err := model.ParsePlatformSpec("cursor")
	if err != nil {
		t.Fatalf("failed to parse target spec: %v", err)
	}

	cfg := &syncConfig{
		sourceSpec:   sourceSpec,
		targetSpec:   targetSpec,
		strategy:     sync.StrategyOverwrite,
		skipBackup:   true,
		sourceSkills: sourceSkills,
	}

	if err := syncSkillsInteractive(cfg); err != nil {
		t.Fatalf("syncSkillsInteractive failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "skill-one.md")); err != nil {
		t.Fatalf("expected selected skill to be synced: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "skill-two.md")); !os.IsNotExist(err) {
		t.Fatalf("expected unselected skill to remain absent")
	}
}

func TestSyncDeleteInteractive_SelectsSubset(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	setPlatformPaths(t, sourceDir, targetDir)

	writeSkill(t, sourceDir, "skill-one.md", "skill-one", "Skill one", "# One\n\nSource one.")
	writeSkill(t, sourceDir, "skill-two.md", "skill-two", "Skill two", "# Two\n\nSource two.")

	writeSkill(t, targetDir, "skill-one.md", "skill-one", "Skill one", "# One\n\nTarget one.")
	writeSkill(t, targetDir, "skill-two.md", "skill-two", "Skill two", "# Two\n\nTarget two.")
	writeSkill(t, targetDir, "other.md", "other", "Other skill", "# Other\n\nTarget other.")

	sourceSkills, err := parsePlatformSkillsWithScope(model.ClaudeCode, nil, false)
	if err != nil {
		t.Fatalf("failed to parse source skills: %v", err)
	}

	originalRunDeleteList := runDeleteList
	t.Cleanup(func() { runDeleteList = originalRunDeleteList })
	runDeleteList = func(skills []model.Skill) (tui.DeleteListResult, error) {
		for _, skill := range skills {
			if skill.Name == "other" {
				t.Fatalf("unexpected non-matching skill in delete list")
			}
		}
		var selected []model.Skill
		for _, skill := range skills {
			if skill.Name == "skill-two" {
				selected = append(selected, skill)
			}
		}
		return tui.DeleteListResult{
			Action:         tui.DeleteActionDelete,
			SelectedSkills: selected,
		}, nil
	}

	sourceSpec, err := model.ParsePlatformSpec("claudecode")
	if err != nil {
		t.Fatalf("failed to parse source spec: %v", err)
	}
	targetSpec, err := model.ParsePlatformSpec("cursor")
	if err != nil {
		t.Fatalf("failed to parse target spec: %v", err)
	}

	cfg := &syncConfig{
		sourceSpec:   sourceSpec,
		targetSpec:   targetSpec,
		skipBackup:   true,
		deleteMode:   true,
		sourceSkills: sourceSkills,
	}

	if err := syncDeleteInteractive(cfg); err != nil {
		t.Fatalf("syncDeleteInteractive failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "skill-two.md")); !os.IsNotExist(err) {
		t.Fatalf("expected selected skill to be deleted")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "skill-one.md")); err != nil {
		t.Fatalf("expected unselected skill to remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "other.md")); err != nil {
		t.Fatalf("expected unrelated skill to remain: %v", err)
	}
}

func setPlatformPaths(t *testing.T, claudePath, cursorPath string) {
	t.Helper()
	t.Setenv("SKILLSYNC_CLAUDE_CODE_PATH", claudePath)
	t.Setenv("SKILLSYNC_CURSOR_PATH", cursorPath)
	t.Setenv("SKILLSYNC_CLAUDE_CODE_SKILLS_PATHS", claudePath)
	t.Setenv("SKILLSYNC_CURSOR_SKILLS_PATHS", cursorPath)
}

func writeSkill(t *testing.T, dir, filename, name, description, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}
	content := "---\n"
	content += "name: " + name + "\n"
	if description != "" {
		content += "description: " + description + "\n"
	}
	content += "---\n\n"
	content += body

	fullPath := filepath.Join(dir, filename)
	if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}
}

func findSkillByName(skills []model.Skill, name string) model.Skill {
	for _, skill := range skills {
		if skill.Name == name {
			return skill
		}
	}
	return model.Skill{}
}
