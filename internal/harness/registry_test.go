package harness

import (
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestRegistryHasSixCanonicalPlatforms(t *testing.T) {
	all := All()
	if len(all) != 6 {
		t.Fatalf("registry has %d platforms, want 6", len(all))
	}
	for _, d := range all {
		if d.Platform == model.PiDev && string(d.Platform) != "pi" {
			t.Fatal("Pi is not canonical")
		}
		if d.FactoryKey == "" {
			t.Errorf("%s missing factory key", d.Platform)
		}
	}
}

func TestPiAliasesResolveWithMetadata(t *testing.T) {
	for _, input := range []string{"pi.dev", "pi-dev", "pidev", "pi-agent", "piagent"} {
		r, err := Resolve(input)
		if err != nil || r.Definition.Platform != model.Pi || r.Alias == nil || !r.Alias.Deprecated {
			t.Errorf("Resolve(%q) = %+v, %v", input, r, err)
		}
	}
}

func TestCanonicalRootsAndDiscoveryOrder(t *testing.T) {
	want := map[model.Platform][]string{
		model.ClaudeCode: {".claude/skills", "~/.claude/skills"}, model.Codex: {".agents/skills", "~/.agents/skills"}, model.Cursor: {".cursor/skills", "~/.cursor/skills"}, model.Copilot: {".github/skills", "~/.copilot/skills"}, model.Gemini: {".gemini/skills", "~/.gemini/skills"}, model.Pi: {".pi/skills", "~/.pi/agent/skills"},
	}
	for p, roots := range want {
		d, ok := Lookup(p)
		if !ok {
			t.Fatal(p)
		}
		got := append(append([]string{}, d.RepoRoots...), d.UserRoots...)
		for i, root := range roots {
			if got[i] != root {
				t.Errorf("%s root %d = %q, want %q", p, i, got[i], root)
			}
			if p != model.Gemini && d.DiscoveryRoots[i] != root {
				t.Errorf("%s discovery %d = %q, want %q", p, i, d.DiscoveryRoots[i], root)
			}
		}
	}
	d, _ := Lookup(model.Codex)
	for i, root := range []string{".codex/skills", "~/.codex/skills", "/etc/codex/skills"} {
		if d.DiscoveryRoots[i+2] != root {
			t.Errorf("Codex extra root %d = %q, want %q", i, d.DiscoveryRoots[i+2], root)
		}
	}
	d, _ = Lookup(model.Gemini)
	if d.DiscoveryRoots[0] != ".agents/skills" || d.DiscoveryRoots[2] != ".gemini/skills" {
		t.Errorf("Gemini discovery order = %v", d.DiscoveryRoots)
	}
}
