package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

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
