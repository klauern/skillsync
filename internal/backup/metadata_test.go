package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/util"
)

func TestLoadIndex_EmptyIndex(t *testing.T) {
	// Setup temp environment with no existing index
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	util.AssertEqual(t, index.Version, IndexVersion)
	util.AssertEqual(t, len(index.Backups), 0)

	if index.Backups == nil {
		t.Error("expected non-nil Backups map")
	}
}

func TestLoadIndex_ExistingIndex(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	// Create metadata directory and index file
	metadataDir := filepath.Join(tempHome, "metadata")
	if err := os.MkdirAll(metadataDir, 0o750); err != nil {
		t.Fatalf("failed to create metadata dir: %v", err)
	}

	now := time.Now().Truncate(time.Second)
	existingIndex := Index{
		Version: "1.0",
		Updated: now,
		Backups: map[string]Metadata{
			"test-id-1": {
				ID:         "test-id-1",
				SourcePath: "/path/to/source.md",
				BackupPath: "/path/to/backup.md",
				Platform:   "claude-code",
				CreatedAt:  now,
				ModifiedAt: now,
				Hash:       "abc123def456",
				Size:       1024,
			},
		},
	}

	data, err := json.MarshalIndent(existingIndex, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal index: %v", err)
	}

	indexPath := filepath.Join(metadataDir, IndexFilename)
	// #nosec G306 - test file can have group-readable permissions
	if err := os.WriteFile(indexPath, data, 0o640); err != nil {
		t.Fatalf("failed to write index: %v", err)
	}

	// Load and verify
	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	util.AssertEqual(t, index.Version, "1.0")
	util.AssertEqual(t, len(index.Backups), 1)

	backup, exists := index.Backups["test-id-1"]
	if !exists {
		t.Fatal("expected backup test-id-1 to exist")
	}

	util.AssertEqual(t, backup.SourcePath, "/path/to/source.md")
	util.AssertEqual(t, backup.Platform, "claude-code")
	util.AssertEqual(t, backup.Size, int64(1024))
}

func TestLoadIndex_MalformedJSON(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	metadataDir := filepath.Join(tempHome, "metadata")
	if err := os.MkdirAll(metadataDir, 0o750); err != nil {
		t.Fatalf("failed to create metadata dir: %v", err)
	}

	indexPath := filepath.Join(metadataDir, IndexFilename)
	// #nosec G306 - test file can have group-readable permissions
	if err := os.WriteFile(indexPath, []byte("{invalid json"), 0o640); err != nil {
		t.Fatalf("failed to write malformed index: %v", err)
	}

	_, err := LoadIndex()
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestSaveIndex_CreatesDirectory(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index := &Index{
		Version: IndexVersion,
		Updated: time.Now(),
		Backups: make(map[string]Metadata),
	}

	if err := SaveIndex(index); err != nil {
		t.Fatalf("SaveIndex failed: %v", err)
	}

	metadataDir := filepath.Join(tempHome, "metadata")
	if _, err := os.Stat(metadataDir); os.IsNotExist(err) {
		t.Error("expected metadata directory to be created")
	}

	indexPath := filepath.Join(metadataDir, IndexFilename)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("expected index file to be created")
	}
}

func TestSaveIndex_UpdatesTimestamp(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	oldTime := time.Now().Add(-1 * time.Hour)
	index := &Index{
		Version: IndexVersion,
		Updated: oldTime,
		Backups: make(map[string]Metadata),
	}

	beforeSave := time.Now()
	if err := SaveIndex(index); err != nil {
		t.Fatalf("SaveIndex failed: %v", err)
	}

	// Updated timestamp should be set to current time
	if index.Updated.Before(beforeSave) {
		t.Error("expected Updated timestamp to be refreshed")
	}
}

