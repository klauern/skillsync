package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/backup"
)

func TestNewBackupListModel(t *testing.T) {
	backups := []backup.Metadata{
		{
			ID:         "20240101-120000-abc12345",
			Platform:   "claude-code",
			SourcePath: "/home/user/.claude/skills/test.md",
			CreatedAt:  time.Now(),
			Size:       1024,
		},
		{
			ID:         "20240102-130000-def67890",
			Platform:   "cursor",
			SourcePath: "/home/user/.cursor/skills/another.md",
			CreatedAt:  time.Now().Add(-24 * time.Hour),
			Size:       2048,
		},
	}

	model := NewBackupListModel(backups)

	if len(model.backups) != 2 {
		t.Errorf("expected 2 backups, got %d", len(model.backups))
	}

	if len(model.filtered) != 2 {
		t.Errorf("expected 2 filtered backups, got %d", len(model.filtered))
	}
}

func TestBackupListModel_Filter(t *testing.T) {
	backups := []backup.Metadata{
		{
			ID:         "20240101-120000-abc12345",
			Platform:   "claude-code",
			SourcePath: "/home/user/.claude/skills/test.md",
			CreatedAt:  time.Now(),
			Size:       1024,
		},
		{
			ID:         "20240102-130000-def67890",
			Platform:   "cursor",
			SourcePath: "/home/user/.cursor/skills/another.md",
			CreatedAt:  time.Now().Add(-24 * time.Hour),
			Size:       2048,
		},
	}

	model := NewBackupListModel(backups)
	model.filter = "claude"
	model.applyFilter()

	if len(model.filtered) != 1 {
		t.Errorf("expected 1 filtered backup, got %d", len(model.filtered))
	}

	if model.filtered[0].Platform != "claude-code" {
		t.Errorf("expected filtered backup to be claude-code, got %s", model.filtered[0].Platform)
	}
}

func TestBackupListModel_ClearFilter(t *testing.T) {
	backups := []backup.Metadata{
		{
			ID:         "20240101-120000-abc12345",
			Platform:   "claude-code",
			SourcePath: "/home/user/.claude/skills/test.md",
			CreatedAt:  time.Now(),
			Size:       1024,
		},
		{
			ID:         "20240102-130000-def67890",
			Platform:   "cursor",
			SourcePath: "/home/user/.cursor/skills/another.md",
			CreatedAt:  time.Now().Add(-24 * time.Hour),
			Size:       2048,
		},
	}

	model := NewBackupListModel(backups)
	model.filter = "claude"
	model.applyFilter()

	if len(model.filtered) != 1 {
		t.Errorf("expected 1 filtered backup, got %d", len(model.filtered))
	}

	// Clear filter
	model.filter = ""
	model.applyFilter()

	if len(model.filtered) != 2 {
		t.Errorf("expected 2 backups after clearing filter, got %d", len(model.filtered))
	}
}

func TestBackupListModel_EmptyBackups(t *testing.T) {
	model := NewBackupListModel([]backup.Metadata{})

	if len(model.backups) != 0 {
		t.Errorf("expected 0 backups, got %d", len(model.backups))
	}

	// View should still work without panicking
	view := model.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestBackupListModel_Init(t *testing.T) {
	backups := []backup.Metadata{
		{
			ID:       "test-backup",
			Platform: "claude-code",
		},
	}

	model := NewBackupListModel(backups)
	cmd := model.Init()

	if cmd != nil {
		t.Error("expected nil command from Init")
	}
}

func TestBackupListModel_QuitKey(t *testing.T) {
	backups := []backup.Metadata{
		{
			ID:       "test-backup",
			Platform: "claude-code",
		},
	}

	model := NewBackupListModel(backups)

	// Simulate pressing 'q'
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	m := newModel.(BackupListModel)
	if !m.quitting {
		t.Error("expected model to be quitting after pressing 'q'")
	}

	// Should return a quit command
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestBackupListModel_HelpToggle(t *testing.T) {
	backups := []backup.Metadata{
		{
			ID:       "test-backup",
			Platform: "claude-code",
		},
	}

	model := NewBackupListModel(backups)

	if model.showHelp {
		t.Error("expected showHelp to be false initially")
	}

	// Simulate pressing '?'
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m := newModel.(BackupListModel)

	if !m.showHelp {
		t.Error("expected showHelp to be true after pressing '?'")
	}

	// Toggle again
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newModel.(BackupListModel)

	if m.showHelp {
		t.Error("expected showHelp to be false after pressing '?' again")
	}
}

func TestBackupListResult_DefaultAction(t *testing.T) {
	model := NewBackupListModel([]backup.Metadata{})
	result := model.Result()

	if result.Action != ActionNone {
		t.Errorf("expected ActionNone, got %v", result.Action)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tc := range tests {
		result := formatSize(tc.bytes)
		if result != tc.expected {
			t.Errorf("formatSize(%d) = %s, expected %s", tc.bytes, result, tc.expected)
		}
	}
}

func TestBackupsToRows(t *testing.T) {
	backups := []backup.Metadata{
		{
			ID:         "test-id",
			Platform:   "claude-code",
			SourcePath: "/short/path.md",
			CreatedAt:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Size:       2048,
		},
	}

	rows := backupsToRows(backups)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if row[0] != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", row[0])
	}
	if row[1] != "claude-code" {
		t.Errorf("expected platform 'claude-code', got '%s'", row[1])
	}
	if row[4] != "2.0 KB" {
		t.Errorf("expected size '2.0 KB', got '%s'", row[4])
	}
}

func TestBackupsToRows_LongPath(t *testing.T) {
	longPath := "/home/user/.config/claude/skills/very/long/path/that/exceeds/forty/characters/file.md"
	backups := []backup.Metadata{
		{
			ID:         "test-id",
			Platform:   "claude-code",
			SourcePath: longPath,
			CreatedAt:  time.Now(),
			Size:       1024,
		},
	}

	rows := backupsToRows(backups)
	source := rows[0][2]

	// Should be truncated with "..." prefix
	if len(source) > 40 {
		t.Errorf("expected source to be truncated to 40 chars, got %d chars", len(source))
	}
	if source[:3] != "..." {
		t.Errorf("expected source to start with '...', got '%s'", source[:3])
	}
}
