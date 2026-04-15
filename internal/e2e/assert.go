package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AssertSuccess fails the test if the command did not succeed.
func AssertSuccess(t *testing.T, r *Result) {
	t.Helper()
	if !r.Success() {
		t.Fatalf("expected success, got error: %v\nstdout: %s", r.Err, r.Stdout)
	}
}

// AssertError fails the test if the command did not return an error.
func AssertError(t *testing.T, r *Result) {
	t.Helper()
	if r.Success() {
		t.Fatalf("expected error, but command succeeded\nstdout: %s", r.Stdout)
	}
}

// AssertExitCode fails the test if the exit code doesn't match.
func AssertExitCode(t *testing.T, r *Result, expected int) {
	t.Helper()
	if r.ExitCode != expected {
		t.Errorf("expected exit code %d, got %d\nerror: %v\nstdout: %s", expected, r.ExitCode, r.Err, r.Stdout)
	}
}

// AssertOutputContains fails the test if stdout doesn't contain the substring.
func AssertOutputContains(t *testing.T, r *Result, substr string) {
	t.Helper()
	if !strings.Contains(r.Stdout, substr) {
		t.Errorf("expected output to contain %q\ngot: %s", substr, r.Stdout)
	}
}

// AssertOutputNotContains fails the test if stdout contains the substring.
func AssertOutputNotContains(t *testing.T, r *Result, substr string) {
	t.Helper()
	if strings.Contains(r.Stdout, substr) {
		t.Errorf("expected output to NOT contain %q\ngot: %s", substr, r.Stdout)
	}
}

// AssertOutputEquals fails the test if stdout doesn't match exactly.
func AssertOutputEquals(t *testing.T, r *Result, expected string) {
	t.Helper()
	if r.Stdout != expected {
		t.Errorf("output mismatch\nexpected: %q\ngot: %q", expected, r.Stdout)
	}
}

// AssertOutputMatches compares output against a golden file.
// It uses the same golden file pattern as the util package.
func AssertOutputMatches(t *testing.T, r *Result, testdataDir, name string) {
	t.Helper()
	goldenPath := filepath.Join(testdataDir, name+".golden")

	if UpdateGolden() {
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("failed to create golden directory: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(r.Stdout), 0o600); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		return
	}

	// #nosec G304 - goldenPath is constructed from trusted testdata directory and test name
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v\nRun with -update to create it", goldenPath, err)
	}

	if r.Stdout != string(want) {
		t.Errorf("output mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, r.Stdout, string(want))
	}
}

// AssertFileExists fails the test if the file doesn't exist.
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", path)
	}
}

// AssertFileNotExists fails the test if the file exists.
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected file to NOT exist: %s", path)
	}
}

// AssertFileContains fails the test if the file doesn't contain the substring.
func AssertFileContains(t *testing.T, path, substr string) {
	t.Helper()
	// #nosec G304 - path is provided by test code
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	if !strings.Contains(string(data), substr) {
		t.Errorf("expected file %s to contain %q\ngot: %s", path, substr, string(data))
	}
}

// AssertFileEquals fails the test if the file content doesn't match exactly.
func AssertFileEquals(t *testing.T, path, expected string) {
	t.Helper()
	// #nosec G304 - path is provided by test code
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	if string(data) != expected {
		t.Errorf("file content mismatch for %s\nexpected: %q\ngot: %q", path, expected, string(data))
	}
}

// AssertErrorContains fails the test if the error message doesn't contain the substring.
func AssertErrorContains(t *testing.T, r *Result, substr string) {
	t.Helper()
	if r.Success() {
		t.Fatalf("expected error containing %q, but command succeeded", substr)
	}
	errMsg := r.Err.Error()
	if !strings.Contains(errMsg, substr) {
		t.Errorf("expected error to contain %q\ngot: %s", substr, errMsg)
	}
}

// updateGoldenFlag tracks whether to update golden files
var updateGoldenFlag = false

// SetUpdateGolden sets the update golden flag (call from TestMain)
func SetUpdateGolden(update bool) {
	updateGoldenFlag = update
}

// UpdateGolden returns whether golden files should be updated
func UpdateGolden() bool {
	return updateGoldenFlag
}
