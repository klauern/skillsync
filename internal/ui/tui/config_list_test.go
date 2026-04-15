package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/config"
)

func TestNewConfigListModel(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	// Should have items for all config sections
	if len(m.items) == 0 {
		t.Error("expected config items, got none")
	}

	if len(m.filtered) != len(m.items) {
		t.Errorf("expected filtered items to match items, got %d vs %d", len(m.filtered), len(m.items))
	}

	// Should not be modified initially
	if m.modified {
		t.Error("expected modified to be false initially")
	}

	// Should have default config
	if m.cfg == nil {
		t.Error("expected cfg to be set")
	}

	if m.defaultCfg == nil {
		t.Error("expected defaultCfg to be set")
	}
}

func TestNewConfigListModel_NilConfig(t *testing.T) {
	m := NewConfigListModel(nil)

	// Should create default config when nil is passed
	if m.cfg == nil {
		t.Error("expected cfg to be set when nil passed")
	}

	if len(m.items) == 0 {
		t.Error("expected config items with default config")
	}
}

func TestConfigListModel_Filter(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	m.filter = "DefaultStrategy"
	m.applyFilter()

	// Should filter to items containing "DefaultStrategy"
	if len(m.filtered) != 1 {
		t.Errorf("expected 1 item matching 'DefaultStrategy', got %d", len(m.filtered))
	}

	if len(m.filtered) > 0 && m.filtered[0].Key != "DefaultStrategy" {
		t.Errorf("expected filtered item to be DefaultStrategy, got %s", m.filtered[0].Key)
	}
}

func TestConfigListModel_FilterByKey(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	m.filter = "threshold"
	m.applyFilter()

	// Should have both similarity thresholds
	if len(m.filtered) != 2 {
		t.Errorf("expected 2 items with 'threshold' filter, got %d", len(m.filtered))
	}

	keys := map[string]bool{}
	for _, item := range m.filtered {
		keys[item.Key] = true
	}
	if !keys["NameThreshold"] || !keys["ContentThreshold"] {
		t.Errorf("expected NameThreshold and ContentThreshold, got %v", keys)
	}
}

func TestConfigListModel_ClearFilter(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)
	originalCount := len(m.items)

	m.filter = "sync"
	m.applyFilter()

	// Clear filter
	m.filter = ""
	m.applyFilter()

	if len(m.filtered) != originalCount {
		t.Errorf("expected %d items after clearing filter, got %d", originalCount, len(m.filtered))
	}
}

func TestConfigListModel_CycleOptions(t *testing.T) {
	cfg := config.Default()
	cfg.Sync.DefaultStrategy = "overwrite"
	m := NewConfigListModel(cfg)

	// Find DefaultStrategy item
	for i, item := range m.filtered {
		if item.Section == "Sync" && item.Key == "DefaultStrategy" {
			m.table.SetCursor(i)
			break
		}
	}

	m.toggleOrCycleCurrentValue()

	// Should cycle to next option (skip)
	if m.cfg.Sync.DefaultStrategy != "skip" {
		t.Errorf("expected strategy to be 'skip' after cycle, got %s", m.cfg.Sync.DefaultStrategy)
	}

	if !m.modified {
		t.Error("expected modified to be true after cycle")
	}
}

func TestConfigListModel_UpdateFloatValue(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	m.updateConfigValue("Similarity", "NameThreshold", "0.85")

	if m.cfg.Similarity.NameThreshold != 0.85 {
		t.Errorf("expected NameThreshold to be 0.85, got %f", m.cfg.Similarity.NameThreshold)
	}
}

func TestConfigListModel_UpdateFloatValue_OutOfRange(t *testing.T) {
	cfg := config.Default()
	original := cfg.Similarity.NameThreshold
	m := NewConfigListModel(cfg)

	// Try to set value > 1.0
	m.updateConfigValue("Similarity", "NameThreshold", "1.5")

	// Should not change
	if m.cfg.Similarity.NameThreshold != original {
		t.Errorf("expected NameThreshold to remain %f, got %f", original, m.cfg.Similarity.NameThreshold)
	}
}

func TestConfigListModel_Init(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init to return nil")
	}
}

func TestConfigListModel_QuitKey(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	// Quit without modifications should exit immediately
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = newModel.(ConfigListModel)

	if !m.quitting {
		t.Error("expected quitting to be true")
	}

	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestConfigListModel_QuitWithModifications(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)
	m.modified = true

	// Quit with modifications should show confirm
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = newModel.(ConfigListModel)

	if !m.confirmMode {
		t.Error("expected confirmMode to be true when quitting with modifications")
	}

	if m.quitting {
		t.Error("expected quitting to be false during confirm")
	}
}

func TestConfigListModel_ConfirmSave(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)
	m.modified = true
	m.confirmMode = true

	// Confirm save
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = newModel.(ConfigListModel)

	if !m.quitting {
		t.Error("expected quitting to be true after confirm")
	}

	if m.result.Action != ConfigActionSave {
		t.Error("expected action to be Save after confirm")
	}

	if cmd == nil {
		t.Error("expected quit command after confirm")
	}
}

func TestConfigListModel_CancelConfirm(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)
	m.modified = true
	m.confirmMode = true

	// Cancel with 'n'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = newModel.(ConfigListModel)

	if m.confirmMode {
		t.Error("expected confirmMode to be false after cancel")
	}

	if m.quitting {
		t.Error("expected quitting to be false after cancel")
	}
}

