package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/model"
)

func TestNewSyncPickerModel(t *testing.T) {
	m := NewSyncPickerModel()

	if len(m.platforms) == 0 {
		t.Fatal("expected platforms to be initialized")
	}
	if len(m.sourceScopes) == 0 {
		t.Fatal("expected source scopes to be initialized")
	}
	if len(m.targetScopes) != 2 {
		t.Fatalf("expected 2 target scopes, got %d", len(m.targetScopes))
	}
	if m.phase != syncPickerPhaseSourcePlatform {
		t.Fatalf("expected source platform phase, got %v", m.phase)
	}
}

func TestSyncPickerModel_CompleteSelectionFlow(t *testing.T) {
	m := NewSyncPickerModel()

	// Source platform
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(SyncPickerModel)
	if m.phase != syncPickerPhaseSourceScope {
		t.Fatalf("expected source scope phase, got %v", m.phase)
	}

	// Source scope (default: all)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(SyncPickerModel)
	if m.phase != syncPickerPhaseTargetPlatform {
		t.Fatalf("expected target platform phase, got %v", m.phase)
	}

	// Target platform
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(SyncPickerModel)
	if m.phase != syncPickerPhaseTargetScope {
		t.Fatalf("expected target scope phase, got %v", m.phase)
	}

	// Target scope
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(SyncPickerModel)

	if cmd == nil {
		t.Fatal("expected quit command after final selection")
	}

	result := m.Result()
	if result.Action != SyncPickerActionSelect {
		t.Fatalf("expected select action, got %v", result.Action)
	}
	if result.Source == "" || result.Target == "" {
		t.Fatalf("expected source/target to be set, got %q -> %q", result.Source, result.Target)
	}
	if result.Source == result.Target {
		t.Fatalf("expected different source/target, got %q", result.Source)
	}
	if result.TargetScope != model.ScopeRepo {
		t.Fatalf("expected default target scope repo, got %q", result.TargetScope)
	}
	if len(result.SourceScopes) != 0 {
		t.Fatalf("expected empty source scopes for all, got %v", result.SourceScopes)
	}
}

func TestSyncPickerModel_CannotSelectSameTargetPlatform(t *testing.T) {
	m := NewSyncPickerModel()

	// Pick source platform and source scope first.
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(SyncPickerModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(SyncPickerModel)

	// Move cursor to source platform and try selecting it as target.
	for i, p := range m.platforms {
		if p == m.source {
			m.cursor = i
			break
		}
	}

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(SyncPickerModel)

	if m.phase != syncPickerPhaseTargetPlatform {
		t.Fatalf("expected to remain in target platform phase, got %v", m.phase)
	}
}

func TestSyncPickerModel_BackNavigation(t *testing.T) {
	m := NewSyncPickerModel()

	// Source platform -> source scope
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(SyncPickerModel)
	if m.phase != syncPickerPhaseSourceScope {
		t.Fatalf("expected source scope phase, got %v", m.phase)
	}

	// Back should return to source platform phase
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(SyncPickerModel)
	if m.phase != syncPickerPhaseSourcePlatform {
		t.Fatalf("expected source platform phase, got %v", m.phase)
	}
}
