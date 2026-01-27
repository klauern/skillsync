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

// DeleteAction represents the action to perform after delete configuration.
type DeleteAction int

const (
	// DeleteActionNone means no action was taken (user quit).
	DeleteActionNone DeleteAction = iota
	// DeleteActionDelete means the user wants to delete selected skills.
	DeleteActionDelete
)

// DeleteListResult contains the result of the delete list TUI interaction.
type DeleteListResult struct {
	Action         DeleteAction
	SelectedSkills []model.Skill
}

// deleteListKeyMap defines the key bindings for the delete list.
type deleteListKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Toggle    key.Binding
	ToggleAll key.Binding
	Confirm   key.Binding
	Filter    key.Binding
	ClearFlt  key.Binding
	Help      key.Binding
	Quit      key.Binding
}

func defaultDeleteListKeyMap() deleteListKeyMap {
	return deleteListKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" ", "tab"),
			key.WithHelp("space/tab", "toggle"),
		),
		ToggleAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "toggle all"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete selected"),
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

// DeleteListModel is the BubbleTea model for interactive skill deletion.
type DeleteListModel struct {
	table       table.Model
	skills      []model.Skill
	filtered    []model.Skill
	selected    map[string]bool // map of skill key to selected state
	keys        deleteListKeyMap
	result      DeleteListResult
	filter      string
	filtering   bool
	showHelp    bool
	confirmMode bool
	width       int
	height      int
	quitting    bool
}

// Styles for the delete list TUI.
var deleteListStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Confirm     lipgloss.Style
	Status      lipgloss.Style
	Warning     lipgloss.Style
	Checkbox    lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:      lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput: lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Confirm:     lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true).Padding(1, 2),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Warning:     lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true),
	Checkbox:    lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
}

// deleteSkillKey creates a unique key for a skill (platform + scope + name combination).
func deleteSkillKey(s model.Skill) string {
	return fmt.Sprintf("%s:%s:%s", s.Platform, s.Scope, s.Name)
}

// NewDeleteListModel creates a new delete list model.
// Only writable skills (repo and user scope) are included.
func NewDeleteListModel(skills []model.Skill) DeleteListModel {
	// Filter to only include deletable skills (repo and user scopes)
	var deletableSkills []model.Skill
	for _, s := range skills {
		if s.Scope == model.ScopeRepo || s.Scope == model.ScopeUser {
			deletableSkills = append(deletableSkills, s)
		}
	}

	// Sort skills alphabetically by name (case-insensitive)
	sort.Slice(deletableSkills, func(i, j int) bool {
		return strings.ToLower(deletableSkills[i].Name) < strings.ToLower(deletableSkills[j].Name)
	})

	columns := []table.Column{
		{Title: " ", Width: 3},            // Checkbox column
		{Title: "Name", Width: 25},        // Skill name
		{Title: "Platform", Width: 12},    // Platform
		{Title: "Scope", Width: 10},       // Scope
		{Title: "Description", Width: 40}, // Description
	}

	// Initialize with no skills selected (deletion is opt-in)
	selected := make(map[string]bool)

	m := DeleteListModel{
		skills:   deletableSkills,
		filtered: deletableSkills,
		selected: selected,
		keys:     defaultDeleteListKeyMap(),
	}

	rows := m.skillsToRows(deletableSkills)

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

	m.table = t
	return m
}

func (m DeleteListModel) skillsToRows(skills []model.Skill) []table.Row {
	rows := make([]table.Row, len(skills))
	for i, s := range skills {
		checkbox := "[ ]"
		if m.selected[deleteSkillKey(s)] {
			checkbox = "[✓]"
		}

		name := s.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}
		platform := string(s.Platform)
		if len(platform) > 12 {
			platform = platform[:9] + "..."
		}
		scope := s.DisplayScope()
		if len(scope) > 10 {
			scope = scope[:7] + "..."
		}
		desc := s.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		rows[i] = table.Row{
			checkbox,
			name,
			platform,
			scope,
			desc,
		}
	}
	return rows
}

