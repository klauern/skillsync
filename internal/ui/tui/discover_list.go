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

type discoverListColumnWidths struct {
	name     int
	platform int
	scope    int
	desc     int
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
	discoverListDetailHeight  = discoverListDetailLines + 1 + 2 // title + content + border
)

// Discover-screen styles beyond the shared listStyles.
var discoverExtraStyles = struct {
	DetailBox      lipgloss.Style
	DetailTitle    lipgloss.Style
	PlatformTab    lipgloss.Style
	PlatformActive lipgloss.Style
	Description    lipgloss.Style
}{
	DetailBox:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
	DetailTitle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
	PlatformTab:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	PlatformActive: lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(true).Padding(0, 1),
	Description:    lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1),
}

type discoverListState struct {
	platformOptions []model.Platform
	platformIndex   int
	columnWidths    discoverListColumnWidths
	hScroll         horizontalTableState
	lastWidth       int
	phase           discoverListPhase
	detailSkill     model.Skill
	vp              viewport.Model
	vpReady         bool
}

func (s *discoverListState) refresh(m *ListModel[model.Skill]) {
	columns, widths := discoverListColumns(m.width, m.allItems)
	s.columnWidths = widths
	s.lastWidth = m.width
	s.hScroll.SetColumns(columns)
	s.hScroll.Apply(&m.table, m.width, m.cfg.ToRows(m.filtered))
}

func (s *discoverListState) renderPlatformTabs() string {
	var tabs []string
	if s.platformIndex == -1 {
		tabs = append(tabs, discoverExtraStyles.PlatformActive.Render("[All]"))
	} else {
		tabs = append(tabs, discoverExtraStyles.PlatformTab.Render(" All "))
	}
	for i, platform := range s.platformOptions {
		if i == s.platformIndex {
			tabs = append(tabs, discoverExtraStyles.PlatformActive.Render(fmt.Sprintf("[%s]", platform)))
		} else {
			tabs = append(tabs, discoverExtraStyles.PlatformTab.Render(fmt.Sprintf(" %s ", platform)))
		}
	}
	return strings.Join(tabs, "")
}

func (s *discoverListState) detailPanel(m *ListModel[model.Skill]) string {
	width := m.width
	if width <= 0 {
		width = discoverListNameWidth + discoverListPlatformWidth + discoverListScopeWidth + discoverListDescWidth +
			(discoverListColumnPadding * discoverListColumnCount)
	}
	contentWidth := max(width-4, 10)
	skill := discoverSelectedSkill(m)
	description := strings.TrimSpace(skill.Description)
	if description == "" {
		description = "No description available."
	}
	lines := wrapText(description, contentWidth, discoverListDetailLines)
	lines = padLines(lines, discoverListDetailLines)
	header := discoverExtraStyles.DetailTitle.Render("Description (selected)")
	content := append([]string{header}, lines...)
	return discoverExtraStyles.DetailBox.Width(width).Render(strings.Join(content, "\n"))
}

func (s *discoverListState) ensureViewport(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	headerHeight := 4
	footerHeight := 4
	vpHeight := max(height-headerHeight-footerHeight, 5)
	if !s.vpReady {
		s.vp = viewport.New(width-2, vpHeight)
		s.vpReady = true
		return
	}
	s.vp.Width = width - 2
	s.vp.Height = vpHeight
}

// DiscoverListModel is the BubbleTea model for interactive skill discovery.
type DiscoverListModel struct {
	ListModel[model.Skill]
	state       *discoverListState
	// Deprecated compatibility fields kept for existing tests and callers.
	skills      []model.Skill
	detailSkill model.Skill
}

// Update implements tea.Model.
func (m DiscoverListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.state.phase == discoverListPhaseDetail {
		return m.updateDetail(msg)
	}
	inner, cmd := m.ListModel.Update(msg)
	m.ListModel = inner.(ListModel[model.Skill])
	// Re-apply hScroll projection after every base update (base.applyFilter sets rows
	// without going through hScroll.Apply, which would leave row/column counts mismatched).
	if m.width > 0 {
		m.state.refresh(&m.ListModel)
	}
	return m, cmd
}

// View implements tea.Model.
func (m DiscoverListModel) View() string {
	if m.state.phase == discoverListPhaseDetail {
		return m.viewDetail()
	}
	return m.ListModel.View()
}

// Result returns the result of the user interaction.
func (m DiscoverListModel) Result() DiscoverListResult {
	if r, ok := m.result.(DiscoverListResult); ok {
		return r
	}
	return DiscoverListResult{}
}

