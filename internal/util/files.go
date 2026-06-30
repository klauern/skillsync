package util

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFileWithPerms ensures the parent directory exists before writing a file.
func WriteFileWithPerms(path string, content []byte, dirPerm, filePerm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	if err := os.WriteFile(path, content, filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
