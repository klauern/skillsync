package similarity

import (
	"math"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestDefaultContentMatcherConfig(t *testing.T) {
	config := DefaultContentMatcherConfig()

	if config.Threshold != 0.6 {
		t.Errorf("expected Threshold 0.6, got %f", config.Threshold)
	}
	if config.Algorithm != "combined" {
		t.Errorf("expected Algorithm 'combined', got %q", config.Algorithm)
	}
	if config.NGramSize != 3 {
		t.Errorf("expected NGramSize 3, got %d", config.NGramSize)
	}
	if !config.LineMode {
		t.Error("expected LineMode true, got false")
	}
}

func TestNewContentMatcher(t *testing.T) {
	tests := []struct {
		name          string
		config        ContentMatcherConfig
		wantThreshold float64
		wantAlgorithm string
		wantNGramSize int
	}{
		{
			name:          "default values for zero config",
			config:        ContentMatcherConfig{},
			wantThreshold: 0.6,
			wantAlgorithm: "combined",
			wantNGramSize: 3,
		},
		{
			name: "custom valid values preserved",
			config: ContentMatcherConfig{
				Threshold: 0.8,
				Algorithm: "lcs",
				NGramSize: 5,
			},
			wantThreshold: 0.8,
			wantAlgorithm: "lcs",
			wantNGramSize: 5,
		},
		{
			name: "invalid threshold corrected",
			config: ContentMatcherConfig{
				Threshold: 1.5,
				Algorithm: "jaccard",
				NGramSize: 2,
			},
			wantThreshold: 0.6,
			wantAlgorithm: "jaccard",
			wantNGramSize: 2,
		},
		{
			name: "negative threshold corrected",
			config: ContentMatcherConfig{
				Threshold: -0.5,
				Algorithm: "lcs",
				NGramSize: 4,
			},
			wantThreshold: 0.6,
			wantAlgorithm: "lcs",
			wantNGramSize: 4,
		},
		{
			name: "negative ngram size corrected",
			config: ContentMatcherConfig{
				Threshold: 0.5,
				Algorithm: "jaccard",
				NGramSize: -1,
			},
			wantThreshold: 0.5,
			wantAlgorithm: "jaccard",
			wantNGramSize: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewContentMatcher(tt.config)
			if matcher.config.Threshold != tt.wantThreshold {
				t.Errorf("Threshold = %f, want %f", matcher.config.Threshold, tt.wantThreshold)
			}
			if matcher.config.Algorithm != tt.wantAlgorithm {
				t.Errorf("Algorithm = %q, want %q", matcher.config.Algorithm, tt.wantAlgorithm)
			}
			if matcher.config.NGramSize != tt.wantNGramSize {
				t.Errorf("NGramSize = %d, want %d", matcher.config.NGramSize, tt.wantNGramSize)
			}
		})
	}
}

func TestLongestCommonSubsequenceLength(t *testing.T) {
	tests := []struct {
		name     string
		source   []string
		target   []string
		expected int
	}{
		{
			name:     "identical sequences",
			source:   []string{"a", "b", "c"},
			target:   []string{"a", "b", "c"},
			expected: 3,
		},
		{
			name:     "empty source",
			source:   []string{},
			target:   []string{"a", "b"},
			expected: 0,
		},
		{
			name:     "empty target",
			source:   []string{"a", "b"},
			target:   []string{},
			expected: 0,
		},
		{
			name:     "both empty",
			source:   []string{},
			target:   []string{},
			expected: 0,
		},
		{
			name:     "no common elements",
			source:   []string{"a", "b", "c"},
			target:   []string{"x", "y", "z"},
			expected: 0,
		},
		{
			name:     "partial overlap",
			source:   []string{"a", "b", "c", "d"},
			target:   []string{"b", "c", "e"},
			expected: 2, // "b", "c"
		},
		{
			name:     "interleaved",
			source:   []string{"a", "b", "c", "d", "e"},
			target:   []string{"a", "c", "e"},
			expected: 3, // "a", "c", "e"
		},
		{
			name:     "lines of code",
			source:   []string{"func main() {", "  fmt.Println(\"hello\")", "}"},
			target:   []string{"func main() {", "  fmt.Println(\"world\")", "}"},
			expected: 2, // first and last line match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := longestCommonSubsequenceLength(tt.source, tt.target)
			if got != tt.expected {
				t.Errorf("longestCommonSubsequenceLength() = %d, want %d", got, tt.expected)
			}
			// Verify symmetry
			got2 := longestCommonSubsequenceLength(tt.target, tt.source)
			if got2 != tt.expected {
				t.Errorf("longestCommonSubsequenceLength() (swapped) = %d, want %d", got2, tt.expected)
			}
		})
	}
}

