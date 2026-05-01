package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/util"
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
	fmt.Printf("  Pi.dev:          %v\n", cfg.Platforms.PiDev.SkillsPaths)

	fmt.Println("\nData paths:")
	fmt.Printf("  Backups:         %s\n", util.SkillsyncBackupsPath())
	fmt.Printf("  Cache:           %s\n", filepath.Join(util.SkillsyncConfigPath(), "cache"))
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

	// Editor may include arguments (e.g. EDITOR="code --wait")
	editorArgs := strings.Fields(editor)
	// #nosec G204 G702 - editor binary is intentionally user-controlled via $EDITOR/$VISUAL
	cmd := exec.Command(editorArgs[0], append(editorArgs[1:], configPath)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
