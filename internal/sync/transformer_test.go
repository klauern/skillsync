package sync

import (
	"strings"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/model"
)

func TestNewTransformer(t *testing.T) {
	tr := NewTransformer()
	if tr == nil {
		t.Error("NewTransformer() returned nil")
	}
}

func TestTransformer_Transform(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "test-skill",
		Description: "A test skill",
		Platform:    model.ClaudeCode,
		Path:        "/source/test-skill.md",
		Tools:       []string{"Read", "Write"},
		Content:     "Test content",
		ModifiedAt:  time.Now(),
	}

	transformed, err := tr.Transform(skill, model.Cursor)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if transformed.Platform != model.Cursor {
		t.Errorf("Expected platform Cursor, got %s", transformed.Platform)
	}

	if transformed.Name != skill.Name {
		t.Errorf("Expected name %s, got %s", skill.Name, transformed.Name)
	}
}

func TestTransformer_TransformPath(t *testing.T) {
	tr := NewTransformer()

	tests := []struct {
		name       string
		sourcePath string
		target     model.Platform
		expected   string
	}{
		{
			name:       "claude to cursor md",
			sourcePath: "/source/test.md",
			target:     model.Cursor,
			expected:   "test.md",
		},
		{
			name:       "cursor mdc preserved",
			sourcePath: "/source/test.mdc",
			target:     model.Cursor,
			expected:   "test.mdc",
		},
		{
			name:       "to claude code",
			sourcePath: "/source/test.mdc",
			target:     model.ClaudeCode,
			expected:   "test.md",
		},
		{
			name:       "agents to codex",
			sourcePath: "/source/AGENTS.md",
			target:     model.Codex,
			expected:   "AGENTS.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := model.Skill{Path: tt.sourcePath}
			result := tr.transformPath(skill, tt.target)
			if result != tt.expected {
				t.Errorf("transformPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTransformer_TransformContent(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "test-skill",
		Description: "A test description",
		Content:     "The main content",
		Tools:       []string{"Read"},
	}

	// Transform to Claude Code (should include tools)
	content, err := tr.transformContent(skill, model.ClaudeCode)
	if err != nil {
		t.Fatalf("transformContent failed: %v", err)
	}

	if !strings.Contains(content, "name: test-skill") {
		t.Error("Content should contain name in frontmatter")
	}
	if !strings.Contains(content, "The main content") {
		t.Error("Content should contain main content")
	}
}

func TestTransformer_BuildFrontmatter_ClaudeCode(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "test",
		Description: "desc",
		Tools:       []string{"Read", "Write"},
	}

	fm := tr.buildFrontmatter(skill, model.ClaudeCode)

	if fm["name"] != "test" {
		t.Error("Frontmatter should contain name")
	}
	if fm["description"] != "desc" {
		t.Error("Frontmatter should contain description")
	}
	if fm["tools"] == nil {
		t.Error("Claude Code frontmatter should contain tools")
	}
}

func TestTransformer_BuildFrontmatter_Cursor(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "test",
		Description: "desc",
		Metadata: map[string]string{
			"globs":       "*.ts",
			"alwaysApply": "true",
		},
	}

	fm := tr.buildFrontmatter(skill, model.Cursor)

	if fm["globs"] != "*.ts" {
		t.Error("Cursor frontmatter should contain globs")
	}
	if fm["alwaysApply"] != "true" {
		t.Error("Cursor frontmatter should contain alwaysApply")
	}
}

func TestTransformer_BuildFrontmatter_Codex(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "test",
		Description: "desc",
	}

	fm := tr.buildFrontmatter(skill, model.Codex)

	// Codex returns nil frontmatter (plain markdown)
	if fm != nil {
		t.Error("Codex frontmatter should be nil")
	}
}

func TestTransformer_TransformMetadata(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Platform: model.Cursor,
		Metadata: map[string]string{
			"globs":       "*.ts",
			"alwaysApply": "true",
			"custom":      "value",
		},
	}

	// Transform to Claude Code - should remove Cursor-specific fields
	metadata := tr.transformMetadata(skill, model.ClaudeCode)

	if _, exists := metadata["globs"]; exists {
		t.Error("Claude Code metadata should not contain globs")
	}
	if _, exists := metadata["alwaysApply"]; exists {
		t.Error("Claude Code metadata should not contain alwaysApply")
	}
	if metadata["custom"] != "value" {
		t.Error("Custom metadata should be preserved")
	}
}

func TestTransformer_TransformMetadata_Codex(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Platform: model.ClaudeCode,
		Metadata: map[string]string{
			"custom": "value",
		},
	}

	metadata := tr.transformMetadata(skill, model.Codex)

	if metadata["source_platform"] != "claude-code" {
		t.Error("Codex metadata should contain source_platform")
	}
}

func TestTransformer_CanTransform(t *testing.T) {
	tr := NewTransformer()

	tests := []struct {
		source   model.Platform
		target   model.Platform
		expected bool
	}{
		{model.ClaudeCode, model.Cursor, true},
		{model.Cursor, model.ClaudeCode, true},
		{model.ClaudeCode, model.Codex, true},
		{model.Platform("invalid"), model.Cursor, false},
		{model.ClaudeCode, model.Platform("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.source)+"->"+string(tt.target), func(t *testing.T) {
			result := tr.CanTransform(tt.source, tt.target)
			if result != tt.expected {
				t.Errorf("CanTransform() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTransformer_MergeContent(t *testing.T) {
	tr := NewTransformer()

	source := "Source content"
	target := "Target content"
	name := "test-skill"

	merged := tr.MergeContent(source, target, name)

	if !strings.Contains(merged, "Target content") {
		t.Error("Merged content should contain target content")
	}
	if !strings.Contains(merged, "Source content") {
		t.Error("Merged content should contain source content")
	}
	if !strings.Contains(merged, "Merged from: test-skill") {
		t.Error("Merged content should contain merge header")
	}
	if !strings.Contains(merged, "---") {
		t.Error("Merged content should contain separator")
	}
}
