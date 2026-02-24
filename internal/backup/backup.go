// Package backup provides automatic backup functionality for skill directories
package backup

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauern/skillsync/internal/util"
)

const (
	// BackupDirPerm is the permission for backup directories (rwxr-x---)
	BackupDirPerm = 0o750
	// BackupFilePerm is the permission for backup files (rw-r-----)
	BackupFilePerm = 0o640
	// maxArchiveEntryBytes limits restored zip entry size to avoid resource exhaustion.
	maxArchiveEntryBytes = 50 * 1024 * 1024 // 50 MiB
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

	var (
		content       []byte
		readErr       error
		backupExt     string
		metadataSize  int64
		metadataMTime = sourceInfo.ModTime()
	)
	if sourceInfo.IsDir() {
		content, readErr = createDirectoryArchive(sourcePath)
		backupExt = ".zip"
	} else {
		// #nosec G304 - sourcePath is controlled by the caller and validated
		content, readErr = os.ReadFile(sourcePath)
		backupExt = filepath.Ext(sourcePath)
	}
	if readErr != nil {
		return nil, fmt.Errorf("failed to read source path %q: %w", sourcePath, readErr)
	}
	metadataSize = int64(len(content))

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

	// Determine backup filename (preserve file extension or use .zip for directory archives)
	backupFilename := backupID + backupExt
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
		ModifiedAt:  metadataMTime,
		Hash:        hashStr,
		Size:        metadataSize,
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

	if filepath.Ext(metadata.BackupPath) == ".zip" {
		if err := restoreDirectoryArchive(content, targetPath); err != nil {
			return fmt.Errorf("failed to restore directory backup: %w", err)
		}
		return nil
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

func createDirectoryArchive(sourcePath string) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return fmt.Errorf("failed to derive relative path for %q: %w", path, err)
		}
		relPath = filepath.ToSlash(relPath)

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("failed to create zip header for %q: %w", path, err)
		}
		header.Name = relPath
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("failed to add %q to archive: %w", path, err)
		}

		// #nosec G304 - path comes from filepath.Walk under trusted sourcePath
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open %q for archive: %w", path, err)
		}

		if _, err := io.Copy(writer, file); err != nil {
			_ = file.Close()
			return fmt.Errorf("failed to write %q to archive: %w", path, err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("failed to close %q while archiving: %w", path, err)
		}
		return nil
	})
	if err != nil {
		_ = zipWriter.Close()
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize archive: %w", err)
	}
	return buf.Bytes(), nil
}

func restoreDirectoryArchive(archive []byte, targetPath string) error {
	if err := os.MkdirAll(targetPath, BackupDirPerm); err != nil {
		return fmt.Errorf("failed to create restore directory: %w", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return fmt.Errorf("failed to read zip archive: %w", err)
	}

	cleanTarget := filepath.Clean(targetPath)
	absTarget, err := filepath.Abs(cleanTarget)
	if err != nil {
		return fmt.Errorf("failed to resolve restore directory: %w", err)
	}
	for _, file := range reader.File {
		cleanName := filepath.Clean(file.Name)
		outPath := filepath.Join(cleanTarget, cleanName)
		absOut, err := filepath.Abs(outPath)
		if err != nil {
			return fmt.Errorf("failed to resolve archived path %q: %w", file.Name, err)
		}
		relPath, err := filepath.Rel(absTarget, absOut)
		if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("invalid archive path %q", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(outPath, BackupDirPerm); err != nil {
				return fmt.Errorf("failed to create restored directory %q: %w", outPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(outPath), BackupDirPerm); err != nil {
			return fmt.Errorf("failed to create restored parent directory for %q: %w", outPath, err)
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open archived file %q: %w", file.Name, err)
		}

		// #nosec G304 -- outPath is validated to remain under cleanTarget above
		outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, BackupFilePerm)
		if err != nil {
			_ = rc.Close()
			return fmt.Errorf("failed to create restored file %q: %w", outPath, err)
		}

		data, err := io.ReadAll(io.LimitReader(rc, maxArchiveEntryBytes+1))
		if err != nil {
			_ = outFile.Close()
			_ = rc.Close()
			return fmt.Errorf("failed to read archived file %q: %w", file.Name, err)
		}
		if len(data) > maxArchiveEntryBytes {
			_ = outFile.Close()
			_ = rc.Close()
			return fmt.Errorf("archived file %q exceeds max restore size of %d bytes", file.Name, maxArchiveEntryBytes)
		}
		if _, err := outFile.Write(data); err != nil {
			_ = outFile.Close()
			_ = rc.Close()
			return fmt.Errorf("failed to restore file %q: %w", outPath, err)
		}
		if err := outFile.Close(); err != nil {
			_ = rc.Close()
			return fmt.Errorf("failed to close restored file %q: %w", outPath, err)
		}
		if err := rc.Close(); err != nil {
			return fmt.Errorf("failed to close archived file %q: %w", file.Name, err)
		}
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
