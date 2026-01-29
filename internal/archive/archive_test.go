package archive

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/model"
)

func TestCreate(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "test-skill",
			Description: "Test skill",
			Platform:    model.ClaudeCode,
			Content:     "# Test Content",
			ModifiedAt:  time.Now(),
		},
	}

	var buf bytes.Buffer
	opts := CreateOptions{
		IncludeMeta: true,
	}

	err := Create(skills, &buf, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Create produced empty output")
	}
}

func TestExtract(t *testing.T) {
	// Create an archive first
	skills := []model.Skill{
		{
			Name:        "test-skill",
			Description: "Test skill",
			Platform:    model.ClaudeCode,
			Scope:       model.ScopeUser,
			Content:     "# Test Content",
			ModifiedAt:  time.Now(),
			Metadata:    map[string]string{"key": "value"},
		},
	}

	var buf bytes.Buffer
	createOpts := CreateOptions{
		IncludeMeta: true,
	}

	err := Create(skills, &buf, createOpts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Extract the archive
	extractOpts := ExtractOptions{
		DryRun: true, // Don't write files in tests
	}

	extracted, manifest, err := Extract(&buf, extractOpts)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if manifest == nil {
		t.Fatal("Manifest is nil")
	}

	if manifest.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", manifest.Version)
	}

	if manifest.SkillCount != 1 {
		t.Errorf("Expected 1 skill in manifest, got %d", manifest.SkillCount)
	}

	if len(extracted) != 1 {
		t.Fatalf("Expected 1 extracted skill, got %d", len(extracted))
	}

	if extracted[0].Name != "test-skill" {
		t.Errorf("Expected skill name 'test-skill', got %s", extracted[0].Name)
	}

	if extracted[0].Platform != model.ClaudeCode {
		t.Errorf("Expected platform claude-code, got %s", extracted[0].Platform)
	}
}

func TestFilterSkills(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)

	skills := []model.Skill{
		{
			Name:       "skill1",
			Platform:   model.ClaudeCode,
			ModifiedAt: yesterday,
		},
		{
			Name:       "skill2",
			Platform:   model.Cursor,
			ModifiedAt: now,
		},
		{
			Name:       "skill3",
			Platform:   model.ClaudeCode,
			ModifiedAt: tomorrow,
		},
	}

	tests := []struct {
		name     string
		opts     CreateOptions
		expected int
	}{
		{
			name:     "no filter",
			opts:     CreateOptions{},
			expected: 3,
		},
		{
			name: "platform filter",
			opts: CreateOptions{
				Platform: model.ClaudeCode,
			},
			expected: 2,
		},
		{
			name: "since filter",
			opts: CreateOptions{
				Since: now,
			},
			expected: 2, // now and tomorrow
		},
		{
			name: "before filter",
			opts: CreateOptions{
				Before: now,
			},
			expected: 1, // yesterday
		},
		{
			name: "combined filters",
			opts: CreateOptions{
				Platform: model.ClaudeCode,
				Since:    yesterday,
				Before:   now.AddDate(0, 0, 1),
			},
			expected: 1, // only skill1 (claude-code, yesterday)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterSkills(skills, tt.opts)
			if len(filtered) != tt.expected {
				t.Errorf("Expected %d filtered skills, got %d", tt.expected, len(filtered))
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple-name", "simple-name"},
		{"name/with/slashes", "name_with_slashes"},
		{"name:with:colons", "name_with_colons"},
		{"name*with?invalid<chars>", "name_with_invalid_chars_"},
		{"name|with\\backslash", "name_with_backslash"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestManifestSerialization(t *testing.T) {
	manifest := Manifest{
		Version:    "1.0",
		CreatedAt:  time.Now(),
		Platform:   "claude-code",
		SkillCount: 1,
		Skills: []ManifestSkill{
			{
				Name:       "test-skill",
				Platform:   "claude-code",
				Scope:      "user",
				ModifiedAt: time.Now(),
				Filename:   "skills/claude-code-test-skill.json",
				Size:       1024,
				Metadata:   map[string]string{"key": "value"},
			},
		},
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	var decoded Manifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal manifest: %v", err)
	}

	if decoded.Version != manifest.Version {
		t.Errorf("Version mismatch: expected %s, got %s", manifest.Version, decoded.Version)
	}

	if decoded.SkillCount != manifest.SkillCount {
		t.Errorf("SkillCount mismatch: expected %d, got %d", manifest.SkillCount, decoded.SkillCount)
	}

	if len(decoded.Skills) != len(manifest.Skills) {
		t.Errorf("Skills count mismatch: expected %d, got %d", len(manifest.Skills), len(decoded.Skills))
	}
}
