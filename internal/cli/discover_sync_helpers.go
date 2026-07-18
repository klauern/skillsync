// Package cli provides command definitions for skillsync.
package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"
	"golang.org/x/term"

	"github.com/klauern/skillsync/internal/backup"
	"github.com/klauern/skillsync/internal/cache"
	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/plugin"
	"github.com/klauern/skillsync/internal/sync"
	"github.com/klauern/skillsync/internal/ui"
	"github.com/klauern/skillsync/internal/ui/tui"
	"github.com/klauern/skillsync/internal/util"
	"github.com/klauern/skillsync/internal/validation"
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

   Supported platforms: ` + model.AllPlatformNames() + `

   Plugin discovery: By default, skills from installed Claude Code plugins
   are included from ~/.skillsync/plugins/. Use --no-plugins to exclude them,
   or specify a Git repository with --repo to fetch plugins from.

   Output formats: table (default), json, yaml
   For interactive browsing, use: skillsync tui`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Filter by platform (" + model.AllPlatformNames() + ")",
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
				for s := range strings.SplitSeq(scopeStr, ",") {
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
	for t := range strings.SplitSeq(typeStr, ",") {
		skillType, err := model.ParseSkillType(strings.TrimSpace(t))
		if err != nil {
			return nil, fmt.Errorf("parse skill type %q: %w", strings.TrimSpace(t), err)
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
		return nil, fmt.Errorf("parse plugin skills: %w", err)
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
		return nil, fmt.Errorf("parse Claude plugin cache skills: %w", err)
	}

	// Build a deduplication index from existing skills
	// Key: skill name + marketplace (to handle same skill in different marketplaces)
	seen := make(map[string]bool)
	for _, s := range existingSkills {
		seen[pluginSkillDedupeKey(s)] = true
	}

	// Filter out duplicates from cache skills
	var uniqueSkills []model.Skill
	for _, s := range cacheSkills {
		key := pluginSkillDedupeKey(s)
		if !seen[key] {
			seen[key] = true
			uniqueSkills = append(uniqueSkills, s)
		}
	}

	return uniqueSkills, nil
}

func pluginSkillDedupeKey(skill model.Skill) string {
	if skill.PluginInfo != nil && skill.PluginInfo.Marketplace != "" {
		return skill.Name + "@" + skill.PluginInfo.Marketplace
	}
	if marketplace, ok := skill.Metadata["marketplace"]; ok && marketplace != "" {
		return skill.Name + "@" + marketplace
	}

	return skill.Name
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

// syncSkillsInteractive runs the interactive TUI for sync skill selection
func syncSkillsInteractive(cfg *syncConfig) error {
	if len(cfg.sourceSkills) == 0 {
		fmt.Println("No skills found to sync.")
		return nil
	}

	// Parse existing target skills for diff preview
	targetSkills, err := parsePlatformSkills(cfg.targetSpec.Platform)
	if err != nil {
		// Not fatal - target may not have any skills yet
		targetSkills = []model.Skill{}
	}

	// Create a map of target skills by name for quick lookup
	targetSkillMap := make(map[string]model.Skill)
	for _, s := range targetSkills {
		targetSkillMap[s.Name] = s
	}

	// Main TUI loop - allows navigating between list and diff preview
	for {
		result, err := runSyncList(cfg.sourceSkills, cfg.sourceSpec.Platform, cfg.targetSpec.Platform, nil)
		if err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		switch result.Action {
		case tui.SyncActionNone:
			// User quit without action
			fmt.Println("Sync cancelled.")
			return nil

		case tui.SyncActionPreview:
			// Show diff preview for selected skill
			var targetSkill *model.Skill
			if ts, exists := targetSkillMap[result.PreviewSkill.Name]; exists {
				targetSkill = &ts
			}

			diffResult, err := runSyncDiff(result.PreviewSkill, targetSkill, cfg.sourceSpec.Platform, cfg.targetSpec.Platform)
			if err != nil {
				return fmt.Errorf("diff preview error: %w", err)
			}

			switch diffResult.Action {
			case tui.DiffActionBack:
				// Continue the loop to go back to the list
				continue
			case tui.DiffActionSync:
				// Sync just this one skill
				if err := executeSyncForSkills(cfg, []model.Skill{diffResult.Skill}, len(cfg.sourceSkills)); err != nil {
					return fmt.Errorf("execute sync for selected skill: %w", err)
				}
				return nil
			case tui.DiffActionNone:
				// User quit
				fmt.Println("Sync cancelled.")
				return nil
			}

		case tui.SyncActionSync:
			// Sync selected skills
			if len(result.SelectedSkills) == 0 {
				fmt.Println("No skills selected.")
				return nil
			}

			if err := executeSyncForSkills(cfg, result.SelectedSkills, len(cfg.sourceSkills)); err != nil {
				return fmt.Errorf("execute sync for selected skills: %w", err)
			}
			return nil
		}
	}
}

// executeSyncForSkills performs the actual sync operation for the given skills
func executeSyncForSkills(cfg *syncConfig, skills []model.Skill, totalAvailable int) error {
	if err := runSyncBackup(cfg); err != nil {
		return fmt.Errorf("prepare sync backup: %w", err)
	}

	opts := sync.Options{
		DryRun:      cfg.dryRun,
		Strategy:    cfg.strategy,
		TargetScope: cfg.targetSpec.TargetScope(),
	}

	syncer := sync.New()
	result, err := syncer.SyncWithSkills(skills, cfg.targetSpec.Platform, opts)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}
	ensureSyncSource(result, cfg.sourceSpec.Platform)
	if totalAvailable <= 0 {
		totalAvailable = len(skills)
	}
	result.SelectedCount = len(skills)
	result.TotalAvailable = totalAvailable

	if cfg.format == "json" {
		return outputSyncResultJSON(result)
	}
	displaySyncResults(result)

	if !result.Success() {
		return summarizeSyncFailures(result, "sync completed with errors")
	}

	return nil
}

// outputSyncResultJSON prints a sync result as JSON, including portability warnings per skill.
func outputSyncResultJSON(result *sync.Result) error {
	type skillJSON struct {
		Name                string   `json:"name"`
		Action              string   `json:"action"`
		TargetPath          string   `json:"target_path,omitempty"`
		Message             string   `json:"message,omitempty"`
		Error               string   `json:"error,omitempty"`
		PortabilityWarnings []string `json:"portability_warnings,omitempty"`
	}
	type resultJSON struct {
		Source   string      `json:"source"`
		Target   string      `json:"target"`
		Strategy string      `json:"strategy"`
		DryRun   bool        `json:"dry_run"`
		Skills   []skillJSON `json:"skills"`
	}

	skills := make([]skillJSON, 0, len(result.Skills))
	for _, sr := range result.Skills {
		sj := skillJSON{
			Name:                sr.Skill.Name,
			Action:              string(sr.Action),
			TargetPath:          sr.TargetPath,
			Message:             sr.Message,
			PortabilityWarnings: sr.PortabilityWarnings,
		}
		if sr.Error != nil {
			sj.Error = sr.Error.Error()
		}
		skills = append(skills, sj)
	}

	out := resultJSON{
		Source:   string(result.Source),
		Target:   string(result.Target),
		Strategy: string(result.Strategy),
		DryRun:   result.DryRun,
		Skills:   skills,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
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
	return outputAnyJSON(skills)
}

// outputYAML prints skills as YAML
func outputYAML(skills []model.Skill) error {
	return outputAnyYAML(skills)
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
	desc := max(termWidth-used, 20)

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

// platformColorFns maps each platform to its display color function.
var platformColorFns = map[model.Platform]func(...any) string{
	model.ClaudeCode: ui.Info,
	model.Cursor:     ui.Success,
	model.Codex:      ui.Warning,
	model.PiDev:      ui.Magenta,
	model.Copilot:    ui.Blue,
	model.Gemini:     ui.Bold,
}

// colorPlatform returns a colored platform name for visual distinction.
func colorPlatform(platform string, width int) string {
	formatted := fmt.Sprintf("%-*s", width, platform)
	if fn, ok := platformColorFns[model.Platform(platform)]; ok {
		return fn(formatted)
	}
	return formatted
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

func syncFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "dry-run",
			Aliases: []string{"d"},
			Usage:   "Preview changes without modifying files",
		},
		&cli.StringFlag{
			Name:    "strategy",
			Aliases: []string{"s"},
			Value:   "overwrite",
			Usage:   "Conflict resolution strategy: overwrite, skip, newer, merge, three-way, interactive",
		},
		&cli.BoolFlag{
			Name:  "skip-backup",
			Usage: "Skip automatic backup before sync",
		},
		&cli.BoolFlag{
			Name:  "skip-validation",
			Usage: "Skip validation checks (not recommended)",
		},
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "Skip confirmation prompts (use with caution)",
		},
		&cli.BoolFlag{
			Name:  "include-plugins",
			Usage: "Include skills from Claude Code plugins (excluded by default)",
		},
		&cli.StringFlag{
			Name:    "type",
			Aliases: []string{"t"},
			Usage:   "Artifact types to sync/delete: skill, prompt, all. Comma-separated for multiple.",
		},
		&cli.BoolFlag{
			Name:  "include-prompts",
			Usage: "Include prompt/command artifacts (equivalent to --type skill,prompt)",
		},
		&cli.BoolFlag{
			Name:  "delete",
			Usage: "After sync, delete target skills not present in source (orphan cleanup)",
		},
		&cli.StringFlag{
			Name:  "format",
			Value: "text",
			Usage: "Output format: text, json",
		},
	}
}

func syncCommand() *cli.Command {
	return &cli.Command{
		Name:      "sync",
		Usage:     "Synchronize skills across platforms",
		UsageText: "skillsync sync [options] <source> <target>",
		Description: `Synchronize skills between AI coding platforms.

   Supported platforms: ` + model.AllPlatformNames() + `

   Platform spec format: platform[:scope[,scope2,...]]
     - cursor           All scopes from cursor (source), user scope (target)
     - cursor:repo      Only repo scope
     - cursor:repo,user Both repo and user scopes (source only)

   Valid source scopes: repo, user, admin, system, builtin, plugin
   Valid target scopes: repo, user (writable locations only)

   Plugin Skills:
     Plugin scope skills (from Claude Code installed plugins) are excluded
     by default. To include them, either:
     - Use --include-plugins flag
     - Explicitly specify plugin scope: claudecode:plugin

   Artifact Types:
     By default, sync only includes skill artifacts.
     Use --include-prompts or --type prompt (or skill,prompt) to include
     command/prompt artifacts.

   Strategies:
     overwrite   - Replace target skills unconditionally (default)
     skip        - Skip skills that already exist in target
     newer       - Copy only if source is newer than target
     merge       - Merge source and target content
     three-way   - Intelligent merge with conflict detection
     interactive - Prompt for each conflict

   Delete Mode:
     Use --delete to remove skills from target that exist in source.
     This is the inverse of sync: instead of copying TO target, it removes
     skills FROM target that match the source skill names.
     Use --interactive --delete to select which matching skills to remove.


   Examples:
     skillsync sync cursor claudecode             # All cursor skills to claudecode user scope
     skillsync sync cursor:repo claudecode:user   # Repo skills to user scope
     skillsync sync cursor:repo,user codex:repo   # Multiple source scopes to repo
     skillsync tui                                # Interactive dashboard mode
     skillsync sync --dry-run cursor codex        # Preview changes
     skillsync sync --strategy=skip cursor codex
     skillsync sync --interactive cursor codex    # Interactive TUI mode
     skillsync sync --interactive --delete cursor codex  # Select which skills to delete
     skillsync sync --include-plugins claudecode cursor  # Include plugin skills
     skillsync sync claudecode:plugin cursor      # Sync only plugin skills
     skillsync sync --include-prompts claudecode codex   # Include prompts/commands
     skillsync sync --type prompt claudecode codex       # Prompts only
     skillsync sync --delete cursor claudecode          # Sync and remove orphaned skills from target

   See also:
     skillsync delete <source> <target>           # Remove skills from target`,
		Flags: append(syncFlags(), &cli.BoolFlag{
			Name:    "interactive",
			Aliases: []string{"i"},
			Usage:   "Interactive TUI mode with skill selection and diff preview (or delete selection with --delete)",
		}),
		Action: func(_ context.Context, cmd *cli.Command) error {
			if !cmd.Bool("interactive") {
				return runSyncCommand(cmd, false)
			}

			deleteMode := cmd.Bool("delete")
			cfg, err := parseSyncConfig(cmd, cmd.Name, deleteMode)
			if err != nil {
				return fmt.Errorf("parse sync config: %w", err)
			}

			if cfg.sourceSpec.HasScopes() {
				cfg.sourceSkills, err = parsePlatformSkillsWithScope(cfg.sourceSpec.Platform, cfg.sourceSpec.Scopes, cfg.includePlugins)
			} else {
				cfg.sourceSkills, err = parsePlatformSkillsWithScope(cfg.sourceSpec.Platform, nil, cfg.includePlugins)
			}
			if err != nil {
				return fmt.Errorf("failed to parse source skills: %w", err)
			}

			if deleteMode {
				return syncDeleteInteractive(cfg)
			}
			return syncSkillsInteractive(cfg)
		},
	}
}

func deleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete skills from target that exist in source",
		UsageText: "skillsync delete [options] <source> <target>",
		Description: `Delete skills from the target platform that also exist in the source.

   Supported platforms: ` + model.AllPlatformNames() + `

   Platform spec format: platform[:scope[,scope2,...]]
     - cursor           All scopes from cursor (source), user scope (target)
     - cursor:repo      Only repo scope
     - cursor:repo,user Both repo and user scopes (source only)

   Valid source scopes: repo, user, admin, system, builtin, plugin
   Valid target scopes: repo, user (writable locations only)

   Plugin Skills:
     Plugin scope skills (from Claude Code installed plugins) are excluded
     by default. To include them, either:
     - Use --include-plugins flag
     - Explicitly specify plugin scope: claudecode:plugin

   Artifact Types:
     By default, delete only includes skill artifacts.
     Use --include-prompts or --type prompt (or skill,prompt) to include
     command/prompt artifacts.

   Flags:
     Delete supports the same flags as sync (including --dry-run).

   Examples:
     skillsync delete cursor claudecode           # Remove cursor skills from claudecode
     skillsync delete cursor:repo claudecode:user # Remove repo skills from user scope
     skillsync tui                                # Interactive dashboard mode
     skillsync delete --dry-run cursor codex      # Preview changes
     skillsync delete --include-plugins claudecode cursor`,
		Flags: syncFlags(),
		Action: func(_ context.Context, cmd *cli.Command) error {
			return runSyncCommand(cmd, true)
		},
	}
}

