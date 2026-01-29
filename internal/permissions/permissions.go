// Package permissions provides permission checking for skillsync operations.
package permissions

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
)

// Checker provides permission checking functionality.
type Checker struct {
	config *Config
}

// NewChecker creates a new permission checker with the given configuration.
func NewChecker(config *Config) *Checker {
	if config == nil {
		config = Default()
	}
	return &Checker{config: config}
}

// CheckOperation checks if an operation is permitted.
// Returns an error if the operation is not allowed.
func (c *Checker) CheckOperation(op OperationType) error {
	// Check if operation is explicitly configured
	if opConfig, exists := c.config.Operations[op]; exists {
		if !opConfig.Enabled {
			return fmt.Errorf("operation %q is disabled by configuration", op)
		}
	}

	// Check if default level allows this operation
	requiredLevel := op.RequiredLevel()
	if !c.config.DefaultLevel.Allows(requiredLevel) {
		return fmt.Errorf("operation %q requires permission level %q, but current level is %q",
			op, requiredLevel, c.config.DefaultLevel)
	}

	return nil
}

// CheckScope checks if writes are allowed to the given scope.
// Returns an error if writes to this scope are not permitted.
func (c *Checker) CheckScope(scope model.SkillScope) error {
	switch scope {
	case model.ScopeUser:
		if !c.config.ScopePermissions.AllowUserScope {
			return fmt.Errorf("writes to user scope are disabled")
		}
	case model.ScopeRepo:
		if !c.config.ScopePermissions.AllowRepoScope {
			return fmt.Errorf("writes to repo scope are disabled")
		}
	case model.ScopeSystem, model.ScopeAdmin:
		if !c.config.ScopePermissions.AllowSystemScope {
			return fmt.Errorf("writes to system/admin scope are disabled (and should remain so)")
		}
	case model.ScopeBuiltin, model.ScopePlugin:
		return fmt.Errorf("writes to %s scope are never allowed", scope)
	default:
		return fmt.Errorf("unknown scope: %s", scope)
	}

	return nil
}

// RequiresConfirmation returns true if the operation requires user confirmation.
func (c *Checker) RequiresConfirmation(op OperationType) bool {
	// Check for operation-specific override
	if opConfig, exists := c.config.Operations[op]; exists {
		if opConfig.RequireConfirmation != nil {
			return *opConfig.RequireConfirmation
		}
	}

	// Use defaults from confirmation config
	switch op {
	case OpDelete:
		return c.config.RequireConfirmation.Delete
	case OpOverwrite:
		return c.config.RequireConfirmation.Overwrite
	case OpBackupDelete:
		return c.config.RequireConfirmation.BackupDelete
	default:
		// Use operation type's default
		return op.RequiresConfirmation()
	}
}

// RequestConfirmation prompts the user for confirmation of an operation.
// Returns true if the user confirms, false otherwise.
func (c *Checker) RequestConfirmation(op OperationType, details string) (bool, error) {
	if !c.RequiresConfirmation(op) {
		return true, nil // No confirmation needed
	}

	// Build prompt
	var prompt string
	switch op {
	case OpDelete:
		prompt = fmt.Sprintf("⚠️  Delete operation: %s\n   This will permanently remove files. Continue?", details)
	case OpOverwrite:
		prompt = fmt.Sprintf("⚠️  Overwrite operation: %s\n   This will replace existing content. Continue?", details)
	case OpBackupDelete:
		prompt = fmt.Sprintf("⚠️  Delete backup: %s\n   This will permanently remove backup files. Continue?", details)
	default:
		prompt = fmt.Sprintf("⚠️  %s operation: %s\n   Continue?", op, details)
	}

	return confirmPrompt(prompt)
}

// confirmPrompt displays a yes/no prompt and returns the user's response.
func confirmPrompt(message string) (bool, error) {
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", message)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	confirmed := response == "y" || response == "yes"

	if !confirmed {
		logging.Info("operation cancelled by user")
	}

	return confirmed, nil
}

// CheckAndConfirm performs both permission check and confirmation if needed.
// This is a convenience method combining CheckOperation and RequestConfirmation.
func (c *Checker) CheckAndConfirm(op OperationType, details string) error {
	// First check if operation is permitted
	if err := c.CheckOperation(op); err != nil {
		return err
	}

	// Then request confirmation if needed
	if c.RequiresConfirmation(op) {
		confirmed, err := c.RequestConfirmation(op, details)
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}
		if !confirmed {
			return fmt.Errorf("operation cancelled by user")
		}
	}

	return nil
}
