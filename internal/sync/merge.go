package sync

import (
	"log/slog"
	"strings"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
)

// MergeResult represents the outcome of a merge operation.
type MergeResult struct {
	// Success indicates if the merge completed without conflicts.
	Success bool

	// Content is the merged content (may contain conflict markers if not successful).
	Content string

	// HasConflictMarkers indicates if the content contains conflict markers.
	HasConflictMarkers bool

	// Conflicts contains details about any conflicts encountered.
	Conflicts []MergeConflict
}

// MergeConflict represents a specific conflict region in the merge.
type MergeConflict struct {
	// StartLine is the line number where the conflict starts.
	StartLine int

	// EndLine is the line number where the conflict ends.
	EndLine int

	// SourceContent is the content from the source version.
	SourceContent string

	// TargetContent is the content from the target version.
	TargetContent string

	// BaseContent is the content from the base version (if available).
	BaseContent string
}

// Merger handles merging of skill content.
type Merger struct {
	// ConflictMarkerStart is the marker for the start of a conflict.
	ConflictMarkerStart string

	// ConflictMarkerMiddle is the marker separating versions.
	ConflictMarkerMiddle string

	// ConflictMarkerEnd is the marker for the end of a conflict.
	ConflictMarkerEnd string
}

// NewMerger creates a new merger with default conflict markers.
func NewMerger() *Merger {
	return &Merger{
		ConflictMarkerStart:  "<<<<<<< SOURCE",
		ConflictMarkerMiddle: "=======",
		ConflictMarkerEnd:    ">>>>>>> TARGET",
	}
}

// ThreeWayMerge performs a three-way merge between source, target, and base versions.
// If base is nil, it falls back to a two-way merge.
func (m *Merger) ThreeWayMerge(source, target model.Skill, base *model.Skill) MergeResult {
	hasBase := base != nil
	logging.Debug("starting three-way merge",
		logging.Skill(source.Name),
		logging.Operation("merge"),
		slog.Bool("has_base", hasBase),
	)

	sourceLines := strings.Split(source.Content, "\n")
	targetLines := strings.Split(target.Content, "\n")

	var baseLines []string
	if hasBase {
		baseLines = strings.Split(base.Content, "\n")
	}

	result := m.mergeLines(sourceLines, targetLines, baseLines)

	logging.Debug("three-way merge completed",
		logging.Skill(source.Name),
		slog.Bool("success", result.Success),
		slog.Bool("has_conflict_markers", result.HasConflictMarkers),
		logging.Count(len(result.Conflicts)),
	)

	return result
}

// TwoWayMerge performs a two-way merge between source and target.
func (m *Merger) TwoWayMerge(source, target model.Skill) MergeResult {
	return m.ThreeWayMerge(source, target, nil)
}

// mergeLines performs the actual line-by-line merge.
func (m *Merger) mergeLines(source, target, base []string) MergeResult {
	// If no base, use a two-way diff-based merge
	if len(base) == 0 {
		return m.twoWayMerge(source, target)
	}

	// Three-way merge using base as common ancestor
	return m.threeWayMergeLines(source, target, base)
}

// twoWayMerge merges source and target without a common base.
func (m *Merger) twoWayMerge(source, target []string) MergeResult {
	logging.Debug("starting two-way merge",
		logging.Operation("merge"),
		slog.Int("source_lines", len(source)),
		slog.Int("target_lines", len(target)),
	)

	result := MergeResult{
		Success:            true,
		HasConflictMarkers: false,
		Conflicts:          make([]MergeConflict, 0),
	}

	// Find LCS to identify common sections
	lcs := m.longestCommonSubsequence(source, target)

	var merged []string
	sourceIdx, targetIdx, lcsIdx := 0, 0, 0

	for sourceIdx < len(source) || targetIdx < len(target) {
		// Check if we're at a common line
		sourceInLCS := sourceIdx < len(source) && lcsIdx < len(lcs) && source[sourceIdx] == lcs[lcsIdx]
		targetInLCS := targetIdx < len(target) && lcsIdx < len(lcs) && target[targetIdx] == lcs[lcsIdx]

		if sourceInLCS && targetInLCS {
			// Common line - add it
			merged = append(merged, source[sourceIdx])
			sourceIdx++
			targetIdx++
			lcsIdx++
		} else {
			// Collect differing sections
			var sourceSection, targetSection []string

			for sourceIdx < len(source) && (lcsIdx >= len(lcs) || source[sourceIdx] != lcs[lcsIdx]) {
				sourceSection = append(sourceSection, source[sourceIdx])
				sourceIdx++
			}

			for targetIdx < len(target) && (lcsIdx >= len(lcs) || target[targetIdx] != lcs[lcsIdx]) {
				targetSection = append(targetSection, target[targetIdx])
				targetIdx++
			}

			// If sections differ, mark as conflict
			if len(sourceSection) > 0 && len(targetSection) > 0 {
				// True conflict - both have changes
				result.Success = false
				result.HasConflictMarkers = true
				conflict := MergeConflict{
					StartLine:     len(merged) + 1,
					SourceContent: strings.Join(sourceSection, "\n"),
					TargetContent: strings.Join(targetSection, "\n"),
				}

				merged = append(merged, m.ConflictMarkerStart)
				merged = append(merged, sourceSection...)
				merged = append(merged, m.ConflictMarkerMiddle)
				merged = append(merged, targetSection...)
				merged = append(merged, m.ConflictMarkerEnd)

				conflict.EndLine = len(merged)
				result.Conflicts = append(result.Conflicts, conflict)
			} else if len(sourceSection) > 0 {
				// Only source has additions
				merged = append(merged, sourceSection...)
			} else if len(targetSection) > 0 {
				// Only target has additions
				merged = append(merged, targetSection...)
			}
		}
	}

	result.Content = strings.Join(merged, "\n")
	return result
}

