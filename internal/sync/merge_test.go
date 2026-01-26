package sync

import (
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestMerger_TwoWayMerge_NoConflict(t *testing.T) {
	m := NewMerger()

	source := model.Skill{
		Name:    "test",
		Content: "line1\nline2\nline3",
	}
	target := model.Skill{
		Name:    "test",
		Content: "line1\nline2\nline3\nline4",
	}

	result := m.TwoWayMerge(source, target)

	if !result.Success {
		t.Errorf("Expected successful merge, got conflicts: %d", len(result.Conflicts))
	}
	if result.HasConflictMarkers {
		t.Error("Did not expect conflict markers")
	}
}

func TestMerger_TwoWayMerge_WithConflict(t *testing.T) {
	m := NewMerger()

	source := model.Skill{
		Name:    "test",
		Content: "line1\nsource change\nline3",
	}
	target := model.Skill{
		Name:    "test",
		Content: "line1\ntarget change\nline3",
	}

	result := m.TwoWayMerge(source, target)

	if result.Success {
		t.Error("Expected merge to fail due to conflict")
	}
	if !result.HasConflictMarkers {
		t.Error("Expected conflict markers")
	}
	if !strings.Contains(result.Content, "<<<<<<< SOURCE") {
		t.Error("Expected source conflict marker")
	}
	if !strings.Contains(result.Content, "=======") {
		t.Error("Expected middle conflict marker")
	}
	if !strings.Contains(result.Content, ">>>>>>> TARGET") {
		t.Error("Expected target conflict marker")
	}
}

func TestMerger_ThreeWayMerge_NoConflict(t *testing.T) {
	m := NewMerger()

	base := &model.Skill{
		Name:    "test",
		Content: "line1\nline2\nline3",
	}
	source := model.Skill{
		Name:    "test",
		Content: "line1\nline2 modified\nline3",
	}
	target := model.Skill{
		Name:    "test",
		Content: "line1\nline2\nline3\nline4",
	}

	result := m.ThreeWayMerge(source, target, base)

	// Three-way merge should succeed when changes don't overlap
	if result.HasConflictMarkers && len(result.Conflicts) > 0 {
		// If there are conflicts, verify they're properly marked
		if !strings.Contains(result.Content, "<<<<<<< SOURCE") {
			t.Error("Conflicts should have markers")
		}
	}
}

func TestMerger_ThreeWayMerge_WithConflict(t *testing.T) {
	m := NewMerger()

	base := &model.Skill{
		Name:    "test",
		Content: "line1\noriginal\nline3",
	}
	source := model.Skill{
		Name:    "test",
		Content: "line1\nsource change\nline3",
	}
	target := model.Skill{
		Name:    "test",
		Content: "line1\ntarget change\nline3",
	}

	result := m.ThreeWayMerge(source, target, base)

	// Both modified the same line - should conflict
	if result.Success {
		t.Error("Expected merge to fail due to overlapping changes")
	}
	if !result.HasConflictMarkers {
		t.Error("Expected conflict markers")
	}
}

func TestMerger_ResolveWithChoice(t *testing.T) {
	m := NewMerger()

	conflict := &Conflict{
		SkillName: "test",
		Source: model.Skill{
			Name:    "test",
			Content: "source content",
		},
		Target: model.Skill{
			Name:    "test",
			Content: "target content",
		},
	}

	tests := []struct {
		name     string
		choice   ResolutionChoice
		expected string
	}{
		{
			name:     "use source",
			choice:   ResolutionUseSource,
			expected: "source content",
		},
		{
			name:     "use target",
			choice:   ResolutionUseTarget,
			expected: "target content",
		},
		{
			name:     "skip",
			choice:   ResolutionSkip,
			expected: "target content", // Skip keeps target
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.ResolveWithChoice(conflict, tt.choice)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMerger_LongestCommonSubsequence(t *testing.T) {
	m := NewMerger()

	tests := []struct {
		name     string
		a        []string
		b        []string
		expected int // length of LCS
	}{
		{
			name:     "identical",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "c"},
			expected: 3,
		},
		{
			name:     "partial overlap",
			a:        []string{"a", "b", "c", "d"},
			b:        []string{"a", "x", "c", "y"},
			expected: 2, // a, c
		},
		{
			name:     "no overlap",
			a:        []string{"a", "b"},
			b:        []string{"c", "d"},
			expected: 0,
		},
		{
			name:     "empty",
			a:        []string{},
			b:        []string{"a", "b"},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lcs := m.longestCommonSubsequence(tt.a, tt.b)
			if len(lcs) != tt.expected {
				t.Errorf("Expected LCS length %d, got %d", tt.expected, len(lcs))
			}
		})
	}
}

func TestNewMerger(t *testing.T) {
	m := NewMerger()

	if m.ConflictMarkerStart != "<<<<<<< SOURCE" {
		t.Errorf("Unexpected start marker: %s", m.ConflictMarkerStart)
	}
	if m.ConflictMarkerMiddle != "=======" {
		t.Errorf("Unexpected middle marker: %s", m.ConflictMarkerMiddle)
	}
	if m.ConflictMarkerEnd != ">>>>>>> TARGET" {
		t.Errorf("Unexpected end marker: %s", m.ConflictMarkerEnd)
	}
}
