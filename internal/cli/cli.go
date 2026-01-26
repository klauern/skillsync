// Package cli provides the command-line interface for skillsync.
package cli

import (
	"context"
	"log/slog"

	"github.com/klauern/skillsync/internal/logging"
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
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Enable verbose output (info level logging)",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Enable debug output (debug level logging, implies verbose)",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return ctx, configureLogging(cmd)
		},
		Commands: []*cli.Command{
			versionCommand(),
			configCommand(),
			syncCommand(),
			discoveryCommand(),
			compareCommand(),
			dedupeCommand(),
			exportCommand(),
			backupCommand(),
			promoteCommand(),
			demoteCommand(),
			scopeCommand(),
		},
	}
	return app.Run(ctx, args)
}

// configureLogging sets up the logging level based on CLI flags.
func configureLogging(cmd *cli.Command) error {
	opts := logging.DefaultOptions()

	if cmd.Bool("debug") {
		opts.Level = slog.LevelDebug
		opts.AddSource = true
	} else if cmd.Bool("verbose") {
		opts.Level = slog.LevelInfo
	}

	logger := logging.New(opts)
	logging.SetDefault(logger)

	logging.Debug("logging configured", slog.String("level", opts.Level.String()))

	return nil
}