func TestGenerateNGrams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		n        int
		expected map[string]struct{}
	}{
		{
			name:  "trigrams from hello",
			input: "hello",
			n:     3,
			expected: map[string]struct{}{
				"hel": {},
				"ell": {},
				"llo": {},
			},
		},
		{
			name:     "string shorter than n",
			input:    "ab",
			n:        3,
			expected: map[string]struct{}{"ab": {}},
		},
		{
			name:     "empty string",
			input:    "",
			n:        3,
			expected: map[string]struct{}{},
		},
		{
			name:  "bigrams",
			input: "abc",
			n:     2,
			expected: map[string]struct{}{
				"ab": {},
				"bc": {},
			},
		},
		{
			name:  "unicode characters",
			input: "日本語",
			n:     2,
			expected: map[string]struct{}{
				"日本": {},
				"本語": {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateNGrams(tt.input, tt.n)
			if len(got) != len(tt.expected) {
				t.Errorf("generateNGrams() returned %d ngrams, want %d", len(got), len(tt.expected))
			}
			for ngram := range tt.expected {
				if _, exists := got[ngram]; !exists {
					t.Errorf("generateNGrams() missing expected ngram %q", ngram)
				}
			}
		})
	}
}

func TestTokenSet(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []string
		expected map[string]struct{}
	}{
		{
			name:   "simple tokens",
			tokens: []string{"a", "b", "c"},
			expected: map[string]struct{}{
				"a": {},
				"b": {},
				"c": {},
			},
		},
		{
			name:   "with duplicates",
			tokens: []string{"a", "b", "a", "c", "b"},
			expected: map[string]struct{}{
				"a": {},
				"b": {},
				"c": {},
			},
		},
		{
			name:   "with empty strings",
			tokens: []string{"a", "", "b", "  ", "c"},
			expected: map[string]struct{}{
				"a": {},
				"b": {},
				"c": {},
			},
		},
		{
			name:   "with whitespace",
			tokens: []string{"  a  ", "b", "  c"},
			expected: map[string]struct{}{
				"a": {},
				"b": {},
				"c": {},
			},
		},
		{
			name:     "all empty",
			tokens:   []string{"", "  ", "\t"},
			expected: map[string]struct{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenSet(tt.tokens)
			if len(got) != len(tt.expected) {
				t.Errorf("tokenSet() returned %d tokens, want %d", len(got), len(tt.expected))
			}
			for token := range tt.expected {
				if _, exists := got[token]; !exists {
					t.Errorf("tokenSet() missing expected token %q", token)
				}
			}
		})
	}
}

func TestJaccardIndex(t *testing.T) {
	tests := []struct {
		name     string
		set1     map[string]struct{}
		set2     map[string]struct{}
		expected float64
		delta    float64
	}{
		{
			name:     "identical sets",
			set1:     map[string]struct{}{"a": {}, "b": {}, "c": {}},
			set2:     map[string]struct{}{"a": {}, "b": {}, "c": {}},
			expected: 1.0,
			delta:    0.001,
		},
		{
			name:     "no overlap",
			set1:     map[string]struct{}{"a": {}, "b": {}},
			set2:     map[string]struct{}{"x": {}, "y": {}},
			expected: 0.0,
			delta:    0.001,
		},
		{
			name:     "both empty",
			set1:     map[string]struct{}{},
			set2:     map[string]struct{}{},
			expected: 1.0,
			delta:    0.001,
		},
		{
			name:     "one empty",
			set1:     map[string]struct{}{"a": {}},
			set2:     map[string]struct{}{},
			expected: 0.0,
			delta:    0.001,
		},
		{
			name:     "partial overlap",
			set1:     map[string]struct{}{"a": {}, "b": {}, "c": {}},
			set2:     map[string]struct{}{"b": {}, "c": {}, "d": {}},
			expected: 0.5, // intersection=2, union=4
			delta:    0.001,
		},
		{
			name:     "subset",
			set1:     map[string]struct{}{"a": {}, "b": {}},
			set2:     map[string]struct{}{"a": {}, "b": {}, "c": {}, "d": {}},
			expected: 0.5, // intersection=2, union=4
			delta:    0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jaccardIndex(tt.set1, tt.set2)
			if math.Abs(got-tt.expected) > tt.delta {
				t.Errorf("jaccardIndex() = %f, want %f (±%f)", got, tt.expected, tt.delta)
			}
			// Verify symmetry
			got2 := jaccardIndex(tt.set2, tt.set1)
			if math.Abs(got2-tt.expected) > tt.delta {
				t.Errorf("jaccardIndex() (swapped) = %f, want %f (±%f)", got2, tt.expected, tt.delta)
			}
		})
	}
}

