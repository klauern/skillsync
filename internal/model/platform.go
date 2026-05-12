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
	// PiAgent is the identifier for the Pi Agent platform.
	PiAgent Platform = "pi-agent"
	// Copilot is the identifier for GitHub Copilot.
	Copilot Platform = "copilot"
	// Gemini is the identifier for the Gemini CLI platform.
	Gemini Platform = "gemini"
	// PiDev is the identifier for the Pi.dev platform.
	PiDev Platform = "pi.dev"
)

// PlatformInfo describes the stable, per-platform metadata used across the codebase.
type PlatformInfo struct {
	Name      string
	Short     string
	ConfigDir string
	DotDir    string
}

var platformInfos = map[Platform]PlatformInfo{
	ClaudeCode: {Name: string(ClaudeCode), Short: "cc", ConfigDir: "claude", DotDir: ".claude"},
	Cursor:     {Name: string(Cursor), Short: "cur", ConfigDir: "cursor", DotDir: ".cursor"},
	Codex:      {Name: string(Codex), Short: "cdx", ConfigDir: "codex", DotDir: ".codex"},
	PiAgent:    {Name: string(PiAgent), Short: "pia", ConfigDir: "agents", DotDir: ".pi"},
	Copilot:    {Name: string(Copilot), Short: "cop", ConfigDir: "github", DotDir: ".github"},
	Gemini:     {Name: string(Gemini), Short: "gem", ConfigDir: "gemini", DotDir: ".gemini"},
	PiDev:      {Name: string(PiDev), Short: "pi", ConfigDir: "pi/agent", DotDir: ".pi/agent"},
}

var allPlatforms = []Platform{ClaudeCode, Cursor, Codex, PiAgent, Copilot, Gemini, PiDev}

var platformAliases = map[string]Platform{
	string(ClaudeCode): ClaudeCode,
	"claudecode":       ClaudeCode,
	"claude":           ClaudeCode,
	string(Cursor):     Cursor,
	string(Codex):      Codex,
	string(PiAgent):    PiAgent,
	"piagent":          PiAgent,
	"pi":               PiAgent,
	string(Copilot):    Copilot,
	"github-copilot":   Copilot,
	"githubcopilot":    Copilot,
	string(Gemini):     Gemini,
	string(PiDev):      PiDev,
	"pidev":            PiDev,
	"pi-dev":           PiDev,
}

// platformInfoFor returns the metadata for a supported platform.
func platformInfoFor(p Platform) (PlatformInfo, bool) {
	info, ok := platformInfos[p]
	return info, ok
}

// PlatformInfoFor returns the metadata for a supported platform.
func PlatformInfoFor(p Platform) (PlatformInfo, bool) {
	return platformInfoFor(p)
}

// IsValid returns true if the platform is recognized.
func (p Platform) IsValid() bool {
	_, ok := platformInfos[p]
	return ok
}

// ConfigDir returns the platform's display directory token (without leading dot).
// Returns "claude" for ClaudeCode, "cursor" for Cursor, "codex" for Codex.
func (p Platform) ConfigDir() string {
	if info, ok := PlatformInfoFor(p); ok {
		return info.ConfigDir
	}
	return string(p)
}

// Short returns an abbreviated platform name for compact display.
func (p Platform) Short() string {
	if info, ok := PlatformInfoFor(p); ok {
		return info.Short
	}
	return string(p)
}

// AllPlatforms returns all supported platforms.
func AllPlatforms() []Platform {
	return append([]Platform(nil), allPlatforms...)
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

	if p, ok := platformAliases[normalized]; ok {
		return p, nil
	}

	valid := make([]string, 0, len(allPlatforms))
	for _, platform := range allPlatforms {
		valid = append(valid, string(platform))
	}
	return "", fmt.Errorf("unknown platform %q (valid: %s)", s, strings.Join(valid, ", "))
}
