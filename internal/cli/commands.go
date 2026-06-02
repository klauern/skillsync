// Package cli provides command definitions for skillsync.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/backup"
	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/export"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/similarity"
	"github.com/klauern/skillsync/internal/sync"
	"github.com/klauern/skillsync/internal/ui"
	"github.com/klauern/skillsync/internal/ui/tui"
	"github.com/klauern/skillsync/internal/util"
)

var (
	runSyncList   = tui.RunSyncList
	runSyncDiff   = tui.RunSyncDiff
	runDeleteList = tui.RunDeleteList
)

func configCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Manage skillsync configuration",
		Description: `Manage skillsync configuration settings.

   Configuration is loaded from: ~/.skillsync/config.yaml
   Environment variables can override any setting with SKILLSYNC_* prefix.

   Examples:
     skillsync config show           # Show current configuration
     skillsync config init           # Create default config file
     skillsync config path           # Show config file path
     skillsync config edit           # Edit config file (opens in $EDITOR)`,
		Commands: []*cli.Command{
			configShowCommand(),
			configInitCommand(),
			configPathCommand(),
			configEditCommand(),
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			// Default action: show configuration
			return showConfig()
		},
	}
}

func configShowCommand() *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "Display current configuration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "yaml",
				Usage:   "Output format: yaml, json",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			format := cmd.String("format")
			return showConfigWithFormat(format)
		},
	}
}

func configInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Create default configuration file",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Overwrite existing config file",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			force := cmd.Bool("force")
			return initConfig(force)
		},
	}
}

func configPathCommand() *cli.Command {
	return &cli.Command{
		Name:  "path",
		Usage: "Display configuration file paths",
		Action: func(_ context.Context, _ *cli.Command) error {
			return showConfigPaths()
		},
	}
}

func configEditCommand() *cli.Command {
	return &cli.Command{
		Name:  "edit",
		Usage: "Edit configuration file in $EDITOR",
		Action: func(_ context.Context, _ *cli.Command) error {
			return editConfig()
		},
	}
}

// showConfig displays the current configuration.
func showConfig() error {
	return showConfigWithFormat("yaml")
}

// showConfigWithFormat displays the configuration in the specified format.
func showConfigWithFormat(format string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(cfg)
	case "yaml":
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		fmt.Println("# skillsync configuration")
		if config.Exists() {
			fmt.Printf("# Loaded from: %s\n", config.FilePath())
		} else {
			fmt.Println("# Using default configuration (no config file found)")
		}
		fmt.Println()
		fmt.Print(string(data))
		return nil
	default:
		return fmt.Errorf("unsupported format: %s (use yaml or json)", format)
	}
}

// initConfig creates a default configuration file.
func initConfig(force bool) error {
	configPath := config.FilePath()

	if config.Exists() && !force {
		return fmt.Errorf("config file already exists at %s (use --force to overwrite)", configPath)
	}

	cfg := config.Default()
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Created config file: %s\n", configPath)
	return nil
}

// showConfigPaths displays all configuration-related paths.
func showConfigPaths() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Configuration paths:")
	fmt.Printf("  Config file:     %s", config.FilePath())
	if config.Exists() {
		fmt.Println(" (exists)")
	} else {
		fmt.Println(" (not found)")
	}
	fmt.Printf("  Config dir:      %s\n", util.SkillsyncConfigPath())

	fmt.Println("\nPlatform paths:")
	fmt.Printf("  Claude Code:     %v\n", cfg.Platforms.ClaudeCode.SkillsPaths)
	fmt.Printf("  Cursor:          %v\n", cfg.Platforms.Cursor.SkillsPaths)
	fmt.Printf("  Codex:           %v\n", cfg.Platforms.Codex.SkillsPaths)
	fmt.Printf("  Pi Agent:        %v\n", cfg.Platforms.PiAgent.SkillsPaths)
	fmt.Printf("  Pi.dev:          %v\n", cfg.Platforms.PiDev.SkillsPaths)

	fmt.Println("\nData paths:")
	fmt.Printf("  Backups:         %s\n", util.SkillsyncBackupsPath())
	fmt.Printf("  Cache:           %s\n", filepath.Join(util.SkillsyncConfigPath(), "cache"))
	fmt.Printf("  Plugins:         %s\n", util.SkillsyncPluginsPath())
	fmt.Printf("  Metadata:        %s\n", util.SkillsyncMetadataPath())

	return nil
}

// editConfig opens the config file in the user's editor.
func editConfig() error {
	configPath := config.FilePath()

	// Ensure config file exists
	if !config.Exists() {
		fmt.Println("No config file found. Creating default configuration...")
		if err := initConfig(false); err != nil {
			return fmt.Errorf("initialize config: %w", err)
		}
	}

	// Find editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		return fmt.Errorf("no editor found - set $EDITOR or $VISUAL environment variable")
	}

	fmt.Printf("Opening %s in %s...\n", configPath, editor)

	// Editor may include arguments (e.g. EDITOR="code --wait")
	editorArgs := strings.Fields(editor)
	// #nosec G204 G702 - editor binary is intentionally user-controlled via $EDITOR/$VISUAL
	cmd := exec.Command(editorArgs[0], append(editorArgs[1:], configPath)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func exportCommand() *cli.Command {
	return &cli.Command{
		Name:      "export",
		Usage:     "Export skills to different formats",
		UsageText: "skillsync export [options]",
		Description: `Export skills to JSON, YAML, or Markdown formats.

   Supported formats: json (default), yaml, markdown

   Examples:
     skillsync export
     skillsync export --format yaml
     skillsync export --platform claude-code --format markdown
     skillsync export --output skills.json`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Filter by platform (" + model.AllPlatformNames() + ")",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "json",
				Usage:   "Output format: json, yaml, markdown",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file path (default: stdout)",
			},
			&cli.BoolFlag{
				Name:  "no-metadata",
				Usage: "Exclude metadata fields from export",
			},
			&cli.BoolFlag{
				Name:  "compact",
				Usage: "Compact output (no pretty-printing)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return runExport(cmd)
		},
	}
}

