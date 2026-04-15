package cli

import (
	"fmt"
	"os"
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

	code := m.Run()

	if hadHome {
		_ = os.Setenv("HOME", oldHome)
	} else {
		_ = os.Unsetenv("HOME")
	}
	_ = os.RemoveAll(tempHome)

	os.Exit(code)
}
