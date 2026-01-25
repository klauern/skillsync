package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func TestVersionVariables(t *testing.T) {
	// Version should be set (even if to "dev")
	if Version == "" {
		t.Error("Version should not be empty")
	}

	// Commit and BuildDate should have defaults
	if Commit == "" {
		t.Error("Commit should not be empty")
	}
	if BuildDate == "" {
		t.Error("BuildDate should not be empty")
	}
}

func TestSyncCommand(t *testing.T) {
	tests := map[string]struct {
		args       []string
		wantErr    bool
		wantOutput string
	}{
		"valid sync": {
			args:       []string{"skillsync", "sync", "claudecode", "cursor"},
			wantErr:    false,
			wantOutput: "Syncing from claude-code to cursor",
		},
		"valid sync with dry-run": {
			args:       []string{"skillsync", "sync", "--dry-run", "cursor", "codex"},
			wantErr:    false,
			wantOutput: "DRY RUN: Would sync from cursor to codex",
		},
		"sync with short dry-run flag": {
			args:       []string{"skillsync", "sync", "-d", "claudecode", "cursor"},
			wantErr:    false,
			wantOutput: "DRY RUN",
		},
		"missing target argument": {
			args:    []string{"skillsync", "sync", "claudecode"},
			wantErr: true,
		},
		"missing both arguments": {
			args:    []string{"skillsync", "sync"},
			wantErr: true,
		},
		"too many arguments": {
			args:    []string{"skillsync", "sync", "claudecode", "cursor", "codex"},
			wantErr: true,
		},
		"invalid source platform": {
			args:    []string{"skillsync", "sync", "invalid", "cursor"},
			wantErr: true,
		},
		"invalid target platform": {
			args:    []string{"skillsync", "sync", "cursor", "invalid"},
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
