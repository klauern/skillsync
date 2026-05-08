// Package piagent implements Pi Agent skill parsing and discovery.
package piagent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/skills"
	"github.com/klauern/skillsync/internal/util"
)

// Parser implements parser.Parser for Pi Agent skills.
type Parser struct {
	basePath string
}

// New creates a Pi Agent parser rooted at basePath.
func New(basePath string) *Parser {
	if basePath == "" {
		basePath = util.PiAgentSkillsPath()
	}
	return &Parser{basePath: basePath}
}

// Parse parses Pi Agent skills from the configured skill directory.
func (p *Parser) Parse() ([]model.Skill, error) {
	return skills.New(p.basePath, model.PiAgent).Parse()
}

// Platform returns the platform this parser handles.
func (p *Parser) Platform() model.Platform {
	return model.PiAgent
}

// DefaultPath returns the default Pi Agent skill directory.
func (p *Parser) DefaultPath() string {
	return util.PiAgentSkillsPath()
}

type settingsFile struct {
	SkillsDirectories []string `json:"skillsDirectories"`
}

// DiscoverSearchPaths resolves Pi Agent search paths in precedence order.
func DiscoverSearchPaths(workingDir string) ([]util.ScopedPath, error) {
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	repoRoot := util.GetRepoRoot(workingDir)
	if repoRoot == "" {
		repoRoot = workingDir
	}

	var result []util.ScopedPath
	seen := make(map[string]bool)
	add := func(path string, scope model.SkillScope) {
		if path == "" {
			return
		}
		cleaned := filepath.Clean(path)
		if seen[cleaned] {
			return
		}
		seen[cleaned] = true
		result = append(result, util.ScopedPath{Path: cleaned, Scope: scope})
	}

	for _, path := range ancestorSkillPaths(workingDir, repoRoot) {
		add(path, model.ScopeRepo)
	}

	projectPaths, err := parseSettingsSkillPaths(filepath.Join(repoRoot, ".pi", "settings.json"))
	if err != nil {
		return nil, err
	}
	for _, path := range projectPaths {
		add(path, model.ScopeRepo)
	}

	add(util.PiAgentSkillsPath(), model.ScopeUser)

	userPaths, err := parseSettingsSkillPaths(filepath.Join(util.HomeDir(), ".config", "pi", "settings.json"))
	if err != nil {
		return nil, err
	}
	for _, path := range userPaths {
		add(path, model.ScopeUser)
	}

	return result, nil
}

// ancestorSkillPaths builds a slice of candidate skill directory paths by walking
// from workingDir up to repoRoot (inclusive). Each entry is the path formed by
// joining the directory with ".agents" and "skills".
func ancestorSkillPaths(workingDir, repoRoot string) []string {
	var paths []string

	dir := filepath.Clean(workingDir)
	root := filepath.Clean(repoRoot)
	for {
		paths = append(paths, filepath.Join(dir, ".agents", "skills"))
		if dir == root {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return paths
}

// parseSettingsSkillPaths reads a Pi settings JSON file and returns its configured skill directories
// with each path expanded relative to the settings file's directory.
// If the file does not exist, it returns (nil, nil). If reading or parsing fails, it returns a
// wrapped error describing the failure. Paths are expanded using util.ExpandPath with the settings
// file's directory as the base.
func parseSettingsSkillPaths(settingsPath string) ([]string, error) {
	data, err := os.ReadFile(settingsPath) // #nosec G304 — settingsPath comes from .pi/settings.json discovery, not user input
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", settingsPath, err)
	}

	var settings settingsFile
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", settingsPath, err)
	}

	baseDir := filepath.Dir(settingsPath)
	paths := make([]string, 0, len(settings.SkillsDirectories))
	for _, raw := range settings.SkillsDirectories {
		paths = append(paths, util.ExpandPath(raw, baseDir))
	}

	return paths, nil
}
