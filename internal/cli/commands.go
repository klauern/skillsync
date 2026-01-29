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
	"sort"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/backup"
	"github.com/klauern/skillsync/internal/cache"
	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/detector"
	"github.com/klauern/skillsync/internal/export"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/codex"
	"github.com/klauern/skillsync/internal/parser/cursor"
	"github.com/klauern/skillsync/internal/parser/plugin"
	"github.com/klauern/skillsync/internal/parser/tiered"
	"github.com/klauern/skillsync/internal/similarity"
	"github.com/klauern/skillsync/internal/sync"
	"github.com/klauern/skillsync/internal/ui"
	"github.com/klauern/skillsync/internal/ui/tui"
	"github.com/klauern/skillsync/internal/util"
	"github.com/klauern/skillsync/internal/validation"
)

func configCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Manage skillsync configuration",
		Description: `Manage skillsync configuration settings.

   Configuration is loaded from: ~/.skillsync/config.yaml
   Environment variables can override any setting with SKILLSYNC_* prefix.

   Examples:
     skillsync config show           # Show current configuration
     skillsync config init           # Create default config file
     skillsync config path           # Show config file path
     skillsync config edit           # Edit config file (opens in $EDITOR)`,
		Commands: []*cli.Command{
			configShowCommand(),
			configInitCommand(),
			configPathCommand(),
			configEditCommand(),
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			// Default action: show configuration
			return showConfig()
		},
	}
}

func configShowCommand() *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "Display current configuration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "yaml",
				Usage:   "Output format: yaml, json",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			format := cmd.String("format")
			return showConfigWithFormat(format)
		},
	}
}

func configInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Create default configuration file",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Overwrite existing config file",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			force := cmd.Bool("force")
			return initConfig(force)
		},
	}
}

func configPathCommand() *cli.Command {
	return &cli.Command{
		Name:  "path",
		Usage: "Display configuration file paths",
		Action: func(_ context.Context, _ *cli.Command) error {
			return showConfigPaths()
		},
	}
}

func configEditCommand() *cli.Command {
	return &cli.Command{
		Name:  "edit",
		Usage: "Edit configuration file in $EDITOR",
		Action: func(_ context.Context, _ *cli.Command) error {
			return editConfig()
		},
	}
}

// showConfig displays the current configuration.
func showConfig() error {
	return showConfigWithFormat("yaml")
}

// showConfigWithFormat displays the configuration in the specified format.
func showConfigWithFormat(format string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(cfg)
	case "yaml":
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		fmt.Println("# skillsync configuration")
		if config.Exists() {
			fmt.Printf("# Loaded from: %s\n", config.FilePath())
		} else {
			fmt.Println("# Using default configuration (no config file found)")
		}
		fmt.Println()
		fmt.Print(string(data))
		return nil
	default:
		return fmt.Errorf("unsupported format: %s (use yaml or json)", format)
	}
}

// initConfig creates a default configuration file.
func initConfig(force bool) error {
	configPath := config.FilePath()

	if config.Exists() && !force {
		return fmt.Errorf("config file already exists at %s (use --force to overwrite)", configPath)
	}

	cfg := config.Default()
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Created config file: %s\n", configPath)
	return nil
}

// showConfigPaths displays all configuration-related paths.
func showConfigPaths() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Configuration paths:")
	fmt.Printf("  Config file:     %s", config.FilePath())
	if config.Exists() {
		fmt.Println(" (exists)")
	} else {
		fmt.Println(" (not found)")
	}
	fmt.Printf("  Config dir:      %s\n", util.SkillsyncConfigPath())

	fmt.Println("\nPlatform paths:")
	fmt.Printf("  Claude Code:     %v\n", cfg.Platforms.ClaudeCode.SkillsPaths)
	fmt.Printf("  Cursor:          %v\n", cfg.Platforms.Cursor.SkillsPaths)
	fmt.Printf("  Codex:           %v\n", cfg.Platforms.Codex.SkillsPaths)

	fmt.Println("\nData paths:")
	fmt.Printf("  Backups:         %s\n", cfg.Backup.Location)
	fmt.Printf("  Cache:           %s\n", cfg.Cache.Location)
	fmt.Printf("  Plugins:         %s\n", util.SkillsyncPluginsPath())
	fmt.Printf("  Metadata:        %s\n", util.SkillsyncMetadataPath())

	return nil
}

// editConfig opens the config file in the user's editor.
func editConfig() error {
	configPath := config.FilePath()

	// Ensure config file exists
	if !config.Exists() {
		fmt.Println("No config file found. Creating default configuration...")
		if err := initConfig(false); err != nil {
			return err
		}
	}

	// Find editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		return fmt.Errorf("no editor found - set $EDITOR or $VISUAL environment variable")
	}

	fmt.Printf("Opening %s in %s...\n", configPath, editor)
	fmt.Println("Note: After editing, run 'skillsync config show' to verify your changes.")

	// We don't actually exec here - just show the command to run
	// This is safer and more portable
	fmt.Printf("\nRun: %s %s\n", editor, configPath)
	return nil
}

