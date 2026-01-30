package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/model"
)

func TestNewDeleteListModel(t *testing.T) {
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

	m := NewDeleteListModel(skills)

	if len(m.skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(m.skills))
	}

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered skills, got %d", len(m.filtered))
	}

	// No skills should be selected by default (deletion is opt-in)
	if len(m.selected) != 0 {
		t.Errorf("expected 0 selected skills by default, got %d", len(m.selected))
	}
}

func TestNewDeleteListModel_FiltersDeletableScopes(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "user-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser, // Deletable
		},
		{
			Name:     "repo-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeRepo, // Deletable
		},
		{
			Name:     "admin-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeAdmin, // NOT deletable
		},
		{
			Name:     "system-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeSystem, // NOT deletable
		},
	}

	m := NewDeleteListModel(skills)

	// Should only include repo and user scopes
	if len(m.skills) != 2 {
		t.Errorf("expected 2 deletable skills, got %d", len(m.skills))
	}

	for _, s := range m.skills {
		if s.Scope != model.ScopeUser && s.Scope != model.ScopeRepo {
			t.Errorf("expected only user or repo scope, got %s", s.Scope)
		}
	}
}

func TestDeleteListModel_Filter(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "test-skill",
			Description: "A test skill",
			Platform:    model.ClaudeCode,
			Scope:       model.ScopeUser,
		},
		{
			Name:        "cursor-skill",
			Description: "A cursor skill",
			Platform:    model.Cursor,
			Scope:       model.ScopeRepo,
		},
	}

	m := NewDeleteListModel(skills)
	m.filter = "cursor"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}

	if m.filtered[0].Name != "cursor-skill" {
		t.Errorf("expected filtered skill to be cursor-skill, got %s", m.filtered[0].Name)
	}
}

func TestDeleteListModel_FilterByScope(t *testing.T) {
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

	m := NewDeleteListModel(skills)
	m.filter = "repo"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}
}

func TestDeleteListModel_FilterByPlatform(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "claude-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
		{
			Name:     "cursor-skill",
			Platform: model.Cursor,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)
	m.filter = "claude"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}

	if m.filtered[0].Name != "claude-skill" {
		t.Errorf("expected filtered skill to be claude-skill, got %s", m.filtered[0].Name)
	}
}

func TestDeleteListModel_Toggle(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)

	// Skill should NOT be selected initially (deletion is opt-in)
	if m.selected[deleteSkillKey(skills[0])] {
		t.Error("expected skill to NOT be selected initially")
	}

	// Simulate pressing space to toggle
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	dm := newModel.(DeleteListModel)

	if !dm.selected[deleteSkillKey(skills[0])] {
		t.Error("expected skill to be selected after toggle")
	}

	// Toggle again
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeySpace})
	dm = newModel.(DeleteListModel)

	if dm.selected[deleteSkillKey(skills[0])] {
		t.Error("expected skill to be deselected after second toggle")
	}
}

func TestDeleteListModel_ToggleAll(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "skill-one",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
		{
			Name:     "skill-two",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
		{
			Name:     "skill-three",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)

	// None should be selected initially
	selectedCount := 0
	for _, s := range m.skills {
		if m.selected[deleteSkillKey(s)] {
			selectedCount++
		}
	}
	if selectedCount != 0 {
		t.Errorf("expected 0 selected initially, got %d", selectedCount)
	}

	// Simulate pressing 'a' to toggle all (should select all since none are selected)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	dm := newModel.(DeleteListModel)

	selectedCount = 0
	for _, s := range dm.skills {
		if dm.selected[deleteSkillKey(s)] {
			selectedCount++
		}
	}
	if selectedCount != 3 {
		t.Errorf("expected 3 selected after toggle all, got %d", selectedCount)
	}

	// Toggle all again (should deselect all)
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	dm = newModel.(DeleteListModel)

	selectedCount = 0
	for _, s := range dm.skills {
		if dm.selected[deleteSkillKey(s)] {
			selectedCount++
		}
	}
	if selectedCount != 0 {
		t.Errorf("expected 0 selected after second toggle all, got %d", selectedCount)
	}
}

func TestDeleteListModel_GetSelectedSkills(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "skill-one",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
		{
			Name:     "skill-two",
			Platform: model.Cursor,
			Scope:    model.ScopeRepo,
		},
	}

	m := NewDeleteListModel(skills)

	// Select one skill
	m.selected[deleteSkillKey(skills[1])] = true

	selected := m.getSelectedSkills()
	if len(selected) != 1 {
		t.Errorf("expected 1 selected skill, got %d", len(selected))
	}

	if selected[0].Name != "skill-two" {
		t.Errorf("expected 'skill-two', got '%s'", selected[0].Name)
	}
}

