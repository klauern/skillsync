// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/klauern/skillsync/internal/model"
)

// DiffAction represents the action to perform after viewing diff.
type DiffAction int

const (
	// DiffActionNone means no action was taken (user quit).
	DiffActionNone DiffAction = iota
	// DiffActionBack means the user wants to go back to selection.
	DiffActionBack
	// DiffActionSync means the user wants to sync this skill.
	DiffActionSync
)

// SyncDiffResult contains the result of the diff viewer interaction.
type SyncDiffResult struct {
	Action DiffAction
	Skill  model.Skill
}

// syncDiffKeyMap defines the key bindings for the diff viewer.
type syncDiffKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Back   key.Binding
	Sync   key.Binding
	Help   key.Binding
	Quit   key.Binding
}

func defaultSyncDiffKeyMap() syncDiffKeyMap {
	return syncDiffKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "scroll up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "scroll down"),
		),
		Back: key.NewBinding(
			key.WithKeys("b", "esc"),
			key.WithHelp("b/esc", "back"),
		),
		Sync: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "sync this skill"),
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

// SyncDiffModel is the BubbleTea model for viewing skill diff.
type SyncDiffModel struct {
	viewport       viewport.Model
	skill          model.Skill
	targetSkill    *model.Skill // nil if target doesn't exist
	keys           syncDiffKeyMap
	result         SyncDiffResult
	showHelp       bool
	width          int
	height         int
	quitting       bool
	sourcePlatform model.Platform
	targetPlatform model.Platform
	ready          bool
}

// Styles for the diff viewer TUI.
var syncDiffStyles = struct {
	Title      lipgloss.Style
	Help       lipgloss.Style
	Status     lipgloss.Style
	Header     lipgloss.Style
	Added      lipgloss.Style
	Removed    lipgloss.Style
	Unchanged  lipgloss.Style
	SectionHdr lipgloss.Style
	Info       lipgloss.Style
}{
	Title:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:       lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Status:     lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Header:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4")),
	Added:      lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
	Removed:    lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	Unchanged:  lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
	SectionHdr: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5")).Padding(1, 0),
	Info:       lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Italic(true),
}

// NewSyncDiffModel creates a new diff viewer model.
func NewSyncDiffModel(skill model.Skill, targetSkill *model.Skill, source, target model.Platform) SyncDiffModel {
	return SyncDiffModel{
		skill:          skill,
		targetSkill:    targetSkill,
		keys:           defaultSyncDiffKeyMap(),
		sourcePlatform: source,
		targetPlatform: target,
	}
}