func TestContentMatcher_Compare(t *testing.T) {
	tests := []struct {
		name     string
		config   ContentMatcherConfig
		content1 string
		content2 string
		expected float64
		delta    float64
	}{
		{
			name:     "identical content",
			config:   DefaultContentMatcherConfig(),
			content1: "hello world",
			content2: "hello world",
			expected: 1.0,
			delta:    0.001,
		},
		{
			name:     "empty content",
			config:   DefaultContentMatcherConfig(),
			content1: "",
			content2: "",
			expected: 1.0, // both empty is considered identical
			delta:    0.001,
		},
		{
			name:     "one empty",
			config:   DefaultContentMatcherConfig(),
			content1: "hello",
			content2: "",
			expected: 0.0,
			delta:    0.001,
		},
		{
			name: "line mode identical lines",
			config: ContentMatcherConfig{
				Algorithm: "lcs",
				LineMode:  true,
			},
			content1: "line1\nline2\nline3",
			content2: "line1\nline2\nline3",
			expected: 1.0,
			delta:    0.001,
		},
		{
			name: "line mode partial match",
			config: ContentMatcherConfig{
				Algorithm: "lcs",
				LineMode:  true,
			},
			content1: "line1\nline2\nline3\nline4",
			content2: "line1\nline3\nline5",
			expected: 0.5, // 2 lines match out of 4
			delta:    0.01,
		},
		{
			name: "jaccard line mode",
			config: ContentMatcherConfig{
				Algorithm: "jaccard",
				LineMode:  true,
			},
			content1: "line1\nline2\nline3",
			content2: "line1\nline2\nline4",
			expected: 0.5, // intersection=2, union=4
			delta:    0.01,
		},
		{
			name: "jaccard character mode",
			config: ContentMatcherConfig{
				Algorithm: "jaccard",
				LineMode:  false,
				NGramSize: 2,
			},
			content1: "hello",
			content2: "hella",
			expected: 0.6, // common: he, el, ll -> 3 of 5 unique bigrams
			delta:    0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewContentMatcher(tt.config)
			got := matcher.Compare(tt.content1, tt.content2)
			if math.Abs(got-tt.expected) > tt.delta {
				t.Errorf("Compare() = %f, want %f (±%f)", got, tt.expected, tt.delta)
			}
		})
	}
}

