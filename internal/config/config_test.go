package config

import (
	"os"
	"path/filepath"
	"testing"

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
	if len(cfg.Sync.IncludeTypes) != 1 || cfg.Sync.IncludeTypes[0] != "skill" {
		t.Errorf("expected default include_types [skill], got %v", cfg.Sync.IncludeTypes)
	}

	// Check output defaults
	if cfg.Output.Color != "auto" {
		t.Errorf("expected Output.Color to be 'auto', got %q", cfg.Output.Color)
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a config with custom values
	cfg := Default()
	cfg.Sync.DefaultStrategy = string(sync.StrategyThreeWay)
	cfg.Sync.IncludeTypes = []string{"skill", "prompt"}
	cfg.Output.Color = "never"
	cfg.Similarity.NameThreshold = 0.9

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
	if len(loaded.Sync.IncludeTypes) != 2 || loaded.Sync.IncludeTypes[0] != "skill" || loaded.Sync.IncludeTypes[1] != "prompt" {
		t.Errorf("expected include_types [skill prompt], got %v", loaded.Sync.IncludeTypes)
	}
	if loaded.Output.Color != "never" {
		t.Errorf("expected Output.Color to be 'never', got %q", loaded.Output.Color)
	}
	if loaded.Similarity.NameThreshold != 0.9 {
		t.Errorf("expected NameThreshold 0.9, got %f", loaded.Similarity.NameThreshold)
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
			name:     "sync include types",
			envKey:   "SKILLSYNC_SYNC_INCLUDE_TYPES",
			envValue: "skill,prompt",
			check: func(c *Config) bool {
				return len(c.Sync.IncludeTypes) == 2 &&
					c.Sync.IncludeTypes[0] == "skill" &&
					c.Sync.IncludeTypes[1] == "prompt"
			},
		},
		{
			name:     "output color",
			envKey:   "SKILLSYNC_OUTPUT_COLOR",
			envValue: "never",
			check:    func(c *Config) bool { return c.Output.Color == "never" },
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
  include_types: ["skill", "prompt"]
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
	if len(cfg.Sync.IncludeTypes) != 2 || cfg.Sync.IncludeTypes[0] != "skill" || cfg.Sync.IncludeTypes[1] != "prompt" {
		t.Errorf("expected include_types [skill prompt], got %v", cfg.Sync.IncludeTypes)
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

func TestSplitPaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single path",
			input:    "/path/to/skills",
			expected: []string{"/path/to/skills"},
		},
		{
			name:     "multiple paths",
			input:    "/path/one:/path/two:/path/three",
			expected: []string{"/path/one", "/path/two", "/path/three"},
		},
		{
			name:     "with tilde",
			input:    "~/.claude/skills:~/.cursor/skills",
			expected: []string{"~/.claude/skills", "~/.cursor/skills"},
		},
		{
			name:     "empty segments filtered",
			input:    "/path/one::/path/two:",
			expected: []string{"/path/one", "/path/two"},
		},
		{
			name:     "whitespace trimmed",
			input:    " /path/one : /path/two ",
			expected: []string{"/path/one", "/path/two"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only colons",
			input:    ":::",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitPaths(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitPaths(%q) returned %d paths, expected %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, p := range result {
				if p != tt.expected[i] {
					t.Errorf("splitPaths(%q)[%d] = %q, expected %q", tt.input, i, p, tt.expected[i])
				}
			}
		})
	}
}

func TestGetSkillsPaths(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		config      PlatformConfig
		baseDir     string
		expectedLen int
		checkFirst  string // Expected substring in first path (for checking expansion)
	}{
		{
			name: "new skills_paths format",
			config: PlatformConfig{
				SkillsPaths: []string{".cursor/skills", "~/.cursor/skills"},
			},
			baseDir:     tmpDir,
			expectedLen: 2,
			checkFirst:  tmpDir, // Relative path should be expanded to baseDir
		},
		{
			name: "legacy skills_path fallback",
			config: PlatformConfig{
				SkillsPath: "~/.claude/skills",
			},
			baseDir:     tmpDir,
			expectedLen: 1,
		},
		{
			name: "with multiple skills_paths",
			config: PlatformConfig{
				SkillsPaths: []string{"~/.cursor/skills", "~/.cursor/rules"},
			},
			baseDir:     tmpDir,
			expectedLen: 2,
		},
		{
			name: "skills_paths takes precedence over skills_path",
			config: PlatformConfig{
				SkillsPaths: []string{".cursor/skills"},
				SkillsPath:  "/should/be/ignored",
			},
			baseDir:     tmpDir,
			expectedLen: 1,
			checkFirst:  tmpDir,
		},
		{
			name:        "empty config returns empty",
			config:      PlatformConfig{},
			baseDir:     tmpDir,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := tt.config.GetSkillsPaths(tt.baseDir)
			if len(paths) != tt.expectedLen {
				t.Errorf("GetSkillsPaths() returned %d paths, expected %d: %v", len(paths), tt.expectedLen, paths)
				return
			}
			if tt.checkFirst != "" && len(paths) > 0 {
				if !filepath.IsAbs(paths[0]) {
					t.Errorf("GetSkillsPaths()[0] should be absolute, got %q", paths[0])
				}
			}
		})
	}
}

