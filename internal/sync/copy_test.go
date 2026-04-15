package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

// TestDetectSourceType verifies detection of file, directory, and symlink source types.
func TestDetectSourceType(t *testing.T) {
	t.Run("regular file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test-skill.md")
		if err := os.WriteFile(filePath, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		sourceType, rootPath := detectSourceType(filePath)

		if sourceType != SourceTypeFile {
			t.Errorf("expected SourceTypeFile, got %s", sourceType.String())
		}
		if rootPath != filePath {
			t.Errorf("expected rootPath %q, got %q", filePath, rootPath)
		}
	})

	t.Run("directory with SKILL.md", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillDir := filepath.Join(tmpDir, "my-skill")
		if err := os.MkdirAll(skillDir, 0o750); err != nil {
			t.Fatalf("failed to create skill directory: %v", err)
		}
		skillFile := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(skillFile, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create SKILL.md: %v", err)
		}

		sourceType, rootPath := detectSourceType(skillFile)

		if sourceType != SourceTypeDirectory {
			t.Errorf("expected SourceTypeDirectory, got %s", sourceType.String())
		}
		if rootPath != skillDir {
			t.Errorf("expected rootPath %q, got %q", skillDir, rootPath)
		}
	})

	t.Run("symlink to directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create target directory with SKILL.md
		targetDir := filepath.Join(tmpDir, "actual-skill")
		if err := os.MkdirAll(targetDir, 0o750); err != nil {
			t.Fatalf("failed to create target directory: %v", err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create SKILL.md: %v", err)
		}

		// Create symlink
		symlinkPath := filepath.Join(tmpDir, "skill-link")
		if err := os.Symlink(targetDir, symlinkPath); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		skillFile := filepath.Join(symlinkPath, "SKILL.md")
		sourceType, rootPath := detectSourceType(skillFile)

		if sourceType != SourceTypeSymlink {
			t.Errorf("expected SourceTypeSymlink, got %s", sourceType.String())
		}
		if rootPath != symlinkPath {
			t.Errorf("expected rootPath %q, got %q", symlinkPath, rootPath)
		}
	})

	t.Run("symlink to file", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create target file
		targetFile := filepath.Join(tmpDir, "actual-skill.md")
		if err := os.WriteFile(targetFile, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create target file: %v", err)
		}

		// Create symlink
		symlinkPath := filepath.Join(tmpDir, "skill-link.md")
		if err := os.Symlink(targetFile, symlinkPath); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		sourceType, rootPath := detectSourceType(symlinkPath)

		if sourceType != SourceTypeSymlink {
			t.Errorf("expected SourceTypeSymlink, got %s", sourceType.String())
		}
		if rootPath != symlinkPath {
			t.Errorf("expected rootPath %q, got %q", symlinkPath, rootPath)
		}
	})

	t.Run("nonexistent path defaults to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonexistent := filepath.Join(tmpDir, "does-not-exist.md")

		sourceType, rootPath := detectSourceType(nonexistent)

		if sourceType != SourceTypeFile {
			t.Errorf("expected SourceTypeFile for nonexistent, got %s", sourceType.String())
		}
		if rootPath != nonexistent {
			t.Errorf("expected rootPath %q, got %q", nonexistent, rootPath)
		}
	})
}

// TestSourceTypeString verifies the String method for SourceType.
func TestSourceTypeString(t *testing.T) {
	tests := []struct {
		sourceType SourceType
		want       string
	}{
		{SourceTypeFile, "file"},
		{SourceTypeDirectory, "directory"},
		{SourceTypeSymlink, "symlink"},
		{SourceType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.sourceType.String(); got != tt.want {
				t.Errorf("SourceType(%d).String() = %q, want %q", tt.sourceType, got, tt.want)
			}
		})
	}
}

