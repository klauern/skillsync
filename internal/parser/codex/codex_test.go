package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestNew(t *testing.T) {
	tests := map[string]struct {
		basePath     string
		wantContains string
	}{
		"empty path uses default": {
			basePath:     "",
			wantContains: ".codex/skills",
		},
		"custom path preserved": {
			basePath:     "/custom/path",
			wantContains: "/custom/path",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			p := New(tt.basePath)
			if tt.basePath == "" {
				// For empty path, just verify it contains .codex/skills
				if p.basePath == "" || !containsPathSubstring(p.basePath, tt.wantContains) {
					t.Errorf("New(%q).basePath = %q, want to contain %q", tt.basePath, p.basePath, tt.wantContains)
				}
			} else {
				if p.basePath != tt.basePath {
					t.Errorf("New(%q).basePath = %q, want %q", tt.basePath, p.basePath, tt.basePath)
				}
			}
		})
	}
}

func containsPathSubstring(path, substr string) bool {
	// Check if the path ends with the substring (e.g., ".codex/skills")
	return len(path) >= len(substr) && path[len(path)-len(substr):] == substr
}

func TestParser_Platform(t *testing.T) {
	p := New("")
	if got := p.Platform(); got != model.Codex {
		t.Errorf("Platform() = %v, want %v", got, model.Codex)
	}
}

func TestParser_DefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	p := New("")
	got := p.DefaultPath()
	if !containsPathSubstring(got, ".codex/skills") {
		t.Errorf("DefaultPath() = %q, want to contain .codex/skills", got)
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
		"config.toml with instructions": {
			files: map[string]string{
				"config.toml": `
model = "o4-mini"
instructions = "Always explain your reasoning."
developer_instructions = "Use functional patterns."
`,
			},
			want: 1,
		},
		"config.toml without instructions": {
			files: map[string]string{
				"config.toml": `
model = "o4-mini"
approval_policy = "on-failure"
`,
			},
			want: 0,
		},
		"single AGENTS.md file": {
			files: map[string]string{
				"AGENTS.md": "# Project Guidelines\n\nAlways run tests before committing.",
			},
			want: 1,
		},
		"nested AGENTS.md files": {
			files: map[string]string{
				"AGENTS.md":            "Root instructions",
				"subproject/AGENTS.md": "Subproject instructions",
			},
			want: 2,
		},
		"config.toml and AGENTS.md combined": {
			files: map[string]string{
				"config.toml": `
model = "gpt-4.1"
instructions = "Global instructions"
`,
				"AGENTS.md": "Project instructions",
			},
			want: 2,
		},
		"single flat legacy .md file": {
			files: map[string]string{
				"my-skill.md": `---
name: my-skill
description: A flat legacy skill
---
# My Skill

Content here.`,
			},
			want: 1,
		},
		"flat .md with filename-derived name": {
			files: map[string]string{
				"quick-reference.md": "# Quick Reference\n\nUse this for fast lookup.",
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

func TestParser_parseConfigFile(t *testing.T) {
	tests := map[string]struct {
		config      string
		wantSkill   bool
		wantContent string
		wantMeta    map[string]string
	}{
		"full config with all instructions": {
			config: `
model = "o4-mini"
approval_policy = "on-failure"
sandbox_mode = "workspace-write"
profile = "development"
instructions = "Always explain your reasoning before making changes."
developer_instructions = "Prefer functional programming patterns."
`,
			wantSkill:   true,
			wantContent: "Always explain your reasoning before making changes.\n\nPrefer functional programming patterns.",
			wantMeta: map[string]string{
				"model":           "o4-mini",
				"approval_policy": "on-failure",
				"sandbox_mode":    "workspace-write",
				"profile":         "development",
			},
		},
		"instructions only": {
			config: `
instructions = "Be concise."
`,
			wantSkill:   true,
			wantContent: "Be concise.",
		},
		"developer_instructions only": {
			config: `
developer_instructions = "Use tests."
`,
			wantSkill:   true,
			wantContent: "Use tests.",
		},
		"no instructions": {
			config: `
model = "o4-mini"
`,
			wantSkill: false,
		},
		"config with profiles": {
			config: `
profile = "review"
instructions = "Review carefully."

[profiles.review]
model = "gpt-4.1"
approval_policy = "untrusted"
sandbox_mode = "read-only"
model_reasoning_effort = "high"
`,
			wantSkill:   true,
			wantContent: "Review carefully.",
			wantMeta: map[string]string{
				"profile": "review",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Write config
			// #nosec G306 - test file permissions
			if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(tt.config), 0o644); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			// Parse
			p := New(tmpDir)
			skill, err := p.parseConfigFile()
			if err != nil {
				t.Errorf("parseConfigFile() error = %v", err)
				return
			}

			if tt.wantSkill {
				if skill == nil {
					t.Error("parseConfigFile() returned nil, want skill")
					return
				}

				if skill.Content != tt.wantContent {
					t.Errorf("skill.Content = %q, want %q", skill.Content, tt.wantContent)
				}

				for key, wantVal := range tt.wantMeta {
					if gotVal, ok := skill.Metadata[key]; !ok || gotVal != wantVal {
						t.Errorf("skill.Metadata[%q] = %q, want %q", key, gotVal, wantVal)
					}
				}

				if skill.Name != "codex-config" {
					t.Errorf("skill.Name = %q, want %q", skill.Name, "codex-config")
				}

				if skill.Platform != model.Codex {
					t.Errorf("skill.Platform = %v, want %v", skill.Platform, model.Codex)
				}
			} else {
				if skill != nil {
					t.Errorf("parseConfigFile() returned skill, want nil")
				}
			}
		})
	}
}

func TestParser_parseAgentsFile(t *testing.T) {
	tests := map[string]struct {
		content     string
		path        string // relative to base
		wantName    string
		wantContent string
	}{
		"root AGENTS.md": {
			content:     "# Guidelines\n\nFollow best practices.",
			path:        "AGENTS.md",
			wantName:    "agents",
			wantContent: "# Guidelines\n\nFollow best practices.",
		},
		"nested AGENTS.md": {
			content:     "# Submodule Rules",
			path:        "submodule/AGENTS.md",
			wantName:    "submodule-agents",
			wantContent: "# Submodule Rules",
		},
		"deeply nested AGENTS.md": {
			content:     "# Deep Rules",
			path:        "a/b/c/AGENTS.md",
			wantName:    "c-agents",
			wantContent: "# Deep Rules",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Create file path
			fullPath := filepath.Join(tmpDir, tt.path)
			// #nosec G301 - test directory permissions
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				t.Fatalf("failed to create dirs: %v", err)
			}
			// #nosec G306 - test file permissions
			if err := os.WriteFile(fullPath, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}

			// Parse
			p := New(tmpDir)
			skill, err := p.parseAgentsFile(fullPath)
			if err != nil {
				t.Errorf("parseAgentsFile() error = %v", err)
				return
			}

			if skill.Name != tt.wantName {
				t.Errorf("skill.Name = %q, want %q", skill.Name, tt.wantName)
			}

			if skill.Content != tt.wantContent {
				t.Errorf("skill.Content = %q, want %q", skill.Content, tt.wantContent)
			}

			if skill.Platform != model.Codex {
				t.Errorf("skill.Platform = %v, want %v", skill.Platform, model.Codex)
			}

			if skill.Metadata["type"] != "agents" {
				t.Errorf("skill.Metadata[type] = %q, want %q", skill.Metadata["type"], "agents")
			}
		})
	}
}

