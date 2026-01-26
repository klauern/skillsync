// Package similarity provides algorithms for finding similar skills by name or content.
package similarity

import (
	"log/slog"
	"strings"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
)

// ContentMatch represents a pair of skills with their content similarity score.
type ContentMatch struct {
	Skill1    model.Skill `json:"skill1"`
	Skill2    model.Skill `json:"skill2"`
	Score     float64     `json:"score"`
	Algorithm string      `json:"algorithm"`
}

// ContentMatcherConfig configures the content similarity matching behavior.
type ContentMatcherConfig struct {
	// Threshold is the minimum similarity score (0.0-1.0) to consider a match.
	// Default: 0.6
	Threshold float64
	// Algorithm specifies which algorithm to use: "lcs", "jaccard", or "combined".
	// Default: "combined"
	Algorithm string
	// NGramSize is the size of n-grams for Jaccard similarity.
	// Default: 3 (trigrams)
	NGramSize int
	// LineMode enables line-based comparison instead of character-based.
	// Default: true
	LineMode bool
}

// DefaultContentMatcherConfig returns sensible defaults for content matching.
func DefaultContentMatcherConfig() ContentMatcherConfig {
	return ContentMatcherConfig{
		Threshold: 0.6,
		Algorithm: "combined",
		NGramSize: 3,
		LineMode:  true,
	}
}

// ContentMatcher finds skills with similar content.
type ContentMatcher struct {
	config ContentMatcherConfig
}

// NewContentMatcher creates a new content matcher with the given configuration.
func NewContentMatcher(config ContentMatcherConfig) *ContentMatcher {
	if config.Threshold <= 0 || config.Threshold > 1 {
		config.Threshold = 0.6
	}
	if config.Algorithm == "" {
		config.Algorithm = "combined"
	}
	if config.NGramSize <= 0 {
		config.NGramSize = 3
	}
	return &ContentMatcher{config: config}
}

// FindSimilar finds all pairs of skills with similar content above the threshold.
func (m *ContentMatcher) FindSimilar(skills []model.Skill) []ContentMatch {
	logging.Debug("finding similar skill content",
		logging.Operation("content_similarity"),
		logging.Count(len(skills)),
		slog.Float64("threshold", m.config.Threshold),
		slog.String("algorithm", m.config.Algorithm),
	)

	var matches []ContentMatch

	// Compare all pairs (O(n^2) but typically small number of skills)
	for i := range len(skills) {
		skillI := skills[i]
		for j := i + 1; j < len(skills); j++ {
			skillJ := skills[j]
			score := m.Compare(skillI.Content, skillJ.Content)
			if score >= m.config.Threshold {
				matches = append(matches, ContentMatch{
					Skill1:    skillI,
					Skill2:    skillJ,
					Score:     score,
					Algorithm: m.config.Algorithm,
				})
				logging.Debug("found similar content",
					slog.String("name1", skillI.Name),
					slog.String("name2", skillJ.Name),
					slog.Float64("score", score),
				)
			}
		}
	}

	logging.Debug("content similarity search complete",
		logging.Operation("content_similarity"),
		slog.Int("matches_found", len(matches)),
	)

	return matches
}

// Compare returns the similarity score between two content strings (0.0-1.0).
func (m *ContentMatcher) Compare(content1, content2 string) float64 {
	// Early exit for exact matches
	if content1 == content2 {
		return 1.0
	}

	// Early exit for empty strings
	if len(content1) == 0 || len(content2) == 0 {
		return 0.0
	}

	switch m.config.Algorithm {
	case "lcs":
		return m.lcsSimilarity(content1, content2)
	case "jaccard":
		return m.jaccardSimilarity(content1, content2)
	case "combined":
		// Use the higher of the two scores
		lcs := m.lcsSimilarity(content1, content2)
		jaccard := m.jaccardSimilarity(content1, content2)
		return max(lcs, jaccard)
	default:
		return m.lcsSimilarity(content1, content2)
	}
}

