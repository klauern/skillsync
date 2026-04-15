package claude

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
			want:     filepath.Join(os.Getenv("HOME"), ".claude", "skills"),
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
	if got := p.Platform(); got != model.ClaudeCode {
		t.Errorf("Platform() = %v, want %v", got, model.ClaudeCode)
	}
}

func TestParser_DefaultPath(t *testing.T) {
	p := New("")
	want := filepath.Join(os.Getenv("HOME"), ".claude", "skills")
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
				"test-skill.md": `---
name: test-skill
description: A test skill
tools: [Read, Write]
---
# Test Skill

This is a test skill.`,
			},
			want: 1,
		},
		"skill without frontmatter uses filename": {
			files: map[string]string{
				"simple-skill.md": `# Simple Skill

Just content, no frontmatter.`,
			},
			want: 1,
		},
		"multiple skills": {
			files: map[string]string{
				"skill1.md": `---
name: skill1
---
Content 1`,
				"skill2.md": `---
name: skill2
---
Content 2`,
			},
			want: 2,
		},
		"nested directory structure": {
			files: map[string]string{
				"skill1.md": `---
name: skill1
---
Root skill`,
				"subdir/skill2.md": `---
name: skill2
---
Nested skill`,
			},
			want: 2,
		},
		"invalid skill name is skipped": {
			files: map[string]string{
				"valid-skill.md": `---
name: valid-skill
---
Content`,
				"invalid skill.md": `---
name: invalid skill name
---
Content`,
			},
			want: 1, // Only valid skill is parsed
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
		wantName    string
		wantDesc    string
		wantTools   []string
		wantContent string
		wantErr     bool
	}{
		"full frontmatter": {
			content: `---
name: full-skill
description: A full skill example
tools: [Read, Write, Bash]
custom: metadata
---
# Full Skill

This is the content.`,
			wantName:    "full-skill",
			wantDesc:    "A full skill example",
			wantTools:   []string{"Read", "Write", "Bash"},
			wantContent: "# Full Skill\n\nThis is the content.",
		},
		"minimal frontmatter": {
			content: `---
name: minimal
---
Content only.`,
			wantName:    "minimal",
			wantDesc:    "",
			wantTools:   nil,
			wantContent: "Content only.",
		},
		"no frontmatter": {
			content: `# No Frontmatter

Just content.`,
			wantName:  "test",
			wantDesc:  "",
			wantTools: nil,
			wantContent: `# No Frontmatter

Just content.`,
		},
		"empty tools array": {
			content: `---
name: empty-tools
tools: []
---
Content.`,
			wantName:    "empty-tools",
			wantDesc:    "",
			wantTools:   []string{},
			wantContent: "Content.",
		},
		"windows line endings": {
			content:     "---\r\nname: windows\r\n---\r\nContent\r\n",
			wantName:    "windows",
			wantDesc:    "",
			wantTools:   nil,
			wantContent: "Content",
		},
		"alternative frontmatter delimiter": {
			content: `+++
name: alternative
description: Using +++ delimiter
+++
Content here.`,
			wantName:    "alternative",
			wantDesc:    "Using +++ delimiter",
			wantTools:   nil,
			wantContent: "Content here.",
		},
		"invalid skill name": {
			content: `---
name: invalid name with spaces
---
Content`,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test.md")
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
			if skill.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", skill.Description, tt.wantDesc)
			}
			if len(skill.Tools) != len(tt.wantTools) {
				t.Errorf("Tools length = %d, want %d", len(skill.Tools), len(tt.wantTools))
			}
			for i, tool := range tt.wantTools {
				if i < len(skill.Tools) && skill.Tools[i] != tool {
					t.Errorf("Tools[%d] = %q, want %q", i, skill.Tools[i], tool)
				}
			}
			if skill.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", skill.Content, tt.wantContent)
			}
			if skill.Platform != model.ClaudeCode {
				t.Errorf("Platform = %v, want %v", skill.Platform, model.ClaudeCode)
			}
			if skill.Path != filePath {
				t.Errorf("Path = %q, want %q", skill.Path, filePath)
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
name: metadata-test
description: Testing metadata
custom_field: custom_value
author: Test Author
version: 1.0.0
---
Content`

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

	// Check that custom fields are in metadata
	expectedMetadata := map[string]string{
		"custom_field": "custom_value",
		"author":       "Test Author",
		"version":      "1.0.0",
	}

	for key, want := range expectedMetadata {
		if got, ok := skill.Metadata[key]; !ok {
			t.Errorf("Metadata missing key %q", key)
		} else if got != want {
			t.Errorf("Metadata[%q] = %q, want %q", key, got, want)
		}
	}

	// Verify that name, description, tools are NOT in metadata
	if _, ok := skill.Metadata["name"]; ok {
		t.Error("name should not be in Metadata")
	}
	if _, ok := skill.Metadata["description"]; ok {
		t.Error("description should not be in Metadata")
	}
	if _, ok := skill.Metadata["tools"]; ok {
		t.Error("tools should not be in Metadata")
	}
}

func TestParser_Parse_SkillMdSupport(t *testing.T) {
	t.Run("SKILL.md files are parsed", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a skill directory with SKILL.md
		skillDir := filepath.Join(tmpDir, "my-skill")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("failed to create skill directory: %v", err)
		}

		skillMd := `---
name: my-skill
description: A skill using Agent Skills Standard
scope: user
---
# My Skill

This is a SKILL.md format skill.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0o644); err != nil {
			t.Fatalf("failed to write SKILL.md: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if len(skills) != 1 {
			t.Fatalf("expected 1 skill, got %d", len(skills))
		}

		skill := skills[0]
		if skill.Name != "my-skill" {
			t.Errorf("Name = %q, want %q", skill.Name, "my-skill")
		}
		if skill.Description != "A skill using Agent Skills Standard" {
			t.Errorf("Description = %q, want %q", skill.Description, "A skill using Agent Skills Standard")
		}
		if skill.Platform != model.ClaudeCode {
			t.Errorf("Platform = %v, want %v", skill.Platform, model.ClaudeCode)
		}
	})

	t.Run("SKILL.md takes precedence over legacy files with same name", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a legacy skill file
		legacyContent := `---
name: duplicate-skill
description: Legacy version
---
Legacy content.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(tmpDir, "duplicate-skill.md"), []byte(legacyContent), 0o644); err != nil {
			t.Fatalf("failed to write legacy file: %v", err)
		}

		// Create a SKILL.md version with same name
		skillDir := filepath.Join(tmpDir, "duplicate-skill")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("failed to create skill directory: %v", err)
		}

		skillMdContent := `---
name: duplicate-skill
description: SKILL.md version
---
SKILL.md content.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMdContent), 0o644); err != nil {
			t.Fatalf("failed to write SKILL.md: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		// Should only have 1 skill (SKILL.md version)
		if len(skills) != 1 {
			t.Fatalf("expected 1 skill (SKILL.md should take precedence), got %d", len(skills))
		}

		skill := skills[0]
		if skill.Description != "SKILL.md version" {
			t.Errorf("Expected SKILL.md version to take precedence, got description: %q", skill.Description)
		}
	})

	t.Run("mixed SKILL.md and legacy files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a legacy-only skill
		legacyContent := `---
name: legacy-only
description: Legacy skill only
---
Legacy content.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(tmpDir, "legacy-only.md"), []byte(legacyContent), 0o644); err != nil {
			t.Fatalf("failed to write legacy file: %v", err)
		}

		// Create a SKILL.md-only skill
		skillDir := filepath.Join(tmpDir, "skillmd-only")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("failed to create skill directory: %v", err)
		}

		skillMdContent := `---
name: skillmd-only
description: SKILL.md skill only
---
SKILL.md content.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMdContent), 0o644); err != nil {
			t.Fatalf("failed to write SKILL.md: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		// Should have both skills
		if len(skills) != 2 {
			t.Fatalf("expected 2 skills, got %d", len(skills))
		}

		// Check both skills are present
		names := make(map[string]bool)
		for _, s := range skills {
			names[s.Name] = true
		}
		if !names["legacy-only"] {
			t.Error("missing legacy-only skill")
		}
		if !names["skillmd-only"] {
			t.Error("missing skillmd-only skill")
		}
	})

	t.Run("Claude-specific tools array in SKILL.md", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a skill directory with SKILL.md containing tools array
		skillDir := filepath.Join(tmpDir, "tool-skill")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("failed to create skill directory: %v", err)
		}

		skillMd := `---
name: tool-skill
description: Skill with tools
tools:
  - Read
  - Write
  - Bash
---
# Tool Skill

This skill has tools.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0o644); err != nil {
			t.Fatalf("failed to write SKILL.md: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if len(skills) != 1 {
			t.Fatalf("expected 1 skill, got %d", len(skills))
		}

		skill := skills[0]
		if len(skill.Tools) != 3 {
			t.Errorf("expected 3 tools, got %d", len(skill.Tools))
		}
		expectedTools := []string{"Read", "Write", "Bash"}
		for i, want := range expectedTools {
			if i < len(skill.Tools) && skill.Tools[i] != want {
				t.Errorf("Tools[%d] = %q, want %q", i, skill.Tools[i], want)
			}
		}
	})
}

// TestParser_BackwardCompatibility tests backward compatibility with legacy formats
func TestParser_BackwardCompatibility(t *testing.T) {
	t.Run("legacy flat file without frontmatter", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create legacy file with no frontmatter
		content := `# No Frontmatter Skill

