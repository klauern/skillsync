package piagent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/util"
)

// TestNew tests the piagent parser constructor with custom and default paths.
func TestNew(t *testing.T) {
	t.Run("custom path is preserved", func(t *testing.T) {
		custom := "/custom/pi/skills"
		p := New(custom)
		if p.basePath != custom {
			t.Errorf("New(%q).basePath = %q, want %q", custom, p.basePath, custom)
		}
	})

	t.Run("empty path uses default", func(t *testing.T) {
		p := New("")
		want := util.PiAgentSkillsPath()
		if p.basePath != want {
			t.Errorf("New(\"\").basePath = %q, want %q", p.basePath, want)
		}
	})
}

// TestParser_Platform verifies the parser reports the correct platform.
func TestParser_Platform(t *testing.T) {
	p := New("/some/path")
	if p.Platform() != model.PiAgent {
		t.Errorf("Platform() = %q, want %q", p.Platform(), model.PiAgent)
	}
}

// TestParser_DefaultPath verifies DefaultPath returns the expected utility path.
func TestParser_DefaultPath(t *testing.T) {
	p := New("/custom")
	want := util.PiAgentSkillsPath()
	if p.DefaultPath() != want {
		t.Errorf("DefaultPath() = %q, want %q", p.DefaultPath(), want)
	}
}

// TestParser_Parse_EmptyDirectory tests that parsing a non-existent directory
// returns no skills and no error.
func TestParser_Parse_EmptyDirectory(t *testing.T) {
	p := New("/nonexistent/piagent/skills")
	skills, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(skills) != 0 {
		t.Errorf("Parse() = %d skills, want 0", len(skills))
	}
}

// TestParseSettingsSkillPaths_NonexistentFile tests that a missing settings file
// returns nil paths without error.
func TestParseSettingsSkillPaths_NonexistentFile(t *testing.T) {
	paths, err := parseSettingsSkillPaths("/nonexistent/settings.json")
	if err != nil {
		t.Errorf("parseSettingsSkillPaths(nonexistent) error = %v, want nil", err)
	}
	if paths != nil {
		t.Errorf("parseSettingsSkillPaths(nonexistent) = %v, want nil", paths)
	}
}

// TestParseSettingsSkillPaths_MalformedJSON tests that malformed JSON returns an error.
func TestParseSettingsSkillPaths_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte("{invalid-json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := parseSettingsSkillPaths(settingsPath)
	if err == nil {
		t.Error("parseSettingsSkillPaths(malformed) expected error, got nil")
	}
}

// TestParseSettingsSkillPaths_EmptySkillsDirectories tests a valid JSON file with
// no skillsDirectories defined returns an empty slice without error.
func TestParseSettingsSkillPaths_EmptySkillsDirectories(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"skillsDirectories":[]}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	paths, err := parseSettingsSkillPaths(settingsPath)
	if err != nil {
		t.Fatalf("parseSettingsSkillPaths(empty dirs) error = %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("parseSettingsSkillPaths(empty dirs) = %v, want empty", paths)
	}
}

// TestParseSettingsSkillPaths_RelativePath tests that relative paths in settings are
// resolved relative to the settings file's parent directory.
func TestParseSettingsSkillPaths_RelativePath(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	content := `{"skillsDirectories":["my-skills","../other"]}`
	if err := os.WriteFile(settingsPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	paths, err := parseSettingsSkillPaths(settingsPath)
	if err != nil {
		t.Fatalf("parseSettingsSkillPaths(relative) error = %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("parseSettingsSkillPaths() returned %d paths, want 2", len(paths))
	}

	// Relative paths should be resolved relative to the settings file's directory
	wantFirst := filepath.Join(dir, "my-skills")
	if paths[0] != wantFirst {
		t.Errorf("paths[0] = %q, want %q", paths[0], wantFirst)
	}
}

// TestAncestorSkillPaths_SameDir tests that when workingDir equals repoRoot,
// only one path is returned (no duplication).
func TestAncestorSkillPaths_SameDir(t *testing.T) {
	dir := t.TempDir()
	paths := ancestorSkillPaths(dir, dir)
	if len(paths) != 1 {
		t.Errorf("ancestorSkillPaths(dir == repoRoot) returned %d paths, want 1", len(paths))
	}
	expected := filepath.Join(dir, ".agents", "skills")
	if paths[0] != expected {
		t.Errorf("paths[0] = %q, want %q", paths[0], expected)
	}
}

// TestAncestorSkillPaths_NestedDir tests that paths from workingDir up to repoRoot
// are all included.
func TestAncestorSkillPaths_NestedDir(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	paths := ancestorSkillPaths(nested, root)

	// Should include: nested/.agents/skills, root/a/.agents/skills, root/.agents/skills
	if len(paths) != 3 {
		t.Errorf("expected 3 ancestor paths, got %d: %v", len(paths), paths)
	}

	// All paths should end in .agents/skills
	for i, p := range paths {
		if filepath.Base(p) != "skills" {
			t.Errorf("path[%d] = %q, expected to end in 'skills'", i, p)
		}
	}
}

// TestDiscoverSearchPaths_NoGitRepo tests DiscoverSearchPaths when the working
// directory is not inside a git repository (falls back to workingDir as repoRoot).
func TestDiscoverSearchPaths_NoGitRepo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Working directory with no .git parent
	workingDir := t.TempDir()

	paths, err := DiscoverSearchPaths(workingDir)
	if err != nil {
		t.Fatalf("DiscoverSearchPaths() error = %v", err)
	}

	// Should at least include the user-level path derived from HOME and workingDir paths
	if len(paths) == 0 {
		t.Error("DiscoverSearchPaths() returned empty paths, expected at least user path")
	}

	// All paths should be non-empty strings
	for i, p := range paths {
		if p.Path == "" {
			t.Errorf("path[%d].Path is empty", i)
		}
	}
}

// TestDiscoverSearchPaths_Deduplication tests that duplicate paths are only
// returned once even if they appear in multiple sources.
func TestDiscoverSearchPaths_Deduplication(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// repo root == working dir (generates one ancestor path)
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o750); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	// settings.json that lists the same path as ancestor skill path
	ancestorPath := filepath.Join(repoRoot, ".agents", "skills")
	if err := os.MkdirAll(ancestorPath, 0o750); err != nil {
		t.Fatalf("mkdir ancestor: %v", err)
	}

	settingsDir := filepath.Join(repoRoot, ".pi")
	if err := os.MkdirAll(settingsDir, 0o750); err != nil {
		t.Fatalf("mkdir .pi: %v", err)
	}
	// Absolute path - same as ancestor path
	settingsJSON := `{"skillsDirectories":["` + filepath.ToSlash(ancestorPath) + `"]}`
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(settingsJSON), 0o600); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	paths, err := DiscoverSearchPaths(repoRoot)
	if err != nil {
		t.Fatalf("DiscoverSearchPaths() error = %v", err)
	}

	// Count occurrences of the ancestor path
	count := 0
	for _, p := range paths {
		if p.Path == ancestorPath {
			count++
		}
	}

	if count > 1 {
		t.Errorf("ancestorPath appeared %d times, want at most 1 (deduplication failed)", count)
	}
}