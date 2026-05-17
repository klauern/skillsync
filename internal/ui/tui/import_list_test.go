package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/model"
)

func testImportSkills() []model.Skill {
	return []model.Skill{
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
}

func newSkillSelectionImportModel(skills []model.Skill) ImportListModel {
	m := NewImportListModel()
	m.state.phase = phaseSkillSelection
	m.state.sourcePath = "/tmp/source"
	m.state.selected = make(map[string]bool, len(skills))
	for _, skill := range skills {
		m.state.selected[importSkillKey(skill)] = true
	}
	m.ListModel = buildImportSkillListModel(m.state, skills)
	m.ListModel.showHelp = m.state.showHelp
	return m
}

func TestNewImportListModel(t *testing.T) {
	m := NewImportListModel()

	if m.state.phase != phaseFilePicker {
		t.Errorf("expected phase %v, got %v", phaseFilePicker, m.state.phase)
	}
	if m.state.targetPlatform != model.ClaudeCode {
		t.Errorf("expected default platform %v, got %v", model.ClaudeCode, m.state.targetPlatform)
	}
	if m.state.targetScope != model.ScopeRepo {
		t.Errorf("expected default scope %v, got %v", model.ScopeRepo, m.state.targetScope)
	}
	if m.state.selected == nil {
		t.Fatal("expected selected map to be initialized")
	}
	if len(m.state.platforms) == 0 {
		t.Fatal("expected platforms to be populated")
	}
	if len(m.state.scopes) == 0 {
		t.Fatal("expected scopes to be populated")
	}
}

func TestImportListModel_Init(t *testing.T) {
	m := NewImportListModel()
	if cmd := m.Init(); cmd == nil {
		t.Fatal("expected command from Init for file picker")
	}
}

func TestImportListModel_FilePickerHelpToggle(t *testing.T) {
	m := NewImportListModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	im := updated.(ImportListModel)
	if !im.state.showHelp {
		t.Fatal("expected showHelp to be true after pressing ?")
	}

	updated, _ = im.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	im = updated.(ImportListModel)
	if im.state.showHelp {
		t.Fatal("expected showHelp to be false after pressing ? again")
	}
}

func TestImportListModel_SkillSelectionTogglesAndAdvance(t *testing.T) {
	skills := testImportSkills()
	m := newSkillSelectionImportModel(skills)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	im := updated.(ImportListModel)
	if im.state.selected[importSkillKey(skills[0])] {
		t.Fatal("expected current skill to be toggled off")
	}
	if im.ListModel.table.Rows()[0][0] != "[ ]" {
		t.Fatalf("expected first checkbox to be unchecked, got %q", im.ListModel.table.Rows()[0][0])
	}

	updated, _ = im.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	im = updated.(ImportListModel)
	for _, skill := range skills {
		if !im.state.selected[importSkillKey(skill)] {
			t.Fatalf("expected %s to be selected after toggle-all", skill.Name)
		}
	}

	updated, _ = im.Update(tea.KeyMsg{Type: tea.KeyEnter})
	im = updated.(ImportListModel)
	if im.state.phase != phaseDestination {
		t.Fatalf("expected phase %v after enter, got %v", phaseDestination, im.state.phase)
	}
}

func TestImportListModel_SkillSelectionBackAndClearFilter(t *testing.T) {
	skills := testImportSkills()
	m := newSkillSelectionImportModel(skills)
	m.ListModel.filter = "test"
	m.ListModel.filtering = false

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	im := updated.(ImportListModel)
	if im.ListModel.filter != "" {
		t.Fatal("expected ctrl+u to clear the filter")
	}

	updated, _ = im.Update(tea.KeyMsg{Type: tea.KeyEsc})
	im = updated.(ImportListModel)
	if im.state.phase != phaseFilePicker {
		t.Fatalf("expected phase %v after esc, got %v", phaseFilePicker, im.state.phase)
	}
}

func TestImportListModel_DestinationAndConfirmFlow(t *testing.T) {
	skills := testImportSkills()
	m := newSkillSelectionImportModel(skills)
	m.state.phase = phaseDestination
	m.state.selected = map[string]bool{
		importSkillKey(skills[0]): true,
		importSkillKey(skills[1]): true,
	}
	m.ListModel = buildImportSkillListModel(m.state, skills)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	im := updated.(ImportListModel)
	if im.state.targetPlatform != im.state.platforms[1] {
		t.Fatalf("expected target platform to advance, got %v", im.state.targetPlatform)
	}

	updated, _ = im.Update(tea.KeyMsg{Type: tea.KeyTab})
	im = updated.(ImportListModel)
	if im.state.targetScope != im.state.scopes[1] {
		t.Fatalf("expected target scope to cycle, got %v", im.state.targetScope)
	}

	updated, _ = im.Update(tea.KeyMsg{Type: tea.KeyEnter})
	im = updated.(ImportListModel)
	if im.state.phase != phaseConfirm {
		t.Fatalf("expected phase %v after enter, got %v", phaseConfirm, im.state.phase)
	}

	updated, _ = im.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	im = updated.(ImportListModel)
	if im.state.phase != phaseDestination {
		t.Fatalf("expected phase %v after n, got %v", phaseDestination, im.state.phase)
	}

	updated, cmd := im.updateConfirm(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	im = updated.(ImportListModel)
	if cmd == nil {
		t.Fatal("expected quit command after confirming import")
	}
	if im.state.result.Action != ImportActionImport {
		t.Fatalf("expected action %v, got %v", ImportActionImport, im.state.result.Action)
	}
	if len(im.state.result.SelectedSkills) != len(skills) {
		t.Fatalf("expected %d selected skills, got %d", len(skills), len(im.state.result.SelectedSkills))
	}
	if im.state.result.SourcePath != m.state.sourcePath {
		t.Fatalf("expected source path %q, got %q", m.state.sourcePath, im.state.result.SourcePath)
	}
}

func TestImportListModel_View(t *testing.T) {
	m := NewImportListModel()
	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty file picker view")
	}
	if !strings.Contains(view, "📥 Import Skills") {
		t.Fatal("expected view to contain the title")
	}
	if !strings.Contains(view, "Step 1/4: Select Source") {
		t.Fatal("expected view to contain the file picker phase indicator")
	}

	skills := testImportSkills()
	m = newSkillSelectionImportModel(skills)
	view = m.View()
	if !strings.Contains(view, "Step 2/4: Select Skills") {
		t.Fatal("expected skill selection view to contain the phase indicator")
	}
	if !strings.Contains(view, skills[0].Name) {
		t.Fatal("expected skill selection view to include a skill name")
	}
}

func TestImportListModel_Result(t *testing.T) {
	m := NewImportListModel()
	result := m.Result()
	if result.Action != ImportActionNone {
		t.Fatalf("expected action %v, got %v", ImportActionNone, result.Action)
	}
}
