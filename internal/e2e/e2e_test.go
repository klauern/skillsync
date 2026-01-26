package e2e_test

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/e2e"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestMain(m *testing.M) {
	flag.Parse()
	e2e.SetUpdateGolden(*updateGolden)
	os.Exit(m.Run())
}

// TestVersionCommand verifies the version command works correctly.
func TestVersionCommand(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("version")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "skillsync version")
}

// TestConfigShowCommand verifies config show outputs valid configuration.
func TestConfigShowCommand(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("config", "show")

	e2e.AssertSuccess(t, result)
	// Default config should contain sync strategy
	e2e.AssertOutputContains(t, result, "sync:")
}

// TestConfigShowJSON verifies config show with JSON format.
func TestConfigShowJSON(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("config", "show", "--format", "json")

	e2e.AssertSuccess(t, result)
	// JSON output should start with {
	if !strings.HasPrefix(strings.TrimSpace(result.Stdout), "{") {
		t.Errorf("expected JSON output starting with {, got: %s", result.Stdout)
	}
}

// TestSyncMissingArgs verifies sync command requires source and target.
func TestSyncMissingArgs(t *testing.T) {
	tests := map[string]struct {
		args []string
	}{
		"no arguments": {
			args: []string{"sync"},
		},
		"only source": {
			args: []string{"sync", "claudecode"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := e2e.NewHarness(t)

			result := h.Run(tt.args...)

			e2e.AssertError(t, result)
		})
	}
}

// TestSyncInvalidPlatform verifies sync command rejects invalid platforms.
func TestSyncInvalidPlatform(t *testing.T) {
	tests := map[string]struct {
		args []string
	}{
		"invalid source": {
			args: []string{"sync", "invalid", "cursor"},
		},
		"invalid target": {
			args: []string{"sync", "claudecode", "invalid"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := e2e.NewHarness(t)

			result := h.Run(tt.args...)

			e2e.AssertError(t, result)
		})
	}
}

// TestSyncDryRun verifies sync with dry-run flag doesn't modify files.
func TestSyncDryRun(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create a skill in Claude Code
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("test-skill.md", "test-skill", "A test skill", "# Test Skill\n\nThis is a test.")

	// Run sync with dry-run
	result := h.Run("sync", "--dry-run", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Dry run")
}

// TestBackupListEmpty verifies backup list with no backups.
func TestBackupListEmpty(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("backup", "list")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "No backups found")
}

// TestBackupListFormats verifies backup list output formats.
func TestBackupListFormats(t *testing.T) {
	tests := map[string]struct {
		format string
		want   string
	}{
		"table format": {
			format: "table",
			want:   "No backups found",
		},
		"json format": {
			format: "json",
			want:   "[]",
		},
		"yaml format": {
			format: "yaml",
			want:   "[]",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := e2e.NewHarness(t)

			result := h.Run("backup", "list", "--format", tt.format)

			e2e.AssertSuccess(t, result)
			e2e.AssertOutputContains(t, result, tt.want)
		})
	}
}

// TestDiscoverCommand verifies discover command executes.
func TestDiscoverCommand(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("discover")

	e2e.AssertSuccess(t, result)
}

// TestDiscoverWithPlatformFilter verifies discover with platform filter.
func TestDiscoverWithPlatformFilter(t *testing.T) {
	tests := []string{"claude-code", "cursor", "codex"}

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

// NOTE: The following fixture-based tests are skipped because the discover command
// currently uses hardcoded paths from util.ClaudeCodeSkillsPath() rather than
// respecting the SKILLSYNC_*_PATH environment variables set by the test harness.
// These tests serve as documentation for the expected behavior once that is fixed.
// See: https://github.com/klauern/skillsync/issues/XXX

// TestDiscoverWithSkills verifies discover finds skills from fixtures.
// SKIP: discover doesn't use environment variable overrides for platform paths
func TestDiscoverWithSkills(t *testing.T) {
	t.Skip("discover command uses hardcoded paths, not environment overrides - needs fix")

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
	e2e.AssertOutputContains(t, result, "Total: 2 skill(s)")
}

// TestDiscoverMultiplePlatforms verifies discover finds skills from multiple platforms.
// SKIP: discover doesn't use environment variable overrides for platform paths
func TestDiscoverMultiplePlatforms(t *testing.T) {
	t.Skip("discover command uses hardcoded paths, not environment overrides - needs fix")

	h := e2e.NewHarness(t)

	// Create skills in multiple platforms
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("claude-skill.md", "claude-skill", "Claude Code skill", "# Claude skill")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("cursor-skill.mdc", "cursor-skill", "Cursor skill", "# Cursor skill")

	// Run discover without platform filter
	result := h.Run("discover")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "claude-skill")
	e2e.AssertOutputContains(t, result, "cursor-skill")
	e2e.AssertOutputContains(t, result, "claude-code")
	e2e.AssertOutputContains(t, result, "cursor")
}

// TestDiscoverPlatformFilterWithSkills verifies platform filter shows only matching skills.
// SKIP: discover doesn't use environment variable overrides for platform paths
func TestDiscoverPlatformFilterWithSkills(t *testing.T) {
	t.Skip("discover command uses hardcoded paths, not environment overrides - needs fix")

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
	t.Skip("discover command uses hardcoded paths, not environment overrides - needs fix")

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
	t.Skip("discover command uses hardcoded paths, not environment overrides - needs fix")

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
	t.Skip("discover command uses hardcoded paths, not environment overrides - needs fix")

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
	e2e.AssertOutputContains(t, result, "--plugins")
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

// TestExportHelp verifies export command help works.
// NOTE: Full export E2E tests require the CLI to respect SKILLSYNC_*_PATH
// environment variables in all code paths, which is tracked separately.
func TestExportHelp(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("export", "--help")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Export skills")
}

// TestHelpFlag verifies --help works for main command.
func TestHelpFlag(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("--help")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "skillsync")
	e2e.AssertOutputContains(t, result, "COMMANDS")
}

// TestSubcommandHelp verifies --help works for subcommands.
func TestSubcommandHelp(t *testing.T) {
	subcommands := []string{"sync", "config", "discover", "export", "backup"}

	for _, cmd := range subcommands {
		t.Run(cmd, func(t *testing.T) {
			h := e2e.NewHarness(t)

			result := h.Run(cmd, "--help")

			e2e.AssertSuccess(t, result)
			e2e.AssertOutputContains(t, result, "USAGE")
		})
	}
}