func discoveryCommand() *cli.Command {
	return &cli.Command{
		Name:    "discover",
		Aliases: []string{"discovery", "list"},
		Usage:   "Discover and list skills across platforms",
		UsageText: `skillsync discover [options]
   skillsync discover --interactive       # Interactive TUI mode
   skillsync discover --platform claude-code
   skillsync discover --no-plugins
   skillsync discover --repo https://github.com/user/plugins
   skillsync discover --format json`,
		Description: `Discover and list skills from all supported AI coding platforms.

   Supported platforms: claude-code, cursor, codex

   Plugin discovery: By default, skills from installed Claude Code plugins
   are included from ~/.skillsync/plugins/. Use --no-plugins to exclude them,
   or specify a Git repository with --repo to fetch plugins from.

   Output formats: table (default), json, yaml

   Use --interactive (-i) for a TUI with keyboard navigation and filtering.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "interactive",
				Aliases: []string{"i"},
				Usage:   "Interactive TUI mode with keyboard navigation",
			},
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Filter by platform (claude-code, cursor, codex)",
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
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			platform := cmd.String("platform")
			scopeStr := cmd.String("scope")
			format := cmd.String("format")
			excludePlugins := cmd.Bool("no-plugins")
			repoURL := cmd.String("repo")
			noCache := cmd.Bool("no-cache")
			interactive := cmd.Bool("interactive")

			// Include plugins by default unless --no-plugins is set
			includePlugins := !excludePlugins

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
			var allSkills []model.Skill
			for _, p := range platforms {
				skills, err := parsePlatformSkillsWithScope(p, scopeFilter)
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

			// Output results
			if interactive {
				return discoverSkillsInteractive(allSkills)
			}
			return outputSkills(allSkills, format)
		},
	}
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
		if err == nil {
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
		result, err := tui.RunSyncList(cfg.sourceSkills, cfg.sourceSpec.Platform, cfg.targetSpec.Platform)
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

			diffResult, err := tui.RunSyncDiff(result.PreviewSkill, targetSkill, cfg.sourceSpec.Platform, cfg.targetSpec.Platform)
			if err != nil {
				return fmt.Errorf("diff preview error: %w", err)
			}

			switch diffResult.Action {
			case tui.DiffActionBack:
				// Continue the loop to go back to the list
				continue
			case tui.DiffActionSync:
				// Sync just this one skill
				if err := executeSyncForSkills(cfg, []model.Skill{diffResult.Skill}); err != nil {
					return err
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

			if err := executeSyncForSkills(cfg, result.SelectedSkills); err != nil {
				return err
			}
			return nil
		}
	}
}

// executeSyncForSkills performs the actual sync operation for the given skills
func executeSyncForSkills(cfg *syncConfig, skills []model.Skill) error {
	// Create backup before sync
	if !cfg.skipBackup {
		prepareBackup(cfg.targetSpec.Platform)
	}

	// Create sync options and execute
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

	displaySyncResults(result)

	if !result.Success() {
		return errors.New("sync completed with errors")
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
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
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
	case "claudecode":
		return ui.Info(formatted)
	case "cursor":
		return ui.Success(formatted)
	case "codex":
		return ui.Warning(formatted)
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

func syncCommand() *cli.Command {
	return &cli.Command{
		Name:      "sync",
		Usage:     "Synchronize skills across platforms",
		UsageText: "skillsync sync [options] <source> <target>",
		Description: `Synchronize skills between AI coding platforms.

   Supported platforms: claudecode, cursor, codex

   Platform spec format: platform[:scope[,scope2,...]]
     - cursor           All scopes from cursor (source), user scope (target)
     - cursor:repo      Only repo scope
     - cursor:repo,user Both repo and user scopes (source only)

   Valid scopes: repo, user, admin, system, builtin
   Target scope must be repo or user (writable locations)

   Strategies:
     overwrite   - Replace target skills unconditionally (default)
     skip        - Skip skills that already exist in target
     newer       - Copy only if source is newer than target
     merge       - Merge source and target content
     three-way   - Intelligent merge with conflict detection
     interactive - Prompt for each conflict

   Examples:
     skillsync sync cursor claudecode           # All cursor skills to claudecode user scope
     skillsync sync cursor:repo claudecode:user # Repo skills to user scope
     skillsync sync cursor:repo,user codex:repo # Multiple source scopes to repo
     skillsync sync --dry-run cursor codex      # Preview changes
     skillsync sync --strategy=skip cursor codex
     skillsync sync --interactive cursor codex  # Interactive TUI mode`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "interactive",
				Aliases: []string{"i"},
				Usage:   "Interactive TUI mode with skill selection and diff preview",
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   "Preview changes without modifying files",
			},
			&cli.StringFlag{
				Name:    "strategy",
				Aliases: []string{"s"},
				Value:   "overwrite",
				Usage:   "Conflict resolution strategy: overwrite, skip, newer, merge, three-way, interactive, smart",
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
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			cfg, err := parseSyncConfig(cmd)
			if err != nil {
				return err
			}

			interactive := cmd.Bool("interactive")

			// Always parse source skills (use tiered parser for scope filtering)
			if cfg.sourceSpec.HasScopes() {
				// User specified scopes - use tiered parser for scope filtering
				cfg.sourceSkills, err = parsePlatformSkillsWithScope(cfg.sourceSpec.Platform, cfg.sourceSpec.Scopes)
			} else {
				// No scopes specified - use basic parser (respects env vars for E2E tests)
				cfg.sourceSkills, err = parsePlatformSkills(cfg.sourceSpec.Platform)
			}
			if err != nil {
				return fmt.Errorf("failed to parse source skills: %w", err)
			}

			// Interactive TUI mode
			if interactive {
				return syncSkillsInteractive(cfg)
			}

			// Validate source skills before sync (unless skipped)
			if !cfg.skipValidation {
				if err := validateSourceSkills(cfg); err != nil {
					return err
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
			if !cfg.dryRun && !cfg.skipBackup {
				prepareBackup(cfg.targetSpec.Platform)
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
			if result.HasConflicts() && cfg.strategy == sync.StrategyInteractive {
				// Gather conflicts
				var conflicts []*sync.Conflict
				for _, sr := range result.Conflicts() {
					if sr.Conflict != nil {
						conflicts = append(conflicts, sr.Conflict)
					}
				}

				var resolved map[string]string

				// Try TUI first, fall back to CLI if TTY not available
				tuiResult, err := tui.RunConflictList(conflicts)
				if err != nil {
					// TTY not available (e.g., in tests or CI), fall back to CLI resolver
					resolver := NewConflictResolver()
					resolver.DisplayConflictSummary(conflicts)
					resolved, err = resolver.ResolveConflicts(conflicts)
					if err != nil {
						return fmt.Errorf("conflict resolution failed: %w", err)
					}
				} else {
					// Handle TUI cancellation
					if tuiResult.Action == tui.ConflictActionNone || tuiResult.Action == tui.ConflictActionCancel {
						fmt.Println("Conflict resolution cancelled")
						return nil
					}

					// Convert TUI resolutions to resolved content map
					resolved = make(map[string]string)
					for _, resolution := range tuiResult.Resolutions {
						if resolution.Resolution == sync.ResolutionSkip {
							continue
						}

						// For merge resolution, use the provided content
						if resolution.Resolution == sync.ResolutionMerge {
							resolved[resolution.SkillName] = resolution.Content
						} else {
							// For source/target, find the appropriate content from conflicts
							for _, conflict := range conflicts {
								if conflict.SkillName == resolution.SkillName {
									if resolution.Resolution == sync.ResolutionUseSource {
										resolved[resolution.SkillName] = conflict.Source.Content
									} else if resolution.Resolution == sync.ResolutionUseTarget {
										resolved[resolution.SkillName] = conflict.Target.Content
									}
									break
								}
							}
						}
					}
				}

				// Apply resolved content
				if !cfg.dryRun && len(resolved) > 0 {
					if err := applyResolvedConflicts(result, resolved); err != nil {
						return fmt.Errorf("failed to apply resolved conflicts: %w", err)
					}
				}

				fmt.Printf("\n✓ Resolved %d conflict(s)\n", len(resolved))
			}

			displaySyncResults(result)

			if !result.Success() {
				return errors.New("sync completed with errors")
			}

			return nil
		},
	}
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
	sourceSkills   []model.Skill
}

// parseSyncConfig parses and validates sync command arguments and flags
func parseSyncConfig(cmd *cli.Command) (*syncConfig, error) {
	args := cmd.Args()
	if args.Len() != 2 {
		return nil, errors.New("sync requires exactly 2 arguments: <source> <target>")
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

	if sourceSpec.Platform == targetSpec.Platform {
		return nil, fmt.Errorf("source and target platforms cannot be the same: %s", sourceSpec.Platform)
	}

	strategyStr := cmd.String("strategy")
	strategy := sync.Strategy(strategyStr)
	if !strategy.IsValid() {
		return nil, fmt.Errorf("invalid strategy %q (valid: overwrite, skip, newer, merge, three-way, interactive, smart)", strategyStr)
	}

	return &syncConfig{
		sourceSpec:     sourceSpec,
		targetSpec:     targetSpec,
		dryRun:         cmd.Bool("dry-run"),
		strategy:       strategy,
		skipBackup:     cmd.Bool("skip-backup"),
		skipValidation: cmd.Bool("skip-validation"),
		yesFlag:        cmd.Bool("yes"),
		sourceSkills:   make([]model.Skill, 0),
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
		return err
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
	fmt.Println("\nCreating backup before sync...")

	// Run automatic cleanup to maintain retention policy
	cleanupOpts := backup.DefaultCleanupOptions()
	cleanupOpts.Platform = string(targetPlatform)

	deleted, err := backup.CleanupBackups(cleanupOpts)
	if err != nil {
		fmt.Printf("Warning: backup cleanup failed: %v\n", err)
	} else if len(deleted) > 0 {
		fmt.Printf("Cleaned up %d old backup(s)\n", len(deleted))
	}

	fmt.Println("Backup infrastructure ready")
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
		}
	}
}

// applyResolvedConflicts writes the resolved conflict content to the target files.
func applyResolvedConflicts(result *sync.Result, resolved map[string]string) error {
	for i := range result.Skills {
		sr := &result.Skills[i]
		if sr.Action == sync.ActionConflict {
			if content, ok := resolved[sr.Skill.Name]; ok {
				// #nosec G306 - skill files should be readable
				if err := os.WriteFile(sr.TargetPath, []byte(content), 0o644); err != nil {
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

// parsePlatformSkills parses skills from the given platform using env-var-respecting paths.
// This is used by the sync command when no specific scopes are requested.
func parsePlatformSkills(platform model.Platform) ([]model.Skill, error) {
	// Get path from validation which respects env vars
	basePath, err := validation.GetPlatformPath(platform)
	if err != nil {
		return nil, fmt.Errorf("failed to get platform path for %s: %w", platform, err)
	}

	// Create a direct parser for this path
	var parser interface{ Parse() ([]model.Skill, error) }
	switch platform {
	case model.ClaudeCode:
		parser = claude.New(basePath)
	case model.Cursor:
		parser = cursor.New(basePath)
	case model.Codex:
		parser = codex.New(basePath)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	return parser.Parse()
}

// parsePlatformSkillsWithScope parses skills from the given platform with optional scope filtering.
// If scopeFilter is nil or empty, all scopes are included.
func parsePlatformSkillsWithScope(platform model.Platform, scopeFilter []model.SkillScope) ([]model.Skill, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	paths, repoRoot, err := platformSkillsPaths(cfg, platform)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return []model.Skill{}, nil
	}

	return parsePlatformSkillsFromPaths(platform, paths, repoRoot, scopeFilter), nil
}

func platformSkillsPaths(cfg *config.Config, platform model.Platform) ([]string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get working directory: %w", err)
	}
	repoRoot := util.GetRepoRoot(cwd)

	var rawPaths []string
	switch platform {
	case model.ClaudeCode:
		rawPaths = cfg.Platforms.ClaudeCode.SkillsPaths
		if len(rawPaths) == 0 && cfg.Platforms.ClaudeCode.SkillsPath != "" { //nolint:staticcheck // backward compatibility
			rawPaths = []string{cfg.Platforms.ClaudeCode.SkillsPath} //nolint:staticcheck // backward compatibility
		}
	case model.Cursor:
		rawPaths = cfg.Platforms.Cursor.SkillsPaths
		if len(rawPaths) == 0 && cfg.Platforms.Cursor.SkillsPath != "" { //nolint:staticcheck // backward compatibility
			rawPaths = []string{cfg.Platforms.Cursor.SkillsPath} //nolint:staticcheck // backward compatibility
		}
	case model.Codex:
		rawPaths = cfg.Platforms.Codex.SkillsPaths
		if len(rawPaths) == 0 && cfg.Platforms.Codex.SkillsPath != "" { //nolint:staticcheck // backward compatibility
			rawPaths = []string{cfg.Platforms.Codex.SkillsPath} //nolint:staticcheck // backward compatibility
		}
	default:
		return nil, repoRoot, fmt.Errorf("unsupported platform: %s", platform)
	}

	return resolveSkillsPaths(rawPaths, cwd, repoRoot), repoRoot, nil
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

func parsePlatformSkillsFromPaths(
	platform model.Platform,
	paths []string,
	repoRoot string,
	scopeFilter []model.SkillScope,
) []model.Skill {
	parserFactory := tiered.ParserFactoryFor(platform)
	skillsByName := make(map[string]model.Skill)

	scopeSet := make(map[model.SkillScope]bool)
	for _, s := range scopeFilter {
		scopeSet[s] = true
	}

	for _, path := range paths {
		scope := inferScopeForPath(path, repoRoot)
		if len(scopeSet) > 0 && !scopeSet[scope] {
			continue
		}

		if _, err := os.Stat(path); err != nil {
			continue
		}

		pathParser := parserFactory(path)
		skills, err := pathParser.Parse()
		if err != nil {
			continue
		}

		for _, skill := range skills {
			skill.Scope = scope
			if existing, exists := skillsByName[skill.Name]; exists {
				if skill.Scope.IsHigherPrecedence(existing.Scope) {
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

func inferScopeForPath(path, repoRoot string) model.SkillScope {
	cleaned := filepath.Clean(path)

	if repoRoot != "" {
		root := filepath.Clean(repoRoot)
		rootWithSep := root + string(os.PathSeparator)
		if cleaned == root || strings.HasPrefix(cleaned, rootWithSep) {
			return model.ScopeRepo
		}
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
			// Target doesn't exist - validate parent directory is writable
			parentDir := filepath.Dir(targetPath)
			if err := validation.ValidatePath(parentDir, targetPlatform); err != nil {
				return fmt.Errorf("target parent directory validation failed: %w", err)
			}
			// Check write permission on parent
			if err := checkWritePermission(parentDir); err != nil {
				return fmt.Errorf("target directory not writable: %w", err)
			}
		} else {
			return fmt.Errorf("target validation failed: %w", err)
		}
	}

	return nil
}

// checkWritePermission verifies a directory is writable
func checkWritePermission(path string) error {
	// If path doesn't exist, check parent
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = "." // fallback to current directory
	}

	testFile := path + "/.skillsync-write-test"
	// #nosec G304 - testFile is constructed from validated path and is not user input
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("cannot write to directory: %w", err)
	}
	_ = f.Close()
	_ = os.Remove(testFile)
	return nil
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

func exportCommand() *cli.Command {
	return &cli.Command{
		Name:      "export",
		Usage:     "Export skills to different formats",
		UsageText: "skillsync export [options]",
		Description: `Export skills to JSON, YAML, or Markdown formats.

   Supported formats: json (default), yaml, markdown

   Examples:
     skillsync export
     skillsync export --format yaml
     skillsync export --platform claude-code --format markdown
     skillsync export --output skills.json`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Filter by platform (claude-code, cursor, codex)",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "json",
				Usage:   "Output format: json, yaml, markdown",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file path (default: stdout)",
			},
			&cli.BoolFlag{
				Name:  "no-metadata",
				Usage: "Exclude metadata fields from export",
			},
			&cli.BoolFlag{
				Name:  "compact",
				Usage: "Compact output (no pretty-printing)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return runExport(cmd)
		},
	}
}

// runExport executes the export command.
func runExport(cmd *cli.Command) error {
	// Parse format
	formatStr := cmd.String("format")
	format, err := export.ParseFormat(formatStr)
	if err != nil {
		return err
	}

	// Parse platform filter
	var platform model.Platform
	platformStr := cmd.String("platform")
	if platformStr != "" {
		p, err := model.ParsePlatform(platformStr)
		if err != nil {
			return fmt.Errorf("invalid platform: %w", err)
		}
		platform = p
	}

	// Build export options
	opts := export.Options{
		Format:          format,
		Pretty:          !cmd.Bool("compact"),
		IncludeMetadata: !cmd.Bool("no-metadata"),
		Platform:        platform,
	}

	// Discover skills
	skills, err := discoverSkillsForExport(platform)
	if err != nil {
		return fmt.Errorf("failed to discover skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Fprintln(os.Stderr, "No skills found to export.")
		return nil
	}

	// Create exporter
	exporter := export.New(opts)

	// Determine output destination
	outputPath := cmd.String("output")
	if outputPath != "" {
		// Write to file
		// #nosec G304 - outputPath is provided by user
		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}

		if err := exporter.Export(skills, file); err != nil {
			_ = file.Close()
			return fmt.Errorf("export failed: %w", err)
		}

		if err := file.Close(); err != nil {
			return fmt.Errorf("failed to close output file: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Exported %d skill(s) to %s\n", len(skills), outputPath)
	} else {
		// Write to stdout
		if err := exporter.Export(skills, os.Stdout); err != nil {
			return fmt.Errorf("export failed: %w", err)
		}
	}

	return nil
}

// discoverSkillsForExport discovers skills optionally filtered by platform.
func discoverSkillsForExport(platform model.Platform) ([]model.Skill, error) {
	var platforms []model.Platform
	if platform != "" {
		platforms = []model.Platform{platform}
	} else {
		platforms = model.AllPlatforms()
	}

	var allSkills []model.Skill
	for _, p := range platforms {
		skills, err := parsePlatformSkills(p)
		if err != nil {
			// Log warning but continue with other platforms
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", p, err)
			continue
		}

		allSkills = append(allSkills, skills...)
	}

	return allSkills, nil
}

func backupCommand() *cli.Command {
	return &cli.Command{
		Name:  "backup",
		Usage: "Manage skillsync backups",
		Description: `Manage backups of skill files.

   Backups are automatically created before sync operations.
   Use these commands to view, verify, restore, and rollback backups.

   Examples:
     skillsync backup list                    # List all backups
     skillsync backup list --platform claude-code
     skillsync backup list --format json
     skillsync backup restore <backup-id>     # Restore a backup
     skillsync backup rollback --file <path>  # Rollback to latest backup`,
		Commands: []*cli.Command{
			backupListCommand(),
			backupRestoreCommand(),
			backupDeleteCommand(),
			backupVerifyCommand(),
			backupRollbackCommand(),
		},
		Action: func(_ context.Context, _ *cli.Command) error {
			// Default action: list backups
			return listBackups("", "table", 0)
		},
	}
}

func backupListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List existing backups with metadata",
		UsageText: `skillsync backup list [options]
   skillsync backup list --interactive       # Interactive TUI mode
   skillsync backup list --platform claude-code
   skillsync backup list --format json
   skillsync backup list --limit 10`,
		Description: `List all backups with their metadata including timestamp, size, and platform.

   Output includes: ID, Platform, Source File, Created At, Size

   Formats: table (default), json, yaml

   Use --interactive (-i) for a TUI with keyboard navigation and actions.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "interactive",
				Aliases: []string{"i"},
				Usage:   "Interactive TUI mode with keyboard navigation",
			},
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Filter by platform (claude-code, cursor, codex)",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "table",
				Usage:   "Output format: table, json, yaml",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n"},
				Value:   0,
				Usage:   "Limit results to N most recent backups (0 = unlimited)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			platform := cmd.String("platform")
			format := cmd.String("format")
			limit := cmd.Int("limit")
			interactive := cmd.Bool("interactive")

			if interactive {
				return listBackupsInteractive(platform)
			}
			return listBackups(platform, format, int(limit))
		},
	}
}

func backupRestoreCommand() *cli.Command {
	return &cli.Command{
		Name:  "restore",
		Usage: "Restore a backup to its original or specified location",
		UsageText: `skillsync backup restore <backup-id> [options]
   skillsync backup restore 20240125-120000-abc12345
   skillsync backup restore 20240125-120000-abc12345 --target /path/to/restore
   skillsync backup restore 20240125-120000-abc12345 --force`,
		Description: `Restore a skill file from a backup.

   By default, restores to the original source path. Use --target to specify
   a different location.

   The restore operation verifies backup integrity using SHA256 hash before
   restoring. Use --force to skip the confirmation prompt.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "target",
				Aliases: []string{"t"},
				Usage:   "Target path for restoration (defaults to original source path)",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation prompt before overwriting",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			if args.Len() < 1 {
				return errors.New("backup ID is required")
			}

			backupID := args.Get(0)
			targetPath := cmd.String("target")
			force := cmd.Bool("force")

			return restoreBackup(backupID, targetPath, force)
		},
	}
}

// listBackups retrieves and displays backups based on filters
func listBackups(platform, format string, limit int) error {
	backups, err := backup.ListBackups(platform)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	// Apply limit if specified
	if limit > 0 && len(backups) > limit {
		backups = backups[:limit]
	}

	return outputBackups(backups, format)
}

// listBackupsInteractive runs the interactive TUI for backup management
func listBackupsInteractive(platform string) error {
	backups, err := backup.ListBackups(platform)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	result, err := tui.RunBackupList(backups)
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Handle the selected action
	switch result.Action {
	case tui.ActionRestore:
		fmt.Printf("\nRestoring backup: %s\n", result.BackupID)
		return restoreBackup(result.BackupID, "", false)
	case tui.ActionDelete:
		fmt.Printf("\nDeleting backup: %s\n", result.BackupID)
		return deleteBackupsByID([]string{result.BackupID}, true) // force=true since already confirmed in TUI
	case tui.ActionVerify:
		fmt.Printf("\nVerifying backup: %s\n", result.BackupID)
		return verifyBackupsByID([]string{result.BackupID})
	case tui.ActionNone:
		// User quit without action
		return nil
	}

	return nil
}

// restoreBackup restores a backup to the original or specified target path
func restoreBackup(backupID, targetPath string, force bool) error {
	// Load index to get backup metadata
	index, err := backup.LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load backup index: %w", err)
	}

	// Find the backup
	metadata, exists := index.Backups[backupID]
	if !exists {
		return fmt.Errorf("backup %q not found", backupID)
	}

	// Use original source path if no target specified
	if targetPath == "" {
		targetPath = metadata.SourcePath
	}

	// Check if target file exists
	targetExists := false
	if _, err := os.Stat(targetPath); err == nil {
		targetExists = true
	}

	// Display restore details
	fmt.Println("\nBackup Details:")
	fmt.Printf("  ID:       %s\n", metadata.ID)
	fmt.Printf("  Platform: %s\n", metadata.Platform)
	fmt.Printf("  Size:     %s\n", formatSize(metadata.Size))
	fmt.Printf("  Created:  %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Source:   %s\n", metadata.SourcePath)
	fmt.Printf("  Target:   %s\n", targetPath)

	if targetExists {
		fmt.Println("\n⚠️  Target file already exists and will be overwritten.")
	}

	// Confirm unless force flag is set
	if !force {
		message := fmt.Sprintf("Restore backup to %s?", targetPath)
		level := riskLevelInfo
		if targetExists {
			level = riskLevelWarning
		}

		confirmed, err := confirmAction(message, level)
		if err != nil {
			return fmt.Errorf("confirmation error: %w", err)
		}
		if !confirmed {
			fmt.Println("Restore cancelled.")
			return nil
		}
	}

	// Perform the restore
	if err := backup.RestoreBackup(backupID, targetPath); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	fmt.Printf("\n✓ Successfully restored backup to %s\n", targetPath)
	return nil
}

func backupDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete old backups",
		UsageText: `skillsync backup delete [options] [backup-id...]
   skillsync backup delete <backup-id>              # Delete specific backup
   skillsync backup delete --older-than 30d         # Delete backups older than 30 days
   skillsync backup delete --keep-latest 5          # Keep only 5 most recent backups
   skillsync backup delete --platform claude-code --keep-latest 3`,
		Description: `Delete backups by ID, age, or count-based retention.

   By ID: Pass one or more backup IDs as arguments
   By Age: Use --older-than with a duration (e.g., 30d, 2w, 168h)
   By Count: Use --keep-latest N to keep only N most recent backups

   Combine --platform with --older-than or --keep-latest to filter by platform.
   Use --force to skip confirmation prompt.

   Examples of duration formats:
     30d   = 30 days
     2w    = 2 weeks (14 days)
     168h  = 168 hours (7 days)`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "older-than",
				Aliases: []string{"o"},
				Usage:   "Delete backups older than duration (e.g., 30d, 2w, 168h)",
			},
			&cli.IntFlag{
				Name:    "keep-latest",
				Aliases: []string{"k"},
				Value:   0,
				Usage:   "Keep only N most recent backups (0 = disabled)",
			},
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Filter by platform (claude-code, cursor, codex)",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation prompt",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			olderThan := cmd.String("older-than")
			keepLatest := cmd.Int("keep-latest")
			platform := cmd.String("platform")
			force := cmd.Bool("force")

			// Determine delete mode based on arguments and flags
			if args.Len() > 0 {
				// Delete by specific IDs
				ids := make([]string, args.Len())
				for i := 0; i < args.Len(); i++ {
					ids[i] = args.Get(i)
				}
				return deleteBackupsByID(ids, force)
			}

			if olderThan != "" || keepLatest > 0 {
				// Delete by retention policy
				return deleteBackupsByPolicy(olderThan, int(keepLatest), platform, force)
			}

			return errors.New("either backup IDs or --older-than/--keep-latest flag is required")
		},
	}
}

func backupVerifyCommand() *cli.Command {
	return &cli.Command{
		Name:  "verify",
		Usage: "Verify backup integrity using SHA256 checksums",
		UsageText: `skillsync backup verify [backup-id...]
   skillsync backup verify                           # Verify all backups
   skillsync backup verify 20240125-120000-abc12345  # Verify specific backup
   skillsync backup verify --platform claude-code    # Verify backups for a platform`,
		Description: `Verify backup integrity by comparing file content against stored SHA256 checksums.

   Without arguments, verifies all backups. Pass one or more backup IDs to verify
   specific backups. Use --platform to filter verification to a specific platform.

   The command reports:
     ✓ OK       - Backup file is intact and matches stored checksum
     ✗ CORRUPT  - Backup file has been modified or corrupted
     ✗ MISSING  - Backup file no longer exists on disk

   Exit codes:
     0 - All verified backups are intact
     1 - One or more backups failed verification`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Filter by platform (claude-code, cursor, codex)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			platform := cmd.String("platform")

			if args.Len() > 0 {
				// Verify specific backup IDs
				ids := make([]string, args.Len())
				for i := 0; i < args.Len(); i++ {
					ids[i] = args.Get(i)
				}
				return verifyBackupsByID(ids)
			}

			// Verify all backups (optionally filtered by platform)
			return verifyAllBackups(platform)
		},
	}
}

