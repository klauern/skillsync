package cli

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/sync"
	"github.com/klauern/skillsync/internal/ui"
)

// newConflictResolverWithReader creates a ConflictResolver with a custom reader for testing.
func newConflictResolverWithReader(r io.Reader) *ConflictResolver {
	return &ConflictResolver{
		reader: bufio.NewReader(r),
	}
}

// captureOutput captures stdout during test execution.
func captureOutput(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	f()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}
	return buf.String()
}

// createTestConflict creates a conflict for testing with the given parameters.
func createTestConflict(name, sourceContent, targetContent string) *sync.Conflict {
	detector := sync.NewConflictDetector()
	source := model.Skill{
		Name:    name,
		Content: sourceContent,
	}
	target := model.Skill{
		Name:    name,
		Content: targetContent,
	}
	conflict := detector.DetectConflict(source, target)
	if conflict == nil {
		// If no conflict detected (same content), create a minimal one
		conflict = &sync.Conflict{
			SkillName:   name,
			Type:        sync.ConflictTypeContent,
			Source:      source,
			Target:      target,
			SourceLines: strings.Split(sourceContent, "\n"),
			TargetLines: strings.Split(targetContent, "\n"),
		}
	}
	return conflict
}

func TestNewConflictResolver(t *testing.T) {
	cr := NewConflictResolver()
	if cr == nil {
		t.Fatal("NewConflictResolver() returned nil")
	}
	if cr.reader == nil {
		t.Error("NewConflictResolver() reader should not be nil")
	}
}

func TestFormatDiffLine(t *testing.T) {
	// Disable colors for predictable test output
	ui.DisableColors()
	defer ui.EnableColors()

	tests := map[string]struct {
		line sync.DiffLine
		want string
	}{
		"added line": {
			line: sync.DiffLine{Type: sync.DiffLineAdded, Content: "new line"},
			want: "+new line",
		},
		"removed line": {
			line: sync.DiffLine{Type: sync.DiffLineRemoved, Content: "old line"},
			want: "-old line",
		},
		"context line": {
			line: sync.DiffLine{Type: sync.DiffLineContext, Content: "unchanged"},
			want: " unchanged",
		},
		"empty content added": {
			line: sync.DiffLine{Type: sync.DiffLineAdded, Content: ""},
			want: "+",
		},
		"empty content removed": {
			line: sync.DiffLine{Type: sync.DiffLineRemoved, Content: ""},
			want: "-",
		},
		"empty content context": {
			line: sync.DiffLine{Type: sync.DiffLineContext, Content: ""},
			want: " ",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := formatDiffLine(tt.line)
			if got != tt.want {
				t.Errorf("formatDiffLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShowDiffPreview(t *testing.T) {
	ui.DisableColors()
	defer ui.EnableColors()

	tests := map[string]struct {
		conflict       *sync.Conflict
		wantContains   []string
		wantNotContain []string
	}{
		"empty hunks": {
			conflict: &sync.Conflict{
				SkillName: "test-skill",
				Hunks:     []sync.DiffHunk{},
			},
			wantContains: []string{
				"Preview of changes:",
				strings.Repeat("-", 50),
			},
		},
		"single hunk with few lines": {
			conflict: &sync.Conflict{
				SkillName: "test-skill",
				Hunks: []sync.DiffHunk{
					{
						SourceStart: 1,
						SourceCount: 2,
						TargetStart: 1,
						TargetCount: 3,
						Lines: []sync.DiffLine{
							{Type: sync.DiffLineContext, Content: "line1"},
							{Type: sync.DiffLineRemoved, Content: "old"},
							{Type: sync.DiffLineAdded, Content: "new"},
						},
					},
				},
			},
			wantContains: []string{
				"Preview of changes:",
				"@@ -1,2 +1,3 @@",
				" line1",
				"-old",
				"+new",
			},
		},
		"truncation when exceeding max lines": {
			conflict: &sync.Conflict{
				SkillName: "test-skill",
				Hunks: []sync.DiffHunk{
					{
						SourceStart: 1,
						SourceCount: 15,
						TargetStart: 1,
						TargetCount: 15,
						Lines: []sync.DiffLine{
							{Type: sync.DiffLineContext, Content: "line1"},
							{Type: sync.DiffLineContext, Content: "line2"},
							{Type: sync.DiffLineContext, Content: "line3"},
							{Type: sync.DiffLineContext, Content: "line4"},
							{Type: sync.DiffLineContext, Content: "line5"},
							{Type: sync.DiffLineContext, Content: "line6"},
							{Type: sync.DiffLineContext, Content: "line7"},
							{Type: sync.DiffLineContext, Content: "line8"},
							{Type: sync.DiffLineContext, Content: "line9"},
							{Type: sync.DiffLineContext, Content: "line10"},
							{Type: sync.DiffLineContext, Content: "line11"},
							{Type: sync.DiffLineContext, Content: "line12"},
						},
					},
				},
			},
			wantContains: []string{
				"Preview of changes:",
				"(truncated)",
			},
			wantNotContain: []string{
				"line11",
				"line12",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cr := NewConflictResolver()
			output := captureOutput(t, func() {
				cr.showDiffPreview(tt.conflict)
			})

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("showDiffPreview() output missing %q\nGot: %s", want, output)
				}
			}
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(output, notWant) {
					t.Errorf("showDiffPreview() output should not contain %q\nGot: %s", notWant, output)
				}
			}
		})
	}
}

func TestShowFullContent(t *testing.T) {
	tests := map[string]struct {
		label        string
		content      string
		wantContains []string
	}{
		"empty content": {
			label:   "SOURCE",
			content: "",
			wantContains: []string{
				"=== SOURCE CONTENT ===",
				strings.Repeat("-", 50),
				"   1 | ",
			},
		},
		"single line content": {
			label:   "TARGET",
			content: "hello world",
			wantContains: []string{
				"=== TARGET CONTENT ===",
				"   1 | hello world",
			},
		},
		"multi-line content": {
			label:   "SOURCE",
			content: "line one\nline two\nline three",
			wantContains: []string{
				"=== SOURCE CONTENT ===",
				"   1 | line one",
				"   2 | line two",
				"   3 | line three",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cr := NewConflictResolver()
			output := captureOutput(t, func() {
				cr.showFullContent(tt.label, tt.content)
			})

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("showFullContent() output missing %q\nGot: %s", want, output)
				}
			}
		})
	}
}