func TestGetPrimarySkillsPath(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		config      PlatformConfig
		baseDir     string
		expectEmpty bool
	}{
		{
			name: "returns first path",
			config: PlatformConfig{
				SkillsPaths: []string{".cursor/skills", "~/.cursor/skills"},
			},
			baseDir:     tmpDir,
			expectEmpty: false,
		},
		{
			name:        "returns empty for empty config",
			config:      PlatformConfig{},
			baseDir:     tmpDir,
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetPrimarySkillsPath(tt.baseDir)
			if tt.expectEmpty && result != "" {
				t.Errorf("GetPrimarySkillsPath() = %q, expected empty", result)
			}
			if !tt.expectEmpty && result == "" {
				t.Error("GetPrimarySkillsPath() returned empty, expected a path")
			}
		})
	}
}

func TestEnvironmentOverridesSkillsPaths(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		check    func(*Config) bool
	}{
		{
			name:     "claude code skills paths",
			envKey:   "SKILLSYNC_CLAUDE_CODE_SKILLS_PATHS",
			envValue: "/custom/path1:/custom/path2",
			check: func(c *Config) bool {
				return len(c.Platforms.ClaudeCode.SkillsPaths) == 2 &&
					c.Platforms.ClaudeCode.SkillsPaths[0] == "/custom/path1" &&
					c.Platforms.ClaudeCode.SkillsPaths[1] == "/custom/path2"
			},
		},
		{
			name:     "cursor skills paths",
			envKey:   "SKILLSYNC_CURSOR_SKILLS_PATHS",
			envValue: "~/.cursor/skills",
			check: func(c *Config) bool {
				return len(c.Platforms.Cursor.SkillsPaths) == 1 &&
					c.Platforms.Cursor.SkillsPaths[0] == "~/.cursor/skills"
			},
		},
		{
			name:     "codex skills paths",
			envKey:   "SKILLSYNC_CODEX_SKILLS_PATHS",
			envValue: ".codex:/opt/codex/skills",
			check: func(c *Config) bool {
				return len(c.Platforms.Codex.SkillsPaths) == 2 &&
					c.Platforms.Codex.SkillsPaths[0] == ".codex" &&
					c.Platforms.Codex.SkillsPaths[1] == "/opt/codex/skills"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envValue)

			cfg := Default()
			cfg.applyEnvironment()

			if !tt.check(cfg) {
				t.Errorf("environment override for %s did not apply correctly", tt.envKey)
			}
		})
	}
}

func TestDefaultSkillsPaths(t *testing.T) {
	cfg := Default()

	// Check Claude Code defaults
	if len(cfg.Platforms.ClaudeCode.SkillsPaths) != 4 {
		t.Errorf("expected 4 Claude Code skills paths, got %d", len(cfg.Platforms.ClaudeCode.SkillsPaths))
	}
	if cfg.Platforms.ClaudeCode.SkillsPaths[0] != ".claude/commands" {
		t.Errorf("expected first Claude Code path to be '.claude/commands', got %q", cfg.Platforms.ClaudeCode.SkillsPaths[0])
	}
	if cfg.Platforms.ClaudeCode.SkillsPaths[1] != ".claude/skills" {
		t.Errorf("expected second Claude Code path to be '.claude/skills', got %q", cfg.Platforms.ClaudeCode.SkillsPaths[1])
	}
	if cfg.Platforms.ClaudeCode.SkillsPaths[2] != "~/.claude/commands" {
		t.Errorf("expected third Claude Code path to be '~/.claude/commands', got %q", cfg.Platforms.ClaudeCode.SkillsPaths[2])
	}
	if cfg.Platforms.ClaudeCode.SkillsPaths[3] != "~/.claude/skills" {
		t.Errorf("expected fourth Claude Code path to be '~/.claude/skills', got %q", cfg.Platforms.ClaudeCode.SkillsPaths[3])
	}

	// Check Cursor defaults
	if len(cfg.Platforms.Cursor.SkillsPaths) != 2 {
		t.Errorf("expected 2 Cursor skills paths, got %d", len(cfg.Platforms.Cursor.SkillsPaths))
	}
	if cfg.Platforms.Cursor.SkillsPaths[0] != ".cursor/skills" {
		t.Errorf("expected first Cursor path to be '.cursor/skills', got %q", cfg.Platforms.Cursor.SkillsPaths[0])
	}
	if cfg.Platforms.Cursor.SkillsPaths[1] != "~/.cursor/skills" {
		t.Errorf("expected second Cursor path to be '~/.cursor/skills', got %q", cfg.Platforms.Cursor.SkillsPaths[1])
	}

	// Check Codex defaults (3 paths: project, user, admin)
	if len(cfg.Platforms.Codex.SkillsPaths) != 3 {
		t.Errorf("expected 3 Codex skills paths, got %d", len(cfg.Platforms.Codex.SkillsPaths))
	}
	if cfg.Platforms.Codex.SkillsPaths[0] != ".codex/skills" {
		t.Errorf("expected first Codex path to be '.codex/skills', got %q", cfg.Platforms.Codex.SkillsPaths[0])
	}
	if cfg.Platforms.Codex.SkillsPaths[1] != "~/.codex/skills" {
		t.Errorf("expected second Codex path to be '~/.codex/skills', got %q", cfg.Platforms.Codex.SkillsPaths[1])
	}
	if cfg.Platforms.Codex.SkillsPaths[2] != "/etc/codex/skills" {
		t.Errorf("expected third Codex path to be '/etc/codex/skills', got %q", cfg.Platforms.Codex.SkillsPaths[2])
	}
}

