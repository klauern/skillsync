package similarity

import (
	"bytes"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/sync"
)

func TestDefaultFormatterConfig(t *testing.T) {
	config := DefaultFormatterConfig()

	if config.Format != FormatUnified {
		t.Errorf("expected default format to be unified, got %s", config.Format)
	}
	if config.ContextLines != 3 {
		t.Errorf("expected default context lines to be 3, got %d", config.ContextLines)
	}
	if config.MaxWidth != 80 {
		t.Errorf("expected default max width to be 80, got %d", config.MaxWidth)
	}
	if !config.ShowLineNumbers {
		t.Error("expected default show line numbers to be true")
	}
	if config.TruncateAt != 0 {
		t.Errorf("expected default truncate at to be 0, got %d", config.TruncateAt)
	}
}

func TestNewFormatter_DefaultsInvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config FormatterConfig
		want   FormatterConfig
	}{
		{
			name:   "negative context lines",
			config: FormatterConfig{ContextLines: -1},
			want:   FormatterConfig{ContextLines: 3, MaxWidth: 80, Format: FormatUnified},
		},
		{
			name:   "zero max width",
			config: FormatterConfig{MaxWidth: 0},
			want:   FormatterConfig{ContextLines: 0, MaxWidth: 80, Format: FormatUnified},
		},
		{
			name:   "empty format",
			config: FormatterConfig{Format: ""},
			want:   FormatterConfig{ContextLines: 0, MaxWidth: 80, Format: FormatUnified},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFormatter(tt.config)
			if f.config.Format != tt.want.Format {
				t.Errorf("format: got %s, want %s", f.config.Format, tt.want.Format)
			}
			if f.config.MaxWidth != tt.want.MaxWidth {
				t.Errorf("max width: got %d, want %d", f.config.MaxWidth, tt.want.MaxWidth)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input string
		width int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 2, "hi"},
		{"hello", 3, "hello"}, // Edge case: width <= 3 returns unchanged
		{"", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncateString(tt.input, tt.width)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}

func TestComparisonResult_DiffSummary(t *testing.T) {
	tests := []struct {
		name   string
		result ComparisonResult
		want   string
	}{
		{
			name:   "empty",
			result: ComparisonResult{},
			want:   "0 hunk(s), +0/-0 lines",
		},
		{
			name: "with changes",
			result: ComparisonResult{
				Hunks:        make([]sync.DiffHunk, 2),
				LinesAdded:   5,
				LinesRemoved: 3,
			},
			want: "2 hunk(s), +5/-3 lines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.DiffSummary()
			if got != tt.want {
				t.Errorf("DiffSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatter_FormatUnified(t *testing.T) {
	skill1 := model.Skill{
		Name:     "test-skill",
		Platform: "claude-code",
		Content:  "line1\nline2\nline3",
	}
	skill2 := model.Skill{
		Name:     "test-skill-copy",
		Platform: "cursor",
		Content:  "line1\nmodified\nline3",
	}

	result := &ComparisonResult{
		Skill1:       skill1,
		Skill2:       skill2,
		NameScore:    0.85,
		ContentScore: 0.75,
		Hunks: []sync.DiffHunk{
			{
				SourceStart: 2,
				SourceCount: 1,
				TargetStart: 2,
				TargetCount: 1,
				Lines: []sync.DiffLine{
					{Type: sync.DiffLineRemoved, Content: "line2"},
					{Type: sync.DiffLineAdded, Content: "modified"},
				},
			},
		},
		LinesAdded:   1,
		LinesRemoved: 1,
	}

	f := NewFormatter(DefaultFormatterConfig())
	var buf bytes.Buffer
	err := f.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()

	// Check header elements
	if !strings.Contains(output, "Comparing: test-skill <-> test-skill-copy") {
		t.Error("missing comparison header")
	}
	if !strings.Contains(output, "Name similarity:") {
		t.Error("missing name similarity")
	}
	if !strings.Contains(output, "85.0%") {
		t.Error("missing name score percentage")
	}
	if !strings.Contains(output, "Content similarity:") {
		t.Error("missing content similarity")
	}

	// Check diff format
	if !strings.Contains(output, "--- test-skill (claude-code)") {
		t.Error("missing unified diff source header")
	}
	if !strings.Contains(output, "+++ test-skill-copy (cursor)") {
		t.Error("missing unified diff target header")
	}
	if !strings.Contains(output, "@@ -2,1 +2,1 @@") {
		t.Error("missing hunk header")
	}
	if !strings.Contains(output, "-line2") {
		t.Error("missing removed line")
	}
	if !strings.Contains(output, "+modified") {
		t.Error("missing added line")
	}
}

func TestFormatter_FormatUnified_Truncation(t *testing.T) {
	result := &ComparisonResult{
		Skill1: model.Skill{Name: "skill1", Platform: "claude-code"},
		Skill2: model.Skill{Name: "skill2", Platform: "cursor"},
		Hunks: []sync.DiffHunk{
			{SourceStart: 1, SourceCount: 1, TargetStart: 1, TargetCount: 1},
			{SourceStart: 10, SourceCount: 1, TargetStart: 10, TargetCount: 1},
			{SourceStart: 20, SourceCount: 1, TargetStart: 20, TargetCount: 1},
		},
	}

	config := DefaultFormatterConfig()
	config.TruncateAt = 1
	f := NewFormatter(config)

	var buf bytes.Buffer
	err := f.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "2 more hunks not shown") {
		t.Error("expected truncation message")
	}
}

func TestFormatter_FormatSummary(t *testing.T) {
	result := &ComparisonResult{
		Skill1:       model.Skill{Name: "skill1", Platform: "claude-code"},
		Skill2:       model.Skill{Name: "skill2", Platform: "cursor"},
		NameScore:    0.9,
		ContentScore: 0.6,
		Hunks: []sync.DiffHunk{
			{SourceStart: 5, SourceCount: 2, TargetStart: 5, TargetCount: 3},
		},
		LinesAdded:   3,
		LinesRemoved: 2,
	}

	config := DefaultFormatterConfig()
	config.Format = FormatSummary
	f := NewFormatter(config)

	var buf bytes.Buffer
	err := f.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Hunks:") {
		t.Error("missing hunks count")
	}
	if !strings.Contains(output, "Lines added:") {
		t.Error("missing lines added")
	}
	if !strings.Contains(output, "+3") {
		t.Error("missing added count")
	}
	if !strings.Contains(output, "Lines removed:") {
		t.Error("missing lines removed")
	}
	if !strings.Contains(output, "-2") {
		t.Error("missing removed count")
	}
	if !strings.Contains(output, "Change locations:") {
		t.Error("missing change locations section")
	}
	if !strings.Contains(output, "@@ -5,2 +5,3 @@") {
		t.Error("missing hunk location")
	}
}

func TestFormatter_FormatSideBySide(t *testing.T) {
	skill1 := model.Skill{
		Name:     "skill1",
		Platform: "claude-code",
		Content:  "line1\nline2",
	}
	skill2 := model.Skill{
		Name:     "skill2",
		Platform: "cursor",
		Content:  "line1\nchanged",
	}

	result := &ComparisonResult{
		Skill1:       skill1,
		Skill2:       skill2,
		ContentScore: 0.5,
		Hunks: []sync.DiffHunk{
			{
				SourceStart: 2,
				SourceCount: 1,
				TargetStart: 2,
				TargetCount: 1,
				Lines: []sync.DiffLine{
					{Type: sync.DiffLineRemoved, Content: "line2"},
					{Type: sync.DiffLineAdded, Content: "changed"},
				},
			},
		},
		LinesAdded:   1,
		LinesRemoved: 1,
	}

	config := DefaultFormatterConfig()
	config.Format = FormatSideBySide
	f := NewFormatter(config)

	var buf bytes.Buffer
	err := f.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()

	// Check header
	if !strings.Contains(output, "Comparing: skill1 <-> skill2") {
		t.Error("missing comparison header")
	}

	// Check column separator
	if !strings.Contains(output, " | ") {
		t.Error("missing column separator")
	}

	// Check line markers exist (- for removed, + for added)
	if !strings.Contains(output, "-") {
		t.Error("missing removed line marker")
	}
	if !strings.Contains(output, "+") {
		t.Error("missing added line marker")
	}
}

func TestFormatter_FormatMultiple(t *testing.T) {
	results := []*ComparisonResult{
		{
			Skill1: model.Skill{Name: "skill1", Platform: "claude-code"},
			Skill2: model.Skill{Name: "skill2", Platform: "cursor"},
		},
		{
			Skill1: model.Skill{Name: "skill3", Platform: "claude-code"},
			Skill2: model.Skill{Name: "skill4", Platform: "codex"},
		},
	}

	f := NewFormatter(DefaultFormatterConfig())
	var buf bytes.Buffer
	err := f.FormatMultiple(&buf, results)
	if err != nil {
		t.Fatalf("FormatMultiple() error = %v", err)
	}

	output := buf.String()

	// Check both results are present
	if !strings.Contains(output, "skill1 <-> skill2") {
		t.Error("missing first comparison")
	}
	if !strings.Contains(output, "skill3 <-> skill4") {
		t.Error("missing second comparison")
	}

	// Check separator between results
	if !strings.Contains(output, strings.Repeat("=", 60)) {
		t.Error("missing separator between results")
	}
}

func TestFormatComparisonTable(t *testing.T) {
	results := []*ComparisonResult{
		{
			Skill1:       model.Skill{Name: "commit"},
			Skill2:       model.Skill{Name: "commit-push"},
			NameScore:    0.75,
			ContentScore: 0.6,
			Hunks:        make([]sync.DiffHunk, 2),
			LinesAdded:   5,
			LinesRemoved: 2,
		},
		{
			Skill1:       model.Skill{Name: "review"},
			Skill2:       model.Skill{Name: "review-pr"},
			NameScore:    0.8,
			ContentScore: 0.0, // No content score
			LinesAdded:   0,
			LinesRemoved: 0,
		},
	}

	var buf bytes.Buffer
	err := FormatComparisonTable(&buf, results)
	if err != nil {
		t.Fatalf("FormatComparisonTable() error = %v", err)
	}

	output := buf.String()

	// Check headers
	if !strings.Contains(output, "SKILL 1") {
		t.Error("missing SKILL 1 header")
	}
	if !strings.Contains(output, "SKILL 2") {
		t.Error("missing SKILL 2 header")
	}
	if !strings.Contains(output, "NAME %") {
		t.Error("missing NAME % header")
	}
	if !strings.Contains(output, "CONTENT %") {
		t.Error("missing CONTENT % header")
	}
	if !strings.Contains(output, "CHANGES") {
		t.Error("missing CHANGES header")
	}

	// Check data rows
	if !strings.Contains(output, "commit") {
		t.Error("missing commit skill")
	}
	if !strings.Contains(output, "75%") {
		t.Error("missing name score")
	}
	if !strings.Contains(output, "60%") {
		t.Error("missing content score")
	}

	// Check "-" for missing content score
	lines := strings.Split(output, "\n")
	foundDash := false
	for _, line := range lines {
		if strings.Contains(line, "review") && strings.Contains(line, "-") {
			foundDash = true
			break
		}
	}
	if !foundDash {
		t.Error("expected '-' for missing content score")
	}
}

func TestFormatComparisonTable_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := FormatComparisonTable(&buf, nil)
	if err != nil {
		t.Fatalf("FormatComparisonTable() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No similar skills found") {
		t.Error("expected empty message")
	}
}

func TestComputeDiff(t *testing.T) {
	skill1 := model.Skill{
		Name:     "test",
		Platform: "claude-code",
		Content:  "line1\nline2\nline3",
	}
	skill2 := model.Skill{
		Name:     "test",
		Platform: "cursor",
		Content:  "line1\nmodified\nline3",
	}

	result := ComputeDiff(skill1, skill2, 1.0, 0.8)

	if result.Skill1.Name != "test" {
		t.Errorf("expected skill1 name 'test', got %s", result.Skill1.Name)
	}
	if result.NameScore != 1.0 {
		t.Errorf("expected name score 1.0, got %f", result.NameScore)
	}
	if result.ContentScore != 0.8 {
		t.Errorf("expected content score 0.8, got %f", result.ContentScore)
	}
	if len(result.Hunks) == 0 {
		t.Error("expected hunks to be computed")
	}
	if result.LinesAdded == 0 {
		t.Error("expected lines added to be counted")
	}
	if result.LinesRemoved == 0 {
		t.Error("expected lines removed to be counted")
	}
}

func TestComputeDiff_IdenticalContent(t *testing.T) {
	skill1 := model.Skill{
		Name:     "test",
		Platform: "claude-code",
		Content:  "same content",
	}
	skill2 := model.Skill{
		Name:     "test",
		Platform: "cursor",
		Content:  "same content",
	}

	result := ComputeDiff(skill1, skill2, 1.0, 1.0)

	if len(result.Hunks) != 0 {
		t.Error("expected no hunks for identical content")
	}
	if result.LinesAdded != 0 || result.LinesRemoved != 0 {
		t.Error("expected no line changes for identical content")
	}
}
