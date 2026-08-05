// Package cli provides the detection command for skillsync.
package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/klauern/skillsync/internal/ui"
	"github.com/klauern/skillsync/internal/util"
)

func detectCommand() *cli.Command {
	return &cli.Command{
		Name:  "detect",
		Usage: "Detect installed platforms",
		UsageText: `skillsync detect [options]
   skillsync detect --format json`,
		Description: `Inspect the default platform locations and report which platforms appear to be installed.

   The command checks the configured default search paths for each supported
   platform and reports whether each one is present, partial, or missing.

   Examples:
     skillsync detect                 # Human-readable table output
     skillsync detect --format json   # Machine-readable JSON`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "table",
				Usage:   "Output format: table, json",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return runDetectCommand(cmd)
		},
	}
}

func runDetectCommand(cmd *cli.Command) error {
	format := strings.ToLower(cmd.String("format"))
	result := util.DetectInstalledPlatforms()

	switch format {
	case "json":
		return outputDetectJSON(result)
	case "table":
		return outputDetectTable(result)
	default:
		ui.PrintError("Unsupported format: %s", format)
		return cli.Exit("", 2)
	}
}

func outputDetectTable(result util.PlatformDetectionResult) error {
	fmt.Printf("Detected %d of %d platform(s)\n\n", len(result.Detected), len(result.Details))

	for _, detail := range result.Details {
		fmt.Printf("  %s %s: %s\n", detectStatusSymbol(detail.Status), detail.Platform, detail.Reason)
	}

	return nil
}

func outputDetectJSON(result util.PlatformDetectionResult) error {
	return outputAnyJSON(result)
}

func detectStatusSymbol(status util.PlatformDetectionStatus) string {
	switch status {
	case util.PlatformDetectionPresent:
		return ui.SymbolSuccess
	case util.PlatformDetectionPartial:
		return ui.SymbolWarning
	default:
		return ui.SymbolSkipped
	}
}
