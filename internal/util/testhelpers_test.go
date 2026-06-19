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

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("CreateTempDir() did not create directory: %s", dir)
	}
}

func TestWriteFile(t *testing.T) {
	dir := CreateTempDir(t)
	path := filepath.Join(dir, "subdir", "test.txt")
	content := "test content"

	WriteFile(t, path, content)

	got := ReadFile(t, path)
	if string(got) != content {
		t.Errorf("file content = %q, want %q", got, content)
	}
}

func TestAssertNoError(t *testing.T) {
	t.Run("passes with nil error", func(t *testing.T) {
		AssertNoError(t, nil)
	})
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
}

func TestGoldenFile(t *testing.T) {
	t.Run("creates golden file in update mode", func(t *testing.T) {
		dir := CreateTempDir(t)
		testdataDir := filepath.Join(dir, "testdata")

		SetUpdateGolden(true)
		defer SetUpdateGolden(false)

		content := "expected output content"
		GoldenFile(t, testdataDir, "test_output", content)

		goldenPath := filepath.Join(testdataDir, "test_output.golden")
		got := ReadFile(t, goldenPath)
		if string(got) != content {
			t.Errorf("golden file content = %q, want %q", got, content)
		}
	})

	t.Run("compares against existing golden file", func(t *testing.T) {
		dir := CreateTempDir(t)
		testdataDir := filepath.Join(dir, "testdata")

		SetUpdateGolden(true)
		content := "matching content"
		GoldenFile(t, testdataDir, "compare_test", content)
		SetUpdateGolden(false)

		GoldenFile(t, testdataDir, "compare_test", content)
	})
}

func TestUpdateGoldenFlag(t *testing.T) {
	original := UpdateGolden()
	defer SetUpdateGolden(original)

	t.Run("default is false", func(t *testing.T) {
		SetUpdateGolden(false)
		if UpdateGolden() {
			t.Error("UpdateGolden() should be false after SetUpdateGolden(false)")
		}
	})

	t.Run("can be set to true", func(t *testing.T) {
		SetUpdateGolden(true)
		if !UpdateGolden() {
			t.Error("UpdateGolden() should be true after SetUpdateGolden(true)")
		}
	})

	t.Run("can be toggled back to false", func(t *testing.T) {
		SetUpdateGolden(true)
		SetUpdateGolden(false)
		if UpdateGolden() {
			t.Error("UpdateGolden() should be false after toggling")
		}
	})
}
