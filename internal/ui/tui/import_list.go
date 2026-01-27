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

// ImportListModel is the BubbleTea model for interactive skill import.
type ImportListModel struct {
	// File picker for source selection
	filepicker filepicker.Model

	// Skill selection table
	table    table.Model
	skills   []model.Skill
	filtered []model.Skill
	selected map[string]bool

	// Destination options
	targetPlatform model.Platform
	targetScope    model.SkillScope
	platforms      []model.Platform
	scopes         []model.SkillScope
	platformCursor int
	scopeCursor    int

	// UI state
	keys       importListKeyMap
	result     ImportListResult
	phase      importPhase
	sourcePath string
	filter     string
	filtering  bool
	showHelp   bool
	width      int
	height     int
	quitting   bool
	err        error
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
}

// importSkillKey creates a unique key for a skill.
func importSkillKey(s model.Skill) string {
	return fmt.Sprintf("%s:%s:%s", s.Platform, s.Scope, s.Name)
}

// NewImportListModel creates a new import list model.
func NewImportListModel() ImportListModel {
	// Initialize file picker
	fp := filepicker.New()
	fp.AllowedTypes = []string{".md"}
	fp.DirAllowed = true
	fp.FileAllowed = true
	fp.ShowPermissions = false
	fp.ShowSize = true
	fp.ShowHidden = false

	// Start in current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	fp.CurrentDirectory = cwd

	// Initialize available platforms and scopes
	platforms := model.AllPlatforms()
	scopes := []model.SkillScope{
		model.ScopeRepo,
		model.ScopeUser,
	}

	return ImportListModel{
		filepicker:     fp,
		keys:           defaultImportListKeyMap(),
		phase:          phaseFilePicker,
		platforms:      platforms,
		scopes:         scopes,
		targetPlatform: model.ClaudeCode, // Default to Claude Code
		targetScope:    model.ScopeRepo,  // Default to repo scope
		selected:       make(map[string]bool),
	}
}

// Init implements tea.Model.
func (m ImportListModel) Init() tea.Cmd {
	return m.filepicker.Init()
}

// Update implements tea.Model.
func (m ImportListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.filepicker.SetHeight(max(msg.Height-8, 10))
		if m.table.Columns() != nil {
			newHeight := max(msg.Height-12, 5)
			m.table.SetHeight(newHeight)
		}

	case tea.KeyMsg:
		// Global quit handling
		if key.Matches(msg, m.keys.Quit) {
			m.quitting = true
			return m, tea.Quit
		}

		// Phase-specific key handling
		switch m.phase {
		case phaseFilePicker:
			return m.updateFilePicker(msg)
		case phaseSkillSelection:
			return m.updateSkillSelection(msg)
		case phaseDestination:
			return m.updateDestination(msg)
		case phaseConfirm:
			return m.updateConfirm(msg)
		}
	}

	// Update file picker in file picker phase
	if m.phase == phaseFilePicker {
		m.filepicker, cmd = m.filepicker.Update(msg)
		cmds = append(cmds, cmd)

		// Check if a file/directory was selected
		if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
			m.sourcePath = path
			if err := m.loadSkillsFromPath(path); err != nil {
				m.err = err
			} else {
				m.phase = phaseSkillSelection
				m.initSkillTable()
			}
		}
		if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
			// User tried to select a non-.md file, treat as directory
			m.sourcePath = path
			if err := m.loadSkillsFromPath(path); err != nil {
				m.err = err
			} else {
				m.phase = phaseSkillSelection
				m.initSkillTable()
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *ImportListModel) updateFilePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return *m, nil
	}

	m.filepicker, cmd = m.filepicker.Update(msg)
	return *m, cmd
}

