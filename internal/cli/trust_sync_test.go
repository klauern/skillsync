package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncDeleteDoesNotRemoveOrphansAfterTrustFailure(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "cursor", "unsafe")
	targetDir := filepath.Join(root, "codex")
	if err := os.MkdirAll(sourceDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(targetDir, "orphan"), 0o750); err != nil {
		t.Fatal(err)
	}

	writeSkill(t, sourceDir, "SKILL.md", "unsafe", "Unsafe skill", "Blocked by runtime trust.")
	if err := os.WriteFile(filepath.Join(sourceDir, "run.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeSkill(t, filepath.Join(targetDir, "orphan"), "SKILL.md", "orphan", "Orphan", "Keep me.")

	t.Setenv("SKILLSYNC_CURSOR_PATH", filepath.Join(root, "cursor"))
	t.Setenv("SKILLSYNC_CURSOR_SKILLS_PATHS", filepath.Join(root, "cursor"))
	t.Setenv("SKILLSYNC_CODEX_PATH", targetDir)

	output := captureStdout(t, func() {
		err := Run(context.Background(), []string{
			"skillsync", "sync", "--delete", "--yes", "--skip-backup", "--skip-validation",
			"cursor", "codex",
		})
		if err == nil {
			t.Fatal("expected trust failure")
		}
	})
	if output == "" {
		t.Fatal("expected sync failure summary output")
	}

	orphan := filepath.Join(targetDir, "orphan", "SKILL.md")
	if _, err := os.Stat(orphan); err != nil {
		t.Fatalf("orphan was removed after trust failure: %v", err)
	}
}
