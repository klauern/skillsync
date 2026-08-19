package gemini

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestNew(t *testing.T) {
	p := New("")
	if p.basePath == "" {
		t.Fatal("expected default Gemini base path")
	}
}

func TestParser_Platform(t *testing.T) {
	p := New("")
	if got := p.Platform(); got != model.Gemini {
		t.Fatalf("Platform() = %v, want %v", got, model.Gemini)
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
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "skills", "review"), 0o755); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "skills", "review", "SKILL.md"), []byte(`---
name: review
description: Review code
---
# Review
`), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "GEMINI.md"), []byte("# Project Gemini Rules\n\nAlways explain tradeoffs."), 0o644); err != nil {
		t.Fatalf("failed to write GEMINI.md: %v", err)
	}

	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("Parse() returned %d skills, want 2", len(skills))
	}

	byName := make(map[string]model.Skill, len(skills))
	for _, skill := range skills {
		byName[skill.Name] = skill
	}

	if _, ok := byName["review"]; !ok {
		t.Fatal("expected review skill to be discovered")
	}
	contextSkill, ok := byName["gemini-md"]
	if !ok {
		t.Fatal("expected GEMINI.md skill to be discovered")
	}
	if contextSkill.Description != "Gemini CLI GEMINI.md instructions" {
		t.Fatalf("unexpected GEMINI.md description: %q", contextSkill.Description)
	}
}

func TestParser_Parse_SharedAgentsCompatibilityRoot(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".gemini", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	shared := filepath.Join(tmpDir, ".agents", "skills", "shared")
	if err := os.MkdirAll(shared, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shared, "SKILL.md"), []byte("---\nname: shared\ndescription: shared\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := New(filepath.Join(tmpDir, ".gemini")).Parse()
	if err != nil {
		t.Fatal(err)
	}
	for _, skill := range got {
		if skill.Name == "shared" {
			return
		}
	}
	t.Fatal("shared .agents skill was not discovered")
}

func TestParser_Parse_ContextOnly(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "GEMINI.md"), []byte("Workspace rules"), 0o644); err != nil {
		t.Fatalf("failed to write GEMINI.md: %v", err)
	}

	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("Parse() returned %d skills, want 1", len(skills))
	}
	if skills[0].Name != "gemini-md" {
		t.Fatalf("Parse() first skill = %q, want gemini-md", skills[0].Name)
	}
}

func TestParser_Parse_WithSkillsRootInput(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "skills")
	if err := os.MkdirAll(filepath.Join(skillsDir, "build"), 0o755); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "build", "SKILL.md"), []byte("# Build"), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "GEMINI.md"), []byte("Instructions"), 0o644); err != nil {
		t.Fatalf("failed to write GEMINI.md: %v", err)
	}

	p := New(skillsDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("Parse() returned %d skills, want 2", len(skills))
	}
}

func TestParser_Parse_TOMLCommands(t *testing.T) {
	tmpDir := t.TempDir()
	commandsDir := filepath.Join(tmpDir, "commands")
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		t.Fatalf("failed to create commands dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(commandsDir, "review.toml"), []byte(`
description = "Review code for quality"
prompt = "Please review the following code: {{args}}"
args = "<code>"
`), 0o644); err != nil {
		t.Fatalf("failed to write review.toml: %v", err)
	}

	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("Parse() returned %d skills, want 1", len(skills))
	}
	cmd := skills[0]
	if cmd.Name != "review" {
		t.Errorf("Name = %q, want %q", cmd.Name, "review")
	}
	if cmd.Trigger != "/review" {
		t.Errorf("Trigger = %q, want %q", cmd.Trigger, "/review")
	}
	if cmd.Type != model.SkillTypePrompt {
		t.Errorf("Type = %v, want SkillTypePrompt", cmd.Type)
	}
	if cmd.Metadata["type"] != "command" {
		t.Errorf("Metadata[type] = %q, want %q", cmd.Metadata["type"], "command")
	}
}

func TestParser_Parse_TOMLCommands_MalformedTOML(t *testing.T) {
	tmpDir := t.TempDir()
	commandsDir := filepath.Join(tmpDir, "commands")
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		t.Fatalf("failed to create commands dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(commandsDir, "bad.toml"), []byte("not valid toml [[["), 0o644); err != nil {
		t.Fatalf("failed to write bad.toml: %v", err)
	}
	// Good command alongside the bad one
	if err := os.WriteFile(filepath.Join(commandsDir, "good.toml"), []byte(`
description = "Good command"
prompt = "Do something good"
`), 0o644); err != nil {
		t.Fatalf("failed to write good.toml: %v", err)
	}

	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	// Malformed file is skipped; good one is parsed
	if len(skills) != 1 {
		t.Fatalf("Parse() returned %d skills, want 1 (malformed TOML should be skipped)", len(skills))
	}
	if skills[0].Name != "good" {
		t.Errorf("expected good command, got %q", skills[0].Name)
	}
}

func TestParser_Parse_TOMLAndSkillsMixed(t *testing.T) {
	tmpDir := t.TempDir()
	// Skill file
	skillsDir := filepath.Join(tmpDir, "skills", "refactor")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("# Refactor"), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}
	// TOML command
	commandsDir := filepath.Join(tmpDir, "commands")
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		t.Fatalf("failed to create commands dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(commandsDir, "explain.toml"), []byte(`
description = "Explain code"
prompt = "Explain this: {{args}}"
`), 0o644); err != nil {
		t.Fatalf("failed to write explain.toml: %v", err)
	}

	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("Parse() returned %d skills, want 2 (1 skill + 1 command)", len(skills))
	}
	byName := make(map[string]model.Skill, len(skills))
	for _, s := range skills {
		byName[s.Name] = s
	}
	if _, ok := byName["refactor"]; !ok {
		t.Error("expected refactor skill")
	}
	if _, ok := byName["explain"]; !ok {
		t.Error("expected explain command")
	}
}
