// Package pidev implements the Parser interface for Pi.dev skills, prompt templates, and instructions.
//
// First-pass scope:
//   - SKILL.md files under the selected skills root
//   - markdown prompt templates under the matching prompts root
//   - AGENTS.md instruction files (global + hierarchical repo chain)
//   - SYSTEM.md / APPEND_SYSTEM.md system-prompt customization files
package pidev

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	coreparser "github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/parser/skills"
	"github.com/klauern/skillsync/internal/util"
)

// Parser implements the parser.Parser interface for Pi.dev artifacts.
type Parser struct {
	basePath string
}

// New creates a new Pi.dev parser.
// If basePath is empty, uses the preferred Pi.dev skills directory.
func New(basePath string) *Parser {
	if basePath == "" {
		basePath = util.PiDevSkillsPath()
	}
	return &Parser{basePath: basePath}
}

// Parse discovers skills, prompt templates, and AGENTS.md instructions.
func (p *Parser) Parse() ([]model.Skill, error) {
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

	// Parse SKILL.md files first so they take precedence over prompt/instruction
	// artifacts with the same discovery name.
	skillsParser := skills.New(p.basePath, p.Platform())
	agentSkills, err := skillsParser.Parse()
	if err != nil {
		logging.Warn(
			"failed to parse SKILL.md files",
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

	promptSkills, err := p.parsePrompts(seenNames)
	if err != nil {
		logging.Warn(
			"failed to parse prompt templates",
			logging.Platform(string(p.Platform())),
			logging.Path(p.promptsRoot()),
			logging.Err(err),
		)
	} else {
		allSkills = append(allSkills, promptSkills...)
	}

	instructionSkills, err := p.parseInstructions(seenNames)
	if err != nil {
		logging.Warn(
			"failed to parse AGENTS.md instructions",
			logging.Platform(string(p.Platform())),
			logging.Path(p.basePath),
			logging.Err(err),
		)
	} else {
		allSkills = append(allSkills, instructionSkills...)
	}

	systemSkills, err := p.parseSystemPrompts(seenNames)
	if err != nil {
		logging.Warn(
			"failed to parse SYSTEM.md files",
			logging.Platform(string(p.Platform())),
			logging.Path(p.configRoot()),
			logging.Err(err),
		)
	} else {
		allSkills = append(allSkills, systemSkills...)
	}

	return allSkills, nil
}

// Platform returns the platform identifier for Pi.dev.
func (p *Parser) Platform() model.Platform {
	return model.PiDev
}

// DefaultPath returns the preferred Pi.dev user-level skills path.
func (p *Parser) DefaultPath() string {
	return util.PiDevSkillsPath()
}

func (p *Parser) promptsRoot() string {
	return filepath.Join(filepath.Dir(p.basePath), "prompts")
}

func (p *Parser) configRoot() string {
	return filepath.Dir(p.basePath)
}

func (p *Parser) parsePrompts(seenNames map[string]bool) ([]model.Skill, error) {
	promptsRoot := p.promptsRoot()
	if _, err := os.Stat(promptsRoot); os.IsNotExist(err) {
		return []model.Skill{}, nil
	}

	patterns := []string{"*.md", "**/*.md"}
	files, err := coreparser.DiscoverFiles(promptsRoot, patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to discover prompt templates in %q: %w", promptsRoot, err)
	}
	if len(files) > 1 {
		sort.Strings(files)
	}

	var results []model.Skill
	for _, filePath := range files {
		skill, err := p.parsePromptFile(filePath)
		if err != nil {
			logging.Warn(
				"failed to parse prompt template",
				logging.Platform(string(p.Platform())),
				logging.Path(filePath),
				logging.Err(err),
			)
			continue
		}
		if seenNames[skill.Name] {
			continue
		}
		seenNames[skill.Name] = true
		results = append(results, skill)
	}

	return results, nil
}

func (p *Parser) parsePromptFile(filePath string) (model.Skill, error) {
	// #nosec G304 - filePath is validated through directory traversal from promptsRoot
	content, err := os.ReadFile(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	result := coreparser.SplitFrontmatter(content)
	metadata := make(map[string]string)
	name := ""
	description := ""
	trigger := ""
	skillType := model.SkillTypePrompt

	if result.HasFrontmatter {
		fm, err := coreparser.ParseYAMLFrontmatter(result.Frontmatter)
		if err != nil {
			return model.Skill{}, fmt.Errorf("failed to parse frontmatter in %q: %w", filePath, err)
		}

		name = extractString(fm, "name")
		description = extractString(fm, "description")
		trigger = extractString(fm, "trigger")
		if typeStr := extractString(fm, "type"); typeStr != "" {
			parsedType, err := model.ParseSkillType(typeStr)
			if err == nil {
				skillType = parsedType
			}
		}

		for key, val := range fm {
			if key == "name" || key == "description" || key == "type" || key == "trigger" {
				continue
			}
			if strVal, ok := val.(string); ok {
				metadata[key] = strVal
			} else {
				metadata[key] = fmt.Sprintf("%v", val)
			}
		}
	}

	if name == "" {
		base := filepath.Base(filePath)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}
	if description == "" {
		description = "Pi.dev prompt template"
	}
	if trigger == "" {
		trigger = "/" + strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	} else if !strings.HasPrefix(trigger, "/") {
		trigger = "/" + trigger
	}

	if err := coreparser.ValidateSkillName(name); err != nil {
		return model.Skill{}, fmt.Errorf("invalid prompt name %q in %q: %w", name, filePath, err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	return model.Skill{
		Name:        name,
		Description: description,
		Platform:    p.Platform(),
		Path:        filePath,
		Metadata:    metadata,
		Content:     coreparser.NormalizeContent(result.Content),
		ModifiedAt:  fileInfo.ModTime(),
		Type:        skillType,
		Trigger:     trigger,
	}, nil
}

func (p *Parser) parseInstructions(seenNames map[string]bool) ([]model.Skill, error) {
	var results []model.Skill

	// Global AGENTS.md can live at either the chosen Pi.dev root (e.g. ~/.pi/agent)
	// or at the project root (e.g. repo/AGENTS.md).
	instructionRoots := []string{
		p.configRoot(),
		filepath.Dir(p.configRoot()),
	}
	for _, root := range instructionRoots {
		if root == "" {
			continue
		}
		if global, err := p.parseAgentsFile(filepath.Join(root, "AGENTS.md"), root); err == nil && global != nil {
			if !seenNames[global.Name] {
				seenNames[global.Name] = true
				results = append(results, *global)
			}
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return results, nil
	}

	repoRoot := util.GetRepoRoot(cwd)
	if repoRoot == "" {
		repoRoot = cwd
	}

	rel, err := filepath.Rel(repoRoot, cwd)
	if err != nil || strings.HasPrefix(rel, "..") {
		return results, nil
	}

	dirs := []string{repoRoot}
	if rel != "." {
		parts := strings.Split(rel, string(os.PathSeparator))
		current := repoRoot
		for _, part := range parts {
			current = filepath.Join(current, part)
			dirs = append(dirs, current)
		}
	}

	for _, dir := range dirs {
		filePath := filepath.Join(dir, "AGENTS.md")
		skill, err := p.parseAgentsFile(filePath, repoRoot)
		if err != nil || skill == nil {
			continue
		}
		if seenNames[skill.Name] {
			continue
		}
		seenNames[skill.Name] = true
		results = append(results, *skill)
	}

	return results, nil
}

func (p *Parser) parseSystemPrompts(seenNames map[string]bool) ([]model.Skill, error) {
	var results []model.Skill

	for _, fileName := range []struct {
		name        string
		description string
		mode        string
	}{
		{name: "SYSTEM.md", description: "Pi.dev SYSTEM.md replacement prompt", mode: "replace"},
		{name: "APPEND_SYSTEM.md", description: "Pi.dev APPEND_SYSTEM.md append prompt", mode: "append"},
	} {
		filePath := filepath.Join(p.configRoot(), fileName.name)
		skill, err := p.parseSystemPromptFile(filePath, fileName.description, fileName.mode)
		if err != nil || skill == nil {
			continue
		}
		if seenNames[skill.Name] {
			continue
		}
		seenNames[skill.Name] = true
		results = append(results, *skill)
	}

	return results, nil
}

func (p *Parser) parseSystemPromptFile(filePath, description, mode string) (*model.Skill, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil
	}

	// #nosec G304 - filePath is validated via explicit system prompt discovery.
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	name := strings.ToLower(strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)))
	if name == "append_system" {
		name = "append-system"
	}
	if err := coreparser.ValidateSkillName(name); err != nil {
		return nil, fmt.Errorf("invalid system prompt name %q in %q: %w", name, filePath, err)
	}

	return &model.Skill{
		Name:        name,
		Description: description,
		Platform:    p.Platform(),
		Path:        filePath,
		Metadata:    map[string]string{"type": "system-prompt", "mode": mode},
		Content:     coreparser.NormalizeContent(string(content)),
		ModifiedAt:  fileInfo.ModTime(),
	}, nil
}

func (p *Parser) parseAgentsFile(filePath, nameRoot string) (*model.Skill, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil
	}

	// #nosec G304 - filePath is validated via explicit AGENTS.md discovery.
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	name := instructionName(filePath, nameRoot)
	if err := coreparser.ValidateSkillName(name); err != nil {
		return nil, fmt.Errorf("invalid instruction name %q in %q: %w", name, filePath, err)
	}

	return &model.Skill{
		Name:        name,
		Description: "Pi.dev AGENTS.md instructions",
		Platform:    p.Platform(),
		Path:        filePath,
		Metadata:    map[string]string{"type": "agents"},
		Content:     coreparser.NormalizeContent(string(content)),
		ModifiedAt:  fileInfo.ModTime(),
	}, nil
}

func instructionName(filePath, root string) string {
	rel, err := filepath.Rel(root, filePath)
	if err != nil {
		return "agents"
	}
	dir := filepath.Dir(rel)
	if dir == "." || dir == "" {
		return "agents"
	}
	return filepath.Base(dir) + "-agents"
}

func extractString(fm map[string]any, key string) string {
	if val, ok := fm[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}
