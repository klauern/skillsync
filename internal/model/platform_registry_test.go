package model

import "testing"

func TestPlatformInfoFor(t *testing.T) {
	tests := map[Platform]struct {
		name      string
		short     string
		configDir string
	}{
		ClaudeCode: {name: "claude-code", short: "cc", configDir: "claude"},
		Cursor:     {name: "cursor", short: "cur", configDir: "cursor"},
		Codex:      {name: "codex", short: "cdx", configDir: "codex"},
		PiAgent:    {name: "pi-agent", short: "pia", configDir: "agents"},
		Copilot:    {name: "copilot", short: "cop", configDir: "github"},
		Gemini:     {name: "gemini", short: "gem", configDir: "gemini"},
		PiDev:      {name: "pi.dev", short: "pi", configDir: "pi/agent"},
	}

	for platform, want := range tests {
		t.Run(string(platform), func(t *testing.T) {
			info, ok := platformInfoFor(platform)
			if !ok {
				t.Fatalf("platformInfoFor(%q) = not found", platform)
			}
			if info.Name != want.name {
				t.Fatalf("platformInfoFor(%q).Name = %q, want %q", platform, info.Name, want.name)
			}
			if info.Short != want.short {
				t.Fatalf("platformInfoFor(%q).Short = %q, want %q", platform, info.Short, want.short)
			}
			if info.ConfigDir != want.configDir {
				t.Fatalf("platformInfoFor(%q).ConfigDir = %q, want %q", platform, info.ConfigDir, want.configDir)
			}
		})
	}
}