// runExport executes the export command.
func runExport(cmd *cli.Command) error {
	// Parse format
	formatStr := cmd.String("format")
	format, err := export.ParseFormat(formatStr)
	if err != nil {
		return fmt.Errorf("parse export format %q: %w", formatStr, err)
	}

	// Parse platform filter
	var platform model.Platform
	platformStr := cmd.String("platform")
	if platformStr != "" {
		p, err := model.ParsePlatform(platformStr)
		if err != nil {
			return fmt.Errorf("invalid platform: %w", err)
		}
		platform = p
	}

	// Build export options
	opts := export.Options{
		Format:          format,
		Pretty:          !cmd.Bool("compact"),
		IncludeMetadata: !cmd.Bool("no-metadata"),
		Platform:        platform,
	}

	// Discover skills
	skills, err := discoverSkillsForExport(platform)
	if err != nil {
		return fmt.Errorf("failed to discover skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Fprintln(os.Stderr, "No skills found to export.")
		return nil
	}

	// Create exporter
	exporter := export.New(opts)

	// Determine output destination
	outputPath := cmd.String("output")
	if outputPath != "" {
		// Write to file
		// #nosec G304 - outputPath is provided by user
		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}

		if err := exporter.Export(skills, file); err != nil {
			_ = file.Close()
			return fmt.Errorf("export failed: %w", err)
		}

		if err := file.Close(); err != nil {
			return fmt.Errorf("failed to close output file: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Exported %d skill(s) to %s\n", len(skills), outputPath)
	} else {
		// Write to stdout
		if err := exporter.Export(skills, os.Stdout); err != nil {
			return fmt.Errorf("export failed: %w", err)
		}
	}

	return nil
}

// discoverSkillsForExport discovers skills optionally filtered by platform.
func discoverSkillsForExport(platform model.Platform) ([]model.Skill, error) {
	var platforms []model.Platform
	if platform != "" {
		platforms = []model.Platform{platform}
	} else {
		platforms = model.AllPlatforms()
	}

	var allSkills []model.Skill
	for _, p := range platforms {
		skills, err := parsePlatformSkills(p)
		if err != nil {
			// Log warning but continue with other platforms
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", p, err)
			continue
		}

		allSkills = append(allSkills, skills...)
	}

	return allSkills, nil
}

func backupCommand() *cli.Command {
	return &cli.Command{
		Name:  "backup",
		Usage: "Manage skillsync backups",
		Description: `Manage backups of skill files.

   Backups are automatically created before sync operations.
   Use these commands to view, verify, and manage backups.

   Examples:
     skillsync backup list                    # List all backups
     skillsync backup create --platform cursor # Create backups for Cursor skills
     skillsync backup list --platform claude-code
     skillsync backup list --format json
     skillsync backup restore <backup-id>     # Restore a backup`,
		Commands: []*cli.Command{
			backupCreateCommand(),
			backupListCommand(),
			backupRestoreCommand(),
			backupDeleteCommand(),
			backupVerifyCommand(),
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			// Default action: list backups
			return listBackups("", "table", 0)
		},
	}
}

func backupCreateCommand() *cli.Command {
	return &cli.Command{
		Name:    "create",
		Aliases: []string{"new"},
		Usage:   "Create backups of skill files",
		UsageText: `skillsync backup create [options]
   skillsync backup create --platform cursor
   skillsync backup create --platform claude-code --scope repo
   skillsync backup create --platform all`,
		Description: `Create backups for skills across platforms.

   By default, backs up all platforms. Use --platform to limit results.
   Use --scope to filter which skill scopes are included.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Platform to back up (" + model.AllPlatformNames() + ", all)",
			},
			&cli.StringFlag{
				Name:    "scope",
				Aliases: []string{"s"},
				Usage:   "Filter by scope (repo, user, admin, system, builtin, plugin, all). Comma-separated for multiple.",
			},
			&cli.BoolFlag{
				Name:  "include-plugins",
				Usage: "Include skills from installed Claude Code plugins",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			platformStr := strings.TrimSpace(cmd.String("platform"))
			scopeStr := strings.TrimSpace(cmd.String("scope"))
			includePlugins := cmd.Bool("include-plugins")

			scopeFilter, err := parseScopeFilter(scopeStr)
			if err != nil {
				return fmt.Errorf("invalid scope filter %q: %w", scopeStr, err)
			}

			var platforms []model.Platform
			if platformStr == "" || platformStr == "all" {
				platforms = model.AllPlatforms()
			} else {
				platform, err := model.ParsePlatform(platformStr)
				if err != nil {
					return fmt.Errorf("invalid platform %q: %w", platformStr, err)
				}
				platforms = []model.Platform{platform}
			}

			totalCreated := 0
			for _, platform := range platforms {
				skills, err := parsePlatformSkillsWithScope(platform, scopeFilter, includePlugins)
				if err != nil {
					return fmt.Errorf("failed to parse %s skills: %w", platform, err)
				}
				if len(skills) == 0 {
					continue
				}

				prepareBackup(platform)
				created, err := createBackupsForSkills(platform, skills, "manual backup", []string{"manual"})
				if err != nil {
					return fmt.Errorf("create backups for %s: %w", platform, err)
				}
				totalCreated += created
			}

			if totalCreated == 0 {
				fmt.Println("No skills found to back up.")
				return nil
			}

			fmt.Printf("\n✓ Created %d backup(s)\n", totalCreated)
			return nil
		},
	}
}

func backupListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List existing backups with metadata",
		UsageText: `skillsync backup list [options]
   skillsync backup list --platform claude-code
   skillsync backup list --format json
   skillsync backup list --limit 10`,
		Description: `List all backups with their metadata including timestamp, size, and platform.

   Output includes: ID, Platform, Source File, Created At, Size

   Formats: table (default), json, yaml
   For interactive backup management, use: skillsync tui`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Filter by platform (" + model.AllPlatformNames() + ")",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "table",
				Usage:   "Output format: table, json, yaml",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n"},
				Value:   0,
				Usage:   "Limit results to N most recent backups (0 = unlimited)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			platform := cmd.String("platform")
			format := cmd.String("format")
			limit := cmd.Int("limit")
			return listBackups(platform, format, int(limit))
		},
	}
}

func backupRestoreCommand() *cli.Command {
	return &cli.Command{
		Name:  "restore",
		Usage: "Restore a backup to its original or specified location",
		UsageText: `skillsync backup restore <backup-id> [options]
   skillsync backup restore 20240125-120000-abc12345
   skillsync backup restore 20240125-120000-abc12345 --target /path/to/restore
   skillsync backup restore 20240125-120000-abc12345 --force`,
		Description: `Restore a skill file from a backup.

   By default, restores to the original source path. Use --target to specify
   a different location.

   The restore operation verifies backup integrity using SHA256 hash before
   restoring. Use --force to skip the confirmation prompt.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "target",
				Aliases: []string{"t"},
				Usage:   "Target path for restoration (defaults to original source path)",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation prompt before overwriting",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			if args.Len() < 1 {
				return errors.New("backup ID is required")
			}

			backupID := args.Get(0)
			targetPath := cmd.String("target")
			force := cmd.Bool("force")

			return restoreBackup(backupID, targetPath, force)
		},
	}
}

// listBackups retrieves and displays backups based on filters
func listBackups(platform, format string, limit int) error {
	backups, err := backup.ListBackups(platform)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	// Apply limit if specified
	if limit > 0 && len(backups) > limit {
		backups = backups[:limit]
	}

	return outputBackups(backups, format)
}

