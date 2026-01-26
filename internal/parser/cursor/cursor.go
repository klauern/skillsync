// Package cursor implements the Parser interface for Cursor skills/rules.
package cursor

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
			seenNames[skill.Name] = true
			allSkills = append(allSkills, skill)
		}
	}

	// Then, discover legacy skill files - Cursor uses .md and .mdc files
	patterns := []string{"*.md", "*.mdc", "**/*.md", "**/*.mdc"}
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

	logging.Debug("discovered skill files",
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
		seenNames[skill.Name] = true
		allSkills = append(allSkills, skill)
	}

	logging.Debug("completed parsing skills",
		logging.Platform(string(p.Platform())),
		logging.Count(len(allSkills)),
	)

	return allSkills, nil
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
