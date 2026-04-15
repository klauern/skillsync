package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type portabilitySnapshot struct {
	Version       int `yaml:"version"`
	GeneratedFrom struct {
		Narrative  string `yaml:"narrative"`
		Structured string `yaml:"structured"`
	} `yaml:"generated_from"`
	PlatformSupport map[string]struct {
		Status           string   `yaml:"status"`
		ArtifactSurfaces []string `yaml:"artifact_surfaces"`
		Notes            []string `yaml:"notes"`
	} `yaml:"platform_support"`
	ArtifactPortability map[string]struct {
		Portability        string   `yaml:"portability"`
		Description        string   `yaml:"description"`
		SupportedPlatforms []string `yaml:"supported_platforms"`
		Notes              []string `yaml:"notes"`
	} `yaml:"artifact_portability"`
	Precedence         map[string][]string `yaml:"precedence"`
	LossyFieldMappings []struct {
		Field         string   `yaml:"field"`
		SupportedBy   []string `yaml:"supported_by"`
		UnsupportedBy []string `yaml:"unsupported_by"`
		Behavior      string   `yaml:"behavior"`
	} `yaml:"lossy_field_mappings"`
	NonportableBehaviors []string `yaml:"nonportable_behaviors"`
}

func TestPortabilitySnapshotFreshness(t *testing.T) {
	root := findRepoRoot(t)

	snapshotPath := filepath.Join(root, "docs", "platforms", "portability-snapshot.yaml")
	assessmentPath := filepath.Join(root, "docs", "platforms", "portability-assessment.md")
	claudePath := filepath.Join(root, "docs", "platforms", "claude.md")
	geminiPath := filepath.Join(root, "docs", "platforms", "gemini.md")
	mappingPath := filepath.Join(root, "docs", "platforms", "cross-platform-mapping.md")
	schemaPath := filepath.Join(root, "docs", "platforms", "schema.yaml")

	snapshotBytes := readFile(t, snapshotPath)
	assessment := strings.ToLower(string(readFile(t, assessmentPath)))
	claude := strings.ToLower(string(readFile(t, claudePath)))
	gemini := strings.ToLower(string(readFile(t, geminiPath)))
	mapping := strings.ToLower(string(readFile(t, mappingPath)))
	schema := strings.ToLower(string(readFile(t, schemaPath)))

	var snapshot portabilitySnapshot
	if err := yaml.Unmarshal(snapshotBytes, &snapshot); err != nil {
		t.Fatalf("parse portability snapshot: %v", err)
	}

	verifySnapshotMetadata(t, snapshot)
	verifyPlatformSupport(t, snapshot)
	verifyArtifactPortability(t, snapshot)
	verifyPrecedence(t, snapshot)
	verifyDocConsistency(t, snapshot, assessment, claude, gemini, mapping, schema)
}

func verifySnapshotMetadata(t *testing.T, snapshot portabilitySnapshot) {
	t.Helper()

	if snapshot.Version != 1 {
		t.Fatalf("snapshot version = %d, want 1", snapshot.Version)
	}
	if snapshot.GeneratedFrom.Narrative != "docs/platforms/portability-assessment.md" {
		t.Fatalf("snapshot narrative source = %q, want docs/platforms/portability-assessment.md", snapshot.GeneratedFrom.Narrative)
	}
	if snapshot.GeneratedFrom.Structured != "docs/platforms/schema.yaml" {
		t.Fatalf("snapshot structured source = %q, want docs/platforms/schema.yaml", snapshot.GeneratedFrom.Structured)
	}
}

func verifyPlatformSupport(t *testing.T, snapshot portabilitySnapshot) {
	t.Helper()

	wantPlatformSupport := map[string]string{
		"claude":  "implemented",
		"cursor":  "implemented",
		"codex":   "implemented",
		"copilot": "reference-only",
		"gemini":  "reference-only",
		"pidev":   "reference-only",
	}
	if len(snapshot.PlatformSupport) != len(wantPlatformSupport) {
		t.Fatalf("platform_support entries = %d, want %d", len(snapshot.PlatformSupport), len(wantPlatformSupport))
	}
	for platform, wantStatus := range wantPlatformSupport {
		entry, ok := snapshot.PlatformSupport[platform]
		if !ok {
			t.Fatalf("platform_support missing %q entry", platform)
		}
		if entry.Status != wantStatus {
			t.Fatalf("platform_support[%q].status = %q, want %q", platform, entry.Status, wantStatus)
		}
		if len(entry.ArtifactSurfaces) == 0 {
			t.Fatalf("platform_support[%q].artifact_surfaces is empty", platform)
		}
	}
}

