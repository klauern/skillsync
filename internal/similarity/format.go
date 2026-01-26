// Package similarity provides algorithms for finding similar skills by name or content.
package similarity

import (
	"fmt"
	"io"
	"strings"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/sync"
)

// OutputFormat specifies the format for comparison output.
type OutputFormat string

const (
	// FormatUnified shows a unified diff format (default).
	FormatUnified OutputFormat = "unified"

	// FormatSideBySide shows side-by-side comparison.
	FormatSideBySide OutputFormat = "side-by-side"

	// FormatSummary shows only summary statistics without full diff.
	FormatSummary OutputFormat = "summary"
)

// ComparisonResult holds a similarity match along with diff information.
type ComparisonResult struct {
	// Skill1 is the first skill being compared.
	Skill1 model.Skill `json:"skill1"`

	// Skill2 is the second skill being compared.
	Skill2 model.Skill `json:"skill2"`

	// NameScore is the name similarity score (0.0-1.0), if computed.
	NameScore float64 `json:"name_score,omitempty"`

	// ContentScore is the content similarity score (0.0-1.0), if computed.
	ContentScore float64 `json:"content_score,omitempty"`

	// Hunks contains the diff hunks showing specific changes.
	Hunks []sync.DiffHunk `json:"hunks,omitempty"`

	// LinesAdded is the count of lines added in Skill2.
	LinesAdded int `json:"lines_added"`

	// LinesRemoved is the count of lines removed from Skill1.
	LinesRemoved int `json:"lines_removed"`
}

// FormatterConfig configures the diff output formatter.
type FormatterConfig struct {
	// Format specifies the output format (unified, side-by-side, summary).
	Format OutputFormat

	// ContextLines is the number of context lines to show around changes.
	// Default: 3
	ContextLines int

	// MaxWidth is the maximum width for side-by-side output.
	// Default: 80 (each side gets half)
	MaxWidth int

	// ShowLineNumbers enables line number display.
	// Default: true
	ShowLineNumbers bool

	// TruncateAt limits the number of hunks displayed.
	// 0 means no limit.
	TruncateAt int
}

// DefaultFormatterConfig returns sensible defaults for formatting.
func DefaultFormatterConfig() FormatterConfig {
	return FormatterConfig{
		Format:          FormatUnified,
		ContextLines:    3,
		MaxWidth:        80,
		ShowLineNumbers: true,
		TruncateAt:      0,
	}
}

// Formatter formats comparison results for display.
type Formatter struct {
	config FormatterConfig
}

// NewFormatter creates a new formatter with the given configuration.
func NewFormatter(config FormatterConfig) *Formatter {
	if config.ContextLines < 0 {
		config.ContextLines = 3
	}
	if config.MaxWidth <= 0 {
		config.MaxWidth = 80
	}
	if config.Format == "" {
		config.Format = FormatUnified
	}
	return &Formatter{config: config}
}

// Format writes the formatted comparison result to the writer.
func (f *Formatter) Format(w io.Writer, result *ComparisonResult) error {
	switch f.config.Format {
	case FormatUnified:
		return f.formatUnified(w, result)
	case FormatSideBySide:
		return f.formatSideBySide(w, result)
	case FormatSummary:
		return f.formatSummary(w, result)
	default:
		return f.formatUnified(w, result)
	}
}

// FormatMultiple writes multiple comparison results to the writer.
func (f *Formatter) FormatMultiple(w io.Writer, results []*ComparisonResult) error {
	for i, result := range results {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w, strings.Repeat("=", 60)); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if err := f.Format(w, result); err != nil {
			return err
		}
	}
	return nil
}