func TestPromptResolution(t *testing.T) {
	tests := map[string]struct {
		input      string
		want       sync.ResolutionChoice
		wantErr    bool
		errContain string
	}{
		"choice 1 - use source": {
			input: "1\n",
			want:  sync.ResolutionUseSource,
		},
		"choice 2 - use target": {
			input: "2\n",
			want:  sync.ResolutionUseTarget,
		},
		"choice 3 - merge": {
			input: "3\n",
			want:  sync.ResolutionMerge,
		},
		"choice 4 - skip": {
			input: "4\n",
			want:  sync.ResolutionSkip,
		},
		"choice 5 then 1 - show source then use source": {
			input: "5\n1\n",
			want:  sync.ResolutionUseSource,
		},
		"choice 6 then 2 - show target then use target": {
			input: "6\n2\n",
			want:  sync.ResolutionUseTarget,
		},
		"invalid then valid - 0 then 1": {
			input: "0\n1\n",
			want:  sync.ResolutionUseSource,
		},
		"invalid then valid - 7 then 3": {
			input: "7\n3\n",
			want:  sync.ResolutionMerge,
		},
		"non-numeric then valid - abc then 4": {
			input: "abc\n4\n",
			want:  sync.ResolutionSkip,
		},
		"whitespace handling": {
			input: "  2  \n",
			want:  sync.ResolutionUseTarget,
		},
		"EOF error": {
			input:      "",
			wantErr:    true,
			errContain: "failed to read input",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			conflict := createTestConflict("test-skill", "source content", "target content")
			reader := strings.NewReader(tt.input)
			cr := newConflictResolverWithReader(reader)

			// Capture output to suppress printing during tests
			_ = captureOutput(t, func() {
				got, err := cr.promptResolution(conflict)
				if tt.wantErr {
					if err == nil {
						t.Errorf("promptResolution() expected error containing %q, got nil", tt.errContain)
					} else if !strings.Contains(err.Error(), tt.errContain) {
						t.Errorf("promptResolution() error = %v, want error containing %q", err, tt.errContain)
					}
					return
				}
				if err != nil {
					t.Errorf("promptResolution() unexpected error: %v", err)
					return
				}
				if got != tt.want {
					t.Errorf("promptResolution() = %v, want %v", got, tt.want)
				}
			})
		})
	}
}

