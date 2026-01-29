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

type contextKey string

const configKey contextKey = "config"

// getConfig retrieves the config from context, falling back to defaults if not present
func getConfig(ctx context.Context) *config.Config {
	if cfg, ok := ctx.Value(configKey).(*config.Config); ok {
		return cfg
	}
	// Fallback to loading from file/defaults
	cfg, err := config.Load()
	if err != nil {
		return config.Default()
	}
	return cfg
}

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
			&cli.StringFlag{
				Name:  "cache-dir",
				Usage: "Override cache directory location (default: ~/.skillsync/cache)",
			},
			&cli.DurationFlag{
				Name:  "cache-ttl",
				Usage: "Override cache time-to-live (e.g., 1h, 30m)",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			// Load config from file and environment
			cfg, err := config.Load()
			if err != nil {
				// If config fails, use defaults
				cfg = config.Default()
			}

			// Apply CLI flag overrides (highest precedence)
			if cacheDir := cmd.String("cache-dir"); cacheDir != "" {
				cfg.Cache.Location = cacheDir
			}
			if cacheTTL := cmd.Duration("cache-ttl"); cacheTTL > 0 {
				cfg.Cache.TTL = cacheTTL
			}

			// Store config in context for commands to access
			ctx = context.WithValue(ctx, configKey, cfg)

			configureColors(cmd)
			return ctx, configureLogging(cmd)
		},
		Commands: []*cli.Command{
			versionCommand(),
			configCommand(),
			syncCommand(),
			discoveryCommand(),
			platformsCommand(),
			pluginsCommand(),
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
