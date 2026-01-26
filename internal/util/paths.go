// Package util provides utility functions for paths and directories.
//
//nolint:revive // var-naming - package name is meaningful
package util

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/klauern/skillsync/internal/model"
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

// SkillsyncPluginsPath returns the skillsync plugins directory
func SkillsyncPluginsPath() string {
	return filepath.Join(SkillsyncConfigPath(), "plugins")
}

// GetRepoRoot attempts to find the root of the current git repository.
// Returns empty string if not in a git repository.
func GetRepoRoot(startDir string) string {
	dir := startDir
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && (info.IsDir() || info.Mode().IsRegular()) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "" // Reached filesystem root
		}
		dir = parent
	}
}

// TieredPathConfig holds configuration for tiered path resolution.
type TieredPathConfig struct {
	// WorkingDir is the current working directory (for repo scope)
	WorkingDir string
	// RepoRoot is the root of the repository (optional, will be detected if empty)
	RepoRoot string
	// Platform is the target platform
	Platform model.Platform
	// AdminPath is an optional admin-level path (e.g., /opt/{platform}/skills)
	AdminPath string
	// SystemPath is an optional system-level path (e.g., /etc/{platform}/skills)
	SystemPath string
}

// GetTieredPaths returns paths for each scope level in precedence order (highest first).
// This enables cascading skill discovery where repo skills override user skills, etc.
func GetTieredPaths(cfg TieredPathConfig) map[model.SkillScope][]string {
	paths := make(map[model.SkillScope][]string)

	platformDir := platformDirName(cfg.Platform)

	// Repo scope: $CWD/.{platform}/skills and $REPO_ROOT/.{platform}/skills
	if cfg.WorkingDir != "" {
		cwdPath := filepath.Join(cfg.WorkingDir, platformDir, "skills")
		paths[model.ScopeRepo] = append(paths[model.ScopeRepo], cwdPath)

		// Also check repo root if different from working dir
		repoRoot := cfg.RepoRoot
		if repoRoot == "" {
			repoRoot = GetRepoRoot(cfg.WorkingDir)
		}
		if repoRoot != "" && repoRoot != cfg.WorkingDir {
			repoPath := filepath.Join(repoRoot, platformDir, "skills")
			paths[model.ScopeRepo] = append(paths[model.ScopeRepo], repoPath)
		}
	}

	// User scope: ~/.{platform}/skills
	userPath := filepath.Join(HomeDir(), platformDir, "skills")
	paths[model.ScopeUser] = []string{userPath}

	// Admin scope: optional, typically /opt/{platform}/skills
	if cfg.AdminPath != "" {
		paths[model.ScopeAdmin] = []string{cfg.AdminPath}
	}

	// System scope: optional, typically /etc/{platform}/skills
	if cfg.SystemPath != "" {
		paths[model.ScopeSystem] = []string{cfg.SystemPath}
	}

	return paths
}

// GetAllSearchPaths returns all search paths in precedence order (highest first).
// This is useful for discovering all available skills across all scopes.
func GetAllSearchPaths(cfg TieredPathConfig) []ScopedPath {
	paths := GetTieredPaths(cfg)
	var result []ScopedPath

	// Return in precedence order: repo, user, admin, system, builtin
	scopes := []model.SkillScope{model.ScopeRepo, model.ScopeUser, model.ScopeAdmin, model.ScopeSystem, model.ScopeBuiltin}
	for _, scope := range scopes {
		for _, p := range paths[scope] {
			result = append(result, ScopedPath{Path: p, Scope: scope})
		}
	}

	return result
}

// ScopedPath represents a path with its associated scope.
type ScopedPath struct {
	Path  string
	Scope model.SkillScope
}

// FilterExistingPaths filters ScopedPaths to only include paths that exist on the filesystem.
func FilterExistingPaths(paths []ScopedPath) []ScopedPath {
	var result []ScopedPath
	for _, sp := range paths {
		if _, err := os.Stat(sp.Path); err == nil {
			result = append(result, sp)
		}
	}
	return result
}

// platformDirName returns the platform-specific directory name.
func platformDirName(p model.Platform) string {
	switch p {
	case model.ClaudeCode:
		return ".claude"
	case model.Cursor:
		return ".cursor"
	case model.Codex:
		return ".codex"
	default:
		return "." + strings.ToLower(string(p))
	}
}

// PlatformSkillsPath returns the user-level skills path for a platform.
func PlatformSkillsPath(p model.Platform) string {
	return filepath.Join(HomeDir(), platformDirName(p), "skills")
}

// RepoSkillsPath returns the repo-level skills path for a platform.
func RepoSkillsPath(p model.Platform, repoRoot string) string {
	return filepath.Join(repoRoot, platformDirName(p), "skills")
}
