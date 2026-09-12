// Package tiered provides a parser that searches multiple locations in precedence order.
package tiered

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/util"
)

// ParserFactory creates a parser for a given path.
// Different platforms use different parser implementations.
type ParserFactory func(basePath string) parser.Parser

// Parser aggregates skills from multiple locations with scope-based precedence.
// Skills from higher-precedence scopes override those with matching names from lower scopes.
type Parser struct {
	platform      model.Platform
	pathConfig    util.TieredPathConfig
	parserFactory ParserFactory
	searchPaths   []util.ScopedPath
	pluginParser  parser.Parser // optional; nil means no plugin-scope layer
}

// Config holds configuration for creating a TieredParser.
type Config struct {
	// Platform is the target platform
	Platform model.Platform
	// WorkingDir is the current working directory (for repo scope discovery)
	WorkingDir string
	// RepoRoot is the repository root (optional, auto-detected if empty)
	RepoRoot string
	// AdminPath is an optional admin-level path
	AdminPath string
	// SystemPath is an optional system-level path
	SystemPath string
	// ParserFactory creates platform-specific parsers
	ParserFactory ParserFactory
	// SearchPaths optionally overrides default scope resolution.
	SearchPaths []util.ScopedPath
	// PluginParser is an optional parser for the plugin scope.
	// When set, plugin-scope skills are sourced from this parser rather than
	// a filesystem path. This is used for Claude Code to surface skills from
	// the installed-plugins cache as a distinct platform layer.
	PluginParser parser.Parser
}

// New creates a new TieredParser with the given configuration.
func New(cfg Config) *Parser {
	return &Parser{
		platform: cfg.Platform,
		pathConfig: util.TieredPathConfig{
			WorkingDir: cfg.WorkingDir,
			RepoRoot:   cfg.RepoRoot,
			Platform:   cfg.Platform,
			AdminPath:  cfg.AdminPath,
			SystemPath: cfg.SystemPath,
		},
		parserFactory: cfg.ParserFactory,
		searchPaths:   cfg.SearchPaths,
		pluginParser:  cfg.PluginParser,
	}
}

// Parse discovers and parses skills from all configured locations.
// Skills are merged with precedence-based deduplication.
func (p *Parser) Parse() ([]model.Skill, error) {
	searchPaths := p.searchPaths
	if len(searchPaths) == 0 {
		searchPaths = util.GetAllSearchPaths(p.pathConfig)
	}

	// Collect skills from all paths, tracking seen names for deduplication
	skillsByName := make(map[string]model.Skill)
	var allSkills []model.Skill

	for _, sp := range searchPaths {
		// Skip non-existent paths
		if p.skipMissingPath(sp.Path) {
			logging.Debug(
				"tiered lookup: path not found",
				logging.Platform(string(p.platform)),
				logging.Path(sp.Path),
				slog.String("scope", string(sp.Scope)),
			)
			continue
		}

		logging.Debug(
			"tiered lookup: searching path",
			logging.Platform(string(p.platform)),
			logging.Path(sp.Path),
			slog.String("scope", string(sp.Scope)),
		)

		// Create a parser for this path
		pathParser := p.parserFactory(sp.Path)

		// Parse skills from this location
		skills, err := pathParser.Parse()
		if err != nil {
			logging.Warn(
				"tiered lookup: failed to parse path",
				logging.Platform(string(p.platform)),
				logging.Path(sp.Path),
				logging.Err(err),
			)
			continue
		}

		// Assign scope to each skill and handle deduplication
		for _, skill := range skills {
			skill.Scope = sp.Scope
			// Re-scope skills that were installed via the plugin system.
			// These appear in user-scope paths as symlinks but belong to the
			// plugin layer, not ordinary user files.
			if skill.PluginInfo != nil {
				skill.Scope = model.ScopePlugin
			}

			// Check for name collision
			if existing, exists := skillsByName[skill.Name]; exists {
				// Keep skill with higher precedence
				if skill.Scope.IsHigherPrecedence(existing.Scope) {
					logging.Debug(
						"tiered lookup: skill override",
						logging.Skill(skill.Name),
						slog.String("newScope", string(skill.Scope)),
						slog.String("existingScope", string(existing.Scope)),
					)
					skillsByName[skill.Name] = skill
				}
			} else {
				skillsByName[skill.Name] = skill
			}
		}
	}

	// Include plugin-scope skills from the dedicated plugin parser (e.g. Claude cache).
	// These are treated as a distinct platform layer separate from filesystem scopes.
	if p.pluginParser != nil {
		pluginSkills, err := p.pluginParser.Parse()
		if err != nil {
			logging.Warn(
				"tiered lookup: failed to parse plugin scope",
				logging.Platform(string(p.platform)),
				logging.Err(err),
			)
		} else {
			for _, skill := range pluginSkills {
				skill.Scope = model.ScopePlugin
				if existing, exists := skillsByName[skill.Name]; exists {
					if skill.Scope.IsHigherPrecedence(existing.Scope) {
						skillsByName[skill.Name] = skill
					}
				} else {
					skillsByName[skill.Name] = skill
				}
			}
		}
	}

	// Convert map to slice
	for _, skill := range skillsByName {
		allSkills = append(allSkills, skill)
	}

	logging.Debug(
		"tiered lookup: completed",
		logging.Platform(string(p.platform)),
		logging.Count(len(allSkills)),
	)

	return allSkills, nil
}

