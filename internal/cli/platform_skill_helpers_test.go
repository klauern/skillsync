package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/model"
)

func TestPlatformSkillsPaths_PiConfiguredPathsOverrideDefaults(t *testing.T) {
	workingDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("failed to change to working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	configuredPath := filepath.Join(workingDir, "custom", "skills")
	configPath := filepath.Join(workingDir, "config.yaml")
	configData := []byte("platforms:\n  pi:\n    skills_paths:\n      - " + configuredPath + "\n")
	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	cfg, err := config.LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	paths, _, err := platformSkillsPaths(cfg, model.Pi)
	if err != nil {
		t.Fatalf("platformSkillsPaths() error = %v", err)
	}
	if len(paths) != 1 || paths[0].Path != configuredPath {
		t.Fatalf("platformSkillsPaths() = %v, want only configured path %q", paths, configuredPath)
	}
}

func TestPlatformSkillsPaths_PiEnvironmentOverrideTakesPrecedence(t *testing.T) {
	workingDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("failed to change to working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	first := filepath.Join(workingDir, "first")
	second := filepath.Join(workingDir, "second")
	t.Setenv("SKILLSYNC_PI_SKILLS_PATHS", first+":"+second)
	configPath := filepath.Join(workingDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	cfg, err := config.LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	paths, _, err := platformSkillsPaths(cfg, model.Pi)
	if err != nil {
		t.Fatalf("platformSkillsPaths() error = %v", err)
	}
	if len(paths) != 2 || paths[0].Path != first || paths[1].Path != second {
		t.Fatalf("platformSkillsPaths() = %v, want environment paths in order", paths)
	}
}

func TestPlatformSkillsPaths_PiEmptyConfiguredPathsDisableDefaults(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("platforms:\n  pi:\n    skills_paths: []\n"), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	cfg, err := config.LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	paths, _, err := platformSkillsPaths(cfg, model.Pi)
	if err != nil {
		t.Fatalf("platformSkillsPaths() error = %v", err)
	}
	if len(paths) != 0 {
		t.Fatalf("platformSkillsPaths() = %v, want no paths", paths)
	}
}

func TestDiscoverSkillsAcrossPlatforms(t *testing.T) {
	original := parsePlatformSkillsFn
	t.Cleanup(func() {
		parsePlatformSkillsFn = original
	})

	parsePlatformSkillsFn = func(platform model.Platform) ([]model.Skill, error) {
		switch platform {
		case model.ClaudeCode:
			return []model.Skill{{Name: "commit", Platform: platform}}, nil
		case model.Cursor:
			return []model.Skill{{Name: "review", Platform: platform}}, nil
		default:
			return nil, errors.New("unexpected platform")
		}
	}

	skills := discoverSkillsAcrossPlatforms([]model.Platform{model.ClaudeCode, model.Cursor})
	if len(skills) != 2 {
		t.Fatalf("discoverSkillsAcrossPlatforms() returned %d skills, want 2", len(skills))
	}
	if skills[0].Name != "commit" || skills[1].Name != "review" {
		t.Fatalf("discoverSkillsAcrossPlatforms() returned skills %q and %q, want commit and review", skills[0].Name, skills[1].Name)
	}
}

func TestDiscoverSkillsAcrossPlatformsForTUIIncludesPlugins(t *testing.T) {
	originalParse := parsePlatformSkillsFn
	originalPlugins := discoverPluginSkillsFn
	t.Cleanup(func() {
		parsePlatformSkillsFn = originalParse
		discoverPluginSkillsFn = originalPlugins
	})

	parsePlatformSkillsFn = func(platform model.Platform) ([]model.Skill, error) {
		return []model.Skill{{Name: string(platform), Platform: platform}}, nil
	}
	discoverPluginSkillsFn = func(_ string, _ bool) ([]model.Skill, error) {
		return []model.Skill{{Name: "plugin-skill", Platform: model.ClaudeCode}}, nil
	}

	skills := discoverSkillsAcrossPlatformsForTUI([]model.Platform{model.ClaudeCode, model.Cursor}, true)
	if len(skills) != 3 {
		t.Fatalf("discoverSkillsAcrossPlatformsForTUI() returned %d skills, want 3", len(skills))
	}
	if skills[2].Name != "plugin-skill" {
		t.Fatalf("discoverSkillsAcrossPlatformsForTUI() last skill = %q, want plugin-skill", skills[2].Name)
	}
}

func TestDiscoverSkillsAcrossPlatformsForTUISkipsPluginLookupWhenDisabled(t *testing.T) {
	originalParse := parsePlatformSkillsFn
	originalPlugins := discoverPluginSkillsFn
	t.Cleanup(func() {
		parsePlatformSkillsFn = originalParse
		discoverPluginSkillsFn = originalPlugins
	})

	parsePlatformSkillsFn = func(platform model.Platform) ([]model.Skill, error) {
		return []model.Skill{{Name: string(platform), Platform: platform}}, nil
	}
	discoverPluginSkillsFn = func(_ string, _ bool) ([]model.Skill, error) {
		t.Fatal("discoverPluginSkillsFn should not be called when plugins are disabled")
		return nil, nil
	}

	skills := discoverSkillsAcrossPlatformsForTUI([]model.Platform{model.ClaudeCode, model.Cursor}, false)
	if len(skills) != 2 {
		t.Fatalf("discoverSkillsAcrossPlatformsForTUI() returned %d skills, want 2", len(skills))
	}
}

func TestDiscoverSkillsAcrossPlatformsForTUIContinuesWhenPluginLookupFails(t *testing.T) {
	originalParse := parsePlatformSkillsFn
	originalPlugins := discoverPluginSkillsFn
	t.Cleanup(func() {
		parsePlatformSkillsFn = originalParse
		discoverPluginSkillsFn = originalPlugins
	})

	parsePlatformSkillsFn = func(platform model.Platform) ([]model.Skill, error) {
		return []model.Skill{{Name: string(platform), Platform: platform}}, nil
	}
	discoverPluginSkillsFn = func(_ string, _ bool) ([]model.Skill, error) {
		return nil, errors.New("boom")
	}

	skills := discoverSkillsAcrossPlatformsForTUI([]model.Platform{model.ClaudeCode}, true)
	if len(skills) != 1 {
		t.Fatalf("discoverSkillsAcrossPlatformsForTUI() returned %d skills, want 1", len(skills))
	}
	if skills[0].Name != string(model.ClaudeCode) {
		t.Fatalf("discoverSkillsAcrossPlatformsForTUI() first skill = %q, want %q", skills[0].Name, model.ClaudeCode)
	}
}

func TestDiscoverSkillsAcrossPlatformsContinuesAfterParseError(t *testing.T) {
	original := parsePlatformSkillsFn
	t.Cleanup(func() {
		parsePlatformSkillsFn = original
	})

	parsePlatformSkillsFn = func(platform model.Platform) ([]model.Skill, error) {
		if platform == model.ClaudeCode {
			return nil, errors.New("boom")
		}
		return []model.Skill{{Name: "review", Platform: platform}}, nil
	}

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stderr = w

	skills := discoverSkillsAcrossPlatforms([]model.Platform{model.ClaudeCode, model.Cursor})

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}
	os.Stderr = oldStderr

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read stderr output: %v", err)
	}

	if len(skills) != 1 || skills[0].Name != "review" {
		t.Fatalf("discoverSkillsAcrossPlatforms() returned %+v, want only cursor skill", skills)
	}
	if !strings.Contains(buf.String(), "Warning: failed parse claude-code: boom") {
		t.Fatalf("stderr = %q, want parse warning", buf.String())
	}
}

func TestDiscoverSkillsForPlatformUsesAllPlatformsWhenEmpty(t *testing.T) {
	original := parsePlatformSkillsFn
	t.Cleanup(func() { parsePlatformSkillsFn = original })

	parsePlatformSkillsFn = func(platform model.Platform) ([]model.Skill, error) {
		return []model.Skill{{Name: string(platform), Platform: platform}}, nil
	}

	skills := discoverSkillsForPlatform("")
	if len(skills) != len(model.AllPlatforms()) {
		t.Fatalf("discoverSkillsForPlatform(\"\") returned %d skills, want %d", len(skills), len(model.AllPlatforms()))
	}
}

func TestDiscoverSkillsForPlatformNameFiltersSinglePlatform(t *testing.T) {
	original := parsePlatformSkillsFn
	t.Cleanup(func() { parsePlatformSkillsFn = original })

	parsePlatformSkillsFn = func(platform model.Platform) ([]model.Skill, error) {
		return []model.Skill{{Name: string(platform), Platform: platform}}, nil
	}

	skills, err := discoverSkillsForPlatformName("cursor")
	if err != nil {
		t.Fatalf("discoverSkillsForPlatformName() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("discoverSkillsForPlatformName() returned %d skills, want 1", len(skills))
	}
	if skills[0].Platform != model.Cursor {
		t.Fatalf("discoverSkillsForPlatformName() returned platform %q, want %q", skills[0].Platform, model.Cursor)
	}
}

func TestDiscoverSkillsForPlatformNameRejectsInvalidPlatform(t *testing.T) {
	_, err := discoverSkillsForPlatformName("not-a-platform")
	if err == nil {
		t.Fatal("discoverSkillsForPlatformName() error = nil, want invalid platform error")
	}
	if !strings.Contains(err.Error(), "invalid platform") {
		t.Fatalf("discoverSkillsForPlatformName() error = %q, want invalid platform", err)
	}
}

func TestDiscoverAllSkillsForTUIIncludesPlugins(t *testing.T) {
	originalParse := parsePlatformSkillsFn
	originalPlugins := discoverPluginSkillsFn
	t.Cleanup(func() {
		parsePlatformSkillsFn = originalParse
		discoverPluginSkillsFn = originalPlugins
	})

	parsePlatformSkillsFn = func(platform model.Platform) ([]model.Skill, error) {
		return []model.Skill{{Name: string(platform), Platform: platform}}, nil
	}
	discoverPluginSkillsFn = func(_ string, _ bool) ([]model.Skill, error) {
		return []model.Skill{{Name: "plugin-skill", Platform: model.ClaudeCode}}, nil
	}

	skills := discoverAllSkillsForTUI(true)
	if len(skills) != len(model.AllPlatforms())+1 {
		t.Fatalf("discoverAllSkillsForTUI(true) returned %d skills, want %d", len(skills), len(model.AllPlatforms())+1)
	}
	if skills[len(skills)-1].Name != "plugin-skill" {
		t.Fatalf("discoverAllSkillsForTUI(true) last skill = %q, want plugin-skill", skills[len(skills)-1].Name)
	}
}
