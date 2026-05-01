package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/cache"
	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/plugin"
	"github.com/klauern/skillsync/internal/ui"
	"github.com/klauern/skillsync/internal/ui/tui"
)

func discoveryCommand() *cli.Command {
	return &cli.Command{
		Name:    "discover",
		Aliases: []string{"discovery", "list"},
		Usage:   "Discover and list skills across platforms",
		UsageText: `skillsync discover [options]
   skillsync discover --platform claude-code
   skillsync discover --no-plugins
   skillsync discover --repo https://github.com/user/plugins
   skillsync discover --format json`,
		Description: `Discover and list skills from all supported AI coding platforms.

   Supported platforms: claude-code, cursor, codex, pi.dev

   Plugin discovery: By default, skills from installed Claude Code plugins
   are included from ~/.skillsync/plugins/. Use --no-plugins to exclude them,
   or specify a Git repository with --repo to fetch plugins from.

   Output formats: table (default), json, yaml
   For interactive browsing, use: skillsync tui`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Filter by platform (claude-code, cursor, codex, pi.dev)",
			},
			&cli.StringFlag{
				Name:    "scope",
				Aliases: []string{"s"},
				Usage:   "Filter by scope (repo, user, admin, system, builtin, plugin, all). Comma-separated for multiple.",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "table",
				Usage:   "Output format: table, json, yaml",
			},
			&cli.BoolFlag{
				Name:  "no-plugins",
				Usage: "Exclude skills from installed Claude Code plugins",
			},
			&cli.StringFlag{
				Name:  "repo",
				Usage: "Git repository URL to discover plugins from",
			},
			&cli.BoolFlag{
				Name:  "no-cache",
				Usage: "Disable plugin skill caching",
			},
			&cli.StringFlag{
				Name:    "type",
				Aliases: []string{"t"},
				Usage:   "Filter by skill type (skill, prompt). Comma-separated for multiple.",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			platform := cmd.String("platform")
			scopeStr := cmd.String("scope")
			format := cmd.String("format")
			excludePlugins := cmd.Bool("no-plugins")
			repoURL := cmd.String("repo")
			noCache := cmd.Bool("no-cache")
			typeStr := cmd.String("type")

			// Include plugins by default unless --no-plugins is set
			includePlugins := !excludePlugins

			// Parse type filter
			typeFilter, err := parseTypeFilter(typeStr)
			if err != nil {
				return fmt.Errorf("invalid type: %w", err)
			}

			// Parse scope filter
			var scopeFilter []model.SkillScope
			if scopeStr != "" && scopeStr != "all" {
				for _, s := range strings.Split(scopeStr, ",") {
					scope, err := model.ParseScope(strings.TrimSpace(s))
					if err != nil {
						return fmt.Errorf("invalid scope: %w", err)
					}
					scopeFilter = append(scopeFilter, scope)
				}
			}

			// Determine which platforms to scan
			var platforms []model.Platform
			if platform != "" {
				p, err := model.ParsePlatform(platform)
				if err != nil {
					return fmt.Errorf("invalid platform: %w", err)
				}
				platforms = []model.Platform{p}
			} else {
				platforms = model.AllPlatforms()
			}

			// Discover skills from each platform
			// Note: plugins are handled separately by discoverPluginSkills below
			var allSkills []model.Skill
			for _, p := range platforms {
				skills, err := parsePlatformSkillsWithScope(p, scopeFilter, false)
				if err != nil {
					// Log error but continue with other platforms
					fmt.Printf("Warning: failed to parse %s: %v\n", p, err)
					continue
				}
				allSkills = append(allSkills, skills...)
			}

			// Discover plugin skills if requested
			if includePlugins {
				pluginSkills, err := discoverPluginSkills(repoURL, !noCache)
				if err != nil {
					fmt.Printf("Warning: failed to discover plugins: %v\n", err)
				} else {
					allSkills = append(allSkills, pluginSkills...)
				}
			}

			// Filter by skill type if specified
			if len(typeFilter) > 0 {
				allSkills = filterBySkillType(allSkills, typeFilter)
			}

			return outputSkills(allSkills, format)
		},
	}
}

// filterBySkillType filters skills by their type.
// Skills with empty type are treated as SkillTypeSkill (the default).
func filterBySkillType(skills []model.Skill, typeFilter []model.SkillType) []model.Skill {
	if len(typeFilter) == 0 {
		return skills
	}

	filtered := make([]model.Skill, 0, len(skills))
	for _, skill := range skills {
		// Empty type defaults to skill
		skillType := skill.Type
		if skillType == "" {
			skillType = model.SkillTypeSkill
		}

		if slices.Contains(typeFilter, skillType) {
			filtered = append(filtered, skill)
		}
	}
	return filtered
}

