package model

import "time"

// Skill represents a unified agent skill across platforms
type Skill struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Platform    Platform          `json:"platform"`
	Path        string            `json:"path"`
	Tools       []string          `json:"tools,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Content     string            `json:"content"`
	ModifiedAt  time.Time         `json:"modified_at"`

	// Agent Skills Standard fields
	Scope                  SkillScope        `json:"scope,omitempty"`
	DisableModelInvocation bool              `json:"disable_model_invocation,omitempty"`
	License                string            `json:"license,omitempty"`
	Compatibility          map[string]string `json:"compatibility,omitempty"`
	Scripts                []string          `json:"scripts,omitempty"`
	References             []string          `json:"references,omitempty"`
	Assets                 []string          `json:"assets,omitempty"`
}

// IsHigherPrecedence returns true if this skill's scope has higher precedence than other.
func (s Skill) IsHigherPrecedence(other Skill) bool {
	return s.Scope.IsHigherPrecedence(other.Scope)
}
