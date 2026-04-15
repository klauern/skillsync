package copilot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestParserParseFixtureCount(t *testing.T) {
	skills := loadCopilotFixtureSkills(t)
	if got, want := len(skills), 5; got != want {
		t.Fatalf("Parse() returned %d skills, want %d", got, want)
	}
}

func TestParserParseRepositoryInstruction(t *testing.T) {
	repo := loadCopilotFixtureSkills(t)["copilot-instructions"]
	if repo.Name != "copilot-instructions" {
		t.Fatalf("repo name = %q, want copilot-instructions", repo.Name)
	}
	if repo.Platform != model.Copilot {
		t.Fatalf("repo Platform = %q, want %q", repo.Platform, model.Copilot)
	}
	if repo.Type != model.SkillTypeSkill {
		t.Fatalf("repo type = %q, want %q", repo.Type, model.SkillTypeSkill)
	}
	if repo.Metadata["type"] != "repository-instructions" {
		t.Fatalf("repo metadata type = %q, want repository-instructions", repo.Metadata["type"])
	}
	if repo.Description != "GitHub Copilot repository instructions" {
		t.Fatalf("repo description = %q, want %q", repo.Description, "GitHub Copilot repository instructions")
	}
}

func TestParserParseScopedInstructions(t *testing.T) {
	skills := loadCopilotFixtureSkills(t)

	goInstructions := skills["go"]
	if goInstructions.Description != "Go implementation guidance" {
		t.Fatalf("go Description = %q, want %q", goInstructions.Description, "Go implementation guidance")
	}
	if goInstructions.Metadata["applyTo"] != "**/*.go" {
		t.Fatalf("go applyTo = %q, want %q", goInstructions.Metadata["applyTo"], "**/*.go")
	}
	if goInstructions.Metadata["excludeAgent"] != "code-review" {
		t.Fatalf("go excludeAgent = %q, want %q", goInstructions.Metadata["excludeAgent"], "code-review")
	}

	docsInstructions := skills["docs-guidance"]
	if docsInstructions.Description != "Documentation guidance" {
		t.Fatalf("docs Description = %q, want %q", docsInstructions.Description, "Documentation guidance")
	}
	if docsInstructions.Metadata["applyTo"] != "**/*.md" {
		t.Fatalf("docs applyTo = %q, want %q", docsInstructions.Metadata["applyTo"], "**/*.md")
	}
}

func TestParserParsePromptFile(t *testing.T) {
	prompt := loadCopilotFixtureSkills(t)["test-gen"]
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
}

func TestParserParseAgentFile(t *testing.T) {
	agent := loadCopilotFixtureSkills(t)["reviewer"]
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
}

func TestParserParseWithoutFrontmatterUsesFilename(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".github", "instructions"), 0o750); err != nil {
		t.Fatalf("mkdir instructions: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".github", "instructions", "python.instructions.md"), []byte("Use Ruff and mypy.\n"), 0o644); err != nil {
		t.Fatalf("write instruction file: %v", err)
	}

	skills, err := New(root).Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("Parse() returned %d skills, want 1", len(skills))
	}
	if got := skills[0].Name; got != "python" {
		t.Fatalf("skill name = %q, want %q", got, "python")
	}
	if got := skills[0].Metadata["type"]; got != "instruction" {
		t.Fatalf("metadata type = %q, want %q", got, "instruction")
	}
}

func TestParserPlatformAndDefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	p := New("")
	if got := p.Platform(); got != model.Copilot {
		t.Fatalf("Platform() = %q, want %q", got, model.Copilot)
	}
	if got := p.DefaultPath(); got != filepath.Join(home, ".github") {
		t.Fatalf("DefaultPath() = %q, want %q", got, filepath.Join(home, ".github"))
	}
}

func TestParserParseNonexistentDirectory(t *testing.T) {
	skills, err := New("/nonexistent/path").Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 0 {
		t.Fatalf("Parse() returned %d skills, want 0", len(skills))
	}
}

func TestInstructionName(t *testing.T) {
	tests := map[string]string{
		"/repo/.github/copilot-instructions.md":             "copilot-instructions",
		"/repo/.github/instructions/python.instructions.md": "python",
	}

	for input, want := range tests {
		if got := instructionName(input); got != want {
			t.Fatalf("instructionName(%q) = %q, want %q", input, got, want)
		}
	}
}

func loadCopilotFixtureSkills(t *testing.T) map[string]model.Skill {
	t.Helper()

	fixtureRoot := filepath.Join(findRepoRoot(t), "testdata", "skills", "copilot")
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
