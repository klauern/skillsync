package piagent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestParser_Parse(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "example")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}

	content := `---
name: example
description: Example Pi skill
---

# Example
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write skill: %v", err)
	}

	p := New(tmpDir)
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("Parse() returned %d skills, want 1", len(skills))
	}
	if skills[0].Platform != model.PiAgent {
		t.Fatalf("skill platform = %q, want %q", skills[0].Platform, model.PiAgent)
	}
}

func TestDiscoverSearchPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repoRoot := filepath.Join(t.TempDir(), "repo")
	workingDir := filepath.Join(repoRoot, "nested", "deeper")
	if err := os.MkdirAll(workingDir, 0o750); err != nil {
		t.Fatalf("failed to create working dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o750); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}

	repoCurrent := filepath.Join(workingDir, ".agents", "skills")
	repoParent := filepath.Join(repoRoot, "nested", ".agents", "skills")
	repoRootSkills := filepath.Join(repoRoot, ".agents", "skills")
	projectRelative := filepath.Join(repoRoot, ".pi", "project-relative-skills")
	projectAbsolute := filepath.Join(repoRoot, "project-absolute-skills")
	userSkills := filepath.Join(home, ".agents", "skills")
	userRelative := filepath.Join(home, ".config", "pi", "user-relative-skills")
	userAbsolute := filepath.Join(home, "user-absolute-skills")

	for _, dir := range []string{
		repoCurrent,
		repoParent,
		repoRootSkills,
		projectRelative,
		projectAbsolute,
		userSkills,
		userRelative,
		userAbsolute,
	} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}

	projectSettings := filepath.Join(repoRoot, ".pi", "settings.json")
	if err := os.MkdirAll(filepath.Dir(projectSettings), 0o750); err != nil {
		t.Fatalf("failed to create project settings dir: %v", err)
	}
	projectJSON := `{"skillsDirectories":["project-relative-skills","` + filepath.ToSlash(projectAbsolute) + `","project-relative-skills"]}`
	if err := os.WriteFile(projectSettings, []byte(projectJSON), 0o600); err != nil {
		t.Fatalf("failed to write project settings: %v", err)
	}

	userSettings := filepath.Join(home, ".config", "pi", "settings.json")
	if err := os.MkdirAll(filepath.Dir(userSettings), 0o750); err != nil {
		t.Fatalf("failed to create user settings dir: %v", err)
	}
	userJSON := `{"skillsDirectories":["user-relative-skills","` + filepath.ToSlash(userAbsolute) + `"]}`
	if err := os.WriteFile(userSettings, []byte(userJSON), 0o600); err != nil {
		t.Fatalf("failed to write user settings: %v", err)
	}

	paths, err := DiscoverSearchPaths(workingDir)
	if err != nil {
		t.Fatalf("DiscoverSearchPaths() error = %v", err)
	}

	gotPaths := make([]string, len(paths))
	gotScopes := make([]model.SkillScope, len(paths))
	for i, path := range paths {
		gotPaths[i] = path.Path
		gotScopes[i] = path.Scope
	}

	wantPaths := []string{
		repoCurrent,
		repoParent,
		repoRootSkills,
		projectRelative,
		projectAbsolute,
		userSkills,
		userRelative,
		userAbsolute,
	}
	if len(gotPaths) != len(wantPaths) {
		t.Fatalf("DiscoverSearchPaths() returned %d paths, want %d: %v", len(gotPaths), len(wantPaths), gotPaths)
	}
	for i, want := range wantPaths {
		if gotPaths[i] != want {
			t.Fatalf("path %d = %q, want %q", i, gotPaths[i], want)
		}
	}

	for i := range 5 {
		if gotScopes[i] != model.ScopeRepo {
			t.Fatalf("scope %d = %q, want repo", i, gotScopes[i])
		}
	}
	for i := 5; i < len(gotScopes); i++ {
		if gotScopes[i] != model.ScopeUser {
			t.Fatalf("scope %d = %q, want user", i, gotScopes[i])
		}
	}
}
