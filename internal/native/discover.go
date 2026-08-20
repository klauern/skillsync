package native

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/claude"
)

// ClaudeDiscoverer converts the Claude plugin index into native installation records.
type ClaudeDiscoverer struct {
	Index *claude.PluginIndex
}

// DiscoverNativePackages returns installed Claude plugins with their install provenance.
func (d ClaudeDiscoverer) DiscoverNativePackages(_ context.Context, platform model.Platform) ([]model.NativePackage, error) {
	if platform != model.ClaudeCode {
		return nil, fmt.Errorf("claude native discovery does not support %s", platform)
	}
	index := d.Index
	if index == nil {
		index = claude.LoadPluginIndex()
	}
	entries := index.Entries()
	packages := make([]model.NativePackage, 0, len(entries))
	for _, entry := range entries {
		packages = append(packages, model.NativePackage{
			Name: entry.PluginKey, Kind: model.NativePackagePlugin, Platform: model.ClaudeCode,
			Provenance: model.NativeProvenanceMarketplace, Version: entry.Version,
			Source: entry.Marketplace, Path: entry.InstallPath, Scope: entry.Scope, Enabled: entry.Enabled,
		})
	}
	sort.Slice(packages, func(i, j int) bool { return packages[i].Key() < packages[j].Key() })
	return packages, nil
}

// GeminiDiscoverer reads extension manifests below one Gemini extensions directory.
type GeminiDiscoverer struct {
	Root string
}

type geminiExtensionManifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
}

// DiscoverNativePackages returns valid Gemini extension manifests.
func (d GeminiDiscoverer) DiscoverNativePackages(_ context.Context, platform model.Platform) ([]model.NativePackage, error) {
	if platform != model.Gemini {
		return nil, fmt.Errorf("gemini native discovery does not support %s", platform)
	}
	entries, err := os.ReadDir(d.Root)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.NativePackage{}, nil
		}
		return nil, fmt.Errorf("read Gemini extensions directory: %w", err)
	}
	packages := make([]model.NativePackage, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(d.Root, entry.Name(), "gemini-extension.json")
		// #nosec G304 -- the path is constrained to an entry returned by ReadDir.
		data, readErr := os.ReadFile(manifestPath)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return nil, fmt.Errorf("read Gemini extension manifest %q: %w", manifestPath, readErr)
		}
		var manifest geminiExtensionManifest
		if unmarshalErr := json.Unmarshal(data, &manifest); unmarshalErr != nil {
			return nil, fmt.Errorf("parse Gemini extension manifest %q: %w", manifestPath, unmarshalErr)
		}
		name := strings.TrimSpace(manifest.Name)
		if name == "" {
			name = entry.Name()
		}
		provenance := model.NativeProvenanceLocal
		if strings.TrimSpace(manifest.Source) != "" {
			provenance = model.NativeProvenanceRepository
		}
		packages = append(packages, model.NativePackage{
			Name: name, Kind: model.NativePackageExtension, Platform: model.Gemini,
			Provenance: provenance, Version: manifest.Version, Source: manifest.Source,
			Path: filepath.Join(d.Root, entry.Name()), Scope: "user", Enabled: true,
			Metadata: map[string]string{"manifest": manifestPath},
		})
	}
	sort.Slice(packages, func(i, j int) bool { return packages[i].Key() < packages[j].Key() })
	return packages, nil
}
