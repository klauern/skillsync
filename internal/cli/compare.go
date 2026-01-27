// Package cli provides command definitions for skillsync.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/tiered"
	"github.com/klauern/skillsync/internal/similarity"
)

func compareCommand() *cli.Command {
	return &cli.Command{
		Name:    "compare",
		Aliases: []string{"cmp", "similar"},
		Usage:   "Find and display similar skills across platforms",
		UsageText: `skillsync compare [options]
   skillsync compare --name-threshold 0.8
   skillsync compare --content-threshold 0.7
   skillsync compare --platform claude-code
   skillsync compare --same-platform
   skillsync compare --format json`,
		Description: `Find skills that may be duplicates or variations based on name or content similarity.

   The compare command analyzes all discovered skills and identifies pairs that
   are similar based on configurable thresholds. This helps identify:
   - Duplicate skills that may need cleanup
   - Skills that have diverged across platforms
   - Potential candidates for consolidation
   - Redundant skills within a single platform (use --same-platform)

   Similarity matching:
   - Name similarity: Compares skill names using Levenshtein and Jaro-Winkler algorithms
   - Content similarity: Compares skill content using LCS and Jaccard algorithms

   Output formats:
   - table: Summary table of similar skill pairs (default)
   - unified: Unified diff format for each pair
   - side-by-side: Side-by-side comparison
   - summary: Statistics only
   - json: Machine-readable JSON output
   - yaml: Machine-readable YAML output`,
		Flags: []cli.Flag{
			&cli.Float64Flag{
				Name:    "name-threshold",
				Aliases: []string{"n"},
				Value:   0, // 0 means "use config value"
				Usage:   "Minimum name similarity score (0.0-1.0, default from config: 0.7)",
			},
			&cli.Float64Flag{
				Name:    "content-threshold",
				Aliases: []string{"c"},
				Value:   0, // 0 means "use config value"
				Usage:   "Minimum content similarity score (0.0-1.0, default from config: 0.6)",
			},
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Filter by platform (claude-code, cursor, codex)",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "table",
				Usage:   "Output format: table, unified, side-by-side, summary, json, yaml",
			},
			&cli.BoolFlag{
				Name:  "name-only",
				Usage: "Only compare by name similarity (skip content comparison)",
			},
			&cli.BoolFlag{
				Name:  "content-only",
				Usage: "Only compare by content similarity (skip name comparison)",
			},
			&cli.StringFlag{
				Name:  "algorithm",
				Value: "", // empty means "use config value"
				Usage: "Similarity algorithm: combined, levenshtein, jaro-winkler (for names) or combined, lcs, jaccard (for content, default from config: combined)",
			},
			&cli.BoolFlag{
				Name:    "same-platform",
				Aliases: []string{"s"},
				Usage:   "Only show similar skills within the same platform (helps find redundant skills)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return runCompare(cmd)
		},
	}
}

// compareConfig holds parsed configuration for the compare command.
type compareConfig struct {
	nameThreshold    float64
	contentThreshold float64
	platform         string
	format           string
	nameOnly         bool
	contentOnly      bool
	algorithm        string
	samePlatform     bool
}

func parseCompareConfig(cmd *cli.Command) (*compareConfig, error) {
	// Load configuration for defaults
	appConfig, err := config.Load()
	if err != nil {
		// If config fails to load, use hardcoded defaults
		appConfig = config.Default()
	}

	cfg := &compareConfig{
		nameThreshold:    cmd.Float64("name-threshold"),
		contentThreshold: cmd.Float64("content-threshold"),
		platform:         cmd.String("platform"),
		format:           cmd.String("format"),
		nameOnly:         cmd.Bool("name-only"),
		contentOnly:      cmd.Bool("content-only"),
		algorithm:        cmd.String("algorithm"),
		samePlatform:     cmd.Bool("same-platform"),
	}

	// Apply config defaults for unset values
	// A value of 0 for thresholds means "use config default"
	if cfg.nameThreshold == 0 {
		cfg.nameThreshold = appConfig.Similarity.NameThreshold
	}
	if cfg.contentThreshold == 0 {
		cfg.contentThreshold = appConfig.Similarity.ContentThreshold
	}
	if cfg.algorithm == "" {
		cfg.algorithm = appConfig.Similarity.Algorithm
	}

	// Validate thresholds (now after applying defaults)
	if cfg.nameThreshold < 0 || cfg.nameThreshold > 1 {
		return nil, errors.New("name-threshold must be between 0.0 and 1.0")
	}
	if cfg.contentThreshold < 0 || cfg.contentThreshold > 1 {
		return nil, errors.New("content-threshold must be between 0.0 and 1.0")
	}

	// Validate mutual exclusivity
	if cfg.nameOnly && cfg.contentOnly {
		return nil, errors.New("cannot use both --name-only and --content-only")
	}

	// Validate format
	validFormats := map[string]bool{
		"table": true, "unified": true, "side-by-side": true,
		"summary": true, "json": true, "yaml": true,
	}
	if !validFormats[cfg.format] {
		return nil, fmt.Errorf("invalid format: %s (use table, unified, side-by-side, summary, json, or yaml)", cfg.format)
	}

	return cfg, nil
}

