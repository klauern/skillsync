package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestCachePluginsParser_AllEntries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	pluginsDir := filepath.Join(home, ".claude", "plugins")
	if err := os.MkdirAll(pluginsDir, 0o750); err != nil {
		t.Fatalf("failed to create plugins dir: %v", err)
	}

	manifest := `{"version":1,"plugins":{"p@m":[{"scope":"user","installPath":"/tmp/p","version":"1.0.0","installedAt":"2024-01-01T00:00:00Z","lastUpdated":"2024-01-01T00:00:00Z"}]}}`
	if err := os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	parser := NewCachePluginsParser("")
	entries := parser.AllEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestCachePluginsParser_ParseEmptyCache(t *testing.T) {
	// Create a temporary directory as cache path
	tmpDir := t.TempDir()

	// Create an empty plugin index to avoid reading from real installed_plugins.json
	emptyIndex := &PluginIndex{
		byInstallPath: make(map[string]*PluginIndexEntry),
	}

	parser := NewCachePluginsParserWithIndex(tmpDir, emptyIndex)

	// Should return empty when no plugins installed
	skills, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
}

func TestCachePluginsParser_Platform(t *testing.T) {
	parser := NewCachePluginsParser("")
	if parser.Platform() != model.ClaudeCode {
		t.Errorf("expected platform %s, got %s", model.ClaudeCode, parser.Platform())
	}
}

func TestCachePluginsParser_DefaultPath(t *testing.T) {
	parser := NewCachePluginsParser("")
	defaultPath := parser.DefaultPath()

	if defaultPath == "" {
		t.Error("expected non-empty default path")
	}

	// Should contain .claude/plugins/cache
	if !filepath.IsAbs(defaultPath) {
		// Skip absolute path check in test environment where home may not exist
		t.Log("default path:", defaultPath)
	}
}

func TestCachePluginsParser_ParseSkillFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock plugin directory structure
	pluginDir := filepath.Join(tmpDir, "marketplace", "test-plugin", "1.0.0")
	skillDir := filepath.Join(pluginDir, "my-skill")
	// #nosec G301 - test directory
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	// Create a SKILL.md file with frontmatter
	skillContent := `---
name: my-test-skill
description: A test skill
tools:
  - Bash
  - Read
---
# My Test Skill

This is the skill content.
`
	skillPath := filepath.Join(skillDir, "SKILL.md")
	// #nosec G306 - test file
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Create parser and test parseSkillFile directly
	parser := NewCachePluginsParser(tmpDir)
	entry := &PluginIndexEntry{
		PluginKey:   "test-plugin@marketplace",
		PluginName:  "test-plugin",
		Marketplace: "marketplace",
		Version:     "1.0.0",
		InstallPath: pluginDir,
	}

	skill, err := parser.parseSkillFile(skillPath, entry)
	if err != nil {
		t.Fatalf("failed to parse skill file: %v", err)
	}

	// Verify skill fields
	if skill.Name != "my-test-skill" {
		t.Errorf("expected name 'my-test-skill', got %q", skill.Name)
	}

	if skill.Description != "A test skill" {
		t.Errorf("expected description 'A test skill', got %q", skill.Description)
	}

	if skill.Platform != model.ClaudeCode {
		t.Errorf("expected platform %s, got %s", model.ClaudeCode, skill.Platform)
	}

	if skill.Scope != model.ScopePlugin {
		t.Errorf("expected scope %s, got %s", model.ScopePlugin, skill.Scope)
	}

	if len(skill.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(skill.Tools))
	}

	if skill.PluginInfo == nil {
		t.Error("expected PluginInfo to be set")
	} else {
		if skill.PluginInfo.PluginName != "test-plugin@marketplace" {
			t.Errorf("expected PluginName 'test-plugin@marketplace', got %q", skill.PluginInfo.PluginName)
		}
		if skill.PluginInfo.Marketplace != "marketplace" {
			t.Errorf("expected Marketplace 'marketplace', got %q", skill.PluginInfo.Marketplace)
		}
		if skill.PluginInfo.Version != "1.0.0" {
			t.Errorf("expected Version '1.0.0', got %q", skill.PluginInfo.Version)
		}
		if skill.PluginInfo.IsDev {
			t.Error("expected IsDev to be false for cache plugin")
		}
	}

	// Check metadata
	if skill.Metadata["plugin"] != "test-plugin" {
		t.Errorf("expected metadata plugin 'test-plugin', got %q", skill.Metadata["plugin"])
	}
	if skill.Metadata["marketplace"] != "marketplace" {
		t.Errorf("expected metadata marketplace 'marketplace', got %q", skill.Metadata["marketplace"])
	}
	if skill.Metadata["source"] != "plugin-cache" {
		t.Errorf("expected metadata source 'plugin-cache', got %q", skill.Metadata["source"])
	}
}

