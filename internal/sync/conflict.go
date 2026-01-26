package sync

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
)

// ConflictType identifies the kind of conflict detected.
type ConflictType string

const (
	// ConflictTypeContent indicates the content differs between source and target.
	ConflictTypeContent ConflictType = "content"

	// ConflictTypeMetadata indicates metadata differs but content is the same.
	ConflictTypeMetadata ConflictType = "metadata"

	// ConflictTypeBoth indicates both content and metadata differ.
	ConflictTypeBoth ConflictType = "both"
)

// ResolutionChoice represents how a conflict should be resolved.
type ResolutionChoice string

const (
	// ResolutionUseSource uses the source version, discarding target changes.
	ResolutionUseSource ResolutionChoice = "source"

	// ResolutionUseTarget keeps the target version, discarding source changes.
	ResolutionUseTarget ResolutionChoice = "target"

	// ResolutionMerge attempts to merge both versions.
	ResolutionMerge ResolutionChoice = "merge"

	// ResolutionSkip skips this skill entirely.
	ResolutionSkip ResolutionChoice = "skip"
)

// Conflict represents a detected conflict between source and target skills.
type Conflict struct {
	// SkillName is the name of the conflicting skill.
	SkillName string

	// Type identifies the kind of conflict.
	Type ConflictType

	// Source is the source skill version.
	Source model.Skill

	// Target is the target skill version.
	Target model.Skill

	// SourceLines contains the source content split into lines.
	SourceLines []string

	// TargetLines contains the target content split into lines.
	TargetLines []string

	// Hunks contains the detected diff hunks showing specific changes.
	Hunks []DiffHunk

	// Resolution holds the chosen resolution (if any).
	Resolution ResolutionChoice

	// ResolvedContent holds the final merged content after resolution.
	ResolvedContent string
}

// DiffHunk represents a contiguous block of changes in a diff.
type DiffHunk struct {
	// SourceStart is the starting line number in the source.
	SourceStart int

	// SourceCount is the number of lines from source.
	SourceCount int

	// TargetStart is the starting line number in the target.
	TargetStart int

	// TargetCount is the number of lines from target.
	TargetCount int

	// Lines contains the diff lines with prefixes (+, -, space).
	Lines []DiffLine
}

// DiffLine represents a single line in a diff.
type DiffLine struct {
	// Type indicates if this line is added, removed, or unchanged.
	Type DiffLineType

	// Content is the actual line content.
	Content string
}

// DiffLineType indicates the type of a diff line.
type DiffLineType string

const (
	// DiffLineContext is an unchanged line (context).
	DiffLineContext DiffLineType = " "

	// DiffLineAdded is a line added in the target.
	DiffLineAdded DiffLineType = "+"

	// DiffLineRemoved is a line removed from source.
	DiffLineRemoved DiffLineType = "-"
)

// String returns a human-readable representation of the diff line.
func (dl DiffLine) String() string {
	return string(dl.Type) + dl.Content
}

// HasConflicts returns true if there are unresolved conflicts.
func (c *Conflict) HasConflicts() bool {
	return c.Resolution == ""
}

// Summary returns a brief description of the conflict.
func (c *Conflict) Summary() string {
	var desc string
	switch c.Type {
	case ConflictTypeContent:
		desc = "content differs"
	case ConflictTypeMetadata:
		desc = "metadata differs"
	case ConflictTypeBoth:
		desc = "content and metadata differ"
	}
	return fmt.Sprintf("%s: %s", c.SkillName, desc)
}

// DiffSummary returns a summary of the changes.
func (c *Conflict) DiffSummary() string {
	var sb strings.Builder

	added := 0
	removed := 0
	for _, hunk := range c.Hunks {
		for _, line := range hunk.Lines {
			switch line.Type {
			case DiffLineAdded:
				added++
			case DiffLineRemoved:
				removed++
			}
		}
	}

	sb.WriteString(fmt.Sprintf("%d hunk(s), ", len(c.Hunks)))
	sb.WriteString(fmt.Sprintf("+%d/-%d lines", added, removed))

	return sb.String()
}

// ConflictDetector detects conflicts between source and target skills.
type ConflictDetector struct{}

// NewConflictDetector creates a new conflict detector.
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{}
}

