// Package codex implements the Parser interface for OpenAI Codex CLI skills.
// Codex uses TOML configuration (config.toml) and AGENTS.md files for instructions.
package codex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/parser/skills"
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

// Parse parses Codex skills from SKILL.md files, config.toml, and AGENTS.md files
// Supports both:
// 1. Agent Skills Standard: SKILL.md files in subdirectories (takes precedence)
// 2. Legacy formats: config.toml instructions and AGENTS.md files
func (p *Parser) Parse() ([]model.Skill, error) {
	// Check if the base path exists
	if _, err := os.Stat(p.basePath); os.IsNotExist(err) {
		logging.Debug("config directory not found",
			logging.Platform(string(p.Platform())),
			logging.Path(p.basePath),
		)
		return []model.Skill{}, nil
	}

	var allSkills []model.Skill
	seenNames := make(map[string]bool)

	// First, parse SKILL.md files (Agent Skills Standard format)
	// These take precedence over legacy formats when names collide
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
		if len(agentSkills) > 0 {
			logging.Debug("discovered SKILL.md files",
				logging.Platform(string(p.Platform())),
				logging.Path(p.basePath),
				logging.Count(len(agentSkills)),
			)
		}
	}

	// Parse config.toml for custom instructions
	configSkill, err := p.parseConfigFile()
	if err == nil && configSkill != nil {
		// Skip if a SKILL.md with the same name was already parsed
		if seenNames[configSkill.Name] {
			logging.Debug("skipping config.toml skill, SKILL.md version takes precedence",
				logging.Skill(configSkill.Name),
				logging.Path(configSkill.Path),
			)
		} else {
			seenNames[configSkill.Name] = true
			allSkills = append(allSkills, *configSkill)
		}
	}

	// Parse AGENTS.md files
	agentsSkills, err := p.parseAgentsFiles(seenNames)
	if err != nil {
		logging.Error("failed to parse AGENTS.md files",
			logging.Platform(string(p.Platform())),
			logging.Path(p.basePath),
			logging.Err(err),
		)
		return nil, fmt.Errorf("failed to parse AGENTS.md files: %w", err)
	}
	allSkills = append(allSkills, agentsSkills...)

	logging.Debug("completed parsing skills",
		logging.Platform(string(p.Platform())),
		logging.Count(len(allSkills)),
	)

	return allSkills, nil
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
// seenNames tracks skill names that have already been parsed (from SKILL.md or config.toml)
func (p *Parser) parseAgentsFiles(seenNames map[string]bool) ([]model.Skill, error) {
	// Discover AGENTS.md files
	patterns := []string{"AGENTS.md", "**/AGENTS.md"}
	files, err := parser.DiscoverFiles(p.basePath, patterns)
	if err != nil {
		logging.Error("failed to discover AGENTS.md files",
			logging.Platform(string(p.Platform())),
			logging.Path(p.basePath),
			logging.Err(err),
		)
		return nil, fmt.Errorf("failed to discover AGENTS.md files: %w", err)
	}

	// Filter out SKILL.md files (already parsed by skills parser)
	var legacyFiles []string
	for _, f := range files {
		if !strings.HasSuffix(f, "SKILL.md") {
			legacyFiles = append(legacyFiles, f)
		}
	}

	logging.Debug("discovered AGENTS.md files",
		logging.Platform(string(p.Platform())),
		logging.Path(p.basePath),
		logging.Count(len(legacyFiles)),
	)

	// Parse each file
	parsedSkills := make([]model.Skill, 0, len(legacyFiles))
	for _, filePath := range legacyFiles {
		skill, err := p.parseAgentsFile(filePath)
		if err != nil {
			logging.Warn("failed to parse AGENTS.md file",
				logging.Platform(string(p.Platform())),
				logging.Path(filePath),
				logging.Err(err),
			)
			continue
		}
		// Skip if a SKILL.md or config.toml skill with the same name was already parsed
		if seenNames[skill.Name] {
			logging.Debug("skipping legacy AGENTS.md skill, higher precedence version exists",
				logging.Skill(skill.Name),
				logging.Path(filePath),
			)
			continue
		}
		seenNames[skill.Name] = true
		parsedSkills = append(parsedSkills, skill)
	}

	return parsedSkills, nil
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
