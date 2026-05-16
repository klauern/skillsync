package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/model"
)

func TestNewScopeListModel_ShowsAllSkillsInitially(t *testing.T) {
	skills := []model.Skill{
		{Name: "user-skill", Platform: model.ClaudeCode, Scope: model.ScopeUser},
		{Name: "repo-skill", Platform: model.Cursor, Scope: model.ScopeRepo},
		{Name: "plugin-skill", Platform: model.ClaudeCode, Scope: model.ScopePlugin},
	}

	m := NewScopeListModel(skills)

	// All skills should be visible initially (scopeIndex = -1 / all).
	if len(m.filtered) != len(skills) {
		t.Errorf("expected %d filtered skills initially, got %d", len(skills), len(m.filtered))
	}
}

func TestScopeListModel_TabCyclesScopeFilter(t *testing.T) {
	skills := []model.Skill{
		{Name: "user-skill-a", Platform: model.ClaudeCode, Scope: model.ScopeUser},
		{Name: "user-skill-b", Platform: model.ClaudeCode, Scope: model.ScopeUser},
		{Name: "repo-skill", Platform: model.Cursor, Scope: model.ScopeRepo},
	}

	m := NewScopeListModel(skills)

	// Press 'l' (NextScope) — should move from "all" to first scope (User).
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = newModel.(ScopeListModel)

	// Only user-scoped skills should be visible.
	if len(m.filtered) != 2 {
		t.Errorf("after tab to User scope: expected 2 filtered skills, got %d", len(m.filtered))
	}
	for _, s := range m.filtered {
		if s.Scope != model.ScopeUser {
			t.Errorf("unexpected skill scope after tab: %s", s.Scope)
		}
	}

	// Press 'l' again — move to next scope (Repo).
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = newModel.(ScopeListModel)

	if len(m.filtered) != 1 {
		t.Errorf("after tab to Repo scope: expected 1 filtered skill, got %d", len(m.filtered))
	}
}

func TestScopeListModel_ShiftTabCyclesScopeBackward(t *testing.T) {
	skills := []model.Skill{
		{Name: "user-skill", Platform: model.ClaudeCode, Scope: model.ScopeUser},
		{Name: "repo-skill", Platform: model.Cursor, Scope: model.ScopeRepo},
	}

	m := NewScopeListModel(skills)

	// Press 'h' (PrevScope) — from "all" should wrap to last scope.
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = newModel.(ScopeListModel)

	// Should show only repo skills (last scope).
	if len(m.filtered) != 1 {
		t.Errorf("expected 1 skill after shift-tab to last scope, got %d", len(m.filtered))
	}
	if m.filtered[0].Scope != model.ScopeRepo {
		t.Errorf("expected ScopeRepo, got %s", m.filtered[0].Scope)
	}
}

func TestScopeListModel_ApplyFilter_TextAndScope(t *testing.T) {
	skills := []model.Skill{
		{Name: "auth-skill", Platform: model.ClaudeCode, Scope: model.ScopeUser, Description: "Authentication"},
		{Name: "build-skill", Platform: model.Cursor, Scope: model.ScopeRepo, Description: "Build automation"},
		{Name: "debug-skill", Platform: model.ClaudeCode, Scope: model.ScopeUser, Description: "Debugging tools"},
	}

	m := NewScopeListModel(skills)

	// Tab to User scope.
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = newModel.(ScopeListModel)

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 user-scoped skills, got %d", len(m.filtered))
	}

	// Apply text filter via '/' then chars.
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = newModel.(ScopeListModel)
	for _, r := range "auth" {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(ScopeListModel)
	}

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 skill matching 'auth' in user scope, got %d", len(m.filtered))
	}
	if m.filtered[0].Name != "auth-skill" {
		t.Errorf("expected 'auth-skill', got %s", m.filtered[0].Name)
	}
}

func TestScopeListModel_SkillsToRows(t *testing.T) {
	skills := []model.Skill{
		{Name: "test-skill", Platform: model.ClaudeCode, Scope: model.ScopeUser, Description: "A test skill"},
	}

	m := NewScopeListModel(skills)
	rows := m.cfg.ToRows(skills)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if len(row) != 4 {
		t.Errorf("expected 4 columns, got %d", len(row))
	}
	if row[0] != "test-skill" {
		t.Errorf("expected name 'test-skill', got %s", row[0])
	}
	if row[1] != "claude-code" {
		t.Errorf("expected platform 'claude-code', got '%s'", row[1])
	}
	if row[2] != "~/.claude/skills" {
		t.Errorf("expected scope '~/.claude/skills', got '%s'", row[2])
	}
}

func TestScopeListModel_Result(t *testing.T) {
	skills := []model.Skill{
		{Name: "test-skill", Platform: model.ClaudeCode, Scope: model.ScopeUser},
	}

	m := NewScopeListModel(skills)

	result := m.Result()
	if result.Action != ScopeActionNone {
		t.Errorf("expected ScopeActionNone, got %d", result.Action)
	}
}

func TestRunScopeList_EmptySkills(t *testing.T) {
	result, err := RunScopeList(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Action != ScopeActionNone {
		t.Errorf("expected ScopeActionNone for empty skills, got %d", result.Action)
	}
}
