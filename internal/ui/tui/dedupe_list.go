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

type dedupeListColumnWidths struct {
	name         int
	platform     int
	scope        int
	similar      int
	nameScore    int
	contentScore int
	desc         int
}

func defaultDedupeListColumnWidths() dedupeListColumnWidths {
	return dedupeListColumnWidths{
		name:         22,
		platform:     12,
		scope:        8,
		similar:      22,
		nameScore:    6,
		contentScore: 8,
		desc:         40,
	}
}

type dedupeListState struct {
	duplicates   []*similarity.ComparisonResult
	flatSkills   []model.Skill
	selected     map[string]bool
	columnWidths dedupeListColumnWidths
	width        int
	height       int
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
	ListModel[model.Skill]
	state        *dedupeListState
	table        table.Model
	duplicates   []*similarity.ComparisonResult
	flatSkills   []model.Skill // flattened list of skills from duplicate pairs
	filtered     []model.Skill
	selected     map[string]bool // map of skill key to selected state
	keys         dedupeListKeyMap
	result       DedupeListResult
	filter       string
	filtering    bool
	showHelp     bool
	confirmMode  bool
	width        int
	height       int
	quitting     bool
	columnWidths dedupeListColumnWidths
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
	Description lipgloss.Style
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
	Description: lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1),
	Checkbox:    lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	Duplicate:   lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
	Score:       lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
}

// dedupeSkillKey creates a unique key for a skill (platform + scope + name combination).
func dedupeSkillKey(s model.Skill) string {
	return fmt.Sprintf("%s:%s:%s", s.Platform, s.Scope, s.Name)
}

// NewDedupeListModel creates a new dedupe list model from comparison results.
// Only writable skills (repo and user scopes) can be selected for deletion.
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

	// Sort by content similarity descending (highest similarity first)
	// Build a map of skill key to best content score for sorting
	bestContentScores := make(map[string]float64)
	for _, dup := range duplicates {
		key1 := dedupeSkillKey(dup.Skill1)
		key2 := dedupeSkillKey(dup.Skill2)
		if dup.ContentScore > bestContentScores[key1] {
			bestContentScores[key1] = dup.ContentScore
		}
		if dup.ContentScore > bestContentScores[key2] {
			bestContentScores[key2] = dup.ContentScore
		}
	}
	sort.Slice(deletableSkills, func(i, j int) bool {
		return bestContentScores[dedupeSkillKey(deletableSkills[i])] > bestContentScores[dedupeSkillKey(deletableSkills[j])]
	})

	columnWidths := defaultDedupeListColumnWidths()
	columns := []table.Column{
		{Title: " ", Width: 3},
		{Title: "Name", Width: columnWidths.name},
		{Title: "Platform", Width: columnWidths.platform},
		{Title: "Scope", Width: columnWidths.scope},
		{Title: "Similar To", Width: columnWidths.similar},
		{Title: "Name%", Width: columnWidths.nameScore},
		{Title: "Content%", Width: columnWidths.contentScore},
		{Title: "Description", Width: columnWidths.desc},
	}

	selected := make(map[string]bool)
	state := &dedupeListState{
		duplicates:   duplicates,
		flatSkills:   deletableSkills,
		selected:     selected,
		columnWidths: columnWidths,
	}

	m := DedupeListModel{
		duplicates:   duplicates,
		flatSkills:   deletableSkills,
		filtered:     deletableSkills,
		selected:     selected,
		keys:         defaultDedupeListKeyMap(),
		columnWidths: columnWidths,
		state:        state,
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
	m.ListModel = NewListModel(deletableSkills, ListConfig[model.Skill]{
		Title:         "🔍 Dedupe Skills - Find and Remove Duplicates",
		Columns:       columns,
		ToRows:        state.skillsToRows,
		Matches:       state.matches,
		ShortHelp:     state.shortHelp,
		FullHelp:      state.fullHelp,
		StatusText:    state.statusText,
		Header:        state.header,
		ExtraBody:     state.extraBody,
		ExtraKeys:     state.extraKeys,
		OnWindowSize:  state.onWindowSize,
		ReservedLines: 12,
	})
	m.ListModel.table = t
	m.ListModel.allItems = deletableSkills
	m.ListModel.filtered = deletableSkills
	m.ListModel.filter = m.filter
	m.ListModel.filtering = m.filtering
	m.ListModel.showHelp = m.showHelp
	m.ListModel.confirmMode = m.confirmMode
	m.ListModel.confirmMsg = ""
	m.ListModel.quitting = m.quitting
	m.ListModel.width = m.width
	m.ListModel.height = m.height
	m.ListModel.result = m.result
	m.syncStateFromCompat()
	return m
}

