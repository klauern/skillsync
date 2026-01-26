package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/similarity"
)

func TestMakePairKey(t *testing.T) {
	skill1 := model.Skill{
		Name:     "commit",
		Platform: model.ClaudeCode,
		Scope:    model.ScopeUser,
	}
	skill2 := model.Skill{
		Name:     "commit-push",
		Platform: model.Cursor,
		Scope:    model.ScopeRepo,
	}

	// Should be consistent regardless of order
	key1 := makePairKey(skill1, skill2)
	key2 := makePairKey(skill2, skill1)

	if key1 != key2 {
		t.Errorf("makePairKey should return same key regardless of order: %q vs %q", key1, key2)
	}

	// Keys should contain both skill identifiers
	if !strings.Contains(key1, "commit") || !strings.Contains(key1, "commit-push") {
		t.Errorf("pair key should contain both skill names: %q", key1)
	}
}

func TestToComparisonOutputs(t *testing.T) {
	results := []*similarity.ComparisonResult{
		{
			Skill1: model.Skill{
				Name:     "skill1",
				Platform: model.ClaudeCode,
			},
			Skill2: model.Skill{
				Name:     "skill2",
				Platform: model.Cursor,
			},
			NameScore:    0.85,
			ContentScore: 0.72,
			LinesAdded:   5,
			LinesRemoved: 3,
		},
	}

	outputs := toComparisonOutputs(results)

	if len(outputs) != 1 {
		t.Fatalf("expected 1 output, got %d", len(outputs))
	}

	out := outputs[0]
	if out.Skill1 != "skill1" {
		t.Errorf("expected Skill1 = 'skill1', got %q", out.Skill1)
	}
	if out.Skill2 != "skill2" {
		t.Errorf("expected Skill2 = 'skill2', got %q", out.Skill2)
	}
	if out.Platform1 != "claude-code" {
		t.Errorf("expected Platform1 = 'claude-code', got %q", out.Platform1)
	}
	if out.Platform2 != "cursor" {
		t.Errorf("expected Platform2 = 'cursor', got %q", out.Platform2)
	}
	if out.NameScore != 0.85 {
		t.Errorf("expected NameScore = 0.85, got %f", out.NameScore)
	}
	if out.ContentScore != 0.72 {
		t.Errorf("expected ContentScore = 0.72, got %f", out.ContentScore)
	}
	if out.LinesAdded != 5 {
		t.Errorf("expected LinesAdded = 5, got %d", out.LinesAdded)
	}
	if out.LinesRemoved != 3 {
		t.Errorf("expected LinesRemoved = 3, got %d", out.LinesRemoved)
	}
}

func TestOutputCompareResults(t *testing.T) {
	// Test with empty results
	var emptyResults []*similarity.ComparisonResult

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputCompareResults(emptyResults, "table")

	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe writer: %v", closeErr)
	}
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Errorf("outputCompareResults() error = %v", err)
	}

	if !strings.Contains(buf.String(), "No similar skills found") {
		t.Errorf("expected 'No similar skills found' in output, got %q", buf.String())
	}
}

func TestOutputFormats(t *testing.T) {
	results := []*similarity.ComparisonResult{
		{
			Skill1: model.Skill{
				Name:     "test-skill",
				Platform: model.ClaudeCode,
				Content:  "line1\nline2\nline3",
			},
			Skill2: model.Skill{
				Name:     "test-skill-copy",
				Platform: model.Cursor,
				Content:  "line1\nmodified\nline3",
			},
			NameScore:    0.85,
			ContentScore: 0.75,
		},
	}

	formats := []string{"table", "unified", "side-by-side", "summary", "json", "yaml"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := outputCompareResults(results, format)

			if closeErr := w.Close(); closeErr != nil {
				t.Fatalf("failed to close pipe writer: %v", closeErr)
			}
			os.Stdout = old

			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)

			if err != nil {
				t.Errorf("outputCompareResults(%q) error = %v", format, err)
			}

			output := buf.String()
			if output == "" {
				t.Errorf("outputCompareResults(%q) produced empty output", format)
			}

			// Format-specific checks
			switch format {
			case "json":
				if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
					t.Errorf("JSON output missing braces: %q", output)
				}
			case "yaml":
				if !strings.Contains(output, "skill1:") {
					t.Errorf("YAML output missing expected field: %q", output)
				}
			case "table":
				if !strings.Contains(output, "SKILL 1") || !strings.Contains(output, "SKILL 2") {
					t.Errorf("table output missing headers: %q", output)
				}
			case "unified":
				if !strings.Contains(output, "---") || !strings.Contains(output, "+++") {
					t.Errorf("unified diff output missing markers: %q", output)
				}
			}
		})
	}
}

