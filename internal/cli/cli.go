// Package cli provides the command-line interface for skillsync.
package cli

import (
	"context"

	"github.com/urfave/cli/v3"
)

var (
	// Version is the current version of the application.
	Version = "dev"
	// Commit is the git commit hash.
	Commit = "unknown"
	// BuildDate is the date and time of the build.
	BuildDate = "unknown"
)

// Run executes the CLI application with the given context and arguments.
func Run(ctx context.Context, args []string) error {
	app := &cli.Command{
		Name:    "skillsync",
		Usage:   "Synchronize agent skills across AI coding platforms",
		Version: Version,
		Commands: []*cli.Command{
			versionCommand(),
			configCommand(),
			syncCommand(),
			discoveryCommand(),
			compareCommand(),
			exportCommand(),
			backupCommand(),
			promoteCommand(),
			demoteCommand(),
			scopeCommand(),
		},
	}
	return app.Run(ctx, args)
}
