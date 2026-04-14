// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/klauern/skillsync/internal/model"
)

// SyncPickerAction represents the action to perform after sync configuration.
type SyncPickerAction int

const (
	// SyncPickerActionNone means no action was taken (user quit).
	SyncPickerActionNone SyncPickerAction = iota
	// SyncPickerActionSelect means the user completed sync configuration.
	SyncPickerActionSelect
)

// SyncPickerResult contains the result of the sync picker TUI interaction.
type SyncPickerResult struct {
	Action       SyncPickerAction
	Source       model.Platform
	SourceScopes []model.SkillScope // Empty means all scopes
	Target       model.Platform
	TargetScope  model.SkillScope
}

type syncPickerPhase int

const (
	syncPickerPhaseSourcePlatform syncPickerPhase = iota
	syncPickerPhaseSourceScope
	syncPickerPhaseTargetPlatform
	syncPickerPhaseTargetScope
)

type sourceScopeOption struct {
	label string
	scope model.SkillScope
}

// syncPickerKeyMap defines the key bindings for the sync picker.
type syncPickerKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Select    key.Binding
	Toggle    key.Binding
	ToggleAll key.Binding
	Back      key.Binding
	Help      key.Binding
	Quit      key.Binding
}

func defaultSyncPickerKeyMap() syncPickerKeyMap {
	return syncPickerKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "next"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" ", "tab"),
			key.WithHelp("space/tab", "toggle"),
		),
		ToggleAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "all"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
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

// SyncPickerModel is the BubbleTea model for choosing sync source/target with scope.
type SyncPickerModel struct {
	platforms         []model.Platform
	sourceScopes      []sourceScopeOption
	sourceSelected    map[model.SkillScope]bool
	targetScopes      []model.SkillScope
	cursor            int
	source            model.Platform
	target            model.Platform
	targetScopeChoice int
	phase             syncPickerPhase
	keys              syncPickerKeyMap
	result            SyncPickerResult
	showHelp          bool
	width             int
	height            int
	quitting          bool
}

// Styles for the sync picker TUI.
var syncPickerStyles = struct {
	Title     lipgloss.Style
	Help      lipgloss.Style
	Item      lipgloss.Style
	Selected  lipgloss.Style
	Disabled  lipgloss.Style
	Status    lipgloss.Style
	Highlight lipgloss.Style
	Summary   lipgloss.Style
}{
	Title:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Item:      lipgloss.NewStyle().Padding(0, 2),
	Selected:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Padding(0, 2),
	Disabled:  lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 2),
	Status:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Highlight: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")),
	Summary:   lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 2),
}

// NewSyncPickerModel creates a new sync picker model.
func NewSyncPickerModel() SyncPickerModel {
	m := SyncPickerModel{
		platforms:      model.AllPlatforms(),
		targetScopes:   []model.SkillScope{model.ScopeRepo, model.ScopeUser},
		keys:           defaultSyncPickerKeyMap(),
		phase:          syncPickerPhaseSourcePlatform,
		sourceSelected: make(map[model.SkillScope]bool),
	}
	m.sourceScopes = make([]sourceScopeOption, 0, len(model.SourceScopeOrder()))
	for _, scope := range model.SourceScopeOrder() {
		m.sourceScopes = append(m.sourceScopes, sourceScopeOption{label: string(scope), scope: scope})
	}
	m.setAllSourceScopes(true)
	return m
}

// Init implements tea.Model.
func (m SyncPickerModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m SyncPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.cursor < m.itemCount()-1 {
				m.cursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.Back):
			if m.phase == syncPickerPhaseSourcePlatform {
				m.quitting = true
				return m, tea.Quit
			}
			m.stepBack()
			return m, nil
		case m.phase == syncPickerPhaseSourceScope && (msg.Type == tea.KeySpace || msg.Type == tea.KeyTab):
			m.toggleSourceScope(m.sourceScopes[m.cursor].scope)
			return m, nil
		case m.phase == syncPickerPhaseSourceScope && key.Matches(msg, m.keys.ToggleAll):
			m.setAllSourceScopes(true)
			return m, nil
		case m.phase == syncPickerPhaseSourceScope && msg.Type == tea.KeyEnter:
			if m.selectedSourceScopeCount() == 0 {
				return m, nil
			}
			return m.stepForward()
		case msg.Type == tea.KeyEnter:
			return m.stepForward()
		}
	}

	return m, nil
}

