// Package cli provides command definitions for skillsync.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/tiered"
	"github.com/klauern/skillsync/internal/util"
)

func promoteCommand() *cli.Command {
	return &cli.Command{
		Name:  "promote",
		Usage: "Promote a skill from project scope to user scope",
		UsageText: `skillsync promote <skill-name> [options]
   skillsync promote my-skill                     # Promote from repo to user
   skillsync promote my-skill --from repo --to user
   skillsync promote my-skill --platform cursor
   skillsync promote my-skill --remove-source     # Move instead of copy`,
		Description: `Promote (copy) a skill from a lower scope to a higher scope.

   By default, promotes from repo (project-local) to user (global) scope.
   The original skill is preserved unless --remove-source is specified.

   Use --force to overwrite if a skill with the same name exists at the target scope.
   Use --rename to specify a new name if there's a conflict.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Platform to promote from (claude-code, cursor, codex)",
			},
			&cli.StringFlag{
				Name:  "from",
				Value: "repo",
				Usage: "Source scope (repo, user, admin, system)",
			},
			&cli.StringFlag{
				Name:  "to",
				Value: "user",
				Usage: "Target scope (user, admin, system)",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Overwrite if skill exists at target scope",
			},
			&cli.StringFlag{
				Name:  "rename",
				Usage: "Rename skill at target (avoids conflicts)",
			},
			&cli.BoolFlag{
				Name:  "remove-source",
				Usage: "Remove skill from source scope after promotion (move instead of copy)",
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   "Preview changes without modifying files",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			if args.Len() < 1 {
				return errors.New("skill name is required")
			}

			skillName := args.Get(0)
			return runPromote(cmd, skillName)
		},
	}
}

func demoteCommand() *cli.Command {
	return &cli.Command{
		Name:  "demote",
		Usage: "Demote a skill from user scope to project scope",
		UsageText: `skillsync demote <skill-name> [options]
   skillsync demote my-skill                     # Demote from user to repo
   skillsync demote my-skill --from user --to repo
   skillsync demote my-skill --platform cursor
   skillsync demote my-skill --remove-source     # Move instead of copy`,
		Description: `Demote (copy) a skill from a higher scope to a lower scope.

   By default, demotes from user (global) to repo (project-local) scope.
   The original skill is preserved unless --remove-source is specified.

   Use --force to overwrite if a skill with the same name exists at the target scope.
   Use --rename to specify a new name if there's a conflict.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Platform to demote from (claude-code, cursor, codex)",
			},
			&cli.StringFlag{
				Name:  "from",
				Value: "user",
				Usage: "Source scope (user, admin, system)",
			},
			&cli.StringFlag{
				Name:  "to",
				Value: "repo",
				Usage: "Target scope (repo, user, admin)",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Overwrite if skill exists at target scope",
			},
			&cli.StringFlag{
				Name:  "rename",
				Usage: "Rename skill at target (avoids conflicts)",
			},
			&cli.BoolFlag{
				Name:  "remove-source",
				Usage: "Remove skill from source scope after demotion (move instead of copy)",
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   "Preview changes without modifying files",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			if args.Len() < 1 {
				return errors.New("skill name is required")
			}

			skillName := args.Get(0)
			return runDemote(cmd, skillName)
		},
	}
}

func scopeCommand() *cli.Command {
	return &cli.Command{
		Name:  "scope",
		Usage: "Manage skill scopes and locations",
		Description: `Commands for managing skills across different scopes.

   Scopes (in precedence order, highest first):
     repo    - Repository-level skills local to a specific project
     user    - User-level skills in the user's home directory
     admin   - Administrator-defined skills
     system  - System-wide skills installed at the system level
     builtin - Built-in skills that ship with the platform

   Higher-precedence skills override lower-precedence ones with the same name.`,
		Commands: []*cli.Command{
			scopeListCommand(),
			scopePruneCommand(),
		},
	}
}

func scopeListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls", "show"},
		Usage:   "Show all locations where a skill exists",
		UsageText: `skillsync scope list <skill-name> [options]
   skillsync scope list my-skill
   skillsync scope list my-skill --platform cursor
   skillsync scope list --all                    # List all skills with their scopes`,
		Description: `Show all scope locations where a skill with the given name exists.

   This helps identify which scopes have a skill and which version takes precedence.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Filter by platform (claude-code, cursor, codex)",
			},
			&cli.BoolFlag{
				Name:  "all",
				Usage: "List all skills grouped by scope instead of searching for a specific skill",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "table",
				Usage:   "Output format: table, json, yaml",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.Bool("all") {
				return runScopeListAll(cmd)
			}

			args := cmd.Args()
			if args.Len() < 1 {
				return errors.New("skill name is required (or use --all to list all skills)")
			}

			skillName := args.Get(0)
			return runScopeList(cmd, skillName)
		},
	}
}

func scopePruneCommand() *cli.Command {
	return &cli.Command{
		Name:  "prune",
		Usage: "Remove duplicate skills from a scope",
		UsageText: `skillsync scope prune [options]
   skillsync scope prune --scope user            # Remove user skills that exist in repo
   skillsync scope prune --scope user --keep-repo
   skillsync scope prune --platform cursor --scope user
   skillsync scope prune --dry-run               # Preview what would be removed`,
		Description: `Remove skills from a scope that are duplicated in a higher-precedence scope.

   This is useful for cleaning up user-level skills that are now defined at the
   project level, or removing redundant copies after demotion.

   Use --keep-repo to never remove repo-level skills (they always win).
   Use --keep-user to never remove user-level skills (useful when pruning admin/system).`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "platform",
				Aliases: []string{"p"},
				Usage:   "Platform to prune (claude-code, cursor, codex). Required.",
			},
			&cli.StringFlag{
				Name:  "scope",
				Usage: "Scope to prune duplicates from (user, admin, system). Required.",
			},
			&cli.BoolFlag{
				Name:  "keep-repo",
				Usage: "Never remove repo-level skills",
			},
			&cli.BoolFlag{
				Name:  "keep-user",
				Usage: "Never remove user-level skills",
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   "Preview what would be removed without making changes",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation prompt",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return runScopePrune(cmd)
		},
	}
}

