package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/model"
)

// testMkdirAll is a test helper that creates directories with test-appropriate permissions.
func testMkdirAll(t *testing.T, path string) {
	t.Helper()
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %q: %v", path, err)
	}
}

// testWriteFile is a test helper that writes files with test-appropriate permissions.
func testWriteFile(t *testing.T, path string, content []byte) {
	t.Helper()
	// #nosec G306 - test file permissions
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write file %q: %v", path, err)
	}
}

func TestNew(t *testing.T) {
	tests := map[string]struct {
		basePath string
		wantNot  string
	}{
		"empty path uses default": {
			basePath: "",
			wantNot:  "",
		},
		"custom path preserved": {
			basePath: "/custom/path",
			wantNot:  "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p := New(tt.basePath)
			if tt.basePath != "" && p.basePath != tt.basePath {
				t.Errorf("New(%q).basePath = %q, want %q", tt.basePath, p.basePath, tt.basePath)
			}
			if tt.basePath == "" && p.basePath == "" {
				t.Error("New(\"\").basePath should not be empty")
			}
		})
	}
}

func TestNewWithRepo(t *testing.T) {
	repoURL := "https://github.com/klauern/skills"
	p := NewWithRepo(repoURL)
	if p.repoURL != repoURL {
		t.Errorf("NewWithRepo(%q).repoURL = %q, want %q", repoURL, p.repoURL, repoURL)
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
	got := p.DefaultPath()
	if got == "" {
		t.Error("DefaultPath() should not be empty")
	}
}

func TestParser_RepoURL(t *testing.T) {
	tests := map[string]struct {
		parser  *Parser
		wantURL string
	}{
		"no repo URL": {
			parser:  New(""),
			wantURL: "",
		},
		"with repo URL": {
			parser:  NewWithRepo("https://github.com/example/repo"),
			wantURL: "https://github.com/example/repo",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := tt.parser.RepoURL(); got != tt.wantURL {
				t.Errorf("RepoURL() = %q, want %q", got, tt.wantURL)
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

func TestParser_Parse_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Errorf("Parse() on empty directory should not error, got: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("Parse() on empty directory should return empty slice, got %d skills", len(skills))
	}
}

func TestParser_Parse_MarketplaceStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create marketplace.json
	marketplace := MarketplaceManifest{
		Name: "test-skills",
		Plugins: []Ref{
			{
				Name:        "test-plugin",
				Description: "A test plugin",
				Source:      "./plugins/test-plugin",
			},
		},
	}
	marketplace.Metadata.Description = "Test marketplace"
	marketplace.Metadata.Version = "1.0.0"

	marketplaceDir := filepath.Join(tmpDir, ".claude-plugin")
	testMkdirAll(t, marketplaceDir)

	marketplaceData, _ := json.Marshal(marketplace)
	marketplacePath := filepath.Join(marketplaceDir, "marketplace.json")
	testWriteFile(t, marketplacePath, marketplaceData)

	// Create plugin structure
	pluginDir := filepath.Join(tmpDir, "plugins", "test-plugin")
	pluginManifestDir := filepath.Join(pluginDir, ".claude-plugin")
	testMkdirAll(t, pluginManifestDir)

	// Create plugin.json
	pluginManifest := Manifest{
		Name:        "test-plugin",
		Description: "A test plugin",
		Version:     "1.0.0",
	}
	pluginManifest.Author.Name = "Test Author"
	pluginData, _ := json.Marshal(pluginManifest)
	pluginManifestPath := filepath.Join(pluginManifestDir, "plugin.json")
	testWriteFile(t, pluginManifestPath, pluginData)

	// Create skill directory with SKILL.md
	skillDir := filepath.Join(pluginDir, "test-skill")
	testMkdirAll(t, skillDir)

	skillContent := `---
name: test-skill
description: A test skill for testing
tools: [Read, Write]
---
# Test Skill

This is a test skill.`
	skillPath := filepath.Join(skillDir, "SKILL.md")
	testWriteFile(t, skillPath, []byte(skillContent))

	// Parse skills
	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("Parse() returned %d skills, want 1", len(skills))
	}

	skill := skills[0]
	if skill.Name != "test-skill" {
		t.Errorf("skill.Name = %q, want %q", skill.Name, "test-skill")
	}
	if skill.Description != "A test skill for testing" {
		t.Errorf("skill.Description = %q, want %q", skill.Description, "A test skill for testing")
	}
	if len(skill.Tools) != 2 {
		t.Errorf("skill.Tools = %v, want [Read Write]", skill.Tools)
	}
	if skill.Platform != model.ClaudeCode {
		t.Errorf("skill.Platform = %v, want %v", skill.Platform, model.ClaudeCode)
	}

	// Check metadata
	if skill.Metadata["plugin"] != "test-plugin" {
		t.Errorf("skill.Metadata[plugin] = %q, want %q", skill.Metadata["plugin"], "test-plugin")
	}
	if skill.Metadata["repository"] != "test-skills" {
		t.Errorf("skill.Metadata[repository] = %q, want %q", skill.Metadata["repository"], "test-skills")
	}
	if skill.Metadata["source"] != "plugin" {
		t.Errorf("skill.Metadata[source] = %q, want %q", skill.Metadata["source"], "plugin")
	}
}

func TestParser_Parse_MultiplePlugins(t *testing.T) {
	tmpDir := t.TempDir()

	// Create marketplace.json with multiple plugins
	marketplace := MarketplaceManifest{
		Name: "multi-plugin-repo",
		Plugins: []Ref{
			{Name: "plugin1", Source: "./plugins/plugin1"},
			{Name: "plugin2", Source: "./plugins/plugin2"},
		},
	}

	marketplaceDir := filepath.Join(tmpDir, ".claude-plugin")
	testMkdirAll(t, marketplaceDir)

	marketplaceData, _ := json.Marshal(marketplace)
	marketplacePath := filepath.Join(marketplaceDir, "marketplace.json")
	testWriteFile(t, marketplacePath, marketplaceData)

	// Create plugin directories with skills
	for i, pluginName := range []string{"plugin1", "plugin2"} {
		pluginDir := filepath.Join(tmpDir, "plugins", pluginName)
		skillDir := filepath.Join(pluginDir, "skill")
		testMkdirAll(t, skillDir)

		skillContent := "---\nname: skill" + string(rune('1'+i)) + "\n---\nContent"
		skillPath := filepath.Join(skillDir, "SKILL.md")
		testWriteFile(t, skillPath, []byte(skillContent))
	}

	// Parse skills
	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("Parse() returned %d skills, want 2", len(skills))
	}
}

func TestParser_Parse_ScanForPlugins(t *testing.T) {
	// Test fallback scanning when no marketplace.json exists
	tmpDir := t.TempDir()

	// Create a plugin structure without marketplace.json
	pluginDir := filepath.Join(tmpDir, "my-plugin")
	pluginManifestDir := filepath.Join(pluginDir, ".claude-plugin")
	testMkdirAll(t, pluginManifestDir)

	// Create plugin.json
	pluginManifest := Manifest{
		Name:        "my-plugin",
		Description: "A standalone plugin",
		Version:     "2.0.0",
	}
	pluginData, _ := json.Marshal(pluginManifest)
	pluginManifestPath := filepath.Join(pluginManifestDir, "plugin.json")
	testWriteFile(t, pluginManifestPath, pluginData)

	// Create skill
	skillDir := filepath.Join(pluginDir, "my-skill")
	testMkdirAll(t, skillDir)

	skillContent := `---
name: standalone-skill
description: A standalone skill
---
Content`
	skillPath := filepath.Join(skillDir, "SKILL.md")
	testWriteFile(t, skillPath, []byte(skillContent))

	// Parse skills
	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("Parse() returned %d skills, want 1", len(skills))
	}

	if skills[0].Name != "standalone-skill" {
		t.Errorf("skill.Name = %q, want %q", skills[0].Name, "standalone-skill")
	}
}

