package tui

import (
	"fmt"
	"strings"

	"github.com/klauern/skillsync/internal/model"
)

// scopedSkillKey creates a unique key for a skill within the TUI.
func scopedSkillKey(skill model.Skill) string {
	return fmt.Sprintf("%s:%s:%s", skill.Platform, skill.Scope, skill.Name)
}

// skillMatchesFilter reports whether a skill matches an already-lowercased text filter.
func skillMatchesFilter(skill model.Skill, lowerFilter string) bool {
	if lowerFilter == "" {
		return true
	}

	return strings.Contains(strings.ToLower(skill.Name), lowerFilter) ||
		strings.Contains(strings.ToLower(string(skill.Platform)), lowerFilter) ||
		strings.Contains(strings.ToLower(skill.DisplayScope()), lowerFilter) ||
		strings.Contains(strings.ToLower(skill.Description), lowerFilter)
}
