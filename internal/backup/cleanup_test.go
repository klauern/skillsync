package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/util"
)

func TestDefaultCleanupOptions(t *testing.T) {
	opts := DefaultCleanupOptions()

	util.AssertEqual(t, opts.MaxBackups, 10)
	util.AssertEqual(t, opts.MaxAge, 30*24*time.Hour)
	util.AssertEqual(t, opts.KeepAtLeastOne, true)
	util.AssertEqual(t, opts.Platform, "")
}

func TestCleanupBackups_AgeBasedCleanup(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Create backup directory
	backupDir := filepath.Join(tempHome, "backups")
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	now := time.Now()

	// Add backups with different ages directly to index
	backups := []Metadata{
		{
			ID:         "backup-new",
			Platform:   "claude-code",
			SourcePath: "/test/file.md",
			CreatedAt:  now.Add(-1 * time.Hour), // 1 hour old
			BackupPath: filepath.Join(backupDir, "backup-new.md"),
		},
		{
			ID:         "backup-medium",
			Platform:   "claude-code",
			SourcePath: "/test/file.md",
			CreatedAt:  now.Add(-24 * time.Hour), // 1 day old
			BackupPath: filepath.Join(backupDir, "backup-medium.md"),
		},
		{
			ID:         "backup-old",
			Platform:   "claude-code",
			SourcePath: "/test/file.md",
			CreatedAt:  now.Add(-48 * time.Hour), // 2 days old
			BackupPath: filepath.Join(backupDir, "backup-old.md"),
		},
	}

	// Create backup files and add to index
	for _, backup := range backups {
		if err := os.WriteFile(backup.BackupPath, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create backup file: %v", err)
		}
		if err := index.AddBackup(backup); err != nil {
			t.Fatalf("AddBackup failed: %v", err)
		}
	}

	// Cleanup backups older than 30 hours (should delete 2-day-old backup)
	cleanupOpts := CleanupOptions{
		MaxAge:         30 * time.Hour,
		MaxBackups:     0, // Unlimited by count
		KeepAtLeastOne: false,
		Platform:       "claude-code",
	}

	deleted, err := CleanupBackups(cleanupOpts)
	if err != nil {
		t.Fatalf("CleanupBackups failed: %v", err)
	}

	// Should delete the 2-day-old backup (48 hours > 30 hours)
	util.AssertEqual(t, len(deleted), 1)
	util.AssertEqual(t, deleted[0], "backup-old")

	// Verify the old backup file is deleted
	if _, err := os.Stat(filepath.Join(backupDir, "backup-old.md")); !os.IsNotExist(err) {
		t.Error("backup-old file should be deleted")
	}

	// Verify remaining backups
	remaining, err := ListBackups("claude-code")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	util.AssertEqual(t, len(remaining), 2)
}

func TestCleanupBackups_UnlimitedCount(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Create backup directory
	backupDir := filepath.Join(tempHome, "backups")
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	now := time.Now()

	// Create many backups (more than default limit)
	for i := range 15 {
		backup := Metadata{
			ID:         fmt.Sprintf("backup-%d", i),
			Platform:   "claude-code",
			SourcePath: "/test/file.md",
			CreatedAt:  now.Add(-time.Duration(i) * time.Hour),
			BackupPath: filepath.Join(backupDir, fmt.Sprintf("backup-%d.md", i)),
		}
		if err := os.WriteFile(backup.BackupPath, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create backup file: %v", err)
		}
		if err := index.AddBackup(backup); err != nil {
			t.Fatalf("AddBackup failed: %v", err)
		}
	}

	// Cleanup with MaxBackups=0 (unlimited) and no age limit
	cleanupOpts := CleanupOptions{
		MaxBackups:     0,
		MaxAge:         0,
		KeepAtLeastOne: false,
		Platform:       "claude-code",
	}

	deleted, err := CleanupBackups(cleanupOpts)
	if err != nil {
		t.Fatalf("CleanupBackups failed: %v", err)
	}

	// No backups should be deleted
	util.AssertEqual(t, len(deleted), 0)

	// All 15 backups should remain
	remaining, err := ListBackups("claude-code")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	util.AssertEqual(t, len(remaining), 15)
}