// TestRemoveExisting verifies that removeExisting correctly removes files, symlinks, and directories.
func TestRemoveExisting(t *testing.T) {
	t.Run("remove regular file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test-file.txt")
		if err := os.WriteFile(filePath, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		if err := removeExisting(filePath); err != nil {
			t.Fatalf("removeExisting failed: %v", err)
		}

		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Error("file should have been removed")
		}
	})

	t.Run("remove symlink", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create target file
		targetFile := filepath.Join(tmpDir, "target.txt")
		if err := os.WriteFile(targetFile, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create target file: %v", err)
		}

		// Create symlink
		symlinkPath := filepath.Join(tmpDir, "symlink")
		if err := os.Symlink(targetFile, symlinkPath); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		if err := removeExisting(symlinkPath); err != nil {
			t.Fatalf("removeExisting failed: %v", err)
		}

		// Symlink should be removed
		if _, err := os.Lstat(symlinkPath); !os.IsNotExist(err) {
			t.Error("symlink should have been removed")
		}

		// Target file should still exist
		if _, err := os.Stat(targetFile); err != nil {
			t.Error("target file should still exist after removing symlink")
		}
	})

	t.Run("remove directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		dirPath := filepath.Join(tmpDir, "test-dir")
		if err := os.MkdirAll(dirPath, 0o750); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		// Add a file inside
		if err := os.WriteFile(filepath.Join(dirPath, "nested.txt"), []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create nested file: %v", err)
		}

		if err := removeExisting(dirPath); err != nil {
			t.Fatalf("removeExisting failed: %v", err)
		}

		if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
			t.Error("directory should have been removed")
		}
	})

	t.Run("nonexistent path returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonexistent := filepath.Join(tmpDir, "does-not-exist")

		if err := removeExisting(nonexistent); err != nil {
			t.Errorf("removeExisting should return nil for nonexistent path, got: %v", err)
		}
	})

	t.Run("remove symlink to directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create target directory
		targetDir := filepath.Join(tmpDir, "target-dir")
		if err := os.MkdirAll(targetDir, 0o750); err != nil {
			t.Fatalf("failed to create target directory: %v", err)
		}

		// Create symlink to directory
		symlinkPath := filepath.Join(tmpDir, "dir-symlink")
		if err := os.Symlink(targetDir, symlinkPath); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		if err := removeExisting(symlinkPath); err != nil {
			t.Fatalf("removeExisting failed: %v", err)
		}

		// Symlink should be removed
		if _, err := os.Lstat(symlinkPath); !os.IsNotExist(err) {
			t.Error("symlink to directory should have been removed")
		}

		// Target directory should still exist
		if _, err := os.Stat(targetDir); err != nil {
			t.Error("target directory should still exist after removing symlink")
		}
	})
}

// TestCopyFile verifies that copyFile copies files preserving content and permissions.
func TestCopyFile(t *testing.T) {
	t.Run("copy file with content", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "source.txt")
		dstPath := filepath.Join(tmpDir, "dest.txt")

		content := "This is test content\nwith multiple lines."
		if err := os.WriteFile(srcPath, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to create source file: %v", err)
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			t.Fatalf("copyFile failed: %v", err)
		}

		// Verify content
		// #nosec G304 - dstPath is constructed from test temp directory
		got, err := os.ReadFile(dstPath)
		if err != nil {
			t.Fatalf("failed to read destination file: %v", err)
		}
		if string(got) != content {
			t.Errorf("content mismatch: got %q, want %q", string(got), content)
		}
	})

	t.Run("preserve file permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "source.txt")
		dstPath := filepath.Join(tmpDir, "dest.txt")

		// Use read-only permissions to verify copyFile preserves mode
		if err := os.WriteFile(srcPath, []byte("content"), 0o400); err != nil {
			t.Fatalf("failed to create source file: %v", err)
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			t.Fatalf("copyFile failed: %v", err)
		}

		srcInfo, _ := os.Stat(srcPath)
		dstInfo, _ := os.Stat(dstPath)

		if srcInfo.Mode() != dstInfo.Mode() {
			t.Errorf("permissions not preserved: src=%v, dst=%v", srcInfo.Mode(), dstInfo.Mode())
		}
	})

	t.Run("copy empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "empty.txt")
		dstPath := filepath.Join(tmpDir, "dest.txt")

		if err := os.WriteFile(srcPath, []byte{}, 0o600); err != nil {
			t.Fatalf("failed to create empty source file: %v", err)
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			t.Fatalf("copyFile failed for empty file: %v", err)
		}

		dstInfo, err := os.Stat(dstPath)
		if err != nil {
			t.Fatalf("failed to stat destination: %v", err)
		}
		if dstInfo.Size() != 0 {
			t.Errorf("expected empty file, got size %d", dstInfo.Size())
		}
	})

	t.Run("error on nonexistent source", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "nonexistent.txt")
		dstPath := filepath.Join(tmpDir, "dest.txt")

		err := copyFile(srcPath, dstPath)
		if err == nil {
			t.Error("expected error for nonexistent source")
		}
	})
}