// verifyBackupsByID verifies specific backups by their IDs
func verifyBackupsByID(ids []string) error {
	fmt.Printf("Verifying %d backup(s)...\n\n", len(ids))

	var failed int
	for _, id := range ids {
		if err := backup.VerifyBackup(id); err != nil {
			fmt.Printf("✗ %-28s FAILED: %v\n", id, err)
			failed++
		} else {
			fmt.Printf("✓ %-28s OK\n", id)
		}
	}

	fmt.Println()
	if failed > 0 {
		fmt.Printf("Verification complete: %d OK, %d FAILED\n", len(ids)-failed, failed)
		return fmt.Errorf("%d backup(s) failed verification", failed)
	}

	fmt.Printf("Verification complete: %d OK\n", len(ids))
	return nil
}

// verifyAllBackups verifies all backups, optionally filtered by platform
func verifyAllBackups(platform string) error {
	backups, err := backup.ListBackups(platform)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		fmt.Println("No backups found to verify.")
		return nil
	}

	fmt.Printf("Verifying %d backup(s)...\n\n", len(backups))

	var ok, failed int
	for _, b := range backups {
		if err := backup.VerifyBackup(b.ID); err != nil {
			fmt.Printf("✗ %-28s %-12s FAILED: %v\n", b.ID, b.Platform, err)
			failed++
		} else {
			fmt.Printf("✓ %-28s %-12s OK\n", b.ID, b.Platform)
			ok++
		}
	}

	fmt.Println()
	if failed > 0 {
		fmt.Printf("Verification complete: %d OK, %d FAILED\n", ok, failed)
		return fmt.Errorf("%d backup(s) failed verification", failed)
	}

	fmt.Printf("Verification complete: %d OK\n", ok)
	return nil
}