func runSyncCommand(cmd *cli.Command, deleteMode bool) error {
	cfg, err := parseSyncConfig(cmd, cmd.Name, deleteMode)
	if err != nil {
		return fmt.Errorf("parse sync config: %w", err)
	}

	// Always parse source skills (use tiered parser for scope filtering)
	// Plugin scope skills are excluded by default unless --include-plugins is set
	// or the plugin scope is explicitly in the source spec (e.g., "claudecode:plugin")
	if cfg.sourceSpec.HasScopes() {
		// User specified scopes - use tiered parser for scope filtering
		cfg.sourceSkills, err = parsePlatformSkillsWithScope(cfg.sourceSpec.Platform, cfg.sourceSpec.Scopes, cfg.includePlugins)
	} else {
		// No scopes specified - use tiered parser with plugin exclusion by default
		cfg.sourceSkills, err = parsePlatformSkillsWithScope(cfg.sourceSpec.Platform, nil, cfg.includePlugins)
	}
	if err != nil {
		return fmt.Errorf("failed to parse source skills: %w", err)
	}

	// Apply artifact type filter policy for sync/delete commands.
	cfg.sourceSkills = filterBySkillType(cfg.sourceSkills, cfg.typeFilter)

	// Delete mode has different flow
	if cfg.deleteMode {
		return syncDeleteMode(cfg)
	}

	// Validate source skills before sync (unless skipped)
	if !cfg.skipValidation {
		if err := validateSourceSkills(cfg); err != nil {
			return fmt.Errorf("validate source skills: %w", err)
		}
	}

	// Show summary and request confirmation (unless --yes or --dry-run)
	if !cfg.dryRun && !cfg.yesFlag {
		confirmed, err := showSyncSummaryAndConfirm(cfg)
		if err != nil {
			return fmt.Errorf("confirmation error: %w", err)
		}
		if !confirmed {
			fmt.Println("Sync cancelled by user")
			return nil
		}
	}

	// Create backup before sync (unless skipped or dry-run)
	if err := runSyncBackup(cfg); err != nil {
		return fmt.Errorf("prepare sync backup: %w", err)
	}

	// Create sync options and execute
	opts := sync.Options{
		DryRun:      cfg.dryRun,
		Strategy:    cfg.strategy,
		TargetScope: cfg.targetSpec.TargetScope(),
	}

	syncer := sync.New()
	result, err := syncer.SyncWithSkills(cfg.sourceSkills, cfg.targetSpec.Platform, opts)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Handle conflicts if interactive strategy is used
	if err := runSyncConflictResolution(cfg, result); err != nil {
		return fmt.Errorf("run sync conflict resolution: %w", err)
	}

	displaySyncResults(result)

	// Post-sync orphan deletion (--delete flag)
	if err := runSyncOrphanDeletion(cfg); err != nil {
		return fmt.Errorf("run sync orphan deletion: %w", err)
	}

	if !result.Success() {
		return summarizeSyncFailures(result, "sync completed with errors")
	}

	return nil
}

