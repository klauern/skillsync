// Package cli provides command definitions for skillsync.
package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func configCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Display current configuration",
		Action: func(_ context.Context, _ *cli.Command) error {
			fmt.Println("Configuration paths:")
			fmt.Println("  Claude Code: ~/.claude/skills/")
			fmt.Println("  Cursor: .cursor/rules/")
			fmt.Println("  Codex: .codex/")
			return nil
		},
	}
}

func discoveryCommand() *cli.Command {
	return &cli.Command{
		Name:  "discovery",
		Usage: "Discover skills on the system",
		Action: func(_ context.Context, _ *cli.Command) error {
			fmt.Println("Discovery command not yet implemented")
			return nil
		},
	}
}
