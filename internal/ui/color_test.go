package ui

import (
	"testing"
)

func TestStatusFunctions(t *testing.T) {
	// Disable colors for consistent test output
	DisableColors()
	defer EnableColors()

	tests := []struct {
		name     string
		fn       func(string) string
		input    string
		contains string
	}{
		{"StatusSuccess empty", StatusSuccess, "", SymbolSuccess},
		{"StatusSuccess with msg", StatusSuccess, "done", SymbolSuccess + " done"},
		{"StatusError empty", StatusError, "", SymbolError},
		{"StatusError with msg", StatusError, "failed", SymbolError + " failed"},
		{"StatusWarning empty", StatusWarning, "", SymbolWarning},
		{"StatusWarning with msg", StatusWarning, "caution", SymbolWarning + " caution"},
		{"StatusSkipped empty", StatusSkipped, "", SymbolSkipped},
		{"StatusSkipped with msg", StatusSkipped, "skip", SymbolSkipped + " skip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.input)
			if got != tt.contains {
				t.Errorf("got %q, want %q", got, tt.contains)
			}
		})
	}
}

func TestColorToggle(t *testing.T) {
	// Save initial state
	initial := IsColorEnabled()

	DisableColors()
	if IsColorEnabled() {
		t.Error("expected colors to be disabled")
	}

	EnableColors()
	if !IsColorEnabled() {
		t.Error("expected colors to be enabled")
	}

	// Restore initial state
	if !initial {
		DisableColors()
	}
}

func TestColorFunctions(t *testing.T) {
	// Disable colors for consistent test output
	DisableColors()
	defer EnableColors()

	// When colors are disabled, these should return the plain text
	if got := Success("test"); got != "test" {
		t.Errorf("Success() = %q, want %q", got, "test")
	}
	if got := Error("test"); got != "test" {
		t.Errorf("Error() = %q, want %q", got, "test")
	}
	if got := Warning("test"); got != "test" {
		t.Errorf("Warning() = %q, want %q", got, "test")
	}
	if got := Info("test"); got != "test" {
		t.Errorf("Info() = %q, want %q", got, "test")
	}
	if got := Bold("test"); got != "test" {
		t.Errorf("Bold() = %q, want %q", got, "test")
	}
	if got := Dim("test"); got != "test" {
		t.Errorf("Dim() = %q, want %q", got, "test")
	}
	if got := Header("test"); got != "test" {
		t.Errorf("Header() = %q, want %q", got, "test")
	}
}
