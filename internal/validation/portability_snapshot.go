// Package validation provides pre-sync validation checks for skill operations.
package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PortabilitySnapshot represents the machine-readable portability snapshot.
type PortabilitySnapshot struct {
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

// DefaultWantPlatformSupport is the expected platform support status.
var DefaultWantPlatformSupport = map[string]string{
	"claude":  "implemented",
	"cursor":  "implemented",
	"codex":   "implemented",
	"copilot": "reference-only",
	"gemini":  "reference-only",
	"pidev":   "reference-only",
}

// DefaultWantPrecedence is the expected scope precedence per platform.
var DefaultWantPrecedence = map[string][]string{
	"claude":  {"enterprise", "personal", "project"},
	"codex":   {"project", "user", "admin"},
	"copilot": {"personal", "repository", "organization"},
	"cursor":  {"project", "global"},
	"gemini":  {"workspace", "user", "extension"},
	"pidev":   {"user", "project"},
}

// DefaultWantArtifacts is the expected artifact portability entries.
var DefaultWantArtifacts = []string{"skill", "command", "agent", "instructions"}

// FindRepoRoot walks up from the current directory to find the repository root
// by looking for docs/platforms/portability-assessment.md.
func FindRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for {
		assessmentPath := filepath.Join(dir, "docs", "platforms", "portability-assessment.md")
		if _, err := os.Stat(assessmentPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root from %s", dir)
		}
		dir = parent
	}
}

// ValidatePortabilitySnapshot checks the portability snapshot against the
// narrative and structured docs. It returns a Result with all drift findings.
func ValidatePortabilitySnapshot(root string) (*Result, error) {
	result := &Result{Valid: true}

	snapshotPath := filepath.Join(root, "docs", "platforms", "portability-snapshot.yaml")
	assessmentPath := filepath.Join(root, "docs", "platforms", "portability-assessment.md")
	claudePath := filepath.Join(root, "docs", "platforms", "claude.md")
	geminiPath := filepath.Join(root, "docs", "platforms", "gemini.md")
	mappingPath := filepath.Join(root, "docs", "platforms", "cross-platform-mapping.md")
	schemaPath := filepath.Join(root, "docs", "platforms", "schema.yaml")

	// #nosec G304 -- reads only repo-controlled documentation paths.
	snapshotBytes, err := os.ReadFile(snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("read portability snapshot: %w", err)
	}

	// #nosec G304 -- reads only repo-controlled documentation paths.
	assessment, err := os.ReadFile(assessmentPath)
	if err != nil {
		return nil, fmt.Errorf("read portability assessment: %w", err)
	}
	// #nosec G304 -- reads only repo-controlled documentation paths.
	claude, err := os.ReadFile(claudePath)
	if err != nil {
		return nil, fmt.Errorf("read claude platform doc: %w", err)
	}
	// #nosec G304 -- reads only repo-controlled documentation paths.
	gemini, err := os.ReadFile(geminiPath)
	if err != nil {
		return nil, fmt.Errorf("read gemini platform doc: %w", err)
	}
	// #nosec G304 -- reads only repo-controlled documentation paths.
	mapping, err := os.ReadFile(mappingPath)
	if err != nil {
		return nil, fmt.Errorf("read cross-platform mapping: %w", err)
	}
	// #nosec G304 -- reads only repo-controlled documentation paths.
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}

	var snapshot PortabilitySnapshot
	if err := yaml.Unmarshal(snapshotBytes, &snapshot); err != nil {
		return nil, fmt.Errorf("parse portability snapshot: %w", err)
	}

	verifySnapshotMetadata(result, snapshot)
	verifyPlatformSupport(result, snapshot)
	verifyArtifactPortability(result, snapshot)
	verifyPrecedence(result, snapshot)
	verifyDocConsistency(result, snapshot, string(assessment), string(claude), string(gemini), string(mapping), string(schema))

	return result, nil
}

func verifySnapshotMetadata(result *Result, snapshot PortabilitySnapshot) {
	if snapshot.Version != 1 {
		result.AddError(fmt.Errorf("snapshot version = %d, want 1", snapshot.Version))
	}
	if snapshot.GeneratedFrom.Narrative != "docs/platforms/portability-assessment.md" {
		result.AddError(fmt.Errorf("snapshot narrative source = %q, want docs/platforms/portability-assessment.md", snapshot.GeneratedFrom.Narrative))
	}
	if snapshot.GeneratedFrom.Structured != "docs/platforms/schema.yaml" {
		result.AddError(fmt.Errorf("snapshot structured source = %q, want docs/platforms/schema.yaml", snapshot.GeneratedFrom.Structured))
	}
}

