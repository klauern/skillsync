// Package validation provides pre-sync validation checks for skill operations.
package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// PortabilitySnapshot represents the machine-readable portability snapshot.
type PortabilitySnapshot struct {
	Version       int    `yaml:"version"`
	VerifiedAt    string `yaml:"verified_at"`
	GeneratedFrom struct {
		Narrative  string `yaml:"narrative"`
		Structured string `yaml:"structured"`
	} `yaml:"generated_from"`
	PlatformSupport map[string]struct {
		Status                     string   `yaml:"status"`
		ImplementationStatus       string   `yaml:"implementation_status"`
		VerifiedAt                 string   `yaml:"verified_at"`
		OfficialSources            []string `yaml:"official_sources"`
		ObservedVersion            string   `yaml:"observed_local_version"`
		ArtifactSurfaces           []string `yaml:"artifact_surfaces"`
		HarnessCapabilities        []string `yaml:"harness_capabilities"`
		ImplementedCapabilities    []string `yaml:"implemented_capabilities"`
		DocumentedOnlyCapabilities []string `yaml:"documented_only_capabilities"`
		NativeOnlyCapabilities     []string `yaml:"native_only_capabilities"`
		Notes                      []string `yaml:"notes"`
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
	"copilot": "implemented",
	"gemini":  "implemented",
	"pi":      "implemented",
}

// DefaultWantPrecedence is the expected scope precedence per platform.
var DefaultWantPrecedence = map[string][]string{
	"claude":  {"enterprise", "personal", "project"},
	"codex":   {"project", "user", "admin"},
	"copilot": {"personal", "repository", "organization"},
	"cursor":  {"project", "global"},
	"gemini":  {"workspace", "user", "extension"},
	"pi":      {"project", "user"},
}

// DefaultWantSources maps each platform to its canonical documentation source.
var DefaultWantSources = map[string]string{
	"claude":  "https://code.claude.com/docs/en/skills",
	"codex":   "https://developers.openai.com/codex/skills",
	"cursor":  "https://cursor.com/docs/skills",
	"copilot": "https://docs.github.com/en/copilot/concepts/agents/about-agent-skills",
	"gemini":  "https://geminicli.com/docs/cli/skills/",
	"pi":      "https://pi.dev/docs/latest/skills",
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
	return ValidatePortabilitySnapshotAt(root, time.Now())
}

// ValidatePortabilitySnapshotAt validates a snapshot against a deterministic
// reference time. Snapshot verification expires after 180 days.
func ValidatePortabilitySnapshotAt(root string, now time.Time) (*Result, error) {
	result := &Result{Valid: true}

	snapshotPath := filepath.Join(root, "docs", "platforms", "portability-snapshot.yaml")
	assessmentPath := filepath.Join(root, "docs", "platforms", "portability-assessment.md")
	claudePath := filepath.Join(root, "docs", "platforms", "claude.md")
	geminiPath := filepath.Join(root, "docs", "platforms", "gemini.md")
	piPath := filepath.Join(root, "docs", "platforms", "pi.md")
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
	pi, err := os.ReadFile(piPath)
	if err != nil {
		return nil, fmt.Errorf("read pi platform doc: %w", err)
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

	verifySnapshotMetadataAt(result, snapshot, now)
	verifyPlatformSupport(result, snapshot, now)
	verifyArtifactPortability(result, snapshot)
	verifyPrecedence(result, snapshot)
	verifyDocConsistency(result, snapshot, string(assessment), string(claude), string(gemini), string(pi), string(mapping), string(schema))

	return result, nil
}

func verifySnapshotMetadataAt(result *Result, snapshot PortabilitySnapshot, now time.Time) {
	if snapshot.Version != 2 {
		result.AddError(fmt.Errorf("snapshot version = %d, want 2", snapshot.Version))
	}
	verified, err := time.Parse("2006-01-02", snapshot.VerifiedAt)
	if err != nil {
		result.AddError(fmt.Errorf("snapshot verified_at %q is not YYYY-MM-DD", snapshot.VerifiedAt))
		return
	}
	if verified.After(now.UTC()) {
		result.AddError(fmt.Errorf("snapshot verified_at %q is in the future", snapshot.VerifiedAt))
	}
	if now.UTC().Sub(verified) > 180*24*time.Hour {
		result.AddError(fmt.Errorf("snapshot verified_at %q is older than 180 days", snapshot.VerifiedAt))
	}
	if snapshot.GeneratedFrom.Narrative != "docs/platforms/portability-assessment.md" {
		result.AddError(fmt.Errorf("snapshot narrative source = %q, want docs/platforms/portability-assessment.md", snapshot.GeneratedFrom.Narrative))
	}
	if snapshot.GeneratedFrom.Structured != "docs/platforms/schema.yaml" {
		result.AddError(fmt.Errorf("snapshot structured source = %q, want docs/platforms/schema.yaml", snapshot.GeneratedFrom.Structured))
	}
}

func verifyPlatformSupport(result *Result, snapshot PortabilitySnapshot, now time.Time) {
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
		if entry.ImplementationStatus != wantStatus {
			result.AddError(fmt.Errorf("platform_support[%q].implementation_status = %q, want %q", platform, entry.ImplementationStatus, wantStatus))
		}
		entryDate, dateErr := time.Parse("2006-01-02", entry.VerifiedAt)
		if dateErr != nil {
			result.AddError(fmt.Errorf("platform_support[%q].verified_at %q is invalid", platform, entry.VerifiedAt))
		} else if entryDate.After(now.UTC()) || now.UTC().Sub(entryDate) > 180*24*time.Hour {
			result.AddError(fmt.Errorf("platform_support[%q].verified_at is stale or future", platform))
		}
		if len(entry.OfficialSources) == 0 || entry.OfficialSources[0] != DefaultWantSources[platform] {
			result.AddError(fmt.Errorf("platform_support[%q].official_sources must include %q", platform, DefaultWantSources[platform]))
		}
		if platform == "claude" && entry.ObservedVersion != "2.1.234" || platform == "codex" && entry.ObservedVersion != "0.147.0" || platform == "pi" && entry.ObservedVersion != "0.84.2" {
			result.AddError(fmt.Errorf("platform_support[%q].observed_local_version is stale or incorrect", platform))
		}
		if len(entry.ArtifactSurfaces) == 0 {
			result.AddError(fmt.Errorf("platform_support[%q].artifact_surfaces is empty", platform))
		}
		if len(entry.HarnessCapabilities) == 0 || len(entry.ImplementedCapabilities) == 0 {
			result.AddError(fmt.Errorf("platform_support[%q] must separate harness and implemented capabilities", platform))
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

func verifyDocConsistency(result *Result, snapshot PortabilitySnapshot, assessment, claude, gemini, pi, mapping, schema string) {
	assessmentLower := strings.ToLower(assessment)
	claudeLower := strings.ToLower(claude)
	geminiLower := strings.ToLower(gemini)
	piLower := strings.ToLower(pi)
	mappingLower := strings.ToLower(mapping)
	schemaLower := strings.ToLower(schema)

	if !strings.Contains(assessmentLower, "docs/platforms/portability-snapshot.yaml") {
		result.AddError(fmt.Errorf("portability assessment does not reference docs/platforms/portability-snapshot.yaml"))
	}

	comparisonText := assessmentLower + "\n" + claudeLower + "\n" + geminiLower + "\n" + piLower + "\n" + mappingLower
	if !strings.Contains(piLower, "pi.dev/docs/latest/skills") || !strings.Contains(piLower, "verified 2026-08-18") {
		result.AddError(fmt.Errorf("pi platform doc must include verified date and official source"))
	}
	if strings.Contains(strings.ToLower(schema), "pidev:") {
		result.AddError(fmt.Errorf("schema.yaml must use canonical pi identity, not pidev"))
	}
	for platform, support := range snapshot.PlatformSupport {
		if !strings.Contains(comparisonText, platform) {
			result.AddError(fmt.Errorf("platform_support platform %q is not reflected in the docs", platform))
		}
		if !strings.Contains(comparisonText, strings.ToLower(support.Status)) {
			result.AddError(fmt.Errorf("platform_support[%q].status %q is not reflected in the docs", platform, support.Status))
		}
	}
	for _, behavior := range snapshot.NonportableBehaviors {
		if !behaviorReflected(comparisonText, behavior) {
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

// behaviorReflected tolerates prose differences while requiring two meaningful
// terms from each snapshot claim to appear in the narrative/reference docs.
func behaviorReflected(text, behavior string) bool {
	terms := strings.Fields(strings.ToLower(behavior))
	found := 0
	for _, term := range terms {
		term = strings.Trim(term, "`.,;:/()")
		if len(term) < 4 || term == "and" || term == "with" {
			continue
		}
		if strings.Contains(text, term) {
			found++
		}
	}
	return found >= 2
}