// ParseWithScopeFilter parses skills but only from the specified scopes.
func (p *Parser) ParseWithScopeFilter(scopes []model.SkillScope) ([]model.Skill, error) {
	searchPaths := p.searchPaths
	if len(searchPaths) == 0 {
		searchPaths = util.GetAllSearchPaths(p.pathConfig)
	}

	// Build scope filter set
	scopeSet := make(map[model.SkillScope]bool)
	for _, s := range scopes {
		scopeSet[s] = true
	}

	// Filter paths to only include requested scopes
	var filteredPaths []util.ScopedPath
	for _, sp := range searchPaths {
		if scopeSet[sp.Scope] {
			filteredPaths = append(filteredPaths, sp)
		}
	}

	skillsByName := make(map[string]model.Skill)
	var allSkills []model.Skill

	for _, sp := range filteredPaths {
		if p.skipMissingPath(sp.Path) {
			continue
		}

		pathParser := p.parserFactory(sp.Path)
		skills, err := pathParser.Parse()
		if err != nil {
			logging.Warn(
				"tiered lookup: failed to parse path",
				logging.Platform(string(p.platform)),
				logging.Path(sp.Path),
				logging.Err(err),
			)
			continue
		}

		for _, skill := range skills {
			skill.Scope = sp.Scope
			if skill.PluginInfo != nil {
				skill.Scope = model.ScopePlugin
			}
			if existing, exists := skillsByName[skill.Name]; exists {
				if skill.Scope.IsHigherPrecedence(existing.Scope) {
					skillsByName[skill.Name] = skill
				}
			} else {
				skillsByName[skill.Name] = skill
			}
		}
	}

	// Include plugin-scope skills if requested and a plugin parser is configured.
	if scopeSet[model.ScopePlugin] && p.pluginParser != nil {
		pluginSkills, err := p.pluginParser.Parse()
		if err != nil {
			logging.Warn(
				"tiered lookup: failed to parse plugin scope",
				logging.Platform(string(p.platform)),
				logging.Err(err),
			)
		} else {
			for _, skill := range pluginSkills {
				skill.Scope = model.ScopePlugin
				if existing, exists := skillsByName[skill.Name]; exists {
					if skill.Scope.IsHigherPrecedence(existing.Scope) {
						skillsByName[skill.Name] = skill
					}
				} else {
					skillsByName[skill.Name] = skill
				}
			}
		}
	}

	for _, skill := range skillsByName {
		allSkills = append(allSkills, skill)
	}

	return allSkills, nil
}

