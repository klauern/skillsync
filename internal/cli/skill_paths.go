package cli

import (
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