func backupRollbackCommand() *cli.Command {
	return &cli.Command{
		Name:  "rollback",
		Usage: "Rollback to a previous backup version of a file",
		UsageText: `skillsync backup rollback [options] [backup-id]
   skillsync backup rollback <backup-id>                # Rollback to specific backup by ID
   skillsync backup rollback --file /path/to/file       # Rollback to latest backup of file
   skillsync backup rollback --file /path/to/file --list # Show backup history for file
   skillsync backup rollback <backup-id> --target /custom/path
   skillsync backup rollback <backup-id> --force        # Skip confirmation`,
		Description: `Rollback to a previous backup version using the automatic backup mechanism.

   Two modes of operation:
   1. Rollback by backup ID: Specify a backup ID to restore that exact version
   2. Rollback by file path: Use --file to rollback to the most recent backup

   Use --list with --file to view backup history for a specific file before
   deciding which version to restore.

   The rollback operation:
   - Verifies backup integrity using SHA256 checksums
   - Shows backup details before confirmation
   - Warns if target file will be overwritten
   - Supports custom target path with --target
   - Can be forced with --force to skip confirmation

   Examples:
     # Show backup history for a file
     skillsync backup rollback --file ~/.claude/skills/myskill.md --list

     # Rollback to latest backup of a file
     skillsync backup rollback --file ~/.claude/skills/myskill.md

     # Rollback to specific backup version
     skillsync backup rollback 20240128-143022-abc12345

     # Rollback with custom target location
     skillsync backup rollback 20240128-143022-abc12345 --target /tmp/restored.md`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "Source file path to rollback (uses latest backup)",
			},
			&cli.BoolFlag{
				Name:    "list",
				Aliases: []string{"l"},
				Usage:   "List backup history for --file (does not rollback)",
			},
			&cli.StringFlag{
				Name:    "target",
				Aliases: []string{"t"},
				Usage:   "Custom target path for rollback (defaults to original source path)",
			},
			&cli.BoolFlag{
				Name:    "force",
				Usage:   "Skip confirmation prompt before overwriting",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Preview rollback without making changes",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			filePath := cmd.String("file")
			listFlag := cmd.Bool("list")
			targetPath := cmd.String("target")
			force := cmd.Bool("force")
			dryRun := cmd.Bool("dry-run")

			// Mode 1: List backup history for a file
			if listFlag {
				if filePath == "" {
					return errors.New("--file is required when using --list")
				}
				return listBackupHistory(filePath)
			}

			// Mode 2: Rollback by file path (latest backup)
			if filePath != "" {
				return rollbackByFile(filePath, targetPath, force, dryRun)
			}

			// Mode 3: Rollback by backup ID
			if args.Len() < 1 {
				return errors.New("backup ID is required (or use --file to rollback by file path)")
			}

			backupID := args.Get(0)
			return rollbackByID(backupID, targetPath, force, dryRun)
		},
	}
}

