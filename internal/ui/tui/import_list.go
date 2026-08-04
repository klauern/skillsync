// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/skills"
)

// ImportAction represents the action to perform after import configuration.
type ImportAction int

const (
	// ImportActionNone means no action was taken (user quit).
	ImportActionNone ImportAction = iota
	// ImportActionImport means the user wants to import selected skills.
	ImportActionImport
)

// ImportListResult contains the result of the import list TUI interaction.
type ImportListResult struct {
	Action         ImportAction
	SelectedSkills []model.Skill
	SourcePath     string
	TargetPlatform model.Platform
	TargetScope    model.SkillScope
}

// importPhase represents the current phase of the import flow.
type importPhase int

const (
	phaseFilePicker importPhase = iota
	phaseSkillSelection
	phaseDestination
	phaseConfirm
)

var importPhaseNames = []string{"Select Source", "Select Skills", "Choose Destination", "Confirm"}

// importListKeyMap defines the key bindings for the import list.
type importListKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Toggle    key.Binding
	ToggleAll key.Binding
	Select    key.Binding
	Back      key.Binding
	Confirm   key.Binding
	Filter    key.Binding
	ClearFlt  key.Binding
	Help      key.Binding
	Quit      key.Binding
}

type importListColumnWidths struct {
	name  int
	desc  int
	scope int
}

type importListState struct {
	phase          importPhase
	result         ImportListResult
	sourcePath     string
	targetPlatform model.Platform
	targetScope    model.SkillScope
	platforms      []model.Platform
	scopes         []model.SkillScope
	platformCursor int
	scopeCursor    int
	showHelp       bool
	err            error
	width          int
	height         int
	selected       map[string]bool
	columnWidths   importListColumnWidths
}

// ImportListModel is the BubbleTea model for interactive skill import.
type ImportListModel struct {
	ListModel[model.Skill]
	filepicker filepicker.Model
	keys       importListKeyMap
	state      *importListState
}

// Styles for the import list TUI.
var importListStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Filter      lipgloss.Style
	FilterInput lipgloss.Style
	Confirm     lipgloss.Style
	Status      lipgloss.Style
	Selected    lipgloss.Style
	Checkbox    lipgloss.Style
	Option      lipgloss.Style
	OptionVal   lipgloss.Style
	Error       lipgloss.Style
	Phase       lipgloss.Style
	Path        lipgloss.Style
	Description lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Filter:      lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	FilterInput: lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Confirm:     lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true).Padding(1, 2),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Selected:    lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
	Checkbox:    lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	Option:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	OptionVal:   lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
	Error:       lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true),
	Phase:       lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true),
	Path:        lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Italic(true),
	Description: lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1),
}

func defaultImportListColumnWidths() importListColumnWidths {
	return importListColumnWidths{
		name:  25,
		desc:  60,
		scope: 10,
	}
}

func defaultImportListKeyMap() importListKeyMap {
	return importListKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" ", "tab"),
			key.WithHelp("space/tab", "toggle"),
		),
		ToggleAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "toggle all"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "confirm import"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ClearFlt: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "clear filter"),
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

func quitImportCmd() tea.Cmd { return func() tea.Msg { return tea.QuitMsg{} } }

func newImportListState() *importListState {
	platforms := model.AllPlatforms()
	targetPlatform := model.ClaudeCode
	platformCursor := 0
	for i, platform := range platforms {
		if platform == targetPlatform {
			platformCursor = i
			break
		}
	}

	scopes := []model.SkillScope{model.ScopeRepo, model.ScopeUser}
	targetScope := model.ScopeRepo
	scopeCursor := 0
	for i, scope := range scopes {
		if scope == targetScope {
			scopeCursor = i
			break
		}
	}

	return &importListState{
		phase:          phaseFilePicker,
		targetPlatform: targetPlatform,
		targetScope:    targetScope,
		platforms:      platforms,
		scopes:         scopes,
		platformCursor: platformCursor,
		scopeCursor:    scopeCursor,
		selected:       make(map[string]bool),
		columnWidths:   defaultImportListColumnWidths(),
	}
}

func selectedImportSkills(state *importListState, skills []model.Skill) []model.Skill {
	var selected []model.Skill
	for _, skill := range skills {
		if state != nil && state.selected[scopedSkillKey(skill)] {
			selected = append(selected, skill)
		}
	}
	return selected
}

