package cli

import (
	"context"

	"github.com/urfave/cli/v3"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func Run(ctx context.Context, args []string) error {
	app := &cli.Command{
		Name:    "skillsync",
		Usage:   "Synchronize agent skills across AI coding platforms",
		Version: Version,
		Commands: []*cli.Command{
			versionCommand(),
			configCommand(),
			discoveryCommand(),
		},
	}
	return app.Run(ctx, args)
}
