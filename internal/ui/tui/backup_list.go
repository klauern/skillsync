// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/klauern/skillsync/internal/backup"
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

// backupListKeyMap defines the key bindings for the backup list.
type backupListKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Restore  key.Binding
	Delete   key.Binding
	Verify   key.Binding
	Filter   key.Binding
	ClearFlt key.Binding
	Help     key.Binding
	Quit     key.Binding
}

func defaultBackupListKeyMap() backupListKeyMap {
	return backupListKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Restore: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restore"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Verify: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "verify"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ClearFlt: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear filter"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// BackupListModel is the BubbleTea model for interactive backup listing.
type BackupListModel struct {
	table       table.Model
	backups     []backup.Metadata
	filtered    []backup.Metadata
	keys        backupListKeyMap
	result      BackupListResult
	filter      string
	filtering   bool
	showHelp    bool
	confirmMode bool
	confirmMsg  string
	width       int
	height      int
	quitting    bool
}

// Styles for the backup list TUI.
var backupListStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Confirm     lipgloss.Style
	Status      lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:      lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput: lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Confirm:     lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true).Padding(1, 2),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
}

// NewBackupListModel creates a new backup list model.
func NewBackupListModel(backups []backup.Metadata) BackupListModel {
	columns := []table.Column{
		{Title: "ID", Width: 28},
		{Title: "Platform", Width: 12},
		{Title: "Source", Width: 40},
		{Title: "Created", Width: 19},
		{Title: "Size", Width: 10},
	}

	rows := backupsToRows(backups)

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return BackupListModel{
		table:    t,
		backups:  backups,
		filtered: backups,
		keys:     defaultBackupListKeyMap(),
	}
}

func backupsToRows(backups []backup.Metadata) []table.Row {
	rows := make([]table.Row, len(backups))
	for i, b := range backups {
		source := b.SourcePath
		if len(source) > 40 {
			source = "..." + source[len(source)-37:]
		}
		rows[i] = table.Row{
			b.ID,
			b.Platform,
			source,
			b.CreatedAt.Format("2006-01-02 15:04"),
			formatSize(b.Size),
		}
	}
	return rows
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// Init implements tea.Model.
func (m BackupListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m BackupListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust table height based on window
		newHeight := msg.Height - 8 // Reserve space for title, help, status
		if newHeight < 5 {
			newHeight = 5
		}
		m.table.SetHeight(newHeight)

	case tea.KeyMsg:
		// Handle confirmation mode
		if m.confirmMode {
			switch msg.String() {
			case "y", "Y":
				m.quitting = true
				return m, tea.Quit
			case "n", "N", "esc":
				m.confirmMode = false
				m.confirmMsg = ""
				return m, nil
			}
			return m, nil
		}

		// Handle filtering mode
		if m.filtering {
			switch msg.String() {
			case "enter":
				m.filtering = false
				return m, nil
			case "esc":
				m.filter = ""
				m.filtering = false
				m.applyFilter()
				return m, nil
			case "backspace":
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
					m.applyFilter()
				}
				return m, nil
			default:
				if len(msg.String()) == 1 {
					m.filter += msg.String()
					m.applyFilter()
				}
				return m, nil
			}
		}

		// Normal mode key handling
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.Filter):
			m.filtering = true
			return m, nil

		case key.Matches(msg, m.keys.ClearFlt):
			m.filter = ""
			m.applyFilter()
			return m, nil

		case key.Matches(msg, m.keys.Restore):
			if len(m.filtered) > 0 {
				selected := m.getSelectedBackup()
				m.result = BackupListResult{
					Action:   ActionRestore,
					BackupID: selected.ID,
					Backup:   selected,
				}
				m.confirmMode = true
				m.confirmMsg = fmt.Sprintf("Restore backup %s? (y/n)", selected.ID)
			}
			return m, nil

		case key.Matches(msg, m.keys.Delete):
			if len(m.filtered) > 0 {
				selected := m.getSelectedBackup()
				m.result = BackupListResult{
					Action:   ActionDelete,
					BackupID: selected.ID,
					Backup:   selected,
				}
				m.confirmMode = true
				m.confirmMsg = fmt.Sprintf("Delete backup %s? (y/n)", selected.ID)
			}
			return m, nil

		case key.Matches(msg, m.keys.Verify):
			if len(m.filtered) > 0 {
				selected := m.getSelectedBackup()
				m.result = BackupListResult{
					Action:   ActionVerify,
					BackupID: selected.ID,
					Backup:   selected,
				}
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *BackupListModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.backups
	} else {
		var filtered []backup.Metadata
		lowerFilter := strings.ToLower(m.filter)
		for _, b := range m.backups {
			if strings.Contains(strings.ToLower(b.ID), lowerFilter) ||
				strings.Contains(strings.ToLower(b.Platform), lowerFilter) ||
				strings.Contains(strings.ToLower(b.SourcePath), lowerFilter) {
				filtered = append(filtered, b)
			}
		}
		m.filtered = filtered
	}
	m.table.SetRows(backupsToRows(m.filtered))
}

func (m BackupListModel) getSelectedBackup() backup.Metadata {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return backup.Metadata{}
}

// View implements tea.Model.
func (m BackupListModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	title := backupListStyles.Title.Render("📦 Skillsync Backups")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Filter indicator
	if m.filter != "" || m.filtering {
		filterStr := backupListStyles.Filter.Render("Filter: ")
		filterVal := backupListStyles.FilterInput.Render(m.filter)
		if m.filtering {
			filterVal += "█"
		}
		b.WriteString(filterStr + filterVal + "\n\n")
	}

	// Confirmation dialog
	if m.confirmMode {
		b.WriteString(m.table.View())
		b.WriteString("\n\n")
		b.WriteString(backupListStyles.Confirm.Render(m.confirmMsg))
		return b.String()
	}

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n")

	// Status bar
	status := fmt.Sprintf("%d backup(s)", len(m.filtered))
	if m.filter != "" {
		status = fmt.Sprintf("%d of %d backup(s) (filtered)", len(m.filtered), len(m.backups))
	}
	b.WriteString(backupListStyles.Status.Render(status))
	b.WriteString("\n")

	// Help
	if m.showHelp {
		help := m.renderFullHelp()
		b.WriteString("\n")
		b.WriteString(help)
	} else {
		help := m.renderShortHelp()
		b.WriteString(help)
	}

	return b.String()
}

func (m BackupListModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"r restore",
		"d delete",
		"v verify",
		"/ filter",
		"? help",
		"q quit",
	}
	return backupListStyles.Help.Render(strings.Join(keys, " • "))
}

func (m BackupListModel) renderFullHelp() string {
	help := `Navigation:
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
	return backupListStyles.Help.Render(help)
}

// Result returns the result of the user interaction.
func (m BackupListModel) Result() BackupListResult {
	return m.result
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
