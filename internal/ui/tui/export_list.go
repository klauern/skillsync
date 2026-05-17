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

	"github.com/klauern/skillsync/internal/export"
	"github.com/klauern/skillsync/internal/model"
)

// ExportAction represents the action to perform after export configuration.
type ExportAction int

const (
	// ExportActionNone means no action was taken (user quit).
	ExportActionNone ExportAction = iota
	// ExportActionExport means the user wants to export selected skills.
	ExportActionExport
)

// ExportListResult contains the result of the export list TUI interaction.
type ExportListResult struct {
	Action          ExportAction
	SelectedSkills  []model.Skill
	Format          export.Format
	IncludeMetadata bool
	Pretty          bool
}

type exportListColumnWidths struct {
	name     int
	platform int
	scope    int
	desc     int
}

func defaultExportListColumnWidths() exportListColumnWidths {
	return exportListColumnWidths{
		name:     25,
		platform: 12,
		scope:    10,
		desc:     60,
	}
}

var exportListStyles = struct {
	Format         lipgloss.Style
	Option         lipgloss.Style
	OptionVal      lipgloss.Style
	PlatformTab    lipgloss.Style
	PlatformActive lipgloss.Style
	Description    lipgloss.Style
}{
	Format:         lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true),
	Option:         lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	OptionVal:      lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
	PlatformTab:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	PlatformActive: lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(true).Padding(0, 1),
	Description:    lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1),
}

type exportListState struct {
	selected        map[string]bool
	platformOptions []model.Platform
	platformIndex   int
	format          export.Format
	includeMetadata bool
	pretty          bool
	columnWidths    exportListColumnWidths
}

// ExportListModel is the BubbleTea model for interactive export skill selection.
type ExportListModel struct {
	ListModel[model.Skill]

	skills []model.Skill

	// Compatibility mirrors for the existing test surface.
	filtered        []model.Skill
	selected        map[string]bool
	platformOptions []model.Platform
	platformIndex   int
	format          export.Format
	includeMetadata bool
	pretty          bool
	columnWidths    exportListColumnWidths

	state *exportListState
}

// skillKey creates a unique key for a skill (name + platform combination).
func skillKey(s model.Skill) string {
	return fmt.Sprintf("%s:%s", s.Platform, s.Name)
}

// Update wraps the base Update and preserves the ExportListModel type.
func (m ExportListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	inner, cmd := m.ListModel.Update(msg)
	m.ListModel = inner.(ListModel[model.Skill])
	m.syncCompat()
	return m, cmd
}

// Result returns the result of the user interaction.
func (m ExportListModel) Result() ExportListResult {
	if r, ok := m.result.(ExportListResult); ok {
		return r
	}
	return ExportListResult{}
}

