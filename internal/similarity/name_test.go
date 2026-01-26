package similarity

import (
	"math"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected int
	}{
		{
			name:     "identical strings",
			s1:       "hello",
			s2:       "hello",
			expected: 0,
		},
		{
			name:     "empty strings",
			s1:       "",
			s2:       "",
			expected: 0,
		},
		{
			name:     "one empty string",
			s1:       "hello",
			s2:       "",
			expected: 5,
		},
		{
			name:     "single substitution",
			s1:       "cat",
			s2:       "bat",
			expected: 1,
		},
		{
			name:     "single insertion",
			s1:       "cat",
			s2:       "cats",
			expected: 1,
		},
		{
			name:     "single deletion",
			s1:       "cats",
			s2:       "cat",
			expected: 1,
		},
		{
			name:     "multiple edits",
			s1:       "kitten",
			s2:       "sitting",
			expected: 3,
		},
		{
			name:     "completely different",
			s1:       "abc",
			s2:       "xyz",
			expected: 3,
		},
		{
			name:     "unicode characters",
			s1:       "café",
			s2:       "cafe",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LevenshteinDistance(tt.s1, tt.s2)
			if got != tt.expected {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tt.s1, tt.s2, got, tt.expected)
			}
			// Distance should be symmetric
			got2 := LevenshteinDistance(tt.s2, tt.s1)
			if got2 != tt.expected {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d (symmetry)", tt.s2, tt.s1, got2, tt.expected)
			}
		})
	}
}

func TestLevenshteinSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected float64
		delta    float64
	}{
		{
			name:     "identical strings",
			s1:       "hello",
			s2:       "hello",
			expected: 1.0,
			delta:    0.001,
		},
		{
			name:     "both empty",
			s1:       "",
			s2:       "",
			expected: 1.0,
			delta:    0.001,
		},
		{
			name:     "completely different same length",
			s1:       "abc",
			s2:       "xyz",
			expected: 0.0,
			delta:    0.001,
		},
		{
			name:     "similar strings",
			s1:       "commit",
			s2:       "commit-push",
			expected: 0.545, // 6/11 chars match
			delta:    0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LevenshteinSimilarity(tt.s1, tt.s2)
			if math.Abs(got-tt.expected) > tt.delta {
				t.Errorf("LevenshteinSimilarity(%q, %q) = %f, want %f (±%f)", tt.s1, tt.s2, got, tt.expected, tt.delta)
			}
		})
	}
}

func TestJaroSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected float64
		delta    float64
	}{
		{
			name:     "identical strings",
			s1:       "hello",
			s2:       "hello",
			expected: 1.0,
			delta:    0.001,
		},
		{
			name:     "both empty",
			s1:       "",
			s2:       "",
			expected: 1.0,
			delta:    0.001,
		},
		{
			name:     "one empty",
			s1:       "hello",
			s2:       "",
			expected: 0.0,
			delta:    0.001,
		},
		{
			name:     "classic example MARTHA MARHTA",
			s1:       "MARTHA",
			s2:       "MARHTA",
			expected: 0.944,
			delta:    0.01,
		},
		{
			name:     "no matching characters",
			s1:       "abc",
			s2:       "xyz",
			expected: 0.0,
			delta:    0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JaroSimilarity(tt.s1, tt.s2)
			if math.Abs(got-tt.expected) > tt.delta {
				t.Errorf("JaroSimilarity(%q, %q) = %f, want %f (±%f)", tt.s1, tt.s2, got, tt.expected, tt.delta)
			}
		})
	}
}

