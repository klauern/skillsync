package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/klauern/skillsync/internal/util"
)

const skillDefinitionFile = "SKILL.md"

func skillParentDir(path string) (string, bool) {
	if filepath.Base(path) != skillDefinitionFile {
		return "", false
	}

	parentDir := filepath.Dir(path)
	if parentDir == path || parentDir == "." {
		return "", false
	}

	return parentDir, true
}

func pruneEmptySkillParentDir(path string) {
	parentDir, ok := skillParentDir(path)
	if !ok {
		return
	}

	_ = os.Remove(parentDir)
}

func copySkillFile(sourcePath, targetPath string) error {
	// #nosec G304 - sourcePath comes from parsed skill files.
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source skill: %w", err)
	}

	// #nosec G301 G306 G703 - targetPath is derived from controlled scope resolution.
	if err := util.WriteFileWithPerms(targetPath, content, 0o750, 0o644); err != nil {
		return fmt.Errorf("failed to write target skill: %w", err)
	}

	return nil
}