func TestParser_Parse_Integration(t *testing.T) {
	// Integration test with realistic Codex configuration
	tmpDir := t.TempDir()

	// Create a realistic .codex directory structure
	files := map[string]string{
		"config.toml": `
model = "o4-mini"
approval_policy = "on-failure"
sandbox_mode = "workspace-write"
instructions = "Always explain your reasoning before making changes."
developer_instructions = "Use functional programming patterns."

[profiles.review]
model = "gpt-4.1"
approval_policy = "untrusted"
`,
		"AGENTS.md": `# Project Guidelines

## Code Style
- Use TypeScript strict mode
- Prefer async/await over callbacks

## Testing
- Run tests before committing
`,
		"modules/api/AGENTS.md": `# API Module Rules

- All endpoints must be documented
- Use proper error handling
`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		// #nosec G301 -- test directory permissions are acceptable
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		// #nosec G306 -- test file permissions are acceptable
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Expect: 1 config skill + 2 AGENTS.md files = 3 skills
	if len(skills) != 3 {
		t.Fatalf("Parse() returned %d skills, want 3", len(skills))
	}

	// Verify config skill
	configSkill := findSkillByName(t, skills, "codex-config")
	if configSkill.Description != "Codex CLI configuration instructions" {
		t.Errorf("config skill description = %q", configSkill.Description)
	}
	if configSkill.Metadata["model"] != "o4-mini" {
		t.Errorf("config skill model = %q, want 'o4-mini'", configSkill.Metadata["model"])
	}

	// Verify root AGENTS.md
	rootAgents := findSkillByName(t, skills, "agents")
	if rootAgents.Metadata["type"] != "agents" {
		t.Errorf("root agents type = %q, want 'agents'", rootAgents.Metadata["type"])
	}

	// Verify nested AGENTS.md
	apiAgents := findSkillByName(t, skills, "api-agents")
	if apiAgents.Metadata["type"] != "agents" {
		t.Errorf("api agents type = %q, want 'agents'", apiAgents.Metadata["type"])
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
	t.Fatalf("skill %q not found in skills: %v", name, skillNames(skills))
	return model.Skill{}
}

func skillNames(skills []model.Skill) []string {
	names := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Name
	}
	return names
}

func TestParser_Parse_SkillMdSupport(t *testing.T) {
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
	if skill.Platform != model.Codex {
		t.Errorf("Platform = %v, want %v", skill.Platform, model.Codex)
	}
}

func TestParser_Parse_SkillMdPrecedenceOverAgents(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an AGENTS.md file that would generate "my-skill-agents" name
	agentsDir := filepath.Join(tmpDir, "my-skill")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatalf("failed to create agents directory: %v", err)
	}

	agentsContent := `# Legacy AGENTS.md
This is legacy content.`
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(agentsDir, "AGENTS.md"), []byte(agentsContent), 0o644); err != nil {
		t.Fatalf("failed to write AGENTS.md: %v", err)
	}

	// Create a SKILL.md version with name that would conflict
	skillMdContent := `---
name: my-skill-agents
description: SKILL.md version
---
SKILL.md content.`
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(agentsDir, "SKILL.md"), []byte(skillMdContent), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Should only have 1 skill (SKILL.md version)
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill (SKILL.md should take precedence), got %d: %v", len(skills), skillNames(skills))
	}

	skill := skills[0]
	if skill.Description != "SKILL.md version" {
		t.Errorf("Expected SKILL.md version to take precedence, got description: %q", skill.Description)
	}
}

