// Package detector provides platform auto-detection functionality.
// It scans the filesystem and environment variables to determine which
// AI coding assistant platforms are installed and configured.
package detector

import (
	"os"
	"path/filepath"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/util"
)

// DetectedPlatform represents a detected platform with confidence level
type DetectedPlatform struct {
	Platform   model.Platform
	ConfigPath string  // Primary config path that was detected
	Confidence float64 // 0.0-1.0, higher means more confident
	Source     string  // How it was detected: "filesystem", "env_var", etc.
}

// DetectAll scans for all supported platforms and returns detected ones
func DetectAll() ([]DetectedPlatform, error) {
	var detected []DetectedPlatform

	// Check each platform
	for _, platform := range model.AllPlatforms() {
		if result, found := DetectPlatform(platform); found {
			detected = append(detected, result)
		}
	}

	return detected, nil
}

// DetectPlatform checks if a specific platform is installed/configured
func DetectPlatform(platform model.Platform) (DetectedPlatform, bool) {
	// Check environment variable override first (highest confidence)
	if envPath := getEnvPath(platform); envPath != "" {
		if pathExists(envPath) {
			return DetectedPlatform{
				Platform:   platform,
				ConfigPath: envPath,
				Confidence: 1.0,
				Source:     "env_var",
			}, true
		}
	}

	// Check default user-level paths
	userPath := getDefaultUserPath(platform)
	if pathExists(userPath) {
		return DetectedPlatform{
			Platform:   platform,
			ConfigPath: userPath,
			Confidence: 0.9,
			Source:     "filesystem",
		}, true
	}

	// Check for platform-specific indicators
	if indicator := getPlatformIndicator(platform); indicator != "" {
		if pathExists(indicator) {
			return DetectedPlatform{
				Platform:   platform,
				ConfigPath: filepath.Dir(indicator),
				Confidence: 0.95,
				Source:     "indicator_file",
			}, true
		}
	}

	// Check project-local paths (lower confidence)
	if cwd, err := os.Getwd(); err == nil {
		projectPath := filepath.Join(cwd, "."+platform.ConfigDir(), "skills")
		if pathExists(projectPath) {
			return DetectedPlatform{
				Platform:   platform,
				ConfigPath: projectPath,
				Confidence: 0.7,
				Source:     "project_local",
			}, true
		}
	}

	return DetectedPlatform{}, false
}

// IsInstalled is a simpler boolean check for platform presence
func IsInstalled(platform model.Platform) bool {
	_, found := DetectPlatform(platform)
	return found
}

// GetConfigPath returns the detected config path for a platform, or empty string
func GetConfigPath(platform model.Platform) string {
	if result, found := DetectPlatform(platform); found {
		return result.ConfigPath
	}
	return ""
}

// getEnvPath returns environment variable path for platform
func getEnvPath(platform model.Platform) string {
	var envVar string
	switch platform {
	case model.ClaudeCode:
		envVar = "SKILLSYNC_CLAUDE_CODE_PATH"
	case model.Cursor:
		envVar = "SKILLSYNC_CURSOR_PATH"
	case model.Codex:
		envVar = "SKILLSYNC_CODEX_PATH"
	}

	if envVar != "" {
		if path := os.Getenv(envVar); path != "" {
			return util.ExpandPath(path, "")
		}
	}
	return ""
}

// getDefaultUserPath returns the default user-level path for platform
func getDefaultUserPath(platform model.Platform) string {
	switch platform {
	case model.ClaudeCode:
		return util.ClaudeCodeSkillsPath()
	case model.Cursor:
		return util.CursorSkillsPath()
	case model.Codex:
		return util.CodexSkillsPath()
	}
	return ""
}

// getPlatformIndicator returns a path to a platform-specific indicator file
// that strongly suggests the platform is installed
func getPlatformIndicator(platform model.Platform) string {
	switch platform {
	case model.ClaudeCode:
		// Check for installed plugins manifest
		return util.ClaudeInstalledPluginsPath()
	case model.Cursor:
		// Cursor doesn't have a unique indicator beyond the skills directory
		return ""
	case model.Codex:
		// Codex might have a config.toml in the config directory
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, ".codex", "skills", "config.toml")
	}
	return ""
}

// pathExists checks if a path exists on the filesystem
func pathExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
