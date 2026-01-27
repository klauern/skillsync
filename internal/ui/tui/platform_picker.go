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

// PlatformPickerAction represents the action to perform after platform selection.
type PlatformPickerAction int

const (
	// PlatformPickerActionNone means no action was taken (user quit).
	PlatformPickerActionNone PlatformPickerAction = iota
	// PlatformPickerActionSelect means the user selected platforms.
	PlatformPickerActionSelect
)

// PlatformPickerResult contains the result of the platform picker TUI interaction.
type PlatformPickerResult struct {
	Action PlatformPickerAction
	Source model.Platform
	Target model.Platform
}

// platformPickerPhase represents the current phase of platform selection.
type platformPickerPhase int

const (
	phaseSourcePlatform platformPickerPhase = iota
	phaseTargetPlatform
)

// platformPickerKeyMap defines the key bindings for the platform picker.
type platformPickerKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Back   key.Binding
	Help   key.Binding
	Quit   key.Binding
}

func defaultPlatformPickerKeyMap() platformPickerKeyMap {
	return platformPickerKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter", "select"),
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

// PlatformPickerModel is the BubbleTea model for platform selection.
type PlatformPickerModel struct {
	platforms []model.Platform
	cursor    int
	source    model.Platform
	target    model.Platform
	phase     platformPickerPhase
	keys      platformPickerKeyMap
	result    PlatformPickerResult
	showHelp  bool
	width     int
	height    int
	quitting  bool
}

// Styles for the platform picker TUI.
var platformPickerStyles = struct {
	Title       lipgloss.Style
	Help        lipgloss.Style
	Item        lipgloss.Style
	Selected    lipgloss.Style
	Description lipgloss.Style
	Status      lipgloss.Style
	Highlight   lipgloss.Style
}{
	Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Padding(0, 1),
	Help:        lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	Item:        lipgloss.NewStyle().Padding(0, 2),
	Selected:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Padding(0, 2),
	Description: lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 4),
	Status:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1),
	Highlight:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")),
}

// NewPlatformPickerModel creates a new platform picker model.
func NewPlatformPickerModel() PlatformPickerModel {
	return PlatformPickerModel{
		platforms: model.AllPlatforms(),
		keys:      defaultPlatformPickerKeyMap(),
		phase:     phaseSourcePlatform,
	}
}

// Init implements tea.Model.
func (m PlatformPickerModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m PlatformPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.platforms)-1 {
				m.cursor++
			}
			return m, nil

		case key.Matches(msg, m.keys.Back):
			if m.phase == phaseTargetPlatform {
				m.phase = phaseSourcePlatform
				m.cursor = 0
				// Find the source platform in the list
				for i, p := range m.platforms {
					if p == m.source {
						m.cursor = i
						break
					}
				}
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Select):
			selected := m.platforms[m.cursor]

			if m.phase == phaseSourcePlatform {
				m.source = selected
				m.phase = phaseTargetPlatform
				m.cursor = 0
				// Start at a different platform if possible
				for i, p := range m.platforms {
					if p != m.source {
						m.cursor = i
						break
					}
				}
				return m, nil
			}

			// Target platform selected
			if selected == m.source {
				// Can't sync to same platform - show error in view
				return m, nil
			}

			m.target = selected
			m.result = PlatformPickerResult{
				Action: PlatformPickerActionSelect,
				Source: m.source,
				Target: m.target,
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m PlatformPickerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	var title string
	if m.phase == phaseSourcePlatform {
		title = platformPickerStyles.Title.Render("🔄 Sync Skills - Select Source Platform")
	} else {
		title = platformPickerStyles.Title.Render("🔄 Sync Skills - Select Target Platform")
	}
	b.WriteString(title)
	b.WriteString("\n\n")

	// Show selected source when picking target
	if m.phase == phaseTargetPlatform {
		sourceLabel := platformPickerStyles.Highlight.Render(string(m.source))
		b.WriteString(fmt.Sprintf("  Source: %s\n\n", sourceLabel))
	}

	// Platform list
	for i, platform := range m.platforms {
		var line string
		platformName := string(platform)

		// Check if this platform is disabled (same as source when picking target)
		disabled := m.phase == phaseTargetPlatform && platform == m.source

		if i == m.cursor {
			if disabled {
				line = platformPickerStyles.Item.Render(fmt.Sprintf("> %s (same as source)", platformName))
			} else {
				line = platformPickerStyles.Selected.Render(fmt.Sprintf("> %s", platformName))
			}
		} else {
			if disabled {
				line = platformPickerStyles.Description.Render(fmt.Sprintf("  %s (same as source)", platformName))
			} else {
				line = platformPickerStyles.Item.Render(fmt.Sprintf("  %s", platformName))
			}
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Status bar
	var status string
	if m.phase == phaseSourcePlatform {
		status = "Select the platform to sync FROM"
	} else {
		status = "Select the platform to sync TO"
	}
	b.WriteString(platformPickerStyles.Status.Render(status))
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

func (m PlatformPickerModel) renderShortHelp() string {
	keys := []string{
		"↑/↓ navigate",
		"enter select",
	}
	if m.phase == phaseTargetPlatform {
		keys = append(keys, "esc back")
	}
	keys = append(keys, "? help", "q quit")
	return platformPickerStyles.Help.Render(strings.Join(keys, " • "))
}

func (m PlatformPickerModel) renderFullHelp() string {
	help := `Navigation:
  ↑/k      Move up
  ↓/j      Move down

Actions:
  Enter    Select platform
  Esc      Go back (when selecting target)

General:
  ?        Toggle full help
  q        Quit`
	return platformPickerStyles.Help.Render(help)
}

// Result returns the result of the user interaction.
func (m PlatformPickerModel) Result() PlatformPickerResult {
	return m.result
}

// RunPlatformPicker runs the interactive platform picker and returns the result.
func RunPlatformPicker() (PlatformPickerResult, error) {
	model := NewPlatformPickerModel()
	finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return PlatformPickerResult{}, err
	}

	if m, ok := finalModel.(PlatformPickerModel); ok {
		return m.Result(), nil
	}

	return PlatformPickerResult{}, nil
}
