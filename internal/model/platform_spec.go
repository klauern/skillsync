// Package model provides data types for skillsync.
package model

import (
	"fmt"
	"strings"
)

// PlatformSpec represents a platform with optional scope specifier(s).
// Supports formats: "cursor", "cursor:repo", "cursor:repo,user"
type PlatformSpec struct {
	Platform Platform
	Scopes   []SkillScope // Empty means all scopes (for source) or user scope (for target)
}

// SourceScopeOrder returns the canonical ordering for selectable source scopes.
// The ordering matches the platform-wide precedence order.
func SourceScopeOrder() []SkillScope {
	return AllScopes()
}

// NormalizeSourceScopes canonicalizes a source-scope selection.
// It removes duplicates, orders scopes canonically, and collapses a full
// selection set to nil so callers can treat it as "all".
func NormalizeSourceScopes(scopes []SkillScope) []SkillScope {
	ordered := SourceScopeOrder()
	seen := make(map[SkillScope]bool, len(scopes))
	normalized := make([]SkillScope, 0, len(scopes))

	for _, orderedScope := range ordered {
		for _, scope := range scopes {
			if scope == orderedScope && !seen[scope] {
				seen[scope] = true
				normalized = append(normalized, scope)
				break
			}
		}
	}

	if len(normalized) == len(ordered) {
		return nil
	}
	return normalized
}

// ParseSourceScopes parses a source scope list.
// The empty string and the literal "all" both mean all scopes.
// Duplicate values are removed and the result is canonicalized.
func ParseSourceScopes(s string) ([]SkillScope, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "all") {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	scopes := make([]SkillScope, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.EqualFold(part, "all") {
			return nil, nil
		}
		scope, err := ParseScope(part)
		if err != nil {
			return nil, fmt.Errorf("invalid scope in %q: %w", s, err)
		}
		scopes = append(scopes, scope)
	}

	if len(scopes) == 0 {
		return nil, fmt.Errorf("no valid scopes found in %q", s)
	}

	return NormalizeSourceScopes(scopes), nil
}

// FormatSourceScopes renders a source scope selection.
// A nil/empty selection or a full selection set renders as "all".
func FormatSourceScopes(scopes []SkillScope) string {
	normalized := NormalizeSourceScopes(scopes)
	if len(normalized) == 0 {
		return "all"
	}

	parts := make([]string, len(normalized))
	for i, scope := range normalized {
		parts[i] = string(scope)
	}
	return strings.Join(parts, ",")
}

// HasScopes returns true if explicit scopes were specified.
func (ps PlatformSpec) HasScopes() bool {
	return len(ps.Scopes) > 0
}

// String returns the string representation of the platform spec.
func (ps PlatformSpec) String() string {
	if len(ps.Scopes) == 0 {
		return string(ps.Platform)
	}
	scopeStrs := make([]string, len(ps.Scopes))
	for i, s := range ps.Scopes {
		scopeStrs[i] = string(s)
	}
	return fmt.Sprintf("%s:%s", ps.Platform, strings.Join(scopeStrs, ","))
}

// NormalizeSource returns a copy of the PlatformSpec with source-scopes
// canonicalized for multi-select usage.
func (ps PlatformSpec) NormalizeSource() PlatformSpec {
	ps.Scopes = NormalizeSourceScopes(ps.Scopes)
	return ps
}

// SourceString returns the canonical string representation for source selection.
// When all scopes are selected, the normalized string is just the platform.
func (ps PlatformSpec) SourceString() string {
	return ps.NormalizeSource().String()
}

// ParsePlatformSpec parses a platform:scope specification string.
// Formats supported:
//   - "cursor"           -> Platform: cursor, Scopes: [] (empty = all/default)
//   - "cursor:repo"      -> Platform: cursor, Scopes: [repo]
//   - "cursor:repo,user" -> Platform: cursor, Scopes: [repo, user]
//
// Returns an error if the platform or any scope is invalid.
func ParsePlatformSpec(s string) (PlatformSpec, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return PlatformSpec{}, fmt.Errorf("platform spec cannot be empty")
	}

	// Split on colon to separate platform from scope(s)
	parts := strings.SplitN(s, ":", 2)
	platformStr := parts[0]

	// Parse platform
	platform, err := ParsePlatform(platformStr)
	if err != nil {
		return PlatformSpec{}, err
	}

	spec := PlatformSpec{
		Platform: platform,
		Scopes:   []SkillScope{},
	}

	// If no scope specified, return with empty scopes
	if len(parts) == 1 {
		return spec, nil
	}

	// Parse scope(s)
	scopeStr := strings.TrimSpace(parts[1])
	if scopeStr == "" {
		return PlatformSpec{}, fmt.Errorf("scope cannot be empty after colon in %q", s)
	}

	// Split by comma for multiple scopes
	scopeParts := strings.Split(scopeStr, ",")
	for _, sp := range scopeParts {
		sp = strings.TrimSpace(sp)
		if sp == "" {
			continue
		}
		scope, err := ParseScope(sp)
		if err != nil {
			return PlatformSpec{}, fmt.Errorf("invalid scope in %q: %w", s, err)
		}
		spec.Scopes = append(spec.Scopes, scope)
	}

	if len(spec.Scopes) == 0 {
		return PlatformSpec{}, fmt.Errorf("no valid scopes found in %q", s)
	}

	return spec, nil
}

// ValidateAsTarget validates the PlatformSpec for use as a sync target.
// Target specs can only have a single scope, and only repo or user are allowed.
func (ps PlatformSpec) ValidateAsTarget() error {
	if len(ps.Scopes) > 1 {
		return fmt.Errorf("target can only have one scope, got %d", len(ps.Scopes))
	}
	if len(ps.Scopes) == 1 {
		scope := ps.Scopes[0]
		if scope != ScopeRepo && scope != ScopeUser {
			return fmt.Errorf("target scope must be 'repo' or 'user', got %q", scope)
		}
	}
	return nil
}

// TargetScope returns the target scope, defaulting to ScopeUser if not specified.
func (ps PlatformSpec) TargetScope() SkillScope {
	if len(ps.Scopes) > 0 {
		return ps.Scopes[0]
	}
	return ScopeUser
}