// parseTypeFilter parses comma-separated skill types.
// Supports "all" to disable type filtering.
func parseTypeFilter(typeStr string) ([]model.SkillType, error) {
	if strings.TrimSpace(typeStr) == "" || strings.EqualFold(strings.TrimSpace(typeStr), "all") {
		return nil, nil
	}

	var typeFilter []model.SkillType
	for _, t := range strings.Split(typeStr, ",") {
		skillType, err := model.ParseSkillType(strings.TrimSpace(t))
		if err != nil {
			return nil, err
		}
		typeFilter = append(typeFilter, skillType)
	}

	return typeFilter, nil
}

// resolveSyncTypeFilter resolves sync/delete type filtering policy:
// 1. CLI --type/--include-prompts overrides config
// 2. sync.include_types config
// 3. default: skill only
func resolveSyncTypeFilter(cmd *cli.Command) ([]model.SkillType, error) {
	typeStr := cmd.String("type")
	includePrompts := cmd.Bool("include-prompts")

	typeFilter, err := parseTypeFilter(typeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --type: %w", err)
	}

	if includePrompts {
		if len(typeFilter) == 0 {
			return []model.SkillType{model.SkillTypeSkill, model.SkillTypePrompt}, nil
		}
		if !slices.Contains(typeFilter, model.SkillTypePrompt) {
			typeFilter = append(typeFilter, model.SkillTypePrompt)
		}
		if !slices.Contains(typeFilter, model.SkillTypeSkill) {
			typeFilter = append(typeFilter, model.SkillTypeSkill)
		}
		return typeFilter, nil
	}

	if len(typeFilter) > 0 {
		return typeFilter, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config for type policy: %w", err)
	}

	if len(cfg.Sync.IncludeTypes) == 0 {
		return []model.SkillType{model.SkillTypeSkill}, nil
	}

	configured := make([]model.SkillType, 0, len(cfg.Sync.IncludeTypes))
	for _, t := range cfg.Sync.IncludeTypes {
		skillType, err := model.ParseSkillType(strings.TrimSpace(t))
		if err != nil {
			return nil, fmt.Errorf("invalid sync.include_types value %q: %w", t, err)
		}
		if !slices.Contains(configured, skillType) {
			configured = append(configured, skillType)
		}
	}

	if len(configured) == 0 {
		return []model.SkillType{model.SkillTypeSkill}, nil
	}

	return configured, nil
}

// discoverPluginSkills discovers skills from Claude Code plugins with optional caching.
// It discovers skills from:
// 1. ~/.skillsync/plugins/ - cloned plugin repositories
// 2. ~/.claude/plugins/cache/ - installed Claude Code plugins
func discoverPluginSkills(repoURL string, useCache bool) ([]model.Skill, error) {
	var pluginParser *plugin.Parser

	if repoURL != "" {
		pluginParser = plugin.NewWithRepo(repoURL)
	} else {
		pluginParser = plugin.New("")
	}

	// Try to use cache for local plugins (not for remote repos which need git pull)
	if useCache && repoURL == "" {
		skillCache, err := cache.New("plugins")
		if err == nil && skillCache.Size() > 0 && !skillCache.IsStale(cache.DefaultTTL) {
			// Return cached skills
			var skills []model.Skill
			for _, entry := range skillCache.Entries {
				skills = append(skills, entry.Skill)
			}
			return skills, nil
		}
	}

	// Parse plugins from ~/.skillsync/plugins/
	skills, err := pluginParser.Parse()
	if err != nil {
		return nil, err
	}

	// Also discover skills from Claude plugin cache (~/.claude/plugins/cache/)
	// Only do this for local discovery (not when fetching from a specific repo)
	if repoURL == "" {
		cacheSkills, err := discoverClaudePluginCacheSkills(skills)
		if err != nil {
			logging.Warn("failed to discover Claude plugin cache skills", logging.Err(err))
		} else {
			skills = append(skills, cacheSkills...)
		}
	}

	// Cache the results for local plugins
	if useCache && repoURL == "" && len(skills) > 0 {
		skillCache, err := cache.New("plugins")
		if err == nil {
			for _, skill := range skills {
				skillCache.Set(skill.Name, skill)
			}
			_ = skillCache.Save()
		}
	}

	return skills, nil
}

