package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/backup"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/ui"
	"github.com/klauern/skillsync/internal/ui/tui"
)

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
				Usage:   "Platform to back up (claude-code, cursor, codex, pi.dev, all)",
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
				return err
			}

			var platforms []model.Platform
			if platformStr == "" || platformStr == "all" {
				platforms = model.AllPlatforms()
			} else {
				platform, err := model.ParsePlatform(platformStr)
				if err != nil {
					return err
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
					return err
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
				Usage:   "Filter by platform (claude-code, cursor, codex, pi.dev)",
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
				Usage:   "Filter by platform (claude-code, cursor, codex, pi.dev)",
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
				Usage:   "Filter by platform (claude-code, cursor, codex, pi.dev)",
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
