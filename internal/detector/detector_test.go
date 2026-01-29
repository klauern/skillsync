package detector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/klauern/skillsync/internal/model"
)

func TestDetectPlatform(t *testing.T) {
	t.Run("detects from environment variable", func(t *testing.T) {
		// Create temporary directory
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, "skills")
		require.NoError(t, os.MkdirAll(skillsDir, 0o755))

		// Set environment variable
		t.Setenv("SKILLSYNC_CLAUDE_CODE_PATH", skillsDir)

		result, found := DetectPlatform(model.ClaudeCode)
		assert.True(t, found)
		assert.Equal(t, model.ClaudeCode, result.Platform)
		assert.Equal(t, skillsDir, result.ConfigPath)
		assert.Equal(t, 1.0, result.Confidence)
		assert.Equal(t, "env_var", result.Source)
	})

	t.Run("detects from default user path", func(t *testing.T) {
		// This test depends on actual filesystem state
		// Skip if ~/.claude/skills doesn't exist
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		claudePath := filepath.Join(homeDir, ".claude", "skills")
		if _, err := os.Stat(claudePath); os.IsNotExist(err) {
			t.Skip("~/.claude/skills not found, skipping")
		}

		result, found := DetectPlatform(model.ClaudeCode)
		assert.True(t, found)
		assert.Equal(t, model.ClaudeCode, result.Platform)
		assert.Contains(t, result.ConfigPath, ".claude/skills")
		assert.Equal(t, 0.9, result.Confidence)
		assert.Equal(t, "filesystem", result.Source)
	})

	t.Run("detects from indicator file", func(t *testing.T) {
		// This test depends on actual filesystem state
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		pluginsPath := filepath.Join(homeDir, ".claude", "plugins", "installed_plugins.json")
		if _, err := os.Stat(pluginsPath); os.IsNotExist(err) {
			t.Skip("installed_plugins.json not found, skipping")
		}

		result, found := DetectPlatform(model.ClaudeCode)
		assert.True(t, found)
		assert.Equal(t, model.ClaudeCode, result.Platform)
		assert.InDelta(t, 0.9, result.Confidence, 0.1) // Could be 0.9 or 0.95
	})

	t.Run("not detected when platform absent", func(t *testing.T) {
		// Test with a clean environment (no env vars set)
		t.Setenv("SKILLSYNC_CURSOR_PATH", "")

		// This might fail if Cursor is actually installed
		// Check first if default path exists
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)
		cursorPath := filepath.Join(homeDir, ".cursor", "skills")

		if _, err := os.Stat(cursorPath); err == nil {
			t.Skip("Cursor is installed, cannot test absence")
		}

		result, found := DetectPlatform(model.Cursor)
		assert.False(t, found)
		assert.Equal(t, model.Platform(""), result.Platform)
	})

	t.Run("env var takes precedence over default path", func(t *testing.T) {
		tmpDir := t.TempDir()
		envPath := filepath.Join(tmpDir, "env-skills")
		require.NoError(t, os.MkdirAll(envPath, 0o755))

		t.Setenv("SKILLSYNC_CODEX_PATH", envPath)

		result, found := DetectPlatform(model.Codex)
		assert.True(t, found)
		assert.Equal(t, envPath, result.ConfigPath)
		assert.Equal(t, "env_var", result.Source)
		assert.Equal(t, 1.0, result.Confidence)
	})
}

func TestDetectAll(t *testing.T) {
	t.Run("returns all detected platforms", func(t *testing.T) {
		detected, err := DetectAll()
		require.NoError(t, err)

		// At minimum, we should detect platforms that exist
		// The actual number depends on the system
		for _, d := range detected {
			assert.Contains(t, []model.Platform{
				model.ClaudeCode,
				model.Cursor,
				model.Codex,
			}, d.Platform)
			assert.NotEmpty(t, d.ConfigPath)
			assert.Greater(t, d.Confidence, 0.0)
			assert.LessOrEqual(t, d.Confidence, 1.0)
		}
	})

	t.Run("detects multiple platforms with env vars", func(t *testing.T) {
		tmpDir := t.TempDir()

		claudePath := filepath.Join(tmpDir, "claude")
		cursorPath := filepath.Join(tmpDir, "cursor")
		require.NoError(t, os.MkdirAll(claudePath, 0o755))
		require.NoError(t, os.MkdirAll(cursorPath, 0o755))

		t.Setenv("SKILLSYNC_CLAUDE_CODE_PATH", claudePath)
		t.Setenv("SKILLSYNC_CURSOR_PATH", cursorPath)

		detected, err := DetectAll()
		require.NoError(t, err)

		// Should detect at least the two we configured
		hasClaudeCode := false
		hasCursor := false
		for _, d := range detected {
			if d.Platform == model.ClaudeCode {
				hasClaudeCode = true
				assert.Equal(t, claudePath, d.ConfigPath)
			}
			if d.Platform == model.Cursor {
				hasCursor = true
				assert.Equal(t, cursorPath, d.ConfigPath)
			}
		}
		assert.True(t, hasClaudeCode, "ClaudeCode should be detected")
		assert.True(t, hasCursor, "Cursor should be detected")
	})
}