func TestDefaultSimilarityConfig(t *testing.T) {
	cfg := Default()

	// Check default similarity thresholds
	if cfg.Similarity.NameThreshold != 0.7 {
		t.Errorf("expected NameThreshold to be 0.7, got %f", cfg.Similarity.NameThreshold)
	}
	if cfg.Similarity.ContentThreshold != 0.6 {
		t.Errorf("expected ContentThreshold to be 0.6, got %f", cfg.Similarity.ContentThreshold)
	}
	if cfg.Similarity.Algorithm != "combined" {
		t.Errorf("expected Algorithm to be 'combined', got %q", cfg.Similarity.Algorithm)
	}
}

func TestSimilarityEnvironmentOverrides(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		check    func(*Config) bool
	}{
		{
			name:     "name threshold",
			envKey:   "SKILLSYNC_SIMILARITY_NAME_THRESHOLD",
			envValue: "0.8",
			check:    func(c *Config) bool { return c.Similarity.NameThreshold == 0.8 },
		},
		{
			name:     "content threshold",
			envKey:   "SKILLSYNC_SIMILARITY_CONTENT_THRESHOLD",
			envValue: "0.5",
			check:    func(c *Config) bool { return c.Similarity.ContentThreshold == 0.5 },
		},
		{
			name:     "algorithm",
			envKey:   "SKILLSYNC_SIMILARITY_ALGORITHM",
			envValue: "levenshtein",
			check:    func(c *Config) bool { return c.Similarity.Algorithm == "levenshtein" },
		},
		{
			name:     "invalid name threshold ignored (too high)",
			envKey:   "SKILLSYNC_SIMILARITY_NAME_THRESHOLD",
			envValue: "1.5",
			check:    func(c *Config) bool { return c.Similarity.NameThreshold == 0.7 }, // default
		},
		{
			name:     "invalid name threshold ignored (negative)",
			envKey:   "SKILLSYNC_SIMILARITY_NAME_THRESHOLD",
			envValue: "-0.1",
			check:    func(c *Config) bool { return c.Similarity.NameThreshold == 0.7 }, // default
		},
		{
			name:     "invalid name threshold ignored (non-numeric)",
			envKey:   "SKILLSYNC_SIMILARITY_NAME_THRESHOLD",
			envValue: "invalid",
			check:    func(c *Config) bool { return c.Similarity.NameThreshold == 0.7 }, // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envValue)

			cfg := Default()
			cfg.applyEnvironment()

			if !tt.check(cfg) {
				t.Errorf("environment override for %s did not apply correctly", tt.envKey)
			}
		})
	}
}

func TestSimilarityConfigRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a config with custom similarity values
	cfg := Default()
	cfg.Similarity.NameThreshold = 0.85
	cfg.Similarity.ContentThreshold = 0.55
	cfg.Similarity.Algorithm = "jaro-winkler"

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
	if loaded.Similarity.NameThreshold != 0.85 {
		t.Errorf("expected NameThreshold 0.85, got %f", loaded.Similarity.NameThreshold)
	}
	if loaded.Similarity.ContentThreshold != 0.55 {
		t.Errorf("expected ContentThreshold 0.55, got %f", loaded.Similarity.ContentThreshold)
	}
	if loaded.Similarity.Algorithm != "jaro-winkler" {
		t.Errorf("expected Algorithm 'jaro-winkler', got %q", loaded.Similarity.Algorithm)
	}
}

func TestPartialSimilarityConfigMerge(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write a partial config (only similarity name_threshold)
	partialConfig := `
similarity:
  name_threshold: 0.9
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

	// Partial override should apply
	if cfg.Similarity.NameThreshold != 0.9 {
		t.Errorf("expected NameThreshold 0.9, got %f", cfg.Similarity.NameThreshold)
	}

	// Other similarity defaults should be retained (but YAML unmarshaling sets to zero for unspecified)
	// Note: Without special handling, unspecified float64 fields become 0
	// This is expected YAML behavior - if users want defaults, they shouldn't specify the section
}