// listBackupsInteractive runs the interactive TUI for backup management
func listBackupsInteractive(platform string) error {
	backups, err := backup.ListBackups(platform)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	result, err := tui.RunBackupList(backups)
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Handle the selected action
	switch result.Action {
	case tui.ActionRestore:
		fmt.Printf("\nRestoring backup: %s\n", result.BackupID)
		return restoreBackup(result.BackupID, "", false)
	case tui.ActionDelete:
		fmt.Printf("\nDeleting backup: %s\n", result.BackupID)
		return deleteBackupsByID([]string{result.BackupID}, true) // force=true since already confirmed in TUI
	case tui.ActionVerify:
		fmt.Printf("\nVerifying backup: %s\n", result.BackupID)
		return verifyBackupsByID([]string{result.BackupID})
	case tui.ActionNone:
		// User quit without action
		return nil
	}

	return nil
}

// restoreBackup restores a backup to the original or specified target path
func restoreBackup(backupID, targetPath string, force bool) error {
	// Load index to get backup metadata
	index, err := backup.LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load backup index: %w", err)
	}

	// Find the backup
	metadata, exists := index.Backups[backupID]
	if !exists {
		return fmt.Errorf("backup %q not found", backupID)
	}

	// Use original source path if no target specified
	if targetPath == "" {
		targetPath = metadata.SourcePath
	}

	// Check if target file exists
	targetExists := false
	if _, err := os.Stat(targetPath); err == nil {
		targetExists = true
	}

	// Display restore details
	fmt.Println("\nBackup Details:")
	fmt.Printf("  ID:       %s\n", metadata.ID)
	fmt.Printf("  Platform: %s\n", metadata.Platform)
	fmt.Printf("  Size:     %s\n", formatSize(metadata.Size))
	fmt.Printf("  Created:  %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Source:   %s\n", metadata.SourcePath)
	fmt.Printf("  Target:   %s\n", targetPath)

	if targetExists {
		fmt.Println("\n⚠️  Target file already exists and will be overwritten.")
	}

	// Confirm unless force flag is set
	if !force {
		message := fmt.Sprintf("Restore backup to %s?", targetPath)
		level := riskLevelInfo
		if targetExists {
			level = riskLevelWarning
		}

		confirmed, err := confirmAction(message, level)
		if err != nil {
			return fmt.Errorf("confirmation error: %w", err)
		}
		if !confirmed {
			fmt.Println("Restore cancelled.")
			return nil
		}
	}

	// Perform the restore
	if err := backup.RestoreBackup(backupID, targetPath); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	fmt.Printf("\n✓ Successfully restored backup to %s\n", targetPath)
	return nil
}

func backupDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete old backups",
		UsageText: `skillsync backup delete [options] [backup-id...]
   skillsync backup delete <backup-id>              # Delete specific backup
   skillsync backup delete --older-than 30d         # Delete backups older than 30 days
   skillsync backup delete --keep-latest 5          # Keep only 5 most recent backups
   skillsync backup delete --platform claude-code --keep-latest 3`,
		Description: `Delete backups by ID, age, or count-based retention.

   By ID: Pass one or more backup IDs as arguments
   By Age: Use --older-than with a duration (e.g., 30d, 2w, 168h)
   By Count: Use --keep-latest N to keep only N most recent backups

   Combine --platform with --older-than or --keep-latest to filter by platform.
   Use --force to skip confirmation prompt.

   Examples of duration formats:
     30d   = 30 days
     2w    = 2 weeks (14 days)
     168h  = 168 hours (7 days)`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "older-than",
				Aliases: []string{"o"},
				Usage:   "Delete backups older than duration (e.g., 30d, 2w, 168h)",
			},
			&cli.IntFlag{
				Name:    "keep-latest",
				Aliases: []string{"k"},
				Value:   0,
				Usage:   "Keep only N most recent backups (0 = disabled)",
			},
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Filter by platform (" + model.AllPlatformNames() + ")",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation prompt",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			olderThan := cmd.String("older-than")
			keepLatest := cmd.Int("keep-latest")
			platform := cmd.String("platform")
			force := cmd.Bool("force")

			// Determine delete mode based on arguments and flags
			if args.Len() > 0 {
				// Delete by specific IDs
				ids := make([]string, args.Len())
				for i := 0; i < args.Len(); i++ {
					ids[i] = args.Get(i)
				}
				return deleteBackupsByID(ids, force)
			}

			if olderThan != "" || keepLatest > 0 {
				// Delete by retention policy
				return deleteBackupsByPolicy(olderThan, int(keepLatest), platform, force)
			}

			return errors.New("either backup IDs or --older-than/--keep-latest flag is required")
		},
	}
}

func backupVerifyCommand() *cli.Command {
	return &cli.Command{
		Name:  "verify",
		Usage: "Verify backup integrity using SHA256 checksums",
		UsageText: `skillsync backup verify [backup-id...]
   skillsync backup verify                           # Verify all backups
   skillsync backup verify 20240125-120000-abc12345  # Verify specific backup
   skillsync backup verify --platform claude-code    # Verify backups for a platform`,
		Description: `Verify backup integrity by comparing file content against stored SHA256 checksums.

   Without arguments, verifies all backups. Pass one or more backup IDs to verify
   specific backups. Use --platform to filter verification to a specific platform.

   The command reports:
     ✓ OK       - Backup file is intact and matches stored checksum
     ✗ CORRUPT  - Backup file has been modified or corrupted
     ✗ MISSING  - Backup file no longer exists on disk

   Exit codes:
     0 - All verified backups are intact
     1 - One or more backups failed verification`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Filter by platform (" + model.AllPlatformNames() + ")",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			platform := cmd.String("platform")

			if args.Len() > 0 {
				// Verify specific backup IDs
				ids := make([]string, args.Len())
				for i := 0; i < args.Len(); i++ {
					ids[i] = args.Get(i)
				}
				return verifyBackupsByID(ids)
			}

			// Verify all backups (optionally filtered by platform)
			return verifyAllBackups(platform)
		},
	}
}

// verifyBackupsByID verifies specific backups by their IDs
func verifyBackupsByID(ids []string) error {
	fmt.Printf("Verifying %d backup(s)...\n\n", len(ids))

	var failed int
	for _, id := range ids {
		if err := backup.VerifyBackup(id); err != nil {
			fmt.Printf("✗ %-28s FAILED: %v\n", id, err)
			failed++
		} else {
			fmt.Printf("✓ %-28s OK\n", id)
		}
	}

	fmt.Println()
	if failed > 0 {
		fmt.Printf("Verification complete: %d OK, %d FAILED\n", len(ids)-failed, failed)
		return fmt.Errorf("%d backup(s) failed verification", failed)
	}

	fmt.Printf("Verification complete: %d OK\n", len(ids))
	return nil
}

