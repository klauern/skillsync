package util

import (
	"os"
	"path/filepath"
)

// HomeDir returns the user's home directory
func HomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

// ClaudeCodeSkillsPath returns the default Claude Code skills directory
func ClaudeCodeSkillsPath() string {
	return filepath.Join(HomeDir(), ".claude", "skills")
}

// CursorRulesPath returns the Cursor rules directory for a project
func CursorRulesPath(projectDir string) string {
	return filepath.Join(projectDir, ".cursor", "rules")
}

// CodexConfigPath returns the Codex config directory for a project
func CodexConfigPath(projectDir string) string {
	return filepath.Join(projectDir, ".codex")
}
