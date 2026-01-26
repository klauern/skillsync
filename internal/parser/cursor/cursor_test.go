package cursor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/model"
)

func TestNew(t *testing.T) {
	tests := map[string]struct {
		basePath string
		want     string
	}{
		"empty path uses default": {
			basePath: "",
			want:     filepath.Join(os.Getenv("HOME"), ".cursor", "skills"),
		},
		"custom path preserved": {
			basePath: "/custom/path",
			want:     "/custom/path",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p := New(tt.basePath)
			if p.basePath != tt.want {
				t.Errorf("New(%q).basePath = %q, want %q", tt.basePath, p.basePath, tt.want)
			}
		})
	}
}

func TestParser_Platform(t *testing.T) {
	p := New("")
	if got := p.Platform(); got != model.Cursor {
		t.Errorf("Platform() = %v, want %v", got, model.Cursor)
	}
}

func TestParser_DefaultPath(t *testing.T) {
	p := New("")
	want := filepath.Join(os.Getenv("HOME"), ".cursor", "skills")
	if got := p.DefaultPath(); got != want {
		t.Errorf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestParser_Parse(t *testing.T) {
	tests := map[string]struct {
		files   map[string]string
		want    int
		wantErr bool
	}{
		"empty directory": {
			files: map[string]string{},
			want:  0,
		},
		"single skill with frontmatter": {
			files: map[string]string{
				"test-rule.md": `---
globs: ["*.go", "*.rs"]
alwaysApply: false
---

# Test Rule

This is a test rule.`,
			},
			want: 1,
		},
		"skill without frontmatter uses filename": {
			files: map[string]string{
				"simple-rule.md": `# Simple Rule

Just content, no frontmatter.`,
			},
			want: 1,
		},
		"multiple skills": {
			files: map[string]string{
				"rule1.md": `---
globs: ["*.js"]
alwaysApply: true
---
Content 1`,
				"rule2.md": `---
globs: ["*.ts"]
---
Content 2`,
			},
			want: 2,
		},
		"nested directory structure": {
			files: map[string]string{
				"rule1.md": `---
globs: ["*.go"]
---
Root rule`,
				"subdir/rule2.md": `---
globs: ["*.rs"]
---
Nested rule`,
			},
			want: 2,
		},
		"both md and mdc files": {
			files: map[string]string{
				"standard.md": `---
globs: ["*.md"]
---
Standard markdown`,
				"cursor-custom.mdc": `---
globs: ["*.mdc"]
---
Cursor markdown`,
			},
			want: 2,
		},
		"invalid skill name is skipped": {
			files: map[string]string{
				"valid-rule.md": `---
globs: ["*.go"]
---
Content`,
				"invalid rule.md": `---
globs: ["*.rs"]
---
Content`,
			},
			want: 1, // Only valid skill is parsed
		},
		"deeply nested files": {
			files: map[string]string{
				"a/b/c/deep-rule.md": `---
globs: ["**/*.deep"]
---
Deep nested rule`,
			},
			want: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Create test files
			for path, content := range tt.files {
				fullPath := filepath.Join(tmpDir, path)
				dir := filepath.Dir(fullPath)
				// #nosec G301 - test directory permissions
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatalf("failed to create directory %q: %v", dir, err)
				}
				// #nosec G306 - test file permissions
				if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
					t.Fatalf("failed to write file %q: %v", fullPath, err)
				}
			}

			// Parse skills
			p := New(tmpDir)
			skills, err := p.Parse()

			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if got := len(skills); got != tt.want {
				t.Errorf("Parse() returned %d skills, want %d", got, tt.want)
			}
		})
	}
}