This skill has no YAML frontmatter at all.
The name should be derived from the filename.

Content without any metadata.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(tmpDir, "no-frontmatter.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if len(skills) != 1 {
			t.Fatalf("expected 1 skill, got %d", len(skills))
		}

		// Name should be derived from filename
		if skills[0].Name != "no-frontmatter" {
			t.Errorf("Name = %q, want %q (derived from filename)", skills[0].Name, "no-frontmatter")
		}
	})

	t.Run("legacy flat file with minimal frontmatter", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create legacy file with only name in frontmatter
		content := `---
name: minimal-frontmatter
---
# Minimal Frontmatter Skill

This skill has only a name in frontmatter (no description).`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(tmpDir, "minimal.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if len(skills) != 1 {
			t.Fatalf("expected 1 skill, got %d", len(skills))
		}

		if skills[0].Name != "minimal-frontmatter" {
			t.Errorf("Name = %q, want %q", skills[0].Name, "minimal-frontmatter")
		}
		if skills[0].Description != "" {
			t.Errorf("Description = %q, want empty string", skills[0].Description)
		}
	})

	t.Run("legacy file with plus delimiter", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create legacy file using +++ delimiter
		content := `+++
name: plus-delimiter
description: Skill using +++ delimiter instead of ---
+++
# Plus Delimiter Skill

This skill uses the alternative +++ frontmatter delimiter.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(tmpDir, "plus-delimiter.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if len(skills) != 1 {
			t.Fatalf("expected 1 skill, got %d", len(skills))
		}

		if skills[0].Name != "plus-delimiter" {
			t.Errorf("Name = %q, want %q", skills[0].Name, "plus-delimiter")
		}
		if skills[0].Description != "Skill using +++ delimiter instead of ---" {
			t.Errorf("Description = %q, want %q", skills[0].Description, "Skill using +++ delimiter instead of ---")
		}
	})

	t.Run("legacy file coexists with SKILL.md format", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create legacy file
		legacyContent := `---
name: legacy-format
description: Legacy flat file skill
---
Legacy skill content.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(tmpDir, "legacy-format.md"), []byte(legacyContent), 0o644); err != nil {
			t.Fatalf("failed to write legacy file: %v", err)
		}

		// Create SKILL.md format skill
		skillDir := filepath.Join(tmpDir, "modern-format")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("failed to create skill directory: %v", err)
		}
		skillMdContent := `---
name: modern-format
description: Agent Skills Standard format skill
---
Modern skill content.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMdContent), 0o644); err != nil {
			t.Fatalf("failed to write SKILL.md: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		// Should have both skills
		if len(skills) != 2 {
			t.Fatalf("expected 2 skills, got %d", len(skills))
		}

		// Verify both are present
		names := make(map[string]bool)
		for _, s := range skills {
			names[s.Name] = true
		}
		if !names["legacy-format"] {
			t.Error("missing legacy-format skill")
		}
		if !names["modern-format"] {
			t.Error("missing modern-format skill")
		}
	})

	t.Run("Windows line endings in legacy file", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create file with Windows line endings
		content := "---\r\nname: windows-legacy\r\ndescription: Windows line endings\r\n---\r\n# Windows Legacy\r\n\r\nContent with CRLF.\r\n"
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(tmpDir, "windows-legacy.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if len(skills) != 1 {
			t.Fatalf("expected 1 skill, got %d", len(skills))
		}

		if skills[0].Name != "windows-legacy" {
			t.Errorf("Name = %q, want %q", skills[0].Name, "windows-legacy")
		}
		// Content should have normalized line endings
		if skills[0].Content != "# Windows Legacy\n\nContent with CRLF." {
			t.Errorf("Content not properly normalized, got %q", skills[0].Content)
		}
	})
}

// TestParser_SkillDirectoryExclusion tests that files inside skill directories are excluded from legacy parsing
func TestParser_SkillDirectoryExclusion(t *testing.T) {
	t.Run("files in patterns/ subdirectory are excluded", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a skill directory with SKILL.md
		skillDir := filepath.Join(tmpDir, "garden")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("failed to create skill directory: %v", err)
		}

		skillMd := `---