func selectedImportSkillCount(state *importListState, skills []model.Skill) int {
	count := 0
	for _, skill := range skills {
		if state != nil && state.selected[scopedSkillKey(skill)] {
			count++
		}
	}
	return count
}

func selectedImportSkillAtCursor(state *importListState, m *ListModel[model.Skill]) model.Skill {
	if m == nil {
		return model.Skill{}
	}
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return model.Skill{}
}

func buildImportSkillListModel(state *importListState, skills []model.Skill) ListModel[model.Skill] {
	widths := state.columnWidths
	if widths.desc == 0 {
		widths = defaultImportListColumnWidths()
		state.columnWidths = widths
	}

	toRows := func(items []model.Skill) []table.Row {
		rows := make([]table.Row, len(items))
		for i, skill := range items {
			checkbox := "[ ]"
			if state.selected[scopedSkillKey(skill)] {
				checkbox = "[✓]"
			}
			rows[i] = table.Row{
				checkbox,
				truncateTableValue(skill.Name, widths.name),
				truncateTableValue(string(skill.Platform), 12),
				truncateTableValue(skill.Description, widths.desc),
				truncateTableValue(skill.DisplayScope(), widths.scope),
			}
		}
		return rows
	}

	toggleKey := key.NewBinding(key.WithKeys(" ", "tab"), key.WithHelp("space/tab", "toggle"))
	toggleAllKey := key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "toggle all"))
	selectKey := key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "destination"))

	statusText := func(filtered, _ int, _ string) string {
		selectedCount := selectedImportSkillCount(state, skills)
		return fmt.Sprintf("%d skill(s) selected of %d visible", selectedCount, filtered)
	}

	header := func() string {
		phase := importListStyles.Phase.Render(fmt.Sprintf("Step %d/%d: %s", phaseSkillSelection+1, len(importPhaseNames), importPhaseNames[phaseSkillSelection]))
		path := importListStyles.Option.Render("Source: ") + importListStyles.Path.Render(state.sourcePath)
		return strings.Join([]string{phase, path}, "\n")
	}

	extraBody := func(m *ListModel[model.Skill]) string {
		selected := selectedImportSkillAtCursor(state, m)
		if selected.Name == "" || selected.Description == "" {
			return ""
		}
		descWidth := max(m.width-2, 40)
		return importListStyles.Description.Render(formatDescription(selected.Description, descWidth))
	}

	extraKeys := func(m *ListModel[model.Skill], msg tea.KeyMsg) bool {
		switch {
		case key.Matches(msg, toggleKey):
			if len(m.filtered) > 0 {
				skill := selectedImportSkillAtCursor(state, m)
				if skill.Name != "" {
					state.selected[scopedSkillKey(skill)] = !state.selected[scopedSkillKey(skill)]
					m.table.SetRows(m.cfg.ToRows(m.filtered))
				}
			}
			return true
		case key.Matches(msg, toggleAllKey):
			selectedCount := 0
			for _, skill := range m.filtered {
				if state.selected[scopedSkillKey(skill)] {
					selectedCount++
				}
			}
			selectAll := selectedCount < len(m.filtered)/2+1
			for _, skill := range m.filtered {
				state.selected[scopedSkillKey(skill)] = selectAll
			}
			m.table.SetRows(m.cfg.ToRows(m.filtered))
			return true
		case key.Matches(msg, selectKey):
			if selectedImportSkillCount(state, skills) > 0 {
				state.phase = phaseDestination
			}
			return true
		}
		return false
	}

	onWindowSize := func(m *ListModel[model.Skill], width, height int) {
		state.width = width
		state.height = height
		const checkboxWidth = 3
		const separatorWidth = 6
		newDesc := width - (checkboxWidth + widths.name + widths.scope + separatorWidth)
		if newDesc < 40 {
			newDesc = 40
		}
		widths.desc = newDesc
		state.columnWidths = widths
		m.table.SetColumns([]table.Column{
			{Title: " ", Width: checkboxWidth},
			{Title: "Name", Width: widths.name},
			{Title: "Platform", Width: 12},
			{Title: "Description", Width: widths.desc},
			{Title: "Scope", Width: widths.scope},
		})
		m.table.SetRows(m.cfg.ToRows(m.filtered))
	}

	shortHelp := func() string {
		return strings.Join([]string{
			"↑/↓ navigate",
			"space/tab toggle",
			"a toggle all",
			"enter next",
			"esc back",
			"/ filter",
			"ctrl+u clear",
			"? help",
			"q quit",
		}, " • ")
	}

	fullHelp := func() string {
		return `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  g/Home   Go to top
  G/End    Go to bottom

Selection:
  Space/Tab  Toggle current skill
  a          Toggle all skills
  Enter      Proceed to destination
  Esc        Go back to source picker

Filter:
  /          Start filtering
  Ctrl+u     Clear filter

General:
  ?          Toggle full help
  q          Quit without importing`
	}

	cfg := ListConfig[model.Skill]{
		Title: "📥 Import Skills",
		Columns: []table.Column{
			{Title: " ", Width: 3},
			{Title: "Name", Width: widths.name},
			{Title: "Platform", Width: 12},
			{Title: "Description", Width: widths.desc},
			{Title: "Scope", Width: widths.scope},
		},
		ToRows:        toRows,
		Matches:       skillMatchesFilter,
		StatusText:    statusText,
		ReservedLines: 12,
		Header:        header,
		ExtraBody:     extraBody,
		ExtraKeys:     extraKeys,
		OnWindowSize:  onWindowSize,
		ShortHelp:     shortHelp,
		FullHelp:      fullHelp,
	}

	m := NewListModel(skills, cfg)
	m.showHelp = state.showHelp
	return m
}

