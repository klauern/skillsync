package e2e_test

import (
	"os"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/e2e"
)

// TestDiscoverCommand verifies discover command executes.
func TestDiscoverCommand(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("discover")

	e2e.AssertSuccess(t, result)
}

// TestDiscoverWithPlatformFilter verifies discover with platform filter.
func TestDiscoverWithPlatformFilter(t *testing.T) {
	tests := []string{"claude-code", "cursor", "codex", "pi.dev"}

	for _, platform := range tests {
		t.Run(platform, func(t *testing.T) {
			h := e2e.NewHarness(t)

			result := h.Run("discover", "--platform", platform)

			e2e.AssertSuccess(t, result)
		})
	}
}

// TestDiscoverOutputFormats verifies discover with different output formats.
// NOTE: We use --platform codex to minimize output size, as the test harness
// has issues with large outputs (pipe buffer deadlock). Codex typically has
// no skills installed, ensuring small output.
func TestDiscoverOutputFormats(t *testing.T) {
	tests := map[string]struct {
		format  string
		wantAny []string // Check if output contains at least one of these patterns
	}{
		"table format": {
			format:  "table",
			wantAny: []string{"NAME", "No skills found."},
		},
		"json format": {
			format:  "json",
			wantAny: []string{"[", "null"},
		},
		"yaml format": {
			format:  "yaml",
			wantAny: []string{"-", "[]"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := e2e.NewHarness(t)

			// Use codex platform to minimize output size (likely empty/small)
			result := h.Run("discover", "--platform", "codex", "--format", tt.format)

			e2e.AssertSuccess(t, result)
			// Check that at least one expected pattern is present
			found := false
			for _, want := range tt.wantAny {
				if strings.Contains(result.Stdout, want) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected output to contain one of %v\ngot: %s", tt.wantAny, result.Stdout)
			}
		})
	}
}

// NOTE: The following fixture-based tests verify that discover respects
// configured platform paths and the SKILLSYNC_*_SKILLS_PATHS environment
// variables set by the test harness.
// See: https://github.com/klauern/skillsync/issues

// TestDiscoverWithSkills verifies discover finds skills from fixtures.
// SKIP: discover doesn't use environment variable overrides for platform paths
func TestDiscoverWithSkills(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create test skills in Claude Code fixture
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("skill1.md", "test-skill-one", "First test skill", "# Test Skill One\n\nContent for skill one.")
	claudeFixture.WriteSkill("skill2.md", "test-skill-two", "Second test skill", "# Test Skill Two\n\nContent for skill two.")

	result := h.Run("discover")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "test-skill-one")
	e2e.AssertOutputContains(t, result, "test-skill-two")
	e2e.AssertOutputContains(t, result, "First test skill")
	// NOTE: Exact count assertion removed because discover now includes
	// installed plugin skills from ~/.claude/plugins/cache/, which varies
	// by user environment. The test verifies fixture skills are discovered.
}

// TestDiscoverMultiplePlatforms verifies discover finds skills from multiple platforms.
// SKIP: discover doesn't use environment variable overrides for platform paths
func TestDiscoverMultiplePlatforms(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create skills in multiple platforms
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("claude-skill.md", "claude-skill", "Claude Code skill", "# Claude skill")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("cursor-skill.mdc", "cursor-skill", "Cursor skill", "# Cursor skill")

	piDevFixture := h.PiDevFixture()
	piDevFixture.WriteSkill("pi-skill/SKILL.md", "pi-skill", "Pi.dev skill", "# Pi skill")

	// Run discover without platform filter
	result := h.Run("discover")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "claude-skill")
	e2e.AssertOutputContains(t, result, "cursor-skill")
	e2e.AssertOutputContains(t, result, "pi-skill")
	e2e.AssertOutputContains(t, result, "claude-code")
	e2e.AssertOutputContains(t, result, "cursor")
	e2e.AssertOutputContains(t, result, "pi.dev")
}

