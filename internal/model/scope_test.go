package model

import "testing"

func TestSkillScopeValidation(t *testing.T) {
	tests := map[string]struct {
		scope SkillScope
		valid bool
	}{
		"builtin valid": {scope: ScopeBuiltin, valid: true},
		"system valid":  {scope: ScopeSystem, valid: true},
		"admin valid":   {scope: ScopeAdmin, valid: true},
		"user valid":    {scope: ScopeUser, valid: true},
		"repo valid":    {scope: ScopeRepo, valid: true},
		"empty invalid": {scope: "", valid: false},
		"unknown":       {scope: "unknown", valid: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.scope.IsValid()
			if got != tt.valid {
				t.Errorf("SkillScope(%q).IsValid() = %v, want %v",
					tt.scope, got, tt.valid)
			}
		})
	}
}

func TestAllScopes(t *testing.T) {
	scopes := AllScopes()

	if len(scopes) != 5 {
		t.Errorf("AllScopes() returned %d scopes, want 5", len(scopes))
	}

	for _, s := range scopes {
		if !s.IsValid() {
			t.Errorf("AllScopes() returned invalid scope: %q", s)
		}
	}

	// Verify precedence order (lowest to highest)
	expectedOrder := []SkillScope{ScopeBuiltin, ScopeSystem, ScopeAdmin, ScopeUser, ScopeRepo}
	for i, s := range scopes {
		if s != expectedOrder[i] {
			t.Errorf("AllScopes()[%d] = %q, want %q", i, s, expectedOrder[i])
		}
	}
}

func TestParseScope(t *testing.T) {
	tests := map[string]struct {
		input   string
		want    SkillScope
		wantErr bool
	}{
		"builtin exact":        {input: "builtin", want: ScopeBuiltin, wantErr: false},
		"system exact":         {input: "system", want: ScopeSystem, wantErr: false},
		"admin exact":          {input: "admin", want: ScopeAdmin, wantErr: false},
		"user exact":           {input: "user", want: ScopeUser, wantErr: false},
		"repo exact":           {input: "repo", want: ScopeRepo, wantErr: false},
		"repository alias":     {input: "repository", want: ScopeRepo, wantErr: false},
		"project alias":        {input: "project", want: ScopeRepo, wantErr: false},
		"local alias":          {input: "local", want: ScopeRepo, wantErr: false},
		"global alias":         {input: "global", want: ScopeUser, wantErr: false},
		"home alias":           {input: "home", want: ScopeUser, wantErr: false},
		"administrator alias":  {input: "administrator", want: ScopeAdmin, wantErr: false},
		"sys alias":            {input: "sys", want: ScopeSystem, wantErr: false},
		"default alias":        {input: "default", want: ScopeBuiltin, wantErr: false},
		"built-in alias":       {input: "built-in", want: ScopeBuiltin, wantErr: false},
		"uppercase normalized": {input: "REPO", want: ScopeRepo, wantErr: false},
		"mixed case":           {input: "User", want: ScopeUser, wantErr: false},
		"with whitespace":      {input: "  admin  ", want: ScopeAdmin, wantErr: false},
		"unknown scope":        {input: "unknown", want: "", wantErr: true},
		"empty string":         {input: "", want: "", wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParseScope(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseScope(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseScope(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSkillScopeString(t *testing.T) {
	tests := map[string]struct {
		scope SkillScope
		want  string
	}{
		"builtin": {scope: ScopeBuiltin, want: "builtin"},
		"system":  {scope: ScopeSystem, want: "system"},
		"admin":   {scope: ScopeAdmin, want: "admin"},
		"user":    {scope: ScopeUser, want: "user"},
		"repo":    {scope: ScopeRepo, want: "repo"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.scope.String()
			if got != tt.want {
				t.Errorf("SkillScope.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSkillScopeDescription(t *testing.T) {
	for _, scope := range AllScopes() {
		desc := scope.Description()
		if desc == "" || desc == "Unknown scope" {
			t.Errorf("SkillScope(%q).Description() should return non-empty description", scope)
		}
	}

	// Unknown scope should return "Unknown scope"
	unknown := SkillScope("invalid")
	if unknown.Description() != "Unknown scope" {
		t.Errorf("Invalid scope Description() = %q, want %q", unknown.Description(), "Unknown scope")
	}
}

func TestSkillScopePrecedence(t *testing.T) {
	tests := map[string]struct {
		scope      SkillScope
		precedence int
	}{
		"builtin is lowest":  {scope: ScopeBuiltin, precedence: 0},
		"system is 1":        {scope: ScopeSystem, precedence: 1},
		"admin is 2":         {scope: ScopeAdmin, precedence: 2},
		"user is 3":          {scope: ScopeUser, precedence: 3},
		"repo is highest":    {scope: ScopeRepo, precedence: 4},
		"invalid returns -1": {scope: "invalid", precedence: -1},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.scope.Precedence()
			if got != tt.precedence {
				t.Errorf("SkillScope(%q).Precedence() = %d, want %d", tt.scope, got, tt.precedence)
			}
		})
	}
}

func TestSkillScopeIsHigherPrecedence(t *testing.T) {
	tests := map[string]struct {
		scope  SkillScope
		other  SkillScope
		higher bool
	}{
		"repo > user":       {scope: ScopeRepo, other: ScopeUser, higher: true},
		"user > admin":      {scope: ScopeUser, other: ScopeAdmin, higher: true},
		"admin > system":    {scope: ScopeAdmin, other: ScopeSystem, higher: true},
		"system > builtin":  {scope: ScopeSystem, other: ScopeBuiltin, higher: true},
		"repo > builtin":    {scope: ScopeRepo, other: ScopeBuiltin, higher: true},
		"builtin not > any": {scope: ScopeBuiltin, other: ScopeRepo, higher: false},
		"same scope":        {scope: ScopeUser, other: ScopeUser, higher: false},
		"user not > repo":   {scope: ScopeUser, other: ScopeRepo, higher: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.scope.IsHigherPrecedence(tt.other)
			if got != tt.higher {
				t.Errorf("SkillScope(%q).IsHigherPrecedence(%q) = %v, want %v",
					tt.scope, tt.other, got, tt.higher)
			}
		})
	}
}
