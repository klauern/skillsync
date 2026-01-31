package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/model"
)

func TestNewSyncListModel(t *testing.T) {
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

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)

	if len(m.skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(m.skills))
	}

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered skills, got %d", len(m.filtered))
	}

	// All skills should be selected by default
	if len(m.selected) != 2 {
		t.Errorf("expected 2 selected skills, got %d", len(m.selected))
	}

	for _, skill := range skills {
		if !m.selected[skill.Name] {
			t.Errorf("expected skill %s to be selected", skill.Name)
		}
	}
}

func TestSyncListModel_Filter(t *testing.T) {
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

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)
	m.filter = "cursor"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}

	if m.filtered[0].Name != "cursor-skill" {
		t.Errorf("expected filtered skill to be cursor-skill, got %s", m.filtered[0].Name)
	}
}

func TestSyncListModel_FilterByScope(t *testing.T) {
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

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)
	m.filter = "repo"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}
}

func TestSyncListModel_Toggle(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)

	// Skill should be selected initially
	if !m.selected["test-skill"] {
		t.Error("expected skill to be selected initially")
	}

	// Simulate pressing space to toggle
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	sm := newModel.(SyncListModel)

	if sm.selected["test-skill"] {
		t.Error("expected skill to be deselected after toggle")
	}

	// Toggle again
	newModel, _ = sm.Update(tea.KeyMsg{Type: tea.KeySpace})
	sm = newModel.(SyncListModel)

	if !sm.selected["test-skill"] {
		t.Error("expected skill to be selected after second toggle")
	}
}

func TestSyncListModel_ToggleAll(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "skill-one",
			Platform: model.ClaudeCode,
		},
		{
			Name:     "skill-two",
			Platform: model.ClaudeCode,
		},
		{
			Name:     "skill-three",
			Platform: model.ClaudeCode,
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)

	// All should be selected initially
	selectedCount := 0
	for _, s := range m.skills {
		if m.selected[s.Name] {
			selectedCount++
		}
	}
	if selectedCount != 3 {
		t.Errorf("expected 3 selected initially, got %d", selectedCount)
	}

	// Simulate pressing 'a' to toggle all (should deselect all since all are selected)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	sm := newModel.(SyncListModel)

	selectedCount = 0
	for _, s := range sm.skills {
		if sm.selected[s.Name] {
			selectedCount++
		}
	}
	if selectedCount != 0 {
		t.Errorf("expected 0 selected after toggle all, got %d", selectedCount)
	}

	// Toggle all again (should select all)
	newModel, _ = sm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	sm = newModel.(SyncListModel)

	selectedCount = 0
	for _, s := range sm.skills {
		if sm.selected[s.Name] {
			selectedCount++
		}
	}
	if selectedCount != 3 {
		t.Errorf("expected 3 selected after second toggle all, got %d", selectedCount)
	}
}

func TestSyncListModel_GetSelectedSkills(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "skill-one",
			Platform: model.ClaudeCode,
		},
		{
			Name:     "skill-two",
			Platform: model.ClaudeCode,
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)

	// Deselect one skill
	m.selected["skill-one"] = false

	selected := m.getSelectedSkills()
	if len(selected) != 1 {
		t.Errorf("expected 1 selected skill, got %d", len(selected))
	}

	if selected[0].Name != "skill-two" {
		t.Errorf("expected 'skill-two', got '%s'", selected[0].Name)
	}
}

func TestSyncListModel_Init(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)
	cmd := m.Init()

	if cmd != nil {
		t.Error("expected nil command from Init")
	}
}

func TestSyncListModel_QuitKey(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)

	// Simulate pressing 'q'
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	sm := newModel.(SyncListModel)
	if !sm.quitting {
		t.Error("expected model to be quitting after pressing 'q'")
	}

	// Should return a quit command
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestSyncListModel_HelpToggle(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)

	if m.showHelp {
		t.Error("expected showHelp to be false initially")
	}

	// Simulate pressing '?'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	sm := newModel.(SyncListModel)

	if !sm.showHelp {
		t.Error("expected showHelp to be true after pressing '?'")
	}

	// Toggle again
	newModel, _ = sm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	sm = newModel.(SyncListModel)

	if sm.showHelp {
		t.Error("expected showHelp to be false after pressing '?' again")
	}
}

