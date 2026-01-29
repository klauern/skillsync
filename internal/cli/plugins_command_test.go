package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/util"
)

func TestParsePluginKey(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantPluginName  string
		wantMarketplace string
	}{
		{
			name:            "standard format",
			input:           "commits@klauern-skills",
			wantPluginName:  "commits",
			wantMarketplace: "klauern-skills",
		},
		{
			name:            "no marketplace",
			input:           "standalone-plugin",
			wantPluginName:  "standalone-plugin",
			wantMarketplace: "",
		},
		{
			name:            "multiple @ signs",
			input:           "plugin@market@place",
			wantPluginName:  "plugin",
			wantMarketplace: "market@place",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPluginName, gotMarketplace := parsePluginKey(tt.input)
			if gotPluginName != tt.wantPluginName {
				t.Errorf("parsePluginKey() pluginName = %v, want %v", gotPluginName, tt.wantPluginName)
			}
			if gotMarketplace != tt.wantMarketplace {
				t.Errorf("parsePluginKey() marketplace = %v, want %v", gotMarketplace, tt.wantMarketplace)
			}
		})
	}
}

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{
			name:  "no truncation needed",
			input: "short",
			width: 10,
			want:  "short",
		},
		{
			name:  "exact width",
			input: "exactly10c",
			width: 10,
			want:  "exactly10c",
		},
		{
			name:  "truncate with ellipsis",
			input: "this is a very long string that needs truncation",
			width: 20,
			want:  "this is a very lo...",
		},
		{
			name:  "width too small for ellipsis",
			input: "test",
			width: 2,
			want:  "te",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateStr(tt.input, tt.width)
			if got != tt.want {
				t.Errorf("truncateStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListInstalledPlugins(t *testing.T) {
	// Create a temporary installed_plugins.json for testing
	tmpDir := t.TempDir()
	pluginsPath := filepath.Join(tmpDir, "installed_plugins.json")

	// Create test manifest
	manifest := claude.InstalledPluginsFile{
		Version: 1,
		Plugins: map[string][]claude.PluginInstallation{
			"test-plugin@test-marketplace": {
				{
					Scope:       "plugin",
					InstallPath: filepath.Join(tmpDir, "plugins", "test"),
					Version:     "1.0.0",
				},
			},
		},
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal test manifest: %v", err)
	}

	if err := os.WriteFile(pluginsPath, data, 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	// Mock util.ClaudeInstalledPluginsPath to return our test file
	// Note: This test would need util package to support test hooks
	// For now, we'll skip actual execution and just verify the parsing logic

	t.Run("plugin key parsing", func(t *testing.T) {
		pluginName, marketplace := parsePluginKey("test-plugin@test-marketplace")
		if pluginName != "test-plugin" {
			t.Errorf("expected plugin name 'test-plugin', got %v", pluginName)
		}
		if marketplace != "test-marketplace" {
			t.Errorf("expected marketplace 'test-marketplace', got %v", marketplace)
		}
	})
}

func TestPluginEntry(t *testing.T) {
	entry := pluginEntry{
		PluginKey:   "test@marketplace",
		PluginName:  "test",
		Marketplace: "marketplace",
		Version:     "1.0.0",
		InstallPath: "/path/to/plugin",
	}

	// Test JSON marshaling
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal plugin entry: %v", err)
	}

	// Test JSON unmarshaling
	var decoded pluginEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal plugin entry: %v", err)
	}

	if decoded.PluginKey != entry.PluginKey {
		t.Errorf("PluginKey mismatch: got %v, want %v", decoded.PluginKey, entry.PluginKey)
	}
	if decoded.Version != entry.Version {
		t.Errorf("Version mismatch: got %v, want %v", decoded.Version, entry.Version)
	}
}

func TestPluginsCommandIntegration(t *testing.T) {
	// Skip if installed_plugins.json doesn't exist
	pluginsPath := util.ClaudeInstalledPluginsPath()
	if _, err := os.Stat(pluginsPath); os.IsNotExist(err) {
		t.Skip("Skipping integration test: installed_plugins.json not found")
	}

	t.Run("load and parse installed plugins", func(t *testing.T) {
		// This is an integration test that uses the actual filesystem
		// #nosec G304 - path is from trusted source
		data, err := os.ReadFile(pluginsPath)
		if err != nil {
			t.Fatalf("failed to read installed plugins: %v", err)
		}

		var manifest claude.InstalledPluginsFile
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatalf("failed to parse installed plugins: %v", err)
		}

		if len(manifest.Plugins) == 0 {
			t.Skip("No plugins installed, skipping test")
		}

		// Verify we can iterate and parse plugin keys
		for pluginKey := range manifest.Plugins {
			pluginName, marketplace := parsePluginKey(pluginKey)
			if pluginName == "" {
				t.Errorf("failed to parse plugin name from key: %s", pluginKey)
			}
			// marketplace can be empty for some plugins
			_ = marketplace
		}
	})
}
