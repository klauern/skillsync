// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/klauern/skillsync/internal/model"
)

// DiscoverAction represents the action to perform on a selected skill.
type DiscoverAction int

const (
	// DiscoverActionNone means no action was taken (user quit).
	DiscoverActionNone DiscoverAction = iota
	// DiscoverActionView means the user wants to view the skill content.
	DiscoverActionView
	// DiscoverActionCopy means the user wants to copy the skill path.
	DiscoverActionCopy
)

// DiscoverListResult contains the result of the discover list TUI interaction.
type DiscoverListResult struct {
	Action DiscoverAction
	Skill  model.Skill
}

// discoverListKeyMap defines the key bindings for the discover list.
type discoverListKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	View     key.Binding
	Copy     key.Binding
	Filter   key.Binding
	ClearFlt key.Binding
	Help     key.Binding
	Quit     key.Binding
}

func defaultDiscoverListKeyMap() discoverListKeyMap {
	return discoverListKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		View: key.NewBinding(
			key.WithKeys("enter", "v"),
			key.WithHelp("enter/v", "view"),
		),
		Copy: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy path"),
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

// DiscoverListModel is the BubbleTea model for interactive skill discovery.
type DiscoverListModel struct {
	table     table.Model
	skills    []model.Skill
	filtered  []model.Skill
	keys      discoverListKeyMap
	result    DiscoverListResult
	filter    string
	filtering bool
	showHelp  bool
	width     int
	height    int
	quitting  bool
}

// Styles for the discover list TUI.
var discoverListStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Status      lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:      lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput: lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
}

// NewDiscoverListModel creates a new discover list model.
func NewDiscoverListModel(skills []model.Skill) DiscoverListModel {
	columns := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Platform", Width: 12},
		{Title: "Scope", Width: 15},
		{Title: "Description", Width: 45},
	}

	// Sort skills alphabetically by name (case-insensitive)
	sort.Slice(skills, func(i, j int) bool {
		return strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
	})

	rows := skillsToRows(skills)

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

	return DiscoverListModel{
		table:    t,
		skills:   skills,
		filtered: skills,
		keys:     defaultDiscoverListKeyMap(),
	}
}

func skillsToRows(skills []model.Skill) []table.Row {
	rows := make([]table.Row, len(skills))
	for i, s := range skills {
		name := s.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}
		desc := s.Description
		if len(desc) > 45 {
			desc = desc[:42] + "..."
		}
		rows[i] = table.Row{
			name,
			string(s.Platform),
			s.DisplayScope(),
			desc,
		}
	}
	return rows
}

// Init implements tea.Model.
func (m DiscoverListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m DiscoverListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust table height based on window
		newHeight := max(msg.Height-8, 5) // Reserve space for title, help, status
		m.table.SetHeight(newHeight)

	case tea.KeyMsg:
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

		case key.Matches(msg, m.keys.View):
			if len(m.filtered) > 0 {
				selected := m.getSelectedSkill()
				m.result = DiscoverListResult{
					Action: DiscoverActionView,
					Skill:  selected,
				}
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil

		case key.Matches(msg, m.keys.Copy):
			if len(m.filtered) > 0 {
				selected := m.getSelectedSkill()
				m.result = DiscoverListResult{
					Action: DiscoverActionCopy,
					Skill:  selected,
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

func (m *DiscoverListModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.skills
	} else {
		var filtered []model.Skill
		lowerFilter := strings.ToLower(m.filter)
		for _, s := range m.skills {
			if strings.Contains(strings.ToLower(s.Name), lowerFilter) ||
				strings.Contains(strings.ToLower(string(s.Platform)), lowerFilter) ||
				strings.Contains(strings.ToLower(s.DisplayScope()), lowerFilter) ||
				strings.Contains(strings.ToLower(s.Description), lowerFilter) {
				filtered = append(filtered, s)
			}
		}
		m.filtered = filtered
	}
	m.table.SetRows(skillsToRows(m.filtered))
}

func (m DiscoverListModel) getSelectedSkill() model.Skill {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return model.Skill{}
}

// View implements tea.Model.
func (m DiscoverListModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	title := discoverListStyles.Title.Render("🔍 Skillsync Skills")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Filter indicator
	if m.filter != "" || m.filtering {
		filterStr := discoverListStyles.Filter.Render("Filter: ")
		filterVal := discoverListStyles.FilterInput.Render(m.filter)
		if m.filtering {
			filterVal += "█"
		}
		b.WriteString(filterStr + filterVal + "\n\n")
	}

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n")

	// Status bar
	status := fmt.Sprintf("%d skill(s)", len(m.filtered))
	if m.filter != "" {
		status = fmt.Sprintf("%d of %d skill(s) (filtered)", len(m.filtered), len(m.skills))
	}
	b.WriteString(discoverListStyles.Status.Render(status))
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

func (m DiscoverListModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"enter view",
		"c copy path",
		"/ filter",
		"? help",
		"q quit",
	}
	return discoverListStyles.Help.Render(strings.Join(keys, " • "))
}

func (m DiscoverListModel) renderFullHelp() string {
	help := `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  g/Home   Go to top
  G/End    Go to bottom

Actions:
  Enter/v  View skill content
  c        Copy skill path

Filter:
  /        Start filtering (by name, platform, scope, or description)
  Esc      Clear filter
  Enter    Finish filtering

General:
  ?        Toggle full help
  q        Quit`
	return discoverListStyles.Help.Render(help)
}

// Result returns the result of the user interaction.
func (m DiscoverListModel) Result() DiscoverListResult {
	return m.result
}

// RunDiscoverList runs the interactive discover list and returns the result.
func RunDiscoverList(skills []model.Skill) (DiscoverListResult, error) {
	if len(skills) == 0 {
		return DiscoverListResult{}, nil
	}

	model := NewDiscoverListModel(skills)
	finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return DiscoverListResult{}, err
	}

	if m, ok := finalModel.(DiscoverListModel); ok {
		return m.Result(), nil
	}

	return DiscoverListResult{}, nil
}