func TestIsInstalled(t *testing.T) {
	t.Run("returns true when platform detected", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, "skills")
		require.NoError(t, os.MkdirAll(skillsDir, 0o755))

		t.Setenv("SKILLSYNC_CLAUDE_CODE_PATH", skillsDir)

		assert.True(t, IsInstalled(model.ClaudeCode))
	})

	t.Run("returns false when platform not detected", func(t *testing.T) {
		// Use non-existent path
		t.Setenv("SKILLSYNC_CODEX_PATH", "/nonexistent/path/that/does/not/exist")

		// Also check default path doesn't exist
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)
		codexPath := filepath.Join(homeDir, ".codex", "skills")

		if _, err := os.Stat(codexPath); err == nil {
			t.Skip("Codex is installed, cannot test absence")
		}

		assert.False(t, IsInstalled(model.Codex))
	})
}

func TestGetConfigPath(t *testing.T) {
	t.Run("returns path when platform detected", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, "skills")
		require.NoError(t, os.MkdirAll(skillsDir, 0o755))

		t.Setenv("SKILLSYNC_CURSOR_PATH", skillsDir)

		path := GetConfigPath(model.Cursor)
		assert.Equal(t, skillsDir, path)
	})

	t.Run("returns empty string when platform not detected", func(t *testing.T) {
		t.Setenv("SKILLSYNC_CODEX_PATH", "/nonexistent/path")

		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)
		codexPath := filepath.Join(homeDir, ".codex", "skills")

		if _, err := os.Stat(codexPath); err == nil {
			t.Skip("Codex is installed, cannot test absence")
		}

		path := GetConfigPath(model.Codex)
		assert.Empty(t, path)
	})
}

func TestPathExists(t *testing.T) {
	t.Run("returns true for existing path", func(t *testing.T) {
		tmpDir := t.TempDir()
		assert.True(t, pathExists(tmpDir))
	})

	t.Run("returns false for non-existent path", func(t *testing.T) {
		assert.False(t, pathExists("/nonexistent/path/that/does/not/exist"))
	})

	t.Run("returns false for empty path", func(t *testing.T) {
		assert.False(t, pathExists(""))
	})
}

func TestGetEnvPath(t *testing.T) {
	t.Run("returns path for ClaudeCode", func(t *testing.T) {
		t.Setenv("SKILLSYNC_CLAUDE_CODE_PATH", "/test/path")
		path := getEnvPath(model.ClaudeCode)
		assert.Equal(t, "/test/path", path)
	})

	t.Run("returns path for Cursor", func(t *testing.T) {
		t.Setenv("SKILLSYNC_CURSOR_PATH", "/cursor/path")
		path := getEnvPath(model.Cursor)
		assert.Equal(t, "/cursor/path", path)
	})

	t.Run("returns path for Codex", func(t *testing.T) {
		t.Setenv("SKILLSYNC_CODEX_PATH", "/codex/path")
		path := getEnvPath(model.Codex)
		assert.Equal(t, "/codex/path", path)
	})

	t.Run("returns empty string when env var not set", func(t *testing.T) {
		t.Setenv("SKILLSYNC_CLAUDE_CODE_PATH", "")
		path := getEnvPath(model.ClaudeCode)
		assert.Empty(t, path)
	})

	t.Run("expands tilde in path", func(t *testing.T) {
		t.Setenv("SKILLSYNC_CLAUDE_CODE_PATH", "~/test/path")
		path := getEnvPath(model.ClaudeCode)
		assert.NotContains(t, path, "~")
		assert.Contains(t, path, "test/path")
	})
}

func TestGetDefaultUserPath(t *testing.T) {
	t.Run("returns correct path for ClaudeCode", func(t *testing.T) {
		path := getDefaultUserPath(model.ClaudeCode)
		assert.Contains(t, path, ".claude/skills")
	})

	t.Run("returns correct path for Cursor", func(t *testing.T) {
		path := getDefaultUserPath(model.Cursor)
		assert.Contains(t, path, ".cursor/skills")
	})

	t.Run("returns correct path for Codex", func(t *testing.T) {
		path := getDefaultUserPath(model.Codex)
		assert.Contains(t, path, ".codex/skills")
	})
}

func TestGetPlatformIndicator(t *testing.T) {
	t.Run("returns indicator for ClaudeCode", func(t *testing.T) {
		indicator := getPlatformIndicator(model.ClaudeCode)
		assert.Contains(t, indicator, "installed_plugins.json")
	})

	t.Run("returns empty for Cursor", func(t *testing.T) {
		indicator := getPlatformIndicator(model.Cursor)
		assert.Empty(t, indicator)
	})

	t.Run("returns indicator for Codex", func(t *testing.T) {
		indicator := getPlatformIndicator(model.Codex)
		assert.Contains(t, indicator, "config.toml")
	})
}
