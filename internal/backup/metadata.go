// Package backup provides automatic backup functionality for skill directories
package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/klauern/skillsync/internal/util"
)

// Metadata contains metadata about a single backup
type Metadata struct {
	ID          string            `json:"id"`          // Unique backup identifier (timestamp-based)
	SourcePath  string            `json:"source_path"` // Original file/directory path
	BackupPath  string            `json:"backup_path"` // Path to backup file
	Platform    string            `json:"platform"`    // Platform (claude-code, cursor, codex)
	CreatedAt   time.Time         `json:"created_at"`  // Backup creation timestamp
	ModifiedAt  time.Time         `json:"modified_at"` // Source modification timestamp
	Hash        string            `json:"hash"`        // SHA256 hash of content
	Size        int64             `json:"size"`        // File size in bytes
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"` // Additional metadata
	Tags        []string          `json:"tags,omitempty"`
}

// Index maintains an index of all backups
type Index struct {
	Version string              `json:"version"`
	Updated time.Time           `json:"updated"`
	Backups map[string]Metadata `json:"backups"` // Key: backup ID
}

const (
	// IndexVersion is the current version of the backup index format
	IndexVersion = "1.0"
	// IndexFilename is the name of the index file
	IndexFilename = "index.json"
)

// LoadIndex loads the backup index from disk
func LoadIndex() (*Index, error) {
	indexPath := filepath.Join(util.SkillsyncMetadataPath(), IndexFilename)

	// If index doesn't exist, return empty index
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return &Index{
			Version: IndexVersion,
			Updated: time.Now(),
			Backups: make(map[string]Metadata),
		}, nil
	}

	// #nosec G304 - indexPath is constructed from trusted util.SkillsyncMetadataPath()
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse index file: %w", err)
	}

	return &index, nil
}

// SaveIndex saves the backup index to disk
func SaveIndex(index *Index) error {
	metadataDir := util.SkillsyncMetadataPath()

	// Ensure metadata directory exists
	if err := os.MkdirAll(metadataDir, 0o750); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	index.Updated = time.Now()

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	indexPath := filepath.Join(metadataDir, IndexFilename)
	// #nosec G306 - index.json is metadata and can be group-readable
	if err := os.WriteFile(indexPath, data, 0o640); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

// AddBackup adds a backup entry to the index and saves it
func (idx *Index) AddBackup(metadata Metadata) error {
	if idx.Backups == nil {
		idx.Backups = make(map[string]Metadata)
	}

	idx.Backups[metadata.ID] = metadata

	return SaveIndex(idx)
}

// RemoveBackup removes a backup entry from the index and saves it
func (idx *Index) RemoveBackup(id string) error {
	delete(idx.Backups, id)
	return SaveIndex(idx)
}

// ListBackups returns all backups sorted by creation time (newest first)
func (idx *Index) ListBackups() []Metadata {
	backups := make([]Metadata, 0, len(idx.Backups))
	for _, backup := range idx.Backups {
		backups = append(backups, backup)
	}

	// Sort by creation time (newest first)
	for i := 0; i < len(backups)-1; i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].CreatedAt.Before(backups[j].CreatedAt) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	return backups
}