func TestSaveIndex_Persistence(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	now := time.Now().Truncate(time.Second)
	index := &Index{
		Version: IndexVersion,
		Updated: now,
		Backups: map[string]Metadata{
			"backup-id": {
				ID:          "backup-id",
				SourcePath:  "/source/path.md",
				BackupPath:  "/backup/path.md",
				Platform:    "cursor",
				CreatedAt:   now,
				ModifiedAt:  now,
				Hash:        "hash123",
				Size:        512,
				Description: "test backup",
				Metadata:    map[string]string{"key": "value"},
				Tags:        []string{"tag1", "tag2"},
			},
		},
	}

	if err := SaveIndex(index); err != nil {
		t.Fatalf("SaveIndex failed: %v", err)
	}

	// Reload and verify all fields
	loaded, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	backup, exists := loaded.Backups["backup-id"]
	if !exists {
		t.Fatal("backup not found after reload")
	}

	util.AssertEqual(t, backup.ID, "backup-id")
	util.AssertEqual(t, backup.SourcePath, "/source/path.md")
	util.AssertEqual(t, backup.BackupPath, "/backup/path.md")
	util.AssertEqual(t, backup.Platform, "cursor")
	util.AssertEqual(t, backup.Hash, "hash123")
	util.AssertEqual(t, backup.Size, int64(512))
	util.AssertEqual(t, backup.Description, "test backup")

	if backup.Metadata["key"] != "value" {
		t.Errorf("expected metadata key=value, got %v", backup.Metadata)
	}

	if len(backup.Tags) != 2 || backup.Tags[0] != "tag1" || backup.Tags[1] != "tag2" {
		t.Errorf("expected tags [tag1 tag2], got %v", backup.Tags)
	}
}

func TestAddBackup_SingleBackup(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	metadata := Metadata{
		ID:         "new-backup",
		SourcePath: "/new/source.md",
		Platform:   "claude-code",
		CreatedAt:  time.Now(),
	}

	if err := index.AddBackup(metadata); err != nil {
		t.Fatalf("AddBackup failed: %v", err)
	}

	util.AssertEqual(t, len(index.Backups), 1)

	// Verify persisted
	reloaded, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	if _, exists := reloaded.Backups["new-backup"]; !exists {
		t.Error("backup not persisted")
	}
}

func TestAddBackup_MultipleBackups(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	backups := []Metadata{
		{ID: "backup-1", SourcePath: "/path/1.md", Platform: "claude-code", CreatedAt: time.Now()},
		{ID: "backup-2", SourcePath: "/path/2.md", Platform: "cursor", CreatedAt: time.Now()},
		{ID: "backup-3", SourcePath: "/path/3.md", Platform: "codex", CreatedAt: time.Now()},
	}

	for _, b := range backups {
		if err := index.AddBackup(b); err != nil {
			t.Fatalf("AddBackup failed: %v", err)
		}
	}

	util.AssertEqual(t, len(index.Backups), 3)

	// Verify all persisted
	reloaded, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	util.AssertEqual(t, len(reloaded.Backups), 3)
}

func TestAddBackup_OverwritesExisting(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	original := Metadata{
		ID:          "same-id",
		SourcePath:  "/original/path.md",
		Platform:    "claude-code",
		Description: "original",
		CreatedAt:   time.Now(),
	}

	updated := Metadata{
		ID:          "same-id",
		SourcePath:  "/updated/path.md",
		Platform:    "cursor",
		Description: "updated",
		CreatedAt:   time.Now(),
	}

	if err := index.AddBackup(original); err != nil {
		t.Fatalf("AddBackup failed: %v", err)
	}

	if err := index.AddBackup(updated); err != nil {
		t.Fatalf("AddBackup failed: %v", err)
	}

	util.AssertEqual(t, len(index.Backups), 1)

	backup := index.Backups["same-id"]
	util.AssertEqual(t, backup.SourcePath, "/updated/path.md")
	util.AssertEqual(t, backup.Platform, "cursor")
	util.AssertEqual(t, backup.Description, "updated")
}

func TestAddBackup_NilBackupsMap(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	// Create index with nil Backups map
	index := &Index{
		Version: IndexVersion,
		Updated: time.Now(),
		Backups: nil,
	}

	metadata := Metadata{
		ID:         "test-backup",
		SourcePath: "/test/path.md",
		Platform:   "claude-code",
		CreatedAt:  time.Now(),
	}

	if err := index.AddBackup(metadata); err != nil {
		t.Fatalf("AddBackup failed: %v", err)
	}

	if index.Backups == nil {
		t.Error("expected Backups map to be initialized")
	}

	util.AssertEqual(t, len(index.Backups), 1)
}

