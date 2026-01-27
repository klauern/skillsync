// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/klauern/skillsync/internal/model"
)

// PromoteDemoteAction represents the action to perform after selection.
type PromoteDemoteAction int

const (
	// PromoteDemoteActionNone means no action was taken (user quit).
	PromoteDemoteActionNone PromoteDemoteAction = iota
	// PromoteDemoteActionPromote means the user wants to promote selected skills.
	PromoteDemoteActionPromote
	// PromoteDemoteActionDemote means the user wants to demote selected skills.
	PromoteDemoteActionDemote
)

// PromoteDemoteListResult contains the result of the promote/demote list TUI interaction.
type PromoteDemoteListResult struct {
	Action         PromoteDemoteAction
	SelectedSkills []model.Skill
	RemoveSource   bool // Whether to remove the source after operation (move vs copy)
}

// promoteDemoteListKeyMap defines the key bindings for the promote/demote list.
type promoteDemoteListKeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Toggle     key.Binding
	ToggleAll  key.Binding
	Promote    key.Binding
	Demote     key.Binding
	ToggleMove key.Binding
	Filter     key.Binding
	ClearFlt   key.Binding
	Help       key.Binding
	Quit       key.Binding
}

func defaultPromoteDemoteListKeyMap() promoteDemoteListKeyMap {
	return promoteDemoteListKeyMap{
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
		Promote: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "promote selected"),
		),
		Demote: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "demote selected"),
		),
		ToggleMove: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "toggle move/copy"),
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

// PromoteDemoteListModel is the BubbleTea model for interactive skill promotion/demotion.
type PromoteDemoteListModel struct {
	table         table.Model
	skills        []model.Skill
	filtered      []model.Skill
	selected      map[string]bool // map of skill key to selected state
	keys          promoteDemoteListKeyMap
	result        PromoteDemoteListResult
	filter        string
	filtering     bool
	showHelp      bool
	confirmMode   bool
	confirmAction PromoteDemoteAction
	width         int
	height        int
	quitting      bool
	removeSource  bool // move instead of copy
}

// Styles for the promote/demote list TUI.
var promoteDemoteListStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Confirm     lipgloss.Style
	Status      lipgloss.Style
	Info        lipgloss.Style
	Checkbox    lipgloss.Style
	Promote     lipgloss.Style
	Demote      lipgloss.Style
	Option      lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:      lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput: lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Confirm:     lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true).Padding(1, 2),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Info:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Checkbox:    lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	Promote:     lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Demote:      lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true),
	Option:      lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
}

// promoteDemoteSkillKey creates a unique key for a skill (platform + scope + name combination).
func promoteDemoteSkillKey(s model.Skill) string {
	return fmt.Sprintf("%s:%s:%s", s.Platform, s.Scope, s.Name)
}

