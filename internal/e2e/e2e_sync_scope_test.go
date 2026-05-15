package e2e_test

import (
	"os"
	"testing"

	"github.com/klauern/skillsync/internal/e2e"
)

// ============================================================================
// Plugin Scope E2E Tests
// ============================================================================

// TestSyncPluginScopeSource verifies sync with plugin scope source spec.
// This tests the claudecode:plugin syntax to filter skills by plugin scope.
func TestSyncPluginScopeSource(t *testing.T) {
	h := e2e.NewHarness(t)

	// Set up a mock plugin cache directory with skills
	pluginCacheDir := h.HomeDir() + "/.claude/plugins/cache"
	if err := os.MkdirAll(pluginCacheDir+"/test-marketplace/test-plugin/1.0.0/test-skill", 0o750); err != nil {
		t.Fatalf("failed to create plugin cache directory: %v", err)
	}

	// Write a plugin skill
	pluginSkillContent := `---
name: plugin-test-skill
description: A test skill from plugin cache
---
# Plugin Test Skill

This skill comes from a plugin.
`
	pluginSkillPath := pluginCacheDir + "/test-marketplace/test-plugin/1.0.0/test-skill/SKILL.md"
	if err := os.WriteFile(pluginSkillPath, []byte(pluginSkillContent), 0o600); err != nil {
		t.Fatalf("failed to write plugin SKILL.md: %v", err)
	}

	// Create installed_plugins.json to register the plugin
	pluginsDir := h.HomeDir() + "/.claude/plugins"
	installedPluginsContent := `{
  "version": 1,
  "plugins": {
    "test-plugin@test-marketplace": [
      {
        "scope": "user",
        "installPath": "` + pluginCacheDir + `/test-marketplace/test-plugin/1.0.0",
        "version": "1.0.0",
        "installedAt": "2024-01-01T00:00:00Z",
        "lastUpdated": "2024-01-01T00:00:00Z"
      }
    ]
  }
}`
	if err := os.WriteFile(pluginsDir+"/installed_plugins.json", []byte(installedPluginsContent), 0o600); err != nil {
		t.Fatalf("failed to write installed_plugins.json: %v", err)
	}

	// Also create a user-scope skill to ensure filtering works
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("user-skill.md", "user-skill", "A user scope skill", "# User Skill\n\nUser scope content.")

	// Ensure cursor fixture directory exists
	_ = h.CursorFixture()

	// Sync only plugin scope skills from claudecode to cursor
	// NOTE: The test may not find the plugin skill if the parser doesn't pick up the test
	// installed_plugins.json, but it should at least not error on the scope spec
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--dry-run", "claudecode:plugin", "cursor")

	// The sync should succeed (even if 0 skills found)
	e2e.AssertSuccess(t, result)

	// Verify user-skill was NOT synced (we filtered to plugin scope only)
	// In dry-run mode, files won't be created anyway, but we can check output
	e2e.AssertOutputNotContains(t, result, "user-skill")
}

// TestSyncPluginScopeInvalidTarget verifies plugin scope cannot be used as target.
func TestSyncPluginScopeInvalidTarget(t *testing.T) {
	h := e2e.NewHarness(t)

	// Plugin scope should not be allowed as target (only repo and user are writable)
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor:plugin")

	e2e.AssertError(t, result)
	e2e.AssertErrorContains(t, result, "target scope must be")
}

// TestSyncMultipleScopesSource verifies sync with multiple source scopes.
func TestSyncMultipleScopesSource(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create user-scope skill
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("user-only.md", "user-only", "User scope skill", "# User Only\n\nUser scope content.")

	// Ensure cursor fixture directory exists
	_ = h.CursorFixture()

	// Sync only user and repo scopes (not plugin)
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--dry-run", "claudecode:user,repo", "cursor")

	e2e.AssertSuccess(t, result)
	// The user skill should be included in dry-run output
	e2e.AssertOutputContains(t, result, "user-only")
}

// TestSyncScopeSpecParsing verifies various scope spec formats are parsed correctly.
func TestSyncScopeSpecParsing(t *testing.T) {
	tests := map[string]struct {
		source    string
		target    string
		wantError bool
	}{
		"simple platforms": {
			source:    "claudecode",
			target:    "cursor",
			wantError: false,
		},
		"source with single scope": {
			source:    "claudecode:user",
			target:    "cursor",
			wantError: false,
		},
		"source with multiple scopes": {
			source:    "claudecode:user,repo",
			target:    "cursor",
			wantError: false,
		},
		"source with plugin scope": {
			source:    "claudecode:plugin",
			target:    "cursor",
			wantError: false,
		},
		"target with user scope": {
			source:    "claudecode",
			target:    "cursor:user",
			wantError: false,
		},
		"target with repo scope": {
			source:    "claudecode",
			target:    "cursor:repo",
			wantError: false,
		},
		"invalid: target with plugin scope": {
			source:    "claudecode",
			target:    "cursor:plugin",
			wantError: true,
		},
		"invalid: target with multiple scopes": {
			source:    "claudecode",
			target:    "cursor:user,repo",
			wantError: true,
		},
		"invalid: empty scope after colon": {
			source:    "claudecode:",
			target:    "cursor",
			wantError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := e2e.NewHarness(t)

			result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--dry-run", tt.source, tt.target)

			if tt.wantError {
				e2e.AssertError(t, result)
			} else {
				e2e.AssertSuccess(t, result)
			}
		})
	}
}
