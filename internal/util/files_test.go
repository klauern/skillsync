package util

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileWithPermsCreatesParentDirectory(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "nested", "skill", "SKILL.md")

	if err := WriteFileWithPerms(targetPath, []byte("content"), 0o750, 0o640); err != nil {
		t.Fatalf("WriteFileWithPerms() error = %v", err)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "content" {
		t.Fatalf("file content = %q, want %q", string(data), "content")
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("file mode = %#o, want %#o", info.Mode().Perm(), 0o640)
	}
}

func TestWriteFileWithPermsReportsDirectoryCreationFailure(t *testing.T) {
	baseDir := t.TempDir()
	blockingFile := filepath.Join(baseDir, "blocked")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	targetPath := filepath.Join(blockingFile, "skill.md")
	err := WriteFileWithPerms(targetPath, []byte("content"), 0o750, 0o644)
	if err == nil {
		t.Fatal("WriteFileWithPerms() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "failed to create parent directory") {
		t.Fatalf("error = %q, want parent directory context", err)
	}
	if !errors.Is(err, os.ErrExist) && !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("error = %q, want directory creation failure", err)
	}
}