func TestParser_Parse_SkillMdPrecedenceOverConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config.toml (produces "codex-config" skill)
	configContent := `
model = "o4-mini"
instructions = "Config instructions"
`
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config.toml: %v", err)
	}

	// Create a SKILL.md version with name "codex-config"
	skillDir := filepath.Join(tmpDir, "codex-config")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	skillMdContent := `---
name: codex-config
description: SKILL.md version of codex-config
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

	// Should only have 1 skill (SKILL.md version takes precedence)
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill (SKILL.md should take precedence), got %d: %v", len(skills), skillNames(skills))
	}

	skill := skills[0]
	if skill.Description != "SKILL.md version of codex-config" {
		t.Errorf("Expected SKILL.md version to take precedence, got description: %q", skill.Description)
	}
}

func TestParser_Parse_MixedSkillMdConfigAgents(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config.toml
	configContent := `
model = "o4-mini"
instructions = "Config instructions"
`
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config.toml: %v", err)
	}

	// Create an AGENTS.md file
	// #nosec G306 - test file permissions
	if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# Root agents"), 0o644); err != nil {
		t.Fatalf("failed to write AGENTS.md: %v", err)
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

	// Should have 3 skills: codex-config, agents, skillmd-only
	if len(skills) != 3 {
		t.Fatalf("expected 3 skills, got %d: %v", len(skills), skillNames(skills))
	}

	// Check all skills are present
	names := make(map[string]bool)
	for _, s := range skills {
		names[s.Name] = true
	}
	if !names["codex-config"] {
		t.Error("missing codex-config skill")
	}
	if !names["agents"] {
		t.Error("missing agents skill")
	}
	if !names["skillmd-only"] {
		t.Error("missing skillmd-only skill")
	}
}

func TestParser_Parse_CodexSkillHumanReadableFrontmatterName(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "agent-development")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	content := `---
name: Agent Development
description: Human-readable Codex name
---
Use this skill for building agents.`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d: %v", len(skills), skillNames(skills))
	}
	if skills[0].Name != "agent-development" {
		t.Fatalf("Name = %q, want %q", skills[0].Name, "agent-development")
	}
	if skills[0].Description != "Human-readable Codex name" {
		t.Fatalf("Description = %q, want %q", skills[0].Description, "Human-readable Codex name")
	}
}

func TestParser_Parse_FlatLegacyMd(t *testing.T) {
	t.Run("flat .md coexists with SKILL.md", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Flat legacy .md at root
		flatContent := `---