func (m SyncPickerModel) itemCount() int {
	switch m.phase {
	case syncPickerPhaseSourcePlatform:
		return len(m.platforms)
	case syncPickerPhaseSourceScope:
		return len(m.sourceScopes)
	case syncPickerPhaseTargetPlatform:
		return len(m.platforms)
	case syncPickerPhaseTargetScope:
		return len(m.targetScopes)
	default:
		return 0
	}
}

func (m *SyncPickerModel) stepBack() {
	switch m.phase {
	case syncPickerPhaseSourceScope:
		m.phase = syncPickerPhaseSourcePlatform
		m.cursor = 0
		for i, p := range m.platforms {
			if p == m.source {
				m.cursor = i
				break
			}
		}
	case syncPickerPhaseTargetPlatform:
		m.phase = syncPickerPhaseSourceScope
		m.cursor = 0
	case syncPickerPhaseTargetScope:
		m.phase = syncPickerPhaseTargetPlatform
		m.cursor = 0
		for i, p := range m.platforms {
			if p == m.target {
				m.cursor = i
				break
			}
		}
	}
}

func (m SyncPickerModel) stepForward() (tea.Model, tea.Cmd) {
	switch m.phase {
	case syncPickerPhaseSourcePlatform:
		m.source = m.platforms[m.cursor]
		m.phase = syncPickerPhaseSourceScope
		m.cursor = 0
		m.setAllSourceScopes(true)
		return m, nil
	case syncPickerPhaseSourceScope:
		m.phase = syncPickerPhaseTargetPlatform
		m.cursor = 0
		for i, p := range m.platforms {
			if p != m.source {
				m.cursor = i
				break
			}
		}
		return m, nil
	case syncPickerPhaseTargetPlatform:
		selected := m.platforms[m.cursor]
		if selected == m.source {
			return m, nil
		}
		m.target = selected
		m.phase = syncPickerPhaseTargetScope
		m.cursor = 0
		return m, nil
	case syncPickerPhaseTargetScope:
		m.targetScopeChoice = m.cursor
		sourceScopes := m.selectedSourceScopes()
		m.result = SyncPickerResult{
			Action:       SyncPickerActionSelect,
			Source:       m.source,
			SourceScopes: sourceScopes,
			Target:       m.target,
			TargetScope:  m.targetScopes[m.targetScopeChoice],
		}
		m.quitting = true
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m SyncPickerModel) selectedSourceScopes() []model.SkillScope {
	scopes := make([]model.SkillScope, 0, len(m.sourceScopes))
	for _, option := range m.sourceScopes {
		if m.sourceSelected[option.scope] {
			scopes = append(scopes, option.scope)
		}
	}
	return model.NormalizeSourceScopes(scopes)
}

func (m *SyncPickerModel) setAllSourceScopes(selected bool) {
	for _, option := range m.sourceScopes {
		m.sourceSelected[option.scope] = selected
	}
}

func (m *SyncPickerModel) toggleSourceScope(scope model.SkillScope) {
	if m.sourceSelected[scope] && m.selectedSourceScopeCount() == 1 {
		return
	}
	m.sourceSelected[scope] = !m.sourceSelected[scope]
}

func (m SyncPickerModel) selectedSourceScopeCount() int {
	count := 0
	for _, option := range m.sourceScopes {
		if m.sourceSelected[option.scope] {
			count++
		}
	}
	return count
}

// View implements tea.Model.
func (m SyncPickerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(syncPickerStyles.Title.Render(m.phaseTitle()))
	b.WriteString("\n\n")

	summary := m.selectionSummary()
	if summary != "" {
		b.WriteString(syncPickerStyles.Summary.Render(summary))
		b.WriteString("\n\n")
	}

	for i := 0; i < m.itemCount(); i++ {
		line, disabled := m.itemLine(i)
		if i == m.cursor {
			if disabled {
				b.WriteString(syncPickerStyles.Disabled.Render("> " + line))
			} else {
				b.WriteString(syncPickerStyles.Selected.Render("> " + line))
			}
		} else {
			if disabled {
				b.WriteString(syncPickerStyles.Disabled.Render("  " + line))
			} else {
				b.WriteString(syncPickerStyles.Item.Render("  " + line))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(syncPickerStyles.Status.Render(m.phaseStatus()))
	b.WriteString("\n")

	if m.showHelp {
		b.WriteString("\n")
		b.WriteString(syncPickerStyles.Help.Render(`Navigation:
  ↑/k      Move up
  ↓/j      Move down

Source scope selection:
  space    Toggle selected scope
  a        Select all scopes
  enter    Continue to target selection

Actions:
  Esc      Go back

General:
  ?        Toggle full help
  q        Quit`))
	} else {
		keys := []string{"↑/↓ navigate", "space toggle", "a all", "enter next", "esc back", "? help", "q quit"}
		b.WriteString(syncPickerStyles.Help.Render(strings.Join(keys, " • ")))
	}

	return b.String()
}

func (m SyncPickerModel) phaseTitle() string {
	switch m.phase {
	case syncPickerPhaseSourcePlatform:
		return "🔄 Sync Skills - Select Source Platform"
	case syncPickerPhaseSourceScope:
		return "🔄 Sync Skills - Select Source Scope"
	case syncPickerPhaseTargetPlatform:
		return "🔄 Sync Skills - Select Target Platform"
	case syncPickerPhaseTargetScope:
		return "🔄 Sync Skills - Select Target Scope"
	default:
		return "🔄 Sync Skills"
	}
}

func (m SyncPickerModel) phaseStatus() string {
	switch m.phase {
	case syncPickerPhaseSourcePlatform:
		return "Choose where to sync FROM"
	case syncPickerPhaseSourceScope:
		return "Choose source scope(s): space toggles, a selects all"
	case syncPickerPhaseTargetPlatform:
		return "Choose where to sync TO"
	case syncPickerPhaseTargetScope:
		return "Choose target write scope (repo or user)"
	default:
		return ""
	}
}

func (m SyncPickerModel) selectionSummary() string {
	parts := make([]string, 0, 4)
	if m.source != "" {
		parts = append(parts, fmt.Sprintf("Source: %s", syncPickerStyles.Highlight.Render(string(m.source))))
	}
	if m.phase > syncPickerPhaseSourceScope {
		scopes := m.selectedSourceScopes()
		parts = append(parts, fmt.Sprintf("Source scopes: %s", syncPickerStyles.Highlight.Render(model.FormatSourceScopes(scopes))))
	}
	if m.target != "" {
		parts = append(parts, fmt.Sprintf("Target: %s", syncPickerStyles.Highlight.Render(string(m.target))))
	}
	if m.phase > syncPickerPhaseTargetScope || (m.phase == syncPickerPhaseTargetScope && m.targetScopeChoice >= 0 && m.targetScopeChoice < len(m.targetScopes)) {
		parts = append(parts, fmt.Sprintf("Target scope: %s", syncPickerStyles.Highlight.Render(string(m.targetScopes[m.targetScopeChoice]))))
	}

	return strings.Join(parts, "  |  ")
}

func (m SyncPickerModel) itemLine(index int) (string, bool) {
	switch m.phase {
	case syncPickerPhaseSourcePlatform:
		return string(m.platforms[index]), false
	case syncPickerPhaseSourceScope:
		option := m.sourceScopes[index]
		checkbox := "[ ]"
		if m.sourceSelected[option.scope] {
			checkbox = "[✓]"
		}
		return fmt.Sprintf("%s %s", checkbox, option.label), false
	case syncPickerPhaseTargetPlatform:
		p := m.platforms[index]
		if p == m.source {
			return fmt.Sprintf("%s (same as source)", p), true
		}
		return string(p), false
	case syncPickerPhaseTargetScope:
		return string(m.targetScopes[index]), false
	default:
		return "", false
	}
}

// Result returns the result of the user interaction.
func (m SyncPickerModel) Result() SyncPickerResult {
	return m.result
}

// RunSyncPicker runs the interactive sync picker and returns selected source/target and scopes.
func RunSyncPicker() (SyncPickerResult, error) {
	model := NewSyncPickerModel()
	finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return SyncPickerResult{}, err
	}

	if m, ok := finalModel.(SyncPickerModel); ok {
		return m.Result(), nil
	}

	return SyncPickerResult{}, nil
}
