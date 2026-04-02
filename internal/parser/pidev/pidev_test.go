package pidev

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestParser_Parse(t *testing.T) {
	basePath := t.TempDir()
	skillDir := filepath.Join(basePath, "test-skill")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}

	content := `---
name: test-skill
description: test
---
# Test
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}

	p := New(basePath)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("Parse() returned %d skills, want 1", len(skills))
	}
	if skills[0].Platform != model.PiDev {
		t.Errorf("skill platform = %s, want %s", skills[0].Platform, model.PiDev)
	}
}

func TestParser_DefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	p := New("")
	expected := filepath.Join(home, ".pi", "agent", "skills")
	if got := p.DefaultPath(); got != expected {
		t.Errorf("DefaultPath() = %q, want %q", got, expected)
	}
}