// discoverClaudePluginCacheSkills discovers skills from installed Claude Code plugins.
// It deduplicates against existingSkills to avoid showing the same skill twice
// (e.g., when a skill exists both as a dev symlink and in the cache).
func discoverClaudePluginCacheSkills(existingSkills []model.Skill) ([]model.Skill, error) {
	cacheParser := claude.NewCachePluginsParser("")
	cacheSkills, err := cacheParser.Parse()
	if err != nil {
		return nil, err
	}

	// Build a deduplication index from existing skills
	// Key: skill name + marketplace (to handle same skill in different marketplaces)
	seen := make(map[string]bool)
	for _, s := range existingSkills {
		key := s.Name
		if s.PluginInfo != nil && s.PluginInfo.Marketplace != "" {
			key = s.Name + "@" + s.PluginInfo.Marketplace
		} else if marketplace, ok := s.Metadata["marketplace"]; ok {
			key = s.Name + "@" + marketplace
		}
		seen[key] = true
	}

	// Filter out duplicates from cache skills
	var uniqueSkills []model.Skill
	for _, s := range cacheSkills {
		key := s.Name
		if s.PluginInfo != nil && s.PluginInfo.Marketplace != "" {
			key = s.Name + "@" + s.PluginInfo.Marketplace
		} else if marketplace, ok := s.Metadata["marketplace"]; ok {
			key = s.Name + "@" + marketplace
		}

		if !seen[key] {
			seen[key] = true
			uniqueSkills = append(uniqueSkills, s)
		}
	}

	return uniqueSkills, nil
}

// discoverSkillsInteractive runs the interactive TUI for skill discovery
func discoverSkillsInteractive(skills []model.Skill) error {
	if len(skills) == 0 {
		fmt.Println("No skills found.")
		return nil
	}

	result, err := tui.RunDiscoverList(skills)
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Handle the selected action
	switch result.Action {
	case tui.DiscoverActionView:
		fmt.Printf("\n%s\n", ui.Bold("Skill: "+result.Skill.Name))
		fmt.Printf("Platform: %s\n", result.Skill.Platform)
		fmt.Printf("Scope: %s\n", result.Skill.DisplayScope())
		fmt.Printf("Path: %s\n", result.Skill.Path)
		if result.Skill.Description != "" {
			fmt.Printf("Description: %s\n", result.Skill.Description)
		}
		if len(result.Skill.Tools) > 0 {
			fmt.Printf("Tools: %s\n", strings.Join(result.Skill.Tools, ", "))
		}
		fmt.Printf("\n%s\n", ui.Dim("--- Content ---"))
		fmt.Println(result.Skill.Content)
	case tui.DiscoverActionCopy:
		fmt.Printf("\nPath: %s\n", result.Skill.Path)
	case tui.DiscoverActionNone:
		// User quit without action
		return nil
	}

	return nil
}

// outputSkills formats and prints skills in the requested format
func outputSkills(skills []model.Skill, format string) error {
	switch format {
	case "json":
		return outputJSON(skills)
	case "yaml":
		return outputYAML(skills)
	case "table":
		return outputTable(skills)
	default:
		return fmt.Errorf("unsupported format: %s (use table, json, or yaml)", format)
	}
}

// outputJSON prints skills as JSON
func outputJSON(skills []model.Skill) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(skills)
}

// outputYAML prints skills as YAML
func outputYAML(skills []model.Skill) error {
	data, err := yaml.Marshal(skills)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	fmt.Print(string(data))
	return nil
}

// columnWidths holds the calculated widths for each table column
type columnWidths struct {
	name     int
	platform int
	source   int
	desc     int
}

// getTerminalWidth returns the current terminal width, or a default of 120 if unavailable
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd())) //nolint:gosec // fd is always a small value
	if err != nil || width <= 0 {
		return 120 // sensible default for non-TTY or error cases
	}
	return width
}

// clamp restricts a value to the range [min, max]
func clamp(value, minVal, maxVal int) int {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}

// calculateColumnWidths determines optimal column widths based on content and terminal size
func calculateColumnWidths(skills []model.Skill, termWidth int) columnWidths {
	// Find max content width for each column
	maxName, maxSource, maxDesc := 0, 0, 0
	for _, s := range skills {
		if len(s.Name) > maxName {
			maxName = len(s.Name)
		}
		if len(s.DisplayScope()) > maxSource {
			maxSource = len(s.DisplayScope())
		}
		if len(s.Description) > maxDesc {
			maxDesc = len(s.Description)
		}
	}

	// Platform is fixed at 12 (claude-code is longest at 10)
	platform := 12

	// Set bounds for name and source
	name := clamp(maxName, 15, 35)
	source := clamp(maxSource, 20, 60)

	// Allocate remaining space to description (minimum 20)
	// 6 accounts for spacing between columns (2 spaces each gap × 3 gaps)
	used := name + platform + source + 6
	desc := termWidth - used
	if desc < 20 {
		desc = 20
	}

	return columnWidths{
		name:     name,
		platform: platform,
		source:   source,
		desc:     desc,
	}
}

