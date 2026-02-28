package backup

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

func TestCreateBackup_DirectoryAndRestore(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	sourceDir := filepath.Join(tempHome, "source-skill")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(filepath.Join(sourceDir, "scripts"), 0o755); err != nil {
		t.Fatalf("failed to create scripts dir: %v", err)
	}
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(filepath.Join(sourceDir, "references"), 0o755); err != nil {
		t.Fatalf("failed to create references dir: %v", err)
	}

	files := map[string]string{
		filepath.Join(sourceDir, "SKILL.md"):                    "# Skill\n\ncontent",
		filepath.Join(sourceDir, "scripts", "setup.sh"):         "#!/bin/sh\necho setup",
		filepath.Join(sourceDir, "references", "guide.md"):      "# guide",
		filepath.Join(sourceDir, "references", "deep", "x.txt"): "ignored",
	}
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(filepath.Join(sourceDir, "references", "deep"), 0o755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	for path, content := range files {
		// #nosec G306 - test file permissions
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write test file %q: %v", path, err)
		}
	}

	metadata, err := CreateBackup(sourceDir, Options{Platform: "claude-code"})
	if err != nil {
		t.Fatalf("CreateBackup for directory failed: %v", err)
	}
	if filepath.Ext(metadata.BackupPath) != ".zip" {
		t.Fatalf("expected directory backup to be a .zip archive, got %q", metadata.BackupPath)
	}

	restoreDir := filepath.Join(tempHome, "restored-skill")
	if err := RestoreBackup(metadata.ID, restoreDir); err != nil {
		t.Fatalf("RestoreBackup for directory failed: %v", err)
	}

	for path, wantContent := range files {
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			t.Fatalf("failed to create relative path: %v", err)
		}
		restoredPath := filepath.Join(restoreDir, rel)
		// #nosec G304 - restoredPath is test-controlled
		gotBytes, err := os.ReadFile(restoredPath)
		if err != nil {
			t.Fatalf("failed to read restored file %q: %v", restoredPath, err)
		}
		if string(gotBytes) != wantContent {
			t.Errorf("restored content mismatch for %q: got %q want %q", rel, string(gotBytes), wantContent)
		}
	}
}

func TestCreateBackup_SkillFileBacksUpDirectory(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	sourceDir := filepath.Join(tempHome, "my-skill")
	// #nosec G301 - test directory permissions
	for _, sub := range []string{"scripts", "references", "assets"} {
		if err := os.MkdirAll(filepath.Join(sourceDir, sub), 0o755); err != nil {
			t.Fatalf("failed to create %s dir: %v", sub, err)
		}
	}
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(filepath.Join(sourceDir, "references", "deep"), 0o755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	files := map[string]string{
		filepath.Join(sourceDir, "SKILL.md"):                    "# Skill entrypoint\n\ncontent",
		filepath.Join(sourceDir, "scripts", "setup.sh"):         "#!/bin/sh\necho setup",
		filepath.Join(sourceDir, "references", "guide.md"):      "# guide",
		filepath.Join(sourceDir, "references", "deep", "x.txt"): "deep content",
		filepath.Join(sourceDir, "assets", "config.yaml"):       "key: value",
	}
	for path, content := range files {
		// #nosec G306 - test file permissions
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write test file %q: %v", path, err)
		}
	}

	// Pass SKILL.md path (as CLI does via skill.Path) - should backup whole dir
	skillFilePath := filepath.Join(sourceDir, "SKILL.md")
	metadata, err := CreateBackup(skillFilePath, Options{Platform: "claude-code"})
	if err != nil {
		t.Fatalf("CreateBackup for SKILL.md file failed: %v", err)
	}
	if filepath.Ext(metadata.BackupPath) != ".zip" {
		t.Fatalf("expected directory backup as .zip, got %q", metadata.BackupPath)
	}
	if metadata.SourcePath != sourceDir {
		t.Errorf("expected SourcePath to be skill dir %q, got %q", sourceDir, metadata.SourcePath)
	}

	restoreDir := filepath.Join(tempHome, "restored-skill")
	if err := RestoreBackup(metadata.ID, restoreDir); err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}

	for path, wantContent := range files {
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			t.Fatalf("failed to create relative path: %v", err)
		}
		restoredPath := filepath.Join(restoreDir, rel)
		// #nosec G304 - restoredPath is test-controlled
		gotBytes, err := os.ReadFile(restoredPath)
		if err != nil {
			t.Fatalf("failed to read restored file %q: %v", restoredPath, err)
		}
		if string(gotBytes) != wantContent {
			t.Errorf("restored content mismatch for %q: got %q want %q", rel, string(gotBytes), wantContent)
		}
	}
}

func TestRestoreBackup_LegacyFileBackup(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	flatFile := filepath.Join(tempHome, "flat-skill.md")
	content := "# Flat skill\n\nsingle file"
	if err := os.WriteFile(flatFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to create flat file: %v", err)
	}

	metadata, err := CreateBackup(flatFile, Options{Platform: "claude-code"})
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}
	if filepath.Ext(metadata.BackupPath) != ".md" {
		t.Errorf("expected legacy file backup extension .md, got %q", metadata.BackupPath)
	}

	restoreFile := filepath.Join(tempHome, "restored-flat.md")
	if err := RestoreBackup(metadata.ID, restoreFile); err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}
	// #nosec G304 - restoreFile is test-controlled
	got, err := os.ReadFile(restoreFile)
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}
	if string(got) != content {
		t.Errorf("restored content mismatch: got %q want %q", string(got), content)
	}
}

