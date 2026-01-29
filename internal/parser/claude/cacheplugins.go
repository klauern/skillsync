// Package claude implements parsers for Claude Code skills and plugins.
package claude

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/util"
)

// CachePluginsParser discovers skills from installed Claude Code plugins in ~/.claude/plugins/cache.
// It reads the installed_plugins.json manifest to enumerate installed plugins and scans
// each plugin directory for SKILL.md files.
type CachePluginsParser struct {
	cachePath   string
	pluginIndex *PluginIndex
}

// NewCachePluginsParser creates a new parser for Claude Code plugin cache.
// If cachePath is empty, uses the default path (~/.claude/plugins/cache).
func NewCachePluginsParser(cachePath string) *CachePluginsParser {
	if cachePath == "" {
		cachePath = util.ClaudePluginCachePath()
	}
	return &CachePluginsParser{cachePath: cachePath}
}

// NewCachePluginsParserWithIndex creates a new parser with a custom plugin index.
// This is useful for testing without relying on the real installed_plugins.json.
func NewCachePluginsParserWithIndex(cachePath string, index *PluginIndex) *CachePluginsParser {
	if cachePath == "" {
		cachePath = util.ClaudePluginCachePath()
	}
	return &CachePluginsParser{
		cachePath:   cachePath,
		pluginIndex: index,
	}
}

// Parse discovers skills from all installed Claude Code plugins.
// It reads the installed_plugins.json manifest and scans each plugin for SKILL.md files.
func (p *CachePluginsParser) Parse() ([]model.Skill, error) {
	// Use provided plugin index or load from default location
	pluginIndex := p.pluginIndex
	if pluginIndex == nil {
		pluginIndex = LoadPluginIndex()
	}

	// If no plugins are installed, return empty
	if len(pluginIndex.byInstallPath) == 0 {
		logging.Debug("no installed plugins found in manifest")
		return []model.Skill{}, nil
	}

	var skills []model.Skill
	seenPaths := make(map[string]bool)

	// Iterate over all installed plugins
	for _, entry := range pluginIndex.byInstallPath {
		// Skip if we've already processed this install path (handles duplicates)
		if seenPaths[entry.InstallPath] {
			continue
		}
		seenPaths[entry.InstallPath] = true

		// Check if the install path exists
		if _, err := os.Stat(entry.InstallPath); os.IsNotExist(err) {
			logging.Debug("plugin install path does not exist",
				logging.Path(entry.InstallPath),
			)
			continue
		}

		// Discover SKILL.md files in this plugin
		pluginSkills, err := p.parsePluginDirectory(entry)
		if err != nil {
			logging.Warn("failed to parse plugin",
				logging.Path(entry.InstallPath),
				logging.Err(err),
			)
			continue
		}

		skills = append(skills, pluginSkills...)
	}

	logging.Debug("discovered skills from Claude plugin cache",
		logging.Count(len(skills)),
	)

	return skills, nil
}

// parsePluginDirectory scans a plugin directory for SKILL.md files and parses them.
func (p *CachePluginsParser) parsePluginDirectory(entry *PluginIndexEntry) ([]model.Skill, error) {
	// Find all SKILL.md files in the plugin directory
	patterns := []string{"**/SKILL.md", "SKILL.md"}
	files, err := parser.DiscoverFiles(entry.InstallPath, patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to discover skill files: %w", err)
	}

	if len(files) == 0 {
		logging.Debug("no SKILL.md files found in plugin",
			logging.Path(entry.InstallPath),
		)
		return []model.Skill{}, nil
	}

	logging.Debug("found skill files in plugin",
		logging.Path(entry.InstallPath),
		logging.Count(len(files)),
	)

	var skills []model.Skill
	for _, filePath := range files {
		skill, err := p.parseSkillFile(filePath, entry)
		if err != nil {
			logging.Warn("failed to parse skill file",
				logging.Path(filePath),
				logging.Err(err),
			)
			continue
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

// parseSkillFile parses a single SKILL.md file with plugin metadata.
func (p *CachePluginsParser) parseSkillFile(filePath string, entry *PluginIndexEntry) (model.Skill, error) {
	// #nosec G304 - filePath is from trusted plugin index
	content, err := os.ReadFile(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	// Split frontmatter from content
	result := parser.SplitFrontmatter(content)

	// Extract metadata from frontmatter
	var name, description string
	var tools []string
	metadata := make(map[string]string)

	if result.HasFrontmatter {
		fm, err := parser.ParseYAMLFrontmatter(result.Frontmatter)
		if err != nil {
			return model.Skill{}, fmt.Errorf("failed to parse frontmatter in %q: %w", filePath, err)
		}

		// Extract name
		if nameVal, ok := fm["name"]; ok {
			if nameStr, ok := nameVal.(string); ok {
				name = nameStr
			}
		}

		// Extract description
		if descVal, ok := fm["description"]; ok {
			if descStr, ok := descVal.(string); ok {
				description = descStr
			}
		}

		// Extract tools array
		if toolsVal, ok := fm["tools"]; ok {
			if toolsSlice, ok := toolsVal.([]any); ok {
				tools = make([]string, 0, len(toolsSlice))
				for _, tool := range toolsSlice {
					if toolStr, ok := tool.(string); ok {
						tools = append(tools, toolStr)
					}
				}
			}
		}

		// Store remaining fields in metadata
		for key, val := range fm {
			if key != "name" && key != "description" && key != "tools" {
				if strVal, ok := val.(string); ok {
					metadata[key] = strVal
				} else {
					metadata[key] = fmt.Sprintf("%v", val)
				}
			}
		}
	}

	// Derive name from directory if not in frontmatter
	if name == "" {
		// Use the parent directory name as skill name
		name = filepath.Base(filepath.Dir(filePath))
	}

	// Validate skill name
	if err := parser.ValidateSkillName(name); err != nil {
		return model.Skill{}, fmt.Errorf("invalid skill name %q in %q: %w", name, filePath, err)
	}

	// Add plugin metadata
	metadata["plugin"] = entry.PluginName
	metadata["marketplace"] = entry.Marketplace
	if entry.Version != "" {
		metadata["plugin_version"] = entry.Version
	}
	metadata["source"] = "plugin-cache"

	// Get file modification time
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	// Normalize content
	normalizedContent := parser.NormalizeContent(result.Content)

	// Create PluginInfo for this skill
	pluginInfo := &model.PluginInfo{
		PluginName:  entry.PluginKey,
		Marketplace: entry.Marketplace,
		Version:     entry.Version,
		InstallPath: entry.InstallPath,
		IsDev:       false, // Cache plugins are never dev
	}

	return model.Skill{
		Name:        name,
		Description: description,
		Platform:    model.ClaudeCode,
		Path:        filePath,
		Tools:       tools,
		Metadata:    metadata,
		Content:     normalizedContent,
		ModifiedAt:  fileInfo.ModTime(),
		Scope:       model.ScopePlugin,
		PluginInfo:  pluginInfo,
	}, nil
}

// Platform returns the platform identifier for this parser.
func (p *CachePluginsParser) Platform() model.Platform {
	return model.ClaudeCode
}

// DefaultPath returns the default path for Claude plugin cache.
func (p *CachePluginsParser) DefaultPath() string {
	return util.ClaudePluginCachePath()
}

// AllEntries returns all plugin entries from the index (useful for testing).
func (p *CachePluginsParser) AllEntries() []*PluginIndexEntry {
	index := LoadPluginIndex()
	entries := make([]*PluginIndexEntry, 0, len(index.byInstallPath))
	for _, entry := range index.byInstallPath {
		entries = append(entries, entry)
	}
	return entries
}