// Init implements tea.Model.
func (m DeleteListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m DeleteListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust table height based on window
		newHeight := max(msg.Height-10, 5) // Reserve space for title, warning, help, status
		m.table.SetHeight(newHeight)

	case tea.KeyMsg:
		// Handle confirmation mode
		if m.confirmMode {
			switch msg.String() {
			case "y", "Y":
				m.result = DeleteListResult{
					Action:         DeleteActionDelete,
					SelectedSkills: m.getSelectedSkills(),
				}
				m.quitting = true
				return m, tea.Quit
			case "n", "N", "esc":
				m.confirmMode = false
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

		case key.Matches(msg, m.keys.Toggle):
			if len(m.filtered) > 0 {
				skill := m.getSelectedSkill()
				m.selected[deleteSkillKey(skill)] = !m.selected[deleteSkillKey(skill)]
				m.table.SetRows(m.skillsToRows(m.filtered))
			}
			return m, nil

		case key.Matches(msg, m.keys.ToggleAll):
			// Count how many are currently selected
			selectedCount := 0
			for _, s := range m.filtered {
				if m.selected[deleteSkillKey(s)] {
					selectedCount++
				}
			}
			// If all or most are selected, deselect all; otherwise select all
			selectAll := selectedCount < len(m.filtered)/2+1
			for _, s := range m.filtered {
				m.selected[deleteSkillKey(s)] = selectAll
			}
			m.table.SetRows(m.skillsToRows(m.filtered))
			return m, nil

		case key.Matches(msg, m.keys.Confirm):
			selectedSkills := m.getSelectedSkills()
			if len(selectedSkills) > 0 {
				m.confirmMode = true
			}
			return m, nil
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *DeleteListModel) applyFilter() {
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
	m.table.SetRows(m.skillsToRows(m.filtered))
}

func (m DeleteListModel) getSelectedSkill() model.Skill {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return model.Skill{}
}

func (m DeleteListModel) getSelectedSkills() []model.Skill {
	var selected []model.Skill
	for _, s := range m.skills {
		if m.selected[deleteSkillKey(s)] {
			selected = append(selected, s)
		}
	}
	return selected
}

// View implements tea.Model.
func (m DeleteListModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title with warning color
	title := deleteListStyles.Title.Render("🗑️  Delete Skills")
	b.WriteString(title)
	b.WriteString("\n")

	// Warning message
	warning := deleteListStyles.Warning.Render("Only repo and user scope skills can be deleted")
	b.WriteString(warning)
	b.WriteString("\n\n")

	// Filter indicator
	if m.filter != "" || m.filtering {
		filterStr := deleteListStyles.Filter.Render("Filter: ")
		filterVal := deleteListStyles.FilterInput.Render(m.filter)
		if m.filtering {
			filterVal += "█"
		}
		b.WriteString(filterStr + filterVal + "\n\n")
	}

	// Confirmation dialog
	if m.confirmMode {
		selectedCount := len(m.getSelectedSkills())
		b.WriteString(m.table.View())
		b.WriteString("\n\n")
		confirmMsg := fmt.Sprintf("⚠️  DELETE %d skill(s)? This cannot be undone! (y/n)", selectedCount)
		b.WriteString(deleteListStyles.Confirm.Render(confirmMsg))
		return b.String()
	}

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n")

	// Status bar
	selectedCount := len(m.getSelectedSkills())
	status := fmt.Sprintf("%d skill(s) selected for deletion of %d", selectedCount, len(m.filtered))
	if m.filter != "" {
		status = fmt.Sprintf("%d selected, %d of %d shown (filtered)", selectedCount, len(m.filtered), len(m.skills))
	}
	b.WriteString(deleteListStyles.Status.Render(status))
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

func (m DeleteListModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"space toggle",
		"a toggle all",
		"d delete",
		"/ filter",
		"? help",
		"q quit",
	}
	return deleteListStyles.Help.Render(strings.Join(keys, " • "))
}

func (m DeleteListModel) renderFullHelp() string {
	help := `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  g/Home   Go to top
  G/End    Go to bottom

Selection:
  Space/Tab  Toggle current skill for deletion
  a          Toggle all skills

Actions:
  d        Confirm and delete selected skills

Filter:
  /        Start filtering (by name, platform, scope, or description)
  Esc      Clear filter
  Enter    Finish filtering

General:
  ?        Toggle full help
  q        Quit without deleting`
	return deleteListStyles.Help.Render(help)
}

// Result returns the result of the user interaction.
func (m DeleteListModel) Result() DeleteListResult {
	return m.result
}

// RunDeleteList runs the interactive delete list and returns the result.
func RunDeleteList(skills []model.Skill) (DeleteListResult, error) {
	if len(skills) == 0 {
		return DeleteListResult{}, nil
	}

	mdl := NewDeleteListModel(skills)
	// Check if any deletable skills exist after filtering
	if len(mdl.skills) == 0 {
		return DeleteListResult{}, nil
	}

	finalModel, err := tea.NewProgram(mdl, tea.WithAltScreen()).Run()
	if err != nil {
		return DeleteListResult{}, err
	}

	if m, ok := finalModel.(DeleteListModel); ok {
		return m.Result(), nil
	}

	return DeleteListResult{}, nil
}
