package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/klauern/skillsync/internal/backup"
	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/ui"
	"github.com/klauern/skillsync/internal/util"
)

// PlatformStats holds statistics for a single platform.
type PlatformStats struct {
	Name       string   `json:"name"`
	SkillCount int      `json:"skill_count"`
	DiskUsage  int64    `json:"disk_usage_bytes"`
	Paths      []string `json:"paths"`
}

// Stats holds overall statistics.
type Stats struct {
	Platforms      []PlatformStats `json:"platforms"`
	TotalSkills    int             `json:"total_skills"`
	TotalDiskUsage int64           `json:"total_disk_usage_bytes"`
	LastBackup     *time.Time      `json:"last_backup,omitempty"`
	CacheEnabled   bool            `json:"cache_enabled"`
	CacheSize      int64           `json:"cache_size_bytes"`
}

func statsCommand() *cli.Command {
	return &cli.Command{
		Name:  "stats",
		Usage: "Display statistics and system information",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "Output in JSON format for scripting",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			logging.Debug("collecting statistics")

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			stats, err := collectStats(cfg)
			if err != nil {
				return fmt.Errorf("failed to collect statistics: %w", err)
			}

			if cmd.Bool("json") {
				return outputStatsJSON(stats)
			}

			return outputStatsTable(stats)
		},
	}
}

// collectStats gathers statistics from all platforms and system components.
func collectStats(cfg *config.Config) (*Stats, error) {
	stats := &Stats{
		CacheEnabled: cfg.Cache.Enabled,
	}

	// Collect platform statistics
	platforms := map[string]config.PlatformConfig{
		"claude-code": cfg.Platforms.ClaudeCode,
		"cursor":      cfg.Platforms.Cursor,
		"codex":       cfg.Platforms.Codex,
	}

	for name, platformCfg := range platforms {
		pStats, err := collectPlatformStats(name, platformCfg)
		if err != nil {
			logging.Warn("failed to collect stats for platform",
				logging.Platform(name),
				logging.Err(err),
			)
			continue
		}
		stats.Platforms = append(stats.Platforms, pStats)
		stats.TotalSkills += pStats.SkillCount
		stats.TotalDiskUsage += pStats.DiskUsage
	}

	// Get last backup time
	if lastBackup, err := getLastBackupTime(cfg); err == nil {
		stats.LastBackup = lastBackup
	}

	// Get cache size
	if cfg.Cache.Enabled {
		if size, err := getCacheSize(cfg.Cache.Location); err == nil {
			stats.CacheSize = size
		}
	}

	return stats, nil
}

// collectPlatformStats gathers statistics for a single platform.
func collectPlatformStats(name string, cfg config.PlatformConfig) (PlatformStats, error) {
	stats := PlatformStats{
		Name:  name,
		Paths: cfg.SkillsPaths,
	}

	// Parse platform enum
	platform, err := model.ParsePlatform(name)
	if err != nil {
		return stats, fmt.Errorf("invalid platform: %w", err)
	}

	// Discover skills using existing helper
	skills, err := parsePlatformSkillsWithScope(platform, nil)
	if err != nil {
		// Not an error if platform has no skills configured
		logging.Debug("no skills found for platform",
			logging.Platform(name),
		)
		return stats, nil
	}

	stats.SkillCount = len(skills)

	// Calculate disk usage across all paths
	cwd, _ := os.Getwd()
	for _, path := range cfg.SkillsPaths {
		expanded := util.ExpandPath(path, cwd)
		if size, err := calculateDiskUsage(expanded); err == nil {
			stats.DiskUsage += size
		}
	}

	return stats, nil
}

// getLastBackupTime returns the timestamp of the most recent backup.
func getLastBackupTime(cfg *config.Config) (*time.Time, error) {
	backupDir := util.ExpandPath(cfg.Backup.Location, "")
	backups, err := backup.ListBackups(backupDir)
	if err != nil {
		return nil, err
	}

	if len(backups) == 0 {
		return nil, nil
	}

	// Backups are sorted by timestamp descending
	return &backups[0].CreatedAt, nil
}

// getCacheSize calculates the total size of the cache directory.
func getCacheSize(cacheDir string) (int64, error) {
	expanded := util.ExpandPath(cacheDir, "")
	return calculateDiskUsage(expanded)
}

// calculateDiskUsage recursively calculates disk usage for a directory.
func calculateDiskUsage(path string) (int64, error) {
	var totalSize int64

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip paths that don't exist or can't be accessed
			if os.IsNotExist(err) || os.IsPermission(err) {
				return nil
			}
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to calculate disk usage: %w", err)
	}

	return totalSize, nil
}

// outputStatsJSON outputs statistics in JSON format.
func outputStatsJSON(stats *Stats) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(stats); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

// outputStatsTable outputs statistics in human-readable table format.
func outputStatsTable(stats *Stats) error {
	// Print header
	fmt.Println(ui.Bold("skillsync Statistics"))
	fmt.Println()

	// Platform statistics
	fmt.Println(ui.Bold("Platforms:"))
	for _, p := range stats.Platforms {
		fmt.Printf("  %s:\n", ui.Info(p.Name))
		fmt.Printf("    Skills:     %d\n", p.SkillCount)
		fmt.Printf("    Disk Usage: %s\n", formatBytes(p.DiskUsage))
		if len(p.Paths) > 0 {
			fmt.Printf("    Paths:      %s\n", p.Paths[0])
			for i := 1; i < len(p.Paths); i++ {
				fmt.Printf("                %s\n", p.Paths[i])
			}
		}
	}
	fmt.Println()

	// Totals
	fmt.Println(ui.Bold("Totals:"))
	fmt.Printf("  Skills:     %d\n", stats.TotalSkills)
	fmt.Printf("  Disk Usage: %s\n", formatBytes(stats.TotalDiskUsage))
	fmt.Println()

	// Backup information
	fmt.Println(ui.Bold("Backups:"))
	if stats.LastBackup != nil {
		fmt.Printf("  Last Backup: %s (%s)\n",
			stats.LastBackup.Format("2006-01-02 15:04:05"),
			formatDuration(time.Since(*stats.LastBackup)))
	} else {
		fmt.Println("  Last Backup: None")
	}
	fmt.Println()

	// Cache information
	fmt.Println(ui.Bold("Cache:"))
	if stats.CacheEnabled {
		fmt.Printf("  Status: %s\n", ui.Success("Enabled"))
		fmt.Printf("  Size:   %s\n", formatBytes(stats.CacheSize))
	} else {
		fmt.Printf("  Status: %s\n", ui.Warning("Disabled"))
	}

	return nil
}

// formatBytes formats byte count in human-readable format.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats duration in human-readable format.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	if days < 30 {
		return fmt.Sprintf("%d days ago", days)
	}
	months := days / 30
	if months == 1 {
		return "1 month ago"
	}
	if months < 12 {
		return fmt.Sprintf("%d months ago", months)
	}
	years := months / 12
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}
