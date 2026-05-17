// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

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
	Selections     map[string]bool // all current selections for state preservation
}

// syncListColumnWidths tracks the responsive widths for the sync table.
type syncListColumnWidths struct {
	name  int
	scope int
	desc  int
}

func defaultSyncListColumnWidths() syncListColumnWidths {
	return syncListColumnWidths{
		name:  25,
		scope: 12,
		desc:  60,
	}
}

// Styles for the sync list TUI.
var syncListStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Confirm     lipgloss.Style
	Status      lipgloss.Style
	Description lipgloss.Style
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
	Description: lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1),
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
	syncListDetailLines   = 3
	syncListDetailGap     = 1
	syncListDetailHeight  = syncListDetailLines + 1 + 2 // title + content + border
	// syncListMaxHOffset is the maximum horizontal scroll offset (2 scrollable cols - 1).
	syncListMaxHOffset = 1
)

// syncListColumns returns visible table columns for the given terminal width and horizontal
// scroll offset. Scrollable columns are: scope(0), description(1).
func syncListColumns(totalWidth int, skills []model.Skill, hOffset int) ([]table.Column, syncListColumnWidths) {
	hOffset = max(0, min(hOffset, syncListMaxHOffset))

	widths := syncListColumnWidths{
		name:  syncListNameWidth,
		scope: syncListScopeWidth,
		desc:  syncListDescWidth,
	}

	showScope := hOffset == 0

	visibleColCount := 3 // checkbox + name + description always visible
	if showScope {
		visibleColCount++
	}

	if totalWidth > 0 {
		baseTotal := syncListCheckboxWidth + widths.name + widths.desc +
			(syncListColumnPadding * visibleColCount)
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
		{Title: " ", Width: syncListCheckboxWidth},
		{Title: "Name", Width: widths.name},
	}
	if showScope {
		columns = append(columns, table.Column{Title: "Scope", Width: widths.scope})
	}
	columns = append(columns, table.Column{Title: "Description", Width: widths.desc})

	return columns, widths
}

type syncListState struct {
	selected       map[string]bool
	sourcePlatform model.Platform
	targetPlatform model.Platform
	columnWidths   syncListColumnWidths
	hScroll        horizontalTableState
}

func newSyncListState(skills []model.Skill, source, target model.Platform, initialSelections map[string]bool) *syncListState {
	selected := make(map[string]bool, len(skills))
	for _, s := range skills {
		if initialSelections != nil {
			selected[s.Name] = initialSelections[s.Name]
		} else {
			selected[s.Name] = true
		}
	}

	return &syncListState{
		selected:       selected,
		sourcePlatform: source,
		targetPlatform: target,
		columnWidths:   defaultSyncListColumnWidths(),
		hScroll:        newHorizontalTableState(nil),
	}
}

func (s *syncListState) selectedCount() int {
	count := 0
	for _, selected := range s.selected {
		if selected {
			count++
		}
	}
	return count
}

func (s *syncListState) copySelections() map[string]bool {
	result := make(map[string]bool, len(s.selected))
	maps.Copy(result, s.selected)
	return result
}

func (s *syncListState) selectedSkills(skills []model.Skill) []model.Skill {
	selected := make([]model.Skill, 0, len(skills))
	for _, skill := range skills {
		if s.selected[skill.Name] {
			selected = append(selected, skill)
		}
	}
	return selected
}

func (s *syncListState) refresh(m *ListModel[model.Skill]) {
	columns, widths := syncListColumns(m.width, m.allItems, s.hScroll.offset)
	s.columnWidths = widths
	s.hScroll.SetColumns(columns)
	s.hScroll.Apply(&m.table, m.width, m.cfg.ToRows(m.filtered))
}

