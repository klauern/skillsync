package cli

import (
	"fmt"
	"os"
	"path/filepath"
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
		return fmt.Errorf("failed read source skill: %w", err)
	}

	targetDir := filepath.Dir(targetPath)
	// #nosec G301 - skill directories need to be readable by platform tooling.
	if err := os.MkdirAll(targetDir, 0o750); err != nil {
		return fmt.Errorf("failed create target directory: %w", err)
	}

	// #nosec G306 G703 - targetPath is derived from controlled scope resolution.
	if err := os.WriteFile(targetPath, content, 0o644); err != nil {
		return fmt.Errorf("failed write target skill: %w", err)
	}

	return nil
}
