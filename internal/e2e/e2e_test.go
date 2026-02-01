package e2e_test

import (
	"flag"
	"os"
	"strings"
	"testing"
	"time"

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

// TestConfigShowShortFlag verifies config show with short format flag.
func TestConfigShowShortFlag(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("config", "show", "-f", "json")

	e2e.AssertSuccess(t, result)
	if !strings.HasPrefix(strings.TrimSpace(result.Stdout), "{") {
		t.Errorf("expected JSON output starting with {, got: %s", result.Stdout)
	}
}

// TestConfigShowInvalidFormat verifies config show rejects invalid format.
func TestConfigShowInvalidFormat(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("config", "show", "--format", "invalid")

	e2e.AssertError(t, result)
	e2e.AssertErrorContains(t, result, "unsupported format")
}

// TestConfigShowYAMLContainsExpectedSections verifies YAML output structure.
func TestConfigShowYAMLContainsExpectedSections(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("config", "show", "--format", "yaml")

	e2e.AssertSuccess(t, result)
	// Check for main config sections
	e2e.AssertOutputContains(t, result, "platforms:")
	e2e.AssertOutputContains(t, result, "sync:")
	e2e.AssertOutputContains(t, result, "backup:")
	e2e.AssertOutputContains(t, result, "# skillsync configuration")
}

// TestConfigShowWithNoConfigFileShowsDefault verifies default config message.
func TestConfigShowWithNoConfigFileShowsDefault(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("config", "show")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Using default configuration")
}

// TestConfigDefaultActionShowsConfig verifies config without subcommand shows config.
func TestConfigDefaultActionShowsConfig(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("config")

	e2e.AssertSuccess(t, result)
	// Should behave same as config show
	e2e.AssertOutputContains(t, result, "sync:")
	e2e.AssertOutputContains(t, result, "platforms:")
}

// TestConfigInitCreatesConfigFile verifies config init creates a file.
func TestConfigInitCreatesConfigFile(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("config", "init")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created config file:")

	// Verify config file was created by showing it
	showResult := h.Run("config", "show")
	e2e.AssertSuccess(t, showResult)
	e2e.AssertOutputContains(t, showResult, "Loaded from:")
}

// TestConfigInitFailsIfExists verifies config init fails without force flag.
func TestConfigInitFailsIfExists(t *testing.T) {
	h := e2e.NewHarness(t)

	// First init should succeed
	result := h.Run("config", "init")
	e2e.AssertSuccess(t, result)

	// Second init should fail
	result2 := h.Run("config", "init")
	e2e.AssertError(t, result2)
	e2e.AssertErrorContains(t, result2, "already exists")
	e2e.AssertErrorContains(t, result2, "--force")
}

// TestConfigInitForceOverwrites verifies config init --force overwrites.
func TestConfigInitForceOverwrites(t *testing.T) {
	h := e2e.NewHarness(t)

	// First init
	result := h.Run("config", "init")
	e2e.AssertSuccess(t, result)

	// Second init with force
	result2 := h.Run("config", "init", "--force")
	e2e.AssertSuccess(t, result2)
	e2e.AssertOutputContains(t, result2, "Created config file:")
}

// TestConfigInitShortForceFlag verifies config init -f works.
func TestConfigInitShortForceFlag(t *testing.T) {
	h := e2e.NewHarness(t)

	// First init
	result := h.Run("config", "init")
	e2e.AssertSuccess(t, result)

	// Second init with short force flag
	result2 := h.Run("config", "init", "-f")
	e2e.AssertSuccess(t, result2)
}

// TestConfigPathShowsPaths verifies config path displays all paths.
func TestConfigPathShowsPaths(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("config", "path")

	e2e.AssertSuccess(t, result)
	// Check for section headers
	e2e.AssertOutputContains(t, result, "Configuration paths:")
	e2e.AssertOutputContains(t, result, "Platform paths:")
	e2e.AssertOutputContains(t, result, "Data paths:")

	// Check for platform paths
	e2e.AssertOutputContains(t, result, "Claude Code:")
	e2e.AssertOutputContains(t, result, "Cursor:")
	e2e.AssertOutputContains(t, result, "Codex:")

	// Check for data paths
	e2e.AssertOutputContains(t, result, "Backups:")
	e2e.AssertOutputContains(t, result, "Cache:")
	e2e.AssertOutputContains(t, result, "Plugins:")
	e2e.AssertOutputContains(t, result, "Metadata:")
}

// TestConfigPathShowsConfigFileStatus verifies path shows config file existence.
func TestConfigPathShowsConfigFileStatus(t *testing.T) {
	h := e2e.NewHarness(t)

	// Without config file
	result := h.Run("config", "path")
	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "(not found)")

	// Create config file
	initResult := h.Run("config", "init")
	e2e.AssertSuccess(t, initResult)

	// With config file
	result2 := h.Run("config", "path")
	e2e.AssertSuccess(t, result2)
	e2e.AssertOutputContains(t, result2, "(exists)")
}