func runCompare(cmd *cli.Command) error {
	cfg, err := parseCompareConfig(cmd)
	if err != nil {
		return err
	}

	// Discover skills
	skills, err := discoverSkillsForCompare(cfg.platform)
	if err != nil {
		return fmt.Errorf("failed to discover skills: %w", err)
	}

	if len(skills) < 2 {
		fmt.Println("Not enough skills to compare (need at least 2).")
		return nil
	}

	// Find similar skills
	results, err := findSimilarSkills(skills, cfg)
	if err != nil {
		return fmt.Errorf("failed to find similar skills: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No similar skills found matching the criteria.")
		return nil
	}

	// Sort by content similarity descending (highest similarity first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].ContentScore > results[j].ContentScore
	})

	// Output results
	return outputCompareResults(results, cfg.format)
}

// discoverSkillsForCompare discovers skills, optionally filtering by platform.
func discoverSkillsForCompare(platform string) ([]model.Skill, error) {
	var platforms []model.Platform
	if platform != "" {
		p, err := model.ParsePlatform(platform)
		if err != nil {
			return nil, fmt.Errorf("invalid platform: %w", err)
		}
		platforms = []model.Platform{p}
	} else {
		platforms = model.AllPlatforms()
	}

	var allSkills []model.Skill
	for _, p := range platforms {
		parser, err := tiered.NewForPlatform(p)
		if err != nil {
			fmt.Printf("Warning: failed to create parser for %s: %v\n", p, err)
			continue
		}
		skills, err := parser.Parse()
		if err != nil {
			fmt.Printf("Warning: failed to parse %s: %v\n", p, err)
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	return allSkills, nil
}

// findSimilarSkills finds similar skill pairs based on configuration.
func findSimilarSkills(skills []model.Skill, cfg *compareConfig) ([]*similarity.ComparisonResult, error) {
	var results []*similarity.ComparisonResult

	// Track pairs we've already compared to avoid duplicates
	comparedPairs := make(map[string]bool)

	// Name similarity matching
	if !cfg.contentOnly {
		nameConfig := similarity.NameMatcherConfig{
			Threshold: cfg.nameThreshold,
			Algorithm: cfg.algorithm,
			Normalize: true,
		}
		nameMatcher := similarity.NewNameMatcher(nameConfig)
		nameMatches := nameMatcher.FindSimilar(skills)

		for _, match := range nameMatches {
			pairKey := makePairKey(match.Skill1, match.Skill2)
			if comparedPairs[pairKey] {
				continue
			}
			comparedPairs[pairKey] = true

			// Compute content score if not name-only
			var contentScore float64
			if !cfg.nameOnly {
				contentConfig := similarity.ContentMatcherConfig{
					Threshold: 0, // Don't filter, we want the score
					Algorithm: cfg.algorithm,
					LineMode:  true,
				}
				contentMatcher := similarity.NewContentMatcher(contentConfig)
				contentScore = contentMatcher.Compare(match.Skill1.Content, match.Skill2.Content)
			}

			result := similarity.ComputeDiff(match.Skill1, match.Skill2, match.Score, contentScore)
			results = append(results, result)
		}
	}

	// Content similarity matching
	if !cfg.nameOnly {
		contentConfig := similarity.ContentMatcherConfig{
			Threshold: cfg.contentThreshold,
			Algorithm: cfg.algorithm,
			LineMode:  true,
		}
		contentMatcher := similarity.NewContentMatcher(contentConfig)
		contentMatches := contentMatcher.FindSimilar(skills)

		for _, match := range contentMatches {
			pairKey := makePairKey(match.Skill1, match.Skill2)
			if comparedPairs[pairKey] {
				continue
			}
			comparedPairs[pairKey] = true

			// Compute name score if not content-only
			var nameScore float64
			if !cfg.contentOnly {
				nameConfig := similarity.NameMatcherConfig{
					Threshold: 0, // Don't filter, we want the score
					Algorithm: cfg.algorithm,
					Normalize: true,
				}
				nameMatcher := similarity.NewNameMatcher(nameConfig)
				nameScore = nameMatcher.Compare(match.Skill1.Name, match.Skill2.Name)
			}

			result := similarity.ComputeDiff(match.Skill1, match.Skill2, nameScore, match.Score)
			results = append(results, result)
		}
	}

	// Filter to same-platform pairs if requested
	if cfg.samePlatform {
		filtered := make([]*similarity.ComparisonResult, 0, len(results))
		for _, result := range results {
			if result.Skill1.Platform == result.Skill2.Platform {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}

	return results, nil
}

// makePairKey creates a consistent key for a skill pair regardless of order.
func makePairKey(s1, s2 model.Skill) string {
	key1 := fmt.Sprintf("%s:%s:%s", s1.Platform, s1.Scope, s1.Name)
	key2 := fmt.Sprintf("%s:%s:%s", s2.Platform, s2.Scope, s2.Name)
	if key1 < key2 {
		return key1 + "|" + key2
	}
	return key2 + "|" + key1
}

// outputCompareResults outputs comparison results in the specified format.
func outputCompareResults(results []*similarity.ComparisonResult, format string) error {
	switch format {
	case "table":
		return similarity.FormatComparisonTable(os.Stdout, results)
	case "unified":
		return outputCompareUnified(results)
	case "side-by-side":
		return outputCompareSideBySide(results)
	case "summary":
		return outputCompareSummary(results)
	case "json":
		return outputCompareJSON(results)
	case "yaml":
		return outputCompareYAML(results)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func outputCompareUnified(results []*similarity.ComparisonResult) error {
	formatter := similarity.NewFormatter(similarity.FormatterConfig{
		Format:          similarity.FormatUnified,
		ContextLines:    3,
		ShowLineNumbers: true,
	})
	return formatter.FormatMultiple(os.Stdout, results)
}

func outputCompareSideBySide(results []*similarity.ComparisonResult) error {
	formatter := similarity.NewFormatter(similarity.FormatterConfig{
		Format:          similarity.FormatSideBySide,
		MaxWidth:        120,
		ShowLineNumbers: true,
	})
	return formatter.FormatMultiple(os.Stdout, results)
}

func outputCompareSummary(results []*similarity.ComparisonResult) error {
	formatter := similarity.NewFormatter(similarity.FormatterConfig{
		Format: similarity.FormatSummary,
	})
	return formatter.FormatMultiple(os.Stdout, results)
}

// comparisonOutput represents the JSON/YAML output structure.
type comparisonOutput struct {
	Skill1       string  `json:"skill1" yaml:"skill1"`
	Scope1       string  `json:"scope1" yaml:"scope1"`
	Platform1    string  `json:"platform1" yaml:"platform1"`
	Skill2       string  `json:"skill2" yaml:"skill2"`
	Scope2       string  `json:"scope2" yaml:"scope2"`
	Platform2    string  `json:"platform2" yaml:"platform2"`
	NameScore    float64 `json:"name_score,omitempty" yaml:"name_score,omitempty"`
	ContentScore float64 `json:"content_score,omitempty" yaml:"content_score,omitempty"`
	LinesAdded   int     `json:"lines_added" yaml:"lines_added"`
	LinesRemoved int     `json:"lines_removed" yaml:"lines_removed"`
	HunkCount    int     `json:"hunk_count" yaml:"hunk_count"`
}

func toComparisonOutputs(results []*similarity.ComparisonResult) []comparisonOutput {
	outputs := make([]comparisonOutput, len(results))
	for i, r := range results {
		outputs[i] = comparisonOutput{
			Skill1:       r.Skill1.Name,
			Scope1:       r.Skill1.Scope.String(),
			Platform1:    string(r.Skill1.Platform),
			Skill2:       r.Skill2.Name,
			Scope2:       r.Skill2.Scope.String(),
			Platform2:    string(r.Skill2.Platform),
			NameScore:    r.NameScore,
			ContentScore: r.ContentScore,
			LinesAdded:   r.LinesAdded,
			LinesRemoved: r.LinesRemoved,
			HunkCount:    len(r.Hunks),
		}
	}
	return outputs
}

func outputCompareJSON(results []*similarity.ComparisonResult) error {
	outputs := toComparisonOutputs(results)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(outputs)
}

func outputCompareYAML(results []*similarity.ComparisonResult) error {
	outputs := toComparisonOutputs(results)
	data, err := yaml.Marshal(outputs)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	fmt.Print(string(data))
	return nil
}