func TestRemoveBackup_ExistingBackup(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Add backup
	metadata := Metadata{
		ID:         "to-remove",
		SourcePath: "/path/to/remove.md",
		Platform:   "claude-code",
		CreatedAt:  time.Now(),
	}

	if err := index.AddBackup(metadata); err != nil {
		t.Fatalf("AddBackup failed: %v", err)
	}

	// Remove backup
	if err := index.RemoveBackup("to-remove"); err != nil {
		t.Fatalf("RemoveBackup failed: %v", err)
	}

	if _, exists := index.Backups["to-remove"]; exists {
		t.Error("backup should be removed from index")
	}

	// Verify persisted
	reloaded, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	if _, exists := reloaded.Backups["to-remove"]; exists {
		t.Error("backup removal not persisted")
	}
}

func TestRemoveBackup_NonExistent(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Remove non-existent backup should not error
	if err := index.RemoveBackup("does-not-exist"); err != nil {
		t.Fatalf("RemoveBackup failed for non-existent: %v", err)
	}
}

func TestRemoveBackup_PreservesOthers(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Add multiple backups
	backups := []Metadata{
		{ID: "keep-1", SourcePath: "/path/1.md", Platform: "claude-code", CreatedAt: time.Now()},
		{ID: "remove", SourcePath: "/path/2.md", Platform: "cursor", CreatedAt: time.Now()},
		{ID: "keep-2", SourcePath: "/path/3.md", Platform: "codex", CreatedAt: time.Now()},
	}

	for _, b := range backups {
		if err := index.AddBackup(b); err != nil {
			t.Fatalf("AddBackup failed: %v", err)
		}
	}

	// Remove one
	if err := index.RemoveBackup("remove"); err != nil {
		t.Fatalf("RemoveBackup failed: %v", err)
	}

	util.AssertEqual(t, len(index.Backups), 2)

	if _, exists := index.Backups["keep-1"]; !exists {
		t.Error("keep-1 should still exist")
	}
	if _, exists := index.Backups["keep-2"]; !exists {
		t.Error("keep-2 should still exist")
	}
}

func TestListBackups_SortsByNewestFirst(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	now := time.Now()
	index := &Index{
		Version: IndexVersion,
		Updated: now,
		Backups: map[string]Metadata{
			"oldest": {
				ID:        "oldest",
				CreatedAt: now.Add(-3 * time.Hour),
			},
			"middle": {
				ID:        "middle",
				CreatedAt: now.Add(-1 * time.Hour),
			},
			"newest": {
				ID:        "newest",
				CreatedAt: now,
			},
		},
	}

	backups := index.ListBackups()

	util.AssertEqual(t, len(backups), 3)
	util.AssertEqual(t, backups[0].ID, "newest")
	util.AssertEqual(t, backups[1].ID, "middle")
	util.AssertEqual(t, backups[2].ID, "oldest")
}

func TestListBackups_EmptyIndex(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index := &Index{
		Version: IndexVersion,
		Updated: time.Now(),
		Backups: make(map[string]Metadata),
	}

	backups := index.ListBackups()

	util.AssertEqual(t, len(backups), 0)
}

func TestListBackups_SingleBackup(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	now := time.Now()
	index := &Index{
		Version: IndexVersion,
		Updated: now,
		Backups: map[string]Metadata{
			"only-one": {
				ID:         "only-one",
				SourcePath: "/single/path.md",
				Platform:   "claude-code",
				CreatedAt:  now,
			},
		},
	}

	backups := index.ListBackups()

	util.AssertEqual(t, len(backups), 1)
	util.AssertEqual(t, backups[0].ID, "only-one")
}

func TestListBackups_DoesNotModifyIndex(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	now := time.Now()
	index := &Index{
		Version: IndexVersion,
		Updated: now,
		Backups: map[string]Metadata{
			"a": {ID: "a", CreatedAt: now.Add(-1 * time.Hour)},
			"b": {ID: "b", CreatedAt: now},
		},
	}

	originalLen := len(index.Backups)
	_ = index.ListBackups()

	util.AssertEqual(t, len(index.Backups), originalLen)

	// Backups map should still contain both entries
	if _, exists := index.Backups["a"]; !exists {
		t.Error("backup 'a' should still exist in map")
	}
	if _, exists := index.Backups["b"]; !exists {
		t.Error("backup 'b' should still exist in map")
	}
}