// formatUnified outputs in unified diff format.
func (f *Formatter) formatUnified(w io.Writer, result *ComparisonResult) error {
	// Header
	if err := f.writeHeader(w, result); err != nil {
		return err
	}

	// Diff header
	if _, err := fmt.Fprintf(w, "--- %s (%s)\n", result.Skill1.Name, result.Skill1.Platform); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "+++ %s (%s)\n", result.Skill2.Name, result.Skill2.Platform); err != nil {
		return err
	}

	// Hunks
	hunksToShow := result.Hunks
	truncated := false
	if f.config.TruncateAt > 0 && len(hunksToShow) > f.config.TruncateAt {
		hunksToShow = hunksToShow[:f.config.TruncateAt]
		truncated = true
	}

	for _, hunk := range hunksToShow {
		// Hunk header
		if _, err := fmt.Fprintf(w, "@@ -%d,%d +%d,%d @@\n",
			hunk.SourceStart, hunk.SourceCount,
			hunk.TargetStart, hunk.TargetCount); err != nil {
			return err
		}

		// Hunk lines
		for _, line := range hunk.Lines {
			if _, err := fmt.Fprintln(w, line.String()); err != nil {
				return err
			}
		}
	}

	if truncated {
		if _, err := fmt.Fprintf(w, "\n... (%d more hunks not shown)\n", len(result.Hunks)-f.config.TruncateAt); err != nil {
			return err
		}
	}

	return nil
}

// formatSideBySide outputs in side-by-side format.
func (f *Formatter) formatSideBySide(w io.Writer, result *ComparisonResult) error {
	// Header
	if err := f.writeHeader(w, result); err != nil {
		return err
	}

	// Calculate column widths
	halfWidth := max((f.config.MaxWidth-3)/2, 20) // -3 for " | " separator, min 20

	// Column headers
	leftHeader := truncateString(result.Skill1.Name, halfWidth)
	rightHeader := truncateString(result.Skill2.Name, halfWidth)
	if _, err := fmt.Fprintf(w, "%-*s | %s\n", halfWidth, leftHeader, rightHeader); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s-+-%s\n", strings.Repeat("-", halfWidth), strings.Repeat("-", halfWidth)); err != nil {
		return err
	}

	// Split content into lines
	lines1 := strings.Split(result.Skill1.Content, "\n")
	lines2 := strings.Split(result.Skill2.Content, "\n")

	// Build a map of changes from hunks for highlighting
	removedLines := make(map[int]bool)
	addedLines := make(map[int]bool)
	for _, hunk := range result.Hunks {
		srcLine := hunk.SourceStart - 1 // Convert to 0-indexed
		tgtLine := hunk.TargetStart - 1
		for _, line := range hunk.Lines {
			switch line.Type {
			case sync.DiffLineRemoved:
				removedLines[srcLine] = true
				srcLine++
			case sync.DiffLineAdded:
				addedLines[tgtLine] = true
				tgtLine++
			case sync.DiffLineContext:
				srcLine++
				tgtLine++
			}
		}
	}

	// Display side-by-side
	maxLines := max(len(lines1), len(lines2))
	lineNumWidth := len(fmt.Sprintf("%d", maxLines))
	contentWidth := halfWidth - lineNumWidth - 3 // -3 for line num padding and marker

	for i := range maxLines {
		var left, right string
		var leftMarker, rightMarker string

		if i < len(lines1) {
			left = truncateString(lines1[i], contentWidth)
			if removedLines[i] {
				leftMarker = "-"
			} else {
				leftMarker = " "
			}
		}

		if i < len(lines2) {
			right = truncateString(lines2[i], contentWidth)
			if addedLines[i] {
				rightMarker = "+"
			} else {
				rightMarker = " "
			}
		}

		var err error
		if f.config.ShowLineNumbers {
			leftNum := ""
			rightNum := ""
			if i < len(lines1) {
				leftNum = fmt.Sprintf("%*d", lineNumWidth, i+1)
			} else {
				leftNum = strings.Repeat(" ", lineNumWidth)
			}
			if i < len(lines2) {
				rightNum = fmt.Sprintf("%*d", lineNumWidth, i+1)
			} else {
				rightNum = strings.Repeat(" ", lineNumWidth)
			}
			_, err = fmt.Fprintf(w, "%s%s %-*s | %s%s %s\n",
				leftNum, leftMarker, contentWidth, left,
				rightNum, rightMarker, right)
		} else {
			_, err = fmt.Fprintf(w, "%s%-*s | %s%s\n",
				leftMarker, halfWidth-1, left,
				rightMarker, right)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// formatSummary outputs only summary statistics.
func (f *Formatter) formatSummary(w io.Writer, result *ComparisonResult) error {
	if err := f.writeHeader(w, result); err != nil {
		return err
	}

	// Statistics
	if _, err := fmt.Fprintf(w, "%-20s %s\n", "Hunks:", fmt.Sprintf("%d", len(result.Hunks))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%-20s %s\n", "Lines added:", fmt.Sprintf("+%d", result.LinesAdded)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%-20s %s\n", "Lines removed:", fmt.Sprintf("-%d", result.LinesRemoved)); err != nil {
		return err
	}

	// Show hunk locations
	if len(result.Hunks) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "Change locations:"); err != nil {
			return err
		}
		for i, hunk := range result.Hunks {
			if f.config.TruncateAt > 0 && i >= f.config.TruncateAt {
				if _, err := fmt.Fprintf(w, "  ... (%d more hunks)\n", len(result.Hunks)-i); err != nil {
					return err
				}
				break
			}
			if _, err := fmt.Fprintf(w, "  @@ -%d,%d +%d,%d @@\n",
				hunk.SourceStart, hunk.SourceCount,
				hunk.TargetStart, hunk.TargetCount); err != nil {
				return err
			}
		}
	}

	return nil
}

