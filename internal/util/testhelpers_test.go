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

func TestAssertNoError(t *testing.T) {
	t.Run("passes with nil error", func(t *testing.T) {
		// AssertNoError should not fail when given nil error
		// We can't directly test that it doesn't call t.Fatalf,
		// but we can verify it completes without issue
		AssertNoError(t, nil)
	})

	// Note: We cannot easily test the failure case since t.Fatalf
	// would terminate the test. The behavior is validated by usage
	// throughout the codebase.
}

func TestAssertEqual(t *testing.T) {
	t.Run("passes with equal strings", func(t *testing.T) {
		AssertEqual(t, "hello", "hello")
	})

	t.Run("passes with equal integers", func(t *testing.T) {
		AssertEqual(t, 42, 42)
	})

	t.Run("passes with equal booleans", func(t *testing.T) {
		AssertEqual(t, true, true)
	})

	// Note: We cannot easily test the failure case since t.Errorf
	// would fail the test. The behavior is validated by usage
	// throughout the codebase.
}

func TestGoldenFile(t *testing.T) {
	t.Run("creates golden file in update mode", func(t *testing.T) {
		dir := CreateTempDir(t)
		testdataDir := filepath.Join(dir, "testdata")

		// Enable update mode
		SetUpdateGolden(true)
		defer SetUpdateGolden(false)

		content := "expected output content"
		GoldenFile(t, testdataDir, "test_output", content)

		// Verify golden file was created
		goldenPath := filepath.Join(testdataDir, "test_output.golden")
		got, err := os.ReadFile(goldenPath) //nolint:gosec // G304 - safe in test
		if err != nil {
			t.Fatalf("golden file was not created: %v", err)
		}

		if string(got) != content {
			t.Errorf("golden file content = %q, want %q", got, content)
		}
	})

	t.Run("compares against existing golden file", func(t *testing.T) {
		dir := CreateTempDir(t)
		testdataDir := filepath.Join(dir, "testdata")

		// First, create the golden file
		SetUpdateGolden(true)
		content := "matching content"
		GoldenFile(t, testdataDir, "compare_test", content)
		SetUpdateGolden(false)

		// Now verify comparison mode works
		GoldenFile(t, testdataDir, "compare_test", content)
	})
}

func TestUpdateGoldenFlag(t *testing.T) {
	// Save original state
	original := UpdateGolden()
	defer SetUpdateGolden(original)

	t.Run("default is false", func(t *testing.T) {
		SetUpdateGolden(false)
		if UpdateGolden() != false {
			t.Error("UpdateGolden() should be false after SetUpdateGolden(false)")
		}
	})

	t.Run("can be set to true", func(t *testing.T) {
		SetUpdateGolden(true)
		if UpdateGolden() != true {
			t.Error("UpdateGolden() should be true after SetUpdateGolden(true)")
		}
	})

	t.Run("can be toggled back to false", func(t *testing.T) {
		SetUpdateGolden(true)
		SetUpdateGolden(false)
		if UpdateGolden() != false {
			t.Error("UpdateGolden() should be false after toggling")
		}
	})
}
