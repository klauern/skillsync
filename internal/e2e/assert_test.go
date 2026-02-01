package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAssertHelpers(t *testing.T) {
	r := &Result{Stdout: "ok", Err: nil, ExitCode: 0}

	AssertSuccess(t, r)
	AssertExitCode(t, r, 0)
	AssertOutputEquals(t, r, "ok")
}

func TestAssertFileEquals(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	AssertFileEquals(t, path, "content")
}
