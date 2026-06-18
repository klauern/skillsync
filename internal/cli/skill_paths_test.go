package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillParentDir(t *testing.T) {
	t.Run("skill definition path returns parent directory", func(t *testing.T) {
		path := filepath.Join("tmp", "skill-a", "SKILL.md")

		parentDir, ok := skillParentDir(path)
		if !ok {
			t.Fatal("expected SKILL.md path to resolve to a parent directory")
		}
		if want := filepath.Join("tmp", "skill-a"); parentDir != want {
			t.Fatalf("skillParentDir() = %q, want %q", parentDir, want)
		}
	})

	t.Run("non-skill path does not return a parent directory", func(t *testing.T) {
		parentDir, ok := skillParentDir(filepath.Join("tmp", "skill-a.md"))
		if ok {
			t.Fatalf("skillParentDir() unexpectedly returned %q", parentDir)
		}
	})

	t.Run("bare skill definition filename does not return current directory", func(t *testing.T) {
		parentDir, ok := skillParentDir(skillDefinitionFile)
		if ok {
			t.Fatalf("skillParentDir() unexpectedly returned %q", parentDir)
		}
	})
}

func TestPruneEmptySkillParentDir(t *testing.T) {
	t.Run("removes empty skill directory after skill file deletion", func(t *testing.T) {
		skillDir := filepath.Join(t.TempDir(), "skill-a")
		if err := os.MkdirAll(skillDir, 0o750); err != nil {
			t.Fatalf("failed to create skill directory: %v", err)
		}

		skillPath := filepath.Join(skillDir, skillDefinitionFile)
		if err := os.WriteFile(skillPath, []byte("# Skill A"), 0o644); err != nil {
			t.Fatalf("failed to write skill file: %v", err)
		}
		if err := os.Remove(skillPath); err != nil {
			t.Fatalf("failed to remove skill file: %v", err)
		}

		pruneEmptySkillParentDir(skillPath)

		if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
			t.Fatalf("expected skill directory to be removed, got err=%v", err)
		}
	})

	t.Run("keeps non-empty skill directory", func(t *testing.T) {
		skillDir := filepath.Join(t.TempDir(), "skill-b")
		if err := os.MkdirAll(skillDir, 0o750); err != nil {
			t.Fatalf("failed to create skill directory: %v", err)
		}

		skillPath := filepath.Join(skillDir, skillDefinitionFile)
		if err := os.WriteFile(skillPath, []byte("# Skill B"), 0o644); err != nil {
			t.Fatalf("failed to write skill file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "notes.md"), []byte("keep"), 0o644); err != nil {
			t.Fatalf("failed to write sibling file: %v", err)
		}
		if err := os.Remove(skillPath); err != nil {
			t.Fatalf("failed to remove skill file: %v", err)
		}

		pruneEmptySkillParentDir(skillPath)

		info, err := os.Stat(skillDir)
		if err != nil {
			t.Fatalf("expected skill directory to remain: %v", err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %q to remain a directory", skillDir)
		}
	})
}