name: garden
description: Zendesk Garden design system
---
# Garden Skill

Main content here.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0o644); err != nil {
			t.Fatalf("failed to write SKILL.md: %v", err)
		}

		// Create patterns/ subdirectory with md files
		patternsDir := filepath.Join(skillDir, "patterns")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(patternsDir, 0o755); err != nil {
			t.Fatalf("failed to create patterns directory: %v", err)
		}

		// Create reference files that should NOT be treated as skills
		referenceFiles := map[string]string{
			"accessibility.md": "# Accessibility Patterns",
			"forms.md":         "# Form Patterns",
			"theming.md":       "# Theming Guide",
		}
		for name, content := range referenceFiles {
			// #nosec G306 - test file permissions
			if err := os.WriteFile(filepath.Join(patternsDir, name), []byte(content), 0o644); err != nil {
				t.Fatalf("failed to write reference file %s: %v", name, err)
			}
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		// Should only have 1 skill (garden), not 4 (garden + accessibility + forms + theming)
		if len(skills) != 1 {
			t.Errorf("expected 1 skill, got %d", len(skills))
			for _, s := range skills {
				t.Logf("  found skill: %s at %s", s.Name, s.Path)
			}
		}

		if skills[0].Name != "garden" {
			t.Errorf("expected garden skill, got %s", skills[0].Name)
		}
	})

	t.Run("files in references/ subdirectory are excluded", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a skill directory with SKILL.md
		skillDir := filepath.Join(tmpDir, "my-skill")
		refsDir := filepath.Join(skillDir, "references")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(refsDir, 0o755); err != nil {
			t.Fatalf("failed to create references directory: %v", err)
		}

		skillMd := `---
name: my-skill
description: A test skill
---
Main content.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0o644); err != nil {
			t.Fatalf("failed to write SKILL.md: %v", err)
		}

		// Create reference file
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(refsDir, "components.md"), []byte("# Components Reference"), 0o644); err != nil {
			t.Fatalf("failed to write reference file: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		// Should only have 1 skill
		if len(skills) != 1 {
			t.Errorf("expected 1 skill, got %d", len(skills))
		}

		// Verify the components.md was NOT parsed as a skill
		for _, s := range skills {
			if s.Name == "components" {
				t.Errorf("components.md should not be parsed as a separate skill")
			}
		}
	})

	t.Run("deeply nested files in skill directories are excluded", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a skill directory with nested structure
		skillDir := filepath.Join(tmpDir, "complex-skill")
		deepDir := filepath.Join(skillDir, "patterns", "advanced", "examples")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(deepDir, 0o755); err != nil {
			t.Fatalf("failed to create deep directory: %v", err)
		}

		skillMd := `---
name: complex-skill
description: Complex skill
---
Content.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0o644); err != nil {
			t.Fatalf("failed to write SKILL.md: %v", err)
		}

		// Create deeply nested file
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(deepDir, "advanced-example.md"), []byte("# Example"), 0o644); err != nil {
			t.Fatalf("failed to write nested file: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		// Should only have 1 skill
		if len(skills) != 1 {
			t.Errorf("expected 1 skill, got %d", len(skills))
		}
	})

	t.Run("legacy files outside skill directories are still parsed", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a SKILL.md skill
		skillDir := filepath.Join(tmpDir, "modern-skill")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("failed to create skill directory: %v", err)
		}

		skillMd := `---
name: modern-skill
description: Modern skill
---
Content.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0o644); err != nil {
			t.Fatalf("failed to write SKILL.md: %v", err)
		}

		// Create a legacy file at root level (NOT inside skill directory)
		legacyContent := `---
