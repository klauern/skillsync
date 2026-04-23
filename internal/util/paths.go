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

// HomeDir returns the user's home directory.
// It panics with a descriptive message if the home directory cannot be
// determined (e.g. in containers or environments without HOME set), so that
// callers get a clear error instead of silently receiving an empty path.
func HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic("skillsync: cannot determine user home directory: " + err.Error())
	}
	return home
}

// ClaudeCodeSkillsPath returns the default Claude Code skills directory
func ClaudeCodeSkillsPath() string {
	return filepath.Join(HomeDir(), ".claude", "skills")
}

// CopilotSkillsPath returns the preferred GitHub Copilot workspace root.
func CopilotSkillsPath() string {
	return filepath.Join(HomeDir(), ".github")
}

// piDevRootCandidates returns the preferred Pi.dev config roots in priority order.
func piDevRootCandidates(home string) []string {
	return []string{
		filepath.Join(home, ".agents"),
		filepath.Join(home, ".pi", "agent"),
	}
}

// piDevProjectRootCandidates returns the preferred Pi.dev project roots in priority order.
func piDevProjectRootCandidates(projectDir string) []string {
	return []string{
		filepath.Join(projectDir, ".agents"),
		filepath.Join(projectDir, ".pi"),
	}
}

// firstExistingDir returns the first directory that exists from the provided candidates.
// If none exist, it returns the first candidate.
func firstExistingDir(candidates ...string) string {
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	if len(candidates) > 0 {
		return candidates[len(candidates)-1]
	}
	return ""
}

// PiDevSkillsPath returns the preferred Pi.dev user-level skills directory.
// When ~/.agents exists, it is preferred over ~/.pi/agent to avoid duplicate sync roots.
func PiDevSkillsPath() string {
	root := firstExistingDir(piDevRootCandidates(HomeDir())...)
	return filepath.Join(root, "skills")
}

// PiDevPromptsPath returns the preferred Pi.dev user-level prompts directory.
// It follows the same root selection as PiDevSkillsPath.
func PiDevPromptsPath() string {
	root := firstExistingDir(piDevRootCandidates(HomeDir())...)
	return filepath.Join(root, "prompts")
}

// PiDevRepoSkillsPath returns the preferred Pi.dev repo-level skills directory.
// When .agents exists, it is preferred over .pi to avoid duplicate sync roots.
func PiDevRepoSkillsPath(projectDir string) string {
	root := firstExistingDir(piDevProjectRootCandidates(projectDir)...)
	return filepath.Join(root, "skills")
}

// PiDevRepoPromptsPath returns the preferred Pi.dev repo-level prompts directory.
func PiDevRepoPromptsPath(projectDir string) string {
	root := firstExistingDir(piDevProjectRootCandidates(projectDir)...)
	return filepath.Join(root, "prompts")
}

// CursorSkillsPath returns the default Cursor skills directory (global)
// This is the new Agent Skills Standard location (~/.cursor/skills)
func CursorSkillsPath() string {
	return filepath.Join(HomeDir(), ".cursor", "skills")
}

// CursorProjectSkillsPath returns the Cursor skills directory for a project
func CursorProjectSkillsPath(projectDir string) string {
	return filepath.Join(projectDir, ".cursor", "skills")
}

// CursorCommandsPath returns the global Cursor commands directory (~/.cursor/commands)
func CursorCommandsPath() string {
	return filepath.Join(HomeDir(), ".cursor", "commands")
}

// CursorProjectCommandsPath returns the Cursor commands directory for a project
func CursorProjectCommandsPath(projectDir string) string {
	return filepath.Join(projectDir, ".cursor", "commands")
}

// CodexConfigPath returns the Codex skills directory for a project
func CodexConfigPath(projectDir string) string {
	return filepath.Join(projectDir, ".codex", "skills")
}

// codexUserRootCandidates returns the preferred Codex user roots in priority order.
func codexUserRootCandidates(home string) []string {
	return []string{
		filepath.Join(home, ".agents"),
		filepath.Join(home, ".codex"),
	}
}

// CodexSkillsPath returns the preferred Codex skills directory (user-level).
// When ~/.agents exists, it is preferred over ~/.codex to avoid duplicate sync roots.
func CodexSkillsPath() string {
	root := firstExistingDir(codexUserRootCandidates(HomeDir())...)
	return filepath.Join(root, "skills")
}

// GeminiPath returns the default Gemini CLI config directory (user-level).
func GeminiPath() string {
	return filepath.Join(HomeDir(), ".gemini")
}

// GeminiRepoPath returns the Gemini CLI config directory for a project.
func GeminiRepoPath(projectDir string) string {
	return filepath.Join(projectDir, ".gemini")
}

// PiDevProjectSkillsPath returns the Pi.dev skills directory for a project.
func PiDevProjectSkillsPath(projectDir string) string {
	return filepath.Join(projectDir, ".pi", "skills")
}

