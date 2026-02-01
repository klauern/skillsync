// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

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
	Detail   key.Binding
	Open     key.Binding
	Copy     key.Binding
	Filter   key.Binding
	ClearFlt key.Binding
	Help     key.Binding
	Back     key.Binding
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
		Detail: key.NewBinding(
			key.WithKeys("enter", "v"),
			key.WithHelp("enter/v", "details"),
		),
		Open: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open"),
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
		Back: key.NewBinding(
			key.WithKeys("b", "esc"),
			key.WithHelp("b/esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// DiscoverListModel is the BubbleTea model for interactive skill discovery.
type DiscoverListModel struct {
	table        table.Model
	skills       []model.Skill
	filtered     []model.Skill
	keys         discoverListKeyMap
	result       DiscoverListResult
	filter       string
	filtering    bool
	showHelp     bool
	width        int
	height       int
	columnWidths discoverListColumnWidths
	phase        discoverListPhase
	detailSkill  model.Skill
	viewport     viewport.Model
	ready        bool
	quitting     bool
}

// Styles for the discover list TUI.
var discoverListStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Status      lipgloss.Style
	DetailBox   lipgloss.Style
	DetailTitle lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:      lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput: lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	DetailBox:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
	DetailTitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
}

type discoverListPhase int

const (
	discoverListPhaseList discoverListPhase = iota
	discoverListPhaseDetail
)

const (
	discoverListNameWidth     = 25
	discoverListPlatformWidth = 12
	discoverListScopeWidth    = 15
	discoverListDescWidth     = 45
	discoverListColumnPadding = 2
	discoverListColumnCount   = 4
	discoverListDetailLines   = 3
	discoverListDetailGap     = 1
	discoverListDetailHeight  = discoverListDetailLines + 1 + 2 // title + content + border
)

type discoverListColumnWidths struct {
	name     int
	platform int
	scope    int
	desc     int
}

// NewDiscoverListModel creates a new discover list model.
func NewDiscoverListModel(skills []model.Skill) DiscoverListModel {
	columns, columnWidths := discoverListColumns(0, skills)

	// Sort skills alphabetically by name (case-insensitive)
	sort.Slice(skills, func(i, j int) bool {
		return strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
	})

	m := DiscoverListModel{
		skills:       skills,
		filtered:     skills,
		keys:         defaultDiscoverListKeyMap(),
		columnWidths: columnWidths,
		phase:        discoverListPhaseList,
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

func (m DiscoverListModel) skillsToRows(skills []model.Skill) []table.Row {
	rows := make([]table.Row, len(skills))
	for i, s := range skills {
		name := truncateDiscoverListValue(s.Name, m.columnWidths.name)
		platform := truncateDiscoverListValue(string(s.Platform), m.columnWidths.platform)
		scope := truncateDiscoverListValue(s.DisplayScope(), m.columnWidths.scope)
		desc := truncateDiscoverListValue(s.Description, m.columnWidths.desc)
		rows[i] = table.Row{
			name,
			platform,
			scope,
			desc,
		}
	}
	return rows
}

func discoverListColumns(totalWidth int, skills []model.Skill) ([]table.Column, discoverListColumnWidths) {
	widths := discoverListColumnWidths{
		name:     discoverListNameWidth,
		platform: discoverListPlatformWidth,
		scope:    discoverListScopeWidth,
		desc:     discoverListDescWidth,
	}

	if totalWidth > 0 {
		baseTotal := widths.name + widths.platform + widths.scope + widths.desc +
			(discoverListColumnPadding * discoverListColumnCount)
		extra := totalWidth - baseTotal
		if extra > 0 {
			maxScopeWidth := widths.scope
			for _, skill := range skills {
				scopeWidth := runewidth.StringWidth(skill.DisplayScope())
				if scopeWidth > maxScopeWidth {
					maxScopeWidth = scopeWidth
				}
			}

			scopeNeeded := maxScopeWidth - widths.scope
			if scopeNeeded > 0 {
				scopeExtra := min(scopeNeeded, extra)
				widths.scope += scopeExtra
				extra -= scopeExtra
			}

			nameExtra := extra / 3
			descExtra := extra - nameExtra
			widths.name += nameExtra
			widths.desc += descExtra
		}
	}

	columns := []table.Column{
		{Title: "Name", Width: widths.name},
		{Title: "Platform", Width: widths.platform},
		{Title: "Scope", Width: widths.scope},
		{Title: "Description", Width: widths.desc},
	}

	return columns, widths
}

func (m *DiscoverListModel) updateColumns(totalWidth int) {
	columns, widths := discoverListColumns(totalWidth, m.skills)
	m.columnWidths = widths
	m.table.SetColumns(columns)
}

func truncateDiscoverListValue(value string, width int) string {
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

func (m DiscoverListModel) detailPanelWidth() int {
	if m.width > 0 {
		return m.width
	}
	return m.columnWidths.name + m.columnWidths.platform + m.columnWidths.scope + m.columnWidths.desc +
		(discoverListColumnPadding * discoverListColumnCount)
}

func (m DiscoverListModel) renderDetailPanel() string {
	width := m.detailPanelWidth()
	contentWidth := max(width-4, 10)

	skill := m.getSelectedSkill()
	description := strings.TrimSpace(skill.Description)
	if description == "" {
		description = "No description available."
	}

	lines := wrapText(description, contentWidth, discoverListDetailLines)
	lines = padLines(lines, discoverListDetailLines)

	header := discoverListStyles.DetailTitle.Render("Description (selected)")
	content := append([]string{header}, lines...)

	return discoverListStyles.DetailBox.Width(width).Render(strings.Join(content, "\n"))
}

// Init implements tea.Model.
func (m DiscoverListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m DiscoverListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.phase {
	case discoverListPhaseDetail:
		return m.updateDetail(msg)
	default:
		return m.updateList(msg)
	}
}

func (m DiscoverListModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust table height based on window
		newHeight := max(msg.Height-10-discoverListDetailHeight-discoverListDetailGap, 5) // Reserve space for title, help, status, detail
		m.table.SetHeight(newHeight)
		m.updateColumns(msg.Width)
		m.table.SetRows(m.skillsToRows(m.filtered))

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

		case key.Matches(msg, m.keys.Detail):
			if len(m.filtered) > 0 {
				m.detailSkill = m.getSelectedSkill()
				m.phase = discoverListPhaseDetail
				m.ready = false
				m.ensureDetailViewport()
				return m, nil
			}
			return m, nil

		case key.Matches(msg, m.keys.Open):
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

func (m DiscoverListModel) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureDetailViewport()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.Back):
			m.phase = discoverListPhaseList
			return m, nil
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
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
	m.table.SetRows(m.skillsToRows(m.filtered))
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

	if m.phase == discoverListPhaseDetail {
		return m.viewDetail()
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

	// Detail panel
	b.WriteString(m.renderDetailPanel())
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

func (m DiscoverListModel) viewDetail() string {
	m.ensureDetailViewport()
	if !m.ready {
		return "Loading..."
	}

	var b strings.Builder

	title := discoverListStyles.Title.Render(fmt.Sprintf("🔍 Skill Details: %s", m.detailSkill.Name))
	b.WriteString(title)
	b.WriteString("\n\n")

	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	scrollPercent := int(m.viewport.ScrollPercent() * 100)
	status := fmt.Sprintf("Scroll: %d%% • Press b or Esc to go back", scrollPercent)
	b.WriteString(discoverListStyles.Status.Render(status))
	b.WriteString("\n")

	if m.showHelp {
		b.WriteString("\n")
		b.WriteString(m.renderDetailHelp())
	} else {
		keys := []string{
			"↑/↓ scroll",
			"b back",
			"? help",
			"q quit",
		}
		b.WriteString(discoverListStyles.Help.Render(strings.Join(keys, " • ")))
	}

	return b.String()
}

func (m *DiscoverListModel) ensureDetailViewport() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	headerHeight := 4
	footerHeight := 4
	viewportHeight := max(m.height-headerHeight-footerHeight, 5)

	if !m.ready {
		m.viewport = viewport.New(m.width-2, viewportHeight)
		m.viewport.SetContent(m.buildDetailContent(m.viewport.Width))
		m.ready = true
		return
	}

	m.viewport.Width = m.width - 2
	m.viewport.Height = viewportHeight
	m.viewport.SetContent(m.buildDetailContent(m.viewport.Width))
}

func (m DiscoverListModel) buildDetailContent(width int) string {
	var b strings.Builder

	skill := m.detailSkill
	if skill.Name == "" {
		return "No skill selected."
	}

	wrappedWidth := max(width, 10)
	indent := "  "

	b.WriteString(discoverListStyles.DetailTitle.Render("Skill"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%sName: %s\n", indent, skill.Name))
	b.WriteString(fmt.Sprintf("%sPlatform: %s\n", indent, skill.Platform))
	b.WriteString(fmt.Sprintf("%sScope: %s\n", indent, skill.DisplayScope()))
	if skill.Path != "" {
		b.WriteString(fmt.Sprintf("%sPath: %s\n", indent, skill.Path))
	}
	if len(skill.Tools) > 0 {
		b.WriteString(fmt.Sprintf("%sTools: %s\n", indent, strings.Join(skill.Tools, ", ")))
	}

	b.WriteString("\n")
	b.WriteString(discoverListStyles.DetailTitle.Render("Description"))
	b.WriteString("\n")

	description := strings.TrimSpace(skill.Description)
	if description == "" {
		description = "No description available."
	}
	b.WriteString(lipgloss.NewStyle().Width(wrappedWidth).Render(description))
	b.WriteString("\n")

	return b.String()
}

func (m DiscoverListModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"enter details",
		"o open",
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
  Enter/v  View details
  o        Open skill content
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

func (m DiscoverListModel) renderDetailHelp() string {
	help := `Navigation:
  ↑/k      Scroll up
  ↓/j      Scroll down
  g/Home   Top
  G/End    Bottom

Actions:
  b/Esc    Back to list

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
