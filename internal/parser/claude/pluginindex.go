// Package claude implements the Parser interface for Claude Code skills.
package claude

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/util"
)

// PluginInstallation represents a single plugin installation entry from installed_plugins.json.
type PluginInstallation struct {
	Enabled      *bool  `json:"enabled,omitempty"` // nil means enabled (default true)
	Scope        string `json:"scope"`
	InstallPath  string `json:"installPath"`
	Version      string `json:"version"`
	InstalledAt  string `json:"installedAt"`
	LastUpdated  string `json:"lastUpdated"`
	GitCommitSha string `json:"gitCommitSha"`
}

// IsEnabled returns whether the plugin installation is enabled.
// Returns true if Enabled is nil (not specified) or explicitly true.
func (pi *PluginInstallation) IsEnabled() bool {
	return pi.Enabled == nil || *pi.Enabled
}

// InstalledPluginsFile represents the structure of installed_plugins.json.
type InstalledPluginsFile struct {
	Version int                             `json:"version"`
	Plugins map[string][]PluginInstallation `json:"plugins"`
}

// PluginIndex provides a lookup index for installed Claude Code plugins.
// It allows quick lookup by install path to determine plugin metadata.
type PluginIndex struct {
	// byInstallPath maps absolute install paths to plugin metadata
	byInstallPath map[string]*PluginIndexEntry
}

// PluginIndexEntry contains information about a single plugin from the index.
type PluginIndexEntry struct {
	// PluginKey is the full key from installed_plugins.json (e.g., "commits@klauern-skills")
	PluginKey string
	// PluginName is the plugin name without marketplace (e.g., "commits")
	PluginName string
	// Marketplace is the marketplace/repository name (e.g., "klauern-skills")
	Marketplace string
	// Version is the installed version
	Version string
	// InstallPath is the absolute installation path
	InstallPath string
	// Scope is the install scope from installed_plugins.json (e.g., "user", "project")
	Scope string
	// Enabled indicates whether this plugin installation is enabled
	Enabled bool
}

// LoadPluginIndex loads and parses the Claude Code installed plugins manifest.
// Returns an empty index if the file doesn't exist or can't be parsed.
func LoadPluginIndex() *PluginIndex {
	index := &PluginIndex{
		byInstallPath: make(map[string]*PluginIndexEntry),
	}

	pluginsPath := util.ClaudeInstalledPluginsPath()

	// #nosec G304 - path is from trusted source (util package)
	data, err := os.ReadFile(pluginsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logging.Debug("failed to read installed_plugins.json",
				logging.Path(pluginsPath),
				logging.Err(err),
			)
		}
		return index
	}

	var manifest InstalledPluginsFile
	if err := json.Unmarshal(data, &manifest); err != nil {
		logging.Warn("failed to parse installed_plugins.json",
			logging.Path(pluginsPath),
			logging.Err(err),
		)
		return index
	}

	// Build index by install path
	for pluginKey, installations := range manifest.Plugins {
		pluginName, marketplace := parsePluginKey(pluginKey)

		for _, inst := range installations {
			// Skip disabled plugins
			if !inst.IsEnabled() {
				logging.Debug("skipping disabled plugin",
					slog.String("plugin", pluginKey),
					logging.Path(inst.InstallPath),
				)
				continue
			}

			// Normalize the install path for consistent lookup
			normalizedPath := filepath.Clean(inst.InstallPath)

			entry := &PluginIndexEntry{
				PluginKey:   pluginKey,
				PluginName:  pluginName,
				Marketplace: marketplace,
				Version:     inst.Version,
				InstallPath: normalizedPath,
				Scope:       inst.Scope,
				Enabled:     inst.IsEnabled(),
			}

			index.byInstallPath[normalizedPath] = entry
		}
	}

	logging.Debug("loaded plugin index",
		logging.Count(len(index.byInstallPath)),
	)

	return index
}

// LookupByPath looks up plugin information by install path.
// Returns nil if the path is not found in the index.
func (idx *PluginIndex) LookupByPath(installPath string) *PluginIndexEntry {
	normalizedPath := filepath.Clean(installPath)
	return idx.byInstallPath[normalizedPath]
}

