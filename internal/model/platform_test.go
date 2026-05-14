package model

import "testing"

func TestPlatformValidation(t *testing.T) {
	tests := map[string]struct {
		platform Platform
		valid    bool
	}{
		"claude code valid": {platform: ClaudeCode, valid: true},
		"copilot valid":     {platform: Copilot, valid: true},
		"cursor valid":      {platform: Cursor, valid: true},
		"codex valid":       {platform: Codex, valid: true},
		"gemini valid":      {platform: Gemini, valid: true},
		"pi.dev valid":      {platform: PiDev, valid: true},
		"empty invalid":     {platform: "", valid: false},
		"unknown invalid":   {platform: "unknown", valid: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.platform.IsValid()
			if got != tt.valid {
				t.Errorf("Platform(%q).IsValid() = %v, want %v", tt.platform, got, tt.valid)
			}
		})
	}
}

func TestAllPlatforms(t *testing.T) {
	platforms := AllPlatforms()

	if len(platforms) != 6 {
		t.Errorf("AllPlatforms() returned %d platforms, want 6", len(platforms))
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
		"copilot":     {platform: Copilot, want: "cop"},
		"cursor":      {platform: Cursor, want: "cur"},
		"codex":       {platform: Codex, want: "cdx"},
		"gemini":      {platform: Gemini, want: "gem"},
		"pi.dev":      {platform: PiDev, want: "pi"},
		"unknown":     {platform: "unknown", want: "unknown"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.platform.Short()
			if got != tt.want {
				t.Errorf("Platform(%q).Short() = %q, want %q", tt.platform, got, tt.want)
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
		"copilot":         {platform: Copilot, want: "github"},
		"cursor":          {platform: Cursor, want: "cursor"},
		"codex":           {platform: Codex, want: "codex"},
		"gemini":          {platform: Gemini, want: "gemini"},
		"pi.dev":          {platform: PiDev, want: "pi/agent"},
		"unknown returns": {platform: "unknown", want: "unknown"},
		"empty":           {platform: "", want: ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.platform.ConfigDir()
			if got != tt.want {
				t.Errorf("Platform(%q).ConfigDir() = %q, want %q", tt.platform, got, tt.want)
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
		"copilot exact":         {input: "copilot", want: Copilot, wantErr: false},
		"github-copilot alias":  {input: "github-copilot", want: Copilot, wantErr: false},
		"cursor exact":          {input: "cursor", want: Cursor, wantErr: false},
		"codex exact":           {input: "codex", want: Codex, wantErr: false},
		"pi agent exact":        {input: "pi-agent", want: PiDev, wantErr: false},
		"pi shorthand":          {input: "pi", want: PiDev, wantErr: false},
		"pia shorthand":         {input: "pia", want: PiDev, wantErr: false},
		"gemini exact":          {input: "gemini", want: Gemini, wantErr: false},
		"pi.dev exact":          {input: "pi.dev", want: PiDev, wantErr: false},
		"pi-dev alias":          {input: "pi-dev", want: PiDev, wantErr: false},
		"pidev normalized":      {input: "pidev", want: PiDev, wantErr: false},
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
