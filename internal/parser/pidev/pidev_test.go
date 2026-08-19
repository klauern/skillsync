package pidev

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
	}{
		{name: "custom path preserved", basePath: "/custom/path"},
		{name: "empty path uses default", basePath: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.basePath)
			if tt.basePath != "" && p.basePath != tt.basePath {
				t.Fatalf("New(%q).basePath = %q, want %q", tt.basePath, p.basePath, tt.basePath)
			}
			if tt.basePath == "" && p.basePath == "" {
				t.Fatal("New(\"\") returned empty basePath")
			}
		})
	}
}

func TestParser_Platform(t *testing.T) {
	p := New("/test")
	if got := p.Platform(); got != model.PiDev {
		t.Fatalf("Platform() = %v, want %v", got, model.PiDev)
	}
}

func TestParser_DefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".pi", "agent", "skills"), 0o750); err != nil {
		t.Fatalf("failed to create default skills path: %v", err)
	}

	p := New("")
	got := p.DefaultPath()
	want := filepath.Join(home, ".pi", "agent", "skills")
	if got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestParser_Parse(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create a repo root so hierarchical AGENTS.md discovery can walk from
	// repo root down to the current working directory.
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o750); err != nil {
		t.Fatalf("failed to create git dir: %v", err)
	}

	// Preferred Pi.dev project root.
	projectRoot := filepath.Join(root, ".agents")
	if err := os.MkdirAll(filepath.Join(projectRoot, "skills", "portable-skill"), 0o750); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "prompts"), 0o750); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	files := map[string]string{
		filepath.Join(projectRoot, "skills", "portable-skill", "SKILL.md"): `---
name: portable-skill
description: A portable Pi.dev skill
---
Skill body.
`,
		filepath.Join(projectRoot, "prompts", "review.md"): `---
name: review
description: Review prompt
argument-hint: "file to review"
category: triage
---
Prompt body.
`,
		filepath.Join(root, "AGENTS.md"):               "# Repo rules\n\nAlways run tests.",
		filepath.Join(root, "docs", "AGENTS.md"):       "# Docs rules\n\nKeep docs current.",
		filepath.Join(projectRoot, "SYSTEM.md"):        "System prompt replacement.",
		filepath.Join(projectRoot, "APPEND_SYSTEM.md"): "Appended system prompt guidance.",
	}
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("failed to create directory for %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	if err := os.Chdir(filepath.Join(root, "docs")); err != nil {
		t.Fatalf("failed to change cwd: %v", err)
	}

	p := New(filepath.Join(projectRoot, "skills"))
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 6 {
		t.Fatalf("Parse() returned %d skills, want 6", len(skills))
	}

	byName := make(map[string]model.Skill)
	for _, skill := range skills {
		byName[skill.Name] = skill
	}

	portable := byName["portable-skill"]
	if portable.Type != model.SkillTypeSkill {
		t.Fatalf("skill type = %q, want skill", portable.Type)
	}
	if portable.Platform != model.PiDev {
		t.Fatalf("skill platform = %q, want %q", portable.Platform, model.PiDev)
	}

	review := byName["review"]
	if review.Type != model.SkillTypePrompt {
		t.Fatalf("prompt type = %q, want prompt", review.Type)
	}
	if review.Trigger != "/review" {
		t.Fatalf("prompt trigger = %q, want /review", review.Trigger)
	}
	if review.Metadata["argument-hint"] != "file to review" {
		t.Fatalf("prompt metadata argument-hint = %q, want %q", review.Metadata["argument-hint"], "file to review")
	}
	if review.Metadata["category"] != "triage" {
		t.Fatalf("prompt metadata category = %q, want %q", review.Metadata["category"], "triage")
	}

	agents := byName["agents"]
	if agents.Metadata["type"] != "agents" {
		t.Fatalf("root AGENTS metadata type = %q, want agents", agents.Metadata["type"])
	}
	if agents.Description != "Pi AGENTS.md instructions" {
		t.Fatalf("root AGENTS description = %q", agents.Description)
	}

	docsAgents := byName["docs-agents"]
	if docsAgents.Metadata["type"] != "agents" {
		t.Fatalf("nested AGENTS metadata type = %q, want agents", docsAgents.Metadata["type"])
	}

	system := byName["system"]
	if system.Metadata["type"] != "system-prompt" {
		t.Fatalf("system metadata type = %q, want system-prompt", system.Metadata["type"])
	}
	if system.Metadata["mode"] != "replace" {
		t.Fatalf("system metadata mode = %q, want replace", system.Metadata["mode"])
	}

	appendSystem := byName["append-system"]
	if appendSystem.Metadata["type"] != "system-prompt" {
		t.Fatalf("append system metadata type = %q, want system-prompt", appendSystem.Metadata["type"])
	}
	if appendSystem.Metadata["mode"] != "append" {
		t.Fatalf("append system metadata mode = %q, want append", appendSystem.Metadata["mode"])
	}
}

func TestInstructionName(t *testing.T) {
	if got := instructionName("/repo/AGENTS.md", "/repo"); got != "agents" {
		t.Fatalf("instructionName() = %q, want agents", got)
	}
	if got := instructionName("/repo/docs/AGENTS.md", "/repo"); got != "docs-agents" {
		t.Fatalf("instructionName() = %q, want docs-agents", got)
	}
}

func TestParser_Parse_NonexistentDirectory(t *testing.T) {
	p := New("/nonexistent/directory/path")
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() on nonexistent directory returned error: %v", err)
	}
	if len(skills) != 0 {
		t.Fatalf("Parse() on nonexistent directory returned %d skills, want 0", len(skills))
	}
}

func TestParser_SettingsSkillsAndInstructionVariants(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, ".pi")
	configured := filepath.Join(root, "configured-skills", "extra")
	if err := os.MkdirAll(configured, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(config, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(config, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config, "settings.json"), []byte(`{"skills":["../configured-skills/extra"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configured, "SKILL.md"), []byte("---\nname: configured\ndescription: configured\n---\nbody\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("claude instructions\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.override.md"), []byte("override instructions\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p := New(filepath.Join(config, "skills"))
	got, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	var configuredFound, claudeFound, overrideFound bool
	for _, skill := range got {
		configuredFound = configuredFound || skill.Name == "configured"
		claudeFound = claudeFound || skill.Name == "agents-claude"
		overrideFound = overrideFound || skill.Name == "agents-agents-override"
	}
	if !configuredFound {
		t.Fatal("settings skills entry was not discovered")
	}
	if claudeFound || !overrideFound {
		names := make([]string, 0, len(got))
		for _, skill := range got {
			names = append(names, skill.Name)
		}
		t.Fatalf("instruction variants not discovered: claude=%v override=%v names=%v", claudeFound, overrideFound, names)
	}
	if got := []string{got[0].Name, got[1].Name, got[2].Name}; got[0] != "configured" || got[1] != "agents-agents-override" || got[2] != "agents" {
		t.Fatalf("unexpected precedence order: %v", got)
	}
}
