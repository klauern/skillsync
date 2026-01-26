package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/backup"
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
			args:       []string{"skillsync", "sync", "--skip-validation", "--yes", "claudecode", "cursor"},
			wantErr:    false,
			wantOutput: "Synced claude-code -> cursor",
		},
		"valid sync with dry-run": {
			args:       []string{"skillsync", "sync", "--dry-run", "--skip-validation", "cursor", "codex"},
			wantErr:    false,
			wantOutput: "Dry run - no changes made",
		},
		"sync with short dry-run flag": {
			args:       []string{"skillsync", "sync", "-d", "--skip-validation", "claudecode", "cursor"},
			wantErr:    false,
			wantOutput: "Dry run",
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

func TestBackupListCommand(t *testing.T) {
	tests := map[string]struct {
		args       []string
		wantErr    bool
		wantOutput string
	}{
		"backup list with no backups": {
			args:       []string{"skillsync", "backup", "list"},
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"backup list with alias": {
			args:       []string{"skillsync", "backup", "ls"},
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"backup default action lists": {
			args:       []string{"skillsync", "backup"},
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"backup list with platform filter": {
			args:       []string{"skillsync", "backup", "list", "--platform", "claude-code"},
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"backup list with short platform flag": {
			args:       []string{"skillsync", "backup", "list", "-p", "cursor"},
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"backup list with limit": {
			args:       []string{"skillsync", "backup", "list", "--limit", "5"},
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"backup list json format empty": {
			args:       []string{"skillsync", "backup", "list", "--format", "json"},
			wantErr:    false,
			wantOutput: "[]",
		},
		"backup list yaml format empty": {
			args:       []string{"skillsync", "backup", "list", "--format", "yaml"},
			wantErr:    false,
			wantOutput: "[]",
		},
		"backup list invalid format": {
			args:    []string{"skillsync", "backup", "list", "--format", "invalid"},
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

func TestFormatSize(t *testing.T) {
	tests := map[string]struct {
		bytes int64
		want  string
	}{
		"bytes": {
			bytes: 500,
			want:  "500 B",
		},
		"kilobytes": {
			bytes: 1536,
			want:  "1.5 KB",
		},
		"megabytes": {
			bytes: 1572864,
			want:  "1.5 MB",
		},
		"gigabytes": {
			bytes: 1610612736,
			want:  "1.5 GB",
		},
		"zero": {
			bytes: 0,
			want:  "0 B",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := formatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestOutputBackupsTable(t *testing.T) {
	tests := map[string]struct {
		backups    []backup.Metadata
		wantOutput string
	}{
		"empty list": {
			backups:    []backup.Metadata{},
			wantOutput: "No backups found",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := outputBackupsTable(tt.backups)

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

			if err != nil {
				t.Errorf("outputBackupsTable() error = %v", err)
				return
			}

			if !strings.Contains(output, tt.wantOutput) {
				t.Errorf("outputBackupsTable() output = %q, want substring %q", output, tt.wantOutput)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := map[string]struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		"days lowercase": {
			input: "30d",
			want:  30 * 24 * time.Hour,
		},
		"days uppercase": {
			input: "30D",
			want:  30 * 24 * time.Hour,
		},
		"weeks lowercase": {
			input: "2w",
			want:  2 * 7 * 24 * time.Hour,
		},
		"weeks uppercase": {
			input: "2W",
			want:  2 * 7 * 24 * time.Hour,
		},
		"hours": {
			input: "168h",
			want:  168 * time.Hour,
		},
		"minutes": {
			input: "30m",
			want:  30 * time.Minute,
		},
		"complex duration": {
			input: "1h30m",
			want:  1*time.Hour + 30*time.Minute,
		},
		"invalid day format": {
			input:   "abcd",
			wantErr: true,
		},
		"invalid week format": {
			input:   "xyzw",
			wantErr: true,
		},
		"empty string": {
			input:   "",
			wantErr: true,
		},
		"single character": {
			input:   "d",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBackupDeleteCommand(t *testing.T) {
	tests := map[string]struct {
		args       []string
		wantErr    bool
		wantOutput string
	}{
		"delete without args or flags": {
			args:    []string{"skillsync", "backup", "delete"},
			wantErr: true,
		},
		"delete with non-existent backup ID": {
			args:    []string{"skillsync", "backup", "delete", "non-existent-id", "--force"},
			wantErr: true,
		},
		"delete with keep-latest no backups": {
			args:       []string{"skillsync", "backup", "delete", "--keep-latest", "5", "--force"},
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"delete with older-than no backups": {
			args:       []string{"skillsync", "backup", "delete", "--older-than", "30d", "--force"},
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"delete with platform filter no backups": {
			args:       []string{"skillsync", "backup", "delete", "--older-than", "1d", "--platform", "claude-code", "--force"},
			wantErr:    false,
			wantOutput: "No backups found",
		},
		"delete with invalid duration": {
			args:    []string{"skillsync", "backup", "delete", "--older-than", "invalid"},
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
