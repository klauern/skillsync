package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/model"
)

func TestNewPromoteDemoteListModel(t *testing.T) {
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
			Path:        "/project/.cursor/skills/another.md",
			Scope:       model.ScopeRepo,
			ModifiedAt:  time.Now().Add(-24 * time.Hour),
		},
	}

	m := NewPromoteDemoteListModel(skills)

	if len(m.skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(m.skills))
	}

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered skills, got %d", len(m.filtered))
	}

	// No skills should be selected by default
	if len(m.selected) != 0 {
		t.Errorf("expected 0 selected skills by default, got %d", len(m.selected))
	}
}

func TestNewPromoteDemoteListModel_FiltersMovableScopes(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "user-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser, // Movable (can be demoted)
		},
		{
			Name:     "repo-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeRepo, // Movable (can be promoted)
		},
		{
			Name:     "admin-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeAdmin, // NOT movable
		},
		{
			Name:     "system-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeSystem, // NOT movable
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// Should only include repo and user scopes
	if len(m.skills) != 2 {
		t.Errorf("expected 2 movable skills, got %d", len(m.skills))
	}

	for _, s := range m.skills {
		if s.Scope != model.ScopeUser && s.Scope != model.ScopeRepo {
			t.Errorf("expected only user or repo scope, got %s", s.Scope)
		}
	}
}

func TestPromoteDemoteListModel_Filter(t *testing.T) {
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

	m := NewPromoteDemoteListModel(skills)
	m.filter = "cursor"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}

	if m.filtered[0].Name != "cursor-skill" {
		t.Errorf("expected filtered skill to be cursor-skill, got %s", m.filtered[0].Name)
	}
}

func TestPromoteDemoteListModel_Toggle(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// Skill should NOT be selected initially
	if m.selected[promoteDemoteSkillKey(skills[0])] {
		t.Error("expected skill to NOT be selected initially")
	}

	// Simulate pressing space to toggle
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	pdm := newModel.(PromoteDemoteListModel)

	if !pdm.selected[promoteDemoteSkillKey(skills[0])] {
		t.Error("expected skill to be selected after toggle")
	}

	// Toggle again
	newModel, _ = pdm.Update(tea.KeyMsg{Type: tea.KeySpace})
	pdm = newModel.(PromoteDemoteListModel)

	if pdm.selected[promoteDemoteSkillKey(skills[0])] {
		t.Error("expected skill to be deselected after second toggle")
	}
}

func TestPromoteDemoteListModel_ToggleAll(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "skill-one",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
		{
			Name:     "skill-two",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeRepo,
		},
		{
			Name:     "skill-three",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// None should be selected initially
	selectedCount := 0
	for _, s := range m.skills {
		if m.selected[promoteDemoteSkillKey(s)] {
			selectedCount++
		}
	}
	if selectedCount != 0 {
		t.Errorf("expected 0 selected initially, got %d", selectedCount)
	}

	// Simulate pressing 'a' to toggle all
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	pdm := newModel.(PromoteDemoteListModel)

	selectedCount = 0
	for _, s := range pdm.skills {
		if pdm.selected[promoteDemoteSkillKey(s)] {
			selectedCount++
		}
	}
	if selectedCount != 3 {
		t.Errorf("expected 3 selected after toggle all, got %d", selectedCount)
	}

	// Toggle all again (should deselect all)
	newModel, _ = pdm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	pdm = newModel.(PromoteDemoteListModel)

	selectedCount = 0
	for _, s := range pdm.skills {
		if pdm.selected[promoteDemoteSkillKey(s)] {
			selectedCount++
		}
	}
	if selectedCount != 0 {
		t.Errorf("expected 0 selected after second toggle all, got %d", selectedCount)
	}
}

func TestPromoteDemoteListModel_GetPromotableSelectedSkills(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "repo-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeRepo, // Can be promoted
		},
		{
			Name:     "user-skill",
			Platform: model.Cursor,
			Scope:    model.ScopeUser, // Cannot be promoted (already at user)
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// Select both skills
	m.selected[promoteDemoteSkillKey(skills[0])] = true
	m.selected[promoteDemoteSkillKey(skills[1])] = true

	promotable := m.getPromotableSelectedSkills()
	if len(promotable) != 1 {
		t.Errorf("expected 1 promotable skill, got %d", len(promotable))
	}

	if promotable[0].Name != "repo-skill" {
		t.Errorf("expected 'repo-skill', got '%s'", promotable[0].Name)
	}
}

func TestPromoteDemoteListModel_GetDemotableSelectedSkills(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "repo-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeRepo, // Cannot be demoted (already at repo)
		},
		{
			Name:     "user-skill",
			Platform: model.Cursor,
			Scope:    model.ScopeUser, // Can be demoted
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// Select both skills
	m.selected[promoteDemoteSkillKey(skills[0])] = true
	m.selected[promoteDemoteSkillKey(skills[1])] = true

	demotable := m.getDemotableSelectedSkills()
	if len(demotable) != 1 {
		t.Errorf("expected 1 demotable skill, got %d", len(demotable))
	}

	if demotable[0].Name != "user-skill" {
		t.Errorf("expected 'user-skill', got '%s'", demotable[0].Name)
	}
}

func TestPromoteDemoteListModel_Init(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)
	cmd := m.Init()

	if cmd != nil {
		t.Error("expected nil command from Init")
	}
}