// NewPromoteDemoteListModel creates a new promote/demote list model.
// Only promotable/demotable skills (repo and user scope) are included.
func NewPromoteDemoteListModel(skills []model.Skill) PromoteDemoteListModel {
	// Filter to only include skills that can be promoted or demoted (repo and user scopes)
	var movableSkills []model.Skill
	for _, s := range skills {
		if s.Scope == model.ScopeRepo || s.Scope == model.ScopeUser {
			movableSkills = append(movableSkills, s)
		}
	}

	columns := []table.Column{
		{Title: " ", Width: 3},            // Checkbox column
		{Title: "Name", Width: 25},        // Skill name
		{Title: "Platform", Width: 12},    // Platform
		{Title: "Scope", Width: 10},       // Current scope
		{Title: "Can Move To", Width: 12}, // Target scope
		{Title: "Description", Width: 30}, // Description
	}

	// Initialize with no skills selected
	selected := make(map[string]bool)

	m := PromoteDemoteListModel{
		skills:   movableSkills,
		filtered: movableSkills,
		selected: selected,
		keys:     defaultPromoteDemoteListKeyMap(),
	}

	rows := m.skillsToRows(movableSkills)

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

func (m PromoteDemoteListModel) skillsToRows(skills []model.Skill) []table.Row {
	rows := make([]table.Row, len(skills))
	for i, s := range skills {
		checkbox := "[ ]"
		if m.selected[promoteDemoteSkillKey(s)] {
			checkbox := "[✓]"
			_ = checkbox
		}
		if m.selected[promoteDemoteSkillKey(s)] {
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

		// Determine target scope based on current scope
		var targetScope string
		switch s.Scope {
		case model.ScopeRepo:
			targetScope = "→ user"
		case model.ScopeUser:
			targetScope = "→ repo"
		default:
			targetScope = "-"
		}

		desc := s.Description
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}
		rows[i] = table.Row{
			checkbox,
			name,
			platform,
			scope,
			targetScope,
			desc,
		}
	}
	return rows
}

// Init implements tea.Model.
func (m PromoteDemoteListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m PromoteDemoteListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust table height based on window
		newHeight := max(msg.Height-12, 5) // Reserve space for title, info, help, status
		m.table.SetHeight(newHeight)

	case tea.KeyMsg:
		// Handle confirmation mode
		if m.confirmMode {
			switch msg.String() {
			case "y", "Y":
				m.result = PromoteDemoteListResult{
					Action:         m.confirmAction,
					SelectedSkills: m.getSelectedSkills(),
					RemoveSource:   m.removeSource,
				}
				m.quitting = true
				return m, tea.Quit
			case "n", "N", "esc":
				m.confirmMode = false
				m.confirmAction = PromoteDemoteActionNone
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
				m.selected[promoteDemoteSkillKey(skill)] = !m.selected[promoteDemoteSkillKey(skill)]
				m.table.SetRows(m.skillsToRows(m.filtered))
			}
			return m, nil

		case key.Matches(msg, m.keys.ToggleAll):
			// Count how many are currently selected
			selectedCount := 0
			for _, s := range m.filtered {
				if m.selected[promoteDemoteSkillKey(s)] {
					selectedCount++
				}
			}
			// If all or most are selected, deselect all; otherwise select all
			selectAll := selectedCount < len(m.filtered)/2+1
			for _, s := range m.filtered {
				m.selected[promoteDemoteSkillKey(s)] = selectAll
			}
			m.table.SetRows(m.skillsToRows(m.filtered))
			return m, nil

		case key.Matches(msg, m.keys.ToggleMove):
			m.removeSource = !m.removeSource
			return m, nil

		case key.Matches(msg, m.keys.Promote):
			// Get skills that can be promoted (repo scope -> user scope)
			promotableSkills := m.getPromotableSelectedSkills()
			if len(promotableSkills) > 0 {
				m.confirmMode = true
				m.confirmAction = PromoteDemoteActionPromote
			}
			return m, nil

		case key.Matches(msg, m.keys.Demote):
			// Get skills that can be demoted (user scope -> repo scope)
			demotableSkills := m.getDemotableSelectedSkills()
			if len(demotableSkills) > 0 {
				m.confirmMode = true
				m.confirmAction = PromoteDemoteActionDemote
			}
			return m, nil
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *PromoteDemoteListModel) applyFilter() {
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

func (m PromoteDemoteListModel) getSelectedSkill() model.Skill {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return model.Skill{}
}

func (m PromoteDemoteListModel) getSelectedSkills() []model.Skill {
	var selected []model.Skill
	for _, s := range m.skills {
		if m.selected[promoteDemoteSkillKey(s)] {
			selected = append(selected, s)
		}
	}
	return selected
}

// getPromotableSelectedSkills returns selected skills that can be promoted (repo -> user).
func (m PromoteDemoteListModel) getPromotableSelectedSkills() []model.Skill {
	var promotable []model.Skill
	for _, s := range m.skills {
		if m.selected[promoteDemoteSkillKey(s)] && s.Scope == model.ScopeRepo {
			promotable = append(promotable, s)
		}
	}
	return promotable
}

// getDemotableSelectedSkills returns selected skills that can be demoted (user -> repo).
func (m PromoteDemoteListModel) getDemotableSelectedSkills() []model.Skill {
	var demotable []model.Skill
	for _, s := range m.skills {
		if m.selected[promoteDemoteSkillKey(s)] && s.Scope == model.ScopeUser {
			demotable = append(demotable, s)
		}
	}
	return demotable
}

// View implements tea.Model.
func (m PromoteDemoteListModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	title := promoteDemoteListStyles.Title.Render("⬆️⬇️  Promote / Demote Skills")
	b.WriteString(title)
	b.WriteString("\n")

	// Info message
	info := promoteDemoteListStyles.Info.Render("Promote: repo → user (global)  |  Demote: user → repo (project)")
	b.WriteString(info)
	b.WriteString("\n")

	// Options line
	moveMode := "copy"
	if m.removeSource {
		moveMode = "move"
	}
	optionStr := promoteDemoteListStyles.Option.Render(fmt.Sprintf("Mode: %s (press 'm' to toggle)", moveMode))
	b.WriteString(optionStr)
	b.WriteString("\n\n")

	// Filter indicator
	if m.filter != "" || m.filtering {
		filterStr := promoteDemoteListStyles.Filter.Render("Filter: ")
		filterVal := promoteDemoteListStyles.FilterInput.Render(m.filter)
		if m.filtering {
			filterVal += "█"
		}
		b.WriteString(filterStr + filterVal + "\n\n")
	}

	// Confirmation dialog
	if m.confirmMode {
		b.WriteString(m.table.View())
		b.WriteString("\n\n")

		var actionText string
		var count int
		switch m.confirmAction {
		case PromoteDemoteActionPromote:
			count = len(m.getPromotableSelectedSkills())
			actionText = fmt.Sprintf("PROMOTE %d skill(s) from repo to user scope", count)
		case PromoteDemoteActionDemote:
			count = len(m.getDemotableSelectedSkills())
			actionText = fmt.Sprintf("DEMOTE %d skill(s) from user to repo scope", count)
		}

		modeText := "copy"
		if m.removeSource {
			modeText = "move (source will be removed)"
		}

		confirmMsg := fmt.Sprintf("⚠️  %s (%s)? (y/n)", actionText, modeText)
		b.WriteString(promoteDemoteListStyles.Confirm.Render(confirmMsg))
		return b.String()
	}

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n")

	// Status bar
	selectedCount := len(m.getSelectedSkills())
	promotableCount := len(m.getPromotableSelectedSkills())
	demotableCount := len(m.getDemotableSelectedSkills())

	status := fmt.Sprintf("%d selected (%d promotable, %d demotable) of %d",
		selectedCount, promotableCount, demotableCount, len(m.filtered))
	if m.filter != "" {
		status = fmt.Sprintf("%d selected (%d↑, %d↓), %d of %d shown (filtered)",
			selectedCount, promotableCount, demotableCount, len(m.filtered), len(m.skills))
	}
	b.WriteString(promoteDemoteListStyles.Status.Render(status))
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

func (m PromoteDemoteListModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"space toggle",
		"a toggle all",
		"p promote",
		"d demote",
		"m move/copy",
		"/ filter",
		"? help",
		"q quit",
	}
	return promoteDemoteListStyles.Help.Render(strings.Join(keys, " • "))
}

func (m PromoteDemoteListModel) renderFullHelp() string {
	help := `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  g/Home   Go to top
  G/End    Go to bottom

Selection:
  Space/Tab  Toggle current skill
  a          Toggle all skills

Actions:
  p        Promote selected repo skills to user scope
  d        Demote selected user skills to repo scope
  m        Toggle move/copy mode (move removes source)

Filter:
  /        Start filtering (by name, platform, scope, or description)
  Esc      Clear filter
  Enter    Finish filtering

General:
  ?        Toggle full help
  q        Quit without changes`
	return promoteDemoteListStyles.Help.Render(help)
}

// Result returns the result of the user interaction.
func (m PromoteDemoteListModel) Result() PromoteDemoteListResult {
	return m.result
}

// RunPromoteDemoteList runs the interactive promote/demote list and returns the result.
func RunPromoteDemoteList(skills []model.Skill) (PromoteDemoteListResult, error) {
	if len(skills) == 0 {
		return PromoteDemoteListResult{}, nil
	}

	mdl := NewPromoteDemoteListModel(skills)
	// Check if any movable skills exist after filtering
	if len(mdl.skills) == 0 {
		return PromoteDemoteListResult{}, nil
	}

	finalModel, err := tea.NewProgram(mdl, tea.WithAltScreen()).Run()
	if err != nil {
		return PromoteDemoteListResult{}, err
	}

	if m, ok := finalModel.(PromoteDemoteListModel); ok {
		return m.Result(), nil
	}

	return PromoteDemoteListResult{}, nil
}
