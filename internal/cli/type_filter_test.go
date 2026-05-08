package cli

import (
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

// TestParseTypeFilter tests the parseTypeFilter function that converts
// comma-separated skill type strings into model.SkillType slices.
func TestParseTypeFilter(t *testing.T) {
	tests := map[string]struct {
		input   string
		want    []model.SkillType
		wantErr bool
	}{
		"empty string returns nil (no filter)": {
			input: "",
			want:  nil,
		},
		"whitespace-only returns nil": {
			input: "   ",
			want:  nil,
		},
		"all keyword returns nil (disable filter)": {
			input: "all",
			want:  nil,
		},
		"ALL uppercase returns nil": {
			input: "ALL",
			want:  nil,
		},
		"skill type only": {
			input: "skill",
			want:  []model.SkillType{model.SkillTypeSkill},
		},
		"prompt type only": {
			input: "prompt",
			want:  []model.SkillType{model.SkillTypePrompt},
		},
		"skill and prompt comma-separated": {
			input: "skill,prompt",
			want:  []model.SkillType{model.SkillTypeSkill, model.SkillTypePrompt},
		},
		"values with spaces are trimmed": {
			input: " skill , prompt ",
			want:  []model.SkillType{model.SkillTypeSkill, model.SkillTypePrompt},
		},
		"unknown type returns error": {
			input:   "unknown-type",
			wantErr: true,
		},
		"mixed valid and invalid returns error": {
			input:   "skill,invalid",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := parseTypeFilter(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseTypeFilter(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseTypeFilter(%q) unexpected error: %v", tt.input, err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("parseTypeFilter(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.want, len(tt.want))
				return
			}
			for i, g := range got {
				if g != tt.want[i] {
					t.Errorf("parseTypeFilter(%q)[%d] = %q, want %q", tt.input, i, g, tt.want[i])
				}
			}
		})
	}
}

// TestFilterBySkillType tests the filterBySkillType function that filters
// a skill slice by a set of skill types.
func TestFilterBySkillType(t *testing.T) {
	skills := []model.Skill{
		{Name: "skill-a", Type: model.SkillTypeSkill},
		{Name: "skill-b", Type: model.SkillTypeSkill},
		{Name: "prompt-a", Type: model.SkillTypePrompt},
		{Name: "prompt-b", Type: model.SkillTypePrompt},
		{Name: "empty-type"}, // zero-value Type, treated as SkillTypeSkill
	}

	tests := map[string]struct {
		typeFilter []model.SkillType
		wantNames  []string
	}{
		"empty filter returns all skills": {
			typeFilter: nil,
			wantNames:  []string{"skill-a", "skill-b", "prompt-a", "prompt-b", "empty-type"},
		},
		"skill filter returns skills and empty-type skills": {
			typeFilter: []model.SkillType{model.SkillTypeSkill},
			wantNames:  []string{"skill-a", "skill-b", "empty-type"},
		},
		"prompt filter returns only prompts": {
			typeFilter: []model.SkillType{model.SkillTypePrompt},
			wantNames:  []string{"prompt-a", "prompt-b"},
		},
		"skill and prompt filter returns everything": {
			typeFilter: []model.SkillType{model.SkillTypeSkill, model.SkillTypePrompt},
			wantNames:  []string{"skill-a", "skill-b", "prompt-a", "prompt-b", "empty-type"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := filterBySkillType(skills, tt.typeFilter)

			if len(got) != len(tt.wantNames) {
				gotNames := make([]string, len(got))
				for i, s := range got {
					gotNames[i] = s.Name
				}
				t.Errorf("filterBySkillType() returned %d skills %v, want %d %v",
					len(got), gotNames, len(tt.wantNames), tt.wantNames)
				return
			}

			// Build a map of returned names for order-independent comparison
			gotNameSet := make(map[string]bool, len(got))
			for _, s := range got {
				gotNameSet[s.Name] = true
			}

			for _, want := range tt.wantNames {
				if !gotNameSet[want] {
					t.Errorf("filterBySkillType() missing expected skill %q", want)
				}
			}
		})
	}
}

// TestFilterBySkillType_EmptyInput verifies filtering an empty slice returns an empty slice.
func TestFilterBySkillType_EmptyInput(t *testing.T) {
	got := filterBySkillType(nil, []model.SkillType{model.SkillTypeSkill})
	if len(got) != 0 {
		t.Errorf("filterBySkillType(nil, ...) = %v, want empty", got)
	}
}

// TestFilterBySkillType_EmptyTypeIsSkill verifies that skills with an empty Type
// field are treated as SkillTypeSkill for filtering purposes.
func TestFilterBySkillType_EmptyTypeIsSkill(t *testing.T) {
	skills := []model.Skill{
		{Name: "no-type"},      // empty Type
		{Name: "prompt-skill", Type: model.SkillTypePrompt},
	}

	// Filter for skill type only
	got := filterBySkillType(skills, []model.SkillType{model.SkillTypeSkill})
	if len(got) != 1 {
		t.Fatalf("expected 1 skill (empty-type), got %d", len(got))
	}
	if got[0].Name != "no-type" {
		t.Errorf("got %q, want %q", got[0].Name, "no-type")
	}
}