// Package cli provides command definitions for skillsync.
package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/klauern/skillsync/internal/backup"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/sync"
	"github.com/klauern/skillsync/internal/validation"
)

func configCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Display current configuration",
		Action: func(_ context.Context, _ *cli.Command) error {
			fmt.Println("Configuration paths:")
			fmt.Println("  Claude Code: ~/.claude/skills/")
			fmt.Println("  Cursor: .cursor/rules/")
			fmt.Println("  Codex: .codex/")
			return nil
		},
	}
}

func discoveryCommand() *cli.Command {
	return &cli.Command{
		Name:  "discovery",
		Usage: "Discover skills on the system",
		Action: func(_ context.Context, _ *cli.Command) error {
			fmt.Println("Discovery command not yet implemented")
			return nil
		},
	}
}

func syncCommand() *cli.Command {
	return &cli.Command{
		Name:      "sync",
		Usage:     "Synchronize skills across platforms",
		UsageText: "skillsync sync [options] <source> <target>",
		Description: `Synchronize skills between AI coding platforms.

   Supported platforms: claudecode, cursor, codex

   Examples:
     skillsync sync claudecode cursor
     skillsync sync --dry-run cursor codex`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   "Preview changes without modifying files",
			},
			&cli.BoolFlag{
				Name:  "skip-backup",
				Usage: "Skip automatic backup before sync",
			},
			&cli.BoolFlag{
				Name:  "skip-validation",
				Usage: "Skip validation checks (not recommended)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			// Parse arguments
			args := cmd.Args()
			if args.Len() != 2 {
				return errors.New("sync requires exactly 2 arguments: <source> <target>")
			}

			// Parse platforms
			sourcePlatform, err := model.ParsePlatform(args.Get(0))
			if err != nil {
				return fmt.Errorf("invalid source platform: %w", err)
			}

			targetPlatform, err := model.ParsePlatform(args.Get(1))
			if err != nil {
				return fmt.Errorf("invalid target platform: %w", err)
			}

			// Get flags
			dryRun := cmd.Bool("dry-run")
			skipBackup := cmd.Bool("skip-backup")
			skipValidation := cmd.Bool("skip-validation")

			// Validate before sync (unless skipped)
			if !skipValidation {
				fmt.Println("Validating sync configuration...")

				// Get platform paths for validation
				sourcePath, err := validation.GetPlatformPath(sourcePlatform)
				if err != nil {
					return fmt.Errorf("failed to get source platform path: %w", err)
				}
				targetPath, err := validation.GetPlatformPath(targetPlatform)
				if err != nil {
					return fmt.Errorf("failed to get target platform path: %w", err)
				}

				// Validate source and target paths exist
				if err := validation.ValidatePath(sourcePath, sourcePlatform); err != nil {
					return fmt.Errorf("source validation failed: %w", err)
				}

				// For target, validate parent directory if path doesn't exist yet
				if err := validation.ValidatePath(targetPath, targetPlatform); err != nil {
					var vErr *validation.Error
					if !errors.As(err, &vErr) || vErr.Message != "path does not exist" {
						return fmt.Errorf("target validation failed: %w", err)
					}
					// Path doesn't exist - validate parent directory
					parentDir := targetPath[:len(targetPath)-1] // rough trim for now
					if err := validation.ValidatePath(parentDir, targetPlatform); err != nil {
						return fmt.Errorf("target parent directory validation failed: %w", err)
					}
				}

				fmt.Println("Validation passed")
			}

			// Create backup before sync (unless skipped or dry-run)
			if !dryRun && !skipBackup {
				fmt.Println("Creating backup before sync...")

				// Run automatic cleanup to maintain retention policy
				cleanupOpts := backup.DefaultCleanupOptions()
				cleanupOpts.Platform = string(targetPlatform)

				deleted, err := backup.CleanupBackups(cleanupOpts)
				if err != nil {
					fmt.Printf("Warning: backup cleanup failed: %v\n", err)
				} else if len(deleted) > 0 {
					fmt.Printf("Cleaned up %d old backup(s)\n", len(deleted))
				}

				// Note: Actual backup creation will be integrated when sync implementation
				// is added. For now, we prepare the backup infrastructure.
				fmt.Println("Backup infrastructure ready")
			}

			// Create sync options
			opts := sync.Options{
				DryRun: dryRun,
			}

			// TODO: Create actual syncer implementation
			// For now, just show what would happen
			if dryRun {
				fmt.Printf("DRY RUN: Would sync from %s to %s\n", sourcePlatform, targetPlatform)
				fmt.Println("\nProposed changes:")
				fmt.Println("  (Sync implementation not yet available)")
				return nil
			}

			fmt.Printf("Syncing from %s to %s...\n", sourcePlatform, targetPlatform)
			fmt.Printf("Options: %+v\n", opts)
			fmt.Println("(Sync implementation not yet available)")
			return nil
		},
	}
}
