// Package config provides configuration management for skillsync.
// It supports YAML configuration files, environment variables, and sensible defaults.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/sync"
	"github.com/klauern/skillsync/internal/util"
)

// Config represents the complete skillsync configuration.
type Config struct {
	// Platforms configures paths for each AI coding platform
	Platforms PlatformsConfig `yaml:"platforms"`

	// Sync configures default synchronization behavior
	Sync SyncConfig `yaml:"sync"`

	// Output configures display preferences
	Output OutputConfig `yaml:"output"`

	// Similarity configures similarity matching thresholds
	Similarity SimilarityConfig `yaml:"similarity"`
}

// PlatformsConfig holds platform-specific configuration.
type PlatformsConfig struct {
	ClaudeCode PlatformConfig `yaml:"claude_code"`
	Cursor     PlatformConfig `yaml:"cursor"`
	Codex      PlatformConfig `yaml:"codex"`
	PiAgent    PlatformConfig `yaml:"pi_agent"`
	Copilot    PlatformConfig `yaml:"copilot"`
	Gemini     PlatformConfig `yaml:"gemini"`
	PiDev      PlatformConfig `yaml:"pidev"`
	// Pi is the canonical Pi configuration. PiDev and PiAgent remain readable
	// compatibility fields for callers and legacy files.
	Pi PlatformConfig `yaml:"pi"`
}

// PlatformConfig holds configuration for a single platform.
type PlatformConfig struct {
	// SkillsPaths is an ordered list of paths to search for skills (project → user → system)
	// Paths can use ~ for home directory or be relative (resolved from working directory)
	SkillsPaths []string `yaml:"skills_paths,omitempty"`
}

// SyncConfig holds synchronization settings.
type SyncConfig struct {
	// DefaultStrategy is the default conflict resolution strategy
	DefaultStrategy string `yaml:"default_strategy"`

	// IncludeTypes controls which artifact types sync/delete include by default.
	// Valid values: skill, prompt.
	IncludeTypes []string `yaml:"include_types,omitempty"`
}

// OutputConfig holds display preferences.
type OutputConfig struct {
	// Color controls color output (auto, always, never)
	Color string `yaml:"color"`
}

// SimilarityConfig holds similarity matching settings.
type SimilarityConfig struct {
	// NameThreshold is the minimum score for name similarity (0.0-1.0)
	NameThreshold float64 `yaml:"name_threshold"`
	// ContentThreshold is the minimum score for content similarity (0.0-1.0)
	ContentThreshold float64 `yaml:"content_threshold"`
	// Algorithm is the default similarity algorithm (levenshtein, jaro-winkler, combined)
	Algorithm string `yaml:"algorithm"`
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		Platforms: PlatformsConfig{
			ClaudeCode: PlatformConfig{
				SkillsPaths: []string{
					".claude/skills",     // Project skills (relative)
					"~/.claude/skills",   // User skills (absolute)
					".claude/commands",   // Project slash commands/prompts (discovery)
					"~/.claude/commands", // User slash commands/prompts (discovery)
				},
			},
			Cursor: PlatformConfig{
				SkillsPaths: []string{
					".cursor/skills",     // Project skills (relative)
					"~/.cursor/skills",   // User skills (absolute)
					".agents/skills",     // Shared project compatibility root
					"~/.agents/skills",   // Shared user compatibility root
					".claude/skills",     // Claude compatibility root
					"~/.claude/skills",   // Claude user compatibility root
					".codex/skills",      // Legacy Codex compatibility root
					"~/.codex/skills",    // Legacy Codex user compatibility root
					".cursor/commands",   // Project slash commands (discovery)
					"~/.cursor/commands", // User slash commands (discovery)
				},
			},
			Codex: PlatformConfig{
				SkillsPaths: []string{
					".agents/skills",    // Canonical project root
					"~/.agents/skills",  // Canonical user root
					".codex/skills",     // Legacy project discovery root
					"~/.codex/skills",   // Legacy user discovery root
					"/etc/codex/skills", // Admin discovery root
				},
			},
			PiAgent: PlatformConfig{
				SkillsPaths: []string{
					".pi/skills",
					"~/.pi/agent/skills",
				},
			},
			Copilot: PlatformConfig{
				SkillsPaths: []string{
					".github/skills",
					"~/.copilot/skills",
					".agents/skills",
					".claude/skills",
				},
			},
			Gemini: PlatformConfig{
				SkillsPaths: []string{
					".agents/skills",
					"~/.agents/skills",
					".gemini/skills",
					"~/.gemini/skills",
					".gemini",   // Context and commands discovery
					"~/.gemini", // User context and commands discovery
				},
			},
			PiDev: PlatformConfig{
				SkillsPaths: []string{
					".pi/skills",
					"~/.pi/agent/skills",
				},
			},
			Pi: PlatformConfig{
				SkillsPaths: []string{".pi/skills", "~/.pi/agent/skills", ".agents/skills", "~/.agents/skills"},
			},
		},
		Sync: SyncConfig{
			DefaultStrategy: string(sync.StrategyOverwrite),
			IncludeTypes:    []string{"skill"},
		},
		Output: OutputConfig{
			Color: "auto",
		},
		Similarity: SimilarityConfig{
			NameThreshold:    0.7, // 70% match required for name similarity
			ContentThreshold: 0.6, // 60% match required for content similarity
			Algorithm:        "combined",
		},
	}
}

