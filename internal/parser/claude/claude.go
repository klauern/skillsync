// Package claude implements the Parser interface for Claude Code skills.
package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/parser/skills"
	"github.com/klauern/skillsync/internal/util"
)

// Parser implements the parser.Parser interface for Claude Code skills
type Parser struct {
	basePath    string
	pluginIndex *PluginIndex
}

// New creates a new Claude Code parser
// If basePath is empty, uses the default Claude Code skills directory (~/.claude/skills)
// The parser supports both the new Agent Skills Standard (SKILL.md) format and
// legacy .claude/skills format with .md files.
func New(basePath string) *Parser {
	if basePath == "" {
		basePath = util.ClaudeCodeSkillsPath()
	}
	return &Parser{
		basePath:    basePath,
		pluginIndex: LoadPluginIndex(),
	}
}

// NewCachePluginsParser creates a new Claude Code parser for the plugin cache directory
// If basePath is empty, uses the default Claude Code plugin cache directory (~/.claude/plugins/cache/)
// This parser is used to discover skills from installed Claude Code plugins.
func NewCachePluginsParser(basePath string) *Parser {
	if basePath == "" {
		basePath = util.ClaudePluginCachePath()
	}
	return &Parser{
		basePath:    basePath,
		pluginIndex: LoadPluginIndex(),
	}
}

// Parse parses Claude Code skills from markdown files with YAML frontmatter
// Supports both:
// 1. Agent Skills Standard: SKILL.md files in subdirectories (takes precedence)
// 2. Legacy format: .md files with optional frontmatter
func (p *Parser) Parse() ([]model.Skill, error) {
	// Check if the base path exists
	if _, err := os.Stat(p.basePath); os.IsNotExist(err) {
		logging.Debug("skills directory not found",
			logging.Platform(string(p.Platform())),
			logging.Path(p.basePath),
		)
		return []model.Skill{}, nil
	}

	var allSkills []model.Skill
	seenNames := make(map[string]bool)

	// First, parse SKILL.md files (Agent Skills Standard format)
	// These take precedence over legacy format when names collide
	skillsParser := skills.New(p.basePath, p.Platform())
	agentSkills, err := skillsParser.Parse()
	if err != nil {
		logging.Warn("failed to parse SKILL.md files",
			logging.Platform(string(p.Platform())),
			logging.Path(p.basePath),
			logging.Err(err),
		)
	} else {
		for _, skill := range agentSkills {
			// Detect if this skill is from a plugin symlink
			skillDir := filepath.Dir(skill.Path)
			if pluginInfo := DetectPluginSource(skillDir, p.pluginIndex); pluginInfo != nil {
				skill.PluginInfo = pluginInfo
			}
			seenNames[skill.Name] = true
			allSkills = append(allSkills, skill)
		}
	}

	// Then, discover legacy skill files - Claude Code uses .md files
	patterns := []string{"*.md", "**/*.md"}
	files, err := parser.DiscoverFiles(p.basePath, patterns)
	if err != nil {
		logging.Error("failed to discover skill files",
			logging.Platform(string(p.Platform())),
			logging.Path(p.basePath),
			logging.Err(err),
		)
		return nil, fmt.Errorf("failed to discover skill files in %q: %w", p.basePath, err)
	}

	// Filter out SKILL.md files (already parsed by skills parser)
	var legacyFiles []string
	for _, f := range files {
		if !strings.HasSuffix(f, "SKILL.md") {
			legacyFiles = append(legacyFiles, f)
		}
	}

	logging.Debug("discovered legacy skill files",
		logging.Platform(string(p.Platform())),
		logging.Path(p.basePath),
		logging.Count(len(legacyFiles)),
	)

	// Parse each legacy skill file
	for _, filePath := range legacyFiles {
		skill, err := p.parseSkillFile(filePath)
		if err != nil {
			logging.Warn("failed to parse skill file",
				logging.Platform(string(p.Platform())),
				logging.Path(filePath),
				logging.Err(err),
			)
			continue
		}
		// Skip if a SKILL.md with the same name was already parsed
		if seenNames[skill.Name] {
			logging.Debug("skipping legacy skill, SKILL.md version takes precedence",
				logging.Skill(skill.Name),
				logging.Path(filePath),
			)
			continue
		}

		// Detect if this skill is from a plugin symlink
		// For legacy skills, check the skill directory (may be same as basePath)
		skillDir := filepath.Dir(filePath)
		if pluginInfo := DetectPluginSource(skillDir, p.pluginIndex); pluginInfo != nil {
			skill.PluginInfo = pluginInfo
		}

		seenNames[skill.Name] = true
		allSkills = append(allSkills, skill)
	}

	logging.Debug("completed parsing skills",
		logging.Platform(string(p.Platform())),
		logging.Count(len(allSkills)),
	)

	return allSkills, nil
}

// parseSkillFile parses a single Claude Code skill file
func (p *Parser) parseSkillFile(filePath string) (model.Skill, error) {
	// Read file content
	// #nosec G304 - filePath is validated through directory traversal from basePath
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

		// Store all other frontmatter fields in metadata
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

	// If no name in frontmatter, derive from filename
	if name == "" {
		base := filepath.Base(filePath)
		name = base[:len(base)-len(filepath.Ext(base))]
	}

	// Validate skill name
	if err := parser.ValidateSkillName(name); err != nil {
		return model.Skill{}, fmt.Errorf("invalid skill name %q in %q: %w", name, filePath, err)
	}

	// Get file modification time
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	// Normalize content
	normalizedContent := parser.NormalizeContent(result.Content)

	// Build and return the skill
	skill := model.Skill{
		Name:        name,
		Description: description,
		Platform:    p.Platform(),
		Path:        filePath,
		Tools:       tools,
		Metadata:    metadata,
		Content:     normalizedContent,
		ModifiedAt:  fileInfo.ModTime(),
	}

	return skill, nil
}

// Platform returns the platform identifier for Claude Code
func (p *Parser) Platform() model.Platform {
	return model.ClaudeCode
}

// DefaultPath returns the default path for Claude Code skills
func (p *Parser) DefaultPath() string {
	return util.ClaudeCodeSkillsPath()
}
