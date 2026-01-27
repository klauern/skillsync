package model

import (
	"testing"
)

func TestParsePlatformSpec(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantPlatform Platform
		wantScopes  []SkillScope
		wantErr     bool
	}{
		// Valid cases - platform only
		{
			name:         "platform only - cursor",
			input:        "cursor",
			wantPlatform: Cursor,
			wantScopes:   []SkillScope{},
			wantErr:      false,
		},
		{
			name:         "platform only - claudecode",
			input:        "claudecode",
			wantPlatform: ClaudeCode,
			wantScopes:   []SkillScope{},
			wantErr:      false,
		},
		{
			name:         "platform only - claude-code",
			input:        "claude-code",
			wantPlatform: ClaudeCode,
			wantScopes:   []SkillScope{},
			wantErr:      false,
		},
		{
			name:         "platform only - codex",
			input:        "codex",
			wantPlatform: Codex,
			wantScopes:   []SkillScope{},
			wantErr:      false,
		},

		// Valid cases - single scope
		{
			name:         "single scope - repo",
			input:        "cursor:repo",
			wantPlatform: Cursor,
			wantScopes:   []SkillScope{ScopeRepo},
			wantErr:      false,
		},
		{
			name:         "single scope - user",
			input:        "cursor:user",
			wantPlatform: Cursor,
			wantScopes:   []SkillScope{ScopeUser},
			wantErr:      false,
		},
		{
			name:         "single scope - admin",
			input:        "claudecode:admin",
			wantPlatform: ClaudeCode,
			wantScopes:   []SkillScope{ScopeAdmin},
			wantErr:      false,
		},

		// Valid cases - multiple scopes
		{
			name:         "multiple scopes - repo,user",
			input:        "cursor:repo,user",
			wantPlatform: Cursor,
			wantScopes:   []SkillScope{ScopeRepo, ScopeUser},
			wantErr:      false,
		},
		{
			name:         "multiple scopes - user,admin,repo",
			input:        "claudecode:user,admin,repo",
			wantPlatform: ClaudeCode,
			wantScopes:   []SkillScope{ScopeUser, ScopeAdmin, ScopeRepo},
			wantErr:      false,
		},

		// Valid cases - with whitespace
		{
			name:         "whitespace - platform with spaces",
			input:        "  cursor  ",
			wantPlatform: Cursor,
			wantScopes:   []SkillScope{},
			wantErr:      false,
		},
		{
			name:         "whitespace - scope with spaces",
			input:        "cursor: repo ",
			wantPlatform: Cursor,
			wantScopes:   []SkillScope{ScopeRepo},
			wantErr:      false,
		},
		{
			name:         "whitespace - multiple scopes with spaces",
			input:        "cursor: repo , user ",
			wantPlatform: Cursor,
			wantScopes:   []SkillScope{ScopeRepo, ScopeUser},
			wantErr:      false,
		},

		// Invalid cases
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid platform",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "invalid scope",
			input:   "cursor:invalid",
			wantErr: true,
		},
		{
			name:    "empty scope after colon",
			input:   "cursor:",
			wantErr: true,
		},
		{
			name:    "invalid platform with valid scope",
			input:   "invalid:repo",
			wantErr: true,
		},
		{
			name:    "partial invalid scope",
			input:   "cursor:repo,invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePlatformSpec(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePlatformSpec(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ParsePlatformSpec(%q) unexpected error: %v", tt.input, err)
				return
			}

			if got.Platform != tt.wantPlatform {
				t.Errorf("ParsePlatformSpec(%q).Platform = %q, want %q", tt.input, got.Platform, tt.wantPlatform)
			}

			if len(got.Scopes) != len(tt.wantScopes) {
				t.Errorf("ParsePlatformSpec(%q).Scopes = %v, want %v", tt.input, got.Scopes, tt.wantScopes)
				return
			}

			for i, scope := range got.Scopes {
				if scope != tt.wantScopes[i] {
					t.Errorf("ParsePlatformSpec(%q).Scopes[%d] = %q, want %q", tt.input, i, scope, tt.wantScopes[i])
				}
			}
		})
	}
}