// TestCopyDir verifies that copyDir recursively copies directories including nested files and symlinks.
func TestCopyDir(t *testing.T) {
	t.Run("copy directory with files", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "source")
		dstDir := filepath.Join(tmpDir, "dest")

		// Create source structure
		if err := os.MkdirAll(srcDir, 0o750); err != nil {
			t.Fatalf("failed to create source dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0o600); err != nil {
			t.Fatalf("failed to create file1: %v", err)
		}
		if err := os.WriteFile(filepath.Join(srcDir, "file2.txt"), []byte("content2"), 0o600); err != nil {
			t.Fatalf("failed to create file2: %v", err)
		}

		if err := copyDir(srcDir, dstDir); err != nil {
			t.Fatalf("copyDir failed: %v", err)
		}

		// Verify files exist
		// #nosec G304 - paths constructed from test temp directory
		got1, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
		if err != nil {
			t.Fatalf("failed to read file1 from dest: %v", err)
		}
		if string(got1) != "content1" {
			t.Errorf("file1 content mismatch: got %q", string(got1))
		}

		// #nosec G304 - paths constructed from test temp directory
		got2, err := os.ReadFile(filepath.Join(dstDir, "file2.txt"))
		if err != nil {
			t.Fatalf("failed to read file2 from dest: %v", err)
		}
		if string(got2) != "content2" {
			t.Errorf("file2 content mismatch: got %q", string(got2))
		}
	})

	t.Run("copy nested directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "source")
		dstDir := filepath.Join(tmpDir, "dest")

		// Create nested structure
		nestedDir := filepath.Join(srcDir, "subdir", "nested")
		if err := os.MkdirAll(nestedDir, 0o750); err != nil {
			t.Fatalf("failed to create nested dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(nestedDir, "deep.txt"), []byte("deep content"), 0o600); err != nil {
			t.Fatalf("failed to create deep file: %v", err)
		}

		if err := copyDir(srcDir, dstDir); err != nil {
			t.Fatalf("copyDir failed: %v", err)
		}

		// Verify nested file
		// #nosec G304 - paths constructed from test temp directory
		got, err := os.ReadFile(filepath.Join(dstDir, "subdir", "nested", "deep.txt"))
		if err != nil {
			t.Fatalf("failed to read nested file: %v", err)
		}
		if string(got) != "deep content" {
			t.Errorf("nested file content mismatch: got %q", string(got))
		}
	})

	t.Run("copy directory with symlink", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "source")
		dstDir := filepath.Join(tmpDir, "dest")

		// Create source structure with a symlink
		if err := os.MkdirAll(srcDir, 0o750); err != nil {
			t.Fatalf("failed to create source dir: %v", err)
		}

		// Create a target file for the symlink
		targetFile := filepath.Join(tmpDir, "target.txt")
		if err := os.WriteFile(targetFile, []byte("target content"), 0o600); err != nil {
			t.Fatalf("failed to create target file: %v", err)
		}

		// Create symlink in source directory
		symlinkPath := filepath.Join(srcDir, "link.txt")
		if err := os.Symlink(targetFile, symlinkPath); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		if err := copyDir(srcDir, dstDir); err != nil {
			t.Fatalf("copyDir failed: %v", err)
		}

		// Verify symlink was copied
		dstSymlink := filepath.Join(dstDir, "link.txt")
		info, err := os.Lstat(dstSymlink)
		if err != nil {
			t.Fatalf("failed to lstat destination symlink: %v", err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Error("expected destination to be a symlink")
		}

		// Verify symlink target
		linkTarget, err := os.Readlink(dstSymlink)
		if err != nil {
			t.Fatalf("failed to read symlink target: %v", err)
		}
		if linkTarget != targetFile {
			t.Errorf("symlink target mismatch: got %q, want %q", linkTarget, targetFile)
		}
	})

	t.Run("error on non-directory source", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcFile := filepath.Join(tmpDir, "file.txt")
		dstDir := filepath.Join(tmpDir, "dest")

		if err := os.WriteFile(srcFile, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create source file: %v", err)
		}

		err := copyDir(srcFile, dstDir)
		if err == nil {
			t.Error("expected error when source is not a directory")
		}
	})

	t.Run("preserve directory permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "source")
		dstDir := filepath.Join(tmpDir, "dest")

		if err := os.MkdirAll(srcDir, 0o750); err != nil {
			t.Fatalf("failed to create source dir: %v", err)
		}

		if err := copyDir(srcDir, dstDir); err != nil {
			t.Fatalf("copyDir failed: %v", err)
		}

		srcInfo, _ := os.Stat(srcDir)
		dstInfo, _ := os.Stat(dstDir)

		// Note: MkdirAll may apply umask, so we check the essential permission bits
		srcPerm := srcInfo.Mode().Perm()
		dstPerm := dstInfo.Mode().Perm()
		if srcPerm != dstPerm {
			t.Errorf("directory permissions not preserved: src=%v, dst=%v", srcPerm, dstPerm)
		}
	})
}