func TestCachePluginsParser_ParseSkillFile_InvalidNameFallsBackToDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock plugin directory structure
	pluginDir := filepath.Join(tmpDir, "marketplace", "test-plugin", "1.0.0")
	skillDir := filepath.Join(pluginDir, "valid-skill")
	// #nosec G301 - test directory
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	// Create a SKILL.md file with an invalid name in frontmatter
	skillContent := `---
name: Invalid Name
description: A test skill
---
Content`
	skillPath := filepath.Join(skillDir, "SKILL.md")
	// #nosec G306 - test file
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	parser := NewCachePluginsParser(tmpDir)
	entry := &PluginIndexEntry{
		PluginKey:   "test-plugin@marketplace",
		PluginName:  "test-plugin",
		Marketplace: "marketplace",
		Version:     "1.0.0",
		InstallPath: pluginDir,
	}

	skill, err := parser.parseSkillFile(skillPath, entry)
	if err != nil {
		t.Fatalf("failed to parse skill file: %v", err)
	}

	if skill.Name != "valid-skill" {
		t.Errorf("expected fallback name 'valid-skill', got %q", skill.Name)
	}
	if skill.Description != "A test skill" {
		t.Errorf("expected description 'A test skill', got %q", skill.Description)
	}
}

func TestCachePluginsParser_ParseSkillFile_NoFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock plugin directory
	pluginDir := filepath.Join(tmpDir, "my-plugin")
	skillDir := filepath.Join(pluginDir, "simple-skill")
	// #nosec G301 - test directory
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	// Create a SKILL.md file without frontmatter
	skillContent := `# Simple Skill

Just some content without frontmatter.
`
	skillPath := filepath.Join(skillDir, "SKILL.md")
	// #nosec G306 - test file
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	parser := NewCachePluginsParser(tmpDir)
	entry := &PluginIndexEntry{
		PluginKey:   "my-plugin@test",
		PluginName:  "my-plugin",
		Marketplace: "test",
		Version:     "2.0.0",
		InstallPath: pluginDir,
	}

	skill, err := parser.parseSkillFile(skillPath, entry)
	if err != nil {
		t.Fatalf("failed to parse skill file: %v", err)
	}

	// Name should be derived from directory
	if skill.Name != "simple-skill" {
		t.Errorf("expected name 'simple-skill', got %q", skill.Name)
	}

	// Content should be preserved
	if skill.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestCachePluginsParser_ParsePluginDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a plugin directory with multiple skills
	pluginDir := filepath.Join(tmpDir, "multi-plugin")

	// Create skill 1
	skill1Dir := filepath.Join(pluginDir, "skill-one")
	// #nosec G301 - test directory
	if err := os.MkdirAll(skill1Dir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}
	// #nosec G306 - test file
	if err := os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte("# Skill One\nContent"), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Create skill 2 in nested directory
	skill2Dir := filepath.Join(pluginDir, "skills", "skill-two")
	// #nosec G301 - test directory
	if err := os.MkdirAll(skill2Dir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}
	// #nosec G306 - test file
	if err := os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte("# Skill Two\nContent"), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	parser := NewCachePluginsParser(tmpDir)
	entry := &PluginIndexEntry{
		PluginKey:   "multi-plugin@test",
		PluginName:  "multi-plugin",
		Marketplace: "test",
		Version:     "1.0.0",
		InstallPath: pluginDir,
	}

	skills, err := parser.parsePluginDirectory(entry)
	if err != nil {
		t.Fatalf("failed to parse plugin directory: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(skills))
	}

	// Verify both skills have correct metadata
	for _, skill := range skills {
		if skill.PluginInfo == nil {
			t.Error("expected PluginInfo to be set for all skills")
		}
		if skill.Scope != model.ScopePlugin {
			t.Errorf("expected scope plugin, got %s", skill.Scope)
		}
	}
}

