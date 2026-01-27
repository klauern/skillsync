package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/model"
)

func TestNewImportListModel(t *testing.T) {
	m := NewImportListModel()

	// Should start in file picker phase
	if m.phase != phaseFilePicker {
		t.Errorf("expected phase %v, got %v", phaseFilePicker, m.phase)
	}

	// Default target platform should be Claude Code
	if m.targetPlatform != model.ClaudeCode {
		t.Errorf("expected default platform %v, got %v", model.ClaudeCode, m.targetPlatform)
	}

	// Default target scope should be repo
	if m.targetScope != model.ScopeRepo {
		t.Errorf("expected default scope %v, got %v", model.ScopeRepo, m.targetScope)
	}

	// Selected map should be initialized
	if m.selected == nil {
		t.Error("expected selected map to be initialized")
	}

	// Should have available platforms
	if len(m.platforms) == 0 {
		t.Error("expected platforms to be populated")
	}

	// Should have available scopes
	if len(m.scopes) == 0 {
		t.Error("expected scopes to be populated")
	}
}

func TestImportListModel_Init(t *testing.T) {
	m := NewImportListModel()
	cmd := m.Init()

	// Init should return a command for file picker initialization
	if cmd == nil {
		t.Error("expected command from Init for file picker")
	}
}

func TestImportListModel_QuitKey(t *testing.T) {
	m := NewImportListModel()

	// Simulate pressing 'q'
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	im := newModel.(ImportListModel)
	if !im.quitting {
		t.Error("expected model to be quitting after pressing 'q'")
	}

	// Should return a quit command
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestImportListModel_HelpToggle(t *testing.T) {
	m := NewImportListModel()

	if m.showHelp {
		t.Error("expected showHelp to be false initially")
	}

	// Simulate pressing '?' in file picker phase
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	im := newModel.(ImportListModel)

	if !im.showHelp {
		t.Error("expected showHelp to be true after pressing '?'")
	}

	// Toggle again
	newModel, _ = im.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	im = newModel.(ImportListModel)

	if im.showHelp {
		t.Error("expected showHelp to be false after pressing '?' again")
	}
}

func TestImportListModel_SkillsToRows(t *testing.T) {
	m := NewImportListModel()
	m.selected = make(map[string]bool)

	skills := []model.Skill{
		{
			Name:        "test-skill",
			Description: "A test skill",
			Platform:    model.ClaudeCode,
			Scope:       model.ScopeUser,
		},
		{
			Name:        "another-skill",
			Description: "Another test skill",
			Platform:    model.Cursor,
			Scope:       model.ScopeRepo,
		},
	}

	// Mark one as selected
	m.selected[importSkillKey(skills[0])] = true

	rows := m.skillsToRows(skills)

	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}

	// First row should have checkbox checked
	if rows[0][0] != "[✓]" {
		t.Errorf("expected first row checkbox to be checked, got %s", rows[0][0])
	}

	// Second row should have checkbox unchecked
	if rows[1][0] != "[ ]" {
		t.Errorf("expected second row checkbox to be unchecked, got %s", rows[1][0])
	}
}

func TestImportListModel_ApplyFilter(t *testing.T) {
	m := NewImportListModel()
	m.skills = []model.Skill{
		{
			Name:        "test-skill",
			Description: "A test skill",
			Platform:    model.ClaudeCode,
		},
		{
			Name:        "cursor-skill",
			Description: "A cursor skill",
			Platform:    model.Cursor,
		},
	}
	m.filtered = m.skills
	m.selected = make(map[string]bool)
	m.initSkillTable()

	// Apply filter for "cursor"
	m.filter = "cursor"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(m.filtered))
	}

	if m.filtered[0].Name != "cursor-skill" {
		t.Errorf("expected filtered skill to be cursor-skill, got %s", m.filtered[0].Name)
	}

	// Clear filter
	m.filter = ""
	m.applyFilter()

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered skills after clear, got %d", len(m.filtered))
	}
}