// listBackupHistory shows all backups for a specific source file
func listBackupHistory(sourcePath string) error {
	history, err := backup.GetBackupHistory(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get backup history: %w", err)
	}

	if len(history) == 0 {
		fmt.Printf("No backups found for: %s\n", sourcePath)
		return nil
	}

	fmt.Printf("\nBackup history for: %s\n", sourcePath)
	fmt.Printf("Found %d backup(s):\n\n", len(history))

	// Display in table format
	fmt.Printf("%-30s %-12s %-20s %s\n", "BACKUP ID", "PLATFORM", "CREATED", "SIZE")
	fmt.Println(strings.Repeat("-", 80))
	for _, b := range history {
		fmt.Printf("%-30s %-12s %-20s %s\n",
			b.ID,
			b.Platform,
			b.CreatedAt.Format("2006-01-02 15:04:05"),
			formatSize(b.Size))
	}

	fmt.Printf("\nUse 'skillsync backup rollback <backup-id>' to restore a specific version\n")
	fmt.Printf("Use 'skillsync backup rollback --file %s' to restore the latest version\n", sourcePath)

	return nil
}

// rollbackByFile rollbacks to the latest backup of a specific file
func rollbackByFile(sourcePath, targetPath string, force, dryRun bool) error {
	history, err := backup.GetBackupHistory(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get backup history: %w", err)
	}

	if len(history) == 0 {
		return fmt.Errorf("no backups found for: %s", sourcePath)
	}

	// Use the most recent backup (first in the sorted list)
	latestBackup := history[0]

	fmt.Printf("Rolling back to latest backup:\n")
	fmt.Printf("  Backup ID: %s\n", latestBackup.ID)
	fmt.Printf("  Created:   %s\n", latestBackup.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()

	return rollbackByID(latestBackup.ID, targetPath, force, dryRun)
}

// rollbackByID rollbacks to a specific backup by ID
func rollbackByID(backupID, targetPath string, force, dryRun bool) error {
	// Load index to get backup metadata
	index, err := backup.LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load backup index: %w", err)
	}

	// Find the backup
	metadata, exists := index.Backups[backupID]
	if !exists {
		return fmt.Errorf("backup %q not found", backupID)
	}

	// Use original source path if no target specified
	if targetPath == "" {
		targetPath = metadata.SourcePath
	}

	// Check if target file exists
	targetExists := false
	if _, err := os.Stat(targetPath); err == nil {
		targetExists = true
	}

	// Verify backup integrity before rollback
	if err := backup.VerifyBackup(backupID); err != nil {
		return fmt.Errorf("backup verification failed: %w", err)
	}

	// Display rollback details
	fmt.Println("\nRollback Details:")
	fmt.Printf("  Backup ID: %s\n", metadata.ID)
	fmt.Printf("  Platform:  %s\n", metadata.Platform)
	fmt.Printf("  Size:      %s\n", formatSize(metadata.Size))
	fmt.Printf("  Created:   %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Source:    %s\n", metadata.SourcePath)
	fmt.Printf("  Target:    %s\n", targetPath)

	if targetExists {
		fmt.Println("\n⚠️  Target file already exists and will be overwritten.")
	}

	if dryRun {
		fmt.Println("\n[DRY RUN] Would rollback to this backup (no changes made)")
		return nil
	}

	// Confirm unless force flag is set
	if !force {
		message := fmt.Sprintf("Rollback to backup %s?", backupID)
		level := riskLevelInfo
		if targetExists {
			level = riskLevelWarning
		}

		confirmed, err := confirmAction(message, level)
		if err != nil {
			return fmt.Errorf("confirmation error: %w", err)
		}
		if !confirmed {
			fmt.Println("Rollback cancelled.")
			return nil
		}
	}

	// Perform the rollback (using existing RestoreBackup function)
	if err := backup.RestoreBackup(backupID, targetPath); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	fmt.Printf("\n✓ Successfully rolled back to backup: %s\n", backupID)
	fmt.Printf("  File restored to: %s\n", targetPath)
	return nil
}

// deleteBackupsByID deletes specific backups by their IDs
func deleteBackupsByID(ids []string, force bool) error {
	// Load index to verify backups exist
	index, err := backup.LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load backup index: %w", err)
	}

	// Verify all IDs exist
	var backupsToDelete []backup.Metadata
	for _, id := range ids {
		metadata, exists := index.Backups[id]
		if !exists {
			return fmt.Errorf("backup %q not found", id)
		}
		backupsToDelete = append(backupsToDelete, metadata)
	}

	// Display what will be deleted
	fmt.Printf("\nBackups to delete (%d):\n", len(backupsToDelete))
	for _, b := range backupsToDelete {
		fmt.Printf("  - %s (%s, %s)\n", b.ID, b.Platform, formatSize(b.Size))
	}

	// Confirm unless force flag is set
	if !force {
		message := fmt.Sprintf("Delete %d backup(s)?", len(backupsToDelete))
		confirmed, err := confirmAction(message, riskLevelWarning)
		if err != nil {
			return fmt.Errorf("confirmation error: %w", err)
		}
		if !confirmed {
			fmt.Println("Delete cancelled.")
			return nil
		}
	}

	// Delete each backup
	var deleted int
	for _, b := range backupsToDelete {
		if err := backup.DeleteBackup(b.ID); err != nil {
			return fmt.Errorf("failed to delete backup %q: %w", b.ID, err)
		}
		deleted++
	}

	fmt.Printf("\n✓ Deleted %d backup(s)\n", deleted)
	return nil
}

// deleteBackupsByPolicy deletes backups based on age or count retention
func deleteBackupsByPolicy(olderThan string, keepLatest int, platform string, force bool) error {
	// Parse duration from --older-than flag
	var maxAge time.Duration
	if olderThan != "" {
		duration, err := parseDuration(olderThan)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", olderThan, err)
		}
		maxAge = duration
	}

	// Get list of backups to analyze
	backups, err := backup.ListBackups(platform)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	// Determine which backups to delete
	now := time.Now()
	var toDelete []backup.Metadata

	for i, b := range backups {
		shouldDelete := false

		// Check age
		if maxAge > 0 && now.Sub(b.CreatedAt) > maxAge {
			shouldDelete = true
		}

		// Check count limit (backups are already sorted newest first)
		if keepLatest > 0 && i >= keepLatest {
			shouldDelete = true
		}

		if shouldDelete {
			toDelete = append(toDelete, b)
		}
	}

	if len(toDelete) == 0 {
		fmt.Println("No backups match the deletion criteria.")
		return nil
	}

	// Display what will be deleted
	var totalSize int64
	fmt.Printf("\nBackups to delete (%d):\n", len(toDelete))
	for _, b := range toDelete {
		fmt.Printf("  - %s (%s, %s, %s)\n",
			b.ID, b.Platform, formatSize(b.Size), b.CreatedAt.Format("2006-01-02"))
		totalSize += b.Size
	}
	fmt.Printf("\nTotal space to free: %s\n", formatSize(totalSize))

	// Show what will be kept
	keptCount := len(backups) - len(toDelete)
	fmt.Printf("Backups remaining: %d\n", keptCount)

	// Confirm unless force flag is set
	if !force {
		message := fmt.Sprintf("Delete %d backup(s)?", len(toDelete))
		confirmed, err := confirmAction(message, riskLevelWarning)
		if err != nil {
			return fmt.Errorf("confirmation error: %w", err)
		}
		if !confirmed {
			fmt.Println("Delete cancelled.")
			return nil
		}
	}

	// Delete each backup
	var deleted int
	for _, b := range toDelete {
		if err := backup.DeleteBackup(b.ID); err != nil {
			return fmt.Errorf("failed to delete backup %q: %w", b.ID, err)
		}
		deleted++
	}

	fmt.Printf("\n✓ Deleted %d backup(s), freed %s\n", deleted, formatSize(totalSize))
	return nil
}

// parseDuration parses a duration string with support for day and week units
func parseDuration(s string) (time.Duration, error) {
	// Check for custom units (days, weeks)
	if len(s) >= 2 {
		lastChar := s[len(s)-1]
		numPart := s[:len(s)-1]

		switch lastChar {
		case 'd', 'D':
			// Days
			var days int
			if _, err := fmt.Sscanf(numPart, "%d", &days); err != nil {
				return 0, fmt.Errorf("invalid day count: %s", numPart)
			}
			return time.Duration(days) * 24 * time.Hour, nil
		case 'w', 'W':
			// Weeks
			var weeks int
			if _, err := fmt.Sscanf(numPart, "%d", &weeks); err != nil {
				return 0, fmt.Errorf("invalid week count: %s", numPart)
			}
			return time.Duration(weeks) * 7 * 24 * time.Hour, nil
		}
	}

	// Fall back to standard Go duration parsing (hours, minutes, seconds)
	return time.ParseDuration(s)
}

