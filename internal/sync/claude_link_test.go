package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/model"
)

func TestSynchronizer_Sync_ClaudeDirectorySkillsLinkCompatibleTargets(t *testing.T) {
	s := New()

	targets := []model.Platform{model.Codex, model.Cursor, model.PiDev}
	for _, target := range targets {
		t.Run(string(target), func(t *testing.T) {
			sourceDir, skillDir := writeClaudeDirectorySkill(t, "linked-skill")
			targetDir := t.TempDir()

			result, err := s.Sync(model.ClaudeCode, target, Options{
				DryRun:     false,
				Strategy:   StrategyOverwrite,
				SourcePath: sourceDir,
				TargetPath: targetDir,
			})
			if err != nil {
				t.Fatalf("Sync failed: %v", err)
			}
			if len(result.Skills) != 1 {
				t.Fatalf("expected 1 skill result, got %d", len(result.Skills))
			}

			sr := result.Skills[0]
			if sr.Action != ActionCreated {
				t.Fatalf("expected created action, got %s", sr.Action)
			}
			if !strings.Contains(sr.Message, "linked Claude skill directory") {
				t.Fatalf("expected linked message, got %q", sr.Message)
			}

			targetSkillPath := filepath.Join(targetDir, "linked-skill")
			assertSymlinkTarget(t, targetSkillPath, skillDir)
		})
	}
}

