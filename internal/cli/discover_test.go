package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
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

func TestOnboardCommand(t *testing.T) {
	tests := map[string]struct {
		args       []string
		wantErr    bool
		wantOutput []string
	}{
		"onboard command": {
			args:    []string{"skillsync", "onboard"},
			wantErr: false,
			wantOutput: []string{
				"SkillSync",
				"Quick start",
				"Common workflows",
			},
		},
		"onboard alias llm": {
			args:    []string{"skillsync", "llm"},
			wantErr: false,
			wantOutput: []string{
				"SkillSync",
				"Quick start",
				"Common workflows",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			ctx := context.Background()
			err := Run(ctx, tt.args)

			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}
			output := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				for _, want := range tt.wantOutput {
					if !strings.Contains(output, want) {
						t.Errorf("Run() output = %q, want substring %q", output, want)
					}
				}
			}
		})
	}
}
