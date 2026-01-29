package skills

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
		platform model.Platform
	}{
		"creates parser with custom path": {
			basePath: "/custom/path",
			platform: model.ClaudeCode,
		},
		"creates parser with cursor platform": {
			basePath: "/cursor/path",
			platform: model.Cursor,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p := New(tt.basePath, tt.platform)
			if p.basePath != tt.basePath {
				t.Errorf("New().basePath = %q, want %q", p.basePath, tt.basePath)
			}
			if p.platform != tt.platform {
				t.Errorf("New().platform = %v, want %v", p.platform, tt.platform)
			}
		})
	}
}

func TestParser_Platform(t *testing.T) {
	tests := map[string]struct {
		platform model.Platform
	}{
		"claude-code": {platform: model.ClaudeCode},
		"cursor":      {platform: model.Cursor},
		"codex":       {platform: model.Codex},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p := New("/test", tt.platform)
			if got := p.Platform(); got != tt.platform {
				t.Errorf("Platform() = %v, want %v", got, tt.platform)
			}
		})
	}
}

func TestParser_DefaultPath(t *testing.T) {
	p := New("/custom/path", model.ClaudeCode)
	if got := p.DefaultPath(); got != "/custom/path" {
		t.Errorf("DefaultPath() = %q, want %q", got, "/custom/path")
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
		"single SKILL.md": {
			files: map[string]string{
				"my-skill/SKILL.md": `---
name: my-skill
description: A test skill
---
# My Skill

This is a test skill.`,
			},
			want: 1,
		},
		"multiple SKILL.md files": {
			files: map[string]string{
				"skill1/SKILL.md": `---
name: skill1
description: First skill
---
Content 1`,
				"skill2/SKILL.md": `---
name: skill2
description: Second skill
---
Content 2`,
			},
			want: 2,
		},
		"nested directory structure": {
			files: map[string]string{
				"root-skill/SKILL.md": `---
name: root-skill
description: Root skill
---
Root content`,
				"category/nested-skill/SKILL.md": `---
name: nested-skill
description: Nested skill
---
Nested content`,
			},
			want: 2,
		},
		"invalid skill name is skipped": {
			files: map[string]string{
				"valid-skill/SKILL.md": `---
name: valid-skill
description: Valid
---
Content`,
				"invalid skill/SKILL.md": `---
name: invalid skill name
description: Invalid
---
Content`,
			},
			want: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()

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

			p := New(tmpDir, model.ClaudeCode)
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
	p := New("/nonexistent/directory/path", model.ClaudeCode)
	skills, err := p.Parse()
	if err != nil {
		t.Errorf("Parse() on nonexistent directory should not error, got: %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("Parse() on nonexistent directory should return empty slice, got %d skills", len(skills))
	}
}

func TestParser_parseSkillFile_BasicFields(t *testing.T) {
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
description: Minimal desc
---
Content only.`,
			wantName:    "minimal",
			wantDesc:    "Minimal desc",
			wantTools:   nil,
			wantContent: "Content only.",
		},
		"no frontmatter derives name from directory": {
			content: `# No Frontmatter

Just content.`,
			wantName:  "test-skill",
			wantDesc:  "",
			wantTools: nil,
			wantContent: `# No Frontmatter

Just content.`,
		},
		"empty tools array": {
			content: `---
name: empty-tools
description: Test
tools: []
---
Content.`,
			wantName:    "empty-tools",
			wantDesc:    "Test",
			wantTools:   []string{},
			wantContent: "Content.",
		},
		"windows line endings": {
			content:     "---\r\nname: windows\r\ndescription: Windows test\r\n---\r\nContent\r\n",
			wantName:    "windows",
			wantDesc:    "Windows test",
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
description: Test
---
Content`,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			skillDir := filepath.Join(tmpDir, "test-skill")
			// #nosec G301 - test directory permissions
			if err := os.MkdirAll(skillDir, 0o755); err != nil {
				t.Fatalf("failed to create skill directory: %v", err)
			}
			filePath := filepath.Join(skillDir, "SKILL.md")
			// #nosec G306 - test file permissions
			if err := os.WriteFile(filePath, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			p := New(tmpDir, model.ClaudeCode)
			skill, err := p.parseSkillFile(filePath)

			if (err != nil) != tt.wantErr {
				t.Fatalf("parseSkillFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

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

			if skill.ModifiedAt.IsZero() {
				t.Error("ModifiedAt should be set")
			}
			if time.Since(skill.ModifiedAt) > 5*time.Second {
				t.Errorf("ModifiedAt seems too old: %v", skill.ModifiedAt)
			}
		})
	}
}

func TestParser_parseSkillFile_AgentSkillsStandardFields(t *testing.T) {
	tests := map[string]struct {
		content                    string
		wantScope                  model.SkillScope
		wantDisableModelInvocation bool
		wantLicense                string
		wantCompatibility          map[string]string
		wantScripts                []string
		wantReferences             []string
		wantAssets                 []string
	}{
		"all agent skills standard fields": {
			content: `---
name: full-agent-skill
description: Complete Agent Skills Standard example
scope: user
disable-model-invocation: true
license: MIT
compatibility:
  claude-code: ">=1.0.0"
  cursor: ">=0.5.0"
scripts:
  - setup.sh
  - validate.sh
references:
  - docs/guide.md
  - https://example.com/docs
assets:
  - templates/config.yaml
  - data/schema.json
---
Content here.`,
			wantScope:                  model.ScopeUser,
			wantDisableModelInvocation: true,
			wantLicense:                "MIT",
			wantCompatibility: map[string]string{
				"claude-code": ">=1.0.0",
				"cursor":      ">=0.5.0",
			},
			wantScripts:    []string{"setup.sh", "validate.sh"},
			wantReferences: []string{"docs/guide.md", "https://example.com/docs"},
			wantAssets:     []string{"templates/config.yaml", "data/schema.json"},
		},
		"scope repo": {
			content: `---
name: repo-skill
description: Repository scope
scope: repo
---
Content`,
			wantScope: model.ScopeRepo,
		},
		"scope system": {
			content: `---
name: system-skill
description: System scope
scope: system
---
Content`,
			wantScope: model.ScopeSystem,
		},
		"scope with alias": {
			content: `---
name: alias-skill
description: Repository alias
scope: repository
---
Content`,
			wantScope: model.ScopeRepo,
		},
		"disable-model-invocation false by default": {
			content: `---
name: invocable-skill
description: Test
---
Content`,
			wantDisableModelInvocation: false,
		},
		"partial fields": {
			content: `---
name: partial-skill
description: Only some Agent Skills fields
license: Apache-2.0
references:
  - readme.md
---
Content`,
			wantLicense:    "Apache-2.0",
			wantReferences: []string{"readme.md"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			skillDir := filepath.Join(tmpDir, "test-skill")
			// #nosec G301 - test directory permissions
			if err := os.MkdirAll(skillDir, 0o755); err != nil {
				t.Fatalf("failed to create skill directory: %v", err)
			}
			filePath := filepath.Join(skillDir, "SKILL.md")
			// #nosec G306 - test file permissions
			if err := os.WriteFile(filePath, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			p := New(tmpDir, model.ClaudeCode)
			skill, err := p.parseSkillFile(filePath)
			if err != nil {
				t.Fatalf("parseSkillFile() error = %v", err)
			}

			if skill.Scope != tt.wantScope {
				t.Errorf("Scope = %v, want %v", skill.Scope, tt.wantScope)
			}
			if skill.DisableModelInvocation != tt.wantDisableModelInvocation {
				t.Errorf("DisableModelInvocation = %v, want %v", skill.DisableModelInvocation, tt.wantDisableModelInvocation)
			}
			if skill.License != tt.wantLicense {
				t.Errorf("License = %q, want %q", skill.License, tt.wantLicense)
			}

			if tt.wantCompatibility != nil {
				for k, v := range tt.wantCompatibility {
					if skill.Compatibility[k] != v {
						t.Errorf("Compatibility[%q] = %q, want %q", k, skill.Compatibility[k], v)
					}
				}
			}

			if !equalSlices(skill.Scripts, tt.wantScripts) {
				t.Errorf("Scripts = %v, want %v", skill.Scripts, tt.wantScripts)
			}
			if !equalSlices(skill.References, tt.wantReferences) {
				t.Errorf("References = %v, want %v", skill.References, tt.wantReferences)
			}
			if !equalSlices(skill.Assets, tt.wantAssets) {
				t.Errorf("Assets = %v, want %v", skill.Assets, tt.wantAssets)
			}
		})
	}
}

func TestParser_parseSkillFile_SkillDirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "structured-skill")

	// Create skill directory structure
	dirs := []string{
		skillDir,
		filepath.Join(skillDir, "scripts"),
		filepath.Join(skillDir, "references"),
		filepath.Join(skillDir, "assets"),
	}
	for _, dir := range dirs {
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory %q: %v", dir, err)
		}
	}

	// Create SKILL.md
	skillContent := `---
name: structured-skill
description: Skill with directory structure
---
Content here.`
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Create files in subdirectories
	testFiles := map[string]string{
		filepath.Join(skillDir, "scripts", "setup.sh"):     "#!/bin/bash\necho setup",
		filepath.Join(skillDir, "scripts", "validate.sh"):  "#!/bin/bash\necho validate",
		filepath.Join(skillDir, "references", "guide.md"):  "# Guide",
		filepath.Join(skillDir, "assets", "config.yaml"):   "key: value",
		filepath.Join(skillDir, "assets", "template.json"): "{}",
	}
	for path, content := range testFiles {
		// #nosec G306 - test file permissions
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file %q: %v", path, err)
		}
	}

	p := New(tmpDir, model.ClaudeCode)
	skill, err := p.parseSkillFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("parseSkillFile() error = %v", err)
	}

	// Verify scripts were discovered
	if len(skill.Scripts) != 2 {
		t.Errorf("Scripts count = %d, want 2", len(skill.Scripts))
	}

	// Verify references were discovered
	if len(skill.References) != 1 {
		t.Errorf("References count = %d, want 1", len(skill.References))
	}

	// Verify assets were discovered
	if len(skill.Assets) != 2 {
		t.Errorf("Assets count = %d, want 2", len(skill.Assets))
	}
}

