package sync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/klauern/skillsync/internal/logging"
)

// SourceType indicates the type of skill source structure.
type SourceType int

const (
	// SourceTypeFile indicates a regular file skill (e.g., skill-name.md).
	SourceTypeFile SourceType = iota
	// SourceTypeDirectory indicates a directory-based skill (e.g., skill-name/SKILL.md).
	SourceTypeDirectory
	// SourceTypeSymlink indicates a symlink to a skill directory or file.
	SourceTypeSymlink
)

// String returns a human-readable string for SourceType.
func (st SourceType) String() string {
	switch st {
	case SourceTypeFile:
		return "file"
	case SourceTypeDirectory:
		return "directory"
	case SourceTypeSymlink:
		return "symlink"
	default:
		return "unknown"
	}
}

// removeExisting removes a file, symlink, or directory at the given path.
// Uses os.Lstat to not follow symlinks, ensuring symlinks are removed as entries.
// Returns nil if the path doesn't exist.
func removeExisting(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to stat %q: %w", path, err)
	}

	if info.IsDir() {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove directory %q: %w", path, err)
		}
		logging.Debug("removed existing directory", logging.Path(path))
	} else {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			logging.Debug("removed existing symlink", logging.Path(path))
		} else {
			logging.Debug("removed existing file", logging.Path(path))
		}
	}

	return nil
}

// copyFile copies a single file from src to dst, preserving permissions.
func copyFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source %q: %w", src, err)
	}

	// #nosec G304 - src is from trusted skill paths
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source %q: %w", src, err)
	}
	defer func() { _ = srcFile.Close() }()

	// Create destination file with same permissions
	// #nosec G302 G304 - preserving source permissions, dst is from trusted paths
	dstFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination %q: %w", dst, err)
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy content to %q: %w", dst, err)
	}

	logging.Debug("copied file",
		logging.Path(src),
	)

	return nil
}

// copyDir recursively copies a directory from src to dst.
// If dst exists, it will be removed first.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source %q: %w", src, err)
	}

	if !srcInfo.IsDir() {
		return fmt.Errorf("source %q is not a directory", src)
	}

	// Create destination directory with same permissions
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory %q: %w", dst, err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory %q: %w", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Check if it's a symlink
			info, err := os.Lstat(srcPath)
			if err != nil {
				return fmt.Errorf("failed to lstat %q: %w", srcPath, err)
			}

			if info.Mode()&os.ModeSymlink != 0 {
				// Copy symlink
				linkTarget, err := os.Readlink(srcPath)
				if err != nil {
					return fmt.Errorf("failed to read symlink %q: %w", srcPath, err)
				}
				if err := os.Symlink(linkTarget, dstPath); err != nil {
					return fmt.Errorf("failed to create symlink %q: %w", dstPath, err)
				}
			} else {
				// Copy regular file
				if err := copyFile(srcPath, dstPath); err != nil {
					return err
				}
			}
		}
	}

	logging.Debug("copied directory",
		logging.Path(src),
	)

	return nil
}

// detectSourceType determines the type of source for a skill path.
// It checks if the skill's directory (parent of SKILL.md) is a symlink, directory, or file.
func detectSourceType(skillPath string) (SourceType, string) {
	// For SKILL.md files, the skill "root" is the parent directory
	baseName := filepath.Base(skillPath)
	var rootPath string

	if isSkillFile(baseName) {
		rootPath = filepath.Dir(skillPath)
	} else {
		// For legacy .md files, the file itself is the skill root
		rootPath = skillPath
	}

	// Use Lstat to not follow symlinks
	info, err := os.Lstat(rootPath)
	if err != nil {
		// If we can't stat, assume it's a file
		return SourceTypeFile, rootPath
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return SourceTypeSymlink, rootPath
	}

	if info.IsDir() {
		return SourceTypeDirectory, rootPath
	}

	return SourceTypeFile, rootPath
}

// getSymlinkTarget returns the symlink target for a path, or empty string if not a symlink.
func getSymlinkTarget(path string) string {
	target, err := os.Readlink(path)
	if err != nil {
		return ""
	}
	return target
}
