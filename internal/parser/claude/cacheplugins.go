// Package claude implements parsers for Claude Code skills and plugins.
package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	seenSkillFiles := make(map[string]bool)

	// Iterate over preferred plugin installations (latest version per plugin key)
	for _, entry := range pluginIndex.entriesForParsing() {
		// Skip if we've already processed this install path (handles duplicates)
		if seenPaths[entry.InstallPath] {
			continue
		}
		seenPaths[entry.InstallPath] = true

		// Skip orphaned plugins
		orphanedMarker := filepath.Join(entry.InstallPath, ".orphaned_at")
		if _, err := os.Stat(orphanedMarker); err == nil {
			logging.Debug(
				"skipping orphaned plugin",
				logging.Path(entry.InstallPath),
			)
			continue
		}

		// Check if the install path exists
		if _, err := os.Stat(entry.InstallPath); os.IsNotExist(err) {
			logging.Debug(
				"plugin install path does not exist",
				logging.Path(entry.InstallPath),
			)
			continue
		}

		// Discover SKILL.md files in this plugin
		pluginSkills, err := p.parsePluginDirectory(entry)
		if err != nil {
			logging.Warn(
				"failed to parse plugin",
				logging.Path(entry.InstallPath),
				logging.Err(err),
			)
			continue
		}

		for _, skill := range pluginSkills {
			fileKey := canonicalFileKey(skill.Path)
			if fileKey != "" && seenSkillFiles[fileKey] {
				continue
			}
			if fileKey != "" {
				seenSkillFiles[fileKey] = true
			}
			skills = append(skills, skill)
		}
	}

	logging.Debug(
		"discovered skills from Claude plugin cache",
		logging.Count(len(skills)),
	)

	return skills, nil
}