func TestImportListModel_GetSelectedSkills(t *testing.T) {
	m := NewImportListModel()
	m.skills = []model.Skill{
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
	m.selected = make(map[string]bool)

	// Select only skill-two
	m.selected[importSkillKey(m.skills[1])] = true

	selected := m.getSelectedSkills()
	if len(selected) != 1 {
		t.Errorf("expected 1 selected skill, got %d", len(selected))
	}

	if selected[0].Name != "skill-two" {
		t.Errorf("expected 'skill-two', got '%s'", selected[0].Name)
	}
}

func TestImportListModel_ImportSkillKey(t *testing.T) {
	skill := model.Skill{
		Name:     "test-skill",
		Platform: model.ClaudeCode,
		Scope:    model.ScopeUser,
	}

	key := importSkillKey(skill)
	expected := "claude-code:user:test-skill"

	if key != expected {
		t.Errorf("expected key %q, got %q", expected, key)
	}
}

func TestImportListModel_Result(t *testing.T) {
	m := NewImportListModel()

	// Result should be empty initially
	result := m.Result()
	if result.Action != ImportActionNone {
		t.Errorf("expected action %v, got %v", ImportActionNone, result.Action)
	}
}

func TestImportListModel_PhaseTransitions(t *testing.T) {
	m := NewImportListModel()

	// Start in file picker phase
	if m.phase != phaseFilePicker {
		t.Errorf("expected initial phase %v, got %v", phaseFilePicker, m.phase)
	}

	// Manually set up for skill selection phase
	m.phase = phaseSkillSelection
	m.skills = []model.Skill{
		{
			Name:       "test-skill",
			Platform:   model.ClaudeCode,
			Scope:      model.ScopeUser,
			ModifiedAt: time.Now(),
		},
	}
	m.filtered = m.skills
	m.selected = map[string]bool{importSkillKey(m.skills[0]): true}
	m.initSkillTable()

	// Press back key to go back to file picker
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	im := newModel.(ImportListModel)

	if im.phase != phaseFilePicker {
		t.Errorf("expected phase %v after esc, got %v", phaseFilePicker, im.phase)
	}
}

func TestImportListModel_DestinationPhase(t *testing.T) {
	m := NewImportListModel()
	m.phase = phaseDestination
	m.platforms = model.AllPlatforms()
	m.scopes = []model.SkillScope{model.ScopeRepo, model.ScopeUser}
	m.platformCursor = 0
	m.scopeCursor = 0
	m.targetPlatform = m.platforms[0]
	m.targetScope = m.scopes[0]

	// Press right to change platform
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	im := newModel.(ImportListModel)

	if im.platformCursor != 1 {
		t.Errorf("expected platform cursor 1, got %d", im.platformCursor)
	}

	// Press space to toggle scope
	newModel, _ = im.Update(tea.KeyMsg{Type: tea.KeySpace})
	im = newModel.(ImportListModel)

	if im.scopeCursor != 1 {
		t.Errorf("expected scope cursor 1, got %d", im.scopeCursor)
	}

	// Press back to go to skill selection
	m.phase = phaseDestination
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	im = newModel.(ImportListModel)

	if im.phase != phaseSkillSelection {
		t.Errorf("expected phase %v after esc, got %v", phaseSkillSelection, im.phase)
	}
}

func TestImportListModel_ConfirmPhase(t *testing.T) {
	m := NewImportListModel()
	m.phase = phaseConfirm
	m.skills = []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Scope:    model.ScopeUser,
		},
	}
	m.selected = map[string]bool{importSkillKey(m.skills[0]): true}
	m.targetPlatform = model.ClaudeCode
	m.targetScope = model.ScopeRepo
	m.sourcePath = "/test/path"

	// Press 'n' to go back to destination
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	im := newModel.(ImportListModel)

	if im.phase != phaseDestination {
		t.Errorf("expected phase %v after n, got %v", phaseDestination, im.phase)
	}

	// Set up for confirm again
	m.phase = phaseConfirm

	// Press 'y' to confirm
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	im = newModel.(ImportListModel)

	if im.result.Action != ImportActionImport {
		t.Errorf("expected action %v, got %v", ImportActionImport, im.result.Action)
	}

	if !im.quitting {
		t.Error("expected model to be quitting after confirm")
	}

	if cmd == nil {
		t.Error("expected quit command after confirm")
	}
}

func TestImportListModel_View(t *testing.T) {
	m := NewImportListModel()

	// View in file picker phase
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	if !contains(view, "Import Skills") {
		t.Error("expected view to contain title")
	}

	if !contains(view, "Step 1/4") {
		t.Error("expected view to contain phase indicator")
	}

	// View while quitting should be empty
	m.quitting = true
	view = m.View()
	if view != "" {
		t.Error("expected empty view when quitting")
	}
}

func TestImportListModel_WindowSizeMsg(t *testing.T) {
	m := NewImportListModel()

	// Send window size message
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	im := newModel.(ImportListModel)

	if im.width != 100 {
		t.Errorf("expected width 100, got %d", im.width)
	}

	if im.height != 50 {
		t.Errorf("expected height 50, got %d", im.height)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
