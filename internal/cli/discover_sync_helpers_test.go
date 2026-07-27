package cli

import (
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestPluginSkillDedupeKey(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		skill model.Skill
		want  string
	}{
		"uses plugin info marketplace": {
			skill: model.Skill{
				Name: "review",
				PluginInfo: &model.PluginInfo{
					Marketplace: "openai",
				},
				Metadata: map[string]string{
					"marketplace": "ignored",
				},
			},
			want: "review@openai",
		},
		"falls back to metadata marketplace": {
			skill: model.Skill{
				Name: "review",
				Metadata: map[string]string{
					"marketplace": "community",
				},
			},
			want: "review@community",
		},
		"ignores empty marketplace metadata": {
			skill: model.Skill{
				Name: "review",
				Metadata: map[string]string{
					"marketplace": "",
				},
			},
			want: "review",
		},
		"falls back to name": {
			skill: model.Skill{
				Name: "review",
			},
			want: "review",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := pluginSkillDedupeKey(tt.skill); got != tt.want {
				t.Fatalf("pluginSkillDedupeKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseScopeFilter(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input   string
		want    []model.SkillScope
		wantErr bool
	}{
		"empty disables filtering": {input: ""},
		"all disables filtering":   {input: "all"},
		"single scope":             {input: "user", want: []model.SkillScope{model.ScopeUser}},
		"multiple scopes":          {input: "repo, plugin", want: []model.SkillScope{model.ScopeRepo, model.ScopePlugin}},
		"invalid scope":            {input: "user,invalid", wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := parseScopeFilter(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseScopeFilter() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseScopeFilter() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseScopeFilter() = %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("parseScopeFilter()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