func TestPromptForConflictMode(t *testing.T) {
	tests := map[string]struct {
		input         string
		conflictCount int
		want          ConflictMode
		wantErr       bool
		errContain    string
	}{
		"choice 1 - interactive": {
			input:         "1\n",
			conflictCount: 3,
			want:          ConflictModeInteractive,
		},
		"choice 2 - use source": {
			input:         "2\n",
			conflictCount: 5,
			want:          ConflictModeUseSource,
		},
		"choice 3 - use target": {
			input:         "3\n",
			conflictCount: 1,
			want:          ConflictModeUseTarget,
		},
		"choice 4 - auto merge": {
			input:         "4\n",
			conflictCount: 2,
			want:          ConflictModeAutoMerge,
		},
		"choice 5 - abort": {
			input:         "5\n",
			conflictCount: 10,
			want:          ConflictModeAbort,
		},
		"invalid choice 0": {
			input:         "0\n",
			conflictCount: 1,
			wantErr:       true,
			errContain:    "invalid choice",
		},
		"invalid choice 6": {
			input:         "6\n",
			conflictCount: 1,
			wantErr:       true,
			errContain:    "invalid choice",
		},
		"non-numeric input": {
			input:         "abc\n",
			conflictCount: 1,
			wantErr:       true,
			errContain:    "invalid choice",
		},
		"EOF error": {
			input:         "",
			conflictCount: 1,
			wantErr:       true,
			errContain:    "failed to read input",
		},
		"whitespace handling": {
			input:         "  3  \n",
			conflictCount: 2,
			want:          ConflictModeUseTarget,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			cr := newConflictResolverWithReader(reader)

			// Capture output to suppress printing during tests
			_ = captureOutput(t, func() {
				got, err := cr.PromptForConflictMode(tt.conflictCount)
				if tt.wantErr {
					if err == nil {
						t.Errorf("PromptForConflictMode() expected error containing %q, got nil", tt.errContain)
					} else if !strings.Contains(err.Error(), tt.errContain) {
						t.Errorf("PromptForConflictMode() error = %v, want error containing %q", err, tt.errContain)
					}
					return
				}
				if err != nil {
					t.Errorf("PromptForConflictMode() unexpected error: %v", err)
					return
				}
				if got != tt.want {
					t.Errorf("PromptForConflictMode() = %v, want %v", got, tt.want)
				}
			})
		})
	}
}

func TestResolveAllWithMode(t *testing.T) {
	tests := map[string]struct {
		mode       ConflictMode
		conflicts  []*sync.Conflict
		wantErr    bool
		errContain string
		validate   func(t *testing.T, result map[string]string, conflicts []*sync.Conflict)
	}{
		"empty conflicts": {
			mode:      ConflictModeUseSource,
			conflicts: []*sync.Conflict{},
			validate: func(t *testing.T, result map[string]string, _ []*sync.Conflict) {
				if len(result) != 0 {
					t.Errorf("expected empty result for empty conflicts, got %d entries", len(result))
				}
			},
		},
		"use source mode": {
			mode: ConflictModeUseSource,
			conflicts: []*sync.Conflict{
				createTestConflict("skill1", "source1", "target1"),
				createTestConflict("skill2", "source2", "target2"),
			},
			validate: func(t *testing.T, result map[string]string, conflicts []*sync.Conflict) {
				if len(result) != 2 {
					t.Errorf("expected 2 results, got %d", len(result))
				}
				if result["skill1"] != "source1" {
					t.Errorf("skill1 = %q, want %q", result["skill1"], "source1")
				}
				if result["skill2"] != "source2" {
					t.Errorf("skill2 = %q, want %q", result["skill2"], "source2")
				}
				for _, c := range conflicts {
					if c.Resolution != sync.ResolutionUseSource {
						t.Errorf("conflict %s resolution = %v, want %v", c.SkillName, c.Resolution, sync.ResolutionUseSource)
					}
				}
			},
		},
		"use target mode": {
			mode: ConflictModeUseTarget,
			conflicts: []*sync.Conflict{
				createTestConflict("skill1", "source1", "target1"),
			},
			validate: func(t *testing.T, result map[string]string, conflicts []*sync.Conflict) {
				if result["skill1"] != "target1" {
					t.Errorf("skill1 = %q, want %q", result["skill1"], "target1")
				}
				if conflicts[0].Resolution != sync.ResolutionUseTarget {
					t.Errorf("resolution = %v, want %v", conflicts[0].Resolution, sync.ResolutionUseTarget)
				}
			},
		},
		"auto merge mode": {
			mode: ConflictModeAutoMerge,
			conflicts: []*sync.Conflict{
				createTestConflict("skill1", "line1\ncommon\nline3", "line1\ncommon\nline4"),
			},
			validate: func(t *testing.T, result map[string]string, conflicts []*sync.Conflict) {
				if len(result) != 1 {
					t.Errorf("expected 1 result, got %d", len(result))
				}
				if conflicts[0].Resolution != sync.ResolutionMerge {
					t.Errorf("resolution = %v, want %v", conflicts[0].Resolution, sync.ResolutionMerge)
				}
				// Merged content should exist
				if result["skill1"] == "" {
					t.Error("merged content should not be empty")
				}
			},
		},
		"invalid mode - interactive": {
			mode: ConflictModeInteractive,
			conflicts: []*sync.Conflict{
				createTestConflict("skill1", "source", "target"),
			},
			wantErr:    true,
			errContain: "invalid conflict mode",
		},
		"invalid mode - abort": {
			mode: ConflictModeAbort,
			conflicts: []*sync.Conflict{
				createTestConflict("skill1", "source", "target"),
			},
			wantErr:    true,
			errContain: "invalid conflict mode",
		},
		"invalid mode - unknown": {
			mode: ConflictMode("unknown"),
			conflicts: []*sync.Conflict{
				createTestConflict("skill1", "source", "target"),
			},
			wantErr:    true,
			errContain: "invalid conflict mode",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cr := NewConflictResolver()
			result, err := cr.ResolveAllWithMode(tt.conflicts, tt.mode)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveAllWithMode() expected error containing %q, got nil", tt.errContain)
				} else if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("ResolveAllWithMode() error = %v, want error containing %q", err, tt.errContain)
				}
				return
			}
			if err != nil {
				t.Errorf("ResolveAllWithMode() unexpected error: %v", err)
				return
			}
			if tt.validate != nil {
				tt.validate(t, result, tt.conflicts)
			}
		})
	}
}

