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

func TestCleanupBackups_KeepAtLeastOne_MultipleGroups(t *testing.T) {
	// Regression test: KeepAtLeastOne must preserve the newest backup within each
	// group, not accidentally remove an entry belonging to a different group.
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	backupDir := filepath.Join(tempHome, "backups")
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	now := time.Now()

	// Two separate source paths (groups), each with 2 old backups.
	// All backups exceed the age limit so keepCount==0 for every group.
	// KeepAtLeastOne must retain the newest backup *per group*.
	groups := []struct {
		sourcePath string
		ids        []string
		ages       []time.Duration // oldest-to-newest order for readability
	}{
		{
			sourcePath: "/test/file1.md",
			ids:        []string{"g1-old", "g1-new"},
			ages:       []time.Duration{48 * time.Hour, 10 * time.Hour},
		},
		{
			sourcePath: "/test/file2.md",
			ids:        []string{"g2-old", "g2-new"},
			ages:       []time.Duration{72 * time.Hour, 12 * time.Hour},
		},
	}

	for _, g := range groups {
		for i, id := range g.ids {
			backup := Metadata{
				ID:         id,
				Platform:   "claude-code",
				SourcePath: g.sourcePath,
				CreatedAt:  now.Add(-g.ages[i]),
				BackupPath: filepath.Join(backupDir, id+".md"),
			}
			if err := os.WriteFile(backup.BackupPath, []byte("content"), 0o600); err != nil {
				t.Fatalf("failed to create backup file: %v", err)
			}
			if err := index.AddBackup(backup); err != nil {
				t.Fatalf("AddBackup failed: %v", err)
			}
		}
	}

	cleanupOpts := CleanupOptions{
		MaxAge:         1 * time.Hour, // all backups are older than 1h
		MaxBackups:     0,
		KeepAtLeastOne: true,
		Platform:       "claude-code",
	}

	deleted, err := CleanupBackups(cleanupOpts)
	if err != nil {
		t.Fatalf("CleanupBackups failed: %v", err)
	}

	// 2 groups × (2 backups − 1 kept) = 2 deleted
	util.AssertEqual(t, len(deleted), 2)

	// The two newest (one per group) must survive.
	remaining, err := ListBackups("claude-code")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	util.AssertEqual(t, len(remaining), 2)

	survivorIDs := make(map[string]bool)
	for _, b := range remaining {
		survivorIDs[b.ID] = true
	}
	if !survivorIDs["g1-new"] {
		t.Errorf("expected g1-new to survive as the newest in group 1")
	}
	if !survivorIDs["g2-new"] {
		t.Errorf("expected g2-new to survive as the newest in group 2")
	}
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
