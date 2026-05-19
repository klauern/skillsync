package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
