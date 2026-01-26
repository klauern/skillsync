package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/sync"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg == nil {
		t.Fatal("Default() returned nil")
	}

	// Check sync defaults
	if cfg.Sync.DefaultStrategy != string(sync.StrategyOverwrite) {
		t.Errorf("expected default strategy %q, got %q", sync.StrategyOverwrite, cfg.Sync.DefaultStrategy)
	}
	if !cfg.Sync.AutoBackup {
		t.Error("expected AutoBackup to be true by default")
	}
	if cfg.Sync.BackupRetentionDays != 30 {
		t.Errorf("expected BackupRetentionDays to be 30, got %d", cfg.Sync.BackupRetentionDays)
	}

	// Check cache defaults
	if !cfg.Cache.Enabled {
		t.Error("expected Cache.Enabled to be true by default")
	}
	if cfg.Cache.TTL != time.Hour {
		t.Errorf("expected Cache.TTL to be 1h, got %v", cfg.Cache.TTL)
	}

	// Check output defaults
	if cfg.Output.Format != "table" {
		t.Errorf("expected Output.Format to be 'table', got %q", cfg.Output.Format)
	}
	if cfg.Output.Color != "auto" {
		t.Errorf("expected Output.Color to be 'auto', got %q", cfg.Output.Color)
	}

	// Check backup defaults
	if !cfg.Backup.Enabled {
		t.Error("expected Backup.Enabled to be true by default")
	}
	if cfg.Backup.MaxBackups != 10 {
		t.Errorf("expected Backup.MaxBackups to be 10, got %d", cfg.Backup.MaxBackups)
	}

	// Check plugins defaults
	if !cfg.Plugins.Enabled {
		t.Error("expected Plugins.Enabled to be true by default")
	}
	if !cfg.Plugins.CachePlugins {
		t.Error("expected Plugins.CachePlugins to be true by default")
	}

	// Check platform defaults
	if !cfg.Platforms.ClaudeCode.BackupEnabled {
		t.Error("expected ClaudeCode.BackupEnabled to be true by default")
	}
	if !cfg.Platforms.Cursor.BackupEnabled {
		t.Error("expected Cursor.BackupEnabled to be true by default")
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a config with custom values
	cfg := Default()
	cfg.Sync.DefaultStrategy = string(sync.StrategyThreeWay)
	cfg.Cache.TTL = 2 * time.Hour
	cfg.Output.Verbose = true
	cfg.Backup.MaxBackups = 20

	// Save to the temporary path
	if err := cfg.SaveToPath(configPath); err != nil {
		t.Fatalf("SaveToPath failed: %v", err)
	}

	// Load from the temporary path
	loaded, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath failed: %v", err)
	}

	// Verify values match
	if loaded.Sync.DefaultStrategy != string(sync.StrategyThreeWay) {
		t.Errorf("expected strategy %q, got %q", sync.StrategyThreeWay, loaded.Sync.DefaultStrategy)
	}
	if loaded.Cache.TTL != 2*time.Hour {
		t.Errorf("expected TTL 2h, got %v", loaded.Cache.TTL)
	}
	if !loaded.Output.Verbose {
		t.Error("expected Verbose to be true")
	}
	if loaded.Backup.MaxBackups != 20 {
		t.Errorf("expected MaxBackups 20, got %d", loaded.Backup.MaxBackups)
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		check    func(*Config) bool
	}{
		{
			name:     "sync strategy",
			envKey:   "SKILLSYNC_SYNC_STRATEGY",
			envValue: "three-way",
			check:    func(c *Config) bool { return c.Sync.DefaultStrategy == "three-way" },
		},
		{
			name:     "sync auto backup",
			envKey:   "SKILLSYNC_SYNC_AUTO_BACKUP",
			envValue: "false",
			check:    func(c *Config) bool { return !c.Sync.AutoBackup },
		},
		{
			name:     "cache enabled",
			envKey:   "SKILLSYNC_CACHE_ENABLED",
			envValue: "false",
			check:    func(c *Config) bool { return !c.Cache.Enabled },
		},
		{
			name:     "cache ttl",
			envKey:   "SKILLSYNC_CACHE_TTL",
			envValue: "30m",
			check:    func(c *Config) bool { return c.Cache.TTL == 30*time.Minute },
		},
		{
			name:     "output format",
			envKey:   "SKILLSYNC_OUTPUT_FORMAT",
			envValue: "json",
			check:    func(c *Config) bool { return c.Output.Format == "json" },
		},
		{
			name:     "output verbose",
			envKey:   "SKILLSYNC_OUTPUT_VERBOSE",
			envValue: "true",
			check:    func(c *Config) bool { return c.Output.Verbose },
		},
		{
			name:     "output color",
			envKey:   "SKILLSYNC_OUTPUT_COLOR",
			envValue: "never",
			check:    func(c *Config) bool { return c.Output.Color == "never" },
		},
		{
			name:     "backup enabled",
			envKey:   "SKILLSYNC_BACKUP_ENABLED",
			envValue: "no",
			check:    func(c *Config) bool { return !c.Backup.Enabled },
		},
		{
			name:     "plugins enabled",
			envKey:   "SKILLSYNC_PLUGINS_ENABLED",
			envValue: "0",
			check:    func(c *Config) bool { return !c.Plugins.Enabled },
		},
		{
			name:     "claude code path",
			envKey:   "SKILLSYNC_CLAUDE_CODE_PATH",
			envValue: "/custom/claude/path",
			check:    func(c *Config) bool { return c.Platforms.ClaudeCode.SkillsPath == "/custom/claude/path" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			t.Setenv(tt.envKey, tt.envValue)

			// Create config and apply environment
			cfg := Default()
			cfg.applyEnvironment()

			// Check if the value was applied
			if !tt.check(cfg) {
				t.Errorf("environment override for %s did not apply correctly", tt.envKey)
			}
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"Yes", true},
		{"YES", true},
		{"on", true},
		{"ON", true},
		{"false", false},
		{"False", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseBool(tt.input)
			if result != tt.expected {
				t.Errorf("parseBool(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetStrategy(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		expected sync.Strategy
	}{
		{"valid overwrite", "overwrite", sync.StrategyOverwrite},
		{"valid skip", "skip", sync.StrategySkip},
		{"valid newer", "newer", sync.StrategyNewer},
		{"valid merge", "merge", sync.StrategyMerge},
		{"valid three-way", "three-way", sync.StrategyThreeWay},
		{"valid interactive", "interactive", sync.StrategyInteractive},
		{"invalid returns default", "invalid-strategy", sync.StrategyOverwrite},
		{"empty returns default", "", sync.StrategyOverwrite},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.Sync.DefaultStrategy = tt.strategy
			result := cfg.GetStrategy()
			if result != tt.expected {
				t.Errorf("GetStrategy() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Set SKILLSYNC_HOME to the temp dir to avoid touching real config
	t.Setenv("SKILLSYNC_HOME", tmpDir)

	// Load should succeed with defaults
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should not fail for non-existent file: %v", err)
	}

	// Should return defaults
	if cfg.Sync.DefaultStrategy != string(sync.StrategyOverwrite) {
		t.Errorf("expected default strategy, got %q", cfg.Sync.DefaultStrategy)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write invalid YAML
	// #nosec G306 - test file permissions are acceptable
	if err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// LoadFromPath should fail
	_, err := LoadFromPath(configPath)
	if err == nil {
		t.Error("LoadFromPath should fail for invalid YAML")
	}
}

func TestPartialConfigMerge(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write a partial config (only sync settings)
	partialConfig := `
sync:
  default_strategy: "skip"
  auto_backup: false
`
	// #nosec G306 - test file permissions are acceptable
	if err := os.WriteFile(configPath, []byte(partialConfig), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Load and verify partial values override defaults
	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath failed: %v", err)
	}

	// Partial overrides should apply
	if cfg.Sync.DefaultStrategy != "skip" {
		t.Errorf("expected strategy 'skip', got %q", cfg.Sync.DefaultStrategy)
	}
	if cfg.Sync.AutoBackup {
		t.Error("expected AutoBackup to be false from partial config")
	}

	// Defaults should still be present for non-specified values
	if !cfg.Cache.Enabled {
		t.Error("expected Cache.Enabled to retain default value true")
	}
	if cfg.Backup.MaxBackups != 10 {
		t.Errorf("expected Backup.MaxBackups to retain default value 10, got %d", cfg.Backup.MaxBackups)
	}
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILLSYNC_HOME", tmpDir)

	// Should not exist initially
	if Exists() {
		t.Error("Exists() should return false for non-existent config")
	}

	// Create config
	cfg := Default()
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Should exist now
	if !Exists() {
		t.Error("Exists() should return true after saving config")
	}
}