// PiAgentSkillsPath returns the default Pi Agent skills directory (user-level).
func PiAgentSkillsPath() string {
	return filepath.Join(HomeDir(), ".agents", "skills")
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

// ClaudePluginCachePath returns the Claude Code plugin cache directory
// This is where Claude Code stores installed plugins from marketplaces.
func ClaudePluginCachePath() string {
	return filepath.Join(HomeDir(), ".claude", "plugins", "cache")
}

// ClaudeInstalledPluginsPath returns the path to Claude Code's installed plugins manifest
func ClaudeInstalledPluginsPath() string {
	return filepath.Join(HomeDir(), ".claude", "plugins", "installed_plugins.json")
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

	switch cfg.Platform {
	case model.Copilot:
		if cfg.WorkingDir != "" {
			cwdPath := filepath.Join(cfg.WorkingDir, ".github")
			paths[model.ScopeRepo] = append(paths[model.ScopeRepo], cwdPath)

			repoRoot := cfg.RepoRoot
			if repoRoot == "" {
				repoRoot = GetRepoRoot(cfg.WorkingDir)
			}
			if repoRoot != "" && repoRoot != cfg.WorkingDir {
				paths[model.ScopeRepo] = append(paths[model.ScopeRepo], filepath.Join(repoRoot, ".github"))
			}
		}

		paths[model.ScopeUser] = []string{filepath.Join(HomeDir(), ".github")}

		if cfg.AdminPath != "" {
			paths[model.ScopeAdmin] = []string{cfg.AdminPath}
		}

		if cfg.SystemPath != "" {
			paths[model.ScopeSystem] = []string{cfg.SystemPath}
		}

		return paths
	case model.Gemini:
		if cfg.WorkingDir != "" {
			cwdPath := filepath.Join(cfg.WorkingDir, ".gemini")
			paths[model.ScopeRepo] = append(paths[model.ScopeRepo], cwdPath)

			repoRoot := cfg.RepoRoot
			if repoRoot == "" {
				repoRoot = GetRepoRoot(cfg.WorkingDir)
			}
			if repoRoot != "" && repoRoot != cfg.WorkingDir {
				paths[model.ScopeRepo] = append(paths[model.ScopeRepo], filepath.Join(repoRoot, ".gemini"))
			}
		}

		paths[model.ScopeUser] = []string{filepath.Join(HomeDir(), ".gemini")}

		if cfg.AdminPath != "" {
			paths[model.ScopeAdmin] = []string{cfg.AdminPath}
		}

		if cfg.SystemPath != "" {
			paths[model.ScopeSystem] = []string{cfg.SystemPath}
		}

		return paths
	}

	// Repo scope: $CWD/.{platform}/skills and $REPO_ROOT/.{platform}/skills
	if cfg.WorkingDir != "" {
		var cwdPath string
		if cfg.Platform == model.PiDev {
			cwdPath = PiDevRepoSkillsPath(cfg.WorkingDir)
		} else {
			cwdPath = filepath.Join(cfg.WorkingDir, platformDir, "skills")
		}
		paths[model.ScopeRepo] = append(paths[model.ScopeRepo], cwdPath)

		// Also check repo root if different from working dir
		repoRoot := cfg.RepoRoot
		if repoRoot == "" {
			repoRoot = GetRepoRoot(cfg.WorkingDir)
		}
		if repoRoot != "" && repoRoot != cfg.WorkingDir {
			var repoPath string
			if cfg.Platform == model.PiDev {
				repoPath = PiDevRepoSkillsPath(repoRoot)
			} else {
				repoPath = filepath.Join(repoRoot, platformDir, "skills")
			}
			paths[model.ScopeRepo] = append(paths[model.ScopeRepo], repoPath)
		}
	}

	// User scope: ~/.{platform}/skills
	if cfg.Platform == model.Codex {
		for _, root := range codexUserRootCandidates(HomeDir()) {
			paths[model.ScopeUser] = append(paths[model.ScopeUser], filepath.Join(root, "skills"))
		}
	} else {
		var userPath string
		if cfg.Platform == model.PiDev {
			userPath = PiDevSkillsPath()
		} else {
			userPath = filepath.Join(HomeDir(), platformDir, "skills")
		}
		paths[model.ScopeUser] = []string{userPath}
	}

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
	case model.PiAgent:
		return ".pi"
	case model.Gemini:
		return ".gemini"
	case model.PiDev:
		return ".pi/agent"
	case model.Copilot:
		return ".github"
	default:
		return "." + strings.ToLower(string(p))
	}
}

// PlatformSkillsPath returns the user-level skills path for a platform.
func PlatformSkillsPath(p model.Platform) string {
	switch p {
	case model.PiAgent:
		return PiAgentSkillsPath()
	case model.Copilot:
		return CopilotSkillsPath()
	case model.Gemini:
		return GeminiPath()
	case model.Codex:
		return CodexSkillsPath()
	case model.PiDev:
		return PiDevSkillsPath()
	default:
		return filepath.Join(HomeDir(), platformDirName(p), "skills")
	}
}

// RepoSkillsPath returns the repo-level skills path for a platform.
func RepoSkillsPath(p model.Platform, repoRoot string) string {
	switch p {
	case model.PiAgent:
		return filepath.Join(repoRoot, ".agents", "skills")
	case model.Copilot:
		return filepath.Join(repoRoot, ".github")
	case model.Gemini:
		return GeminiRepoPath(repoRoot)
	case model.PiDev:
		return PiDevRepoSkillsPath(repoRoot)
	default:
		return filepath.Join(repoRoot, platformDirName(p), "skills")
	}
}

// ExpandPath expands a path by replacing ~ with the home directory
// and resolving relative paths from the given base directory.
// If baseDir is empty, relative paths are resolved from the current working directory.
func ExpandPath(path, baseDir string) string {
	if path == "" {
		return path
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(HomeDir(), path[2:])
	} else if path == "~" {
		return HomeDir()
	}

	// If already absolute, return as-is
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	// Resolve relative paths from baseDir (or working directory if baseDir is empty)
	if baseDir == "" {
		if wd, err := os.Getwd(); err == nil {
			baseDir = wd
		}
	}

	return filepath.Clean(filepath.Join(baseDir, path))
}

// ExpandPaths expands multiple paths using ExpandPath.
func ExpandPaths(paths []string, baseDir string) []string {
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		expanded := ExpandPath(p, baseDir)
		if expanded != "" {
			result = append(result, expanded)
		}
	}
	return result
}
