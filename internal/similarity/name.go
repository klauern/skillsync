// Package similarity provides algorithms for finding similar skills by name or content.
package similarity

import (
	"log/slog"
	"strings"
	"unicode"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
)

// NameMatch represents a pair of skills with their similarity score.
type NameMatch struct {
	Skill1     model.Skill `json:"skill1"`
	Skill2     model.Skill `json:"skill2"`
	Score      float64     `json:"score"`
	Algorithm  string      `json:"algorithm"`
	Normalized bool        `json:"normalized"`
}

// NameMatcherConfig configures the name similarity matching behavior.
type NameMatcherConfig struct {
	// Threshold is the minimum similarity score (0.0-1.0) to consider a match.
	// Default: 0.7
	Threshold float64
	// Algorithm specifies which algorithm to use: "levenshtein", "jaro-winkler", or "combined".
	// Default: "combined"
	Algorithm string
	// Normalize enables string normalization before comparison.
	// Default: true
	Normalize bool
	// CaseSensitive disables case-insensitive matching.
	// Default: false (case-insensitive)
	CaseSensitive bool
}

// DefaultNameMatcherConfig returns sensible defaults for name matching.
func DefaultNameMatcherConfig() NameMatcherConfig {
	return NameMatcherConfig{
		Threshold:     0.7,
		Algorithm:     "combined",
		Normalize:     true,
		CaseSensitive: false,
	}
}

// NameMatcher finds skills with similar names.
type NameMatcher struct {
	config NameMatcherConfig
}

// NewNameMatcher creates a new name matcher with the given configuration.
func NewNameMatcher(config NameMatcherConfig) *NameMatcher {
	if config.Threshold <= 0 || config.Threshold > 1 {
		config.Threshold = 0.7
	}
	if config.Algorithm == "" {
		config.Algorithm = "combined"
	}
	return &NameMatcher{config: config}
}

// FindSimilar finds all pairs of skills with similar names above the threshold.
func (m *NameMatcher) FindSimilar(skills []model.Skill) []NameMatch {
	logging.Debug("finding similar skill names",
		logging.Operation("name_similarity"),
		logging.Count(len(skills)),
		slog.Float64("threshold", m.config.Threshold),
		slog.String("algorithm", m.config.Algorithm),
	)

	var matches []NameMatch

	// Compare all pairs (O(n^2) but typically small number of skills)
	for i := range len(skills) {
		for j := i + 1; j < len(skills); j++ {
			score := m.Compare(skills[i].Name, skills[j].Name)
			if score >= m.config.Threshold {
				matches = append(matches, NameMatch{
					Skill1:     skills[i],
					Skill2:     skills[j],
					Score:      score,
					Algorithm:  m.config.Algorithm,
					Normalized: m.config.Normalize,
				})
				logging.Debug("found similar names",
					slog.String("name1", skills[i].Name),
					slog.String("name2", skills[j].Name),
					slog.Float64("score", score),
				)
			}
		}
	}

	logging.Debug("name similarity search complete",
		logging.Operation("name_similarity"),
		slog.Int("matches_found", len(matches)),
	)

	return matches
}

// Compare returns the similarity score between two names (0.0-1.0).
func (m *NameMatcher) Compare(name1, name2 string) float64 {
	// Normalize if configured
	if m.config.Normalize {
		name1 = normalizeName(name1)
		name2 = normalizeName(name2)
	} else if !m.config.CaseSensitive {
		name1 = strings.ToLower(name1)
		name2 = strings.ToLower(name2)
	}

	// Early exit for exact matches
	if name1 == name2 {
		return 1.0
	}

	// Early exit for empty strings
	if len(name1) == 0 || len(name2) == 0 {
		return 0.0
	}

	switch m.config.Algorithm {
	case "levenshtein":
		return LevenshteinSimilarity(name1, name2)
	case "jaro-winkler":
		return JaroWinkler(name1, name2)
	case "combined":
		// Use the higher of the two scores
		lev := LevenshteinSimilarity(name1, name2)
		jw := JaroWinkler(name1, name2)
		return max(lev, jw)
	default:
		return LevenshteinSimilarity(name1, name2)
	}
}

