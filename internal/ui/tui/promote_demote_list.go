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
	"github.com/mattn/go-runewidth"

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

const (
	promoteDemoteCheckboxWidth  = 3
	promoteDemoteNameWidth      = 25
	promoteDemotePlatformWidth  = 12
	promoteDemoteScopeWidth     = 10
	promoteDemoteCanMoveToWidth = 12
	promoteDemoteDescWidth      = 30
	promoteDemoteColumnPadding  = 2
	// promoteDemoteMaxHOffset is the maximum horizontal scroll offset (4 scrollable cols - 1).
	promoteDemoteMaxHOffset = 3
)

// PromoteDemoteListResult contains the result of the promote/demote list TUI interaction.
type PromoteDemoteListResult struct {
	Action         PromoteDemoteAction
	SelectedSkills []model.Skill
	RemoveSource   bool // Whether to remove the source after operation (move vs copy)
}

// promoteDemoteListKeyMap defines the key bindings for the promote/demote list.
type promoteDemoteListKeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Toggle      key.Binding
	ToggleAll   key.Binding
	Promote     key.Binding
	Demote      key.Binding
	ToggleMove  key.Binding
	Filter      key.Binding
	ClearFlt    key.Binding
	NextPlat    key.Binding
	PrevPlat    key.Binding
	Help        key.Binding
	Quit        key.Binding
	ScrollLeft  key.Binding
	ScrollRight key.Binding
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

// promoteDemoteListState is the shared mutable state used by render/filter callbacks.
// It is intentionally separate so Bubble Tea's copied models continue to see live updates.
type promoteDemoteListState struct {
	skills          []model.Skill
	filtered        []model.Skill
	selected        map[string]bool
	platformOptions []model.Platform
	platformIndex   int
	removeSource    bool
	columnWidths    promoteDemoteColumnWidths
	hOffset         int
}

// PromoteDemoteListModel is the BubbleTea model for interactive skill promotion/demotion.
type PromoteDemoteListModel struct {
	ListModel[model.Skill]
	state         *promoteDemoteListState
	keys          promoteDemoteListKeyMap
	result        PromoteDemoteListResult
	confirmAction PromoteDemoteAction
	// Mirror fields retained for tests and direct inspection.
	hScroll         horizontalTableState
	skills          []model.Skill
	filtered        []model.Skill
	selected        map[string]bool
	platformOptions []model.Platform
	platformIndex   int
	removeSource    bool
	columnWidths    promoteDemoteColumnWidths
	hOffset         int
}

// Styles for the promote/demote list TUI.
var promoteDemoteListStyles = struct {
	Title          lipgloss.Style
	Help           lipgloss.Style
	Filter         lipgloss.Style
	FilterInput    lipgloss.Style
	Confirm        lipgloss.Style
	Status         lipgloss.Style
	Info           lipgloss.Style
	Checkbox       lipgloss.Style
	Promote        lipgloss.Style
	Demote         lipgloss.Style
	Option         lipgloss.Style
	PlatformTab    lipgloss.Style
	PlatformActive lipgloss.Style
	Description    lipgloss.Style
}{
	Title:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:           lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:         lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput:    lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Confirm:        lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true).Padding(1, 2),
	Status:         lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Info:           lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Checkbox:       lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	Promote:        lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Demote:         lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true),
	Option:         lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
	PlatformTab:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	PlatformActive: lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(true).Padding(0, 1),
	Description:    lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1),
}

type promoteDemoteColumnWidths struct {
	name      int
	platform  int
	scope     int
	canMoveTo int
	desc      int
}

func defaultPromoteDemoteColumnWidths() promoteDemoteColumnWidths {
	return promoteDemoteColumnWidths{
		name:      promoteDemoteNameWidth,
		platform:  promoteDemotePlatformWidth,
		scope:     promoteDemoteScopeWidth,
		canMoveTo: promoteDemoteCanMoveToWidth,
		desc:      promoteDemoteDescWidth,
	}
}

