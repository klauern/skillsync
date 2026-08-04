package backup

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIsUnderSkillDirectory tests the isUnderSkillDirectory helper that determines
// whether a file lives inside a skill directory (one containing SKILL.md).
func TestIsUnderSkillDirectory(t *testing.T) {
	root := t.TempDir()

	// Build a directory tree:
	//   root/
	//     skill-a/
	//       SKILL.md          <- skill entrypoint
	//       readme.md         <- non-entrypoint inside skill dir
	//       sub/
	//         helper.md       <- deeply nested inside skill dir
	//     plain/
	//       notes.md          <- NOT inside any skill dir
	//     loose.md            <- top-level file, NOT inside skill dir

	skillADir := filepath.Join(root, "skill-a")
	subDir := filepath.Join(skillADir, "sub")
	plainDir := filepath.Join(root, "plain")

	for _, dir := range []string{skillADir, subDir, plainDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	skillMd := filepath.Join(skillADir, "SKILL.md")
	readmeMd := filepath.Join(skillADir, "readme.md")
	helperMd := filepath.Join(subDir, "helper.md")
	notesMd := filepath.Join(plainDir, "notes.md")
	looseMd := filepath.Join(root, "loose.md")

	for _, path := range []string{skillMd, readmeMd, helperMd, notesMd, looseMd} {
		if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	rootClean := filepath.Clean(root)

	tests := map[string]struct {
		filePath string
		want     bool
	}{
		"readme inside skill dir is under skill directory": {
			filePath: readmeMd,
			want:     true,
		},
		"deeply nested file is under skill directory": {
			filePath: helperMd,
			want:     true,
		},
		"file in plain dir (no SKILL.md sibling) is NOT under skill directory": {
			filePath: notesMd,
			want:     false,
		},
		"top-level loose file is NOT under skill directory": {
			filePath: looseMd,
			want:     false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := isUnderSkillDirectory(tt.filePath, rootClean, make(skillDirectoryCache))
			if got != tt.want {
				t.Errorf("isUnderSkillDirectory(%q, %q) = %v, want %v",
					tt.filePath, rootClean, got, tt.want)
			}
		})
	}
}

// TestIsUnderSkillDirectory_SkillMdItself verifies the SKILL.md entrypoint
// itself IS considered under a skill directory (since it IS the entrypoint).
func TestIsUnderSkillDirectory_SkillMdItself(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "my-skill")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	skillMd := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillMd, []byte("content"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	// SKILL.md itself: its parent dir contains SKILL.md (itself), so it should return true.
	got := isUnderSkillDirectory(skillMd, filepath.Clean(root), make(skillDirectoryCache))
	if !got {
		t.Errorf("isUnderSkillDirectory(SKILL.md, root) = false, want true")
	}
}

// TestIsUnderSkillDirectory_MultipleSkillVariants verifies that lowercase and
// mixed-case skill entrypoints (skill.md, Skill.md) are also detected.
func TestIsUnderSkillDirectory_MultipleSkillVariants(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "my-skill")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Use lowercase skill.md as the entrypoint
	lowercaseSKILL := filepath.Join(skillDir, "skill.md")
	if err := os.WriteFile(lowercaseSKILL, []byte("content"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	// A sibling file inside the same dir
	sibling := filepath.Join(skillDir, "extra.md")
	if err := os.WriteFile(sibling, []byte("extra"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	rootClean := filepath.Clean(root)
	got := isUnderSkillDirectory(sibling, rootClean, make(skillDirectoryCache))
	if !got {
		t.Errorf("isUnderSkillDirectory with skill.md variant = false, want true")
	}
}

// TestResolveSourcePath_SkillEntrypointResolvesToParent tests that resolveSourcePath
// resolves a SKILL.md file to its parent directory.
func TestResolveSourcePath_SkillEntrypointResolvesToParent(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "my-skill")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	skillMd := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillMd, []byte("content"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := resolveSourcePath(skillMd)
	if err != nil {
		t.Fatalf("resolveSourcePath(%q) unexpected error: %v", skillMd, err)
	}
	if got != skillDir {
		t.Errorf("resolveSourcePath(%q) = %q, want %q", skillMd, got, skillDir)
	}
}

// TestResolveSourcePath_DirectoryPassthrough tests that a directory path is returned unchanged.
func TestResolveSourcePath_DirectoryPassthrough(t *testing.T) {
	root := t.TempDir()
	got, err := resolveSourcePath(root)
	if err != nil {
		t.Fatalf("resolveSourcePath(%q) unexpected error: %v", root, err)
	}
	if got != root {
		t.Errorf("resolveSourcePath(%q) = %q, want %q", root, got, root)
	}
}

// TestResolveSourcePath_RegularFilePassthrough tests that a regular (non-entrypoint) file
// path is returned unchanged.
func TestResolveSourcePath_RegularFilePassthrough(t *testing.T) {
	root := t.TempDir()
	regularFile := filepath.Join(root, "something.md")
	if err := os.WriteFile(regularFile, []byte("content"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := resolveSourcePath(regularFile)
	if err != nil {
		t.Fatalf("resolveSourcePath(%q) unexpected error: %v", regularFile, err)
	}
	if got != regularFile {
		t.Errorf("resolveSourcePath(%q) = %q, want %q", regularFile, got, regularFile)
	}
}

// TestResolveSourcePath_NonexistentPath tests that a nonexistent path returns an error.
func TestResolveSourcePath_NonexistentPath(t *testing.T) {
	_, err := resolveSourcePath("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("resolveSourcePath(nonexistent) expected error, got nil")
	}
}
