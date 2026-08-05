// Package cli provides command definitions for skillsync.
package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/klauern/skillsync/internal/ui"
	"github.com/klauern/skillsync/internal/validation"
)

func portabilityCheckCommand() *cli.Command {
	return &cli.Command{
		Name:    "portability-check",
		Aliases: []string{"pc"},
		Usage:   "Check portability snapshot freshness against docs",
		UsageText: `skillsync portability-check [options]
   skillsync portability-check
   skillsync portability-check --format json`,
		Description: `Validate that the portability snapshot stays aligned with narrative docs.

   This command compares docs/platforms/portability-snapshot.yaml against the
   platform reference docs and reports drift. Run it after editing any platform
   documentation to ensure the snapshot is still accurate.

   Exit codes:
     0  snapshot is fresh (no drift detected)
     1  snapshot has drift (mismatches found)
     2  internal error (could not read files)

   Examples:
     skillsync portability-check           # Human-readable table output
     skillsync portability-check --format json  # Machine-readable JSON`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "table",
				Usage:   "Output format: table, json",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return runPortabilityCheck(cmd)
		},
	}
}

func runPortabilityCheck(cmd *cli.Command) error {
	format := strings.ToLower(cmd.String("format"))

	root, err := validation.FindRepoRoot()
	if err != nil {
		ui.PrintError("Cannot find repository root: %v", err)
		return cli.Exit("", 2)
	}

	result, err := validation.ValidatePortabilitySnapshot(root)
	if err != nil {
		ui.PrintError("Failed to validate portability snapshot: %v", err)
		return cli.Exit("", 2)
	}

	switch format {
	case "json":
		return outputPortabilityCheckJSON(result)
	case "table":
		return outputPortabilityCheckTable(result)
	default:
		ui.PrintError("Unsupported format: %s", format)
		return cli.Exit("", 2)
	}
}

func outputPortabilityCheckTable(result *validation.Result) error {
	if result.Valid && len(result.Warnings) == 0 {
		ui.PrintSuccess("Portability snapshot is fresh. No drift detected.")
		return nil
	}

	if result.Valid {
		ui.PrintWarning("Portability snapshot passed with warnings.")
	} else {
		ui.PrintError("Portability snapshot has drift. Revalidation needed.")
	}

	fmt.Println()

	if len(result.Errors) > 0 {
		fmt.Println(ui.Header("Mismatches:"))
		for i, err := range result.Errors {
			fmt.Printf("  %d. %s\n", i+1, err)
		}
		fmt.Println()
	}

	if len(result.Warnings) > 0 {
		fmt.Println(ui.Header("Warnings:"))
		for i, w := range result.Warnings {
			fmt.Printf("  %d. %s\n", i+1, w)
		}
		fmt.Println()
	}

	fmt.Println(ui.Dim("To revalidate: update docs/platforms/portability-snapshot.yaml, then run this check again."))

	return cli.Exit("", 1)
}

func outputPortabilityCheckJSON(result *validation.Result) error {
	type jsonResult struct {
		Fresh    bool     `json:"fresh"`
		Errors   []string `json:"errors,omitempty"`
		Warnings []string `json:"warnings,omitempty"`
	}

	out := jsonResult{
		Fresh:    result.Valid && len(result.Warnings) == 0,
		Warnings: result.Warnings,
	}
	for _, e := range result.Errors {
		out.Errors = append(out.Errors, e.Error())
	}

	if err := outputAnyJSON(out); err != nil {
		return err
	}

	if !out.Fresh {
		return cli.Exit("", 1)
	}
	return nil
}