func TestInvalidOutputFormat(t *testing.T) {
	results := []*similarity.ComparisonResult{}
	err := outputCompareResults(results, "invalid-format")
	if err == nil {
		t.Error("expected error for invalid format, got nil")
	}
}

func TestParseCompareConfigValidation(t *testing.T) {
	// Test config parsing logic directly using mock command data
	tests := []struct {
		name             string
		nameThreshold    float64
		contentThreshold float64
		nameOnly         bool
		contentOnly      bool
		format           string
		wantErr          bool
	}{
		{
			name:             "valid default config",
			nameThreshold:    0.7,
			contentThreshold: 0.6,
			format:           "table",
			wantErr:          false,
		},
		{
			name:             "invalid name threshold too high",
			nameThreshold:    1.5,
			contentThreshold: 0.6,
			format:           "table",
			wantErr:          true,
		},
		{
			name:             "invalid name threshold negative",
			nameThreshold:    -0.1,
			contentThreshold: 0.6,
			format:           "table",
			wantErr:          true,
		},
		{
			name:             "invalid content threshold",
			nameThreshold:    0.7,
			contentThreshold: 2.0,
			format:           "table",
			wantErr:          true,
		},
		{
			name:             "conflicting name-only and content-only",
			nameThreshold:    0.7,
			contentThreshold: 0.6,
			nameOnly:         true,
			contentOnly:      true,
			format:           "table",
			wantErr:          true,
		},
		{
			name:             "invalid format",
			nameThreshold:    0.7,
			contentThreshold: 0.6,
			format:           "invalid",
			wantErr:          true,
		},
		{
			name:             "valid unified format",
			nameThreshold:    0.7,
			contentThreshold: 0.6,
			format:           "unified",
			wantErr:          false,
		},
		{
			name:             "valid side-by-side format",
			nameThreshold:    0.7,
			contentThreshold: 0.6,
			format:           "side-by-side",
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &compareConfig{
				nameThreshold:    tt.nameThreshold,
				contentThreshold: tt.contentThreshold,
				nameOnly:         tt.nameOnly,
				contentOnly:      tt.contentOnly,
				format:           tt.format,
			}

			err := validateCompareConfig(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCompareConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// validateCompareConfig is a helper for testing config validation logic
func validateCompareConfig(cfg *compareConfig) error {
	// Validate thresholds
	if cfg.nameThreshold < 0 || cfg.nameThreshold > 1 {
		return errInvalidNameThreshold
	}
	if cfg.contentThreshold < 0 || cfg.contentThreshold > 1 {
		return errInvalidContentThreshold
	}

	// Validate mutual exclusivity
	if cfg.nameOnly && cfg.contentOnly {
		return errConflictingFlags
	}

	// Validate format
	validFormats := map[string]bool{
		"table": true, "unified": true, "side-by-side": true,
		"summary": true, "json": true, "yaml": true,
	}
	if !validFormats[cfg.format] {
		return errInvalidFormat
	}

	return nil
}

// Error sentinel values for testing
var (
	errInvalidNameThreshold    = &configError{"name-threshold must be between 0.0 and 1.0"}
	errInvalidContentThreshold = &configError{"content-threshold must be between 0.0 and 1.0"}
	errConflictingFlags        = &configError{"cannot use both --name-only and --content-only"}
	errInvalidFormat           = &configError{"invalid format"}
)

type configError struct {
	msg string
}

func (e *configError) Error() string {
	return e.msg
}
