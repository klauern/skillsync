// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles contains reusable lipgloss styles for the TUI.
var Styles = struct {
	Title    lipgloss.Style
	Selected lipgloss.Style
	Normal   lipgloss.Style
}{
	Title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
	Selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")),
	Normal:   lipgloss.NewStyle(),
}

// Run starts a BubbleTea program with the given model.
func Run(model tea.Model) (tea.Model, error) {
	p := tea.NewProgram(model)
	return p.Run()
}
