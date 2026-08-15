package cli

import "github.com/klauern/skillsync/internal/model"

// findOrphanedSkills returns target skills whose names do not appear in source.
func findOrphanedSkills(sourceSkills, targetSkills []model.Skill) []model.Skill {
	if len(targetSkills) == 0 {
		return nil
	}

	sourceNames := make(map[string]bool, len(sourceSkills))
	for _, skill := range sourceSkills {
		sourceNames[skill.Name] = true
	}

	var orphans []model.Skill
	for _, skill := range targetSkills {
		if !sourceNames[skill.Name] {
			orphans = append(orphans, skill)
		}
	}

	return orphans
}

func selectSourceSkillsForDelete(sourceSkills, selectedTargets []model.Skill) []model.Skill {
	if len(sourceSkills) == 0 || len(selectedTargets) == 0 {
		return nil
	}

	sourceByName := make(map[string]model.Skill, len(sourceSkills))
	for _, skill := range sourceSkills {
		if _, exists := sourceByName[skill.Name]; !exists {
			sourceByName[skill.Name] = skill
		}
	}

	selected := make([]model.Skill, 0, len(selectedTargets))
	for _, skill := range selectedTargets {
		if sourceSkill, ok := sourceByName[skill.Name]; ok {
			selected = append(selected, sourceSkill)
		}
	}

	return selected
}
