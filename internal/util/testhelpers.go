//nolint:revive // var-naming - package name is meaningful
package util

import (
	"os"
	"path/filepath"
	"testing"
)

// CreateTempDir creates a temporary directory for testing.
func CreateTempDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "skillsync-test-*")
	if err != nil {
		t.Fatalf("failed create temp dir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	return dir
}

// WriteFile writes content to a file in a test directory.
func WriteFile(t *testing.T, path, content string) {
	t.Helper()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("failed create directory: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed write file: %v", err)
	}
}

// ReadTestFile reads a caller-controlled fixture or temp file in test code.
func ReadTestFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path) //nolint:gosec // G304 - test helper only reads caller-controlled fixture/temp paths
	if err != nil {
		t.Fatalf("failed read file: %v", err)
	}

	return data
}

// AssertNoError fails the test if err is not nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertEqual fails if got != want.
func AssertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// GoldenFile compares got against a golden file or updates it when -update is set.
func GoldenFile(t *testing.T, testdataDir, name, got string) {
	t.Helper()

	goldenPath := filepath.Join(testdataDir, name+".golden")
	if UpdateGolden() {
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("failed create golden directory: %v", err)
		}

		if err := os.WriteFile(goldenPath, []byte(got), 0o600); err != nil {
			t.Fatalf("failed write golden file: %v", err)
		}

		return
	}

	want := ReadTestFile(t, goldenPath)
	if got != string(want) {
		t.Errorf("output mismatch %s\n--- got ---\n%s\n--- want ---\n%s", name, got, string(want))
	}
}

// updateGoldenFlag is set via the -update flag.
var updateGoldenFlag bool

// SetUpdateGolden sets the update golden flag, typically from TestMain.
func SetUpdateGolden(update bool) {
	updateGoldenFlag = update
}

// UpdateGolden reports whether golden files should be updated.
func UpdateGolden() bool {
	return updateGoldenFlag
}
