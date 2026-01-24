package cli

import (
	"testing"
)

func TestVersionVariables(t *testing.T) {
	// Version should be set (even if to "dev")
	if Version == "" {
		t.Error("Version should not be empty")
	}

	// Commit and BuildDate should have defaults
	if Commit == "" {
		t.Error("Commit should not be empty")
	}
	if BuildDate == "" {
		t.Error("BuildDate should not be empty")
	}
}
