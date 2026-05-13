package tiered

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/parser/mock"
	"github.com/klauern/skillsync/internal/util"
)

func TestGetSearchPaths_WithExplicitPaths(t *testing.T) {
	explicit := []util.ScopedPath{
		{Path: "/some/user/path", Scope: model.ScopeUser},
		{Path: "/some/repo/path", Scope: model.ScopeRepo},
	}
	p := New(Config{
		Platform:      model.ClaudeCode,
		SearchPaths:   explicit,
		ParserFactory: func(_ string) parser.Parser { return mock.New(model.ClaudeCode) },
	})

	got := p.GetSearchPaths()
	if len(got) != len(explicit) {
		t.Fatalf("GetSearchPaths() = %d paths, want %d", len(got), len(explicit))
	}
	for i, want := range explicit {
		if got[i] != want {
			t.Errorf("GetSearchPaths()[%d] = %+v, want %+v", i, got[i], want)
		}
	}
}

func TestParseFromScope_WithSearchPaths_Match(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "skills")
	// #nosec G301
	if err := os.MkdirAll(skillsDir, 0o750); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}

	wantSkill := model.Skill{Name: "my-skill", Description: "found via search paths"}
	m := mock.New(model.ClaudeCode)
	m.WithSkills([]model.Skill{wantSkill})

	p := New(Config{
		Platform: model.ClaudeCode,
		SearchPaths: []util.ScopedPath{
			{Path: skillsDir, Scope: model.ScopeUser},
		},
		ParserFactory: func(_ string) parser.Parser { return m },
	})

	skills, err := p.ParseFromScope(model.ScopeUser)
	if err != nil {
		t.Fatalf("ParseFromScope() unexpected error: %v", err)
	}
	if len(skills) != 1 || skills[0].Name != wantSkill.Name {
		t.Errorf("ParseFromScope() = %v, want skill %q", skills, wantSkill.Name)
	}
	if skills[0].Scope != model.ScopeUser {
		t.Errorf("skill.Scope = %v, want %v", skills[0].Scope, model.ScopeUser)
	}
}

func TestParseFromScope_WithSearchPaths_NoMatch(t *testing.T) {
	p := New(Config{
		Platform: model.ClaudeCode,
		SearchPaths: []util.ScopedPath{
			{Path: "/some/user/path", Scope: model.ScopeUser},
		},
		ParserFactory: func(_ string) parser.Parser { return mock.New(model.ClaudeCode) },
	})

	skills, err := p.ParseFromScope(model.ScopeRepo)
	if err != nil {
		t.Fatalf("ParseFromScope() unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("ParseFromScope() = %d skills, want 0 when no path matches scope", len(skills))
	}
}

func TestParseFromScope_WithSearchPaths_NonexistentPath(t *testing.T) {
	p := New(Config{
		Platform: model.ClaudeCode,
		SearchPaths: []util.ScopedPath{
			{Path: "/does/not/exist/at/all", Scope: model.ScopeUser},
		},
		ParserFactory: func(_ string) parser.Parser { return mock.New(model.ClaudeCode) },
	})

	skills, err := p.ParseFromScope(model.ScopeUser)
	if err != nil {
		t.Fatalf("ParseFromScope() unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("ParseFromScope() = %d skills, want 0 for nonexistent path", len(skills))
	}
}

func TestParseFromScope_PluginScope_NoParser(t *testing.T) {
	p := New(Config{
		Platform:      model.ClaudeCode,
		WorkingDir:    t.TempDir(),
		ParserFactory: func(_ string) parser.Parser { return mock.New(model.ClaudeCode) },
	})

	skills, err := p.ParseFromScope(model.ScopePlugin)
	if err != nil {
		t.Fatalf("ParseFromScope(plugin) unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("ParseFromScope(plugin) = %d skills, want 0 with no plugin parser", len(skills))
	}
}

func TestParseFromScope_NoMatchingScope(t *testing.T) {
	p := New(Config{
		Platform:      model.Codex,
		WorkingDir:    t.TempDir(),
		ParserFactory: func(_ string) parser.Parser { return mock.New(model.Codex) },
	})

	skills, err := p.ParseFromScope(model.ScopeSystem)
	if err != nil {
		t.Fatalf("ParseFromScope(system) unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("ParseFromScope(system) = %d skills, want 0", len(skills))
	}
}

func TestMergeSkills_HigherPrecedenceWins(t *testing.T) {
	// ScopeRepo (4) has higher precedence than ScopeUser (3).
	s1 := []model.Skill{
		{Name: "skill-x", Description: "user version", Scope: model.ScopeUser},
	}
	s2 := []model.Skill{
		{Name: "skill-x", Description: "repo version", Scope: model.ScopeRepo},
	}

	merged := MergeSkills(s1, s2)
	if len(merged) != 1 {
		t.Fatalf("MergeSkills() = %d skills, want 1", len(merged))
	}
	if merged[0].Description != "repo version" {
		t.Errorf("MergeSkills() kept %q, want repo version (higher precedence)", merged[0].Description)
	}
}

func TestNewForPlatformWithDir_UnknownPlatform(t *testing.T) {
	_, err := NewForPlatformWithDir("unknown-platform", t.TempDir())
	if err == nil {
		t.Error("NewForPlatformWithDir() expected error for unknown platform, got nil")
	}
}
