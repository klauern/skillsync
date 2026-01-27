package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePluginKey(t *testing.T) {
	tests := map[string]struct {
		key             string
		wantPluginName  string
		wantMarketplace string
	}{
		"full key": {
			key:             "commits@klauern-skills",
			wantPluginName:  "commits",
			wantMarketplace: "klauern-skills",
		},
		"no marketplace": {
			key:             "my-plugin",
			wantPluginName:  "my-plugin",
			wantMarketplace: "",
		},
		"empty key": {
			key:             "",
			wantPluginName:  "",
			wantMarketplace: "",
		},
		"multiple @ signs": {
			key:             "plugin@marketplace@extra",
			wantPluginName:  "plugin",
			wantMarketplace: "marketplace@extra",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotPlugin, gotMarketplace := parsePluginKey(tt.key)
			if gotPlugin != tt.wantPluginName {
				t.Errorf("parsePluginKey(%q) plugin = %q, want %q", tt.key, gotPlugin, tt.wantPluginName)
			}
			if gotMarketplace != tt.wantMarketplace {
				t.Errorf("parsePluginKey(%q) marketplace = %q, want %q", tt.key, gotMarketplace, tt.wantMarketplace)
			}
		})
	}
}

func TestExtractMarketplaceFromDevPath(t *testing.T) {
	tests := map[string]struct {
		path string
		want string
	}{
		"klauern-skills": {
			path: "/Users/nklauer/dev/klauern-skills/plugins/commits/conventional-commits",
			want: "klauern-skills",
		},
		"go project": {
			path: "/Users/nklauer/dev/go/beads/examples/claude-code-skill",
			want: "beads",
		},
		"src directory": {
			path: "/Users/nklauer/dev/src/my-project/skills/my-skill",
			want: "my-project",
		},
		"no dev directory": {
			path: "/opt/skills/my-skill",
			want: "",
		},
		"dev at end": {
			path: "/Users/nklauer/dev",
			want: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := extractMarketplaceFromDevPath(tt.path)
			if got != tt.want {
				t.Errorf("extractMarketplaceFromDevPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestDetectPluginSource_NotSymlink(t *testing.T) {
	// Create a regular directory (not a symlink)
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")
	// #nosec G301 - test directory
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Should return nil for non-symlink
	pluginInfo := DetectPluginSource(skillDir, nil)
	if pluginInfo != nil {
		t.Errorf("expected nil for non-symlink, got %+v", pluginInfo)
	}
}

func TestDetectPluginSource_DevSymlink(t *testing.T) {
	// Create source and symlink
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "dev", "klauern-skills", "plugins", "commits", "conventional-commits")
	symlinkDir := filepath.Join(tmpDir, "skills", "conventional-commits")

	// Create source directory
	// #nosec G301 - test directory
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}

	// Create skills directory
	// #nosec G301 - test directory
	if err := os.MkdirAll(filepath.Dir(symlinkDir), 0o755); err != nil {
		t.Fatalf("failed to create skills directory: %v", err)
	}

	// Create symlink
	if err := os.Symlink(sourceDir, symlinkDir); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Detect plugin source
	pluginInfo := DetectPluginSource(symlinkDir, nil)
	if pluginInfo == nil {
		t.Fatal("expected PluginInfo for symlink, got nil")
	}

	if !pluginInfo.IsDev {
		t.Error("expected IsDev to be true for dev symlink")
	}

	if pluginInfo.Marketplace != "klauern-skills" {
		t.Errorf("Marketplace = %q, want %q", pluginInfo.Marketplace, "klauern-skills")
	}

	if pluginInfo.SymlinkTarget == "" {
		t.Error("SymlinkTarget should not be empty")
	}
}

func TestPluginIndex_LookupByPath(t *testing.T) {
	index := &PluginIndex{
		byInstallPath: map[string]*PluginIndexEntry{
			"/home/user/.claude/plugins/cache/klauern-skills/commits/1.1.0": {
				PluginKey:   "commits@klauern-skills",
				PluginName:  "commits",
				Marketplace: "klauern-skills",
				Version:     "1.1.0",
				InstallPath: "/home/user/.claude/plugins/cache/klauern-skills/commits/1.1.0",
			},
		},
	}

	t.Run("exact match", func(t *testing.T) {
		entry := index.LookupByPath("/home/user/.claude/plugins/cache/klauern-skills/commits/1.1.0")
		if entry == nil {
			t.Fatal("expected entry, got nil")
		}
		if entry.PluginKey != "commits@klauern-skills" {
			t.Errorf("PluginKey = %q, want %q", entry.PluginKey, "commits@klauern-skills")
		}
	})

	t.Run("not found", func(t *testing.T) {
		entry := index.LookupByPath("/home/user/.claude/plugins/cache/other/1.0.0")
		if entry != nil {
			t.Errorf("expected nil for unknown path, got %+v", entry)
		}
	})
}

func TestPluginIndex_LookupByPathPrefix(t *testing.T) {
	index := &PluginIndex{
		byInstallPath: map[string]*PluginIndexEntry{
			"/home/user/.claude/plugins/cache/klauern-skills/commits/1.1.0": {
				PluginKey:   "commits@klauern-skills",
				PluginName:  "commits",
				Marketplace: "klauern-skills",
				Version:     "1.1.0",
				InstallPath: "/home/user/.claude/plugins/cache/klauern-skills/commits/1.1.0",
			},
		},
	}

	t.Run("nested path matches prefix", func(t *testing.T) {
		entry := index.LookupByPathPrefix("/home/user/.claude/plugins/cache/klauern-skills/commits/1.1.0/conventional-commits")
		if entry == nil {
			t.Fatal("expected entry, got nil")
		}
		if entry.PluginKey != "commits@klauern-skills" {
			t.Errorf("PluginKey = %q, want %q", entry.PluginKey, "commits@klauern-skills")
		}
	})

	t.Run("exact match also works", func(t *testing.T) {
		entry := index.LookupByPathPrefix("/home/user/.claude/plugins/cache/klauern-skills/commits/1.1.0")
		if entry == nil {
			t.Fatal("expected entry, got nil")
		}
	})

	t.Run("no match", func(t *testing.T) {
		entry := index.LookupByPathPrefix("/home/user/.claude/skills/local-skill")
		if entry != nil {
			t.Errorf("expected nil for non-matching path, got %+v", entry)
		}
	})
}

func TestLoadPluginIndex_NonexistentFile(t *testing.T) {
	// LoadPluginIndex should return empty index for nonexistent file
	// (relies on util.ClaudeInstalledPluginsPath() returning a non-existent path in tests)
	// This test mainly verifies it doesn't panic
	index := LoadPluginIndex()
	if index == nil {
		t.Error("expected non-nil index even when file doesn't exist")
	}
}