func (m *DedupeListModel) syncStateFromCompat() {
	if m == nil || m.state == nil {
		return
	}
	m.state.duplicates = m.duplicates
	m.state.flatSkills = m.flatSkills
	m.state.selected = m.selected
	m.state.columnWidths = m.columnWidths
	m.state.width = m.width
	m.state.height = m.height
}

func (m *DedupeListModel) syncListModelFromCompat() {
	if m == nil {
		return
	}
	m.syncStateFromCompat()
	m.ListModel.filter = m.filter
	m.ListModel.filtering = m.filtering
	m.ListModel.showHelp = m.showHelp
	m.ListModel.confirmMode = m.confirmMode
	m.ListModel.quitting = m.quitting
	m.ListModel.width = m.width
	m.ListModel.height = m.height
	m.ListModel.filtered = m.filtered
	m.ListModel.allItems = m.flatSkills
	m.ListModel.table = m.table
	m.ListModel.result = m.result
}

func (m *DedupeListModel) syncCompatFromBase() {
	if m == nil {
		return
	}
	m.table = m.ListModel.table
	m.filtered = m.ListModel.filtered
	m.filter = m.ListModel.filter
	m.filtering = m.ListModel.filtering
	m.showHelp = m.ListModel.showHelp
	m.confirmMode = m.ListModel.confirmMode
	m.quitting = m.ListModel.quitting
	m.width = m.ListModel.width
	m.height = m.ListModel.height
	if result, ok := m.ListModel.result.(DedupeListResult); ok {
		m.result = result
	}
	if m.state != nil {
		m.state.duplicates = m.duplicates
		m.state.flatSkills = m.flatSkills
		m.state.selected = m.selected
		m.state.columnWidths = m.columnWidths
		m.state.width = m.width
		m.state.height = m.height
	}
}

func (s *dedupeListState) selectedSkills() []model.Skill {
	if s == nil {
		return nil
	}
	selected := make([]model.Skill, 0, len(s.flatSkills))
	for _, skill := range s.flatSkills {
		if s.selected[dedupeSkillKey(skill)] {
			selected = append(selected, skill)
		}
	}
	return selected
}

func (s *dedupeListState) selectedSkill(m *ListModel[model.Skill]) model.Skill {
	if s == nil || m == nil {
		return model.Skill{}
	}
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(m.filtered) {
		return model.Skill{}
	}
	return m.filtered[cursor]
}

func (s *dedupeListState) matches(skill model.Skill, lowerFilter string) bool {
	if s == nil {
		return true
	}
	if lowerFilter == "" {
		return true
	}
	lowerName := strings.ToLower(skill.Name)
	lowerDesc := strings.ToLower(skill.Description)
	lowerScope := strings.ToLower(skill.DisplayScope())
	lowerPlatform := strings.ToLower(string(skill.Platform))
	return strings.Contains(lowerName, lowerFilter) ||
		strings.Contains(lowerDesc, lowerFilter) ||
		strings.Contains(lowerScope, lowerFilter) ||
		strings.Contains(lowerPlatform, lowerFilter)
}

func (s *dedupeListState) shortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"space/tab toggle",
		"a toggle all",
		"d delete",
		"/ filter",
		"esc clear/back",
		"? help",
		"q quit",
	}
	return strings.Join(keys, " • ")
}

