package validation

import (
	"os"
	"path/filepath"
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
		if contains(msg, "snapshot version") || contains(msg, "snapshot narrative source") || contains(msg, "snapshot structured source") {
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
		if contains(msg, "platform_support") {
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
		if contains(msg, "artifact_portability") {
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
		if contains(msg, "precedence") {
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
		if contains(msg, "not reflected in the docs") || contains(msg, "must include") || contains(msg, "must state") || contains(msg, "must explain") || contains(msg, "must call out") || contains(msg, "must describe") {
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

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
