// Package model provides data types for skillsync.
package model

import (
	"fmt"
	"log/slog"
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
	// PiAgent is the identifier for the Pi Agent platform.
	PiAgent Platform = "pi-agent"
	// Copilot is the identifier for GitHub Copilot.
	Copilot Platform = "copilot"
	// Gemini is the identifier for the Gemini CLI platform.
	Gemini Platform = "gemini"
	// PiDev is the identifier for the Pi.dev platform.
	PiDev Platform = "pi.dev"
)

// PlatformInfo holds static metadata for a platform.
type PlatformInfo struct {
	// Short is the abbreviated name (e.g. "cc").
	Short string
	// ConfigDir is the config directory token without leading dot (e.g. "claude").
	ConfigDir string
	// DotDir is the hidden directory name with leading dot (e.g. ".claude").
	DotDir string
	// DisplayName is the human-readable platform name (e.g. "Claude Code").
	DisplayName string
	// ValidExtensions lists allowed file extensions (e.g. [".md", ".txt"]).
	// Empty means any extension is accepted.
	ValidExtensions []string
	// AllowsEmptyExt indicates the platform accepts files without an extension.
	AllowsEmptyExt bool
}

// platformRegistry maps each Platform to its static metadata.
var platformRegistry = map[Platform]PlatformInfo{
	ClaudeCode: {Short: "cc", ConfigDir: "claude", DotDir: ".claude", DisplayName: "Claude Code", ValidExtensions: []string{".md", ".txt"}, AllowsEmptyExt: true},
	Cursor:     {Short: "cur", ConfigDir: "cursor", DotDir: ".cursor", DisplayName: "Cursor", ValidExtensions: []string{".md", ".mdc"}},
	Codex:      {Short: "cdx", ConfigDir: "codex", DotDir: ".codex", DisplayName: "Codex", ValidExtensions: []string{".md", ".toml"}},
	PiAgent:    {Short: "pia", ConfigDir: "agents", DotDir: ".pi", DisplayName: "Pi Agent", ValidExtensions: []string{".md"}},
	Copilot:    {Short: "cop", ConfigDir: "github", DotDir: ".github", DisplayName: "Copilot", ValidExtensions: []string{".md"}},
	Gemini:     {Short: "gem", ConfigDir: "gemini", DotDir: ".gemini", DisplayName: "Gemini", ValidExtensions: []string{".md"}},
	PiDev:      {Short: "pi", ConfigDir: "pi/agent", DotDir: ".pi/agent", DisplayName: "Pi.dev", ValidExtensions: []string{".md"}},
}

// PlatformInfoFor returns the PlatformInfo for p, or (zero, false) if unrecognized.
func PlatformInfoFor(p Platform) (PlatformInfo, bool) {
	info, ok := platformRegistry[p]
	return info, ok
}

// IsValid returns true if the platform is recognized.
func (p Platform) IsValid() bool {
	_, ok := platformRegistry[p]
	return ok
}

// ConfigDir returns the platform's display directory token (without leading dot).
// Returns "claude" for ClaudeCode, "cursor" for Cursor, "codex" for Codex.
func (p Platform) ConfigDir() string {
	if info, ok := platformRegistry[p]; ok {
		return info.ConfigDir
	}
	return string(p)
}

// Short returns an abbreviated platform name for compact display.
func (p Platform) Short() string {
	if info, ok := platformRegistry[p]; ok {
		return info.Short
	}
	return string(p)
}

// AllPlatforms returns all supported platforms.
func AllPlatforms() []Platform {
	return []Platform{ClaudeCode, Cursor, Codex, PiAgent, Copilot, Gemini, PiDev}
}

// AllPlatformNames returns a comma-separated string of all supported platform names.
func AllPlatformNames() string {
	ps := AllPlatforms()
	names := make([]string, len(ps))
	for i, p := range ps {
		names[i] = string(p)
	}
	return strings.Join(names, ", ")
}

// ParsePlatform converts a string to a Platform type.
// Accepts both kebab-case (claude-code) and single-word (claudecode) formats.
// Returns an error if the platform is not recognized.
func ParsePlatform(s string) (Platform, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))

	// Deprecation check: pi-agent aliases should migrate to pi-dev / PiDev.
	switch normalized {
	case "pi-agent", "piagent", "pia":
		slog.Warn(
			"platform name is deprecated; use 'pi-dev' instead",
			"platform", s,
			"replacement", "pi-dev",
		)
		return PiAgent, nil
	}

	// Try exact match first.
	p := Platform(normalized)
	if p.IsValid() {
		return p, nil
	}

	// Try normalized formats.
	switch normalized {
	case "claudecode", "claude":
		return ClaudeCode, nil
	case "cursor":
		return Cursor, nil
	case "codex":
		return Codex, nil
	case "copilot", "github-copilot", "githubcopilot":
		return Copilot, nil
	case "gemini":
		return Gemini, nil
	case "pi.dev", "pidev", "pi-dev", "pi":
		return PiDev, nil
	default:
		valid := make([]string, 0, len(AllPlatforms()))
		for _, platform := range AllPlatforms() {
			valid = append(valid, string(platform))
		}
		return "", fmt.Errorf("unknown platform %q (valid: %s)", s, strings.Join(valid, ", "))
	}
}