func (s *dedupeListState) fullHelp() string {
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

func (s *dedupeListState) header() string {
	if s == nil {
		return ""
	}
	parts := []string{
		dedupeListStyles.Warning.Render("Select duplicate skills to delete. Only repo/user scope skills shown."),
		fmt.Sprintf("Found %d duplicate pairs across %d skills", len(s.duplicates), len(s.flatSkills)),
	}
	return strings.Join(parts, "\n")
}

func (s *dedupeListState) statusText(filtered, total int, filter string) string {
	selectedCount := len(s.selectedSkills())
	status := fmt.Sprintf("%d skill(s) selected for deletion of %d", selectedCount, filtered)
	if filter != "" {
		status = fmt.Sprintf("%d selected, %d of %d shown (filtered)", selectedCount, filtered, total)
	}
	return status
}

func (s *dedupeListState) detailPanel(m *ListModel[model.Skill]) string {
	if s == nil || m == nil {
		return ""
	}
	skill := s.selectedSkill(m)
	if skill.Name == "" || skill.Description == "" {
		return ""
	}
	descWidth := max(m.width-2, 40)
	return dedupeListStyles.Description.Render(formatDescription(skill.Description, descWidth))
}

func (s *dedupeListState) extraBody(m *ListModel[model.Skill]) string {
	return s.detailPanel(m)
}

func (s *dedupeListState) findSimilarSkill(skill model.Skill) (model.Skill, float64, float64) {
	if s == nil {
		return model.Skill{}, 0, 0
	}
	return findBestDuplicateMatch(s.duplicates, skill)
}

func findBestDuplicateMatch(duplicates []*similarity.ComparisonResult, skill model.Skill) (model.Skill, float64, float64) {
	skillKey := dedupeSkillKey(skill)
	var bestMatch model.Skill
	var bestNameScore, bestContentScore float64

	for _, dup := range duplicates {
		key1 := dedupeSkillKey(dup.Skill1)
		key2 := dedupeSkillKey(dup.Skill2)

		if key1 == skillKey {
			score := dup.NameScore + dup.ContentScore
			if score > bestNameScore+bestContentScore {
				bestMatch = dup.Skill2
				bestNameScore = dup.NameScore
				bestContentScore = dup.ContentScore
			}
		} else if key2 == skillKey {
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

func (s *dedupeListState) skillsToRows(skills []model.Skill) []table.Row {
	if s == nil {
		return nil
	}
	return dedupeSkillsToRows(skills, s.selected, s.columnWidths, s.findSimilarSkill)
}

func dedupeSkillsToRows(skills []model.Skill, selected map[string]bool, widths dedupeListColumnWidths, findSimilar func(model.Skill) (model.Skill, float64, float64)) []table.Row {
	if widths.desc == 0 {
		widths = defaultDedupeListColumnWidths()
	}
	rows := make([]table.Row, len(skills))
	for i, skill := range skills {
		checkbox := "[ ]"
		if selected[dedupeSkillKey(skill)] {
			checkbox = "[✓]"
		}
		similarSkill, nameScore, contentScore := findSimilar(skill)
		rows[i] = table.Row{
			checkbox,
			truncateTableValue(skill.Name, widths.name),
			truncateTableValue(string(skill.Platform), widths.platform),
			truncateTableValue(skill.DisplayScope(), widths.scope),
			truncateTableValue(similarSkill.Name, widths.similar),
			fmt.Sprintf("%.0f%%", nameScore*100),
			fmt.Sprintf("%.0f%%", contentScore*100),
			truncateTableValue(skill.Description, widths.desc),
		}
	}
	return rows
}

func (s *dedupeListState) onWindowSize(m *ListModel[model.Skill], width, height int) {
	if s == nil || m == nil {
		return
	}
	s.width = width
	s.height = height
	newHeight := max(height-12, 5)
	m.table.SetHeight(newHeight)
	s.applyColumnWidths(m, width)
}

func (s *dedupeListState) applyColumnWidths(m *ListModel[model.Skill], totalWidth int) {
	if s == nil || m == nil {
		return
	}
	widths := defaultDedupeListColumnWidths()
	if totalWidth > 0 {
		const checkboxWidth = 3
		const separatorWidth = 14
		descWidth := totalWidth - (checkboxWidth + widths.name + widths.platform + widths.scope + widths.similar + widths.nameScore + widths.contentScore + separatorWidth)
		if descWidth < 40 {
			descWidth = 40
		}
		widths.desc = descWidth
	}
	s.columnWidths = widths
	m.table.SetColumns([]table.Column{
		{Title: " ", Width: 3},
		{Title: "Name", Width: widths.name},
		{Title: "Platform", Width: widths.platform},
		{Title: "Scope", Width: widths.scope},
		{Title: "Similar To", Width: widths.similar},
		{Title: "Name%", Width: widths.nameScore},
		{Title: "Content%", Width: widths.contentScore},
		{Title: "Description", Width: widths.desc},
	})
	m.table.SetRows(s.skillsToRows(m.filtered))
}

func (s *dedupeListState) extraKeys(m *ListModel[model.Skill], msg tea.KeyMsg) bool {
	if s == nil || m == nil {
		return false
	}
	switch msg.String() {
	case " ", "tab":
		skill := s.selectedSkill(m)
		if skill.Name != "" {
			s.selected[dedupeSkillKey(skill)] = !s.selected[dedupeSkillKey(skill)]
			m.table.SetRows(s.skillsToRows(m.filtered))
		}
		return true
	case "a":
		selectedCount := 0
		for _, skill := range m.filtered {
			if s.selected[dedupeSkillKey(skill)] {
				selectedCount++
			}
		}
		selectAll := selectedCount < len(m.filtered)/2+1
		for _, skill := range m.filtered {
			s.selected[dedupeSkillKey(skill)] = selectAll
		}
		m.table.SetRows(s.skillsToRows(m.filtered))
		return true
	case "d":
		selectedSkills := s.selectedSkills()
		if len(selectedSkills) == 0 {
			return true
		}
		m.result = DedupeListResult{Action: DedupeActionDelete, SelectedSkills: selectedSkills}
		m.confirmMode = true
		m.confirmMsg = fmt.Sprintf("⚠️  DELETE %d duplicate skill(s)? This cannot be undone! (y/n)", len(selectedSkills))
		return true
	}
	return false
}

// findSimilarSkill finds the best matching similar skill for a given skill.
func (m DedupeListModel) findSimilarSkill(skill model.Skill) (model.Skill, float64, float64) {
	return findBestDuplicateMatch(m.duplicates, skill)
}

func (m DedupeListModel) skillsToRows(skills []model.Skill) []table.Row {
	if m.state != nil {
		return m.state.skillsToRows(skills)
	}
	return dedupeSkillsToRows(skills, m.selected, m.columnWidths, m.findSimilarSkill)
}

// Init implements tea.Model.
func (m DedupeListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m DedupeListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.syncListModelFromCompat()
	if msg, ok := msg.(tea.KeyMsg); ok {
		// Declining the delete confirmation must roll back the staged delete
		// result. The 'd' handler sets result=DedupeActionDelete before opening
		// the confirm prompt, but the generic confirm handler only clears
		// confirmMode on decline — not the result — so a later quit would
		// otherwise return a stale delete the user already rejected. Resetting
		// to a typed zero (not nil) lets syncCompatFromBase propagate it.
		if m.confirmMode {
			switch msg.String() {
			case "n", "N", "esc":
				m.result = DedupeListResult{}
				m.ListModel.result = DedupeListResult{}
			}
		}
		if m.state != nil && !m.filtering && !m.confirmMode && m.state.extraKeys(&m.ListModel, msg) {
			m.syncCompatFromBase()
			return m, nil
		}
	}

	updated, cmd := m.ListModel.Update(msg)
	if next, ok := updated.(ListModel[model.Skill]); ok {
		m.ListModel = next
	}
	m.syncCompatFromBase()
	return m, cmd
}

func (m *DedupeListModel) applyFilter() {
	if m == nil {
		return
	}
	m.syncListModelFromCompat()
	m.ListModel.applyFilter()
	m.syncCompatFromBase()
}

// View implements tea.Model.
func (m DedupeListModel) View() string {
	m.syncListModelFromCompat()
	view := m.ListModel.View()
	m.syncCompatFromBase()
	return view
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