func TestDeleteListModel_Init(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)
	cmd := m.Init()

	if cmd != nil {
		t.Error("expected nil command from Init")
	}
}

func TestDeleteListModel_QuitKey(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)

	// Simulate pressing 'q'
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	dm := newModel.(DeleteListModel)
	if !dm.quitting {
		t.Error("expected model to be quitting after pressing 'q'")
	}

	// Should return a quit command
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestDeleteListModel_HelpToggle(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)

	if m.showHelp {
		t.Error("expected showHelp to be false initially")
	}

	// Simulate pressing '?'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	dm := newModel.(DeleteListModel)

	if !dm.showHelp {
		t.Error("expected showHelp to be true after pressing '?'")
	}

	// Toggle again
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	dm = newModel.(DeleteListModel)

	if dm.showHelp {
		t.Error("expected showHelp to be false after pressing '?' again")
	}
}

func TestDeleteListModel_ConfirmDelete(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)

	// Select the skill first
	m.selected[deleteSkillKey(skills[0])] = true

	// Simulate pressing 'd' to confirm
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	dm := newModel.(DeleteListModel)

	// Should enter confirm mode
	if !dm.confirmMode {
		t.Error("expected confirmMode to be true after pressing 'd'")
	}

	// Confirm with 'y'
	newModel, cmd := dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	dm = newModel.(DeleteListModel)

	result := dm.Result()
	if result.Action != DeleteActionDelete {
		t.Errorf("expected DeleteActionDelete, got %v", result.Action)
	}

	if len(result.SelectedSkills) != 1 {
		t.Errorf("expected 1 selected skill, got %d", len(result.SelectedSkills))
	}

	if !dm.quitting {
		t.Error("expected model to be quitting after delete confirmation")
	}

	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestDeleteListModel_CancelConfirm(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)

	// Select the skill first
	m.selected[deleteSkillKey(skills[0])] = true

	// Enter confirm mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	dm := newModel.(DeleteListModel)

	if !dm.confirmMode {
		t.Error("expected confirmMode to be true")
	}

	// Cancel with 'n'
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	dm = newModel.(DeleteListModel)

	if dm.confirmMode {
		t.Error("expected confirmMode to be false after canceling")
	}

	if dm.quitting {
		t.Error("expected model to not be quitting after cancel")
	}
}

func TestDeleteListResult_DefaultAction(t *testing.T) {
	m := NewDeleteListModel([]model.Skill{})
	result := m.Result()

	if result.Action != DeleteActionNone {
		t.Errorf("expected DeleteActionNone, got %v", result.Action)
	}
}

func TestDeleteListModel_EmptySkills(t *testing.T) {
	m := NewDeleteListModel([]model.Skill{})

	if len(m.skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(m.skills))
	}

	// View should still work without panicking
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestDeleteListModel_FilterMode(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)

	if m.filtering {
		t.Error("expected filtering to be false initially")
	}

	// Enter filter mode with '/'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	dm := newModel.(DeleteListModel)

	if !dm.filtering {
		t.Error("expected filtering to be true after pressing '/'")
	}

	// Type some characters
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	dm = newModel.(DeleteListModel)

	if dm.filter != "t" {
		t.Errorf("expected filter 't', got '%s'", dm.filter)
	}

	// Exit filter mode with enter
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	dm = newModel.(DeleteListModel)

	if dm.filtering {
		t.Error("expected filtering to be false after pressing enter")
	}
}