// TestDiscoverPlatformFilterWithSkills verifies platform filter shows only matching skills.
// SKIP: discover doesn't use environment variable overrides for platform paths
func TestDiscoverPlatformFilterWithSkills(t *testing.T) {

	h := e2e.NewHarness(t)

	// Create skills in multiple platforms
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("claude-skill.md", "claude-skill", "Claude Code skill", "# Claude skill")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("cursor-skill.mdc", "cursor-skill", "Cursor skill", "# Cursor skill")

	// Filter to Claude Code only
	result := h.Run("discover", "--platform", "claude-code")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "claude-skill")
	e2e.AssertOutputNotContains(t, result, "cursor-skill")
}

// TestDiscoverJSONFormatWithSkills verifies JSON output contains skill data.
// SKIP: discover doesn't use environment variable overrides for platform paths
func TestDiscoverJSONFormatWithSkills(t *testing.T) {

	h := e2e.NewHarness(t)

	// Create a test skill
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("json-test.md", "json-test-skill", "Skill for JSON test", "# JSON Test Skill")

	result := h.Run("discover", "--format", "json")

	e2e.AssertSuccess(t, result)
	// Verify JSON structure
	if !strings.HasPrefix(strings.TrimSpace(result.Stdout), "[") {
		t.Errorf("expected JSON array starting with [, got: %s", result.Stdout)
	}
	e2e.AssertOutputContains(t, result, `"name": "json-test-skill"`)
	e2e.AssertOutputContains(t, result, `"platform": "claude-code"`)
}

// TestDiscoverYAMLFormatWithSkills verifies YAML output contains skill data.
// SKIP: discover doesn't use environment variable overrides for platform paths
func TestDiscoverYAMLFormatWithSkills(t *testing.T) {

	h := e2e.NewHarness(t)

	// Create a test skill
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("yaml-test.md", "yaml-test-skill", "Skill for YAML test", "# YAML Test Skill")

	result := h.Run("discover", "--format", "yaml")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "name: yaml-test-skill")
	e2e.AssertOutputContains(t, result, "platform: claude-code")
}

// TestDiscoverTableFormatWithSkills verifies table output structure.
// SKIP: discover doesn't use environment variable overrides for platform paths
func TestDiscoverTableFormatWithSkills(t *testing.T) {

	h := e2e.NewHarness(t)

	// Create a test skill
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("table-test.md", "table-test-skill", "Skill for table test", "# Table Test Skill")

	result := h.Run("discover", "--format", "table")

	e2e.AssertSuccess(t, result)
	// Verify table headers
	e2e.AssertOutputContains(t, result, "NAME")
	e2e.AssertOutputContains(t, result, "PLATFORM")
	e2e.AssertOutputContains(t, result, "DESCRIPTION")
	// Verify skill data
	e2e.AssertOutputContains(t, result, "table-test-skill")
	e2e.AssertOutputContains(t, result, "Total: 1 skill(s)")
}

// TestDiscoverInvalidPlatform verifies discover rejects invalid platform.
func TestDiscoverInvalidPlatform(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("discover", "--platform", "invalid-platform")

	e2e.AssertError(t, result)
	e2e.AssertErrorContains(t, result, "invalid platform")
}

// TestDiscoverInvalidFormat verifies discover rejects invalid format.
func TestDiscoverInvalidFormat(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("discover", "--format", "invalid-format")

	e2e.AssertError(t, result)
	e2e.AssertErrorContains(t, result, "unsupported format")
}

// TestDiscoverHelp verifies discover help output.
func TestDiscoverHelp(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("discover", "--help")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "USAGE")
	e2e.AssertOutputContains(t, result, "--platform")
	e2e.AssertOutputContains(t, result, "--format")
	e2e.AssertOutputContains(t, result, "--no-plugins")
	e2e.AssertOutputContains(t, result, "--repo")
	e2e.AssertOutputContains(t, result, "--no-cache")
}

// TestDiscoverAliases verifies discover command aliases work.
func TestDiscoverAliases(t *testing.T) {
	aliases := []string{"discover", "discovery", "list"}

	for _, alias := range aliases {
		t.Run(alias, func(t *testing.T) {
			h := e2e.NewHarness(t)

			result := h.Run(alias)

			e2e.AssertSuccess(t, result)
		})
	}
}

