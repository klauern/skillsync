// Package cursor implements the Parser interface for Cursor skills/rules.
package cursor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/util"
)

// Parser implements the parser.Parser interface for Cursor skills
type Parser struct {
	basePath string
}

// New creates a new Cursor parser
// If basePath is empty, uses the default Cursor rules directory (~/.cursor/rules)
func New(basePath string) *Parser {
	if basePath == "" {
		basePath = util.CursorSkillsPath()
	}
	return &Parser{basePath: basePath}
}

// Parse parses Cursor skills from markdown files with YAML frontmatter
// Cursor uses .md and .mdc files with optional globs and alwaysApply fields
func (p *Parser) Parse() ([]model.Skill, error) {
	// Check if the base path exists
	if _, err := os.Stat(p.basePath); os.IsNotExist(err) {
		// Return empty slice if directory doesn't exist (not an error)
		return []model.Skill{}, nil
	}

	// Discover skill files - Cursor uses .md and .mdc files
	patterns := []string{"*.md", "*.mdc", "**/*.md", "**/*.mdc"}
	files, err := parser.DiscoverFiles(p.basePath, patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to discover skill files in %q: %w", p.basePath, err)
	}

	// Parse each skill file
	skills := make([]model.Skill, 0, len(files))
	for _, filePath := range files {
		skill, err := p.parseSkillFile(filePath)
		if err != nil {
			// Log the error but continue parsing other files
			// TODO: Consider adding structured logging here
			continue
		}
		skills = append(skills, skill)
	}

	return skills, nil
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
