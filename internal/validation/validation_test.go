package validation

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestValidateSourceTarget_Valid(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")

	// #nosec G301 - test directory permissions are acceptable
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	// #nosec G301 - test directory permissions are acceptable
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	// Create a test skill file
	skillPath := filepath.Join(sourceDir, "test.md")
	// #nosec G306 - test file permissions are acceptable
	if err := os.WriteFile(skillPath, []byte("# Test Skill"), 0o644); err != nil {
		t.Fatalf("failed to create test skill: %v", err)
	}

	skills := []model.Skill{
		{
			Name:     "test",
			Platform: model.ClaudeCode,
			Path:     skillPath,
			Content:  "# Test Skill",
			Metadata: make(map[string]string),
		},
	}

	// Mock platform paths - we'll validate paths exist
	opts := DefaultOptions()
	opts.CheckConflicts = false // Skip conflict check for this test

	// For now, test with paths that exist
	sourcePlatform := model.ClaudeCode
	targetPlatform := model.Cursor

	// Set temp paths for testing (this would normally use util paths)
	result, err := ValidateSourceTarget(sourcePlatform, targetPlatform, skills, opts)
	if err == nil && !result.Valid {
		t.Errorf("expected validation to pass or have warnings, got errors: %v", result.Errors)
	}
	if result != nil && result.HasErrors() {
		for _, e := range result.Errors {
			t.Logf("Validation error (expected for mock paths): %v", e)
		}
	}
}

func TestValidateSkill_EmptyName(t *testing.T) {
	skill := model.Skill{
		Name:     "",
		Platform: model.ClaudeCode,
		Path:     "/some/path.md",
	}

	opts := DefaultOptions()
	err := validateSkill(skill, 0, opts)

	if err == nil {
		t.Error("expected error for skill with empty name")
	}

	var vErr *Error
	if ok := errors.As(err, &vErr); !ok {
		t.Errorf("expected validation.Error, got %T", err)
	} else if vErr.Field != "skills[0].name" {
		t.Errorf("expected field 'skills[0].name', got %q", vErr.Field)
	}
}

func TestValidateSkill_EmptyPath(t *testing.T) {
	skill := model.Skill{
		Name:     "test",
		Platform: model.ClaudeCode,
		Path:     "",
	}

	opts := DefaultOptions()
	err := validateSkill(skill, 0, opts)

	if err == nil {
		t.Error("expected error for skill with empty path")
	}

	var vErr *Error
	if ok := errors.As(err, &vErr); !ok {
		t.Errorf("expected validation.Error, got %T", err)
	}
}