// TestGetSymlinkTarget verifies getSymlinkTarget returns the correct target or empty string.
func TestGetSymlinkTarget(t *testing.T) {
	t.Run("valid symlink", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create target
		targetPath := filepath.Join(tmpDir, "target")
		if err := os.WriteFile(targetPath, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create target: %v", err)
		}

		// Create symlink
		symlinkPath := filepath.Join(tmpDir, "link")
		if err := os.Symlink(targetPath, symlinkPath); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		got := getSymlinkTarget(symlinkPath)
		if got != targetPath {
			t.Errorf("getSymlinkTarget() = %q, want %q", got, targetPath)
		}
	})

	t.Run("not a symlink returns empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "file.txt")
		if err := os.WriteFile(filePath, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		got := getSymlinkTarget(filePath)
		if got != "" {
			t.Errorf("getSymlinkTarget() on regular file = %q, want empty string", got)
		}
	})

	t.Run("nonexistent path returns empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonexistent := filepath.Join(tmpDir, "nonexistent")

		got := getSymlinkTarget(nonexistent)
		if got != "" {
			t.Errorf("getSymlinkTarget() on nonexistent = %q, want empty string", got)
		}
	})

	t.Run("relative symlink target", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create target
		if err := os.WriteFile(filepath.Join(tmpDir, "target.txt"), []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create target: %v", err)
		}

		// Create symlink with relative path
		symlinkPath := filepath.Join(tmpDir, "link")
		if err := os.Symlink("target.txt", symlinkPath); err != nil {
			t.Fatalf("failed to create symlink: %v", err)
		}

		got := getSymlinkTarget(symlinkPath)
		if got != "target.txt" {
			t.Errorf("getSymlinkTarget() = %q, want %q", got, "target.txt")
		}
	})
}

// TestSymlinkPreservation verifies that syncing a symlink skill creates a symlink at target.
func TestSymlinkPreservation(t *testing.T) {
	s := New()

	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source", "skills")
	targetDir := filepath.Join(tmpDir, "target", "skills")

	// Create source skills directory
	if err := os.MkdirAll(sourceDir, 0o750); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	// Create actual skill directory outside the skills folder
	actualSkillDir := filepath.Join(tmpDir, "actual-skill")
	if err := os.MkdirAll(actualSkillDir, 0o750); err != nil {
		t.Fatalf("failed to create actual skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(actualSkillDir, "SKILL.md"), []byte("---\nname: my-dev-skill\n---\nSkill content"), 0o600); err != nil {
		t.Fatalf("failed to create SKILL.md: %v", err)
	}

	// Create symlink in source skills directory pointing to actual skill
	symlinkPath := filepath.Join(sourceDir, "my-dev-skill")
	if err := os.Symlink(actualSkillDir, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Create a source skill model that represents the symlink
	sourceSkill := model.Skill{
		Name:     "my-dev-skill",
		Path:     filepath.Join(symlinkPath, "SKILL.md"),
		Platform: model.ClaudeCode,
		PluginInfo: &model.PluginInfo{
			SymlinkTarget: actualSkillDir,
			IsDev:         true,
		},
	}

	// Sync the skill
	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		TargetPath: targetDir,
	}

	result, err := s.SyncWithSkills([]model.Skill{sourceSkill}, model.Codex, opts)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill result, got %d", len(result.Skills))
	}

	if result.Skills[0].Action == ActionFailed {
		t.Fatalf("sync failed with error: %v", result.Skills[0].Error)
	}

	// Verify target is a symlink
	targetSkillPath := filepath.Join(targetDir, "my-dev-skill")
	info, err := os.Lstat(targetSkillPath)
	if err != nil {
		t.Fatalf("failed to stat target: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected target to be a symlink, but it's not")
	}

	// Verify symlink target
	linkTarget, err := os.Readlink(targetSkillPath)
	if err != nil {
		t.Fatalf("failed to read symlink target: %v", err)
	}
	if linkTarget != actualSkillDir {
		t.Errorf("symlink target = %q, want %q", linkTarget, actualSkillDir)
	}
}