func TestConfigListModel_CancelConfirmWithEsc(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)
	m.modified = true
	m.confirmMode = true

	// Cancel with Esc
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = newModel.(ConfigListModel)

	if m.confirmMode {
		t.Error("expected confirmMode to be false after esc")
	}
}

func TestConfigListModel_HelpToggle(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	if m.showHelp {
		t.Error("expected showHelp to be false initially")
	}

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newModel.(ConfigListModel)

	if !m.showHelp {
		t.Error("expected showHelp to be true after ?")
	}

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newModel.(ConfigListModel)

	if m.showHelp {
		t.Error("expected showHelp to be false after second ?")
	}
}

func TestConfigListModel_FilterMode(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	// Enter filter mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = newModel.(ConfigListModel)

	if !m.filtering {
		t.Error("expected filtering to be true after /")
	}

	// Type filter
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = newModel.(ConfigListModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = newModel.(ConfigListModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = newModel.(ConfigListModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = newModel.(ConfigListModel)

	if m.filter != "sync" {
		t.Errorf("expected filter 'sync', got '%s'", m.filter)
	}

	// Exit filter mode
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(ConfigListModel)

	if m.filtering {
		t.Error("expected filtering to be false after enter")
	}

	if m.filter != "sync" {
		t.Errorf("expected filter to remain 'sync', got '%s'", m.filter)
	}
}

func TestConfigListModel_BackspaceInFilter(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)
	m.filtering = true
	m.filter = "sync"

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = newModel.(ConfigListModel)

	if m.filter != "syn" {
		t.Errorf("expected filter 'syn' after backspace, got '%s'", m.filter)
	}
}

func TestConfigListModel_EditMode(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	// Navigate to a numeric field (Similarity NameThreshold)
	for i, item := range m.filtered {
		if item.Section == "Similarity" && item.Key == "NameThreshold" {
			m.table.SetCursor(i)
			break
		}
	}

	// Enter edit mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = newModel.(ConfigListModel)

	if !m.editing {
		t.Error("expected editing to be true after 'e'")
	}
}

func TestConfigListModel_EditModeNotForOptionsOutput(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	// Navigate to an options field (Output Color)
	for i, item := range m.filtered {
		if item.Section == "Output" && item.Key == "Color" {
			m.table.SetCursor(i)
			break
		}
	}

	// Try to enter edit mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = newModel.(ConfigListModel)

	// Should not enter edit mode for options fields
	if m.editing {
		t.Error("expected editing to be false for options fields")
	}
}

func TestConfigListModel_EditModeNotForOptions(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	// Navigate to an options field (Sync DefaultStrategy)
	for i, item := range m.filtered {
		if item.Section == "Sync" && item.Key == "DefaultStrategy" {
			m.table.SetCursor(i)
			break
		}
	}

	// Try to enter edit mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = newModel.(ConfigListModel)

	// Should not enter edit mode for options fields (use toggle instead)
	if m.editing {
		t.Error("expected editing to be false for options fields")
	}
}

func TestConfigListModel_SaveKey(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = newModel.(ConfigListModel)

	if !m.quitting {
		t.Error("expected quitting to be true after save")
	}

	if m.result.Action != ConfigActionSave {
		t.Error("expected action to be Save")
	}

	if m.result.Config == nil {
		t.Error("expected Config to be set in result")
	}

	if cmd == nil {
		t.Error("expected quit command after save")
	}
}

func TestConfigListModel_WindowResize(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(ConfigListModel)

	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}

	if m.height != 40 {
		t.Errorf("expected height 40, got %d", m.height)
	}
}

func TestConfigListModel_View(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	view := m.View()

	// Should contain title
	if view == "" {
		t.Error("expected non-empty view")
	}

	// Should contain Configuration
	if !containsString(view, "Configuration") {
		t.Error("expected view to contain 'Configuration'")
	}
}

func TestConfigListModel_ViewWithModified(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)
	m.modified = true

	view := m.View()

	if !containsString(view, "[modified]") {
		t.Error("expected view to contain '[modified]' when modified")
	}
}

func TestConfigListModel_ViewShortHelp(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)

	view := m.View()

	if !containsString(view, "navigate") {
		t.Error("expected short help to contain 'navigate'")
	}
}

func TestConfigListModel_ViewFullHelp(t *testing.T) {
	cfg := config.Default()
	m := NewConfigListModel(cfg)
	m.showHelp = true

	view := m.View()

	if !containsString(view, "Navigation:") {
		t.Error("expected full help to contain 'Navigation:'")
	}
}

func TestConfigListResult_DefaultAction(t *testing.T) {
	result := ConfigListResult{}
	if result.Action != ConfigActionNone {
		t.Errorf("expected default action to be None, got %v", result.Action)
	}
}

func TestRunConfigList_NilConfig(_ *testing.T) {
	// This is more of an integration test, but we can verify it doesn't panic
	// We can't actually run the full TUI in tests without special handling
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
		hasError bool
	}{
		{"0.5", 0.5, false},
		{"1.0", 1.0, false},
		{"0.75", 0.75, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tc := range tests {
		result, err := parseFloat(tc.input)
		if tc.hasError {
			if err == nil {
				t.Errorf("parseFloat(%q) expected error, got none", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseFloat(%q) unexpected error: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("parseFloat(%q) = %f, expected %f", tc.input, result, tc.expected)
			}
		}
	}
}

// Note: containsString helper is defined in dashboard_test.go
