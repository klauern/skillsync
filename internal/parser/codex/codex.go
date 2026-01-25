// Package codex implements the Parser interface for OpenAI Codex CLI skills.
// Codex uses TOML configuration (config.toml) and AGENTS.md files for instructions.
package codex

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/util"
)

// Parser implements the parser.Parser interface for Codex skills
type Parser struct {
	basePath string
}

// Config represents the Codex config.toml structure
type Config struct {
	Model                 string               `toml:"model"`
	ApprovalPolicy        string               `toml:"approval_policy"`
	SandboxMode           string               `toml:"sandbox_mode"`
	Instructions          string               `toml:"instructions"`
	DeveloperInstructions string               `toml:"developer_instructions"`
	Profile               string               `toml:"profile"`
	Profiles              map[string]Profile   `toml:"profiles"`
	TUI                   TUIConfig            `toml:"tui"`
	History               HistoryConfig        `toml:"history"`
	Features              FeaturesConfig       `toml:"features"`
	MCPServers            map[string]MCPServer `toml:"mcp_servers"`
}

// Profile represents a named configuration profile
type Profile struct {
	Model                string `toml:"model"`
	ApprovalPolicy       string `toml:"approval_policy"`
	SandboxMode          string `toml:"sandbox_mode"`
	ModelReasoningEffort string `toml:"model_reasoning_effort"`
}

// TUIConfig represents TUI settings
type TUIConfig struct {
	Notifications bool `toml:"notifications"`
	Animations    bool `toml:"animations"`
}

// HistoryConfig represents history settings
type HistoryConfig struct {
	Persistence string `toml:"persistence"`
	MaxBytes    int64  `toml:"max_bytes"`
}

// FeaturesConfig represents feature toggles
type FeaturesConfig struct {
	ShellTool bool `toml:"shell_tool"`
}

// MCPServer represents an MCP server configuration
type MCPServer struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
}

// New creates a new Codex parser
// If basePath is empty, uses the default Codex config directory (~/.codex)
func New(basePath string) *Parser {
	if basePath == "" {
		basePath = filepath.Join(util.HomeDir(), ".codex")
	}
	return &Parser{basePath: basePath}
}

// Parse parses Codex skills from config.toml and AGENTS.md files
func (p *Parser) Parse() ([]model.Skill, error) {
	// Check if the base path exists
	if _, err := os.Stat(p.basePath); os.IsNotExist(err) {
		// Return empty slice if directory doesn't exist (not an error)
		return []model.Skill{}, nil
	}

	var skills []model.Skill

	// Parse config.toml for custom instructions
	configSkill, err := p.parseConfigFile()
	if err == nil && configSkill != nil {
		skills = append(skills, *configSkill)
	}

	// Parse AGENTS.md files
	agentsSkills, err := p.parseAgentsFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to parse AGENTS.md files: %w", err)
	}
	skills = append(skills, agentsSkills...)

	return skills, nil
}

// parseConfigFile parses the config.toml file and extracts instructions as a skill
func (p *Parser) parseConfigFile() (*model.Skill, error) {
	configPath := filepath.Join(p.basePath, "config.toml")

	// Check if config exists
	fileInfo, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		return nil, nil // Not an error, just no config
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat config file: %w", err)
	}

	// Parse TOML config
	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config.toml: %w", err)
	}

	// Only create a skill if there are instructions
	if config.Instructions == "" && config.DeveloperInstructions == "" {
		return nil, nil
	}

	// Combine instructions
	content := ""
	if config.Instructions != "" {
		content = config.Instructions
	}
	if config.DeveloperInstructions != "" {
		if content != "" {
			content += "\n\n"
		}
		content += config.DeveloperInstructions
	}

	// Build metadata from config
	metadata := make(map[string]string)
	if config.Model != "" {
		metadata["model"] = config.Model
	}
	if config.ApprovalPolicy != "" {
		metadata["approval_policy"] = config.ApprovalPolicy
	}
	if config.SandboxMode != "" {
		metadata["sandbox_mode"] = config.SandboxMode
	}
	if config.Profile != "" {
		metadata["profile"] = config.Profile
	}

	skill := model.Skill{
		Name:        "codex-config",
		Description: "Codex CLI configuration instructions",
		Platform:    model.Codex,
		Path:        configPath,
		Metadata:    metadata,
		Content:     parser.NormalizeContent(content),
		ModifiedAt:  fileInfo.ModTime(),
	}

	return &skill, nil
}

// parseAgentsFiles finds and parses AGENTS.md files
func (p *Parser) parseAgentsFiles() ([]model.Skill, error) {
	// Discover AGENTS.md files
	patterns := []string{"AGENTS.md", "**/AGENTS.md"}
	files, err := parser.DiscoverFiles(p.basePath, patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to discover AGENTS.md files: %w", err)
	}

	// Parse each file
	skills := make([]model.Skill, 0, len(files))
	for _, filePath := range files {
		skill, err := p.parseAgentsFile(filePath)
		if err != nil {
			// Skip files with errors
			continue
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

// parseAgentsFile parses a single AGENTS.md file
func (p *Parser) parseAgentsFile(filePath string) (model.Skill, error) {
	// Read file content
	// #nosec G304 - filePath is validated through directory traversal from basePath
	content, err := os.ReadFile(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	// Get file modification time
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	// Generate name from relative path
	relPath, err := filepath.Rel(p.basePath, filePath)
	if err != nil {
		relPath = filepath.Base(filePath)
	}

	// Create name: use directory name if nested, otherwise just "agents"
	name := "agents"
	dir := filepath.Dir(relPath)
	if dir != "." && dir != "" {
		// Use the directory name as part of the skill name
		name = filepath.Base(dir) + "-agents"
	}

	// Validate skill name
	if err := parser.ValidateSkillName(name); err != nil {
		// Generate a safe name
		name = "codex-agents"
	}

	skill := model.Skill{
		Name:        name,
		Description: "Codex AGENTS.md instructions",
		Platform:    model.Codex,
		Path:        filePath,
		Metadata:    map[string]string{"type": "agents"},
		Content:     parser.NormalizeContent(string(content)),
		ModifiedAt:  fileInfo.ModTime(),
	}

	return skill, nil
}

// Platform returns the platform identifier for Codex
func (p *Parser) Platform() model.Platform {
	return model.Codex
}

// DefaultPath returns the default path for Codex configuration
func (p *Parser) DefaultPath() string {
	return filepath.Join(util.HomeDir(), ".codex")
}