func TestPromoteDemoteListModel_QuitKey(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// Simulate pressing 'q'
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	pdm := newModel.(PromoteDemoteListModel)
	if !pdm.quitting {
		t.Error("expected model to be quitting after pressing 'q'")
	}

	// Should return a quit command
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestPromoteDemoteListModel_HelpToggle(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)

	if m.showHelp {
		t.Error("expected showHelp to be false initially")
	}

	// Simulate pressing '?'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	pdm := newModel.(PromoteDemoteListModel)

	if !pdm.showHelp {
		t.Error("expected showHelp to be true after pressing '?'")
	}

	// Toggle again
	newModel, _ = pdm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	pdm = newModel.(PromoteDemoteListModel)

	if pdm.showHelp {
		t.Error("expected showHelp to be false after pressing '?' again")
	}
}

func TestPromoteDemoteListModel_ToggleMoveMode(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)

	if m.removeSource {
		t.Error("expected removeSource to be false initially (copy mode)")
	}

	// Simulate pressing 'm'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	pdm := newModel.(PromoteDemoteListModel)

	if !pdm.removeSource {
		t.Error("expected removeSource to be true after pressing 'm' (move mode)")
	}

	// Toggle again
	newModel, _ = pdm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	pdm = newModel.(PromoteDemoteListModel)

	if pdm.removeSource {
		t.Error("expected removeSource to be false after pressing 'm' again (copy mode)")
	}
}