// parsePluginDirectory scans a plugin directory for SKILL.md files and commands/*.md files.
func (p *CachePluginsParser) parsePluginDirectory(entry *PluginIndexEntry) ([]model.Skill, error) {
	// Find all SKILL files in the plugin directory (SKILL.md, skill.md, Skill.md)
	skillPatterns := []string{
		"**/SKILL.md", "SKILL.md",
		"**/skill.md", "skill.md",
		"**/Skill.md", "Skill.md",
	}
	skillFiles, err := parser.DiscoverFiles(entry.InstallPath, skillPatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to discover skill files: %w", err)
	}
	skillFiles = deduplicateBySameFile(skillFiles)

	// Also find command files in commands/ subdirectories (plugins using legacy command format)
	commandPatterns := []string{"commands/*.md", "**/commands/*.md"}
	commandFiles, err := parser.DiscoverFiles(entry.InstallPath, commandPatterns)
	if err != nil {
		logging.Warn(
			"failed to discover command files",
			logging.Path(entry.InstallPath),
			logging.Err(err),
		)
	}
	commandFiles = deduplicateBySameFile(commandFiles)

	// Build a set of SKILL.md directories to skip duplicate command files that are
	// already covered by a SKILL.md in the same directory.
	skillDirs := make(map[string]bool)
	for _, f := range skillFiles {
		skillDirs[filepath.Dir(f)] = true
	}
	var filteredCommandFiles []string
	for _, f := range commandFiles {
		if !skillDirs[filepath.Dir(f)] {
			filteredCommandFiles = append(filteredCommandFiles, f)
		}
	}

	allFiles := append(skillFiles, filteredCommandFiles...)
	if len(allFiles) == 0 {
		logging.Debug(
			"no skill or command files found in plugin",
			logging.Path(entry.InstallPath),
		)
		return []model.Skill{}, nil
	}

	logging.Debug(
		"found skill/command files in plugin",
		logging.Path(entry.InstallPath),
		logging.Count(len(allFiles)),
	)

	var skills []model.Skill
	for _, filePath := range allFiles {
		skill, err := p.parseSkillFile(filePath, entry)
		if err != nil {
			logging.Warn(
				"failed to parse skill file",
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

		// Extract tools array — command files use "allowed-tools", skills use "tools".
		for _, toolsKey := range []string{"tools", "allowed-tools"} {
			if toolsVal, ok := fm[toolsKey]; ok {
				switch v := toolsVal.(type) {
				case []any:
					tools = make([]string, 0, len(v))
					for _, tool := range v {
						if toolStr, ok := tool.(string); ok {
							tools = append(tools, toolStr)
						}
					}
				case string:
					for _, part := range strings.Split(v, ",") {
						if t := strings.TrimSpace(part); t != "" {
							tools = append(tools, t)
						}
					}
				}
				if len(tools) > 0 {
					break
				}
			}
		}

		// Store remaining fields in metadata
		for key, val := range fm {
			if key != "name" && key != "description" && key != "tools" && key != "allowed-tools" {
				if strVal, ok := val.(string); ok {
					metadata[key] = strVal
				} else {
					metadata[key] = fmt.Sprintf("%v", val)
				}
			}
		}
	}

	// Detect whether this file is a Claude command (lives inside a commands/ directory).
	isCommand := parser.IsCommandFile(filePath)

	// Derive name from filename (for commands) or parent directory (for SKILL.md files).
	if name == "" {
		if isCommand {
			base := filepath.Base(filePath)
			name = base[:len(base)-len(filepath.Ext(base))]
		} else {
			name = filepath.Base(filepath.Dir(filePath))
		}
	}

	// Validate skill name, falling back to directory name for non-conforming plugin skills.
	if err := parser.ValidateSkillName(name); err != nil {
		fallback := filepath.Base(filepath.Dir(filePath))
		if fallback != name {
			if fallbackErr := parser.ValidateSkillName(fallback); fallbackErr == nil {
				name = fallback
			} else {
				return model.Skill{}, fmt.Errorf("invalid skill name %q in %q: %w", name, filePath, err)
			}
		} else {
			return model.Skill{}, fmt.Errorf("invalid skill name %q in %q: %w", name, filePath, err)
		}
	}

	// Add plugin metadata
	metadata["plugin"] = entry.PluginName
	metadata["marketplace"] = entry.Marketplace
	if entry.Version != "" {
		metadata["plugin_version"] = entry.Version
	}
	if entry.Scope != "" {
		metadata["install_scope"] = entry.Scope
	}
	metadata["source"] = "plugin-cache"

	// Get file modification time
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	// Normalize content
	normalizedContent := parser.NormalizeContent(result.Content)

	// For command files: default to prompt type and derive slash trigger from filename.
	skillType := model.SkillTypeSkill
	var trigger string
	if isCommand {
		skillType = model.SkillTypePrompt
		base := filepath.Base(filePath)
		stem := base[:len(base)-len(filepath.Ext(base))]
		trigger = "/" + stem
	}

	// Create PluginInfo for this skill
	pluginInfo := &model.PluginInfo{
		PluginName:   entry.PluginKey,
		Marketplace:  entry.Marketplace,
		Version:      entry.Version,
		InstallPath:  entry.InstallPath,
		IsDev:        false, // Cache plugins are never dev
		InstallScope: entry.Scope,
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
		Type:        skillType,
		Trigger:     trigger,
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

// deduplicateBySameFile removes paths that refer to the same physical file.
// Handles case-insensitive filesystems where SKILL.md, skill.md, and Skill.md
// can resolve to the same file but produce different path strings.
func deduplicateBySameFile(paths []string) []string {
	var result []string
	var resultInfo []os.FileInfo
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		dup := false
		for _, ri := range resultInfo {
			if os.SameFile(info, ri) {
				dup = true
				break
			}
		}
		if !dup {
			result = append(result, p)
			resultInfo = append(resultInfo, info)
		}
	}
	return result
}

func canonicalFileKey(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	// Use inode identity where available. Fall back to eval path.
	if abs, err := filepath.EvalSymlinks(path); err == nil && abs != "" {
		return filepath.Clean(abs) + "|" + info.Name()
	}
	return filepath.Clean(path) + "|" + info.Name()
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