// Init implements tea.Model.
func (m SyncDiffModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m SyncDiffModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 4 // Title + spacing
		footerHeight := 3 // Status + help
		viewportHeight := max(msg.Height-headerHeight-footerHeight, 5)

		if !m.ready {
			m.viewport = viewport.New(msg.Width-2, viewportHeight)
			m.viewport.SetContent(m.buildDiffContent())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 2
			m.viewport.Height = viewportHeight
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.Back):
			m.result = SyncDiffResult{
				Action: DiffActionBack,
				Skill:  m.skill,
			}
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Sync):
			m.result = SyncDiffResult{
				Action: DiffActionSync,
				Skill:  m.skill,
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m SyncDiffModel) buildDiffContent() string {
	var b strings.Builder

	// Skill metadata
	b.WriteString(syncDiffStyles.SectionHdr.Render("Skill Information"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Name:        %s\n", m.skill.Name))
	b.WriteString(fmt.Sprintf("  Platform:    %s\n", m.skill.Platform))
	b.WriteString(fmt.Sprintf("  Scope:       %s\n", m.skill.DisplayScope()))
	if m.skill.Description != "" {
		b.WriteString(fmt.Sprintf("  Description: %s\n", m.skill.Description))
	}
	b.WriteString("\n")

	// Check if target exists
	if m.targetSkill == nil {
		b.WriteString(syncDiffStyles.Info.Render("  This is a NEW skill - will be created in target"))
		b.WriteString("\n\n")

		// Show source content
		b.WriteString(syncDiffStyles.SectionHdr.Render(fmt.Sprintf("Source Content (%s)", m.sourcePlatform)))
		b.WriteString("\n")
		b.WriteString(formatContentWithLineNumbers(m.skill.Content, syncDiffStyles.Added))
	} else {
		// Show diff between source and target
		b.WriteString(syncDiffStyles.Info.Render("  Skill exists in target - showing comparison"))
		b.WriteString("\n\n")

		// Source content
		b.WriteString(syncDiffStyles.SectionHdr.Render(fmt.Sprintf("Source (%s)", m.sourcePlatform)))
		b.WriteString("\n")
		b.WriteString(formatContentWithLineNumbers(m.skill.Content, syncDiffStyles.Added))
		b.WriteString("\n")

		// Target content
		b.WriteString(syncDiffStyles.SectionHdr.Render(fmt.Sprintf("Target (%s) - Current", m.targetPlatform)))
		b.WriteString("\n")
		b.WriteString(formatContentWithLineNumbers(m.targetSkill.Content, syncDiffStyles.Removed))

		// Show simple diff summary
		if m.skill.Content == m.targetSkill.Content {
			b.WriteString("\n")
			b.WriteString(syncDiffStyles.Info.Render("  Contents are identical - no changes needed"))
		} else {
			b.WriteString("\n")
			srcLines := strings.Count(m.skill.Content, "\n") + 1
			tgtLines := strings.Count(m.targetSkill.Content, "\n") + 1
			b.WriteString(syncDiffStyles.Info.Render(fmt.Sprintf("  Source: %d lines, Target: %d lines", srcLines, tgtLines)))
		}
	}

	return b.String()
}

func formatContentWithLineNumbers(content string, style lipgloss.Style) string {
	lines := strings.Split(content, "\n")
	var b strings.Builder

	for i, line := range lines {
		lineNum := fmt.Sprintf("%4d │ ", i+1)
		b.WriteString(syncDiffStyles.Unchanged.Render(lineNum))
		b.WriteString(style.Render(line))
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// View implements tea.Model.
func (m SyncDiffModel) View() string {
	if m.quitting {
		return ""
	}

	if !m.ready {
		return "Loading..."
	}

	var b strings.Builder

	// Title
	title := syncDiffStyles.Title.Render(fmt.Sprintf("📄 Preview: %s", m.skill.Name))
	b.WriteString(title)
	b.WriteString("\n\n")

	// Viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Status bar
	scrollPercent := int(m.viewport.ScrollPercent() * 100)
	status := fmt.Sprintf("Scroll: %d%% • %s → %s", scrollPercent, m.sourcePlatform, m.targetPlatform)
	b.WriteString(syncDiffStyles.Status.Render(status))
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

func (m SyncDiffModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ scroll",
		"b back",
		"y sync",
		"? help",
		"q quit",
	}
	return syncDiffStyles.Help.Render(strings.Join(keys, " • "))
}

func (m SyncDiffModel) renderFullHelp() string {
	help := `Navigation:
  ↑/k      Scroll up
  ↓/j      Scroll down
  PgUp     Page up
  PgDown   Page down

Actions:
  b/Esc    Go back to skill list
  y        Sync this skill

General:
  ?        Toggle full help
  q        Quit`
	return syncDiffStyles.Help.Render(help)
}

// Result returns the result of the user interaction.
func (m SyncDiffModel) Result() SyncDiffResult {
	return m.result
}

// RunSyncDiff runs the interactive diff viewer and returns the result.
func RunSyncDiff(skill model.Skill, targetSkill *model.Skill, source, target model.Platform) (SyncDiffResult, error) {
	mdl := NewSyncDiffModel(skill, targetSkill, source, target)
	finalModel, err := tea.NewProgram(mdl, tea.WithAltScreen()).Run()
	if err != nil {
		return SyncDiffResult{}, err
	}

	if m, ok := finalModel.(SyncDiffModel); ok {
		return m.Result(), nil
	}

	return SyncDiffResult{}, nil
}