func TestCleanupBackups_KeepAtLeastOne_AllWouldBeDeleted(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Create backup directory
	backupDir := filepath.Join(tempHome, "backups")
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	now := time.Now()

	// Create 3 backups, all older than the age limit
	for i := range 3 {
		backup := Metadata{
			ID:         fmt.Sprintf("backup-%d", i),
			Platform:   "claude-code",
			SourcePath: "/test/file.md",
			CreatedAt:  now.Add(-time.Duration(10+i) * 24 * time.Hour), // All 10+ days old
			BackupPath: filepath.Join(backupDir, fmt.Sprintf("backup-%d.md", i)),
		}
		if err := os.WriteFile(backup.BackupPath, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create backup file: %v", err)
		}
		if err := index.AddBackup(backup); err != nil {
			t.Fatalf("AddBackup failed: %v", err)
		}
	}

	// Cleanup with age limit that would delete all backups
	cleanupOpts := CleanupOptions{
		MaxAge:         24 * time.Hour, // 1 day - all backups are older
		MaxBackups:     0,
		KeepAtLeastOne: true, // Should preserve the newest
		Platform:       "claude-code",
	}

	deleted, err := CleanupBackups(cleanupOpts)
	if err != nil {
		t.Fatalf("CleanupBackups failed: %v", err)
	}

	// Should delete 2 backups, keeping the newest one
	util.AssertEqual(t, len(deleted), 2)

	// Verify exactly 1 backup remains
	remaining, err := ListBackups("claude-code")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	util.AssertEqual(t, len(remaining), 1)
}

func TestCleanupBackups_KeepAtLeastOne_Disabled(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Create backup directory
	backupDir := filepath.Join(tempHome, "backups")
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	now := time.Now()

	// Create 3 backups, all older than the age limit
	for i := range 3 {
		backup := Metadata{
			ID:         fmt.Sprintf("backup-%d", i),
			Platform:   "claude-code",
			SourcePath: "/test/file.md",
			CreatedAt:  now.Add(-time.Duration(10+i) * 24 * time.Hour), // All 10+ days old
			BackupPath: filepath.Join(backupDir, fmt.Sprintf("backup-%d.md", i)),
		}
		if err := os.WriteFile(backup.BackupPath, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create backup file: %v", err)
		}
		if err := index.AddBackup(backup); err != nil {
			t.Fatalf("AddBackup failed: %v", err)
		}
	}

	// Cleanup with age limit that would delete all backups
	cleanupOpts := CleanupOptions{
		MaxAge:         24 * time.Hour, // 1 day - all backups are older
		MaxBackups:     0,
		KeepAtLeastOne: false, // All can be deleted
		Platform:       "claude-code",
	}

	deleted, err := CleanupBackups(cleanupOpts)
	if err != nil {
		t.Fatalf("CleanupBackups failed: %v", err)
	}

	// Should delete all 3 backups
	util.AssertEqual(t, len(deleted), 3)

	// Verify no backups remain
	remaining, err := ListBackups("claude-code")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	util.AssertEqual(t, len(remaining), 0)
}

func TestCleanupBackups_PlatformFilteringAllPlatforms(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Create backup directory
	backupDir := filepath.Join(tempHome, "backups")
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	now := time.Now()

	// Create backups for multiple platforms
	platforms := []string{"claude-code", "cursor", "codex"}
	for _, platform := range platforms {
		for i := range 4 {
			backup := Metadata{
				ID:         fmt.Sprintf("%s-backup-%d", platform, i),
				Platform:   platform,
				SourcePath: "/test/file.md",
				CreatedAt:  now.Add(-time.Duration(i) * time.Hour),
				BackupPath: filepath.Join(backupDir, fmt.Sprintf("%s-backup-%d.md", platform, i)),
			}
			if err := os.WriteFile(backup.BackupPath, []byte("content"), 0o600); err != nil {
				t.Fatalf("failed to create backup file: %v", err)
			}
			if err := index.AddBackup(backup); err != nil {
				t.Fatalf("AddBackup failed: %v", err)
			}
		}
	}

	// Cleanup with empty platform (all platforms), keeping only 2 per source
	cleanupOpts := CleanupOptions{
		MaxBackups:     2,
		MaxAge:         0,
		KeepAtLeastOne: false,
		Platform:       "", // All platforms
	}

	deleted, err := CleanupBackups(cleanupOpts)
	if err != nil {
		t.Fatalf("CleanupBackups failed: %v", err)
	}

	// Each platform had 4 backups, now should have 2 each
	// 3 platforms * 2 deleted = 6 total deleted
	util.AssertEqual(t, len(deleted), 6)

	// Verify remaining backups per platform
	for _, platform := range platforms {
		remaining, err := ListBackups(platform)
		if err != nil {
			t.Fatalf("ListBackups failed: %v", err)
		}
		util.AssertEqual(t, len(remaining), 2)
	}
}

