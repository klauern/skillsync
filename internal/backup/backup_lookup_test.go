package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/util"
)

func TestLoadBackupMetadata(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	testFile := filepath.Join(tempHome, "test.md")
	if err := os.WriteFile(testFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	created, err := CreateBackup(testFile, Options{Platform: "claude-code"})
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	index, metadata, err := loadBackupMetadata(created.ID)
	if err != nil {
		t.Fatalf("loadBackupMetadata failed: %v", err)
	}

	if metadata.ID != created.ID {
		t.Fatalf("loadBackupMetadata returned %q, want %q", metadata.ID, created.ID)
	}

	if _, exists := index.Backups[created.ID]; !exists {
		t.Fatalf("loadBackupMetadata returned index missing %q", created.ID)
	}
}

func TestLoadBackupMetadata_NotFound(t *testing.T) {
	tempHome := util.CreateTempDir(t)
	t.Setenv("SKILLSYNC_HOME", tempHome)

	_, _, err := loadBackupMetadata("missing-backup")
	if err == nil {
		t.Fatal("expected loadBackupMetadata to fail for missing backup")
	}

	expected := fmt.Sprintf("backup %q not found", "missing-backup")
	if err.Error() != expected {
		t.Fatalf("loadBackupMetadata error = %q, want %q", err.Error(), expected)
	}
}
