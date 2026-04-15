//nolint:revive // var-naming - package name is meaningful
package util

import (
	"os"
	"path/filepath"
	"testing"
)

// CreateTempDir creates a temporary directory for testing
func CreateTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "skillsync-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}

// WriteFile writes content to a file in the test directory
func WriteFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertEqual fails if got != want
func AssertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// GoldenFile compares got against a golden file and updates it if -update flag is set.
// The golden file path is relative to the testdata directory.
func GoldenFile(t *testing.T, testdataDir, name, got string) {
	t.Helper()
	goldenPath := filepath.Join(testdataDir, name+".golden")

	if UpdateGolden() {
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("failed to create golden directory: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o600); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		return
	}

	// #nosec G304 - goldenPath is constructed from trusted testdata directory and test name
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v\nRun with -update to create it", goldenPath, err)
	}

	if got != string(want) {
		t.Errorf("output mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, string(want))
	}
}

// updateGolden is set via -update flag
var updateGoldenFlag = false

// SetUpdateGolden sets the update golden flag (call from TestMain)
func SetUpdateGolden(update bool) {
	updateGoldenFlag = update
}

// UpdateGolden returns whether golden files should be updated
func UpdateGolden() bool {
	return updateGoldenFlag
}
