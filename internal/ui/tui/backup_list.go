// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/backup"
	"github.com/klauern/skillsync/internal/ui"
)

// BackupAction represents the action to perform on a selected backup.
type BackupAction int

const (
	// ActionNone means no action was taken (user quit).
	ActionNone BackupAction = iota
	// ActionRestore means the user wants to restore the selected backup.
	ActionRestore
	// ActionDelete means the user wants to delete the selected backup.
	ActionDelete
	// ActionVerify means the user wants to verify the selected backup.
	ActionVerify
)

// BackupListResult contains the result of the backup list TUI interaction.
type BackupListResult struct {
	Action   BackupAction
	BackupID string
	Backup   backup.Metadata
}

// BackupListModel is the BubbleTea model for interactive backup listing.
type BackupListModel struct {
	ListModel[backup.Metadata]
}

// Update wraps the base Update and preserves the BackupListModel type.
func (m BackupListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	inner, cmd := m.ListModel.Update(msg)
	m.ListModel = inner.(ListModel[backup.Metadata])
	return m, cmd
}

// Result returns the result of the user interaction.
func (m BackupListModel) Result() BackupListResult {
	if r, ok := m.result.(BackupListResult); ok {
		return r
	}
	return BackupListResult{}
}

// NewBackupListModel creates a new backup list model.
func NewBackupListModel(backups []backup.Metadata) BackupListModel {
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	restoreKey := key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restore"))
	deleteKey := key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete"))
	verifyKey := key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "verify"))

	cfg := ListConfig[backup.Metadata]{
		Title: "📦 Skillsync Backups",
		Columns: []table.Column{
			{Title: "ID", Width: 28},
			{Title: "Platform", Width: 12},
			{Title: "Source", Width: 40},
			{Title: "Created", Width: 19},
			{Title: "Size", Width: 10},
		},
		ToRows:  backupsToRows,
		Matches: backupMatches,
		Actions: []ActionBinding[backup.Metadata]{
			{
				Binding: restoreKey,
				Apply: func(b backup.Metadata) any {
					return BackupListResult{Action: ActionRestore, BackupID: b.ID, Backup: b}
				},
				NeedsConfirm: func(b backup.Metadata) string {
					return fmt.Sprintf("Restore backup %s? (y/n)", b.ID)
				},
			},
			{
				Binding: deleteKey,
				Apply: func(b backup.Metadata) any {
					return BackupListResult{Action: ActionDelete, BackupID: b.ID, Backup: b}
				},
				NeedsConfirm: func(b backup.Metadata) string {
					return fmt.Sprintf("Delete backup %s? (y/n)", b.ID)
				},
			},
			{
				Binding: verifyKey,
				Apply: func(b backup.Metadata) any {
					return BackupListResult{Action: ActionVerify, BackupID: b.ID, Backup: b}
				},
			},
		},
		StatusText: func(filtered, total int, filter string) string {
			if filter != "" {
				return fmt.Sprintf("%d of %d backup(s) (filtered)", filtered, total)
			}
			return fmt.Sprintf("%d backup(s)", filtered)
		},
		ShortHelp: func() string {
			return strings.Join([]string{
				"↑/↓ navigate", "r restore", "d delete", "v verify", "/ filter", "? help", "q quit",
			}, " • ")
		},
		FullHelp: func() string {
			return `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  g/Home   Go to top
  G/End    Go to bottom

Actions:
  r        Restore selected backup
  d        Delete selected backup
  v        Verify selected backup

Filter:
  /        Start filtering
  Esc      Clear filter
  Enter    Finish filtering

General:
  ?        Toggle full help
  q        Quit`
		},
	}

	return BackupListModel{ListModel: NewListModel(backups, cfg)}
}

func backupMatches(b backup.Metadata, lf string) bool {
	if lf == "" {
		return true
	}
	return strings.Contains(strings.ToLower(b.ID), lf) ||
		strings.Contains(strings.ToLower(b.Platform), lf) ||
		strings.Contains(strings.ToLower(b.SourcePath), lf)
}

func backupsToRows(backups []backup.Metadata) []table.Row {
	rows := make([]table.Row, len(backups))
	for i, b := range backups {
		rows[i] = table.Row{
			truncateTableValue(b.ID, 28),
			truncateTableValue(b.Platform, 12),
			truncateTableValueFromStart(b.SourcePath, 40),
			b.CreatedAt.Format("2006-01-02 15:04"),
			ui.FormatSize(b.Size),
		}
	}
	return rows
}

// RunBackupList runs the interactive backup list and returns the result.
func RunBackupList(backups []backup.Metadata) (BackupListResult, error) {
	if len(backups) == 0 {
		return BackupListResult{}, nil
	}

	model := NewBackupListModel(backups)
	finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return BackupListResult{}, err
	}

	if m, ok := finalModel.(BackupListModel); ok {
		return m.Result(), nil
	}

	return BackupListResult{}, nil
}