func TestParser_parseSkillFile(t *testing.T) {
	tests := map[string]struct {
		dirName     string
		content     string
		wantName    string
		wantDesc    string
		wantTools   []string
		wantContent string
		wantErr     bool
	}{
		"full frontmatter": {
			dirName: "full-skill",
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
			dirName: "minimal",
			content: `---
name: minimal
---
Content only.`,
			wantName:    "minimal",
			wantDesc:    "",
			wantTools:   nil,
			wantContent: "Content only.",
		},
		"no frontmatter uses dirname": {
			dirName: "dirname-skill",
			content: `# No Frontmatter

Just content.`,
			wantName:  "dirname-skill",
			wantDesc:  "",
			wantTools: nil,
			wantContent: `# No Frontmatter

Just content.`,
		},
		"invalid skill name": {
			dirName: "valid-dir",
			content: `---
name: invalid name with spaces
---
Content`,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temporary directory structure
			tmpDir := t.TempDir()
			skillDir := filepath.Join(tmpDir, tt.dirName)
			testMkdirAll(t, skillDir)

			filePath := filepath.Join(skillDir, "SKILL.md")
			testWriteFile(t, filePath, []byte(tt.content))

			// Parse the file
			p := New(tmpDir)
			skill, err := p.parseSkillFile(filePath, nil, "")

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

			// Verify ModifiedAt is set
			if skill.ModifiedAt.IsZero() {
				t.Error("ModifiedAt should be set")
			}
			if time.Since(skill.ModifiedAt) > 5*time.Second {
				t.Errorf("ModifiedAt seems too old: %v", skill.ModifiedAt)
			}
		})
	}
}