// runSyncBackup creates a backup before sync unless skipped or dry-run.
func runSyncBackup(cfg *syncConfig) error {
	if cfg.dryRun || cfg.skipBackup {
		return nil
	}
	prepareBackup(cfg.targetSpec.Platform)
	created, err := backupExistingTargetSkills(
		cfg.targetSpec.Platform,
		cfg.targetSpec.TargetScope(),
		cfg.sourceSkills,
		"pre-sync backup",
		[]string{"sync"},
	)
	if err != nil {
		return fmt.Errorf("create backups for %s: %w", cfg.targetSpec.Platform, err)
	}
	if created > 0 {
		fmt.Printf("✓ Created %d backup(s)\n", created)
	}
	return nil
}

// runSyncConflictResolution handles interactive conflict resolution after sync.
func runSyncConflictResolution(cfg *syncConfig, result *sync.Result) error {
	if !result.HasConflicts() || cfg.strategy != sync.StrategyInteractive {
		return nil
	}

	resolver := NewConflictResolver()

	// Gather conflicts
	var conflicts []*sync.Conflict
	for _, sr := range result.Conflicts() {
		if sr.Conflict != nil {
			conflicts = append(conflicts, sr.Conflict)
		}
	}

	// Display summary and resolve
	resolver.DisplayConflictSummary(conflicts)
	resolved, err := resolver.ResolveConflicts(conflicts)
	if err != nil {
		return fmt.Errorf("conflict resolution failed: %w", err)
	}

	// Apply resolved content
	if !cfg.dryRun {
		if err := applyResolvedConflicts(result, resolved); err != nil {
			return fmt.Errorf("failed to apply resolved conflicts: %w", err)
		}
	}

	fmt.Printf("\nResolved %d conflict(s)\n", len(resolved))
	return nil
}

// runSyncOrphanDeletion handles post-sync orphan deletion when --delete flag is set.
func runSyncOrphanDeletion(cfg *syncConfig) error {
	if !cfg.deleteOrphans || cfg.deleteMode {
		return nil
	}

	targetSkills, err := parsePlatformSkillsWithScope(
		cfg.targetSpec.Platform,
		[]model.SkillScope{cfg.targetSpec.TargetScope()},
		cfg.includePlugins,
	)
	if err != nil {
		return fmt.Errorf("failed to parse target skills for orphan detection: %w", err)
	}

	orphans := findOrphanedSkills(cfg.sourceSkills, targetSkills)
	if len(orphans) == 0 {
		fmt.Println("\nNo orphaned skills found in target")
		return nil
	}

	fmt.Printf("\nFound %d orphaned skill(s) in target (not in source):\n", len(orphans))
	for _, s := range orphans {
		fmt.Printf("  - %s\n", s.Name)
	}

	if cfg.dryRun {
		fmt.Println("\n(dry-run) Would delete the above orphaned skills")
		return nil
	}

	deleteCfg := &syncConfig{
		sourceSpec:     cfg.sourceSpec,
		targetSpec:     cfg.targetSpec,
		dryRun:         cfg.dryRun,
		skipBackup:     cfg.skipBackup,
		yesFlag:        cfg.yesFlag,
		deleteMode:     true,
		includePlugins: cfg.includePlugins,
		sourceSkills:   orphans,
	}
	if err := executeDeleteWithConfirmation(deleteCfg, orphans, false); err != nil {
		return fmt.Errorf("orphan deletion failed: %w", err)
	}
	return nil
}