func TestPromoteDemoteListModel_ConfirmPromote(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "repo-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeRepo, // Can be promoted
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// Select the skill first
	m.selected[promoteDemoteSkillKey(skills[0])] = true

	// Simulate pressing 'p' to promote
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	pdm := newModel.(PromoteDemoteListModel)

	// Should enter confirm mode with promote action
	if !pdm.confirmMode {
		t.Error("expected confirmMode to be true after pressing 'p'")
	}

	if pdm.confirmAction != PromoteDemoteActionPromote {
		t.Errorf("expected confirmAction to be Promote, got %v", pdm.confirmAction)
	}

	// Confirm with 'y'
	newModel, cmd := pdm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	pdm = newModel.(PromoteDemoteListModel)

	result := pdm.Result()
	if result.Action != PromoteDemoteActionPromote {
		t.Errorf("expected PromoteDemoteActionPromote, got %v", result.Action)
	}

	if len(result.SelectedSkills) != 1 {
		t.Errorf("expected 1 selected skill, got %d", len(result.SelectedSkills))
	}

	if !pdm.quitting {
		t.Error("expected model to be quitting after promote confirmation")
	}

	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestPromoteDemoteListModel_ConfirmDemote(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "user-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser, // Can be demoted
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// Select the skill first
	m.selected[promoteDemoteSkillKey(skills[0])] = true

	// Simulate pressing 'd' to demote
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	pdm := newModel.(PromoteDemoteListModel)

	// Should enter confirm mode with demote action
	if !pdm.confirmMode {
		t.Error("expected confirmMode to be true after pressing 'd'")
	}

	if pdm.confirmAction != PromoteDemoteActionDemote {
		t.Errorf("expected confirmAction to be Demote, got %v", pdm.confirmAction)
	}

	// Confirm with 'y'
	newModel, _ = pdm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	pdm = newModel.(PromoteDemoteListModel)

	result := pdm.Result()
	if result.Action != PromoteDemoteActionDemote {
		t.Errorf("expected PromoteDemoteActionDemote, got %v", result.Action)
	}
}

func TestPromoteDemoteListModel_CancelConfirm(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "user-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// Select the skill first
	m.selected[promoteDemoteSkillKey(skills[0])] = true

	// Enter confirm mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	pdm := newModel.(PromoteDemoteListModel)

	if !pdm.confirmMode {
		t.Error("expected confirmMode to be true")
	}

	// Cancel with 'n'
	newModel, _ = pdm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	pdm = newModel.(PromoteDemoteListModel)

	if pdm.confirmMode {
		t.Error("expected confirmMode to be false after canceling")
	}

	if pdm.quitting {
		t.Error("expected model to not be quitting after cancel")
	}

	if pdm.confirmAction != PromoteDemoteActionNone {
		t.Errorf("expected confirmAction to be None after cancel, got %v", pdm.confirmAction)
	}
}

func TestPromoteDemoteListResult_DefaultAction(t *testing.T) {
	m := NewPromoteDemoteListModel([]model.Skill{})
	result := m.Result()

	if result.Action != PromoteDemoteActionNone {
		t.Errorf("expected PromoteDemoteActionNone, got %v", result.Action)
	}
}

func TestPromoteDemoteListModel_EmptySkills(t *testing.T) {
	m := NewPromoteDemoteListModel([]model.Skill{})

	if len(m.skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(m.skills))
	}

	// View should still work without panicking
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestPromoteDemoteListModel_FilterMode(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)

	if m.filtering {
		t.Error("expected filtering to be false initially")
	}

	// Enter filter mode with '/'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	pdm := newModel.(PromoteDemoteListModel)

	if !pdm.filtering {
		t.Error("expected filtering to be true after pressing '/'")
	}

	// Type some characters
	newModel, _ = pdm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	pdm = newModel.(PromoteDemoteListModel)

	if pdm.filter != "t" {
		t.Errorf("expected filter 't', got '%s'", pdm.filter)
	}

	// Exit filter mode with enter
	newModel, _ = pdm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pdm = newModel.(PromoteDemoteListModel)

	if pdm.filtering {
		t.Error("expected filtering to be false after pressing enter")
	}
}

func TestPromoteDemoteListModel_WindowResize(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// Simulate window resize
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	pdm := newModel.(PromoteDemoteListModel)

	if pdm.width != 120 {
		t.Errorf("expected width 120, got %d", pdm.width)
	}

	if pdm.height != 40 {
		t.Errorf("expected height 40, got %d", pdm.height)
	}
}

