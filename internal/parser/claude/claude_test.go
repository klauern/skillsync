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
