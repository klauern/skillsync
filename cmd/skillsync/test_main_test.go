package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	tempHome, err := os.MkdirTemp("", "skillsync-cmd-test-")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.RemoveAll(tempHome)
	}()

	setEnvOrPanic := func(key, value string) {
		if err := os.Setenv(key, value); err != nil {
			panic(err)
		}
	}

	setEnvOrPanic("HOME", tempHome)

	claudePath := filepath.Join(tempHome, ".claude", "skills")
	cursorPath := filepath.Join(tempHome, ".cursor", "skills")
	codexPath := filepath.Join(tempHome, ".codex", "skills")

	_ = os.MkdirAll(claudePath, 0o750)
	_ = os.MkdirAll(cursorPath, 0o750)
	_ = os.MkdirAll(codexPath, 0o750)

	setEnvOrPanic("SKILLSYNC_CLAUDE_CODE_PATH", claudePath)
	setEnvOrPanic("SKILLSYNC_CURSOR_PATH", cursorPath)
	setEnvOrPanic("SKILLSYNC_CODEX_PATH", codexPath)

	setEnvOrPanic("SKILLSYNC_CLAUDE_CODE_SKILLS_PATHS", claudePath)
	setEnvOrPanic("SKILLSYNC_CURSOR_SKILLS_PATHS", cursorPath)
	setEnvOrPanic("SKILLSYNC_CODEX_SKILLS_PATHS", codexPath)

	os.Exit(m.Run())
}
