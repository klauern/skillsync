package sync

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/harness"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/trust"
)

func TestSixHarnessStandardBundleMatrix(t *testing.T) {
	targets := []model.Platform{model.ClaudeCode, model.Codex, model.Cursor, model.Copilot, model.Gemini, model.Pi}
	for _, target := range targets {
		t.Run(string(target), func(t *testing.T) {
			source := t.TempDir()
			bundle := filepath.Join(source, "matrix-skill")
			for _, dir := range []string{"scripts", "references", "assets"} {
				if err := os.MkdirAll(filepath.Join(bundle, dir), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			content := "---\nname: matrix-skill\ndescription: Matrix skill\nlicense: Apache-2.0\ncompatibility: Requires git\nmetadata:\n  owner: skillsync\nx-claude-runtime: enabled\n---\n# Matrix\n"
			if err := os.WriteFile(filepath.Join(bundle, "SKILL.md"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			for _, file := range []string{"scripts/run.sh", "references/guide.md", "assets/icon.svg"} {
				if err := os.WriteFile(filepath.Join(bundle, file), []byte(file), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			def, ok := harness.Lookup(target)
			if !ok {
				t.Fatal("missing registry definition")
			}
			targetRoot := filepath.Join(t.TempDir(), strings.TrimPrefix(def.RepoRoots[0], "."))
			if err := os.MkdirAll(targetRoot, 0o755); err != nil {
				t.Fatal(err)
			}
			result, err := New().Sync(model.ClaudeCode, target, Options{
				SourcePath: source,
				TargetPath: targetRoot,
				Strategy:   StrategyOverwrite,
				TrustPolicy: trust.Policy{Allowed: map[trust.Risk]bool{
					trust.RiskExecutable: true,
				}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Skills) != 1 {
				t.Fatalf("synced skills = %d, want 1", len(result.Skills))
			}
			entry := filepath.Join(targetRoot, "matrix-skill", "SKILL.md")
			entryContent, err := os.ReadFile(entry)
			if err != nil {
				t.Fatalf("canonical entry %s: %v", entry, err)
			}
			if !strings.Contains(string(entryContent), "license: Apache-2.0") || !strings.Contains(string(entryContent), "compatibility: Requires git") {
				t.Fatalf("shared fields missing from %s: %s", target, entryContent)
			}
			for _, file := range []string{"scripts/run.sh", "references/guide.md", "assets/icon.svg"} {
				if _, err := os.Stat(filepath.Join(targetRoot, "matrix-skill", file)); err != nil {
					t.Fatalf("bundle file %s not copied: %v", file, err)
				}
			}
			if target == model.ClaudeCode {
				if !strings.Contains(string(entryContent), "x-claude-runtime: enabled") {
					t.Fatal("same-harness sync discarded raw frontmatter")
				}
				if len(result.Skills[0].PortabilityWarnings) > 0 {
					t.Errorf("unexpected same-harness warnings: %v", result.Skills[0].PortabilityWarnings)
				}
			} else {
				if strings.Contains(string(entryContent), "x-claude-runtime") {
					t.Fatalf("cross-harness sync retained runtime extension for %s", target)
				}
				if !slices.Contains(result.Skills[0].PortabilityWarnings, "x-claude-runtime") {
					t.Fatalf("cross-harness sync did not report discarded extension for %s: %v", target, result.Skills[0].PortabilityWarnings)
				}
			}
		})
	}
}