func (m *ImportListModel) updateSkillSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle filtering mode
	if m.filtering {
		switch msg.String() {
		case "enter":
			m.filtering = false
			return *m, nil
		case "esc":
			m.filter = ""
			m.filtering = false
			m.applyFilter()
			return *m, nil
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.applyFilter()
			}
			return *m, nil
		default:
			if len(msg.String()) == 1 {
				m.filter += msg.String()
				m.applyFilter()
			}
			return *m, nil
		}
	}

	switch {
	case key.Matches(msg, m.keys.Back):
		// Go back to file picker
		m.phase = phaseFilePicker
		m.skills = nil
		m.filtered = nil
		m.selected = make(map[string]bool)
		m.err = nil
		return *m, nil

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return *m, nil

	case key.Matches(msg, m.keys.Filter):
		m.filtering = true
		return *m, nil

	case key.Matches(msg, m.keys.ClearFlt):
		m.filter = ""
		m.applyFilter()
		return *m, nil

	case key.Matches(msg, m.keys.Toggle):
		if len(m.filtered) > 0 {
			skill := m.getSelectedSkill()
			k := importSkillKey(skill)
			m.selected[k] = !m.selected[k]
			m.table.SetRows(m.skillsToRows(m.filtered))
		}
		return *m, nil

	case key.Matches(msg, m.keys.ToggleAll):
		selectedCount := 0
		for _, s := range m.filtered {
			if m.selected[importSkillKey(s)] {
				selectedCount++
			}
		}
		selectAll := selectedCount < len(m.filtered)/2+1
		for _, s := range m.filtered {
			m.selected[importSkillKey(s)] = selectAll
		}
		m.table.SetRows(m.skillsToRows(m.filtered))
		return *m, nil

	case key.Matches(msg, m.keys.Select):
		if len(m.getSelectedSkills()) > 0 {
			m.phase = phaseDestination
		}
		return *m, nil
	}

	m.table, cmd = m.table.Update(msg)
	return *m, cmd
}

func (m *ImportListModel) updateDestination(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.phase = phaseSkillSelection
		return *m, nil

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return *m, nil

	case key.Matches(msg, m.keys.Up):
		// Toggle between platform and scope selection
		// Currently on scope, move to platform
		return *m, nil

	case key.Matches(msg, m.keys.Down):
		// Toggle between platform and scope selection
		return *m, nil

	case key.Matches(msg, m.keys.Left):
		// Cycle platform left
		if m.platformCursor > 0 {
			m.platformCursor--
			m.targetPlatform = m.platforms[m.platformCursor]
		}
		return *m, nil

	case key.Matches(msg, m.keys.Right):
		// Cycle platform right
		if m.platformCursor < len(m.platforms)-1 {
			m.platformCursor++
			m.targetPlatform = m.platforms[m.platformCursor]
		}
		return *m, nil

	case key.Matches(msg, m.keys.Toggle):
		// Toggle scope
		m.scopeCursor = (m.scopeCursor + 1) % len(m.scopes)
		m.targetScope = m.scopes[m.scopeCursor]
		return *m, nil

	case key.Matches(msg, m.keys.Select):
		m.phase = phaseConfirm
		return *m, nil
	}

	return *m, nil
}

func (m *ImportListModel) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.phase = phaseDestination
		return *m, nil

	case key.Matches(msg, m.keys.Confirm), msg.String() == "Y":
		m.result = ImportListResult{
			Action:         ImportActionImport,
			SelectedSkills: m.getSelectedSkills(),
			SourcePath:     m.sourcePath,
			TargetPlatform: m.targetPlatform,
			TargetScope:    m.targetScope,
		}
		m.quitting = true
		return *m, tea.Quit

	case msg.String() == "n", msg.String() == "N":
		m.phase = phaseDestination
		return *m, nil
	}

	return *m, nil
}

func (m *ImportListModel) loadSkillsFromPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot access path: %w", err)
	}

	var baseDir string
	if info.IsDir() {
		baseDir = path
	} else {
		baseDir = filepath.Dir(path)
	}

	// Use the skills parser to discover and parse skills
	skillsParser := skills.New(baseDir, model.ClaudeCode)
	parsedSkills, err := skillsParser.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse skills: %w", err)
	}

	if len(parsedSkills) == 0 {
		return fmt.Errorf("no SKILL.md files found in %s", baseDir)
	}

	m.skills = parsedSkills

	// Sort skills alphabetically by name (case-insensitive)
	sort.Slice(m.skills, func(i, j int) bool {
		return strings.ToLower(m.skills[i].Name) < strings.ToLower(m.skills[j].Name)
	})

	m.filtered = m.skills

	// Select all skills by default
	for _, s := range m.skills {
		m.selected[importSkillKey(s)] = true
	}

	return nil
}

