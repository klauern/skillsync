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
	Up          key.Binding
	Down        key.Binding
	Toggle      key.Binding
	ToggleAll   key.Binding
	View        key.Binding
	Confirm     key.Binding
	Filter      key.Binding
	ClearFlt    key.Binding
	NextPlat    key.Binding
	PrevPlat    key.Binding
	Help        key.Binding
	Quit        key.Binding
	Back        key.Binding
	ScrollLeft  key.Binding
	ScrollRight key.Binding
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
		View: key.NewBinding(
			key.WithKeys("enter", "v"),
			key.WithHelp("enter/v", "view details"),
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
		NextPlat: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "next platform"),
		),
		PrevPlat: key.NewBinding(
			key.WithKeys("shift+tab", "h"),
			key.WithHelp("S-tab/h", "prev platform"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Back: key.NewBinding(
			key.WithKeys("b", "esc"),
			key.WithHelp("b/esc", "back"),
		),
		ScrollLeft: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "scroll cols left"),
		),
		ScrollRight: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "scroll cols right"),
		),
	}
}

type deleteListPhase int

const (
	deleteListPhaseList deleteListPhase = iota
	deleteListPhaseDetail
)

// DeleteListModel is the BubbleTea model for interactive skill deletion.
type DeleteListModel struct {
	table           table.Model
	hScroll         horizontalTableState
	skills          []model.Skill
	filtered        []model.Skill
	selected        map[string]bool // map of skill key to selected state
	keys            deleteListKeyMap
	result          DeleteListResult
	filter          string
	filtering       bool
	platformOptions []model.Platform
	platformIndex   int // Index into platformOptions (-1 = all)
	showHelp        bool
	confirmMode     bool
	width           int
	height          int
	quitting        bool
	columnWidths    deleteListColumnWidths
	phase           deleteListPhase
	viewport        viewport.Model
	ready           bool
	detailSkill     model.Skill
	hOffset         int // horizontal column scroll offset (0 = show all)
}

// Styles for the delete list TUI.
var deleteListStyles = struct {
	Title          lipgloss.Style
	Help           lipgloss.Style
	Filter         lipgloss.Style
	FilterInput    lipgloss.Style
	Confirm        lipgloss.Style
	Status         lipgloss.Style
	Warning        lipgloss.Style
	Checkbox       lipgloss.Style
	DetailBox      lipgloss.Style
	DetailTitle    lipgloss.Style
	PlatformTab    lipgloss.Style
	PlatformActive lipgloss.Style
}{
	Title:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1")).Padding(0, 1),
	Help:           lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:         lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput:    lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Confirm:        lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true).Padding(1, 2),
	Status:         lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Warning:        lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true),
	Checkbox:       lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	DetailBox:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
	DetailTitle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")),
	PlatformTab:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	PlatformActive: lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("52")).Bold(true).Padding(0, 1),
}

const (
	deleteListCheckboxWidth = 3
	deleteListNameWidth     = 20
	deleteListPlatformWidth = 12
	deleteListScopeWidth    = 10
	deleteListDescWidth     = 40
	deleteListColumnPadding = 2
	deleteListDetailLines   = 3
	deleteListDetailGap     = 1
	deleteListDetailHeight  = deleteListDetailLines + 1 + 2 // title + content + border
	// deleteListMaxHOffset is the maximum horizontal scroll offset (3 scrollable cols - 1).
	deleteListMaxHOffset = 2
)

type deleteListColumnWidths struct {
	name     int
	platform int
	scope    int
	desc     int
}

