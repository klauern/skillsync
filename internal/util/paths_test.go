//nolint:revive // var-naming - package name is consistent with main package
package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestHomeDir(t *testing.T) {
	home := HomeDir()
	if home == "" {
		t.Error("HomeDir() returned empty string")
	}

	// Verify it's an absolute path
	if !filepath.IsAbs(home) {
		t.Errorf("HomeDir() returned relative path: %s", home)
	}
}

func TestClaudeCodeSkillsPath(t *testing.T) {
	path := ClaudeCodeSkillsPath()

	expected := filepath.Join(HomeDir(), ".claude", "skills")
	if path != expected {
		t.Errorf("ClaudeCodeSkillsPath() = %q, want %q", path, expected)
	}
}

func TestCursorRulesPath(t *testing.T) {
	projectDir := "/test/project"
	path := CursorRulesPath(projectDir)

	expected := "/test/project/.cursor/rules"
	if path != expected {
		t.Errorf("CursorRulesPath(%q) = %q, want %q", projectDir, path, expected)
	}
}

func TestCodexConfigPath(t *testing.T) {
	projectDir := "/test/project"
	path := CodexConfigPath(projectDir)

	expected := "/test/project/.codex"
	if path != expected {
		t.Errorf("CodexConfigPath(%q) = %q, want %q", projectDir, path, expected)
	}
}

func TestGetRepoRoot(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string // Returns start dir
		cleanup  func(t *testing.T, dir string)
		wantRoot bool // Whether we expect to find a root
	}{
		{
			name: "finds git repo root",
			setup: func(t *testing.T) string {
				// Create temp dir with .git
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				if err := os.Mkdir(gitDir, 0o750); err != nil {
					t.Fatalf("failed to create .git dir: %v", err)
				}
				// Create a subdirectory to start from
				subDir := filepath.Join(tmpDir, "sub", "dir")
				if err := os.MkdirAll(subDir, 0o750); err != nil {
					t.Fatalf("failed to create sub dir: %v", err)
				}
				return subDir
			},
			wantRoot: true,
		},
		{
			name: "handles git worktree (file instead of dir)",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				gitFile := filepath.Join(tmpDir, ".git")
				// Git worktrees use a file pointing to the main repo
				if err := os.WriteFile(gitFile, []byte("gitdir: /some/path"), 0o600); err != nil {
					t.Fatalf("failed to create .git file: %v", err)
				}
				return tmpDir
			},
			wantRoot: true,
		},
		{
			name: "returns empty when no git repo",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantRoot: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startDir := tt.setup(t)

			got := GetRepoRoot(startDir)

			if tt.wantRoot && got == "" {
				t.Error("GetRepoRoot() returned empty, expected to find root")
			}
			if !tt.wantRoot && got != "" {
				t.Errorf("GetRepoRoot() = %q, expected empty", got)
			}
		})
	}
}

func TestGetTieredPaths(t *testing.T) {
	home := HomeDir()

	tests := []struct {
		name       string
		cfg        TieredPathConfig
		checkScope model.SkillScope
		wantPaths  bool // At minimum we expect some paths
	}{
		{
			name: "claude code with working dir",
			cfg: TieredPathConfig{
				WorkingDir: "/test/project",
				Platform:   model.ClaudeCode,
			},
			checkScope: model.ScopeUser,
			wantPaths:  true,
		},
		{
			name: "cursor with admin path",
			cfg: TieredPathConfig{
				WorkingDir: "/test/project",
				Platform:   model.Cursor,
				AdminPath:  "/opt/cursor/skills",
			},
			checkScope: model.ScopeAdmin,
			wantPaths:  true,
		},
		{
			name: "codex with system path",
			cfg: TieredPathConfig{
				WorkingDir: "/test/project",
				Platform:   model.Codex,
				SystemPath: "/etc/codex/skills",
			},
			checkScope: model.ScopeSystem,
			wantPaths:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := GetTieredPaths(tt.cfg)

			// Check that we got paths for the expected scope
			scopePaths, ok := paths[tt.checkScope]
			if tt.wantPaths && (!ok || len(scopePaths) == 0) {
				t.Errorf("GetTieredPaths() missing paths for scope %s", tt.checkScope)
			}

			// Verify user scope path format
			if userPaths, ok := paths[model.ScopeUser]; ok && len(userPaths) > 0 {
				expectedPrefix := filepath.Join(home, ".")
				if userPaths[0][:len(expectedPrefix)] != expectedPrefix {
					t.Errorf("User path %q doesn't start with expected prefix %q", userPaths[0], expectedPrefix)
				}
			}
		})
	}
}