func (m *ImportListModel) initSkillTable() {
	columns := []table.Column{
		{Title: " ", Width: 3},
		{Title: "Name", Width: 25},
		{Title: "Description", Width: 40},
		{Title: "Scope", Width: 10},
	}

	rows := m.skillsToRows(m.filtered)

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
}

func (m ImportListModel) skillsToRows(skills []model.Skill) []table.Row {
	rows := make([]table.Row, len(skills))
	for i, s := range skills {
		checkbox := "[ ]"
		if m.selected[importSkillKey(s)] {
			checkbox = "[✓]"
		}

		name := s.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}
		desc := s.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		scope := s.DisplayScope()
		if len(scope) > 10 {
			scope = scope[:7] + "..."
		}

		rows[i] = table.Row{checkbox, name, desc, scope}
	}
	return rows
}

func (m *ImportListModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.skills
	} else {
		var filtered []model.Skill
		lowerFilter := strings.ToLower(m.filter)
		for _, s := range m.skills {
			if strings.Contains(strings.ToLower(s.Name), lowerFilter) ||
				strings.Contains(strings.ToLower(s.Description), lowerFilter) {
				filtered = append(filtered, s)
			}
		}
		m.filtered = filtered
	}
	m.table.SetRows(m.skillsToRows(m.filtered))
}

func (m ImportListModel) getSelectedSkill() model.Skill {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return model.Skill{}
}

func (m ImportListModel) getSelectedSkills() []model.Skill {
	var selected []model.Skill
	for _, s := range m.skills {
		if m.selected[importSkillKey(s)] {
			selected = append(selected, s)
		}
	}
	return selected
}

// View implements tea.Model.
func (m ImportListModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	title := importListStyles.Title.Render("📥 Import Skills")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Phase indicator
	phaseNames := []string{"Select Source", "Select Skills", "Choose Destination", "Confirm"}
	phaseIndicator := importListStyles.Phase.Render(fmt.Sprintf("Step %d/%d: %s", m.phase+1, len(phaseNames), phaseNames[m.phase]))
	b.WriteString(phaseIndicator)
	b.WriteString("\n\n")

	// Error display
	if m.err != nil {
		b.WriteString(importListStyles.Error.Render(fmt.Sprintf("Error: %s", m.err.Error())))
		b.WriteString("\n\n")
	}

	// Phase-specific view
	switch m.phase {
	case phaseFilePicker:
		b.WriteString(m.viewFilePicker())
	case phaseSkillSelection:
		b.WriteString(m.viewSkillSelection())
	case phaseDestination:
		b.WriteString(m.viewDestination())
	case phaseConfirm:
		b.WriteString(m.viewConfirm())
	}

	// Help
	b.WriteString("\n")
	if m.showHelp {
		b.WriteString(m.renderFullHelp())
	} else {
		b.WriteString(m.renderShortHelp())
	}

	return b.String()
}

func (m ImportListModel) viewFilePicker() string {
	var b strings.Builder
	b.WriteString("Navigate to a directory containing SKILL.md files or select a specific file:\n\n")
	b.WriteString(m.filepicker.View())
	return b.String()
}

func (m ImportListModel) viewSkillSelection() string {
	var b strings.Builder

	// Source path
	pathLabel := importListStyles.Option.Render("Source: ")
	pathVal := importListStyles.Path.Render(m.sourcePath)
	b.WriteString(pathLabel + pathVal + "\n\n")

	// Filter indicator
	if m.filter != "" || m.filtering {
		filterStr := importListStyles.Filter.Render("Filter: ")
		filterVal := importListStyles.FilterInput.Render(m.filter)
		if m.filtering {
			filterVal += "█"
		}
		b.WriteString(filterStr + filterVal + "\n\n")
	}

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n")

	// Status bar
	selectedCount := len(m.getSelectedSkills())
	status := fmt.Sprintf("%d skill(s) selected of %d total", selectedCount, len(m.filtered))
	b.WriteString(importListStyles.Status.Render(status))

	return b.String()
}

