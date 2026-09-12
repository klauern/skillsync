// Package util provides utility functions for paths and directories.
//
//nolint:revive // var-naming - package name is meaningful
package util

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/klauern/skillsync/internal/harness"
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

// CopilotSkillsPath returns the canonical GitHub Copilot user skills root.
func CopilotSkillsPath() string {
	return filepath.Join(HomeDir(), ".copilot", "skills")
}

// PiSkillsPath returns the canonical Pi user skills root.
func PiSkillsPath() string {
	return filepath.Join(HomeDir(), ".pi", "agent", "skills")
}

// PiPromptsPath returns the canonical Pi user prompts root.
func PiPromptsPath() string {
	return filepath.Join(HomeDir(), ".pi", "agent", "prompts")
}

// PiDevSkillsPath is a deprecated source-compatible alias for PiSkillsPath.
func PiDevSkillsPath() string {
	return PiSkillsPath()
}

// PiDevPromptsPath is a deprecated source-compatible alias for PiPromptsPath.
func PiDevPromptsPath() string {
	return PiPromptsPath()
}

// PiRepoSkillsPath returns the canonical Pi repository skills root.
func PiRepoSkillsPath(projectDir string) string {
	return filepath.Join(projectDir, ".pi", "skills")
}

// PiDevRepoSkillsPath is a deprecated source-compatible alias for PiRepoSkillsPath.
func PiDevRepoSkillsPath(projectDir string) string {
	return PiRepoSkillsPath(projectDir)
}

// PiRepoPromptsPath returns the canonical Pi repository prompts root.
func PiRepoPromptsPath(projectDir string) string {
	return filepath.Join(projectDir, ".pi", "prompts")
}

// PiDevRepoPromptsPath is a deprecated source-compatible alias for PiRepoPromptsPath.
func PiDevRepoPromptsPath(projectDir string) string {
	return PiRepoPromptsPath(projectDir)
}

// CursorSkillsPath returns the default Cursor skills directory (global)
// This is the new Agent Skills Standard location (~/.cursor/skills)
func CursorSkillsPath() string {
	return filepath.Join(HomeDir(), ".cursor", "skills")
}

// CursorCommandsPath returns the global Cursor commands directory (~/.cursor/commands)
func CursorCommandsPath() string {
	return filepath.Join(HomeDir(), ".cursor", "commands")
}

// CursorProjectCommandsPath returns the Cursor commands directory for a project
func CursorProjectCommandsPath(projectDir string) string {
	return filepath.Join(projectDir, ".cursor", "commands")
}

// CodexConfigPath returns the canonical Codex skills directory for a project.
func CodexConfigPath(projectDir string) string {
	return filepath.Join(projectDir, ".agents", "skills")
}

// CodexSkillsPath returns the canonical Codex user skills directory.
func CodexSkillsPath() string {
	return filepath.Join(HomeDir(), ".agents", "skills")
}

// GeminiPath returns the default Gemini CLI config directory (user-level).
func GeminiPath() string {
	return filepath.Join(HomeDir(), ".gemini")
}

// GeminiRepoPath returns the Gemini CLI config directory for a project.
func GeminiRepoPath(projectDir string) string {
	return filepath.Join(projectDir, ".gemini")
}

// GeminiSkillsPath returns the canonical Gemini CLI user skills root.
func GeminiSkillsPath() string {
	return filepath.Join(GeminiPath(), "skills")
}

// GeminiRepoSkillsPath returns the canonical Gemini CLI repository skills root.
func GeminiRepoSkillsPath(projectDir string) string {
	return filepath.Join(GeminiRepoPath(projectDir), "skills")
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
	definition, ok := harness.Lookup(cfg.Platform)
	if !ok {
		return paths
	}

	repoRoots := repoSearchRoots(cfg)
	for _, root := range definition.DiscoveryRoots {
		switch {
		case strings.HasPrefix(root, "~/"):
			appendUniquePath(paths, model.ScopeUser, filepath.Join(HomeDir(), root[2:]))
		case filepath.IsAbs(root):
			appendUniquePath(paths, model.ScopeSystem, filepath.Clean(root))
		default:
			for _, repoRoot := range repoRoots {
				appendUniquePath(paths, model.ScopeRepo, filepath.Join(repoRoot, root))
			}
		}
	}

	if cfg.AdminPath != "" {
		paths[model.ScopeAdmin] = []string{cfg.AdminPath}
	}
	if cfg.SystemPath != "" {
		paths[model.ScopeSystem] = prependUniquePath(paths[model.ScopeSystem], cfg.SystemPath)
	}

	return paths
}

func repoSearchRoots(cfg TieredPathConfig) []string {
	if cfg.WorkingDir == "" {
		return nil
	}
	roots := []string{cfg.WorkingDir}
	repoRoot := cfg.RepoRoot
	if repoRoot == "" {
		repoRoot = GetRepoRoot(cfg.WorkingDir)
	}
	if repoRoot != "" && filepath.Clean(repoRoot) != filepath.Clean(cfg.WorkingDir) {
		roots = append(roots, repoRoot)
	}
	return roots
}

func appendUniquePath(paths map[model.SkillScope][]string, scope model.SkillScope, path string) {
	for _, existing := range paths[scope] {
		if existing == path {
			return
		}
	}
	paths[scope] = append(paths[scope], path)
}

func prependUniquePath(paths []string, path string) []string {
	result := []string{path}
	for _, existing := range paths {
		if existing != path {
			result = append(result, existing)
		}
	}
	return result
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

// platformDirName returns the platform-specific hidden directory name.
func platformDirName(p model.Platform) string {
	if info, ok := model.PlatformInfoFor(p); ok {
		return info.DotDir
	}
	return "." + strings.ToLower(string(p))
}

// platformSkillsPathFns maps platforms with non-standard user-level skills paths
// to their resolver functions.
var platformSkillsPathFns = map[model.Platform]func() string{
	model.Copilot: CopilotSkillsPath,
	model.Gemini:  GeminiSkillsPath,
	model.Codex:   CodexSkillsPath,
	model.Pi:      PiSkillsPath,
}

// PlatformSkillsPath returns the user-level skills path for a platform.
func PlatformSkillsPath(p model.Platform) string {
	if fn, ok := platformSkillsPathFns[p]; ok {
		return fn()
	}
	return filepath.Join(HomeDir(), platformDirName(p), "skills")
}

// repoSkillsPathFns maps platforms with non-standard repo-level skills paths
// to their resolver functions.
var repoSkillsPathFns = map[model.Platform]func(string) string{
	model.Copilot: func(root string) string { return filepath.Join(root, ".github", "skills") },
	model.Gemini:  GeminiRepoSkillsPath,
	model.Codex:   func(root string) string { return filepath.Join(root, ".agents", "skills") },
	model.Pi:      PiRepoSkillsPath,
}

// RepoSkillsPath returns the repo-level skills path for a platform.
func RepoSkillsPath(p model.Platform, repoRoot string) string {
	if fn, ok := repoSkillsPathFns[p]; ok {
		return fn(repoRoot)
	}
	return filepath.Join(repoRoot, platformDirName(p), "skills")
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