func TestValidateSkill_ValidExtension(t *testing.T) {
	tests := []struct {
		name     string
		platform model.Platform
		path     string
		wantErr  bool
	}{
		{
			name:     "Claude Code .md",
			platform: model.ClaudeCode,
			path:     "/skills/test.md",
			wantErr:  false,
		},
		{
			name:     "Claude Code .txt",
			platform: model.ClaudeCode,
			path:     "/skills/test.txt",
			wantErr:  false,
		},
		{
			name:     "Claude Code no extension",
			platform: model.ClaudeCode,
			path:     "/skills/test",
			wantErr:  false,
		},
		{
			name:     "Claude Code invalid extension",
			platform: model.ClaudeCode,
			path:     "/skills/test.json",
			wantErr:  true,
		},
		{
			name:     "Cursor .md",
			platform: model.Cursor,
			path:     "/skills/test.md",
			wantErr:  false,
		},
		{
			name:     "Cursor .mdc",
			platform: model.Cursor,
			path:     "/skills/test.mdc",
			wantErr:  false,
		},
		{
			name:     "Cursor invalid extension",
			platform: model.Cursor,
			path:     "/skills/test.txt",
			wantErr:  true,
		},
		{
			name:     "Codex .json",
			platform: model.Codex,
			path:     "/skills/test.json",
			wantErr:  false,
		},
		{
			name:     "Codex invalid extension",
			platform: model.Codex,
			path:     "/skills/test.md",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := model.Skill{
				Name:     "test",
				Platform: tt.platform,
				Path:     tt.path,
				Content:  "test content",
			}

			// Create the file so existence check passes
			tmpDir := t.TempDir()
			skillPath := filepath.Join(tmpDir, filepath.Base(tt.path))
			// #nosec G306 - test file permissions are acceptable
			if err := os.WriteFile(skillPath, []byte("test"), 0o644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}
			skill.Path = skillPath

			opts := DefaultOptions()
			err := validateSkill(skill, 0, opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateSkill() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSkill_StrictMode(t *testing.T) {
	t.Run("strict mode requires content", func(t *testing.T) {
		skill := model.Skill{
			Name:     "test",
			Platform: model.ClaudeCode,
			Path:     "/some/path.md",
			Content:  "",
		}

		opts := DefaultOptions()
		opts.StrictMode = true

		err := validateSkill(skill, 0, opts)

		if err == nil {
			t.Error("expected error in strict mode with empty content")
		}
	})

	t.Run("normal mode allows empty content", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillPath := filepath.Join(tmpDir, "test.md")
		// #nosec G306 - test file permissions are acceptable
		if err := os.WriteFile(skillPath, []byte(""), 0o644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		skill := model.Skill{
			Name:     "test",
			Platform: model.ClaudeCode,
			Path:     skillPath,
			Content:  "",
		}

		opts := DefaultOptions()
		opts.StrictMode = false

		err := validateSkill(skill, 0, opts)
		if err != nil {
			t.Errorf("unexpected error in normal mode: %v", err)
		}
	})
}

func TestValidateSkillsFormat_DuplicateNames(t *testing.T) {
	skills := []model.Skill{
		{Name: "test", Platform: model.ClaudeCode, Path: "/a.md", Content: "a"},
		{Name: "test", Platform: model.ClaudeCode, Path: "/b.md", Content: "b"},
	}

	result, err := ValidateSkillsFormat(skills, model.ClaudeCode)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Valid {
		t.Error("expected validation to fail with duplicate names")
	}

	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestValidateSkillsFormat_EmptyName(t *testing.T) {
	skills := []model.Skill{
		{Name: "", Platform: model.ClaudeCode, Path: "/a.md", Content: "a"},
	}

	result, err := ValidateSkillsFormat(skills, model.ClaudeCode)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Valid {
		t.Error("expected validation to fail with empty name")
	}
}

func TestValidateSkillsFormat_EmptyContentWarning(t *testing.T) {
	skills := []model.Skill{
		{Name: "test", Platform: model.ClaudeCode, Path: "/a.md", Content: ""},
	}

	result, err := ValidateSkillsFormat(skills, model.ClaudeCode)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Valid {
		t.Error("expected validation to pass (empty content is a warning)")
	}

	// We get 2 warnings: one for empty content, one for path not accessible
	if len(result.Warnings) < 1 {
		t.Errorf("expected at least 1 warning, got %d", len(result.Warnings))
	}
}

func TestValidateSkillsFormat_NoSkills(t *testing.T) {
	skills := []model.Skill{}

	result, err := ValidateSkillsFormat(skills, model.ClaudeCode)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// With no skills, validation still passes but with a warning
	if !result.Valid {
		t.Error("expected validation to pass with warning")
	}

	if len(result.Warnings) == 0 {
		t.Error("expected warning for no skills")
	}
}

func TestValidatePath_Valid(t *testing.T) {
	tmpDir := t.TempDir()

	err := ValidatePath(tmpDir, model.ClaudeCode)
	if err != nil {
		t.Errorf("unexpected error for valid path: %v", err)
	}
}

func TestValidatePath_NotExist(t *testing.T) {
	err := ValidatePath("/nonexistent/path/that/does/not/exist", model.ClaudeCode)

	if err == nil {
		t.Error("expected error for nonexistent path")
	}

	var vErr *Error
	if ok := errors.As(err, &vErr); !ok {
		t.Errorf("expected validation.Error, got %T", err)
	}
}

func TestValidatePath_NotDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")
	// #nosec G306 - test file permissions are acceptable
	if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err := ValidatePath(filePath, model.ClaudeCode)

	if err == nil {
		t.Error("expected error for file path (not directory)")
	}
}

func TestValidatePath_Empty(t *testing.T) {
	err := ValidatePath("", model.ClaudeCode)

	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestResult_Error(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		result := &Result{Valid: true}
		if err := result.Error(); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("single error", func(t *testing.T) {
		result := &Result{
			Valid:  false,
			Errors: []error{errors.New("test error")},
		}
		if err := result.Error(); err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		result := &Result{
			Valid:  false,
			Errors: []error{errors.New("error 1"), errors.New("error 2")},
		}
		if err := result.Error(); err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestResult_Summary(t *testing.T) {
	t.Run("all valid", func(t *testing.T) {
		result := &Result{Valid: true}
		if summary := result.Summary(); summary != "All validations passed" {
			t.Errorf("unexpected summary: %s", summary)
		}
	})

	t.Run("valid with warnings", func(t *testing.T) {
		result := &Result{
			Valid:    true,
			Warnings: []string{"warning 1", "warning 2"},
		}
		if summary := result.Summary(); summary != "Validation passed with warnings (2 warning(s))" {
			t.Errorf("unexpected summary: %s", summary)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		result := &Result{
			Valid:    false,
			Warnings: []string{"warning 1"},
		}
		if summary := result.Summary(); summary != "Validation failed (1 warning(s))" {
			t.Errorf("unexpected summary: %s", summary)
		}
	})
}

func TestGetPlatformPath(t *testing.T) {
	tests := []struct {
		name     string
		platform model.Platform
		wantErr  bool
	}{
		{
			name:     "Claude Code",
			platform: model.ClaudeCode,
			wantErr:  false,
		},
		{
			name:     "Cursor",
			platform: model.Cursor,
			wantErr:  false,
		},
		{
			name:     "Codex",
			platform: model.Codex,
			wantErr:  false,
		},
		{
			name:     "Invalid platform",
			platform: model.Platform("invalid"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := GetPlatformPath(tt.platform)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetPlatformPath() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && path == "" {
				t.Error("expected non-empty path")
			}
		})
	}
}

func TestGetPlatformPath_CodexDefaultsToUserSkills(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SKILLSYNC_CODEX_PATH", "")

	got, err := GetPlatformPath(model.Codex)
	if err != nil {
		t.Fatalf("GetPlatformPath() error = %v", err)
	}

	expected := filepath.Join(home, ".codex", "skills")
	if got != expected {
		t.Errorf("GetPlatformPath(Codex) = %q, want %q", got, expected)
	}
}

func TestValidationError_Error(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		err := &Error{
			Field:   "test",
			Message: "failed",
			Err:     errors.New("underlying"),
		}
		if msg := err.Error(); msg == "" {
			t.Error("expected non-empty error message")
		}
	})

	t.Run("without underlying error", func(t *testing.T) {
		err := &Error{
			Field:   "test",
			Message: "failed",
		}
		if msg := err.Error(); msg == "" {
			t.Error("expected non-empty error message")
		}
	})
}

func TestValidateWritePermission(t *testing.T) {
	t.Run("writable directory", func(t *testing.T) {
		// Test write permission using a real platform
		// We use ClaudeCode as it has a default path
		t.Skip("requires platform path mocking - tested via integration")
	})

	t.Run("non-writable directory", func(t *testing.T) {
		// Create a read-only directory
		tmpDir := t.TempDir()
		readOnlyDir := filepath.Join(tmpDir, "readonly")
		// #nosec G301 - test directory permissions are acceptable
		if err := os.Mkdir(readOnlyDir, 0o444); err != nil {
			t.Fatalf("failed to create read-only dir: %v", err)
		}

		// This would require mocking platform path
		t.Skip("requires platform path mocking - tested via integration")
	})
}

func TestValidatePlatform_SourceRequiresExistingPath(t *testing.T) {
	tmpDir := t.TempDir()
	missing := filepath.Join(tmpDir, "missing")
	t.Setenv("SKILLSYNC_CLAUDE_CODE_PATH", missing)

	err := validatePlatform(model.ClaudeCode, "source", true)
	if err == nil {
		t.Error("expected error for missing source path")
	}
}

func TestValidatePlatform_TargetAllowsMissingPath(t *testing.T) {
	tmpDir := t.TempDir()
	missing := filepath.Join(tmpDir, "missing")
	t.Setenv("SKILLSYNC_CURSOR_PATH", missing)

	err := validatePlatform(model.Cursor, "target", false)
	if err != nil {
		t.Errorf("expected no error for missing target path, got %v", err)
	}
}

func TestCheckConflicts_TargetFileExists(t *testing.T) {
	targetDir := t.TempDir()
	t.Setenv("SKILLSYNC_CURSOR_PATH", targetDir)

	sourceDir := t.TempDir()
	skillPath := filepath.Join(sourceDir, "skill.md")
	// #nosec G306 - test file permissions are acceptable
	if err := os.WriteFile(skillPath, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	targetFile := filepath.Join(targetDir, "skill.md")
	// #nosec G306 - test file permissions are acceptable
	if err := os.WriteFile(targetFile, []byte("existing"), 0o644); err != nil {
		t.Fatalf("failed to create target file: %v", err)
	}

	skills := []model.Skill{
		{
			Name:     "skill",
			Platform: model.ClaudeCode,
			Path:     skillPath,
		},
	}

	err := checkConflicts(model.ClaudeCode, model.Cursor, skills)
	if err == nil {
		t.Error("expected conflict error when target file exists")
	}
}

func TestValidateWritePermission_WithEnvOverrides(t *testing.T) {
	t.Run("writable directory", func(t *testing.T) {
		writable := t.TempDir()
		t.Setenv("SKILLSYNC_CURSOR_PATH", writable)

		if err := validateWritePermission(model.Cursor); err != nil {
			t.Errorf("expected writable dir to pass, got %v", err)
		}
	})

	t.Run("read-only directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		readOnlyDir := filepath.Join(tmpDir, "readonly")
		// #nosec G301 - test directory permissions are acceptable
		if err := os.MkdirAll(readOnlyDir, 0o755); err != nil {
			t.Fatalf("failed to create read-only dir: %v", err)
		}
		// #nosec G302 - test directory permissions are acceptable
		if err := os.Chmod(readOnlyDir, 0o500); err != nil {
			t.Fatalf("failed to set read-only permissions: %v", err)
		}
		t.Setenv("SKILLSYNC_CURSOR_PATH", readOnlyDir)

		if err := validateWritePermission(model.Cursor); err == nil {
			t.Error("expected error for read-only dir")
		}
	})
}

func TestGetPlatformPathForScope(t *testing.T) {
	t.Run("user scope uses env override", func(t *testing.T) {
		path := t.TempDir()
		t.Setenv("SKILLSYNC_CLAUDE_CODE_PATH", path)

		got, err := GetPlatformPathForScope(model.ClaudeCode, model.ScopeUser)
		if err != nil {
			t.Fatalf("GetPlatformPathForScope() error = %v", err)
		}
		if got != path {
			t.Errorf("expected %q, got %q", path, got)
		}
	})

	t.Run("repo scope uses repo root", func(t *testing.T) {
		repoRoot := t.TempDir()
		// #nosec G301 - test directory permissions are acceptable
		if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
			t.Fatalf("failed to create repo root: %v", err)
		}
		subDir := filepath.Join(repoRoot, "subdir")
		// #nosec G301 - test directory permissions are acceptable
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get cwd: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chdir(cwd)
		})
		if err := os.Chdir(subDir); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}

		got, err := GetPlatformPathForScope(model.ClaudeCode, model.ScopeRepo)
		if err != nil {
			t.Fatalf("GetPlatformPathForScope() error = %v", err)
		}
		resolvedRoot, err := filepath.EvalSymlinks(repoRoot)
		if err != nil {
			t.Fatalf("failed to resolve repo root: %v", err)
		}
		expected := filepath.Join(resolvedRoot, ".claude", "skills")
		if filepath.Clean(got) != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("invalid scope returns error", func(t *testing.T) {
		_, err := GetPlatformPathForScope(model.ClaudeCode, model.SkillScope("invalid"))
		if err == nil {
			t.Error("expected error for invalid scope")
		}
	})
}