// verifyAllBackups verifies all backups, optionally filtered by platform
func verifyAllBackups(platform string) error {
	backups, err := backup.ListBackups(platform)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		fmt.Println("No backups found to verify.")
		return nil
	}

	fmt.Printf("Verifying %d backup(s)...\n\n", len(backups))

	var ok, failed int
	for _, b := range backups {
		if err := backup.VerifyBackup(b.ID); err != nil {
			fmt.Printf("✗ %-28s %-12s FAILED: %v\n", b.ID, b.Platform, err)
			failed++
		} else {
			fmt.Printf("✓ %-28s %-12s OK\n", b.ID, b.Platform)
			ok++
		}
	}

	fmt.Println()
	if failed > 0 {
		fmt.Printf("Verification complete: %d OK, %d FAILED\n", ok, failed)
		return fmt.Errorf("%d backup(s) failed verification", failed)
	}

	fmt.Printf("Verification complete: %d OK\n", ok)
	return nil
}

// deleteBackupsByID deletes specific backups by their IDs
func deleteBackupsByID(ids []string, force bool) error {
	// Load index to verify backups exist
	index, err := backup.LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load backup index: %w", err)
	}

	// Verify all IDs exist
	var backupsToDelete []backup.Metadata
	for _, id := range ids {
		metadata, exists := index.Backups[id]
		if !exists {
			return fmt.Errorf("backup %q not found", id)
		}
		backupsToDelete = append(backupsToDelete, metadata)
	}

	// Display what will be deleted
	fmt.Printf("\nBackups to delete (%d):\n", len(backupsToDelete))
	for _, b := range backupsToDelete {
		fmt.Printf("  - %s (%s, %s)\n", b.ID, b.Platform, formatSize(b.Size))
	}

	// Confirm unless force flag is set
	if !force {
		message := fmt.Sprintf("Delete %d backup(s)?", len(backupsToDelete))
		confirmed, err := confirmAction(message, riskLevelWarning)
		if err != nil {
			return fmt.Errorf("confirmation error: %w", err)
		}
		if !confirmed {
			fmt.Println("Delete cancelled.")
			return nil
		}
	}

	// Delete each backup
	var deleted int
	for _, b := range backupsToDelete {
		if err := backup.DeleteBackup(b.ID); err != nil {
			return fmt.Errorf("failed to delete backup %q: %w", b.ID, err)
		}
		deleted++
	}

	fmt.Printf("\n✓ Deleted %d backup(s)\n", deleted)
	return nil
}

// deleteBackupsByPolicy deletes backups based on age or count retention
func deleteBackupsByPolicy(olderThan string, keepLatest int, platform string, force bool) error {
	// Parse duration from --older-than flag
	var maxAge time.Duration
	if olderThan != "" {
		duration, err := parseDuration(olderThan)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", olderThan, err)
		}
		maxAge = duration
	}

	// Get list of backups to analyze
	backups, err := backup.ListBackups(platform)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	// Determine which backups to delete
	now := time.Now()
	var toDelete []backup.Metadata

	for i, b := range backups {
		shouldDelete := false

		// Check age
		if maxAge > 0 && now.Sub(b.CreatedAt) > maxAge {
			shouldDelete = true
		}

		// Check count limit (backups are already sorted newest first)
		if keepLatest > 0 && i >= keepLatest {
			shouldDelete = true
		}

		if shouldDelete {
			toDelete = append(toDelete, b)
		}
	}

	if len(toDelete) == 0 {
		fmt.Println("No backups match the deletion criteria.")
		return nil
	}

	// Display what will be deleted
	var totalSize int64
	fmt.Printf("\nBackups to delete (%d):\n", len(toDelete))
	for _, b := range toDelete {
		fmt.Printf("  - %s (%s, %s, %s)\n",
			b.ID, b.Platform, formatSize(b.Size), b.CreatedAt.Format("2006-01-02"))
		totalSize += b.Size
	}
	fmt.Printf("\nTotal space to free: %s\n", formatSize(totalSize))

	// Show what will be kept
	keptCount := len(backups) - len(toDelete)
	fmt.Printf("Backups remaining: %d\n", keptCount)

	// Confirm unless force flag is set
	if !force {
		message := fmt.Sprintf("Delete %d backup(s)?", len(toDelete))
		confirmed, err := confirmAction(message, riskLevelWarning)
		if err != nil {
			return fmt.Errorf("confirmation error: %w", err)
		}
		if !confirmed {
			fmt.Println("Delete cancelled.")
			return nil
		}
	}

	// Delete each backup
	var deleted int
	for _, b := range toDelete {
		if err := backup.DeleteBackup(b.ID); err != nil {
			return fmt.Errorf("failed to delete backup %q: %w", b.ID, err)
		}
		deleted++
	}

	fmt.Printf("\n✓ Deleted %d backup(s), freed %s\n", deleted, formatSize(totalSize))
	return nil
}

// parseDuration parses a duration string with support for day and week units
func parseDuration(s string) (time.Duration, error) {
	// Check for custom units (days, weeks)
	if len(s) >= 2 {
		lastChar := s[len(s)-1]
		numPart := s[:len(s)-1]

		switch lastChar {
		case 'd', 'D':
			// Days
			var days int
			if _, err := fmt.Sscanf(numPart, "%d", &days); err != nil {
				return 0, fmt.Errorf("invalid day count: %s", numPart)
			}
			return time.Duration(days) * 24 * time.Hour, nil
		case 'w', 'W':
			// Weeks
			var weeks int
			if _, err := fmt.Sscanf(numPart, "%d", &weeks); err != nil {
				return 0, fmt.Errorf("invalid week count: %s", numPart)
			}
			return time.Duration(weeks) * 7 * 24 * time.Hour, nil
		}
	}

	// Fall back to standard Go duration parsing (hours, minutes, seconds)
	return time.ParseDuration(s)
}

// outputBackups formats and prints backups in the requested format
func outputBackups(backups []backup.Metadata, format string) error {
	switch format {
	case "json":
		return outputBackupsJSON(backups)
	case "yaml":
		return outputBackupsYAML(backups)
	case "table":
		return outputBackupsTable(backups)
	default:
		return fmt.Errorf("unsupported format: %s (use table, json, or yaml)", format)
	}
}

// outputBackupsJSON prints backups as JSON
func outputBackupsJSON(backups []backup.Metadata) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(backups)
}

// outputBackupsYAML prints backups as YAML
func outputBackupsYAML(backups []backup.Metadata) error {
	data, err := yaml.Marshal(backups)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	fmt.Print(string(data))
	return nil
}

