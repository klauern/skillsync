package template

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/skills"
	"github.com/klauern/skillsync/internal/util"
)

// TemplateType represents the type of skill template
type TemplateType string

const (
	CommandWrapper TemplateType = "command-wrapper"
	Workflow       TemplateType = "workflow"
	Utility        TemplateType = "utility"
)

// TemplateData holds the data passed to templates
type TemplateData struct {
	Name        string
	Description string
	Platform    string
	Scope       string
	Author      string
	Year        int
	Tools       []string
	Scripts     []string
	References  []string
}

// Generator handles skill template generation
type Generator struct {
	templates map[TemplateType]*template.Template
}

// New creates a new template generator with built-in templates
func New() (*Generator, error) {
	g := &Generator{
		templates: make(map[TemplateType]*template.Template),
	}

	// Load built-in templates
	if err := g.loadBuiltinTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load built-in templates: %w", err)
	}

	return g, nil
}

// loadBuiltinTemplates loads all built-in templates
func (g *Generator) loadBuiltinTemplates() error {
	templates := map[TemplateType]string{
		CommandWrapper: commandWrapperTemplate,
		Workflow:       workflowTemplate,
		Utility:        utilityTemplate,
	}

	for typ, content := range templates {
		tmpl, err := template.New(string(typ)).Parse(content)
		if err != nil {
			return fmt.Errorf("failed to parse %s template: %w", typ, err)
		}
		g.templates[typ] = tmpl
	}

	return nil
}

// LoadCustomTemplate loads a custom template from a file
func (g *Generator) LoadCustomTemplate(name string, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	g.templates[TemplateType(name)] = tmpl
	return nil
}

// Generate generates a skill from a template
func (g *Generator) Generate(typ TemplateType, data TemplateData) (string, error) {
	tmpl, exists := g.templates[typ]
	if !exists {
		return "", fmt.Errorf("template %s not found", typ)
	}

	// Set defaults
	if data.Year == 0 {
		data.Year = time.Now().Year()
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// ValidateGenerated validates that generated content is a valid skill
func (g *Generator) ValidateGenerated(content string) error {
	// Use the IsAgentSkillsFormat validator
	if !skills.IsAgentSkillsFormat([]byte(content)) {
		return errors.New("generated content is not a valid Agent Skills Standard format")
	}
	return nil
}

// CreateSkillFile creates a skill file at the specified location
func (g *Generator) CreateSkillFile(typ TemplateType, data TemplateData, platform model.Platform, scope model.SkillScope) (string, error) {
	// Generate content
	content, err := g.Generate(typ, data)
	if err != nil {
		return "", err
	}

	// Validate content
	if err := g.ValidateGenerated(content); err != nil {
		return "", err
	}

	// Determine base path
	basePath, err := getBasePath(platform, scope)
	if err != nil {
		return "", err
	}

	// Create skill directory
	skillDir := filepath.Join(basePath, data.Name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Write SKILL.md
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write skill file: %w", err)
	}

	// Create subdirectories if needed
	if len(data.Scripts) > 0 {
		scriptsDir := filepath.Join(skillDir, "scripts")
		if err := os.MkdirAll(scriptsDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create scripts directory: %w", err)
		}
	}

	if len(data.References) > 0 {
		refsDir := filepath.Join(skillDir, "references")
		if err := os.MkdirAll(refsDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create references directory: %w", err)
		}
	}

	return skillPath, nil
}

// getBasePath returns the base path for a given platform and scope
func getBasePath(platform model.Platform, scope model.SkillScope) (string, error) {
	var basePath string

	switch platform {
	case model.ClaudeCode:
		if scope == model.ScopeRepo {
			basePath = ".claude/skills"
		} else {
			basePath = util.ClaudeCodeSkillsPath()
		}
	case model.Cursor:
		if scope == model.ScopeRepo {
			basePath = ".cursor/skills"
		} else {
			basePath = util.CursorSkillsPath()
		}
	case model.Codex:
		if scope == model.ScopeRepo {
			basePath = ".codex/skills"
		} else {
			basePath = util.CodexSkillsPath()
		}
	default:
		return "", fmt.Errorf("unsupported platform: %s", platform)
	}

	return basePath, nil
}

// ListTemplates returns a list of available template types
func (g *Generator) ListTemplates() []string {
	templates := make([]string, 0, len(g.templates))
	for typ := range g.templates {
		templates = append(templates, string(typ))
	}
	return templates
}

// ParseTemplateType parses a template type string
func ParseTemplateType(s string) (TemplateType, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "command-wrapper", "command", "cmd":
		return CommandWrapper, nil
	case "workflow":
		return Workflow, nil
	case "utility", "util":
		return Utility, nil
	default:
		return "", errors.New("unknown template type")
	}
}