func TestParser_parseSkillFile_CombineFrontmatterAndDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "combined-skill")

	// Create directories
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(filepath.Join(skillDir, "scripts"), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Create SKILL.md with scripts defined in frontmatter
	skillContent := `---
name: combined-skill
description: Combines frontmatter and directory scripts
scripts:
  - external-script.sh
---
Content here.`
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Create script in directory
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(skillDir, "scripts", "local.sh"), []byte("#!/bin/bash"), 0o644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	p := New(tmpDir, model.ClaudeCode)
	skill, err := p.parseSkillFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("parseSkillFile() error = %v", err)
	}

	// Should have both frontmatter-defined and directory-discovered scripts
	if len(skill.Scripts) < 2 {
		t.Errorf("Scripts count = %d, want at least 2 (frontmatter + directory)", len(skill.Scripts))
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
	skillDir := filepath.Join(tmpDir, "metadata-skill")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	filePath := filepath.Join(skillDir, "SKILL.md")
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	p := New(tmpDir, model.ClaudeCode)
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

	// Verify that standard fields are NOT in metadata
	standardFields := []string{
		"name", "description", "tools", "scope", "disable-model-invocation",
		"license", "compatibility", "scripts", "references", "assets",
	}
	for _, field := range standardFields {
		if _, ok := skill.Metadata[field]; ok {
			t.Errorf("%q should not be in Metadata", field)
		}
	}
}

