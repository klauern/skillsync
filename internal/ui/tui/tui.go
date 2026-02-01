// Package tui provides interactive terminal UI components using BubbleTea.
package tui

import (
	"strings"
	"unicode/utf8"

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

func wrapTextLines(text string, width int) []string {
	if width <= 0 {
		return []string{""}
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""}
	}

	words := strings.Fields(text)
	lines := make([]string, 0, len(words))
	line := ""

	addLine := func(value string) {
		if value == "" {
			return
		}
		lines = append(lines, value)
	}

	for _, word := range words {
		wordWidth := runewidth.StringWidth(word)
		if wordWidth > width {
			if line != "" {
				addLine(line)
				line = ""
			}
			remaining := word
			for remaining != "" {
				segment, rest := splitWordByWidth(remaining, width)
				if segment == "" {
					break
				}
				addLine(segment)
				remaining = rest
			}
			continue
		}

		if line == "" {
			line = word
			continue
		}

		if runewidth.StringWidth(line)+1+wordWidth <= width {
			line = line + " " + word
			continue
		}

		addLine(line)
		line = word
	}

	if line != "" {
		addLine(line)
	}

	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}

func splitWordByWidth(word string, width int) (string, string) {
	if width <= 0 {
		return "", word
	}

	var b strings.Builder
	used := 0

	for len(word) > 0 {
		r, size := utf8.DecodeRuneInString(word)
		if r == utf8.RuneError && size == 0 {
			break
		}
		runeWidth := runewidth.RuneWidth(r)
		if used+runeWidth > width && used > 0 {
			break
		}
		b.WriteRune(r)
		used += runeWidth
		word = word[size:]
		if used == width {
			break
		}
	}

	return b.String(), word
}

func wrapLabeledText(label, value string, width int) string {
	if width <= 0 {
		return label + value
	}

	labelWidth := runewidth.StringWidth(label)
	available := width - labelWidth
	if available < 1 {
		available = 1
	}

	lines := wrapTextLines(value, available)
	if len(lines) == 0 {
		return label
	}

	var b strings.Builder
	b.WriteString(label)
	b.WriteString(lines[0])

	indent := strings.Repeat(" ", labelWidth)
	for _, line := range lines[1:] {
		b.WriteString("\n")
		b.WriteString(indent)
		b.WriteString(line)
	}

	return b.String()
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
