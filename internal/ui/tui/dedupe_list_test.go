package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/similarity"
)

func makeComparison(skill1, skill2 model.Skill, nameScore, contentScore float64) *similarity.ComparisonResult {
	return &similarity.ComparisonResult{
		Skill1:       skill1,
		Skill2:       skill2,
		NameScore:    nameScore,
		ContentScore: contentScore,
	}
}

func TestNewDedupeListModel_FiltersDeletableScopes(t *testing.T) {
	repoSkill := model.Skill{Name: "repo-skill", Platform: model.Cursor, Scope: model.ScopeRepo}
	pluginSkill := model.Skill{Name: "plugin-skill", Platform: model.Cursor, Scope: model.ScopePlugin}

	duplicates := []*similarity.ComparisonResult{
		makeComparison(repoSkill, pluginSkill, 0.4, 0.6),
	}

	dedupeModel := NewDedupeListModel(duplicates)
	if len(dedupeModel.flatSkills) != 1 {
		t.Fatalf("expected 1 deletable skill, got %d", len(dedupeModel.flatSkills))
	}
	if dedupeModel.flatSkills[0].Scope != model.ScopeRepo {
		t.Errorf("expected repo scope skill to remain, got %q", dedupeModel.flatSkills[0].Scope)
	}
}

func TestDedupeListModel_FindSimilarSkill_PicksBest(t *testing.T) {
	skillA := model.Skill{Name: "alpha", Platform: model.ClaudeCode, Scope: model.ScopeRepo}
	skillB := model.Skill{Name: "beta", Platform: model.Cursor, Scope: model.ScopeRepo}
	skillC := model.Skill{Name: "gamma", Platform: model.Codex, Scope: model.ScopeRepo}

	duplicates := []*similarity.ComparisonResult{
		makeComparison(skillA, skillB, 0.5, 0.4),
		makeComparison(skillA, skillC, 0.8, 0.3),
	}

	dedupeModel := NewDedupeListModel(duplicates)
	match, nameScore, contentScore := dedupeModel.findSimilarSkill(skillA)
	if match.Name != "gamma" {
		t.Errorf("expected best match to be gamma, got %s", match.Name)
	}
	if nameScore != 0.8 || contentScore != 0.3 {
		t.Errorf("expected scores 0.8/0.3, got %.1f/%.1f", nameScore, contentScore)
	}
}

func TestDedupeListModel_ToggleAll(t *testing.T) {
	skillA := model.Skill{Name: "alpha", Platform: model.ClaudeCode, Scope: model.ScopeRepo}
	skillB := model.Skill{Name: "beta", Platform: model.Cursor, Scope: model.ScopeRepo}

	duplicates := []*similarity.ComparisonResult{
		makeComparison(skillA, skillB, 0.5, 0.5),
	}

	dedupeModel := NewDedupeListModel(duplicates)
	newModel, _ := dedupeModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	selectedModel := newModel.(DedupeListModel)

	for _, skill := range selectedModel.filtered {
		if !selectedModel.selected[dedupeSkillKey(skill)] {
			t.Errorf("expected skill %s to be selected after toggle all", skill.Name)
		}
	}

	newModel, _ = selectedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	deselectedModel := newModel.(DedupeListModel)
	for _, skill := range deselectedModel.filtered {
		if deselectedModel.selected[dedupeSkillKey(skill)] {
			t.Errorf("expected skill %s to be deselected after second toggle", skill.Name)
		}
	}
}

func TestDedupeListModel_ApplyFilter(t *testing.T) {
	skillA := model.Skill{Name: "alpha", Platform: model.ClaudeCode, Scope: model.ScopeRepo}
	skillB := model.Skill{Name: "beta", Platform: model.Cursor, Scope: model.ScopeRepo}

	duplicates := []*similarity.ComparisonResult{
		makeComparison(skillA, skillB, 0.5, 0.5),
	}

	dedupeModel := NewDedupeListModel(duplicates)
	dedupeModel.filter = "cursor"
	dedupeModel.applyFilter()

	if len(dedupeModel.filtered) != 1 {
		t.Fatalf("expected 1 filtered skill, got %d", len(dedupeModel.filtered))
	}
	if dedupeModel.filtered[0].Platform != model.Cursor {
		t.Errorf("expected cursor skill after filter, got %q", dedupeModel.filtered[0].Platform)
	}
}
