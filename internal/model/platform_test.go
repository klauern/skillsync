package model

import "testing"

func TestPlatformValidation(t *testing.T) {
	tests := map[string]struct {
		platform Platform
		valid    bool
	}{
		"claude code valid": {platform: ClaudeCode, valid: true},
		"cursor valid":      {platform: Cursor, valid: true},
		"codex valid":       {platform: Codex, valid: true},
		"empty invalid":     {platform: "", valid: false},
		"unknown invalid":   {platform: "unknown", valid: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.platform.IsValid()
			if got != tt.valid {
				t.Errorf("Platform(%q).IsValid() = %v, want %v",
					tt.platform, got, tt.valid)
			}
		})
	}
}

func TestAllPlatforms(t *testing.T) {
	platforms := AllPlatforms()

	if len(platforms) != 3 {
		t.Errorf("AllPlatforms() returned %d platforms, want 3", len(platforms))
	}

	for _, p := range platforms {
		if !p.IsValid() {
			t.Errorf("AllPlatforms() returned invalid platform: %q", p)
		}
	}
}

func TestPlatformShort(t *testing.T) {
	tests := map[string]struct {
		platform Platform
		want     string
	}{
		"claude code": {platform: ClaudeCode, want: "cc"},
		"cursor":      {platform: Cursor, want: "cur"},
		"codex":       {platform: Codex, want: "cdx"},
		"unknown":     {platform: "unknown", want: "unknown"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.platform.Short()
			if got != tt.want {
				t.Errorf("Platform(%q).Short() = %q, want %q",
					tt.platform, got, tt.want)
			}
		})
	}
}

func TestPlatformConfigDir(t *testing.T) {
	tests := map[string]struct {
		platform Platform
		want     string
	}{
		"claude code":     {platform: ClaudeCode, want: "claude"},
		"cursor":          {platform: Cursor, want: "cursor"},
		"codex":           {platform: Codex, want: "codex"},
		"unknown returns": {platform: "unknown", want: "unknown"},
		"empty":           {platform: "", want: ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.platform.ConfigDir()
			if got != tt.want {
				t.Errorf("Platform(%q).ConfigDir() = %q, want %q",
					tt.platform, got, tt.want)
			}
		})
	}
}

func TestParsePlatform(t *testing.T) {
	tests := map[string]struct {
		input   string
		want    Platform
		wantErr bool
	}{
		"claude-code exact":     {input: "claude-code", want: ClaudeCode, wantErr: false},
		"claudecode normalized": {input: "claudecode", want: ClaudeCode, wantErr: false},
		"claude shorthand":      {input: "claude", want: ClaudeCode, wantErr: false},
		"cursor exact":          {input: "cursor", want: Cursor, wantErr: false},
		"codex exact":           {input: "codex", want: Codex, wantErr: false},
		"uppercase normalized":  {input: "CURSOR", want: Cursor, wantErr: false},
		"mixed case":            {input: "ClaudeCode", want: ClaudeCode, wantErr: false},
		"with whitespace":       {input: "  cursor  ", want: Cursor, wantErr: false},
		"unknown platform":      {input: "unknown", want: "", wantErr: true},
		"empty string":          {input: "", want: "", wantErr: true},
		"invalid name":          {input: "vscode", want: "", wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParsePlatform(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePlatform(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParsePlatform(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