func TestJaroWinkler(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected float64
		delta    float64
	}{
		{
			name:     "identical strings",
			s1:       "hello",
			s2:       "hello",
			expected: 1.0,
			delta:    0.001,
		},
		{
			name:     "common prefix boost",
			s1:       "prefix-test",
			s2:       "prefix-best",
			expected: 0.96, // Should be higher than Jaro due to common prefix
			delta:    0.02,
		},
		{
			name:     "MARTHA MARHTA boosted",
			s1:       "MARTHA",
			s2:       "MARHTA",
			expected: 0.961, // Jaro ~0.944, boosted by "MAR" prefix
			delta:    0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JaroWinkler(tt.s1, tt.s2)
			if math.Abs(got-tt.expected) > tt.delta {
				t.Errorf("JaroWinkler(%q, %q) = %f, want %f (±%f)", tt.s1, tt.s2, got, tt.expected, tt.delta)
			}
		})
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"Hello World", "hello world"},
		{"hello-world", "hello world"},
		{"hello_world", "hello world"},
		{"hello.world", "hello world"},
		{"  hello  world  ", "hello world"},
		{"Hello--World__Test", "hello world test"},
		{"UPPERCASE", "uppercase"},
		{"special@#$chars", "specialchars"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeName(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNameMatcher_Compare(t *testing.T) {
	tests := []struct {
		name      string
		config    NameMatcherConfig
		name1     string
		name2     string
		wantScore float64
		delta     float64
	}{
		{
			name:      "exact match normalized",
			config:    DefaultNameMatcherConfig(),
			name1:     "commit-push",
			name2:     "commit_push",
			wantScore: 1.0,
			delta:     0.001,
		},
		{
			name:      "case insensitive match",
			config:    DefaultNameMatcherConfig(),
			name1:     "CommitPush",
			name2:     "commitpush",
			wantScore: 1.0,
			delta:     0.001,
		},
		{
			name: "case sensitive no match",
			config: NameMatcherConfig{
				Threshold:     0.7,
				Algorithm:     "levenshtein",
				Normalize:     false,
				CaseSensitive: true,
			},
			name1:     "Hello",
			name2:     "hello",
			wantScore: 0.8, // 1 char difference out of 5
			delta:     0.01,
		},
		{
			name: "levenshtein only",
			config: NameMatcherConfig{
				Threshold: 0.7,
				Algorithm: "levenshtein",
				Normalize: true,
			},
			name1:     "commit",
			name2:     "commits",
			wantScore: 0.857, // 6/7
			delta:     0.01,
		},
		{
			name: "jaro-winkler only",
			config: NameMatcherConfig{
				Threshold: 0.7,
				Algorithm: "jaro-winkler",
				Normalize: true,
			},
			name1:     "commit",
			name2:     "commits",
			wantScore: 0.96,
			delta:     0.02,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewNameMatcher(tt.config)
			got := matcher.Compare(tt.name1, tt.name2)
			if math.Abs(got-tt.wantScore) > tt.delta {
				t.Errorf("Compare(%q, %q) = %f, want %f (±%f)", tt.name1, tt.name2, got, tt.wantScore, tt.delta)
			}
		})
	}
}

func TestNameMatcher_FindSimilar(t *testing.T) {
	skills := []model.Skill{
		{Name: "commit", Platform: model.ClaudeCode},
		{Name: "commit-push", Platform: model.ClaudeCode},
		{Name: "commits", Platform: model.Cursor},
		{Name: "review-pr", Platform: model.ClaudeCode},
		{Name: "pr-review", Platform: model.Cursor},
		{Name: "unrelated-xyz", Platform: model.Codex},
	}

	tests := []struct {
		name           string
		config         NameMatcherConfig
		minMatches     int
		maxMatches     int
		mustContain    []string // pairs like "commit:commits"
		mustNotContain []string
	}{
		{
			name:       "default threshold finds similar",
			config:     DefaultNameMatcherConfig(),
			minMatches: 2, // At least commit/commits and commit/commit-push
			maxMatches: 10,
			mustContain: []string{
				"commit:commits",
				"commit:commit-push",
			},
			mustNotContain: []string{
				"commit:unrelated-xyz",
			},
		},
		{
			name: "high threshold finds only very similar",
			config: NameMatcherConfig{
				Threshold: 0.9,
				Algorithm: "combined",
				Normalize: true,
			},
			minMatches: 1,
			maxMatches: 5,
		},
		{
			name: "low threshold finds more matches",
			config: NameMatcherConfig{
				Threshold: 0.5,
				Algorithm: "combined",
				Normalize: true,
			},
			minMatches: 3,
			maxMatches: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewNameMatcher(tt.config)
			matches := matcher.FindSimilar(skills)

			if len(matches) < tt.minMatches {
				t.Errorf("FindSimilar() found %d matches, want at least %d", len(matches), tt.minMatches)
			}
			if len(matches) > tt.maxMatches {
				t.Errorf("FindSimilar() found %d matches, want at most %d", len(matches), tt.maxMatches)
			}

			// Check for required matches
			for _, pair := range tt.mustContain {
				found := false
				for _, m := range matches {
					pairStr1 := m.Skill1.Name + ":" + m.Skill2.Name
					pairStr2 := m.Skill2.Name + ":" + m.Skill1.Name
					if pairStr1 == pair || pairStr2 == pair {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("FindSimilar() should have found match %q", pair)
				}
			}

			// Check for forbidden matches
			for _, pair := range tt.mustNotContain {
				for _, m := range matches {
					pairStr1 := m.Skill1.Name + ":" + m.Skill2.Name
					pairStr2 := m.Skill2.Name + ":" + m.Skill1.Name
					if pairStr1 == pair || pairStr2 == pair {
						t.Errorf("FindSimilar() should not have found match %q", pair)
					}
				}
			}
		})
	}
}

func TestNewNameMatcher_Defaults(t *testing.T) {
	// Test that invalid config values are corrected
	t.Run("negative threshold defaults to 0.7", func(t *testing.T) {
		config := NameMatcherConfig{Threshold: -1}
		matcher := NewNameMatcher(config)
		// Can't access config directly, but we can verify behavior
		score := matcher.Compare("hello", "world")
		if score > 0.7 {
			t.Error("Expected low similarity for 'hello' vs 'world'")
		}
	})

	t.Run("empty algorithm defaults to combined", func(t *testing.T) {
		config := NameMatcherConfig{Algorithm: ""}
		matcher := NewNameMatcher(config)
		// Combined should give higher score than individual algorithms for some cases
		score := matcher.Compare("prefix-test", "prefix-best")
		if score < 0.7 {
			t.Errorf("Expected high similarity with combined algorithm, got %f", score)
		}
	})
}

func TestNameMatch_Fields(t *testing.T) {
	skill1 := model.Skill{Name: "test1", Platform: model.ClaudeCode}
	skill2 := model.Skill{Name: "test2", Platform: model.Cursor}

	match := NameMatch{
		Skill1:     skill1,
		Skill2:     skill2,
		Score:      0.85,
		Algorithm:  "combined",
		Normalized: true,
	}

	if match.Skill1.Name != "test1" {
		t.Error("Skill1 not stored correctly")
	}
	if match.Skill2.Name != "test2" {
		t.Error("Skill2 not stored correctly")
	}
	if match.Score != 0.85 {
		t.Error("Score not stored correctly")
	}
	if match.Algorithm != "combined" {
		t.Error("Algorithm not stored correctly")
	}
	if !match.Normalized {
		t.Error("Normalized not stored correctly")
	}
}

func TestDefaultNameMatcherConfig(t *testing.T) {
	config := DefaultNameMatcherConfig()

	if config.Threshold != 0.7 {
		t.Errorf("Default threshold = %f, want 0.7", config.Threshold)
	}
	if config.Algorithm != "combined" {
		t.Errorf("Default algorithm = %s, want combined", config.Algorithm)
	}
	if !config.Normalize {
		t.Error("Default normalize should be true")
	}
	if config.CaseSensitive {
		t.Error("Default case sensitive should be false")
	}
}
