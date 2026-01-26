// Package cli provides command definitions for skillsync.
package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/backup"
	"github.com/klauern/skillsync/internal/cache"
	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/export"
	"github.com/klauern/skillsync/internal/model"
	parsercli "github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/codex"
	"github.com/klauern/skillsync/internal/parser/cursor"
	"github.com/klauern/skillsync/internal/parser/plugin"
	"github.com/klauern/skillsync/internal/sync"
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
	fmt.Printf("  Claude Code:     %s\n", cfg.Platforms.ClaudeCode.SkillsPath)
	fmt.Printf("  Cursor:          %s\n", cfg.Platforms.Cursor.SkillsPath)
	fmt.Printf("  Codex:           %s\n", cfg.Platforms.Codex.SkillsPath)

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
   skillsync discover --platform claude-code
   skillsync discover --plugins
   skillsync discover --plugins --repo https://github.com/user/plugins
   skillsync discover --format json`,
		Description: `Discover and list skills from all supported AI coding platforms.

   Supported platforms: claude-code, cursor, codex

   Plugin discovery: Use --plugins to scan installed Claude Code plugins
   from ~/.skillsync/plugins/ or specify a Git repository with --repo.

   Output formats: table (default), json, yaml`,
		Flags: []cli.Flag{
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
			&cli.BoolFlag{
				Name:  "plugins",
				Usage: "Include skills from installed Claude Code plugins",
			},
			&cli.StringFlag{
				Name:  "repo",
				Usage: "Git repository URL to discover plugins from (implies --plugins)",
			},
			&cli.BoolFlag{
				Name:  "no-cache",
				Usage: "Disable plugin skill caching",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			platform := cmd.String("platform")
			format := cmd.String("format")
			includePlugins := cmd.Bool("plugins")
			repoURL := cmd.String("repo")
			noCache := cmd.Bool("no-cache")

			// --repo implies --plugins
			if repoURL != "" {
				includePlugins = true
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
				skills, err := parsePlatformSkills(p)
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
			return outputSkills(allSkills, format)
		},
	}
}

// discoverPluginSkills discovers skills from Claude Code plugins with optional caching
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

	// Parse plugins
	skills, err := pluginParser.Parse()
	if err != nil {
		return nil, err
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

// outputTable prints skills in a table format
func outputTable(skills []model.Skill) error {
	if len(skills) == 0 {
		fmt.Println("No skills found.")
		return nil
	}

	fmt.Printf("%-25s %-12s %-50s\n", "NAME", "PLATFORM", "DESCRIPTION")
	fmt.Printf("%-25s %-12s %-50s\n", "----", "--------", "-----------")

	for _, skill := range skills {
		name := skill.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}

		desc := skill.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		fmt.Printf("%-25s %-12s %-50s\n", name, skill.Platform, desc)
	}

	fmt.Printf("\nTotal: %d skill(s)\n", len(skills))
	return nil
}

func syncCommand() *cli.Command {
	return &cli.Command{
		Name:      "sync",
		Usage:     "Synchronize skills across platforms",
		UsageText: "skillsync sync [options] <source> <target>",
		Description: `Synchronize skills between AI coding platforms.

   Supported platforms: claudecode, cursor, codex

   Strategies:
     overwrite   - Replace target skills unconditionally (default)
     skip        - Skip skills that already exist in target
     newer       - Copy only if source is newer than target
     merge       - Merge source and target content
     three-way   - Intelligent merge with conflict detection
     interactive - Prompt for each conflict

   Examples:
     skillsync sync claudecode cursor
     skillsync sync --dry-run cursor codex
     skillsync sync --strategy=skip claude-code cursor`,
		Flags: []cli.Flag{
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
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			cfg, err := parseSyncConfig(cmd)
			if err != nil {
				return err
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
				prepareBackup(cfg.targetPlatform)
			}

			// Create sync options and execute
			opts := sync.Options{
				DryRun:   cfg.dryRun,
				Strategy: cfg.strategy,
			}

			syncer := sync.New()
			result, err := syncer.Sync(cfg.sourcePlatform, cfg.targetPlatform, opts)
			if err != nil {
				return fmt.Errorf("sync failed: %w", err)
			}

			// Handle conflicts if interactive strategy is used
			if result.HasConflicts() && cfg.strategy == sync.StrategyInteractive {
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
	sourcePlatform model.Platform
	targetPlatform model.Platform
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

	sourcePlatform, err := model.ParsePlatform(args.Get(0))
	if err != nil {
		return nil, fmt.Errorf("invalid source platform: %w", err)
	}

	targetPlatform, err := model.ParsePlatform(args.Get(1))
	if err != nil {
		return nil, fmt.Errorf("invalid target platform: %w", err)
	}

	strategyStr := cmd.String("strategy")
	strategy := sync.Strategy(strategyStr)
	if !strategy.IsValid() {
		return nil, fmt.Errorf("invalid strategy %q (valid: overwrite, skip, newer, merge, three-way, interactive)", strategyStr)
	}

	return &syncConfig{
		sourcePlatform: sourcePlatform,
		targetPlatform: targetPlatform,
		dryRun:         cmd.Bool("dry-run"),
		strategy:       strategy,
		skipBackup:     cmd.Bool("skip-backup"),
		skipValidation: cmd.Bool("skip-validation"),
		yesFlag:        cmd.Bool("yes"),
		sourceSkills:   make([]model.Skill, 0),
	}, nil
}

// validateSourceSkills validates source skills and returns them
func validateSourceSkills(cfg *syncConfig) error {
	fmt.Println("Validating source skills...")

	var err error
	cfg.sourceSkills, err = parsePlatformSkills(cfg.sourcePlatform)
	if err != nil {
		return fmt.Errorf("failed to parse source skills: %w", err)
	}

	// Validate skill formats
	formatResult, err := validation.ValidateSkillsFormat(cfg.sourceSkills, cfg.sourcePlatform)
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

	// Validate paths and permissions
	if err := validateSyncPaths(cfg.sourcePlatform, cfg.targetPlatform); err != nil {
		return err
	}

	fmt.Println("Validation passed")
	return nil
}

// showSyncSummaryAndConfirm shows sync summary and requests user confirmation
func showSyncSummaryAndConfirm(cfg *syncConfig) (bool, error) {
	fmt.Printf("\n=== Sync Summary ===\n")
	fmt.Printf("Source: %s\n", cfg.sourcePlatform)
	fmt.Printf("Target: %s\n", cfg.targetPlatform)
	fmt.Printf("Strategy: %s (%s)\n", cfg.strategy, cfg.strategy.Description())

	if len(cfg.sourceSkills) > 0 {
		fmt.Printf("Skills to sync: %d\n", len(cfg.sourceSkills))
		for i, skill := range cfg.sourceSkills {
			fmt.Printf("  %d. %s\n", i+1, skill.Name)
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

// parsePlatformSkills parses skills from the given platform
func parsePlatformSkills(platform model.Platform) ([]model.Skill, error) {
	var parserInstance parsercli.Parser

	switch platform {
	case model.ClaudeCode:
		parserInstance = claude.New("")
	case model.Cursor:
		parserInstance = cursor.New("")
	case model.Codex:
		parserInstance = codex.New("")
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	skills, err := parserInstance.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse skills from %s: %w", platform, err)
	}

	return skills, nil
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

// validateSyncPaths validates source and target paths before sync
func validateSyncPaths(sourcePlatform, targetPlatform model.Platform) error {
	// Validate source path exists
	sourcePath, err := validation.GetPlatformPath(sourcePlatform)
	if err != nil {
		return fmt.Errorf("source path error: %w", err)
	}

	if err := validation.ValidatePath(sourcePath, sourcePlatform); err != nil {
		return fmt.Errorf("source validation failed: %w", err)
	}

	// Validate target path (or parent if it doesn't exist)
	targetPath, err := validation.GetPlatformPath(targetPlatform)
	if err != nil {
		return fmt.Errorf("target path error: %w", err)
	}

	if err := validation.ValidatePath(targetPath, targetPlatform); err != nil {
		var vErr *validation.Error
		if errors.As(err, &vErr) && vErr.Message == "path does not exist" {
			// Target doesn't exist - validate parent directory is writable
			parentDir := targetPath[:len(targetPath)-1]
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
		var parser parsercli.Parser
		switch p {
		case model.ClaudeCode:
			parser = claude.New("")
		case model.Cursor:
			parser = cursor.New("")
		case model.Codex:
			parser = codex.New("")
		default:
			continue
		}

		skills, err := parser.Parse()
		if err != nil {
			// Log warning but continue with other platforms
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", p, err)
			continue
		}

		allSkills = append(allSkills, skills...)
	}

	return allSkills, nil
}
