package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

// TestIsNestedPath directly tests the isNestedPath helper used by filterNestedDirectorySkills.
func TestIsNestedPath(t *testing.T) {
	tests := map[string]struct {
		path   string
		parent string
		want   bool
	}{
		"same path is not nested": {
			path:   "/a/b/c",
			parent: "/a/b/c",
			want:   false,
		},
		"direct child is nested": {
			path:   "/a/b/c/d",
			parent: "/a/b/c",
			want:   true,
		},
		"deep descendant is nested": {
			path:   "/a/b/c/d/e/f",
			parent: "/a/b/c",
			want:   true,
		},
		"sibling is not nested": {
			path:   "/a/b/d",
			parent: "/a/b/c",
			want:   false,
		},
		"parent of parent is not nested": {
			path:   "/a/b",
			parent: "/a/b/c",
			want:   false,
		},
		"root-level sibling is not nested": {
			path:   "/other",
			parent: "/a/b/c",
			want:   false,
		},
		"path with common prefix but not subdirectory is not nested": {
			path:   "/a/b/cd",
			parent: "/a/b/c",
			want:   false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := isNestedPath(tt.path, tt.parent)
			if got != tt.want {
				t.Errorf("isNestedPath(%q, %q) = %v, want %v", tt.path, tt.parent, got, tt.want)
			}
		})
	}
}

// TestPathDepth directly tests the pathDepth helper used by filterNestedDirectorySkills.
func TestPathDepth(t *testing.T) {
	tests := map[string]struct {
		path string
		want int
	}{
		"root slash": {
			path: "/",
			want: 0,
		},
		"dot is depth 0": {
			path: ".",
			want: 0,
		},
		"single component": {
			path: "/a",
			want: 1,
		},
		"two components": {
			path: "/a/b",
			want: 2,
		},
		"three components": {
			path: "/a/b/c",
			want: 3,
		},
		"trailing slash cleaned": {
			path: "/a/b/c/",
			want: 3,
		},
		"relative single": {
			path: "a",
			want: 0,
		},
		"relative two components": {
			path: "a/b",
			want: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := pathDepth(tt.path)
			if got != tt.want {
				t.Errorf("pathDepth(%q) = %d, want %d", tt.path, got, tt.want)
			}
		})
	}
}

// TestShouldLinkClaudeDirectorySkill directly tests the shouldLinkClaudeDirectorySkill
// logic that determines when a Claude directory skill should be symlinked on compatible targets.
func TestShouldLinkClaudeDirectorySkill(t *testing.T) {
	// Create a directory skill for testing (on disk so detectSourceType works)
	skillDir := t.TempDir()
	skillMdPath := filepath.Join(skillDir, "my-skill", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillMdPath), 0o750); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	if err := os.WriteFile(skillMdPath, []byte("---\nname: my-skill\n---\nContent"), 0o600); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Create a flat file skill for comparison
	flatFile := filepath.Join(skillDir, "flat-skill.md")
	if err := os.WriteFile(flatFile, []byte("---\nname: flat\n---\nFlat"), 0o600); err != nil {
		t.Fatalf("failed to write flat skill: %v", err)
	}

	tests := map[string]struct {
		skill  model.Skill
		target model.Platform
		want   bool
	}{
		"Claude directory skill to Codex should link": {
			skill: model.Skill{
				Name:     "my-skill",
				Platform: model.ClaudeCode,
				Path:     skillMdPath,
			},
			target: model.Codex,
			want:   true,
		},
		"Claude directory skill to Cursor should link": {
			skill: model.Skill{
				Name:     "my-skill",
				Platform: model.ClaudeCode,
				Path:     skillMdPath,
			},
			target: model.Cursor,
			want:   true,
		},
		"Claude directory skill to PiDev should link": {
			skill: model.Skill{
				Name:     "my-skill",
				Platform: model.ClaudeCode,
				Path:     skillMdPath,
			},
			target: model.PiDev,
			want:   true,
		},
		"Claude directory skill to ClaudeCode should NOT link": {
			skill: model.Skill{
				Name:     "my-skill",
				Platform: model.ClaudeCode,
				Path:     skillMdPath,
			},
			target: model.ClaudeCode,
			want:   false,
		},
		"Claude directory skill to Gemini should NOT link": {
			skill: model.Skill{
				Name:     "my-skill",
				Platform: model.ClaudeCode,
				Path:     skillMdPath,
			},
			target: model.Gemini,
			want:   false,
		},
		"non-Claude platform to Codex should NOT link": {
			skill: model.Skill{
				Name:     "my-skill",
				Platform: model.Cursor,
				Path:     skillMdPath,
			},
			target: model.Codex,
			want:   false,
		},
		"Claude flat file to Codex should NOT link": {
			skill: model.Skill{
				Name:     "flat-skill",
				Platform: model.ClaudeCode,
				Path:     flatFile,
			},
			target: model.Codex,
			want:   false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := shouldLinkClaudeDirectorySkill(tt.skill, tt.target)
			if got != tt.want {
				t.Errorf("shouldLinkClaudeDirectorySkill(%q, %q) = %v, want %v",
					tt.skill.Name, tt.target, got, tt.want)
			}
		})
	}
}

