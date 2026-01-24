package cli

import (
	"context"
	"fmt"
	"runtime"

	"github.com/urfave/cli/v3"
)

func versionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Display version and build information",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Printf("skillsync version %s\n", Version)
			fmt.Printf("  commit: %s\n", Commit)
			fmt.Printf("  built: %s\n", BuildDate)
			fmt.Printf("  go: %s\n", runtime.Version())
			return nil
		},
	}
}