func TestContentMatcher_Compare_Algorithms(t *testing.T) {
	content1 := "func main() {\n\tfmt.Println(\"hello\")\n}"
	content2 := "func main() {\n\tfmt.Println(\"world\")\n}"

	tests := []struct {
		algorithm string
		minScore  float64 // at least this score
		maxScore  float64 // at most this score
	}{
		{"lcs", 0.5, 0.8},      // 2 of 3 lines match
		{"jaccard", 0.4, 0.8},  // some line overlap
		{"combined", 0.5, 0.8}, // max of both
	}

	for _, tt := range tests {
		t.Run(tt.algorithm, func(t *testing.T) {
			config := ContentMatcherConfig{
				Algorithm: tt.algorithm,
				LineMode:  true,
			}
			matcher := NewContentMatcher(config)
			score := matcher.Compare(content1, content2)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("Compare() with algorithm %q = %f, want between %f and %f",
					tt.algorithm, score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestContentMatcher_FindSimilar(t *testing.T) {
	skills := []model.Skill{
		{Name: "skill1", Content: "line1\nline2\nline3"},
		{Name: "skill2", Content: "line1\nline2\nline4"},                  // similar to skill1
		{Name: "skill3", Content: "completely\ndifferent\ncontent\nhere"}, // different
		{Name: "skill4", Content: "line1\nline2\nline3"},                  // identical to skill1
	}

	config := ContentMatcherConfig{
		Threshold: 0.5,
		Algorithm: "lcs",
		LineMode:  true,
	}
	matcher := NewContentMatcher(config)
	matches := matcher.FindSimilar(skills)

	// Expected matches:
	// - skill1 & skill2 (2/3 lines match = 0.67)
	// - skill1 & skill4 (identical = 1.0)
	// - skill2 & skill4 (2/3 lines match = 0.67)

	if len(matches) != 3 {
		t.Errorf("FindSimilar() returned %d matches, want 3", len(matches))
		for _, m := range matches {
			t.Logf("  %s <-> %s: %.2f", m.Skill1.Name, m.Skill2.Name, m.Score)
		}
	}

	// Verify no match includes skill3
	for _, m := range matches {
		if m.Skill1.Name == "skill3" || m.Skill2.Name == "skill3" {
			t.Errorf("FindSimilar() incorrectly matched skill3: %s <-> %s",
				m.Skill1.Name, m.Skill2.Name)
		}
	}
}

func TestContentMatcher_FindSimilar_Empty(t *testing.T) {
	matcher := NewContentMatcher(DefaultContentMatcherConfig())

	// Empty slice
	matches := matcher.FindSimilar([]model.Skill{})
	if len(matches) != 0 {
		t.Errorf("FindSimilar() with empty slice returned %d matches, want 0", len(matches))
	}

	// Single skill (no pairs possible)
	matches = matcher.FindSimilar([]model.Skill{{Name: "only", Content: "content"}})
	if len(matches) != 0 {
		t.Errorf("FindSimilar() with single skill returned %d matches, want 0", len(matches))
	}
}

func TestContentMatcher_FindSimilar_HighThreshold(t *testing.T) {
	skills := []model.Skill{
		{Name: "skill1", Content: "line1\nline2\nline3"},
		{Name: "skill2", Content: "line1\nline2\nline4"}, // 67% similar
	}

	config := ContentMatcherConfig{
		Threshold: 0.9, // High threshold
		Algorithm: "lcs",
		LineMode:  true,
	}
	matcher := NewContentMatcher(config)
	matches := matcher.FindSimilar(skills)

	// With 0.9 threshold, 67% similar skills should not match
	if len(matches) != 0 {
		t.Errorf("FindSimilar() with high threshold returned %d matches, want 0", len(matches))
	}
}

func TestContentMatcher_RealWorldContent(t *testing.T) {
	// Simulate real skill content
	skill1Content := `# Git Commit Helper
This skill helps create conventional commits.

## Usage
Run /commit to start the commit process.

## Features
- Follows conventional commit format
- Adds appropriate prefixes`

	skill2Content := `# Git Commit Assistant
This skill helps create conventional commits.

## Usage
Run /commit to begin committing.

## Features
- Follows conventional commit format
- Supports multiple commit types`

	skill3Content := `# Code Review Tool
This skill helps review pull requests.

## Usage
Run /review to start reviewing.

## Features
- Checks code quality
- Suggests improvements`

	config := DefaultContentMatcherConfig()
	matcher := NewContentMatcher(config)

	// Similar skills should have high score
	score12 := matcher.Compare(skill1Content, skill2Content)
	if score12 < 0.4 {
		t.Errorf("Similar skills score = %f, want >= 0.4", score12)
	}

	// Different skills should have lower score than similar ones
	score13 := matcher.Compare(skill1Content, skill3Content)
	if score13 > 0.5 {
		t.Errorf("Different skills score = %f, want <= 0.5", score13)
	}

	// Same skill should be identical
	score11 := matcher.Compare(skill1Content, skill1Content)
	if score11 != 1.0 {
		t.Errorf("Identical content score = %f, want 1.0", score11)
	}
}

func BenchmarkLongestCommonSubsequenceLength(b *testing.B) {
	// Create test sequences
	source := make([]string, 100)
	target := make([]string, 100)
	for i := range 100 {
		source[i] = "line" + string(rune('A'+i%26))
		target[i] = "line" + string(rune('A'+(i+5)%26))
	}

	b.ResetTimer()
	for b.Loop() {
		longestCommonSubsequenceLength(source, target)
	}
}

func BenchmarkJaccardIndex(b *testing.B) {
	set1 := make(map[string]struct{})
	set2 := make(map[string]struct{})
	for i := range 100 {
		set1["item"+string(rune('A'+i%26))] = struct{}{}
		set2["item"+string(rune('A'+(i+5)%26))] = struct{}{}
	}

	b.ResetTimer()
	for b.Loop() {
		jaccardIndex(set1, set2)
	}
}

func BenchmarkContentMatcher_Compare(b *testing.B) {
	content1 := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10"
	content2 := "line1\nline2\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11"

	matcher := NewContentMatcher(DefaultContentMatcherConfig())

	b.ResetTimer()
	for b.Loop() {
		matcher.Compare(content1, content2)
	}
}
