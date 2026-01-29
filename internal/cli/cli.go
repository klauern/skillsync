// Package cli provides the command-line interface for skillsync.
package cli

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/ui"
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
			&cli.BoolFlag{
				Name:  "no-color",
				Usage: "Disable colored output",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			configureColors(cmd)
			return ctx, configureLogging(cmd)
		},
		Commands: []*cli.Command{
			versionCommand(),
			statsCommand(),
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
			tuiCommand(),
		},
	}
	return app.Run(ctx, args)
}

// configureColors sets up color output based on CLI flags and config.
// Priority order: NO_COLOR env var > --no-color flag > config setting > auto-detect
func configureColors(cmd *cli.Command) {
	// --no-color flag takes precedence over config
	if cmd.Bool("no-color") {
		ui.DisableColors()
		return
	}

	// Load config to get color setting
	cfg, err := config.Load()
	if err != nil {
		// If config fails to load, use auto-detection
		ui.ConfigureColors("auto")
		return
	}

	// Use config's color setting (handles NO_COLOR env var internally)
	ui.ConfigureColors(cfg.Output.Color)
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