func TestParser_Parse_NonexistentDirectory(t *testing.T) {
	p := New("/nonexistent/directory/path")
	skills, err := p.Parse()
	if err != nil {
		t.Errorf("Parse() on nonexistent directory should not error, got: %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("Parse() on nonexistent directory should return empty slice, got %d skills", len(skills))
	}
}

func TestParser_parseSkillFile(t *testing.T) {
	tests := map[string]struct {
		content     string
		filename    string
		wantName    string
		wantContent string
		wantMeta    map[string]string
		wantErr     bool
	}{
		"full frontmatter with globs": {
			filename: "go-rule.md",
			content: `---
globs: ["*.go", "internal/**/*.go"]
alwaysApply: false
---

# Go Style Guide

Follow Go conventions.`,
			wantName:    "go-rule",
			wantContent: "# Go Style Guide\n\nFollow Go conventions.",
			wantMeta: map[string]string{
				"globs":       "[*.go internal/**/*.go]",
				"alwaysApply": "false",
			},
		},
		"frontmatter with alwaysApply true": {
			filename: "always-rule.md",
			content: `---
alwaysApply: true
---

Always apply this rule.`,
			wantName:    "always-rule",
			wantContent: "Always apply this rule.",
			wantMeta: map[string]string{
				"alwaysApply": "true",
			},
		},
		"minimal frontmatter": {
			filename: "minimal.md",
			content: `---
globs: ["*.rs"]
---
Content only.`,
			wantName:    "minimal",
			wantContent: "Content only.",
			wantMeta: map[string]string{
				"globs": "[*.rs]",
			},
		},
		"no frontmatter": {
			filename: "test.md",
			content: `# No Frontmatter

Just content.`,
			wantName: "test",
			wantContent: `# No Frontmatter

Just content.`,
			wantMeta: map[string]string{},
		},
		"empty globs array": {
			filename: "empty-globs.md",
			content: `---
globs: []
---
Content.`,
			wantName:    "empty-globs",
			wantContent: "Content.",
			wantMeta: map[string]string{
				"globs": "[]",
			},
		},
		"windows line endings": {
			filename:    "windows.md",
			content:     "---\r\nglobs: [\"*.go\"]\r\n---\r\nContent\r\n",
			wantName:    "windows",
			wantContent: "Content",
			wantMeta: map[string]string{
				"globs": "[*.go]",
			},
		},
		"alternative frontmatter delimiter": {
			filename: "alt.md",
			content: `+++
globs: ["*.ts"]
+++
Content here.`,
			wantName:    "alt",
			wantContent: "Content here.",
			wantMeta: map[string]string{
				"globs": "[*.ts]",
			},
		},
		"mdc file extension": {
			filename: "cursor-rule.mdc",
			content: `---
globs: ["*.mdc"]
alwaysApply: true
---

Cursor markdown format.`,
			wantName:    "cursor-rule",
			wantContent: "Cursor markdown format.",
			wantMeta: map[string]string{
				"globs":       "[*.mdc]",
				"alwaysApply": "true",
			},
		},
		"name in frontmatter takes precedence": {
			filename: "filename.md",
			content: `---
name: custom-name
globs: ["*.go"]
---
Content`,
			wantName:    "custom-name",
			wantContent: "Content",
			wantMeta: map[string]string{
				"globs": "[*.go]",
			},
		},
		"frontmatter only no content": {
			filename: "frontmatter-only.md",
			content: `---
globs: ["*.go"]
---
`,
			wantName:    "frontmatter-only",
			wantContent: "",
			wantMeta: map[string]string{
				"globs": "[*.go]",
			},
		},
		"invalid skill name in filename": {
			filename: "bad name.md",
			content: `---
globs: ["*.go"]
---
Content`,
			wantErr: true, // "bad name" has spaces
		},
		"complex globs pattern": {
			filename: "complex.md",
			content: `---
globs: ["src/**/*.ts", "tests/**/*.test.ts", "!**/*.node.ts"]
alwaysApply: false
priority: 1
---

Complex patterns.`,
			wantName:    "complex",
			wantContent: "Complex patterns.",
			wantMeta: map[string]string{
				"globs":       "[src/**/*.ts tests/**/*.test.ts !**/*.node.ts]",
				"alwaysApply": "false",
				"priority":    "1",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, tt.filename)
			// #nosec G306 - test file permissions
			if err := os.WriteFile(filePath, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Parse the file
			p := New(tmpDir)
			skill, err := p.parseSkillFile(filePath)

			if (err != nil) != tt.wantErr {
				t.Fatalf("parseSkillFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Verify skill fields
			if skill.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", skill.Name, tt.wantName)
			}
			if skill.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", skill.Content, tt.wantContent)
			}
			if skill.Platform != model.Cursor {
				t.Errorf("Platform = %v, want %v", skill.Platform, model.Cursor)
			}
			if skill.Path != filePath {
				t.Errorf("Path = %q, want %q", skill.Path, filePath)
			}
			if skill.Description != "" {
				t.Errorf("Description should be empty for Cursor, got %q", skill.Description)
			}
			if skill.Tools != nil {
				t.Errorf("Tools should be nil for Cursor, got %v", skill.Tools)
			}

			// Verify metadata
			for key, want := range tt.wantMeta {
				if got, ok := skill.Metadata[key]; !ok {
					t.Errorf("Metadata missing key %q", key)
				} else if got != want {
					t.Errorf("Metadata[%q] = %q, want %q", key, got, want)
				}
			}

			// Verify ModifiedAt is set and recent
			if skill.ModifiedAt.IsZero() {
				t.Error("ModifiedAt should be set")
			}
			if time.Since(skill.ModifiedAt) > 5*time.Second {
				t.Errorf("ModifiedAt seems too old: %v", skill.ModifiedAt)
			}
		})
	}
}

func TestParser_parseSkillFile_Metadata(t *testing.T) {
	content := `---
globs: ["*.go", "*.rs"]
alwaysApply: true
custom_field: custom_value
author: Test Author
version: 1.0.0
priority: 5
---

Content with metadata`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := New(tmpDir)
	skill, err := p.parseSkillFile(filePath)
	if err != nil {
		t.Fatalf("parseSkillFile() error = %v", err)
	}

	// Check that all frontmatter fields are in metadata
	expectedMetadata := map[string]string{
		"globs":        "[*.go *.rs]",
		"alwaysApply":  "true",
		"custom_field": "custom_value",
		"author":       "Test Author",
		"version":      "1.0.0",
		"priority":     "5",
	}

	for key, want := range expectedMetadata {
		if got, ok := skill.Metadata[key]; !ok {
			t.Errorf("Metadata missing key %q", key)
		} else if got != want {
			t.Errorf("Metadata[%q] = %q, want %q", key, got, want)
		}
	}

	// Verify that name is NOT in metadata (if specified)
	if _, ok := skill.Metadata["name"]; ok {
		t.Error("name should not be in Metadata")
	}
}

func TestParser_Parse_Integration(t *testing.T) {
	// Integration test with real Cursor-style rules
	tmpDir := t.TempDir()

	// Create a realistic Cursor rules directory structure
	files := map[string]string{
		"go-style.md": `---
globs: ["*.go"]
---

# Go Style Guide

Follow standard Go conventions:
- Use gofmt
- Keep functions short`,
		"typescript.mdc": `---
globs: ["*.ts", "*.tsx"]
alwaysApply: true
---

# TypeScript Rules

Always use strict mode.`,
		"docs/writing-style.md": `---
globs: ["*.md", "**/*.md"]
---

Write in clear, simple English.`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		// #nosec G306 - test file permissions
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(skills) != 3 {
		t.Fatalf("Parse() returned %d skills, want 3", len(skills))
	}

	// Verify go-style skill
	goSkill := findSkillByName(t, skills, "go-style")
	if goSkill.Metadata["globs"] != "[*.go]" {
		t.Errorf("go-style globs = %q, want [*.go]", goSkill.Metadata["globs"])
	}

	// Verify typescript skill (mdc extension)
	tsSkill := findSkillByName(t, skills, "typescript")
	if tsSkill.Metadata["globs"] != "[*.ts *.tsx]" {
		t.Errorf("typescript globs = %q, want [*.ts *.tsx]", tsSkill.Metadata["globs"])
	}
	if tsSkill.Metadata["alwaysApply"] != "true" {
		t.Errorf("typescript alwaysApply = %q, want true", tsSkill.Metadata["alwaysApply"])
	}

	// Verify nested skill
	docSkill := findSkillByName(t, skills, "writing-style")
	if docSkill.Metadata["globs"] != "[*.md **/*.md]" {
		t.Errorf("writing-style globs = %q, want [*.md **/*.md]", docSkill.Metadata["globs"])
	}
}

// findSkillByName is a test helper to find a skill by name
func findSkillByName(t *testing.T, skills []model.Skill, name string) model.Skill {
	t.Helper()
	for _, s := range skills {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("skill %q not found", name)
	return model.Skill{}
}

func TestParser_Parse_SkillMD(t *testing.T) {
	// Test parsing SKILL.md files in Agent Skills Standard format
	tmpDir := t.TempDir()

	// Create a SKILL.md file in a subdirectory
	skillDir := filepath.Join(tmpDir, "my-cursor-skill")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	skillContent := `---
name: my-cursor-skill
description: A test skill for Cursor
tools: ["read", "write"]
disable-model-invocation: true
---

# My Cursor Skill

This is a skill for Cursor that follows the Agent Skills Standard.
`
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("Parse() returned %d skills, want 1", len(skills))
	}

	skill := skills[0]
	if skill.Name != "my-cursor-skill" {
		t.Errorf("Name = %q, want %q", skill.Name, "my-cursor-skill")
	}
	if skill.Description != "A test skill for Cursor" {
		t.Errorf("Description = %q, want %q", skill.Description, "A test skill for Cursor")
	}
	if skill.Platform != model.Cursor {
		t.Errorf("Platform = %v, want %v", skill.Platform, model.Cursor)
	}
	if len(skill.Tools) != 2 || skill.Tools[0] != "read" || skill.Tools[1] != "write" {
		t.Errorf("Tools = %v, want [read write]", skill.Tools)
	}
	if !skill.DisableModelInvocation {
		t.Error("DisableModelInvocation should be true")
	}
}

func TestParser_Parse_MixedFormats(t *testing.T) {
	// Test that both legacy .md/.mdc files and SKILL.md files are discovered
	tmpDir := t.TempDir()

	// Create a legacy .md file
	legacyContent := `---
globs: ["*.go"]
alwaysApply: true
---

# Legacy Rule

This is a legacy Cursor rule.`
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(tmpDir, "legacy-rule.md"), []byte(legacyContent), 0o644); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	// Create a SKILL.md file
	skillDir := filepath.Join(tmpDir, "agent-skill")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}
	skillContent := `---
name: agent-skill
description: An Agent Skills Standard skill
---

# Agent Skill

This follows the Agent Skills Standard.`
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(skills) != 2 {
		t.Fatalf("Parse() returned %d skills, want 2", len(skills))
	}

	// Verify both skills are present
	_ = findSkillByName(t, skills, "legacy-rule")
	_ = findSkillByName(t, skills, "agent-skill")
}

func TestParser_Parse_SkillMDPrecedence(t *testing.T) {
	// Test that SKILL.md takes precedence over legacy files with the same name
	tmpDir := t.TempDir()

	// Create a legacy .md file with name "my-skill"
	legacyContent := `---
globs: ["*.old"]
---

# Legacy Content`
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(tmpDir, "my-skill.md"), []byte(legacyContent), 0o644); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	// Create a SKILL.md file with the same name
	skillDir := filepath.Join(tmpDir, "my-skill")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}
	skillContent := `---
name: my-skill
description: SKILL.md version
---

# Agent Skills Standard Content`
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Should only return 1 skill (SKILL.md takes precedence)
	if len(skills) != 1 {
		t.Fatalf("Parse() returned %d skills, want 1", len(skills))
	}

	skill := skills[0]
	if skill.Name != "my-skill" {
		t.Errorf("Name = %q, want %q", skill.Name, "my-skill")
	}
	if skill.Description != "SKILL.md version" {
		t.Errorf("Description = %q, want SKILL.md version (SKILL.md should take precedence)", skill.Description)
	}
}
