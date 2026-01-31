// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
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

func wrapText(text string, width, maxLines int) []string {
	if width <= 0 {
		return []string{""}
	}
	if maxLines <= 0 {
		maxLines = 1
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""}
	}

	words := strings.Fields(text)
	lines := make([]string, 0, maxLines)
	line := ""
	truncated := false

	for _, word := range words {
		wordWidth := runewidth.StringWidth(word)
		if line == "" {
			if wordWidth > width {
				lines = append(lines, runewidth.Truncate(word, width, ""))
			} else {
				line = word
			}
		} else if runewidth.StringWidth(line)+1+wordWidth <= width {
			line = line + " " + word
		} else {
			lines = append(lines, line)
			line = ""
			if wordWidth > width {
				lines = append(lines, runewidth.Truncate(word, width, ""))
			} else {
				line = word
			}
		}

		if len(lines) >= maxLines {
			truncated = true
			line = ""
			break
		}
	}

	if len(lines) < maxLines && line != "" {
		lines = append(lines, line)
	} else if len(lines) >= maxLines && line != "" {
		truncated = true
	}

	if truncated && len(lines) > 0 {
		if width >= 3 {
			last := runewidth.Truncate(lines[len(lines)-1], width-3, "")
			lines[len(lines)-1] = last + "..."
		} else {
			lines[len(lines)-1] = runewidth.Truncate(lines[len(lines)-1], width, "")
		}
	}

	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	return lines
}

func padLines(lines []string, count int) []string {
	if count <= 0 {
		return lines
	}
	if len(lines) >= count {
		return lines
	}
	padded := make([]string, count)
	copy(padded, lines)
	for i := len(lines); i < count; i++ {
		padded[i] = ""
	}
	return padded
}
