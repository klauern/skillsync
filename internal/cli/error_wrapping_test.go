package cli

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/sync"
	"github.com/klauern/skillsync/internal/ui/tui"
)

func TestExecuteDeleteWrapsUnderlyingErrors(t *testing.T) {
	err := executeDelete(tui.DeleteListResult{
		SelectedSkills: []model.Skill{
			{
				Name:  "missing-skill",
				Path:  filepath.Join(t.TempDir(), "missing", "SKILL.md"),
				Scope: model.ScopeUser,
			},
		},
	})
	if err == nil {
		t.Fatal("executeDelete() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "some deletions failed") {
		t.Fatalf("executeDelete() error = %v, want summary context", err)
	}
	if !strings.Contains(err.Error(), "missing-skill") {
		t.Fatalf("executeDelete() error = %v, want skill context", err)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("executeDelete() error = %v, want wrapped fs.ErrNotExist", err)
	}
}

func TestExecutePromoteDemoteWrapsUnderlyingErrors(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	err := executePromoteDemote(tui.PromoteDemoteListResult{
		Action: tui.PromoteDemoteActionDemote,
		SelectedSkills: []model.Skill{
			{
				Name:     "missing-skill",
				Path:     filepath.Join(t.TempDir(), "missing", "SKILL.md"),
				Scope:    model.ScopeUser,
				Platform: model.Cursor,
			},
		},
	})
	if err == nil {
		t.Fatal("executePromoteDemote() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "some operations failed") {
		t.Fatalf("executePromoteDemote() error = %v, want summary context", err)
	}
	if !strings.Contains(err.Error(), "missing-skill") {
		t.Fatalf("executePromoteDemote() error = %v, want skill context", err)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("executePromoteDemote() error = %v, want wrapped fs.ErrNotExist", err)
	}
}

func TestSummarizeSyncFailuresWrapsSkillErrors(t *testing.T) {
	want := fs.ErrPermission
	result := &sync.Result{
		Skills: []sync.SkillResult{
			{
				Skill:  model.Skill{Name: "alpha"},
				Action: sync.ActionFailed,
				Error:  want,
			},
			{
				Skill:  model.Skill{Name: "beta"},
				Action: sync.ActionFailed,
			},
		},
	}

	err := summarizeSyncFailures(result, "sync completed with errors")
	if err == nil {
		t.Fatal("summarizeSyncFailures() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "sync completed with errors") {
		t.Fatalf("summarizeSyncFailures() error = %v, want summary context", err)
	}
	if !strings.Contains(err.Error(), "alpha") || !strings.Contains(err.Error(), "beta") {
		t.Fatalf("summarizeSyncFailures() error = %v, want per-skill context", err)
	}
	if !errors.Is(err, want) {
		t.Fatalf("summarizeSyncFailures() error = %v, want wrapped permission error", err)
	}
}
