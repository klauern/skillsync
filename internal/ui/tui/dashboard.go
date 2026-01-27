// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DashboardView represents the available TUI views in the dashboard.
type DashboardView int

const (
	// DashboardViewNone means no view was selected (user quit).
	DashboardViewNone DashboardView = iota
	// DashboardViewDiscover opens the skill discovery view.
	DashboardViewDiscover
	// DashboardViewBackups opens the backup management view.
	DashboardViewBackups
	// DashboardViewSync opens the sync operations view.
	DashboardViewSync
	// DashboardViewCompare opens the compare/dedupe view.
	DashboardViewCompare
	// DashboardViewConfig opens the config management view.
	DashboardViewConfig
	// DashboardViewExport opens the import/export view.
	DashboardViewExport
	// DashboardViewScope opens the scope management view.
	DashboardViewScope
	// DashboardViewPromote opens the promote/demote view.
	DashboardViewPromote
	// DashboardViewDelete opens the delete/remove skills view.
	DashboardViewDelete
)

// DashboardResult contains the result of the dashboard TUI interaction.
type DashboardResult struct {
	View DashboardView
}

// MenuItem represents a menu item in the dashboard.
type MenuItem struct {
	Title       string
	Description string
	View        DashboardView
}

// dashboardKeyMap defines the key bindings for the dashboard.
type dashboardKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Help   key.Binding
	Quit   key.Binding
}

func defaultDashboardKeyMap() dashboardKeyMap {
	return dashboardKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter", "select"),
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

// DashboardModel is the BubbleTea model for the main dashboard.
type DashboardModel struct {
	items    []MenuItem
	cursor   int
	keys     dashboardKeyMap
	result   DashboardResult
	showHelp bool
	width    int
	height   int
	quitting bool
}

// Styles for the dashboard TUI.
var dashboardStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Item        lipgloss.Style
	Selected    lipgloss.Style
	Description lipgloss.Style
	Status      lipgloss.Style
	Border      lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Item:        lipgloss.NewStyle().Padding(0, 2),
	Selected:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Padding(0, 2),
	Description: lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 4),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Border:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(1, 2),
}

// defaultMenuItems returns the default menu items for the dashboard.
func defaultMenuItems() []MenuItem {
	return []MenuItem{
		{
			Title:       "Discover Skills",
			Description: "Browse and search skills across all platforms",
			View:        DashboardViewDiscover,
		},
		{
			Title:       "Backup Management",
			Description: "List, restore, delete, and verify backups",
			View:        DashboardViewBackups,
		},
		{
			Title:       "Sync Operations",
			Description: "Synchronize skills between platforms",
			View:        DashboardViewSync,
		},
		{
			Title:       "Compare & Dedupe",
			Description: "Find and remove duplicate skills",
			View:        DashboardViewCompare,
		},
		{
			Title:       "Import / Export",
			Description: "Export skills to file or import from backup",
			View:        DashboardViewExport,
		},
		{
			Title:       "Scope Management",
			Description: "View and change skill scopes (global/project)",
			View:        DashboardViewScope,
		},
		{
			Title:       "Promote / Demote",
			Description: "Move skills between global and project scopes",
			View:        DashboardViewPromote,
		},
		{
			Title:       "Configuration",
			Description: "View and edit skillsync settings",
			View:        DashboardViewConfig,
		},
		{
			Title:       "Delete Skills",
			Description: "Remove skills from repo or user scopes",
			View:        DashboardViewDelete,
		},
	}
}

// NewDashboardModel creates a new dashboard model.
func NewDashboardModel() DashboardModel {
	return DashboardModel{
		items: defaultMenuItems(),
		keys:  defaultDashboardKeyMap(),
	}
}

// Init implements tea.Model.
func (m DashboardModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
			return m, nil

		case key.Matches(msg, m.keys.Select):
			m.result = DashboardResult{
				View: m.items[m.cursor].View,
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m DashboardModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	title := dashboardStyles.Title.Render("🛠️  Skillsync Dashboard")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Menu items
	for i, item := range m.items {
		var line string
		if i == m.cursor {
			line = dashboardStyles.Selected.Render(fmt.Sprintf("> %s", item.Title))
		} else {
			line = dashboardStyles.Item.Render(fmt.Sprintf("  %s", item.Title))
		}
		b.WriteString(line)
		b.WriteString("\n")

		// Show description for selected item
		if i == m.cursor {
			desc := dashboardStyles.Description.Render(item.Description)
			b.WriteString(desc)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// Status bar
	status := "Use ↑/↓ to navigate, Enter to select"
	b.WriteString(dashboardStyles.Status.Render(status))
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

func (m DashboardModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"enter select",
		"? help",
		"q quit",
	}
	return dashboardStyles.Help.Render(strings.Join(keys, " • "))
}

func (m DashboardModel) renderFullHelp() string {
	help := `Navigation:
  ↑/k      Move up
  ↓/j      Move down

Actions:
  Enter    Select menu item
  Space    Select menu item

General:
  ?        Toggle full help
  q        Quit dashboard`
	return dashboardStyles.Help.Render(help)
}

// Result returns the result of the user interaction.
func (m DashboardModel) Result() DashboardResult {
	return m.result
}

// RunDashboard runs the interactive dashboard and returns the result.
func RunDashboard() (DashboardResult, error) {
	model := NewDashboardModel()
	finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return DashboardResult{}, err
	}

	if m, ok := finalModel.(DashboardModel); ok {
		return m.Result(), nil
	}

	return DashboardResult{}, nil
}
