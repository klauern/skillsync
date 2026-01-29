// Package cli provides command definitions for skillsync.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/klauern/skillsync/internal/model"
)

func dedupeCommand() *cli.Command {
	return &cli.Command{
		Name:    "dedupe",
		Aliases: []string{"dedup", "cleanup"},
		Usage:   "Remediate duplicate or similar skills",
		UsageText: `skillsync dedupe <subcommand> [options]
   skillsync dedupe delete my-skill --platform cursor --scope user
   skillsync dedupe rename my-skill new-name --platform cursor --scope repo`,
		Description: `Commands for cleaning up duplicate or similar skills found by the compare command.

   Typical workflow:
   1. Run 'skillsync compare' to find similar skills
   2. Review the output and decide which duplicates to remove or rename
   3. Use 'skillsync dedupe delete' to remove exact duplicates
   4. Use 'skillsync dedupe rename' to differentiate similar but distinct skills

   Subcommands:
     delete  - Remove a duplicate skill from a specific platform/scope
     rename  - Rename a skill to differentiate it from similar skills

   Both commands require explicit --platform and --scope flags to prevent
   accidental deletions or renames.

   Examples:
     # Preview deletion
     skillsync dedupe delete my-skill -p cursor -s user --dry-run

     # Delete a duplicate (with confirmation)
     skillsync dedupe delete my-skill -p cursor -s user

     # Force delete without confirmation
     skillsync dedupe delete my-skill -p cursor -s user --force

     # Rename to differentiate
     skillsync dedupe rename my-skill project-my-skill -p cursor -s repo

     # Preview rename
     skillsync dedupe rename my-skill v2-my-skill -p cursor -s user --dry-run`,
		Commands: []*cli.Command{
			dedupeDeleteCommand(),
			dedupeRenameCommand(),
		},
	}
}

func dedupeDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a duplicate skill",
		UsageText: `skillsync dedupe delete <skill-name> [options]
   skillsync dedupe delete my-skill --platform claude-code --scope user
   skillsync dedupe delete my-skill --platform cursor --scope repo --force`,
		Description: `Delete a skill from a specific platform and scope.

   This command is useful for removing duplicate skills identified by the compare command.

   By default, you must specify the platform and scope to avoid accidental deletions.
   Use --force to skip the confirmation prompt.
   Use --dry-run to preview what would be deleted without making changes.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "platform",
				Aliases:  []string{"p"},
				Usage:    "Platform where the skill exists (claude-code, cursor, codex). Required.",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "scope",
				Aliases:  []string{"s"},
				Usage:    "Scope where the skill exists (repo, user). Required.",
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation prompt",
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   "Preview what would be deleted without making changes",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			if args.Len() < 1 {
				return errors.New("skill name is required")
			}
			skillName := args.Get(0)
			return runDedupeDelete(cmd, skillName)
		},
	}
}

func dedupeRenameCommand() *cli.Command {
	return &cli.Command{
		Name:  "rename",
		Usage: "Rename a skill to differentiate it",
		UsageText: `skillsync dedupe rename <old-name> <new-name> [options]
   skillsync dedupe rename my-skill my-skill-v2 --platform claude-code --scope user
   skillsync dedupe rename my-skill project-my-skill --platform cursor --scope repo`,
		Description: `Rename a skill to a new name within the same platform and scope.

   This command is useful for differentiating similar skills identified by the compare command.
   Instead of deleting duplicates, you can rename them to make their purpose clearer.

   By default, you must specify the platform and scope to avoid accidental renames.
   Use --force to skip the confirmation prompt.
   Use --dry-run to preview what would be renamed without making changes.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "platform",
				Aliases:  []string{"p"},
				Usage:    "Platform where the skill exists (claude-code, cursor, codex). Required.",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "scope",
				Aliases:  []string{"s"},
				Usage:    "Scope where the skill exists (repo, user). Required.",
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation prompt",
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   "Preview what would be renamed without making changes",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			if args.Len() < 2 {
				return errors.New("both old skill name and new skill name are required")
			}
			oldName := args.Get(0)
			newName := args.Get(1)
			return runDedupeRename(cmd, oldName, newName)
		},
	}
}

