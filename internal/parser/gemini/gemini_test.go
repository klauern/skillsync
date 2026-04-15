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
