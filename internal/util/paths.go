// Package util provides utility functions for paths and directories.
//
//nolint:revive // var-naming - package name is meaningful
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

// CursorSkillsPath returns the default Cursor rules directory (global)
func CursorSkillsPath() string {
	return filepath.Join(HomeDir(), ".cursor", "rules")
}

// CodexConfigPath returns the Codex config directory for a project
func CodexConfigPath(projectDir string) string {
	return filepath.Join(projectDir, ".codex")
}

// SkillsyncConfigPath returns the skillsync configuration directory
// Supports SKILLSYNC_HOME environment variable override
func SkillsyncConfigPath() string {
	if configHome := os.Getenv("SKILLSYNC_HOME"); configHome != "" {
		return configHome
	}
	return filepath.Join(HomeDir(), ".skillsync")
}

// SkillsyncBackupsPath returns the skillsync backups directory
func SkillsyncBackupsPath() string {
	return filepath.Join(SkillsyncConfigPath(), "backups")
}

// SkillsyncMetadataPath returns the skillsync metadata directory
func SkillsyncMetadataPath() string {
	return filepath.Join(SkillsyncConfigPath(), "metadata")
}
