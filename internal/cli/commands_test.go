package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/validation"
)

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
		"claude-code platform": {
			platform: "claude-code",
			contains: "claude-code",
		},
		"cursor platform": {
			platform: "cursor",
			contains: "cursor",
		},
		"codex platform": {
			platform: "codex",
			contains: "codex",
		},
		"pi.dev platform": {
			platform: "pi.dev",
			contains: "pi.dev",
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
		"non-existent path returns error": {
			setup: func(_ *testing.T) string {
				return "/non/existent/path"
			},
			wantErr: true, // Non-existent paths should return an error
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
