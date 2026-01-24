package model

// Platform represents a supported AI coding platform
type Platform string

const (
	ClaudeCode Platform = "claude-code"
	Cursor     Platform = "cursor"
	Codex      Platform = "codex"
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

// All returns all supported platforms
func AllPlatforms() []Platform {
	return []Platform{ClaudeCode, Cursor, Codex}
}