func TestParseSkillFile_ConvenienceFunction(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	content := `---
name: test-skill
description: Test description
---
Content`
	filePath := filepath.Join(skillDir, "SKILL.md")
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	skill, err := ParseSkillFile(filePath, model.Cursor)
	if err != nil {
		t.Fatalf("ParseSkillFile() error = %v", err)
	}

	if skill.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", skill.Name, "test-skill")
	}
	if skill.Platform != model.Cursor {
		t.Errorf("Platform = %v, want %v", skill.Platform, model.Cursor)
	}
}

func TestParseSkillContent(t *testing.T) {
	tests := map[string]struct {
		content     []byte
		name        string
		wantName    string
		wantDesc    string
		wantLicense string
		wantErr     bool
	}{
		"basic content": {
			content: []byte(`---
name: content-skill
description: Parsed from content
license: MIT
---
Body content`),
			name:        "fallback-name",
			wantName:    "content-skill",
			wantDesc:    "Parsed from content",
			wantLicense: "MIT",
		},
		"use provided name when not in frontmatter": {
			content: []byte(`---
description: No name in frontmatter
---
Body`),
			name:     "provided-name",
			wantName: "provided-name",
			wantDesc: "No name in frontmatter",
		},
		"empty name fails": {
			content: []byte(`---
description: No name
---
Body`),
			name:    "",
			wantErr: true,
		},
		"invalid name fails": {
			content: []byte(`---
name: invalid name spaces
description: Test
---
Body`),
			name:    "fallback",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			skill, err := ParseSkillContent(tt.content, tt.name, model.ClaudeCode)

			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseSkillContent() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if skill.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", skill.Name, tt.wantName)
			}
			if skill.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", skill.Description, tt.wantDesc)
			}
			if skill.License != tt.wantLicense {
				t.Errorf("License = %q, want %q", skill.License, tt.wantLicense)
			}
		})
	}
}

