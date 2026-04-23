package copilot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestParsePromptsAndAgents(t *testing.T) {
	byName := parseCopilotSkills(t)

	if got, want := len(byName), 6; got != want {
		t.Fatalf("Parse() returned %d skills, want %d", got, want)
	}

	t.Run("repo instructions", func(t *testing.T) {
		assertRepoInstructions(t, mustSkill(t, byName, "copilot-instructions"))
	})
	t.Run("docs instructions", func(t *testing.T) {
		assertInstruction(t, mustSkill(t, byName, "docs-guidance"), "docs-guidance", "Documentation guidance", "**/*.md")
	})
	t.Run("go instructions", func(t *testing.T) {
		skill := mustSkill(t, byName, "go")
		assertInstruction(t, skill, "go", "Go implementation guidance", "**/*.go")
		if got := skill.Metadata["excludeAgent"]; got != "code-review" {
			t.Fatalf("instruction metadata excludeAgent = %q, want code-review", got)
		}
	})
	t.Run("instructions", func(t *testing.T) {
		assertInstruction(t, mustSkill(t, byName, "go-style"), "go-style", "Go-specific repository guidance", "**/*.go")
	})
	t.Run("prompt", func(t *testing.T) {
		assertPrompt(t, mustSkill(t, byName, "test-gen"))
	})
	t.Run("agent", func(t *testing.T) {
		assertAgent(t, mustSkill(t, byName, "reviewer"))
	})
}

func parseCopilotSkills(t *testing.T) map[string]model.Skill {
	t.Helper()

	fixtureRoot := filepath.Join(findRepoRoot(t), "testdata", "skills", "copilot", ".github")

	skills, err := New(fixtureRoot).Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	byName := make(map[string]model.Skill, len(skills))
	for _, skill := range skills {
		byName[skill.Name] = skill
	}
	return byName
}

func mustSkill(t *testing.T, byName map[string]model.Skill, name string) model.Skill {
	t.Helper()

	skill, ok := byName[name]
	if !ok {
		t.Fatalf("missing skill %q", name)
	}
	return skill
}

func assertRepoInstructions(t *testing.T, skill model.Skill) {
	t.Helper()

	if skill.Type != model.SkillTypeSkill {
		t.Fatalf("repo instructions type = %q, want %q", skill.Type, model.SkillTypeSkill)
	}
	if skill.Metadata[model.MetadataKeyCopilotArtifact] != model.CopilotArtifactRepositoryInstructions {
		t.Fatalf("repo instructions artifact = %q, want %q", skill.Metadata[model.MetadataKeyCopilotArtifact], model.CopilotArtifactRepositoryInstructions)
	}
	if skill.Trigger != "" {
		t.Fatalf("repo instructions trigger = %q, want empty", skill.Trigger)
	}
}

func assertInstruction(t *testing.T, skill model.Skill, wantName, wantDescription, wantApplyTo string) {
	t.Helper()

	if skill.Type != model.SkillTypeSkill {
		t.Fatalf("instruction type = %q, want %q", skill.Type, model.SkillTypeSkill)
	}
	if skill.Metadata[model.MetadataKeyCopilotArtifact] != model.CopilotArtifactInstructions {
		t.Fatalf("instruction artifact = %q, want %q", skill.Metadata[model.MetadataKeyCopilotArtifact], model.CopilotArtifactInstructions)
	}
	if skill.Name != wantName {
		t.Fatalf("instruction name = %q, want %q", skill.Name, wantName)
	}
	if skill.Metadata["applyTo"] != wantApplyTo {
		t.Fatalf("instruction metadata applyTo = %q, want %q", skill.Metadata["applyTo"], wantApplyTo)
	}
	if skill.Description != wantDescription {
		t.Fatalf("instruction description = %q, want %q", skill.Description, wantDescription)
	}
}