// configFileName is the name of the config file.
const configFileName = "config.yaml"

// FilePath returns the path to the config file.
func FilePath() string {
	return filepath.Join(util.SkillsyncConfigPath(), configFileName)
}

// Load loads the configuration from file, merging with defaults.
// If the config file doesn't exist, returns default configuration.
func Load() (*Config, error) {
	cfg := Default()

	// Try to load from file
	configPath := FilePath()
	// #nosec G304 - configPath is constructed from trusted config directory
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file, use defaults with environment overrides
			cfg.applyEnvironment()
			return cfg, nil
		}
		return nil, fmt.Errorf("parse config %q: %w", configPath, err)
	}

	// Parse YAML over defaults
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", configPath, err)
	}
	if err := cfg.Platforms.normalizePiFromYAML(data); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", configPath, err)
	}

	// Apply environment variable overrides
	cfg.applyEnvironment()

	return cfg, nil
}

// LoadFromPath loads configuration from a specific path.
func LoadFromPath(path string) (*Config, error) {
	cfg := Default()

	// #nosec G304 - path is provided by caller
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	if err := cfg.Platforms.normalizePiFromYAML(data); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	cfg.applyEnvironment()
	return cfg, nil
}

// Save writes the configuration to the config file.
func (c *Config) Save() error {
	configPath := FilePath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0o750); err != nil {
		return fmt.Errorf("create config directory %q: %w", filepath.Dir(configPath), err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// #nosec G306 - config file should be readable by user
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("write config %q: %w", configPath, err)
	}
	return nil
}

// MarshalYAML emits only the canonical pi key while retaining legacy fields
// in memory for source compatibility.
func (p PlatformsConfig) MarshalYAML() (any, error) {
	return struct {
		ClaudeCode PlatformConfig `yaml:"claude_code"`
		Cursor     PlatformConfig `yaml:"cursor"`
		Codex      PlatformConfig `yaml:"codex"`
		Copilot    PlatformConfig `yaml:"copilot"`
		Gemini     PlatformConfig `yaml:"gemini"`
		Pi         PlatformConfig `yaml:"pi"`
	}{p.ClaudeCode, p.Cursor, p.Codex, p.Copilot, p.Gemini, p.canonicalPi()}, nil
}

func (p *PlatformsConfig) normalizePi() {
	if len(p.Pi.SkillsPaths) == 0 {
		if len(p.PiDev.SkillsPaths) > 0 {
			p.Pi = p.PiDev
		} else if len(p.PiAgent.SkillsPaths) > 0 {
			p.Pi = p.PiAgent
		}
	}
	p.PiDev, p.PiAgent = p.Pi, p.Pi
}

func (p *PlatformsConfig) normalizePiFromYAML(data []byte) error {
	var raw struct {
		Platforms map[string]yaml.Node `yaml:"platforms"`
	}
	if err := yaml.Unmarshal(data, &raw); err == nil {
		if _, ok := raw.Platforms["pi"]; !ok {
			if node, ok := raw.Platforms["pidev"]; ok {
				var legacy PlatformConfig
				if err := node.Decode(&legacy); err != nil {
					return fmt.Errorf("decode platforms.pidev: %w", err)
				}
				p.Pi = legacy
			} else if node, ok := raw.Platforms["pi_agent"]; ok {
				var legacy PlatformConfig
				if err := node.Decode(&legacy); err != nil {
					return fmt.Errorf("decode platforms.pi_agent: %w", err)
				}
				p.Pi = legacy
			}
		}
	}
	p.PiDev, p.PiAgent = p.Pi, p.Pi
	return nil
}

func (p PlatformsConfig) canonicalPi() PlatformConfig {
	if len(p.Pi.SkillsPaths) > 0 {
		return p.Pi
	}
	if len(p.PiDev.SkillsPaths) > 0 {
		return p.PiDev
	}
	return p.PiAgent
}

// SaveToPath writes the configuration to a specific path.
func (c *Config) SaveToPath(path string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create config directory %q: %w", filepath.Dir(path), err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// #nosec G306 - config file should be readable by user
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	return nil
}