func verifyArtifactPortability(t *testing.T, snapshot portabilitySnapshot) {
	t.Helper()

	wantArtifacts := []string{"skill", "command", "agent", "instructions"}
	if len(snapshot.ArtifactPortability) != len(wantArtifacts) {
		t.Fatalf("artifact_portability entries = %d, want %d", len(snapshot.ArtifactPortability), len(wantArtifacts))
	}
	for _, key := range wantArtifacts {
		entry, ok := snapshot.ArtifactPortability[key]
		if !ok {
			t.Fatalf("artifact_portability missing %q entry", key)
		}
		if entry.Portability == "" {
			t.Fatalf("artifact_portability[%q].portability is empty", key)
		}
		if len(entry.SupportedPlatforms) == 0 {
			t.Fatalf("artifact_portability[%q].supported_platforms is empty", key)
		}
	}
}

func verifyPrecedence(t *testing.T, snapshot portabilitySnapshot) {
	t.Helper()

	wantPrecedence := map[string][]string{
		"claude":  {"enterprise", "personal", "project"},
		"codex":   {"project", "user", "admin"},
		"copilot": {"personal", "repository", "organization"},
		"cursor":  {"project", "global"},
		"gemini":  {"workspace", "user", "extension"},
		"pidev":   {"user", "project"},
	}
	if len(snapshot.Precedence) != len(wantPrecedence) {
		t.Fatalf("precedence entries = %d, want %d", len(snapshot.Precedence), len(wantPrecedence))
	}
	for platform, wantOrder := range wantPrecedence {
		gotOrder, ok := snapshot.Precedence[platform]
		if !ok {
			t.Fatalf("precedence missing %q entry", platform)
		}
		if strings.Join(gotOrder, ",") != strings.Join(wantOrder, ",") {
			t.Fatalf("precedence[%q] = %v, want %v", platform, gotOrder, wantOrder)
		}
	}
}

func verifyDocConsistency(t *testing.T, snapshot portabilitySnapshot, assessment, claude, gemini, mapping, schema string) {
	t.Helper()

	if !strings.Contains(assessment, "docs/platforms/portability-snapshot.yaml") {
		t.Fatalf("portability assessment does not reference docs/platforms/portability-snapshot.yaml")
	}

	comparisonText := assessment + "\n" + claude + "\n" + gemini + "\n" + mapping
	for platform, support := range snapshot.PlatformSupport {
		if !strings.Contains(comparisonText, platform) {
			t.Fatalf("platform_support platform %q is not reflected in the docs", platform)
		}
		if !strings.Contains(comparisonText, strings.ToLower(support.Status)) {
			t.Fatalf("platform_support[%q].status %q is not reflected in the docs", platform, support.Status)
		}
	}
	for _, behavior := range snapshot.NonportableBehaviors {
		if !strings.Contains(comparisonText, strings.ToLower(behavior)) {
			t.Fatalf("nonportable behavior %q is not reflected in the docs", behavior)
		}
	}
	for _, mapping := range snapshot.LossyFieldMappings {
		if !strings.Contains(comparisonText, strings.ToLower(mapping.Field)) {
			t.Fatalf("lossy field %q is not reflected in the docs", mapping.Field)
		}
	}

	if !strings.Contains(gemini, "first-pass sync boundary") {
		t.Fatalf("gemini platform doc must include an explicit first-pass sync boundary section")
	}
	if !strings.Contains(gemini, "skills, commands, and context") {
		t.Fatalf("gemini platform doc must state the first-pass syncable subset")
	}
	if !strings.Contains(gemini, "metadata only where safe") {
		t.Fatalf("gemini platform doc must explain metadata-only preservation for extension fields")
	}
	if !strings.Contains(gemini, "not first-pass sync targets") {
		t.Fatalf("gemini platform doc must call out extension-only runtime surfaces as non-goals")
	}
	if !strings.Contains(schema, "first_pass_sync:") {
		t.Fatalf("schema.yaml must include a first_pass_sync section for gemini")
	}
	if !strings.Contains(schema, "metadata_only_surfaces:") {
		t.Fatalf("schema.yaml must describe metadata-only Gemini extension surfaces")
	}
	if !strings.Contains(schema, "unsupported_runtime_surfaces:") {
		t.Fatalf("schema.yaml must describe unsupported Gemini runtime surfaces")
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

func readFile(t *testing.T, path string) []byte {
	t.Helper()

	// #nosec G304 -- test helper reads only repo-controlled documentation paths.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}