// outputBackupsTable prints backups in a table format with colored output
func outputBackupsTable(backups []backup.Metadata) error {
	if len(backups) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	// Sort by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	// Print colored headers
	fmt.Printf("%s %s %s %s %s\n",
		ui.Header(fmt.Sprintf("%-28s", "ID")),
		ui.Header(fmt.Sprintf("%-12s", "PLATFORM")),
		ui.Header(fmt.Sprintf("%-45s", "SOURCE")),
		ui.Header(fmt.Sprintf("%-20s", "CREATED")),
		ui.Header("SIZE"))
	fmt.Printf("%-28s %-12s %-45s %-20s %s\n", "--", "--------", "------", "-------", "----")

	for _, b := range backups {
		// Truncate source path if too long (use left-truncation to preserve the meaningful end)
		source := b.SourcePath
		if len(source) > 45 {
			source = "..." + source[len(source)-42:]
		}

		// Format size
		size := formatSize(b.Size)

		// Format creation time
		created := b.CreatedAt.Format("2006-01-02 15:04:05")

		// Color platform names for visual distinction
		platform := colorPlatform(string(b.Platform), 12)

		fmt.Printf("%-28s %s %-45s %-20s %s\n", b.ID, platform, source, created, size)
	}

	fmt.Printf("\nTotal: %d backup(s)\n", len(backups))
	return nil
}

// formatSize formats a byte size into a human-readable string
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// tuiCommand returns the TUI dashboard command.
func tuiCommand() *cli.Command {
	return &cli.Command{
		Name:    "tui",
		Aliases: []string{"ui"},
		Usage:   "Launch the interactive TUI dashboard",
		Description: `Launch the unified interactive TUI application for skillsync.

   The TUI provides a menu-driven interface to access all skillsync features:
   - Discover skills across all platforms
   - Manage backups (list, restore, delete, verify)
   - Sync operations between platforms
   - Compare and dedupe skills
   - Import/export operations
   - Scope and promote/demote management
   - Configuration settings

   Use arrow keys to navigate, Enter to select, and q to quit.`,
		Action: func(_ context.Context, _ *cli.Command) error {
			return runTUI()
		},
	}
}

// runTUI launches the interactive TUI dashboard and handles view navigation.
func runTUI() error {
	for {
		result, err := tui.RunDashboard()
		if err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		switch result.View {
		case tui.DashboardViewNone:
			// User quit the dashboard
			return nil

		case tui.DashboardViewDiscover:
			if err := runDiscoverTUI(); err != nil {
				return fmt.Errorf("run discover TUI: %w", err)
			}

		case tui.DashboardViewBackups:
			if err := runBackupsTUI(); err != nil {
				return fmt.Errorf("run backups TUI: %w", err)
			}

		case tui.DashboardViewSync:
			if err := runSyncTUI(); err != nil {
				return fmt.Errorf("run sync TUI: %w", err)
			}

		case tui.DashboardViewCompare:
			if err := runCompareTUI(); err != nil {
				return fmt.Errorf("run compare TUI: %w", err)
			}

		case tui.DashboardViewConfig:
			if err := runConfigTUI(); err != nil {
				return fmt.Errorf("run config TUI: %w", err)
			}

		case tui.DashboardViewExport:
			if err := runExportTUI(); err != nil {
				return fmt.Errorf("run export TUI: %w", err)
			}

		case tui.DashboardViewImport:
			if err := runImportTUI(); err != nil {
				return fmt.Errorf("run import TUI: %w", err)
			}

		case tui.DashboardViewScope:
			if err := runScopeTUI(); err != nil {
				return fmt.Errorf("run scope TUI: %w", err)
			}

		case tui.DashboardViewPromote:
			if err := runPromoteDemoteTUI(); err != nil {
				return fmt.Errorf("run promote/demote TUI: %w", err)
			}

		case tui.DashboardViewDelete:
			if err := runDeleteTUI(); err != nil {
				return fmt.Errorf("run delete TUI: %w", err)
			}

		case tui.DashboardViewConflicts:
			if err := runConflictsTUI(); err != nil {
				return fmt.Errorf("run conflicts TUI: %w", err)
			}
		}
	}
}