func TestRestoreBackup_DirectoryRejectsPathTraversal(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	// Build a malicious zip containing a path traversal entry.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("../escape.txt")
	if err != nil {
		t.Fatalf("failed to create malicious zip entry: %v", err)
	}
	if _, err := w.Write([]byte("escape")); err != nil {
		t.Fatalf("failed to write malicious zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	zipBytes := buf.Bytes()
	hash := sha256.Sum256(zipBytes)
	hashStr := hex.EncodeToString(hash[:])

	backupPath := filepath.Join(util.SkillsyncBackupsPath(), "claude-code", "malicious.zip")
	// #nosec G301 - test directory permissions
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}
	// #nosec G306 - test file permissions
	if err := os.WriteFile(backupPath, zipBytes, 0o644); err != nil {
		t.Fatalf("failed to write malicious backup: %v", err)
	}

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("failed to load index: %v", err)
	}
	meta := Metadata{
		ID:         "malicious-backup",
		SourcePath: filepath.Join(tempHome, "source-dir"),
		BackupPath: backupPath,
		Platform:   "claude-code",
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
		Hash:       hashStr,
		Size:       int64(len(zipBytes)),
	}
	if err := index.AddBackup(meta); err != nil {
		t.Fatalf("failed to add malicious backup metadata: %v", err)
	}

	restoreDir := filepath.Join(tempHome, "restore-target")
	err = RestoreBackup(meta.ID, restoreDir)
	if err == nil {
		t.Fatal("expected RestoreBackup to reject path traversal archive entry")
	}
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

	// Create test directory with multiple files (no SKILL.md - orphan files)
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

func TestDirectory_SkillDirectoriesProduceZipBackups(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	skillsRoot := filepath.Join(tempHome, "skills")
	skillA := filepath.Join(skillsRoot, "skill-a")
	skillB := filepath.Join(skillsRoot, "skill-b")
	// #nosec G301 - test directory permissions
	for _, d := range []string{filepath.Join(skillA, "scripts"), filepath.Join(skillB, "references")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}
	files := map[string]string{
		filepath.Join(skillA, "SKILL.md"):             "# Skill A",
		filepath.Join(skillA, "scripts", "init.sh"):   "#!/bin/sh",
		filepath.Join(skillB, "SKILL.md"):             "# Skill B",
		filepath.Join(skillB, "references", "doc.md"): "reference",
	}
	for path, content := range files {
		// #nosec G306 - test file permissions
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write %q: %v", path, err)
		}
	}

	opts := Options{Platform: "claude-code"}
	backups, err := Directory(skillsRoot, opts)
	if err != nil {
		t.Fatalf("Directory failed: %v", err)
	}
	util.AssertEqual(t, len(backups), 2)

	zipCount := 0
	for _, b := range backups {
		if filepath.Ext(b.BackupPath) == ".zip" {
			zipCount++
		}
	}
	util.AssertEqual(t, zipCount, 2)

	// Restore one backup and verify directory contents preserved
	restoreDir := filepath.Join(tempHome, "restored")
	if err := RestoreBackup(backups[0].ID, restoreDir); err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}
	skillMD := filepath.Join(restoreDir, "SKILL.md")
	if _, err := os.Stat(skillMD); err != nil {
		t.Fatalf("restored SKILL.md missing: %v", err)
	}
	// Each skill has either scripts/init.sh (skill-a) or references/doc.md (skill-b)
	hasScripts := func() bool {
		_, err := os.Stat(filepath.Join(restoreDir, "scripts", "init.sh"))
		return err == nil
	}
	hasRefs := func() bool {
		_, err := os.Stat(filepath.Join(restoreDir, "references", "doc.md"))
		return err == nil
	}
	if !hasScripts() && !hasRefs() {
		t.Error("restored skill dir should contain scripts/init.sh or references/doc.md")
	}
}

func TestDirectory_DeduplicatesSkillEntrypointVariants(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	skillsRoot := filepath.Join(tempHome, "skills")
	skillDir := filepath.Join(skillsRoot, "skill-a")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Upper"), 0o600); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("# Lower"), 0o600); err != nil {
		t.Fatalf("failed to write skill.md: %v", err)
	}

	upperInfo, upperErr := os.Stat(filepath.Join(skillDir, "SKILL.md"))
	lowerInfo, lowerErr := os.Stat(filepath.Join(skillDir, "skill.md"))
	if upperErr == nil && lowerErr == nil && os.SameFile(upperInfo, lowerInfo) {
		t.Skip("filesystem is case-insensitive; cannot create distinct SKILL.md and skill.md")
	}

	backups, err := Directory(skillsRoot, Options{Platform: "claude-code"})
	if err != nil {
		t.Fatalf("Directory failed: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("expected one backup for a single skill directory, got %d", len(backups))
	}
	if filepath.Ext(backups[0].BackupPath) != ".zip" {
		t.Fatalf("expected zip backup for skill directory, got %q", backups[0].BackupPath)
	}
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
