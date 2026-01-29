package permissions_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/permissions"
)

func TestLevel_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		level permissions.Level
		want  bool
	}{
		{"read-only is valid", permissions.LevelReadOnly, true},
		{"write is valid", permissions.LevelWrite, true},
		{"destructive is valid", permissions.LevelDestructive, true},
		{"invalid level", permissions.Level("invalid"), false},
		{"empty level", permissions.Level(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.IsValid(); got != tt.want {
				t.Errorf("Level.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLevel_Allows(t *testing.T) {
	tests := []struct {
		name     string
		current  permissions.Level
		required permissions.Level
		want     bool
	}{
		{"read-only allows read-only", permissions.LevelReadOnly, permissions.LevelReadOnly, true},
		{"read-only denies write", permissions.LevelReadOnly, permissions.LevelWrite, false},
		{"read-only denies destructive", permissions.LevelReadOnly, permissions.LevelDestructive, false},
		{"write allows read-only", permissions.LevelWrite, permissions.LevelReadOnly, true},
		{"write allows write", permissions.LevelWrite, permissions.LevelWrite, true},
		{"write denies destructive", permissions.LevelWrite, permissions.LevelDestructive, false},
		{"destructive allows read-only", permissions.LevelDestructive, permissions.LevelReadOnly, true},
		{"destructive allows write", permissions.LevelDestructive, permissions.LevelWrite, true},
		{"destructive allows destructive", permissions.LevelDestructive, permissions.LevelDestructive, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.current.Allows(tt.required); got != tt.want {
				t.Errorf("Level.Allows() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOperationType_RequiredLevel(t *testing.T) {
	tests := []struct {
		name string
		op   permissions.OperationType
		want permissions.Level
	}{
		{"read requires read-only", permissions.OpRead, permissions.LevelReadOnly},
		{"write requires write", permissions.OpWrite, permissions.LevelWrite},
		{"backup requires write", permissions.OpBackup, permissions.LevelWrite},
		{"delete requires destructive", permissions.OpDelete, permissions.LevelDestructive},
		{"overwrite requires destructive", permissions.OpOverwrite, permissions.LevelDestructive},
		{"backup-delete requires destructive", permissions.OpBackupDelete, permissions.LevelDestructive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.RequiredLevel(); got != tt.want {
				t.Errorf("OperationType.RequiredLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOperationType_IsDestructive(t *testing.T) {
	tests := []struct {
		name string
		op   permissions.OperationType
		want bool
	}{
		{"read is not destructive", permissions.OpRead, false},
		{"write is not destructive", permissions.OpWrite, false},
		{"backup is not destructive", permissions.OpBackup, false},
		{"delete is destructive", permissions.OpDelete, true},
		{"overwrite is destructive", permissions.OpOverwrite, true},
		{"backup-delete is destructive", permissions.OpBackupDelete, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.IsDestructive(); got != tt.want {
				t.Errorf("OperationType.IsDestructive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOperationType_RequiresConfirmation(t *testing.T) {
	tests := []struct {
		name string
		op   permissions.OperationType
		want bool
	}{
		{"read does not require confirmation", permissions.OpRead, false},
		{"write does not require confirmation", permissions.OpWrite, false},
		{"backup does not require confirmation", permissions.OpBackup, false},
		{"delete requires confirmation", permissions.OpDelete, true},
		{"overwrite requires confirmation", permissions.OpOverwrite, true},
		{"backup-delete requires confirmation", permissions.OpBackupDelete, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.RequiresConfirmation(); got != tt.want {
				t.Errorf("OperationType.RequiresConfirmation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_Default(t *testing.T) {
	cfg := permissions.Default()

	if cfg == nil {
		t.Fatal("Default() returned nil")
	}

	if cfg.DefaultLevel != permissions.LevelDestructive {
		t.Errorf("DefaultLevel = %v, want %v", cfg.DefaultLevel, permissions.LevelDestructive)
	}

	if !cfg.RequireConfirmation.Delete {
		t.Error("RequireConfirmation.Delete should be true by default")
	}

	if !cfg.RequireConfirmation.BackupDelete {
		t.Error("RequireConfirmation.BackupDelete should be true by default")
	}

	if !cfg.ScopePermissions.AllowUserScope {
		t.Error("ScopePermissions.AllowUserScope should be true by default")
	}

	if !cfg.ScopePermissions.AllowRepoScope {
		t.Error("ScopePermissions.AllowRepoScope should be true by default")
	}

	if cfg.ScopePermissions.AllowSystemScope {
		t.Error("ScopePermissions.AllowSystemScope should be false by default")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name         string
		config       *permissions.Config
		wantValid    bool
		wantErrors   int
		wantWarnings int
	}{
		{
			name:       "default config is valid",
			config:     permissions.Default(),
			wantValid:  true,
			wantErrors: 0,
		},
		{
			name: "invalid default level",
			config: &permissions.Config{
				DefaultLevel: permissions.Level("invalid"),
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "system scope enabled warns",
			config: &permissions.Config{
				DefaultLevel: permissions.LevelDestructive,
				ScopePermissions: permissions.ScopePermissionsConfig{
					AllowSystemScope: true,
					AllowUserScope:   true,
					AllowRepoScope:   true,
				},
			},
			wantValid:    true,
			wantWarnings: 1,
		},
		{
			name: "all scopes disabled warns",
			config: &permissions.Config{
				DefaultLevel: permissions.LevelDestructive,
				ScopePermissions: permissions.ScopePermissionsConfig{
					AllowSystemScope: false,
					AllowUserScope:   false,
					AllowRepoScope:   false,
				},
			},
			wantValid:    true,
			wantWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Validate()

			if result.Valid != tt.wantValid {
				t.Errorf("Validate().Valid = %v, want %v", result.Valid, tt.wantValid)
			}

			if len(result.Errors) != tt.wantErrors {
				t.Errorf("Validate() errors count = %d, want %d", len(result.Errors), tt.wantErrors)
			}

			if tt.wantWarnings > 0 && len(result.Warnings) < tt.wantWarnings {
				t.Errorf("Validate() warnings count = %d, want at least %d", len(result.Warnings), tt.wantWarnings)
			}
		})
	}
}

func TestConfig_SaveAndLoad(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "permissions-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	configPath := filepath.Join(tmpDir, "permissions.yaml")

	// Create and save config
	original := permissions.Default()
	original.DefaultLevel = permissions.LevelWrite
	original.RequireConfirmation.Delete = false

	if err := original.SaveToPath(configPath); err != nil {
		t.Fatalf("SaveToPath() error = %v", err)
	}

	// Load config
	loaded, err := permissions.LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	// Verify loaded config matches
	if loaded.DefaultLevel != original.DefaultLevel {
		t.Errorf("loaded DefaultLevel = %v, want %v", loaded.DefaultLevel, original.DefaultLevel)
	}

	if loaded.RequireConfirmation.Delete != original.RequireConfirmation.Delete {
		t.Errorf("loaded RequireConfirmation.Delete = %v, want %v",
			loaded.RequireConfirmation.Delete, original.RequireConfirmation.Delete)
	}
}

func TestConfig_LoadFromPath_NotExists(t *testing.T) {
	// Loading from non-existent path should return defaults
	cfg, err := permissions.LoadFromPath("/nonexistent/path/permissions.yaml")
	if err != nil {
		t.Fatalf("LoadFromPath() unexpected error = %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadFromPath() returned nil config")
	}

	// Should have default values
	if cfg.DefaultLevel != permissions.LevelDestructive {
		t.Errorf("DefaultLevel = %v, want default %v", cfg.DefaultLevel, permissions.LevelDestructive)
	}
}

func TestChecker_CheckOperation(t *testing.T) {
	tests := []struct {
		name      string
		config    *permissions.Config
		operation permissions.OperationType
		wantErr   bool
	}{
		{
			name:      "destructive level allows delete",
			config:    &permissions.Config{DefaultLevel: permissions.LevelDestructive},
			operation: permissions.OpDelete,
			wantErr:   false,
		},
		{
			name:      "write level allows write",
			config:    &permissions.Config{DefaultLevel: permissions.LevelWrite},
			operation: permissions.OpWrite,
			wantErr:   false,
		},
		{
			name:      "read-only level denies write",
			config:    &permissions.Config{DefaultLevel: permissions.LevelReadOnly},
			operation: permissions.OpWrite,
			wantErr:   true,
		},
		{
			name:      "read-only level denies delete",
			config:    &permissions.Config{DefaultLevel: permissions.LevelReadOnly},
			operation: permissions.OpDelete,
			wantErr:   true,
		},
		{
			name: "disabled operation denied",
			config: &permissions.Config{
				DefaultLevel: permissions.LevelDestructive,
				Operations: map[permissions.OperationType]permissions.OperationConfig{
					permissions.OpDelete: {Enabled: false},
				},
			},
			operation: permissions.OpDelete,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := permissions.NewChecker(tt.config)
			err := checker.CheckOperation(tt.operation)

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckOperation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChecker_CheckScope(t *testing.T) {
	tests := []struct {
		name    string
		config  *permissions.Config
		scope   model.SkillScope
		wantErr bool
	}{
		{
			name: "user scope allowed",
			config: &permissions.Config{
				ScopePermissions: permissions.ScopePermissionsConfig{
					AllowUserScope: true,
				},
			},
			scope:   model.ScopeUser,
			wantErr: false,
		},
		{
			name: "repo scope allowed",
			config: &permissions.Config{
				ScopePermissions: permissions.ScopePermissionsConfig{
					AllowRepoScope: true,
				},
			},
			scope:   model.ScopeRepo,
			wantErr: false,
		},
		{
			name: "user scope denied",
			config: &permissions.Config{
				ScopePermissions: permissions.ScopePermissionsConfig{
					AllowUserScope: false,
				},
			},
			scope:   model.ScopeUser,
			wantErr: true,
		},
		{
			name: "system scope denied by default",
			config: &permissions.Config{
				ScopePermissions: permissions.ScopePermissionsConfig{
					AllowSystemScope: false,
				},
			},
			scope:   model.ScopeSystem,
			wantErr: true,
		},
		{
			name: "system scope allowed when enabled",
			config: &permissions.Config{
				ScopePermissions: permissions.ScopePermissionsConfig{
					AllowSystemScope: true,
				},
			},
			scope:   model.ScopeSystem,
			wantErr: false,
		},
		{
			name:    "builtin scope always denied",
			config:  permissions.Default(),
			scope:   model.ScopeBuiltin,
			wantErr: true,
		},
		{
			name:    "plugin scope always denied",
			config:  permissions.Default(),
			scope:   model.ScopePlugin,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := permissions.NewChecker(tt.config)
			err := checker.CheckScope(tt.scope)

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckScope() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChecker_RequiresConfirmation(t *testing.T) {
	tests := []struct {
		name      string
		config    *permissions.Config
		operation permissions.OperationType
		want      bool
	}{
		{
			name:      "delete requires confirmation by default",
			config:    permissions.Default(),
			operation: permissions.OpDelete,
			want:      true,
		},
		{
			name:      "overwrite no confirmation by default",
			config:    permissions.Default(),
			operation: permissions.OpOverwrite,
			want:      false,
		},
		{
			name: "operation override takes precedence",
			config: &permissions.Config{
				RequireConfirmation: permissions.ConfirmationConfig{
					Delete: true,
				},
				Operations: map[permissions.OperationType]permissions.OperationConfig{
					permissions.OpDelete: {
						Enabled:             true,
						RequireConfirmation: boolPtr(false),
					},
				},
			},
			operation: permissions.OpDelete,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := permissions.NewChecker(tt.config)
			got := checker.RequiresConfirmation(tt.operation)

			if got != tt.want {
				t.Errorf("RequiresConfirmation() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}
