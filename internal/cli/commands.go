// Package cli provides command definitions for skillsync.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/klauern/skillsync/internal/backup"
	"github.com/klauern/skillsync/internal/model"
	parsercli "github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/cursor"
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

			// Parse and validate source skills before sync (unless skipped)
			var sourceSkills []model.Skill
			if !skipValidation {
				fmt.Println("Validating source skills...")

				sourceSkills, err = parsePlatformSkills(sourcePlatform)
				if err != nil {
					return fmt.Errorf("failed to parse source skills: %w", err)
				}

				// Validate skill formats
				formatResult, err := validation.ValidateSkillsFormat(sourceSkills, sourcePlatform)
				if err != nil {
					return fmt.Errorf("validation error: %w", err)
				}

				// Show warnings
				for _, warning := range formatResult.Warnings {
					fmt.Printf("  Warning: %s\n", warning)
				}

				// Check for validation errors
				if formatResult.HasErrors() {
					fmt.Println("\nValidation failed - the following issues were found:")
					for i, e := range formatResult.Errors {
						fmt.Printf("  %d. %s\n", i+1, formatValidationError(e, sourceSkills))
					}
					return errors.New("skill validation failed - fix the issues above and try again")
				}

				if len(sourceSkills) == 0 {
					fmt.Println("  No skills found in source directory")
				} else {
					fmt.Printf("  Found %d valid skill(s)\n", len(sourceSkills))
				}

				// Validate paths and permissions
				if err := validateSyncPaths(sourcePlatform, targetPlatform); err != nil {
					return err
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

// parsePlatformSkills parses skills from the given platform
func parsePlatformSkills(platform model.Platform) ([]model.Skill, error) {
	var parserInstance parsercli.Parser

	switch platform {
	case model.ClaudeCode:
		parserInstance = claude.New("")
	case model.Cursor:
		parserInstance = cursor.New("")
	case model.Codex:
		return nil, fmt.Errorf("codex platform parsing not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	skills, err := parserInstance.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse skills from %s: %w", platform, err)
	}

	return skills, nil
}

// formatValidationError formats a validation error for display with context
func formatValidationError(err error, skills []model.Skill) string {
	var vErr *validation.Error
	if errors.As(err, &vErr) {
		msg := vErr.Message
		// Add helpful suggestions for common errors
		switch {
		case vErr.Field == "skills[0].name" || msg == "skill name cannot be empty":
			msg += " (ensure each skill file has a name in frontmatter or a valid filename)"
		case strings.Contains(msg, "duplicate skill name"):
			msg += " (rename one of the conflicting skills)"
		case strings.Contains(msg, "cannot access skill file"):
			msg += " (check file path and permissions)"
		}
		return fmt.Sprintf("%s: %s", vErr.Field, msg)
	}

	// Handle Errors collection
	var vErrors validation.Errors
	if errors.As(err, &vErrors) {
		var msgs []string
		for _, e := range vErrors {
			msgs = append(msgs, formatValidationError(e, skills))
		}
		return strings.Join(msgs, "; ")
	}

	return err.Error()
}

// validateSyncPaths validates source and target paths before sync
func validateSyncPaths(sourcePlatform, targetPlatform model.Platform) error {
	// Validate source path exists
	sourcePath, err := validation.GetPlatformPath(sourcePlatform)
	if err != nil {
		return fmt.Errorf("source path error: %w", err)
	}

	if err := validation.ValidatePath(sourcePath, sourcePlatform); err != nil {
		return fmt.Errorf("source validation failed: %w", err)
	}

	// Validate target path (or parent if it doesn't exist)
	targetPath, err := validation.GetPlatformPath(targetPlatform)
	if err != nil {
		return fmt.Errorf("target path error: %w", err)
	}

	if err := validation.ValidatePath(targetPath, targetPlatform); err != nil {
		var vErr *validation.Error
		if errors.As(err, &vErr) && vErr.Message == "path does not exist" {
			// Target doesn't exist - validate parent directory is writable
			parentDir := targetPath[:len(targetPath)-1]
			if err := validation.ValidatePath(parentDir, targetPlatform); err != nil {
				return fmt.Errorf("target parent directory validation failed: %w", err)
			}
			// Check write permission on parent
			if err := checkWritePermission(parentDir); err != nil {
				return fmt.Errorf("target directory not writable: %w", err)
			}
		} else {
			return fmt.Errorf("target validation failed: %w", err)
		}
	}

	return nil
}

// checkWritePermission verifies a directory is writable
func checkWritePermission(path string) error {
	// If path doesn't exist, check parent
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = "." // fallback to current directory
	}

	testFile := path + "/.skillsync-write-test"
	// #nosec G304 - testFile is constructed from validated path and is not user input
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("cannot write to directory: %w", err)
	}
	_ = f.Close()
	_ = os.Remove(testFile)
	return nil
}
