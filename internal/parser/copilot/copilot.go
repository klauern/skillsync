// Package copilot implements the Parser interface for GitHub Copilot artifacts.
package copilot

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

// Parser implements the parser.Parser interface for GitHub Copilot prompt and agent artifacts.
type Parser struct {
	basePath string
}

// New creates a new Copilot parser.
// If basePath is empty, uses the preferred GitHub Copilot workspace root.
func New(basePath string) *Parser {
	if basePath == "" {
		basePath = util.CopilotSkillsPath()
	}
	return &Parser{basePath: basePath}
}

// Parse discovers supported Copilot artifacts from the configured .github root.
func (p *Parser) Parse() ([]model.Skill, error) {
	if _, err := os.Stat(p.basePath); os.IsNotExist(err) {
		logging.Debug("copilot directory not found",
			logging.Platform(string(p.Platform())),
			logging.Path(p.basePath),
		)
		return []model.Skill{}, nil
	}

	var allSkills []model.Skill
	seenNames := make(map[string]bool)

	repositoryInstructions, err := p.parseRepositoryInstructions(seenNames)
	if err != nil {
		return nil, err
	}
	allSkills = append(allSkills, repositoryInstructions...)

	instructionSkills, err := p.parseInstructionFiles(seenNames)
	if err != nil {
		return nil, err
	}
	allSkills = append(allSkills, instructionSkills...)

	promptSkills, err := p.parsePromptFiles(seenNames)
	if err != nil {
		return nil, err
	}
	allSkills = append(allSkills, promptSkills...)

	agentSkills, err := p.parseAgentFiles(seenNames)
	if err != nil {
		return nil, err
	}
	allSkills = append(allSkills, agentSkills...)

	return allSkills, nil
}

// Platform returns the platform identifier for GitHub Copilot.
func (p *Parser) Platform() model.Platform {
	return model.Copilot
}

// DefaultPath returns the default Copilot workspace root.
func (p *Parser) DefaultPath() string {
	return util.CopilotSkillsPath()
}

func (p *Parser) parseRepositoryInstructions(seen map[string]bool) ([]model.Skill, error) {
	filePath := filepath.Join(p.basePath, "copilot-instructions.md")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to stat Copilot repository instructions %q: %w", filePath, err)
	}

	skill, err := p.parseArtifactFile(filePath, model.SkillTypeSkill, false, model.CopilotArtifactRepositoryInstructions)
	if err != nil {
		logging.Warn("failed to parse Copilot repository instructions",
			logging.Path(filePath),
			logging.Err(err),
		)
		return nil, nil
	}
	if seen[skill.Name] {
		return nil, nil
	}
	seen[skill.Name] = true
	return []model.Skill{skill}, nil
}

func (p *Parser) parseInstructionFiles(seen map[string]bool) ([]model.Skill, error) {
	files, err := parser.DiscoverFiles(p.basePath, []string{"instructions/*.instructions.md"})
	if err != nil {
		return nil, fmt.Errorf("failed to discover Copilot instruction files in %q: %w", p.basePath, err)
	}

	var results []model.Skill
	for _, filePath := range files {
		skill, err := p.parseArtifactFile(filePath, model.SkillTypeSkill, false, model.CopilotArtifactInstructions)
		if err != nil {
			logging.Warn("failed to parse Copilot instruction file",
				logging.Path(filePath),
				logging.Err(err),
			)
			continue
		}
		if seen[skill.Name] {
			continue
		}
		seen[skill.Name] = true
		results = append(results, skill)
	}

	return results, nil
}

func (p *Parser) parsePromptFiles(seen map[string]bool) ([]model.Skill, error) {
	files, err := parser.DiscoverFiles(p.basePath, []string{"prompts/*.prompt.md"})
	if err != nil {
		return nil, fmt.Errorf("failed to discover Copilot prompt files in %q: %w", p.basePath, err)
	}

	var results []model.Skill
	for _, filePath := range files {
		skill, err := p.parseArtifactFile(filePath, model.SkillTypePrompt, true, model.CopilotArtifactPrompt)
		if err != nil {
			logging.Warn("failed to parse Copilot prompt file",
				logging.Path(filePath),
				logging.Err(err),
			)
			continue
		}
		if seen[skill.Name] {
			continue
		}
		seen[skill.Name] = true
		results = append(results, skill)
	}

	return results, nil
}

func (p *Parser) parseAgentFiles(seen map[string]bool) ([]model.Skill, error) {
	files, err := parser.DiscoverFiles(p.basePath, []string{"agents/*.agent.md"})
	if err != nil {
		return nil, fmt.Errorf("failed to discover Copilot agent files in %q: %w", p.basePath, err)
	}

	var results []model.Skill
	for _, filePath := range files {
		skill, err := p.parseArtifactFile(filePath, model.SkillTypeSkill, false, model.CopilotArtifactAgent)
		if err != nil {
			logging.Warn("failed to parse Copilot agent file",
				logging.Path(filePath),
				logging.Err(err),
			)
			continue
		}
		if seen[skill.Name] {
			continue
		}
		seen[skill.Name] = true
		results = append(results, skill)
	}

	return results, nil
}

func (p *Parser) parseArtifactFile(
	filePath string,
	defaultType model.SkillType,
	isPrompt bool,
	artifactType string,
) (model.Skill, error) {
	// #nosec G304 - filePath is validated through directory traversal from basePath
	content, err := os.ReadFile(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	result := parser.SplitFrontmatter(content)
	skill := model.Skill{
		Platform: p.Platform(),
		Path:     filePath,
		Metadata: make(map[string]string),
		Type:     defaultType,
	}
	if artifactType != "" {
		skill.Metadata[model.MetadataKeyCopilotArtifact] = artifactType
	}

	if result.HasFrontmatter {
		fm, err := parser.ParseYAMLFrontmatter(result.Frontmatter)
		if err != nil {
			return model.Skill{}, fmt.Errorf("failed to parse frontmatter in %q: %w", filePath, err)
		}

		if name := extractString(fm, "name"); name != "" {
			skill.Name = name
		}
		skill.Description = extractString(fm, "description")
		skill.Tools = extractTools(fm, "tools")

		if typeStr := extractString(fm, "type"); typeStr != "" {
			if parsed, err := model.ParseSkillType(typeStr); err == nil {
				skill.Type = parsed
			}
		}

		for key, val := range fm {
			if key == "name" || key == "description" || key == "tools" || key == "type" {
				continue
			}
			if strVal, ok := val.(string); ok {
				skill.Metadata[key] = strVal
			} else {
				skill.Metadata[key] = fmt.Sprintf("%v", val)
			}
		}
	}

	if skill.Name == "" {
		skill.Name = artifactStem(filePath)
	}

	if isPrompt && skill.Trigger == "" {
		skill.Trigger = "/" + artifactStem(filePath)
	}

	if err := parser.ValidateSkillName(skill.Name); err != nil {
		return model.Skill{}, fmt.Errorf("invalid Copilot artifact name %q in %q: %w", skill.Name, filePath, err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	skill.Content = parser.NormalizeContent(result.Content)
	skill.ModifiedAt = fileInfo.ModTime()

	return skill, nil
}

func artifactStem(filePath string) string {
	base := filepath.Base(filePath)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	stem = strings.TrimSuffix(stem, ".prompt")
	stem = strings.TrimSuffix(stem, ".agent")
	stem = strings.TrimSuffix(stem, ".instructions")
	return stem
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
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
		return result
	default:
		return nil
	}
}