// syncConfig holds the parsed configuration for a sync command
type syncConfig struct {
	sourceSpec     model.PlatformSpec
	targetSpec     model.PlatformSpec
	dryRun         bool
	strategy       sync.Strategy
	skipBackup     bool
	skipValidation bool
	yesFlag        bool
	deleteMode     bool
	deleteOrphans  bool
	includePlugins bool
	typeFilter     []model.SkillType
	sourceSkills   []model.Skill
	format         string
}

// parseSyncConfig parses and validates sync command arguments and flags
func parseSyncConfig(cmd *cli.Command, commandName string, deleteMode bool) (*syncConfig, error) {
	args := cmd.Args()
	if args.Len() != 2 {
		return nil, fmt.Errorf("%s requires exactly 2 arguments: <source> <target>", commandName)
	}

	// Parse source platform spec (e.g., "cursor", "cursor:repo", "cursor:repo,user")
	sourceSpec, err := model.ParsePlatformSpec(args.Get(0))
	if err != nil {
		return nil, fmt.Errorf("invalid source: %w", err)
	}

	// Parse target platform spec (e.g., "claudecode", "claudecode:user")
	targetSpec, err := model.ParsePlatformSpec(args.Get(1))
	if err != nil {
		return nil, fmt.Errorf("invalid target: %w", err)
	}

	// Validate target spec (only single scope, only repo/user allowed)
	if err := targetSpec.ValidateAsTarget(); err != nil {
		return nil, fmt.Errorf("invalid target: %w", err)
	}

	sourceSpec = sourceSpec.NormalizeSource()

	if sourceSpec.Platform == targetSpec.Platform {
		return nil, fmt.Errorf("source and target platforms cannot be the same: %s", sourceSpec.Platform)
	}

	typeFilter, err := resolveSyncTypeFilter(cmd)
	if err != nil {
		return nil, fmt.Errorf("resolve sync type filter: %w", err)
	}

	strategyStr := cmd.String("strategy")
	strategy := sync.Strategy(strategyStr)
	if !strategy.IsValid() {
		return nil, fmt.Errorf("invalid strategy %q (valid: overwrite, skip, newer, merge, three-way, interactive)", strategyStr)
	}

	return &syncConfig{
		sourceSpec:     sourceSpec,
		targetSpec:     targetSpec,
		dryRun:         cmd.Bool("dry-run"),
		strategy:       strategy,
		skipBackup:     cmd.Bool("skip-backup"),
		skipValidation: cmd.Bool("skip-validation"),
		yesFlag:        cmd.Bool("yes"),
		deleteMode:     deleteMode,
		deleteOrphans:  cmd.Bool("delete"),
		includePlugins: cmd.Bool("include-plugins"),
		typeFilter:     typeFilter,
		sourceSkills:   make([]model.Skill, 0),
		format:         cmd.String("format"),
	}, nil
}

// validateSourceSkills validates source skills (assumes skills are already parsed in cfg.sourceSkills)
func validateSourceSkills(cfg *syncConfig) error {
	fmt.Println("Validating source skills...")

	// Validate skill formats
	formatResult, err := validation.ValidateSkillsFormat(cfg.sourceSkills, cfg.sourceSpec.Platform)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Show warnings
	for _, warning := range formatResult.Warnings {
		fmt.Printf("  Warning: %s\n", warning)
	}

	// Check for validation errors
	if formatResult.HasErrors() {
		fmt.Println("\nValidation failed - the following issues were found:")
		for i, e := range formatResult.Errors {
			fmt.Printf("  %d. %s\n", i+1, formatValidationError(e, cfg.sourceSkills))
		}
		return errors.New("skill validation failed - fix the issues above and try again")
	}

	if len(cfg.sourceSkills) == 0 {
		fmt.Println("  No skills found in source directory")
	} else {
		fmt.Printf("  Found %d valid skill(s)\n", len(cfg.sourceSkills))
	}

	// Validate target path and permissions
	// Note: Skip source path validation since skills were already successfully parsed
	// from potentially multiple scopes (project, user, admin, system). The primary
	// platform path may not exist, but that's fine if other scopes have skills.
	if err := validateTargetPath(cfg.targetSpec.Platform); err != nil {
		return fmt.Errorf("validate target path: %w", err)
	}

	fmt.Println("Validation passed")
	return nil
}

