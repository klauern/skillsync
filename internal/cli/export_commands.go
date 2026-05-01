package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/klauern/skillsync/internal/export"
	"github.com/klauern/skillsync/internal/model"
)

func exportCommand() *cli.Command {
	return &cli.Command{
		Name:      "export",
		Usage:     "Export skills to different formats",
		UsageText: "skillsync export [options]",
		Description: `Export skills to JSON, YAML, or Markdown formats.

   Supported formats: json (default), yaml, markdown

   Examples:
     skillsync export
     skillsync export --format yaml
     skillsync export --platform claude-code --format markdown
     skillsync export --output skills.json`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Filter by platform (claude-code, cursor, codex, pi.dev)",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "json",
				Usage:   "Output format: json, yaml, markdown",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file path (default: stdout)",
			},
			&cli.BoolFlag{
				Name:  "no-metadata",
				Usage: "Exclude metadata fields from export",
			},
			&cli.BoolFlag{
				Name:  "compact",
				Usage: "Compact output (no pretty-printing)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return runExport(cmd)
		},
	}
}

// runExport executes the export command.
func runExport(cmd *cli.Command) error {
	// Parse format
	formatStr := cmd.String("format")
	format, err := export.ParseFormat(formatStr)
	if err != nil {
		return err
	}

	// Parse platform filter
	var platform model.Platform
	platformStr := cmd.String("platform")
	if platformStr != "" {
		p, err := model.ParsePlatform(platformStr)
		if err != nil {
			return fmt.Errorf("invalid platform: %w", err)
		}
		platform = p
	}

	// Build export options
	opts := export.Options{
		Format:          format,
		Pretty:          !cmd.Bool("compact"),
		IncludeMetadata: !cmd.Bool("no-metadata"),
		Platform:        platform,
	}

	// Discover skills
	skills, err := discoverSkillsForExport(platform)
	if err != nil {
		return fmt.Errorf("failed to discover skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Fprintln(os.Stderr, "No skills found to export.")
		return nil
	}

	// Create exporter
	exporter := export.New(opts)

	// Determine output destination
	outputPath := cmd.String("output")
	if outputPath != "" {
		// Write to file
		// #nosec G304 - outputPath is provided by user
		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}

		if err := exporter.Export(skills, file); err != nil {
			_ = file.Close()
			return fmt.Errorf("export failed: %w", err)
		}

		if err := file.Close(); err != nil {
			return fmt.Errorf("failed to close output file: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Exported %d skill(s) to %s\n", len(skills), outputPath)
	} else {
		// Write to stdout
		if err := exporter.Export(skills, os.Stdout); err != nil {
			return fmt.Errorf("export failed: %w", err)
		}
	}

	return nil
}

// discoverSkillsForExport discovers skills optionally filtered by platform.
func discoverSkillsForExport(platform model.Platform) ([]model.Skill, error) {
	var platforms []model.Platform
	if platform != "" {
		platforms = []model.Platform{platform}
	} else {
		platforms = model.AllPlatforms()
	}

	var allSkills []model.Skill
	for _, p := range platforms {
		skills, err := parsePlatformSkills(p)
		if err != nil {
			// Log warning but continue with other platforms
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", p, err)
			continue
		}

		allSkills = append(allSkills, skills...)
	}

	return allSkills, nil
}