// TestMappingWarning_DisableModelInvocation tests the warning emitted when a skill
// has DisableModelInvocation set and is synced to a non-ClaudeCode platform.
func TestMappingWarning_DisableModelInvocation(t *testing.T) {
	tests := map[string]struct {
		skill    model.Skill
		target   model.Platform
		wantWarn bool
	}{
		"disable-model-invocation to Cursor warns": {
			skill: model.Skill{
				Name:                   "inline-skill",
				DisableModelInvocation: true,
			},
			target:   model.Cursor,
			wantWarn: true,
		},
		"disable-model-invocation to Codex warns": {
			skill: model.Skill{
				Name:                   "inline-skill",
				DisableModelInvocation: true,
			},
			target:   model.Codex,
			wantWarn: true,
		},
		"disable-model-invocation to PiDev warns": {
			skill: model.Skill{
				Name:                   "inline-skill",
				DisableModelInvocation: true,
			},
			target:   model.PiDev,
			wantWarn: true,
		},
		"disable-model-invocation to ClaudeCode does not warn": {
			skill: model.Skill{
				Name:                   "inline-skill",
				DisableModelInvocation: true,
			},
			target:   model.ClaudeCode,
			wantWarn: false,
		},
		"disable-model-invocation false to Cursor does not warn": {
			skill: model.Skill{
				Name:                   "plain-skill",
				DisableModelInvocation: false,
			},
			target:   model.Cursor,
			wantWarn: false,
		},
	}

	const warnSubstring = "lossy mapping: disable-model-invocation preserved as metadata only"

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			msg := mappingWarning(tt.skill, tt.target)
			gotWarn := strings.Contains(msg, warnSubstring)
			if gotWarn != tt.wantWarn {
				if tt.wantWarn {
					t.Errorf("mappingWarning() = %q, expected to contain %q", msg, warnSubstring)
				} else {
					t.Errorf("mappingWarning() = %q, expected NOT to contain %q", msg, warnSubstring)
				}
			}
		})
	}
}

// TestFilterNestedDirectorySkills tests that nested skills within a parent directory
// skill are correctly identified and skipped to avoid duplicate copies.
func TestFilterNestedDirectorySkills(t *testing.T) {
	// Build a temp tree:
	//   root/
	//     parent/SKILL.md       <- directory skill (parent)
	//       child/SKILL.md      <- nested directory skill (should be skipped)
	//     sibling/SKILL.md      <- sibling directory skill (should NOT be skipped)
	//     flat.md               <- flat file (should NOT be skipped)

	root := t.TempDir()

	parentDir := filepath.Join(root, "parent")
	childDir := filepath.Join(parentDir, "child")
	siblingDir := filepath.Join(root, "sibling")

	for _, dir := range []string{parentDir, childDir, siblingDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	parentSKILL := filepath.Join(parentDir, "SKILL.md")
	childSKILL := filepath.Join(childDir, "SKILL.md")
	siblingSKILL := filepath.Join(siblingDir, "SKILL.md")
	flatFile := filepath.Join(root, "flat.md")

	for path, content := range map[string]string{
		parentSKILL:  "---\nname: parent\n---\nParent content",
		childSKILL:   "---\nname: child\n---\nChild content",
		siblingSKILL: "---\nname: sibling\n---\nSibling content",
		flatFile:     "---\nname: flat\n---\nFlat content",
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	skills := []model.Skill{
		{Name: "parent", Path: parentSKILL, Platform: model.ClaudeCode},
		{Name: "child", Path: childSKILL, Platform: model.ClaudeCode},
		{Name: "sibling", Path: siblingSKILL, Platform: model.ClaudeCode},
		{Name: "flat", Path: flatFile, Platform: model.ClaudeCode},
	}

	filtered, skipped := filterNestedDirectorySkills(skills)

	// The child is nested under parent, so it should be skipped
	// Parent, sibling, and flat are not nested under anything, so they should be kept
	if len(filtered)+len(skipped) != len(skills) {
		t.Errorf("filtered(%d) + skipped(%d) != total(%d)", len(filtered), len(skipped), len(skills))
	}

	// Build name sets
	filteredNames := make(map[string]bool)
	for _, s := range filtered {
		filteredNames[s.Name] = true
	}
	skippedNames := make(map[string]bool)
	for _, sr := range skipped {
		skippedNames[sr.Skill.Name] = true
	}

	// Child should be skipped
	if !skippedNames["child"] {
		t.Errorf("expected 'child' to be skipped as nested, but it was not")
	}
	// Parent, sibling, flat should be kept
	for _, name := range []string{"parent", "sibling", "flat"} {
		if !filteredNames[name] {
			t.Errorf("expected %q to be kept but it was not in filtered list", name)
		}
	}
}

// TestFilterNestedDirectorySkills_SingleSkill ensures that a single skill is never filtered.
func TestFilterNestedDirectorySkills_SingleSkill(t *testing.T) {
	skillDir := t.TempDir()
	skillPath := filepath.Join(skillDir, "my-skill", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(skillPath, []byte("---\nname: my-skill\n---\nContent"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	skills := []model.Skill{{Name: "my-skill", Path: skillPath, Platform: model.ClaudeCode}}
	filtered, skipped := filterNestedDirectorySkills(skills)

	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered skill, got %d", len(filtered))
	}
	if len(skipped) != 0 {
		t.Errorf("expected 0 skipped, got %d", len(skipped))
	}
}

// TestFilterNestedDirectorySkills_EmptyInput ensures empty input returns no results.
func TestFilterNestedDirectorySkills_EmptyInput(t *testing.T) {
	filtered, skipped := filterNestedDirectorySkills(nil)
	if len(filtered) != 0 {
		t.Errorf("expected 0 filtered, got %d", len(filtered))
	}
	if len(skipped) != 0 {
		t.Errorf("expected 0 skipped, got %d", len(skipped))
	}
}
