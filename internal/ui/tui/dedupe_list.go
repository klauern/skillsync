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
	"github.com/klauern/skillsync/internal/similarity"
)

// DedupeAction represents the action to perform after dedupe selection.
type DedupeAction int

const (
	// DedupeActionNone means no action was taken (user quit).
	DedupeActionNone DedupeAction = iota
	// DedupeActionDelete means the user wants to delete selected duplicate skills.
	DedupeActionDelete
)

// DedupeListResult contains the result of the dedupe list TUI interaction.
type DedupeListResult struct {
	Action         DedupeAction
	SelectedSkills []model.Skill
}

// DuplicateGroup represents a group of skills that are similar/duplicates.
type DuplicateGroup struct {
	Skills       []model.Skill
	NameScore    float64
	ContentScore float64
}

// dedupeListKeyMap defines the key bindings for the dedupe list.
type dedupeListKeyMap struct {
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

func defaultDedupeListKeyMap() dedupeListKeyMap {
	return dedupeListKeyMap{
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

// DedupeListModel is the BubbleTea model for interactive duplicate skill management.
type DedupeListModel struct {
	table       table.Model
	duplicates  []*similarity.ComparisonResult
	flatSkills  []model.Skill // flattened list of skills from duplicate pairs
	filtered    []model.Skill
	selected    map[string]bool // map of skill key to selected state
	keys        dedupeListKeyMap
	result      DedupeListResult
	filter      string
	filtering   bool
	showHelp    bool
	confirmMode bool
	width       int
	height      int
	quitting    bool
}

// Styles for the dedupe list TUI.
var dedupeListStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Confirm     lipgloss.Style
	Status      lipgloss.Style
	Warning     lipgloss.Style
	Checkbox    lipgloss.Style
	Duplicate   lipgloss.Style
	Score       lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:      lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput: lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Confirm:     lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true).Padding(1, 2),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Warning:     lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true),
	Checkbox:    lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	Duplicate:   lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
	Score:       lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
}

// dedupeSkillKey creates a unique key for a skill (platform + scope + name combination).
func dedupeSkillKey(s model.Skill) string {
	return fmt.Sprintf("%s:%s:%s", s.Platform, s.Scope, s.Name)
}

