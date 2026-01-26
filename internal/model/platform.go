// Package model provides data types for skillsync.
package model

import (
	"fmt"
	"strings"
)

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

// ConfigDir returns the platform's config directory name (without leading dot).
// Returns "claude" for ClaudeCode, "cursor" for Cursor, "codex" for Codex.
func (p Platform) ConfigDir() string {
	switch p {
	case ClaudeCode:
		return "claude"
	case Cursor:
		return "cursor"
	case Codex:
		return "codex"
	default:
		return string(p)
	}
}

// AllPlatforms returns all supported platforms.
func AllPlatforms() []Platform {
	return []Platform{ClaudeCode, Cursor, Codex}
}

// ParsePlatform converts a string to a Platform type.
// Accepts both kebab-case (claude-code) and single-word (claudecode) formats.
// Returns an error if the platform is not recognized.
func ParsePlatform(s string) (Platform, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))

	// Try exact match first
	p := Platform(normalized)
	if p.IsValid() {
		return p, nil
	}

	// Try normalized formats
	switch normalized {
	case "claudecode", "claude":
		return ClaudeCode, nil
	case "cursor":
		return Cursor, nil
	case "codex":
		return Codex, nil
	default:
		return "", fmt.Errorf("unknown platform %q (valid: claudecode, cursor, codex)", s)
	}
}