func TestDisplayConflictSummary(t *testing.T) {
	tests := map[string]struct {
		conflicts    []*sync.Conflict
		wantContains []string
	}{
		"empty conflicts": {
			conflicts: []*sync.Conflict{},
			wantContains: []string{
				"=== Conflict Summary ===",
				"SKILL",
				"TYPE",
				"CHANGES",
			},
		},
		"single conflict": {
			conflicts: []*sync.Conflict{
				{
					SkillName: "my-skill",
					Type:      sync.ConflictTypeContent,
					Hunks: []sync.DiffHunk{
						{
							Lines: []sync.DiffLine{
								{Type: sync.DiffLineAdded, Content: "new"},
								{Type: sync.DiffLineRemoved, Content: "old"},
							},
						},
					},
				},
			},
			wantContains: []string{
				"=== Conflict Summary ===",
				"my-skill",
				"content",
				"+1/-1",
			},
		},
		"multiple conflicts": {
			conflicts: []*sync.Conflict{
				{
					SkillName: "skill-alpha",
					Type:      sync.ConflictTypeContent,
					Hunks:     []sync.DiffHunk{},
				},
				{
					SkillName: "skill-beta",
					Type:      sync.ConflictTypeMetadata,
					Hunks:     []sync.DiffHunk{},
				},
				{
					SkillName: "skill-gamma",
					Type:      sync.ConflictTypeBoth,
					Hunks:     []sync.DiffHunk{},
				},
			},
			wantContains: []string{
				"skill-alpha",
				"content",
				"skill-beta",
				"metadata",
				"skill-gamma",
				"both",
			},
		},
		"long skill name truncation": {
			conflicts: []*sync.Conflict{
				{
					SkillName: "this-is-a-very-long-skill-name-that-exceeds-thirty-characters",
					Type:      sync.ConflictTypeContent,
					Hunks:     []sync.DiffHunk{},
				},
			},
			wantContains: []string{
				"this-is-a-very-long-skill-n...",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cr := NewConflictResolver()
			output := captureOutput(t, func() {
				cr.DisplayConflictSummary(tt.conflicts)
			})

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("DisplayConflictSummary() output missing %q\nGot: %s", want, output)
				}
			}
		})
	}
}