// runDiscoverTUI runs the discover skills TUI view.
func runDiscoverTUI() error {
	// Discover skills from all platforms
	var allSkills []model.Skill
	for _, p := range model.AllPlatforms() {
		skills, err := parsePlatformSkillsWithScope(p, nil, false)
		if err != nil {
			// Log error but continue with other platforms
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	// Include plugin skills
	pluginSkills, err := discoverPluginSkills("", true)
	if err == nil {
		allSkills = append(allSkills, pluginSkills...)
	}

	if len(allSkills) == 0 {
		ui.Info("No skills found across any platform")
		return nil
	}

	return discoverSkillsInteractive(allSkills)
}

// runBackupsTUI runs the backup management TUI view.
func runBackupsTUI() error {
	return listBackupsInteractive("")
}

// runSyncTUI runs the sync TUI view.
func runSyncTUI() error {
	// Step 1: Pick source/target platform and scope
	pickerResult, err := tui.RunSyncPicker()
	if err != nil {
		return fmt.Errorf("sync picker error: %w", err)
	}

	if pickerResult.Action == tui.SyncPickerActionNone {
		return nil // User cancelled
	}

	sourcePlatform := pickerResult.Source
	sourceScopes := pickerResult.SourceScopes
	targetPlatform := pickerResult.Target
	targetScope := pickerResult.TargetScope

	// Step 2: Parse skills from the source platform
	sourceSkills, err := parsePlatformSkillsWithScope(sourcePlatform, sourceScopes, false)
	if err != nil {
		return fmt.Errorf("failed to parse source skills: %w", err)
	}

	if len(sourceSkills) == 0 {
		sourceScopeLabel := model.FormatSourceScopes(sourceScopes)
		ui.Info(fmt.Sprintf("No skills found in %s:%s", sourcePlatform, sourceScopeLabel))
		return nil
	}

	// Step 3: Run the sync list TUI to select skills
	syncResult, err := tui.RunSyncList(sourceSkills, sourcePlatform, targetPlatform, nil)
	if err != nil {
		return fmt.Errorf("sync list error: %w", err)
	}

	if syncResult.Action == tui.SyncActionNone {
		return nil // User cancelled
	}

	if syncResult.Action == tui.SyncActionPreview {
		// Show preview for the selected skill
		skill := syncResult.PreviewSkill
		ui.Info(fmt.Sprintf("Preview: %s", skill.Name))
		fmt.Println()
		fmt.Println(skill.Content)
		return nil
	}

	// Step 4: Perform the sync for selected skills
	if len(syncResult.SelectedSkills) == 0 {
		ui.Info("No skills selected for sync")
		return nil
	}

	// Create backup before sync
	prepareBackup(targetPlatform)
	created, err := backupExistingTargetSkills(
		targetPlatform,
		targetScope,
		syncResult.SelectedSkills,
		"pre-sync backup",
		[]string{"sync"},
	)
	if err != nil {
		return fmt.Errorf("create backups for %s: %w", targetPlatform, err)
	}
	if created > 0 {
		fmt.Printf("✓ Created %d backup(s)\n", created)
	}

	// Perform sync
	syncer := sync.New()
	opts := sync.Options{
		Strategy:    sync.StrategyOverwrite,
		TargetScope: targetScope,
	}
	result, err := syncer.SyncWithSkills(syncResult.SelectedSkills, targetPlatform, opts)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Display results
	changed := result.TotalChanged()
	ui.Success(fmt.Sprintf("Synced %d skills from %s to %s", changed, sourcePlatform, targetPlatform))
	if len(result.Skipped()) > 0 {
		ui.Info(fmt.Sprintf("Skipped %d skills (already up to date)", len(result.Skipped())))
	}
	if result.HasConflicts() {
		ui.Warning(fmt.Sprintf("%d conflicts detected - use 'Resolve Conflicts' to handle them", len(result.Conflicts())))
	}

	// Post-sync orphan detection and cleanup
	targetSkills, err := parsePlatformSkillsWithScope(
		targetPlatform,
		[]model.SkillScope{targetScope},
		false,
	)
	if err != nil {
		ui.Warning(fmt.Sprintf("Could not check for orphaned skills: %v", err))
		return nil
	}

	orphans := findOrphanedSkills(sourceSkills, targetSkills)
	if len(orphans) == 0 {
		return nil
	}

	fmt.Printf("\nFound %d orphaned skill(s) in %s not present in %s\n", len(orphans), targetPlatform, sourcePlatform)
	confirmed, err := confirmAction(
		"Review orphaned skills for cleanup?",
		riskLevelInfo,
	)
	if err != nil || !confirmed {
		return nil
	}

	deleteResult, err := tui.RunDeleteList(orphans)
	if err != nil {
		return fmt.Errorf("delete list error: %w", err)
	}

	if deleteResult.Action == tui.DeleteActionDelete {
		return executeDelete(deleteResult)
	}

	return nil
}

// runConfigTUI runs the configuration editor TUI view.
func runConfigTUI() error {
	cfg, err := config.Load()
	if err != nil {
		ui.Warning(fmt.Sprintf("Could not load config: %v", err))
		cfg = config.Default()
	}

	result, err := tui.RunConfigList(cfg)
	if err != nil {
		return fmt.Errorf("config TUI error: %w", err)
	}

	// Handle the result
	if result.Action == tui.ConfigActionNone {
		return nil
	}

	if result.Action == tui.ConfigActionSave {
		if err := result.Config.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}
		ui.Success("Configuration saved to " + config.FilePath())
	}

	return nil
}

// runExportTUI runs the export TUI view.
func runExportTUI() error {
	// Discover skills from all platforms
	skills, err := discoverSkillsForExport("")
	if err != nil {
		return fmt.Errorf("failed to discover skills: %w", err)
	}

	if len(skills) == 0 {
		ui.Info("No skills found to export")
		return nil
	}

	result, err := tui.RunExportList(skills)
	if err != nil {
		return fmt.Errorf("export TUI error: %w", err)
	}

	// Handle the result
	if result.Action == tui.ExportActionNone {
		return nil
	}

	if result.Action == tui.ExportActionExport {
		return executeExport(result)
	}

	return nil
}

// executeExport performs the actual export based on TUI result.
func executeExport(result tui.ExportListResult) error {
	if len(result.SelectedSkills) == 0 {
		ui.Info("No skills selected for export")
		return nil
	}

	// Build export options
	opts := export.Options{
		Format:          result.Format,
		Pretty:          result.Pretty,
		IncludeMetadata: result.IncludeMetadata,
	}

	// Create exporter
	exporter := export.New(opts)

	// Write to stdout
	if err := exporter.Export(result.SelectedSkills, os.Stdout); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nExported %d skill(s) as %s\n", len(result.SelectedSkills), result.Format)
	return nil
}

// runImportTUI runs the import skills TUI view.
func runImportTUI() error {
	result, err := tui.RunImportList()
	if err != nil {
		return fmt.Errorf("import TUI error: %w", err)
	}

	// Handle the result
	if result.Action == tui.ImportActionNone {
		return nil
	}

	if result.Action == tui.ImportActionImport {
		return executeImport(result)
	}

	return nil
}

// executeImport performs the actual import based on TUI result.
func executeImport(result tui.ImportListResult) error {
	if len(result.SelectedSkills) == 0 {
		ui.Info("No skills selected for import")
		return nil
	}

	// Create synchronizer for the import operation
	syncer := sync.New()
	opts := sync.Options{
		Strategy:    sync.StrategyOverwrite,
		DryRun:      false,
		TargetScope: result.TargetScope,
	}

	// Perform the import (sync from source to target)
	syncResult, err := syncer.SyncWithSkills(result.SelectedSkills, result.TargetPlatform, opts)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	// Report results
	var imported, skipped, failed int
	for _, skill := range syncResult.Skills {
		switch skill.Action {
		case sync.ActionCreated, sync.ActionUpdated:
			imported++
		case sync.ActionSkipped:
			skipped++
		case sync.ActionFailed:
			failed++
			ui.Error(fmt.Sprintf("Failed to import %s: %s", skill.Skill.Name, skill.Error))
		}
	}

	if imported > 0 {
		ui.Success(fmt.Sprintf("Imported %d skill(s) to %s (%s)", imported, result.TargetPlatform, result.TargetScope))
	}
	if skipped > 0 {
		ui.Info(fmt.Sprintf("Skipped %d skill(s) (already up to date)", skipped))
	}
	if failed > 0 {
		ui.Warning(fmt.Sprintf("%d skill(s) failed to import", failed))
	}

	return nil
}

// runDeleteTUI runs the delete skills TUI view.
func runDeleteTUI() error {
	// Discover skills from all platforms
	var allSkills []model.Skill
	for _, p := range model.AllPlatforms() {
		skills, err := parsePlatformSkillsWithScope(p, nil, false)
		if err != nil {
			// Log error but continue with other platforms
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	if len(allSkills) == 0 {
		ui.Info("No skills found")
		return nil
	}

	result, err := runDeleteList(allSkills)
	if err != nil {
		return fmt.Errorf("delete TUI error: %w", err)
	}

	// Handle the result
	if result.Action == tui.DeleteActionNone {
		return nil
	}

	if result.Action == tui.DeleteActionDelete {
		return executeDelete(result)
	}

	return nil
}

// executeDelete performs the actual deletion based on TUI result.
func executeDelete(result tui.DeleteListResult) error {
	if len(result.SelectedSkills) == 0 {
		ui.Info("No skills selected for deletion")
		return nil
	}

	// Delete each selected skill
	var deleted int
	var errors []string
	for _, skill := range result.SelectedSkills {
		// Verify the skill is in a writable scope
		if skill.Scope != model.ScopeRepo && skill.Scope != model.ScopeUser {
			errors = append(errors, fmt.Sprintf("%s: scope %q is not writable", skill.Name, skill.Scope))
			continue
		}

		// Delete the skill file
		if err := os.Remove(skill.Path); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", skill.Name, err))
			continue
		}

		// Try to remove the parent directory if it's empty (for directory-based skills)
		if strings.HasSuffix(skill.Path, "/SKILL.md") {
			parentDir := skill.Path[:len(skill.Path)-len("/SKILL.md")]
			_ = os.Remove(parentDir) // Ignore error - directory may not be empty
		}

		deleted++
	}

	if deleted > 0 {
		ui.Success(fmt.Sprintf("Deleted %d skill(s)", deleted))
	}

	if len(errors) > 0 {
		for _, e := range errors {
			ui.Error(fmt.Sprintf("Failed: %s", e))
		}
		return fmt.Errorf("some deletions failed")
	}

	return nil
}

// runScopeTUI runs the scope management TUI view.
func runScopeTUI() error {
	// Discover skills from all platforms
	var allSkills []model.Skill
	for _, p := range model.AllPlatforms() {
		skills, err := parsePlatformSkillsWithScope(p, nil, false)
		if err != nil {
			// Log error but continue with other platforms
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	// Include plugin skills
	pluginSkills, err := discoverPluginSkills("", true)
	if err == nil {
		allSkills = append(allSkills, pluginSkills...)
	}

	if len(allSkills) == 0 {
		ui.Info("No skills found")
		return nil
	}

	result, err := tui.RunScopeList(allSkills)
	if err != nil {
		return fmt.Errorf("scope TUI error: %w", err)
	}

	// Handle the result
	if result.Action == tui.ScopeActionNone {
		return nil
	}

	if result.Action == tui.ScopeActionView {
		// Display skill details
		skill := result.SelectedSkill
		fmt.Println()
		fmt.Println(ui.Bold(fmt.Sprintf("Skill: %s", skill.Name)))
		fmt.Println(ui.Info(fmt.Sprintf("Platform:    %s", skill.Platform)))
		fmt.Println(ui.Info(fmt.Sprintf("Scope:       %s", skill.DisplayScope())))
		fmt.Println(ui.Info(fmt.Sprintf("Description: %s", skill.Description)))
		fmt.Println(ui.Info(fmt.Sprintf("Path:        %s", skill.Path)))
		if len(skill.Tools) > 0 {
			fmt.Println(ui.Info(fmt.Sprintf("Tools:       %s", strings.Join(skill.Tools, ", "))))
		}
		if len(skill.References) > 0 {
			fmt.Println(ui.Info(fmt.Sprintf("References:  %s", strings.Join(skill.References, ", "))))
		}
	}

	return nil
}

// runConflictsTUI runs the conflict resolution TUI view.
// This scans for potential conflicts across platforms and shows them for resolution.
func runConflictsTUI() error {
	// Discover skills from all platforms to find potential conflicts
	platformSkills := make(map[model.Platform][]model.Skill)
	for _, p := range model.AllPlatforms() {
		skills, err := parsePlatformSkillsWithScope(p, nil, false)
		if err != nil {
			continue
		}
		if len(skills) > 0 {
			platformSkills[p] = skills
		}
	}

	// Need at least 2 platforms with skills to have potential conflicts
	if len(platformSkills) < 2 {
		ui.Info("Not enough platforms with skills to check for conflicts")
		ui.Info("Skills need to exist on at least 2 platforms to detect conflicts")
		return nil
	}

	// Find skills that exist on multiple platforms with different content
	detector := sync.NewConflictDetector()
	var conflicts []*sync.Conflict

	// Build map of skill name -> skills across platforms
	skillMap := make(map[string][]model.Skill)
	for _, skills := range platformSkills {
		for _, skill := range skills {
			skillMap[skill.Name] = append(skillMap[skill.Name], skill)
		}
	}

	// Check each skill that exists on multiple platforms
	for _, skills := range skillMap {
		if len(skills) < 2 {
			continue
		}
		// Compare first skill with others
		for i := 1; i < len(skills); i++ {
			conflict := detector.DetectConflict(skills[0], skills[i])
			if conflict != nil {
				conflicts = append(conflicts, conflict)
			}
		}
	}

	if len(conflicts) == 0 {
		ui.Success("No conflicts found across platforms")
		ui.Info("All skills with the same name have identical content")
		return nil
	}

	// Run the conflict resolution TUI
	result, err := tui.RunConflictList(conflicts)
	if err != nil {
		return fmt.Errorf("conflict TUI error: %w", err)
	}

	if result.Action == tui.ConflictActionNone || result.Action == tui.ConflictActionCancel {
		return nil
	}

	// Apply resolutions
	if result.Action == tui.ConflictActionResolve {
		applied := 0
		for _, resolution := range result.Resolutions {
			if resolution.Resolution == sync.ResolutionSkip {
				continue
			}
			// Find the skills involved in this conflict
			skills := skillMap[resolution.SkillName]
			if len(skills) == 0 {
				continue
			}

			// Determine content to write based on resolution
			var content string
			switch resolution.Resolution {
			case sync.ResolutionUseSource:
				content = skills[0].Content
			case sync.ResolutionUseTarget:
				if len(skills) > 1 {
					content = skills[1].Content
				}
			case sync.ResolutionMerge:
				content = resolution.Content
			default:
				continue
			}

			// Update all instances of this skill across platforms
			for _, skill := range skills {
				if skill.Content == content {
					continue // Already has the resolved content
				}
				if skill.Path != "" {
					if err := os.WriteFile(skill.Path, []byte(content), 0o600); err != nil {
						ui.Warning(fmt.Sprintf("Failed to update %s on %s: %v", skill.Name, skill.Platform, err))
						continue
					}
					applied++
				}
			}
		}
		if applied > 0 {
			ui.Success(fmt.Sprintf("Applied %d resolution(s)", applied))
		}
	}

	return nil
}

// runCompareTUI runs the compare skills TUI view with side-by-side comparison.
func runCompareTUI() error {
	// Discover skills from all platforms
	var allSkills []model.Skill
	for _, p := range model.AllPlatforms() {
		skills, err := parsePlatformSkillsWithScope(p, nil, false)
		if err != nil {
			// Log error but continue with other platforms
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	if len(allSkills) < 2 {
		ui.Info("Not enough skills to compare (need at least 2)")
		return nil
	}

	// Load config for thresholds
	appConfig, err := config.Load()
	if err != nil {
		appConfig = config.Default()
	}

	// Find similar skills using default thresholds
	comparisons, err := findDuplicatesForTUI(allSkills, appConfig)
	if err != nil {
		return fmt.Errorf("failed to find similar skills: %w", err)
	}
	comparisons = filterComparisonResultsByPlatform(comparisons, false)

	if len(comparisons) == 0 {
		ui.Info("No similar skills found to compare")
		return nil
	}

	result, err := tui.RunCompareList(comparisons)
	if err != nil {
		return fmt.Errorf("compare TUI error: %w", err)
	}

	// Only proceed to dedupe when the user explicitly asked for it ('d').
	// Quitting the compare TUI ('q'/ctrl+c) leaves CompareActionNone and must
	// not drop the user into the deletion workflow.
	if result.Action != tui.CompareActionDedupe {
		return nil
	}

	// Offer an interactive dedupe action on the same duplicates so the user can
	// select and delete them without re-typing platform/scope/name. RunDedupeList
	// filters to writable scopes and returns early (no TUI) when nothing is
	// deletable, so the empty case is handled.
	dedupeResult, err := tui.RunDedupeList(comparisons)
	if err != nil {
		return fmt.Errorf("dedupe TUI error: %w", err)
	}
	if dedupeResult.Action == tui.DedupeActionDelete {
		return deleteSelectedDuplicates(dedupeResult.SelectedSkills)
	}

	return nil
}

// findDuplicatesForTUI finds duplicate skill pairs using similarity matching.
func findDuplicatesForTUI(skills []model.Skill, cfg *config.Config) ([]*similarity.ComparisonResult, error) {
	var results []*similarity.ComparisonResult
	comparedPairs := make(map[string]bool)

	// Name similarity matching
	nameConfig := similarity.NameMatcherConfig{
		Threshold: cfg.Similarity.NameThreshold,
		Algorithm: cfg.Similarity.Algorithm,
		Normalize: true,
	}
	nameMatcher := similarity.NewNameMatcher(nameConfig)
	nameMatches := nameMatcher.FindSimilar(skills)

	for _, match := range nameMatches {
		pairKey := makeDupePairKey(match.Skill1, match.Skill2)
		if comparedPairs[pairKey] {
			continue
		}
		comparedPairs[pairKey] = true

		// Compute content score
		contentConfig := similarity.ContentMatcherConfig{
			Threshold: 0, // Don't filter, we want the score
			Algorithm: cfg.Similarity.Algorithm,
			LineMode:  true,
		}
		contentMatcher := similarity.NewContentMatcher(contentConfig)
		contentScore := contentMatcher.Compare(match.Skill1.Content, match.Skill2.Content)

		result := similarity.ComputeDiff(match.Skill1, match.Skill2, match.Score, contentScore)
		results = append(results, result)
	}

	// Content similarity matching
	contentConfig := similarity.ContentMatcherConfig{
		Threshold: cfg.Similarity.ContentThreshold,
		Algorithm: cfg.Similarity.Algorithm,
		LineMode:  true,
	}
	contentMatcher := similarity.NewContentMatcher(contentConfig)
	contentMatches := contentMatcher.FindSimilar(skills)

	for _, match := range contentMatches {
		pairKey := makeDupePairKey(match.Skill1, match.Skill2)
		if comparedPairs[pairKey] {
			continue
		}
		comparedPairs[pairKey] = true

		// Compute name score
		nameConfig := similarity.NameMatcherConfig{
			Threshold: 0, // Don't filter, we want the score
			Algorithm: cfg.Similarity.Algorithm,
			Normalize: true,
		}
		nameMatcher := similarity.NewNameMatcher(nameConfig)
		nameScore := nameMatcher.Compare(match.Skill1.Name, match.Skill2.Name)

		result := similarity.ComputeDiff(match.Skill1, match.Skill2, nameScore, match.Score)
		results = append(results, result)
	}

	return results, nil
}

// makeDupePairKey creates a consistent key for a skill pair regardless of order.
func makeDupePairKey(s1, s2 model.Skill) string {
	key1 := fmt.Sprintf("%s:%s:%s", s1.Platform, s1.Scope, s1.Name)
	key2 := fmt.Sprintf("%s:%s:%s", s2.Platform, s2.Scope, s2.Name)
	if key1 < key2 {
		return key1 + "|" + key2
	}
	return key2 + "|" + key1
}

// runPromoteDemoteTUI runs the promote/demote skills TUI view.
func runPromoteDemoteTUI() error {
	// Discover skills from all platforms
	var allSkills []model.Skill
	for _, p := range model.AllPlatforms() {
		skills, err := parsePlatformSkillsWithScope(p, nil, false)
		if err != nil {
			// Log error but continue with other platforms
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	if len(allSkills) == 0 {
		ui.Info("No skills found")
		return nil
	}

	result, err := tui.RunPromoteDemoteList(allSkills)
	if err != nil {
		return fmt.Errorf("promote/demote TUI error: %w", err)
	}

	// Handle the result
	if result.Action == tui.PromoteDemoteActionNone {
		return nil
	}

	return executePromoteDemote(result)
}

// executePromoteDemote performs the actual promote/demote based on TUI result.
func executePromoteDemote(result tui.PromoteDemoteListResult) error {
	if len(result.SelectedSkills) == 0 {
		ui.Info("No skills selected")
		return nil
	}

	isPromotion := result.Action == tui.PromoteDemoteActionPromote
	operation := "Demote"
	if isPromotion {
		operation = "Promote"
	}

	var processed int
	var errors []string

	for _, skill := range result.SelectedSkills {
		// Determine source and target scopes based on operation type
		var fromScope, toScope model.SkillScope
		if isPromotion {
			// Promote: repo -> user
			if skill.Scope != model.ScopeRepo {
				continue // Skip skills that can't be promoted
			}
			fromScope = model.ScopeRepo
			toScope = model.ScopeUser
		} else {
			// Demote: user -> repo
			if skill.Scope != model.ScopeUser {
				continue // Skip skills that can't be demoted
			}
			fromScope = model.ScopeUser
			toScope = model.ScopeRepo
		}

		// Get target path
		targetPath, err := getSkillPathForScope(skill.Platform, toScope, skill.Name)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to determine target path: %v", skill.Name, err))
			continue
		}

		// Ensure target directory exists
		// #nosec G301 - skill directories need to be readable by the platform
		targetDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetDir, 0o750); err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to create target directory: %v", skill.Name, err))
			continue
		}

		// Read source content
		// #nosec G304 - skill.Path comes from parsed skill files
		content, err := os.ReadFile(skill.Path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to read source: %v", skill.Name, err))
			continue
		}

		// Write to target
		// #nosec G306 G703 - skill files should be readable; targetPath comes from getSkillPathForScope (controlled internal function)
		if err := os.WriteFile(targetPath, content, 0o644); err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to write to target: %v", skill.Name, err))
			continue
		}

		// Remove source if requested
		if result.RemoveSource {
			if err := os.Remove(skill.Path); err != nil {
				errors = append(errors, fmt.Sprintf("%s: copied but failed to remove source: %v", skill.Name, err))
				// Don't continue - the copy was successful
			}
		}

		processed++
		_ = fromScope // Used for clarity in logic above
	}

	if processed > 0 {
		modeText := "copied"
		if result.RemoveSource {
			modeText = "moved"
		}
		ui.Success(fmt.Sprintf("%sd %d skill(s) (%s)", operation, processed, modeText))
	}

	if len(errors) > 0 {
		for _, e := range errors {
			ui.Error(fmt.Sprintf("Failed: %s", e))
		}
		return fmt.Errorf("some operations failed")
	}

	return nil
}
