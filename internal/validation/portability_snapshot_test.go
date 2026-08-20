package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestPortabilitySnapshotV2Baseline(t *testing.T) {
	if len(DefaultWantPlatformSupport) != 6 {
		t.Fatalf("implemented harness baseline has %d entries, want 6", len(DefaultWantPlatformSupport))
	}
	if _, oldNamePresent := DefaultWantPlatformSupport["pidev"]; oldNamePresent {
		t.Fatal("deprecated pidev identity must not be part of v2 baseline")
	}
	for platform, source := range DefaultWantSources {
		if source == "" || !strings.HasPrefix(source, "https://") {
			t.Errorf("%s source is not an official HTTPS URL: %q", platform, source)
		}
	}
}

func TestBehaviorReflectedAllowsProseVariation(t *testing.T) {
	if !behaviorReflected("codex documents AGENTS.md chaining and fallback behavior", "AGENTS.md chaining, overrides, and fallback behavior") {
		t.Fatal("expected equivalent prose to satisfy reference check")
	}
	if behaviorReflected("skills are portable", "Pi trust checks and invocation behavior") {
		t.Fatal("unrelated prose must not satisfy reference check")
	}
}

func TestSnapshotVerificationDateWindow(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	base := PortabilitySnapshot{}
	base.Version = 2
	base.VerifiedAt = "2026-08-18"
	base.GeneratedFrom.Narrative = "docs/platforms/portability-assessment.md"
	base.GeneratedFrom.Structured = "docs/platforms/schema.yaml"
	result := &Result{Valid: true}
	verifySnapshotMetadataAt(result, base, now)
	if result.HasErrors() {
		t.Fatalf("fresh date rejected: %v", result.Errors)
	}
	base.VerifiedAt = "2025-01-01"
	result = &Result{Valid: true}
	verifySnapshotMetadataAt(result, base, now)
	if !result.HasErrors() {
		t.Fatal("stale date accepted")
	}
	base.VerifiedAt = "2026-09-01"
	result = &Result{Valid: true}
	verifySnapshotMetadataAt(result, base, now)
	if !result.HasErrors() {
		t.Fatal("future date accepted")
	}
	base.VerifiedAt = "not-a-date"
	result = &Result{Valid: true}
	verifySnapshotMetadataAt(result, base, now)
	if !result.HasErrors() {
		t.Fatal("malformed date accepted")
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
