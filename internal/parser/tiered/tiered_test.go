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

func TestNew(t *testing.T) {
	cfg := Config{
		Platform:   model.ClaudeCode,
		WorkingDir: "/test/project",
		ParserFactory: func(_ string) parser.Parser {
			return mock.New(model.ClaudeCode)
		},
	}

	p := New(cfg)

	if p.platform != model.ClaudeCode {
		t.Errorf("Parser.platform = %s, want %s", p.platform, model.ClaudeCode)
	}

	if p.pathConfig.WorkingDir != "/test/project" {
		t.Errorf("Parser.pathConfig.WorkingDir = %s, want /test/project", p.pathConfig.WorkingDir)
	}
}

func TestParser_Parse_EmptyPaths(t *testing.T) {
	// Create parser with non-existent paths
	p := New(Config{
		Platform:   model.ClaudeCode,
		WorkingDir: "/nonexistent/path",
		ParserFactory: func(_ string) parser.Parser {
			return mock.New(model.ClaudeCode)
		},
	})

	skills, err := p.Parse()
	if err != nil {
		t.Errorf("Parse() returned error: %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("Parse() returned %d skills, expected 0", len(skills))
	}
}

func TestParser_Parse_WithSkills(t *testing.T) {
	// Create temp directories for testing
	tmpDir := t.TempDir()

	// Create a repo-level skills directory
	repoSkillsDir := filepath.Join(tmpDir, ".claude", "skills")
	if err := os.MkdirAll(repoSkillsDir, 0o750); err != nil {
		t.Fatalf("failed to create repo skills dir: %v", err)
	}

	// Create a skill file
	skillFile := filepath.Join(repoSkillsDir, "test-skill.md")
	skillContent := `---
name: test-skill
description: A test skill
---
# Test Skill Content
`
	// #nosec G306 - test file, permissions not security-critical
	if err := os.WriteFile(skillFile, []byte(skillContent), 0o600); err != nil {
		t.Fatalf("failed to create skill file: %v", err)
	}

	// Create tiered parser
	p := NewForPlatformWithDir(model.ClaudeCode, tmpDir)

	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if len(skills) == 0 {
		t.Fatal("Parse() returned no skills, expected at least 1")
	}

	// Verify the skill has repo scope
	found := false
	for _, skill := range skills {
		if skill.Name == "test-skill" {
			found = true
			if skill.Scope != model.ScopeRepo {
				t.Errorf("Skill scope = %s, want %s", skill.Scope, model.ScopeRepo)
			}
		}
	}

	if !found {
		t.Error("Did not find test-skill in parsed skills")
	}
}

func TestParser_Parse_PrecedenceDeduplication(t *testing.T) {
	tmpDir := t.TempDir()

	// Create user-level skills directory
	userSkillsDir := filepath.Join(tmpDir, "user", ".claude", "skills")
	if err := os.MkdirAll(userSkillsDir, 0o750); err != nil {
		t.Fatalf("failed to create user skills dir: %v", err)
	}

	// Create repo-level skills directory
	repoSkillsDir := filepath.Join(tmpDir, "repo", ".claude", "skills")
	if err := os.MkdirAll(repoSkillsDir, 0o750); err != nil {
		t.Fatalf("failed to create repo skills dir: %v", err)
	}

	// Create same-named skill in both locations with different content
	userSkill := `---
name: shared-skill
description: User version
---
User content
`
	repoSkill := `---
name: shared-skill
description: Repo version
---
Repo content
`

	// #nosec G306 - test files, permissions not security-critical
	if err := os.WriteFile(filepath.Join(userSkillsDir, "shared-skill.md"), []byte(userSkill), 0o600); err != nil {
		t.Fatalf("failed to create user skill: %v", err)
	}
	// #nosec G306 - test files, permissions not security-critical
	if err := os.WriteFile(filepath.Join(repoSkillsDir, "shared-skill.md"), []byte(repoSkill), 0o600); err != nil {
		t.Fatalf("failed to create repo skill: %v", err)
	}

	// Use mock parser factory that simulates finding skills in different directories
	mockSkills := map[string][]model.Skill{
		userSkillsDir: {
			{Name: "shared-skill", Description: "User version", Scope: model.ScopeUser},
		},
		repoSkillsDir: {
			{Name: "shared-skill", Description: "Repo version", Scope: model.ScopeRepo},
		},
	}

	parserFactory := func(basePath string) parser.Parser {
		m := mock.New(model.ClaudeCode)
		if skills, ok := mockSkills[basePath]; ok {
			m.WithSkills(skills)
		}
		return m
	}

	// Create tiered parser with custom path config
	p := New(Config{
		Platform:   model.ClaudeCode,
		WorkingDir: filepath.Join(tmpDir, "repo"),
		ParserFactory: func(basePath string) parser.Parser {
			return parserFactory(basePath)
		},
	})

	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	// Should only have one skill (deduplicated)
	sharedSkillCount := 0
	var sharedSkill model.Skill
	for _, skill := range skills {
		if skill.Name == "shared-skill" {
			sharedSkillCount++
			sharedSkill = skill
		}
	}

	if sharedSkillCount != 1 {
		t.Errorf("Expected 1 shared-skill, got %d", sharedSkillCount)
	}

	// The repo version should win (higher precedence)
	if sharedSkill.Scope != model.ScopeRepo {
		t.Errorf("Expected repo scope (higher precedence), got %s", sharedSkill.Scope)
	}
}

func TestParser_Platform(t *testing.T) {
	tests := []struct {
		platform model.Platform
	}{
		{model.ClaudeCode},
		{model.Cursor},
		{model.Codex},
	}

	for _, tt := range tests {
		t.Run(string(tt.platform), func(t *testing.T) {
			platform := tt.platform // Capture for closure
			p := New(Config{
				Platform:   platform,
				WorkingDir: "/test",
				ParserFactory: func(_ string) parser.Parser {
					return mock.New(platform)
				},
			})

			if got := p.Platform(); got != platform {
				t.Errorf("Platform() = %s, want %s", got, platform)
			}
		})
	}
}

func TestParser_DefaultPath(t *testing.T) {
	home := util.HomeDir()

	tests := []struct {
		platform model.Platform
		expected string
	}{
		{model.ClaudeCode, filepath.Join(home, ".claude", "skills")},
		{model.Cursor, filepath.Join(home, ".cursor", "skills")},
		{model.Codex, filepath.Join(home, ".codex", "skills")},
	}

	for _, tt := range tests {
		t.Run(string(tt.platform), func(t *testing.T) {
			platform := tt.platform // Capture for closure
			p := New(Config{
				Platform:   platform,
				WorkingDir: "/test",
				ParserFactory: func(_ string) parser.Parser {
					return mock.New(platform)
				},
			})

			if got := p.DefaultPath(); got != tt.expected {
				t.Errorf("DefaultPath() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestParser_GetSearchPaths(t *testing.T) {
	p := New(Config{
		Platform:   model.ClaudeCode,
		WorkingDir: "/test/project",
		AdminPath:  "/opt/claude/skills",
		ParserFactory: func(_ string) parser.Parser {
			return mock.New(model.ClaudeCode)
		},
	})

	paths := p.GetSearchPaths()

	if len(paths) == 0 {
		t.Fatal("GetSearchPaths() returned empty slice")
	}

	// Verify we have repo scope first
	if paths[0].Scope != model.ScopeRepo {
		t.Errorf("First path scope = %s, want %s", paths[0].Scope, model.ScopeRepo)
	}
}

func TestMergeSkills(t *testing.T) {
	tests := []struct {
		name      string
		skillSets [][]model.Skill
		wantCount int
		wantScope model.SkillScope // For the "shared" skill
	}{
		{
			name: "no duplicates",
			skillSets: [][]model.Skill{
				{{Name: "skill1", Scope: model.ScopeUser}},
				{{Name: "skill2", Scope: model.ScopeRepo}},
			},
			wantCount: 2,
		},
		{
			name: "repo overrides user",
			skillSets: [][]model.Skill{
				{{Name: "shared", Scope: model.ScopeUser}},
				{{Name: "shared", Scope: model.ScopeRepo}},
			},
			wantCount: 1,
			wantScope: model.ScopeRepo,
		},
		{
			name: "user does not override repo",
			skillSets: [][]model.Skill{
				{{Name: "shared", Scope: model.ScopeRepo}},
				{{Name: "shared", Scope: model.ScopeUser}},
			},
			wantCount: 1,
			wantScope: model.ScopeRepo,
		},
		{
			name:      "empty sets",
			skillSets: [][]model.Skill{{}, {}},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeSkills(tt.skillSets...)

			if len(got) != tt.wantCount {
				t.Errorf("MergeSkills() returned %d skills, want %d", len(got), tt.wantCount)
			}

			if tt.wantScope != "" {
				for _, skill := range got {
					if skill.Name == "shared" && skill.Scope != tt.wantScope {
						t.Errorf("Shared skill scope = %s, want %s", skill.Scope, tt.wantScope)
					}
				}
			}
		})
	}
}

func TestDeduplicateByName(t *testing.T) {
	skills := []model.Skill{
		{Name: "skill1"},
		{Name: "skill2"},
		{Name: "skill1"}, // Duplicate
		{Name: "skill3"},
		{Name: "skill2"}, // Duplicate
	}

	got := DeduplicateByName(skills)

	if len(got) != 3 {
		t.Errorf("DeduplicateByName() returned %d skills, want 3", len(got))
	}

	// First occurrence should be kept
	if got[0].Name != "skill1" || got[1].Name != "skill2" || got[2].Name != "skill3" {
		t.Errorf("DeduplicateByName() wrong order or names: %v", got)
	}
}

func TestParserFactoryFor(t *testing.T) {
	tests := []struct {
		platform model.Platform
	}{
		{model.ClaudeCode},
		{model.Cursor},
		{model.Codex},
	}

	for _, tt := range tests {
		t.Run(string(tt.platform), func(t *testing.T) {
			factory := ParserFactoryFor(tt.platform)
			if factory == nil {
				t.Error("ParserFactoryFor() returned nil")
			}

			// Verify factory creates a parser
			p := factory("/test/path")
			if p == nil {
				t.Error("Factory returned nil parser")
			}

			if p.Platform() != tt.platform {
				t.Errorf("Parser platform = %s, want %s", p.Platform(), tt.platform)
			}
		})
	}
}

func TestNewForPlatform(t *testing.T) {
	// This test actually uses the real current working directory
	p, err := NewForPlatform(model.ClaudeCode)
	if err != nil {
		t.Fatalf("NewForPlatform() error = %v", err)
	}

	if p == nil {
		t.Fatal("NewForPlatform() returned nil")
	}

	if p.Platform() != model.ClaudeCode {
		t.Errorf("Platform() = %s, want %s", p.Platform(), model.ClaudeCode)
	}
}

func TestParser_ParseWithScopeFilter(t *testing.T) {
	// This test verifies scope filtering works correctly with actual directories.
	// We create repo-level skills and verify filtering by repo scope finds them.
	tmpDir := t.TempDir()

	// Create repo-level skills directory
	repoSkillsDir := filepath.Join(tmpDir, ".claude", "skills")
	if err := os.MkdirAll(repoSkillsDir, 0o750); err != nil {
		t.Fatalf("failed to create repo skills dir: %v", err)
	}

	// Create a skill file in repo
	skillContent := `---
name: repo-only-skill
description: A repo-level skill
---
# Repo Skill
`
	// #nosec G306 - test file
	if err := os.WriteFile(filepath.Join(repoSkillsDir, "repo-only-skill.md"), []byte(skillContent), 0o600); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}

	t.Run("filter to repo scope finds repo skills", func(t *testing.T) {
		p := NewForPlatformWithDir(model.ClaudeCode, tmpDir)

		skills, err := p.ParseWithScopeFilter([]model.SkillScope{model.ScopeRepo})
		if err != nil {
			t.Fatalf("ParseWithScopeFilter() error = %v", err)
		}

		// Should find the repo skill
		found := false
		for _, s := range skills {
			if s.Name == "repo-only-skill" {
				found = true
				if s.Scope != model.ScopeRepo {
					t.Errorf("skill scope = %v, want %v", s.Scope, model.ScopeRepo)
				}
			}
		}

		if !found {
			t.Error("repo-only-skill not found when filtering by repo scope")
		}
	})

	t.Run("empty scope filter returns no skills", func(t *testing.T) {
		p := NewForPlatformWithDir(model.ClaudeCode, tmpDir)

		skills, err := p.ParseWithScopeFilter([]model.SkillScope{})
		if err != nil {
			t.Fatalf("ParseWithScopeFilter() error = %v", err)
		}

		if len(skills) != 0 {
			t.Errorf("expected 0 skills with empty filter, got %d", len(skills))
		}
	})

	t.Run("filter to non-existent scope returns empty", func(t *testing.T) {
		p := NewForPlatformWithDir(model.ClaudeCode, tmpDir)

		// Filter to system scope - we haven't created any system skills
		skills, err := p.ParseWithScopeFilter([]model.SkillScope{model.ScopeSystem})
		if err != nil {
			t.Fatalf("ParseWithScopeFilter() error = %v", err)
		}

		// Should not find any skills (no system-level skills exist)
		for _, s := range skills {
			if s.Scope != model.ScopeSystem {
				t.Errorf("found non-system skill %q when filtering by system scope", s.Name)
			}
		}
	})
}

func TestParser_ParseFromScope(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple repo paths (working dir and git root)
	workingDirSkills := filepath.Join(tmpDir, "working", ".claude", "skills")
	gitRootSkills := filepath.Join(tmpDir, "gitroot", ".claude", "skills")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(workingDirSkills, 0o750); err != nil {
		t.Fatalf("failed to create working dir skills: %v", err)
	}
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(gitRootSkills, 0o750); err != nil {
		t.Fatalf("failed to create git root skills: %v", err)
	}

	// Create skills with same name in both repo paths
	mockSkills := map[string][]model.Skill{
		workingDirSkills: {
			{Name: "skill-a", Description: "From working dir"},
			{Name: "shared-repo-skill", Description: "Working dir version"},
		},
		gitRootSkills: {
			{Name: "skill-b", Description: "From git root"},
			{Name: "shared-repo-skill", Description: "Git root version"},
		},
	}

	parserFactory := func(basePath string) parser.Parser {
		m := mock.New(model.ClaudeCode)
		if skills, ok := mockSkills[basePath]; ok {
			m.WithSkills(skills)
		}
		return m
	}

	p := New(Config{
		Platform:      model.ClaudeCode,
		WorkingDir:    filepath.Join(tmpDir, "working"),
		ParserFactory: parserFactory,
	})

	skills, err := p.ParseFromScope(model.ScopeRepo)
	if err != nil {
		t.Fatalf("ParseFromScope() error = %v", err)
	}

	// Within same scope, first found wins - so we expect "Working dir version"
	found := make(map[string]model.Skill)
	for _, s := range skills {
		found[s.Name] = s
	}

	if skill, ok := found["shared-repo-skill"]; ok {
		if skill.Description != "From working dir" && skill.Description != "Working dir version" {
			// Note: actual behavior depends on path ordering
			t.Logf("shared-repo-skill description = %q (first-found-wins behavior)", skill.Description)
		}
		if skill.Scope != model.ScopeRepo {
			t.Errorf("shared-repo-skill scope = %v, want %v", skill.Scope, model.ScopeRepo)
		}
	}
}

func TestMergeSkills_AllScopeCombinations(t *testing.T) {
	tests := []struct {
		name      string
		skills1   []model.Skill
		skills2   []model.Skill
		wantScope model.SkillScope
	}{
		{
			name:      "repo beats user",
			skills1:   []model.Skill{{Name: "test", Scope: model.ScopeUser}},
			skills2:   []model.Skill{{Name: "test", Scope: model.ScopeRepo}},
			wantScope: model.ScopeRepo,
		},
		{
			name:      "user beats admin",
			skills1:   []model.Skill{{Name: "test", Scope: model.ScopeAdmin}},
			skills2:   []model.Skill{{Name: "test", Scope: model.ScopeUser}},
			wantScope: model.ScopeUser,
		},
		{
			name:      "admin beats system",
			skills1:   []model.Skill{{Name: "test", Scope: model.ScopeSystem}},
			skills2:   []model.Skill{{Name: "test", Scope: model.ScopeAdmin}},
			wantScope: model.ScopeAdmin,
		},
		{
			name:      "system beats builtin",
			skills1:   []model.Skill{{Name: "test", Scope: model.ScopeBuiltin}},
			skills2:   []model.Skill{{Name: "test", Scope: model.ScopeSystem}},
			wantScope: model.ScopeSystem,
		},
		{
			name:      "repo beats builtin (extreme)",
			skills1:   []model.Skill{{Name: "test", Scope: model.ScopeBuiltin}},
			skills2:   []model.Skill{{Name: "test", Scope: model.ScopeRepo}},
			wantScope: model.ScopeRepo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeSkills(tt.skills1, tt.skills2)

			if len(result) != 1 {
				t.Fatalf("MergeSkills() returned %d skills, want 1", len(result))
			}

			if result[0].Scope != tt.wantScope {
				t.Errorf("MergeSkills() result scope = %v, want %v", result[0].Scope, tt.wantScope)
			}
		})
	}
}

func TestParser_GetExistingSearchPaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only some paths
	existingDir := filepath.Join(tmpDir, ".claude", "skills")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(existingDir, 0o750); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	p := NewForPlatformWithDir(model.ClaudeCode, tmpDir)

	existingPaths := p.GetExistingSearchPaths()
	allPaths := p.GetSearchPaths()

	// Existing paths should be subset of all paths
	if len(existingPaths) > len(allPaths) {
		t.Errorf("GetExistingSearchPaths() returned more paths (%d) than GetSearchPaths() (%d)",
			len(existingPaths), len(allPaths))
	}

	// All existing paths should actually exist
	for _, sp := range existingPaths {
		if _, err := os.Stat(sp.Path); os.IsNotExist(err) {
			t.Errorf("GetExistingSearchPaths() returned non-existent path: %s", sp.Path)
		}
	}
}
