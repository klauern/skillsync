// Package config provides configuration management for skillsync.
// It supports YAML configuration files, environment variables, and sensible defaults.
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

	// Cache configures caching behavior
	Cache CacheConfig `yaml:"cache"`

	// Plugins configures plugin discovery settings
	Plugins PluginsConfig `yaml:"plugins"`

	// Output configures display preferences
	Output OutputConfig `yaml:"output"`

	// Backup configures backup behavior
	Backup BackupConfig `yaml:"backup"`

	// Similarity configures similarity matching thresholds
	Similarity SimilarityConfig `yaml:"similarity"`
}

// PlatformsConfig holds platform-specific configuration.
type PlatformsConfig struct {
	ClaudeCode PlatformConfig `yaml:"claude_code"`
	Cursor     PlatformConfig `yaml:"cursor"`
	Codex      PlatformConfig `yaml:"codex"`
}

// PlatformConfig holds configuration for a single platform.
type PlatformConfig struct {
	// SkillsPaths is an ordered list of paths to search for skills (project → user → system)
	// Paths can use ~ for home directory or be relative (resolved from working directory)
	SkillsPaths []string `yaml:"skills_paths,omitempty"`
	// BackupEnabled indicates whether to backup this platform's skills
	BackupEnabled bool `yaml:"backup_enabled"`

	// Deprecated: Use SkillsPaths instead. Kept for backward compatibility during migration.
	SkillsPath string `yaml:"skills_path,omitempty"`
}

// SyncConfig holds synchronization settings.
type SyncConfig struct {
	// DefaultStrategy is the default conflict resolution strategy
	DefaultStrategy string `yaml:"default_strategy"`
	// AutoBackup enables automatic backup before sync
	AutoBackup bool `yaml:"auto_backup"`
	// BackupRetentionDays is how long to keep backups
	BackupRetentionDays int `yaml:"backup_retention_days"`
}

// CacheConfig holds caching settings.
type CacheConfig struct {
	// Enabled enables or disables caching
	Enabled bool `yaml:"enabled"`
	// TTL is the time-to-live for cache entries
	TTL time.Duration `yaml:"ttl"`
	// Location is the cache directory path
	Location string `yaml:"location"`
}

// PluginsConfig holds plugin settings.
type PluginsConfig struct {
	// Enabled enables plugin discovery
	Enabled bool `yaml:"enabled"`
	// RepositoryURL is the default plugin repository
	RepositoryURL string `yaml:"repository_url,omitempty"`
	// CachePlugins enables caching of plugin skills
	CachePlugins bool `yaml:"cache_plugins"`
	// AutoFetch automatically fetches remote plugins
	AutoFetch bool `yaml:"auto_fetch"`
}

// OutputConfig holds display preferences.
type OutputConfig struct {
	// Format is the default output format (table, json, yaml)
	Format string `yaml:"format"`
	// Color controls color output (auto, always, never)
	Color string `yaml:"color"`
	// Verbose enables verbose output
	Verbose bool `yaml:"verbose"`
}

// BackupConfig holds backup settings.
type BackupConfig struct {
	// Enabled enables automatic backups
	Enabled bool `yaml:"enabled"`
	// Location is the backup directory path
	Location string `yaml:"location"`
	// MaxBackups is the maximum number of backups to keep
	MaxBackups int `yaml:"max_backups"`
	// CleanupOnSync enables cleanup during sync
	CleanupOnSync bool `yaml:"cleanup_on_sync"`
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
					".claude/skills",   // Project (relative)
					"~/.claude/skills", // User (absolute)
				},
				BackupEnabled: true,
			},
			Cursor: PlatformConfig{
				SkillsPaths: []string{
					".cursor/skills",   // Project (relative)
					"~/.cursor/skills", // User (absolute)
				},
				BackupEnabled: true,
			},
			Codex: PlatformConfig{
				SkillsPaths: []string{
					".codex", // Project (relative)
				},
				BackupEnabled: true,
			},
		},
		Sync: SyncConfig{
			DefaultStrategy:     string(sync.StrategyOverwrite),
			AutoBackup:          true,
			BackupRetentionDays: 30,
		},
		Cache: CacheConfig{
			Enabled:  true,
			TTL:      time.Hour,
			Location: filepath.Join(util.SkillsyncConfigPath(), "cache"),
		},
		Plugins: PluginsConfig{
			Enabled:      true,
			CachePlugins: true,
			AutoFetch:    false,
		},
		Output: OutputConfig{
			Format:  "table",
			Color:   "auto",
			Verbose: false,
		},
		Backup: BackupConfig{
			Enabled:       true,
			Location:      util.SkillsyncBackupsPath(),
			MaxBackups:    10,
			CleanupOnSync: true,
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
		return nil, err
	}

	// Parse YAML over defaults
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
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
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.applyEnvironment()
	return cfg, nil
}

