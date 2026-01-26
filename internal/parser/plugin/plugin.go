// Package plugin implements the Parser interface for Claude Code plugin repositories.
// It handles Git operations for plugin discovery and parses plugin manifests and skill definitions.
package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/util"
)

// MarketplaceManifest represents the root .claude-plugin/marketplace.json file
type MarketplaceManifest struct {
	Name     string `json:"name"`
	Metadata struct {
		Description string `json:"description"`
		Version     string `json:"version"`
	} `json:"metadata"`
	Owner struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"owner"`
	Plugins []Ref `json:"plugins"`
}

// Ref represents a reference to a plugin in the marketplace manifest
type Ref struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

// Manifest represents a plugin's .claude-plugin/plugin.json file
type Manifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Author      struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"author"`
}

// Parser implements the parser.Parser interface for Claude Code plugin repositories
type Parser struct {
	basePath string
	repoURL  string
}

// New creates a new plugin repository parser.
// If basePath is empty, uses the default plugins directory (~/.skillsync/plugins).
// The repoURL is optional and used for cloning remote repositories.
func New(basePath string) *Parser {
	if basePath == "" {
		basePath = util.SkillsyncPluginsPath()
	}
	return &Parser{basePath: basePath}
}

// NewWithRepo creates a parser for a specific Git repository URL.
// The repository will be cloned to the plugins directory if not already present.
func NewWithRepo(repoURL string) *Parser {
	return &Parser{
		basePath: util.SkillsyncPluginsPath(),
		repoURL:  repoURL,
	}
}

// Parse parses Claude Code plugins from a local directory or cloned repository.
// If a repoURL is configured, it will clone/pull the repository first.
func (p *Parser) Parse() ([]model.Skill, error) {
	// If we have a repo URL, handle Git operations first
	repoPath := p.basePath
	if p.repoURL != "" {
		var err error
		repoPath, err = p.ensureRepo()
		if err != nil {
			logging.Error("failed to ensure repository",
				logging.Platform(string(p.Platform())),
				logging.Path(p.repoURL),
				logging.Err(err),
			)
			return nil, fmt.Errorf("failed to ensure repository: %w", err)
		}
	}

	// Check if the base path exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		logging.Debug("plugins directory not found",
			logging.Platform(string(p.Platform())),
			logging.Path(repoPath),
		)
		return []model.Skill{}, nil
	}

	// Try to parse as a plugin repository with marketplace.json
	skills, err := p.parseMarketplace(repoPath)
	if err == nil && len(skills) > 0 {
		logging.Debug("parsed marketplace plugins",
			logging.Platform(string(p.Platform())),
			logging.Path(repoPath),
			logging.Count(len(skills)),
		)
		return skills, nil
	}

	// Fall back to scanning for individual plugins
	scannedSkills, err := p.scanForPlugins(repoPath)
	if err == nil {
		logging.Debug("completed scanning plugins",
			logging.Platform(string(p.Platform())),
			logging.Path(repoPath),
			logging.Count(len(scannedSkills)),
		)
	}
	return scannedSkills, err
}

// parseMarketplace parses skills from a repository with .claude-plugin/marketplace.json
func (p *Parser) parseMarketplace(repoPath string) ([]model.Skill, error) {
	marketplacePath := filepath.Join(repoPath, ".claude-plugin", "marketplace.json")

	// #nosec G304 - path is constructed from trusted repoPath
	data, err := os.ReadFile(marketplacePath)
	if err != nil {
		return nil, fmt.Errorf("marketplace.json not found: %w", err)
	}

	var manifest MarketplaceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		logging.Error("failed to parse marketplace.json",
			logging.Platform(string(p.Platform())),
			logging.Path(marketplacePath),
			logging.Err(err),
		)
		return nil, fmt.Errorf("failed to parse marketplace.json: %w", err)
	}

	logging.Debug("discovered marketplace plugins",
		logging.Platform(string(p.Platform())),
		logging.Path(marketplacePath),
		logging.Count(len(manifest.Plugins)),
	)

	var skills []model.Skill

	// Parse each plugin referenced in the marketplace
	for _, pluginRef := range manifest.Plugins {
		pluginPath := filepath.Join(repoPath, strings.TrimPrefix(pluginRef.Source, "./"))
		pluginSkills, err := p.parsePlugin(pluginPath, manifest.Name)
		if err != nil {
			logging.Warn("failed to parse plugin",
				logging.Platform(string(p.Platform())),
				logging.Path(pluginPath),
				logging.Err(err),
			)
			continue
		}
		skills = append(skills, pluginSkills...)
	}

	return skills, nil
}

