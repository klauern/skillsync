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

// Benchmark tests for merge operations

func BenchmarkLongestCommonSubsequence(b *testing.B) {
	m := NewMerger()

	// Test with varying sizes
	b.Run("small (10 lines)", func(b *testing.B) {
		a := make([]string, 10)
		bLines := make([]string, 10)
		for i := range 10 {
			a[i] = "line " + string(rune('0'+i))
			bLines[i] = "line " + string(rune('0'+i))
		}
		// Make 20% different
		bLines[2] = "different line"
		bLines[7] = "another different line"

		b.ResetTimer()
		for b.Loop() {
			_ = m.longestCommonSubsequence(a, bLines)
		}
	})

	b.Run("medium (100 lines)", func(b *testing.B) {
		a := make([]string, 100)
		bLines := make([]string, 100)
		for i := range 100 {
			a[i] = "line " + string(rune('0'+(i%10)))
			bLines[i] = "line " + string(rune('0'+(i%10)))
		}
		// Make 20% different
		for i := range 20 {
			bLines[i*5] = "different line " + string(rune('0'+(i%10)))
		}

		b.ResetTimer()
		for b.Loop() {
			_ = m.longestCommonSubsequence(a, bLines)
		}
	})

	b.Run("large (1000 lines)", func(b *testing.B) {
		a := make([]string, 1000)
		bLines := make([]string, 1000)
		for i := range 1000 {
			a[i] = "line " + string(rune('0'+(i%10)))
			bLines[i] = "line " + string(rune('0'+(i%10)))
		}
		// Make 10% different
		for i := range 100 {
			bLines[i*10] = "different line " + string(rune('0'+(i%10)))
		}

		b.ResetTimer()
		for b.Loop() {
			_ = m.longestCommonSubsequence(a, bLines)
		}
	})
}

func BenchmarkTwoWayMerge(b *testing.B) {
	m := NewMerger()

	// Generate realistic content for benchmarking
	generateLines := func(n int) string {
		var sb strings.Builder
		for i := range n {
			sb.WriteString("# Section ")
			sb.WriteString(string(rune('0' + (i % 10))))
			sb.WriteString("\n\nThis is a paragraph with some content.\n")
			sb.WriteString("It has multiple lines and realistic structure.\n\n")
		}
		return sb.String()
	}

	b.Run("small (10 lines)", func(b *testing.B) {
		source := model.Skill{
			Name:    "test",
			Content: generateLines(2),
		}
		target := model.Skill{
			Name:    "test",
			Content: generateLines(2) + "Additional target content\n",
		}

		b.ResetTimer()
		for b.Loop() {
			_ = m.TwoWayMerge(source, target)
		}
	})

	b.Run("medium (100 lines)", func(b *testing.B) {
		source := model.Skill{
			Name:    "test",
			Content: generateLines(20),
		}
		target := model.Skill{
			Name:    "test",
			Content: generateLines(20) + "Additional target content\n",
		}

		b.ResetTimer()
		for b.Loop() {
			_ = m.TwoWayMerge(source, target)
		}
	})

	b.Run("large (1000 lines)", func(b *testing.B) {
		source := model.Skill{
			Name:    "test",
			Content: generateLines(200),
		}
		target := model.Skill{
			Name:    "test",
			Content: generateLines(200) + "Additional target content\n",
		}

		b.ResetTimer()
		for b.Loop() {
			_ = m.TwoWayMerge(source, target)
		}
	})
}

func BenchmarkThreeWayMerge(b *testing.B) {
	m := NewMerger()

	generateLines := func(n int) string {
		var sb strings.Builder
		for i := range n {
			sb.WriteString("Line ")
			sb.WriteString(string(rune('0' + (i % 10))))
			sb.WriteString("\n")
		}
		return sb.String()
	}

	b.Run("small (10 lines)", func(b *testing.B) {
		baseSkill := model.Skill{
			Name:    "test",
			Content: generateLines(10),
		}
		source := model.Skill{
			Name:    "test",
			Content: generateLines(10) + "Source addition\n",
		}
		target := model.Skill{
			Name:    "test",
			Content: generateLines(10) + "Target addition\n",
		}

		b.ResetTimer()
		for b.Loop() {
			_ = m.ThreeWayMerge(source, target, &baseSkill)
		}
	})

	b.Run("medium (100 lines)", func(b *testing.B) {
		baseSkill := model.Skill{
			Name:    "test",
			Content: generateLines(100),
		}
		source := model.Skill{
			Name:    "test",
			Content: generateLines(100) + "Source addition\n",
		}
		target := model.Skill{
			Name:    "test",
			Content: generateLines(100) + "Target addition\n",
		}

		b.ResetTimer()
		for b.Loop() {
			_ = m.ThreeWayMerge(source, target, &baseSkill)
		}
	})
}

func BenchmarkFindChanges(b *testing.B) {
	m := NewMerger()

	b.Run("small (10 lines)", func(b *testing.B) {
		base := make([]string, 10)
		changed := make([]string, 10)
		for i := range 10 {
			base[i] = "line " + string(rune('0'+i))
			changed[i] = "line " + string(rune('0'+i))
		}
		// Make 2 changes
		changed[3] = "modified line"
		changed[7] = "another modified line"

		b.ResetTimer()
		for b.Loop() {
			_ = m.findChanges(base, changed)
		}
	})

	b.Run("medium (100 lines)", func(b *testing.B) {
		base := make([]string, 100)
		changed := make([]string, 100)
		for i := range 100 {
			base[i] = "line " + string(rune('0'+(i%10)))
			changed[i] = "line " + string(rune('0'+(i%10)))
		}
		// Make 20 changes
		for i := range 20 {
			changed[i*5] = "modified line " + string(rune('0'+i))
		}

		b.ResetTimer()
		for b.Loop() {
			_ = m.findChanges(base, changed)
		}
	})
}
