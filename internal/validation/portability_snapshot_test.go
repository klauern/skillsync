package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPortabilitySnapshotFreshness(t *testing.T) {
	root := findRepoRoot(t)

	result, err := ValidatePortabilitySnapshot(root)
	if err != nil {
		t.Fatalf("validate portability snapshot: %v", err)
	}

	if result.HasErrors() {
		for _, e := range result.Errors {
			t.Error(e)
		}
	}
}

func TestValidatePortabilitySnapshot_MetadataDrift(t *testing.T) {
	root := findRepoRoot(t)
	result, err := ValidatePortabilitySnapshot(root)
	if err != nil {
		t.Fatalf("validate portability snapshot: %v", err)
	}

	// The real repo snapshot should pass metadata checks.
	for _, e := range result.Errors {
		msg := e.Error()
		if strings.Contains(msg, "snapshot version") || strings.Contains(msg, "snapshot narrative source") || strings.Contains(msg, "snapshot structured source") {
			t.Errorf("metadata drift: %v", e)
		}
	}
}

func TestValidatePortabilitySnapshot_PlatformSupport(t *testing.T) {
	root := findRepoRoot(t)
	result, err := ValidatePortabilitySnapshot(root)
	if err != nil {
		t.Fatalf("validate portability snapshot: %v", err)
	}

	for _, e := range result.Errors {
		msg := e.Error()
		if strings.Contains(msg, "platform_support") {
			t.Errorf("platform support drift: %v", e)
		}
	}
}

func TestValidatePortabilitySnapshot_ArtifactPortability(t *testing.T) {
	root := findRepoRoot(t)
	result, err := ValidatePortabilitySnapshot(root)
	if err != nil {
		t.Fatalf("validate portability snapshot: %v", err)
	}

	for _, e := range result.Errors {
		msg := e.Error()
		if strings.Contains(msg, "artifact_portability") {
			t.Errorf("artifact portability drift: %v", e)
		}
	}
}

func TestValidatePortabilitySnapshot_Precedence(t *testing.T) {
	root := findRepoRoot(t)
	result, err := ValidatePortabilitySnapshot(root)
	if err != nil {
		t.Fatalf("validate portability snapshot: %v", err)
	}

	for _, e := range result.Errors {
		msg := e.Error()
		if strings.Contains(msg, "precedence") {
			t.Errorf("precedence drift: %v", e)
		}
	}
}

func TestValidatePortabilitySnapshot_DocConsistency(t *testing.T) {
	root := findRepoRoot(t)
	result, err := ValidatePortabilitySnapshot(root)
	if err != nil {
		t.Fatalf("validate portability snapshot: %v", err)
	}

	for _, e := range result.Errors {
		msg := e.Error()
		if strings.Contains(msg, "not reflected in the docs") || strings.Contains(msg, "must include") || strings.Contains(msg, "must state") || strings.Contains(msg, "must explain") || strings.Contains(msg, "must call out") || strings.Contains(msg, "must describe") {
			t.Errorf("doc consistency drift: %v", e)
		}
	}
}

func TestFindRepoRoot(t *testing.T) {
	root, err := FindRepoRoot()
	if err != nil {
		t.Fatalf("FindRepoRoot: %v", err)
	}

	// Verify the root contains expected docs.
	if _, err := os.Stat(filepath.Join(root, "docs", "platforms", "portability-assessment.md")); err != nil {
		t.Fatalf("expected docs in repo root %s: %v", root, err)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	for {
		assessmentPath := filepath.Join(dir, "docs", "platforms", "portability-assessment.md")
		if _, err := os.Stat(assessmentPath); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repository root from %s", dir)
		}
		dir = parent
	}
}