// NewDedupeListModel creates a new dedupe list model from comparison results.
// Only writable skills (repo and user scope) can be selected for deletion.
func NewDedupeListModel(duplicates []*similarity.ComparisonResult) DedupeListModel {
	// Flatten duplicates into a list of skills, avoiding duplicates
	seenSkills := make(map[string]bool)
	var flatSkills []model.Skill

	for _, dup := range duplicates {
		// Add both skills from each pair if not already seen
		key1 := dedupeSkillKey(dup.Skill1)
		key2 := dedupeSkillKey(dup.Skill2)

		if !seenSkills[key1] {
			seenSkills[key1] = true
			flatSkills = append(flatSkills, dup.Skill1)
		}
		if !seenSkills[key2] {
			seenSkills[key2] = true
			flatSkills = append(flatSkills, dup.Skill2)
		}
	}

	// Filter to only include deletable skills (repo and user scopes)
	var deletableSkills []model.Skill
	for _, s := range flatSkills {
		if s.Scope == model.ScopeRepo || s.Scope == model.ScopeUser {
			deletableSkills = append(deletableSkills, s)
		}
	}

	columns := []table.Column{
		{Title: " ", Width: 3},            // Checkbox column
		{Title: "Name", Width: 22},        // Skill name
		{Title: "Platform", Width: 12},    // Platform
		{Title: "Scope", Width: 8},        // Scope
		{Title: "Similar To", Width: 22},  // Similar skill name
		{Title: "Name%", Width: 6},        // Name similarity
		{Title: "Content%", Width: 8},     // Content similarity
		{Title: "Description", Width: 25}, // Description
	}

	// Initialize with no skills selected
	selected := make(map[string]bool)

	m := DedupeListModel{
		duplicates: duplicates,
		flatSkills: deletableSkills,
		filtered:   deletableSkills,
		selected:   selected,
		keys:       defaultDedupeListKeyMap(),
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

// findSimilarSkill finds the best matching similar skill for a given skill.
func (m DedupeListModel) findSimilarSkill(skill model.Skill) (model.Skill, float64, float64) {
	skillKey := dedupeSkillKey(skill)
	var bestMatch model.Skill
	var bestNameScore, bestContentScore float64

	for _, dup := range m.duplicates {
		key1 := dedupeSkillKey(dup.Skill1)
		key2 := dedupeSkillKey(dup.Skill2)

		if key1 == skillKey {
			// This skill is Skill1, its pair is Skill2
			score := dup.NameScore + dup.ContentScore
			if score > bestNameScore+bestContentScore {
				bestMatch = dup.Skill2
				bestNameScore = dup.NameScore
				bestContentScore = dup.ContentScore
			}
		} else if key2 == skillKey {
			// This skill is Skill2, its pair is Skill1
			score := dup.NameScore + dup.ContentScore
			if score > bestNameScore+bestContentScore {
				bestMatch = dup.Skill1
				bestNameScore = dup.NameScore
				bestContentScore = dup.ContentScore
			}
		}
	}

	return bestMatch, bestNameScore, bestContentScore
}

func (m DedupeListModel) skillsToRows(skills []model.Skill) []table.Row {
	rows := make([]table.Row, len(skills))
	for i, s := range skills {
		checkbox := "[ ]"
		if m.selected[dedupeSkillKey(s)] {
			checkbox = "[✓]"
		}

		name := s.Name
		if len(name) > 22 {
			name = name[:19] + "..."
		}
		platform := string(s.Platform)
		if len(platform) > 12 {
			platform = platform[:9] + "..."
		}
		scope := s.DisplayScope()
		if len(scope) > 8 {
			scope = scope[:5] + "..."
		}

		// Find similar skill info
		similarSkill, nameScore, contentScore := m.findSimilarSkill(s)
		similarName := similarSkill.Name
		if len(similarName) > 22 {
			similarName = similarName[:19] + "..."
		}

		nameScoreStr := fmt.Sprintf("%.0f%%", nameScore*100)
		contentScoreStr := fmt.Sprintf("%.0f%%", contentScore*100)

		desc := s.Description
		if len(desc) > 25 {
			desc = desc[:22] + "..."
		}

		rows[i] = table.Row{
			checkbox,
			name,
			platform,
			scope,
			similarName,
			nameScoreStr,
			contentScoreStr,
			desc,
		}
	}
	return rows
}

// Init implements tea.Model.
func (m DedupeListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m DedupeListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust table height based on window
		newHeight := max(msg.Height-12, 5) // Reserve space for title, warning, help, status
		m.table.SetHeight(newHeight)

	case tea.KeyMsg:
		// Handle confirmation mode
		if m.confirmMode {
			switch msg.String() {
			case "y", "Y":
				m.result = DedupeListResult{
					Action:         DedupeActionDelete,
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
				m.selected[dedupeSkillKey(skill)] = !m.selected[dedupeSkillKey(skill)]
				m.table.SetRows(m.skillsToRows(m.filtered))
			}
			return m, nil

		case key.Matches(msg, m.keys.ToggleAll):
			// Count how many are currently selected
			selectedCount := 0
			for _, s := range m.filtered {
				if m.selected[dedupeSkillKey(s)] {
					selectedCount++
				}
			}
			// If all or most are selected, deselect all; otherwise select all
			selectAll := selectedCount < len(m.filtered)/2+1
			for _, s := range m.filtered {
				m.selected[dedupeSkillKey(s)] = selectAll
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

func (m *DedupeListModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.flatSkills
	} else {
		var filtered []model.Skill
		lowerFilter := strings.ToLower(m.filter)
		for _, s := range m.flatSkills {
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

func (m DedupeListModel) getSelectedSkill() model.Skill {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return model.Skill{}
}

func (m DedupeListModel) getSelectedSkills() []model.Skill {
	var selected []model.Skill
	for _, s := range m.flatSkills {
		if m.selected[dedupeSkillKey(s)] {
			selected = append(selected, s)
		}
	}
	return selected
}

// View implements tea.Model.
func (m DedupeListModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	title := dedupeListStyles.Title.Render("🔍 Dedupe Skills - Find and Remove Duplicates")
	b.WriteString(title)
	b.WriteString("\n")

	// Info message
	info := dedupeListStyles.Warning.Render("Select duplicate skills to delete. Only repo/user scope skills shown.")
	b.WriteString(info)
	b.WriteString("\n")

	// Duplicate count info
	dupInfo := fmt.Sprintf("Found %d duplicate pairs across %d skills", len(m.duplicates), len(m.flatSkills))
	b.WriteString(dedupeListStyles.Status.Render(dupInfo))
	b.WriteString("\n\n")

	// Filter indicator
	if m.filter != "" || m.filtering {
		filterStr := dedupeListStyles.Filter.Render("Filter: ")
		filterVal := dedupeListStyles.FilterInput.Render(m.filter)
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
		confirmMsg := fmt.Sprintf("⚠️  DELETE %d duplicate skill(s)? This cannot be undone! (y/n)", selectedCount)
		b.WriteString(dedupeListStyles.Confirm.Render(confirmMsg))
		return b.String()
	}

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n")

	// Status bar
	selectedCount := len(m.getSelectedSkills())
	status := fmt.Sprintf("%d skill(s) selected for deletion of %d", selectedCount, len(m.filtered))
	if m.filter != "" {
		status = fmt.Sprintf("%d selected, %d of %d shown (filtered)", selectedCount, len(m.filtered), len(m.flatSkills))
	}
	b.WriteString(dedupeListStyles.Status.Render(status))
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

func (m DedupeListModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"space toggle",
		"a toggle all",
		"d delete",
		"/ filter",
		"? help",
		"q quit",
	}
	return dedupeListStyles.Help.Render(strings.Join(keys, " • "))
}

func (m DedupeListModel) renderFullHelp() string {
	help := `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  g/Home   Go to top
  G/End    Go to bottom

Selection:
  Space/Tab  Toggle current skill for deletion
  a          Toggle all skills

Actions:
  d        Confirm and delete selected duplicate skills

Filter:
  /        Start filtering (by name, platform, scope, or description)
  Esc      Clear filter
  Enter    Finish filtering

General:
  ?        Toggle full help
  q        Quit without changes

Tip: Keep the version you want, delete the duplicates!`
	return dedupeListStyles.Help.Render(help)
}

// Result returns the result of the user interaction.
func (m DedupeListModel) Result() DedupeListResult {
	return m.result
}

// RunDedupeList runs the interactive dedupe list and returns the result.
func RunDedupeList(duplicates []*similarity.ComparisonResult) (DedupeListResult, error) {
	if len(duplicates) == 0 {
		return DedupeListResult{}, nil
	}

	mdl := NewDedupeListModel(duplicates)
	// Check if any deletable skills exist after filtering
	if len(mdl.flatSkills) == 0 {
		return DedupeListResult{}, nil
	}

	finalModel, err := tea.NewProgram(mdl, tea.WithAltScreen()).Run()
	if err != nil {
		return DedupeListResult{}, err
	}

	if m, ok := finalModel.(DedupeListModel); ok {
		return m.Result(), nil
	}

	return DedupeListResult{}, nil
}