// LookupByPathPrefix looks up plugin information by checking if any indexed
// path is a prefix of the given path. This is useful when the actual skill
// path is nested within the plugin install directory.
func (idx *PluginIndex) LookupByPathPrefix(path string) *PluginIndexEntry {
	normalizedPath := filepath.Clean(path)

	for installPath, entry := range idx.byInstallPath {
		if strings.HasPrefix(normalizedPath, installPath+string(os.PathSeparator)) ||
			normalizedPath == installPath {
			return entry
		}
	}

	return nil
}

// parsePluginKey splits a plugin key like "commits@klauern-skills" into
// plugin name and marketplace.
func parsePluginKey(key string) (pluginName, marketplace string) {
	parts := strings.SplitN(key, "@", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return key, ""
}

// DetectPluginSource examines a skill directory path to determine if it's
// a symlink to a plugin or development directory.
// Returns PluginInfo if the path is a symlink, nil otherwise.
func DetectPluginSource(skillDirPath string, pluginIndex *PluginIndex) *model.PluginInfo {
	// Check if the skill directory is a symlink
	linkTarget, err := os.Readlink(skillDirPath)
	if err != nil {
		// Not a symlink or error reading
		return nil
	}

	// Resolve to absolute path
	var resolvedTarget string
	if filepath.IsAbs(linkTarget) {
		resolvedTarget = filepath.Clean(linkTarget)
	} else {
		// Relative symlink - resolve from parent directory
		parentDir := filepath.Dir(skillDirPath)
		resolvedTarget = filepath.Clean(filepath.Join(parentDir, linkTarget))
	}

	pluginInfo := &model.PluginInfo{
		SymlinkTarget: linkTarget,
		InstallPath:   resolvedTarget,
	}

	// Check if target is within plugin cache
	pluginCachePath := util.ClaudePluginCachePath()
	if strings.HasPrefix(resolvedTarget, pluginCachePath+string(os.PathSeparator)) {
		// This is an installed plugin
		pluginInfo.IsDev = false

		// Try to extract marketplace and version from path
		// Path format: ~/.claude/plugins/cache/{marketplace}/{plugin}/{version}/...
		relPath := strings.TrimPrefix(resolvedTarget, pluginCachePath+string(os.PathSeparator))
		parts := strings.Split(relPath, string(os.PathSeparator))
		if len(parts) >= 2 {
			pluginInfo.Marketplace = parts[0]
			if len(parts) >= 3 {
				pluginInfo.Version = parts[2]
				pluginInfo.PluginName = parts[1] + "@" + parts[0]
			}
		}

		// Try to get more accurate info from the plugin index
		if pluginIndex != nil {
			if entry := pluginIndex.LookupByPathPrefix(resolvedTarget); entry != nil {
				pluginInfo.PluginName = entry.PluginKey
				pluginInfo.Marketplace = entry.Marketplace
				pluginInfo.Version = entry.Version
			}
		}
	} else {
		// Development symlink - points outside plugin cache
		pluginInfo.IsDev = true

		// Try to identify marketplace from dev path patterns
		// Common patterns:
		// - /Users/xxx/dev/klauern-skills/plugins/...
		// - /Users/xxx/dev/go/beads/examples/...
		pluginInfo.Marketplace = extractMarketplaceFromDevPath(resolvedTarget)
	}

	return pluginInfo
}

// extractMarketplaceFromDevPath attempts to identify a marketplace name from
// a development path. Returns empty string if not identifiable.
func extractMarketplaceFromDevPath(path string) string {
	// Look for common patterns in dev paths
	parts := strings.Split(path, string(os.PathSeparator))

	// Look for paths containing "dev" followed by a project name
	for i, part := range parts {
		if part == "dev" && i+1 < len(parts) {
			// Check if next part looks like a marketplace name
			candidate := parts[i+1]
			// Skip common intermediate directories
			if candidate == "go" || candidate == "src" || candidate == "projects" {
				if i+2 < len(parts) {
					return parts[i+2]
				}
			}
			return candidate
		}
	}

	return ""
}