func newImportListModelWithState() ImportListModel {
	fp := filepicker.New()
	fp.AllowedTypes = []string{".md"}
	fp.DirAllowed = true
	fp.FileAllowed = true
	fp.ShowPermissions = false
	fp.ShowSize = true
	fp.ShowHidden = false

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	fp.CurrentDirectory = cwd

	return ImportListModel{
		filepicker: fp,
		keys:       defaultImportListKeyMap(),
		state:      newImportListState(),
	}
}

// NewImportListModel creates a new import list model.
func NewImportListModel() ImportListModel {
	return newImportListModelWithState()
}

// Init implements tea.Model.
func (m ImportListModel) Init() tea.Cmd {
	return m.filepicker.Init()
}

func (m ImportListModel) updateFilePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, m.keys.Help):
			m.state.showHelp = !m.state.showHelp
			return m, nil
		case key.Matches(keyMsg, m.keys.Quit):
			return m, quitImportCmd()
		}
	}

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)
	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		m.state.sourcePath = path
		skills, err := m.loadSkillsFromPath(path)
		if err != nil {
			m.state.err = err
		} else {
			m.state.err = nil
			m.state.phase = phaseSkillSelection
			m.state.selected = make(map[string]bool)
			for _, skill := range skills {
				m.state.selected[scopedSkillKey(skill)] = true
			}
			m.ListModel = buildImportSkillListModel(m.state, skills)
			m.ListModel.showHelp = m.state.showHelp
		}
	}
	if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		m.state.sourcePath = path
		skills, err := m.loadSkillsFromPath(path)
		if err != nil {
			m.state.err = err
		} else {
			m.state.err = nil
			m.state.phase = phaseSkillSelection
			m.state.selected = make(map[string]bool)
			for _, skill := range skills {
				m.state.selected[scopedSkillKey(skill)] = true
			}
			m.ListModel = buildImportSkillListModel(m.state, skills)
			m.ListModel.showHelp = m.state.showHelp
		}
	}

	return m, cmd
}

func (m ImportListModel) updateSkillSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, m.keys.ClearFlt) {
			m.ListModel.filter = ""
			m.ListModel.filtering = false
			m.ListModel.applyFilter()
			return m, nil
		}
		if key.Matches(keyMsg, m.keys.Back) && !m.ListModel.filtering {
			m.state.phase = phaseFilePicker
			m.state.err = nil
			return m, nil
		}
	}

	inner, cmd := m.ListModel.Update(msg)
	if lm, ok := inner.(ListModel[model.Skill]); ok {
		m.ListModel = lm
		m.state.showHelp = m.ListModel.showHelp
	}
	return m, cmd
}