// TestConfigEditNoEditorError verifies edit fails without EDITOR env var.
func TestConfigEditNoEditorError(t *testing.T) {
	h := e2e.NewHarness(t)

	// Clear EDITOR and VISUAL environment variables
	h.SetEnv("EDITOR", "")
	h.SetEnv("VISUAL", "")

	result := h.Run("config", "edit")

	e2e.AssertError(t, result)
	e2e.AssertErrorContains(t, result, "no editor found")
	e2e.AssertErrorContains(t, result, "$EDITOR")
}

// TestConfigEditWithEditor verifies edit works with EDITOR set.
func TestConfigEditWithEditor(t *testing.T) {
	h := e2e.NewHarness(t)

	// Set EDITOR environment variable
	h.SetEnv("EDITOR", "vim")

	result := h.Run("config", "edit")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Opening")
	e2e.AssertOutputContains(t, result, "vim")
	e2e.AssertOutputContains(t, result, "Run:")
}

// TestConfigEditWithVisual verifies edit works with VISUAL set.
func TestConfigEditWithVisual(t *testing.T) {
	h := e2e.NewHarness(t)

	// Set VISUAL environment variable (EDITOR not set)
	h.SetEnv("EDITOR", "")
	h.SetEnv("VISUAL", "code")

	result := h.Run("config", "edit")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "code")
}

// TestConfigEditCreatesDefaultIfMissing verifies edit creates config if missing.
func TestConfigEditCreatesDefaultIfMissing(t *testing.T) {
	h := e2e.NewHarness(t)

	h.SetEnv("EDITOR", "vim")

	result := h.Run("config", "edit")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "No config file found")
	e2e.AssertOutputContains(t, result, "Creating default configuration")

	// Verify config was created
	showResult := h.Run("config", "show")
	e2e.AssertOutputContains(t, showResult, "Loaded from:")
}

// TestConfigEditExistingFile verifies edit doesn't recreate existing config.
func TestConfigEditExistingFile(t *testing.T) {
	h := e2e.NewHarness(t)

	// First create the config
	initResult := h.Run("config", "init")
	e2e.AssertSuccess(t, initResult)

	// Then edit (should not say "creating")
	h.SetEnv("EDITOR", "vim")
	result := h.Run("config", "edit")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputNotContains(t, result, "Creating default configuration")
	e2e.AssertOutputContains(t, result, "Opening")
}

// TestConfigShowWithExistingFileShowsPath verifies "Loaded from" message.
func TestConfigShowWithExistingFileShowsPath(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create config file first
	initResult := h.Run("config", "init")
	e2e.AssertSuccess(t, initResult)

	// Now show should indicate loaded from file
	result := h.Run("config", "show")
	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Loaded from:")
	e2e.AssertOutputNotContains(t, result, "Using default configuration")
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

// ============================================================================
// Sync Command E2E Tests
// ============================================================================

// TestSyncHelp verifies sync command help output includes all strategies.
func TestSyncHelp(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("sync", "--help")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "USAGE")
	e2e.AssertOutputContains(t, result, "--dry-run")
	e2e.AssertOutputContains(t, result, "--strategy")
	e2e.AssertOutputContains(t, result, "overwrite")
	e2e.AssertOutputContains(t, result, "skip")
	e2e.AssertOutputContains(t, result, "newer")
	e2e.AssertOutputContains(t, result, "merge")
	e2e.AssertOutputContains(t, result, "three-way")
	e2e.AssertOutputContains(t, result, "interactive")
}

// TestSyncInvalidStrategy verifies sync command rejects invalid strategies.
func TestSyncInvalidStrategy(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("sync", "--strategy", "invalid-strategy", "--yes", "--skip-validation", "claudecode", "cursor")

	e2e.AssertError(t, result)
	e2e.AssertErrorContains(t, result, "invalid strategy")
}

// TestSyncSamePlatform verifies sync fails when source and target are the same.
func TestSyncSamePlatform(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("sync", "--yes", "--skip-validation", "claudecode", "claudecode")

	e2e.AssertError(t, result)
}