// TestDiscoverShortFlags verifies short flag versions work.
func TestDiscoverShortFlags(t *testing.T) {
	h := e2e.NewHarness(t)

	// Test -p for --platform (use codex to minimize output)
	result := h.Run("discover", "-p", "codex")
	e2e.AssertSuccess(t, result)

	// Test -f for --format (use codex to minimize output, verify JSON starts with [ or null)
	result = h.Run("discover", "-p", "codex", "-f", "json")
	e2e.AssertSuccess(t, result)
	trimmed := strings.TrimSpace(result.Stdout)
	if !strings.HasPrefix(trimmed, "[") && trimmed != "null" {
		t.Errorf("expected JSON output starting with [ or null, got: %s", result.Stdout)
	}
}

// TestDiscoverPiDev verifies Pi.dev skills can be discovered directly.
func TestDiscoverPiDev(t *testing.T) {
	h := e2e.NewHarness(t)

	piFixture := h.PiDevFixture()
	piFixture.WriteSkill("discoverable/SKILL.md", "discoverable", "Pi discovery", "# Discoverable\n")

	result := h.Run("discover", "--platform", "pi-dev", "--format", "json")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, `"platform": "pi.dev"`)
	e2e.AssertOutputContains(t, result, `"name": "discoverable"`)
	e2e.AssertOutputContains(t, result, `"name": "agents"`)
}

// TestDiscoverClaudeCommandArtifactsAsPrompts verifies command-style files are
// discovered as prompt artifacts.
func TestDiscoverClaudeCommandArtifactsAsPrompts(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteFile("review.md", `---
description: review command
allowed-tools: Bash, Read
---
# /review

Review this code.`)

	result := h.Run("discover", "--platform", "claudecode", "--type", "prompt", "--format", "json", "--no-plugins")
	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, `"name": "review"`)
	e2e.AssertOutputContains(t, result, `"type": "prompt"`)
	e2e.AssertOutputContains(t, result, `"trigger": "/review"`)
}

// TestDiscoverCopilot verifies Copilot skills can be discovered directly.
func TestDiscoverCopilot(t *testing.T) {
	h := e2e.NewHarness(t)

	copilotFixture := h.CopilotFixture()
	copilotFixture.WriteFile("copilot-instructions.md", "# Copilot Instructions\n\nWorkspace rules.")

	result := h.Run("discover", "--platform", "copilot", "--format", "json")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, `"platform": "copilot"`)
}

// TestDiscoverGemini verifies Gemini skills can be discovered directly.
func TestDiscoverGemini(t *testing.T) {
	h := e2e.NewHarness(t)

	geminiFixture := h.GeminiFixture()
	if err := os.MkdirAll(geminiFixture.Path("skills/my-gemini"), 0o750); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	geminiFixture.WriteSkill("skills/my-gemini/SKILL.md", "my-gemini", "Gemini discovery", "# My Gemini Skill\n")

	result := h.Run("discover", "--platform", "gemini", "--format", "json")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, `"platform": "gemini"`)
	e2e.AssertOutputContains(t, result, `"name": "my-gemini"`)
}

// TestDiscoverGeminiTOMLCommands verifies Gemini TOML commands are discovered.
func TestDiscoverGeminiTOMLCommands(t *testing.T) {
	h := e2e.NewHarness(t)

	geminiFixture := h.GeminiFixture()
	if err := os.MkdirAll(geminiFixture.Path("commands"), 0o750); err != nil {
		t.Fatalf("failed to create commands dir: %v", err)
	}
	geminiFixture.WriteFile("commands/review.toml", `description = "Review code"
prompt = "Review this: {{args}}"
`)

	result := h.Run("discover", "--platform", "gemini", "--format", "json")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, `"platform": "gemini"`)
	e2e.AssertOutputContains(t, result, `"name": "review"`)
}

// TestCompareCommandBasic verifies the compare command runs successfully.
func TestCompareCommandBasic(t *testing.T) {
	h := e2e.NewHarness(t)

	src := h.ClaudeCodeFixture()
	src.WriteSkill("a.md", "alpha", "", "# A")
	src.WriteSkill("b.md", "alpha-copy", "", "# A")

	result := h.Run("compare", "--platform", "claude-code", "--format", "summary")
	e2e.AssertSuccess(t, result)
}