func TestSynchronizer_Sync_ClaudeDirectorySkillDryRun(t *testing.T) {
	s := New()

	sourceDir, _ := writeClaudeDirectorySkill(t, "dry-run-linked")
	targetDir := t.TempDir()

	result, err := s.Sync(model.ClaudeCode, model.Cursor, Options{
		DryRun:     true,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill result, got %d", len(result.Skills))
	}

	sr := result.Skills[0]
	if sr.TargetPath != filepath.Join(targetDir, "dry-run-linked") {
		t.Fatalf("TargetPath = %q, want %q", sr.TargetPath, filepath.Join(targetDir, "dry-run-linked"))
	}
	if sr.Action != ActionCreated {
		t.Fatalf("expected created action, got %s", sr.Action)
	}
	if !strings.Contains(sr.Message, "linked Claude skill directory") {
		t.Fatalf("expected linked message, got %q", sr.Message)
	}

	if _, err := os.Lstat(filepath.Join(targetDir, "dry-run-linked")); !os.IsNotExist(err) {
		t.Fatalf("expected no target entry to be created in dry-run, got err=%v", err)
	}
}

func TestSynchronizer_Sync_ClaudeDirectorySkillMixedCaseEntrypointCopiesCanonicalFile(t *testing.T) {
	s := New()
	sourceDir, skillDir := writeClaudeDirectorySkill(t, "mixed-case-linked")
	if err := os.Rename(filepath.Join(skillDir, "SKILL.md"), filepath.Join(skillDir, "SkIlL.Md")); err != nil {
		t.Fatalf("failed to rename entrypoint: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "reference.md"), []byte("reference"), 0o600); err != nil {
		t.Fatalf("failed to write companion file: %v", err)
	}
	targetDir := t.TempDir()

	result, err := s.Sync(model.ClaudeCode, model.Codex, Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if len(result.Skills) != 1 || result.Skills[0].Action != ActionCreated {
		t.Fatalf("expected one created skill, got %#v", result.Skills)
	}

	targetSkillDir := filepath.Join(targetDir, "mixed-case-linked")
	entrypoint := filepath.Join(targetSkillDir, "SKILL.md")
	info, err := os.Lstat(entrypoint)
	if err != nil {
		t.Fatalf("failed to stat canonical entrypoint: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("expected canonical entrypoint to be copied, not linked")
	}
	entries, err := os.ReadDir(targetSkillDir)
	if err != nil {
		t.Fatalf("failed to read target skill directory: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() == "SkIlL.Md" {
			t.Fatal("case-variant entrypoint should not be copied")
		}
	}
	if _, err := os.Stat(filepath.Join(targetSkillDir, "reference.md")); err != nil {
		t.Fatalf("expected companion file to be copied: %v", err)
	}
}

func TestSynchronizer_Sync_ClaudeDirectorySkillOverwriteAndSkipExistingEntries(t *testing.T) {
	s := New()

	tests := []struct {
		name         string
		strategy     Strategy
		existingKind string
		setup        func(t *testing.T, existingPath string) string
		assert       func(t *testing.T, existingPath, sourceSkillDir, existingTarget string)
	}{
		{
			name:         "overwrite existing directory",
			strategy:     StrategyOverwrite,
			existingKind: "directory",
			setup: func(t *testing.T, existingPath string) string {
				if err := os.MkdirAll(existingPath, 0o750); err != nil {
					t.Fatalf("failed to create existing directory: %v", err)
				}
				if err := os.WriteFile(filepath.Join(existingPath, "old.txt"), []byte("old"), 0o600); err != nil {
					t.Fatalf("failed to write existing file: %v", err)
				}
				return ""
			},
			assert: func(t *testing.T, existingPath, sourceSkillDir, existingTarget string) {
				assertSymlinkTarget(t, existingPath, sourceSkillDir)
			},
		},
		{
			name:         "overwrite existing file",
			strategy:     StrategyOverwrite,
			existingKind: "file",
			setup: func(t *testing.T, existingPath string) string {
				if err := os.WriteFile(existingPath, []byte("old content"), 0o600); err != nil {
					t.Fatalf("failed to create existing file: %v", err)
				}
				return ""
			},
			assert: func(t *testing.T, existingPath, sourceSkillDir, existingTarget string) {
				assertSymlinkTarget(t, existingPath, sourceSkillDir)
			},
		},
		{
			name:         "overwrite existing symlink",
			strategy:     StrategyOverwrite,
			existingKind: "symlink",
			setup: func(t *testing.T, existingPath string) string {
				externalDir := filepath.Join(t.TempDir(), "external-skill")
				if err := os.MkdirAll(externalDir, 0o750); err != nil {
					t.Fatalf("failed to create external dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(externalDir, "SKILL.md"), []byte("---\nname: existing-skill\n---\nold"), 0o600); err != nil {
					t.Fatalf("failed to create external SKILL.md: %v", err)
				}
				if err := os.Symlink(externalDir, existingPath); err != nil {
					t.Fatalf("failed to create existing symlink: %v", err)
				}
				return externalDir
			},
			assert: func(t *testing.T, existingPath, sourceSkillDir, existingTarget string) {
				assertSymlinkTarget(t, existingPath, sourceSkillDir)
			},
		},
		{
			name:         "skip existing directory",
			strategy:     StrategySkip,
			existingKind: "directory",
			setup: func(t *testing.T, existingPath string) string {
				if err := os.MkdirAll(existingPath, 0o750); err != nil {
					t.Fatalf("failed to create existing directory: %v", err)
				}
				if err := os.WriteFile(filepath.Join(existingPath, "keep.txt"), []byte("keep"), 0o600); err != nil {
					t.Fatalf("failed to write existing file: %v", err)
				}
				return ""
			},
			assert: func(t *testing.T, existingPath, sourceSkillDir, existingTarget string) {
				info, err := os.Stat(existingPath)
				if err != nil {
					t.Fatalf("expected directory to remain, stat failed: %v", err)
				}
				if !info.IsDir() {
					t.Fatalf("expected existing path to remain a directory")
				}
				if _, err := os.Stat(filepath.Join(existingPath, "keep.txt")); err != nil {
					t.Fatalf("expected existing directory contents to remain: %v", err)
				}
			},
		},
		{
			name:         "skip existing file",
			strategy:     StrategySkip,
			existingKind: "file",
			setup: func(t *testing.T, existingPath string) string {
				if err := os.WriteFile(existingPath, []byte("keep"), 0o600); err != nil {
					t.Fatalf("failed to create existing file: %v", err)
				}
				return ""
			},
			assert: func(t *testing.T, existingPath, sourceSkillDir, existingTarget string) {
				// #nosec G304 - existingPath is a test-controlled temp path.
				content, err := os.ReadFile(existingPath)
				if err != nil {
					t.Fatalf("expected file to remain, read failed: %v", err)
				}
				if string(content) != "keep" {
					t.Fatalf("expected existing file content to remain, got %q", string(content))
				}
			},
		},
		{
			name:         "skip existing symlink",
			strategy:     StrategySkip,
			existingKind: "symlink",
			setup: func(t *testing.T, existingPath string) string {
				externalDir := filepath.Join(t.TempDir(), "external-skill")
				if err := os.MkdirAll(externalDir, 0o750); err != nil {
					t.Fatalf("failed to create external dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(externalDir, "SKILL.md"), []byte("---\nname: existing-skill\n---\nold"), 0o600); err != nil {
					t.Fatalf("failed to create external SKILL.md: %v", err)
				}
				if err := os.Symlink(externalDir, existingPath); err != nil {
					t.Fatalf("failed to create existing symlink: %v", err)
				}
				return externalDir
			},
			assert: func(t *testing.T, existingPath, sourceSkillDir, existingTarget string) {
				assertSymlinkTarget(t, existingPath, existingTarget)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceDir, skillDir := writeClaudeDirectorySkill(t, "existing-skill")
			targetDir := t.TempDir()
			targetPath := filepath.Join(targetDir, "existing-skill")

			existingTarget := tt.setup(t, targetPath)

			result, err := s.Sync(model.ClaudeCode, model.Cursor, Options{
				DryRun:     false,
				Strategy:   tt.strategy,
				SourcePath: sourceDir,
				TargetPath: targetDir,
			})
			if err != nil {
				t.Fatalf("Sync failed: %v", err)
			}
			if len(result.Skills) != 1 {
				t.Fatalf("expected 1 skill result, got %d", len(result.Skills))
			}

			sr := result.Skills[0]
			switch tt.strategy {
			case StrategySkip:
				if sr.Action != ActionSkipped {
					t.Fatalf("expected skipped action, got %s", sr.Action)
				}
				if strings.Contains(sr.Message, "linked Claude skill directory") {
					t.Fatalf("skip result should not report a link: %q", sr.Message)
				}
			default:
				if sr.Action != ActionUpdated {
					t.Fatalf("expected updated action, got %s", sr.Action)
				}
				if !strings.Contains(sr.Message, "linked Claude skill directory") {
					t.Fatalf("expected linked message, got %q", sr.Message)
				}
			}

			tt.assert(t, targetPath, skillDir, existingTarget)
		})
	}
}

func TestSynchronizer_Sync_ClaudeFlatAndPromptArtifactsRemainTransforms(t *testing.T) {
	s := New()

	t.Run("flat skill stays a file", func(t *testing.T) {
		sourceFile := filepath.Join(t.TempDir(), "flat-skill.md")
		if err := os.WriteFile(sourceFile, []byte(`---
name: flat-skill
---

Flat content.`), 0o600); err != nil {
			t.Fatalf("failed to write source file: %v", err)
		}

		targetDir := t.TempDir()
		result, err := s.SyncWithSkills([]model.Skill{{
			Name:     "flat-skill",
			Platform: model.ClaudeCode,
			Path:     sourceFile,
			Content:  "Flat content.",
		}}, model.Cursor, Options{
			DryRun:     false,
			Strategy:   StrategyOverwrite,
			TargetPath: targetDir,
		})
		if err != nil {
			t.Fatalf("SyncWithSkills failed: %v", err)
		}
		if len(result.Skills) != 1 {
			t.Fatalf("expected 1 skill result, got %d", len(result.Skills))
		}
		targetFile := filepath.Join(targetDir, "flat-skill.md")
		info, err := os.Lstat(targetFile)
		if err != nil {
			t.Fatalf("failed to stat target file: %v", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("expected flat skill to remain a regular file")
		}
	})

	t.Run("prompt artifact stays transformed", func(t *testing.T) {
		sourceFile := filepath.Join(t.TempDir(), "review.md")
		if err := os.WriteFile(sourceFile, []byte(`---
description: review prompt
---

Review this.`), 0o600); err != nil {
			t.Fatalf("failed to write source file: %v", err)
		}

		targetDir := t.TempDir()
		result, err := s.SyncWithSkills([]model.Skill{{
			Name:     "review",
			Platform: model.ClaudeCode,
			Path:     sourceFile,
			Content:  "Review this.",
			Type:     model.SkillTypePrompt,
			Trigger:  "/review",
		}}, model.Codex, Options{
			DryRun:     false,
			Strategy:   StrategyOverwrite,
			TargetPath: targetDir,
		})
		if err != nil {
			t.Fatalf("SyncWithSkills failed: %v", err)
		}
		if len(result.Skills) != 1 {
			t.Fatalf("expected 1 skill result, got %d", len(result.Skills))
		}

		targetFile := filepath.Join(targetDir, "review", "SKILL.md")
		info, err := os.Lstat(targetFile)
		if err != nil {
			t.Fatalf("failed to stat target file: %v", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("expected prompt artifact to remain a regular file")
		}
	})
}

func writeClaudeDirectorySkill(t *testing.T, name string) (string, string) {
	t.Helper()

	sourceDir := t.TempDir()
	skillDir := filepath.Join(sourceDir, name)
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	content := "---\nname: " + name + "\ndescription: linked directory skill\n---\n\nSkill content."
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Make the directory mtime stable enough for newer-strategy fallback tests.
	now := time.Now()
	if err := os.Chtimes(skillDir, now, now); err != nil {
		t.Fatalf("failed to update skill directory mtime: %v", err)
	}
	if err := os.Chtimes(filepath.Join(skillDir, "SKILL.md"), now, now); err != nil {
		t.Fatalf("failed to update SKILL.md mtime: %v", err)
	}

	return sourceDir, skillDir
}

func assertSymlinkTarget(t *testing.T, path, want string) {
	t.Helper()
	// On macOS, t.TempDir() lives under /var which is itself a symlink to /private/var.
	// os.Readlink returns the path as written at symlink-creation time, not the resolved path.
	// These assertions therefore rely on processSkill NOT calling filepath.EvalSymlinks on
	// the source before symlinking. If that contract changes, update callers to pass
	// filepath.EvalSymlinks(want) instead.

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("failed to lstat %s: %v", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to be a symlink", path)
	}

	got, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("failed to read symlink target for %s: %v", path, err)
	}
	if got != want {
		t.Fatalf("symlink target = %q, want %q", got, want)
	}
}