func TestSyncListModel_PreviewAction(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Content:  "Test content",
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)

	// Simulate pressing 'p' for preview
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	sm := newModel.(SyncListModel)
	result := sm.Result()

	if result.Action != SyncActionPreview {
		t.Errorf("expected SyncActionPreview, got %v", result.Action)
	}

	if result.PreviewSkill.Name != "test-skill" {
		t.Errorf("expected skill name 'test-skill', got '%s'", result.PreviewSkill.Name)
	}

	if !sm.quitting {
		t.Error("expected model to be quitting after preview action")
	}

	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestSyncListModel_ConfirmSync(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)

	// Simulate pressing 'y' to confirm
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	sm := newModel.(SyncListModel)

	// Should enter confirm mode
	if !sm.confirmMode {
		t.Error("expected confirmMode to be true after pressing 'y'")
	}

	// Confirm with 'y'
	newModel, cmd := sm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	sm = newModel.(SyncListModel)

	result := sm.Result()
	if result.Action != SyncActionSync {
		t.Errorf("expected SyncActionSync, got %v", result.Action)
	}

	if len(result.SelectedSkills) != 1 {
		t.Errorf("expected 1 selected skill, got %d", len(result.SelectedSkills))
	}

	if !sm.quitting {
		t.Error("expected model to be quitting after sync confirmation")
	}

	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestSyncListModel_CancelConfirm(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)

	// Enter confirm mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	sm := newModel.(SyncListModel)

	if !sm.confirmMode {
		t.Error("expected confirmMode to be true")
	}

	// Cancel with 'n'
	newModel, _ = sm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	sm = newModel.(SyncListModel)

	if sm.confirmMode {
		t.Error("expected confirmMode to be false after canceling")
	}

	if sm.quitting {
		t.Error("expected model to not be quitting after cancel")
	}
}

func TestSyncListResult_DefaultAction(t *testing.T) {
	m := NewSyncListModel([]model.Skill{}, model.ClaudeCode, model.Cursor)
	result := m.Result()

	if result.Action != SyncActionNone {
		t.Errorf("expected SyncActionNone, got %v", result.Action)
	}
}

func TestSyncListModel_EmptySkills(t *testing.T) {
	m := NewSyncListModel([]model.Skill{}, model.ClaudeCode, model.Cursor)

	if len(m.skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(m.skills))
	}

	// View should still work without panicking
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestSyncListModel_FilterMode(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)

	if m.filtering {
		t.Error("expected filtering to be false initially")
	}

	// Enter filter mode with '/'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	sm := newModel.(SyncListModel)

	if !sm.filtering {
		t.Error("expected filtering to be true after pressing '/'")
	}

	// Type some characters
	newModel, _ = sm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	sm = newModel.(SyncListModel)

	if sm.filter != "t" {
		t.Errorf("expected filter 't', got '%s'", sm.filter)
	}

	// Exit filter mode with enter
	newModel, _ = sm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	sm = newModel.(SyncListModel)

	if sm.filtering {
		t.Error("expected filtering to be false after pressing enter")
	}
}

func TestSyncListModel_WindowResize(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)

	// Simulate window resize
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	sm := newModel.(SyncListModel)

	if sm.width != 120 {
		t.Errorf("expected width 120, got %d", sm.width)
	}

	if sm.height != 40 {
		t.Errorf("expected height 40, got %d", sm.height)
	}
}

func TestSyncListModel_WindowResize_AdjustsColumnWidths(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "test-skill",
			Description: "A test skill description that should fit",
			Platform:    model.ClaudeCode,
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)

	// Simulate a wide window resize
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	sm := newModel.(SyncListModel)

	cols := sm.table.Columns()
	if len(cols) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(cols))
	}
	if cols[1].Width < 20 {
		t.Errorf("expected name column width >= 20, got %d", cols[1].Width)
	}
	if cols[2].Width <= 12 {
		t.Errorf("expected scope column to expand, got %d", cols[2].Width)
	}
	if cols[3].Width <= 50 {
		t.Errorf("expected description column to expand, got %d", cols[3].Width)
	}
}

func TestSyncListModel_ViewIncludesDetailPanel(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "test-skill",
			Description: "A test skill description that should appear in the panel",
			Platform:    model.ClaudeCode,
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	sm := newModel.(SyncListModel)

	view := sm.View()
	if !strings.Contains(view, "Description (selected)") {
		t.Error("expected detail panel header in view")
	}
	if !strings.Contains(view, "A test skill description") {
		t.Error("expected detail panel to include description")
	}
}

func TestRunSyncList_EmptySkills(t *testing.T) {
	result, err := RunSyncList([]model.Skill{}, model.ClaudeCode, model.Cursor)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result.Action != SyncActionNone {
		t.Errorf("expected SyncActionNone, got %v", result.Action)
	}
}

func TestSyncListModel_SkillsToRows_WithCheckbox(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "test-skill",
			Description: "A test skill description",
			Platform:    model.ClaudeCode,
			Scope:       model.ScopeUser,
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)
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
}

func TestSyncListModel_SkillsToRows_Unchecked(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewSyncListModel(skills, model.ClaudeCode, model.Cursor)
	m.selected["test-skill"] = false

	rows := m.skillsToRows(skills)
	row := rows[0]

	if row[0] != "[ ]" {
		t.Errorf("expected checkbox '[ ]', got '%s'", row[0])
	}
}
