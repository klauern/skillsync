// Package harness contains the canonical platform registry shared by config,
// discovery, and parser selection code.
package harness

import (
	"fmt"
	"strings"

	"github.com/klauern/skillsync/internal/model"
)

// Alias describes an accepted platform spelling.
type Alias struct {
	Name       string
	Deprecated bool
}

// Definition contains canonical paths and parser metadata for a harness.
type Definition struct {
	Platform         model.Platform
	DisplayName      string
	Aliases          []Alias
	RepoRoots        []string
	UserRoots        []string
	DiscoveryRoots   []string
	ArtifactSurfaces []string
	FactoryKey       string
}

// Resolution contains canonical platform information for an input spelling.
type Resolution struct {
	Definition Definition
	Input      string
	Alias      *Alias
}

var definitions = []Definition{
	{Platform: model.ClaudeCode, DisplayName: "Claude Code", RepoRoots: []string{".claude/skills"}, UserRoots: []string{"~/.claude/skills"}, DiscoveryRoots: []string{".claude/skills", "~/.claude/skills"}, ArtifactSurfaces: []string{"skills", "commands"}, FactoryKey: "claude"},
	{Platform: model.Codex, DisplayName: "Codex", RepoRoots: []string{".agents/skills"}, UserRoots: []string{"~/.agents/skills"}, DiscoveryRoots: []string{".agents/skills", "~/.agents/skills", ".codex/skills", "~/.codex/skills", "/etc/codex/skills"}, ArtifactSurfaces: []string{"skills"}, FactoryKey: "codex"},
	{Platform: model.Cursor, DisplayName: "Cursor", RepoRoots: []string{".cursor/skills"}, UserRoots: []string{"~/.cursor/skills"}, DiscoveryRoots: []string{".cursor/skills", "~/.cursor/skills", ".agents/skills", "~/.agents/skills", ".claude/skills", "~/.claude/skills", ".codex/skills", "~/.codex/skills"}, ArtifactSurfaces: []string{"skills", "commands"}, FactoryKey: "cursor"},
	{Platform: model.Copilot, DisplayName: "Copilot", RepoRoots: []string{".github/skills"}, UserRoots: []string{"~/.copilot/skills"}, DiscoveryRoots: []string{".github/skills", "~/.copilot/skills", ".agents/skills", ".claude/skills"}, ArtifactSurfaces: []string{"skills", "agents", "prompts"}, FactoryKey: "copilot"},
	{Platform: model.Gemini, DisplayName: "Gemini CLI", RepoRoots: []string{".gemini/skills"}, UserRoots: []string{"~/.gemini/skills"}, DiscoveryRoots: []string{".agents/skills", "~/.agents/skills", ".gemini/skills", "~/.gemini/skills"}, ArtifactSurfaces: []string{"skills", "commands"}, FactoryKey: "gemini"},
	{Platform: model.Pi, DisplayName: "Pi", Aliases: []Alias{{"pi.dev", true}, {"pi-dev", true}, {"pidev", true}, {"pi-agent", true}, {"piagent", true}}, RepoRoots: []string{".pi/skills"}, UserRoots: []string{"~/.pi/agent/skills"}, DiscoveryRoots: []string{".pi/skills", "~/.pi/agent/skills", ".agents/skills", "~/.agents/skills"}, ArtifactSurfaces: []string{"skills", "prompts", "settings"}, FactoryKey: "pi"},
}

// All returns the six canonical harness definitions.
func All() []Definition { return append([]Definition(nil), definitions...) }

// Lookup returns the definition for a canonical platform.
func Lookup(p model.Platform) (Definition, bool) {
	for _, d := range definitions {
		if d.Platform == p {
			return d, true
		}
	}
	return Definition{}, false
}

// Resolve resolves a canonical platform or deprecated alias.
func Resolve(input string) (Resolution, error) {
	n := strings.ToLower(strings.TrimSpace(input))
	for _, d := range definitions {
		if n == string(d.Platform) {
			return Resolution{Definition: d, Input: n}, nil
		}
		for i := range d.Aliases {
			if n == d.Aliases[i].Name {
				a := d.Aliases[i]
				return Resolution{Definition: d, Input: n, Alias: &a}, nil
			}
		}
	}
	return Resolution{}, fmt.Errorf("unknown platform %q", input)
}