func TestRunPromoteDemoteList_EmptySkills(t *testing.T) {
	result, err := RunPromoteDemoteList([]model.Skill{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result.Action != PromoteDemoteActionNone {
		t.Errorf("expected PromoteDemoteActionNone, got %v", result.Action)
	}
}

func TestRunPromoteDemoteList_NoMovableSkills(t *testing.T) {
	// Skills with non-movable scopes
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

	result, err := RunPromoteDemoteList(skills)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Should return empty result since no movable skills
	if result.Action != PromoteDemoteActionNone {
		t.Errorf("expected PromoteDemoteActionNone, got %v", result.Action)
	}
}

func TestPromoteDemoteSkillKey(t *testing.T) {
	skill := model.Skill{
		Name:     "test-skill",
		Platform: model.ClaudeCode,
		Scope:    model.ScopeUser,
	}

	key := promoteDemoteSkillKey(skill)
	expected := "claude-code:user:test-skill"

	if key != expected {
		t.Errorf("expected key '%s', got '%s'", expected, key)
	}
}

func TestPromoteDemoteListModel_NoPromotableSkillsNoConfirm(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "user-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser, // Already at user, can't be promoted
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// Select the skill
	m.selected[promoteDemoteSkillKey(skills[0])] = true

	// Try to promote - should not enter confirm mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	pdm := newModel.(PromoteDemoteListModel)

	if pdm.confirmMode {
		t.Error("expected confirmMode to be false when no promotable skills selected")
	}
}

func TestPromoteDemoteListModel_NoDemotableSkillsNoConfirm(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "repo-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeRepo, // Already at repo, can't be demoted
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// Select the skill
	m.selected[promoteDemoteSkillKey(skills[0])] = true

	// Try to demote - should not enter confirm mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	pdm := newModel.(PromoteDemoteListModel)

	if pdm.confirmMode {
		t.Error("expected confirmMode to be false when no demotable skills selected")
	}
}

func TestPromoteDemoteListModel_ResultIncludesRemoveSource(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "user-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// Enable move mode
	m.removeSource = true

	// Select the skill
	m.selected[promoteDemoteSkillKey(skills[0])] = true

	// Demote and confirm
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	pdm := newModel.(PromoteDemoteListModel)

	newModel, _ = pdm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	pdm = newModel.(PromoteDemoteListModel)

	result := pdm.Result()
	if !result.RemoveSource {
		t.Error("expected RemoveSource to be true in result")
	}
}

func TestPromoteDemoteListModel_ViewRenderShortHelp(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)
	help := m.renderShortHelp()

	// Should contain key bindings
	if help == "" {
		t.Error("expected non-empty short help")
	}
}

func TestPromoteDemoteListModel_ViewRenderFullHelp(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)
	help := m.renderFullHelp()

	// Should contain navigation section
	if help == "" {
		t.Error("expected non-empty full help")
	}
}

func TestPromoteDemoteListModel_ClearFilter(t *testing.T) {
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

	m := NewPromoteDemoteListModel(skills)

	// Apply a filter
	m.filter = "test"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}

	// Clear filter with Esc
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	pdm := newModel.(PromoteDemoteListModel)

	if pdm.filter != "" {
		t.Errorf("expected empty filter, got '%s'", pdm.filter)
	}

	if len(pdm.filtered) != 2 {
		t.Errorf("expected 2 filtered skills after clearing, got %d", len(pdm.filtered))
	}
}

func TestPromoteDemoteListModel_BackspaceInFilter(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// Enter filter mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	pdm := newModel.(PromoteDemoteListModel)

	// Type "test"
	for _, r := range "test" {
		newModel, _ = pdm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		pdm = newModel.(PromoteDemoteListModel)
	}

	if pdm.filter != "test" {
		t.Errorf("expected filter 'test', got '%s'", pdm.filter)
	}

	// Backspace
	newModel, _ = pdm.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	pdm = newModel.(PromoteDemoteListModel)

	if pdm.filter != "tes" {
		t.Errorf("expected filter 'tes', got '%s'", pdm.filter)
	}
}

func TestPromoteDemoteListModel_CancelConfirmWithEsc(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "user-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)

	// Select the skill first
	m.selected[promoteDemoteSkillKey(skills[0])] = true

	// Enter confirm mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	pdm := newModel.(PromoteDemoteListModel)

	if !pdm.confirmMode {
		t.Error("expected confirmMode to be true")
	}

	// Cancel with Esc
	newModel, _ = pdm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	pdm = newModel.(PromoteDemoteListModel)

	if pdm.confirmMode {
		t.Error("expected confirmMode to be false after pressing Esc")
	}

	if pdm.quitting {
		t.Error("expected model to not be quitting after cancel")
	}
}

func TestPromoteDemoteListModel_SkillsToRows_ShowsTargetScope(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "repo-skill",
			Description: "A repo skill",
			Platform:    model.ClaudeCode,
			Scope:       model.ScopeRepo,
		},
		{
			Name:        "user-skill",
			Description: "A user skill",
			Platform:    model.ClaudeCode,
			Scope:       model.ScopeUser,
		},
	}

	m := NewPromoteDemoteListModel(skills)
	rows := m.skillsToRows(skills)

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	// Check repo skill row - target should be "→ user"
	repoRow := rows[0]
	if repoRow[4] != "→ user" {
		t.Errorf("expected target '→ user' for repo skill, got '%s'", repoRow[4])
	}

	// Check user skill row - target should be "→ repo"
	userRow := rows[1]
	if userRow[4] != "→ repo" {
		t.Errorf("expected target '→ repo' for user skill, got '%s'", userRow[4])
	}
}
