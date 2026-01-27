package model

import (
	"testing"
	"time"
)

func TestSkillIsHigherPrecedence(t *testing.T) {
	tests := map[string]struct {
		skill  Skill
		other  Skill
		higher bool
	}{
		"repo skill overrides user skill": {
			skill:  Skill{Name: "test", Scope: ScopeRepo},
			other:  Skill{Name: "test", Scope: ScopeUser},
			higher: true,
		},
		"user skill overrides admin skill": {
			skill:  Skill{Name: "test", Scope: ScopeUser},
			other:  Skill{Name: "test", Scope: ScopeAdmin},
			higher: true,
		},
		"admin skill overrides system skill": {
			skill:  Skill{Name: "test", Scope: ScopeAdmin},
			other:  Skill{Name: "test", Scope: ScopeSystem},
			higher: true,
		},
		"system skill overrides builtin skill": {
			skill:  Skill{Name: "test", Scope: ScopeSystem},
			other:  Skill{Name: "test", Scope: ScopeBuiltin},
			higher: true,
		},
		"builtin skill does not override repo skill": {
			skill:  Skill{Name: "test", Scope: ScopeBuiltin},
			other:  Skill{Name: "test", Scope: ScopeRepo},
			higher: false,
		},
		"same scope has no higher precedence": {
			skill:  Skill{Name: "test", Scope: ScopeUser},
			other:  Skill{Name: "test", Scope: ScopeUser},
			higher: false,
		},
		"empty scope is lowest precedence": {
			skill:  Skill{Name: "test", Scope: ""},
			other:  Skill{Name: "test", Scope: ScopeBuiltin},
			higher: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.skill.IsHigherPrecedence(tt.other)
			if got != tt.higher {
				t.Errorf("Skill.IsHigherPrecedence() = %v, want %v", got, tt.higher)
			}
		})
	}
}

func TestSkillAgentSkillsFields(t *testing.T) {
	// Test that all new Agent Skills Standard fields can be set and retrieved
	skill := Skill{
		Name:        "test-skill",
		Description: "A test skill",
		Platform:    ClaudeCode,
		Path:        "/path/to/skill.md",
		Content:     "skill content",
		ModifiedAt:  time.Now(),

		// Agent Skills Standard fields
		Scope:                  ScopeUser,
		DisableModelInvocation: true,
		License:                "MIT",
		Compatibility: map[string]string{
			"claude-code": ">=1.0.0",
			"cursor":      ">=0.30.0",
		},
		Scripts:    []string{"setup.sh", "validate.sh"},
		References: []string{"https://docs.example.com", "related-skill"},
		Assets:     []string{"template.json", "config.yaml"},
	}

	// Verify basic fields
	if skill.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", skill.Name, "test-skill")
	}
	if skill.Description != "A test skill" {
		t.Errorf("Description = %q, want %q", skill.Description, "A test skill")
	}
	if skill.Platform != ClaudeCode {
		t.Errorf("Platform = %q, want %q", skill.Platform, ClaudeCode)
	}

	// Verify Agent Skills Standard fields
	if skill.Scope != ScopeUser {
		t.Errorf("Scope = %q, want %q", skill.Scope, ScopeUser)
	}
	if !skill.DisableModelInvocation {
		t.Error("DisableModelInvocation = false, want true")
	}
	if skill.License != "MIT" {
		t.Errorf("License = %q, want %q", skill.License, "MIT")
	}
	if len(skill.Compatibility) != 2 {
		t.Errorf("Compatibility has %d entries, want 2", len(skill.Compatibility))
	}
	if skill.Compatibility["claude-code"] != ">=1.0.0" {
		t.Errorf("Compatibility[claude-code] = %q, want %q",
			skill.Compatibility["claude-code"], ">=1.0.0")
	}
	if len(skill.Scripts) != 2 {
		t.Errorf("Scripts has %d entries, want 2", len(skill.Scripts))
	}
	if skill.Scripts[0] != "setup.sh" {
		t.Errorf("Scripts[0] = %q, want %q", skill.Scripts[0], "setup.sh")
	}
	if len(skill.References) != 2 {
		t.Errorf("References has %d entries, want 2", len(skill.References))
	}
	if len(skill.Assets) != 2 {
		t.Errorf("Assets has %d entries, want 2", len(skill.Assets))
	}
}

func TestSkillDisplayScope(t *testing.T) {
	tests := map[string]struct {
		skill Skill
		want  string
	}{
		"user scope claude": {
			skill: Skill{Platform: ClaudeCode, Scope: ScopeUser},
			want:  "~/.claude",
		},
		"user scope cursor": {
			skill: Skill{Platform: Cursor, Scope: ScopeUser},
			want:  "~/.cursor",
		},
		"user scope codex": {
			skill: Skill{Platform: Codex, Scope: ScopeUser},
			want:  "~/.codex",
		},
		"repo scope claude": {
			skill: Skill{Platform: ClaudeCode, Scope: ScopeRepo},
			want:  ".claude",
		},
		"repo scope cursor": {
			skill: Skill{Platform: Cursor, Scope: ScopeRepo},
			want:  ".cursor",
		},
		"plugin scope with name": {
			skill: Skill{
				Platform: ClaudeCode,
				Scope:    ScopePlugin,
				Metadata: map[string]string{"plugin": "my-plugin"},
			},
			want: "plugin:my-plugin",
		},
		"plugin scope without name": {
			skill: Skill{Platform: ClaudeCode, Scope: ScopePlugin},
			want:  "plugin",
		},
		"plugin scope empty metadata": {
			skill: Skill{
				Platform: ClaudeCode,
				Scope:    ScopePlugin,
				Metadata: map[string]string{},
			},
			want: "plugin",
		},
		"system scope": {
			skill: Skill{Platform: ClaudeCode, Scope: ScopeSystem},
			want:  "system",
		},
		"admin scope": {
			skill: Skill{Platform: ClaudeCode, Scope: ScopeAdmin},
			want:  "admin",
		},
		"builtin scope": {
			skill: Skill{Platform: ClaudeCode, Scope: ScopeBuiltin},
			want:  "builtin",
		},
		"empty scope": {
			skill: Skill{Platform: ClaudeCode, Scope: ""},
			want:  "-",
		},
		"unknown scope": {
			skill: Skill{Platform: ClaudeCode, Scope: "custom"},
			want:  "custom",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.skill.DisplayScope()
			if got != tt.want {
				t.Errorf("Skill.DisplayScope() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSkillZeroValues(t *testing.T) {
	// Test that a skill with zero values for new fields works correctly
	skill := Skill{
		Name:     "minimal-skill",
		Platform: Cursor,
		Content:  "content",
	}

	// Zero-value scope should return -1 for precedence
	if skill.Scope.Precedence() != -1 {
		t.Errorf("Empty Scope.Precedence() = %d, want -1", skill.Scope.Precedence())
	}

	// Skill with zero-value scope should not be higher precedence than any valid scope
	otherSkill := Skill{Name: "other", Scope: ScopeBuiltin}
	if skill.IsHigherPrecedence(otherSkill) {
		t.Error("Skill with empty scope should not be higher precedence than builtin")
	}

	// Zero-value bool should be false
	if skill.DisableModelInvocation {
		t.Error("Zero-value DisableModelInvocation should be false")
	}

	// Zero-value slices should be nil
	if skill.Scripts != nil {
		t.Error("Zero-value Scripts should be nil")
	}
	if skill.References != nil {
		t.Error("Zero-value References should be nil")
	}
	if skill.Assets != nil {
		t.Error("Zero-value Assets should be nil")
	}
}