func (m ImportListModel) updateDestination(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, m.keys.Back):
			m.state.phase = phaseSkillSelection
			return m, nil
		case key.Matches(keyMsg, m.keys.Help):
			m.state.showHelp = !m.state.showHelp
			return m, nil
		case key.Matches(keyMsg, m.keys.Quit):
			return m, quitImportCmd()
		case key.Matches(keyMsg, m.keys.Left):
			if m.state.platformCursor > 0 {
				m.state.platformCursor--
				m.state.targetPlatform = m.state.platforms[m.state.platformCursor]
			}
			return m, nil
		case key.Matches(keyMsg, m.keys.Right):
			if m.state.platformCursor < len(m.state.platforms)-1 {
				m.state.platformCursor++
				m.state.targetPlatform = m.state.platforms[m.state.platformCursor]
			}
			return m, nil
		case keyMsg.Type == tea.KeySpace || keyMsg.Type == tea.KeyTab:
			if len(m.state.scopes) > 0 {
				m.state.scopeCursor = (m.state.scopeCursor + 1) % len(m.state.scopes)
				m.state.targetScope = m.state.scopes[m.state.scopeCursor]
			}
			return m, nil
		case keyMsg.Type == tea.KeyEnter:
			m.state.phase = phaseConfirm
			return m, nil
		}
	}

	return m, nil
}

func (m ImportListModel) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, m.keys.Back):
			m.state.phase = phaseDestination
			return m, nil
		case key.Matches(keyMsg, m.keys.Help):
			m.state.showHelp = !m.state.showHelp
			return m, nil
		case key.Matches(keyMsg, m.keys.Quit):
			return m, quitImportCmd()
		case keyMsg.String() == "y" || keyMsg.String() == "Y":
			m.state.result = ImportListResult{
				Action:         ImportActionImport,
				SelectedSkills: selectedImportSkills(m.state, m.ListModel.allItems),
				SourcePath:     m.state.sourcePath,
				TargetPlatform: m.state.targetPlatform,
				TargetScope:    m.state.targetScope,
			}
			return m, quitImportCmd()
		case keyMsg.String() == "n" || keyMsg.String() == "N":
			m.state.phase = phaseDestination
			return m, nil
		}
	}

	return m, nil
}

func (m ImportListModel) loadSkillsFromPath(path string) ([]model.Skill, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	var baseDir string
	if info.IsDir() {
		baseDir = path
	} else {
		baseDir = filepath.Dir(path)
	}

	skillsParser := skills.New(baseDir, model.ClaudeCode)
	parsedSkills, err := skillsParser.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse skills: %w", err)
	}

	if len(parsedSkills) == 0 {
		return nil, fmt.Errorf("no SKILL.md files found in %s", baseDir)
	}

	sort.Slice(parsedSkills, func(i, j int) bool {
		return strings.ToLower(parsedSkills[i].Name) < strings.ToLower(parsedSkills[j].Name)
	})

	return parsedSkills, nil
}

func (m ImportListModel) flowHeader() string {
	var b strings.Builder
	b.WriteString(importListStyles.Title.Render("📥 Import Skills"))
	b.WriteString("\n\n")
	b.WriteString(importListStyles.Phase.Render(fmt.Sprintf("Step %d/%d: %s", m.state.phase+1, len(importPhaseNames), importPhaseNames[m.state.phase])))
	return b.String()
}

func (m ImportListModel) viewFilePicker() string {
	var b strings.Builder
	b.WriteString("Navigate to a directory containing SKILL.md files or select a specific file:\n\n")
	b.WriteString(m.filepicker.View())
	return b.String()
}

func (m ImportListModel) viewDestination() string {
	var b strings.Builder
	b.WriteString("Choose where to import the selected skills:\n\n")

	platformLabel := importListStyles.Option.Render("Platform: ")
	var platformOptions []string
	for i, platform := range m.state.platforms {
		if i == m.state.platformCursor {
			platformOptions = append(platformOptions, importListStyles.Selected.Render(fmt.Sprintf("[%s]", platform)))
		} else {
			platformOptions = append(platformOptions, fmt.Sprintf(" %s ", platform))
		}
	}
	b.WriteString(platformLabel + strings.Join(platformOptions, " ") + "\n")

	scopeLabel := importListStyles.Option.Render("Scope:    ")
	var scopeOptions []string
	for i, scope := range m.state.scopes {
		name := string(scope)
		if i == m.state.scopeCursor {
			scopeOptions = append(scopeOptions, importListStyles.Selected.Render(fmt.Sprintf("[%s]", name)))
		} else {
			scopeOptions = append(scopeOptions, fmt.Sprintf(" %s ", name))
		}
	}
	b.WriteString(scopeLabel + strings.Join(scopeOptions, " ") + "\n\n")

	summary := fmt.Sprintf("Will import %d skill(s) to %s (%s scope)", len(selectedImportSkills(m.state, m.ListModel.allItems)), m.state.targetPlatform, m.state.targetScope)
	b.WriteString(importListStyles.Status.Render(summary))
	return b.String()
}