// threeWayMergeLines performs a three-way merge using base as common ancestor.
func (m *Merger) threeWayMergeLines(source, target, base []string) MergeResult {
	result := MergeResult{
		Success:            true,
		HasConflictMarkers: false,
		Conflicts:          make([]MergeConflict, 0),
	}

	// Find LCS of source-base and target-base
	sourceChanges := m.findChanges(base, source)
	targetChanges := m.findChanges(base, target)

	// Merge changes
	merged := m.applyChanges(base, sourceChanges, targetChanges, &result)

	result.Content = strings.Join(merged, "\n")
	return result
}

// Change represents a modification from base to a derived version.
type Change struct {
	// Type is "add", "delete", or "modify"
	Type string

	// BaseStart is the starting line in base (0-indexed)
	BaseStart int

	// BaseEnd is the ending line in base (exclusive)
	BaseEnd int

	// NewLines are the replacement lines
	NewLines []string
}

// findChanges identifies changes between base and derived version.
func (m *Merger) findChanges(base, derived []string) []Change {
	var changes []Change

	lcs := m.longestCommonSubsequence(base, derived)

	baseIdx, derivedIdx, lcsIdx := 0, 0, 0

	for baseIdx < len(base) || derivedIdx < len(derived) {
		baseInLCS := baseIdx < len(base) && lcsIdx < len(lcs) && base[baseIdx] == lcs[lcsIdx]
		derivedInLCS := derivedIdx < len(derived) && lcsIdx < len(lcs) && derived[derivedIdx] == lcs[lcsIdx]

		if baseInLCS && derivedInLCS {
			// Common line
			baseIdx++
			derivedIdx++
			lcsIdx++
		} else {
			// Collect change
			changeStart := baseIdx
			var newLines []string

			// Collect lines removed from base
			for baseIdx < len(base) && (lcsIdx >= len(lcs) || base[baseIdx] != lcs[lcsIdx]) {
				baseIdx++
			}

			// Collect lines added in derived
			for derivedIdx < len(derived) && (lcsIdx >= len(lcs) || derived[derivedIdx] != lcs[lcsIdx]) {
				newLines = append(newLines, derived[derivedIdx])
				derivedIdx++
			}

			changeType := "modify"
			if changeStart == baseIdx && len(newLines) > 0 {
				changeType = "add"
			} else if len(newLines) == 0 {
				changeType = "delete"
			}

			changes = append(changes, Change{
				Type:      changeType,
				BaseStart: changeStart,
				BaseEnd:   baseIdx,
				NewLines:  newLines,
			})
		}
	}

	return changes
}

