package gemini

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestNew(t *testing.T) {
	t.Run("custom path preserved", func(t *testing.T) {
		p := New("/custom/gemini")
		if p.basePath != "/custom/gemini" {
			t.Fatalf("New(custom).basePath = %q", p.basePath)
		}
	})

	t.Run("empty path uses home default", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		p := New("")
		want := filepath.Join(home, ".gemini")
		if p.basePath != want {
			t.Fatalf("New(\"\").basePath = %q, want %q", p.basePath, want)
		}
	})
}

func TestParser_Platform(t *testing.T) {
	p := New("/test")
	if got := p.Platform(); got != model.Platform("gemini") {
		t.Fatalf("Platform() = %q, want gemini", got)
	}
}

func TestParser_DefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	p := New("")
	want := filepath.Join(home, ".gemini")
	if got := p.DefaultPath(); got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestParser_Parse(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		filepath.Join(root, "commands", "deploy.toml"): `description = "Deploy the current service"
prompt = """
Review the staged changes.
!{git diff --staged}

Deploy with {{args}} after checking:
@{docs/release.md}
"""`,
		filepath.Join(root, "commands", "git", "commit.toml"): `prompt = "Create a commit for {{args}}"`,
		filepath.Join(root, "agents", "reviewer.md"): `---
name: reviewer
description: Review code changes for correctness
kind: local
tools: [read_file, grep]
model: gemini-2.5-pro
temperature: 0.2
max_turns: 8
timeout_mins: 12
---
Inspect the diff and call out concrete risks.`,
	}

	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("failed to create %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	p := New(root)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 3 {
		t.Fatalf("Parse() returned %d artifacts, want 3", len(skills))
	}

	byName := make(map[string]model.Skill, len(skills))
	for _, skill := range skills {
		byName[skill.Name] = skill
	}

	deploy := byName["deploy"]
	if deploy.Type != model.SkillTypePrompt {
		t.Fatalf("deploy.Type = %q, want prompt", deploy.Type)
	}
	if deploy.Trigger != "/deploy" {
		t.Fatalf("deploy.Trigger = %q, want /deploy", deploy.Trigger)
	}
	if deploy.Metadata["source_format"] != "toml" {
		t.Fatalf("deploy.Metadata[source_format] = %q, want toml", deploy.Metadata["source_format"])
	}
	if deploy.Metadata["argument_syntax"] != "{{args}},!{shell},@{path}" {
		t.Fatalf("deploy.Metadata[argument_syntax] = %q", deploy.Metadata["argument_syntax"])
	}
	if deploy.Content == "" || deploy.Content[0] == '\n' {
		t.Fatalf("deploy.Content should contain the prompt verbatim")
	}

	commit := byName["git-commit"]
	if commit.Trigger != "/git:commit" {
		t.Fatalf("commit.Trigger = %q, want /git:commit", commit.Trigger)
	}
	if commit.Description != "Gemini custom command" {
		t.Fatalf("commit.Description = %q, want fallback description", commit.Description)
	}

	reviewer := byName["reviewer"]
	if reviewer.Metadata["type"] != "agents" {
		t.Fatalf("reviewer.Metadata[type] = %q, want agents", reviewer.Metadata["type"])
	}
	if reviewer.Metadata["kind"] != "local" {
		t.Fatalf("reviewer.Metadata[kind] = %q, want local", reviewer.Metadata["kind"])
	}
	if reviewer.Metadata["tools"] != "[read_file grep]" {
		t.Fatalf("reviewer.Metadata[tools] = %q", reviewer.Metadata["tools"])
	}
	if reviewer.Metadata["max_turns"] != "8" {
		t.Fatalf("reviewer.Metadata[max_turns] = %q, want 8", reviewer.Metadata["max_turns"])
	}
}

func TestParser_parseCommandFileRequiresPrompt(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "commands", "broken.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create commands dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`description = "missing prompt"`), 0o644); err != nil {
		t.Fatalf("failed to write command file: %v", err)
	}

	p := New(root)
	if _, err := p.parseCommandFile(path); err == nil {
		t.Fatal("parseCommandFile() error = nil, want error")
	}
}

func TestParser_parseAgentFileRequiresFrontmatterFields(t *testing.T) {
	tests := map[string]string{
		"missing frontmatter": "Agent body only.",
		"missing name": `---
description: Missing name
---
Body.`,
		"missing description": `---
name: reviewer
---
Body.`,
	}

	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "agents", "reviewer.md")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("failed to create agents dir: %v", err)
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatalf("failed to write agent file: %v", err)
			}

			p := New(root)
			if _, err := p.parseAgentFile(path); err == nil {
				t.Fatal("parseAgentFile() error = nil, want error")
			}
		})
	}
}

func TestCommandIdentity(t *testing.T) {
	root := "/tmp/project/.gemini/commands"

	name, trigger, err := commandIdentity(filepath.Join(root, "review.toml"), root)
	if err != nil {
		t.Fatalf("commandIdentity() error = %v", err)
	}
	if name != "review" || trigger != "/review" {
		t.Fatalf("commandIdentity() = (%q, %q)", name, trigger)
	}

	name, trigger, err = commandIdentity(filepath.Join(root, "git", "commit.toml"), root)
	if err != nil {
		t.Fatalf("commandIdentity() error = %v", err)
	}
	if name != "git-commit" || trigger != "/git:commit" {
		t.Fatalf("commandIdentity() = (%q, %q)", name, trigger)
	}
}
