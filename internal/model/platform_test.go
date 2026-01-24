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
