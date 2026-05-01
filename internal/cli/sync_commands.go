package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/klauern/skillsync/internal/backup"
	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/codex"
	"github.com/klauern/skillsync/internal/parser/cursor"
	"github.com/klauern/skillsync/internal/parser/pidev"
	"github.com/klauern/skillsync/internal/parser/tiered"
	"github.com/klauern/skillsync/internal/sync"
	"github.com/klauern/skillsync/internal/util"
	"github.com/klauern/skillsync/internal/validation"
)

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

   Examples:
     skillsync sync cursor claudecode             # All cursor skills to claudecode user scope
     skillsync sync cursor:repo claudecode:user   # Repo skills to user scope
     skillsync sync cursor:repo,user codex:repo   # Multiple source scopes to repo
     skillsync tui                                # Interactive dashboard mode
     skillsync sync --dry-run cursor codex        # Preview changes
     skillsync sync --strategy=skip cursor codex
     skillsync sync --include-plugins claudecode cursor  # Include plugin skills
     skillsync sync claudecode:plugin cursor      # Sync only plugin skills
     skillsync sync --include-prompts claudecode codex   # Include prompts/commands
     skillsync sync --type prompt claudecode codex       # Prompts only
     skillsync sync --delete cursor claudecode          # Sync and remove orphaned skills from target

   See also:
     skillsync delete <source> <target>           # Remove skills from target`,
		Flags: syncFlags(),
		Action: func(_ context.Context, cmd *cli.Command) error {
			return runSyncCommand(cmd, false)
		},
	}
}

func deleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete skills from target that exist in source",
		UsageText: "skillsync delete [options] <source> <target>",
		Description: `Delete skills from the target platform that also exist in the source.

   Supported platforms: claudecode, cursor, codex

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
		return err
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
	if err := runSyncBackup(cfg); err != nil {
		return err
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
		return err
	}

	displaySyncResults(result)

	// Post-sync orphan deletion (--delete flag)
	if err := runSyncOrphanDeletion(cfg); err != nil {
		return err
	}

	if !result.Success() {
		return errors.New("sync completed with errors")
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
		return err
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
	if err := executeDeleteForSkills(deleteCfg, orphans, false); err != nil {
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
		return nil, err
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

// syncDeleteMode handles the delete sync mode: removing skills from target that exist in source.
func syncDeleteMode(cfg *syncConfig) error {
	return executeDeleteForSkills(cfg, cfg.sourceSkills, false)
}

func filterDeleteCandidates(sourceSkills, targetSkills []model.Skill) []model.Skill {
	if len(sourceSkills) == 0 || len(targetSkills) == 0 {
		return nil
	}

	sourceNames := make(map[string]bool)
	for _, skill := range sourceSkills {
		sourceNames[skill.Name] = true
	}

	candidates := make([]model.Skill, 0, len(targetSkills))
	for _, skill := range targetSkills {
		if sourceNames[skill.Name] {
			candidates = append(candidates, skill)
		}
	}

	return candidates
}

// findOrphanedSkills returns target skills whose names don't appear in the source list.
// These are candidates for deletion when --delete is used with sync.
func findOrphanedSkills(sourceSkills, targetSkills []model.Skill) []model.Skill {
	if len(targetSkills) == 0 {
		return nil
	}

	sourceNames := make(map[string]bool, len(sourceSkills))
	for _, skill := range sourceSkills {
		sourceNames[skill.Name] = true
	}

	var orphans []model.Skill
	for _, skill := range targetSkills {
		if !sourceNames[skill.Name] {
			orphans = append(orphans, skill)
		}
	}

	return orphans
}

func selectSourceSkillsForDelete(sourceSkills, selectedTargets []model.Skill) []model.Skill {
	if len(sourceSkills) == 0 || len(selectedTargets) == 0 {
		return nil
	}

	sourceByName := make(map[string]model.Skill)
	for _, skill := range sourceSkills {
		if _, exists := sourceByName[skill.Name]; !exists {
			sourceByName[skill.Name] = skill
		}
	}

	selected := make([]model.Skill, 0, len(selectedTargets))
	for _, skill := range selectedTargets {
		if sourceSkill, ok := sourceByName[skill.Name]; ok {
			selected = append(selected, sourceSkill)
		}
	}

	return selected
}

func executeDeleteForSkills(cfg *syncConfig, skills []model.Skill, confirmed bool) error {
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
			return err
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

	displaySyncResults(result)

	if !result.Success() {
		return errors.New("delete sync completed with errors")
	}

	return nil
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

func parseScopeFilter(scopeStr string) ([]model.SkillScope, error) {
	if scopeStr == "" || scopeStr == "all" {
		return nil, nil
	}

	var scopeFilter []model.SkillScope
	for _, s := range strings.Split(scopeStr, ",") {
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
	case model.PiDev:
		parser = pidev.New(basePath)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	return parser.Parse()
}

// parsePlatformSkillsWithScope parses skills from the given platform with optional scope filtering.
// If scopeFilter is nil or empty, all scopes are included. Plugin scope skills are excluded by
// default unless includePlugins is true or the plugin scope is explicitly in scopeFilter.
func parsePlatformSkillsWithScope(platform model.Platform, scopeFilter []model.SkillScope, includePlugins bool) ([]model.Skill, error) {
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

	return parsePlatformSkillsFromPaths(platform, paths, repoRoot, scopeFilter, includePlugins), nil
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
	case model.PiDev:
		rawPaths = cfg.Platforms.PiDev.SkillsPaths
		if len(rawPaths) == 0 && cfg.Platforms.PiDev.SkillsPath != "" { //nolint:staticcheck // backward compatibility
			rawPaths = []string{cfg.Platforms.PiDev.SkillsPath} //nolint:staticcheck // backward compatibility
		}
	default:
		return nil, repoRoot, fmt.Errorf("unsupported platform: %s", platform)
	}

	paths := resolveSkillsPaths(rawPaths, cwd, repoRoot)

	// Backward compatibility: older config files may only include .claude/skills
	// paths and omit .claude/commands paths. Always include command paths for
	// Claude Code so prompt artifacts can be discovered when requested.
	if platform == model.ClaudeCode {
		commandPaths := resolveSkillsPaths([]string{".claude/commands", "~/.claude/commands"}, cwd, repoRoot)
		seen := make(map[string]bool, len(paths))
		for _, p := range paths {
			seen[p] = true
		}
		for _, p := range commandPaths {
			if !seen[p] {
				paths = append(paths, p)
			}
		}
	}

	return paths, repoRoot, nil
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
	includePlugins bool,
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
			logging.Warn("failed to parse skills", logging.Err(err), logging.Path(path))
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

	// For Claude Code, include plugin cache skills if:
	// - plugin scope is explicitly requested in scopeFilter, OR
	// - includePlugins is true (--include-plugins flag)
	// Note: plugin scope is excluded by default when no scope filter is specified
	pluginExplicitlyRequested := scopeSet[model.ScopePlugin]
	if platform == model.ClaudeCode && (pluginExplicitlyRequested || includePlugins) {
		pluginSkills := parseClaudePluginCacheSkills()
		for _, skill := range pluginSkills {
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

	// Within the same scope, prefer regular skills over prompts/commands for
	// same-name collisions (matches Claude documented behavior and is safer
	// cross-platform for merged command systems).
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

// parseClaudePluginCacheSkills discovers skills from Claude Code's installed plugin cache.
func parseClaudePluginCacheSkills() []model.Skill {
	cacheParser := claude.NewCachePluginsParser("")
	skills, err := cacheParser.Parse()
	if err != nil {
		return []model.Skill{}
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

	// Check if path is within Claude plugin cache (must check before home directory)
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
			if err := checkWritePermission(parentDir); err != nil {
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
			return "", err
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("no existing parent directory found for %q", path)
		}
		cur = parent
	}
}

// checkWritePermission verifies a directory is writable
func checkWritePermission(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot access directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	testFile := filepath.Join(path, ".skillsync-write-test")
	// #nosec G304 - testFile is constructed from validated path and is not user input
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("cannot write to directory: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(testFile)
		return fmt.Errorf("failed to close write-test file: %w", err)
	}
	if err := os.Remove(testFile); err != nil {
		return fmt.Errorf("failed to remove write-test file: %w", err)
	}
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