func assertPrompt(t *testing.T, skill model.Skill) {
	t.Helper()

	if skill.Name != "test-gen" {
		t.Fatalf("prompt name = %q, want test-gen", skill.Name)
	}
	if skill.Type != model.SkillTypePrompt {
		t.Fatalf("prompt type = %q, want %q", skill.Type, model.SkillTypePrompt)
	}
	if skill.Trigger != "/test-gen" {
		t.Fatalf("prompt trigger = %q, want /test-gen", skill.Trigger)
	}
	if skill.Description != "Generate tests for the active file" {
		t.Fatalf("prompt description = %q, want %q", skill.Description, "Generate tests for the active file")
	}
	if got, want := len(skill.Tools), 2; got != want {
		t.Fatalf("prompt tools length = %d, want %d", got, want)
	}
	if skill.Tools[0] != "read" || skill.Tools[1] != "search" {
		t.Fatalf("prompt tools = %v, want [read search]", skill.Tools)
	}
	if skill.Metadata["agent"] != "agent" {
		t.Fatalf("prompt metadata agent = %q, want agent", skill.Metadata["agent"])
	}
	if skill.Metadata["model"] != "GPT-4o" {
		t.Fatalf("prompt metadata model = %q, want GPT-4o", skill.Metadata["model"])
	}
	if skill.Metadata["argument-hint"] != "Describe the test scenarios" {
		t.Fatalf("prompt metadata argument-hint = %q, want Describe the test scenarios", skill.Metadata["argument-hint"])
	}
	if skill.Metadata[model.MetadataKeyCopilotArtifact] != model.CopilotArtifactPrompt {
		t.Fatalf("prompt metadata artifact = %q, want %q", skill.Metadata[model.MetadataKeyCopilotArtifact], model.CopilotArtifactPrompt)
	}
}

func assertAgent(t *testing.T, skill model.Skill) {
	t.Helper()

	if skill.Name != "reviewer" {
		t.Fatalf("agent name = %q, want reviewer", skill.Name)
	}
	if skill.Type != model.SkillTypeSkill {
		t.Fatalf("agent type = %q, want %q", skill.Type, model.SkillTypeSkill)
	}
	if skill.Trigger != "" {
		t.Fatalf("agent trigger = %q, want empty", skill.Trigger)
	}
	if skill.Description != "Security-focused code reviewer" {
		t.Fatalf("agent description = %q, want %q", skill.Description, "Security-focused code reviewer")
	}
	if got, want := len(skill.Tools), 3; got != want {
		t.Fatalf("agent tools length = %d, want %d", got, want)
	}
	if skill.Tools[0] != "read" || skill.Tools[1] != "search" || skill.Tools[2] != "web" {
		t.Fatalf("agent tools = %v, want [read search web]", skill.Tools)
	}
	if skill.Metadata["model"] != "[Claude Opus 4.5 GPT-4o]" {
		t.Fatalf("agent metadata model = %q, want [Claude Opus 4.5 GPT-4o]", skill.Metadata["model"])
	}
	if skill.Metadata["agents"] != "[implementer investigator]" {
		t.Fatalf("agent metadata agents = %q, want [implementer investigator]", skill.Metadata["agents"])
	}
	if skill.Metadata["argument-hint"] != "Describe the review focus" {
		t.Fatalf("agent metadata argument-hint = %q, want Describe the review focus", skill.Metadata["argument-hint"])
	}
	if skill.Metadata["user-invokable"] != "true" {
		t.Fatalf("agent metadata user-invokable = %q, want true", skill.Metadata["user-invokable"])
	}
	if skill.Metadata["disable-model-invocation"] != "false" {
		t.Fatalf("agent metadata disable-model-invocation = %q, want false", skill.Metadata["disable-model-invocation"])
	}
	if skill.Metadata["target"] != "vscode" {
		t.Fatalf("agent metadata target = %q, want vscode", skill.Metadata["target"])
	}
	if skill.Metadata[model.MetadataKeyCopilotArtifact] != model.CopilotArtifactAgent {
		t.Fatalf("agent metadata artifact = %q, want %q", skill.Metadata[model.MetadataKeyCopilotArtifact], model.CopilotArtifactAgent)
	}
}

func TestDefaultPath(t *testing.T) {
	if got := New("").DefaultPath(); filepath.Base(got) != ".github" {
		t.Fatalf("DefaultPath() = %q, want path rooted at .github", got)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Abs(.) failed: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "testdata", "skills")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root")
		}
		dir = parent
	}
}
