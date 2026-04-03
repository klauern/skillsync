package parser_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/codex"
	"github.com/klauern/skillsync/internal/parser/cursor"
	"github.com/klauern/skillsync/internal/parser/pidev"
)

func TestFixtureDiscoveryCountsByPlatform(t *testing.T) {
	fixtureRoot := filepath.Join(findRepoRoot(t), "testdata", "skills")

	tests := map[string]struct {
		parse func() (int, error)
		want  int
	}{
		"claude-code": {
			parse: func() (int, error) {
				skills, err := claude.New(filepath.Join(fixtureRoot, "claude")).Parse()
				return len(skills), err
			},
			want: 7,
		},
		"cursor": {
			parse: func() (int, error) {
				skills, err := cursor.New(filepath.Join(fixtureRoot, "cursor")).Parse()
				return len(skills), err
			},
			want: 2,
		},
		"codex": {
			parse: func() (int, error) {
				skills, err := codex.New(filepath.Join(fixtureRoot, "codex")).Parse()
				return len(skills), err
			},
			want: 6,
		},
		"pidev": {
			parse: func() (int, error) {
				return parsePiDevFixtureCount(t, fixtureRoot)
			},
			want: 7,
		},
	}

	for platform, tt := range tests {
		t.Run(platform, func(t *testing.T) {
			got, err := tt.parse()
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if got != tt.want {
				t.Fatalf("discovered %d skills, want %d", got, tt.want)
			}
		})
	}
}

func TestCodexFixturesIncludeHumanReadableNameSkill(t *testing.T) {
	fixtureRoot := filepath.Join(findRepoRoot(t), "testdata", "skills", "codex")

	skills, err := codex.New(fixtureRoot).Parse()
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		names = append(names, skill.Name)
	}

	if !slices.Contains(names, "agent-development") {
		t.Fatalf("expected parsed Codex fixtures to include %q, got %v", "agent-development", names)
	}
}

func TestPiDevFixturesParseSkillsPromptsAndInstructions(t *testing.T) {
	fixtureRoot := filepath.Join(findRepoRoot(t), "testdata", "skills", "pidev")
	tmpRoot := t.TempDir()
	copyFixtureTree(t, fixtureRoot, tmpRoot)

	if err := os.MkdirAll(filepath.Join(tmpRoot, ".git"), 0o755); err != nil {
		t.Fatalf("failed to create git dir: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(filepath.Join(tmpRoot, "instructions")); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	skills, err := pidev.New(filepath.Join(tmpRoot, "skills")).Parse()
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if got := len(skills); got != 7 {
		t.Fatalf("discovered %d skills, want 7", got)
	}

	byName := make(map[string]model.Skill, len(skills))
	for _, skill := range skills {
		byName[skill.Name] = skill
	}

	for _, name := range []string{"basic-skill", "structured-skill", "user-prompt", "agents", "instructions-agents", "system", "append-system"} {
		if _, ok := byName[name]; !ok {
			t.Fatalf("expected %q in parsed fixtures, got %v", name, mapsKeys(byName))
		}
	}

	system := byName["system"]
	if system.Metadata["type"] != "system-prompt" {
		t.Fatalf("system metadata type = %q, want system-prompt", system.Metadata["type"])
	}
	if system.Metadata["mode"] != "replace" {
		t.Fatalf("system metadata mode = %q, want replace", system.Metadata["mode"])
	}

	appendSystem := byName["append-system"]
	if appendSystem.Metadata["type"] != "system-prompt" {
		t.Fatalf("append system metadata type = %q, want system-prompt", appendSystem.Metadata["type"])
	}
	if appendSystem.Metadata["mode"] != "append" {
		t.Fatalf("append system metadata mode = %q, want append", appendSystem.Metadata["mode"])
	}

	structured := byName["structured-skill"]
	if !slices.Contains(structured.Scripts, "scripts/setup.sh") {
		t.Fatalf("structured skill scripts = %v, want to include scripts/setup.sh", structured.Scripts)
	}
	if !slices.Contains(structured.References, "references/naming.md") {
		t.Fatalf("structured skill references = %v, want to include references/naming.md", structured.References)
	}
	if !slices.Contains(structured.Assets, "assets/data.txt") {
		t.Fatalf("structured skill assets = %v, want to include assets/data.txt", structured.Assets)
	}

	prompt := byName["user-prompt"]
	if prompt.Type != model.SkillTypePrompt {
		t.Fatalf("prompt type = %q, want %q", prompt.Type, model.SkillTypePrompt)
	}
	if prompt.Trigger != "/user-prompt" {
		t.Fatalf("prompt trigger = %q, want /user-prompt", prompt.Trigger)
	}
	if prompt.Metadata["category"] != "review" {
		t.Fatalf("prompt category metadata = %q, want review", prompt.Metadata["category"])
	}

	agents := byName["agents"]
	if agents.Metadata["type"] != "agents" {
		t.Fatalf("root AGENTS metadata type = %q, want agents", agents.Metadata["type"])
	}

	instructions := byName["instructions-agents"]
	if instructions.Metadata["type"] != "agents" {
		t.Fatalf("nested AGENTS metadata type = %q, want agents", instructions.Metadata["type"])
	}
}

func parsePiDevFixtureCount(t *testing.T, fixtureRoot string) (int, error) {
	t.Helper()

	tmpRoot := t.TempDir()
	copyFixtureTree(t, filepath.Join(fixtureRoot, "pidev"), tmpRoot)

	if err := os.MkdirAll(filepath.Join(tmpRoot, ".git"), 0o755); err != nil {
		return 0, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return 0, err
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(filepath.Join(tmpRoot, "instructions")); err != nil {
		return 0, err
	}

	skills, err := pidev.New(filepath.Join(tmpRoot, "skills")).Parse()
	return len(skills), err
}

func copyFixtureTree(t *testing.T, srcRoot, dstRoot string) {
	t.Helper()

	err := filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o600)
	})
	if err != nil {
		t.Fatalf("copy fixture tree: %v", err)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "testdata", "skills")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repository root from %s", dir)
		}
		dir = parent
	}
}

func mapsKeys(m map[string]model.Skill) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}
