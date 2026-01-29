package permissions

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/util"
	"github.com/klauern/skillsync/internal/validation"
)

// Config represents the permissions configuration.
type Config struct {
	// DefaultLevel is the default permission level for operations
	DefaultLevel Level `yaml:"default_level"`

	// Operations maps operation types to specific permission requirements
	Operations map[OperationType]OperationConfig `yaml:"operations,omitempty"`

	// RequireConfirmation controls whether confirmation is required for operations
	RequireConfirmation ConfirmationConfig `yaml:"require_confirmation"`

	// ScopePermissions controls write permissions per scope
	ScopePermissions ScopePermissionsConfig `yaml:"scope_permissions"`
}

// OperationConfig holds permission settings for a specific operation type.
type OperationConfig struct {
	// Enabled controls whether this operation is allowed
	Enabled bool `yaml:"enabled"`

	// RequireConfirmation overrides the default confirmation requirement
	RequireConfirmation *bool `yaml:"require_confirmation,omitempty"`
}

// ConfirmationConfig controls when user confirmation is required.
type ConfirmationConfig struct {
	// Delete controls confirmation for delete operations
	Delete bool `yaml:"delete"`

	// Overwrite controls confirmation for overwrite operations
	Overwrite bool `yaml:"overwrite"`

	// BackupDelete controls confirmation for backup deletion
	BackupDelete bool `yaml:"backup_delete"`

	// PromoteWithRemoval controls confirmation when promoting/demoting with source removal
	PromoteWithRemoval bool `yaml:"promote_with_removal"`
}

// ScopePermissionsConfig controls which scopes are writable.
type ScopePermissionsConfig struct {
	// AllowUserScope allows writes to user scope (~/.{platform}/skills)
	AllowUserScope bool `yaml:"allow_user_scope"`

	// AllowRepoScope allows writes to repo scope (.{platform}/skills)
	AllowRepoScope bool `yaml:"allow_repo_scope"`

	// AllowSystemScope allows writes to system scope (admin/system paths)
	// WARNING: This is dangerous and should typically remain false
	AllowSystemScope bool `yaml:"allow_system_scope"`
}

// Default returns the default permissions configuration.
func Default() *Config {
	return &Config{
		DefaultLevel: LevelDestructive, // Allow all operations by default (backward compatible)
		Operations:   make(map[OperationType]OperationConfig),
		RequireConfirmation: ConfirmationConfig{
			Delete:             true,
			Overwrite:          false, // Auto-backup handles safety
			BackupDelete:       true,
			PromoteWithRemoval: true,
		},
		ScopePermissions: ScopePermissionsConfig{
			AllowUserScope:   true,
			AllowRepoScope:   true,
			AllowSystemScope: false, // Never allow system writes by default
		},
	}
}

// Load loads the permissions configuration from the default location.
// If no config file exists, returns the default configuration.
func Load() (*Config, error) {
	configPath := filepath.Join(util.SkillsyncConfigPath(), "permissions.yaml")
	return LoadFromPath(configPath)
}

// LoadFromPath loads the permissions configuration from the specified path.
// If the file doesn't exist, returns the default configuration.
func LoadFromPath(path string) (*Config, error) {
	cfg := Default()

	// If file doesn't exist, return defaults
	data, err := os.ReadFile(path) // #nosec G304 - path is from config directory
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read permissions config: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse permissions config: %w", err)
	}

	// Validate
	if result := cfg.Validate(); !result.Valid {
		return nil, fmt.Errorf("invalid permissions config: %w", result.Error())
	}

	return cfg, nil
}

// Save saves the permissions configuration to the default location.
func (c *Config) Save() error {
	configPath := filepath.Join(util.SkillsyncConfigPath(), "permissions.yaml")
	return c.SaveToPath(configPath)
}

// SaveToPath saves the permissions configuration to the specified path.
func (c *Config) SaveToPath(path string) error {
	// Validate before saving
	if result := c.Validate(); !result.Valid {
		return fmt.Errorf("cannot save invalid config: %w", result.Error())
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0o644); err != nil { // #nosec G306 - config file should be readable
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Validate validates the permissions configuration.
func (c *Config) Validate() validation.Result {
	result := validation.Result{Valid: true}

	// Validate default level
	if !c.DefaultLevel.IsValid() {
		result.AddError(&validation.Error{
			Field:   "default_level",
			Message: fmt.Sprintf("invalid permission level: %q", c.DefaultLevel),
		})
	}

	// Validate operation configs
	for opType, opConfig := range c.Operations {
		// Check if operation type is valid
		switch opType {
		case OpRead, OpWrite, OpDelete, OpOverwrite, OpBackup, OpBackupDelete:
			// Valid operation type
		default:
			result.AddWarning(fmt.Sprintf("unknown operation type: %q", opType))
		}

		// If operation is disabled but has settings, warn
		if !opConfig.Enabled && opConfig.RequireConfirmation != nil {
			result.AddWarning(fmt.Sprintf("operation %q is disabled but has confirmation settings", opType))
		}
	}

	// Warn if system scope is enabled
	if c.ScopePermissions.AllowSystemScope {
		result.AddWarning("system scope writes are enabled - this can be dangerous")
	}

	// Warn if all scopes are disabled
	if !c.ScopePermissions.AllowUserScope && !c.ScopePermissions.AllowRepoScope {
		result.AddWarning("all writable scopes are disabled - sync and write operations will fail")
	}

	return result
}