func (m ImportListModel) viewConfirm() string {
	var b strings.Builder
	selectedSkills := selectedImportSkills(m.state, m.ListModel.allItems)
	b.WriteString(importListStyles.Confirm.Render(fmt.Sprintf(
		"Import %d skill(s) to %s (%s)? (y/n)",
		len(selectedSkills),
		m.state.targetPlatform,
		m.state.targetScope,
	)))
	b.WriteString("\n\n")
	b.WriteString("Skills to import:\n")
	for i, skill := range selectedSkills {
		if i >= 10 {
			fmt.Fprintf(&b, "  ... and %d more\n", len(selectedSkills)-10)
			break
		}
		fmt.Fprintf(&b, "  • %s\n", skill.Name)
	}
	return b.String()
}

func (m ImportListModel) renderShortHelp() string {
	var keys []string
	switch m.state.phase {
	case phaseFilePicker:
		keys = []string{"↑/↓ navigate", "enter select", "? help", "q quit"}
	case phaseDestination:
		keys = []string{"←/→ platform", "space scope", "enter next", "esc back", "? help", "q quit"}
	case phaseConfirm:
		keys = []string{"y confirm", "n/esc back", "q quit"}
	}
	return importListStyles.Help.Render(strings.Join(keys, " • "))
}

func (m ImportListModel) renderFullHelp() string {
	switch m.state.phase {
	case phaseFilePicker:
		return importListStyles.Help.Render(`Browse to a directory containing SKILL.md files, or select a specific file.

Navigation:
  ↑/↓      Move through entries
  Enter    Open directory or select file

General:
  ?        Toggle full help
  q        Quit without importing`)
	case phaseDestination:
		return importListStyles.Help.Render(`Destination:
  ←/h      Previous platform
  →/l      Next platform
  Space    Cycle scope
  Enter    Continue to confirmation
  Esc      Go back to skill selection

General:
  ?        Toggle full help
  q        Quit without importing`)
	case phaseConfirm:
		return importListStyles.Help.Render(`Confirmation:
  y        Confirm import
  n/esc    Go back to destination selection
  q        Quit without importing`)
	default:
		return ""
	}
}

// Update implements tea.Model.
func (m ImportListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.state == nil {
		m.state = newImportListState()
	}

	switch m.state.phase {
	case phaseFilePicker:
		return m.updateFilePicker(msg)
	case phaseSkillSelection:
		return m.updateSkillSelection(msg)
	case phaseDestination:
		return m.updateDestination(msg)
	case phaseConfirm:
		return m.updateConfirm(msg)
	default:
		return m, nil
	}
}

// View implements tea.Model.
func (m ImportListModel) View() string {
	if m.state == nil {
		return ""
	}
	if m.state.phase == phaseSkillSelection {
		m.ListModel.showHelp = m.state.showHelp
		var b strings.Builder
		if m.state.err != nil {
			b.WriteString(importListStyles.Error.Render(fmt.Sprintf("Error: %s", m.state.err.Error())))
			b.WriteString("\n\n")
		}
		b.WriteString(m.ListModel.View())
		return b.String()
	}

	var b strings.Builder
	if m.state.err != nil {
		b.WriteString(importListStyles.Error.Render(fmt.Sprintf("Error: %s", m.state.err.Error())))
		b.WriteString("\n\n")
	}
	b.WriteString(m.flowHeader())
	b.WriteString("\n\n")

	switch m.state.phase {
	case phaseFilePicker:
		b.WriteString(m.viewFilePicker())
	case phaseDestination:
		b.WriteString(m.viewDestination())
	case phaseConfirm:
		b.WriteString(m.viewConfirm())
	}

	b.WriteString("\n")
	if m.state.showHelp {
		b.WriteString(m.renderFullHelp())
	} else {
		b.WriteString(m.renderShortHelp())
	}
	return b.String()
}

// Result returns the result of the user interaction.
func (m ImportListModel) Result() ImportListResult {
	if m.state == nil {
		return ImportListResult{}
	}
	return m.state.result
}

// RunImportList runs the interactive import list and returns the result.
func RunImportList() (ImportListResult, error) {
	mdl := NewImportListModel()
	finalModel, err := tea.NewProgram(mdl, tea.WithAltScreen()).Run()
	if err != nil {
		return ImportListResult{}, err
	}

	if m, ok := finalModel.(ImportListModel); ok {
		return m.Result(), nil
	}

	return ImportListResult{}, nil
}