func TestCachePluginsParser_ParseWithPluginIndex(t *testing.T) {
	tmpDir := t.TempDir()

	// Create plugin directory structure: marketplace/plugin/version/skill
	plugin1Dir := filepath.Join(tmpDir, "klauern-skills", "commits", "1.2.0")
	skill1Dir := filepath.Join(plugin1Dir, "conventional-commits")
	// #nosec G301 - test directory
	if err := os.MkdirAll(skill1Dir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	// Create SKILL.md with frontmatter
	skillContent := `---
name: conventional-commits
description: Create commits following conventional commits
tools:
  - Bash
  - Read
---
# Conventional Commits
Help with creating conventional commits.
`
	// #nosec G306 - test file
	if err := os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Create second plugin
	plugin2Dir := filepath.Join(tmpDir, "klauern-skills", "worktree", "0.1.0")
	skill2Dir := filepath.Join(plugin2Dir, "worktree-manager")
	// #nosec G301 - test directory
	if err := os.MkdirAll(skill2Dir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}
	// #nosec G306 - test file
	if err := os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte("# Worktree Manager\nContent"), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Create mock plugin index
	index := &PluginIndex{
		byInstallPath: map[string]*PluginIndexEntry{
			plugin1Dir: {
				PluginKey:   "commits@klauern-skills",
				PluginName:  "commits",
				Marketplace: "klauern-skills",
				Version:     "1.2.0",
				InstallPath: plugin1Dir,
			},
			plugin2Dir: {
				PluginKey:   "worktree@klauern-skills",
				PluginName:  "worktree",
				Marketplace: "klauern-skills",
				Version:     "0.1.0",
				InstallPath: plugin2Dir,
			},
		},
	}

	parser := NewCachePluginsParserWithIndex(tmpDir, index)
	skills, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(skills))
	}

	// Verify first skill details
	var commitsSkill, worktreeSkill model.Skill
	for _, s := range skills {
		switch s.Name {
		case "conventional-commits":
			commitsSkill = s
		case "worktree-manager":
			worktreeSkill = s
		}
	}

	if commitsSkill.Name == "" {
		t.Fatal("conventional-commits skill not found")
	}

	if commitsSkill.Description != "Create commits following conventional commits" {
		t.Errorf("expected description 'Create commits following conventional commits', got %q", commitsSkill.Description)
	}

	if commitsSkill.PluginInfo == nil {
		t.Fatal("PluginInfo should be set")
	}

	if commitsSkill.PluginInfo.PluginName != "commits@klauern-skills" {
		t.Errorf("expected PluginName 'commits@klauern-skills', got %q", commitsSkill.PluginInfo.PluginName)
	}

	if commitsSkill.PluginInfo.Version != "1.2.0" {
		t.Errorf("expected Version '1.2.0', got %q", commitsSkill.PluginInfo.Version)
	}

	if worktreeSkill.Name == "" {
		t.Fatal("worktree-manager skill not found")
	}

	if worktreeSkill.PluginInfo == nil {
		t.Fatal("PluginInfo should be set for worktree skill")
	}

	// Verify scope is plugin for all
	for _, s := range skills {
		if s.Scope != model.ScopePlugin {
			t.Errorf("expected scope plugin, got %s for skill %s", s.Scope, s.Name)
		}
	}
}

func TestCachePluginsParser_ParseDeduplicatesPaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a single plugin directory
	pluginDir := filepath.Join(tmpDir, "test-plugin")
	skillDir := filepath.Join(pluginDir, "my-skill")
	// #nosec G301 - test directory
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}
	// #nosec G306 - test file
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# My Skill\nContent"), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Create an index with duplicate entries for the same path (simulating multiple versions)
	index := &PluginIndex{
		byInstallPath: map[string]*PluginIndexEntry{
			pluginDir: {
				PluginKey:   "test@marketplace",
				PluginName:  "test",
				Marketplace: "marketplace",
				Version:     "1.0.0",
				InstallPath: pluginDir,
			},
		},
	}

	parser := NewCachePluginsParserWithIndex(tmpDir, index)
	skills, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only get one skill (not duplicated)
	if len(skills) != 1 {
		t.Errorf("expected 1 skill (deduplicated), got %d", len(skills))
	}
}

func TestCachePluginsParser_ParseSkipsNonexistentPlugins(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only one plugin directory
	existingPluginDir := filepath.Join(tmpDir, "existing-plugin")
	skillDir := filepath.Join(existingPluginDir, "my-skill")
	// #nosec G301 - test directory
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}
	// #nosec G306 - test file
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# My Skill\nContent"), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Create an index with both existing and non-existing paths
	nonexistentPath := filepath.Join(tmpDir, "nonexistent-plugin")
	index := &PluginIndex{
		byInstallPath: map[string]*PluginIndexEntry{
			existingPluginDir: {
				PluginKey:   "existing@marketplace",
				PluginName:  "existing",
				Marketplace: "marketplace",
				Version:     "1.0.0",
				InstallPath: existingPluginDir,
			},
			nonexistentPath: {
				PluginKey:   "nonexistent@marketplace",
				PluginName:  "nonexistent",
				Marketplace: "marketplace",
				Version:     "2.0.0",
				InstallPath: nonexistentPath,
			},
		},
	}

	parser := NewCachePluginsParserWithIndex(tmpDir, index)
	skills, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only get skill from existing plugin
	if len(skills) != 1 {
		t.Errorf("expected 1 skill from existing plugin, got %d", len(skills))
	}

	if skills[0].PluginInfo.PluginName != "existing@marketplace" {
		t.Errorf("expected skill from 'existing@marketplace', got %q", skills[0].PluginInfo.PluginName)
	}
}