// Save writes the configuration to the config file.
func (c *Config) Save() error {
	configPath := FilePath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0o750); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	// #nosec G306 - config file should be readable by user
	return os.WriteFile(configPath, data, 0o644)
}

// SaveToPath writes the configuration to a specific path.
func (c *Config) SaveToPath(path string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	// #nosec G306 - config file should be readable by user
	return os.WriteFile(path, data, 0o644)
}

// applyEnvironment applies environment variable overrides.
// Environment variables follow the pattern SKILLSYNC_<SECTION>_<KEY>.
func (c *Config) applyEnvironment() {
	// Sync settings
	if v := os.Getenv("SKILLSYNC_SYNC_STRATEGY"); v != "" {
		c.Sync.DefaultStrategy = v
	}
	if v := os.Getenv("SKILLSYNC_SYNC_AUTO_BACKUP"); v != "" {
		c.Sync.AutoBackup = parseBool(v)
	}

	// Cache settings
	if v := os.Getenv("SKILLSYNC_CACHE_ENABLED"); v != "" {
		c.Cache.Enabled = parseBool(v)
	}
	if v := os.Getenv("SKILLSYNC_CACHE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Cache.TTL = d
		}
	}
	if v := os.Getenv("SKILLSYNC_CACHE_LOCATION"); v != "" {
		c.Cache.Location = v
	}

	// Output settings
	if v := os.Getenv("SKILLSYNC_OUTPUT_FORMAT"); v != "" {
		c.Output.Format = v
	}
	if v := os.Getenv("SKILLSYNC_OUTPUT_COLOR"); v != "" {
		c.Output.Color = v
	}
	if v := os.Getenv("SKILLSYNC_OUTPUT_VERBOSE"); v != "" {
		c.Output.Verbose = parseBool(v)
	}

	// Platform paths - new colon-separated format
	if v := os.Getenv("SKILLSYNC_CLAUDE_CODE_SKILLS_PATHS"); v != "" {
		c.Platforms.ClaudeCode.SkillsPaths = splitPaths(v)
	}
	if v := os.Getenv("SKILLSYNC_CURSOR_SKILLS_PATHS"); v != "" {
		c.Platforms.Cursor.SkillsPaths = splitPaths(v)
	}
	if v := os.Getenv("SKILLSYNC_CODEX_SKILLS_PATHS"); v != "" {
		c.Platforms.Codex.SkillsPaths = splitPaths(v)
	}

	// Deprecated: single path environment variables (for backward compatibility)
	if v := os.Getenv("SKILLSYNC_CLAUDE_CODE_PATH"); v != "" {
		c.Platforms.ClaudeCode.SkillsPath = v
	}
	if v := os.Getenv("SKILLSYNC_CURSOR_PATH"); v != "" {
		c.Platforms.Cursor.SkillsPath = v
	}
	if v := os.Getenv("SKILLSYNC_CODEX_PATH"); v != "" {
		c.Platforms.Codex.SkillsPath = v
	}

	// Backup settings
	if v := os.Getenv("SKILLSYNC_BACKUP_ENABLED"); v != "" {
		c.Backup.Enabled = parseBool(v)
	}
	if v := os.Getenv("SKILLSYNC_BACKUP_LOCATION"); v != "" {
		c.Backup.Location = v
	}

	// Plugins settings
	if v := os.Getenv("SKILLSYNC_PLUGINS_ENABLED"); v != "" {
		c.Plugins.Enabled = parseBool(v)
	}
	if v := os.Getenv("SKILLSYNC_PLUGINS_REPOSITORY"); v != "" {
		c.Plugins.RepositoryURL = v
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
}

// parseBool parses a boolean from common string representations.
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
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

// GetStrategy returns the sync strategy from config, validating it.
func (c *Config) GetStrategy() sync.Strategy {
	strategy := sync.Strategy(c.Sync.DefaultStrategy)
	if strategy.IsValid() {
		return strategy
	}
	return sync.StrategyOverwrite
}

// GetSkillsPaths returns all skills paths for this platform, expanded and in order.
// If SkillsPaths is empty but deprecated SkillsPath is set, falls back to that.
// The baseDir is used for resolving relative paths.
func (pc *PlatformConfig) GetSkillsPaths(baseDir string) []string {
	var paths []string

	// Use new SkillsPaths if available
	if len(pc.SkillsPaths) > 0 {
		paths = util.ExpandPaths(pc.SkillsPaths, baseDir)
	} else if pc.SkillsPath != "" {
		// Fall back to deprecated SkillsPath for backward compatibility
		expanded := util.ExpandPath(pc.SkillsPath, baseDir)
		if expanded != "" {
			paths = []string{expanded}
		}
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
