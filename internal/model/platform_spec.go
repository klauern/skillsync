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
