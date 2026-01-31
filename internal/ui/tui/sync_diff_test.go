package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/model"
)

func TestSyncDiffModel_BuildDiffContent_NewSkill(t *testing.T) {
	skill := model.Skill{
		Name:     "new-skill",
		Platform: model.ClaudeCode,
		Scope:    model.ScopeUser,
		Content:  "line1\nline2",
	}

	model := NewSyncDiffModel(skill, nil, model.ClaudeCode, model.Cursor)
	content := model.buildDiffContent()

	if !strings.Contains(content, "NEW skill") {
		t.Errorf("expected NEW skill notice in diff content")
	}
	if !strings.Contains(content, "Source Content") {
		t.Errorf("expected source content section in diff content")
	}
}

func TestSyncDiffModel_BuildDiffContent_Identical(t *testing.T) {
	skill := model.Skill{
		Name:     "existing-skill",
		Platform: model.ClaudeCode,
		Scope:    model.ScopeUser,
		Content:  "same",
	}
	target := model.Skill{
		Name:     "existing-skill",
		Platform: model.Cursor,
		Scope:    model.ScopeRepo,
		Content:  "same",
	}

	model := NewSyncDiffModel(skill, &target, model.ClaudeCode, model.Cursor)
	content := model.buildDiffContent()

	if !strings.Contains(content, "Contents are identical") {
		t.Errorf("expected identical content notice")
	}
}

func TestSyncDiffModel_Update_SyncAction(t *testing.T) {
	skill := model.Skill{Name: "sync-skill"}
	model := NewSyncDiffModel(skill, nil, model.ClaudeCode, model.Cursor)

	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	updated := newModel.(SyncDiffModel)

	if updated.result.Action != DiffActionSync {
		t.Errorf("expected DiffActionSync, got %v", updated.result.Action)
	}
	if !updated.quitting {
		t.Error("expected model to be quitting after sync action")
	}
	if cmd == nil {
		t.Error("expected quit command after sync action")
	}
}

func TestFormatContentWithLineNumbers(t *testing.T) {
	content := formatContentWithLineNumbers("line1\nline2", syncDiffStyles.Unchanged)
	if !strings.Contains(content, "1 │") || !strings.Contains(content, "2 │") {
		t.Errorf("expected line numbers in formatted content")
	}
}