// NewExportListModel creates a new export list model.
func NewExportListModel(skills []model.Skill) ExportListModel {
	sort.Slice(skills, func(i, j int) bool {
		return strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
	})

	platformSet := make(map[model.Platform]bool)
	selected := make(map[string]bool, len(skills))
	for _, skill := range skills {
		platformSet[skill.Platform] = true
		selected[skillKey(skill)] = true
	}

	var platformOptions []model.Platform
	for _, platform := range model.AllPlatforms() {
		if platformSet[platform] {
			platformOptions = append(platformOptions, platform)
		}
	}

	state := &exportListState{
		selected:        selected,
		platformOptions: platformOptions,
		platformIndex:   -1,
		format:          export.FormatJSON,
		includeMetadata: true,
		pretty:          true,
		columnWidths:    defaultExportListColumnWidths(),
	}

	toggleKey := key.NewBinding(key.WithKeys(" ", "tab"), key.WithHelp("space/tab", "toggle"))
	toggleAllKey := key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "toggle all"))

	cfg := ListConfig[model.Skill]{
		Title: "📤 Export Skills",
		Columns: []table.Column{
			{Title: " ", Width: 3},
			{Title: "Name", Width: state.columnWidths.name},
			{Title: "Platform", Width: state.columnWidths.platform},
			{Title: "Scope", Width: state.columnWidths.scope},
			{Title: "Description", Width: state.columnWidths.desc},
		},
		ToRows: func(items []model.Skill) []table.Row {
			return exportSkillsToRows(items, state)
		},
		Matches: func(skill model.Skill, lowerFilter string) bool {
			if state.platformIndex >= 0 && skill.Platform != state.platformOptions[state.platformIndex] {
				return false
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
		StatusText: func(filtered, total int, filter string) string {
			selectedCount := exportSelectedCount(skills, state.selected)
			if filter != "" || state.platformIndex >= 0 {
				return fmt.Sprintf("%d selected, %d of %d shown (filtered)", selectedCount, filtered, total)
			}
			return fmt.Sprintf("%d skill(s) selected of %d", selectedCount, filtered)
		},
		Header: func(_ *ListModel[model.Skill]) string {
			return exportHeader(state)
		},
		ExtraBody: func(m *ListModel[model.Skill]) string {
			cursor := m.table.Cursor()
			if cursor < 0 || cursor >= len(m.filtered) {
				return ""
			}
			selected := m.filtered[cursor]
			if selected.Name == "" || selected.Description == "" {
				return ""
			}
			descWidth := max(m.width-2, 40)
			return exportListStyles.Description.Render(formatDescription(selected.Description, descWidth))
		},
		ExtraKeys: func(m *ListModel[model.Skill], msg tea.KeyMsg) bool {
			switch {
			case msg.String() == "l":
				if len(state.platformOptions) > 0 {
					state.platformIndex++
					if state.platformIndex >= len(state.platformOptions) {
						state.platformIndex = -1
					}
					exportRefresh(m)
				}
				return true
			case msg.String() == "h":
				if len(state.platformOptions) > 0 {
					state.platformIndex--
					if state.platformIndex < -1 {
						state.platformIndex = len(state.platformOptions) - 1
					}
					exportRefresh(m)
				}
				return true
			case key.Matches(msg, toggleKey):
				if skill := exportSelectedSkill(m); skill.Name != "" {
					key := skillKey(skill)
					state.selected[key] = !state.selected[key]
					exportRefresh(m)
				}
				return true
			case key.Matches(msg, toggleAllKey):
				if len(m.filtered) > 0 {
					selectedCount := 0
					for _, skill := range m.filtered {
						if state.selected[skillKey(skill)] {
							selectedCount++
						}
					}
					selectAll := selectedCount < len(m.filtered)/2+1
					for _, skill := range m.filtered {
						state.selected[skillKey(skill)] = selectAll
					}
					exportRefresh(m)
				}
				return true
			case msg.String() == "f":
				switch state.format {
				case export.FormatJSON:
					state.format = export.FormatYAML
				case export.FormatYAML:
					state.format = export.FormatMarkdown
				default:
					state.format = export.FormatJSON
				}
				return true
			case msg.String() == "m":
				state.includeMetadata = !state.includeMetadata
				return true
			case msg.String() == "y":
				selectedSkills := exportSelectedSkills(skills, state.selected)
				if len(selectedSkills) == 0 {
					return true
				}
				m.result = ExportListResult{
					Action:          ExportActionExport,
					SelectedSkills:  selectedSkills,
					Format:          state.format,
					IncludeMetadata: state.includeMetadata,
					Pretty:          state.pretty,
				}
				m.confirmMode = true
				m.confirmMsg = fmt.Sprintf("Export %d skill(s) as %s? (y/n)", len(selectedSkills), strings.ToUpper(string(state.format)))
				return true
			default:
				return false
			}
		},
		ShortHelp: func() string {
			return strings.Join([]string{
				"↑/↓ navigate",
				"h/l platform",
				"space toggle",
				"a toggle all",
				"f format",
				"m metadata",
				"y export",
				"/ filter",
				"? help",
				"q quit",
			}, " • ")
		},
		FullHelp: func() string {
			return `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  g/Home   Go to top
  G/End    Go to bottom

Platform Filtering:
  h        Previous platform
  l        Next platform

Selection:
  Space/Tab  Toggle current skill
  a          Toggle all skills

Export Options:
  f        Cycle format (JSON → YAML → Markdown)
  m        Toggle metadata inclusion

Actions:
  y        Confirm and export selected skills

Filter:
  /        Start filtering (by name, platform, scope, or description)
  Esc      Clear filter
  Enter    Finish filtering

General:
  ?        Toggle full help
  q        Quit without exporting`
		},
	}

	mdl := ExportListModel{
		ListModel: NewListModel(skills, cfg),
		skills:    skills,
		state:     state,
	}
	mdl.syncCompat()
	return mdl
}

func exportSkillsToRows(skills []model.Skill, state *exportListState) []table.Row {
	widths := state.columnWidths
	if widths.desc == 0 {
		widths = defaultExportListColumnWidths()
	}
	rows := make([]table.Row, len(skills))
	for i, skill := range skills {
		checkbox := "[ ]"
		if state.selected[skillKey(skill)] {
			checkbox = "[✓]"
		}
		rows[i] = table.Row{
			checkbox,
			truncateTableValue(skill.Name, widths.name),
			truncateTableValue(string(skill.Platform), widths.platform),
			truncateTableValue(skill.DisplayScope(), widths.scope),
			truncateTableValue(skill.Description, 40),
		}
	}
	return rows
}

func exportHeader(state *exportListState) string {
	formatLabel := exportListStyles.Option.Render("Format: ")
	formatVal := exportListStyles.Format.Render(strings.ToUpper(string(state.format)))

	metadataLabel := exportListStyles.Option.Render("  Metadata: ")
	metadataVal := "No"
	if state.includeMetadata {
		metadataVal = "Yes"
	}
	optionsLine := formatLabel + formatVal + metadataLabel + exportListStyles.OptionVal.Render(metadataVal)

	return optionsLine + "\n\n" + renderExportPlatformTabs(state)
}

func renderExportPlatformTabs(state *exportListState) string {
	var tabs []string

	if state.platformIndex == -1 {
		tabs = append(tabs, exportListStyles.PlatformActive.Render("[All]"))
	} else {
		tabs = append(tabs, exportListStyles.PlatformTab.Render(" All "))
	}

	for i, platform := range state.platformOptions {
		if i == state.platformIndex {
			tabs = append(tabs, exportListStyles.PlatformActive.Render(fmt.Sprintf("[%s]", platform)))
		} else {
			tabs = append(tabs, exportListStyles.PlatformTab.Render(fmt.Sprintf(" %s ", platform)))
		}
	}

	return strings.Join(tabs, "")
}

func exportSelectedCount(skills []model.Skill, selected map[string]bool) int {
	count := 0
	for _, skill := range skills {
		if selected[skillKey(skill)] {
			count++
		}
	}
	return count
}

func exportSelectedSkills(skills []model.Skill, selected map[string]bool) []model.Skill {
	var chosen []model.Skill
	for _, skill := range skills {
		if selected[skillKey(skill)] {
			chosen = append(chosen, skill)
		}
	}
	return chosen
}

func exportSelectedSkill(m *ListModel[model.Skill]) model.Skill {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return model.Skill{}
}

func exportRefresh(m *ListModel[model.Skill]) {
	m.applyFilter()
	m.table.SetRows(m.cfg.ToRows(m.filtered))
}

func (m *ExportListModel) syncCompat() {
	m.filtered = m.ListModel.filtered
	if m.state != nil {
		m.selected = m.state.selected
		m.platformOptions = m.state.platformOptions
		m.platformIndex = m.state.platformIndex
		m.format = m.state.format
		m.includeMetadata = m.state.includeMetadata
		m.pretty = m.state.pretty
		m.columnWidths = m.state.columnWidths
	}
}

func (m *ExportListModel) syncStateFromCompat() {
	if m.state == nil {
		return
	}
	m.state.selected = m.selected
	m.state.platformOptions = m.platformOptions
	m.state.platformIndex = m.platformIndex
	m.state.format = m.format
	m.state.includeMetadata = m.includeMetadata
	m.state.pretty = m.pretty
	if m.columnWidths.desc != 0 {
		m.state.columnWidths = m.columnWidths
	}
}

// applyFilter is kept for tests and compatibility with the pre-refactor API.
func (m *ExportListModel) applyFilter() {
	m.syncStateFromCompat()
	m.ListModel.applyFilter()
	m.syncCompat()
}

func (m ExportListModel) skillsToRows(skills []model.Skill) []table.Row {
	if m.state == nil {
		return nil
	}
	return exportSkillsToRows(skills, m.state)
}

func (m ExportListModel) getSelectedSkills() []model.Skill {
	if m.state == nil {
		return nil
	}
	return exportSelectedSkills(m.skills, m.state.selected)
}

func (m ExportListModel) renderShortHelp() string {
	return listStyles.Help.Render(m.cfg.ShortHelp())
}

func (m ExportListModel) renderFullHelp() string {
	return listStyles.Help.Render(m.cfg.FullHelp())
}

// RunExportList runs the interactive export list and returns the result.
func RunExportList(skills []model.Skill) (ExportListResult, error) {
	if len(skills) == 0 {
		return ExportListResult{}, nil
	}

	mdl := NewExportListModel(skills)
	finalModel, err := tea.NewProgram(mdl, tea.WithAltScreen()).Run()
	if err != nil {
		return ExportListResult{}, err
	}

	if m, ok := finalModel.(ExportListModel); ok {
		return m.Result(), nil
	}

	return ExportListResult{}, nil
}