// applyChanges applies source and target changes to base, detecting conflicts.
func (m *Merger) applyChanges(base []string, sourceChanges, targetChanges []Change, result *MergeResult) []string {
	// Build a map of base line indices to changes
	sourceChangeMap := make(map[int]Change)
	targetChangeMap := make(map[int]Change)

	for _, c := range sourceChanges {
		for i := c.BaseStart; i < max(c.BaseEnd, c.BaseStart+1); i++ {
			sourceChangeMap[i] = c
		}
	}

	for _, c := range targetChanges {
		for i := c.BaseStart; i < max(c.BaseEnd, c.BaseStart+1); i++ {
			targetChangeMap[i] = c
		}
	}

	var merged []string
	processedSource := make(map[int]bool)
	processedTarget := make(map[int]bool)

	for i := 0; i <= len(base); i++ {
		sourceChange, hasSource := sourceChangeMap[i]
		targetChange, hasTarget := targetChangeMap[i]

		if hasSource && !processedSource[sourceChange.BaseStart] {
			processedSource[sourceChange.BaseStart] = true

			if hasTarget && !processedTarget[targetChange.BaseStart] {
				processedTarget[targetChange.BaseStart] = true

				// Both have changes at this position
				if m.changesEqual(sourceChange, targetChange) {
					// Same change - apply once
					merged = append(merged, sourceChange.NewLines...)
				} else {
					// Conflict!
					result.Success = false
					result.HasConflictMarkers = true

					conflict := MergeConflict{
						StartLine:     len(merged) + 1,
						SourceContent: strings.Join(sourceChange.NewLines, "\n"),
						TargetContent: strings.Join(targetChange.NewLines, "\n"),
					}

					if sourceChange.BaseStart < len(base) {
						baseSection := base[sourceChange.BaseStart:min(sourceChange.BaseEnd, len(base))]
						conflict.BaseContent = strings.Join(baseSection, "\n")
					}

					merged = append(merged, m.ConflictMarkerStart)
					merged = append(merged, sourceChange.NewLines...)
					merged = append(merged, m.ConflictMarkerMiddle)
					merged = append(merged, targetChange.NewLines...)
					merged = append(merged, m.ConflictMarkerEnd)

					conflict.EndLine = len(merged)
					result.Conflicts = append(result.Conflicts, conflict)
				}
			} else {
				// Only source change
				merged = append(merged, sourceChange.NewLines...)
			}
		} else if hasTarget && !processedTarget[targetChange.BaseStart] {
			processedTarget[targetChange.BaseStart] = true
			// Only target change
			merged = append(merged, targetChange.NewLines...)
		} else if i < len(base) && !hasSource && !hasTarget {
			// No changes - keep base line
			merged = append(merged, base[i])
		}
	}

	return merged
}

// changesEqual checks if two changes are equivalent.
func (m *Merger) changesEqual(a, b Change) bool {
	if a.Type != b.Type || a.BaseStart != b.BaseStart || a.BaseEnd != b.BaseEnd {
		return false
	}
	if len(a.NewLines) != len(b.NewLines) {
		return false
	}
	for i := range a.NewLines {
		if a.NewLines[i] != b.NewLines[i] {
			return false
		}
	}
	return true
}

// longestCommonSubsequence finds the LCS of two string slices.
func (m *Merger) longestCommonSubsequence(a, b []string) []string {
	lenA, lenB := len(a), len(b)
	if lenA == 0 || lenB == 0 {
		return nil
	}

	// Build LCS length table
	dp := make([][]int, lenA+1)
	for i := range dp {
		dp[i] = make([]int, lenB+1)
	}

	for i := 1; i <= lenA; i++ {
		for j := 1; j <= lenB; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to find the LCS
	lcs := make([]string, dp[lenA][lenB])
	i, j, idx := lenA, lenB, dp[lenA][lenB]-1

	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			lcs[idx] = a[i-1]
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

// ResolveWithChoice resolves a conflict using the specified choice.
func (m *Merger) ResolveWithChoice(conflict *Conflict, choice ResolutionChoice) string {
	logging.Debug("resolving conflict with choice",
		logging.Skill(conflict.SkillName),
		logging.Operation("resolve"),
		slog.String("choice", string(choice)),
	)

	switch choice {
	case ResolutionUseSource:
		logging.Debug("using source content",
			logging.Skill(conflict.SkillName),
		)
		return conflict.Source.Content
	case ResolutionUseTarget:
		logging.Debug("using target content",
			logging.Skill(conflict.SkillName),
		)
		return conflict.Target.Content
	case ResolutionMerge:
		logging.Debug("merging content",
			logging.Skill(conflict.SkillName),
		)
		result := m.TwoWayMerge(conflict.Source, conflict.Target)
		return result.Content
	case ResolutionSkip:
		logging.Debug("skipping, keeping target content",
			logging.Skill(conflict.SkillName),
		)
		return conflict.Target.Content // Keep existing
	default:
		logging.Debug("using source content (default)",
			logging.Skill(conflict.SkillName),
		)
		return conflict.Source.Content
	}
}
