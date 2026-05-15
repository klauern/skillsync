package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/e2e"
)

// TestVersionCommand verifies the version command works correctly.
func TestVersionCommand(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("version")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "skillsync version")
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

// TestExportHelp verifies export command help works.
// NOTE: Full export E2E tests require the CLI to respect SKILLSYNC_*_PATH
// environment variables in all code paths, which is tracked separately.
func TestExportHelp(t *testing.T) {
	h := e2e.NewHarness(t)

	result := h.Run("export", "--help")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Export skills")
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
	e2e.AssertOutputContains(t, result, "output:")
	e2e.AssertOutputContains(t, result, "similarity:")
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

	configData, err := os.ReadFile(filepath.Join(h.HomeDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("failed to read created config file: %v", err)
	}

	configText := string(configData)
	if !strings.Contains(configText, ".claude/commands") {
		t.Fatalf("expected config to include .claude/commands, got:\n%s", configText)
	}
	if !strings.Contains(configText, "~/.claude/commands") {
		t.Fatalf("expected config to include ~/.claude/commands, got:\n%s", configText)
	}
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

	// Use "echo" as a portable no-op editor that exits immediately
	h.SetEnv("EDITOR", "echo")

	result := h.Run("config", "edit")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Opening")
}

// TestConfigEditWithVisual verifies edit works with VISUAL set.
func TestConfigEditWithVisual(t *testing.T) {
	h := e2e.NewHarness(t)

	// Use "echo" as a portable no-op editor that exits immediately
	h.SetEnv("EDITOR", "")
	h.SetEnv("VISUAL", "echo")

	result := h.Run("config", "edit")

	e2e.AssertSuccess(t, result)
	e2e.AssertOutputContains(t, result, "Opening")
}

// TestConfigEditCreatesDefaultIfMissing verifies edit creates config if missing.
func TestConfigEditCreatesDefaultIfMissing(t *testing.T) {
	h := e2e.NewHarness(t)

	h.SetEnv("EDITOR", "echo")

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
	h.SetEnv("EDITOR", "echo")
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
