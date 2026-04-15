package copilot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestParsePromptsAndAgents(t *testing.T) {
	fixtureRoot := filepath.Join(findRepoRoot(t), "testdata", "skills", "copilot", ".github")

	skills, err := New(fixtureRoot).Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if got, want := len(skills), 4; got != want {
		t.Fatalf("Parse() returned %d skills, want %d", got, want)
	}

	byName := make(map[string]model.Skill, len(skills))
	for _, skill := range skills {
		byName[skill.Name] = skill
	}

	verifyRepoInstructions(t, byName["copilot-instructions"])
	verifyInstructions(t, byName["go-style"])
	verifyPrompt(t, byName["test-gen"])
	verifyAgent(t, byName["reviewer"])
}

func TestDefaultPath(t *testing.T) {
	if got := New("").DefaultPath(); filepath.Base(got) != ".github" {
		t.Fatalf("DefaultPath() = %q, want path rooted at .github", got)
	}
}

func verifyRepoInstructions(t *testing.T, repoInstructions model.Skill) {
	t.Helper()

	if repoInstructions.Type != model.SkillTypeSkill {
		t.Fatalf("repo instructions type = %q, want %q", repoInstructions.Type, model.SkillTypeSkill)
	}
	if repoInstructions.Metadata[model.MetadataKeyCopilotArtifact] != model.CopilotArtifactRepositoryInstructions {
		t.Fatalf("repo instructions artifact = %q, want %q", repoInstructions.Metadata[model.MetadataKeyCopilotArtifact], model.CopilotArtifactRepositoryInstructions)
	}
	if repoInstructions.Trigger != "" {
		t.Fatalf("repo instructions trigger = %q, want empty", repoInstructions.Trigger)
	}
}

func verifyInstructions(t *testing.T, instructions model.Skill) {
	t.Helper()

	if instructions.Type != model.SkillTypeSkill {
		t.Fatalf("instructions type = %q, want %q", instructions.Type, model.SkillTypeSkill)
	}
	if instructions.Metadata[model.MetadataKeyCopilotArtifact] != model.CopilotArtifactInstructions {
		t.Fatalf("instructions artifact = %q, want %q", instructions.Metadata[model.MetadataKeyCopilotArtifact], model.CopilotArtifactInstructions)
	}
	if instructions.Metadata["applyTo"] != "**/*.go" {
		t.Fatalf("instructions metadata applyTo = %q, want **/*.go", instructions.Metadata["applyTo"])
	}
	if instructions.Description != "Go-specific repository guidance" {
		t.Fatalf("instructions description = %q, want Go-specific repository guidance", instructions.Description)
	}
}

func verifyPrompt(t *testing.T, prompt model.Skill) {
	t.Helper()

	if prompt.Name != "test-gen" {
		t.Fatalf("prompt name = %q, want test-gen", prompt.Name)
	}
	if prompt.Type != model.SkillTypePrompt {
		t.Fatalf("prompt type = %q, want %q", prompt.Type, model.SkillTypePrompt)
	}
	if prompt.Trigger != "/test-gen" {
		t.Fatalf("prompt trigger = %q, want /test-gen", prompt.Trigger)
	}
	if prompt.Description != "Generate tests for the active file" {
		t.Fatalf("prompt description = %q, want %q", prompt.Description, "Generate tests for the active file")
	}
	if got, want := len(prompt.Tools), 2; got != want {
		t.Fatalf("prompt tools length = %d, want %d", got, want)
	}
	if prompt.Tools[0] != "read" || prompt.Tools[1] != "search" {
		t.Fatalf("prompt tools = %v, want [read search]", prompt.Tools)
	}
	if prompt.Metadata["agent"] != "agent" {
		t.Fatalf("prompt metadata agent = %q, want agent", prompt.Metadata["agent"])
	}
	if prompt.Metadata["model"] != "GPT-4o" {
		t.Fatalf("prompt metadata model = %q, want GPT-4o", prompt.Metadata["model"])
	}
	if prompt.Metadata["argument-hint"] != "Describe the test scenarios" {
		t.Fatalf("prompt metadata argument-hint = %q, want Describe the test scenarios", prompt.Metadata["argument-hint"])
	}
	if prompt.Metadata[model.MetadataKeyCopilotArtifact] != model.CopilotArtifactPrompt {
		t.Fatalf("prompt metadata artifact = %q, want %q", prompt.Metadata[model.MetadataKeyCopilotArtifact], model.CopilotArtifactPrompt)
	}
}

func verifyAgent(t *testing.T, agent model.Skill) {
	t.Helper()

	if agent.Name != "reviewer" {
		t.Fatalf("agent name = %q, want reviewer", agent.Name)
	}
	if agent.Type != model.SkillTypeSkill {
		t.Fatalf("agent type = %q, want %q", agent.Type, model.SkillTypeSkill)
	}
	if agent.Trigger != "" {
		t.Fatalf("agent trigger = %q, want empty", agent.Trigger)
	}
	if agent.Description != "Security-focused code reviewer" {
		t.Fatalf("agent description = %q, want %q", agent.Description, "Security-focused code reviewer")
	}
	if got, want := len(agent.Tools), 3; got != want {
		t.Fatalf("agent tools length = %d, want %d", got, want)
	}
	if agent.Tools[0] != "read" || agent.Tools[1] != "search" || agent.Tools[2] != "web" {
		t.Fatalf("agent tools = %v, want [read search web]", agent.Tools)
	}
	if agent.Metadata["model"] != "[Claude Opus 4.5 GPT-4o]" {
		t.Fatalf("agent metadata model = %q, want [Claude Opus 4.5 GPT-4o]", agent.Metadata["model"])
	}
	if agent.Metadata["agents"] != "[implementer investigator]" {
		t.Fatalf("agent metadata agents = %q, want [implementer investigator]", agent.Metadata["agents"])
	}
	if agent.Metadata["argument-hint"] != "Describe the review focus" {
		t.Fatalf("agent metadata argument-hint = %q, want Describe the review focus", agent.Metadata["argument-hint"])
	}
	if agent.Metadata["user-invokable"] != "true" {
		t.Fatalf("agent metadata user-invokable = %q, want true", agent.Metadata["user-invokable"])
	}
	if agent.Metadata["disable-model-invocation"] != "false" {
		t.Fatalf("agent metadata disable-model-invocation = %q, want false", agent.Metadata["disable-model-invocation"])
	}
	if agent.Metadata["target"] != "vscode" {
		t.Fatalf("agent metadata target = %q, want vscode", agent.Metadata["target"])
	}
	if agent.Metadata[model.MetadataKeyCopilotArtifact] != model.CopilotArtifactAgent {
		t.Fatalf("agent metadata artifact = %q, want %q", agent.Metadata[model.MetadataKeyCopilotArtifact], model.CopilotArtifactAgent)
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
