package tui

import (
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/similarity"
	"github.com/klauern/skillsync/internal/sync"
)

func TestCompareListModel_ComparisonsToRows_TruncatesAndFormats(t *testing.T) {
	longName := "this-is-a-very-long-skill-name"
	comparison := &similarity.ComparisonResult{
		Skill1: model.Skill{
			Name:     longName,
			Platform: model.ClaudeCode,
		},
		Skill2: model.Skill{
			Name:     longName + "-two",
			Platform: model.Cursor,
		},
		NameScore:    0.84,
		ContentScore: 0.91,
		Hunks: []sync.DiffHunk{
			{
				SourceStart: 1,
				SourceCount: 1,
				TargetStart: 1,
				TargetCount: 1,
			},
		},
		LinesAdded:   2,
		LinesRemoved: 1,
	}

	model := NewCompareListModel([]*similarity.ComparisonResult{comparison})
	rows := model.comparisonsToRows([]*similarity.ComparisonResult{comparison})

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	expectedName := longName[:19] + "..."
	if row[0] != expectedName {
		t.Errorf("expected truncated name %q, got %q", expectedName, row[0])
	}
	if row[1] != "cc" || row[3] != "cur" {
		t.Errorf("expected platform short names cc/cur, got %q/%q", row[1], row[3])
	}
	if row[4] != "84%" {
		t.Errorf("expected name score 84%%, got %q", row[4])
	}
	if row[5] != "91%" {
		t.Errorf("expected content score 91%%, got %q", row[5])
	}
	if !strings.Contains(row[6], "1 hunk(s)") {
		t.Errorf("expected changes summary to include hunk count, got %q", row[6])
	}
}

func TestCompareListModel_BuildDiffContent_WrapsDescription(t *testing.T) {
	comparison := &similarity.ComparisonResult{
		Skill1: model.Skill{
			Name:        "skill-one",
			Platform:    model.ClaudeCode,
			Scope:       model.ScopeUser,
			Description: "alpha beta gamma",
		},
		Skill2: model.Skill{
			Name:     "skill-two",
			Platform: model.Cursor,
			Scope:    model.ScopeRepo,
		},
	}

	model := NewCompareListModel([]*similarity.ComparisonResult{comparison})
	model.viewport.Width = 30

	content := model.buildDiffContent(comparison)
	label := "  Description: "
	expected := label + "alpha beta\n" + strings.Repeat(" ", len(label)) + "gamma"
	if !strings.Contains(content, expected) {
		t.Errorf("expected wrapped description, got %q", content)
	}
}

func TestCompareListModel_ApplyFilter_ByPlatform(t *testing.T) {
	comp1 := &similarity.ComparisonResult{
		Skill1: model.Skill{Name: "alpha", Platform: model.ClaudeCode},
		Skill2: model.Skill{Name: "beta", Platform: model.ClaudeCode},
	}
	comp2 := &similarity.ComparisonResult{
		Skill1: model.Skill{Name: "gamma", Platform: model.ClaudeCode},
		Skill2: model.Skill{Name: "delta", Platform: model.Cursor},
	}

	model := NewCompareListModel([]*similarity.ComparisonResult{comp1, comp2})
	model.filter = "cursor"
	model.applyFilter()

	if len(model.filtered) != 1 {
		t.Fatalf("expected 1 filtered comparison, got %d", len(model.filtered))
	}
	if model.filtered[0] != comp2 {
		t.Errorf("expected cursor comparison to remain after filter")
	}
}

func TestCompareListModel_BuildDiffContent_WithHunks(t *testing.T) {
	comp := &similarity.ComparisonResult{
		Skill1: model.Skill{
			Name:     "skill-one",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
			Content:  "one",
		},
		Skill2: model.Skill{
			Name:     "skill-two",
			Platform: model.Cursor,
			Scope:    model.ScopeRepo,
			Content:  "two",
		},
		NameScore:    0.6,
		ContentScore: 0.7,
		Hunks: []sync.DiffHunk{
			{
				SourceStart: 1,
				SourceCount: 1,
				TargetStart: 1,
				TargetCount: 1,
				Lines: []sync.DiffLine{
					{Type: sync.DiffLineRemoved, Content: "one"},
					{Type: sync.DiffLineAdded, Content: "two"},
				},
			},
		},
		LinesAdded:   1,
		LinesRemoved: 1,
	}

	model := NewCompareListModel([]*similarity.ComparisonResult{comp})
	content := model.buildDiffContent(comp)

	if !strings.Contains(content, "Differences") {
		t.Errorf("expected diff content to include Differences section")
	}
	if !strings.Contains(content, "@@ -1,1 +1,1 @@") {
		t.Errorf("expected diff content to include hunk header")
	}
}

func TestCompareListModel_BuildDiffContent_ContentPreview(t *testing.T) {
	comp := &similarity.ComparisonResult{
		Skill1: model.Skill{Name: "one", Content: "a\nb"},
		Skill2: model.Skill{Name: "two", Content: "a\nc"},
	}

	model := NewCompareListModel([]*similarity.ComparisonResult{comp})
	content := model.buildDiffContent(comp)

	if !strings.Contains(content, "Content Preview") {
		t.Errorf("expected content preview for diff without hunks")
	}
	if !strings.Contains(content, "Skill 1: 2 lines") {
		t.Errorf("expected line count summary in preview")
	}
}

func TestCompareListModel_BuildDiffContent_IdenticalContent(t *testing.T) {
	comp := &similarity.ComparisonResult{
		Skill1: model.Skill{Name: "one", Content: "same"},
		Skill2: model.Skill{Name: "two", Content: "same"},
	}

	model := NewCompareListModel([]*similarity.ComparisonResult{comp})
	content := model.buildDiffContent(comp)

	if !strings.Contains(content, "Contents are identical") {
		t.Errorf("expected identical content notice")
	}
}

func TestRunCompareList_EmptyComparisons(t *testing.T) {
	result, err := RunCompareList(nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Action != CompareActionNone {
		t.Fatalf("expected CompareActionNone, got %v", result.Action)
	}
}