func TestIsAgentSkillsFormat(t *testing.T) {
	tests := map[string]struct {
		content []byte
		want    bool
	}{
		"valid agent skills format": {
			content: []byte(`---
name: valid-skill
description: Has required fields
---
Content`),
			want: true,
		},
		"missing description": {
			content: []byte(`---
name: incomplete
---
Content`),
			want: false,
		},
		"missing name": {
			content: []byte(`---
description: No name
---
Content`),
			want: false,
		},
		"no frontmatter": {
			content: []byte(`# Just markdown

No frontmatter here.`),
			want: false,
		},
		"invalid yaml": {
			content: []byte(`---
invalid: [yaml
---
Content`),
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := IsAgentSkillsFormat(tt.content); got != tt.want {
				t.Errorf("IsAgentSkillsFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasSkillDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid skill directory
	validDir := filepath.Join(tmpDir, "valid-skill")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(validDir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(validDir, "SKILL.md"), []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Create a directory without SKILL.md
	invalidDir := filepath.Join(tmpDir, "invalid-skill")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(invalidDir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	tests := map[string]struct {
		path string
		want bool
	}{
		"valid skill directory": {path: validDir, want: true},
		"missing SKILL.md":      {path: invalidDir, want: false},
		"nonexistent directory": {path: "/nonexistent/path", want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := HasSkillDirectory(tt.path); got != tt.want {
				t.Errorf("HasSkillDirectory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListSkillDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skill directories
	skillDirs := []string{
		filepath.Join(tmpDir, "skill1"),
		filepath.Join(tmpDir, "category", "skill2"),
	}
	for _, dir := range skillDirs {
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to write SKILL.md: %v", err)
		}
	}

	dirs, err := ListSkillDirectories(tmpDir)
	if err != nil {
		t.Fatalf("ListSkillDirectories() error = %v", err)
	}

	if len(dirs) != 2 {
		t.Errorf("ListSkillDirectories() returned %d dirs, want 2", len(dirs))
	}
}

func TestGetSkillDirectoryContents(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")

	// Create full directory structure
	dirs := []string{
		skillDir,
		filepath.Join(skillDir, "scripts"),
		filepath.Join(skillDir, "references"),
		filepath.Join(skillDir, "assets"),
	}
	for _, dir := range dirs {
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
	}

	// Create files
	files := map[string]string{
		filepath.Join(skillDir, "SKILL.md"):              "content",
		filepath.Join(skillDir, "scripts", "setup.sh"):   "script",
		filepath.Join(skillDir, "references", "doc.md"):  "doc",
		filepath.Join(skillDir, "assets", "config.yaml"): "config",
	}
	for path, content := range files {
		// #nosec G306 - test file permissions
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	contents, err := GetSkillDirectoryContents(skillDir)
	if err != nil {
		t.Fatalf("GetSkillDirectoryContents() error = %v", err)
	}

	if contents.SkillFile != filepath.Join(skillDir, "SKILL.md") {
		t.Errorf("SkillFile = %q, want SKILL.md path", contents.SkillFile)
	}
	if len(contents.Scripts) != 1 {
		t.Errorf("Scripts count = %d, want 1", len(contents.Scripts))
	}
	if len(contents.References) != 1 {
		t.Errorf("References count = %d, want 1", len(contents.References))
	}
	if len(contents.Assets) != 1 {
		t.Errorf("Assets count = %d, want 1", len(contents.Assets))
	}
}

func TestGetSkillDirectoryContents_NoSkillMd(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := GetSkillDirectoryContents(tmpDir)
	if err == nil {
		t.Error("GetSkillDirectoryContents() should error when SKILL.md is missing")
	}
}

func TestNormalizeKey(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
	}{
		"already kebab-case":        {input: "disable-model-invocation", want: "disable-model-invocation"},
		"camelCase":                 {input: "disableModelInvocation", want: "disable-model-invocation"},
		"snake_case":                {input: "disable_model_invocation", want: "disable-model-invocation"},
		"simple word":               {input: "license", want: "license"},
		"camelCase without mapping": {input: "customField", want: "custom-field"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := NormalizeKey(tt.input); got != tt.want {
				t.Errorf("NormalizeKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDeriveNameFromPath(t *testing.T) {
	tests := map[string]struct {
		path string
		want string
	}{
		"simple path": {path: "/skills/my-skill/SKILL.md", want: "my-skill"},
		"nested path": {path: "/category/subcategory/skill-name/SKILL.md", want: "skill-name"},
		"current dir": {path: "SKILL.md", want: "."},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := deriveNameFromPath(tt.path); got != tt.want {
				t.Errorf("deriveNameFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// Helper function to compare slices
func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestCaseInsensitiveSkillMd tests case-insensitive SKILL.md detection
func TestCaseInsensitiveSkillMd(t *testing.T) {
	t.Run("lowercase skill.md is discovered", func(t *testing.T) {
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

		p := New(tmpDir, model.ClaudeCode)
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

	t.Run("HasSkillDirectory detects lowercase skill.md", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillDir := filepath.Join(tmpDir, "test-skill")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		// Create lowercase skill.md
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to write skill.md: %v", err)
		}

		if !HasSkillDirectory(skillDir) {
			t.Error("HasSkillDirectory should return true for lowercase skill.md")
		}
	})

	t.Run("ListSkillDirectories finds lowercase skill.md", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create skill with lowercase
		lowerDir := filepath.Join(tmpDir, "lower-skill")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(lowerDir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(lowerDir, "skill.md"), []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		// Create skill with uppercase
		upperDir := filepath.Join(tmpDir, "upper-skill")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(upperDir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(upperDir, "SKILL.md"), []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		dirs, err := ListSkillDirectories(tmpDir)
		if err != nil {
			t.Fatalf("ListSkillDirectories() error = %v", err)
		}

		if len(dirs) != 2 {
			t.Errorf("ListSkillDirectories() returned %d dirs, want 2", len(dirs))
		}
	})

	t.Run("GetSkillDirectoryContents works with lowercase skill.md", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillDir := filepath.Join(tmpDir, "test-skill")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		// Create lowercase skill.md
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to write skill.md: %v", err)
		}

		contents, err := GetSkillDirectoryContents(skillDir)
		if err != nil {
			t.Fatalf("GetSkillDirectoryContents() error = %v", err)
		}

		// Should find the lowercase file (we check SKILL.md, then skill.md, then Skill.md)
		expectedPath := filepath.Join(skillDir, "skill.md")
		if contents.SkillFile != expectedPath {
			// On case-insensitive filesystems, SKILL.md might match skill.md
			// Check that we at least found a skill file
			if contents.SkillFile == "" {
				t.Error("GetSkillDirectoryContents should find skill.md")
			}
		}
	})
}
