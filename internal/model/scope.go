package model

import (
	"fmt"
	"strings"
)

// SkillScope represents the scope level of a skill in the tiered lookup system.
// Scopes follow a precedence order where more specific scopes override more general ones.
type SkillScope string

const (
	// ScopeBuiltin represents built-in skills that ship with the platform.
	ScopeBuiltin SkillScope = "builtin"

	// ScopeSystem represents system-wide skills installed at the system level.
	ScopeSystem SkillScope = "system"

	// ScopeAdmin represents administrator-defined skills.
	ScopeAdmin SkillScope = "admin"

	// ScopeUser represents user-level skills in the user's home directory.
	ScopeUser SkillScope = "user"

	// ScopeRepo represents repository-level skills local to a specific project.
	ScopeRepo SkillScope = "repo"
)

// scopePrecedence defines the order of precedence for skill scopes.
// Higher index = higher precedence (overrides lower).
var scopePrecedence = map[SkillScope]int{
	ScopeBuiltin: 0,
	ScopeSystem:  1,
	ScopeAdmin:   2,
	ScopeUser:    3,
	ScopeRepo:    4,
}

// IsValid returns true if the scope is recognized.
func (s SkillScope) IsValid() bool {
	_, ok := scopePrecedence[s]
	return ok
}

// AllScopes returns all supported skill scopes in precedence order (lowest to highest).
func AllScopes() []SkillScope {
	return []SkillScope{ScopeBuiltin, ScopeSystem, ScopeAdmin, ScopeUser, ScopeRepo}
}

// String returns the string representation of the scope.
func (s SkillScope) String() string {
	return string(s)
}

// Description returns a human-readable description of the scope.
func (s SkillScope) Description() string {
	switch s {
	case ScopeBuiltin:
		return "Built-in skills that ship with the platform"
	case ScopeSystem:
		return "System-wide skills installed at the system level"
	case ScopeAdmin:
		return "Administrator-defined skills"
	case ScopeUser:
		return "User-level skills in the user's home directory"
	case ScopeRepo:
		return "Repository-level skills local to a specific project"
	default:
		return "Unknown scope"
	}
}

// Precedence returns the precedence level of the scope.
// Higher values indicate higher precedence (overrides lower).
func (s SkillScope) Precedence() int {
	if p, ok := scopePrecedence[s]; ok {
		return p
	}
	return -1
}

// IsHigherPrecedence returns true if this scope has higher precedence than other.
// A scope with higher precedence overrides a scope with lower precedence.
func (s SkillScope) IsHigherPrecedence(other SkillScope) bool {
	return s.Precedence() > other.Precedence()
}

// ParseScope converts a string to a SkillScope type.
// Returns an error if the scope is not recognized.
func ParseScope(s string) (SkillScope, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))

	// Try exact match first
	scope := SkillScope(normalized)
	if scope.IsValid() {
		return scope, nil
	}

	// Try common aliases
	switch normalized {
	case "repository", "project", "local":
		return ScopeRepo, nil
	case "global", "home":
		return ScopeUser, nil
	case "administrator":
		return ScopeAdmin, nil
	case "sys":
		return ScopeSystem, nil
	case "default", "built-in":
		return ScopeBuiltin, nil
	default:
		return "", fmt.Errorf("unknown scope %q (valid: builtin, system, admin, user, repo)", s)
	}
}