// lcsSimilarity calculates similarity based on Longest Common Subsequence.
// Returns the ratio of LCS length to the maximum content length.
func (m *ContentMatcher) lcsSimilarity(content1, content2 string) float64 {
	var seq1, seq2 []string

	if m.config.LineMode {
		seq1 = strings.Split(content1, "\n")
		seq2 = strings.Split(content2, "\n")
	} else {
		// Character mode: split into individual characters
		seq1 = strings.Split(content1, "")
		seq2 = strings.Split(content2, "")
	}

	lcsLen := longestCommonSubsequenceLength(seq1, seq2)
	maxLen := max(len(seq1), len(seq2))

	if maxLen == 0 {
		return 1.0
	}

	return float64(lcsLen) / float64(maxLen)
}

// longestCommonSubsequenceLength finds the length of the LCS of two string slices.
// This is an optimized version that only returns the length, not the actual LCS.
func longestCommonSubsequenceLength(source, target []string) int {
	m, n := len(source), len(target)
	if m == 0 || n == 0 {
		return 0
	}

	// Use two rows instead of full matrix to save memory: O(min(m,n)) space
	// Ensure we iterate over the longer sequence to minimize space usage
	if m < n {
		source, target = target, source
		m, n = n, m
	}

	prev := make([]int, n+1)
	curr := make([]int, n+1)

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if source[i-1] == target[j-1] {
				curr[j] = prev[j-1] + 1
			} else {
				curr[j] = max(prev[j], curr[j-1])
			}
		}
		prev, curr = curr, prev
	}

	return prev[n]
}

// jaccardSimilarity calculates Jaccard similarity using n-grams.
// Returns the size of intersection divided by size of union of n-gram sets.
func (m *ContentMatcher) jaccardSimilarity(content1, content2 string) float64 {
	var ngrams1, ngrams2 map[string]struct{}

	if m.config.LineMode {
		// In line mode, use lines as tokens directly
		ngrams1 = tokenSet(strings.Split(content1, "\n"))
		ngrams2 = tokenSet(strings.Split(content2, "\n"))
	} else {
		// In character mode, use character n-grams
		ngrams1 = generateNGrams(content1, m.config.NGramSize)
		ngrams2 = generateNGrams(content2, m.config.NGramSize)
	}

	return jaccardIndex(ngrams1, ngrams2)
}

// generateNGrams creates a set of character n-grams from a string.
func generateNGrams(s string, n int) map[string]struct{} {
	ngrams := make(map[string]struct{})
	runes := []rune(s)

	if len(runes) < n {
		// If string is shorter than n, use the whole string as single ngram
		if len(runes) > 0 {
			ngrams[s] = struct{}{}
		}
		return ngrams
	}

	for i := 0; i <= len(runes)-n; i++ {
		ngram := string(runes[i : i+n])
		ngrams[ngram] = struct{}{}
	}

	return ngrams
}

// tokenSet creates a set from a slice of strings (removes duplicates).
func tokenSet(tokens []string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, token := range tokens {
		// Normalize: trim whitespace and skip empty lines
		token = strings.TrimSpace(token)
		if token != "" {
			set[token] = struct{}{}
		}
	}
	return set
}

// jaccardIndex calculates the Jaccard index between two sets.
// Returns |intersection| / |union|.
func jaccardIndex(set1, set2 map[string]struct{}) float64 {
	if len(set1) == 0 && len(set2) == 0 {
		return 1.0
	}
	if len(set1) == 0 || len(set2) == 0 {
		return 0.0
	}

	// Calculate intersection size
	intersection := 0
	for key := range set1 {
		if _, exists := set2[key]; exists {
			intersection++
		}
	}

	// Union size = |set1| + |set2| - |intersection|
	union := len(set1) + len(set2) - intersection

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}
