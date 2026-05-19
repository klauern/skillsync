package cli

import (
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestFilterDeleteCandidates(t *testing.T) {
	sourceSkills := []model.Skill{
		{Name: "alpha", Platform: model.Cursor},
		{Name: "gamma", Platform: model.Cursor},
	}
	targetSkills := []model.Skill{
		{Name: "alpha", Platform: model.ClaudeCode, Scope: model.ScopeUser},
		{Name: "beta", Platform: model.ClaudeCode, Scope: model.ScopeUser},
		{Name: "gamma", Platform: model.ClaudeCode, Scope: model.ScopeRepo},
	}

	candidates := filterDeleteCandidates(sourceSkills, targetSkills)

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].Name != "alpha" || candidates[1].Name != "gamma" {
		t.Errorf("unexpected candidate order: %v, %v", candidates[0].Name, candidates[1].Name)
	}
	if candidates[0].Platform != model.ClaudeCode || candidates[1].Platform != model.ClaudeCode {
		t.Error("expected candidates to keep target platform metadata")
	}
}

func TestSelectSourceSkillsForDelete(t *testing.T) {
	sourceSkills := []model.Skill{
		{Name: "alpha", Platform: model.Cursor},
		{Name: "beta", Platform: model.Cursor},
	}
	selectedTargets := []model.Skill{
		{Name: "beta", Platform: model.ClaudeCode},
		{Name: "alpha", Platform: model.ClaudeCode},
		{Name: "missing", Platform: model.ClaudeCode},
	}

	selected := selectSourceSkillsForDelete(sourceSkills, selectedTargets)

	if len(selected) != 2 {
		t.Fatalf("expected 2 selected source skills, got %d", len(selected))
	}
	if selected[0].Name != "beta" || selected[1].Name != "alpha" {
		t.Errorf("unexpected selected order: %v, %v", selected[0].Name, selected[1].Name)
	}
	if selected[0].Platform != model.Cursor || selected[1].Platform != model.Cursor {
		t.Error("expected selected skills to use source platform metadata")
	}
}