// ParseFromScope parses skills from only a single scope.
func (p *Parser) ParseFromScope(scope model.SkillScope) ([]model.Skill, error) {
	// Plugin scope is served by the dedicated plugin parser, not filesystem paths.
	if scope == model.ScopePlugin {
		if p.pluginParser == nil {
			return []model.Skill{}, nil
		}
		skills, err := p.pluginParser.Parse()
		if err != nil {
			return nil, fmt.Errorf("parse plugin skills: %w", err)
		}
		for i := range skills {
			skills[i].Scope = model.ScopePlugin
		}
		return skills, nil
	}

	if len(p.searchPaths) > 0 {
		var scopePaths []string
		for _, sp := range p.searchPaths {
			if sp.Scope == scope {
				scopePaths = append(scopePaths, sp.Path)
			}
		}
		if len(scopePaths) == 0 {
			return []model.Skill{}, nil
		}

		var allSkills []model.Skill
		seen := make(map[string]bool)
		for _, path := range scopePaths {
			if p.skipMissingPath(path) {
				continue
			}

			pathParser := p.parserFactory(path)
			skills, err := pathParser.Parse()
			if err != nil {
				logging.Warn(
					"tiered lookup: failed to parse path",
					logging.Platform(string(p.platform)),
					logging.Path(path),
					logging.Err(err),
				)
				continue
			}

			for _, skill := range skills {
				skill.Scope = scope
				if !seen[skill.Name] {
					seen[skill.Name] = true
					allSkills = append(allSkills, skill)
				}
			}
		}

		return allSkills, nil
	}

	paths := util.GetTieredPaths(p.pathConfig)
	scopePaths, ok := paths[scope]
	if !ok || len(scopePaths) == 0 {
		return []model.Skill{}, nil
	}

	var allSkills []model.Skill
	seen := make(map[string]bool)

	for _, path := range scopePaths {
		if p.skipMissingPath(path) {
			continue
		}

		pathParser := p.parserFactory(path)
		skills, err := pathParser.Parse()
		if err != nil {
			logging.Warn(
				"tiered lookup: failed to parse path",
				logging.Platform(string(p.platform)),
				logging.Path(path),
				logging.Err(err),
			)
			continue
		}

		for _, skill := range skills {
			skill.Scope = scope
			// Within same scope, first found wins (for multiple repo paths)
			if !seen[skill.Name] {
				seen[skill.Name] = true
				allSkills = append(allSkills, skill)
			}
		}
	}

	return allSkills, nil
}

// Platform returns the platform this parser handles.
func (p *Parser) Platform() model.Platform {
	return p.platform
}

func (p *Parser) skipMissingPath(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return false
	}
	// Codex and Copilot parsers derive native configuration/instruction roots
	// from their canonical skills roots, so sibling artifacts can exist even
	// when the skills directory itself does not.
	return p.platform != model.Codex && p.platform != model.Copilot
}

// DefaultPath returns the user-level skills path for the platform.
func (p *Parser) DefaultPath() string {
	return util.PlatformSkillsPath(p.platform)
}

// GetSearchPaths returns the configured search paths in precedence order.
func (p *Parser) GetSearchPaths() []util.ScopedPath {
	if len(p.searchPaths) > 0 {
		return append([]util.ScopedPath(nil), p.searchPaths...)
	}
	return util.GetAllSearchPaths(p.pathConfig)
}

// GetExistingSearchPaths returns only the search paths that exist on disk.
func (p *Parser) GetExistingSearchPaths() []util.ScopedPath {
	return util.FilterExistingPaths(util.GetAllSearchPaths(p.pathConfig))
}

// MergeSkills merges multiple skill slices, applying precedence-based deduplication.
// Skills with higher precedence scopes override those with lower precedence.
// This is useful for combining skills from different sources.
func MergeSkills(skillSets ...[]model.Skill) []model.Skill {
	skillsByName := make(map[string]model.Skill)

	for _, skills := range skillSets {
		for _, skill := range skills {
			if existing, exists := skillsByName[skill.Name]; exists {
				if skill.Scope.IsHigherPrecedence(existing.Scope) {
					skillsByName[skill.Name] = skill
				}
			} else {
				skillsByName[skill.Name] = skill
			}
		}
	}

	result := make([]model.Skill, 0, len(skillsByName))
	for _, skill := range skillsByName {
		result = append(result, skill)
	}

	return result
}

// DeduplicateByName removes duplicate skills by name, keeping the first occurrence.
// Unlike MergeSkills, this doesn't consider scope precedence.
func DeduplicateByName(skills []model.Skill) []model.Skill {
	seen := make(map[string]bool)
	result := make([]model.Skill, 0, len(skills))

	for _, skill := range skills {
		if !seen[skill.Name] {
			seen[skill.Name] = true
			result = append(result, skill)
		}
	}

	return result
}