// showSyncSummaryAndConfirm shows sync summary and requests user confirmation
func showSyncSummaryAndConfirm(cfg *syncConfig) (bool, error) {
	fmt.Printf("\n=== Sync Summary ===\n")
	fmt.Printf("Source: %s\n", cfg.sourceSpec)
	fmt.Printf("Target: %s\n", cfg.targetSpec)
	fmt.Printf("Strategy: %s (%s)\n", cfg.strategy, cfg.strategy.Description())
	if len(cfg.typeFilter) > 0 {
		typeNames := make([]string, 0, len(cfg.typeFilter))
		for _, t := range cfg.typeFilter {
			typeNames = append(typeNames, t.String())
		}
		fmt.Printf("Types: %s\n", strings.Join(typeNames, ", "))
	}

	if len(cfg.sourceSkills) > 0 {
		fmt.Printf("Skills to sync: %d\n", len(cfg.sourceSkills))
		for i, skill := range cfg.sourceSkills {
			scopeStr := string(skill.Scope)
			if scopeStr == "" {
				scopeStr = "-"
			}
			fmt.Printf("  %d. %s [%s]\n", i+1, skill.Name, scopeStr)
		}
	}

	if cfg.skipBackup {
		fmt.Println("Warning: Backup will be skipped (--skip-backup flag)")
	}

	// Determine risk level
	level := riskLevelInfo
	if cfg.skipBackup || cfg.skipValidation {
		level = riskLevelWarning
	}

	return confirmAction("Proceed with sync?", level)
}

// prepareBackup runs backup cleanup before sync
func prepareBackup(targetPlatform model.Platform) {
	fmt.Println("\nPreparing backups...")

	// Run automatic cleanup to maintain retention policy
	cleanupOpts := backup.DefaultCleanupOptions()
	cleanupOpts.Platform = string(targetPlatform)

	deleted, err := backup.CleanupBackups(cleanupOpts)
	if err != nil {
		fmt.Printf("Warning: backup cleanup failed: %v\n", err)
	} else if len(deleted) > 0 {
		fmt.Printf("Cleaned up %d old backup(s)\n", len(deleted))
	}

	fmt.Println("Backup cleanup complete")
}

func createBackupsForSkills(platform model.Platform, skills []model.Skill, description string, tags []string) (int, error) {
	created := 0
	for _, skill := range skills {
		if skill.Path == "" {
			continue
		}

		metadata := map[string]string{
			"skill": skill.Name,
		}
		if skill.Scope != "" {
			metadata["scope"] = string(skill.Scope)
		}

		opts := backup.Options{
			Platform:    string(platform),
			Description: description,
			Metadata:    metadata,
			Tags:        tags,
		}

		if _, err := backup.CreateBackup(skill.Path, opts); err != nil {
			return created, fmt.Errorf("failed to back up %q: %w", skill.Path, err)
		}
		created++
	}

	return created, nil
}

func backupExistingTargetSkills(
	targetPlatform model.Platform,
	targetScope model.SkillScope,
	sourceSkills []model.Skill,
	description string,
	tags []string,
) (int, error) {
	if len(sourceSkills) == 0 {
		return 0, nil
	}

	targetSkills, err := parsePlatformSkillsWithScope(targetPlatform, []model.SkillScope{targetScope}, false)
	if err != nil {
		return 0, fmt.Errorf("failed to parse target skills for backup: %w", err)
	}

	targetByName := make(map[string]model.Skill)
	for _, skill := range targetSkills {
		if skill.Name != "" {
			targetByName[skill.Name] = skill
		}
	}
	if len(targetByName) == 0 {
		return 0, nil
	}

	toBackup := make([]model.Skill, 0, len(sourceSkills))
	for _, skill := range sourceSkills {
		if targetSkill, ok := targetByName[skill.Name]; ok {
			toBackup = append(toBackup, targetSkill)
		}
	}

	if len(toBackup) == 0 {
		return 0, nil
	}

	return createBackupsForSkills(targetPlatform, toBackup, description, tags)
}

func ensureSyncSource(result *sync.Result, sourcePlatform model.Platform) {
	if result == nil {
		return
	}
	if result.Source == "" {
		result.Source = sourcePlatform
	}
}

// displaySyncResults shows the results of a sync operation
func displaySyncResults(result *sync.Result) {
	fmt.Println()
	fmt.Print(result.Summary())

	if len(result.Skills) > 0 {
		fmt.Println("\nDetails:")
		for _, sr := range result.Skills {
			var status string
			switch sr.Action {
			case sync.ActionFailed:
				status = "✗"
			case sync.ActionSkipped:
				status = "-"
			default:
				status = "✓"
			}
			fmt.Printf("  %s %s: %s", status, sr.Skill.Name, sr.Action)
			if sr.Message != "" {
				fmt.Printf(" (%s)", sr.Message)
			}
			if sr.Error != nil {
				fmt.Printf(" - Error: %v", sr.Error)
			}
			fmt.Println()
			if len(sr.PortabilityWarnings) > 0 {
				fmt.Printf("    ⚠ lossy fields for %s: %s\n",
					result.Target, strings.Join(sr.PortabilityWarnings, ", "))
			}
		}
	}
}

