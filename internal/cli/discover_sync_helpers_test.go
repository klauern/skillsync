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
