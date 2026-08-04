// Package cursor implements the Parser interface for Cursor skills/rules.
package cursor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/parser/skills"
	"github.com/klauern/skillsync/internal/util"
)

// Parser implements the parser.Parser interface for Cursor skills
type Parser struct {
	basePath string
}

// New creates a new Cursor parser
// If basePath is empty, uses the default Cursor skills directory (~/.cursor/skills)
// The parser supports both the new Agent Skills Standard (SKILL.md) format and
// legacy .cursor/rules format with .md/.mdc files.
func New(basePath string) *Parser {
	if basePath == "" {
		basePath = util.CursorSkillsPath()
	}
	return &Parser{basePath: basePath}
}

// Parse parses Cursor skills from markdown files with YAML frontmatter
// Supports both:
// 1. Legacy format: .md and .mdc files with optional globs and alwaysApply fields
// 2. Agent Skills Standard: SKILL.md files in subdirectories
func (p *Parser) Parse() ([]model.Skill, error) {
	// Check if the base path exists
	if _, err := os.Stat(p.basePath); os.IsNotExist(err) {
		logging.Debug(
			"skills directory not found",
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
		logging.Warn(
			"failed to parse SKILL.md files",
			logging.Platform(string(p.Platform())),
			logging.Path(p.basePath),
			logging.Err(err),
		)
	}

	// Collect all skill directories to exclude their contents from legacy parsing
	// This prevents reference files (patterns/, references/, etc.) from being treated as skills
	skillDirs := make(map[string]bool)
	for _, skill := range agentSkills {
		skillDir := filepath.Dir(skill.Path)
		seenNames[skill.Name] = true
		skillDirs[skillDir] = true
		allSkills = append(allSkills, skill)
	}

	// Then, discover legacy skill files - Cursor uses .md and .mdc files
	patterns := []string{"*.md", "*.mdc", "**/*.md", "**/*.mdc"}
	files, err := parser.DiscoverFiles(p.basePath, patterns)
	if err != nil {
		logging.Error(
			"failed to discover skill files",
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
		if parser.IsSkillEntrypointName(base) {
			continue
		}
		// Skip files inside skill directories
		if parser.IsInsideSkillDir(f, skillDirs) {
			logging.Debug(
				"skipping file inside skill directory",
				logging.Path(f),
			)
			continue
		}
		legacyFiles = append(legacyFiles, f)
	}

	logging.Debug(
		"discovered skill files",
		logging.Platform(string(p.Platform())),
		logging.Path(p.basePath),
		logging.Count(len(legacyFiles)),
	)

	// Parse each legacy skill file
	for _, filePath := range legacyFiles {
		skill, err := p.parseSkillFile(filePath)
		if err != nil {
			logging.Warn(
				"failed to parse skill file",
				logging.Platform(string(p.Platform())),
				logging.Path(filePath),
				logging.Err(err),
			)
			continue
		}
		// Skip if a SKILL.md with the same name was already parsed
		if seenNames[skill.Name] {
			logging.Debug(
				"skipping legacy skill, SKILL.md version takes precedence",
				logging.Skill(skill.Name),
				logging.Path(filePath),
			)
			continue
		}
		seenNames[skill.Name] = true
		allSkills = append(allSkills, skill)
	}

	// Finally, parse .cursor/commands/*.md files as prompt artifacts
	commandSkills, err := p.parseCommandFiles(seenNames)
	if err != nil {
		logging.Warn(
			"failed to parse command files",
			logging.Platform(string(p.Platform())),
			logging.Err(err),
		)
	}
	allSkills = append(allSkills, commandSkills...)

	logging.Debug(
		"completed parsing skills",
		logging.Platform(string(p.Platform())),
		logging.Count(len(allSkills)),
	)

	return allSkills, nil
}

// parseCommandFiles discovers and parses .cursor/commands/*.md files as SkillTypePrompt artifacts.
// The commands directory is inferred as a sibling of the skills basePath
// (e.g., ~/.cursor/commands when basePath is ~/.cursor/skills).
func (p *Parser) parseCommandFiles(seen map[string]bool) ([]model.Skill, error) {
	// Commands live alongside skills: ~/.cursor/commands or .cursor/commands
	commandsDir := filepath.Join(filepath.Dir(p.basePath), "commands")

	if _, err := os.Stat(commandsDir); os.IsNotExist(err) {
		logging.Debug(
			"commands directory not found",
			logging.Platform(string(p.Platform())),
			logging.Path(commandsDir),
		)
		return nil, nil
	}

	files, err := parser.DiscoverFiles(commandsDir, []string{"*.md", "**/*.md"})
	if err != nil {
		return nil, fmt.Errorf("failed to discover command files in %q: %w", commandsDir, err)
	}

	logging.Debug(
		"discovered command files",
		logging.Platform(string(p.Platform())),
		logging.Path(commandsDir),
		logging.Count(len(files)),
	)

	var results []model.Skill
	for _, filePath := range files {
		skill, err := p.parseCommandFile(filePath)
		if err != nil {
			logging.Warn(
				"failed to parse command file",
				logging.Platform(string(p.Platform())),
				logging.Path(filePath),
				logging.Err(err),
			)
			continue
		}
		if seen[skill.Name] {
			logging.Debug(
				"skipping command, higher-precedence artifact takes priority",
				logging.Skill(skill.Name),
				logging.Path(filePath),
			)
			continue
		}
		seen[skill.Name] = true
		results = append(results, skill)
	}

	return results, nil
}

// parseCommandFile parses a single .cursor/commands/*.md file into a SkillTypePrompt artifact.
// The slash trigger is derived from the filename stem (e.g., review.md → /review).
// All frontmatter fields are preserved in Metadata for mode-linked semantics.
func (p *Parser) parseCommandFile(filePath string) (model.Skill, error) {
	// #nosec G304 - filePath is validated through directory traversal from commandsDir
	content, err := os.ReadFile(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	result := parser.SplitFrontmatter(content)

	base := filepath.Base(filePath)
	stem := base[:len(base)-len(filepath.Ext(base))]

	skill := model.Skill{
		Platform: p.Platform(),
		Path:     filePath,
		Type:     model.SkillTypePrompt,
		Trigger:  "/" + stem,
		Metadata: make(map[string]string),
	}

	if result.HasFrontmatter {
		fm, err := parser.ParseYAMLFrontmatter(result.Frontmatter)
		if err != nil {
			return model.Skill{}, fmt.Errorf("failed to parse frontmatter in %q: %w", filePath, err)
		}

		if nameVal, ok := fm["name"]; ok {
			if nameStr, ok := nameVal.(string); ok && nameStr != "" {
				skill.Name = nameStr
			}
		}
		if descVal, ok := fm["description"]; ok {
			if descStr, ok := descVal.(string); ok {
				skill.Description = descStr
			}
		}

		// Preserve all frontmatter fields in metadata for mode-linked semantics
		for key, val := range fm {
			if key == "name" || key == "description" {
				continue
			}
			switch v := val.(type) {
			case string:
				skill.Metadata[key] = v
			case []any:
				parts := make([]string, 0, len(v))
				for _, item := range v {
					if s, ok := item.(string); ok {
						parts = append(parts, s)
					} else {
						parts = append(parts, fmt.Sprintf("%v", item))
					}
				}
				skill.Metadata[key] = fmt.Sprintf("%v", parts)
			default:
				skill.Metadata[key] = fmt.Sprintf("%v", val)
			}
		}
	}

	if skill.Name == "" {
		skill.Name = stem
	}

	if err := parser.ValidateSkillName(skill.Name); err != nil {
		return model.Skill{}, fmt.Errorf("invalid command name %q in %q: %w", skill.Name, filePath, err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	skill.Content = parser.NormalizeContent(result.Content)
	skill.ModifiedAt = fileInfo.ModTime()

	return skill, nil
}

// parseSkillFile parses a single Cursor skill file
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
	var name string
	metadata := make(map[string]string)

	if result.HasFrontmatter {
		fm, err := parser.ParseYAMLFrontmatter(result.Frontmatter)
		if err != nil {
			return model.Skill{}, fmt.Errorf("failed to parse frontmatter in %q: %w", filePath, err)
		}

		// Cursor skills typically don't have a name field in frontmatter
		// But we handle it if present
		if nameVal, ok := fm["name"]; ok {
			if nameStr, ok := nameVal.(string); ok {
				name = nameStr
			}
		}

		// Store all frontmatter fields in metadata
		// This includes Cursor-specific fields like globs and alwaysApply
		for key, val := range fm {
			if key != "name" {
				if strVal, ok := val.(string); ok {
					metadata[key] = strVal
				} else {
					// Handle arrays (like globs) by converting to string representation
					if sliceVal, ok := val.([]any); ok {
						strSlice := make([]string, 0, len(sliceVal))
						for _, item := range sliceVal {
							if itemStr, ok := item.(string); ok {
								strSlice = append(strSlice, itemStr)
							} else {
								strSlice = append(strSlice, fmt.Sprintf("%v", item))
							}
						}
						metadata[key] = fmt.Sprintf("%v", strSlice)
					} else {
						metadata[key] = fmt.Sprintf("%v", val)
					}
				}
			}
		}
	}

	// If no name in frontmatter, derive from filename (common for Cursor)
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
		Description: "", // Cursor doesn't typically use description
		Platform:    p.Platform(),
		Path:        filePath,
		Tools:       nil, // Cursor doesn't specify tools in frontmatter
		Metadata:    metadata,
		Content:     normalizedContent,
		ModifiedAt:  fileInfo.ModTime(),
	}

	return skill, nil
}

// Platform returns the platform identifier for Cursor
func (p *Parser) Platform() model.Platform {
	return model.Cursor
}

// DefaultPath returns the default path for Cursor skills
func (p *Parser) DefaultPath() string {
	return util.CursorSkillsPath()
}