func TestDeleteListModel_WindowResize(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)

	// Simulate window resize
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	dm := newModel.(DeleteListModel)

	if dm.width != 120 {
		t.Errorf("expected width 120, got %d", dm.width)
	}

	if dm.height != 40 {
		t.Errorf("expected height 40, got %d", dm.height)
	}
}

func TestRunDeleteList_EmptySkills(t *testing.T) {
	result, err := RunDeleteList([]model.Skill{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result.Action != DeleteActionNone {
		t.Errorf("expected DeleteActionNone, got %v", result.Action)
	}
}

func TestRunDeleteList_NoDeletableSkills(t *testing.T) {
	// Skills with non-deletable scopes
	skills := []model.Skill{
		{
			Name:     "admin-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeAdmin,
		},
		{
			Name:     "system-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeSystem,
		},
	}

	result, err := RunDeleteList(skills)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Should return empty result since no deletable skills
	if result.Action != DeleteActionNone {
		t.Errorf("expected DeleteActionNone, got %v", result.Action)
	}
}

func TestDeleteListModel_SkillsToRows_WithCheckbox(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "test-skill",
			Description: "A test skill description",
			Platform:    model.ClaudeCode,
			Scope:       model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)

	// Select the skill
	m.selected[deleteSkillKey(skills[0])] = true

	rows := m.skillsToRows(skills)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	// First column should be checkbox
	if row[0] != "[✓]" {
		t.Errorf("expected checkbox '[✓]', got '%s'", row[0])
	}
	if row[1] != "test-skill" {
		t.Errorf("expected name 'test-skill', got '%s'", row[1])
	}
	if row[2] != "claude-code" {
		t.Errorf("expected platform 'claude-code', got '%s'", row[2])
	}
}

func TestDeleteListModel_SkillsToRows_Unchecked(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)
	// Don't select the skill (default is unselected)

	rows := m.skillsToRows(skills)
	row := rows[0]

	if row[0] != "[ ]" {
		t.Errorf("expected checkbox '[ ]', got '%s'", row[0])
	}
}

func TestDeleteSkillKey(t *testing.T) {
	skill := model.Skill{
		Name:     "test-skill",
		Platform: model.ClaudeCode,
		Scope:    model.ScopeUser,
	}

	key := deleteSkillKey(skill)
	expected := "claude-code:user:test-skill"

	if key != expected {
		t.Errorf("expected key '%s', got '%s'", expected, key)
	}
}

func TestDeleteListModel_SamePlatformDifferentNames(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "skill-a",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
		{
			Name:     "skill-b",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)

	// Neither should be selected by default
	if m.selected[deleteSkillKey(skills[0])] {
		t.Error("expected skill-a to NOT be selected")
	}
	if m.selected[deleteSkillKey(skills[1])] {
		t.Error("expected skill-b to NOT be selected")
	}

	// Select one
	m.selected[deleteSkillKey(skills[0])] = true

	// Only skill-a should be selected
	selected := m.getSelectedSkills()
	if len(selected) != 1 {
		t.Errorf("expected 1 selected, got %d", len(selected))
	}
	if selected[0].Name != "skill-a" {
		t.Errorf("expected skill-a, got %s", selected[0].Name)
	}
}

func TestDeleteListModel_SameNameDifferentPlatforms(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "common-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
		{
			Name:     "common-skill",
			Platform: model.Cursor,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)

	// Both should have unique keys
	key1 := deleteSkillKey(skills[0])
	key2 := deleteSkillKey(skills[1])

	if key1 == key2 {
		t.Error("expected different keys for same name on different platforms")
	}

	// Select one
	m.selected[key1] = true

	selected := m.getSelectedSkills()
	if len(selected) != 1 {
		t.Errorf("expected 1 selected, got %d", len(selected))
	}
	if selected[0].Platform != model.ClaudeCode {
		t.Errorf("expected claude-code platform, got %s", selected[0].Platform)
	}
}