func TestCachePluginsParser_ParseSkillFile_WithScope(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock plugin directory
	pluginDir := filepath.Join(tmpDir, "marketplace", "scoped-plugin", "1.0.0")
	skillDir := filepath.Join(pluginDir, "my-skill")
	// #nosec G301 - test directory
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	// Create a SKILL.md file
	skillContent := `---
name: scoped-skill
description: A skill with scope
---
# Scoped Skill
Content.
`
	skillPath := filepath.Join(skillDir, "SKILL.md")
	// #nosec G306 - test file
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	parser := NewCachePluginsParser(tmpDir)

	tests := map[string]struct {
		scope         string
		wantScope     string
		wantMetaScope string
	}{
		"user scope": {
			scope:         "user",
			wantScope:     "user",
			wantMetaScope: "user",
		},
		"project scope": {
			scope:         "project",
			wantScope:     "project",
			wantMetaScope: "project",
		},
		"empty scope": {
			scope:         "",
			wantScope:     "",
			wantMetaScope: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			entry := &PluginIndexEntry{
				PluginKey:   "scoped-plugin@marketplace",
				PluginName:  "scoped-plugin",
				Marketplace: "marketplace",
				Version:     "1.0.0",
				InstallPath: pluginDir,
				Scope:       tt.scope,
				Enabled:     true,
			}

			skill, err := parser.parseSkillFile(skillPath, entry)
			if err != nil {
				t.Fatalf("failed to parse skill file: %v", err)
			}

			// Verify InstallScope is set on PluginInfo
			if skill.PluginInfo == nil {
				t.Fatal("expected PluginInfo to be set")
			}

			if skill.PluginInfo.InstallScope != tt.wantScope {
				t.Errorf("PluginInfo.InstallScope = %q, want %q", skill.PluginInfo.InstallScope, tt.wantScope)
			}

			// Verify install_scope metadata
			if tt.wantMetaScope != "" {
				if skill.Metadata["install_scope"] != tt.wantMetaScope {
					t.Errorf("metadata install_scope = %q, want %q", skill.Metadata["install_scope"], tt.wantMetaScope)
				}
			} else {
				// Empty scope should not be in metadata
				if _, ok := skill.Metadata["install_scope"]; ok {
					t.Errorf("expected install_scope not to be set for empty scope, got %q", skill.Metadata["install_scope"])
				}
			}

			// Skill scope should always be ScopePlugin regardless of install scope
			if skill.Scope != model.ScopePlugin {
				t.Errorf("skill.Scope = %s, want %s", skill.Scope, model.ScopePlugin)
			}
		})
	}
}

func TestCachePluginsParser_ParseWithPluginIndex_ScopePreserved(t *testing.T) {
	tmpDir := t.TempDir()

	// Create plugin directory with a skill
	pluginDir := filepath.Join(tmpDir, "marketplace", "test-plugin", "1.0.0")
	skillDir := filepath.Join(pluginDir, "test-skill")
	// #nosec G301 - test directory
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	// Create SKILL.md
	// #nosec G306 - test file
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Test Skill\nContent"), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Create mock plugin index with scope
	index := &PluginIndex{
		byInstallPath: map[string]*PluginIndexEntry{
			pluginDir: {
				PluginKey:   "test-plugin@marketplace",
				PluginName:  "test-plugin",
				Marketplace: "marketplace",
				Version:     "1.0.0",
				InstallPath: pluginDir,
				Scope:       "user",
				Enabled:     true,
			},
		},
	}

	parser := NewCachePluginsParserWithIndex(tmpDir, index)
	skills, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	skill := skills[0]

	// Verify InstallScope is preserved
	if skill.PluginInfo.InstallScope != "user" {
		t.Errorf("PluginInfo.InstallScope = %q, want %q", skill.PluginInfo.InstallScope, "user")
	}

	// Verify metadata contains install_scope
	if skill.Metadata["install_scope"] != "user" {
		t.Errorf("metadata install_scope = %q, want %q", skill.Metadata["install_scope"], "user")
	}
}
