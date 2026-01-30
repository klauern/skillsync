package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	tempHome, err := os.MkdirTemp("", "skillsync-home-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp HOME: %v\n", err)
		os.Exit(1)
	}

	oldHome, hadHome := os.LookupEnv("HOME")
	if err := os.Setenv("HOME", tempHome); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set HOME: %v\n", err)
		_ = os.RemoveAll(tempHome)
		os.Exit(1)
	}

	claudePath := filepath.Join(tempHome, ".claude", "skills")
	cursorPath := filepath.Join(tempHome, ".cursor", "skills")
	codexPath := filepath.Join(tempHome, ".codex", "skills")

	_ = os.MkdirAll(claudePath, 0o750)
	_ = os.MkdirAll(cursorPath, 0o750)
	_ = os.MkdirAll(codexPath, 0o750)

	code := m.Run()

	if hadHome {
		_ = os.Setenv("HOME", oldHome)
	} else {
		_ = os.Unsetenv("HOME")
	}
	_ = os.RemoveAll(tempHome)

	os.Exit(code)
}