name: legacy-skill
description: Legacy skill
---
Legacy content.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(tmpDir, "legacy-skill.md"), []byte(legacyContent), 0o644); err != nil {
			t.Fatalf("failed to write legacy file: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		// Should have 2 skills
		if len(skills) != 2 {
			t.Errorf("expected 2 skills, got %d", len(skills))
		}

		// Verify both skills are present
		names := make(map[string]bool)
		for _, s := range skills {
			names[s.Name] = true
		}
		if !names["modern-skill"] {
			t.Error("missing modern-skill")
		}
		if !names["legacy-skill"] {
			t.Error("missing legacy-skill")
		}
	})
}

func TestParser_Parse_CommandFilesAsPrompts(t *testing.T) {
	tmpDir := t.TempDir()
	commandsDir := filepath.Join(tmpDir, "commands")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		t.Fatalf("failed to create commands directory: %v", err)
	}

	commandContent := `---
description: Review code quality
allowed-tools: Bash, Read, Grep
---
# /review

Run a focused review.`

	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(commandsDir, "review.md"), []byte(commandContent), 0o644); err != nil {
		t.Fatalf("failed to write command file: %v", err)
	}

	p := New(commandsDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 parsed artifact, got %d", len(skills))
	}

	s := skills[0]
	if s.Name != "review" {
		t.Errorf("Name = %q, want %q", s.Name, "review")
	}
	if s.Type != model.SkillTypePrompt {
		t.Errorf("Type = %q, want %q", s.Type, model.SkillTypePrompt)
	}
	if s.Trigger != "/review" {
		t.Errorf("Trigger = %q, want %q", s.Trigger, "/review")
	}
	if s.Description != "Review code quality" {
		t.Errorf("Description = %q, want %q", s.Description, "Review code quality")
	}
	if len(s.Tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(s.Tools))
	}
	wantTools := []string{"Bash", "Read", "Grep"}
	for i, want := range wantTools {
		if s.Tools[i] != want {
			t.Errorf("Tools[%d] = %q, want %q", i, s.Tools[i], want)
		}
	}
}