// writeHeader writes the common header for all formats.
func (f *Formatter) writeHeader(w io.Writer, result *ComparisonResult) error {
	if _, err := fmt.Fprintf(w, "Comparing: %s <-> %s\n", result.Skill1.Name, result.Skill2.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, strings.Repeat("-", 50)); err != nil {
		return err
	}

	// Similarity scores
	if result.NameScore > 0 {
		if _, err := fmt.Fprintf(w, "%-20s %.1f%%\n", "Name similarity:", result.NameScore*100); err != nil {
			return err
		}
	}
	if result.ContentScore > 0 {
		if _, err := fmt.Fprintf(w, "%-20s %.1f%%\n", "Content similarity:", result.ContentScore*100); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	return nil
}

// DiffSummary returns a compact summary string for a comparison result.
func (result *ComparisonResult) DiffSummary() string {
	return fmt.Sprintf("%d hunk(s), +%d/-%d lines", len(result.Hunks), result.LinesAdded, result.LinesRemoved)
}

// truncateString truncates a string to the given width, adding "..." if needed.
func truncateString(s string, width int) string {
	if width <= 3 {
		return s
	}
	if len(s) <= width {
		return s
	}
	return s[:width-3] + "..."
}

// ComputeDiff generates diff hunks between two skills and returns a ComparisonResult.
// This bridges the similarity matching with diff output formatting.
func ComputeDiff(skill1, skill2 model.Skill, nameScore, contentScore float64) *ComparisonResult {
	detector := sync.NewConflictDetector()
	conflict := detector.DetectConflict(skill1, skill2)

	result := &ComparisonResult{
		Skill1:       skill1,
		Skill2:       skill2,
		NameScore:    nameScore,
		ContentScore: contentScore,
	}

	if conflict != nil {
		result.Hunks = conflict.Hunks

		// Count added/removed lines
		for _, hunk := range conflict.Hunks {
			for _, line := range hunk.Lines {
				switch line.Type {
				case sync.DiffLineAdded:
					result.LinesAdded++
				case sync.DiffLineRemoved:
					result.LinesRemoved++
				}
			}
		}
	}

	return result
}

// FormatComparisonTable formats multiple comparison results as a table.
func FormatComparisonTable(w io.Writer, results []*ComparisonResult) error {
	if len(results) == 0 {
		_, err := fmt.Fprintln(w, "No similar skills found.")
		return err
	}

	// Header
	if _, err := fmt.Fprintf(w, "%-25s %-25s %-8s %-8s %-15s\n",
		"SKILL 1", "SKILL 2", "NAME %", "CONTENT %", "CHANGES"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%-25s %-25s %-8s %-8s %-15s\n",
		"-------", "-------", "------", "---------", "-------"); err != nil {
		return err
	}

	// Rows
	for _, r := range results {
		name1 := truncateString(r.Skill1.Name, 25)
		name2 := truncateString(r.Skill2.Name, 25)
		nameScore := "-"
		contentScore := "-"
		if r.NameScore > 0 {
			nameScore = fmt.Sprintf("%.0f%%", r.NameScore*100)
		}
		if r.ContentScore > 0 {
			contentScore = fmt.Sprintf("%.0f%%", r.ContentScore*100)
		}
		changes := r.DiffSummary()
		if _, err := fmt.Fprintf(w, "%-25s %-25s %-8s %-8s %-15s\n",
			name1, name2, nameScore, contentScore, changes); err != nil {
			return err
		}
	}
	return nil
}