func (m DiscoverListModel) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	backKey := key.NewBinding(key.WithKeys("b", "esc"))

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.state.ensureViewport(m.width, m.height)
		if m.state.vpReady {
			m.state.vp.SetContent(m.buildDetailContent(m.state.vp.Width))
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, backKey):
			m.state.phase = discoverListPhaseList
			return m, nil
		}
	}

	m.state.vp, cmd = m.state.vp.Update(msg)
	return m, cmd
}

func (m DiscoverListModel) viewDetail() string {
	m.state.ensureViewport(m.width, m.height)
	if !m.state.vpReady {
		return "Loading..."
	}
	m.state.vp.SetContent(m.buildDetailContent(m.state.vp.Width))

	var b strings.Builder
	detailSkill := m.detailSkill
	if detailSkill.Name == "" {
		detailSkill = m.state.detailSkill
	}
	b.WriteString(listStyles.Title.Render(fmt.Sprintf("🔍 Skill Details: %s", detailSkill.Name)))
	b.WriteString("\n\n")
	b.WriteString(m.state.vp.View())
	b.WriteString("\n")
	b.WriteString(listStyles.Status.Render(fmt.Sprintf("Scroll: %d%% • Press b or Esc to go back",
		int(m.state.vp.ScrollPercent()*100))))
	b.WriteString("\n")
	if m.showHelp {
		b.WriteString("\n")
		b.WriteString(listStyles.Help.Render(discoverDetailFullHelp))
	} else {
		b.WriteString(listStyles.Help.Render("↑/↓ scroll • b back • ? help • q quit"))
	}
	return b.String()
}

const discoverDetailFullHelp = `Navigation:
  ↑/k      Scroll up
  ↓/j      Scroll down
  g/Home   Top
  G/End    Bottom

Actions:
  b/Esc    Back to list

General:
  ?        Toggle full help
  q        Quit`