func TestParser_Parse_CommandFrontmatterTypeOverride(t *testing.T) {
	tmpDir := t.TempDir()
	commandsDir := filepath.Join(tmpDir, "commands")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		t.Fatalf("failed to create commands directory: %v", err)
	}

	commandContent := `---
name: build
type: prompt
trigger: /build-fast
allowed-tools: [Bash, Read]
---
Build quickly.`

	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(commandsDir, "build.md"), []byte(commandContent), 0o644); err != nil {
		t.Fatalf("failed to write command file: %v", err)
	}

	p := New(commandsDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 parsed artifact, got %d", len(skills))
	}

	s := skills[0]
	if s.Type != model.SkillTypePrompt {
		t.Errorf("Type = %q, want %q", s.Type, model.SkillTypePrompt)
	}
	if s.Trigger != "/build-fast" {
		t.Errorf("Trigger = %q, want %q", s.Trigger, "/build-fast")
	}
}

func TestParser_Parse_CommandPathSkillStyleFrontmatterStaysSkill(t *testing.T) {
	tmpDir := t.TempDir()
	commandsDir := filepath.Join(tmpDir, "commands")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		t.Fatalf("failed to create commands directory: %v", err)
	}

	// This resembles legacy skill frontmatter stored in commands path.
	// We keep it as skill for backward compatibility.
	content := `---
name: legacy-skill
description: legacy skill style
tools: [Read]
---
Legacy content.`
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(commandsDir, "legacy-skill.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	p := New(commandsDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 parsed artifact, got %d", len(skills))
	}
	if skills[0].Type != model.SkillTypeSkill {
		t.Errorf("Type = %q, want %q", skills[0].Type, model.SkillTypeSkill)
	}
	if skills[0].Trigger != "" {
		t.Errorf("Trigger = %q, want empty", skills[0].Trigger)
	}
}

