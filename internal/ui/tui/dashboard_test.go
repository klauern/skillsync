package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewDashboardModel(t *testing.T) {
	model := NewDashboardModel()

	if len(model.items) == 0 {
		t.Error("expected menu items to be populated")
	}

	if model.cursor != 0 {
		t.Errorf("expected cursor to be 0, got %d", model.cursor)
	}

	if model.quitting {
		t.Error("expected quitting to be false initially")
	}
}

func TestDashboardModel_Init(t *testing.T) {
	model := NewDashboardModel()
	cmd := model.Init()

	if cmd != nil {
		t.Error("expected nil command from Init")
	}
}

func TestDashboardModel_Navigation(t *testing.T) {
	model := NewDashboardModel()

	// Initially at position 0
	if model.cursor != 0 {
		t.Errorf("expected cursor to be 0, got %d", model.cursor)
	}

	// Move down
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m := newModel.(DashboardModel)

	if m.cursor != 1 {
		t.Errorf("expected cursor to be 1 after pressing 'j', got %d", m.cursor)
	}

	// Move down with arrow key
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(DashboardModel)

	if m.cursor != 2 {
		t.Errorf("expected cursor to be 2 after pressing down, got %d", m.cursor)
	}

	// Move up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(DashboardModel)

	if m.cursor != 1 {
		t.Errorf("expected cursor to be 1 after pressing 'k', got %d", m.cursor)
	}

	// Move up with arrow key
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(DashboardModel)

	if m.cursor != 0 {
		t.Errorf("expected cursor to be 0 after pressing up, got %d", m.cursor)
	}
}

func TestDashboardModel_NavigationBounds(t *testing.T) {
	model := NewDashboardModel()

	// Try to move up at the top - should stay at 0
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	m := newModel.(DashboardModel)

	if m.cursor != 0 {
		t.Errorf("expected cursor to stay at 0 when at top, got %d", m.cursor)
	}

	// Move to the last item
	for i := 0; i < len(model.items); i++ {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = newModel.(DashboardModel)
	}

	expectedLast := len(model.items) - 1
	if m.cursor != expectedLast {
		t.Errorf("expected cursor to be at last item (%d), got %d", expectedLast, m.cursor)
	}

	// Try to move down past the end - should stay at last
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(DashboardModel)

	if m.cursor != expectedLast {
		t.Errorf("expected cursor to stay at last item (%d), got %d", expectedLast, m.cursor)
	}
}

func TestDashboardModel_Selection(t *testing.T) {
	model := NewDashboardModel()

	// Select first item (Discover)
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(DashboardModel)

	if !m.quitting {
		t.Error("expected model to be quitting after selection")
	}

	if m.result.View != DashboardViewDiscover {
		t.Errorf("expected DashboardViewDiscover, got %v", m.result.View)
	}

	if cmd == nil {
		t.Error("expected quit command after selection")
	}
}

func TestDashboardModel_SelectionWithSpace(t *testing.T) {
	model := NewDashboardModel()

	// Select with space
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m := newModel.(DashboardModel)

	if !m.quitting {
		t.Error("expected model to be quitting after selection with space")
	}

	if m.result.View != DashboardViewDiscover {
		t.Errorf("expected DashboardViewDiscover, got %v", m.result.View)
	}

	if cmd == nil {
		t.Error("expected quit command after selection")
	}
}

func TestDashboardModel_QuitKey(t *testing.T) {
	model := NewDashboardModel()

	// Quit with 'q'
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m := newModel.(DashboardModel)

	if !m.quitting {
		t.Error("expected model to be quitting after pressing 'q'")
	}

	if m.result.View != DashboardViewNone {
		t.Errorf("expected DashboardViewNone after quit, got %v", m.result.View)
	}

	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestDashboardModel_HelpToggle(t *testing.T) {
	model := NewDashboardModel()

	if model.showHelp {
		t.Error("expected showHelp to be false initially")
	}

	// Toggle help on
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m := newModel.(DashboardModel)

	if !m.showHelp {
		t.Error("expected showHelp to be true after pressing '?'")
	}

	// Toggle help off
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newModel.(DashboardModel)

	if m.showHelp {
		t.Error("expected showHelp to be false after pressing '?' again")
	}
}

func TestDashboardModel_View(t *testing.T) {
	model := NewDashboardModel()

	view := model.View()

	if view == "" {
		t.Error("expected non-empty view")
	}

	// Should contain the title
	if !containsString(view, "Skillsync Dashboard") {
		t.Error("expected view to contain 'Skillsync Dashboard'")
	}

	// Should contain menu items
	if !containsString(view, "Discover Skills") {
		t.Error("expected view to contain 'Discover Skills'")
	}
}

func TestDashboardModel_ViewQuitting(t *testing.T) {
	model := NewDashboardModel()
	model.quitting = true

	view := model.View()

	if view != "" {
		t.Error("expected empty view when quitting")
	}
}

func TestDashboardModel_WindowSize(t *testing.T) {
	model := NewDashboardModel()

	newModel, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m := newModel.(DashboardModel)

	if m.width != 100 {
		t.Errorf("expected width to be 100, got %d", m.width)
	}

	if m.height != 50 {
		t.Errorf("expected height to be 50, got %d", m.height)
	}
}

func TestDashboardResult_DefaultView(t *testing.T) {
	model := NewDashboardModel()
	result := model.Result()

	if result.View != DashboardViewNone {
		t.Errorf("expected DashboardViewNone, got %v", result.View)
	}
}

func TestDefaultMenuItems(t *testing.T) {
	items := defaultMenuItems()

	if len(items) == 0 {
		t.Error("expected non-empty menu items")
	}

	// Verify each item has required fields
	for i, item := range items {
		if item.Title == "" {
			t.Errorf("item %d has empty title", i)
		}
		if item.Description == "" {
			t.Errorf("item %d has empty description", i)
		}
	}

	// Verify expected items exist
	expectedTitles := []string{
		"Discover Skills",
		"Backup Management",
		"Sync Operations",
		"Configuration",
	}

	for _, expected := range expectedTitles {
		found := false
		for _, item := range items {
			if item.Title == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find menu item '%s'", expected)
		}
	}
}

func TestDashboardModel_SelectEachView(t *testing.T) {
	items := defaultMenuItems()

	for i, item := range items {
		model := NewDashboardModel()
		model.cursor = i

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m := newModel.(DashboardModel)

		if m.result.View != item.View {
			t.Errorf("selecting item %d (%s): expected view %v, got %v",
				i, item.Title, item.View, m.result.View)
		}
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || containsSubstr(s, substr)))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
