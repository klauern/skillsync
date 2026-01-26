package sync

import (
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestConflictDetector_DetectConflict(t *testing.T) {
	cd := NewConflictDetector()

	tests := []struct {
		name           string
		source         model.Skill
		target         model.Skill
		expectConflict bool
		conflictType   ConflictType
	}{
		{
			name: "identical content - no conflict",
			source: model.Skill{
				Name:    "test-skill",
				Content: "# Test\nSome content",
			},
			target: model.Skill{
				Name:    "test-skill",
				Content: "# Test\nSome content",
			},
			expectConflict: false,
		},
		{
			name: "different content - conflict",
			source: model.Skill{
				Name:    "test-skill",
				Content: "# Test\nSource content",
			},
			target: model.Skill{
				Name:    "test-skill",
				Content: "# Test\nTarget content",
			},
			expectConflict: true,
			conflictType:   ConflictTypeContent,
		},
		{
			name: "different metadata - conflict",
			source: model.Skill{
				Name:        "test-skill",
				Content:     "# Test\nSame content",
				Description: "Source description",
			},
			target: model.Skill{
				Name:        "test-skill",
				Content:     "# Test\nSame content",
				Description: "Target description",
			},
			expectConflict: true,
			conflictType:   ConflictTypeMetadata,
		},
		{
			name: "different content and metadata - both conflict",
			source: model.Skill{
				Name:        "test-skill",
				Content:     "Source content",
				Description: "Source description",
			},
			target: model.Skill{
				Name:        "test-skill",
				Content:     "Target content",
				Description: "Target description",
			},
			expectConflict: true,
			conflictType:   ConflictTypeBoth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflict := cd.DetectConflict(tt.source, tt.target)

			if tt.expectConflict && conflict == nil {
				t.Error("Expected conflict but got nil")
			}
			if !tt.expectConflict && conflict != nil {
				t.Errorf("Expected no conflict but got: %v", conflict.Type)
			}
			if conflict != nil && conflict.Type != tt.conflictType {
				t.Errorf("Expected conflict type %s, got %s", tt.conflictType, conflict.Type)
			}
		})
	}
}

func TestConflict_Summary(t *testing.T) {
	conflict := &Conflict{
		SkillName: "test-skill",
		Type:      ConflictTypeContent,
	}

	summary := conflict.Summary()
	if summary != "test-skill: content differs" {
		t.Errorf("Unexpected summary: %s", summary)
	}
}

func TestConflict_HasConflicts(t *testing.T) {
	conflict := &Conflict{
		SkillName:  "test-skill",
		Resolution: "",
	}

	if !conflict.HasConflicts() {
		t.Error("Expected HasConflicts to return true for unresolved conflict")
	}

	conflict.Resolution = ResolutionUseSource
	if conflict.HasConflicts() {
		t.Error("Expected HasConflicts to return false for resolved conflict")
	}
}

func TestConflictDetector_ComputeDiff(t *testing.T) {
	cd := NewConflictDetector()

	source := model.Skill{
		Name:    "test",
		Content: "line1\nline2\nline3",
	}
	target := model.Skill{
		Name:    "test",
		Content: "line1\nmodified\nline3",
	}

	conflict := cd.DetectConflict(source, target)
	if conflict == nil {
		t.Fatal("Expected conflict")
	}

	if len(conflict.Hunks) == 0 {
		t.Error("Expected at least one diff hunk")
	}
}

func TestDiffLine_String(t *testing.T) {
	tests := []struct {
		line     DiffLine
		expected string
	}{
		{
			line:     DiffLine{Type: DiffLineContext, Content: "unchanged"},
			expected: " unchanged",
		},
		{
			line:     DiffLine{Type: DiffLineAdded, Content: "added"},
			expected: "+added",
		},
		{
			line:     DiffLine{Type: DiffLineRemoved, Content: "removed"},
			expected: "-removed",
		},
	}

	for _, tt := range tests {
		result := tt.line.String()
		if result != tt.expected {
			t.Errorf("Expected %q, got %q", tt.expected, result)
		}
	}
}