// syncDeleteInteractive runs the interactive TUI for delete mode with selection.
func syncDeleteInteractive(cfg *syncConfig) error {
	if len(cfg.sourceSkills) == 0 {
		fmt.Println("No skills found to delete.")
		return nil
	}

	// Parse target skills with scope filtering to build delete candidates.
	targetScope := cfg.targetSpec.TargetScope()
	targetSkills, err := parsePlatformSkillsWithScope(cfg.targetSpec.Platform, []model.SkillScope{targetScope}, false)
	if err != nil {
		return fmt.Errorf("failed to parse target skills: %w", err)
	}

	sourceByName := make(map[string]model.Skill)
	for _, skill := range cfg.sourceSkills {
		sourceByName[skill.Name] = skill
	}

	var candidates []model.Skill
	for _, skill := range targetSkills {
		if _, exists := sourceByName[skill.Name]; exists {
			candidates = append(candidates, skill)
		}
	}

	if len(candidates) == 0 {
		fmt.Println("No matching target skills found to delete.")
		return nil
	}

	result, err := runDeleteList(candidates)
	if err != nil {
		return fmt.Errorf("delete selection error: %w", err)
	}

	switch result.Action {
	case tui.DeleteActionNone:
		fmt.Println("Delete cancelled.")
		return nil
	case tui.DeleteActionDelete:
		if len(result.SelectedSkills) == 0 {
			fmt.Println("No skills selected for deletion.")
			return nil
		}

		selectedSources := selectSourceSkillsForDelete(cfg.sourceSkills, result.SelectedSkills)

		if len(selectedSources) == 0 {
			fmt.Println("No matching source skills selected for deletion.")
			return nil
		}

		return executeDeleteForSkills(cfg, selectedSources, len(candidates))
	}

	return nil
}

// syncDeleteMode handles the delete sync mode: removing skills from target that exist in source.
func syncDeleteMode(cfg *syncConfig) error {
	return executeDeleteWithConfirmation(cfg, cfg.sourceSkills, false)
}

func executeDeleteWithConfirmation(cfg *syncConfig, skills []model.Skill, confirmed bool) error {
	if len(skills) == 0 {
		fmt.Println("No skills selected.")
		return nil
	}

	// Build list of skill names to delete
	skillNames := make([]string, len(skills))
	for i, skill := range skills {
		skillNames[i] = skill.Name
	}

	// Show what will be deleted
	fmt.Printf("Delete mode: Will remove %d skill(s) from %s that exist in %s\n",
		len(skills), cfg.targetSpec.Platform, cfg.sourceSpec.Platform)
	fmt.Println("\nSkills to delete:")
	for _, name := range skillNames {
		fmt.Printf("  - %s\n", name)
	}

	// Request confirmation (unless already confirmed, --yes, or --dry-run)
	if !confirmed && !cfg.dryRun && !cfg.yesFlag {
		confirmed, err := confirmAction(
			fmt.Sprintf("Delete %d skill(s) from %s?", len(skills), cfg.targetSpec.Platform),
			riskLevelDangerous,
		)
		if err != nil {
			return fmt.Errorf("confirmation error: %w", err)
		}
		if !confirmed {
			fmt.Println("Delete cancelled by user")
			return nil
		}
	}

	return executeDeleteForSkills(cfg, skills, len(skills))
}

func executeDeleteForSkills(cfg *syncConfig, skills []model.Skill, totalAvailable int) error {
	if len(skills) == 0 {
		fmt.Println("No skills selected for deletion.")
		return nil
	}

	// Create backup before deletion (unless skipped or dry-run)
	if !cfg.dryRun && !cfg.skipBackup {
		prepareBackup(cfg.targetSpec.Platform)
		created, err := backupExistingTargetSkills(
			cfg.targetSpec.Platform,
			cfg.targetSpec.TargetScope(),
			skills,
			"pre-delete backup",
			[]string{"delete"},
		)
		if err != nil {
			return fmt.Errorf("parse sync config: %w", err)
		}
		if created > 0 {
			fmt.Printf("✓ Created %d backup(s)\n", created)
		}
	}

	// Create options and execute delete
	opts := sync.Options{
		DryRun:      cfg.dryRun,
		TargetScope: cfg.targetSpec.TargetScope(),
		DeleteMode:  true,
	}

	syncer := sync.New()
	result, err := syncer.DeleteWithSkills(skills, cfg.targetSpec.Platform, opts)
	if err != nil {
		return fmt.Errorf("delete sync failed: %w", err)
	}
	ensureSyncSource(result, cfg.sourceSpec.Platform)
	if totalAvailable <= 0 {
		totalAvailable = len(skills)
	}
	result.SelectedCount = len(skills)
	result.TotalAvailable = totalAvailable

	displaySyncResults(result)

	if !result.Success() {
		return summarizeSyncFailures(result, "delete sync completed with errors")
	}

	return nil
}

func summarizeSyncFailures(result *sync.Result, summary string) error {
	failed := result.Failed()
	if len(failed) == 0 {
		return errors.New(summary)
	}

	errs := make([]error, 0, len(failed))
	for _, sr := range failed {
		if sr.Error != nil {
			errs = append(errs, fmt.Errorf("%s: %w", sr.Skill.Name, sr.Error))
			continue
		}
		errs = append(errs, fmt.Errorf("%s: operation failed", sr.Skill.Name))
	}

	return fmt.Errorf("%s: %w", summary, errors.Join(errs...))
}

