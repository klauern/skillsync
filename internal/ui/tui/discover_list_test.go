package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/model"
)

func TestNewDiscoverListModel(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "test-skill",
			Description: "A test skill",
			Platform:    model.ClaudeCode,
			Path:        "/home/user/.claude/skills/test.md",
			Scope:       model.ScopeUser,
			ModifiedAt:  time.Now(),
		},
		{
			Name:        "another-skill",
			Description: "Another test skill",
			Platform:    model.Cursor,
			Path:        "/home/user/.cursor/skills/another.md",
			Scope:       model.ScopeRepo,
			ModifiedAt:  time.Now().Add(-24 * time.Hour),
		},
	}

	model := NewDiscoverListModel(skills)

	if len(model.skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(model.skills))
	}

	if len(model.filtered) != 2 {
		t.Errorf("expected 2 filtered skills, got %d", len(model.filtered))
	}
}

func TestDiscoverListModel_Filter(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "test-skill",
			Description: "A test skill",
			Platform:    model.ClaudeCode,
			Path:        "/home/user/.claude/skills/test.md",
			Scope:       model.ScopeUser,
		},
		{
			Name:        "cursor-skill",
			Description: "A cursor skill",
			Platform:    model.Cursor,
			Path:        "/home/user/.cursor/skills/another.md",
			Scope:       model.ScopeRepo,
		},
	}

	m := NewDiscoverListModel(skills)
	m.filter = "cursor"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}

	if m.filtered[0].Platform != model.Cursor {
		t.Errorf("expected filtered skill to be cursor, got %s", m.filtered[0].Platform)
	}
}

func TestDiscoverListModel_FilterByScope(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "user-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
		{
			Name:     "repo-skill",
			Platform: model.Cursor,
			Scope:    model.ScopeRepo,
		},
	}

	m := NewDiscoverListModel(skills)
	m.filter = "repo"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}
}

func TestDiscoverListModel_FilterByDescription(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "skill-one",
			Description: "Handles authentication",
			Platform:    model.ClaudeCode,
		},
		{
			Name:        "skill-two",
			Description: "Manages database connections",
			Platform:    model.Cursor,
		},
	}

	m := NewDiscoverListModel(skills)
	m.filter = "database"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}

	if m.filtered[0].Name != "skill-two" {
		t.Errorf("expected 'skill-two', got %s", m.filtered[0].Name)
	}
}

func TestDiscoverListModel_ClearFilter(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "skill-one",
			Platform: model.ClaudeCode,
		},
		{
			Name:     "skill-two",
			Platform: model.Cursor,
		},
	}

	m := NewDiscoverListModel(skills)
	m.filter = "cursor"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}

	// Clear filter
	m.filter = ""
	m.applyFilter()

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 skills after clearing filter, got %d", len(m.filtered))
	}
}

func TestDiscoverListModel_EmptySkills(t *testing.T) {
	m := NewDiscoverListModel([]model.Skill{})

	if len(m.skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(m.skills))
	}

	// View should still work without panicking
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestDiscoverListModel_Init(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewDiscoverListModel(skills)
	cmd := m.Init()

	if cmd != nil {
		t.Error("expected nil command from Init")
	}
}

func TestDiscoverListModel_QuitKey(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewDiscoverListModel(skills)

	// Simulate pressing 'q'
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	dm := newModel.(DiscoverListModel)
	if !dm.quitting {
		t.Error("expected model to be quitting after pressing 'q'")
	}

	// Should return a quit command
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestDiscoverListModel_HelpToggle(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewDiscoverListModel(skills)

	if m.showHelp {
		t.Error("expected showHelp to be false initially")
	}

	// Simulate pressing '?'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	dm := newModel.(DiscoverListModel)

	if !dm.showHelp {
		t.Error("expected showHelp to be true after pressing '?'")
	}

	// Toggle again
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	dm = newModel.(DiscoverListModel)

	if dm.showHelp {
		t.Error("expected showHelp to be false after pressing '?' again")
	}
}

func TestDiscoverListModel_ViewAction(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Content:  "Test content",
		},
	}

	m := NewDiscoverListModel(skills)

	// Simulate pressing 'v' for view
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})

	dm := newModel.(DiscoverListModel)
	result := dm.Result()

	if result.Action != DiscoverActionView {
		t.Errorf("expected DiscoverActionView, got %v", result.Action)
	}

	if result.Skill.Name != "test-skill" {
		t.Errorf("expected skill name 'test-skill', got '%s'", result.Skill.Name)
	}

	if !dm.quitting {
		t.Error("expected model to be quitting after view action")
	}

	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestDiscoverListModel_CopyAction(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Path:     "/path/to/skill.md",
		},
	}

	m := NewDiscoverListModel(skills)

	// Simulate pressing 'c' for copy
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	dm := newModel.(DiscoverListModel)
	result := dm.Result()

	if result.Action != DiscoverActionCopy {
		t.Errorf("expected DiscoverActionCopy, got %v", result.Action)
	}

	if result.Skill.Path != "/path/to/skill.md" {
		t.Errorf("expected path '/path/to/skill.md', got '%s'", result.Skill.Path)
	}

	if !dm.quitting {
		t.Error("expected model to be quitting after copy action")
	}

	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestDiscoverListResult_DefaultAction(t *testing.T) {
	m := NewDiscoverListModel([]model.Skill{})
	result := m.Result()

	if result.Action != DiscoverActionNone {
		t.Errorf("expected DiscoverActionNone, got %v", result.Action)
	}
}

