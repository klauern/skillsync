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
	mappingPath := filepath.Join(root, "docs", "platforms", "cross-platform-mapping.md")

	snapshotBytes := readFile(t, snapshotPath)
	assessment := strings.ToLower(string(readFile(t, assessmentPath)))
	claude := strings.ToLower(string(readFile(t, claudePath)))
	mapping := strings.ToLower(string(readFile(t, mappingPath)))

	var snapshot portabilitySnapshot
	if err := yaml.Unmarshal(snapshotBytes, &snapshot); err != nil {
		t.Fatalf("parse portability snapshot: %v", err)
	}

	if snapshot.Version != 1 {
		t.Fatalf("snapshot version = %d, want 1", snapshot.Version)
	}
	if snapshot.GeneratedFrom.Narrative != "docs/platforms/portability-assessment.md" {
		t.Fatalf("snapshot narrative source = %q, want docs/platforms/portability-assessment.md", snapshot.GeneratedFrom.Narrative)
	}
	if snapshot.GeneratedFrom.Structured != "docs/platforms/schema.yaml" {
		t.Fatalf("snapshot structured source = %q, want docs/platforms/schema.yaml", snapshot.GeneratedFrom.Structured)
	}

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

	wantPrecedence := map[string][]string{
		"claude":  {"enterprise", "personal", "project"},
		"codex":   {"project", "user", "admin"},
		"copilot": {"personal", "repository", "organization"},
		"cursor":  {"project", "global"},
		"gemini":  {"workspace", "user", "extension"},
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

	if !strings.Contains(assessment, "docs/platforms/portability-snapshot.yaml") {
		t.Fatalf("portability assessment does not reference docs/platforms/portability-snapshot.yaml")
	}

	comparisonText := assessment + "\n" + claude + "\n" + mapping
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

	// #nosec G304 -- test helper reads repository-controlled fixture/doc paths.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}
