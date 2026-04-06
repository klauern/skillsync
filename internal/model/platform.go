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
	// Gemini is the identifier for the Gemini CLI platform.
	Gemini Platform = "gemini"
	// PiDev is the identifier for the Pi.dev platform.
	PiDev Platform = "pi.dev"
)

// IsValid returns true if the platform is recognized
func (p Platform) IsValid() bool {
	switch p {
	case ClaudeCode, Cursor, Codex, Gemini, PiDev:
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
	case Gemini:
		return "gemini"
	case PiDev:
		return "pi/agent"
	default:
		return string(p)
	}
}

// Short returns an abbreviated platform name for compact display.
// Returns "cc" for ClaudeCode, "cur" for Cursor, "cdx" for Codex.
func (p Platform) Short() string {
	switch p {
	case ClaudeCode:
		return "cc"
	case Cursor:
		return "cur"
	case Codex:
		return "cdx"
	case Gemini:
		return "gem"
	case PiDev:
		return "pi"
	default:
		return string(p)
	}
}

// AllPlatforms returns all supported platforms.
func AllPlatforms() []Platform {
	return []Platform{ClaudeCode, Cursor, Codex, Gemini, PiDev}
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
	case "gemini":
		return Gemini, nil
	case "pi.dev", "pidev", "pi-dev":
		return PiDev, nil
	default:
		valid := make([]string, 0, len(AllPlatforms()))
		for _, platform := range AllPlatforms() {
			valid = append(valid, string(platform))
		}
		return "", fmt.Errorf("unknown platform %q (valid: %s)", s, strings.Join(valid, ", "))
	}
}