func TestSkillsToRows(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "test-skill",
			Description: "A test skill description",
			Platform:    model.ClaudeCode,
			Scope:       model.ScopeUser,
		},
	}

	rows := skillsToRows(skills)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if row[0] != "test-skill" {
		t.Errorf("expected name 'test-skill', got '%s'", row[0])
	}
	if row[1] != "claude-code" {
		t.Errorf("expected platform 'claude-code', got '%s'", row[1])
	}
	if row[2] != "~/.claude" {
		t.Errorf("expected scope '~/.claude', got '%s'", row[2])
	}
}

func TestSkillsToRows_LongValues(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "this-is-a-very-long-skill-name-that-exceeds-the-limit",
			Description: "This is a very long description that should be truncated when displayed in the table view",
			Platform:    model.ClaudeCode,
		},
	}

	rows := skillsToRows(skills)
	row := rows[0]

	// Name should be truncated to 25 chars
	if len(row[0]) > 25 {
		t.Errorf("expected name to be truncated to 25 chars, got %d chars", len(row[0]))
	}
	if row[0][len(row[0])-3:] != "..." {
		t.Errorf("expected name to end with '...', got '%s'", row[0])
	}

	// Description should be truncated to 45 chars
	if len(row[3]) > 45 {
		t.Errorf("expected description to be truncated to 45 chars, got %d chars", len(row[3]))
	}
	if row[3][len(row[3])-3:] != "..." {
		t.Errorf("expected description to end with '...', got '%s'", row[3])
	}
}

func TestDiscoverListModel_FilterMode(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewDiscoverListModel(skills)

	if m.filtering {
		t.Error("expected filtering to be false initially")
	}

	// Enter filter mode with '/'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	dm := newModel.(DiscoverListModel)

	if !dm.filtering {
		t.Error("expected filtering to be true after pressing '/'")
	}

	// Type some characters
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	dm = newModel.(DiscoverListModel)

	if dm.filter != "t" {
		t.Errorf("expected filter 't', got '%s'", dm.filter)
	}

	// Exit filter mode with enter
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	dm = newModel.(DiscoverListModel)

	if dm.filtering {
		t.Error("expected filtering to be false after pressing enter")
	}
}

func TestDiscoverListModel_FilterModeEscape(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
		{
			Name:     "other-skill",
			Platform: model.Cursor,
		},
	}

	m := NewDiscoverListModel(skills)

	// Enter filter mode and type
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	dm := newModel.(DiscoverListModel)
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	dm = newModel.(DiscoverListModel)
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	dm = newModel.(DiscoverListModel)

	// Cancel filter with escape
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyEscape})
	dm = newModel.(DiscoverListModel)

	if dm.filtering {
		t.Error("expected filtering to be false after escape")
	}

	if dm.filter != "" {
		t.Errorf("expected filter to be cleared, got '%s'", dm.filter)
	}

	// Should show all skills again
	if len(dm.filtered) != 2 {
		t.Errorf("expected 2 skills after clearing filter, got %d", len(dm.filtered))
	}
}

func TestDiscoverListModel_FilterBackspace(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewDiscoverListModel(skills)

	// Enter filter mode and type
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	dm := newModel.(DiscoverListModel)
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	dm = newModel.(DiscoverListModel)
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	dm = newModel.(DiscoverListModel)

	if dm.filter != "te" {
		t.Errorf("expected filter 'te', got '%s'", dm.filter)
	}

	// Backspace
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	dm = newModel.(DiscoverListModel)

	if dm.filter != "t" {
		t.Errorf("expected filter 't' after backspace, got '%s'", dm.filter)
	}
}

func TestDiscoverListModel_WindowResize(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewDiscoverListModel(skills)

	// Simulate window resize
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	dm := newModel.(DiscoverListModel)

	if dm.width != 120 {
		t.Errorf("expected width 120, got %d", dm.width)
	}

	if dm.height != 40 {
		t.Errorf("expected height 40, got %d", dm.height)
	}
}

func TestRunDiscoverList_EmptySkills(t *testing.T) {
	result, err := RunDiscoverList([]model.Skill{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result.Action != DiscoverActionNone {
		t.Errorf("expected DiscoverActionNone, got %v", result.Action)
	}
}
