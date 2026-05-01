package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/export"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/similarity"
	"github.com/klauern/skillsync/internal/sync"
	"github.com/klauern/skillsync/internal/ui"
	"github.com/klauern/skillsync/internal/ui/tui"
)

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
		skills, err := parsePlatformSkillsWithScope(p, nil, false)
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
	// Step 1: Pick source/target platform and scope
	pickerResult, err := tui.RunSyncPicker()
	if err != nil {
		return fmt.Errorf("sync picker error: %w", err)
	}

	if pickerResult.Action == tui.SyncPickerActionNone {
		return nil // User cancelled
	}

	sourcePlatform := pickerResult.Source
	sourceScopes := pickerResult.SourceScopes
	targetPlatform := pickerResult.Target
	targetScope := pickerResult.TargetScope

	// Step 2: Parse skills from the source platform
	sourceSkills, err := parsePlatformSkillsWithScope(sourcePlatform, sourceScopes, false)
	if err != nil {
		return fmt.Errorf("failed to parse source skills: %w", err)
	}

	if len(sourceSkills) == 0 {
		sourceScopeLabel := model.FormatSourceScopes(sourceScopes)
		ui.Info(fmt.Sprintf("No skills found in %s:%s", sourcePlatform, sourceScopeLabel))
		return nil
	}

	// Step 3: Run the sync list TUI to select skills
	syncResult, err := tui.RunSyncList(sourceSkills, sourcePlatform, targetPlatform, nil)
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
	created, err := backupExistingTargetSkills(
		targetPlatform,
		targetScope,
		syncResult.SelectedSkills,
		"pre-sync backup",
		[]string{"sync"},
	)
	if err != nil {
		return err
	}
	if created > 0 {
		fmt.Printf("✓ Created %d backup(s)\n", created)
	}

	// Perform sync
	syncer := sync.New()
	opts := sync.Options{
		Strategy:    sync.StrategyOverwrite,
		TargetScope: targetScope,
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

	// Post-sync orphan detection and cleanup
	targetSkills, err := parsePlatformSkillsWithScope(
		targetPlatform,
		[]model.SkillScope{targetScope},
		false,
	)
	if err != nil {
		ui.Warning(fmt.Sprintf("Could not check for orphaned skills: %v", err))
		return nil
	}

	orphans := findOrphanedSkills(sourceSkills, targetSkills)
	if len(orphans) == 0 {
		return nil
	}

	fmt.Printf("\nFound %d orphaned skill(s) in %s not present in %s\n", len(orphans), targetPlatform, sourcePlatform)
	confirmed, err := confirmAction(
		"Review orphaned skills for cleanup?",
		riskLevelInfo,
	)
	if err != nil || !confirmed {
		return nil
	}

	deleteResult, err := tui.RunDeleteList(orphans)
	if err != nil {
		return fmt.Errorf("delete list error: %w", err)
	}

	if deleteResult.Action == tui.DeleteActionDelete {
		return executeDelete(deleteResult)
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
		skills, err := parsePlatformSkillsWithScope(p, nil, false)
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
	var errs []string
	for _, skill := range result.SelectedSkills {
		// Verify the skill is in a writable scope
		if skill.Scope != model.ScopeRepo && skill.Scope != model.ScopeUser {
			errs = append(errs, fmt.Sprintf("%s: scope %q is not writable", skill.Name, skill.Scope))
			continue
		}

		// Delete the skill file
		if err := os.Remove(skill.Path); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", skill.Name, err))
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

	if len(errs) > 0 {
		for _, e := range errs {
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
		skills, err := parsePlatformSkillsWithScope(p, nil, false)
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
		skills, err := parsePlatformSkillsWithScope(p, nil, false)
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
		skills, err := parsePlatformSkillsWithScope(p, nil, false)
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
		skills, err := parsePlatformSkillsWithScope(p, nil, false)
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
	var errs []string

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
			errs = append(errs, fmt.Sprintf("%s: failed to determine target path: %v", skill.Name, err))
			continue
		}

		// Ensure target directory exists
		// #nosec G301 - skill directories need to be readable by the platform
		targetDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetDir, 0o750); err != nil {
			errs = append(errs, fmt.Sprintf("%s: failed to create target directory: %v", skill.Name, err))
			continue
		}

		// Read source content
		// #nosec G304 - skill.Path comes from parsed skill files
		content, err := os.ReadFile(skill.Path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: failed to read source: %v", skill.Name, err))
			continue
		}

		// Write to target
		// #nosec G306 G703 - skill files should be readable; targetPath comes from getSkillPathForScope (controlled internal function)
		if err := os.WriteFile(targetPath, content, 0o644); err != nil {
			errs = append(errs, fmt.Sprintf("%s: failed to write to target: %v", skill.Name, err))
			continue
		}

		// Remove source if requested
		if result.RemoveSource {
			if err := os.Remove(skill.Path); err != nil {
				errs = append(errs, fmt.Sprintf("%s: copied but failed to remove source: %v", skill.Name, err))
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

	if len(errs) > 0 {
		for _, e := range errs {
			ui.Error(fmt.Sprintf("Failed: %s", e))
		}
		return fmt.Errorf("some operations failed")
	}

	return nil
}
