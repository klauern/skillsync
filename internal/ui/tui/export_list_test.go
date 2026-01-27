package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/export"
	"github.com/klauern/skillsync/internal/model"
)

func TestNewExportListModel(t *testing.T) {
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

	m := NewExportListModel(skills)

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
		if !m.selected[skillKey(skill)] {
			t.Errorf("expected skill %s to be selected", skill.Name)
		}
	}

	// Default export options
	if m.format != export.FormatJSON {
		t.Errorf("expected format JSON, got %s", m.format)
	}

	if !m.includeMetadata {
		t.Error("expected includeMetadata to be true by default")
	}

	if !m.pretty {
		t.Error("expected pretty to be true by default")
	}
}

func TestExportListModel_Filter(t *testing.T) {
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

	m := NewExportListModel(skills)
	m.filter = "cursor"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}

	if m.filtered[0].Name != "cursor-skill" {
		t.Errorf("expected filtered skill to be cursor-skill, got %s", m.filtered[0].Name)
	}
}

func TestExportListModel_FilterByScope(t *testing.T) {
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

	m := NewExportListModel(skills)
	m.filter = "repo"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}
}

func TestExportListModel_FilterByPlatform(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "claude-skill",
			Platform: model.ClaudeCode,
		},
		{
			Name:     "cursor-skill",
			Platform: model.Cursor,
		},
	}

	m := NewExportListModel(skills)
	m.filter = "claude"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}

	if m.filtered[0].Name != "claude-skill" {
		t.Errorf("expected filtered skill to be claude-skill, got %s", m.filtered[0].Name)
	}
}

func TestExportListModel_Toggle(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)

	// Skill should be selected initially
	if !m.selected[skillKey(skills[0])] {
		t.Error("expected skill to be selected initially")
	}

	// Simulate pressing space to toggle
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	em := newModel.(ExportListModel)

	if em.selected[skillKey(skills[0])] {
		t.Error("expected skill to be deselected after toggle")
	}

	// Toggle again
	newModel, _ = em.Update(tea.KeyMsg{Type: tea.KeySpace})
	em = newModel.(ExportListModel)

	if !em.selected[skillKey(skills[0])] {
		t.Error("expected skill to be selected after second toggle")
	}
}

func TestExportListModel_ToggleAll(t *testing.T) {
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

	m := NewExportListModel(skills)

	// All should be selected initially
	selectedCount := 0
	for _, s := range m.skills {
		if m.selected[skillKey(s)] {
			selectedCount++
		}
	}
	if selectedCount != 3 {
		t.Errorf("expected 3 selected initially, got %d", selectedCount)
	}

	// Simulate pressing 'a' to toggle all (should deselect all since all are selected)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	em := newModel.(ExportListModel)

	selectedCount = 0
	for _, s := range em.skills {
		if em.selected[skillKey(s)] {
			selectedCount++
		}
	}
	if selectedCount != 0 {
		t.Errorf("expected 0 selected after toggle all, got %d", selectedCount)
	}

	// Toggle all again (should select all)
	newModel, _ = em.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	em = newModel.(ExportListModel)

	selectedCount = 0
	for _, s := range em.skills {
		if em.selected[skillKey(s)] {
			selectedCount++
		}
	}
	if selectedCount != 3 {
		t.Errorf("expected 3 selected after second toggle all, got %d", selectedCount)
	}
}

func TestExportListModel_GetSelectedSkills(t *testing.T) {
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

	m := NewExportListModel(skills)

	// Deselect one skill
	m.selected[skillKey(skills[0])] = false

	selected := m.getSelectedSkills()
	if len(selected) != 1 {
		t.Errorf("expected 1 selected skill, got %d", len(selected))
	}

	if selected[0].Name != "skill-two" {
		t.Errorf("expected 'skill-two', got '%s'", selected[0].Name)
	}
}

func TestExportListModel_Init(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)
	cmd := m.Init()

	if cmd != nil {
		t.Error("expected nil command from Init")
	}
}

func TestExportListModel_QuitKey(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)

	// Simulate pressing 'q'
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	em := newModel.(ExportListModel)
	if !em.quitting {
		t.Error("expected model to be quitting after pressing 'q'")
	}

	// Should return a quit command
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestExportListModel_HelpToggle(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)

	if m.showHelp {
		t.Error("expected showHelp to be false initially")
	}

	// Simulate pressing '?'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	em := newModel.(ExportListModel)

	if !em.showHelp {
		t.Error("expected showHelp to be true after pressing '?'")
	}

	// Toggle again
	newModel, _ = em.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	em = newModel.(ExportListModel)

	if em.showHelp {
		t.Error("expected showHelp to be false after pressing '?' again")
	}
}

