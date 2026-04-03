// Package copilot implements discovery of GitHub Copilot repository and scoped instructions.
package copilot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
)

const (
	repositoryInstructionsPath = ".github/copilot-instructions.md"
	scopedInstructionsGlob     = ".github/instructions/*.instructions.md"
)

// Parser implements the parser.Parser interface for GitHub Copilot instructions.
type Parser struct {
	basePath string
}

// New creates a new Copilot parser.
// The base path is the repository root containing the .github directory.
func New(basePath string) *Parser {
	if basePath == "" {
		basePath = "."
	}
	return &Parser{basePath: basePath}
}

// Parse parses GitHub Copilot repository instructions and scoped instruction files.
func (p *Parser) Parse() ([]model.Skill, error) {
	if _, err := os.Stat(p.basePath); os.IsNotExist(err) {
		logging.Debug("repository root not found",
			logging.Platform(string(p.Platform())),
			logging.Path(p.basePath),
		)
		return []model.Skill{}, nil
	}

	var allSkills []model.Skill

	repoInstruction, err := p.parseRepositoryInstructions()
	if err != nil {
		return nil, err
	}
	if repoInstruction != nil {
		allSkills = append(allSkills, *repoInstruction)
	}

	scopedInstructions, err := p.parseScopedInstructions()
	if err != nil {
		return nil, err
	}
	allSkills = append(allSkills, scopedInstructions...)

	return allSkills, nil
}

func (p *Parser) parseRepositoryInstructions() (*model.Skill, error) {
	filePath := filepath.Join(p.basePath, repositoryInstructionsPath)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil
	}

	skill, err := p.parseInstructionFile(filePath, "GitHub Copilot repository instructions", "repository-instructions")
	if err != nil {
		return nil, err
	}

	return &skill, nil
}

func (p *Parser) parseScopedInstructions() ([]model.Skill, error) {
	files, err := parser.DiscoverFiles(p.basePath, []string{scopedInstructionsGlob})
	if err != nil {
		return nil, fmt.Errorf("failed to discover scoped instruction files: %w", err)
	}

	skills := make([]model.Skill, 0, len(files))
	for _, filePath := range files {
		skill, err := p.parseInstructionFile(filePath, "GitHub Copilot scoped instructions", "instruction")
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

func (p *Parser) parseInstructionFile(filePath, defaultDescription, instructionType string) (model.Skill, error) {
	// #nosec G304 - filePath comes from explicit repository-local discovery.
	content, err := os.ReadFile(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	result := parser.SplitFrontmatter(content)
	metadata := map[string]string{"type": instructionType}

	name := instructionName(filePath)
	description := defaultDescription

	if result.HasFrontmatter {
		fm, err := parser.ParseYAMLFrontmatter(result.Frontmatter)
		if err != nil {
			return model.Skill{}, fmt.Errorf("failed to parse frontmatter in %q: %w", filePath, err)
		}

		if parsedName := extractString(fm, "name"); parsedName != "" {
			name = parsedName
		}
		if parsedDescription := extractString(fm, "description"); parsedDescription != "" {
			description = parsedDescription
		}

		for key, val := range fm {
			if key == "name" || key == "description" {
				continue
			}
			metadata[key] = stringifyFrontmatterValue(val)
		}
	}

	if err := parser.ValidateSkillName(name); err != nil {
		return model.Skill{}, fmt.Errorf("invalid instruction name %q in %q: %w", name, filePath, err)
	}

	return model.Skill{
		Name:        name,
		Description: description,
		Platform:    p.Platform(),
		Path:        filePath,
		Metadata:    metadata,
		Content:     parser.NormalizeContent(result.Content),
		ModifiedAt:  fileInfo.ModTime(),
	}, nil
}

func instructionName(filePath string) string {
	base := filepath.Base(filePath)
	if strings.HasSuffix(base, ".instructions.md") {
		return strings.TrimSuffix(base, ".instructions.md")
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func extractString(fm map[string]any, key string) string {
	if val, ok := fm[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func stringifyFrontmatterValue(val any) string {
	switch typed := val.(type) {
	case string:
		return typed
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, stringifyFrontmatterValue(item))
		}
		return fmt.Sprintf("%v", items)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

// Platform returns the platform identifier for Copilot.
func (p *Parser) Platform() model.Platform {
	return model.Copilot
}

// DefaultPath returns the default base path for Copilot parsing.
func (p *Parser) DefaultPath() string {
	return "."
}