func (s *syncListState) detailPanel(m *ListModel[model.Skill]) string {
	width := m.width
	if width <= 0 {
		width = syncListCheckboxWidth + s.columnWidths.name + s.columnWidths.scope + s.columnWidths.desc +
			(syncListColumnPadding * 4)
	}

	contentWidth := max(width-4, 10)
	skill := selectedSyncSkill(m)
	description := strings.TrimSpace(skill.Description)
	if description == "" {
		description = "No description available."
	}

	lines := wrapText(description, contentWidth, syncListDetailLines)
	lines = padLines(lines, syncListDetailLines)

	head := syncListStyles.DetailTitle.Render("Description (selected)")
	content := append([]string{head}, lines...)
	return syncListStyles.DetailBox.Width(width).Render(strings.Join(content, "\n"))
}

// SyncListModel is the BubbleTea model for interactive sync skill selection.
type SyncListModel struct {
	ListModel[model.Skill]
	state *syncListState
}

// Update wraps the base Update and preserves the SyncListModel type.
func (m SyncListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	inner, cmd := m.ListModel.Update(msg)
	m.ListModel = inner.(ListModel[model.Skill])
	return m, cmd
}

// Result returns the result of the user interaction.
func (m SyncListModel) Result() SyncListResult {
	if r, ok := m.result.(SyncListResult); ok {
		return r
	}
	return SyncListResult{}
}

// applyFilter is kept for tests and direct state manipulation.
func (m *SyncListModel) applyFilter() {
	m.ListModel.applyFilter()
	if m.state != nil {
		m.state.refresh(&m.ListModel)
	}
}

// getSelectedSkills is kept for tests and direct state inspection.
func (m SyncListModel) getSelectedSkills() []model.Skill {
	if m.state == nil {
		return nil
	}
	return m.state.selectedSkills(m.allItems)
}

// skillsToRows is kept for tests and compatibility with the pre-refactor API.
func (m SyncListModel) skillsToRows(skills []model.Skill) []table.Row {
	if m.state == nil {
		return nil
	}
	return m.cfg.ToRows(skills)
}

// GetSelections returns a copy of the current selections map.
func (m SyncListModel) GetSelections() map[string]bool {
	if m.state == nil {
		return nil
	}
	return m.state.copySelections()
}

