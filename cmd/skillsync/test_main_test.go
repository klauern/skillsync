package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	tempHome, err := os.MkdirTemp("", "skillsync-cmd-test-")
	if err != nil {
		panic(err)
	}

	setEnvOrPanic := func(key, value string) {
		if err := os.Setenv(key, value); err != nil {
			panic(err)
		}
	}
	mkdirAllOrPanic := func(path string) {
		if err := os.MkdirAll(path, 0o750); err != nil {
			panic(err)
		}
	}

	setEnvOrPanic("HOME", tempHome)

	claudePath := filepath.Join(tempHome, ".claude", "skills")
	cursorPath := filepath.Join(tempHome, ".cursor", "skills")
	codexPath := filepath.Join(tempHome, ".codex", "skills")

	mkdirAllOrPanic(claudePath)
	mkdirAllOrPanic(cursorPath)
	mkdirAllOrPanic(codexPath)

	setEnvOrPanic("SKILLSYNC_CLAUDE_CODE_PATH", claudePath)
	setEnvOrPanic("SKILLSYNC_CURSOR_PATH", cursorPath)
	setEnvOrPanic("SKILLSYNC_CODEX_PATH", codexPath)

	setEnvOrPanic("SKILLSYNC_CLAUDE_CODE_SKILLS_PATHS", claudePath)
	setEnvOrPanic("SKILLSYNC_CURSOR_SKILLS_PATHS", cursorPath)
	setEnvOrPanic("SKILLSYNC_CODEX_SKILLS_PATHS", codexPath)

	code := m.Run()
	if err := os.RemoveAll(tempHome); err != nil {
		fmt.Fprintf(os.Stderr, "failed to clean up temp home: %v\n", err)
	}
	os.Exit(code)
}