func (m ImportListModel) viewDestination() string {
	var b strings.Builder

	b.WriteString("Choose where to import the selected skills:\n\n")

	// Platform selection
	platformLabel := importListStyles.Option.Render("Platform: ")
	var platformOptions []string
	for i, p := range m.platforms {
		if i == m.platformCursor {
			platformOptions = append(platformOptions, importListStyles.Selected.Render(fmt.Sprintf("[%s]", p)))
		} else {
			platformOptions = append(platformOptions, fmt.Sprintf(" %s ", p))
		}
	}
	b.WriteString(platformLabel + strings.Join(platformOptions, " ") + "\n")

	// Scope selection
	scopeLabel := importListStyles.Option.Render("Scope:    ")
	var scopeOptions []string
	for i, s := range m.scopes {
		scopeName := string(s)
		if i == m.scopeCursor {
			scopeOptions = append(scopeOptions, importListStyles.Selected.Render(fmt.Sprintf("[%s]", scopeName)))
		} else {
			scopeOptions = append(scopeOptions, fmt.Sprintf(" %s ", scopeName))
		}
	}
	b.WriteString(scopeLabel + strings.Join(scopeOptions, " ") + "\n\n")

	// Summary
	selectedCount := len(m.getSelectedSkills())
	summary := fmt.Sprintf("Will import %d skill(s) to %s (%s scope)", selectedCount, m.targetPlatform, m.targetScope)
	b.WriteString(importListStyles.Status.Render(summary))

	return b.String()
}

func (m ImportListModel) viewConfirm() string {
	var b strings.Builder

	selectedSkills := m.getSelectedSkills()
	b.WriteString(importListStyles.Confirm.Render(fmt.Sprintf(
		"Import %d skill(s) to %s (%s)? (y/n)",
		len(selectedSkills),
		m.targetPlatform,
		m.targetScope,
	)))
	b.WriteString("\n\n")

	// Show skill names
	b.WriteString("Skills to import:\n")
	for i, s := range selectedSkills {
		if i >= 10 {
			b.WriteString(fmt.Sprintf("  ... and %d more\n", len(selectedSkills)-10))
			break
		}
		b.WriteString(fmt.Sprintf("  • %s\n", s.Name))
	}

	return b.String()
}

func (m ImportListModel) renderShortHelp() string {
	var keys []string
	switch m.phase {
	case phaseFilePicker:
		keys = []string{"↑/↓ navigate", "enter select", "? help", "q quit"}
	case phaseSkillSelection:
		keys = []string{"↑/↓ navigate", "space toggle", "a toggle all", "enter next", "esc back", "/ filter", "? help", "q quit"}
	case phaseDestination:
		keys = []string{"←/→ platform", "space scope", "enter next", "esc back", "? help", "q quit"}
	case phaseConfirm:
		keys = []string{"y confirm", "n/esc back", "q quit"}
	}
	return importListStyles.Help.Render(strings.Join(keys, " • "))
}

func (m ImportListModel) renderFullHelp() string {
	help := `Navigation:
  ↑/k      Move up
  ↓/j      Move down
  ←/h      Previous option
  →/l      Next option

Selection (skill list):
  Space/Tab  Toggle current skill
  a          Toggle all skills
  /          Start filtering
  Ctrl+u     Clear filter

Flow:
  Enter    Proceed to next step
  Esc      Go back to previous step
  y        Confirm import (final step)
  n        Cancel (at confirm step)

General:
  ?        Toggle full help
  q        Quit without importing`
	return importListStyles.Help.Render(help)
}

// Result returns the result of the user interaction.
func (m ImportListModel) Result() ImportListResult {
	return m.result
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
