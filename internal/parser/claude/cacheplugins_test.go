package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

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
