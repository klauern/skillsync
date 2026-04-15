package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/util"
)

func TestDedupeDeleteCommand(t *testing.T) {
	tests := map[string]struct {
		args    []string
		wantErr bool
	}{
		"missing skill name": {
			args:    []string{"skillsync", "dedupe", "delete", "--platform", "cursor", "--scope", "user"},
			wantErr: true,
		},
		"missing platform": {
			args:    []string{"skillsync", "dedupe", "delete", "my-skill", "--scope", "user"},
			wantErr: true,
		},
		"missing scope": {
			args:    []string{"skillsync", "dedupe", "delete", "my-skill", "--platform", "cursor"},
			wantErr: true,
		},
		"invalid platform": {
			args:    []string{"skillsync", "dedupe", "delete", "my-skill", "--platform", "invalid", "--scope", "user"},
			wantErr: true,
		},
		"invalid scope": {
			args:    []string{"skillsync", "dedupe", "delete", "my-skill", "--platform", "cursor", "--scope", "invalid"},
			wantErr: true,
		},
		"non-writable scope admin": {
			args:    []string{"skillsync", "dedupe", "delete", "my-skill", "--platform", "cursor", "--scope", "admin"},
			wantErr: true,
		},
		"non-writable scope system": {
			args:    []string{"skillsync", "dedupe", "delete", "my-skill", "--platform", "cursor", "--scope", "system"},
			wantErr: true,
		},
		"non-existent skill": {
			args:    []string{"skillsync", "dedupe", "delete", "non-existent-skill-xyz", "--platform", "cursor", "--scope", "user"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			err := Run(ctx, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDedupeRenameCommand(t *testing.T) {
	tests := map[string]struct {
		args    []string
		wantErr bool
	}{
		"missing skill names": {
			args:    []string{"skillsync", "dedupe", "rename", "--platform", "cursor", "--scope", "user"},
			wantErr: true,
		},
		"missing new name": {
			args:    []string{"skillsync", "dedupe", "rename", "old-skill", "--platform", "cursor", "--scope", "user"},
			wantErr: true,
		},
		"missing platform": {
			args:    []string{"skillsync", "dedupe", "rename", "old-skill", "new-skill", "--scope", "user"},
			wantErr: true,
		},
		"missing scope": {
			args:    []string{"skillsync", "dedupe", "rename", "old-skill", "new-skill", "--platform", "cursor"},
			wantErr: true,
		},
		"invalid platform": {
			args:    []string{"skillsync", "dedupe", "rename", "old-skill", "new-skill", "--platform", "invalid", "--scope", "user"},
			wantErr: true,
		},
		"invalid scope": {
			args:    []string{"skillsync", "dedupe", "rename", "old-skill", "new-skill", "--platform", "cursor", "--scope", "invalid"},
			wantErr: true,
		},
		"non-writable scope admin": {
			args:    []string{"skillsync", "dedupe", "rename", "old-skill", "new-skill", "--platform", "cursor", "--scope", "admin"},
			wantErr: true,
		},
		"non-writable scope system": {
			args:    []string{"skillsync", "dedupe", "rename", "old-skill", "new-skill", "--platform", "cursor", "--scope", "system"},
			wantErr: true,
		},
		"same name": {
			args:    []string{"skillsync", "dedupe", "rename", "my-skill", "my-skill", "--platform", "cursor", "--scope", "user"},
			wantErr: true,
		},
		"non-existent skill": {
			args:    []string{"skillsync", "dedupe", "rename", "non-existent-skill-xyz", "new-name", "--platform", "cursor", "--scope", "user"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			err := Run(ctx, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDedupeDeleteDryRun(t *testing.T) {
	// Create a temporary directory structure with a skill
	tempDir := t.TempDir()

	// Set up environment to use temp dir for skills
	oldHome := os.Getenv("HOME")
	t.Setenv("HOME", tempDir)
	defer t.Setenv("HOME", oldHome)

	// Create a test skill in user scope for cursor
	skillDir := filepath.Join(tempDir, ".cursor", "skills", "test-delete-skill")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	skillContent := "# Test Skill\nThis is a test skill for deletion."
	// #nosec G306 - test files need to be readable
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to create skill file: %v", err)
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := Run(ctx, []string{"skillsync", "dedupe", "delete", "test-delete-skill", "--platform", "cursor", "--scope", "user", "--dry-run"})

	// Restore stdout
	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe writer: %v", closeErr)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatalf("failed to read output: %v", copyErr)
	}
	output := buf.String()

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	if !strings.Contains(output, "Dry run") {
		t.Errorf("expected dry run message in output, got: %q", output)
	}

	if !strings.Contains(output, "test-delete-skill") {
		t.Errorf("expected skill name in output, got: %q", output)
	}

	// Verify file still exists (dry run shouldn't delete)
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Error("skill file should not have been deleted in dry run mode")
	}
}

func TestDedupeRenameDryRun(t *testing.T) {
	// Create a temporary directory structure with a skill
	tempDir := t.TempDir()

	// Set up environment to use temp dir for skills
	oldHome := os.Getenv("HOME")
	t.Setenv("HOME", tempDir)
	defer t.Setenv("HOME", oldHome)

	// Create a test skill in user scope for cursor
	skillDir := filepath.Join(tempDir, ".cursor", "skills", "test-rename-skill")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	skillContent := "# Test Skill\nThis is a test skill for renaming."
	// #nosec G306 - test files need to be readable
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to create skill file: %v", err)
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := Run(ctx, []string{"skillsync", "dedupe", "rename", "test-rename-skill", "new-test-skill", "--platform", "cursor", "--scope", "user", "--dry-run"})

	// Restore stdout
	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe writer: %v", closeErr)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatalf("failed to read output: %v", copyErr)
	}
	output := buf.String()

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	if !strings.Contains(output, "Dry run") {
		t.Errorf("expected dry run message in output, got: %q", output)
	}

	if !strings.Contains(output, "test-rename-skill") {
		t.Errorf("expected old skill name in output, got: %q", output)
	}

	if !strings.Contains(output, "new-test-skill") {
		t.Errorf("expected new skill name in output, got: %q", output)
	}

	// Verify original file still exists (dry run shouldn't rename)
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Error("skill file should not have been moved in dry run mode")
	}
}

func TestDedupeDeleteWithForce(t *testing.T) {
	// Create a temporary directory structure with a skill
	tempDir := t.TempDir()

	// Set up environment to use temp dir for skills
	oldHome := os.Getenv("HOME")
	t.Setenv("HOME", tempDir)
	defer t.Setenv("HOME", oldHome)

	// Create a test skill in user scope for cursor
	skillDir := filepath.Join(tempDir, ".cursor", "skills", "test-force-delete-skill")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	skillContent := "# Test Skill\nThis is a test skill for force deletion."
	// #nosec G306 - test files need to be readable
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to create skill file: %v", err)
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := Run(ctx, []string{"skillsync", "dedupe", "delete", "test-force-delete-skill", "--platform", "cursor", "--scope", "user", "--force"})

	// Restore stdout
	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe writer: %v", closeErr)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatalf("failed to read output: %v", copyErr)
	}
	output := buf.String()

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	if !strings.Contains(output, "Deleted skill") {
		t.Errorf("expected success message in output, got: %q", output)
	}

	// Verify file was deleted
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Error("skill file should have been deleted with --force flag")
	}
}

func TestDedupeRenameWithForce(t *testing.T) {
	// Create a temporary directory structure with a skill
	tempDir := t.TempDir()

	// Set up environment to use temp dir for skills
	oldHome := os.Getenv("HOME")
	t.Setenv("HOME", tempDir)
	defer t.Setenv("HOME", oldHome)

	// Create a test skill in user scope for cursor
	skillDir := filepath.Join(tempDir, ".cursor", "skills", "test-force-rename-skill")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	skillContent := "# Test Skill\nThis is a test skill for force renaming."
	// #nosec G306 - test files need to be readable
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to create skill file: %v", err)
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := Run(ctx, []string{"skillsync", "dedupe", "rename", "test-force-rename-skill", "renamed-skill", "--platform", "cursor", "--scope", "user", "--force"})

	// Restore stdout
	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe writer: %v", closeErr)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatalf("failed to read output: %v", copyErr)
	}
	output := buf.String()

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	if !strings.Contains(output, "Renamed skill") {
		t.Errorf("expected success message in output, got: %q", output)
	}

	// Verify original file was removed
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Error("original skill file should have been removed after rename")
	}

	// Verify new file exists
	newSkillPath := filepath.Join(tempDir, ".cursor", "skills", "renamed-skill", "SKILL.md")
	if _, err := os.Stat(newSkillPath); os.IsNotExist(err) {
		t.Error("renamed skill file should exist")
	}

	// Verify content was preserved
	// #nosec G304 - newSkillPath is constructed in test from temp dir
	newContent, err := os.ReadFile(newSkillPath)
	if err != nil {
		t.Fatalf("failed to read renamed skill: %v", err)
	}
	if string(newContent) != skillContent {
		t.Errorf("renamed skill content mismatch: got %q, want %q", string(newContent), skillContent)
	}
}

func TestDedupeCommandHelp(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	_ = Run(ctx, []string{"skillsync", "dedupe", "--help"})

	// Restore stdout
	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe writer: %v", closeErr)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatalf("failed to read output: %v", copyErr)
	}
	output := buf.String()

	// Verify help contains expected content
	expectedStrings := []string{
		"dedupe",
		"delete",
		"rename",
		"duplicate",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("expected help output to contain %q, got: %q", expected, output)
		}
	}
}