func TestParser_parseSkillFile_WithManifest(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	testMkdirAll(t, skillDir)

	skillContent := `---
name: test-skill
description: Test skill
---
Content`
	skillPath := filepath.Join(skillDir, "SKILL.md")
	testWriteFile(t, skillPath, []byte(skillContent))

	// Create plugin manifest
	pluginManifest := &Manifest{
		Name:        "my-plugin",
		Description: "My plugin",
		Version:     "1.2.3",
	}
	pluginManifest.Author.Name = "Test Author"

	p := New(tmpDir)
	skill, err := p.parseSkillFile(skillPath, pluginManifest, "my-repo")
	if err != nil {
		t.Fatalf("parseSkillFile() error = %v", err)
	}

	// Check metadata from plugin manifest
	if skill.Metadata["plugin"] != "my-plugin" {
		t.Errorf("Metadata[plugin] = %q, want %q", skill.Metadata["plugin"], "my-plugin")
	}
	if skill.Metadata["plugin_version"] != "1.2.3" {
		t.Errorf("Metadata[plugin_version] = %q, want %q", skill.Metadata["plugin_version"], "1.2.3")
	}
	if skill.Metadata["author"] != "Test Author" {
		t.Errorf("Metadata[author] = %q, want %q", skill.Metadata["author"], "Test Author")
	}
	if skill.Metadata["repository"] != "my-repo" {
		t.Errorf("Metadata[repository] = %q, want %q", skill.Metadata["repository"], "my-repo")
	}
	if skill.Metadata["source"] != "plugin" {
		t.Errorf("Metadata[source] = %q, want %q", skill.Metadata["source"], "plugin")
	}
}

func TestDeriveRepoName(t *testing.T) {
	tests := map[string]struct {
		url  string
		want string
	}{
		"https URL": {
			url:  "https://github.com/klauern/skills",
			want: "klauern-skills",
		},
		"https URL with .git": {
			url:  "https://github.com/klauern/skills.git",
			want: "klauern-skills",
		},
		"ssh URL": {
			url:  "git@github.com:klauern/skills.git",
			want: "klauern-skills",
		},
		"ssh URL without .git": {
			url:  "git@github.com:klauern/skills",
			want: "klauern-skills",
		},
		"simple name": {
			url:  "skills",
			want: "skills",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := deriveRepoName(tt.url)
			if got != tt.want {
				t.Errorf("deriveRepoName(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
