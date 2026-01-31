package ui

import (
	"bytes"
	"io"
	"os"
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

func TestConfigureColors(t *testing.T) {
	// Save initial state
	initial := IsColorEnabled()
	defer func() {
		if initial {
			EnableColors()
		} else {
			DisableColors()
		}
	}()
	originalNoColor, hadNoColor := os.LookupEnv("NO_COLOR")
	t.Cleanup(func() {
		if hadNoColor {
			_ = os.Setenv("NO_COLOR", originalNoColor)
		} else {
			_ = os.Unsetenv("NO_COLOR")
		}
	})

	tests := []struct {
		name          string
		setting       string
		envNoColor    bool
		expectEnabled bool
	}{
		{"never disables colors", "never", false, false},
		{"always enables colors", "always", false, true},
		{"auto keeps default", "auto", false, true}, // Default with fatih/color
		{"empty keeps default", "", false, true},
		{"NO_COLOR env overrides always", "always", true, false},
		{"NO_COLOR env overrides auto", "auto", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to enabled state before each test
			EnableColors()

			if tt.envNoColor {
				if err := os.Setenv("NO_COLOR", "1"); err != nil {
					t.Fatalf("failed to set NO_COLOR: %v", err)
				}
			} else {
				if err := os.Unsetenv("NO_COLOR"); err != nil {
					t.Fatalf("failed to unset NO_COLOR: %v", err)
				}
			}

			ConfigureColors(tt.setting)

			if got := IsColorEnabled(); got != tt.expectEnabled {
				t.Errorf("IsColorEnabled() = %v, want %v", got, tt.expectEnabled)
			}
		})
	}
}

func TestPrintSuccess(t *testing.T) {
	DisableColors()
	defer EnableColors()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintSuccess("operation completed")

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if output != SymbolSuccess+" operation completed\n" {
		t.Errorf("PrintSuccess() output = %q, want %q", output, SymbolSuccess+" operation completed\n")
	}
}

func TestPrintError(t *testing.T) {
	DisableColors()
	defer EnableColors()

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	PrintError("operation failed")

	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if output != SymbolError+" operation failed\n" {
		t.Errorf("PrintError() output = %q, want %q", output, SymbolError+" operation failed\n")
	}
}

func TestPrintWarning(t *testing.T) {
	DisableColors()
	defer EnableColors()

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	PrintWarning("be careful")

	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if output != SymbolWarning+" be careful\n" {
		t.Errorf("PrintWarning() output = %q, want %q", output, SymbolWarning+" be careful\n")
	}
}

func TestPrintInfo(t *testing.T) {
	DisableColors()
	defer EnableColors()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintInfo("informational message")

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if output != "informational message\n" {
		t.Errorf("PrintInfo() output = %q, want %q", output, "informational message\n")
	}
}

func TestPrintFunctionsWithFormat(t *testing.T) {
	DisableColors()
	defer EnableColors()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintSuccess("processed %d items", 42)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	expected := SymbolSuccess + " processed 42 items\n"
	if output != expected {
		t.Errorf("PrintSuccess() with format = %q, want %q", output, expected)
	}
}