func TestDeleteListModel_SameNameDifferentScopes(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "common-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
		{
			Name:     "common-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeRepo,
		},
	}

	_ = NewDeleteListModel(skills) // Ensure model can be created

	// Both should have unique keys (platform + scope + name)
	key1 := deleteSkillKey(skills[0])
	key2 := deleteSkillKey(skills[1])

	if key1 == key2 {
		t.Error("expected different keys for same name in different scopes")
	}
}

func TestDeleteListModel_ClearFilter(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
		{
			Name:     "other-skill",
			Platform: model.Cursor,
			Scope:    model.ScopeRepo,
		},
	}

	m := NewDeleteListModel(skills)

	// Apply a filter
	m.filter = "test"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}

	// Clear filter with Esc
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	dm := newModel.(DeleteListModel)

	if dm.filter != "" {
		t.Errorf("expected empty filter, got '%s'", dm.filter)
	}

	if len(dm.filtered) != 2 {
		t.Errorf("expected 2 filtered skills after clearing, got %d", len(dm.filtered))
	}
}

func TestDeleteListModel_BackspaceInFilter(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)

	// Enter filter mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	dm := newModel.(DeleteListModel)

	// Type "test"
	for _, r := range "test" {
		newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		dm = newModel.(DeleteListModel)
	}

	if dm.filter != "test" {
		t.Errorf("expected filter 'test', got '%s'", dm.filter)
	}

	// Backspace
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	dm = newModel.(DeleteListModel)

	if dm.filter != "tes" {
		t.Errorf("expected filter 'tes', got '%s'", dm.filter)
	}
}

func TestDeleteListModel_NoSelectionNoConfirm(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)

	// Don't select any skills (default behavior)

	// Try to delete - should not enter confirm mode with no selection
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	dm := newModel.(DeleteListModel)

	if dm.confirmMode {
		t.Error("expected confirmMode to be false when no skills selected")
	}
}

func TestDeleteListModel_ViewRenderShortHelp(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)
	help := m.renderShortHelp()

	// Should contain key bindings
	if help == "" {
		t.Error("expected non-empty short help")
	}
}

func TestDeleteListModel_ViewRenderFullHelp(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)
	help := m.renderFullHelp()

	// Should contain navigation section
	if help == "" {
		t.Error("expected non-empty full help")
	}
}

func TestDeleteListModel_LongNameTruncation(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "this-is-a-very-long-skill-name-that-should-be-truncated",
			Description: "This is a very long description that should also be truncated to fit the table column width properly",
			Platform:    model.ClaudeCode,
			Scope:       model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)
	rows := m.skillsToRows(skills)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	// Name should be truncated to 25 chars with ellipsis
	if len(row[1]) > 25 {
		t.Errorf("expected name to be max 25 chars, got %d: %s", len(row[1]), row[1])
	}
	// Description should be truncated to 60 chars with ellipsis
	if len(row[4]) > 60 {
		t.Errorf("expected description to be max 60 chars, got %d: %s", len(row[4]), row[4])
	}
}

func TestDeleteListModel_CancelConfirmWithEsc(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewDeleteListModel(skills)

	// Select the skill first
	m.selected[deleteSkillKey(skills[0])] = true

	// Enter confirm mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	dm := newModel.(DeleteListModel)

	if !dm.confirmMode {
		t.Error("expected confirmMode to be true")
	}

	// Cancel with Esc
	newModel, _ = dm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	dm = newModel.(DeleteListModel)

	if dm.confirmMode {
		t.Error("expected confirmMode to be false after pressing Esc")
	}

	if dm.quitting {
		t.Error("expected model to not be quitting after cancel")
	}
}
