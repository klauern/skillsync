package tui

import (
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestScopedSkillKeyIncludesIdentityFields(t *testing.T) {
	t.Parallel()

	base := model.Skill{
		Name:     "review",
		Platform: model.ClaudeCode,
		Scope:    model.ScopeUser,
	}

	tests := []struct {
		name  string
		skill model.Skill
		want  string
	}{
		{name: "base", skill: base, want: "claude-code:user:review"},
		{name: "platform", skill: model.Skill{Name: base.Name, Platform: model.Codex, Scope: base.Scope}, want: "codex:user:review"},
		{name: "scope", skill: model.Skill{Name: base.Name, Platform: base.Platform, Scope: model.ScopeRepo}, want: "claude-code:repo:review"},
		{name: "name", skill: model.Skill{Name: "test", Platform: base.Platform, Scope: base.Scope}, want: "claude-code:user:test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := scopedSkillKey(tt.skill); got != tt.want {
				t.Fatalf("scopedSkillKey() = %q, want %q", got, tt.want)
			}
		})
	}
}