func TestExportListModel_FormatCycle(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)

	// Default format is JSON
	if m.format != export.FormatJSON {
		t.Errorf("expected JSON format, got %s", m.format)
	}

	// Simulate pressing 'f' to cycle format
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	em := newModel.(ExportListModel)

	if em.format != export.FormatYAML {
		t.Errorf("expected YAML format, got %s", em.format)
	}

	// Cycle again
	newModel, _ = em.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	em = newModel.(ExportListModel)

	if em.format != export.FormatMarkdown {
		t.Errorf("expected Markdown format, got %s", em.format)
	}

	// Cycle back to JSON
	newModel, _ = em.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	em = newModel.(ExportListModel)

	if em.format != export.FormatJSON {
		t.Errorf("expected JSON format, got %s", em.format)
	}
}

func TestExportListModel_MetadataToggle(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)

	// Default is include metadata
	if !m.includeMetadata {
		t.Error("expected includeMetadata to be true by default")
	}

	// Simulate pressing 'm' to toggle metadata
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	em := newModel.(ExportListModel)

	if em.includeMetadata {
		t.Error("expected includeMetadata to be false after toggle")
	}

	// Toggle again
	newModel, _ = em.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	em = newModel.(ExportListModel)

	if !em.includeMetadata {
		t.Error("expected includeMetadata to be true after second toggle")
	}
}

func TestExportListModel_ConfirmExport(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)

	// Simulate pressing 'y' to confirm
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	em := newModel.(ExportListModel)

	// Should enter confirm mode
	if !em.confirmMode {
		t.Error("expected confirmMode to be true after pressing 'y'")
	}

	// Confirm with 'y'
	newModel, cmd := em.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	em = newModel.(ExportListModel)

	result := em.Result()
	if result.Action != ExportActionExport {
		t.Errorf("expected ExportActionExport, got %v", result.Action)
	}

	if len(result.SelectedSkills) != 1 {
		t.Errorf("expected 1 selected skill, got %d", len(result.SelectedSkills))
	}

	if result.Format != export.FormatJSON {
		t.Errorf("expected JSON format, got %s", result.Format)
	}

	if !result.IncludeMetadata {
		t.Error("expected IncludeMetadata to be true")
	}

	if !em.quitting {
		t.Error("expected model to be quitting after export confirmation")
	}

	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestExportListModel_CancelConfirm(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)

	// Enter confirm mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	em := newModel.(ExportListModel)

	if !em.confirmMode {
		t.Error("expected confirmMode to be true")
	}

	// Cancel with 'n'
	newModel, _ = em.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	em = newModel.(ExportListModel)

	if em.confirmMode {
		t.Error("expected confirmMode to be false after canceling")
	}

	if em.quitting {
		t.Error("expected model to not be quitting after cancel")
	}
}

func TestExportListResult_DefaultAction(t *testing.T) {
	m := NewExportListModel([]model.Skill{})
	result := m.Result()

	if result.Action != ExportActionNone {
		t.Errorf("expected ExportActionNone, got %v", result.Action)
	}
}

func TestExportListModel_EmptySkills(t *testing.T) {
	m := NewExportListModel([]model.Skill{})

	if len(m.skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(m.skills))
	}

	// View should still work without panicking
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestExportListModel_FilterMode(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)

	if m.filtering {
		t.Error("expected filtering to be false initially")
	}

	// Enter filter mode with '/'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	em := newModel.(ExportListModel)

	if !em.filtering {
		t.Error("expected filtering to be true after pressing '/'")
	}

	// Type some characters
	newModel, _ = em.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	em = newModel.(ExportListModel)

	if em.filter != "t" {
		t.Errorf("expected filter 't', got '%s'", em.filter)
	}

	// Exit filter mode with enter
	newModel, _ = em.Update(tea.KeyMsg{Type: tea.KeyEnter})
	em = newModel.(ExportListModel)

	if em.filtering {
		t.Error("expected filtering to be false after pressing enter")
	}
}

func TestExportListModel_WindowResize(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)

	// Simulate window resize
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	em := newModel.(ExportListModel)

	if em.width != 120 {
		t.Errorf("expected width 120, got %d", em.width)
	}

	if em.height != 40 {
		t.Errorf("expected height 40, got %d", em.height)
	}
}

func TestRunExportList_EmptySkills(t *testing.T) {
	result, err := RunExportList([]model.Skill{})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result.Action != ExportActionNone {
		t.Errorf("expected ExportActionNone, got %v", result.Action)
	}
}

func TestExportListModel_SkillsToRows_WithCheckbox(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "test-skill",
			Description: "A test skill description",
			Platform:    model.ClaudeCode,
			Scope:       model.ScopeUser,
		},
	}

	m := NewExportListModel(skills)
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