func TestGetAllSearchPaths(t *testing.T) {
	cfg := TieredPathConfig{
		WorkingDir: "/test/project",
		Platform:   model.ClaudeCode,
		AdminPath:  "/opt/claude/skills",
	}

	paths := GetAllSearchPaths(cfg)

	// Verify paths are in precedence order (highest first: repo, user, admin, system, builtin)
	if len(paths) == 0 {
		t.Fatal("GetAllSearchPaths() returned empty slice")
	}

	// First path should be repo scope
	if paths[0].Scope != model.ScopeRepo {
		t.Errorf("First path has scope %s, expected %s", paths[0].Scope, model.ScopeRepo)
	}

	// Check that we have expected scopes in order
	seenScopes := make(map[model.SkillScope]int)
	for i, sp := range paths {
		if _, ok := seenScopes[sp.Scope]; !ok {
			seenScopes[sp.Scope] = i
		}
	}

	// Repo should come before User
	if repoIdx, ok := seenScopes[model.ScopeRepo]; ok {
		if userIdx, ok := seenScopes[model.ScopeUser]; ok {
			if repoIdx > userIdx {
				t.Error("Repo scope paths should come before User scope paths")
			}
		}
	}
}

func TestPlatformDirName(t *testing.T) {
	tests := []struct {
		platform model.Platform
		expected string
	}{
		{model.ClaudeCode, ".claude"},
		{model.Cursor, ".cursor"},
		{model.Codex, ".codex"},
	}

	for _, tt := range tests {
		t.Run(string(tt.platform), func(t *testing.T) {
			got := platformDirName(tt.platform)
			if got != tt.expected {
				t.Errorf("platformDirName(%s) = %q, want %q", tt.platform, got, tt.expected)
			}
		})
	}
}

func TestPlatformSkillsPath(t *testing.T) {
	home := HomeDir()

	tests := []struct {
		platform model.Platform
		expected string
	}{
		{model.ClaudeCode, filepath.Join(home, ".claude", "skills")},
		{model.Cursor, filepath.Join(home, ".cursor", "skills")},
		{model.Codex, filepath.Join(home, ".codex", "skills")},
	}

	for _, tt := range tests {
		t.Run(string(tt.platform), func(t *testing.T) {
			got := PlatformSkillsPath(tt.platform)
			if got != tt.expected {
				t.Errorf("PlatformSkillsPath(%s) = %q, want %q", tt.platform, got, tt.expected)
			}
		})
	}
}

func TestRepoSkillsPath(t *testing.T) {
	tests := []struct {
		platform model.Platform
		repoRoot string
		expected string
	}{
		{model.ClaudeCode, "/test/repo", "/test/repo/.claude/skills"},
		{model.Cursor, "/test/repo", "/test/repo/.cursor/skills"},
		{model.Codex, "/test/repo", "/test/repo/.codex/skills"},
	}

	for _, tt := range tests {
		t.Run(string(tt.platform), func(t *testing.T) {
			got := RepoSkillsPath(tt.platform, tt.repoRoot)
			if got != tt.expected {
				t.Errorf("RepoSkillsPath(%s, %s) = %q, want %q", tt.platform, tt.repoRoot, got, tt.expected)
			}
		})
	}
}

func TestFilterExistingPaths(t *testing.T) {
	// Create temp dirs for testing
	tmpDir := t.TempDir()
	existingPath := filepath.Join(tmpDir, "exists")
	if err := os.Mkdir(existingPath, 0o750); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	paths := []ScopedPath{
		{Path: existingPath, Scope: model.ScopeRepo},
		{Path: filepath.Join(tmpDir, "does-not-exist"), Scope: model.ScopeUser},
	}

	filtered := FilterExistingPaths(paths)

	if len(filtered) != 1 {
		t.Errorf("FilterExistingPaths() returned %d paths, expected 1", len(filtered))
	}

	if len(filtered) > 0 && filtered[0].Path != existingPath {
		t.Errorf("FilterExistingPaths() returned wrong path: %s", filtered[0].Path)
	}
}