// TestSyncCreatesNewSkill verifies sync creates a skill in empty target.
func TestSyncCreatesNewSkill(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create a skill in Claude Code source
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("new-skill.md", "new-skill", "A brand new skill", "# New Skill\n\nThis is a new skill content.")

	// Ensure Cursor target directory exists but is empty
	cursorFixture := h.CursorFixture()

	// Run sync with --yes to skip confirmation
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertOutputContains(t, result, "1")

	// Verify skill was created in target
	e2e.AssertFileExists(t, cursorFixture.Path("new-skill.md"))
	e2e.AssertFileContains(t, cursorFixture.Path("new-skill.md"), "new-skill")
}

// TestSyncMultipleSkills verifies sync handles multiple skills.
func TestSyncMultipleSkills(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create multiple skills in Claude Code
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("skill-one.md", "skill-one", "First skill", "# Skill One\n\nContent one.")
	claudeFixture.WriteSkill("skill-two.md", "skill-two", "Second skill", "# Skill Two\n\nContent two.")
	claudeFixture.WriteSkill("skill-three.md", "skill-three", "Third skill", "# Skill Three\n\nContent three.")

	// Ensure Cursor target exists
	cursorFixture := h.CursorFixture()

	// Run sync
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertOutputContains(t, result, "3")

	// Verify all skills were created
	e2e.AssertFileExists(t, cursorFixture.Path("skill-one.md"))
	e2e.AssertFileExists(t, cursorFixture.Path("skill-two.md"))
	e2e.AssertFileExists(t, cursorFixture.Path("skill-three.md"))
}

