package model

import "testing"

func TestSkillTypeValidation(t *testing.T) {
	tests := map[string]struct {
		skillType SkillType
		valid     bool
	}{
		"skill valid":   {skillType: SkillTypeSkill, valid: true},
		"prompt valid":  {skillType: SkillTypePrompt, valid: true},
		"empty invalid": {skillType: "", valid: false},
		"unknown":       {skillType: "unknown", valid: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.skillType.IsValid()
			if got != tt.valid {
				t.Errorf("SkillType(%q).IsValid() = %v, want %v",
					tt.skillType, got, tt.valid)
			}
		})
	}
}

func TestAllSkillTypes(t *testing.T) {
	types := AllSkillTypes()

	if len(types) != 2 {
		t.Errorf("AllSkillTypes() returned %d types, want 2", len(types))
	}

	for _, st := range types {
		if !st.IsValid() {
			t.Errorf("AllSkillTypes() returned invalid type: %q", st)
		}
	}

	// Verify expected types exist
	expected := []SkillType{SkillTypeSkill, SkillTypePrompt}
	for i, st := range types {
		if st != expected[i] {
			t.Errorf("AllSkillTypes()[%d] = %q, want %q", i, st, expected[i])
		}
	}
}

func TestParseSkillType(t *testing.T) {
	tests := map[string]struct {
		input   string
		want    SkillType
		wantErr bool
	}{
		"skill exact":          {input: "skill", want: SkillTypeSkill, wantErr: false},
		"prompt exact":         {input: "prompt", want: SkillTypePrompt, wantErr: false},
		"empty returns skill":  {input: "", want: SkillTypeSkill, wantErr: false},
		"command alias":        {input: "command", want: SkillTypePrompt, wantErr: false},
		"slash-command alias":  {input: "slash-command", want: SkillTypePrompt, wantErr: false},
		"slashcommand alias":   {input: "slashcommand", want: SkillTypePrompt, wantErr: false},
		"agent alias":          {input: "agent", want: SkillTypeSkill, wantErr: false},
		"agent-skill alias":    {input: "agent-skill", want: SkillTypeSkill, wantErr: false},
		"agentskill alias":     {input: "agentskill", want: SkillTypeSkill, wantErr: false},
		"uppercase normalized": {input: "SKILL", want: SkillTypeSkill, wantErr: false},
		"mixed case":           {input: "Prompt", want: SkillTypePrompt, wantErr: false},
		"with whitespace":      {input: "  skill  ", want: SkillTypeSkill, wantErr: false},
		"unknown type":         {input: "unknown", want: "", wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParseSkillType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSkillType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseSkillType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSkillTypeString(t *testing.T) {
	tests := map[string]struct {
		skillType SkillType
		want      string
	}{
		"skill":  {skillType: SkillTypeSkill, want: "skill"},
		"prompt": {skillType: SkillTypePrompt, want: "prompt"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.skillType.String()
			if got != tt.want {
				t.Errorf("SkillType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSkillTypeDescription(t *testing.T) {
	for _, st := range AllSkillTypes() {
		desc := st.Description()
		if desc == "" || desc == "Unknown skill type" {
			t.Errorf("SkillType(%q).Description() should return non-empty description", st)
		}
	}

	// Unknown type should return "Unknown skill type"
	unknown := SkillType("invalid")
	if unknown.Description() != "Unknown skill type" {
		t.Errorf("Invalid type Description() = %q, want %q", unknown.Description(), "Unknown skill type")
	}
}
