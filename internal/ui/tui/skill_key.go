package tui

import (
	"fmt"

	"github.com/klauern/skillsync/internal/model"
)

// scopedSkillKey creates a unique key for a skill within the TUI.
func scopedSkillKey(skill model.Skill) string {
	return fmt.Sprintf("%s:%s:%s", skill.Platform, skill.Scope, skill.Name)
}
