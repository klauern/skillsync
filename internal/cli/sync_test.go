package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/util"
)

func TestSyncCommandArguments(t *testing.T) {
	tests := map[string]struct {
		args    []string
		wantErr bool
	}{
		"same source and target platform": {
			args:    []string{"skillsync", "sync", "cursor", "cursor"},
			wantErr: true,
		},
		"invalid strategy": {
			args:    []string{"skillsync", "sync", "--strategy", "invalid", "cursor", "codex"},
			wantErr: true,
		},
		"invalid source scope in spec": {
			args:    []string{"skillsync", "sync", "cursor:invalid", "codex"},
			wantErr: true,
		},
		"invalid target scope in spec": {
			args:    []string{"skillsync", "sync", "cursor", "codex:admin"},
			wantErr: true,
		},
		"valid source scope in spec": {
			args:    []string{"skillsync", "sync", "--trust", "external-reference,executable,native-config", "--skip-validation", "--yes", "cursor:user", "codex"},
			wantErr: false,
		},
		"valid target scope user in spec": {
			args:    []string{"skillsync", "sync", "--trust", "external-reference,executable,native-config", "--skip-validation", "--yes", "cursor", "codex:user"},
			wantErr: false,
		},
		"valid multiple source scopes in spec": {
			args:    []string{"skillsync", "sync", "--trust", "external-reference,executable,native-config", "--skip-validation", "--yes", "cursor:user,repo", "codex"},
			wantErr: false,
		},
		"invalid multiple target scopes": {
			args:    []string{"skillsync", "sync", "cursor", "codex:user,repo"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			ctx := context.Background()
			err := Run(ctx, tt.args)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Drain the pipe
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTargetPath(t *testing.T) {
	// Create a temp directory for testing
	tempDir := t.TempDir()

	tests := map[string]struct {
		setup   func() string
		wantErr bool
	}{
		"existing writable directory": {
			setup: func() string {
				dir := filepath.Join(tempDir, "existing")
				if err := os.MkdirAll(dir, 0o750); err != nil {
					t.Fatalf("failed to create test dir: %v", err)
				}
				return dir
			},
			wantErr: false,
		},
		"non-existing with writable parent": {
			setup: func() string {
				parent := filepath.Join(tempDir, "writable-parent")
				if err := os.MkdirAll(parent, 0o750); err != nil {
					t.Fatalf("failed to create test dir: %v", err)
				}
				return filepath.Join(parent, "new-dir")
			},
			wantErr: false,
		},
		"non-existing with missing parent but writable ancestor": {
			setup: func() string {
				ancestor := filepath.Join(tempDir, "ancestor")
				if err := os.MkdirAll(ancestor, 0o750); err != nil {
					t.Fatalf("failed to create test dir: %v", err)
				}
				return filepath.Join(ancestor, "missing-parent", "child")
			},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			path := tt.setup()

			// Set environment variable to override platform path
			t.Setenv("SKILLSYNC_CURSOR_PATH", path)

			err := validateTargetPath(model.Cursor)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateTargetPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInferScopeForPath(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Create a mock repo root
	repoRoot := filepath.Join(tempDir, "myrepo")
	if err := os.MkdirAll(repoRoot, 0o750); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}

	// Create a mock plugin cache directory
	pluginCachePath := filepath.Join(tempDir, ".claude", "plugins", "cache")
	if err := os.MkdirAll(pluginCachePath, 0o750); err != nil {
		t.Fatalf("failed to create plugin cache: %v", err)
	}

	tests := map[string]struct {
		path      string
		repoRoot  string
		wantScope model.SkillScope
	}{
		"repo path": {
			path:      filepath.Join(repoRoot, ".claude", "skills"),
			repoRoot:  repoRoot,
			wantScope: model.ScopeRepo,
		},
		"user path": {
			path:      filepath.Join(tempDir, ".claude", "skills"),
			repoRoot:  "",
			wantScope: model.ScopeUser,
		},
		"plugin cache path": {
			path:      pluginCachePath,
			repoRoot:  "",
			wantScope: model.ScopePlugin,
		},
		"plugin cache subdir": {
			path:      filepath.Join(pluginCachePath, "beads-marketplace", "beads", "0.49.0"),
			repoRoot:  "",
			wantScope: model.ScopePlugin,
		},
		"plugin cache takes precedence over user": {
			// Plugin cache is under home directory but should be detected as plugin scope
			path:      filepath.Join(pluginCachePath, "some-plugin"),
			repoRoot:  "",
			wantScope: model.ScopePlugin,
		},
		"system path": {
			path:      "/etc/skillsync/skills",
			repoRoot:  "",
			wantScope: model.ScopeSystem,
		},
		"admin path": {
			path:      "/opt/skillsync/skills",
			repoRoot:  "",
			wantScope: model.ScopeAdmin,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := inferScopeForPath(tt.path, tt.repoRoot)
			if got != tt.wantScope {
				t.Errorf("inferScopeForPath(%q, %q) = %q, want %q", tt.path, tt.repoRoot, got, tt.wantScope)
			}
		})
	}
}

func TestParsePlatformSkillsFromPathsWithPluginScope(t *testing.T) {
	// Set up isolated test environment
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Create user skills directory with a skill
	userSkillsDir := filepath.Join(tempDir, ".claude", "skills", "my-skill")
	if err := os.MkdirAll(userSkillsDir, 0o750); err != nil {
		t.Fatalf("failed to create user skills dir: %v", err)
	}
	userSkillContent := `---
name: my-skill
description: A user skill
---
# My Skill
This is a user skill.
`
	if err := os.WriteFile(filepath.Join(userSkillsDir, "SKILL.md"), []byte(userSkillContent), 0o600); err != nil {
		t.Fatalf("failed to write user skill: %v", err)
	}

	tests := map[string]struct {
		paths          []util.ScopedPath
		scopeFilter    []model.SkillScope
		platform       model.Platform
		includePlugins bool
		wantScopes     []model.SkillScope
	}{
		"user scope filter excludes plugins": {
			paths:          []util.ScopedPath{{Path: filepath.Join(tempDir, ".claude", "skills"), Scope: model.ScopeUser}},
			scopeFilter:    []model.SkillScope{model.ScopeUser},
			platform:       model.ClaudeCode,
			includePlugins: false,
			wantScopes:     []model.SkillScope{model.ScopeUser},
		},
		"plugin scope filter includes only plugins": {
			paths:       []util.ScopedPath{{Path: filepath.Join(tempDir, ".claude", "skills"), Scope: model.ScopeUser}},
			scopeFilter: []model.SkillScope{model.ScopePlugin},
			platform:    model.ClaudeCode,
			// Note: This will return no skills since there's no real plugin cache set up
			// The test validates that user skills are excluded when only plugin scope is requested
			includePlugins: false,
			wantScopes:     []model.SkillScope{},
		},
		"no filter excludes plugins by default": {
			paths:          []util.ScopedPath{{Path: filepath.Join(tempDir, ".claude", "skills"), Scope: model.ScopeUser}},
			scopeFilter:    nil,
			platform:       model.ClaudeCode,
			includePlugins: false,
			// With includePlugins=false, plugins are excluded even with no scope filter
			wantScopes: []model.SkillScope{model.ScopeUser},
		},
		"no filter with includePlugins includes user scope": {
			paths:          []util.ScopedPath{{Path: filepath.Join(tempDir, ".claude", "skills"), Scope: model.ScopeUser}},
			scopeFilter:    nil,
			platform:       model.ClaudeCode,
			includePlugins: true,
			// With includePlugins=true, plugins would be included (but no plugin cache in test)
			// User scope skills are always included
			wantScopes: []model.SkillScope{model.ScopeUser},
		},
		"non-claude platform ignores plugins": {
			paths:          []util.ScopedPath{{Path: filepath.Join(tempDir, ".cursor", "rules"), Scope: model.ScopeUser}},
			scopeFilter:    []model.SkillScope{model.ScopePlugin},
			platform:       model.Cursor,
			includePlugins: false,
			wantScopes:     []model.SkillScope{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			skills := parsePlatformSkillsFromPaths(tt.platform, tt.paths, "", tt.scopeFilter, tt.includePlugins)

			// Verify all returned skills have expected scopes
			for _, skill := range skills {
				found := slices.Contains(tt.wantScopes, skill.Scope)
				if !found && len(tt.wantScopes) > 0 {
					t.Errorf("skill %q has scope %q, want one of %v", skill.Name, skill.Scope, tt.wantScopes)
				}
			}

			// If we expect no scopes, verify no skills returned
			if len(tt.wantScopes) == 0 && len(skills) > 0 {
				// This is acceptable - we may still get skills from plugin cache
				// but they should have plugin scope
				for _, skill := range skills {
					if skill.Scope != model.ScopePlugin {
						t.Errorf("expected only plugin scope skills or empty, got scope %q for skill %q", skill.Scope, skill.Name)
					}
				}
			}
		})
	}
}

func TestPlatformSkillsPaths_PiDevIncludesDefaultSearchRoots(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repoRoot := filepath.Join(t.TempDir(), "repo")
	workingDir := filepath.Join(repoRoot, "nested")
	if err := os.MkdirAll(workingDir, 0o750); err != nil {
		t.Fatalf("failed to create working dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o750); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}

	for _, dir := range []string{
		filepath.Join(workingDir, ".pi", "skills"),
		filepath.Join(repoRoot, ".pi", "skills"),
		filepath.Join(workingDir, ".agents", "skills"),
		filepath.Join(repoRoot, ".agents", "skills"),
		filepath.Join(home, ".pi", "agent", "skills"),
		filepath.Join(home, ".agents", "skills"),
	} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	cfg := config.Default()
	paths, gotRepoRoot, err := platformSkillsPaths(cfg, model.PiDev)
	if err != nil {
		t.Fatalf("platformSkillsPaths() error = %v", err)
	}

	wantRepoRoot, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		wantRepoRoot = filepath.Clean(repoRoot)
	}
	gotRepoRoot, err = filepath.EvalSymlinks(gotRepoRoot)
	if err != nil {
		gotRepoRoot = filepath.Clean(gotRepoRoot)
	}

	if gotRepoRoot != wantRepoRoot {
		t.Fatalf("repoRoot = %q, want %q", gotRepoRoot, wantRepoRoot)
	}

	wantPaths := map[string]model.SkillScope{
		filepath.Clean(filepath.Join(workingDir, ".pi", "skills")):     model.ScopeRepo,
		filepath.Clean(filepath.Join(repoRoot, ".pi", "skills")):       model.ScopeRepo,
		filepath.Clean(filepath.Join(workingDir, ".agents", "skills")): model.ScopeRepo,
		filepath.Clean(filepath.Join(repoRoot, ".agents", "skills")):   model.ScopeRepo,
		filepath.Clean(filepath.Join(home, ".pi", "agent", "skills")):  model.ScopeUser,
		filepath.Clean(filepath.Join(home, ".agents", "skills")):       model.ScopeUser,
	}
	if len(paths) != len(wantPaths) {
		t.Fatalf("platformSkillsPaths() returned %d paths, want %d: %v", len(paths), len(wantPaths), paths)
	}
	for _, sp := range paths {
		gotPath := strings.TrimPrefix(filepath.Clean(sp.Path), "/private")
		wantScope, ok := wantPaths[gotPath]
		if !ok {
			t.Fatalf("unexpected path %q (scope %q)", gotPath, sp.Scope)
		}
		if sp.Scope != wantScope {
			t.Fatalf("path %q scope = %q, want %q", gotPath, sp.Scope, wantScope)
		}
		delete(wantPaths, gotPath)
	}
	if len(wantPaths) != 0 {
		t.Fatalf("missing expected paths: %v", wantPaths)
	}
}

func TestParsePlatformSkillsFromPaths_ClaudeSkillOverridesCommandSameName(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	commandsDir := filepath.Join(tempDir, ".claude", "commands")
	skillsDir := filepath.Join(tempDir, ".claude", "skills")
	if err := os.MkdirAll(commandsDir, 0o750); err != nil {
		t.Fatalf("failed to create commands dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(skillsDir, "review"), 0o750); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}

	commandContent := `---
description: command version
allowed-tools: Bash, Read
---
# /review
Command content.`
	if err := os.WriteFile(filepath.Join(commandsDir, "review.md"), []byte(commandContent), 0o600); err != nil {
		t.Fatalf("failed to write command file: %v", err)
	}

	skillContent := `---
name: review
description: skill version
---
# Review Skill
Skill content.`
	if err := os.WriteFile(filepath.Join(skillsDir, "review", "SKILL.md"), []byte(skillContent), 0o600); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}

	// Command path intentionally first to verify same-scope override behavior.
	skills := parsePlatformSkillsFromPaths(
		model.ClaudeCode,
		[]util.ScopedPath{
			{Path: commandsDir, Scope: model.ScopeUser},
			{Path: skillsDir, Scope: model.ScopeUser},
		},
		"",
		nil,
		false,
	)

	if len(skills) != 1 {
		t.Fatalf("expected 1 merged artifact, got %d", len(skills))
	}
	if skills[0].Name != "review" {
		t.Fatalf("expected review artifact, got %q", skills[0].Name)
	}
	if skills[0].Type == model.SkillTypePrompt {
		t.Fatalf("expected same-name skill to override command, got prompt type")
	}
	if skills[0].Description != "skill version" {
		t.Fatalf("expected skill version to win, got description %q", skills[0].Description)
	}
}

func TestSyncDefaultExcludesPromptArtifacts(t *testing.T) {
	tempDir := t.TempDir()
	claudeCommands := filepath.Join(tempDir, ".claude", "commands")
	cursorSkills := filepath.Join(tempDir, ".cursor", "skills")
	if err := os.MkdirAll(claudeCommands, 0o750); err != nil {
		t.Fatalf("failed to create claude commands dir: %v", err)
	}
	if err := os.MkdirAll(cursorSkills, 0o750); err != nil {
		t.Fatalf("failed to create cursor dir: %v", err)
	}

	commandContent := `---
description: review command
allowed-tools: Bash, Read
---
Review content.`
	if err := os.WriteFile(filepath.Join(claudeCommands, "review.md"), []byte(commandContent), 0o600); err != nil {
		t.Fatalf("failed to write command file: %v", err)
	}

	t.Setenv("SKILLSYNC_HOME", tempDir)
	t.Setenv("SKILLSYNC_CLAUDE_CODE_PATH", claudeCommands)
	t.Setenv("SKILLSYNC_CLAUDE_CODE_SKILLS_PATHS", claudeCommands)
	t.Setenv("SKILLSYNC_CURSOR_PATH", cursorSkills)
	t.Setenv("SKILLSYNC_CURSOR_SKILLS_PATHS", cursorSkills)
	t.Setenv("SKILLSYNC_CODEX_PATH", filepath.Join(tempDir, ".codex"))

	ctx := context.Background()
	err := Run(ctx, []string{"skillsync", "sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor"})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(cursorSkills, "review.md")); !os.IsNotExist(err) {
		t.Fatalf("expected prompt artifact to be excluded by default sync type policy")
	}
}

func TestSyncIncludePromptsIncludesPromptArtifacts(t *testing.T) {
	tempDir := t.TempDir()
	claudeCommands := filepath.Join(tempDir, ".claude", "commands")
	cursorSkills := filepath.Join(tempDir, ".cursor", "skills")
	if err := os.MkdirAll(claudeCommands, 0o750); err != nil {
		t.Fatalf("failed to create claude commands dir: %v", err)
	}
	if err := os.MkdirAll(cursorSkills, 0o750); err != nil {
		t.Fatalf("failed to create cursor dir: %v", err)
	}

	commandContent := `---
description: review command
allowed-tools: Bash, Read
---
Review content.`
	if err := os.WriteFile(filepath.Join(claudeCommands, "review.md"), []byte(commandContent), 0o600); err != nil {
		t.Fatalf("failed to write command file: %v", err)
	}

	t.Setenv("SKILLSYNC_HOME", tempDir)
	t.Setenv("SKILLSYNC_CLAUDE_CODE_PATH", claudeCommands)
	t.Setenv("SKILLSYNC_CLAUDE_CODE_SKILLS_PATHS", claudeCommands)
	t.Setenv("SKILLSYNC_CURSOR_PATH", cursorSkills)
	t.Setenv("SKILLSYNC_CURSOR_SKILLS_PATHS", cursorSkills)
	t.Setenv("SKILLSYNC_CODEX_PATH", filepath.Join(tempDir, ".codex"))

	ctx := context.Background()
	err := Run(ctx, []string{"skillsync", "sync", "--yes", "--skip-backup", "--skip-validation", "--include-prompts", "claudecode", "cursor"})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(cursorSkills, "review.md")); err != nil {
		t.Fatalf("expected prompt artifact to be synced with --include-prompts: %v", err)
	}
}

func TestSyncIncludePromptsAutoIncludesClaudeCommandPaths(t *testing.T) {
	tempDir := t.TempDir()
	claudeCommands := filepath.Join(tempDir, ".claude", "commands")
	claudeSkills := filepath.Join(tempDir, ".claude", "skills")
	cursorSkills := filepath.Join(tempDir, ".cursor", "skills")
	if err := os.MkdirAll(claudeCommands, 0o750); err != nil {
		t.Fatalf("failed to create claude commands dir: %v", err)
	}
	if err := os.MkdirAll(claudeSkills, 0o750); err != nil {
		t.Fatalf("failed to create claude skills dir: %v", err)
	}
	if err := os.MkdirAll(cursorSkills, 0o750); err != nil {
		t.Fatalf("failed to create cursor dir: %v", err)
	}

	commandContent := `---
description: review command
allowed-tools: Bash, Read
---
Review content.`
	if err := os.WriteFile(filepath.Join(claudeCommands, "review.md"), []byte(commandContent), 0o600); err != nil {
		t.Fatalf("failed to write command file: %v", err)
	}

	// Simulate an older config that only contains Claude skills paths.
	t.Setenv("HOME", tempDir)
	t.Setenv("SKILLSYNC_HOME", filepath.Join(tempDir, ".skillsync"))
	configYAML := `platforms:
  claude_code:
    skills_paths:
      - .claude/skills
      - ~/.claude/skills
  cursor:
    skills_paths:
      - .cursor/skills
      - ~/.cursor/skills
  codex:
    skills_paths:
      - .codex/skills
      - ~/.codex/skills
sync:
  default_strategy: overwrite
  include_types:
    - skill
`
	configPath := filepath.Join(tempDir, ".skillsync", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o750); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	ctx := context.Background()
	err := Run(ctx, []string{"skillsync", "sync", "--yes", "--skip-backup", "--skip-validation", "--include-prompts", "claudecode", "cursor"})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(cursorSkills, "review.md")); err != nil {
		t.Fatalf("expected prompt artifact to be synced from .claude/commands: %v", err)
	}
}
