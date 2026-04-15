// Package copilot implements discovery of GitHub Copilot instructions, prompts, and agents.
package copilot

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/util"
)

const (
	repositoryInstructionsPath = "copilot-instructions.md"
	instructionsGlob           = "instructions/*.instructions.md"
	promptsGlob                = "prompts/*.prompt.md"
	agentsGlob                 = "agents/*.agent.md"
)

// Parser implements the parser.Parser interface for GitHub Copilot artifacts.
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

// Parse discovers instructions, prompt files, and agent files from the configured .github root.
func (p *Parser) Parse() ([]model.Skill, error) {
	root := p.githubRoot()
	if _, err := os.Stat(root); os.IsNotExist(err) {
		logging.Debug("copilot directory not found",
			logging.Platform(string(p.Platform())),
			logging.Path(root),
		)
		return []model.Skill{}, nil
	}

	var allSkills []model.Skill
	seenNames := make(map[string]bool)

	repoSkills, err := p.parseRepositoryInstructions(seenNames)
	if err != nil {
		logging.Warn("failed to parse Copilot repository instructions",
			logging.Platform(string(p.Platform())),
			logging.Path(root),
			logging.Err(err),
		)
	} else {
		allSkills = append(allSkills, repoSkills...)
	}

	instructionSkills, err := p.parseScopedInstructions(seenNames)
	if err != nil {
		logging.Warn("failed to parse Copilot scoped instructions",
			logging.Platform(string(p.Platform())),
			logging.Path(root),
			logging.Err(err),
		)
	} else {
		allSkills = append(allSkills, instructionSkills...)
	}

	promptSkills, err := p.parsePromptFiles(seenNames)
	if err != nil {
		logging.Warn("failed to parse Copilot prompt files",
			logging.Platform(string(p.Platform())),
			logging.Path(root),
			logging.Err(err),
		)
	} else {
		allSkills = append(allSkills, promptSkills...)
	}

	agentSkills, err := p.parseAgentFiles(seenNames)
	if err != nil {
		logging.Warn("failed to parse Copilot agent files",
			logging.Platform(string(p.Platform())),
			logging.Path(root),
			logging.Err(err),
		)
	} else {
		allSkills = append(allSkills, agentSkills...)
	}

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

func (p *Parser) githubRoot() string {
	if filepath.Base(p.basePath) == ".github" {
		return p.basePath
	}
	return filepath.Join(p.basePath, ".github")
}

func (p *Parser) parseRepositoryInstructions(seen map[string]bool) ([]model.Skill, error) {
	filePath := filepath.Join(p.githubRoot(), repositoryInstructionsPath)
	skill, err := p.parseInstructionFile(filePath, "GitHub Copilot repository instructions", "repository-instructions")
	if err != nil || skill == nil {
		return nil, err
	}
	if seen[skill.Name] {
		return []model.Skill{}, nil
	}
	seen[skill.Name] = true
	return []model.Skill{*skill}, nil
}

func (p *Parser) parseScopedInstructions(seen map[string]bool) ([]model.Skill, error) {
	files, err := parser.DiscoverFiles(p.githubRoot(), []string{instructionsGlob})
	if err != nil {
		return nil, fmt.Errorf("failed to discover Copilot scoped instruction files in %q: %w", p.githubRoot(), err)
	}
	if len(files) > 1 {
		sort.Strings(files)
	}

	var results []model.Skill
	for _, filePath := range files {
		skill, err := p.parseInstructionFile(filePath, "GitHub Copilot scoped instructions", "instruction")
		if err != nil {
			logging.Warn("failed to parse Copilot scoped instruction file",
				logging.Path(filePath),
				logging.Err(err),
			)
			continue
		}
		if skill == nil || seen[skill.Name] {
			continue
		}
		seen[skill.Name] = true
		results = append(results, *skill)
	}

	return results, nil
}

func (p *Parser) parsePromptFiles(seen map[string]bool) ([]model.Skill, error) {
	files, err := parser.DiscoverFiles(p.githubRoot(), []string{promptsGlob})
	if err != nil {
		return nil, fmt.Errorf("failed to discover Copilot prompt files in %q: %w", p.githubRoot(), err)
	}
	if len(files) > 1 {
		sort.Strings(files)
	}

	var results []model.Skill
	for _, filePath := range files {
		skill, err := p.parseArtifactFile(filePath, model.SkillTypePrompt, true)
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
	files, err := parser.DiscoverFiles(p.githubRoot(), []string{agentsGlob})
	if err != nil {
		return nil, fmt.Errorf("failed to discover Copilot agent files in %q: %w", p.githubRoot(), err)
	}
	if len(files) > 1 {
		sort.Strings(files)
	}

	var results []model.Skill
	for _, filePath := range files {
		skill, err := p.parseArtifactFile(filePath, model.SkillTypeSkill, false)
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

func (p *Parser) parseInstructionFile(filePath, defaultDescription, instructionType string) (*model.Skill, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil
	}

	// #nosec G304 - filePath is validated through explicit repository-local discovery.
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	result := parser.SplitFrontmatter(content)
	metadata := map[string]string{"type": instructionType}
	name := instructionName(filePath)
	description := defaultDescription

	if result.HasFrontmatter {
		fm, err := parser.ParseYAMLFrontmatter(result.Frontmatter)
		if err != nil {
			return nil, fmt.Errorf("failed to parse frontmatter in %q: %w", filePath, err)
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
		return nil, fmt.Errorf("invalid instruction name %q in %q: %w", name, filePath, err)
	}

	return &model.Skill{
		Name:        name,
		Description: description,
		Platform:    p.Platform(),
		Path:        filePath,
		Metadata:    metadata,
		Content:     parser.NormalizeContent(result.Content),
		ModifiedAt:  fileInfo.ModTime(),
		Type:        model.SkillTypeSkill,
	}, nil
}

func (p *Parser) parseArtifactFile(filePath string, defaultType model.SkillType, isPrompt bool) (model.Skill, error) {
	// #nosec G304 - filePath is validated through directory traversal from githubRoot.
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
			skill.Metadata[key] = stringifyFrontmatterValue(val)
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

func instructionName(filePath string) string {
	base := filepath.Base(filePath)
	if strings.HasSuffix(base, ".instructions.md") {
		return strings.TrimSuffix(base, ".instructions.md")
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func artifactStem(filePath string) string {
	base := filepath.Base(filePath)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	stem = strings.TrimSuffix(stem, ".prompt")
	stem = strings.TrimSuffix(stem, ".agent")
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