// applyResolvedConflicts writes the resolved conflict content to the target files.
func applyResolvedConflicts(result *sync.Result, resolved map[string]string) error {
	for i := range result.Skills {
		sr := &result.Skills[i]
		if sr.Action == sync.ActionConflict {
			if content, ok := resolved[sr.Skill.Name]; ok {
				// #nosec G301 G306 - skill files should be readable.
				if err := util.WriteFileWithPerms(sr.TargetPath, []byte(content), 0o750, 0o644); err != nil {
					return fmt.Errorf("failed to write resolved content for %s: %w", sr.Skill.Name, err)
				}
				// Update the action to indicate it was resolved
				sr.Action = sync.ActionMerged
				sr.Message = "conflict resolved by user"
			}
		}
	}
	return nil
}

func parseScopeFilter(scopeStr string) ([]model.SkillScope, error) {
	if scopeStr == "" || scopeStr == "all" {
		return nil, nil
	}

	var scopeFilter []model.SkillScope
	for s := range strings.SplitSeq(scopeStr, ",") {
		scope, err := model.ParseScope(strings.TrimSpace(s))
		if err != nil {
			return nil, fmt.Errorf("invalid scope: %w", err)
		}
		scopeFilter = append(scopeFilter, scope)
	}

	if len(scopeFilter) == 0 {
		return nil, fmt.Errorf("no valid scopes found in %q", scopeStr)
	}

	return scopeFilter, nil
}

// parsePlatformSkills parses skills for a platform using env-var-respecting paths.
// This is used by sync commands when no specific scopes are requested.
func parsePlatformSkills(platform model.Platform) ([]model.Skill, error) {
	return parsePlatformSkillsWithScope(platform, nil, false)
}

// formatValidationError formats a validation error for display with context
func formatValidationError(err error, skills []model.Skill) string {
	var vErr *validation.Error
	if errors.As(err, &vErr) {
		msg := vErr.Message
		// Add helpful suggestions for common errors
		switch {
		case vErr.Field == "skills[0].name" || msg == "skill name cannot be empty":
			msg += " (ensure each skill file has a name in frontmatter or a valid filename)"
		case strings.Contains(msg, "duplicate skill name"):
			msg += " (rename one of the conflicting skills)"
		case strings.Contains(msg, "cannot access skill file"):
			msg += " (check file path and permissions)"
		}
		return fmt.Sprintf("%s: %s", vErr.Field, msg)
	}

	// Handle Errors collection
	var vErrors validation.Errors
	if errors.As(err, &vErrors) {
		var msgs []string
		for _, e := range vErrors {
			msgs = append(msgs, formatValidationError(e, skills))
		}
		return strings.Join(msgs, "; ")
	}

	return err.Error()
}

// validateTargetPath validates the target path before sync
func validateTargetPath(targetPlatform model.Platform) error {
	// Validate target path (or parent if it doesn't exist)
	targetPath, err := validation.GetPlatformPath(targetPlatform)
	if err != nil {
		return fmt.Errorf("target path error: %w", err)
	}

	if err := validation.ValidatePath(targetPath, targetPlatform); err != nil {
		var vErr *validation.Error
		if errors.As(err, &vErr) && strings.Contains(vErr.Message, "path does not exist") {
			// Target doesn't exist - validate nearest existing parent is writable
			parentDir, err := nearestExistingDir(filepath.Dir(targetPath))
			if err != nil {
				return fmt.Errorf("target parent directory validation failed: %w", err)
			}
			if err := validation.CheckWritePermission(parentDir); err != nil {
				return fmt.Errorf("target directory not writable: %w", err)
			}
		} else {
			return fmt.Errorf("target validation failed: %w", err)
		}
	}

	return nil
}

// nearestExistingDir walks up from the provided path until it finds an existing directory.
func nearestExistingDir(path string) (string, error) {
	cur := path
	for {
		info, err := os.Stat(cur)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("path is not a directory: %s", cur)
			}
			return cur, nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat %q: %w", cur, err)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("no existing parent directory found for %q", path)
		}
		cur = parent
	}
}

// riskLevel defines the severity level for confirmation prompts
type riskLevel int

const (
	riskLevelInfo      riskLevel = iota // Informational, low risk
	riskLevelWarning                    // Warning, moderate risk
	riskLevelDangerous                  // Dangerous, high risk
)

// confirmAction prompts the user for confirmation before proceeding with an action
func confirmAction(message string, level riskLevel) (bool, error) {
	// Build prompt based on risk level
	var prompt string
	var defaultYes bool

	switch level {
	case riskLevelInfo:
		prompt = fmt.Sprintf("%s [Y/n]", message)
		defaultYes = true
	case riskLevelWarning:
		prompt = fmt.Sprintf("%s [y/N]", message)
		defaultYes = false
	case riskLevelDangerous:
		prompt = fmt.Sprintf("⚠️  %s [y/N] (This operation cannot be undone)", message)
		defaultYes = false
	default:
		prompt = fmt.Sprintf("%s [y/N]", message)
		defaultYes = false
	}

	fmt.Printf("\n%s ", prompt)

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	// Trim whitespace and convert to lowercase
	response = strings.TrimSpace(strings.ToLower(response))

	// Handle empty input (use default)
	if response == "" {
		return defaultYes, nil
	}

	// Parse response
	return response == "y" || response == "yes", nil
}
