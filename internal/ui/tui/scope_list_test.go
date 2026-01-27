package tui

import (
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestNewScopeListModel_CollectsScopeOptions(t *testing.T) {
	skills := []model.Skill{
		{Name: "user-skill", Platform: model.ClaudeCode, Scope: model.ScopeUser},
		{Name: "repo-skill", Platform: model.Cursor, Scope: model.ScopeRepo},
		{Name: "plugin-skill", Platform: model.ClaudeCode, Scope: model.ScopePlugin},
	}

	m := NewScopeListModel(skills)

	// Should have 3 scope options: User, Repo, Plugin (in precedence order)
	if len(m.scopeOptions) != 3 {
		t.Errorf("expected 3 scope options, got %d", len(m.scopeOptions))
	}

	// Verify order (precedence: User < Repo < Plugin)
	expectedOrder := []model.SkillScope{model.ScopeUser, model.ScopeRepo, model.ScopePlugin}
	for i, expected := range expectedOrder {
		if i >= len(m.scopeOptions) {
			break
		}
		if m.scopeOptions[i] != expected {
			t.Errorf("scope at index %d: expected %s, got %s", i, expected, m.scopeOptions[i])
		}
	}

	// Initially should show all skills
	if len(m.filtered) != len(skills) {
		t.Errorf("expected %d filtered skills, got %d", len(skills), len(m.filtered))
	}

	// scopeIndex should be -1 (all)
	if m.scopeIndex != -1 {
		t.Errorf("expected scopeIndex -1, got %d", m.scopeIndex)
	}
}

func TestScopeListModel_ApplyFilter_ByScopeAndText(t *testing.T) {
	skills := []model.Skill{
		{Name: "auth-skill", Platform: model.ClaudeCode, Scope: model.ScopeUser, Description: "Authentication"},
		{Name: "build-skill", Platform: model.Cursor, Scope: model.ScopeRepo, Description: "Build automation"},
		{Name: "debug-skill", Platform: model.ClaudeCode, Scope: model.ScopeUser, Description: "Debugging tools"},
	}

	m := NewScopeListModel(skills)

	// Test scope filtering: select "User" scope (index 0)
	m.scopeIndex = 0 // User scope
	m.applyFilter()

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 user-scoped skills, got %d", len(m.filtered))
	}

	// Test text filtering on top of scope filter
	m.filter = "auth"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 skill matching 'auth' in user scope, got %d", len(m.filtered))
	}

	if m.filtered[0].Name != "auth-skill" {
		t.Errorf("expected 'auth-skill', got %s", m.filtered[0].Name)
	}

	// Reset to all scopes
	m.scopeIndex = -1
	m.filter = "build"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 skill matching 'build', got %d", len(m.filtered))
	}
}

func TestScopeListModel_SkillsToRows(t *testing.T) {
	skills := []model.Skill{
		{Name: "test-skill", Platform: model.ClaudeCode, Scope: model.ScopeUser, Description: "A test skill"},
	}

	m := NewScopeListModel(skills)
	rows := m.skillsToRows(skills)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if len(row) != 4 {
		t.Errorf("expected 4 columns, got %d", len(row))
	}

	// Check columns: Name, Platform, Scope, Description
	if row[0] != "test-skill" {
		t.Errorf("expected name 'test-skill', got %s", row[0])
	}
	if row[1] != "claude-code" {
		t.Errorf("expected platform 'claude-code', got %s", row[1])
	}
	// Scope is displayed via DisplayScope()
	if row[2] != "~/.claude/skills" {
		t.Errorf("expected scope '~/.claude/skills', got %s", row[2])
	}
}

func TestScopeListModel_Result(t *testing.T) {
	skills := []model.Skill{
		{Name: "test-skill", Platform: model.ClaudeCode, Scope: model.ScopeUser},
	}

	m := NewScopeListModel(skills)

	// Initially result should be empty
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
