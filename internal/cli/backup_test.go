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