func TestCleanupBackups_PlatformFilteringSpecificPlatform(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Create backup directory
	backupDir := filepath.Join(tempHome, "backups")
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	now := time.Now()

	// Create backups for multiple platforms
	platforms := []string{"claude-code", "cursor"}
	for _, platform := range platforms {
		for i := range 4 {
			backup := Metadata{
				ID:         fmt.Sprintf("%s-backup-%d", platform, i),
				Platform:   platform,
				SourcePath: "/test/file.md",
				CreatedAt:  now.Add(-time.Duration(i) * time.Hour),
				BackupPath: filepath.Join(backupDir, fmt.Sprintf("%s-backup-%d.md", platform, i)),
			}
			if err := os.WriteFile(backup.BackupPath, []byte("content"), 0o600); err != nil {
				t.Fatalf("failed to create backup file: %v", err)
			}
			if err := index.AddBackup(backup); err != nil {
				t.Fatalf("AddBackup failed: %v", err)
			}
		}
	}

	// Cleanup only claude-code platform
	cleanupOpts := CleanupOptions{
		MaxBackups:     2,
		MaxAge:         0,
		KeepAtLeastOne: false,
		Platform:       "claude-code",
	}

	deleted, err := CleanupBackups(cleanupOpts)
	if err != nil {
		t.Fatalf("CleanupBackups failed: %v", err)
	}

	// Only 2 claude-code backups should be deleted
	util.AssertEqual(t, len(deleted), 2)

	// Claude-code should have 2 remaining
	claudeBackups, err := ListBackups("claude-code")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	util.AssertEqual(t, len(claudeBackups), 2)

	// Cursor should still have all 4 (untouched)
	cursorBackups, err := ListBackups("cursor")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	util.AssertEqual(t, len(cursorBackups), 4)
}

func TestCleanupBackups_MultipleSourcePaths(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Create backup directory
	backupDir := filepath.Join(tempHome, "backups")
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	now := time.Now()

	// Create backups for multiple source paths
	sourcePaths := []string{"/test/file1.md", "/test/file2.md"}
	for _, sourcePath := range sourcePaths {
		for i := range 4 {
			backup := Metadata{
				ID:         fmt.Sprintf("backup-%s-%d", filepath.Base(sourcePath), i),
				Platform:   "claude-code",
				SourcePath: sourcePath,
				CreatedAt:  now.Add(-time.Duration(i) * time.Hour),
				BackupPath: filepath.Join(backupDir, fmt.Sprintf("backup-%s-%d.md", filepath.Base(sourcePath), i)),
			}
			if err := os.WriteFile(backup.BackupPath, []byte("content"), 0o600); err != nil {
				t.Fatalf("failed to create backup file: %v", err)
			}
			if err := index.AddBackup(backup); err != nil {
				t.Fatalf("AddBackup failed: %v", err)
			}
		}
	}

	// Cleanup keeping 2 backups per source path
	cleanupOpts := CleanupOptions{
		MaxBackups:     2,
		MaxAge:         0,
		KeepAtLeastOne: false,
		Platform:       "claude-code",
	}

	deleted, err := CleanupBackups(cleanupOpts)
	if err != nil {
		t.Fatalf("CleanupBackups failed: %v", err)
	}

	// Each source path had 4 backups, 2 deleted each = 4 total
	util.AssertEqual(t, len(deleted), 4)

	// Verify total remaining is 4 (2 per source path)
	remaining, err := ListBackups("claude-code")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	util.AssertEqual(t, len(remaining), 4)
}

func TestCleanupBackups_EmptyIndex(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	// Cleanup on empty index
	cleanupOpts := CleanupOptions{
		MaxBackups:     5,
		MaxAge:         24 * time.Hour,
		KeepAtLeastOne: true,
		Platform:       "",
	}

	deleted, err := CleanupBackups(cleanupOpts)
	if err != nil {
		t.Fatalf("CleanupBackups failed: %v", err)
	}

	util.AssertEqual(t, len(deleted), 0)
}

func TestCleanupBackups_CombinedAgeAndCount(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Create backup directory
	backupDir := filepath.Join(tempHome, "backups")
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	now := time.Now()

	// Create 5 backups with varying ages
	// backup-0: 1h old (keep - within count and age)
	// backup-1: 2h old (keep - within count and age)
	// backup-2: 3h old (keep - within count and age)
	// backup-3: 50h old (delete - exceeds age)
	// backup-4: 100h old (delete - exceeds age and would exceed count)
	ages := []time.Duration{1, 2, 3, 50, 100}
	for i, age := range ages {
		backup := Metadata{
			ID:         fmt.Sprintf("backup-%d", i),
			Platform:   "claude-code",
			SourcePath: "/test/file.md",
			CreatedAt:  now.Add(-age * time.Hour),
			BackupPath: filepath.Join(backupDir, fmt.Sprintf("backup-%d.md", i)),
		}
		if err := os.WriteFile(backup.BackupPath, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create backup file: %v", err)
		}
		if err := index.AddBackup(backup); err != nil {
			t.Fatalf("AddBackup failed: %v", err)
		}
	}

	// Cleanup with both age and count limits
	cleanupOpts := CleanupOptions{
		MaxBackups:     4,              // Would keep 4
		MaxAge:         24 * time.Hour, // But age limit kicks in first
		KeepAtLeastOne: false,
		Platform:       "claude-code",
	}

	deleted, err := CleanupBackups(cleanupOpts)
	if err != nil {
		t.Fatalf("CleanupBackups failed: %v", err)
	}

	// backups 3 and 4 should be deleted (exceed 24h age limit)
	util.AssertEqual(t, len(deleted), 2)

	// Verify 3 backups remain
	remaining, err := ListBackups("claude-code")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	util.AssertEqual(t, len(remaining), 3)
}

