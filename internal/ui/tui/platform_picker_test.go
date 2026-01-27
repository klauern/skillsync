package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/model"
)

func TestNewPlatformPickerModel(t *testing.T) {
	m := NewPlatformPickerModel()

	if len(m.platforms) != len(model.AllPlatforms()) {
		t.Errorf("expected %d platforms, got %d", len(model.AllPlatforms()), len(m.platforms))
	}

	if m.phase != phaseSourcePlatform {
		t.Errorf("expected phase to be phaseSourcePlatform, got %d", m.phase)
	}

	if m.cursor != 0 {
		t.Errorf("expected cursor to be 0, got %d", m.cursor)
	}
}

func TestPlatformPickerModel_Init(t *testing.T) {
	m := NewPlatformPickerModel()
	cmd := m.Init()

	if cmd != nil {
		t.Error("expected Init to return nil")
	}
}

func TestPlatformPickerModel_Update_Navigation(t *testing.T) {
	m := NewPlatformPickerModel()

	// Test down navigation
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := m.Update(downMsg)
	m = newModel.(PlatformPickerModel)

	if m.cursor != 1 {
		t.Errorf("expected cursor to be 1 after down, got %d", m.cursor)
	}

	// Test up navigation
	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ = m.Update(upMsg)
	m = newModel.(PlatformPickerModel)

	if m.cursor != 0 {
		t.Errorf("expected cursor to be 0 after up, got %d", m.cursor)
	}

	// Test cursor doesn't go negative
	newModel, _ = m.Update(upMsg)
	m = newModel.(PlatformPickerModel)

	if m.cursor != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", m.cursor)
	}
}

func TestPlatformPickerModel_Update_SourceSelection(t *testing.T) {
	m := NewPlatformPickerModel()

	// Select first platform as source
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := m.Update(enterMsg)
	m = newModel.(PlatformPickerModel)

	if m.phase != phaseTargetPlatform {
		t.Errorf("expected phase to be phaseTargetPlatform after selection, got %d", m.phase)
	}

	if m.source != model.AllPlatforms()[0] {
		t.Errorf("expected source to be %s, got %s", model.AllPlatforms()[0], m.source)
	}
}

func TestPlatformPickerModel_Update_TargetSelection(t *testing.T) {
	m := NewPlatformPickerModel()

	// Select source
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := m.Update(enterMsg)
	m = newModel.(PlatformPickerModel)

	// Move to different platform and select as target
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ = m.Update(downMsg)
	m = newModel.(PlatformPickerModel)

	newModel, cmd := m.Update(enterMsg)
	m = newModel.(PlatformPickerModel)

	// Should quit after selecting target
	if cmd == nil {
		t.Error("expected quit command after target selection")
	}

	if m.result.Action != PlatformPickerActionSelect {
		t.Errorf("expected action to be PlatformPickerActionSelect, got %d", m.result.Action)
	}

	if m.result.Source != model.AllPlatforms()[0] {
		t.Errorf("expected source to be %s, got %s", model.AllPlatforms()[0], m.result.Source)
	}

	if m.result.Target == m.result.Source {
		t.Error("expected target to be different from source")
	}
}

func TestPlatformPickerModel_Update_BackNavigation(t *testing.T) {
	m := NewPlatformPickerModel()

	// Select source
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := m.Update(enterMsg)
	m = newModel.(PlatformPickerModel)

	// Press escape to go back
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ = m.Update(escMsg)
	m = newModel.(PlatformPickerModel)

	if m.phase != phaseSourcePlatform {
		t.Errorf("expected phase to be phaseSourcePlatform after escape, got %d", m.phase)
	}
}

func TestPlatformPickerModel_Update_Quit(t *testing.T) {
	m := NewPlatformPickerModel()

	// Press q to quit
	quitMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, cmd := m.Update(quitMsg)
	m = newModel.(PlatformPickerModel)

	if cmd == nil {
		t.Error("expected quit command")
	}

	if m.result.Action != PlatformPickerActionNone {
		t.Errorf("expected action to be PlatformPickerActionNone, got %d", m.result.Action)
	}
}

func TestPlatformPickerModel_View(t *testing.T) {
	m := NewPlatformPickerModel()

	view := m.View()

	// Should contain title
	if len(view) == 0 {
		t.Error("expected non-empty view")
	}

	// Should contain platform names
	for _, p := range model.AllPlatforms() {
		if !strings.Contains(view, string(p)) {
			t.Errorf("expected view to contain platform %s", p)
		}
	}
}

func TestPlatformPickerModel_View_TargetPhase(t *testing.T) {
	m := NewPlatformPickerModel()

	// Move to target phase
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := m.Update(enterMsg)
	m = newModel.(PlatformPickerModel)

	view := m.View()

	// Should show source selection
	if !strings.Contains(view, "Source:") {
		t.Error("expected view to show Source label in target phase")
	}
}

func TestPlatformPickerModel_CannotSelectSamePlatform(t *testing.T) {
	m := NewPlatformPickerModel()

	// Select first platform as source
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := m.Update(enterMsg)
	m = newModel.(PlatformPickerModel)

	// Try to select same platform as target (cursor should be at position 0)
	// First, navigate to the source platform position
	m.cursor = 0
	for i, p := range m.platforms {
		if p == m.source {
			m.cursor = i
			break
		}
	}

	// Try to select - should not quit because same platform
	prevPhase := m.phase
	newModel, cmd := m.Update(enterMsg)
	m = newModel.(PlatformPickerModel)

	// If cursor is on source platform, selection should be blocked
	if m.platforms[m.cursor] == m.source {
		if m.phase != prevPhase {
			t.Error("should not have changed phase when selecting same platform")
		}
		if cmd != nil && m.quitting {
			t.Error("should not quit when selecting same platform as target")
		}
	}
}