// TestDirectoryPreservation verifies that syncing a directory skill copies the directory structure.
func TestDirectoryPreservation(t *testing.T) {
	s := New()

	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source", "skills")
	targetDir := filepath.Join(tmpDir, "target", "skills")

	// Create source skills directory
	if err := os.MkdirAll(sourceDir, 0o750); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	// Create a directory-based skill in source
	skillDir := filepath.Join(sourceDir, "complex-skill")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: complex-skill\n---\nMain skill content"), 0o600); err != nil {
		t.Fatalf("failed to create SKILL.md: %v", err)
	}
	// Add additional files
	if err := os.WriteFile(filepath.Join(skillDir, "helper.md"), []byte("Helper content"), 0o600); err != nil {
		t.Fatalf("failed to create helper.md: %v", err)
	}

	// Create a source skill model
	sourceSkill := model.Skill{
		Name:     "complex-skill",
		Path:     filepath.Join(skillDir, "SKILL.md"),
		Content:  "Main skill content",
		Platform: model.ClaudeCode,
	}

	// Sync the skill
	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		TargetPath: targetDir,
	}

	result, err := s.SyncWithSkills([]model.Skill{sourceSkill}, model.Codex, opts)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill result, got %d", len(result.Skills))
	}

	if result.Skills[0].Action == ActionFailed {
		t.Fatalf("sync failed with error: %v", result.Skills[0].Error)
	}

	// Verify target is a directory (not a .md file)
	targetSkillPath := filepath.Join(targetDir, "complex-skill")
	info, err := os.Stat(targetSkillPath)
	if err != nil {
		t.Fatalf("failed to stat target: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected target to be a directory, but it's not")
	}

	// Verify SKILL.md exists
	skillMdPath := filepath.Join(targetSkillPath, "SKILL.md")
	if _, err := os.Stat(skillMdPath); err != nil {
		t.Errorf("SKILL.md not found in target directory: %v", err)
	}

	// Verify additional files were copied
	helperPath := filepath.Join(targetSkillPath, "helper.md")
	if _, err := os.Stat(helperPath); err != nil {
		t.Errorf("helper.md not found in target directory: %v", err)
	}
}

// TestSyncOverwriteExistingSymlink verifies that syncing overwrites existing entries.
func TestSyncOverwriteExistingSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(targetDir, 0o750); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	// Create an existing file at the target location
	existingPath := filepath.Join(targetDir, "skill-name")
	if err := os.WriteFile(existingPath, []byte("old content"), 0o600); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	// Now remove it using removeExisting
	if err := removeExisting(existingPath); err != nil {
		t.Fatalf("removeExisting failed: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(existingPath); !os.IsNotExist(err) {
		t.Error("existing file should have been removed")
	}

	// Now create a directory at the same location
	if err := os.MkdirAll(existingPath, 0o750); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Remove it
	if err := removeExisting(existingPath); err != nil {
		t.Fatalf("removeExisting failed for directory: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(existingPath); !os.IsNotExist(err) {
		t.Error("existing directory should have been removed")
	}
}

// TestCopyDirWithMixedContent tests copying a directory with various file types.
func TestCopyDirWithMixedContent(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "source")
	dstDir := filepath.Join(tmpDir, "dest")

	// Create complex structure
	if err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0o750); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Regular file
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("skill content"), 0o600); err != nil {
		t.Fatalf("failed to create SKILL.md: %v", err)
	}

	// File in subdir
	if err := os.WriteFile(filepath.Join(srcDir, "subdir", "nested.md"), []byte("nested content"), 0o600); err != nil {
		t.Fatalf("failed to create nested.md: %v", err)
	}

	// External target for symlink
	externalTarget := filepath.Join(tmpDir, "external.txt")
	if err := os.WriteFile(externalTarget, []byte("external"), 0o600); err != nil {
		t.Fatalf("failed to create external target: %v", err)
	}

	// Symlink in source
	if err := os.Symlink(externalTarget, filepath.Join(srcDir, "link.txt")); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Copy
	if err := copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	// Verify all components
	// 1. SKILL.md
	// #nosec G304 - paths constructed from test temp directory
	content, err := os.ReadFile(filepath.Join(dstDir, "SKILL.md"))
	if err != nil {
		t.Errorf("failed to read SKILL.md: %v", err)
	} else if string(content) != "skill content" {
		t.Errorf("SKILL.md content mismatch: got %q", string(content))
	}

	// 2. Nested file
	// #nosec G304 - paths constructed from test temp directory
	content, err = os.ReadFile(filepath.Join(dstDir, "subdir", "nested.md"))
	if err != nil {
		t.Errorf("failed to read nested.md: %v", err)
	} else if string(content) != "nested content" {
		t.Errorf("nested.md content mismatch: got %q", string(content))
	}

	// 3. Symlink
	linkPath := filepath.Join(dstDir, "link.txt")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Errorf("failed to lstat symlink: %v", err)
	} else if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected link.txt to be a symlink")
	}

	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Errorf("failed to readlink: %v", err)
	} else if target != externalTarget {
		t.Errorf("symlink target mismatch: got %q, want %q", target, externalTarget)
	}
}
