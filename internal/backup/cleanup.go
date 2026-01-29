// Package backup provides automatic backup functionality for skill directories
package backup

import (
	"fmt"
	"time"
)

// CleanupOptions configures backup cleanup behavior
type CleanupOptions struct {
	// MaxBackups limits the number of backups to keep per platform (0 = unlimited)
	MaxBackups int

	// MaxAge is the maximum age of backups to keep (0 = unlimited)
	MaxAge time.Duration

	// KeepAtLeastOne ensures at least one backup is kept per source file
	KeepAtLeastOne bool

	// Platform filters cleanup to a specific platform (empty = all platforms)
	Platform string

	// DryRun previews what would be deleted without actually deleting
	DryRun bool
}

// DefaultCleanupOptions returns sensible defaults for cleanup
func DefaultCleanupOptions() CleanupOptions {
	return CleanupOptions{
		MaxBackups:     10,                  // Keep last 10 backups per platform
		MaxAge:         30 * 24 * time.Hour, // Keep backups for 30 days
		KeepAtLeastOne: true,
		Platform:       "",
	}
}

// CleanupBackups removes old backups based on the specified options
func CleanupBackups(opts CleanupOptions) ([]string, error) {
	// Load index
	index, err := LoadIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to load backup index: %w", err)
	}

	// Group backups by platform and source path
	type backupGroup struct {
		platform   string
		sourcePath string
		backups    []Metadata
	}

	groups := make(map[string]*backupGroup)

	for _, backup := range index.Backups {
		// Filter by platform if specified
		if opts.Platform != "" && backup.Platform != opts.Platform {
			continue
		}

		key := backup.Platform + ":" + backup.SourcePath
		if _, exists := groups[key]; !exists {
			groups[key] = &backupGroup{
				platform:   backup.Platform,
				sourcePath: backup.SourcePath,
				backups:    make([]Metadata, 0),
			}
		}
		groups[key].backups = append(groups[key].backups, backup)
	}

	// Sort backups in each group by creation time (newest first)
	for _, group := range groups {
		for i := 0; i < len(group.backups)-1; i++ {
			for j := i + 1; j < len(group.backups); j++ {
				if group.backups[i].CreatedAt.Before(group.backups[j].CreatedAt) {
					group.backups[i], group.backups[j] = group.backups[j], group.backups[i]
				}
			}
		}
	}

	// Determine which backups to delete
	var toDelete []string
	now := time.Now()

	for _, group := range groups {
		keepCount := 0
		for idx, backup := range group.backups {
			shouldDelete := false

			// Check age
			if opts.MaxAge > 0 && now.Sub(backup.CreatedAt) > opts.MaxAge {
				shouldDelete = true
			}

			// Check count limit
			if opts.MaxBackups > 0 && idx >= opts.MaxBackups {
				shouldDelete = true
			}

			if !shouldDelete {
				keepCount++
			}

			if shouldDelete {
				toDelete = append(toDelete, backup.ID)
			}
		}

		// If KeepAtLeastOne is true and we're deleting everything, keep the newest
		if opts.KeepAtLeastOne && keepCount == 0 && len(toDelete) > 0 {
			// Remove the first item from toDelete (newest backup)
			toDelete = toDelete[1:]
		}
	}

	// Delete backups (or just return the list in dry-run mode)
	var deleted []string
	for _, backupID := range toDelete {
		if opts.DryRun {
			// In dry-run mode, just collect the IDs without deleting
			deleted = append(deleted, backupID)
		} else {
			if err := DeleteBackup(backupID); err != nil {
				return deleted, fmt.Errorf("failed to delete backup %q: %w", backupID, err)
			}
			deleted = append(deleted, backupID)
		}
	}

	return deleted, nil
}

// GetStats returns statistics about backups
func GetStats() (*Stats, error) {
	index, err := LoadIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to load backup index: %w", err)
	}

	stats := &Stats{
		TotalBackups:      len(index.Backups),
		TotalSize:         0,
		BackupsByPlatform: make(map[string]int),
		OldestBackup:      time.Now(),
		NewestBackup:      time.Time{},
	}

	for _, backup := range index.Backups {
		// Total size
		stats.TotalSize += backup.Size

		// Count by platform
		stats.BackupsByPlatform[backup.Platform]++

		// Oldest and newest
		if backup.CreatedAt.Before(stats.OldestBackup) {
			stats.OldestBackup = backup.CreatedAt
		}
		if backup.CreatedAt.After(stats.NewestBackup) {
			stats.NewestBackup = backup.CreatedAt
		}
	}

	// Reset oldest if no backups
	if stats.TotalBackups == 0 {
		stats.OldestBackup = time.Time{}
	}

	return stats, nil
}

// Stats contains statistics about backups
type Stats struct {
	TotalBackups      int
	TotalSize         int64
	BackupsByPlatform map[string]int
	OldestBackup      time.Time
	NewestBackup      time.Time
}