// deleteListColumns returns visible table columns for the given terminal width and horizontal
// scroll offset. Scrollable columns are: platform(0), scope(1), description(2).
func deleteListColumns(totalWidth int, skills []model.Skill, hOffset int) ([]table.Column, deleteListColumnWidths) {
	hOffset = max(0, min(hOffset, deleteListMaxHOffset))

	widths := deleteListColumnWidths{
		name:     deleteListNameWidth,
		platform: deleteListPlatformWidth,
		scope:    deleteListScopeWidth,
		desc:     deleteListDescWidth,
	}

	showPlatform := hOffset == 0
	showScope := hOffset <= 1

	visibleColCount := 3 // checkbox + name + description always visible
	if showPlatform {
		visibleColCount++
	}
	if showScope {
		visibleColCount++
	}

	if totalWidth > 0 {
		baseTotal := deleteListCheckboxWidth + widths.name + widths.desc +
			(deleteListColumnPadding * visibleColCount)
		if showPlatform {
			baseTotal += widths.platform
		}
		if showScope {
			baseTotal += widths.scope
		}

		extra := totalWidth - baseTotal
		if extra > 0 {
			if showScope {
				maxScopeWidth := widths.scope
				for _, skill := range skills {
					if w := runewidth.StringWidth(skill.DisplayScope()); w > maxScopeWidth {
						maxScopeWidth = w
					}
				}
				scopeNeeded := maxScopeWidth - widths.scope
				if scopeNeeded > 0 {
					scopeExtra := min(scopeNeeded, extra)
					widths.scope += scopeExtra
					extra -= scopeExtra
				}
			}

			nameExtra := extra / 3
			descExtra := extra - nameExtra
			widths.name += nameExtra
			widths.desc += descExtra
		}
	}

	columns := []table.Column{
		{Title: " ", Width: deleteListCheckboxWidth},
		{Title: "Name", Width: widths.name},
	}
	if showPlatform {
		columns = append(columns, table.Column{Title: "Platform", Width: widths.platform})
	}
	if showScope {
		columns = append(columns, table.Column{Title: "Scope", Width: widths.scope})
	}
	columns = append(columns, table.Column{Title: "Description", Width: widths.desc})

	return columns, widths
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

	platformSet := make(map[model.Platform]bool)
	for _, s := range deletableSkills {
		platformSet[s.Platform] = true
	}

	var platformOptions []model.Platform
	for _, platform := range model.AllPlatforms() {
		if platformSet[platform] {
			platformOptions = append(platformOptions, platform)
		}
	}

	columns, columnWidths := deleteListColumns(0, deletableSkills, 0)

	// Initialize with no skills selected (deletion is opt-in)
	selected := make(map[string]bool)

	m := DeleteListModel{
		skills:          deletableSkills,
		filtered:        deletableSkills,
		selected:        selected,
		keys:            defaultDeleteListKeyMap(),
		columnWidths:    columnWidths,
		phase:           deleteListPhaseList,
		hScroll:         newHorizontalTableState(columns),
		platformOptions: platformOptions,
		platformIndex:   -1,
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
		Background(lipgloss.Color("52")).
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
			checkbox = "[x]"
		}

		row := table.Row{
			checkbox,
			truncateTableValue(s.Name, m.columnWidths.name),
		}
		if m.hOffset == 0 {
			row = append(row, truncateTableValue(string(s.Platform), m.columnWidths.platform))
		}
		if m.hOffset <= 1 {
			row = append(row, truncateTableValue(s.DisplayScope(), m.columnWidths.scope))
		}
		row = append(row, truncateTableValue(s.Description, m.columnWidths.desc))
		rows[i] = row
	}
	return rows
}

func (m *DeleteListModel) shiftHOffset(delta int) {
	newOffset := m.hOffset + delta
	if newOffset < 0 || newOffset > deleteListMaxHOffset {
		return
	}
	m.hOffset = newOffset
	m.updateColumns(m.width)
	m.table.SetRows(m.skillsToRows(m.filtered))
}

func (m *DeleteListModel) updateColumns(totalWidth int) {
	columns, widths := deleteListColumns(totalWidth, m.skills, m.hOffset)
	m.columnWidths = widths
	m.hScroll.SetColumns(columns)
	m.refreshTable()
}

func (m *DeleteListModel) refreshTable() {
	m.hScroll.Apply(&m.table, m.width, m.skillsToRows(m.filtered))
}

func (m DeleteListModel) detailPanelWidth() int {
	if m.width > 0 {
		return m.width
	}
	return deleteListCheckboxWidth + m.columnWidths.name + m.columnWidths.platform + m.columnWidths.scope + m.columnWidths.desc +
		(deleteListColumnPadding * 5)
}