func verifyPlatformSupport(result *Result, snapshot PortabilitySnapshot) {
	if len(snapshot.PlatformSupport) != len(DefaultWantPlatformSupport) {
		result.AddError(fmt.Errorf("platform_support entries = %d, want %d", len(snapshot.PlatformSupport), len(DefaultWantPlatformSupport)))
	}
	for platform, wantStatus := range DefaultWantPlatformSupport {
		entry, ok := snapshot.PlatformSupport[platform]
		if !ok {
			result.AddError(fmt.Errorf("platform_support missing %q entry", platform))
			continue
		}
		if entry.Status != wantStatus {
			result.AddError(fmt.Errorf("platform_support[%q].status = %q, want %q", platform, entry.Status, wantStatus))
		}
		if len(entry.ArtifactSurfaces) == 0 {
			result.AddError(fmt.Errorf("platform_support[%q].artifact_surfaces is empty", platform))
		}
	}
}

func verifyArtifactPortability(result *Result, snapshot PortabilitySnapshot) {
	if len(snapshot.ArtifactPortability) != len(DefaultWantArtifacts) {
		result.AddError(fmt.Errorf("artifact_portability entries = %d, want %d", len(snapshot.ArtifactPortability), len(DefaultWantArtifacts)))
	}
	for _, key := range DefaultWantArtifacts {
		entry, ok := snapshot.ArtifactPortability[key]
		if !ok {
			result.AddError(fmt.Errorf("artifact_portability missing %q entry", key))
			continue
		}
		if entry.Portability == "" {
			result.AddError(fmt.Errorf("artifact_portability[%q].portability is empty", key))
		}
		if len(entry.SupportedPlatforms) == 0 {
			result.AddError(fmt.Errorf("artifact_portability[%q].supported_platforms is empty", key))
		}
	}
}

func verifyPrecedence(result *Result, snapshot PortabilitySnapshot) {
	if len(snapshot.Precedence) != len(DefaultWantPrecedence) {
		result.AddError(fmt.Errorf("precedence entries = %d, want %d", len(snapshot.Precedence), len(DefaultWantPrecedence)))
	}
	for platform, wantOrder := range DefaultWantPrecedence {
		gotOrder, ok := snapshot.Precedence[platform]
		if !ok {
			result.AddError(fmt.Errorf("precedence missing %q entry", platform))
			continue
		}
		if strings.Join(gotOrder, ",") != strings.Join(wantOrder, ",") {
			result.AddError(fmt.Errorf("precedence[%q] = %v, want %v", platform, gotOrder, wantOrder))
		}
	}
}

func verifyDocConsistency(result *Result, snapshot PortabilitySnapshot, assessment, claude, gemini, mapping, schema string) {
	assessmentLower := strings.ToLower(assessment)
	claudeLower := strings.ToLower(claude)
	geminiLower := strings.ToLower(gemini)
	mappingLower := strings.ToLower(mapping)
	schemaLower := strings.ToLower(schema)

	if !strings.Contains(assessmentLower, "docs/platforms/portability-snapshot.yaml") {
		result.AddError(fmt.Errorf("portability assessment does not reference docs/platforms/portability-snapshot.yaml"))
	}

	comparisonText := assessmentLower + "\n" + claudeLower + "\n" + geminiLower + "\n" + mappingLower
	for platform, support := range snapshot.PlatformSupport {
		if !strings.Contains(comparisonText, platform) {
			result.AddError(fmt.Errorf("platform_support platform %q is not reflected in the docs", platform))
		}
		if !strings.Contains(comparisonText, strings.ToLower(support.Status)) {
			result.AddError(fmt.Errorf("platform_support[%q].status %q is not reflected in the docs", platform, support.Status))
		}
	}
	for _, behavior := range snapshot.NonportableBehaviors {
		if !strings.Contains(comparisonText, strings.ToLower(behavior)) {
			result.AddError(fmt.Errorf("nonportable behavior %q is not reflected in the docs", behavior))
		}
	}
	for _, mapping := range snapshot.LossyFieldMappings {
		if !strings.Contains(comparisonText, strings.ToLower(mapping.Field)) {
			result.AddError(fmt.Errorf("lossy field %q is not reflected in the docs", mapping.Field))
		}
	}

	if !strings.Contains(geminiLower, "first-pass sync boundary") {
		result.AddError(fmt.Errorf("gemini platform doc must include an explicit first-pass sync boundary section"))
	}
	if !strings.Contains(geminiLower, "skills, commands, and context") {
		result.AddError(fmt.Errorf("gemini platform doc must state the first-pass syncable subset"))
	}
	if !strings.Contains(geminiLower, "metadata only where safe") {
		result.AddError(fmt.Errorf("gemini platform doc must explain metadata-only preservation for extension fields"))
	}
	if !strings.Contains(geminiLower, "not first-pass sync targets") {
		result.AddError(fmt.Errorf("gemini platform doc must call out extension-only runtime surfaces as non-goals"))
	}
	if !strings.Contains(schemaLower, "first_pass_sync:") {
		result.AddError(fmt.Errorf("schema.yaml must include a first_pass_sync section for gemini"))
	}
	if !strings.Contains(schemaLower, "metadata_only_surfaces:") {
		result.AddError(fmt.Errorf("schema.yaml must describe metadata-only Gemini extension surfaces"))
	}
	if !strings.Contains(schemaLower, "unsupported_runtime_surfaces:") {
		result.AddError(fmt.Errorf("schema.yaml must describe unsupported Gemini runtime surfaces"))
	}
}