func TestMetadata_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := Metadata{
		ID:          "test-id",
		SourcePath:  "/source/path.md",
		BackupPath:  "/backup/path.md",
		Platform:    "claude-code",
		CreatedAt:   now,
		ModifiedAt:  now.Add(-1 * time.Hour),
		Hash:        "abc123def456789",
		Size:        2048,
		Description: "Test description",
		Metadata:    map[string]string{"key1": "value1", "key2": "value2"},
		Tags:        []string{"tag1", "tag2", "tag3"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored Metadata
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	util.AssertEqual(t, restored.ID, original.ID)
	util.AssertEqual(t, restored.SourcePath, original.SourcePath)
	util.AssertEqual(t, restored.BackupPath, original.BackupPath)
	util.AssertEqual(t, restored.Platform, original.Platform)
	util.AssertEqual(t, restored.Hash, original.Hash)
	util.AssertEqual(t, restored.Size, original.Size)
	util.AssertEqual(t, restored.Description, original.Description)

	// Compare times with tolerance for JSON marshaling
	if !restored.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", restored.CreatedAt, original.CreatedAt)
	}
	if !restored.ModifiedAt.Equal(original.ModifiedAt) {
		t.Errorf("ModifiedAt mismatch: got %v, want %v", restored.ModifiedAt, original.ModifiedAt)
	}

	// Check metadata map
	if len(restored.Metadata) != len(original.Metadata) {
		t.Errorf("Metadata map length mismatch: got %d, want %d", len(restored.Metadata), len(original.Metadata))
	}
	for k, v := range original.Metadata {
		if restored.Metadata[k] != v {
			t.Errorf("Metadata[%s] mismatch: got %s, want %s", k, restored.Metadata[k], v)
		}
	}

	// Check tags
	if len(restored.Tags) != len(original.Tags) {
		t.Errorf("Tags length mismatch: got %d, want %d", len(restored.Tags), len(original.Tags))
	}
	for i, tag := range original.Tags {
		if restored.Tags[i] != tag {
			t.Errorf("Tags[%d] mismatch: got %s, want %s", i, restored.Tags[i], tag)
		}
	}
}

func TestMetadata_OmitemptyFields(t *testing.T) {
	metadata := Metadata{
		ID:         "minimal",
		SourcePath: "/path.md",
		Platform:   "claude-code",
		CreatedAt:  time.Now(),
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)

	// These fields should be omitted when empty
	if contains(jsonStr, "description") {
		t.Error("description should be omitted when empty")
	}
	if contains(jsonStr, "metadata") {
		t.Error("metadata should be omitted when nil")
	}
	if contains(jsonStr, "tags") {
		t.Error("tags should be omitted when nil")
	}
}

func TestIndex_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := Index{
		Version: "1.0",
		Updated: now,
		Backups: map[string]Metadata{
			"backup-1": {
				ID:         "backup-1",
				SourcePath: "/path/1.md",
				Platform:   "claude-code",
				CreatedAt:  now,
			},
			"backup-2": {
				ID:         "backup-2",
				SourcePath: "/path/2.md",
				Platform:   "cursor",
				CreatedAt:  now.Add(-1 * time.Hour),
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored Index
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	util.AssertEqual(t, restored.Version, original.Version)
	util.AssertEqual(t, len(restored.Backups), len(original.Backups))

	for id, origBackup := range original.Backups {
		restoredBackup, exists := restored.Backups[id]
		if !exists {
			t.Errorf("backup %s not found after round trip", id)
			continue
		}
		util.AssertEqual(t, restoredBackup.SourcePath, origBackup.SourcePath)
		util.AssertEqual(t, restoredBackup.Platform, origBackup.Platform)
	}
}

func TestIndex_EmptyBackupsRoundTrip(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	index := &Index{
		Version: IndexVersion,
		Updated: time.Now(),
		Backups: make(map[string]Metadata),
	}

	if err := SaveIndex(index); err != nil {
		t.Fatalf("SaveIndex failed: %v", err)
	}

	loaded, err := LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	util.AssertEqual(t, loaded.Version, IndexVersion)
	util.AssertEqual(t, len(loaded.Backups), 0)

	if loaded.Backups == nil {
		t.Error("expected non-nil Backups map after loading empty index")
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
