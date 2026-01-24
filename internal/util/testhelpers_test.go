// Package util provides tests for utility functions.
//
//nolint:revive // var-naming - package name is meaningful
package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateTempDir(t *testing.T) {
	dir := CreateTempDir(t)

	// Verify directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("CreateTempDir() did not create directory: %s", dir)
	}

	// Directory should be cleaned up after test automatically
}

func TestWriteFile(t *testing.T) {
	dir := CreateTempDir(t)
	path := filepath.Join(dir, "subdir", "test.txt")
	content := "test content"

	WriteFile(t, path, content)

	// Verify file exists and has correct content
	got, err := os.ReadFile(path) //nolint:gosec // G304 - safe in test code using temp directory
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(got) != content {
		t.Errorf("file content = %q, want %q", got, content)
	}
}
