package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/template"
	"github.com/klauern/skillsync/internal/ui"
)

func newCommand() *cli.Command {
	return &cli.Command{
		Name:  "new",
		Usage: "Create a new skill from a template",
		UsageText: `skillsync new <skill-name> [options]
   skillsync new my-skill --platform claude-code --scope repo
   skillsync new my-skill --platform cursor --template workflow
   skillsync new my-skill --template utility --interactive`,
		Description: `Create a new skill with scaffolding from built-in or custom templates.

   Built-in templates:
     command-wrapper  Wrap an external command or tool
     workflow         Orchestrate multiple steps in a workflow
     utility          Provide helper functionality

   Examples:
     # Create a basic command wrapper skill for Claude Code
     skillsync new my-skill --platform claude-code --scope repo

     # Create a workflow skill for Cursor
     skillsync new my-workflow --platform cursor --template workflow

     # Interactive setup
     skillsync new my-skill --interactive

     # Use custom template
     skillsync new my-skill --template-file ./my-template.md`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "platform",
				Aliases:  []string{"p"},
				Usage:    "Target platform (claude-code, cursor, codex)",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "scope",
				Aliases: []string{"s"},
				Value:   "repo",
				Usage:   "Target scope (builtin, system, admin, user, repo, plugin)",
			},
			&cli.StringFlag{
				Name:    "template",
				Aliases: []string{"t"},
				Value:   "command-wrapper",
				Usage:   "Template type (command-wrapper, workflow, utility)",
			},
			&cli.StringFlag{
				Name:  "template-file",
				Usage: "Path to custom template file",
			},
			&cli.StringFlag{
				Name:  "description",
				Usage: "Brief description of the skill",
			},
			&cli.BoolFlag{
				Name:    "interactive",
				Aliases: []string{"i"},
				Usage:   "Interactive setup wizard",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Preview generated content without creating files",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			if args.Len() < 1 {
				return errors.New("skill name is required")
			}

			skillName := args.Get(0)
			return runNew(cmd, skillName)
		},
	}
}

func runNew(cmd *cli.Command, skillName string) error {
	logging.Debug("creating new skill", slog.String("name", skillName))

	// Validate skill name
	if err := validateSkillName(skillName); err != nil {
		return fmt.Errorf("invalid skill name: %w", err)
	}

	// Parse platform
	platformStr := cmd.String("platform")
	platform, err := model.ParsePlatform(platformStr)
	if err != nil {
		return fmt.Errorf("invalid platform: %w", err)
	}

	// Parse scope
	scopeStr := cmd.String("scope")
	scope, err := model.ParseScope(scopeStr)
	if err != nil {
		return fmt.Errorf("invalid scope: %w", err)
	}

	// Get description
	description := cmd.String("description")
	if description == "" {
		if cmd.Bool("interactive") {
			description = promptForDescription()
		} else {
			description = fmt.Sprintf("A %s skill", skillName)
		}
	}

	// Create generator
	gen, err := template.New()
	if err != nil {
		return fmt.Errorf("failed to initialize template generator: %w", err)
	}

	// Load custom template if specified
	customTemplateFile := cmd.String("template-file")
	var templateType template.TemplateType
	if customTemplateFile != "" {
		if err := gen.LoadCustomTemplate(skillName, customTemplateFile); err != nil {
			return fmt.Errorf("failed to load custom template: %w", err)
		}
		templateType = template.TemplateType(skillName)
	} else {
		// Parse template type
		templateStr := cmd.String("template")
		templateType, err = template.ParseTemplateType(templateStr)
		if err != nil {
			return fmt.Errorf("invalid template type: %w", err)
		}
	}

	// Prepare template data
	data := template.TemplateData{
		Name:        skillName,
		Description: description,
		Platform:    platformStr,
		Scope:       scopeStr,
		Tools:       getDefaultTools(templateType),
	}

	// Interactive mode - prompt for additional details
	if cmd.Bool("interactive") {
		if err := promptForTemplateData(&data); err != nil {
			return fmt.Errorf("interactive setup failed: %w", err)
		}
	}

	// Dry-run mode - just show content
	if cmd.Bool("dry-run") {
		content, err := gen.Generate(templateType, data)
		if err != nil {
			return fmt.Errorf("failed to generate content: %w", err)
		}

		ui.Info("Generated content preview:")
		fmt.Println(strings.Repeat("-", 80))
		fmt.Println(content)
		fmt.Println(strings.Repeat("-", 80))
		return nil
	}

	// Create the skill file
	skillPath, err := gen.CreateSkillFile(templateType, data, platform, scope)
	if err != nil {
		return fmt.Errorf("failed to create skill: %w", err)
	}

	ui.Success("Created new skill: %s", skillPath)
	ui.Info("Edit the skill file to customize the behavior.")

	// Show next steps
	showNextSteps(skillName, platform, scope)

	return nil
}

// validateSkillName validates that a skill name is valid
func validateSkillName(name string) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}

	// Check for valid characters (alphanumeric, hyphens, underscores, colons, slashes)
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == ':' || r == '/') {
			return fmt.Errorf("name contains invalid character %q (allowed: alphanumeric, -, _, :, /)", r)
		}
	}

	return nil
}

// getDefaultTools returns default tools for a given template type
func getDefaultTools(templateType template.TemplateType) []string {
	switch templateType {
	case template.CommandWrapper:
		return []string{"Bash", "Read"}
	case template.Workflow:
		return []string{"Bash", "Read", "Write"}
	case template.Utility:
		return []string{"Read", "Write"}
	default:
		return []string{}
	}
}

// promptForDescription prompts the user for a skill description
func promptForDescription() string {
	fmt.Print("Enter skill description: ")
	var description string
	fmt.Scanln(&description)
	return description
}

// promptForTemplateData prompts for additional template data in interactive mode
func promptForTemplateData(data *template.TemplateData) error {
	// Could add more interactive prompts here for tools, scripts, etc.
	// For now, just use defaults
	return nil
}

// showNextSteps displays helpful next steps after creating a skill
func showNextSteps(skillName string, platform model.Platform, scope model.SkillScope) {
	ui.Info("\nNext steps:")
	fmt.Printf("  1. Edit the skill file to customize behavior\n")
	fmt.Printf("  2. Test the skill: /%s\n", skillName)

	if scope == model.ScopeRepo {
		fmt.Printf("  3. Commit the skill to version control\n")
	}

	fmt.Printf("  4. Use 'skillsync sync' to sync across platforms\n")
}
