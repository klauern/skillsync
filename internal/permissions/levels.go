// Package permissions provides a permission model for skillsync operations.
package permissions

import "slices"

// Level represents a permission level for operations.
type Level string

const (
	// LevelReadOnly allows only read operations (list, show, compare, etc.)
	LevelReadOnly Level = "read-only"

	// LevelWrite allows read operations plus non-destructive writes (sync, add)
	LevelWrite Level = "write"

	// LevelDestructive allows all operations including destructive ones (delete, overwrite)
	LevelDestructive Level = "destructive"
)

// OperationType represents a category of operation.
type OperationType string

const (
	// OpRead represents read-only operations.
	OpRead OperationType = "read"

	// OpWrite represents write operations (sync, promote, demote).
	OpWrite OperationType = "write"

	// OpDelete represents delete operations.
	OpDelete OperationType = "delete"

	// OpOverwrite represents operations that overwrite existing files.
	OpOverwrite OperationType = "overwrite"

	// OpBackup represents backup operations.
	OpBackup OperationType = "backup"

	// OpBackupDelete represents backup deletion operations.
	OpBackupDelete OperationType = "backup-delete"
)

// RequiresConfirmation returns true if the operation type typically requires user confirmation.
func (ot OperationType) RequiresConfirmation() bool {
	switch ot {
	case OpDelete, OpOverwrite, OpBackupDelete:
		return true
	default:
		return false
	}
}

// IsDestructive returns true if the operation is destructive (data loss possible).
func (ot OperationType) IsDestructive() bool {
	switch ot {
	case OpDelete, OpBackupDelete:
		return true
	case OpOverwrite:
		return true // Can cause data loss if not backed up
	default:
		return false
	}
}

// RequiredLevel returns the minimum permission level required for this operation type.
func (ot OperationType) RequiredLevel() Level {
	switch ot {
	case OpRead:
		return LevelReadOnly
	case OpWrite, OpBackup:
		return LevelWrite
	case OpDelete, OpOverwrite, OpBackupDelete:
		return LevelDestructive
	default:
		return LevelDestructive // Conservative default
	}
}

// ValidLevels returns all valid permission levels.
func ValidLevels() []Level {
	return []Level{LevelReadOnly, LevelWrite, LevelDestructive}
}

// IsValid returns true if the permission level is valid.
func (l Level) IsValid() bool {
	return slices.Contains(ValidLevels(), l)
}

// Allows returns true if this level allows operations of the given level.
func (l Level) Allows(required Level) bool {
	// Map levels to numeric values for comparison
	levelValue := map[Level]int{
		LevelReadOnly:    1,
		LevelWrite:       2,
		LevelDestructive: 3,
	}

	current, okCurrent := levelValue[l]
	req, okReq := levelValue[required]

	if !okCurrent || !okReq {
		return false
	}

	return current >= req
}
