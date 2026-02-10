// Package claude implements the Parser interface for Claude Code skills.
package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
	}

	// Collect all skill directories to exclude their contents from legacy parsing
	// This prevents reference files (patterns/, references/, etc.) from being treated as skills
	skillDirs := make(map[string]bool)
	for _, skill := range agentSkills {
		// Detect if this skill is from a plugin symlink
		skillDir := filepath.Dir(skill.Path)
		if pluginInfo := DetectPluginSource(skillDir, p.pluginIndex); pluginInfo != nil {
			skill.PluginInfo = pluginInfo
		}
		seenNames[skill.Name] = true
		skillDirs[skillDir] = true
		allSkills = append(allSkills, skill)
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

	// Filter out SKILL.md files and files inside skill directories
	// This prevents reference files (patterns/, references/, templates/, etc.) from being treated as skills
	var legacyFiles []string
	for _, f := range files {
		// Skip SKILL.md files (case-insensitive)
		base := filepath.Base(f)
		if strings.EqualFold(base, "SKILL.md") {
			continue
		}
		// Skip files inside skill directories
		if isInsideSkillDir(f, skillDirs) {
			logging.Debug("skipping file inside skill directory",
				logging.Path(f),
			)
			continue
		}
		legacyFiles = append(legacyFiles, f)
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
	var name, description, trigger string
	var tools []string
	metadata := make(map[string]string)
	skillType := model.SkillTypeSkill
	isCommandPath := isClaudeCommandFile(filePath)
	hasExplicitName := false
	commandMetadataHint := false

	if result.HasFrontmatter {
		fm, err := parser.ParseYAMLFrontmatter(result.Frontmatter)
		if err != nil {
			return model.Skill{}, fmt.Errorf("failed to parse frontmatter in %q: %w", filePath, err)
		}

		// Extract name
		if nameVal, ok := fm["name"]; ok {
			if nameStr, ok := nameVal.(string); ok {
				name = nameStr
				hasExplicitName = true
			}
		}

		// Extract description
		if descVal, ok := fm["description"]; ok {
			if descStr, ok := descVal.(string); ok {
				description = descStr
			}
		}

		// Extract tool allowlist.
		// Claude command files commonly use `allowed-tools`, while skills use `tools`.
		tools = extractTools(fm, "tools")
		if len(tools) == 0 {
			tools = extractTools(fm, "allowed-tools")
		}
		if _, ok := fm["allowed-tools"]; ok {
			commandMetadataHint = true
		}
		if _, ok := fm["argument-hint"]; ok {
			commandMetadataHint = true
		}
		if _, ok := fm["model"]; ok {
			commandMetadataHint = true
		}

		// Extract type and trigger (for command/prompt artifacts).
		if typeStr := extractString(fm, "type"); typeStr != "" {
			parsedType, err := model.ParseSkillType(typeStr)
			if err != nil {
				return model.Skill{}, fmt.Errorf("failed to parse type in %q: %w", filePath, err)
			}
			skillType = parsedType
		}
		trigger = extractString(fm, "trigger")
		if trigger != "" {
			commandMetadataHint = true
		}

		// Store all other frontmatter fields in metadata
		for key, val := range fm {
			if key != "name" && key != "description" && key != "tools" && key != "allowed-tools" && key != "type" && key != "trigger" {
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

	// Command files default to prompt type and filename-derived slash trigger.
	commandLike := isCommandPath && (!hasExplicitName || commandMetadataHint || skillType == model.SkillTypePrompt)
	if commandLike {
		if skillType == model.SkillTypeSkill {
			skillType = model.SkillTypePrompt
		}
		if trigger == "" {
			base := filepath.Base(filePath)
			stem := base[:len(base)-len(filepath.Ext(base))]
			trigger = "/" + stem
		}
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
		Type:        skillType,
		Trigger:     trigger,
	}

	return skill, nil
}

// Platform returns the platform identifier for Claude Code
func (p *Parser) Platform() model.Platform {
	return model.ClaudeCode
}

func extractString(fm map[string]any, key string) string {
	if val, ok := fm[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}

func extractTools(fm map[string]any, key string) []string {
	val, ok := fm[key]
	if !ok {
		return nil
	}

	switch v := val.(type) {
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if tool, ok := item.(string); ok {
				result = append(result, strings.TrimSpace(tool))
			}
		}
		return result
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		parts := strings.Split(v, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			tool := strings.TrimSpace(part)
			if tool != "" {
				result = append(result, tool)
			}
		}
		return result
	default:
		return nil
	}
}

func isClaudeCommandFile(path string) bool {
	if !strings.EqualFold(filepath.Ext(path), ".md") {
		return false
	}

	parts := strings.Split(filepath.ToSlash(path), "/")
	return slices.Contains(parts, "commands")
}

// isInsideSkillDir checks if a file path is inside any of the skill directories.
// This is used to filter out reference files (patterns/, references/, etc.) from legacy parsing.
func isInsideSkillDir(filePath string, skillDirs map[string]bool) bool {
	dir := filepath.Dir(filePath)
	// Walk up the directory tree to check if any parent is a skill directory
	for dir != "/" && dir != "." && dir != "" {
		if skillDirs[dir] {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return false
}

// DefaultPath returns the default path for Claude Code skills
func (p *Parser) DefaultPath() string {
	return util.ClaudeCodeSkillsPath()
}
