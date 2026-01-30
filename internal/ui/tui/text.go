package tui

import "strings"

func truncateText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(text) <= width {
		return text
	}
	if width <= 3 {
		return text[:width]
	}
	return text[:width-3] + "..."
}

func formatDescription(text string, width int) string {
	return formatDetail("Description: ", text, width)
}

func formatDetail(label, text string, width int) string {
	if width <= len(label) {
		return label + text
	}
	wrapped := wrapText(text, width-len(label))
	lines := strings.Split(wrapped, "\n")
	if len(lines) == 0 {
		return label
	}

	var b strings.Builder
	for i, line := range lines {
		if i == 0 {
			b.WriteString(label)
			b.WriteString(line)
			continue
		}
		b.WriteString("\n")
		b.WriteString(strings.Repeat(" ", len(label)))
		b.WriteString(line)
	}
	return b.String()
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	cleaned := strings.ReplaceAll(text, "\n", " ")
	words := strings.Fields(cleaned)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var line strings.Builder
	for _, word := range words {
		if line.Len() == 0 {
			line.WriteString(word)
			continue
		}
		if line.Len()+1+len(word) > width {
			lines = append(lines, line.String())
			line.Reset()
			line.WriteString(word)
			continue
		}
		line.WriteByte(' ')
		line.WriteString(word)
	}
	if line.Len() > 0 {
		lines = append(lines, line.String())
	}
	return strings.Join(lines, "\n")
}