func TestPlatformSpec_HasScopes(t *testing.T) {
	tests := []struct {
		name string
		spec PlatformSpec
		want bool
	}{
		{
			name: "no scopes",
			spec: PlatformSpec{Platform: Cursor, Scopes: []SkillScope{}},
			want: false,
		},
		{
			name: "nil scopes",
			spec: PlatformSpec{Platform: Cursor, Scopes: nil},
			want: false,
		},
		{
			name: "one scope",
			spec: PlatformSpec{Platform: Cursor, Scopes: []SkillScope{ScopeRepo}},
			want: true,
		},
		{
			name: "multiple scopes",
			spec: PlatformSpec{Platform: Cursor, Scopes: []SkillScope{ScopeRepo, ScopeUser}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.spec.HasScopes(); got != tt.want {
				t.Errorf("PlatformSpec.HasScopes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlatformSpec_String(t *testing.T) {
	tests := []struct {
		name string
		spec PlatformSpec
		want string
	}{
		{
			name: "platform only",
			spec: PlatformSpec{Platform: Cursor, Scopes: []SkillScope{}},
			want: "cursor",
		},
		{
			name: "single scope",
			spec: PlatformSpec{Platform: Cursor, Scopes: []SkillScope{ScopeRepo}},
			want: "cursor:repo",
		},
		{
			name: "multiple scopes",
			spec: PlatformSpec{Platform: ClaudeCode, Scopes: []SkillScope{ScopeRepo, ScopeUser}},
			want: "claude-code:repo,user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.spec.String(); got != tt.want {
				t.Errorf("PlatformSpec.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPlatformSpec_ValidateAsTarget(t *testing.T) {
	tests := []struct {
		name    string
		spec    PlatformSpec
		wantErr bool
	}{
		{
			name:    "no scope - valid",
			spec:    PlatformSpec{Platform: Cursor, Scopes: []SkillScope{}},
			wantErr: false,
		},
		{
			name:    "repo scope - valid",
			spec:    PlatformSpec{Platform: Cursor, Scopes: []SkillScope{ScopeRepo}},
			wantErr: false,
		},
		{
			name:    "user scope - valid",
			spec:    PlatformSpec{Platform: Cursor, Scopes: []SkillScope{ScopeUser}},
			wantErr: false,
		},
		{
			name:    "admin scope - invalid",
			spec:    PlatformSpec{Platform: Cursor, Scopes: []SkillScope{ScopeAdmin}},
			wantErr: true,
		},
		{
			name:    "system scope - invalid",
			spec:    PlatformSpec{Platform: Cursor, Scopes: []SkillScope{ScopeSystem}},
			wantErr: true,
		},
		{
			name:    "multiple scopes - invalid",
			spec:    PlatformSpec{Platform: Cursor, Scopes: []SkillScope{ScopeRepo, ScopeUser}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.ValidateAsTarget()
			if (err != nil) != tt.wantErr {
				t.Errorf("PlatformSpec.ValidateAsTarget() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPlatformSpec_TargetScope(t *testing.T) {
	tests := []struct {
		name string
		spec PlatformSpec
		want SkillScope
	}{
		{
			name: "no scope - defaults to user",
			spec: PlatformSpec{Platform: Cursor, Scopes: []SkillScope{}},
			want: ScopeUser,
		},
		{
			name: "repo scope",
			spec: PlatformSpec{Platform: Cursor, Scopes: []SkillScope{ScopeRepo}},
			want: ScopeRepo,
		},
		{
			name: "user scope",
			spec: PlatformSpec{Platform: Cursor, Scopes: []SkillScope{ScopeUser}},
			want: ScopeUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.spec.TargetScope(); got != tt.want {
				t.Errorf("PlatformSpec.TargetScope() = %v, want %v", got, tt.want)
			}
		})
	}
}