func TestDedupeRenameConflict(t *testing.T) {
	// Create a temporary directory structure with two skills
	tempDir := t.TempDir()

	// Set up environment to use temp dir for skills
	oldHome := os.Getenv("HOME")
	t.Setenv("HOME", tempDir)
	defer t.Setenv("HOME", oldHome)

	// Create the first skill
	skillDir1 := filepath.Join(tempDir, ".cursor", "skills", "original-skill")
	if err := os.MkdirAll(skillDir1, 0o750); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}
	skillPath1 := filepath.Join(skillDir1, "SKILL.md")
	// #nosec G306 - test files need to be readable
	if err := os.WriteFile(skillPath1, []byte("# Original Skill"), 0o644); err != nil {
		t.Fatalf("failed to create skill file: %v", err)
	}

	// Create the second skill that we'll try to rename to
	skillDir2 := filepath.Join(tempDir, ".cursor", "skills", "conflict-skill")
	if err := os.MkdirAll(skillDir2, 0o750); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}
	skillPath2 := filepath.Join(skillDir2, "SKILL.md")
	// #nosec G306 - test files need to be readable
	if err := os.WriteFile(skillPath2, []byte("# Conflict Skill"), 0o644); err != nil {
		t.Fatalf("failed to create skill file: %v", err)
	}

	ctx := context.Background()

	// Try to rename original-skill to conflict-skill without force - should fail
	err := Run(ctx, []string{"skillsync", "dedupe", "rename", "original-skill", "conflict-skill", "--platform", "cursor", "--scope", "user"})
	if err == nil {
		t.Error("expected error when renaming to existing skill without --force")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

// TestDedupeDeleteRepoScope tests deletion from repo scope
func TestDedupeDeleteRepoScope(t *testing.T) {
	// Create a temporary directory structure with a skill
	tempDir := t.TempDir()

	// Change to temp directory to simulate repo scope
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// Create a test skill in repo scope for claude-code
	skillDir := util.RepoSkillsPath("claude-code", tempDir)
	skillDir = filepath.Join(skillDir, "repo-test-skill")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	skillContent := "# Repo Test Skill\nThis is a test skill in repo scope."
	// #nosec G306 - test files need to be readable
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to create skill file: %v", err)
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err = Run(ctx, []string{"skillsync", "dedupe", "delete", "repo-test-skill", "--platform", "claude-code", "--scope", "repo", "--force"})

	// Restore stdout
	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe writer: %v", closeErr)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatalf("failed to read output: %v", copyErr)
	}
	output := buf.String()

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	if !strings.Contains(output, "Deleted skill") {
		t.Errorf("expected success message in output, got: %q", output)
	}

	// Verify file was deleted
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Error("skill file should have been deleted")
	}
}
