package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/config"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/codex"
	"github.com/klauern/skillsync/internal/parser/copilot"
	"github.com/klauern/skillsync/internal/parser/cursor"
	"github.com/klauern/skillsync/internal/parser/gemini"
	"github.com/klauern/skillsync/internal/parser/pidev"
)

type matrixPlatform struct {
	name  string
	parse func(t *testing.T) model.Skill
}

type matrixFixtureSpec struct {
	relPath string
	content string
	parser  func(tmpDir string) interface{ Parse() ([]model.Skill, error) }
}

const (
	defaultMatrixFixture = `---
name: matrix-skill
description: matrix fixture
---

Compare matrix fixture.
`
	cursorMatrixFixture = `---
name: matrix-skill
globs: ["*.go"]
---

Compare matrix fixture.
`
	copilotMatrixFixture = `---
description: matrix fixture
tools:
- read
- search
agent: agent
model: GPT-4o
argument-hint: Compare matrix fixture
---

Compare matrix fixture.
`
)

func TestCompareAndDedupePlatformPairs(t *testing.T) {
	platforms := []matrixPlatform{
		{name: "claude-code", parse: parseClaudeFixture},
		{name: "cursor", parse: parseCursorFixture},
		{name: "codex", parse: parseCodexFixture},
		{name: "copilot", parse: parseCopilotFixture},
		{name: "gemini", parse: parseGeminiFixture},
		{name: "pi.dev", parse: parsePiDevFixture},
		{name: "pi-agent", parse: parsePiAgentFixture},
	}

	for _, source := range platforms {
		source := source
		for _, target := range platforms {
			target := target
			t.Run(fmt.Sprintf("%s_to_%s", source.name, target.name), func(t *testing.T) {
				sourceSkill := source.parse(t)
				targetSkill := target.parse(t)
				skills := []model.Skill{sourceSkill, targetSkill}

				compareCfg := &compareConfig{
					nameThreshold:        0.7,
					contentThreshold:     0.6,
					includeCrossPlatform: true,
					format:               "table",
					algorithm:            "combined",
				}

				compareResults, err := findSimilarSkills(skills, compareCfg)
				if err != nil {
					t.Fatalf("findSimilarSkills() error = %v", err)
				}
				if len(compareResults) != 1 {
					t.Fatalf("findSimilarSkills() returned %d results, want 1", len(compareResults))
				}

				compareOutput := captureStdout(t, func() {
					if err := outputCompareResults(compareResults, "table"); err != nil {
						t.Fatalf("outputCompareResults() error = %v", err)
					}
				})
				if strings.Count(compareOutput, "matrix-skill") < 2 {
					t.Fatalf("compare output %q does not contain both matrix-skill rows", compareOutput)
				}

				dupeResults, err := findDuplicatesForTUI(skills, config.Default())
				if err != nil {
					t.Fatalf("findDuplicatesForTUI() error = %v", err)
				}
				if len(dupeResults) != 1 {
					t.Fatalf("findDuplicatesForTUI() returned %d results, want 1", len(dupeResults))
				}
			})
		}
	}
}

func parseClaudeFixture(t *testing.T) model.Skill {
	t.Helper()
	return parseMatrixFixture(t, matrixFixtureSpec{
		relPath: "matrix-skill.md",
		content: defaultMatrixFixture,
		parser: func(tmpDir string) interface{ Parse() ([]model.Skill, error) } {
			return claude.New(tmpDir)
		},
	})
}

func parseCursorFixture(t *testing.T) model.Skill {
	t.Helper()
	return parseMatrixFixture(t, matrixFixtureSpec{
		relPath: "matrix-skill.md",
		content: cursorMatrixFixture,
		parser: func(tmpDir string) interface{ Parse() ([]model.Skill, error) } {
			return cursor.New(tmpDir)
		},
	})
}

func parseCodexFixture(t *testing.T) model.Skill {
	t.Helper()
	return parseMatrixFixture(t, matrixFixtureSpec{
		relPath: filepath.Join("matrix-skill", "SKILL.md"),
		content: defaultMatrixFixture,
		parser: func(tmpDir string) interface{ Parse() ([]model.Skill, error) } {
			return codex.New(tmpDir)
		},
	})
}

func parseCopilotFixture(t *testing.T) model.Skill {
	t.Helper()
	return parseMatrixFixture(t, matrixFixtureSpec{
		relPath: filepath.Join(".github", "prompts", "matrix-skill.prompt.md"),
		content: copilotMatrixFixture,
		parser: func(tmpDir string) interface{ Parse() ([]model.Skill, error) } {
			return copilot.New(filepath.Join(tmpDir, ".github"))
		},
	})
}

func parseGeminiFixture(t *testing.T) model.Skill {
	t.Helper()
	return parseMatrixFixture(t, matrixFixtureSpec{
		relPath: filepath.Join("skills", "matrix-skill", "SKILL.md"),
		content: defaultMatrixFixture,
		parser: func(tmpDir string) interface{ Parse() ([]model.Skill, error) } {
			return gemini.New(filepath.Join(tmpDir, "skills"))
		},
	})
}

func parsePiDevFixture(t *testing.T) model.Skill {
	t.Helper()
	return parseMatrixFixture(t, matrixFixtureSpec{
		relPath: filepath.Join("skills", "matrix-skill", "SKILL.md"),
		content: defaultMatrixFixture,
		parser: func(tmpDir string) interface{ Parse() ([]model.Skill, error) } {
			return pidev.New(tmpDir)
		},
	})
}

func parsePiAgentFixture(t *testing.T) model.Skill {
	t.Helper()

	return parsePiDevFixture(t)
}

func mustParseMatrixSkill(t *testing.T, parser interface{ Parse() ([]model.Skill, error) }) model.Skill {
	t.Helper()

	skills, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	for _, skill := range skills {
		if skill.Name == "matrix-skill" {
			return skill
		}
	}
	t.Fatalf("Parse() did not return matrix-skill; got %d skills", len(skills))
	return model.Skill{}
}

func parseMatrixFixture(t *testing.T, spec matrixFixtureSpec) model.Skill {
	t.Helper()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, spec.relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create fixture dir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(spec.content), 0o644); err != nil {
		t.Fatalf("failed to write fixture %s: %v", path, err)
	}

	return mustParseMatrixSkill(t, spec.parser(tmpDir))
}