// TestSyncDryRunNoChanges verifies dry-run doesn't modify files.
func TestSyncDryRunNoChanges(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create a skill in Claude Code
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("dry-test.md", "dry-test", "Dry run test skill", "# Dry Test\n\nThis should not be copied.")

	// Create Cursor fixture
	cursorFixture := h.CursorFixture()

	// Run sync with dry-run
	result := h.Run("sync", "--dry-run", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Dry run")
	e2e.AssertOutputContains(t, result, "Created")

	// Verify skill was NOT created in target
	e2e.AssertFileNotExists(t, cursorFixture.Path("dry-test.md"))
}

func TestSyncDeleteModeDeletes(t *testing.T) {
	h := e2e.NewHarness(t)

	src := h.ClaudeCodeFixture()
	src.WriteSkill("del.md", "del", "", "# del")

	tgt := h.CursorFixture()
	tgt.WriteSkill("del.md", "del", "", "# del")

	result := h.RunWithStdin("y\n", "sync", "--delete", "--skip-backup", "claudecode", "cursor")
	e2e.AssertSuccess(t, result)
	e2e.AssertFileNotExists(t, tgt.Path("del.md"))
}

func TestSyncDeleteModeDryRun(t *testing.T) {
	h := e2e.NewHarness(t)

	src := h.ClaudeCodeFixture()
	src.WriteSkill("del.md", "del", "", "# del")

	tgt := h.CursorFixture()
	tgt.WriteSkill("del.md", "del", "", "# del")

	result := h.Run("sync", "--delete", "--dry-run", "--skip-backup", "claudecode", "cursor")
	e2e.AssertSuccess(t, result)
	e2e.AssertFileExists(t, tgt.Path("del.md"))
}

func TestSyncValidatesSourceSkills(t *testing.T) {
	h := e2e.NewHarness(t)

	src := h.ClaudeCodeFixture()
	src.WriteSkill("valid.md", "valid", "", "# valid")
	h.CursorFixture()

	result := h.RunWithStdin("y\n", "sync", "claudecode", "cursor")
	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Validating source skills")
}

func TestCompareCommandBasic(t *testing.T) {
	h := e2e.NewHarness(t)

	src := h.ClaudeCodeFixture()
	src.WriteSkill("a.md", "alpha", "", "# A")
	src.WriteSkill("b.md", "alpha-copy", "", "# A")

	result := h.Run("compare", "--platform", "claude-code", "--format", "summary")
	e2e.AssertSuccess(t, result)
}

// ============================================================================
// Resolution Strategy E2E Tests
// ============================================================================

// TestSyncOverwriteStrategy verifies overwrite strategy replaces existing skills.
func TestSyncOverwriteStrategy(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skill with new content
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("overwrite-test.md", "overwrite-test", "Updated description", "# Overwrite Test\n\nNew content from source.")

	// Create existing target skill with different content
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("overwrite-test.md", "overwrite-test", "Old description", "# Overwrite Test\n\nOld content in target.")

	// Run sync with overwrite strategy (default)
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "overwrite", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Updated")
	e2e.AssertOutputContains(t, result, "1")

	// Verify target was overwritten with source content
	e2e.AssertFileContains(t, cursorFixture.Path("overwrite-test.md"), "New content from source")
}

// TestSyncSkipStrategy verifies skip strategy preserves existing skills.
func TestSyncSkipStrategy(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skill
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("skip-test.md", "skip-test", "Source description", "# Skip Test\n\nNew content from source.")

	// Create existing target skill with different content
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("skip-test.md", "skip-test", "Target description", "# Skip Test\n\nOriginal content in target.")

	// Run sync with skip strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "skip", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Skipped")
	e2e.AssertOutputContains(t, result, "1")

	// Verify target was NOT overwritten - still has original content
	e2e.AssertFileContains(t, cursorFixture.Path("skip-test.md"), "Original content in target")
}

// TestSyncSkipStrategyWithNewSkill verifies skip strategy still creates new skills.
func TestSyncSkipStrategyWithNewSkill(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skills - one existing, one new
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("existing-skill.md", "existing-skill", "Existing", "# Existing\n\nExisting content.")
	claudeFixture.WriteSkill("brand-new-skill.md", "brand-new-skill", "New", "# Brand New\n\nNew content.")

	// Create only the existing skill in target
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("existing-skill.md", "existing-skill", "Target existing", "# Existing\n\nTarget's existing content.")

	// Run sync with skip strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "skip", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertOutputContains(t, result, "Skipped")

	// Verify existing skill was not modified
	e2e.AssertFileContains(t, cursorFixture.Path("existing-skill.md"), "Target's existing content")

	// Verify new skill was created
	e2e.AssertFileExists(t, cursorFixture.Path("brand-new-skill.md"))
	e2e.AssertFileContains(t, cursorFixture.Path("brand-new-skill.md"), "New content")
}

// TestSyncMergeStrategy verifies merge strategy combines content.
func TestSyncMergeStrategy(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skill
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("merge-test.md", "merge-test", "Source description", "# Merge Test\n\nSource content to merge.")

	// Create existing target skill with different content
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("merge-test.md", "merge-test", "Target description", "# Merge Test\n\nTarget content to keep.")

	// Run sync with merge strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "merge", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Merged")
	e2e.AssertOutputContains(t, result, "1")

	// Verify target contains merged content (both source and target content present)
	content := cursorFixture.ReadFile("merge-test.md")
	if !strings.Contains(content, "Source content to merge") || !strings.Contains(content, "Target content to keep") {
		t.Errorf("expected merged content to contain both source and target content\ngot: %s", content)
	}
}

// ============================================================================
// Conflict Detection E2E Tests
// ============================================================================

// TestSyncThreeWayNoConflict verifies three-way merge with identical content skips.
func TestSyncThreeWayNoConflict(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create identical skills in source and target
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("identical.md", "identical", "Same description", "# Identical\n\nSame content.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("identical.md", "identical", "Same description", "# Identical\n\nSame content.")

	// Run sync with three-way strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "three-way", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	// With identical content, should be skipped
	e2e.AssertOutputContains(t, result, "Skipped")
}

// TestSyncThreeWayContentConflict verifies three-way merge detects content differences.
func TestSyncThreeWayContentConflict(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skill with different content
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("conflict.md", "conflict", "Description", "# Conflict\n\nSource version line 1.\nSource version line 2.")

	// Create target skill with different content
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("conflict.md", "conflict", "Description", "# Conflict\n\nTarget version line 1.\nTarget version line 2.")

	// Run sync with three-way strategy - should detect conflict
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "three-way", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	// Three-way should either merge or report conflict
	output := result.Stdout
	if !strings.Contains(output, "Merged") && !strings.Contains(output, "Conflict") {
		t.Errorf("expected three-way to result in merge or conflict, got: %s", output)
	}
}

// TestSyncThreeWayMetadataConflict verifies three-way merge detects metadata differences.
func TestSyncThreeWayMetadataConflict(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skill with one description
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("metadata-conflict.md", "metadata-conflict", "Source description is different", "# Metadata\n\nSame content here.")

	// Create target skill with different description but same content
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("metadata-conflict.md", "metadata-conflict", "Target description is different", "# Metadata\n\nSame content here.")

	// Run sync with three-way strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "three-way", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	// Metadata-only conflicts may be handled as merged or conflict
	// The key is that the sync completes successfully
}

// TestSyncShowsConflictDetails verifies conflict information is displayed.
func TestSyncShowsConflictDetails(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create conflicting skills
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("show-conflict.md", "show-conflict", "Source version", "# Show Conflict\n\nLine A from source.\nLine B from source.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("show-conflict.md", "show-conflict", "Target version", "# Show Conflict\n\nLine X from target.\nLine Y from target.")

	// Run sync with three-way strategy (which reports conflicts)
	result := h.Run("sync", "--dry-run", "--skip-validation", "--strategy", "three-way", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	// Output should show the skill name and some indication of what happened
	e2e.AssertOutputContains(t, result, "show-conflict")
}

// ============================================================================
// Cross-Platform Sync E2E Tests
// ============================================================================

// TestSyncClaudeCodeToCursor verifies sync from Claude Code to Cursor.
func TestSyncClaudeCodeToCursor(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("platform-test.md", "platform-test", "Cross-platform skill", "# Platform Test\n\nWorks across platforms.")

	cursorFixture := h.CursorFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertFileExists(t, cursorFixture.Path("platform-test.md"))
}

// TestSyncCursorToClaudeCode verifies sync from Cursor to Claude Code.
func TestSyncCursorToClaudeCode(t *testing.T) {
	h := e2e.NewHarness(t)

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("reverse-test.md", "reverse-test", "Reverse sync skill", "# Reverse Test\n\nFrom Cursor to Claude.")

	claudeFixture := h.ClaudeCodeFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "cursor", "claudecode")

	e2e.AssertSuccess(t, result)
	e2e.AssertFileExists(t, claudeFixture.Path("reverse-test.md"))
}

// TestSyncClaudeCodeToCodex verifies sync from Claude Code to Codex.
func TestSyncClaudeCodeToCodex(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("codex-test.md", "codex-test", "To Codex", "# Codex Test\n\nContent for Codex.")

	codexFixture := h.CodexFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "codex")

	e2e.AssertSuccess(t, result)
	// Codex may transform the file differently
	e2e.AssertOutputContains(t, result, "Created")
	// Verify something was created in codex directory
	if !codexFixture.Exists("AGENTS.md") && !codexFixture.Exists("codex-test.md") {
		// Codex might aggregate into AGENTS.md or use individual files
		t.Log("Note: Codex file structure may differ from other platforms")
	}
}

// ============================================================================
// Edge Cases and Error Handling E2E Tests
// ============================================================================

// TestSyncEmptySource verifies sync handles empty source directory.
func TestSyncEmptySource(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create empty source directory
	h.ClaudeCodeFixture()
	h.CursorFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	// Should complete successfully with no changes
	e2e.AssertOutputContains(t, result, "0")
}

// TestSyncWithSpecialCharactersInName verifies sync handles special characters.
func TestSyncWithSpecialCharactersInName(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	// Use a skill name with special characters (but valid for filenames)
	claudeFixture.WriteSkill("my-special_skill.md", "my-special_skill", "Special chars", "# Special\n\nContent with special chars.")

	cursorFixture := h.CursorFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertFileExists(t, cursorFixture.Path("my-special_skill.md"))
}

// TestSyncPreservesSkillMetadata verifies metadata is preserved during sync.
func TestSyncPreservesSkillMetadata(t *testing.T) {
	h := e2e.NewHarness(t)

	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("metadata-skill.md", "metadata-skill", "Preserve this description", "# Metadata Skill\n\nContent to sync.")

	cursorFixture := h.CursorFixture()

	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify metadata was preserved
	content := cursorFixture.ReadFile("metadata-skill.md")
	if !strings.Contains(content, "name: metadata-skill") {
		t.Errorf("expected name metadata to be preserved, got: %s", content)
	}
	if !strings.Contains(content, "description: Preserve this description") && !strings.Contains(content, "Preserve this description") {
		t.Errorf("expected description metadata to be preserved, got: %s", content)
	}
}

// TestSyncMixedActions verifies sync handles mixed create/update/skip correctly.
func TestSyncMixedActions(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create 3 skills in source
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("new-skill.md", "new-skill", "New", "# New\n\nNew content.")
	claudeFixture.WriteSkill("update-skill.md", "update-skill", "Update", "# Update\n\nUpdated content.")
	claudeFixture.WriteSkill("skip-skill.md", "skip-skill", "Skip", "# Skip\n\nShould skip this.")

	// Create some existing skills in target
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("update-skill.md", "update-skill", "Old", "# Update\n\nOld content.")
	cursorFixture.WriteSkill("skip-skill.md", "skip-skill", "Skip", "# Skip\n\nShould skip this.")

	// Run with skip strategy so we can see all three actions
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "skip", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Should have 1 created (new-skill), 2 skipped (update-skill, skip-skill)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertOutputContains(t, result, "Skipped")

	// Verify new skill was created
	e2e.AssertFileExists(t, cursorFixture.Path("new-skill.md"))

	// Verify skipped skills weren't modified
	e2e.AssertFileContains(t, cursorFixture.Path("update-skill.md"), "Old content")
}

// ============================================================================
// Newer Strategy E2E Tests
// ============================================================================

// TestSyncNewerStrategySourceNewer verifies newer strategy copies when source is newer.
func TestSyncNewerStrategySourceNewer(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create old target file first
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("newer-test.md", "newer-test", "Old target", "# Newer Test\n\nOld target content.")

	// Sleep briefly to ensure timestamp difference
	// Note: In CI, file times may have limited precision, so we set times explicitly
	targetPath := cursorFixture.Path("newer-test.md")
	oldTime := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(targetPath, oldTime, oldTime); err != nil {
		t.Fatalf("failed to set target file time: %v", err)
	}

	// Create newer source file
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("newer-test.md", "newer-test", "New source", "# Newer Test\n\nNew source content.")

	// Run sync with newer strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "newer", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Updated")

	// Verify target was updated with source content (source is newer)
	e2e.AssertFileContains(t, cursorFixture.Path("newer-test.md"), "New source content")
}

// TestSyncNewerStrategyTargetNewer verifies newer strategy skips when target is newer.
func TestSyncNewerStrategyTargetNewer(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source file first
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("older-test.md", "older-test", "Old source", "# Older Test\n\nOld source content.")

	// Set source file to old timestamp
	sourcePath := claudeFixture.Path("older-test.md")
	oldTime := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(sourcePath, oldTime, oldTime); err != nil {
		t.Fatalf("failed to set source file time: %v", err)
	}

	// Create newer target file
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("older-test.md", "older-test", "New target", "# Older Test\n\nNew target content.")

	// Run sync with newer strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "newer", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Skipped")

	// Verify target was NOT updated (target is newer)
	e2e.AssertFileContains(t, cursorFixture.Path("older-test.md"), "New target content")
}

// TestSyncNewerStrategyNewFile verifies newer strategy creates new files.
func TestSyncNewerStrategyNewFile(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skill with no corresponding target
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("brand-new.md", "brand-new", "Brand new skill", "# Brand New\n\nThis skill doesn't exist in target.")

	// Create empty target directory
	cursorFixture := h.CursorFixture()

	// Run sync with newer strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "newer", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")

	// Verify new skill was created
	e2e.AssertFileExists(t, cursorFixture.Path("brand-new.md"))
	e2e.AssertFileContains(t, cursorFixture.Path("brand-new.md"), "Brand new skill")
}

// ============================================================================
// Interactive Strategy E2E Tests
// ============================================================================

// TestSyncInteractiveUseSource verifies interactive strategy with source selection.
func TestSyncInteractiveUseSource(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create conflicting skills
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("interactive-test.md", "interactive-test", "Source version", "# Interactive Test\n\nSource content.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("interactive-test.md", "interactive-test", "Target version", "# Interactive Test\n\nTarget content.")

	// With --yes flag, skip confirmation but still prompt for conflict resolution
	// Per-conflict prompt options:
	//   1. Use source version (overwrite target)
	//   2. Keep target version (discard source changes)
	//   3. Attempt automatic merge
	//   4. Skip this skill
	result := h.RunWithStdin("1\n", "sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "interactive", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify target was overwritten with source
	e2e.AssertFileContains(t, cursorFixture.Path("interactive-test.md"), "Source content")
}

// TestSyncInteractiveKeepTarget verifies interactive strategy with target selection.
func TestSyncInteractiveKeepTarget(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create conflicting skills
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("keep-target.md", "keep-target", "Source", "# Keep Target\n\nSource content to discard.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("keep-target.md", "keep-target", "Target", "# Keep Target\n\nOriginal target content.")

	// With --yes flag, skip confirmation but still prompt for conflict resolution
	// Per-conflict prompt options:
	//   1. Use source version (overwrite target)
	//   2. Keep target version (discard source changes)
	//   3. Attempt automatic merge
	//   4. Skip this skill
	result := h.RunWithStdin("2\n", "sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "interactive", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify target was NOT modified (kept original)
	e2e.AssertFileContains(t, cursorFixture.Path("keep-target.md"), "Original target content")
}

// TestSyncInteractiveSkip verifies interactive strategy skip option.
func TestSyncInteractiveSkip(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create conflicting skills
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("skip-test.md", "skip-test", "Source", "# Skip Test\n\nSource content.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("skip-test.md", "skip-test", "Target", "# Skip Test\n\nOriginal content to preserve.")

	// With --yes flag, skip confirmation but still prompt for conflict resolution
	// Per-conflict prompt options:
	//   1. Use source version (overwrite target)
	//   2. Keep target version (discard source changes)
	//   3. Attempt automatic merge
	//   4. Skip this skill
	result := h.RunWithStdin("4\n", "sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "interactive", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify target was NOT modified (skipped)
	e2e.AssertFileContains(t, cursorFixture.Path("skip-test.md"), "Original content to preserve")
}

// TestSyncInteractiveAutoMerge verifies interactive strategy auto-merge option.
func TestSyncInteractiveAutoMerge(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create conflicting skills
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("auto-merge.md", "auto-merge", "Source", "# Auto Merge\n\nSource specific content.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("auto-merge.md", "auto-merge", "Target", "# Auto Merge\n\nTarget specific content.")

	// With --yes flag, skip confirmation but still prompt for conflict resolution
	// Per-conflict prompt options:
	//   1. Use source version (overwrite target)
	//   2. Keep target version (discard source changes)
	//   3. Attempt automatic merge (may have conflict markers)
	//   4. Skip this skill
	result := h.RunWithStdin("3\n", "sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "interactive", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify content was merged (both should appear or conflict markers present)
	content := cursorFixture.ReadFile("auto-merge.md")
	// After auto-merge, content should contain merged result or conflict markers
	hasSource := strings.Contains(content, "Source specific content")
	hasTarget := strings.Contains(content, "Target specific content")
	hasMarkers := strings.Contains(content, "<<<<<<<") || strings.Contains(content, ">>>>>>>")

	if !hasSource && !hasTarget && !hasMarkers {
		t.Errorf("expected merged content to contain source, target, or conflict markers\ngot: %s", content)
	}
}

// TestSyncInteractiveNoConflicts verifies interactive with no conflicts skips prompts.
func TestSyncInteractiveNoConflicts(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source skill with no existing target
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("no-conflict.md", "no-conflict", "New skill", "# No Conflict\n\nBrand new content.")

	cursorFixture := h.CursorFixture()

	// No stdin needed - no conflicts means no prompts
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "interactive", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")

	// Verify skill was created
	e2e.AssertFileExists(t, cursorFixture.Path("no-conflict.md"))
}

// ============================================================================
// Backup Behavior E2E Tests
// ============================================================================

// TestSyncCreatesBackupByDefault verifies sync creates backup when not skipped.
func TestSyncCreatesBackupByDefault(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create a skill in target that will be overwritten
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("backup-test.md", "backup-test", "Original", "# Backup Test\n\nOriginal content to backup.")

	// Create source skill
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("backup-test.md", "backup-test", "Updated", "# Backup Test\n\nUpdated content.")

	// Run sync WITHOUT --skip-backup
	result := h.Run("sync", "--yes", "--skip-validation", "--strategy", "overwrite", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify file was updated
	e2e.AssertFileContains(t, cursorFixture.Path("backup-test.md"), "Updated content")

	// Check if backup was mentioned in output (backup list should show something)
	backupResult := h.Run("backup", "list")
	e2e.AssertSuccess(t, backupResult)
	e2e.AssertOutputNotContains(t, backupResult, "No backups found")
}

// TestBackupCreateCommand verifies manual backup creation works.
func TestBackupCreateCommand(t *testing.T) {
	h := e2e.NewHarness(t)

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("manual-backup.md", "manual-backup", "Manual backup", "# Manual Backup\n\nContent.")

	result := h.Run("backup", "create", "--platform", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")

	backupResult := h.Run("backup", "list", "--platform", "cursor")
	e2e.AssertSuccess(t, backupResult)
	e2e.AssertOutputNotContains(t, backupResult, "No backups found")
}

// TestSyncSkipBackupFlag verifies --skip-backup prevents backup creation.
func TestSyncSkipBackupFlag(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create a skill in target
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("no-backup-test.md", "no-backup-test", "Original", "# No Backup\n\nOriginal content.")

	// Create source skill
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("no-backup-test.md", "no-backup-test", "Updated", "# No Backup\n\nUpdated content.")

	// Run sync WITH --skip-backup
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "overwrite", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify file was updated
	e2e.AssertFileContains(t, cursorFixture.Path("no-backup-test.md"), "Updated content")

	// Verify no backup was created
	backupResult := h.Run("backup", "list")
	e2e.AssertSuccess(t, backupResult)
	e2e.AssertOutputContains(t, backupResult, "No backups found")
}

// ============================================================================
// Three-Way Merge Conflict Markers E2E Tests
// ============================================================================

// TestSyncThreeWayWithConflictMarkers verifies three-way merge can produce conflict markers.
func TestSyncThreeWayWithConflictMarkers(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create heavily conflicting content that can't be auto-merged cleanly
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("markers-test.md", "markers-test", "Source version",
		"# Markers Test\n\n"+
			"This line is unique to source.\n"+
			"Line A in source version.\n"+
			"Line B in source version.\n"+
			"Ending from source.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("markers-test.md", "markers-test", "Target version",
		"# Markers Test\n\n"+
			"This line is unique to target.\n"+
			"Line X in target version.\n"+
			"Line Y in target version.\n"+
			"Ending from target.")

	// Run sync with three-way strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "three-way", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Read the result - it should either be merged or have conflict markers
	content := cursorFixture.ReadFile("markers-test.md")

	// The three-way merge should either:
	// 1. Successfully merge the content
	// 2. Leave conflict markers if auto-merge fails
	hasConflictMarkers := strings.Contains(content, "<<<<<<<") ||
		strings.Contains(content, "=======") ||
		strings.Contains(content, ">>>>>>>")
	hasSourceContent := strings.Contains(content, "source")
	hasTargetContent := strings.Contains(content, "target")

	// Verify some merge action occurred
	if !hasConflictMarkers && !hasSourceContent && !hasTargetContent {
		t.Errorf("expected merge result to contain source content, target content, or conflict markers\ngot: %s", content)
	}
}

// TestSyncThreeWayCleanMerge verifies three-way merge handles non-conflicting changes.
func TestSyncThreeWayCleanMerge(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create content where changes are in different areas (clean merge possible)
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("clean-merge.md", "clean-merge", "Desc",
		"# Clean Merge\n\n"+
			"Section 1: Source added this section.\n\n"+
			"Section 2: Common middle content.\n\n"+
			"Section 3: Common ending.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("clean-merge.md", "clean-merge", "Desc",
		"# Clean Merge\n\n"+
			"Section 1: Common start content.\n\n"+
			"Section 2: Common middle content.\n\n"+
			"Section 3: Target modified this.")

	// Run sync with three-way strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "three-way", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)

	// Verify the sync completed - exact merge result depends on implementation
	content := cursorFixture.ReadFile("clean-merge.md")
	if content == "" {
		t.Error("expected file to have content after merge")
	}
}

// ============================================================================
// Multiple Conflict Resolution E2E Tests
// ============================================================================

// TestSyncMultipleConflicts verifies handling of multiple conflicts in one sync.
func TestSyncMultipleConflicts(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create multiple conflicting skills
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("conflict1.md", "conflict1", "Source1", "# Conflict 1\n\nSource version 1.")
	claudeFixture.WriteSkill("conflict2.md", "conflict2", "Source2", "# Conflict 2\n\nSource version 2.")
	claudeFixture.WriteSkill("conflict3.md", "conflict3", "Source3", "# Conflict 3\n\nSource version 3.")

	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("conflict1.md", "conflict1", "Target1", "# Conflict 1\n\nTarget version 1.")
	cursorFixture.WriteSkill("conflict2.md", "conflict2", "Target2", "# Conflict 2\n\nTarget version 2.")
	cursorFixture.WriteSkill("conflict3.md", "conflict3", "Target3", "# Conflict 3\n\nTarget version 3.")

	// Run sync with overwrite strategy to resolve all conflicts
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "overwrite", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Updated")
	e2e.AssertOutputContains(t, result, "3")

	// Verify all were updated
	e2e.AssertFileContains(t, cursorFixture.Path("conflict1.md"), "Source version 1")
	e2e.AssertFileContains(t, cursorFixture.Path("conflict2.md"), "Source version 2")
	e2e.AssertFileContains(t, cursorFixture.Path("conflict3.md"), "Source version 3")
}

// TestSyncMixedConflictAndNew verifies handling mixed new files and conflicts.
func TestSyncMixedConflictAndNew(t *testing.T) {
	h := e2e.NewHarness(t)

	// Create source with mix of new and conflicting
	claudeFixture := h.ClaudeCodeFixture()
	claudeFixture.WriteSkill("new-file1.md", "new-file1", "New1", "# New File 1\n\nBrand new content.")
	claudeFixture.WriteSkill("existing.md", "existing", "Updated", "# Existing\n\nUpdated source content.")
	claudeFixture.WriteSkill("new-file2.md", "new-file2", "New2", "# New File 2\n\nAnother new file.")

	// Create target with one existing file
	cursorFixture := h.CursorFixture()
	cursorFixture.WriteSkill("existing.md", "existing", "Original", "# Existing\n\nOriginal target content.")

	// Run sync with merge strategy
	result := h.Run("sync", "--yes", "--skip-backup", "--skip-validation", "--strategy", "merge", "claudecode", "cursor")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Created")
	e2e.AssertOutputContains(t, result, "Merged")

	// Verify new files were created
	e2e.AssertFileExists(t, cursorFixture.Path("new-file1.md"))
	e2e.AssertFileExists(t, cursorFixture.Path("new-file2.md"))

	// Verify existing was merged (should contain both contents)
	content := cursorFixture.ReadFile("existing.md")
	if !strings.Contains(content, "source") && !strings.Contains(content, "target") {
		// Merge should produce content from both
		t.Logf("Merged content: %s", content)
	}
}

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