// DetectConflict checks if there's a conflict between source and target skills.
func (cd *ConflictDetector) DetectConflict(source, target model.Skill) *Conflict {
	logging.Debug("checking for conflicts",
		logging.Skill(source.Name),
		logging.Operation("conflict_detection"),
	)

	contentDiffers := source.Content != target.Content
	metadataDiffers := cd.metadataDiffers(source, target)

	if !contentDiffers && !metadataDiffers {
		logging.Debug("no conflict detected",
			logging.Skill(source.Name),
		)
		return nil // No conflict
	}

	conflict := &Conflict{
		SkillName:   source.Name,
		Source:      source,
		Target:      target,
		SourceLines: strings.Split(source.Content, "\n"),
		TargetLines: strings.Split(target.Content, "\n"),
	}

	if contentDiffers && metadataDiffers {
		conflict.Type = ConflictTypeBoth
	} else if contentDiffers {
		conflict.Type = ConflictTypeContent
	} else {
		conflict.Type = ConflictTypeMetadata
	}

	logging.Debug("conflict detected",
		logging.Skill(source.Name),
		slog.String("conflict_type", string(conflict.Type)),
		slog.Bool("content_differs", contentDiffers),
		slog.Bool("metadata_differs", metadataDiffers),
	)

	// Compute diff hunks for content conflicts
	if contentDiffers {
		conflict.Hunks = cd.computeDiff(conflict.SourceLines, conflict.TargetLines)
		logging.Debug("computed diff hunks",
			logging.Skill(source.Name),
			logging.Count(len(conflict.Hunks)),
		)
	}

	return conflict
}

// metadataDiffers checks if metadata differs between source and target.
func (cd *ConflictDetector) metadataDiffers(source, target model.Skill) bool {
	if source.Description != target.Description {
		return true
	}

	if len(source.Tools) != len(target.Tools) {
		return true
	}

	for i, tool := range source.Tools {
		if i >= len(target.Tools) || tool != target.Tools[i] {
			return true
		}
	}

	if len(source.Metadata) != len(target.Metadata) {
		return true
	}

	for key, val := range source.Metadata {
		if targetVal, ok := target.Metadata[key]; !ok || val != targetVal {
			return true
		}
	}

	return false
}

// computeDiff computes the diff hunks between source and target lines.
// This implements a simplified diff algorithm based on longest common subsequence.
func (cd *ConflictDetector) computeDiff(source, target []string) []DiffHunk {
	// Find the longest common subsequence to guide the diff
	lcs := cd.longestCommonSubsequence(source, target)

	var hunks []DiffHunk
	var currentHunk *DiffHunk

	sourceIdx, targetIdx, lcsIdx := 0, 0, 0

	for sourceIdx < len(source) || targetIdx < len(target) {
		// Check if current lines are in the LCS
		inLCS := lcsIdx < len(lcs) &&
			sourceIdx < len(source) &&
			targetIdx < len(target) &&
			source[sourceIdx] == lcs[lcsIdx] &&
			target[targetIdx] == lcs[lcsIdx]

		if inLCS {
			// Common line - add as context or close hunk
			if currentHunk != nil {
				// Add context line to close the hunk
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    DiffLineContext,
					Content: source[sourceIdx],
				})
				hunks = append(hunks, *currentHunk)
				currentHunk = nil
			}
			sourceIdx++
			targetIdx++
			lcsIdx++
		} else {
			// Different lines - start or continue a hunk
			if currentHunk == nil {
				currentHunk = &DiffHunk{
					SourceStart: sourceIdx + 1,
					TargetStart: targetIdx + 1,
				}
			}

			// Check if source line is not in common
			if sourceIdx < len(source) && (lcsIdx >= len(lcs) || source[sourceIdx] != lcs[lcsIdx]) {
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    DiffLineRemoved,
					Content: source[sourceIdx],
				})
				currentHunk.SourceCount++
				sourceIdx++
			}

			// Check if target line is not in common
			if targetIdx < len(target) && (lcsIdx >= len(lcs) || target[targetIdx] != lcs[lcsIdx]) {
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    DiffLineAdded,
					Content: target[targetIdx],
				})
				currentHunk.TargetCount++
				targetIdx++
			}
		}
	}

	// Don't forget the last hunk
	if currentHunk != nil {
		hunks = append(hunks, *currentHunk)
	}

	return hunks
}

// longestCommonSubsequence finds the LCS of two string slices.
func (cd *ConflictDetector) longestCommonSubsequence(source, target []string) []string {
	m, n := len(source), len(target)
	if m == 0 || n == 0 {
		return nil
	}

	// Build LCS length table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if source[i-1] == target[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to find the LCS
	lcs := make([]string, dp[m][n])
	i, j, idx := m, n, dp[m][n]-1

	for i > 0 && j > 0 {
		if source[i-1] == target[j-1] {
			lcs[idx] = source[i-1]
			i--
			j--
			idx--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}