func (m DeleteListModel) renderDetailPanel() string {
	width := m.detailPanelWidth()
	contentWidth := max(width-4, 10)

	skill := m.getSelectedSkill()
	description := strings.TrimSpace(skill.Description)
	if description == "" {
		description = "No description available."
	}

	lines := wrapText(description, contentWidth, deleteListDetailLines)
	lines = padLines(lines, deleteListDetailLines)

	header := deleteListStyles.DetailTitle.Render("Description (selected)")
	content := append([]string{header}, lines...)

	return deleteListStyles.DetailBox.Width(width).Render(strings.Join(content, "\n"))
}

// Init implements tea.Model.
func (m DeleteListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m DeleteListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.phase {
	case deleteListPhaseDetail:
		return m.updateDetail(msg)
	default:
		return m.updateList(msg)
	}
}

//nolint:gocyclo // interactive table/event handling is intentionally centralized here
func (m DeleteListModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust table height based on window
		newHeight := max(msg.Height-10-deleteListDetailHeight-deleteListDetailGap, 5) // Reserve space for title, warning, help, status, detail
		m.table.SetHeight(newHeight)
		m.updateColumns(msg.Width)

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
		case msg.String() == "left":
			if m.hScroll.MoveLeft() {
				m.refreshTable()
			}
			return m, nil

		case msg.String() == "right":
			if m.hScroll.MoveRight(m.width) {
				m.refreshTable()
			}
			return m, nil

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

		case key.Matches(msg, m.keys.NextPlat):
			if len(m.platformOptions) > 0 {
				m.platformIndex++
				if m.platformIndex >= len(m.platformOptions) {
					m.platformIndex = -1
				}
				m.applyFilter()
			}
			return m, nil

		case key.Matches(msg, m.keys.PrevPlat):
			if len(m.platformOptions) > 0 {
				m.platformIndex--
				if m.platformIndex < -1 {
					m.platformIndex = len(m.platformOptions) - 1
				}
				m.applyFilter()
			}
			return m, nil

		case key.Matches(msg, m.keys.Toggle):
			if len(m.filtered) > 0 {
				skill := m.getSelectedSkill()
				m.selected[deleteSkillKey(skill)] = !m.selected[deleteSkillKey(skill)]
				m.refreshTable()
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
			m.refreshTable()
			return m, nil

		case key.Matches(msg, m.keys.View):
			if len(m.filtered) > 0 {
				m.detailSkill = m.getSelectedSkill()
				m.phase = deleteListPhaseDetail
				m.ready = false
				m.ensureDetailViewport()
				return m, nil
			}
			return m, nil

		case key.Matches(msg, m.keys.Confirm):
			selectedSkills := m.getSelectedSkills()
			if len(selectedSkills) > 0 {
				m.confirmMode = true
			}
			return m, nil

		case key.Matches(msg, m.keys.ScrollLeft):
			m.shiftHOffset(-1)
			return m, nil

		case key.Matches(msg, m.keys.ScrollRight):
			m.shiftHOffset(1)
			return m, nil
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m DeleteListModel) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.phase = deleteListPhaseList
			return m, nil
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *DeleteListModel) applyFilter() {
	filtered := m.skills

	if m.platformIndex >= 0 && m.platformIndex < len(m.platformOptions) {
		selectedPlatform := m.platformOptions[m.platformIndex]
		var platformFiltered []model.Skill
		for _, s := range filtered {
			if s.Platform == selectedPlatform {
				platformFiltered = append(platformFiltered, s)
			}
		}
		filtered = platformFiltered
	}

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
	m.refreshTable()
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

	if m.phase == deleteListPhaseDetail {
		return m.viewDetail()
	}

	var b strings.Builder

	// Title with warning color
	title := deleteListStyles.Title.Render("🗑️  Delete Skills")
	b.WriteString(title)
	b.WriteString("\n")

	// Warning message
	warning := deleteListStyles.Warning.Render("Selection marks skills for deletion. Only repo and user scope skills can be deleted.")
	b.WriteString(warning)
	b.WriteString("\n\n")

	b.WriteString(m.renderPlatformTabs())
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

	// Detail panel
	b.WriteString(m.renderDetailPanel())
	b.WriteString("\n")

	// Status bar
	selectedCount := len(m.getSelectedSkills())
	status := fmt.Sprintf("%d skill(s) marked for deletion of %d", selectedCount, len(m.filtered))
	if m.filter != "" || m.platformIndex >= 0 {
		status = fmt.Sprintf("%d marked for deletion, %d of %d shown (filtered)", selectedCount, len(m.filtered), len(m.skills))
	}
	if m.hOffset > 0 || m.hOffset < deleteListMaxHOffset {
		status += "  " + hScrollIndicator(m.hOffset, deleteListMaxHOffset)
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

func (m DeleteListModel) viewDetail() string {
	m.ensureDetailViewport()
	if !m.ready {
		return "Loading..."
	}

	var b strings.Builder

	title := deleteListStyles.Title.Render(fmt.Sprintf("🗑️  Delete Skill Details: %s", m.detailSkill.Name))
	b.WriteString(title)
	b.WriteString("\n\n")

	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	scrollPercent := int(m.viewport.ScrollPercent() * 100)
	status := fmt.Sprintf("Scroll: %d%% • Press b or Esc to go back", scrollPercent)
	b.WriteString(deleteListStyles.Status.Render(status))
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
		b.WriteString(deleteListStyles.Help.Render(strings.Join(keys, " • ")))
	}

	return b.String()
}

func (m *DeleteListModel) ensureDetailViewport() {
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

func (m DeleteListModel) buildDetailContent(width int) string {
	var b strings.Builder

	skill := m.detailSkill
	if skill.Name == "" {
		return "No skill selected."
	}

	wrappedWidth := max(width, 10)
	indent := "  "

	b.WriteString(deleteListStyles.DetailTitle.Render("Skill"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "%sName: %s\n", indent, skill.Name)
	fmt.Fprintf(&b, "%sPlatform: %s\n", indent, skill.Platform)
	fmt.Fprintf(&b, "%sScope: %s\n", indent, skill.DisplayScope())
	if skill.Path != "" {
		fmt.Fprintf(&b, "%sPath: %s\n", indent, skill.Path)
	}

	marked := "no"
	if m.selected[deleteSkillKey(skill)] {
		marked = "yes"
	}
	fmt.Fprintf(&b, "%sMarked for deletion: %s\n", indent, marked)

	b.WriteString("\n")
	b.WriteString(deleteListStyles.DetailTitle.Render("Description"))
	b.WriteString("\n")

	description := strings.TrimSpace(skill.Description)
	if description == "" {
		description = "No description available."
	}
	b.WriteString(lipgloss.NewStyle().Width(wrappedWidth).Render(description))
	b.WriteString("\n")

	return b.String()
}

func (m DeleteListModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"←/→ scroll cols",
		"tab/S-tab platform",
		"space mark/unmark delete",
		"a toggle all",
		"enter details",
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
  ←/h      Show previous columns
  →/l      Show next columns
  g/Home   Go to top
  G/End    Go to bottom
  ←/→      Scroll columns left/right

Platform Filtering:
  h/S-tab   Previous platform
  l         Next platform

Selection:
  Space/Tab  Toggle current skill for deletion
  a          Toggle all skills

Actions:
  Enter/v  View skill details
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

func (m DeleteListModel) renderDetailHelp() string {
	help := `Navigation:
  ↑/k      Scroll up
  ↓/j      Scroll down
  PgUp     Page up
  PgDown   Page down

Actions:
  b/Esc    Go back to list

General:
  ?        Toggle full help
 q        Quit`
	return deleteListStyles.Help.Render(help)
}

func (m DeleteListModel) renderPlatformTabs() string {
	var tabs []string

	if m.platformIndex == -1 {
		tabs = append(tabs, deleteListStyles.PlatformActive.Render("[All]"))
	} else {
		tabs = append(tabs, deleteListStyles.PlatformTab.Render(" All "))
	}

	for i, platform := range m.platformOptions {
		if i == m.platformIndex {
			tabs = append(tabs, deleteListStyles.PlatformActive.Render(fmt.Sprintf("[%s]", platform)))
		} else {
			tabs = append(tabs, deleteListStyles.PlatformTab.Render(fmt.Sprintf(" %s ", platform)))
		}
	}

	return strings.Join(tabs, "")
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