// outputBackups formats and prints backups in the requested format
func outputBackups(backups []backup.Metadata, format string) error {
	switch format {
	case "json":
		return outputBackupsJSON(backups)
	case "yaml":
		return outputBackupsYAML(backups)
	case "table":
		return outputBackupsTable(backups)
	default:
		return fmt.Errorf("unsupported format: %s (use table, json, or yaml)", format)
	}
}

// outputBackupsJSON prints backups as JSON
func outputBackupsJSON(backups []backup.Metadata) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(backups)
}

// outputBackupsYAML prints backups as YAML
func outputBackupsYAML(backups []backup.Metadata) error {
	data, err := yaml.Marshal(backups)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	fmt.Print(string(data))
	return nil
}

// outputBackupsTable prints backups in a table format with colored output
func outputBackupsTable(backups []backup.Metadata) error {
	if len(backups) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	// Sort by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	// Print colored headers
	fmt.Printf("%s %s %s %s %s\n",
		ui.Header(fmt.Sprintf("%-28s", "ID")),
		ui.Header(fmt.Sprintf("%-12s", "PLATFORM")),
		ui.Header(fmt.Sprintf("%-45s", "SOURCE")),
		ui.Header(fmt.Sprintf("%-20s", "CREATED")),
		ui.Header("SIZE"))
	fmt.Printf("%-28s %-12s %-45s %-20s %s\n", "--", "--------", "------", "-------", "----")

	for _, b := range backups {
		// Truncate source path if too long (use left-truncation to preserve the meaningful end)
		source := b.SourcePath
		if len(source) > 45 {
			source = "..." + source[len(source)-42:]
		}

		// Format size
		size := formatSize(b.Size)

		// Format creation time
		created := b.CreatedAt.Format("2006-01-02 15:04:05")

		// Color platform names for visual distinction
		platform := colorPlatform(string(b.Platform), 12)

		fmt.Printf("%-28s %s %-45s %-20s %s\n", b.ID, platform, source, created, size)
	}

	fmt.Printf("\nTotal: %d backup(s)\n", len(backups))
	return nil
}

// formatSize formats a byte size into a human-readable string
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// tuiCommand returns the TUI dashboard command.
func tuiCommand() *cli.Command {
	return &cli.Command{
		Name:    "tui",
		Aliases: []string{"ui"},
		Usage:   "Launch the interactive TUI dashboard",
		Description: `Launch the unified interactive TUI application for skillsync.

   The TUI provides a menu-driven interface to access all skillsync features:
   - Discover skills across all platforms
   - Manage backups (list, restore, delete, verify)
   - Sync operations between platforms
   - Compare and dedupe skills
   - Import/export operations
   - Scope and promote/demote management
   - Configuration settings

   Use arrow keys to navigate, Enter to select, and q to quit.`,
		Action: func(_ context.Context, _ *cli.Command) error {
			return runTUI()
		},
	}
}

// runTUI launches the interactive TUI dashboard and handles view navigation.
func runTUI() error {
	for {
		result, err := tui.RunDashboard()
		if err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		switch result.View {
		case tui.DashboardViewNone:
			// User quit the dashboard
			return nil

		case tui.DashboardViewDiscover:
			if err := runDiscoverTUI(); err != nil {
				return err
			}

		case tui.DashboardViewBackups:
			if err := runBackupsTUI(); err != nil {
				return err
			}

		case tui.DashboardViewSync:
			if err := runSyncTUI(); err != nil {
				return err
			}

		case tui.DashboardViewCompare:
			if err := runCompareTUI(); err != nil {
				return err
			}

		case tui.DashboardViewConfig:
			if err := runConfigTUI(); err != nil {
				return err
			}

		case tui.DashboardViewExport:
			if err := runExportTUI(); err != nil {
				return err
			}

		case tui.DashboardViewImport:
			if err := runImportTUI(); err != nil {
				return err
			}

		case tui.DashboardViewScope:
			if err := runScopeTUI(); err != nil {
				return err
			}

		case tui.DashboardViewPromote:
			if err := runPromoteDemoteTUI(); err != nil {
				return err
			}

		case tui.DashboardViewDelete:
			if err := runDeleteTUI(); err != nil {
				return err
			}

		case tui.DashboardViewConflicts:
			if err := runConflictsTUI(); err != nil {
				return err
			}
		}
	}
}