// parsePlugin parses all skills from a single plugin directory
func (p *Parser) parsePlugin(pluginPath, repoName string) ([]model.Skill, error) {
	// Read plugin manifest if available
	var pluginManifest *Manifest
	manifestPath := filepath.Join(pluginPath, ".claude-plugin", "plugin.json")
	// #nosec G304 - path is constructed from trusted pluginPath
	if data, err := os.ReadFile(manifestPath); err == nil {
		var m Manifest
		if json.Unmarshal(data, &m) == nil {
			pluginManifest = &m
		}
	}

	// Find all SKILL.md files in the plugin directory
	patterns := []string{"**/SKILL.md", "SKILL.md"}
	files, err := parser.DiscoverFiles(pluginPath, patterns)
	if err != nil {
		logging.Error("failed to discover skill files",
			logging.Platform(string(p.Platform())),
			logging.Path(pluginPath),
			logging.Err(err),
		)
		return nil, fmt.Errorf("failed to discover skill files: %w", err)
	}

	logging.Debug("discovered skill files in plugin",
		logging.Platform(string(p.Platform())),
		logging.Path(pluginPath),
		logging.Count(len(files)),
	)

	var skills []model.Skill
	for _, filePath := range files {
		skill, err := p.parseSkillFile(filePath, pluginManifest, repoName)
		if err != nil {
			logging.Warn("failed to parse skill file",
				logging.Platform(string(p.Platform())),
				logging.Path(filePath),
				logging.Err(err),
			)
			continue
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

// parseSkillFile parses a single SKILL.md file
func (p *Parser) parseSkillFile(filePath string, pluginManifest *Manifest, repoName string) (model.Skill, error) {
	// #nosec G304 - filePath is validated through directory traversal
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

		// Store remaining fields in metadata
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

	// Derive name from directory if not in frontmatter
	if name == "" {
		// Use the parent directory name as skill name
		name = filepath.Base(filepath.Dir(filePath))
	}

	// Validate skill name
	if err := parser.ValidateSkillName(name); err != nil {
		return model.Skill{}, fmt.Errorf("invalid skill name %q in %q: %w", name, filePath, err)
	}

	// Add plugin metadata if available
	if pluginManifest != nil {
		metadata["plugin"] = pluginManifest.Name
		if pluginManifest.Version != "" {
			metadata["plugin_version"] = pluginManifest.Version
		}
		if pluginManifest.Author.Name != "" {
			metadata["author"] = pluginManifest.Author.Name
		}
	}

	// Add repository name to metadata
	if repoName != "" {
		metadata["repository"] = repoName
	}

	// Mark as from plugin source
	metadata["source"] = "plugin"

	// Get file modification time
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	// Normalize content
	normalizedContent := parser.NormalizeContent(result.Content)

	return model.Skill{
		Name:        name,
		Description: description,
		Platform:    p.Platform(),
		Path:        filePath,
		Tools:       tools,
		Metadata:    metadata,
		Content:     normalizedContent,
		ModifiedAt:  fileInfo.ModTime(),
		Scope:       model.ScopePlugin,
	}, nil
}

// scanForPlugins scans a directory for plugin directories (those with .claude-plugin/plugin.json)
func (p *Parser) scanForPlugins(basePath string) ([]model.Skill, error) {
	var skills []model.Skill

	// Walk the directory looking for plugin.json files
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Look for .claude-plugin/plugin.json
		if info.IsDir() {
			return nil
		}

		if filepath.Base(path) == "plugin.json" && strings.Contains(filepath.Dir(path), ".claude-plugin") {
			pluginDir := filepath.Dir(filepath.Dir(path)) // Go up from .claude-plugin/plugin.json
			pluginSkills, err := p.parsePlugin(pluginDir, "")
			if err == nil {
				skills = append(skills, pluginSkills...)
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan for plugins: %w", err)
	}

	return skills, nil
}

// ensureRepo ensures the repository is cloned and up to date
func (p *Parser) ensureRepo() (string, error) {
	if p.repoURL == "" {
		return p.basePath, nil
	}

	// Create plugins directory if needed
	if err := os.MkdirAll(p.basePath, 0o750); err != nil {
		return "", fmt.Errorf("failed to create plugins directory: %w", err)
	}

	// Derive repo name from URL
	repoName := deriveRepoName(p.repoURL)
	repoPath := filepath.Join(p.basePath, repoName)

	// Check if repo already exists
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		// Repo exists, pull updates (ignore errors - can use existing clone)
		if err := p.gitPull(repoPath); err != nil {
			logging.Debug("git pull failed, using existing clone",
				logging.Platform(string(p.Platform())),
				logging.Path(repoPath),
				logging.Err(err),
			)
		}
		return repoPath, nil
	}

	// Clone the repository
	if err := p.gitClone(p.repoURL, repoPath); err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return repoPath, nil
}

// gitClone clones a Git repository
func (p *Parser) gitClone(url, dest string) error {
	// #nosec G204 - url and dest are from trusted configuration
	cmd := exec.Command("git", "clone", "--depth", "1", url, dest)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// gitPull updates a Git repository
func (p *Parser) gitPull(repoPath string) error {
	// #nosec G204 - repoPath is from trusted configuration
	cmd := exec.Command("git", "-C", repoPath, "pull", "--ff-only")
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// deriveRepoName extracts a repository name from a Git URL
func deriveRepoName(url string) string {
	// Handle SSH URLs (git@github.com:user/repo.git)
	if strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, ":")
		if len(parts) == 2 {
			path := parts[1]
			path = strings.TrimSuffix(path, ".git")
			// Use user/repo format
			return strings.ReplaceAll(path, "/", "-")
		}
	}

	// Handle HTTPS URLs
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		// Use last two parts (user/repo)
		return parts[len(parts)-2] + "-" + parts[len(parts)-1]
	}

	// Fallback to last part
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "unknown"
}

// Platform returns the platform identifier for plugins
func (p *Parser) Platform() model.Platform {
	return model.ClaudeCode
}

// DefaultPath returns the default path for plugin repositories
func (p *Parser) DefaultPath() string {
	return util.SkillsyncPluginsPath()
}

// RepoURL returns the configured repository URL
func (p *Parser) RepoURL() string {
	return p.repoURL
}
