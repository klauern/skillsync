package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/tiered"
	"github.com/klauern/skillsync/internal/util"
)

var parsePlatformSkillsFn = parsePlatformSkills
var discoverPluginSkillsFn = discoverPluginSkills

// parsePlatformSkillsWithScope loads configured search paths for a platform and
// parses matching skills for the requested scopes.
func parsePlatformSkillsWithScope(platform model.Platform, scopeFilter []model.SkillScope, includePlugins bool) ([]model.Skill, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	paths, repoRoot, err := platformSkillsPaths(cfg, platform)
	if err != nil {
		return nil, fmt.Errorf("resolve platform skills paths for %s: %w", platform, err)
	}
	if len(paths) == 0 {
		return []model.Skill{}, nil
	}

	return parsePlatformSkillsFromPaths(platform, paths, repoRoot, scopeFilter, includePlugins), nil
}

func discoverSkillsAcrossPlatforms(platforms []model.Platform) []model.Skill {
	var allSkills []model.Skill

	for _, platform := range platforms {
		skills, err := parsePlatformSkillsFn(platform)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed parse %s: %v\n", platform, err)
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	return allSkills
}

func discoverSkillsAcrossPlatformsForTUI(platforms []model.Platform, includePlugins bool) []model.Skill {
	allSkills := discoverSkillsAcrossPlatforms(platforms)
	if !includePlugins {
		return allSkills
	}

	pluginSkills, err := discoverPluginSkillsFn("", true)
	if err != nil {
		return allSkills
	}

	return append(allSkills, pluginSkills...)
}

var platformConfigGetters = map[model.Platform]func(*config.Config) *config.PlatformConfig{
	model.ClaudeCode: func(cfg *config.Config) *config.PlatformConfig { return &cfg.Platforms.ClaudeCode },
	model.Cursor:     func(cfg *config.Config) *config.PlatformConfig { return &cfg.Platforms.Cursor },
	model.Codex:      func(cfg *config.Config) *config.PlatformConfig { return &cfg.Platforms.Codex },
	model.Copilot:    func(cfg *config.Config) *config.PlatformConfig { return &cfg.Platforms.Copilot },
	model.Gemini:     func(cfg *config.Config) *config.PlatformConfig { return &cfg.Platforms.Gemini },
	model.PiDev:      func(cfg *config.Config) *config.PlatformConfig { return &cfg.Platforms.PiDev },
}

func platformRawSkillsPaths(cfg *config.Config, platform model.Platform) ([]string, error) {
	getter, ok := platformConfigGetters[platform]
	if !ok {
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	return getter(cfg).SkillsPaths, nil
}

// platformSkillsPaths returns the resolved, deduplicated search paths for a
// platform along with the detected repository root.
func platformSkillsPaths(cfg *config.Config, platform model.Platform) ([]util.ScopedPath, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get working directory: %w", err)
	}

	repoRoot := util.GetRepoRoot(cwd)
	rawPaths, err := platformRawSkillsPaths(cfg, platform)
	if err != nil {
		return nil, repoRoot, err
	}

	if platform == model.PiDev {
		return platformSkillsPathsForPiDev(rawPaths, cwd, repoRoot), repoRoot, nil
	}

	paths := scopedPathsFromStrings(resolveSkillsPaths(rawPaths, cwd, repoRoot), repoRoot)
	if platform == model.ClaudeCode {
		paths = appendClaudeCodeCommandPaths(paths, cwd, repoRoot)
	}

	return paths, repoRoot, nil
}

func platformSkillsPathsForPiDev(rawPaths []string, cwd, repoRoot string) []util.ScopedPath {
	discoveredPaths := util.GetAllSearchPaths(util.TieredPathConfig{
		WorkingDir: cwd,
		RepoRoot:   repoRoot,
		Platform:   model.PiDev,
	})

	paths := make([]util.ScopedPath, 0, len(discoveredPaths)+len(rawPaths))
	seen := make(map[string]bool, len(discoveredPaths)+len(rawPaths))
	appendScopedPath := func(sp util.ScopedPath) {
		if seen[sp.Path] {
			return
		}
		paths = append(paths, sp)
		seen[sp.Path] = true
	}

	for _, sp := range discoveredPaths {
		appendScopedPath(sp)
	}
	for _, p := range resolveSkillsPaths(rawPaths, cwd, repoRoot) {
		appendScopedPath(util.ScopedPath{Path: p, Scope: inferScopeForPath(p, repoRoot)})
	}

	return paths
}

func appendClaudeCodeCommandPaths(paths []util.ScopedPath, cwd, repoRoot string) []util.ScopedPath {
	commandPaths := resolveSkillsPaths([]string{".claude/commands", "~/.claude/commands"}, cwd, repoRoot)
	seen := make(map[string]bool, len(paths))
	for _, p := range paths {
		seen[p.Path] = true
	}

	for _, p := range commandPaths {
		if seen[p] {
			continue
		}
		paths = append(paths, util.ScopedPath{Path: p, Scope: inferScopeForPath(p, repoRoot)})
	}

	return paths
}

func resolveSkillsPaths(rawPaths []string, cwd, repoRoot string) []string {
	paths := make([]string, 0, len(rawPaths))
	seen := make(map[string]bool)
	addPath := func(path string) {
		if path == "" || seen[path] {
			return
		}
		paths = append(paths, path)
		seen[path] = true
	}

	for _, rawPath := range rawPaths {
		rawPath = strings.TrimSpace(rawPath)
		if rawPath == "" {
			continue
		}

		if filepath.IsAbs(rawPath) || strings.HasPrefix(rawPath, "~") {
			addPath(util.ExpandPath(rawPath, cwd))
			continue
		}

		addPath(util.ExpandPath(rawPath, cwd))
		if repoRoot != "" && repoRoot != cwd {
			addPath(util.ExpandPath(rawPath, repoRoot))
		}
	}

	return paths
}

func scopedPathsFromStrings(paths []string, repoRoot string) []util.ScopedPath {
	scoped := make([]util.ScopedPath, 0, len(paths))
	for _, path := range paths {
		scoped = append(scoped, util.ScopedPath{Path: path, Scope: inferScopeForPath(path, repoRoot)})
	}
	return scoped
}

func parsePlatformSkillsFromPaths(
	platform model.Platform,
	paths []util.ScopedPath,
	repoRoot string,
	scopeFilter []model.SkillScope,
	includePlugins bool,
) []model.Skill {
	parserFactory, err := tiered.ParserFactoryFor(platform)
	if err != nil {
		logging.Warn("unsupported platform for skill parsing", logging.Err(err))
		return nil
	}

	skillsByName := make(map[string]model.Skill)
	scopeSet := make(map[model.SkillScope]bool, len(scopeFilter))
	for _, scope := range scopeFilter {
		scopeSet[scope] = true
	}

	for _, sp := range paths {
		scope := sp.Scope
		if scope == "" {
			scope = inferScopeForPath(sp.Path, repoRoot)
		}
		if len(scopeSet) > 0 && !scopeSet[scope] {
			continue
		}
		if _, err := os.Stat(sp.Path); err != nil {
			continue
		}

		pathParser := parserFactory(sp.Path)
		skills, err := pathParser.Parse()
		if err != nil {
			logging.Warn("failed to parse skills", logging.Err(err), logging.Path(sp.Path))
			continue
		}

		for _, skill := range skills {
			skill.Scope = scope
			if existing, exists := skillsByName[skill.Name]; exists {
				if shouldOverrideSkill(existing, skill) {
					skillsByName[skill.Name] = skill
				}
				continue
			}
			skillsByName[skill.Name] = skill
		}
	}

	pluginExplicitlyRequested := scopeSet[model.ScopePlugin]
	if platform == model.ClaudeCode && (pluginExplicitlyRequested || includePlugins) {
		for _, skill := range parseClaudePluginCacheSkills() {
			if existing, exists := skillsByName[skill.Name]; exists {
				if shouldOverrideSkill(existing, skill) {
					skillsByName[skill.Name] = skill
				}
				continue
			}
			skillsByName[skill.Name] = skill
		}
	}

	result := make([]model.Skill, 0, len(skillsByName))
	for _, skill := range skillsByName {
		result = append(result, skill)
	}
	return result
}

func shouldOverrideSkill(existing, candidate model.Skill) bool {
	if candidate.Scope.IsHigherPrecedence(existing.Scope) {
		return true
	}
	if existing.Scope != candidate.Scope {
		return false
	}

	existingType := existing.Type
	if existingType == "" {
		existingType = model.SkillTypeSkill
	}
	candidateType := candidate.Type
	if candidateType == "" {
		candidateType = model.SkillTypeSkill
	}

	return existingType == model.SkillTypePrompt && candidateType == model.SkillTypeSkill
}

func parseClaudePluginCacheSkills() []model.Skill {
	cacheParser := claude.NewCachePluginsParser("")
	skills, err := cacheParser.Parse()
	if err != nil {
		logging.Warn("failed to discover Claude plugin cache skills", logging.Err(err))
		return nil
	}
	return skills
}

func inferScopeForPath(path, repoRoot string) model.SkillScope {
	cleaned := filepath.Clean(path)

	if repoRoot != "" {
		root := filepath.Clean(repoRoot)
		rootWithSep := root + string(os.PathSeparator)
		if cleaned == root || strings.HasPrefix(cleaned, rootWithSep) {
			return model.ScopeRepo
		}
	}

	pluginCachePath := filepath.Clean(util.ClaudePluginCachePath())
	pluginCacheWithSep := pluginCachePath + string(os.PathSeparator)
	if cleaned == pluginCachePath || strings.HasPrefix(cleaned, pluginCacheWithSep) {
		return model.ScopePlugin
	}

	home := filepath.Clean(util.HomeDir())
	homeWithSep := home + string(os.PathSeparator)
	if home != "" && (cleaned == home || strings.HasPrefix(cleaned, homeWithSep)) {
		return model.ScopeUser
	}

	etcPrefix := string(os.PathSeparator) + "etc" + string(os.PathSeparator)
	if strings.HasPrefix(cleaned, etcPrefix) {
		return model.ScopeSystem
	}

	optPrefix := string(os.PathSeparator) + "opt" + string(os.PathSeparator)
	if strings.HasPrefix(cleaned, optPrefix) {
		return model.ScopeAdmin
	}

	return model.ScopeUser
}