// promoteDemoteListColumns returns the visible table columns and their widths for the given
// terminal width and horizontal scroll offset. hOffset hides the leading scrollable columns
// (platform → scope → canMoveTo) one at a time, redistributing their space to the remaining ones.
func promoteDemoteListColumns(totalWidth int, skills []model.Skill, hOffset int) ([]table.Column, promoteDemoteColumnWidths) {
	hOffset = max(0, min(hOffset, promoteDemoteMaxHOffset))

	widths := defaultPromoteDemoteColumnWidths()

	// Visible scrollable columns: platform(0), scope(1), canMoveTo(2), description(3)
	// hOffset hides columns from the front of that list.
	showPlatform := hOffset == 0
	showScope := hOffset <= 1
	showCanMoveTo := hOffset <= 2
	// description is always visible (it's the last scrollable column)

	visibleColCount := 2 // checkbox + name always shown
	if showPlatform {
		visibleColCount++
	}
	if showScope {
		visibleColCount++
	}
	if showCanMoveTo {
		visibleColCount++
	}
	visibleColCount++ // description

	if totalWidth > 0 {
		baseTotal := promoteDemoteCheckboxWidth + widths.name + widths.desc +
			(promoteDemoteColumnPadding * visibleColCount)
		if showPlatform {
			baseTotal += widths.platform
		}
		if showScope {
			baseTotal += widths.scope
		}
		if showCanMoveTo {
			baseTotal += widths.canMoveTo
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
		{Title: " ", Width: promoteDemoteCheckboxWidth},
		{Title: "Name", Width: widths.name},
	}
	if showPlatform {
		columns = append(columns, table.Column{Title: "Platform", Width: widths.platform})
	}
	if showScope {
		columns = append(columns, table.Column{Title: "Scope", Width: widths.scope})
	}
	if showCanMoveTo {
		columns = append(columns, table.Column{Title: "Can Move To", Width: widths.canMoveTo})
	}
	columns = append(columns, table.Column{Title: "Description", Width: widths.desc})

	return columns, widths
}

// promoteDemoteSkillKey creates a unique key for a skill (platform + scope + name combination).
func promoteDemoteSkillKey(s model.Skill) string {
	return fmt.Sprintf("%s:%s:%s", s.Platform, s.Scope, s.Name)
}

func newPromoteDemoteListState(skills []model.Skill) *promoteDemoteListState {
	var movableSkills []model.Skill
	for _, s := range skills {
		if s.Scope == model.ScopeRepo || s.Scope == model.ScopeUser {
			movableSkills = append(movableSkills, s)
		}
	}

	sort.Slice(movableSkills, func(i, j int) bool {
		return strings.ToLower(movableSkills[i].Name) < strings.ToLower(movableSkills[j].Name)
	})

	platformSet := make(map[model.Platform]bool)
	for _, s := range movableSkills {
		platformSet[s.Platform] = true
	}

	var platformOptions []model.Platform
	for _, platform := range model.AllPlatforms() {
		if platformSet[platform] {
			platformOptions = append(platformOptions, platform)
		}
	}

	return &promoteDemoteListState{
		skills:          movableSkills,
		filtered:        movableSkills,
		selected:        make(map[string]bool),
		platformOptions: platformOptions,
		platformIndex:   -1,
		columnWidths:    defaultPromoteDemoteColumnWidths(),
	}
}

func newPromoteDemoteConfig(state *promoteDemoteListState) ListConfig[model.Skill] {
	return ListConfig[model.Skill]{
		Title: "⬆️⬇️  Promote / Demote Skills",
		Columns: []table.Column{
			{Title: " ", Width: promoteDemoteCheckboxWidth},
			{Title: "Name", Width: promoteDemoteNameWidth},
			{Title: "Platform", Width: promoteDemotePlatformWidth},
			{Title: "Scope", Width: promoteDemoteScopeWidth},
			{Title: "Can Move To", Width: promoteDemoteCanMoveToWidth},
			{Title: "Description", Width: promoteDemoteDescWidth},
		},
		ToRows: func(skills []model.Skill) []table.Row {
			return promoteDemoteListRows(state, skills)
		},
		Matches: func(skill model.Skill, lowerFilter string) bool {
			if state.platformIndex >= 0 && state.platformIndex < len(state.platformOptions) {
				if skill.Platform != state.platformOptions[state.platformIndex] {
					return false
				}
			}
			if lowerFilter == "" {
				return true
			}
			return strings.Contains(strings.ToLower(skill.Name), lowerFilter) ||
				strings.Contains(strings.ToLower(string(skill.Platform)), lowerFilter) ||
				strings.Contains(strings.ToLower(skill.DisplayScope()), lowerFilter) ||
				strings.Contains(strings.ToLower(skill.Description), lowerFilter)
		},
		ReservedLines: 12,
		Header: func() string {
			return promoteDemoteListHeader(state)
		},
		ExtraBody: func(m *ListModel[model.Skill]) string {
			return promoteDemoteListExtraBody(state, m)
		},
		StatusText: func(filtered, total int, filter string) string {
			return promoteDemoteListStatus(state, filtered, total, filter)
		},
		ShortHelp: func() string {
			return promoteDemoteShortHelp()
		},
		FullHelp: func() string {
			return promoteDemoteFullHelp()
		},
		OnWindowSize: func(m *ListModel[model.Skill], width, height int) {
			_ = height
			promoteDemoteUpdateColumns(state, &m.table, m.filtered, width)
		},
	}
}

func promoteDemoteListRows(state *promoteDemoteListState, skills []model.Skill) []table.Row {
	widths := state.columnWidths
	if widths.desc == 0 {
		widths = defaultPromoteDemoteColumnWidths()
	}
	rows := make([]table.Row, len(skills))
	for i, s := range skills {
		checkbox := "[ ]"
		if state.selected[promoteDemoteSkillKey(s)] {
			checkbox = "[✓]"
		}

		var targetScope string
		switch s.Scope {
		case model.ScopeRepo:
			targetScope = "→ user"
		case model.ScopeUser:
			targetScope = "→ repo"
		default:
			targetScope = "-"
		}

		row := table.Row{
			checkbox,
			truncateTableValue(s.Name, widths.name),
		}
		if state.hOffset == 0 {
			row = append(row, truncateTableValue(string(s.Platform), widths.platform))
		}
		if state.hOffset <= 1 {
			row = append(row, truncateTableValue(s.DisplayScope(), widths.scope))
		}
		if state.hOffset <= 2 {
			row = append(row, truncateTableValue(targetScope, widths.canMoveTo))
		}
		row = append(row, truncateTableValue(s.Description, widths.desc))
		rows[i] = row
	}
	return rows
}

func promoteDemoteUpdateColumns(state *promoteDemoteListState, t *table.Model, skills []model.Skill, totalWidth int) {
	columns, widths := promoteDemoteListColumns(totalWidth, skills, state.hOffset)
	state.columnWidths = widths
	rows := promoteDemoteListRows(state, skills)
	if state.hOffset > 0 {
		// keep hScroll state in sync with the columns that are actually visible
		state.hOffset = max(0, min(state.hOffset, promoteDemoteMaxHOffset))
	}
	projected := newHorizontalTableState(columns)
	projected.offset = state.hOffset
	projected.Apply(t, totalWidth, rows)
}

func promoteDemoteListHeader(state *promoteDemoteListState) string {
	moveMode := "copy"
	if state.removeSource {
		moveMode = "move"
	}
	return strings.Join([]string{
		promoteDemoteListStyles.Info.Render("Promote: repo → user (global)  |  Demote: user → repo (project)"),
		promoteDemoteListStyles.Option.Render(fmt.Sprintf("Mode: %s (press 'm' to toggle)", moveMode)),
		promoteDemoteRenderPlatformTabs(state),
	}, "\n")
}

func promoteDemoteListExtraBody(state *promoteDemoteListState, m *ListModel[model.Skill]) string {
	_ = state
	selected := promoteDemoteSelectedSkill(m, state)
	if selected.Name == "" || selected.Description == "" {
		return ""
	}
	descWidth := max(m.width-2, 40)
	return promoteDemoteListStyles.Description.Render(formatDescription(selected.Description, descWidth))
}

func promoteDemoteListStatus(state *promoteDemoteListState, filtered, total int, filter string) string {
	selectedCount := 0
	for _, s := range state.skills {
		if state.selected[promoteDemoteSkillKey(s)] {
			selectedCount++
		}
	}
	promotableCount := len(promoteDemotePromotableSelectedSkills(state))
	demotableCount := len(promoteDemoteDemotableSelectedSkills(state))

	status := fmt.Sprintf("%d selected (%d promotable, %d demotable) of %d",
		selectedCount, promotableCount, demotableCount, filtered)
	if filter != "" || state.platformIndex >= 0 {
		status = fmt.Sprintf("%d selected (%d↑, %d↓), %d of %d shown (filtered)",
			selectedCount, promotableCount, demotableCount, filtered, total)
	}
	if state.hOffset > 0 || state.hOffset < promoteDemoteMaxHOffset {
		scrollIndicator := hScrollIndicator(state.hOffset, promoteDemoteMaxHOffset)
		if scrollIndicator != "" {
			status += "  " + scrollIndicator
		}
	}
	return status
}

func promoteDemoteShortHelp() string {
	return strings.Join([]string{
		"↑/↓ navigate",
		"←/→ scroll cols",
		"tab/S-tab platform",
		"space toggle",
		"a toggle all",
		"p promote",
		"d demote",
		"m move/copy",
		"/ filter",
		"? help",
		"q quit",
	}, " • ")
}

func promoteDemoteFullHelp() string {
	return `Navigation:
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
}

func promoteDemoteRenderPlatformTabs(state *promoteDemoteListState) string {
	var tabs []string

	if state.platformIndex == -1 {
		tabs = append(tabs, promoteDemoteListStyles.PlatformActive.Render("[All]"))
	} else {
		tabs = append(tabs, promoteDemoteListStyles.PlatformTab.Render(" All "))
	}

	for i, platform := range state.platformOptions {
		if i == state.platformIndex {
			tabs = append(tabs, promoteDemoteListStyles.PlatformActive.Render(fmt.Sprintf("[%s]", platform)))
		} else {
			tabs = append(tabs, promoteDemoteListStyles.PlatformTab.Render(fmt.Sprintf(" %s ", platform)))
		}
	}

	return strings.Join(tabs, "")
}

func promoteDemoteSelectedSkill(m *ListModel[model.Skill], state *promoteDemoteListState) model.Skill {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		skill := m.filtered[cursor]
		if state.selected[promoteDemoteSkillKey(skill)] {
			return skill
		}
		return skill
	}
	return model.Skill{}
}

func promoteDemotePromotableSelectedSkills(state *promoteDemoteListState) []model.Skill {
	var promotable []model.Skill
	for _, s := range state.skills {
		if state.selected[promoteDemoteSkillKey(s)] && s.Scope == model.ScopeRepo {
			promotable = append(promotable, s)
		}
	}
	return promotable
}

func promoteDemoteDemotableSelectedSkills(state *promoteDemoteListState) []model.Skill {
	var demotable []model.Skill
	for _, s := range state.skills {
		if state.selected[promoteDemoteSkillKey(s)] && s.Scope == model.ScopeUser {
			demotable = append(demotable, s)
		}
	}
	return demotable
}

func (m *PromoteDemoteListModel) syncFromState() {
	m.skills = m.state.skills
	m.filtered = m.state.filtered
	m.selected = m.state.selected
	m.platformOptions = m.state.platformOptions
	m.platformIndex = m.state.platformIndex
	m.removeSource = m.state.removeSource
	m.columnWidths = m.state.columnWidths
	m.hOffset = m.state.hOffset
}

func (m *PromoteDemoteListModel) syncStateFromBase() {
	m.state.skills = m.ListModel.allItems
	m.state.filtered = m.ListModel.filtered
	m.state.selected = m.selected
	m.state.platformOptions = m.platformOptions
	m.state.platformIndex = m.platformIndex
	m.state.removeSource = m.removeSource
	m.state.columnWidths = m.columnWidths
	m.state.hOffset = m.hOffset
	m.syncFromState()
}

func (m *PromoteDemoteListModel) updateColumns(totalWidth int) {
	promoteDemoteUpdateColumns(m.state, &m.table, m.filtered, totalWidth)
	m.columnWidths = m.state.columnWidths
	m.hOffset = m.state.hOffset
}

func (m *PromoteDemoteListModel) currentModel() *ListModel[model.Skill] {
	return &m.ListModel
}

// NewPromoteDemoteListModel creates a new promote/demote list model.
// Only promotable/demotable skills (repo and user scope) are included.
func NewPromoteDemoteListModel(skills []model.Skill) PromoteDemoteListModel {
	state := newPromoteDemoteListState(skills)
	columns, widths := promoteDemoteListColumns(0, state.skills, 0)
	state.columnWidths = widths
	state.hOffset = 0

	m := PromoteDemoteListModel{
		state:           state,
		keys:            defaultPromoteDemoteListKeyMap(),
		selected:        state.selected,
		platformIndex:   state.platformIndex,
		platformOptions: state.platformOptions,
		removeSource:    state.removeSource,
		columnWidths:    state.columnWidths,
		hOffset:         state.hOffset,
		hScroll:         newHorizontalTableState(columns),
		skills:          state.skills,
		filtered:        state.filtered,
	}

	cfg := newPromoteDemoteConfig(state)
	m.ListModel = NewListModel(state.skills, cfg)
	m.selected = state.selected
	m.syncFromState()
	m.updateColumns(0)
	return m
}

// Init implements tea.Model.
func (m PromoteDemoteListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
//
//nolint:gocyclo // interactive table/event handling is intentionally centralized here
func (m PromoteDemoteListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirmMode {
			switch msg.String() {
			case "y", "Y":
				if m.confirmAction == PromoteDemoteActionPromote {
					m.result = PromoteDemoteListResult{
						Action:         PromoteDemoteActionPromote,
						SelectedSkills: promoteDemotePromotableSelectedSkills(m.state),
						RemoveSource:   m.removeSource,
					}
				} else if m.confirmAction == PromoteDemoteActionDemote {
					m.result = PromoteDemoteListResult{
						Action:         PromoteDemoteActionDemote,
						SelectedSkills: promoteDemoteDemotableSelectedSkills(m.state),
						RemoveSource:   m.removeSource,
					}
				}
				m.quitting = true
				m.confirmMode = false
				return m, tea.Quit
			case "n", "N", "esc":
				m.confirmMode = false
				m.confirmAction = PromoteDemoteActionNone
				m.confirmMsg = ""
				return m, nil
			default:
				return m, nil
			}
		}

		if m.filtering {
			inner, cmd := m.ListModel.Update(msg)
			m.ListModel = inner.(ListModel[model.Skill])
			m.syncStateFromBase()
			return m, cmd
		}

		switch {
		case msg.String() == "left":
			if m.hScroll.MoveLeft() {
				m.state.hOffset = m.hScroll.offset
				m.hOffset = m.hScroll.offset
				m.updateColumns(m.width)
			}
			return m, nil

		case msg.String() == "right":
			if m.hScroll.MoveRight(m.width) {
				m.state.hOffset = m.hScroll.offset
				m.hOffset = m.hScroll.offset
				m.updateColumns(m.width)
			}
			return m, nil

		case key.Matches(msg, m.keys.NextPlat):
			if len(m.platformOptions) > 0 {
				m.platformIndex++
				if m.platformIndex >= len(m.platformOptions) {
					m.platformIndex = -1
				}
				m.state.platformIndex = m.platformIndex
				m.applyFilter()
				m.syncStateFromBase()
				m.updateColumns(m.width)
			}
			return m, nil

		case key.Matches(msg, m.keys.PrevPlat):
			if len(m.platformOptions) > 0 {
				m.platformIndex--
				if m.platformIndex < -1 {
					m.platformIndex = len(m.platformOptions) - 1
				}
				m.state.platformIndex = m.platformIndex
				m.applyFilter()
				m.syncStateFromBase()
				m.updateColumns(m.width)
			}
			return m, nil

		case key.Matches(msg, m.keys.Toggle):
			if len(m.filtered) > 0 {
				skill := m.getSelectedSkill()
				if skill.Name != "" {
					key := promoteDemoteSkillKey(skill)
					m.selected[key] = !m.selected[key]
					m.state.selected = m.selected
					m.updateColumns(m.width)
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.ToggleAll):
			selectedCount := 0
			for _, s := range m.filtered {
				if m.selected[promoteDemoteSkillKey(s)] {
					selectedCount++
				}
			}
			selectAll := selectedCount < len(m.filtered)/2+1
			for _, s := range m.filtered {
				m.selected[promoteDemoteSkillKey(s)] = selectAll
			}
			m.state.selected = m.selected
			m.updateColumns(m.width)
			return m, nil

		case key.Matches(msg, m.keys.ToggleMove):
			m.removeSource = !m.removeSource
			m.state.removeSource = m.removeSource
			return m, nil

		case key.Matches(msg, m.keys.Promote):
			promotableSkills := promoteDemotePromotableSelectedSkills(m.state)
			if len(promotableSkills) > 0 {
				m.confirmMode = true
				m.confirmAction = PromoteDemoteActionPromote
				m.confirmMsg = fmt.Sprintf("⚠️  PROMOTE %d skill(s) from repo to user scope (%s)? (y/n)",
					len(promotableSkills), moveModeLabel(m.removeSource))
			}
			return m, nil

		case key.Matches(msg, m.keys.Demote):
			demotableSkills := promoteDemoteDemotableSelectedSkills(m.state)
			if len(demotableSkills) > 0 {
				m.confirmMode = true
				m.confirmAction = PromoteDemoteActionDemote
				m.confirmMsg = fmt.Sprintf("⚠️  DEMOTE %d skill(s) from user to repo scope (%s)? (y/n)",
					len(demotableSkills), moveModeLabel(m.removeSource))
			}
			return m, nil

		case key.Matches(msg, m.keys.ScrollLeft):
			if m.hScroll.MoveLeft() {
				m.state.hOffset = m.hScroll.offset
				m.hOffset = m.hScroll.offset
				m.updateColumns(m.width)
			}
			return m, nil

		case key.Matches(msg, m.keys.ScrollRight):
			if m.hScroll.MoveRight(m.width) {
				m.state.hOffset = m.hScroll.offset
				m.hOffset = m.hScroll.offset
				m.updateColumns(m.width)
			}
			return m, nil
		}
	}

	inner, cmd := m.ListModel.Update(msg)
	m.ListModel = inner.(ListModel[model.Skill])
	m.syncStateFromBase()
	m.updateColumns(m.width)
	return m, cmd
}

func moveModeLabel(removeSource bool) string {
	if removeSource {
		return "move (source will be removed)"
	}
	return "copy"
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

func (m *PromoteDemoteListModel) applyFilter() {
	m.state.platformIndex = m.platformIndex
	m.state.removeSource = m.removeSource
	m.state.hOffset = m.hOffset
	m.state.selected = m.selected
	m.ListModel.applyFilter()
	m.syncStateFromBase()
}

func (m PromoteDemoteListModel) skillsToRows(skills []model.Skill) []table.Row {
	return promoteDemoteListRows(m.state, skills)
}

func (m PromoteDemoteListModel) renderShortHelp() string {
	return promoteDemoteShortHelp()
}

func (m PromoteDemoteListModel) renderFullHelp() string {
	return promoteDemoteFullHelp()
}

// getPromotableSelectedSkills returns selected skills that can be promoted (repo -> user).
func (m PromoteDemoteListModel) getPromotableSelectedSkills() []model.Skill {
	return promoteDemotePromotableSelectedSkills(m.state)
}

// getDemotableSelectedSkills returns selected skills that can be demoted (user -> repo).
func (m PromoteDemoteListModel) getDemotableSelectedSkills() []model.Skill {
	return promoteDemoteDemotableSelectedSkills(m.state)
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
