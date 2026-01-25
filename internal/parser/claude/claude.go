// Package claude implements the Parser interface for Claude Code skills.
package claude

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/util"
)

// Parser implements the parser.Parser interface for Claude Code skills
type Parser struct {
	basePath string
}

// New creates a new Claude Code parser
// If basePath is empty, uses the default Claude Code skills directory (~/.claude/skills)
func New(basePath string) *Parser {
	if basePath == "" {
		basePath = util.ClaudeCodeSkillsPath()
	}
	return &Parser{basePath: basePath}
}

// Parse parses Claude Code skills from markdown files with YAML frontmatter
func (p *Parser) Parse() ([]model.Skill, error) {
	// Check if the base path exists
	if _, err := os.Stat(p.basePath); os.IsNotExist(err) {
		// Return empty slice if directory doesn't exist (not an error)
		return []model.Skill{}, nil
	}

	// Discover skill files - Claude Code uses .md files
	patterns := []string{"*.md", "**/*.md"}
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
			if toolsSlice, ok := toolsVal.([]interface{}); ok {
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