// outputTable prints skills in a table format with colored output
func outputTable(skills []model.Skill) error {
	if len(skills) == 0 {
		fmt.Println("No skills found.")
		return nil
	}

	// Sort skills alphabetically by name (case-insensitive)
	sort.Slice(skills, func(i, j int) bool {
		return strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
	})

	// Calculate dynamic column widths based on content and terminal size
	termWidth := getTerminalWidth()
	widths := calculateColumnWidths(skills, termWidth)

	// Print colored headers
	// SOURCE shows where skills come from: ~/.claude/skills (user), .claude/skills (repo),
	// or with plugin info: ~/.claude/skills (plugin: name@marketplace)
	fmt.Printf("%s %s %s %s\n",
		ui.Header(fmt.Sprintf("%-*s", widths.name, "NAME")),
		ui.Header(fmt.Sprintf("%-*s", widths.platform, "PLATFORM")),
		ui.Header(fmt.Sprintf("%-*s", widths.source, "SOURCE")),
		ui.Header(fmt.Sprintf("%-*s", widths.desc, "DESCRIPTION")))
	fmt.Printf("%-*s %-*s %-*s %-*s\n",
		widths.name, "----",
		widths.platform, "--------",
		widths.source, "------",
		widths.desc, "-----------")

	for _, skill := range skills {
		name := skill.Name
		if len(name) > widths.name {
			name = name[:widths.name-3] + "..."
		}

		// Sanitize description: replace newlines with spaces for table display
		desc := strings.ReplaceAll(skill.Description, "\n", " ")
		desc = strings.ReplaceAll(desc, "\r", "")
		// Collapse multiple spaces
		for strings.Contains(desc, "  ") {
			desc = strings.ReplaceAll(desc, "  ", " ")
		}
		desc = strings.TrimSpace(desc)
		if len(desc) > widths.desc {
			desc = desc[:widths.desc-3] + "..."
		}

		// Color platform names for visual distinction
		platform := colorPlatform(string(skill.Platform), widths.platform)

		// Color source for visual distinction by scope type
		source := colorSource(skill, widths.source)

		fmt.Printf("%-*s %s %s %-*s\n", widths.name, name, platform, source, widths.desc, desc)
	}

	fmt.Printf("\nTotal: %d skill(s)\n", len(skills))
	return nil
}

// colorPlatform returns a colored platform name for visual distinction
func colorPlatform(platform string, width int) string {
	// Use consistent width formatting with colors
	formatted := fmt.Sprintf("%-*s", width, platform)
	switch platform {
	case "claude-code":
		return ui.Info(formatted)
	case "cursor":
		return ui.Success(formatted)
	case "codex":
		return ui.Warning(formatted)
	case "pi.dev":
		return ui.Magenta(formatted)
	default:
		return formatted
	}
}

// colorSource returns a colored source string based on the skill's scope and plugin info.
// Colors:
//   - user (~/.xxx) = cyan
//   - repo (.xxx) = green
//   - plugin (installed) = yellow
//   - plugin (dev symlink) = magenta
//   - system/admin/builtin = dim
func colorSource(skill model.Skill, width int) string {
	source := skill.DisplayScope()
	if len(source) > width {
		source = source[:width-3] + "..."
	}
	formatted := fmt.Sprintf("%-*s", width, source)

	// Check for plugin symlinks first (more specific than scope)
	if skill.PluginInfo != nil {
		if skill.PluginInfo.IsDev {
			return ui.Magenta(formatted) // magenta for dev symlinks
		}
		return ui.Warning(formatted) // yellow for installed plugin symlinks
	}

	switch skill.Scope {
	case model.ScopeUser:
		return ui.Info(formatted) // cyan for user-level skills
	case model.ScopeRepo:
		return ui.Success(formatted) // green for repo-level skills
	case model.ScopePlugin:
		return ui.Warning(formatted) // yellow for plugin skills
	case model.ScopeSystem, model.ScopeAdmin, model.ScopeBuiltin:
		return ui.Dim(formatted) // dim for system/admin/builtin
	default:
		return formatted
	}
}
