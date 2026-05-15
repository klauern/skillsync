package e2e_test

import (
	"testing"

	"github.com/klauern/skillsync/internal/e2e"
)

// TestBackupListEmpty verifies backup list with no backups.
func TestBackupListEmpty(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("backup", "list")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "No backups found")
}

// TestBackupListFormats verifies backup list output formats.
func TestBackupListFormats(t *testing.T) {
	tests := map[string]struct {
		format string
		want   string
	}{
		"table format": {
			format: "table",
			want:   "No backups found",
		},
		"json format": {
			format: "json",
			want:   "[]",
		},
		"yaml format": {
			format: "yaml",
			want:   "[]",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := e2e.NewHarness(t)

			result := h.Run("backup", "list", "--format", tt.format)

			e2e.AssertSuccess(t, result)
			e2e.AssertOutputContains(t, result, tt.want)
		})
	}
}

// TestBackupCreateCommand verifies manual backup creation works.
func TestBackupCreateCommand(t *testing.T) {
	h := e2e.NewHarness(t)

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("manual-backup.md", "manual-backup", "Manual backup", "# Manual Backup\n\nContent.")

	result := h.Run("backup", "create", "--platform", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")

	backupResult := h.Run("backup", "list", "--platform", "cursor")
	e2e.AssertSuccess(t, backupResult)
	e2e.AssertOutputNotContains(t, backupResult, "No backups found")
}

// TestSyncCreatesBackupByDefault verifies sync creates backup when not skipped.
func TestSyncCreatesBackupByDefault(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create a skill in target that will be overwritten
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("backup-test.md", "backup-test", "Original", "# Backup Test\n\nOriginal content to backup.")

	// Create source skill
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("backup-test.md", "backup-test", "Updated", "# Backup Test\n\nUpdated content.")

	// Run sync WITHOUT --skip-backup
	result := h.Run("sync", "--yes", "--skip-validation", "--strategy", "overwrite", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify file was updated
	e2e.AssertFileContains(t, cursorFixture.Path("backup-test.md"), "Updated content")

	// Check if backup was mentioned in output (backup list should show something)
	backupResult := h.Run("backup", "list")
	e2e.AssertSuccess(t, backupResult)
	e2e.AssertOutputNotContains(t, backupResult, "No backups found")
}

// TestSyncSkipBackupFlag verifies --skip-backup prevents backup creation.
func TestSyncSkipBackupFlag(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create a skill in target
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("no-backup-test.md", "no-backup-test", "Original", "# No Backup\n\nOriginal content.")

	// Create source skill
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("no-backup-test.md", "no-backup-test", "Updated", "# No Backup\n\nUpdated content.")

	// Run sync WITH --skip-backup
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "overwrite", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify file was updated
	e2e.AssertFileContains(t, cursorFixture.Path("no-backup-test.md"), "Updated content")

	// Verify no backup was created
	backupResult := h.Run("backup", "list")
	e2e.AssertSuccess(t, backupResult)
	e2e.AssertOutputContains(t, backupResult, "No backups found")
}
