package model

import (
	"fmt"
	"strings"
)

// SkillType represents the type of skill.
// Skills can be regular agent skills or slash commands/prompts that users invoke.
type SkillType string

const (
	// SkillTypeSkill represents a regular agent skill (default).
	SkillTypeSkill SkillType = "skill"

	// SkillTypePrompt represents a slash command or reusable prompt.
	// In Claude Code these are invoked via /command-name.
	// In Cursor these are called "prompts".
	// In Codex these are called "prompts".
	SkillTypePrompt SkillType = "prompt"
)

// IsValid returns true if the skill type is recognized.
func (t SkillType) IsValid() bool {
	switch t {
	case SkillTypeSkill, SkillTypePrompt:
		return true
	default:
		return false
	}
}

// AllSkillTypes returns all supported skill types.
func AllSkillTypes() []SkillType {
	return []SkillType{SkillTypeSkill, SkillTypePrompt}
}

// String returns the string representation of the skill type.
func (t SkillType) String() string {
	return string(t)
}

// Description returns a human-readable description of the skill type.
func (t SkillType) Description() string {
	switch t {
	case SkillTypeSkill:
		return "Regular agent skill with context and instructions"
	case SkillTypePrompt:
		return "Slash command or reusable prompt invoked by users"
	default:
		return "Unknown skill type"
	}
}

// ParseSkillType converts a string to a SkillType.
// Returns SkillTypeSkill (default) if the string is empty.
// Returns an error if the type is not recognized.
func ParseSkillType(s string) (SkillType, error) {
	if s == "" {
		return SkillTypeSkill, nil
	}

	normalized := strings.ToLower(strings.TrimSpace(s))

	// Try exact match first
	t := SkillType(normalized)
	if t.IsValid() {
		return t, nil
	}

	// Try common aliases
	switch normalized {
	case "command", "slash-command", "slashcommand":
		return SkillTypePrompt, nil
	case "agent", "agent-skill", "agentskill":
		return SkillTypeSkill, nil
	default:
		return "", fmt.Errorf("unknown skill type %q (valid: skill, prompt)", s)
	}
}
