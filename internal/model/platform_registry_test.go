package model

import (
	"slices"
	"testing"
)

func TestPlatformInfoFor(t *testing.T) {
	tests := []struct {
		name string
		p    Platform
		want PlatformInfo
	}{
		{
			name: "claude code",
			p:    ClaudeCode,
			want: PlatformInfo{
				Short:           "cc",
				ConfigDir:       "claude",
				DotDir:          ".claude",
				DisplayName:     "Claude Code",
				ValidExtensions: []string{".md", ".txt"},
				AllowsEmptyExt:  true,
			},
		},
		{
			name: "cursor",
			p:    Cursor,
			want: PlatformInfo{
				Short:           "cur",
				ConfigDir:       "cursor",
				DotDir:          ".cursor",
				DisplayName:     "Cursor",
				ValidExtensions: []string{".md", ".mdc"},
			},
		},
		{
			name: "codex",
			p:    Codex,
			want: PlatformInfo{
				Short:           "cdx",
				ConfigDir:       "codex",
				DotDir:          ".codex",
				DisplayName:     "Codex",
				ValidExtensions: []string{".md", ".toml"},
			},
		},
		{
			name: "pi agent",
			p:    PiDev,
			want: PlatformInfo{
				Short:           "pi",
				ConfigDir:       "pi/agent",
				DotDir:          ".pi/agent",
				DisplayName:     "Pi",
				ValidExtensions: []string{".md"},
			},
		},
		{
			name: "copilot",
			p:    Copilot,
			want: PlatformInfo{
				Short:           "cop",
				ConfigDir:       "github",
				DotDir:          ".github",
				DisplayName:     "Copilot",
				ValidExtensions: []string{".md"},
			},
		},
		{
			name: "gemini",
			p:    Gemini,
			want: PlatformInfo{
				Short:           "gem",
				ConfigDir:       "gemini",
				DotDir:          ".gemini",
				DisplayName:     "Gemini",
				ValidExtensions: []string{".md"},
			},
		},
		{
			name: "pi.dev",
			p:    PiDev,
			want: PlatformInfo{
				Short:           "pi",
				ConfigDir:       "pi/agent",
				DotDir:          ".pi/agent",
				DisplayName:     "Pi",
				ValidExtensions: []string{".md"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := PlatformInfoFor(tt.p)
			if !ok {
				t.Fatalf("PlatformInfoFor(%q) = false, want true", tt.p)
			}
			if got.Short != tt.want.Short {
				t.Errorf("PlatformInfoFor(%q).Short = %q, want %q", tt.p, got.Short, tt.want.Short)
			}
			if got.ConfigDir != tt.want.ConfigDir {
				t.Errorf("PlatformInfoFor(%q).ConfigDir = %q, want %q", tt.p, got.ConfigDir, tt.want.ConfigDir)
			}
			if got.DotDir != tt.want.DotDir {
				t.Errorf("PlatformInfoFor(%q).DotDir = %q, want %q", tt.p, got.DotDir, tt.want.DotDir)
			}
			if got.DisplayName != tt.want.DisplayName {
				t.Errorf("PlatformInfoFor(%q).DisplayName = %q, want %q", tt.p, got.DisplayName, tt.want.DisplayName)
			}
			if !slices.Equal(got.ValidExtensions, tt.want.ValidExtensions) {
				t.Errorf("PlatformInfoFor(%q).ValidExtensions = %v, want %v", tt.p, got.ValidExtensions, tt.want.ValidExtensions)
			}
			if got.AllowsEmptyExt != tt.want.AllowsEmptyExt {
				t.Errorf("PlatformInfoFor(%q).AllowsEmptyExt = %v, want %v", tt.p, got.AllowsEmptyExt, tt.want.AllowsEmptyExt)
			}
		})
	}
}

func TestPlatformInfoForUnknownPlatform(t *testing.T) {
	if _, ok := PlatformInfoFor(Platform("unknown")); ok {
		t.Fatal("PlatformInfoFor(unknown) = ok=true, want false")
	}
}