// runDiscoverTUI runs the discover skills TUI view.
func runDiscoverTUI() error {
	// Discover skills from all platforms
	var allSkills []model.Skill
	for _, p := range model.AllPlatforms() {
		skills, err := parsePlatformSkillsWithScope(p, nil)
		if err != nil {
			// Log error but continue with other platforms
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	// Include plugin skills
	pluginSkills, err := discoverPluginSkills("", true)
	if err == nil {
		allSkills = append(allSkills, pluginSkills...)
	}

	if len(allSkills) == 0 {
		ui.Info("No skills found across any platform")
		return nil
	}

	return discoverSkillsInteractive(allSkills)
}

// runBackupsTUI runs the backup management TUI view.
func runBackupsTUI() error {
	return listBackupsInteractive("")
}

// runSyncTUI runs the sync TUI view.
func runSyncTUI() error {
	// Step 1: Pick source and target platforms
	pickerResult, err := tui.RunPlatformPicker()
	if err != nil {
		return fmt.Errorf("platform picker error: %w", err)
	}

	if pickerResult.Action == tui.PlatformPickerActionNone {
		return nil // User cancelled
	}

	sourcePlatform := pickerResult.Source
	targetPlatform := pickerResult.Target

	// Step 2: Parse skills from the source platform
	sourceSkills, err := parsePlatformSkillsWithScope(sourcePlatform, nil)
	if err != nil {
		return fmt.Errorf("failed to parse source skills: %w", err)
	}

	if len(sourceSkills) == 0 {
		ui.Info(fmt.Sprintf("No skills found in %s", sourcePlatform))
		return nil
	}

	// Step 3: Run the sync list TUI to select skills
	syncResult, err := tui.RunSyncList(sourceSkills, sourcePlatform, targetPlatform)
	if err != nil {
		return fmt.Errorf("sync list error: %w", err)
	}

	if syncResult.Action == tui.SyncActionNone {
		return nil // User cancelled
	}

	if syncResult.Action == tui.SyncActionPreview {
		// Show preview for the selected skill
		skill := syncResult.PreviewSkill
		ui.Info(fmt.Sprintf("Preview: %s", skill.Name))
		fmt.Println()
		fmt.Println(skill.Content)
		return nil
	}

	// Step 4: Perform the sync for selected skills
	if len(syncResult.SelectedSkills) == 0 {
		ui.Info("No skills selected for sync")
		return nil
	}

	// Create backup before sync
	prepareBackup(targetPlatform)

	// Perform sync
	syncer := sync.New()
	opts := sync.Options{
		Strategy: sync.StrategyOverwrite,
	}
	result, err := syncer.SyncWithSkills(syncResult.SelectedSkills, targetPlatform, opts)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Display results
	changed := result.TotalChanged()
	ui.Success(fmt.Sprintf("Synced %d skills from %s to %s", changed, sourcePlatform, targetPlatform))
	if len(result.Skipped()) > 0 {
		ui.Info(fmt.Sprintf("Skipped %d skills (already up to date)", len(result.Skipped())))
	}
	if result.HasConflicts() {
		ui.Warning(fmt.Sprintf("%d conflicts detected - use 'Resolve Conflicts' to handle them", len(result.Conflicts())))
	}

	return nil
}

// runConfigTUI runs the configuration editor TUI view.
func runConfigTUI() error {
	cfg, err := config.Load()
	if err != nil {
		ui.Warning(fmt.Sprintf("Could not load config: %v", err))
		cfg = config.Default()
	}

	result, err := tui.RunConfigList(cfg)
	if err != nil {
		return fmt.Errorf("config TUI error: %w", err)
	}

	// Handle the result
	if result.Action == tui.ConfigActionNone {
		return nil
	}

	if result.Action == tui.ConfigActionSave {
		if err := result.Config.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}
		ui.Success("Configuration saved to " + config.FilePath())
	}

	return nil
}

// runExportTUI runs the export TUI view.
func runExportTUI() error {
	// Discover skills from all platforms
	skills, err := discoverSkillsForExport("")
	if err != nil {
		return fmt.Errorf("failed to discover skills: %w", err)
	}

	if len(skills) == 0 {
		ui.Info("No skills found to export")
		return nil
	}

	result, err := tui.RunExportList(skills)
	if err != nil {
		return fmt.Errorf("export TUI error: %w", err)
	}

	// Handle the result
	if result.Action == tui.ExportActionNone {
		return nil
	}

	if result.Action == tui.ExportActionExport {
		return executeExport(result)
	}

	return nil
}

// executeExport performs the actual export based on TUI result.
func executeExport(result tui.ExportListResult) error {
	if len(result.SelectedSkills) == 0 {
		ui.Info("No skills selected for export")
		return nil
	}

	// Build export options
	opts := export.Options{
		Format:          result.Format,
		Pretty:          result.Pretty,
		IncludeMetadata: result.IncludeMetadata,
	}

	// Create exporter
	exporter := export.New(opts)

	// Write to stdout
	if err := exporter.Export(result.SelectedSkills, os.Stdout); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nExported %d skill(s) as %s\n", len(result.SelectedSkills), result.Format)
	return nil
}

// runImportTUI runs the import skills TUI view.
func runImportTUI() error {
	result, err := tui.RunImportList()
	if err != nil {
		return fmt.Errorf("import TUI error: %w", err)
	}

	// Handle the result
	if result.Action == tui.ImportActionNone {
		return nil
	}

	if result.Action == tui.ImportActionImport {
		return executeImport(result)
	}

	return nil
}

// executeImport performs the actual import based on TUI result.
func executeImport(result tui.ImportListResult) error {
	if len(result.SelectedSkills) == 0 {
		ui.Info("No skills selected for import")
		return nil
	}

	// Create synchronizer for the import operation
	syncer := sync.New()
	opts := sync.Options{
		Strategy:    sync.StrategyOverwrite,
		DryRun:      false,
		TargetScope: result.TargetScope,
	}

	// Perform the import (sync from source to target)
	syncResult, err := syncer.SyncWithSkills(result.SelectedSkills, result.TargetPlatform, opts)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	// Report results
	var imported, skipped, failed int
	for _, skill := range syncResult.Skills {
		switch skill.Action {
		case sync.ActionCreated, sync.ActionUpdated:
			imported++
		case sync.ActionSkipped:
			skipped++
		case sync.ActionFailed:
			failed++
			ui.Error(fmt.Sprintf("Failed to import %s: %s", skill.Skill.Name, skill.Error))
		}
	}

	if imported > 0 {
		ui.Success(fmt.Sprintf("Imported %d skill(s) to %s (%s)", imported, result.TargetPlatform, result.TargetScope))
	}
	if skipped > 0 {
		ui.Info(fmt.Sprintf("Skipped %d skill(s) (already up to date)", skipped))
	}
	if failed > 0 {
		ui.Warning(fmt.Sprintf("%d skill(s) failed to import", failed))
	}

	return nil
}

// runDeleteTUI runs the delete skills TUI view.
func runDeleteTUI() error {
	// Discover skills from all platforms
	var allSkills []model.Skill
	for _, p := range model.AllPlatforms() {
		skills, err := parsePlatformSkillsWithScope(p, nil)
		if err != nil {
			// Log error but continue with other platforms
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	if len(allSkills) == 0 {
		ui.Info("No skills found")
		return nil
	}

	result, err := tui.RunDeleteList(allSkills)
	if err != nil {
		return fmt.Errorf("delete TUI error: %w", err)
	}

	// Handle the result
	if result.Action == tui.DeleteActionNone {
		return nil
	}

	if result.Action == tui.DeleteActionDelete {
		return executeDelete(result)
	}

	return nil
}

// executeDelete performs the actual deletion based on TUI result.
func executeDelete(result tui.DeleteListResult) error {
	if len(result.SelectedSkills) == 0 {
		ui.Info("No skills selected for deletion")
		return nil
	}

	// Delete each selected skill
	var deleted int
	var errors []string
	for _, skill := range result.SelectedSkills {
		// Verify the skill is in a writable scope
		if skill.Scope != model.ScopeRepo && skill.Scope != model.ScopeUser {
			errors = append(errors, fmt.Sprintf("%s: scope %q is not writable", skill.Name, skill.Scope))
			continue
		}

		// Delete the skill file
		if err := os.Remove(skill.Path); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", skill.Name, err))
			continue
		}

		// Try to remove the parent directory if it's empty (for directory-based skills)
		if strings.HasSuffix(skill.Path, "/SKILL.md") {
			parentDir := skill.Path[:len(skill.Path)-len("/SKILL.md")]
			_ = os.Remove(parentDir) // Ignore error - directory may not be empty
		}

		deleted++
	}

	if deleted > 0 {
		ui.Success(fmt.Sprintf("Deleted %d skill(s)", deleted))
	}

	if len(errors) > 0 {
		for _, e := range errors {
			ui.Error(fmt.Sprintf("Failed: %s", e))
		}
		return fmt.Errorf("some deletions failed")
	}

	return nil
}

// runScopeTUI runs the scope management TUI view.
func runScopeTUI() error {
	// Discover skills from all platforms
	var allSkills []model.Skill
	for _, p := range model.AllPlatforms() {
		skills, err := parsePlatformSkillsWithScope(p, nil)
		if err != nil {
			// Log error but continue with other platforms
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	// Include plugin skills
	pluginSkills, err := discoverPluginSkills("", true)
	if err == nil {
		allSkills = append(allSkills, pluginSkills...)
	}

	if len(allSkills) == 0 {
		ui.Info("No skills found")
		return nil
	}

	result, err := tui.RunScopeList(allSkills)
	if err != nil {
		return fmt.Errorf("scope TUI error: %w", err)
	}

	// Handle the result
	if result.Action == tui.ScopeActionNone {
		return nil
	}

	if result.Action == tui.ScopeActionView {
		// Display skill details
		skill := result.SelectedSkill
		fmt.Println()
		fmt.Println(ui.Bold(fmt.Sprintf("Skill: %s", skill.Name)))
		fmt.Println(ui.Info(fmt.Sprintf("Platform:    %s", skill.Platform)))
		fmt.Println(ui.Info(fmt.Sprintf("Scope:       %s", skill.DisplayScope())))
		fmt.Println(ui.Info(fmt.Sprintf("Description: %s", skill.Description)))
		fmt.Println(ui.Info(fmt.Sprintf("Path:        %s", skill.Path)))
		if len(skill.Tools) > 0 {
			fmt.Println(ui.Info(fmt.Sprintf("Tools:       %s", strings.Join(skill.Tools, ", "))))
		}
		if len(skill.References) > 0 {
			fmt.Println(ui.Info(fmt.Sprintf("References:  %s", strings.Join(skill.References, ", "))))
		}
	}

	return nil
}

// runConflictsTUI runs the conflict resolution TUI view.
// This scans for potential conflicts across platforms and shows them for resolution.
func runConflictsTUI() error {
	// Discover skills from all platforms to find potential conflicts
	platformSkills := make(map[model.Platform][]model.Skill)
	for _, p := range model.AllPlatforms() {
		skills, err := parsePlatformSkillsWithScope(p, nil)
		if err != nil {
			continue
		}
		if len(skills) > 0 {
			platformSkills[p] = skills
		}
	}

	// Need at least 2 platforms with skills to have potential conflicts
	if len(platformSkills) < 2 {
		ui.Info("Not enough platforms with skills to check for conflicts")
		ui.Info("Skills need to exist on at least 2 platforms to detect conflicts")
		return nil
	}

	// Find skills that exist on multiple platforms with different content
	detector := sync.NewConflictDetector()
	var conflicts []*sync.Conflict

	// Build map of skill name -> skills across platforms
	skillMap := make(map[string][]model.Skill)
	for _, skills := range platformSkills {
		for _, skill := range skills {
			skillMap[skill.Name] = append(skillMap[skill.Name], skill)
		}
	}

	// Check each skill that exists on multiple platforms
	for _, skills := range skillMap {
		if len(skills) < 2 {
			continue
		}
		// Compare first skill with others
		for i := 1; i < len(skills); i++ {
			conflict := detector.DetectConflict(skills[0], skills[i])
			if conflict != nil {
				conflicts = append(conflicts, conflict)
			}
		}
	}

	if len(conflicts) == 0 {
		ui.Success("No conflicts found across platforms")
		ui.Info("All skills with the same name have identical content")
		return nil
	}

	// Run the conflict resolution TUI
	result, err := tui.RunConflictList(conflicts)
	if err != nil {
		return fmt.Errorf("conflict TUI error: %w", err)
	}

	if result.Action == tui.ConflictActionNone || result.Action == tui.ConflictActionCancel {
		return nil
	}

	// Apply resolutions
	if result.Action == tui.ConflictActionResolve {
		applied := 0
		for _, resolution := range result.Resolutions {
			if resolution.Resolution == sync.ResolutionSkip {
				continue
			}
			// Find the skills involved in this conflict
			skills := skillMap[resolution.SkillName]
			if len(skills) == 0 {
				continue
			}

			// Determine content to write based on resolution
			var content string
			switch resolution.Resolution {
			case sync.ResolutionUseSource:
				content = skills[0].Content
			case sync.ResolutionUseTarget:
				if len(skills) > 1 {
					content = skills[1].Content
				}
			case sync.ResolutionMerge:
				content = resolution.Content
			default:
				continue
			}

			// Update all instances of this skill across platforms
			for _, skill := range skills {
				if skill.Content == content {
					continue // Already has the resolved content
				}
				if skill.Path != "" {
					if err := os.WriteFile(skill.Path, []byte(content), 0o600); err != nil {
						ui.Warning(fmt.Sprintf("Failed to update %s on %s: %v", skill.Name, skill.Platform, err))
						continue
					}
					applied++
				}
			}
		}
		if applied > 0 {
			ui.Success(fmt.Sprintf("Applied %d resolution(s)", applied))
		}
	}

	return nil
}

// runCompareTUI runs the compare skills TUI view with side-by-side comparison.
func runCompareTUI() error {
	// Discover skills from all platforms
	var allSkills []model.Skill
	for _, p := range model.AllPlatforms() {
		skills, err := parsePlatformSkillsWithScope(p, nil)
		if err != nil {
			// Log error but continue with other platforms
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	if len(allSkills) < 2 {
		ui.Info("Not enough skills to compare (need at least 2)")
		return nil
	}

	// Load config for thresholds
	appConfig, err := config.Load()
	if err != nil {
		appConfig = config.Default()
	}

	// Find similar skills using default thresholds
	comparisons, err := findDuplicatesForTUI(allSkills, appConfig)
	if err != nil {
		return fmt.Errorf("failed to find similar skills: %w", err)
	}

	if len(comparisons) == 0 {
		ui.Info("No similar skills found to compare")
		return nil
	}

	result, err := tui.RunCompareList(comparisons)
	if err != nil {
		return fmt.Errorf("compare TUI error: %w", err)
	}

	// Handle the result
	if result.Action == tui.CompareActionNone {
		return nil
	}

	// For CompareActionView, the TUI already displayed the comparison
	// No additional action needed

	return nil
}

// findDuplicatesForTUI finds duplicate skill pairs using similarity matching.
func findDuplicatesForTUI(skills []model.Skill, cfg *config.Config) ([]*similarity.ComparisonResult, error) {
	var results []*similarity.ComparisonResult
	comparedPairs := make(map[string]bool)

	// Name similarity matching
	nameConfig := similarity.NameMatcherConfig{
		Threshold: cfg.Similarity.NameThreshold,
		Algorithm: cfg.Similarity.Algorithm,
		Normalize: true,
	}
	nameMatcher := similarity.NewNameMatcher(nameConfig)
	nameMatches := nameMatcher.FindSimilar(skills)

	for _, match := range nameMatches {
		pairKey := makeDupePairKey(match.Skill1, match.Skill2)
		if comparedPairs[pairKey] {
			continue
		}
		comparedPairs[pairKey] = true

		// Compute content score
		contentConfig := similarity.ContentMatcherConfig{
			Threshold: 0, // Don't filter, we want the score
			Algorithm: cfg.Similarity.Algorithm,
			LineMode:  true,
		}
		contentMatcher := similarity.NewContentMatcher(contentConfig)
		contentScore := contentMatcher.Compare(match.Skill1.Content, match.Skill2.Content)

		result := similarity.ComputeDiff(match.Skill1, match.Skill2, match.Score, contentScore)
		results = append(results, result)
	}

	// Content similarity matching
	contentConfig := similarity.ContentMatcherConfig{
		Threshold: cfg.Similarity.ContentThreshold,
		Algorithm: cfg.Similarity.Algorithm,
		LineMode:  true,
	}
	contentMatcher := similarity.NewContentMatcher(contentConfig)
	contentMatches := contentMatcher.FindSimilar(skills)

	for _, match := range contentMatches {
		pairKey := makeDupePairKey(match.Skill1, match.Skill2)
		if comparedPairs[pairKey] {
			continue
		}
		comparedPairs[pairKey] = true

		// Compute name score
		nameConfig := similarity.NameMatcherConfig{
			Threshold: 0, // Don't filter, we want the score
			Algorithm: cfg.Similarity.Algorithm,
			Normalize: true,
		}
		nameMatcher := similarity.NewNameMatcher(nameConfig)
		nameScore := nameMatcher.Compare(match.Skill1.Name, match.Skill2.Name)

		result := similarity.ComputeDiff(match.Skill1, match.Skill2, nameScore, match.Score)
		results = append(results, result)
	}

	return results, nil
}

// makeDupePairKey creates a consistent key for a skill pair regardless of order.
func makeDupePairKey(s1, s2 model.Skill) string {
	key1 := fmt.Sprintf("%s:%s:%s", s1.Platform, s1.Scope, s1.Name)
	key2 := fmt.Sprintf("%s:%s:%s", s2.Platform, s2.Scope, s2.Name)
	if key1 < key2 {
		return key1 + "|" + key2
	}
	return key2 + "|" + key1
}

// runPromoteDemoteTUI runs the promote/demote skills TUI view.
func runPromoteDemoteTUI() error {
	// Discover skills from all platforms
	var allSkills []model.Skill
	for _, p := range model.AllPlatforms() {
		skills, err := parsePlatformSkillsWithScope(p, nil)
		if err != nil {
			// Log error but continue with other platforms
			continue
		}
		allSkills = append(allSkills, skills...)
	}

	if len(allSkills) == 0 {
		ui.Info("No skills found")
		return nil
	}

	result, err := tui.RunPromoteDemoteList(allSkills)
	if err != nil {
		return fmt.Errorf("promote/demote TUI error: %w", err)
	}

	// Handle the result
	if result.Action == tui.PromoteDemoteActionNone {
		return nil
	}

	return executePromoteDemote(result)
}

// executePromoteDemote performs the actual promote/demote based on TUI result.
func executePromoteDemote(result tui.PromoteDemoteListResult) error {
	if len(result.SelectedSkills) == 0 {
		ui.Info("No skills selected")
		return nil
	}

	isPromotion := result.Action == tui.PromoteDemoteActionPromote
	operation := "Demote"
	if isPromotion {
		operation = "Promote"
	}

	var processed int
	var errors []string

	for _, skill := range result.SelectedSkills {
		// Determine source and target scopes based on operation type
		var fromScope, toScope model.SkillScope
		if isPromotion {
			// Promote: repo -> user
			if skill.Scope != model.ScopeRepo {
				continue // Skip skills that can't be promoted
			}
			fromScope = model.ScopeRepo
			toScope = model.ScopeUser
		} else {
			// Demote: user -> repo
			if skill.Scope != model.ScopeUser {
				continue // Skip skills that can't be demoted
			}
			fromScope = model.ScopeUser
			toScope = model.ScopeRepo
		}

		// Get target path
		targetPath, err := getSkillPathForScope(skill.Platform, toScope, skill.Name)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to determine target path: %v", skill.Name, err))
			continue
		}

		// Ensure target directory exists
		// #nosec G301 - skill directories need to be readable by the platform
		targetDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetDir, 0o750); err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to create target directory: %v", skill.Name, err))
			continue
		}

		// Read source content
		// #nosec G304 - skill.Path comes from parsed skill files
		content, err := os.ReadFile(skill.Path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to read source: %v", skill.Name, err))
			continue
		}

		// Write to target
		// #nosec G306 - skill files should be readable
		if err := os.WriteFile(targetPath, content, 0o644); err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to write to target: %v", skill.Name, err))
			continue
		}

		// Remove source if requested
		if result.RemoveSource {
			if err := os.Remove(skill.Path); err != nil {
				errors = append(errors, fmt.Sprintf("%s: copied but failed to remove source: %v", skill.Name, err))
				// Don't continue - the copy was successful
			}
		}

		processed++
		_ = fromScope // Used for clarity in logic above
	}

	if processed > 0 {
		modeText := "copied"
		if result.RemoveSource {
			modeText = "moved"
		}
		ui.Success(fmt.Sprintf("%sd %d skill(s) (%s)", operation, processed, modeText))
	}

	if len(errors) > 0 {
		for _, e := range errors {
			ui.Error(fmt.Sprintf("Failed: %s", e))
		}
		return fmt.Errorf("some operations failed")
	}

	return nil
}

func platformsCommand() *cli.Command {
	return &cli.Command{
		Name:    "platforms",
		Aliases: []string{"detect"},
		Usage:   "Detect installed AI coding platforms",
		UsageText: `skillsync platforms [options]
   skillsync platforms               # List all detected platforms
   skillsync platforms --format json # Output as JSON`,
		Description: `Detect which AI coding assistant platforms are installed on this system.

   Checks for: claude-code, cursor, codex

   Detection methods:
   - Environment variables (SKILLSYNC_*_PATH)
   - Default user paths (~/.claude/skills, ~/.cursor/skills, ~/.codex/skills)
   - Platform-specific indicator files
   - Project-local configurations

   Output shows detected platforms with their config paths and confidence levels.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "table",
				Usage:   "Output format: table, json, yaml",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			format := cmd.String("format")

			detected, err := detector.DetectAll()
			if err != nil {
				return fmt.Errorf("failed to detect platforms: %w", err)
			}

			if len(detected) == 0 {
				fmt.Println("No platforms detected.")
				fmt.Println("\nSupported platforms:")
				fmt.Println("  - claude-code (~/.claude/skills)")
				fmt.Println("  - cursor      (~/.cursor/skills)")
				fmt.Println("  - codex       (~/.codex/skills)")
				return nil
			}

			// Output based on format
			switch format {
			case "json":
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(detected); err != nil {
					return fmt.Errorf("failed to encode JSON: %w", err)
				}
			case "yaml":
				encoder := yaml.NewEncoder(os.Stdout)
				if err := encoder.Encode(detected); err != nil {
					return fmt.Errorf("failed to encode YAML: %w", err)
				}
			default: // table
				fmt.Printf("\nDetected platforms (%d):\n\n", len(detected))
				for _, d := range detected {
					confidenceStr := fmt.Sprintf("%.0f%%", d.Confidence*100)
					fmt.Printf("  %s  %s  %s\n",
						ui.Info(fmt.Sprintf("%-12s", d.Platform)),
						ui.Warning(fmt.Sprintf("%-10s", confidenceStr)),
						d.ConfigPath)
					fmt.Printf("    %s\n\n", ui.Dim(fmt.Sprintf("Source: %s", d.Source)))
				}
			}

			return nil
		},
	}
}
