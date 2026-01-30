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
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/klauern/skillsync/internal/model"
)

// ScopeAction represents the action to perform after scope interaction.
type ScopeAction int

const (
	// ScopeActionNone means no action was taken (user quit).
	ScopeActionNone ScopeAction = iota
	// ScopeActionView means the user wants to view skill details.
	ScopeActionView
)

// ScopeListResult contains the result of the scope list TUI interaction.
type ScopeListResult struct {
	Action        ScopeAction
	SelectedSkill model.Skill
}

// scopeListKeyMap defines the key bindings for the scope list.
type scopeListKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	View      key.Binding
	Filter    key.Binding
	ClearFlt  key.Binding
	NextScope key.Binding
	PrevScope key.Binding
	Help      key.Binding
	Quit      key.Binding
}

type scopeListColumnWidths struct {
	name     int
	platform int
	scope    int
	desc     int
}

func defaultScopeListColumnWidths() scopeListColumnWidths {
	return scopeListColumnWidths{
		name:     25,
		platform: 12,
		scope:    40,
		desc:     50,
	}
}

func defaultScopeListKeyMap() scopeListKeyMap {
	return scopeListKeyMap{
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
			key.WithHelp("enter/v", "view details"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ClearFlt: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear filter"),
		),
		NextScope: key.NewBinding(
			key.WithKeys("tab", "l"),
			key.WithHelp("tab/l", "next scope"),
		),
		PrevScope: key.NewBinding(
			key.WithKeys("shift+tab", "h"),
			key.WithHelp("S-tab/h", "prev scope"),
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

// ScopeListModel is the BubbleTea model for interactive scope management.
type ScopeListModel struct {
	table        table.Model
	skills       []model.Skill
	filtered     []model.Skill
	keys         scopeListKeyMap
	result       ScopeListResult
	filter       string
	filtering    bool
	scopeOptions []model.SkillScope
	scopeIndex   int // Index into scopeOptions (-1 = all)
	showHelp     bool
	width        int
	height       int
	quitting     bool
	columnWidths scopeListColumnWidths
}

// Styles for the scope list TUI.
var scopeListStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Status      lipgloss.Style
	ScopeTab    lipgloss.Style
	ScopeActive lipgloss.Style
	Info        lipgloss.Style
	Description lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:      lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput: lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	ScopeTab:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	ScopeActive: lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(true).Padding(0, 1),
	Info:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Description: lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1),
}

// NewScopeListModel creates a new scope list model.
func NewScopeListModel(skills []model.Skill) ScopeListModel {
	columnWidths := defaultScopeListColumnWidths()
	columns := []table.Column{
		{Title: "Name", Width: columnWidths.name},
		{Title: "Platform", Width: columnWidths.platform},
		{Title: "Scope", Width: columnWidths.scope},
		{Title: "Description", Width: columnWidths.desc},
	}

	// Collect unique scopes from skills
	scopeSet := make(map[model.SkillScope]bool)
	for _, s := range skills {
		scopeSet[s.Scope] = true
	}

	// Build scope options in precedence order
	scopeOptions := []model.SkillScope{}
	for _, scope := range model.AllScopes() {
		if scopeSet[scope] {
			scopeOptions = append(scopeOptions, scope)
		}
	}

	// Sort skills alphabetically by name (case-insensitive)
	sort.Slice(skills, func(i, j int) bool {
		return strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
	})

	m := ScopeListModel{
		skills:       skills,
		filtered:     skills,
		keys:         defaultScopeListKeyMap(),
		scopeOptions: scopeOptions,
		scopeIndex:   -1, // -1 means "all"
		columnWidths: columnWidths,
	}

	rows := m.skillsToRows(skills)

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

func (m ScopeListModel) skillsToRows(skills []model.Skill) []table.Row {
	widths := m.columnWidths
	if widths.desc == 0 {
		widths = defaultScopeListColumnWidths()
	}
	rows := make([]table.Row, len(skills))
	for i, s := range skills {
		name := truncateText(s.Name, widths.name)
		platform := truncateText(string(s.Platform), widths.platform)
		scope := truncateText(s.DisplayScope(), widths.scope)
		desc := truncateText(s.Description, widths.desc)
		rows[i] = table.Row{
			name,
			platform,
			scope,
			desc,
		}
	}
	return rows
}

// Init implements tea.Model.
func (m ScopeListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m ScopeListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust table height based on window
		newHeight := max(msg.Height-12, 5) // Reserve space for title, scope tabs, help, status
		m.table.SetHeight(newHeight)
		m.applyColumnWidths(msg.Width)

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

		case key.Matches(msg, m.keys.NextScope):
			if len(m.scopeOptions) > 0 {
				m.scopeIndex++
				if m.scopeIndex >= len(m.scopeOptions) {
					m.scopeIndex = -1 // Wrap to "all"
				}
				m.applyFilter()
			}
			return m, nil

		case key.Matches(msg, m.keys.PrevScope):
			if len(m.scopeOptions) > 0 {
				m.scopeIndex--
				if m.scopeIndex < -1 {
					m.scopeIndex = len(m.scopeOptions) - 1 // Wrap to last scope
				}
				m.applyFilter()
			}
			return m, nil

		case key.Matches(msg, m.keys.View):
			if len(m.filtered) > 0 {
				skill := m.getSelectedSkill()
				m.result = ScopeListResult{
					Action:        ScopeActionView,
					SelectedSkill: skill,
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

func (m *ScopeListModel) applyFilter() {
	// Start with all skills
	filtered := m.skills

	// Apply scope filter if not "all"
	if m.scopeIndex >= 0 && m.scopeIndex < len(m.scopeOptions) {
		selectedScope := m.scopeOptions[m.scopeIndex]
		var scopeFiltered []model.Skill
		for _, s := range filtered {
			if s.Scope == selectedScope {
				scopeFiltered = append(scopeFiltered, s)
			}
		}
		filtered = scopeFiltered
	}

	// Apply text filter
	if m.filter != "" {
		var textFiltered []model.Skill
		lowerFilter := strings.ToLower(m.filter)
		for _, s := range filtered {
			if strings.Contains(strings.ToLower(s.Name), lowerFilter) ||
				strings.Contains(strings.ToLower(string(s.Platform)), lowerFilter) ||
				strings.Contains(strings.ToLower(s.DisplayScope()), lowerFilter) ||
				strings.Contains(strings.ToLower(s.Description), lowerFilter) {
				textFiltered = append(textFiltered, s)
			}
		}
		filtered = textFiltered
	}

	m.filtered = filtered
	m.table.SetRows(m.skillsToRows(m.filtered))
}

func (m ScopeListModel) getSelectedSkill() model.Skill {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return model.Skill{}
}

// View implements tea.Model.
func (m ScopeListModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	title := scopeListStyles.Title.Render("📂 Scope Management - Browse Skills by Scope")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Scope tabs
	b.WriteString(m.renderScopeTabs())
	b.WriteString("\n\n")

	// Filter indicator
	if m.filter != "" || m.filtering {
		filterStr := scopeListStyles.Filter.Render("Filter: ")
		filterVal := scopeListStyles.FilterInput.Render(m.filter)
		if m.filtering {
			filterVal += "█"
		}
		b.WriteString(filterStr + filterVal + "\n\n")
	}

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n")

	// Status bar
	status := m.renderStatus()
	b.WriteString(scopeListStyles.Status.Render(status))
	b.WriteString("\n")

	selected := m.getSelectedSkill()
	if selected.Name != "" && selected.Description != "" {
		descWidth := max(m.width-2, 40)
		formatted := formatDescription(selected.Description, descWidth)
		b.WriteString(scopeListStyles.Description.Render(formatted))
		b.WriteString("\n")
	}

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

func (m ScopeListModel) renderScopeTabs() string {
	var tabs []string

	// "All" tab
	if m.scopeIndex == -1 {
		tabs = append(tabs, scopeListStyles.ScopeActive.Render("[All]"))
	} else {
		tabs = append(tabs, scopeListStyles.ScopeTab.Render(" All "))
	}

	// Individual scope tabs
	titleCaser := cases.Title(language.English)
	for i, scope := range m.scopeOptions {
		scopeName := titleCaser.String(string(scope))
		if i == m.scopeIndex {
			tabs = append(tabs, scopeListStyles.ScopeActive.Render(fmt.Sprintf("[%s]", scopeName)))
		} else {
			tabs = append(tabs, scopeListStyles.ScopeTab.Render(fmt.Sprintf(" %s ", scopeName)))
		}
	}

	return strings.Join(tabs, "")
}

func (m ScopeListModel) renderStatus() string {
	// Count skills by scope
	scopeCounts := make(map[model.SkillScope]int)
	for _, s := range m.skills {
		scopeCounts[s.Scope]++
	}

	// Build counts string
	var counts []string
	for _, scope := range m.scopeOptions {
		counts = append(counts, fmt.Sprintf("%s: %d", scope, scopeCounts[scope]))
	}

	status := fmt.Sprintf("Showing %d of %d skills", len(m.filtered), len(m.skills))
	if len(counts) > 0 {
		status += " | " + strings.Join(counts, ", ")
	}
	return status
}

func (m ScopeListModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"tab/S-tab scope",
		"enter view",
		"/ filter",
		"? help",
		"q quit",
	}
	return scopeListStyles.Help.Render(strings.Join(keys, " • "))
}

func (m ScopeListModel) renderFullHelp() string {
	help := `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  g/Home   Go to top
  G/End    Go to bottom

Scope Filtering:
  Tab/l       Next scope
  Shift-Tab/h Previous scope

Actions:
  Enter/v  View skill details

Text Filter:
  /        Start filtering (by name, platform, scope, or description)
  Esc      Clear filter
  Enter    Finish filtering

General:
  ?        Toggle full help
  q        Quit`
	return scopeListStyles.Help.Render(help)
}

func (m *ScopeListModel) applyColumnWidths(totalWidth int) {
	widths := defaultScopeListColumnWidths()
	if totalWidth > 0 {
		const separatorWidth = 6
		descWidth := totalWidth - (widths.name + widths.platform + widths.scope + separatorWidth)
		if descWidth < 40 {
			descWidth = 40
		}
		widths.desc = descWidth
	}

	m.columnWidths = widths
	m.table.SetColumns([]table.Column{
		{Title: "Name", Width: widths.name},
		{Title: "Platform", Width: widths.platform},
		{Title: "Scope", Width: widths.scope},
		{Title: "Description", Width: widths.desc},
	})
	m.table.SetRows(m.skillsToRows(m.filtered))
}

// Result returns the result of the user interaction.
func (m ScopeListModel) Result() ScopeListResult {
	return m.result
}

// RunScopeList runs the interactive scope list and returns the result.
func RunScopeList(skills []model.Skill) (ScopeListResult, error) {
	if len(skills) == 0 {
		return ScopeListResult{}, nil
	}

	mdl := NewScopeListModel(skills)
	finalModel, err := tea.NewProgram(mdl, tea.WithAltScreen()).Run()
	if err != nil {
		return ScopeListResult{}, err
	}

	if m, ok := finalModel.(ScopeListModel); ok {
		return m.Result(), nil
	}

	return ScopeListResult{}, nil
}
