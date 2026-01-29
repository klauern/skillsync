// Package backup provides automatic backup functionality for skill directories
package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/klauern/skillsync/internal/util"
)

const (
	// BackupDirPerm is the permission for backup directories (rwxr-x---)
	BackupDirPerm = 0o750
	// BackupFilePerm is the permission for backup files (rw-r-----)
	BackupFilePerm = 0o640
)

// Options configures backup behavior
type Options struct {
	Platform    string            // Platform identifier (claude-code, cursor, codex)
	Description string            // Human-readable description
	Metadata    map[string]string // Additional metadata
	Tags        []string          // Tags for categorization
}

// CreateBackup creates a backup of the specified file or directory
func CreateBackup(sourcePath string, opts Options) (*Metadata, error) {
	// Ensure backups directory exists
	backupsDir := util.SkillsyncBackupsPath()
	if err := os.MkdirAll(backupsDir, BackupDirPerm); err != nil {
		return nil, fmt.Errorf("failed to create backups directory: %w", err)
	}

	// Get source file info
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat source path %q: %w", sourcePath, err)
	}

	// Read source file
	// #nosec G304 - sourcePath is controlled by the caller and validated
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file %q: %w", sourcePath, err)
	}

	// Generate hash
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	// Generate backup ID (timestamp-based)
	backupID := time.Now().Format("20060102-150405-") + hashStr[:8]

	// Create platform-specific backup directory
	platformDir := filepath.Join(backupsDir, opts.Platform)
	if err := os.MkdirAll(platformDir, BackupDirPerm); err != nil {
		return nil, fmt.Errorf("failed to create platform backup directory: %w", err)
	}

	// Determine backup filename (preserve extension)
	backupFilename := backupID + filepath.Ext(sourcePath)
	backupPath := filepath.Join(platformDir, backupFilename)

	// Write backup file
	if err := os.WriteFile(backupPath, content, BackupFilePerm); err != nil {
		return nil, fmt.Errorf("failed to write backup file: %w", err)
	}

	// Create metadata
	metadata := &Metadata{
		ID:          backupID,
		SourcePath:  sourcePath,
		BackupPath:  backupPath,
		Platform:    opts.Platform,
		CreatedAt:   time.Now(),
		ModifiedAt:  sourceInfo.ModTime(),
		Hash:        hashStr,
		Size:        sourceInfo.Size(),
		Description: opts.Description,
		Metadata:    opts.Metadata,
		Tags:        opts.Tags,
	}

	// Load index and add backup
	index, err := LoadIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to load backup index: %w", err)
	}

	if err := index.AddBackup(*metadata); err != nil {
		return nil, fmt.Errorf("failed to add backup to index: %w", err)
	}

	return metadata, nil
}

// RestoreBackup restores a backup to the specified target path
func RestoreBackup(backupID string, targetPath string) error {
	// Load index
	index, err := LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load backup index: %w", err)
	}

	// Find backup
	metadata, exists := index.Backups[backupID]
	if !exists {
		return fmt.Errorf("backup %q not found", backupID)
	}

	// Read backup file
	content, err := os.ReadFile(metadata.BackupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Verify hash
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])
	if hashStr != metadata.Hash {
		return fmt.Errorf("backup file corrupted: hash mismatch")
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, BackupDirPerm); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Write target file
	if err := os.WriteFile(targetPath, content, BackupFilePerm); err != nil {
		return fmt.Errorf("failed to write target file: %w", err)
	}

	return nil
}

// ListBackups returns all backups, optionally filtered by platform
func ListBackups(platform string) ([]Metadata, error) {
	index, err := LoadIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to load backup index: %w", err)
	}

	backups := index.ListBackups()

	// Filter by platform if specified
	if platform != "" {
		filtered := make([]Metadata, 0)
		for _, backup := range backups {
			if backup.Platform == platform {
				filtered = append(filtered, backup)
			}
		}
		return filtered, nil
	}

	return backups, nil
}

// DeleteBackup deletes a backup and removes it from the index
func DeleteBackup(backupID string) error {
	// Load index
	index, err := LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load backup index: %w", err)
	}

	// Find backup
	metadata, exists := index.Backups[backupID]
	if !exists {
		return fmt.Errorf("backup %q not found", backupID)
	}

	// Delete backup file
	if err := os.Remove(metadata.BackupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete backup file: %w", err)
	}

	// Remove from index
	if err := index.RemoveBackup(backupID); err != nil {
		return fmt.Errorf("failed to remove backup from index: %w", err)
	}

	return nil
}

// Directory creates backups of all files in a directory
func Directory(sourcePath string, opts Options) ([]Metadata, error) {
	var backups []Metadata

	// Walk directory
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Create backup for each file
		metadata, err := CreateBackup(path, opts)
		if err != nil {
			return fmt.Errorf("failed to backup %q: %w", path, err)
		}

		backups = append(backups, *metadata)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to backup directory: %w", err)
	}

	return backups, nil
}

// VerifyBackup verifies that a backup file is intact and matches its hash
func VerifyBackup(backupID string) error {
	// Load index
	index, err := LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load backup index: %w", err)
	}

	// Find backup
	metadata, exists := index.Backups[backupID]
	if !exists {
		return fmt.Errorf("backup %q not found", backupID)
	}

	// Check if file exists
	if _, err := os.Stat(metadata.BackupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file missing: %s", metadata.BackupPath)
	}

	// Read and hash file
	file, err := os.Open(metadata.BackupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close backup file: %w", closeErr)
		}
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	hashStr := hex.EncodeToString(hash.Sum(nil))
	if hashStr != metadata.Hash {
		return fmt.Errorf("backup file corrupted: hash mismatch (expected %s, got %s)", metadata.Hash, hashStr)
	}

	return nil
}

// GetBackupHistory returns all backups for a specific source file, sorted by creation time (newest first)
func GetBackupHistory(sourcePath string) ([]Metadata, error) {
	index, err := LoadIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to load backup index: %w", err)
	}

	// Filter backups by source path
	var history []Metadata
	for _, backup := range index.Backups {
		if backup.SourcePath == sourcePath {
			history = append(history, backup)
		}
	}

	// Sort by creation time (newest first)
	for i := 0; i < len(history)-1; i++ {
		for j := i + 1; j < len(history); j++ {
			if history[i].CreatedAt.Before(history[j].CreatedAt) {
				history[i], history[j] = history[j], history[i]
			}
		}
	}

	return history, nil
}