func (m DiscoverListModel) buildDetailContent(width int) string {
	var b strings.Builder
	skill := m.detailSkill
	if skill.Name == "" {
		skill = m.state.detailSkill
	}
	if skill.Name == "" {
		return "No skill selected."
	}
	indent := "  "
	b.WriteString(discoverExtraStyles.DetailTitle.Render("Skill"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "%sName: %s\n", indent, skill.Name)
	fmt.Fprintf(&b, "%sPlatform: %s\n", indent, skill.Platform)
	fmt.Fprintf(&b, "%sScope: %s\n", indent, skill.DisplayScope())
	if skill.Path != "" {
		fmt.Fprintf(&b, "%sPath: %s\n", indent, skill.Path)
	}
	if len(skill.Tools) > 0 {
		fmt.Fprintf(&b, "%sTools: %s\n", indent, strings.Join(skill.Tools, ", "))
	}
	b.WriteString("\n")
	b.WriteString(discoverExtraStyles.DetailTitle.Render("Description"))
	b.WriteString("\n")
	description := strings.TrimSpace(skill.Description)
	if description == "" {
		description = "No description available."
	}
	b.WriteString(lipgloss.NewStyle().Width(max(width, 10)).Render(description))
	b.WriteString("\n")
	return b.String()
}

func discoverSelectedSkill(m *ListModel[model.Skill]) model.Skill {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return model.Skill{}
}

// skillsToRows is kept for test compatibility.
func (m DiscoverListModel) skillsToRows(skills []model.Skill) []table.Row {
	return m.cfg.ToRows(skills)
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

// NewDiscoverListModel creates a new discover list model.
func NewDiscoverListModel(skills []model.Skill) DiscoverListModel {
	sort.Slice(skills, func(i, j int) bool {
		return strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
	})

	platformSet := make(map[model.Platform]bool)
	for _, s := range skills {
		platformSet[s.Platform] = true
	}
	var platformOptions []model.Platform
	for _, p := range model.AllPlatforms() {
		if platformSet[p] {
			platformOptions = append(platformOptions, p)
		}
	}

	columns, columnWidths := discoverListColumns(0, skills)
	state := &discoverListState{
		platformOptions: platformOptions,
		platformIndex:   -1,
		columnWidths:    columnWidths,
		hScroll:         newHorizontalTableState(columns),
	}

	detailKey := key.NewBinding(key.WithKeys("enter", "v"), key.WithHelp("enter/v", "details"))
	openKey := key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open"))
	copyKey := key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy path"))
	nextPlatKey := key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab/S-tab", "platform"))
	prevPlatKey := key.NewBinding(key.WithKeys("shift+tab"))

	cfg := ListConfig[model.Skill]{
		Title:   "🔍 Skillsync Skills",
		Columns: columns,
		ToRows: func(items []model.Skill) []table.Row {
			rows := make([]table.Row, len(items))
			for i, s := range items {
				rows[i] = table.Row{
					truncateTableValue(s.Name, state.columnWidths.name),
					truncateTableValue(string(s.Platform), state.columnWidths.platform),
					truncateTableValue(s.DisplayScope(), state.columnWidths.scope),
					truncateTableValue(s.Description, state.columnWidths.desc),
				}
			}
			return rows
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
		ReservedLines: 17,
		Actions: []ActionBinding[model.Skill]{
			{
				Binding: openKey,
				Apply: func(skill model.Skill) any {
					return DiscoverListResult{Action: DiscoverActionView, Skill: skill}
				},
			},
			{
				Binding: copyKey,
				Apply: func(skill model.Skill) any {
					return DiscoverListResult{Action: DiscoverActionCopy, Skill: skill}
				},
			},
		},
		ShortHelp: func() string {
			return strings.Join([]string{
				"↑/↓ navigate",
				"←/→ columns",
				"tab/S-tab platform",
				"enter details",
				"o open",
				"c copy path",
				"/ filter",
				"? help",
				"q quit",
			}, " • ")
		},
		FullHelp: func() string {
			return `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  ←/h      Show previous columns
  →/l      Show next columns
  g/Home   Go to top
  G/End    Go to bottom

Platform Filtering:
  Tab/l       Next platform
  Shift-Tab/h Previous platform

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
		},
		StatusText: func(filtered, total int, filter string) string {
			status := fmt.Sprintf("%d skill(s)", filtered)
			if filter != "" {
				status = fmt.Sprintf("%d of %d skill(s) (filtered)", filtered, total)
			}
			if scrollStatus := state.hScroll.Summary(state.lastWidth); scrollStatus != "" {
				status += " • " + scrollStatus
			}
			return status
		},
		Header: func() string {
			return state.renderPlatformTabs()
		},
		ExtraBody: func(m *ListModel[model.Skill]) string {
			var b strings.Builder
			b.WriteString(state.detailPanel(m))
			b.WriteString("\n")
			selected := discoverSelectedSkill(m)
			if selected.Name != "" && selected.Description != "" {
				descWidth := max(m.width-2, 40)
				b.WriteString(discoverExtraStyles.Description.Render(formatDescription(selected.Description, descWidth)))
				b.WriteString("\n")
			}
			return b.String()
		},
		ExtraKeys: func(m *ListModel[model.Skill], msg tea.KeyMsg) bool {
			switch {
			case msg.String() == "left" || msg.String() == "h":
				if state.hScroll.MoveLeft() {
					state.refresh(m)
				}
				return true
			case msg.String() == "right" || msg.String() == "l":
				if state.hScroll.MoveRight(m.width) {
					state.refresh(m)
				}
				return true
			case key.Matches(msg, nextPlatKey):
				if len(state.platformOptions) > 0 {
					state.platformIndex++
					if state.platformIndex >= len(state.platformOptions) {
						state.platformIndex = -1
					}
					m.applyFilter()
					state.refresh(m)
				}
				return true
			case key.Matches(msg, prevPlatKey):
				if len(state.platformOptions) > 0 {
					state.platformIndex--
					if state.platformIndex < -1 {
						state.platformIndex = len(state.platformOptions) - 1
					}
					m.applyFilter()
					state.refresh(m)
				}
				return true
			case key.Matches(msg, detailKey):
				if len(m.filtered) > 0 {
					state.detailSkill = discoverSelectedSkill(m)
					state.phase = discoverListPhaseDetail
					state.vpReady = false
					state.ensureViewport(m.width, m.height)
				}
				return true
			}
			return false
		},
		OnWindowSize: func(m *ListModel[model.Skill], width, _ int) {
			state.refresh(m)
		},
	}

	mdl := DiscoverListModel{
		ListModel: NewListModel(skills, cfg),
		state:     state,
		skills:    skills,
	}
	state.refresh(&mdl.ListModel)
	return mdl
}

// RunDiscoverList runs the interactive discover list and returns the result.
func RunDiscoverList(skills []model.Skill) (DiscoverListResult, error) {
	if len(skills) == 0 {
		return DiscoverListResult{}, nil
	}

	mdl := NewDiscoverListModel(skills)
	finalModel, err := tea.NewProgram(mdl, tea.WithAltScreen()).Run()
	if err != nil {
		return DiscoverListResult{}, err
	}

	if m, ok := finalModel.(DiscoverListModel); ok {
		return m.Result(), nil
	}
	return DiscoverListResult{}, nil
}