func TestResolveConflicts(t *testing.T) {
	tests := map[string]struct {
		input      string
		conflicts  []*sync.Conflict
		wantErr    bool
		errContain string
		validate   func(t *testing.T, result map[string]string, conflicts []*sync.Conflict)
	}{
		"empty conflicts": {
			input:     "",
			conflicts: []*sync.Conflict{},
			validate: func(t *testing.T, result map[string]string, _ []*sync.Conflict) {
				if len(result) != 0 {
					t.Errorf("expected empty result for empty conflicts, got %d entries", len(result))
				}
			},
		},
		"single conflict - use source": {
			input: "1\n",
			conflicts: []*sync.Conflict{
				createTestConflict("skill1", "source content", "target content"),
			},
			validate: func(t *testing.T, result map[string]string, conflicts []*sync.Conflict) {
				if len(result) != 1 {
					t.Errorf("expected 1 result, got %d", len(result))
				}
				if result["skill1"] != "source content" {
					t.Errorf("skill1 = %q, want %q", result["skill1"], "source content")
				}
				if conflicts[0].Resolution != sync.ResolutionUseSource {
					t.Errorf("resolution = %v, want %v", conflicts[0].Resolution, sync.ResolutionUseSource)
				}
			},
		},
		"single conflict - use target": {
			input: "2\n",
			conflicts: []*sync.Conflict{
				createTestConflict("skill1", "source content", "target content"),
			},
			validate: func(t *testing.T, result map[string]string, _ []*sync.Conflict) {
				if result["skill1"] != "target content" {
					t.Errorf("skill1 = %q, want %q", result["skill1"], "target content")
				}
			},
		},
		"single conflict - skip": {
			input: "4\n",
			conflicts: []*sync.Conflict{
				createTestConflict("skill1", "source content", "target content"),
			},
			validate: func(t *testing.T, result map[string]string, conflicts []*sync.Conflict) {
				// Skip keeps target
				if result["skill1"] != "target content" {
					t.Errorf("skill1 = %q, want %q", result["skill1"], "target content")
				}
				if conflicts[0].Resolution != sync.ResolutionSkip {
					t.Errorf("resolution = %v, want %v", conflicts[0].Resolution, sync.ResolutionSkip)
				}
			},
		},
		"multiple conflicts - different choices": {
			input: "1\n2\n3\n",
			conflicts: []*sync.Conflict{
				createTestConflict("skill1", "source1", "target1"),
				createTestConflict("skill2", "source2", "target2"),
				createTestConflict("skill3", "line1\ncommon\nline3", "line1\ncommon\nline4"),
			},
			validate: func(t *testing.T, result map[string]string, conflicts []*sync.Conflict) {
				if len(result) != 3 {
					t.Errorf("expected 3 results, got %d", len(result))
				}
				if result["skill1"] != "source1" {
					t.Errorf("skill1 = %q, want %q", result["skill1"], "source1")
				}
				if result["skill2"] != "target2" {
					t.Errorf("skill2 = %q, want %q", result["skill2"], "target2")
				}
				if conflicts[0].Resolution != sync.ResolutionUseSource {
					t.Errorf("conflict 0 resolution = %v, want %v", conflicts[0].Resolution, sync.ResolutionUseSource)
				}
				if conflicts[1].Resolution != sync.ResolutionUseTarget {
					t.Errorf("conflict 1 resolution = %v, want %v", conflicts[1].Resolution, sync.ResolutionUseTarget)
				}
				if conflicts[2].Resolution != sync.ResolutionMerge {
					t.Errorf("conflict 2 resolution = %v, want %v", conflicts[2].Resolution, sync.ResolutionMerge)
				}
			},
		},
		"read error mid-resolution": {
			input: "1\n", // Not enough input for second conflict
			conflicts: []*sync.Conflict{
				createTestConflict("skill1", "source1", "target1"),
				createTestConflict("skill2", "source2", "target2"),
			},
			wantErr:    true,
			errContain: "failed to get resolution",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			cr := newConflictResolverWithReader(reader)

			var result map[string]string
			var err error

			// Capture output to suppress printing during tests
			_ = captureOutput(t, func() {
				result, err = cr.ResolveConflicts(tt.conflicts)
			})

			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveConflicts() expected error containing %q, got nil", tt.errContain)
				} else if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("ResolveConflicts() error = %v, want error containing %q", err, tt.errContain)
				}
				return
			}
			if err != nil {
				t.Errorf("ResolveConflicts() unexpected error: %v", err)
				return
			}
			if tt.validate != nil {
				tt.validate(t, result, tt.conflicts)
			}
		})
	}
}

func TestConflictModeConstants(t *testing.T) {
	// Test that all conflict mode constants have expected values
	tests := map[string]struct {
		mode ConflictMode
		want string
	}{
		"interactive": {
			mode: ConflictModeInteractive,
			want: "interactive",
		},
		"use-source": {
			mode: ConflictModeUseSource,
			want: "use-source",
		},
		"use-target": {
			mode: ConflictModeUseTarget,
			want: "use-target",
		},
		"auto-merge": {
			mode: ConflictModeAutoMerge,
			want: "auto-merge",
		},
		"abort": {
			mode: ConflictModeAbort,
			want: "abort",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if string(tt.mode) != tt.want {
				t.Errorf("ConflictMode = %q, want %q", tt.mode, tt.want)
			}
		})
	}
}
