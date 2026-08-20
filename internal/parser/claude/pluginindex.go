// Package claude implements the Parser interface for Claude Code skills.
package claude

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
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
	// latestByPluginKey stores the preferred (latest) installation per plugin key
	latestByPluginKey map[string]*PluginIndexEntry
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
// LoadPluginIndex constructs a PluginIndex populated from the installed_plugins.json manifest.
// If the manifest cannot be read or parsed, it returns an empty index.
// Install paths are normalized and the index records the preferred (newest) installation for each plugin key; disabled installations are ignored.
func LoadPluginIndex() *PluginIndex {
	index := &PluginIndex{
		byInstallPath:     make(map[string]*PluginIndexEntry),
		latestByPluginKey: make(map[string]*PluginIndexEntry),
	}

	pluginsPath := util.ClaudeInstalledPluginsPath()

	// #nosec G304 - path is from trusted source (util package)
	data, err := os.ReadFile(pluginsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logging.Debug(
				"failed to read installed_plugins.json",
				logging.Path(pluginsPath),
				logging.Err(err),
			)
		}
		return index
	}

	var manifest InstalledPluginsFile
	if err := json.Unmarshal(data, &manifest); err != nil {
		logging.Warn(
			"failed to parse installed_plugins.json",
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
				logging.Debug(
					"skipping disabled plugin",
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
			if existing, ok := index.latestByPluginKey[pluginKey]; !ok || isVersionNewer(entry.Version, existing.Version) {
				index.latestByPluginKey[pluginKey] = entry
			}
		}
	}

	logging.Debug(
		"loaded plugin index",
		logging.Count(len(index.byInstallPath)),
	)

	return index
}

// entriesForParsing returns one preferred entry per plugin key.
// It prefers the latest semver installation for each plugin.
func (idx *PluginIndex) entriesForParsing() []*PluginIndexEntry {
	if idx == nil {
		return nil
	}
	if len(idx.latestByPluginKey) > 0 {
		entries := make([]*PluginIndexEntry, 0, len(idx.latestByPluginKey))
		for _, entry := range idx.latestByPluginKey {
			entries = append(entries, entry)
		}
		return entries
	}

	// Backward compatibility for tests constructing PluginIndex manually.
	latest := make(map[string]*PluginIndexEntry)
	for _, entry := range idx.byInstallPath {
		if entry == nil {
			continue
		}
		if existing, ok := latest[entry.PluginKey]; !ok || isVersionNewer(entry.Version, existing.Version) {
			latest[entry.PluginKey] = entry
		}
	}

	entries := make([]*PluginIndexEntry, 0, len(latest))
	for _, entry := range latest {
		entries = append(entries, entry)
	}
	return entries
}

// Entries returns one preferred enabled installation for each plugin key.
// The returned values are copies and callers can modify them safely.
func (idx *PluginIndex) Entries() []PluginIndexEntry {
	entries := idx.entriesForParsing()
	result := make([]PluginIndexEntry, 0, len(entries))
	for _, entry := range entries {
		if entry != nil {
			result = append(result, *entry)
		}
	}
	return result
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

// isVersionNewer compares plugin versions and returns true when candidate should
// be preferred over current. Semver versions are compared numerically; non-semver
// strings fall back to lexical comparison.
func isVersionNewer(candidate, current string) bool {
	if current == "" {
		return true
	}
	if candidate == "" {
		return false
	}
	cMajor, cMinor, cPatch, cPre, cOK := parseSemver(candidate)
	oMajor, oMinor, oPatch, oPre, oOK := parseSemver(current)

	if cOK && oOK {
		if cMajor != oMajor {
			return cMajor > oMajor
		}
		if cMinor != oMinor {
			return cMinor > oMinor
		}
		if cPatch != oPatch {
			return cPatch > oPatch
		}
		// Release versions are newer than prerelease versions.
		if cPre == "" && oPre != "" {
			return true
		}
		if cPre != "" && oPre == "" {
			return false
		}
		return compareSemverPrerelease(cPre, oPre) > 0
	}

	if cOK && !oOK {
		return true
	}
	if !cOK && oOK {
		return false
	}

	return candidate > current
}

// compareSemverPrerelease compares two semver prerelease strings following
// semver 2.0 precedence rules. Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Identifiers are split on '.', numeric identifiers are compared as integers,
// and numeric identifiers always have lower precedence than alphanumeric ones.
func compareSemverPrerelease(a, b string) int {
	if a == b {
		return 0
	}

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	n := min(len(aParts), len(bParts))

	for i := range n {
		aNum, aIsNum := strconv.Atoi(aParts[i])
		bNum, bIsNum := strconv.Atoi(bParts[i])

		switch {
		case aIsNum == nil && bIsNum == nil:
			// Both numeric: compare as integers.
			if aNum < bNum {
				return -1
			}
			if aNum > bNum {
				return 1
			}
		case aIsNum == nil && bIsNum != nil:
			// Numeric identifiers have lower precedence than alphanumeric.
			return -1
		case aIsNum != nil && bIsNum == nil:
			return 1
		default:
			// Both alphanumeric: compare lexically.
			if aParts[i] < bParts[i] {
				return -1
			}
			if aParts[i] > bParts[i] {
				return 1
			}
		}
	}

	// All compared identifiers are equal; the version with more identifiers
	// has higher precedence.
	if len(aParts) < len(bParts) {
		return -1
	}
	if len(aParts) > len(bParts) {
		return 1
	}
	return 0
}

func parseSemver(v string) (major, minor, patch int, pre string, ok bool) {
	normalized := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V"))
	if normalized == "" {
		return 0, 0, 0, "", false
	}

	// Strip build metadata (semver 2.0: ignored for precedence).
	if idx := strings.Index(normalized, "+"); idx >= 0 {
		normalized = normalized[:idx]
	}

	parts := strings.SplitN(normalized, "-", 2)
	core := parts[0]
	if len(parts) == 2 {
		pre = parts[1]
	}

	segments := strings.Split(core, ".")
	if len(segments) < 1 || len(segments) > 3 {
		return 0, 0, 0, "", false
	}

	values := []int{0, 0, 0}
	for i, segment := range segments {
		if segment == "" {
			return 0, 0, 0, "", false
		}
		n, err := strconv.Atoi(segment)
		if err != nil {
			return 0, 0, 0, "", false
		}
		values[i] = n
	}

	return values[0], values[1], values[2], pre, true
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