// NewSyncListModel creates a new sync list model.
func NewSyncListModel(skills []model.Skill, source, target model.Platform, initialSelections map[string]bool) SyncListModel {
	// Sort skills alphabetically by name (case-insensitive).
	sort.Slice(skills, func(i, j int) bool {
		return strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
	})

	state := newSyncListState(skills, source, target, initialSelections)

	previewKey := key.NewBinding(
		key.WithKeys("p", "enter"),
		key.WithHelp("p/enter", "preview diff"),
	)
	toggleKey := key.NewBinding(
		key.WithKeys(" ", "tab"),
		key.WithHelp("space/tab", "toggle"),
	)
	toggleAllKey := key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "toggle all"),
	)

	cfg := ListConfig[model.Skill]{
		Title: fmt.Sprintf("🔄 Sync Skills: %s → %s", source, target),
		Columns: func() []table.Column {
			columns, _ := syncListColumns(0, skills, 0)
			return columns
		}(),
		ToRows: func(items []model.Skill) []table.Row {
			widths := state.columnWidths
			if widths.desc == 0 {
				widths = defaultSyncListColumnWidths()
			}
			rows := make([]table.Row, len(items))
			for i, skill := range items {
				row := table.Row{
					state.checkboxFor(skill.Name),
					truncateTableValue(skill.Name, widths.name),
				}
				if state.hScroll.offset == 0 {
					row = append(row, truncateTableValue(skill.DisplayScope(), widths.scope))
				}
				row = append(row, truncateTableValue(skill.Description, widths.desc))
				rows[i] = row
			}
			return rows
		},
		Matches: func(skill model.Skill, lowerFilter string) bool {
			if lowerFilter == "" {
				return true
			}
			return strings.Contains(strings.ToLower(skill.Name), lowerFilter) ||
				strings.Contains(strings.ToLower(string(skill.Platform)), lowerFilter) ||
				strings.Contains(strings.ToLower(skill.DisplayScope()), lowerFilter) ||
				strings.Contains(strings.ToLower(skill.Description), lowerFilter)
		},
		ReservedLines: 15,
		Actions: []ActionBinding[model.Skill]{
			{
				Binding: previewKey,
				Apply: func(skill model.Skill) any {
					return SyncListResult{
						Action:         SyncActionPreview,
						PreviewSkill:   skill,
						SelectedSkills: state.selectedSkills(skills),
						Selections:     state.copySelections(),
					}
				},
			},
		},
		ShortHelp: func() string {
			return syncListStyles.Help.Render(strings.Join([]string{
				"↑/↓ navigate",
				"←/→ scroll cols",
				"space/tab toggle",
				"a toggle all",
				"p preview",
				"y sync",
				"/ filter",
				"? help",
				"q quit",
			}, " • "))
		},
		FullHelp: func() string {
			return syncListStyles.Help.Render(`Navigation:
  ↑/k      Move up
  ↓/j      Move down
  ←/h      Show previous columns
  →/l      Show next columns
  g/Home   Go to top
  G/End    Go to bottom
  ←/→      Scroll columns left/right

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
  q        Quit without syncing`)
		},
		StatusText: func(filtered, total int, filter string) string {
			selectedCount := state.selectedCount()
			status := fmt.Sprintf("%d skill(s) selected of %d", selectedCount, filtered)
			if filter != "" {
				status = fmt.Sprintf("%d selected, %d of %d shown (filtered)", selectedCount, filtered, total)
			}
			if scroll := hScrollIndicator(state.hScroll.offset, syncListMaxHOffset); scroll != "" {
				status += "  " + scroll
			}
			return status
		},
		ExtraBody: func(m *ListModel[model.Skill]) string {
			return state.detailPanel(m)
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
			case key.Matches(msg, toggleKey):
				if skill := selectedSyncSkill(m); skill.Name != "" {
					state.selected[skill.Name] = !state.selected[skill.Name]
					state.refresh(m)
				}
				return true
			case key.Matches(msg, toggleAllKey):
				if len(m.filtered) > 0 {
					selectedCount := 0
					for _, skill := range m.filtered {
						if state.selected[skill.Name] {
							selectedCount++
						}
					}
					selectAll := selectedCount < len(m.filtered)/2+1
					for _, skill := range m.filtered {
						state.selected[skill.Name] = selectAll
					}
					state.refresh(m)
				}
				return true
			case msg.String() == "y":
				selectedSkills := state.selectedSkills(skills)
				if len(selectedSkills) == 0 {
					return true
				}
				m.result = SyncListResult{
					Action:         SyncActionSync,
					SelectedSkills: selectedSkills,
					Selections:     state.copySelections(),
				}
				m.confirmMode = true
				m.confirmMsg = fmt.Sprintf("Sync %d skill(s) to %s? (y/n)", len(selectedSkills), state.targetPlatform)
				return true
			default:
				return false
			}
		},
		OnWindowSize: func(m *ListModel[model.Skill], width, _ int) {
			state.refresh(m)
		},
	}

	mdl := SyncListModel{
		ListModel: NewListModel(skills, cfg),
		state:     state,
	}
	state.refresh(&mdl.ListModel)
	return mdl
}

func (s *syncListState) checkboxFor(name string) string {
	if s.selected[name] {
		return "[✓]"
	}
	return "[ ]"
}

func selectedSyncSkill(m *ListModel[model.Skill]) model.Skill {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return model.Skill{}
}

// RunSyncList runs the interactive sync list and returns the result.
func RunSyncList(skills []model.Skill, source, target model.Platform, initialSelections map[string]bool) (SyncListResult, error) {
	if len(skills) == 0 {
		return SyncListResult{}, nil
	}

	mdl := NewSyncListModel(skills, source, target, initialSelections)
	finalModel, err := tea.NewProgram(mdl, tea.WithAltScreen()).Run()
	if err != nil {
		return SyncListResult{}, err
	}

	if m, ok := finalModel.(SyncListModel); ok {
		result := m.Result()
		if result.Action != SyncActionNone {
			result.Selections = m.GetSelections()
		}
		return result, nil
	}

	return SyncListResult{}, nil
}