// runPromote executes the promote command.
func runPromote(cmd *cli.Command, skillName string) error {
	return runScopeMove(cmd, skillName, true)
}

// runDemote executes the demote command.
func runDemote(cmd *cli.Command, skillName string) error {
	return runScopeMove(cmd, skillName, false)
}

// runScopeMove handles both promote and demote operations.
func runScopeMove(cmd *cli.Command, skillName string, isPromotion bool) error {
	platformStr := cmd.String("platform")
	fromStr := cmd.String("from")
	toStr := cmd.String("to")
	force := cmd.Bool("force")
	rename := cmd.String("rename")
	removeSource := cmd.Bool("remove-source")
	dryRun := cmd.Bool("dry-run")

	// Parse scopes
	fromScope, err := model.ParseScope(fromStr)
	if err != nil {
		return fmt.Errorf("invalid source scope: %w", err)
	}

	toScope, err := model.ParseScope(toStr)
	if err != nil {
		return fmt.Errorf("invalid target scope: %w", err)
	}

	// Validate scope direction
	if isPromotion {
		if !toScope.IsHigherPrecedence(fromScope) && toScope != fromScope {
			return fmt.Errorf("promotion requires target scope (%s) to have higher precedence than source scope (%s)", toScope, fromScope)
		}
	} else {
		if !fromScope.IsHigherPrecedence(toScope) && fromScope != toScope {
			return fmt.Errorf("demotion requires source scope (%s) to have higher precedence than target scope (%s)", fromScope, toScope)
		}
	}

	// Validate writable scopes
	writableScopes := map[model.SkillScope]bool{
		model.ScopeRepo: true,
		model.ScopeUser: true,
	}
	if !writableScopes[toScope] {
		return fmt.Errorf("target scope %q is not writable (only repo and user are supported)", toScope)
	}

	// Get platforms to process
	var platforms []model.Platform
	if platformStr != "" {
		p, err := model.ParsePlatform(platformStr)
		if err != nil {
			return fmt.Errorf("invalid platform: %w", err)
		}
		platforms = []model.Platform{p}
	} else {
		platforms = model.AllPlatforms()
	}

	// Find and process the skill
	for _, platform := range platforms {
		skill, err := findSkillInScope(platform, skillName, fromScope)
		if err != nil {
			continue // Try next platform
		}
		if skill == nil {
			continue
		}

		// Determine target name
		targetName := skillName
		if rename != "" {
			targetName = rename
		}

		// Check if skill exists at target scope
		existingSkill, _ := findSkillInScope(platform, targetName, toScope)
		if existingSkill != nil && !force && rename == "" {
			return fmt.Errorf("skill %q already exists at %s scope (use --force to overwrite or --rename to use a different name)", targetName, toScope)
		}

		// Get target path
		targetPath, err := getSkillPathForScope(platform, toScope, targetName)
		if err != nil {
			return fmt.Errorf("failed to determine target path: %w", err)
		}

		// Display operation details
		operation := "Promote"
		if !isPromotion {
			operation = "Demote"
		}

		fmt.Printf("\n%s Details:\n", operation)
		fmt.Printf("  Platform:     %s\n", platform)
		fmt.Printf("  Skill:        %s\n", skillName)
		fmt.Printf("  Source:       %s (%s)\n", skill.Path, fromScope)
		fmt.Printf("  Target:       %s (%s)\n", targetPath, toScope)
		if rename != "" {
			fmt.Printf("  Renamed to:   %s\n", targetName)
		}
		if removeSource {
			fmt.Println("  Remove source: yes")
		}

		if dryRun {
			fmt.Println("\n[Dry run - no changes made]")
			return nil
		}

		// Ensure target directory exists
		// #nosec G301 - skill directories need to be readable by the platform
		targetDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetDir, 0o750); err != nil {
			return fmt.Errorf("failed to create target directory: %w", err)
		}

		// Read source content
		// #nosec G304 - skill.Path comes from parsed skill files
		content, err := os.ReadFile(skill.Path)
		if err != nil {
			return fmt.Errorf("failed to read source skill: %w", err)
		}

		// Write to target
		// #nosec G306 - skill files should be readable
		if err := os.WriteFile(targetPath, content, 0o644); err != nil {
			return fmt.Errorf("failed to write skill to target: %w", err)
		}

		fmt.Printf("\n✓ Copied skill to %s\n", targetPath)

		// Remove source if requested
		if removeSource {
			if err := os.Remove(skill.Path); err != nil {
				return fmt.Errorf("failed to remove source skill: %w", err)
			}
			fmt.Printf("✓ Removed source skill from %s\n", skill.Path)
		}

		return nil
	}

	// No skill found in any platform
	scopeList := string(fromScope)
	if platformStr != "" {
		return fmt.Errorf("skill %q not found in %s scope for platform %s", skillName, scopeList, platformStr)
	}
	return fmt.Errorf("skill %q not found in %s scope across any platform", skillName, scopeList)
}

