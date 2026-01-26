// Package ui provides terminal UI utilities for skillsync.
package ui

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// Color function types for styled output.
var (
	// Success is used for successful operations (green).
	Success = color.New(color.FgGreen).SprintFunc()
	// Error is used for errors and failures (red).
	Error = color.New(color.FgRed).SprintFunc()
	// Warning is used for warnings and cautions (yellow).
	Warning = color.New(color.FgYellow).SprintFunc()
	// Info is used for informational messages (cyan).
	Info = color.New(color.FgCyan).SprintFunc()
	// Bold is used for emphasis (bold white).
	Bold = color.New(color.Bold).SprintFunc()
	// Dim is used for secondary information (faint).
	Dim = color.New(color.Faint).SprintFunc()
	// Header is used for table headers (bold cyan).
	Header = color.New(color.FgCyan, color.Bold).SprintFunc()
)

// Status symbols with colors.
const (
	SymbolSuccess = "✓"
	SymbolError   = "✗"
	SymbolWarning = "⚠"
	SymbolSkipped = "-"
	SymbolPending = "○"
)

// StatusSuccess returns a green checkmark with optional message.
func StatusSuccess(msg string) string {
	if msg == "" {
		return Success(SymbolSuccess)
	}
	return Success(SymbolSuccess) + " " + msg
}

// StatusError returns a red X with optional message.
func StatusError(msg string) string {
	if msg == "" {
		return Error(SymbolError)
	}
	return Error(SymbolError) + " " + msg
}

// StatusWarning returns a yellow warning with optional message.
func StatusWarning(msg string) string {
	if msg == "" {
		return Warning(SymbolWarning)
	}
	return Warning(SymbolWarning) + " " + msg
}

// StatusSkipped returns a dimmed skip symbol with optional message.
func StatusSkipped(msg string) string {
	if msg == "" {
		return Dim(SymbolSkipped)
	}
	return Dim(SymbolSkipped) + " " + msg
}

// DisableColors disables all color output.
// This is useful for piping output or for users who prefer no colors.
func DisableColors() {
	color.NoColor = true
}

// EnableColors enables color output.
func EnableColors() {
	color.NoColor = false
}

// IsColorEnabled returns whether colors are currently enabled.
func IsColorEnabled() bool {
	return !color.NoColor
}

// ConfigureColors sets up color output based on the provided setting.
// Supports "auto", "always", "never", and respects NO_COLOR env var.
// Priority order: NO_COLOR env var > explicit setting (always/never) > auto-detect
func ConfigureColors(colorSetting string) {
	// NO_COLOR env var takes highest priority per https://no-color.org/
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		DisableColors()
		return
	}

	switch colorSetting {
	case "never":
		DisableColors()
	case "always":
		EnableColors()
	case "auto", "":
		// auto-detection is already handled by fatih/color library
		// which checks if stdout is a terminal
	}
}

// PrintSuccess prints a success message to stdout (green checkmark + message).
func PrintSuccess(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(StatusSuccess(msg))
}

// PrintError prints an error message to stderr (red X + message).
func PrintError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, StatusError(msg))
}

// PrintWarning prints a warning message to stderr (yellow warning + message).
func PrintWarning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, StatusWarning(msg))
}

// PrintInfo prints an info message to stdout (cyan).
func PrintInfo(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(Info(msg))
}
