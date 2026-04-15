package cli

import (
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestFindOrphanedSkills(t *testing.T) {
	tests := map[string]struct {
		sourceSkills []model.Skill
		targetSkills []model.Skill
		wantOrphans  int
		wantNames    []string
	}{
		"no orphans when target subset of source": {
			sourceSkills: []model.Skill{
				{Name: "skill-a"},
				{Name: "skill-b"},
			},
			targetSkills: []model.Skill{
				{Name: "skill-a"},
			},
			wantOrphans: 0,
		},
		"finds orphans in target not in source": {
			sourceSkills: []model.Skill{
				{Name: "skill-a"},
			},
			targetSkills: []model.Skill{
				{Name: "skill-a"},
				{Name: "skill-b"},
				{Name: "skill-c"},
			},
			wantOrphans: 2,
			wantNames:   []string{"skill-b", "skill-c"},
		},
		"empty source means all targets are orphans": {
			sourceSkills: []model.Skill{},
			targetSkills: []model.Skill{
				{Name: "skill-a"},
			},
			wantOrphans: 1,
			wantNames:   []string{"skill-a"},
		},
		"empty target means no orphans": {
			sourceSkills: []model.Skill{
				{Name: "skill-a"},
			},
			targetSkills: nil,
			wantOrphans:  0,
		},
		"no orphans when sets are equal": {
			sourceSkills: []model.Skill{
				{Name: "skill-a"},
				{Name: "skill-b"},
			},
			targetSkills: []model.Skill{
				{Name: "skill-a"},
				{Name: "skill-b"},
			},
			wantOrphans: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			orphans := findOrphanedSkills(tt.sourceSkills, tt.targetSkills)
			if len(orphans) != tt.wantOrphans {
				t.Errorf("got %d orphans, want %d", len(orphans), tt.wantOrphans)
			}
			if tt.wantNames != nil {
				gotNames := make(map[string]bool)
				for _, s := range orphans {
					gotNames[s.Name] = true
				}
				for _, name := range tt.wantNames {
					if !gotNames[name] {
						t.Errorf("expected orphan %q not found", name)
					}
				}
			}
		})
	}
}

func TestDeleteOrphansPrompt(t *testing.T) {
	// Test that findOrphanedSkills correctly identifies orphans
	// when used in the sync flow context
	source := []model.Skill{
		{Name: "keep-this", Platform: model.ClaudeCode},
	}
	target := []model.Skill{
		{Name: "keep-this", Platform: model.Cursor},
		{Name: "remove-this", Platform: model.Cursor, Scope: model.ScopeUser, Path: "/tmp/test"},
	}

	orphans := findOrphanedSkills(source, target)
	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(orphans))
	}
	if orphans[0].Name != "remove-this" {
		t.Errorf("expected orphan name %q, got %q", "remove-this", orphans[0].Name)
	}
}