// applyEnvironment applies environment variable overrides.
// Environment variables follow the pattern SKILLSYNC_<SECTION>_<KEY>.
func (c *Config) applyEnvironment() {
	// Sync settings
	if v := os.Getenv("SKILLSYNC_SYNC_STRATEGY"); v != "" {
		c.Sync.DefaultStrategy = v
	}
	if v := os.Getenv("SKILLSYNC_SYNC_INCLUDE_TYPES"); v != "" {
		types := strings.Split(v, ",")
		parsed := make([]string, 0, len(types))
		for _, t := range types {
			t = strings.TrimSpace(t)
			if t != "" {
				parsed = append(parsed, t)
			}
		}
		c.Sync.IncludeTypes = parsed
	}

	// Output settings
	if v := os.Getenv("SKILLSYNC_OUTPUT_COLOR"); v != "" {
		c.Output.Color = v
	}

	// Platform paths.
	// Prefer the newer colon-separated *_SKILLS_PATHS variables, but continue
	// to honor the legacy single-path *_PATH aliases used by older tests and
	// sync/validation code paths.
	if v := firstNonEmptyEnv("SKILLSYNC_CLAUDE_CODE_SKILLS_PATHS", "SKILLSYNC_CLAUDE_CODE_PATH"); v != "" {
		c.Platforms.ClaudeCode.SkillsPaths = splitPaths(v)
	}
	if v := firstNonEmptyEnv("SKILLSYNC_CURSOR_SKILLS_PATHS", "SKILLSYNC_CURSOR_PATH"); v != "" {
		c.Platforms.Cursor.SkillsPaths = splitPaths(v)
	}
	if v := firstNonEmptyEnv("SKILLSYNC_CODEX_SKILLS_PATHS", "SKILLSYNC_CODEX_PATH"); v != "" {
		c.Platforms.Codex.SkillsPaths = splitPaths(v)
	}
	if v := firstNonEmptyEnv(
		"SKILLSYNC_PI_SKILLS_PATHS", "SKILLSYNC_PI_PATH",
		"SKILLSYNC_PI_DEV_SKILLS_PATHS",
		"SKILLSYNC_PIDEV_SKILLS_PATHS",
		"SKILLSYNC_PI_DEV_PATH",
		"SKILLSYNC_PIDEV_PATH",
	); v != "" {
		c.Platforms.Pi.SkillsPaths = splitPaths(v)
	} else if v := firstNonEmptyEnv("SKILLSYNC_PI_AGENT_SKILLS_PATHS", "SKILLSYNC_PI_AGENT_PATH"); v != "" {
		c.Platforms.Pi.SkillsPaths = splitPaths(v)
	}
	if v := firstNonEmptyEnv("SKILLSYNC_COPILOT_SKILLS_PATHS", "SKILLSYNC_COPILOT_PATH"); v != "" {
		c.Platforms.Copilot.SkillsPaths = splitPaths(v)
	}
	if v := firstNonEmptyEnv("SKILLSYNC_GEMINI_SKILLS_PATHS", "SKILLSYNC_GEMINI_PATH"); v != "" {
		c.Platforms.Gemini.SkillsPaths = splitPaths(v)
	}

	// Similarity settings
	if v := os.Getenv("SKILLSYNC_SIMILARITY_NAME_THRESHOLD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 && f <= 1 {
			c.Similarity.NameThreshold = f
		}
	}
	if v := os.Getenv("SKILLSYNC_SIMILARITY_CONTENT_THRESHOLD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 && f <= 1 {
			c.Similarity.ContentThreshold = f
		}
	}
	if v := os.Getenv("SKILLSYNC_SIMILARITY_ALGORITHM"); v != "" {
		c.Similarity.Algorithm = v
	}
	c.Platforms.normalizePi()
}

// splitPaths splits a colon-separated path string into individual paths.
// Empty segments are filtered out.
func splitPaths(s string) []string {
	parts := strings.Split(s, ":")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

// GetStrategy returns the sync strategy from config, validating it.
func (c *Config) GetStrategy() sync.Strategy {
	strategy := sync.Strategy(c.Sync.DefaultStrategy)
	if strategy.IsValid() {
		return strategy
	}
	return sync.StrategyOverwrite
}

// GetSkillsPaths returns all skills paths for this platform, expanded and in order.
// The baseDir is used for resolving relative paths.
func (pc *PlatformConfig) GetSkillsPaths(baseDir string) []string {
	var paths []string

	if len(pc.SkillsPaths) > 0 {
		paths = util.ExpandPaths(pc.SkillsPaths, baseDir)
	}

	return paths
}

// GetPrimarySkillsPath returns the first (highest priority) skills path for this platform.
// This is useful when writing new skills - they go to the highest priority location.
// Returns empty string if no paths are configured.
func (pc *PlatformConfig) GetPrimarySkillsPath(baseDir string) string {
	paths := pc.GetSkillsPaths(baseDir)
	if len(paths) > 0 {
		return paths[0]
	}
	return ""
}

// Exists returns true if a config file exists.
func Exists() bool {
	_, err := os.Stat(FilePath())
	return err == nil
}