func TestIsClaudeCommandFile(t *testing.T) {
	tests := map[string]struct {
		path string
		want bool
	}{
		"repo commands": {
			path: "/tmp/repo/.claude/commands/review.md",
			want: true,
		},
		"user commands": {
			path: "/Users/test/.claude/commands/build.md",
			want: true,
		},
		"skills path": {
			path: "/tmp/repo/.claude/skills/review.md",
			want: false,
		},
		"non markdown": {
			path: "/tmp/repo/.claude/commands/review.txt",
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := isClaudeCommandFile(tt.path)
			if got != tt.want {
				t.Errorf("isClaudeCommandFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestParser_CaseInsensitiveSkillMd tests that lowercase skill.md files are recognized
func TestParser_CaseInsensitiveSkillMd(t *testing.T) {
	t.Run("lowercase skill.md is recognized", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a skill directory with lowercase skill.md
		skillDir := filepath.Join(tmpDir, "lowercase-skill")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("failed to create skill directory: %v", err)
		}

		skillMd := `---
name: lowercase-skill
description: A skill with lowercase skill.md
---
Content here.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte(skillMd), 0o644); err != nil {
			t.Fatalf("failed to write skill.md: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if len(skills) != 1 {
			t.Fatalf("expected 1 skill, got %d", len(skills))
		}

		if skills[0].Name != "lowercase-skill" {
			t.Errorf("Name = %q, want %q", skills[0].Name, "lowercase-skill")
		}
	})

	t.Run("files inside lowercase skill.md directories are excluded", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a skill directory with lowercase skill.md
		skillDir := filepath.Join(tmpDir, "garden")
		patternsDir := filepath.Join(skillDir, "patterns")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(patternsDir, 0o755); err != nil {
			t.Fatalf("failed to create patterns directory: %v", err)
		}

		skillMd := `---
name: garden
description: Garden skill
---
Content.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte(skillMd), 0o644); err != nil {
			t.Fatalf("failed to write skill.md: %v", err)
		}

		// Create a pattern file that should be excluded
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(patternsDir, "accessibility.md"), []byte("# Accessibility"), 0o644); err != nil {
			t.Fatalf("failed to write pattern file: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		// Should only have 1 skill (garden), not 2
		if len(skills) != 1 {
			t.Errorf("expected 1 skill, got %d", len(skills))
			for _, s := range skills {
				t.Logf("  found skill: %s", s.Name)
			}
		}

		if skills[0].Name != "garden" {
			t.Errorf("expected garden skill, got %s", skills[0].Name)
		}
	})
}