func TestGetStats_EmptyIndex(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	stats, err := GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	util.AssertEqual(t, stats.TotalBackups, 0)
	util.AssertEqual(t, stats.TotalSize, int64(0))
	util.AssertEqual(t, len(stats.BackupsByPlatform), 0)

	// OldestBackup should be zero time when no backups
	if !stats.OldestBackup.IsZero() {
		t.Errorf("expected zero time for OldestBackup, got %v", stats.OldestBackup)
	}
	if !stats.NewestBackup.IsZero() {
		t.Errorf("expected zero time for NewestBackup, got %v", stats.NewestBackup)
	}
}

func TestGetStats_SingleBackup(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	now := time.Now()
	backup := Metadata{
		ID:         "single-backup",
		Platform:   "claude-code",
		SourcePath: "/test/file.md",
		CreatedAt:  now,
		Size:       1024,
	}

	if err := index.AddBackup(backup); err != nil {
		t.Fatalf("AddBackup failed: %v", err)
	}

	stats, err := GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	util.AssertEqual(t, stats.TotalBackups, 1)
	util.AssertEqual(t, stats.TotalSize, int64(1024))
	util.AssertEqual(t, stats.BackupsByPlatform["claude-code"], 1)

	// Oldest and newest should be the same
	if !stats.OldestBackup.Equal(now) {
		t.Errorf("expected OldestBackup to equal creation time")
	}
	if !stats.NewestBackup.Equal(now) {
		t.Errorf("expected NewestBackup to equal creation time")
	}
}

func TestGetStats_MultiplePlatforms(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	now := time.Now()
	platforms := map[string]int{
		"claude-code": 5,
		"cursor":      3,
		"codex":       2,
	}

	backupID := 0
	for platform, count := range platforms {
		for i := range count {
			backup := Metadata{
				ID:         fmt.Sprintf("backup-%d", backupID),
				Platform:   platform,
				SourcePath: "/test/file.md",
				CreatedAt:  now.Add(-time.Duration(backupID) * time.Hour),
				Size:       int64(100 * (backupID + 1)),
			}
			if err := index.AddBackup(backup); err != nil {
				t.Fatalf("AddBackup failed: %v", err)
			}
			backupID++
			_ = i // silence unused variable
		}
	}

	stats, err := GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	util.AssertEqual(t, stats.TotalBackups, 10)
	util.AssertEqual(t, stats.BackupsByPlatform["claude-code"], 5)
	util.AssertEqual(t, stats.BackupsByPlatform["cursor"], 3)
	util.AssertEqual(t, stats.BackupsByPlatform["codex"], 2)

	// Total size should be sum of all sizes
	expectedSize := int64(0)
	for i := range 10 {
		expectedSize += int64(100 * (i + 1))
	}
	util.AssertEqual(t, stats.TotalSize, expectedSize)
}

func TestGetStats_OldestNewestTracking(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	now := time.Now()
	oldest := now.Add(-72 * time.Hour) // 3 days ago
	newest := now.Add(-1 * time.Hour)  // 1 hour ago

	backups := []Metadata{
		{ID: "backup-1", Platform: "claude-code", SourcePath: "/a", CreatedAt: now.Add(-24 * time.Hour)},
		{ID: "backup-2", Platform: "cursor", SourcePath: "/b", CreatedAt: oldest},
		{ID: "backup-3", Platform: "codex", SourcePath: "/c", CreatedAt: newest},
		{ID: "backup-4", Platform: "claude-code", SourcePath: "/d", CreatedAt: now.Add(-48 * time.Hour)},
	}

	for _, backup := range backups {
		if err := index.AddBackup(backup); err != nil {
			t.Fatalf("AddBackup failed: %v", err)
		}
	}

	stats, err := GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if !stats.OldestBackup.Equal(oldest) {
		t.Errorf("OldestBackup = %v, want %v", stats.OldestBackup, oldest)
	}
	if !stats.NewestBackup.Equal(newest) {
		t.Errorf("NewestBackup = %v, want %v", stats.NewestBackup, newest)
	}
}