func TestExportListModel_SkillsToRows_Unchecked(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)
	m.selected[skillKey(skills[0])] = false

	rows := m.skillsToRows(skills)
	row := rows[0]

	if row[0] != "[ ]" {
		t.Errorf("expected checkbox '[ ]', got '%s'", row[0])
	}
}

func TestSkillKey(t *testing.T) {
	skill := model.Skill{
		Name:     "test-skill",
		Platform: model.ClaudeCode,
	}

	key := skillKey(skill)
	expected := "claude-code:test-skill"

	if key != expected {
		t.Errorf("expected key '%s', got '%s'", expected, key)
	}
}

func TestExportListModel_SamePlatformDifferentNames(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "skill-a",
			Platform: model.ClaudeCode,
		},
		{
			Name:     "skill-b",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)

	// Both should be selected by default
	if !m.selected[skillKey(skills[0])] {
		t.Error("expected skill-a to be selected")
	}
	if !m.selected[skillKey(skills[1])] {
		t.Error("expected skill-b to be selected")
	}

	// Deselect one
	m.selected[skillKey(skills[0])] = false

	// Only skill-b should be selected
	selected := m.getSelectedSkills()
	if len(selected) != 1 {
		t.Errorf("expected 1 selected, got %d", len(selected))
	}
	if selected[0].Name != "skill-b" {
		t.Errorf("expected skill-b, got %s", selected[0].Name)
	}
}

func TestExportListModel_SameNameDifferentPlatforms(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "common-skill",
			Platform: model.ClaudeCode,
		},
		{
			Name:     "common-skill",
			Platform: model.Cursor,
		},
	}

	m := NewExportListModel(skills)

	// Both should have unique keys and be selected
	key1 := skillKey(skills[0])
	key2 := skillKey(skills[1])

	if key1 == key2 {
		t.Error("expected different keys for same name on different platforms")
	}

	if !m.selected[key1] {
		t.Error("expected claude-code:common-skill to be selected")
	}
	if !m.selected[key2] {
		t.Error("expected cursor:common-skill to be selected")
	}

	// Deselect one, the other should remain
	m.selected[key1] = false

	selected := m.getSelectedSkills()
	if len(selected) != 1 {
		t.Errorf("expected 1 selected, got %d", len(selected))
	}
	if selected[0].Platform != model.Cursor {
		t.Errorf("expected cursor platform, got %s", selected[0].Platform)
	}
}

func TestExportListModel_ClearFilter(t *testing.T) {
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

	m := NewExportListModel(skills)

	// Apply a filter
	m.filter = "test"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}

	// Clear filter with Esc
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	em := newModel.(ExportListModel)

	if em.filter != "" {
		t.Errorf("expected empty filter, got '%s'", em.filter)
	}

	if len(em.filtered) != 2 {
		t.Errorf("expected 2 filtered skills after clearing, got %d", len(em.filtered))
	}
}

func TestExportListModel_BackspaceInFilter(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)

	// Enter filter mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	em := newModel.(ExportListModel)

	// Type "test"
	for _, r := range "test" {
		newModel, _ = em.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		em = newModel.(ExportListModel)
	}

	if em.filter != "test" {
		t.Errorf("expected filter 'test', got '%s'", em.filter)
	}

	// Backspace
	newModel, _ = em.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	em = newModel.(ExportListModel)

	if em.filter != "tes" {
		t.Errorf("expected filter 'tes', got '%s'", em.filter)
	}
}

func TestExportListModel_NoSelectionNoConfirm(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)

	// Deselect all skills
	m.selected[skillKey(skills[0])] = false

	// Try to confirm - should not enter confirm mode with no selection
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	em := newModel.(ExportListModel)

	if em.confirmMode {
		t.Error("expected confirmMode to be false when no skills selected")
	}
}

func TestExportListModel_ViewRenderShortHelp(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)
	help := m.renderShortHelp()

	// Should contain key bindings
	if help == "" {
		t.Error("expected non-empty short help")
	}
}

func TestExportListModel_ViewRenderFullHelp(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)
	help := m.renderFullHelp()

	// Should contain navigation section
	if help == "" {
		t.Error("expected non-empty full help")
	}
}

func TestExportListModel_LongNameTruncation(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "this-is-a-very-long-skill-name-that-should-be-truncated",
			Description: "This is a very long description that should also be truncated to fit the table column width properly",
			Platform:    model.ClaudeCode,
		},
	}

	m := NewExportListModel(skills)
	rows := m.skillsToRows(skills)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	// Name should be truncated to 25 chars with ellipsis
	if len(row[1]) > 25 {
		t.Errorf("expected name to be max 25 chars, got %d: %s", len(row[1]), row[1])
	}
	// Description should be truncated to 40 chars with ellipsis
	if len(row[4]) > 40 {
		t.Errorf("expected description to be max 40 chars, got %d: %s", len(row[4]), row[4])
	}
}