// runScopeList shows all locations where a skill exists.
func runScopeList(cmd *cli.Command, skillName string) error {
	platformStr := cmd.String("platform")
	format := cmd.String("format")

	var platforms []model.Platform
	if platformStr != "" {
		p, err := model.ParsePlatform(platformStr)
		if err != nil {
			return fmt.Errorf("invalid platform: %w", err)
		}
		platforms = []model.Platform{p}
	} else {
		platforms = model.AllPlatforms()
	}

	type skillLocation struct {
		Platform model.Platform   `json:"platform" yaml:"platform"`
		Scope    model.SkillScope `json:"scope" yaml:"scope"`
		Path     string           `json:"path" yaml:"path"`
		Active   bool             `json:"active" yaml:"active"` // True if this is the version that takes precedence
	}

	var locations []skillLocation

	for _, platform := range platforms {
		// Get all skills across all scopes
		tieredParser, err := tiered.NewForPlatform(platform)
		if err != nil {
			continue
		}

		// Check each scope
		for _, scope := range model.AllScopes() {
			skills, err := tieredParser.ParseFromScope(scope)
			if err != nil {
				continue
			}

			for _, skill := range skills {
				if skill.Name == skillName {
					locations = append(locations, skillLocation{
						Platform: platform,
						Scope:    scope,
						Path:     skill.Path,
						Active:   false, // Will be updated below
					})
				}
			}
		}
	}

	if len(locations) == 0 {
		fmt.Printf("Skill %q not found in any scope.\n", skillName)
		return nil
	}

	// Mark the highest-precedence location as active for each platform
	platformActive := make(map[model.Platform]bool)
	for i := range locations {
		loc := &locations[i]
		if !platformActive[loc.Platform] {
			// First occurrence for this platform has highest precedence
			// (since we iterate scopes in precedence order)
			loc.Active = true
			platformActive[loc.Platform] = true
		}
	}

	// Re-sort to show active first by iterating in reverse precedence order
	// and marking based on highest precedence
	platformHighest := make(map[model.Platform]int)
	for i, loc := range locations {
		if existing, ok := platformHighest[loc.Platform]; ok {
			if loc.Scope.Precedence() > locations[existing].Scope.Precedence() {
				locations[existing].Active = false
				locations[i].Active = true
				platformHighest[loc.Platform] = i
			} else {
				locations[i].Active = false
			}
		} else {
			locations[i].Active = true
			platformHighest[loc.Platform] = i
		}
	}

	switch format {
	case "json":
		return outputAnyJSON(locations)
	case "yaml":
		return outputAnyYAML(locations)
	case "table":
		fmt.Printf("\nLocations for skill %q:\n\n", skillName)
		fmt.Printf("%-12s %-8s %-6s %s\n", "PLATFORM", "SCOPE", "ACTIVE", "PATH")
		fmt.Printf("%-12s %-8s %-6s %s\n", "--------", "-----", "------", "----")

		for _, loc := range locations {
			activeStr := ""
			if loc.Active {
				activeStr = "✓"
			}
			fmt.Printf("%-12s %-8s %-6s %s\n", loc.Platform, loc.Scope, activeStr, loc.Path)
		}
		fmt.Printf("\nFound %d location(s)\n", len(locations))
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	return nil
}

// runScopeListAll lists all skills grouped by scope.
func runScopeListAll(cmd *cli.Command) error {
	platformStr := cmd.String("platform")
	format := cmd.String("format")

	var platforms []model.Platform
	if platformStr != "" {
		p, err := model.ParsePlatform(platformStr)
		if err != nil {
			return fmt.Errorf("invalid platform: %w", err)
		}
		platforms = []model.Platform{p}
	} else {
		platforms = model.AllPlatforms()
	}

	type scopedSkill struct {
		Platform model.Platform   `json:"platform" yaml:"platform"`
		Scope    model.SkillScope `json:"scope" yaml:"scope"`
		Name     string           `json:"name" yaml:"name"`
		Path     string           `json:"path" yaml:"path"`
	}

	var allSkills []scopedSkill

	for _, platform := range platforms {
		tieredParser, err := tiered.NewForPlatform(platform)
		if err != nil {
			continue
		}

		for _, scope := range model.AllScopes() {
			skills, err := tieredParser.ParseFromScope(scope)
			if err != nil {
				continue
			}

			for _, skill := range skills {
				allSkills = append(allSkills, scopedSkill{
					Platform: platform,
					Scope:    scope,
					Name:     skill.Name,
					Path:     skill.Path,
				})
			}
		}
	}

	if len(allSkills) == 0 {
		fmt.Println("No skills found.")
		return nil
	}

	switch format {
	case "json":
		return outputAnyJSON(allSkills)
	case "yaml":
		return outputAnyYAML(allSkills)
	case "table":
		fmt.Printf("%-12s %-8s %-25s %s\n", "PLATFORM", "SCOPE", "NAME", "PATH")
		fmt.Printf("%-12s %-8s %-25s %s\n", "--------", "-----", "----", "----")

		for _, s := range allSkills {
			name := s.Name
			if len(name) > 25 {
				name = name[:22] + "..."
			}
			fmt.Printf("%-12s %-8s %-25s %s\n", s.Platform, s.Scope, name, s.Path)
		}
		fmt.Printf("\nTotal: %d skill(s)\n", len(allSkills))
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	return nil
}

// runScopePrune removes duplicate skills from a scope.
func runScopePrune(cmd *cli.Command) error {
	platformStr := cmd.String("platform")
	scopeStr := cmd.String("scope")
	keepRepo := cmd.Bool("keep-repo")
	keepUser := cmd.Bool("keep-user")
	dryRun := cmd.Bool("dry-run")
	force := cmd.Bool("force")

	if platformStr == "" {
		return errors.New("--platform is required")
	}

	if scopeStr == "" {
		return errors.New("--scope is required")
	}

	platform, err := model.ParsePlatform(platformStr)
	if err != nil {
		return fmt.Errorf("invalid platform: %w", err)
	}

	scopeToPrune, err := model.ParseScope(scopeStr)
	if err != nil {
		return fmt.Errorf("invalid scope: %w", err)
	}

	// Validate scope is prunable
	if scopeToPrune == model.ScopeBuiltin {
		return errors.New("cannot prune builtin scope")
	}

	// Check keep flags
	if keepRepo && scopeToPrune == model.ScopeRepo {
		return errors.New("cannot use --keep-repo when pruning repo scope")
	}
	if keepUser && scopeToPrune == model.ScopeUser {
		return errors.New("cannot use --keep-user when pruning user scope")
	}

	// Get tiered parser
	tieredParser, err := tiered.NewForPlatform(platform)
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}

	// Get all skills with deduplication to find which ones have higher-precedence duplicates
	allSkills, err := tieredParser.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse skills: %w", err)
	}

	// Build map of active skills (highest precedence wins)
	activeSkills := make(map[string]model.Skill)
	for _, skill := range allSkills {
		activeSkills[skill.Name] = skill
	}

	// Get skills from the scope we want to prune
	scopeSkills, err := tieredParser.ParseFromScope(scopeToPrune)
	if err != nil {
		return fmt.Errorf("failed to parse skills from %s scope: %w", scopeToPrune, err)
	}

	// Find skills to prune (those that are shadowed by higher-precedence skills)
	var toPrune []model.Skill
	for _, skill := range scopeSkills {
		active, exists := activeSkills[skill.Name]
		if !exists {
			continue // Should not happen
		}

		// If the active skill is at a higher precedence scope, this one is a duplicate
		if active.Scope.IsHigherPrecedence(scopeToPrune) {
			// Check keep flags
			if keepRepo && active.Scope == model.ScopeRepo {
				continue // Don't prune if higher-precedence is repo and --keep-repo is set
			}
			if keepUser && active.Scope == model.ScopeUser {
				continue
			}
			toPrune = append(toPrune, skill)
		}
	}

	if len(toPrune) == 0 {
		fmt.Printf("No duplicate skills found in %s scope for %s.\n", scopeToPrune, platform)
		return nil
	}

	// Display what will be pruned
	fmt.Printf("\nSkills to prune from %s scope (%d):\n", scopeToPrune, len(toPrune))
	for _, skill := range toPrune {
		active := activeSkills[skill.Name]
		fmt.Printf("  - %s\n", skill.Name)
		fmt.Printf("    Remove: %s\n", skill.Path)
		fmt.Printf("    Keeping: %s (%s scope)\n", active.Path, active.Scope)
	}

	if dryRun {
		fmt.Println("\n[Dry run - no changes made]")
		return nil
	}

	// Confirm unless force flag is set
	if !force {
		message := fmt.Sprintf("Remove %d duplicate skill(s)?", len(toPrune))
		confirmed, err := confirmAction(message, riskLevelWarning)
		if err != nil {
			return fmt.Errorf("confirmation error: %w", err)
		}
		if !confirmed {
			fmt.Println("Prune cancelled.")
			return nil
		}
	}

	// Delete the duplicate skills
	var deleted int
	for _, skill := range toPrune {
		if err := os.Remove(skill.Path); err != nil {
			fmt.Printf("Warning: failed to remove %s: %v\n", skill.Path, err)
			continue
		}
		deleted++
	}

	fmt.Printf("\n✓ Removed %d duplicate skill(s) from %s scope\n", deleted, scopeToPrune)
	return nil
}