// normalizeName prepares a name for comparison by:
// - Converting to lowercase
// - Removing special characters except hyphens and underscores
// - Collapsing multiple separators
// - Trimming whitespace
func normalizeName(s string) string {
	s = strings.ToLower(s)

	// Replace common separators with single space
	var result strings.Builder
	result.Grow(len(s))

	prevSpace := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			result.WriteRune(r)
			prevSpace = false
		case r == '-' || r == '_' || r == ' ' || r == '.':
			if !prevSpace {
				result.WriteRune(' ')
				prevSpace = true
			}
		}
	}

	return strings.TrimSpace(result.String())
}

// LevenshteinDistance calculates the minimum number of single-character edits
// (insertions, deletions, or substitutions) required to change one string into another.
func LevenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Convert to runes for proper Unicode handling
	r1 := []rune(s1)
	r2 := []rune(s2)

	// Optimize for shorter string as columns
	if len(r1) < len(r2) {
		r1, r2 = r2, r1
	}

	// Use two rows instead of full matrix to save memory: O(min(m,n)) space
	prev := make([]int, len(r2)+1)
	curr := make([]int, len(r2)+1)

	// Initialize first row
	for j := range prev {
		prev[j] = j
	}

	// Fill the matrix
	for i := 1; i <= len(r1); i++ {
		curr[0] = i
		for j := 1; j <= len(r2); j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			curr[j] = min(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}

	return prev[len(r2)]
}

// LevenshteinSimilarity returns a normalized similarity score (0.0-1.0)
// based on Levenshtein distance.
func LevenshteinSimilarity(s1, s2 string) float64 {
	if len(s1) == 0 && len(s2) == 0 {
		return 1.0
	}

	distance := LevenshteinDistance(s1, s2)
	maxLen := max(len([]rune(s1)), len([]rune(s2)))

	return 1.0 - float64(distance)/float64(maxLen)
}

// JaroSimilarity calculates the Jaro similarity between two strings.
// Returns a value between 0.0 (no similarity) and 1.0 (identical).
func JaroSimilarity(s1, s2 string) float64 {
	r1 := []rune(s1)
	r2 := []rune(s2)

	if len(r1) == 0 && len(r2) == 0 {
		return 1.0
	}
	if len(r1) == 0 || len(r2) == 0 {
		return 0.0
	}

	// Match window: max(|s1|, |s2|) / 2 - 1
	matchWindow := max(0, max(len(r1), len(r2))/2-1)

	s1Matches := make([]bool, len(r1))
	s2Matches := make([]bool, len(r2))

	matches := 0
	transpositions := 0

	// Find matching characters
	for i := range r1 {
		start := max(0, i-matchWindow)
		end := min(len(r2), i+matchWindow+1)

		for j := start; j < end; j++ {
			if s2Matches[j] || r1[i] != r2[j] {
				continue
			}
			s1Matches[i] = true
			s2Matches[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	// Count transpositions
	k := 0
	for i := range r1 {
		if !s1Matches[i] {
			continue
		}
		for !s2Matches[k] {
			k++
		}
		if r1[i] != r2[k] {
			transpositions++
		}
		k++
	}

	jaro := (float64(matches)/float64(len(r1)) +
		float64(matches)/float64(len(r2)) +
		float64(matches-transpositions/2)/float64(matches)) / 3.0

	return jaro
}

// JaroWinkler calculates the Jaro-Winkler similarity, which gives more
// weight to strings that match from the beginning (good for names).
// Returns a value between 0.0 and 1.0.
func JaroWinkler(s1, s2 string) float64 {
	jaro := JaroSimilarity(s1, s2)

	// Calculate common prefix length (up to 4 characters)
	r1 := []rune(s1)
	r2 := []rune(s2)

	prefixLen := 0
	maxPrefix := min(4, min(len(r1), len(r2)))
	for i := range maxPrefix {
		if r1[i] == r2[i] {
			prefixLen++
		} else {
			break
		}
	}

	// Winkler modification: boost score for common prefix
	// Standard scaling factor is 0.1
	const scalingFactor = 0.1
	return jaro + float64(prefixLen)*scalingFactor*(1.0-jaro)
}