// runDedupeDelete executes the dedupe delete command.
func runDedupeDelete(cmd *cli.Command, skillName string) error {
	platformStr := cmd.String("platform")
	scopeStr := cmd.String("scope")
	force := cmd.Bool("force")
	dryRun := cmd.Bool("dry-run")

	// Parse platform
	platform, err := model.ParsePlatform(platformStr)
	if err != nil {
		return fmt.Errorf("invalid platform: %w", err)
	}

	// Parse and validate scope
	scope, err := model.ParseScope(scopeStr)
	if err != nil {
		return fmt.Errorf("invalid scope: %w", err)
	}

	// Validate writable scope
	if scope != model.ScopeRepo && scope != model.ScopeUser {
		return fmt.Errorf("scope %q is not writable (only repo and user are supported)", scope)
	}

	// Find the skill
	skill, err := findSkillInScope(platform, skillName, scope)
	if err != nil {
		return fmt.Errorf("failed to find skill: %w", err)
	}
	if skill == nil {
		return fmt.Errorf("skill %q not found in %s scope for platform %s", skillName, scope, platform)
	}

	// Display what will be deleted
	fmt.Printf("\nSkill to delete:\n")
	fmt.Printf("  Name:     %s\n", skill.Name)
	fmt.Printf("  Platform: %s\n", platform)
	fmt.Printf("  Scope:    %s\n", scope)
	fmt.Printf("  Path:     %s\n", skill.Path)

	if dryRun {
		fmt.Println("\n[Dry run - no changes made]")
		return nil
	}

	// Confirm unless force flag is set
	if !force {
		message := fmt.Sprintf("Delete skill %q from %s scope?", skillName, scope)
		confirmed, err := confirmAction(message, riskLevelDangerous)
		if err != nil {
			return fmt.Errorf("confirmation error: %w", err)
		}
		if !confirmed {
			fmt.Println("Delete cancelled.")
			return nil
		}
	}

	// Delete the skill file
	if err := os.Remove(skill.Path); err != nil {
		return fmt.Errorf("failed to delete skill: %w", err)
	}

	// Try to remove the parent directory if it's empty (for directory-based skills)
	parentDir := skill.Path[:len(skill.Path)-len("/SKILL.md")]
	if parentDir != skill.Path {
		// Only try if it looks like a skill directory
		_ = os.Remove(parentDir) // Ignore error - directory may not be empty or may not exist
	}

	fmt.Printf("\n✓ Deleted skill %q from %s scope\n", skillName, scope)
	return nil
}

// runDedupeRename executes the dedupe rename command.
func runDedupeRename(cmd *cli.Command, oldName, newName string) error {
	platformStr := cmd.String("platform")
	scopeStr := cmd.String("scope")
	force := cmd.Bool("force")
	dryRun := cmd.Bool("dry-run")

	// Parse platform
	platform, err := model.ParsePlatform(platformStr)
	if err != nil {
		return fmt.Errorf("invalid platform: %w", err)
	}

	// Parse and validate scope
	scope, err := model.ParseScope(scopeStr)
	if err != nil {
		return fmt.Errorf("invalid scope: %w", err)
	}

	// Validate writable scope
	if scope != model.ScopeRepo && scope != model.ScopeUser {
		return fmt.Errorf("scope %q is not writable (only repo and user are supported)", scope)
	}

	// Validate new name is different
	if oldName == newName {
		return errors.New("new name must be different from old name")
	}

	// Find the source skill
	skill, err := findSkillInScope(platform, oldName, scope)
	if err != nil {
		return fmt.Errorf("failed to find skill: %w", err)
	}
	if skill == nil {
		return fmt.Errorf("skill %q not found in %s scope for platform %s", oldName, scope, platform)
	}

	// Check if new name already exists
	existingSkill, _ := findSkillInScope(platform, newName, scope)
	if existingSkill != nil && !force {
		return fmt.Errorf("skill %q already exists in %s scope (use --force to overwrite)", newName, scope)
	}

	// Get target path for new name
	targetPath, err := getSkillPathForScope(platform, scope, newName)
	if err != nil {
		return fmt.Errorf("failed to determine target path: %w", err)
	}

	// Display what will be renamed
	fmt.Printf("\nSkill to rename:\n")
	fmt.Printf("  Platform:  %s\n", platform)
	fmt.Printf("  Scope:     %s\n", scope)
	fmt.Printf("  Old name:  %s\n", oldName)
	fmt.Printf("  New name:  %s\n", newName)
	fmt.Printf("  Old path:  %s\n", skill.Path)
	fmt.Printf("  New path:  %s\n", targetPath)

	if dryRun {
		fmt.Println("\n[Dry run - no changes made]")
		return nil
	}

	// Confirm unless force flag is set
	if !force {
		message := fmt.Sprintf("Rename skill %q to %q?", oldName, newName)
		confirmed, err := confirmAction(message, riskLevelWarning)
		if err != nil {
			return fmt.Errorf("confirmation error: %w", err)
		}
		if !confirmed {
			fmt.Println("Rename cancelled.")
			return nil
		}
	}

	// Read source content
	// #nosec G304 - skill.Path comes from parsed skill files
	content, err := os.ReadFile(skill.Path)
	if err != nil {
		return fmt.Errorf("failed to read source skill: %w", err)
	}

	// Ensure target directory exists
	// #nosec G301 - skill directories need to be readable by the platform
	targetDir := targetPath[:len(targetPath)-len("/SKILL.md")]
	if err := os.MkdirAll(targetDir, 0o750); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Write to new location
	// #nosec G306 - skill files should be readable
	if err := os.WriteFile(targetPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write skill to new location: %w", err)
	}

	// Remove old skill
	if err := os.Remove(skill.Path); err != nil {
		// Try to clean up the new file if we can't remove the old one
		_ = os.Remove(targetPath)
		return fmt.Errorf("failed to remove old skill: %w", err)
	}

	// Try to remove old parent directory if it's empty
	oldParentDir := skill.Path[:len(skill.Path)-len("/SKILL.md")]
	if oldParentDir != skill.Path {
		_ = os.Remove(oldParentDir) // Ignore error - directory may not be empty
	}

	fmt.Printf("\n✓ Renamed skill %q to %q in %s scope\n", oldName, newName, scope)
	return nil
}