// findSkillInScope finds a skill by name in a specific scope.
func findSkillInScope(platform model.Platform, skillName string, scope model.SkillScope) (*model.Skill, error) {
	tieredParser, err := tiered.NewForPlatform(platform)
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}

	skills, err := tieredParser.ParseFromScope(scope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse skills from %s scope: %w", scope, err)
	}

	for _, skill := range skills {
		if skill.Name == skillName {
			return &skill, nil
		}
	}

	return nil, nil
}

// getSkillPathForScope returns the path where a skill should be written for a given scope.
func getSkillPathForScope(platform model.Platform, scope model.SkillScope, skillName string) (string, error) {
	var basePath string

	switch scope {
	case model.ScopeRepo:
		// Use current working directory
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
		basePath = util.RepoSkillsPath(platform, wd)
	case model.ScopeUser:
		basePath = util.PlatformSkillsPath(platform)
	default:
		return "", fmt.Errorf("scope %q is not writable", scope)
	}

	// Construct the skill file path
	// Skills can be either:
	// 1. A directory with SKILL.md: basePath/skill-name/SKILL.md
	// 2. A single .md file: basePath/skill-name.md
	// We'll use the directory format for consistency with Agent Skills Standard
	return filepath.Join(basePath, skillName, "SKILL.md"), nil
}

// outputAnyJSON outputs any value as JSON.
func outputAnyJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

// outputAnyYAML outputs any value as YAML.
func outputAnyYAML(v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	fmt.Print(string(data))
	return nil
}
