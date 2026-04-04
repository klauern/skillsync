// Package gemini implements parser slices for Gemini CLI artifacts.
package gemini

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/util"
)

// Parser implements parser.Parser for Gemini command and agent artifacts.
type Parser struct {
	basePath string
}

// New creates a Gemini parser rooted at a Gemini config directory.
func New(basePath string) *Parser {
	if basePath == "" {
		basePath = filepath.Join(util.HomeDir(), ".gemini")
	}
	return &Parser{basePath: basePath}
}

// Parse parses Gemini command and agent artifacts from the configured root.
func (p *Parser) Parse() ([]model.Skill, error) {
	if _, err := os.Stat(p.basePath); os.IsNotExist(err) {
		return []model.Skill{}, nil
	}

	var all []model.Skill
	seenNames := make(map[string]bool)

	commands, err := p.parseCommands(seenNames)
	if err != nil {
		return nil, err
	}
	all = append(all, commands...)

	agents, err := p.parseAgents(seenNames)
	if err != nil {
		return nil, err
	}
	all = append(all, agents...)

	return all, nil
}

func (p *Parser) parseCommands(seenNames map[string]bool) ([]model.Skill, error) {
	root := filepath.Join(p.basePath, "commands")
	patterns := []string{"*.toml", "**/*.toml"}
	files, err := parser.DiscoverFiles(root, patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to discover Gemini command files in %q: %w", root, err)
	}
	sort.Strings(files)

	results := make([]model.Skill, 0, len(files))
	for _, filePath := range files {
		skill, err := p.parseCommandFile(filePath)
		if err != nil {
			return nil, err
		}
		if seenNames[skill.Name] {
			continue
		}
		seenNames[skill.Name] = true
		results = append(results, skill)
	}
	return results, nil
}

func (p *Parser) parseCommandFile(filePath string) (model.Skill, error) {
	// #nosec G304 - filePath is discovered under the configured Gemini root.
	content, err := os.ReadFile(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to read Gemini command file %q: %w", filePath, err)
	}

	var raw map[string]any
	if _, err := toml.Decode(string(content), &raw); err != nil {
		return model.Skill{}, fmt.Errorf("failed to parse Gemini command TOML %q: %w", filePath, err)
	}

	prompt := extractString(raw, "prompt")
	if prompt == "" {
		return model.Skill{}, fmt.Errorf("Gemini command %q missing required prompt", filePath)
	}

	name, trigger, err := commandIdentity(filePath, filepath.Join(p.basePath, "commands"))
	if err != nil {
		return model.Skill{}, err
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to stat Gemini command file %q: %w", filePath, err)
	}

	metadata := collectMetadata(raw, "prompt", "description")
	metadata["source_format"] = "toml"
	if syntax := detectArgumentSyntax(prompt); syntax != "" {
		metadata["argument_syntax"] = syntax
	}

	description := extractString(raw, "description")
	if description == "" {
		description = "Gemini custom command"
	}

	return model.Skill{
		Name:        name,
		Description: description,
		Platform:    p.Platform(),
		Path:        filePath,
		Metadata:    metadata,
		Content:     prompt,
		ModifiedAt:  fileInfo.ModTime(),
		Type:        model.SkillTypePrompt,
		Trigger:     trigger,
	}, nil
}

func (p *Parser) parseAgents(seenNames map[string]bool) ([]model.Skill, error) {
	root := filepath.Join(p.basePath, "agents")
	patterns := []string{"*.md", "**/*.md"}
	files, err := parser.DiscoverFiles(root, patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to discover Gemini agent files in %q: %w", root, err)
	}
	sort.Strings(files)

	results := make([]model.Skill, 0, len(files))
	for _, filePath := range files {
		skill, err := p.parseAgentFile(filePath)
		if err != nil {
			return nil, err
		}
		if seenNames[skill.Name] {
			continue
		}
		seenNames[skill.Name] = true
		results = append(results, skill)
	}
	return results, nil
}

func (p *Parser) parseAgentFile(filePath string) (model.Skill, error) {
	// #nosec G304 - filePath is discovered under the configured Gemini root.
	content, err := os.ReadFile(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to read Gemini agent file %q: %w", filePath, err)
	}

	result := parser.SplitFrontmatter(content)
	if !result.HasFrontmatter {
		return model.Skill{}, fmt.Errorf("Gemini agent %q missing required frontmatter", filePath)
	}

	fm, err := parser.ParseYAMLFrontmatter(result.Frontmatter)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to parse Gemini agent frontmatter in %q: %w", filePath, err)
	}

	name := extractString(fm, "name")
	description := extractString(fm, "description")
	if name == "" {
		return model.Skill{}, fmt.Errorf("Gemini agent %q missing required name", filePath)
	}
	if description == "" {
		return model.Skill{}, fmt.Errorf("Gemini agent %q missing required description", filePath)
	}
	if err := parser.ValidateSkillName(name); err != nil {
		return model.Skill{}, fmt.Errorf("invalid Gemini agent name %q in %q: %w", name, filePath, err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to stat Gemini agent file %q: %w", filePath, err)
	}

	metadata := collectMetadata(fm, "name", "description")
	metadata["type"] = "agents"

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

func commandIdentity(filePath, commandsRoot string) (string, string, error) {
	relPath, err := filepath.Rel(commandsRoot, filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to derive Gemini command identity for %q: %w", filePath, err)
	}

	relNoExt := strings.TrimSuffix(relPath, filepath.Ext(relPath))
	parts := strings.Split(filepath.ToSlash(relNoExt), "/")
	if len(parts) == 0 {
		return "", "", fmt.Errorf("failed to derive Gemini command identity for %q", filePath)
	}

	name := strings.Join(parts, "-")
	if err := parser.ValidateSkillName(name); err != nil {
		return "", "", fmt.Errorf("invalid Gemini command name %q in %q: %w", name, filePath, err)
	}

	trigger := "/" + parts[0]
	if len(parts) > 1 {
		trigger += ":" + strings.Join(parts[1:], ":")
	}

	return name, trigger, nil
}

func detectArgumentSyntax(prompt string) string {
	var syntax []string
	if strings.Contains(prompt, "{{args}}") {
		syntax = append(syntax, "{{args}}")
	}
	if strings.Contains(prompt, "!{") {
		syntax = append(syntax, "!{shell}")
	}
	if strings.Contains(prompt, "@{") {
		syntax = append(syntax, "@{path}")
	}
	return strings.Join(syntax, ",")
}

func collectMetadata(raw map[string]any, exclude ...string) map[string]string {
	skip := make(map[string]bool, len(exclude))
	for _, key := range exclude {
		skip[key] = true
	}

	metadata := make(map[string]string)
	keys := make([]string, 0, len(raw))
	for key := range raw {
		if skip[key] {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		val := raw[key]
		if strVal, ok := val.(string); ok {
			metadata[key] = strVal
			continue
		}
		metadata[key] = fmt.Sprintf("%v", val)
	}

	return metadata
}

func extractString(raw map[string]any, key string) string {
	val, ok := raw[key]
	if !ok {
		return ""
	}
	strVal, ok := val.(string)
	if !ok {
		return ""
	}
	return strVal
}

// Platform returns the Gemini platform identifier for this parser slice.
func (p *Parser) Platform() model.Platform {
	return model.Platform("gemini")
}

// DefaultPath returns the default Gemini config root.
func (p *Parser) DefaultPath() string {
	return filepath.Join(util.HomeDir(), ".gemini")
}
