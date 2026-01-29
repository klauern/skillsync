package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/util"
)

func TestCreateBackup(t *testing.T) {
	// Setup temp environment
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	// Create test file
	testFile := filepath.Join(tempHome, "test-skill.md")
	content := "# Test Skill\n\nThis is a test skill."
	if err := os.WriteFile(testFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create backup
	opts := Options{
		Platform:    "claude-code",
		Description: "Test backup",
		Tags:        []string{"test"},
	}

	metadata, err := CreateBackup(testFile, opts)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify metadata
	util.AssertEqual(t, metadata.Platform, "claude-code")
	util.AssertEqual(t, metadata.Description, "Test backup")
	util.AssertEqual(t, metadata.SourcePath, testFile)

	if len(metadata.Hash) != 64 {
		t.Errorf("expected hash length 64, got %d", len(metadata.Hash))
	}

	// Verify backup file exists
	if _, err := os.Stat(metadata.BackupPath); os.IsNotExist(err) {
		t.Errorf("backup file does not exist: %s", metadata.BackupPath)
	}

	// Verify backup content matches original
	backupContent, err := os.ReadFile(metadata.BackupPath)
	if err != nil {
		t.Fatalf("failed to read backup file: %v", err)
	}

	util.AssertEqual(t, string(backupContent), content)
}

func TestBackupIndex(t *testing.T) {
	// Setup temp environment
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	// Load empty index
	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	util.AssertEqual(t, index.Version, IndexVersion)
	util.AssertEqual(t, len(index.Backups), 0)

	// Add backup
	metadata := Metadata{
		ID:         "test-backup-1",
		SourcePath: "/test/file.md",
		Platform:   "claude-code",
		CreatedAt:  time.Now(),
		Hash:       "abc123",
	}

	if err := index.AddBackup(metadata); err != nil {
		t.Fatalf("AddBackup failed: %v", err)
	}

	// Reload and verify
	index, err = LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	util.AssertEqual(t, len(index.Backups), 1)
	backup, exists := index.Backups["test-backup-1"]
	if !exists {
		t.Fatal("backup not found in index")
	}

	util.AssertEqual(t, backup.SourcePath, "/test/file.md")
	util.AssertEqual(t, backup.Platform, "claude-code")
}

func TestListBackups(t *testing.T) {
	// Setup temp environment
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Add multiple backups with different timestamps
	backups := []Metadata{
		{
			ID:         "backup-1",
			Platform:   "claude-code",
			CreatedAt:  time.Now().Add(-2 * time.Hour),
			SourcePath: "/test/file1.md",
		},
		{
			ID:         "backup-2",
			Platform:   "cursor",
			CreatedAt:  time.Now().Add(-1 * time.Hour),
			SourcePath: "/test/file2.md",
		},
		{
			ID:         "backup-3",
			Platform:   "claude-code",
			CreatedAt:  time.Now(),
			SourcePath: "/test/file3.md",
		},
	}

	for _, backup := range backups {
		if err := index.AddBackup(backup); err != nil {
			t.Fatalf("AddBackup failed: %v", err)
		}
	}

	// List all backups
	allBackups, err := ListBackups("")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	util.AssertEqual(t, len(allBackups), 3)

	// Verify sorted by newest first
	if allBackups[0].ID != "backup-3" {
		t.Errorf("expected newest backup first, got %s", allBackups[0].ID)
	}

	// List claude-code backups only
	claudeBackups, err := ListBackups("claude-code")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	util.AssertEqual(t, len(claudeBackups), 2)
}

func TestRestoreBackup(t *testing.T) {
	// Setup temp environment
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	// Create original file
	originalFile := filepath.Join(tempHome, "original.md")
	originalContent := "# Original Content"
	if err := os.WriteFile(originalFile, []byte(originalContent), 0o600); err != nil {
		t.Fatalf("failed to create original file: %v", err)
	}

	// Create backup
	opts := Options{Platform: "claude-code"}
	metadata, err := CreateBackup(originalFile, opts)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Restore to different location
	restoreFile := filepath.Join(tempHome, "restored.md")
	if err := RestoreBackup(metadata.ID, restoreFile); err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}

	// Verify restored content
	// #nosec G304 - restoreFile is controlled by test
	restoredContent, err := os.ReadFile(restoreFile)
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}

	util.AssertEqual(t, string(restoredContent), originalContent)
}

func TestDeleteBackup(t *testing.T) {
	// Setup temp environment
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	// Create test file and backup
	testFile := filepath.Join(tempHome, "test.md")
	if err := os.WriteFile(testFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	opts := Options{Platform: "claude-code"}
	metadata, err := CreateBackup(testFile, opts)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	backupPath := metadata.BackupPath

	// Delete backup
	if err := DeleteBackup(metadata.ID); err != nil {
		t.Fatalf("DeleteBackup failed: %v", err)
	}

	// Verify backup file is deleted
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Errorf("backup file still exists: %s", backupPath)
	}

	// Verify removed from index
	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	if _, exists := index.Backups[metadata.ID]; exists {
		t.Error("backup still exists in index")
	}
}

