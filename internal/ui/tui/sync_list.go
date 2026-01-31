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

// SyncAction represents the action to perform after skill selection.
type SyncAction int

const (
	// SyncActionNone means no action was taken (user quit).
	SyncActionNone SyncAction = iota
	// SyncActionSync means the user wants to sync selected skills.
	SyncActionSync
	// SyncActionPreview means the user wants to preview a skill's diff.
	SyncActionPreview
)

// SyncListResult contains the result of the sync list TUI interaction.
type SyncListResult struct {
	Action         SyncAction
	SelectedSkills []model.Skill
	PreviewSkill   model.Skill
}

// syncListKeyMap defines the key bindings for the sync list.
type syncListKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Toggle    key.Binding
	ToggleAll key.Binding
	Preview   key.Binding
	Confirm   key.Binding
	Filter    key.Binding
	ClearFlt  key.Binding
	Help      key.Binding
	Quit      key.Binding
}

func defaultSyncListKeyMap() syncListKeyMap {
	return syncListKeyMap{
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
		Preview: key.NewBinding(
			key.WithKeys("p", "enter"),
			key.WithHelp("p/enter", "preview diff"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "sync selected"),
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

// SyncListModel is the BubbleTea model for interactive sync skill selection.
type SyncListModel struct {
	table          table.Model
	skills         []model.Skill
	filtered       []model.Skill
	selected       map[string]bool // map of skill name to selected state
	keys           syncListKeyMap
	result         SyncListResult
	filter         string
	filtering      bool
	showHelp       bool
	confirmMode    bool
	width          int
	height         int
	quitting       bool
	sourcePlatform model.Platform
	targetPlatform model.Platform
	columnWidths   syncListColumnWidths
}

// Styles for the sync list TUI.
var syncListStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Confirm     lipgloss.Style
	Status      lipgloss.Style
	Selected    lipgloss.Style
	Checkbox    lipgloss.Style
	DetailBox   lipgloss.Style
	DetailTitle lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:      lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput: lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Confirm:     lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true).Padding(1, 2),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Selected:    lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Checkbox:    lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	DetailBox:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
	DetailTitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
}

const (
	syncListCheckboxWidth = 3
	syncListNameWidth     = 20
	syncListScopeWidth    = 12
	syncListDescWidth     = 50
	syncListColumnPadding = 2
	syncListColumnCount   = 4
	syncListDetailLines   = 3
	syncListDetailGap     = 1
	syncListDetailHeight  = syncListDetailLines + 1 + 2 // title + content + border
)

type syncListColumnWidths struct {
	name  int
	scope int
	desc  int
}

func syncListColumns(totalWidth int) ([]table.Column, syncListColumnWidths) {
	widths := syncListColumnWidths{
		name:  syncListNameWidth,
		scope: syncListScopeWidth,
		desc:  syncListDescWidth,
	}

	if totalWidth > 0 {
		baseTotal := syncListCheckboxWidth + widths.name + widths.scope + widths.desc +
			(syncListColumnPadding * syncListColumnCount)
		extra := totalWidth - baseTotal
		if extra > 0 {
			scopeExtra := extra / 3
			descExtra := extra - scopeExtra
			widths.scope += scopeExtra
			widths.desc += descExtra
		}
	}

	columns := []table.Column{
		{Title: " ", Width: syncListCheckboxWidth}, // Checkbox column
		{Title: "Name", Width: widths.name},
		{Title: "Scope", Width: widths.scope},
		{Title: "Description", Width: widths.desc},
	}

	return columns, widths
}

// NewSyncListModel creates a new sync list model.
func NewSyncListModel(skills []model.Skill, source, target model.Platform) SyncListModel {
	columns, columnWidths := syncListColumns(0)

	// Sort skills alphabetically by name (case-insensitive)
	sort.Slice(skills, func(i, j int) bool {
		return strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
	})

	// Initialize all skills as selected by default
	selected := make(map[string]bool)
	for _, s := range skills {
		selected[s.Name] = true
	}

	m := SyncListModel{
		skills:         skills,
		filtered:       skills,
		selected:       selected,
		keys:           defaultSyncListKeyMap(),
		sourcePlatform: source,
		targetPlatform: target,
		columnWidths:   columnWidths,
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

func (m SyncListModel) skillsToRows(skills []model.Skill) []table.Row {
	rows := make([]table.Row, len(skills))
	for i, s := range skills {
		checkbox := "[ ]"
		if m.selected[s.Name] {
			checkbox = "[✓]"
		}

		name := truncateSyncListValue(s.Name, m.columnWidths.name)
		scope := truncateSyncListValue(s.DisplayScope(), m.columnWidths.scope)
		desc := truncateSyncListValue(s.Description, m.columnWidths.desc)
		rows[i] = table.Row{
			checkbox,
			name,
			scope,
			desc,
		}
	}
	return rows
}

func (m *SyncListModel) updateColumns(totalWidth int) {
	columns, widths := syncListColumns(totalWidth)
	m.columnWidths = widths
	m.table.SetColumns(columns)
}

func truncateSyncListValue(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(value) <= width {
		return value
	}
	if width <= 3 {
		return value[:width]
	}
	return value[:width-3] + "..."
}

func (m SyncListModel) detailPanelWidth() int {
	if m.width > 0 {
		return m.width
	}
	return syncListCheckboxWidth + m.columnWidths.name + m.columnWidths.scope + m.columnWidths.desc +
		(syncListColumnPadding * syncListColumnCount)
}

func (m SyncListModel) renderDetailPanel() string {
	width := m.detailPanelWidth()
	contentWidth := max(width-4, 10)

	skill := m.getSelectedSkill()
	description := strings.TrimSpace(skill.Description)
	if description == "" {
		description = "No description available."
	}

	lines := wrapText(description, contentWidth, syncListDetailLines)
	lines = padLines(lines, syncListDetailLines)

	header := syncListStyles.DetailTitle.Render("Description (selected)")
	content := append([]string{header}, lines...)

	return syncListStyles.DetailBox.Width(width).Render(strings.Join(content, "\n"))
}

// Init implements tea.Model.
func (m SyncListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m SyncListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust table height based on window
		newHeight := max(msg.Height-10-syncListDetailHeight-syncListDetailGap, 5) // Reserve space for title, help, status, detail
		m.table.SetHeight(newHeight)
		m.updateColumns(msg.Width)
		m.table.SetRows(m.skillsToRows(m.filtered))

	case tea.KeyMsg:
		// Handle confirmation mode
		if m.confirmMode {
			switch msg.String() {
			case "y", "Y":
				m.result = SyncListResult{
					Action:         SyncActionSync,
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
				m.selected[skill.Name] = !m.selected[skill.Name]
				m.table.SetRows(m.skillsToRows(m.filtered))
			}
			return m, nil

		case key.Matches(msg, m.keys.ToggleAll):
			// Count how many are currently selected
			selectedCount := 0
			for _, s := range m.filtered {
				if m.selected[s.Name] {
					selectedCount++
				}
			}
			// If all or most are selected, deselect all; otherwise select all
			selectAll := selectedCount < len(m.filtered)/2+1
			for _, s := range m.filtered {
				m.selected[s.Name] = selectAll
			}
			m.table.SetRows(m.skillsToRows(m.filtered))
			return m, nil

		case key.Matches(msg, m.keys.Preview):
			if len(m.filtered) > 0 {
				skill := m.getSelectedSkill()
				m.result = SyncListResult{
					Action:       SyncActionPreview,
					PreviewSkill: skill,
				}
				m.quitting = true
				return m, tea.Quit
			}
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

func (m *SyncListModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.skills
	} else {
		var filtered []model.Skill
		lowerFilter := strings.ToLower(m.filter)
		for _, s := range m.skills {
			if strings.Contains(strings.ToLower(s.Name), lowerFilter) ||
				strings.Contains(strings.ToLower(s.DisplayScope()), lowerFilter) ||
				strings.Contains(strings.ToLower(s.Description), lowerFilter) {
				filtered = append(filtered, s)
			}
		}
		m.filtered = filtered
	}
	m.table.SetRows(m.skillsToRows(m.filtered))
}

func (m SyncListModel) getSelectedSkill() model.Skill {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return model.Skill{}
}

func (m SyncListModel) getSelectedSkills() []model.Skill {
	var selected []model.Skill
	for _, s := range m.skills {
		if m.selected[s.Name] {
			selected = append(selected, s)
		}
	}
	return selected
}

// View implements tea.Model.
func (m SyncListModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	title := syncListStyles.Title.Render(fmt.Sprintf("🔄 Sync Skills: %s → %s", m.sourcePlatform, m.targetPlatform))
	b.WriteString(title)
	b.WriteString("\n\n")

	// Filter indicator
	if m.filter != "" || m.filtering {
		filterStr := syncListStyles.Filter.Render("Filter: ")
		filterVal := syncListStyles.FilterInput.Render(m.filter)
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
		confirmMsg := fmt.Sprintf("Sync %d skill(s) to %s? (y/n)", selectedCount, m.targetPlatform)
		b.WriteString(syncListStyles.Confirm.Render(confirmMsg))
		return b.String()
	}

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n")

	// Detail panel
	b.WriteString(m.renderDetailPanel())
	b.WriteString("\n")

	// Status bar
	selectedCount := len(m.getSelectedSkills())
	status := fmt.Sprintf("%d skill(s) selected of %d", selectedCount, len(m.filtered))
	if m.filter != "" {
		status = fmt.Sprintf("%d selected, %d of %d shown (filtered)", selectedCount, len(m.filtered), len(m.skills))
	}
	b.WriteString(syncListStyles.Status.Render(status))
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

func (m SyncListModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"space toggle",
		"a toggle all",
		"p preview",
		"y sync",
		"/ filter",
		"? help",
		"q quit",
	}
	return syncListStyles.Help.Render(strings.Join(keys, " • "))
}

func (m SyncListModel) renderFullHelp() string {
	help := `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  g/Home   Go to top
  G/End    Go to bottom

Selection:
  Space/Tab  Toggle current skill
  a          Toggle all skills

Actions:
  p/Enter  Preview diff for current skill
  y        Confirm and sync selected skills

Filter:
  /        Start filtering (by name, scope, or description)
  Esc      Clear filter
  Enter    Finish filtering

General:
  ?        Toggle full help
  q        Quit without syncing`
	return syncListStyles.Help.Render(help)
}

// Result returns the result of the user interaction.
func (m SyncListModel) Result() SyncListResult {
	return m.result
}

// RunSyncList runs the interactive sync list and returns the result.
func RunSyncList(skills []model.Skill, source, target model.Platform) (SyncListResult, error) {
	if len(skills) == 0 {
		return SyncListResult{}, nil
	}

	mdl := NewSyncListModel(skills, source, target)
	finalModel, err := tea.NewProgram(mdl, tea.WithAltScreen()).Run()
	if err != nil {
		return SyncListResult{}, err
	}

	if m, ok := finalModel.(SyncListModel); ok {
		return m.Result(), nil
	}

	return SyncListResult{}, nil
}
