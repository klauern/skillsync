// Package model provides data types for skillsync.
package model

// Platform represents a supported AI coding platform.
type Platform string

const (
	// ClaudeCode is the identifier for the Claude Code platform.
	ClaudeCode Platform = "claude-code"
	// Cursor is the identifier for the Cursor platform.
	Cursor Platform = "cursor"
	// Codex is the identifier for the Codex platform.
	Codex Platform = "codex"
)

// IsValid returns true if the platform is recognized
func (p Platform) IsValid() bool {
	switch p {
	case ClaudeCode, Cursor, Codex:
		return true
	default:
		return false
	}
}

// AllPlatforms returns all supported platforms.
func AllPlatforms() []Platform {
	return []Platform{ClaudeCode, Cursor, Codex}
}
