package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

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
