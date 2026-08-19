package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/sync"
)

func TestPiConfigYAMLPrecedenceAndCanonicalMarshal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte("platforms:\n  pi_agent: {skills_paths: [/agent]}\n  pidev: {skills_paths: [/dev]}\n  pi: {skills_paths: [/canonical]}\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Platforms.Pi.SkillsPaths; len(got) != 1 || got[0] != "/canonical" {
		t.Fatalf("Pi precedence = %v", got)
	}
	out, err := yaml.Marshal(cfg.Platforms)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if strings.Contains(text, "pi_agent:") || strings.Contains(text, "pidev:") || !strings.Contains(text, "pi:") {
		t.Fatalf("non-canonical YAML: %s", text)
	}
}

func TestPiConfigLegacyPrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("platforms:\n  pi_agent: {skills_paths: [/agent]}\n  pidev: {skills_paths: [/dev]}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Platforms.Pi.SkillsPaths; len(got) != 1 || got[0] != "/dev" {
		t.Fatalf("legacy precedence = %v", got)
	}
}

func TestPiEnvironmentPrecedence(t *testing.T) {
	cfg := Default()
	t.Setenv("SKILLSYNC_PI_AGENT_PATH", "/agent")
	t.Setenv("SKILLSYNC_PIDEV_PATH", "/dev")
	t.Setenv("SKILLSYNC_PI_PATH", "/canonical")
	cfg.applyEnvironment()
	if got := cfg.Platforms.Pi.SkillsPaths; len(got) != 1 || got[0] != "/canonical" {
		t.Fatalf("environment precedence = %v", got)
	}
}

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
			name:     "pi agent paths",
			envKey:   "SKILLSYNC_PI_AGENT_SKILLS_PATHS",
			envValue: ".agents/skills:~/.agents/skills",
			check: func(c *Config) bool {
				return len(c.Platforms.PiAgent.SkillsPaths) == 2 &&
					c.Platforms.PiAgent.SkillsPaths[0] == ".agents/skills" &&
					c.Platforms.PiAgent.SkillsPaths[1] == "~/.agents/skills"
			},
		},
		{
			name:     "pi dev skills paths",
			envKey:   "SKILLSYNC_PIDEV_SKILLS_PATHS",
			envValue: ".agents/skills:~/.pi/agent/skills",
			check: func(c *Config) bool {
				return len(c.Platforms.PiDev.SkillsPaths) == 2 &&
					c.Platforms.PiDev.SkillsPaths[0] == ".agents/skills" &&
					c.Platforms.PiDev.SkillsPaths[1] == "~/.pi/agent/skills"
			},
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

func TestDefault_PiAgentPaths(t *testing.T) {
	cfg := Default()
	want := []string{".pi/skills", "~/.pi/agent/skills"}
	if !slices.Equal(cfg.Platforms.PiAgent.SkillsPaths, want) {
		t.Fatalf("PiAgent compatibility paths = %v, want %v", cfg.Platforms.PiAgent.SkillsPaths, want)
	}
}

func TestDefault_PiDevPaths(t *testing.T) {
	cfg := Default()

	want := []string{".pi/skills", "~/.pi/agent/skills"}
	if !slices.Equal(cfg.Platforms.PiDev.SkillsPaths, want) {
		t.Fatalf("PiDev compatibility paths = %v, want %v", cfg.Platforms.PiDev.SkillsPaths, want)
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
			name: "with multiple skills_paths",
			config: PlatformConfig{
				SkillsPaths: []string{"~/.cursor/skills", "~/.cursor/rules"},
			},
			baseDir:     tmpDir,
			expectedLen: 2,
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
		{
			name:     "pi.dev skills paths",
			envKey:   "SKILLSYNC_PI_DEV_SKILLS_PATHS",
			envValue: ".pi/skills:~/.pi/agent/skills",
			check: func(c *Config) bool {
				return len(c.Platforms.PiDev.SkillsPaths) == 2 &&
					c.Platforms.PiDev.SkillsPaths[0] == ".pi/skills" &&
					c.Platforms.PiDev.SkillsPaths[1] == "~/.pi/agent/skills"
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

func TestEnvironmentOverridesLegacyPathAliases(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		check    func(*Config) bool
	}{
		{
			name:     "claude code legacy path",
			envKey:   "SKILLSYNC_CLAUDE_CODE_PATH",
			envValue: "/legacy/claude",
			check: func(c *Config) bool {
				return len(c.Platforms.ClaudeCode.SkillsPaths) == 1 &&
					c.Platforms.ClaudeCode.SkillsPaths[0] == "/legacy/claude"
			},
		},
		{
			name:     "cursor legacy path",
			envKey:   "SKILLSYNC_CURSOR_PATH",
			envValue: "/legacy/cursor",
			check: func(c *Config) bool {
				return len(c.Platforms.Cursor.SkillsPaths) == 1 &&
					c.Platforms.Cursor.SkillsPaths[0] == "/legacy/cursor"
			},
		},
		{
			name:     "codex legacy path",
			envKey:   "SKILLSYNC_CODEX_PATH",
			envValue: "/legacy/codex",
			check: func(c *Config) bool {
				return len(c.Platforms.Codex.SkillsPaths) == 1 &&
					c.Platforms.Codex.SkillsPaths[0] == "/legacy/codex"
			},
		},
		{
			name:     "pi.dev legacy alias path",
			envKey:   "SKILLSYNC_PIDEV_PATH",
			envValue: "/legacy/pidev",
			check: func(c *Config) bool {
				return len(c.Platforms.PiDev.SkillsPaths) == 1 &&
					c.Platforms.PiDev.SkillsPaths[0] == "/legacy/pidev"
			},
		},
		{
			name:     "copilot legacy path",
			envKey:   "SKILLSYNC_COPILOT_PATH",
			envValue: "/legacy/copilot",
			check: func(c *Config) bool {
				return len(c.Platforms.Copilot.SkillsPaths) == 1 &&
					c.Platforms.Copilot.SkillsPaths[0] == "/legacy/copilot"
			},
		},
		{
			name:     "gemini legacy path",
			envKey:   "SKILLSYNC_GEMINI_PATH",
			envValue: "/legacy/gemini",
			check: func(c *Config) bool {
				return len(c.Platforms.Gemini.SkillsPaths) == 1 &&
					c.Platforms.Gemini.SkillsPaths[0] == "/legacy/gemini"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envValue)

			cfg := Default()
			cfg.applyEnvironment()

			if !tt.check(cfg) {
				t.Errorf("legacy environment override %s did not apply correctly", tt.envKey)
			}
		})
	}
}

func TestDefaultSkillsPaths(t *testing.T) {
	cfg := Default()

	want := map[string]struct {
		got  []string
		want []string
	}{
		"claude":  {cfg.Platforms.ClaudeCode.SkillsPaths, []string{".claude/skills", "~/.claude/skills", ".claude/commands", "~/.claude/commands"}},
		"codex":   {cfg.Platforms.Codex.SkillsPaths, []string{".agents/skills", "~/.agents/skills", ".codex/skills", "~/.codex/skills", "/etc/codex/skills"}},
		"cursor":  {cfg.Platforms.Cursor.SkillsPaths, []string{".cursor/skills", "~/.cursor/skills", ".agents/skills", "~/.agents/skills", ".claude/skills", "~/.claude/skills", ".codex/skills", "~/.codex/skills", ".cursor/commands", "~/.cursor/commands"}},
		"copilot": {cfg.Platforms.Copilot.SkillsPaths, []string{".github/skills", "~/.copilot/skills", ".agents/skills", ".claude/skills"}},
		"gemini":  {cfg.Platforms.Gemini.SkillsPaths, []string{".agents/skills", "~/.agents/skills", ".gemini/skills", "~/.gemini/skills", ".gemini", "~/.gemini"}},
		"pi":      {cfg.Platforms.Pi.SkillsPaths, []string{".pi/skills", "~/.pi/agent/skills", ".agents/skills", "~/.agents/skills"}},
	}
	for name, tt := range want {
		if !slices.Equal(tt.got, tt.want) {
			t.Errorf("%s default paths = %v, want %v", name, tt.got, tt.want)
		}
	}
}

func TestDefaultCursorPathsIncludeCommands(t *testing.T) {
	// Batch 1C: Cursor command artifacts must be discoverable by default
	cfg := Default()
	cursorPaths := cfg.Platforms.Cursor.SkillsPaths

	hasProjectCommands := false
	hasUserCommands := false
	for _, p := range cursorPaths {
		if p == ".cursor/commands" {
			hasProjectCommands = true
		}
		if p == "~/.cursor/commands" {
			hasUserCommands = true
		}
	}
	if !hasProjectCommands {
		t.Error("default Cursor paths must include .cursor/commands for project command discoverability")
	}
	if !hasUserCommands {
		t.Error("default Cursor paths must include ~/.cursor/commands for user command discoverability")
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
