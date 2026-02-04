// Package cli provides the onboarding command for skillsync.
package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"
)

func onboardCommand() *cli.Command {
	return &cli.Command{
		Name:    "onboard",
		Aliases: []string{"llm"},
		Usage:   "Print LLM-friendly usage guidance for SkillSync",
		Action: func(_ context.Context, _ *cli.Command) error {
			fmt.Print(onboardGuide())
			return nil
		},
	}
}

func onboardGuide() string {
	return strings.TrimSpace(`
# SkillSync LLM Onboarding

## Purpose
- Sync AI coding skills across Claude Code, Cursor, and Codex.
- Keep skills consistent, deduplicate, and back up before changes.

## Key concepts
- Platform: claude-code, cursor, codex.
- Scope: repo, user, admin, system, builtin, plugin.
- Writable scopes: repo and user.
- Sync is one-way: source -> target.

## Quick start
1. skillsync config init
2. skillsync discover --format table
3. skillsync sync --dry-run cursor claude-code
4. skillsync sync cursor claude-code

## Common workflows
- Discover: skillsync discover --platform cursor --scope repo,user
- Sync: skillsync sync --strategy newer cursor claude-code
- Delete: skillsync delete --dry-run cursor codex
- Compare: skillsync compare
- Dedupe: skillsync dedupe list --platform cursor
- Export: skillsync export --format json --output skills.json
- Backups: skillsync backup list | backup verify | backup restore <id>
- Promote/Demote: skillsync promote my-skill | skillsync demote my-skill
- Scope: skillsync scope list my-skill
- TUI: skillsync tui

## Tips for safe usage
- Use --dry-run before any sync or delete.
- Prefer --strategy newer for two-way workflows.
- Include plugin skills with --include-plugins or :plugin scope.
- Use --format json|yaml for machine-readable output.
- Every command has help: skillsync <command> --help

## Sync patterns
- Source/target syntax: platform[:scope[,scope2]]
- Example: skillsync sync cursor:repo,user codex:repo
- Strategies: overwrite, skip, newer, merge, three-way, interactive
`) + "\n"
}
