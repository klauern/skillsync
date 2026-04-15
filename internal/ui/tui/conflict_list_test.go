package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/sync"
)

func makeConflict(name string, withHunks bool) *sync.Conflict {
	source := model.Skill{
		Name:     name,
		Platform: model.ClaudeCode,
		Scope:    model.ScopeUser,
		Content:  "one\ntwo",
	}
	target := model.Skill{
		Name:     name,
		Platform: model.Cursor,
		Scope:    model.ScopeRepo,
		Content:  "one\nthree",
	}
	conflict := &sync.Conflict{
		SkillName: name,
		Type:      sync.ConflictTypeContent,
		Source:    source,
		Target:    target,
	}
	if withHunks {
		conflict.Hunks = []sync.DiffHunk{
			{
				SourceStart: 1,
				SourceCount: 1,
				TargetStart: 1,
				TargetCount: 1,
				Lines: []sync.DiffLine{
					{Type: sync.DiffLineRemoved, Content: "two"},
					{Type: sync.DiffLineAdded, Content: "three"},
				},
			},
		}
	}
	return conflict
}

func TestConflictListModel_BuildDetailContent_WithHunks(t *testing.T) {
	conflict := makeConflict("skill-one", true)
	model := NewConflictListModel([]*sync.Conflict{conflict})
	model.cursor = 0

	content := model.buildDetailContent()
	if !strings.Contains(content, "Changes") {
		t.Errorf("expected Changes section in detail view")
	}
	if !strings.Contains(content, "@@ -1,1 +1,1 @@") {
		t.Errorf("expected hunk header in detail view")
	}
}

func TestConflictListModel_BuildDetailContent_NoHunks(t *testing.T) {
	conflict := makeConflict("skill-two", false)
	model := NewConflictListModel([]*sync.Conflict{conflict})
	model.cursor = 0

	content := model.buildDetailContent()
	if !strings.Contains(content, "Source Content") {
		t.Errorf("expected Source Content section when no hunks")
	}
	if !strings.Contains(content, "Target Content") {
		t.Errorf("expected Target Content section when no hunks")
	}
	if !strings.Contains(content, "1 │") {
		t.Errorf("expected line numbers in content view")
	}
}

func TestConflictListModel_BuildResolutions(t *testing.T) {
	conflictA := makeConflict("alpha", true)
	conflictB := makeConflict("beta", true)

	model := NewConflictListModel([]*sync.Conflict{conflictA, conflictB})
	model.resolutions["alpha"] = sync.ResolutionUseSource
	model.resolutions["beta"] = sync.ResolutionMerge

	resolutions := model.buildResolutions()
	if len(resolutions) != 2 {
		t.Fatalf("expected 2 resolutions, got %d", len(resolutions))
	}
	if resolutions[0].Content != conflictA.Source.Content {
		t.Errorf("expected source content for ResolutionUseSource")
	}
	if resolutions[1].Content != conflictB.Source.Content {
		t.Errorf("expected source content for ResolutionMerge fallback")
	}
}

func TestConflictListModel_AllResolved(t *testing.T) {
	conflictA := makeConflict("alpha", true)
	conflictB := makeConflict("beta", true)

	model := NewConflictListModel([]*sync.Conflict{conflictA, conflictB})
	if model.allResolved() {
		t.Error("expected allResolved to be false with no resolutions")
	}

	model.resolutions["alpha"] = sync.ResolutionUseSource
	if model.allResolved() {
		t.Error("expected allResolved to be false with partial resolutions")
	}

	model.resolutions["beta"] = sync.ResolutionUseTarget
	if !model.allResolved() {
		t.Error("expected allResolved to be true with all resolutions")
	}
}

func TestConflictListModel_UpdateTableRow(t *testing.T) {
	conflict := makeConflict("skill-one", true)
	model := NewConflictListModel([]*sync.Conflict{conflict})

	model.resolveConflictAt(0, sync.ResolutionUseSource)
	rows := model.table.Rows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0][0] != "✓" {
		t.Errorf("expected resolved status, got %q", rows[0][0])
	}
	if rows[0][4] != string(sync.ResolutionUseSource) {
		t.Errorf("expected resolution column to be %q, got %q", sync.ResolutionUseSource, rows[0][4])
	}
}

func TestConflictListModel_ConfirmFlow(t *testing.T) {
	conflict := makeConflict("skill-one", true)
	model := NewConflictListModel([]*sync.Conflict{conflict})

	model.resolveConflictAt(0, sync.ResolutionUseSource)
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	confirmModel := newModel.(ConflictListModel)
	if !confirmModel.confirmMode {
		t.Error("expected confirm mode after pressing 'y'")
	}

	newModel, cmd := confirmModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	finalModel := newModel.(ConflictListModel)
	if finalModel.result.Action != ConflictActionResolve {
		t.Errorf("expected resolve action, got %v", finalModel.result.Action)
	}
	if !finalModel.quitting {
		t.Error("expected model to be quitting after confirmation")
	}
	if cmd == nil {
		t.Error("expected quit command after confirmation")
	}
}

func TestFormatConflictContentWithLineNumbers(t *testing.T) {
	result := formatConflictContentWithLineNumbers("line1\nline2", conflictStyles.Context)
	if !strings.Contains(result, "1 │") || !strings.Contains(result, "2 │") {
		t.Errorf("expected line numbers in formatted output")
	}
}

func TestRunConflictList_EmptyConflicts(t *testing.T) {
	result, err := RunConflictList(nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Action != ConflictActionNone {
		t.Fatalf("expected ConflictActionNone, got %v", result.Action)
	}
}
