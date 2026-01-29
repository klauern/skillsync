package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/backup"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/validation"
)

func TestConfigCommand(t *testing.T) {
	tests := map[string]struct {
		args       []string
		wantErr    bool
		wantOutput string
	}{
		"config show default": {
			args:       []string{"skillsync", "config"},
			wantErr:    false,
			wantOutput: "skillsync configuration",
		},
		"config show subcommand": {
			args:       []string{"skillsync", "config", "show"},
			wantErr:    false,
			wantOutput: "skillsync configuration",
		},
		"config show yaml format": {
			args:       []string{"skillsync", "config", "show", "--format", "yaml"},
			wantErr:    false,
			wantOutput: "skillsync configuration",
		},
		"config show json format": {
			args:       []string{"skillsync", "config", "show", "--format", "json"},
			wantErr:    false,
			wantOutput: "Platforms",
		},
		"config show short flag": {
			args:       []string{"skillsync", "config", "show", "-f", "json"},
			wantErr:    false,
			wantOutput: "Platforms",
		},
		"config show invalid format": {
			args:    []string{"skillsync", "config", "show", "--format", "invalid"},
			wantErr: true,
		},
		"config path subcommand": {
			args:       []string{"skillsync", "config", "path"},
			wantErr:    false,
			wantOutput: "Configuration paths",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run command
			ctx := context.Background()
			err := Run(ctx, tt.args)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}
			output := buf.String()

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check output if no error expected
			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("Run() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestConfigInitCommand(t *testing.T) {
	tests := map[string]struct {
		setup      func(t *testing.T) string
		args       []string
		wantErr    bool
		wantOutput string
	}{
		"init creates config in temp dir": {
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				t.Setenv("HOME", tempDir)
				return tempDir
			},
			args:       []string{"skillsync", "config", "init"},
			wantErr:    false,
			wantOutput: "Created config file",
		},
		"init fails without force when config exists": {
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				t.Setenv("HOME", tempDir)
				// Create existing config
				configDir := filepath.Join(tempDir, ".skillsync")
				if err := os.MkdirAll(configDir, 0o750); err != nil {
					t.Fatalf("failed to create config dir: %v", err)
				}
				configPath := filepath.Join(configDir, "config.yaml")
				if err := os.WriteFile(configPath, []byte("existing: config"), 0o600); err != nil {
					t.Fatalf("failed to write existing config: %v", err)
				}
				return tempDir
			},
			args:    []string{"skillsync", "config", "init"},
			wantErr: true,
		},
		"init with force overwrites existing config": {
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				t.Setenv("HOME", tempDir)
				// Create existing config
				configDir := filepath.Join(tempDir, ".skillsync")
				if err := os.MkdirAll(configDir, 0o750); err != nil {
					t.Fatalf("failed to create config dir: %v", err)
				}
				configPath := filepath.Join(configDir, "config.yaml")
				if err := os.WriteFile(configPath, []byte("existing: config"), 0o600); err != nil {
					t.Fatalf("failed to write existing config: %v", err)
				}
				return tempDir
			},
			args:       []string{"skillsync", "config", "init", "--force"},
			wantErr:    false,
			wantOutput: "Created config file",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_ = tt.setup(t)

			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run command
			ctx := context.Background()
			err := Run(ctx, tt.args)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}
			output := buf.String()

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check output if no error expected
			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("Run() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestConfigEditCommand(t *testing.T) {
	tests := map[string]struct {
		setup      func(t *testing.T)
		args       []string
		wantErr    bool
		wantOutput string
	}{
		"edit with EDITOR set": {
			setup: func(t *testing.T) {
				tempDir := t.TempDir()
				t.Setenv("HOME", tempDir)
				t.Setenv("EDITOR", "vim")
			},
			args:       []string{"skillsync", "config", "edit"},
			wantErr:    false,
			wantOutput: "Run:",
		},
		"edit with VISUAL set": {
			setup: func(t *testing.T) {
				tempDir := t.TempDir()
				t.Setenv("HOME", tempDir)
				t.Setenv("EDITOR", "")
				t.Setenv("VISUAL", "code")
			},
			args:       []string{"skillsync", "config", "edit"},
			wantErr:    false,
			wantOutput: "Run:",
		},
		"edit without editor set": {
			setup: func(t *testing.T) {
				tempDir := t.TempDir()
				t.Setenv("HOME", tempDir)
				t.Setenv("EDITOR", "")
				t.Setenv("VISUAL", "")
			},
			args:    []string{"skillsync", "config", "edit"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.setup(t)

			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run command
			ctx := context.Background()
			err := Run(ctx, tt.args)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}
			output := buf.String()

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check output if no error expected
			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("Run() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestDiscoveryCommand(t *testing.T) {
	// Set up isolated test environment - both HOME and working directory
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Change to temp directory to avoid picking up repo-local skills
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	tests := map[string]struct {
		args       []string
		wantErr    bool
		wantOutput string
	}{
		"discover with no skills": {
			args:       []string{"skillsync", "discover", "--no-plugins"},
			wantErr:    false,
			wantOutput: "No skills found",
		},
		"discover with alias list": {
			args:       []string{"skillsync", "list", "--no-plugins"},
			wantErr:    false,
			wantOutput: "No skills found",
		},
		"discover with alias discovery": {
			args:       []string{"skillsync", "discovery", "--no-plugins"},
			wantErr:    false,
			wantOutput: "No skills found",
		},
		"discover json format empty": {
			args:       []string{"skillsync", "discover", "--no-plugins", "--format", "json"},
			wantErr:    false,
			wantOutput: "null", // JSON encoder outputs null for empty/nil slices
		},
		"discover yaml format empty": {
			args:       []string{"skillsync", "discover", "--no-plugins", "--format", "yaml"},
			wantErr:    false,
			wantOutput: "[]",
		},
		"discover invalid format": {
			args:    []string{"skillsync", "discover", "--format", "invalid"},
			wantErr: true,
		},
		"discover invalid platform": {
			args:    []string{"skillsync", "discover", "--platform", "invalid"},
			wantErr: true,
		},
		"discover valid platform": {
			args:       []string{"skillsync", "discover", "--platform", "cursor", "--no-plugins"},
			wantErr:    false,
			wantOutput: "No skills found",
		},
		"discover invalid scope": {
			args:    []string{"skillsync", "discover", "--scope", "invalid"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run command
			ctx := context.Background()
			err := Run(ctx, tt.args)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}
			output := buf.String()

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check output if no error expected
			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("Run() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestExportCommand(t *testing.T) {
	// Set up isolated test environment - both HOME and working directory
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Change to temp directory to avoid picking up repo-local skills
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	tests := map[string]struct {
		args       []string
		wantErr    bool
		wantOutput string
	}{
		"export with no skills": {
			args:       []string{"skillsync", "export"},
			wantErr:    false,
			wantOutput: "No skills found",
		},
		"export json format": {
			args:       []string{"skillsync", "export", "--format", "json"},
			wantErr:    false,
			wantOutput: "No skills found",
		},
		"export yaml format": {
			args:       []string{"skillsync", "export", "--format", "yaml"},
			wantErr:    false,
			wantOutput: "No skills found",
		},
		"export invalid format": {
			args:    []string{"skillsync", "export", "--format", "invalid"},
			wantErr: true,
		},
		"export invalid platform": {
			args:    []string{"skillsync", "export", "--platform", "invalid"},
			wantErr: true,
		},
		"export valid platform": {
			args:       []string{"skillsync", "export", "--platform", "cursor"},
			wantErr:    false,
			wantOutput: "No skills found",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			rOut, wOut, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()
			os.Stdout = wOut
			os.Stderr = wErr

			// Run command
			ctx := context.Background()
			err := Run(ctx, tt.args)

			// Restore stdout and stderr
			if err := wOut.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			if err := wErr.Close(); err != nil {
				t.Fatalf("failed to close stderr pipe writer: %v", err)
			}
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			// Read captured output
			var bufOut, bufErr bytes.Buffer
			if _, err := io.Copy(&bufOut, rOut); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}
			if _, err := io.Copy(&bufErr, rErr); err != nil {
				t.Fatalf("failed to read captured stderr: %v", err)
			}
			output := bufOut.String() + bufErr.String()

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check output if no error expected
			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("Run() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestExportToFile(t *testing.T) {
	// Set up isolated test environment - both HOME and working directory
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	outputFile := filepath.Join(tempDir, "export.json")

	// Change to temp directory to avoid picking up repo-local skills
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	// Capture stderr (where "No skills found" goes)
	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	ctx := context.Background()
	err = Run(ctx, []string{"skillsync", "export", "--output", outputFile})

	if closeErr := wErr.Close(); closeErr != nil {
		t.Fatalf("failed to close stderr pipe writer: %v", closeErr)
	}
	os.Stderr = oldStderr

	// Drain the pipe
	var bufErr bytes.Buffer
	if _, copyErr := io.Copy(&bufErr, rErr); copyErr != nil {
		t.Fatalf("failed to read captured stderr: %v", copyErr)
	}

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
}

func TestBackupRestoreCommand(t *testing.T) {
	tests := map[string]struct {
		args    []string
		wantErr bool
	}{
		"restore without backup ID": {
			args:    []string{"skillsync", "backup", "restore"},
			wantErr: true,
		},
		"restore with non-existent ID": {
			args:    []string{"skillsync", "backup", "restore", "non-existent-id"},
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

func TestBackupVerifyCommand(t *testing.T) {
	tests := map[string]struct {
		args       []string
		wantErr    bool
		wantOutput string
	}{
		"verify all with no backups": {
			args:       []string{"skillsync", "backup", "verify"},
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"verify with platform filter": {
			args:       []string{"skillsync", "backup", "verify", "--platform", "cursor"},
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"verify specific non-existent ID": {
			args:    []string{"skillsync", "backup", "verify", "non-existent-id"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			ctx := context.Background()
			err := Run(ctx, tt.args)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}
			output := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("Run() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestOutputSkills(t *testing.T) {
	tests := map[string]struct {
		skills     []model.Skill
		format     string
		wantErr    bool
		wantOutput string
	}{
		"empty skills table": {
			skills:     []model.Skill{},
			format:     "table",
			wantErr:    false,
			wantOutput: "No skills found",
		},
		"empty skills json": {
			skills:     []model.Skill{},
			format:     "json",
			wantErr:    false,
			wantOutput: "[]",
		},
		"empty skills yaml": {
			skills:     []model.Skill{},
			format:     "yaml",
			wantErr:    false,
			wantOutput: "[]",
		},
		"skills with data table": {
			skills: []model.Skill{
				{Name: "test-skill", Platform: model.Cursor, Description: "Test description"},
			},
			format:     "table",
			wantErr:    false,
			wantOutput: "test-skill",
		},
		"skills with data json": {
			skills: []model.Skill{
				{Name: "test-skill", Platform: model.Cursor, Description: "Test description"},
			},
			format:     "json",
			wantErr:    false,
			wantOutput: "test-skill",
		},
		"skills with data yaml": {
			skills: []model.Skill{
				{Name: "test-skill", Platform: model.Cursor, Description: "Test description"},
			},
			format:     "yaml",
			wantErr:    false,
			wantOutput: "test-skill",
		},
		"invalid format": {
			skills:  []model.Skill{},
			format:  "invalid",
			wantErr: true,
		},
		"skill with long name truncation": {
			skills: []model.Skill{
				{Name: "this-is-a-very-long-skill-name-that-should-be-truncated", Platform: model.Cursor},
			},
			format:     "table",
			wantErr:    false,
			wantOutput: "...",
		},
		"skill with long description truncation": {
			skills: []model.Skill{
				{Name: "test", Platform: model.Cursor, Description: "This is a very long description that should definitely be truncated when displayed in table format"},
			},
			format:     "table",
			wantErr:    false,
			wantOutput: "...",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := outputSkills(tt.skills, tt.format)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}
			output := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("outputSkills() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("outputSkills() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestColorPlatform(t *testing.T) {
	tests := map[string]struct {
		platform string
		contains string
	}{
		"claudecode platform": {
			platform: "claudecode",
			contains: "claudecode",
		},
		"cursor platform": {
			platform: "cursor",
			contains: "cursor",
		},
		"codex platform": {
			platform: "codex",
			contains: "codex",
		},
		"unknown platform": {
			platform: "unknown",
			contains: "unknown",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := colorPlatform(tt.platform, 12)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("colorPlatform(%q, 12) = %q, want to contain %q", tt.platform, result, tt.contains)
			}
		})
	}
}

func TestOutputBackups(t *testing.T) {
	now := time.Now()
	tests := map[string]struct {
		backups    []backup.Metadata
		format     string
		wantErr    bool
		wantOutput string
	}{
		"empty backups table": {
			backups:    []backup.Metadata{},
			format:     "table",
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"empty backups json": {
			backups:    []backup.Metadata{},
			format:     "json",
			wantErr:    false,
			wantOutput: "[]",
		},
		"empty backups yaml": {
			backups:    []backup.Metadata{},
			format:     "yaml",
			wantErr:    false,
			wantOutput: "[]",
		},
		"backups with data table": {
			backups: []backup.Metadata{
				{ID: "test-backup-123", Platform: "cursor", SourcePath: "/path/to/skill", Size: 1024, CreatedAt: now},
			},
			format:     "table",
			wantErr:    false,
			wantOutput: "test-backup-123",
		},
		"backups with data json": {
			backups: []backup.Metadata{
				{ID: "test-backup-123", Platform: "cursor", SourcePath: "/path/to/skill", Size: 1024, CreatedAt: now},
			},
			format:     "json",
			wantErr:    false,
			wantOutput: "test-backup-123",
		},
		"backups with data yaml": {
			backups: []backup.Metadata{
				{ID: "test-backup-123", Platform: "cursor", SourcePath: "/path/to/skill", Size: 1024, CreatedAt: now},
			},
			format:     "yaml",
			wantErr:    false,
			wantOutput: "test-backup-123",
		},
		"invalid format": {
			backups: []backup.Metadata{},
			format:  "invalid",
			wantErr: true,
		},
		"backup with long source path truncation": {
			backups: []backup.Metadata{
				{ID: "test-id", Platform: "cursor", SourcePath: "/this/is/a/very/long/path/to/a/skill/file/that/should/be/truncated", Size: 1024, CreatedAt: now},
			},
			format:     "table",
			wantErr:    false,
			wantOutput: "...",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := outputBackups(tt.backups, tt.format)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}
			output := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("outputBackups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("outputBackups() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestFormatValidationError(t *testing.T) {
	skills := []model.Skill{
		{Name: "test-skill", Platform: model.Cursor},
	}

	tests := map[string]struct {
		err        error
		wantSubstr string
	}{
		"validation error with empty name": {
			err:        &validation.Error{Field: "skills[0].name", Message: "skill name cannot be empty"},
			wantSubstr: "ensure each skill file has a name",
		},
		"validation error with duplicate name": {
			err:        &validation.Error{Field: "skills", Message: "duplicate skill name found"},
			wantSubstr: "rename one of the conflicting skills",
		},
		"validation error with file access": {
			err:        &validation.Error{Field: "skills[0].path", Message: "cannot access skill file"},
			wantSubstr: "check file path and permissions",
		},
		"generic error": {
			err:        &validation.Error{Field: "other", Message: "some error"},
			wantSubstr: "other: some error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatValidationError(tt.err, skills)
			if !strings.Contains(result, tt.wantSubstr) {
				t.Errorf("formatValidationError() = %q, want substring %q", result, tt.wantSubstr)
			}
		})
	}
}

func TestCheckWritePermission(t *testing.T) {
	tests := map[string]struct {
		setup   func(t *testing.T) string
		wantErr bool
	}{
		"writable directory": {
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
		},
		"non-existent path falls back to current dir": {
			setup: func(_ *testing.T) string {
				return "/non/existent/path"
			},
			wantErr: false, // Falls back to "." which should be writable
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			path := tt.setup(t)
			err := checkWritePermission(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkWritePermission(%q) error = %v, wantErr %v", path, err, tt.wantErr)
			}
		})
	}
}

func TestListBackups(t *testing.T) {
	tests := map[string]struct {
		platform   string
		format     string
		limit      int
		wantErr    bool
		wantOutput string
	}{
		"list all with no backups": {
			platform:   "",
			format:     "table",
			limit:      0,
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"list with platform filter": {
			platform:   "cursor",
			format:     "table",
			limit:      0,
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"list with limit": {
			platform:   "",
			format:     "table",
			limit:      5,
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"list json format": {
			platform:   "",
			format:     "json",
			limit:      0,
			wantErr:    false,
			wantOutput: "[]",
		},
		"list invalid format": {
			platform: "",
			format:   "invalid",
			limit:    0,
			wantErr:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := listBackups(tt.platform, tt.format, tt.limit)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}
			output := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("listBackups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("listBackups() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestVerifyAllBackups(t *testing.T) {
	tests := map[string]struct {
		platform   string
		wantErr    bool
		wantOutput string
	}{
		"verify all with no backups": {
			platform:   "",
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"verify with platform filter no backups": {
			platform:   "cursor",
			wantErr:    false,
			wantOutput: "No backups found",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := verifyAllBackups(tt.platform)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}
			output := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("verifyAllBackups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("verifyAllBackups() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestVerifyBackupsByID(t *testing.T) {
	tests := map[string]struct {
		ids     []string
		wantErr bool
	}{
		"verify non-existent ID": {
			ids:     []string{"non-existent-id"},
			wantErr: true,
		},
		"verify multiple non-existent IDs": {
			ids:     []string{"id-1", "id-2"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := verifyBackupsByID(tt.ids)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Drain the pipe
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("verifyBackupsByID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteBackupsByPolicy(t *testing.T) {
	tests := map[string]struct {
		olderThan  string
		keepLatest int
		platform   string
		force      bool
		wantErr    bool
		wantOutput string
	}{
		"delete older than with no backups": {
			olderThan:  "30d",
			keepLatest: 0,
			platform:   "",
			force:      true,
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"delete keep latest with no backups": {
			olderThan:  "",
			keepLatest: 5,
			platform:   "",
			force:      true,
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"delete with invalid duration": {
			olderThan: "invalid",
			force:     true,
			wantErr:   true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := deleteBackupsByPolicy(tt.olderThan, tt.keepLatest, tt.platform, tt.force)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}
			output := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("deleteBackupsByPolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("deleteBackupsByPolicy() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestDeleteBackupsByID(t *testing.T) {
	tests := map[string]struct {
		ids     []string
		force   bool
		wantErr bool
	}{
		"delete non-existent ID": {
			ids:     []string{"non-existent-id"},
			force:   true,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := deleteBackupsByID(tt.ids, tt.force)
			if (err != nil) != tt.wantErr {
				t.Errorf("deleteBackupsByID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSyncCommandArguments(t *testing.T) {
	tests := map[string]struct {
		args    []string
		wantErr bool
	}{
		"same source and target platform": {
			args:    []string{"skillsync", "sync", "cursor", "cursor"},
			wantErr: true,
		},
		"invalid strategy": {
			args:    []string{"skillsync", "sync", "--strategy", "invalid", "cursor", "codex"},
			wantErr: true,
		},
		"invalid source scope in spec": {
			args:    []string{"skillsync", "sync", "cursor:invalid", "codex"},
			wantErr: true,
		},
		"invalid target scope in spec": {
			args:    []string{"skillsync", "sync", "cursor", "codex:admin"},
			wantErr: true,
		},
		"valid source scope in spec": {
			args:    []string{"skillsync", "sync", "--skip-validation", "--yes", "cursor:user", "codex"},
			wantErr: false,
		},
		"valid target scope user in spec": {
			args:    []string{"skillsync", "sync", "--skip-validation", "--yes", "cursor", "codex:user"},
			wantErr: false,
		},
		"valid multiple source scopes in spec": {
			args:    []string{"skillsync", "sync", "--skip-validation", "--yes", "cursor:user,repo", "codex"},
			wantErr: false,
		},
		"invalid multiple target scopes": {
			args:    []string{"skillsync", "sync", "cursor", "codex:user,repo"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			ctx := context.Background()
			err := Run(ctx, tt.args)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Drain the pipe
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTargetPath(t *testing.T) {
	// Create a temp directory for testing
	tempDir := t.TempDir()

	tests := map[string]struct {
		setup   func() string
		wantErr bool
	}{
		"existing writable directory": {
			setup: func() string {
				dir := filepath.Join(tempDir, "existing")
				if err := os.MkdirAll(dir, 0o750); err != nil {
					t.Fatalf("failed to create test dir: %v", err)
				}
				return dir
			},
			wantErr: false,
		},
		"non-existing with writable parent": {
			setup: func() string {
				parent := filepath.Join(tempDir, "writable-parent")
				if err := os.MkdirAll(parent, 0o750); err != nil {
					t.Fatalf("failed to create test dir: %v", err)
				}
				return filepath.Join(parent, "new-dir")
			},
			wantErr: false,
		},
		"non-existing with missing parent but writable ancestor": {
			setup: func() string {
				ancestor := filepath.Join(tempDir, "ancestor")
				if err := os.MkdirAll(ancestor, 0o750); err != nil {
					t.Fatalf("failed to create test dir: %v", err)
				}
				return filepath.Join(ancestor, "missing-parent", "child")
			},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			path := tt.setup()

			// Set environment variable to override platform path
			t.Setenv("SKILLSYNC_CURSOR_PATH", path)

			err := validateTargetPath(model.Cursor)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateTargetPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInferScopeForPath(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Create a mock repo root
	repoRoot := filepath.Join(tempDir, "myrepo")
	if err := os.MkdirAll(repoRoot, 0o750); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}

	// Create a mock plugin cache directory
	pluginCachePath := filepath.Join(tempDir, ".claude", "plugins", "cache")
	if err := os.MkdirAll(pluginCachePath, 0o750); err != nil {
		t.Fatalf("failed to create plugin cache: %v", err)
	}

	tests := map[string]struct {
		path      string
		repoRoot  string
		wantScope model.SkillScope
	}{
		"repo path": {
			path:      filepath.Join(repoRoot, ".claude", "skills"),
			repoRoot:  repoRoot,
			wantScope: model.ScopeRepo,
		},
		"user path": {
			path:      filepath.Join(tempDir, ".claude", "skills"),
			repoRoot:  "",
			wantScope: model.ScopeUser,
		},
		"plugin cache path": {
			path:      pluginCachePath,
			repoRoot:  "",
			wantScope: model.ScopePlugin,
		},
		"plugin cache subdir": {
			path:      filepath.Join(pluginCachePath, "beads-marketplace", "beads", "0.49.0"),
			repoRoot:  "",
			wantScope: model.ScopePlugin,
		},
		"plugin cache takes precedence over user": {
			// Plugin cache is under home directory but should be detected as plugin scope
			path:      filepath.Join(pluginCachePath, "some-plugin"),
			repoRoot:  "",
			wantScope: model.ScopePlugin,
		},
		"system path": {
			path:      "/etc/skillsync/skills",
			repoRoot:  "",
			wantScope: model.ScopeSystem,
		},
		"admin path": {
			path:      "/opt/skillsync/skills",
			repoRoot:  "",
			wantScope: model.ScopeAdmin,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := inferScopeForPath(tt.path, tt.repoRoot)
			if got != tt.wantScope {
				t.Errorf("inferScopeForPath(%q, %q) = %q, want %q", tt.path, tt.repoRoot, got, tt.wantScope)
			}
		})
	}
}

func TestParsePlatformSkillsFromPathsWithPluginScope(t *testing.T) {
	// Set up isolated test environment
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Create user skills directory with a skill
	userSkillsDir := filepath.Join(tempDir, ".claude", "skills", "my-skill")
	if err := os.MkdirAll(userSkillsDir, 0o750); err != nil {
		t.Fatalf("failed to create user skills dir: %v", err)
	}
	userSkillContent := `---
name: my-skill
description: A user skill
---
# My Skill
This is a user skill.
`
	if err := os.WriteFile(filepath.Join(userSkillsDir, "SKILL.md"), []byte(userSkillContent), 0o600); err != nil {
		t.Fatalf("failed to write user skill: %v", err)
	}

	tests := map[string]struct {
		paths          []string
		scopeFilter    []model.SkillScope
		platform       model.Platform
		includePlugins bool
		wantScopes     []model.SkillScope
	}{
		"user scope filter excludes plugins": {
			paths:          []string{filepath.Join(tempDir, ".claude", "skills")},
			scopeFilter:    []model.SkillScope{model.ScopeUser},
			platform:       model.ClaudeCode,
			includePlugins: false,
			wantScopes:     []model.SkillScope{model.ScopeUser},
		},
		"plugin scope filter includes only plugins": {
			paths:       []string{filepath.Join(tempDir, ".claude", "skills")},
			scopeFilter: []model.SkillScope{model.ScopePlugin},
			platform:    model.ClaudeCode,
			// Note: This will return no skills since there's no real plugin cache set up
			// The test validates that user skills are excluded when only plugin scope is requested
			includePlugins: false,
			wantScopes:     []model.SkillScope{},
		},
		"no filter excludes plugins by default": {
			paths:          []string{filepath.Join(tempDir, ".claude", "skills")},
			scopeFilter:    nil,
			platform:       model.ClaudeCode,
			includePlugins: false,
			// With includePlugins=false, plugins are excluded even with no scope filter
			wantScopes: []model.SkillScope{model.ScopeUser},
		},
		"no filter with includePlugins includes user scope": {
			paths:          []string{filepath.Join(tempDir, ".claude", "skills")},
			scopeFilter:    nil,
			platform:       model.ClaudeCode,
			includePlugins: true,
			// With includePlugins=true, plugins would be included (but no plugin cache in test)
			// User scope skills are always included
			wantScopes: []model.SkillScope{model.ScopeUser},
		},
		"non-claude platform ignores plugins": {
			paths:          []string{filepath.Join(tempDir, ".cursor", "rules")},
			scopeFilter:    []model.SkillScope{model.ScopePlugin},
			platform:       model.Cursor,
			includePlugins: false,
			wantScopes:     []model.SkillScope{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			skills := parsePlatformSkillsFromPaths(tt.platform, tt.paths, "", tt.scopeFilter, tt.includePlugins)

			// Verify all returned skills have expected scopes
			for _, skill := range skills {
				found := false
				for _, wantScope := range tt.wantScopes {
					if skill.Scope == wantScope {
						found = true
						break
					}
				}
				if !found && len(tt.wantScopes) > 0 {
					t.Errorf("skill %q has scope %q, want one of %v", skill.Name, skill.Scope, tt.wantScopes)
				}
			}

			// If we expect no scopes, verify no skills returned
			if len(tt.wantScopes) == 0 && len(skills) > 0 {
				// This is acceptable - we may still get skills from plugin cache
				// but they should have plugin scope
				for _, skill := range skills {
					if skill.Scope != model.ScopePlugin {
						t.Errorf("expected only plugin scope skills or empty, got scope %q for skill %q", skill.Scope, skill.Name)
					}
				}
			}
		})
	}
}