name: flat-legacy
description: Flat legacy skill
---
Flat content.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(tmpDir, "flat-legacy.md"), []byte(flatContent), 0o644); err != nil {
			t.Fatalf("failed to write flat .md: %v", err)
		}

		// SKILL.md in subdir
		skillDir := filepath.Join(tmpDir, "my-skill")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("failed to create skill dir: %v", err)
		}
		skillMd := `---
name: my-skill
description: SKILL.md format
---
SKILL content.`
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0o644); err != nil {
			t.Fatalf("failed to write SKILL.md: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if len(skills) != 2 {
			t.Fatalf("expected 2 skills, got %d: %v", len(skills), skillNames(skills))
		}

		names := make(map[string]bool)
		for _, s := range skills {
			names[s.Name] = true
		}
		if !names["flat-legacy"] {
			t.Error("missing flat-legacy skill")
		}
		if !names["my-skill"] {
			t.Error("missing my-skill skill")
		}
	})

	t.Run("flat .md inside skill dir is excluded", func(t *testing.T) {
		tmpDir := t.TempDir()

		skillDir := filepath.Join(tmpDir, "garden")
		// #nosec G301 - test directory permissions
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("failed to create skill dir: %v", err)
		}

		// SKILL.md makes this a skill directory
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: garden\ndescription: Garden\n---\n"), 0o644); err != nil {
			t.Fatalf("failed to write SKILL.md: %v", err)
		}

		// README.md inside skill dir should be excluded (reference file)
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(skillDir, "README.md"), []byte("# Garden\nReference only."), 0o644); err != nil {
			t.Fatalf("failed to write README.md: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		// Only 1 skill (garden from SKILL.md), README.md should not be parsed
		if len(skills) != 1 {
			t.Fatalf("expected 1 skill, got %d: %v", len(skills), skillNames(skills))
		}
		if skills[0].Name != "garden" {
			t.Errorf("skill.Name = %q, want garden", skills[0].Name)
		}
	})

	t.Run("AGENTS.md and flat .md both parsed", func(t *testing.T) {
		tmpDir := t.TempDir()

		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte("# Agents instructions"), 0o644); err != nil {
			t.Fatalf("failed to write AGENTS.md: %v", err)
		}
		// #nosec G306 - test file permissions
		if err := os.WriteFile(filepath.Join(tmpDir, "custom.md"), []byte("---\nname: custom\ndescription: Custom flat\n---\nContent"), 0o644); err != nil {
			t.Fatalf("failed to write custom.md: %v", err)
		}

		p := New(tmpDir)
		skills, err := p.Parse()
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}

		if len(skills) != 2 {
			t.Fatalf("expected 2 skills, got %d: %v", len(skills), skillNames(skills))
		}
		names := make(map[string]bool)
		for _, s := range skills {
			names[s.Name] = true
		}
		if !names["agents"] {
			t.Error("missing agents skill")
		}
		if !names["custom"] {
			t.Error("missing custom skill")
		}
	})
}

func TestParser_parseFlatMdFile(t *testing.T) {
	tests := map[string]struct {
		content     string
		filename    string
		wantName    string
		wantDesc    string
		wantContent string
	}{
		"with frontmatter": {
			content:     "---\nname: test-skill\ndescription: Test description\n---\n# Body\n\nContent.",
			filename:    "arbitrary.md",
			wantName:    "test-skill",
			wantDesc:    "Test description",
			wantContent: "# Body\n\nContent.",
		},
		"filename-derived name": {
			content:     "# No frontmatter\n\nJust content.",
			filename:    "quick-ref.md",
			wantName:    "quick-ref",
			wantDesc:    "",
			wantContent: "# No frontmatter\n\nJust content.",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			fullPath := filepath.Join(tmpDir, tt.filename)
			// #nosec G306 - test file permissions
			if err := os.WriteFile(fullPath, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}

			p := New(tmpDir)
			skill, err := p.parseFlatMdFile(fullPath)
			if err != nil {
				t.Fatalf("parseFlatMdFile() error = %v", err)
			}

			if skill.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", skill.Name, tt.wantName)
			}
			if skill.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", skill.Description, tt.wantDesc)
			}
			if skill.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", skill.Content, tt.wantContent)
			}
			if skill.Platform != model.Codex {
				t.Errorf("Platform = %v, want %v", skill.Platform, model.Codex)
			}
		})
	}
}

func TestParser_Parse_SkillMdAgentSkillsStandard(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a skill directory with SKILL.md containing Agent Skills Standard fields
	skillDir := filepath.Join(tmpDir, "standard-skill")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	skillMd := `---
name: standard-skill
description: Skill with Agent Skills Standard fields
scope: repo
license: MIT
disable-model-invocation: true
tools:
  - Read
  - Write
---
# Standard Skill

This skill has Agent Skills Standard fields.`
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
	if skill.Name != "standard-skill" {
		t.Errorf("Name = %q, want %q", skill.Name, "standard-skill")
	}
	if skill.Scope != model.ScopeRepo {
		t.Errorf("Scope = %q, want %q", skill.Scope, model.ScopeRepo)
	}
	if skill.License != "MIT" {
		t.Errorf("License = %q, want %q", skill.License, "MIT")
	}
	if !skill.DisableModelInvocation {
		t.Error("DisableModelInvocation should be true")
	}
	if len(skill.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(skill.Tools))
	}
}
