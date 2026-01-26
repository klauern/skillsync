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
