// Package cli provides command definitions for skillsync.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/util"
)

// pluginsCommand returns the plugins command for managing Claude Code plugins.
func pluginsCommand() *cli.Command {
	return &cli.Command{
		Name:    "plugins",
		Aliases: []string{"plugin"},
		Usage:   "Manage Claude Code plugins and plugin-sourced skills",
		Description: `Manage Claude Code plugins and discover skills from installed plugins.

   This command provides plugin-specific operations including:
   - List installed plugins from ~/.claude/plugins/installed_plugins.json
   - Discover skills from installed plugins in ~/.claude/plugins/cache/
   - Show plugin details and metadata

   For general skill discovery across all platforms, use 'skillsync discover'.`,
		Commands: []*cli.Command{
			pluginsListCommand(),
			pluginsSkillsCommand(),
		},
	}
}

// pluginsListCommand lists all installed Claude Code plugins.
func pluginsListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List installed Claude Code plugins",
		Description: `List all installed Claude Code plugins from ~/.claude/plugins/installed_plugins.json.

   Shows plugin name, marketplace, version, and installation path.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "table",
				Usage:   "Output format: table, json",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			format := cmd.String("format")
			return listInstalledPlugins(format)
		},
	}
}

// pluginsSkillsCommand lists skills from installed plugins.
func pluginsSkillsCommand() *cli.Command {
	return &cli.Command{
		Name:    "skills",
		Aliases: []string{"skill"},
		Usage:   "List skills from installed Claude Code plugins",
		Description: `Discover and list skills from installed Claude Code plugins.

   Scans ~/.claude/plugins/cache/ for plugin skills and shows their metadata.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "table",
				Usage:   "Output format: table, json",
			},
			&cli.StringFlag{
				Name:    "plugin",
				Aliases: []string{"p"},
				Usage:   "Filter by plugin name (e.g., commits@klauern-skills)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			format := cmd.String("format")
			pluginFilter := cmd.String("plugin")
			return listPluginSkills(format, pluginFilter)
		},
	}
}

// pluginEntry represents a single plugin installation for display.
type pluginEntry struct {
	PluginKey   string `json:"pluginKey"`
	PluginName  string `json:"pluginName"`
	Marketplace string `json:"marketplace"`
	Version     string `json:"version"`
	InstallPath string `json:"installPath"`
}

// listInstalledPlugins lists all installed Claude Code plugins.
func listInstalledPlugins(format string) error {
	// Load and parse the installed plugins manifest directly
	pluginsPath := util.ClaudeInstalledPluginsPath()

	// #nosec G304 - path is from trusted source (util package)
	data, err := os.ReadFile(pluginsPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No installed plugins found. File does not exist:", pluginsPath)
			return nil
		}
		return fmt.Errorf("failed to read installed plugins: %w", err)
	}

	var manifest claude.InstalledPluginsFile
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse installed plugins: %w", err)
	}

	var entries []pluginEntry
	seen := make(map[string]bool)

	for pluginKey, installations := range manifest.Plugins {
		pluginName, marketplace := parsePluginKey(pluginKey)

		for _, inst := range installations {
			key := pluginKey + "@" + inst.Version
			if seen[key] {
				continue
			}
			seen[key] = true

			entries = append(entries, pluginEntry{
				PluginKey:   pluginKey,
				PluginName:  pluginName,
				Marketplace: marketplace,
				Version:     inst.Version,
				InstallPath: inst.InstallPath,
			})
		}
	}

	if len(entries) == 0 {
		fmt.Println("No installed plugins found.")
		return nil
	}

	// Sort by plugin key
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].PluginKey < entries[j].PluginKey
	})

	switch format {
	case "json":
		return outputPluginsJSON(entries)
	case "table":
		return outputPluginsTable(entries)
	default:
		return fmt.Errorf("unsupported format: %s (use table or json)", format)
	}
}

// listPluginSkills lists skills from installed Claude Code plugins.
func listPluginSkills(format, pluginFilter string) error {
	// Use the Claude cache parser to discover plugin skills
	cacheParser := claude.NewCachePluginsParser("")
	skills, err := cacheParser.Parse()
	if err != nil {
		return fmt.Errorf("failed to discover plugin skills: %w", err)
	}

	// Filter by plugin if requested
	if pluginFilter != "" {
		var filtered []model.Skill
		for _, skill := range skills {
			if skill.PluginInfo != nil && skill.PluginInfo.PluginName == pluginFilter {
				filtered = append(filtered, skill)
			}
		}
		skills = filtered
	}

	if len(skills) == 0 {
		if pluginFilter != "" {
			fmt.Printf("No skills found for plugin: %s\n", pluginFilter)
		} else {
			fmt.Println("No plugin skills found.")
		}
		return nil
	}

	// Sort by name
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	// Output results
	return outputSkills(skills, format)
}

// outputPluginsJSON outputs plugin entries in JSON format.
func outputPluginsJSON(entries []pluginEntry) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entries)
}

// outputPluginsTable outputs plugin entries in table format.
func outputPluginsTable(entries []pluginEntry) error {
	// Print header
	fmt.Printf("%-40s %-20s %-15s\n", "PLUGIN", "MARKETPLACE", "VERSION")
	fmt.Printf("%-40s %-20s %-15s\n",
		strings.Repeat("-", 40),
		strings.Repeat("-", 20),
		strings.Repeat("-", 15))

	// Print entries
	for _, entry := range entries {
		fmt.Printf("%-40s %-20s %-15s\n",
			truncateStr(entry.PluginKey, 40),
			truncateStr(entry.Marketplace, 20),
			truncateStr(entry.Version, 15))
	}

	fmt.Printf("\nTotal: %d plugin(s)\n", len(entries))
	return nil
}

// truncateStr truncates a string to the specified width.
func truncateStr(s string, width int) string {
	if len(s) <= width {
		return s
	}
	if width < 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

// parsePluginKey splits a plugin key like "commits@klauern-skills" into name and marketplace.
func parsePluginKey(key string) (pluginName, marketplace string) {
	parts := strings.SplitN(key, "@", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return key, ""
}