func TestVerifyBackup(t *testing.T) {
	// Setup temp environment
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	// Create test file and backup
	testFile := filepath.Join(tempHome, "test.md")
	content := "test content"
	if err := os.WriteFile(testFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	opts := Options{Platform: "claude-code"}
	metadata, err := CreateBackup(testFile, opts)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify intact backup
	if err := VerifyBackup(metadata.ID); err != nil {
		t.Errorf("VerifyBackup failed for intact backup: %v", err)
	}

	// Corrupt backup file
	if err := os.WriteFile(metadata.BackupPath, []byte("corrupted"), 0o600); err != nil {
		t.Fatalf("failed to corrupt backup file: %v", err)
	}

	// Verify should fail
	if err := VerifyBackup(metadata.ID); err == nil {
		t.Error("VerifyBackup should fail for corrupted backup")
	}
}

func TestCleanupBackups(t *testing.T) {
	// Setup temp environment
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	// Create test file
	testFile := filepath.Join(tempHome, "test.md")

	opts := Options{Platform: "claude-code"}

	// Create 5 backups with different content and timestamps
	for i := range 5 {
		content := fmt.Sprintf("test content version %d", i)
		if err := os.WriteFile(testFile, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		if _, err := CreateBackup(testFile, opts); err != nil {
			t.Fatalf("CreateBackup failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Verify 5 backups exist
	backups, err := ListBackups("claude-code")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	util.AssertEqual(t, len(backups), 5)

	// Cleanup keeping only 3 most recent
	cleanupOpts := CleanupOptions{
		MaxBackups:     3,
		KeepAtLeastOne: true,
		Platform:       "claude-code",
	}

	deleted, err := CleanupBackups(cleanupOpts)
	if err != nil {
		t.Fatalf("CleanupBackups failed: %v", err)
	}

	// Debug: print what we got
	if len(deleted) != 2 {
		t.Logf("Expected 2 deletions, got %d", len(deleted))
		allBackups, _ := ListBackups("claude-code")
		for i, b := range allBackups {
			t.Logf("  Backup %d: source=%s, created=%s", i, b.SourcePath, b.CreatedAt)
		}
	}

	util.AssertEqual(t, len(deleted), 2)

	// Verify only 3 backups remain
	backups, err = ListBackups("claude-code")
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	util.AssertEqual(t, len(backups), 3)
}

func TestDirectory(t *testing.T) {
	// Setup temp environment
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	// Create test directory with multiple files
	testDir := filepath.Join(tempHome, "skills")
	if err := os.MkdirAll(testDir, 0o750); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	files := []string{"skill1.md", "skill2.md", "skill3.json"}
	for _, file := range files {
		path := filepath.Join(testDir, file)
		if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Backup entire directory
	opts := Options{Platform: "claude-code"}
	backups, err := Directory(testDir, opts)
	if err != nil {
		t.Fatalf("Directory failed: %v", err)
	}

	util.AssertEqual(t, len(backups), 3)
}

func TestGetStats(t *testing.T) {
	// Setup temp environment
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	// Create test files and backups
	platforms := []string{"claude-code", "claude-code", "cursor"}
	for i, platform := range platforms {
		testFile := filepath.Join(tempHome, "test.md")
		content := fmt.Sprintf("test content %d", i)
		if err := os.WriteFile(testFile, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		opts := Options{Platform: platform}
		if _, err := CreateBackup(testFile, opts); err != nil {
			t.Fatalf("CreateBackup failed: %v", err)
		}
	}

	// Get stats
	stats, err := GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	util.AssertEqual(t, stats.TotalBackups, 3)
	util.AssertEqual(t, stats.BackupsByPlatform["claude-code"], 2)
	util.AssertEqual(t, stats.BackupsByPlatform["cursor"], 1)

	if stats.TotalSize == 0 {
		t.Error("expected non-zero total size")
	}
}

func TestGetBackupHistory(t *testing.T) {
	// Setup temp environment
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	// Create test file
	testFile := filepath.Join(tempHome, "test.md")

	// Create multiple backups of the same file
	opts := Options{Platform: "claude-code"}

	var backupIDs []string
	for i := range 3 {
		content := fmt.Sprintf("test content version %d", i)
		if err := os.WriteFile(testFile, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		metadata, err := CreateBackup(testFile, opts)
		if err != nil {
			t.Fatalf("CreateBackup failed: %v", err)
		}
		backupIDs = append(backupIDs, metadata.ID)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Get backup history for the file
	history, err := GetBackupHistory(testFile)
	if err != nil {
		t.Fatalf("GetBackupHistory failed: %v", err)
	}

	// Verify we got all 3 backups
	util.AssertEqual(t, len(history), 3)

	// Verify sorted by creation time (newest first)
	if history[0].ID != backupIDs[2] {
		t.Errorf("expected newest backup first, got %s, want %s", history[0].ID, backupIDs[2])
	}
	if history[2].ID != backupIDs[0] {
		t.Errorf("expected oldest backup last, got %s, want %s", history[2].ID, backupIDs[0])
	}

	// Verify all backups are for the same source path
	for _, b := range history {
		if b.SourcePath != testFile {
			t.Errorf("expected source path %s, got %s", testFile, b.SourcePath)
		}
	}

	// Test with non-existent file
	emptyHistory, err := GetBackupHistory("/nonexistent/file.md")
	if err != nil {
		t.Fatalf("GetBackupHistory failed for non-existent file: %v", err)
	}
	util.AssertEqual(t, len(emptyHistory), 0)
}
