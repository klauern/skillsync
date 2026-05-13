package copilot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestParse_NonexistentBasePath(t *testing.T) {
	skills, err := New("/nonexistent/copilot/path/that/does/not/exist").Parse()
	if err != nil {
		t.Fatalf("Parse() unexpected error for nonexistent path: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("Parse() = %d skills, want 0 for nonexistent path", len(skills))
	}
}

func TestParseRepositoryInstructions_NotPresent(t *testing.T) {
	dir := t.TempDir()
	p := New(dir)
	skills, err := p.parseRepositoryInstructions(make(map[string]bool))
	if err != nil {
		t.Fatalf("parseRepositoryInstructions() unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("parseRepositoryInstructions() = %d skills, want 0 when file absent", len(skills))
	}
}

func TestParseRepositoryInstructions_Duplicate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "copilot-instructions.md"), []byte("# My Instructions\nHello"), 0o600); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	p := New(dir)
	seen := make(map[string]bool)
	seen["copilot-instructions"] = true

	skills, err := p.parseRepositoryInstructions(seen)
	if err != nil {
		t.Fatalf("parseRepositoryInstructions() unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("parseRepositoryInstructions() = %d skills, want 0 for seen duplicate", len(skills))
	}
}

func TestParseArtifactFile_ReadError(t *testing.T) {
	p := New(t.TempDir())
	_, err := p.parseArtifactFile("/nonexistent/file.md", model.SkillTypeSkill, false, "")
	if err == nil {
		t.Error("parseArtifactFile() expected error for nonexistent file, got nil")
	}
}

func TestParseArtifactFile_MalformedFrontmatter(t *testing.T) {
	dir := t.TempDir()
	badFile := filepath.Join(dir, "bad.md")
	// Valid-looking frontmatter delimiters but invalid YAML
	if err := os.WriteFile(badFile, []byte("---\nkey: [unclosed bracket\n---\nContent"), 0o600); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	p := New(dir)
	_, err := p.parseArtifactFile(badFile, model.SkillTypeSkill, false, "")
	if err == nil {
		t.Error("parseArtifactFile() expected error for malformed frontmatter, got nil")
	}
}

func TestParseArtifactFile_TypeField(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "typed.md")
	if err := os.WriteFile(f, []byte("---\nname: my-skill\ntype: prompt\n---\nContent"), 0o600); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	p := New(dir)
	skill, err := p.parseArtifactFile(f, model.SkillTypeSkill, false, "")
	if err != nil {
		t.Fatalf("parseArtifactFile() unexpected error: %v", err)
	}
	if skill.Type != model.SkillTypePrompt {
		t.Errorf("skill.Type = %q, want %q", skill.Type, model.SkillTypePrompt)
	}
}

func TestExtractTools_StringCSV(t *testing.T) {
	fm := map[string]any{"tools": "read, write, search"}
	got := extractTools(fm, "tools")
	want := []string{"read", "write", "search"}
	if len(got) != len(want) {
		t.Fatalf("extractTools() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("extractTools()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExtractTools_EmptyString(t *testing.T) {
	fm := map[string]any{"tools": "   "}
	got := extractTools(fm, "tools")
	if got != nil {
		t.Errorf("extractTools() = %v, want nil for blank string", got)
	}
}

func TestExtractTools_OtherType(t *testing.T) {
	fm := map[string]any{"tools": 42}
	got := extractTools(fm, "tools")
	if got != nil {
		t.Errorf("extractTools() = %v, want nil for non-string/slice value", got)
	}
}

func TestExtractTools_Missing(t *testing.T) {
	fm := map[string]any{}
	got := extractTools(fm, "tools")
	if got != nil {
		t.Errorf("extractTools() = %v, want nil for missing key", got)
	}
}

func TestParseInstructionFiles_DuplicateSkipped(t *testing.T) {
	dir := t.TempDir()
	instrDir := filepath.Join(dir, "instructions")
	if err := os.MkdirAll(instrDir, 0o750); err != nil {
		t.Fatalf("failed to create instructions dir: %v", err)
	}
	content := "---\nname: my-instruction\ndescription: Test\n---\nContent"
	if err := os.WriteFile(filepath.Join(instrDir, "my-instruction.instructions.md"), []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	p := New(dir)
	seen := map[string]bool{"my-instruction": true}
	skills, err := p.parseInstructionFiles(seen)
	if err != nil {
		t.Fatalf("parseInstructionFiles() unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("parseInstructionFiles() = %d skills, want 0 for seen duplicate", len(skills))
	}
}

func TestParsePromptFiles_DuplicateSkipped(t *testing.T) {
	dir := t.TempDir()
	promptDir := filepath.Join(dir, "prompts")
	if err := os.MkdirAll(promptDir, 0o750); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}
	content := "---\nname: my-prompt\ndescription: Test\n---\nContent"
	if err := os.WriteFile(filepath.Join(promptDir, "my-prompt.prompt.md"), []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	p := New(dir)
	seen := map[string]bool{"my-prompt": true}
	skills, err := p.parsePromptFiles(seen)
	if err != nil {
		t.Fatalf("parsePromptFiles() unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("parsePromptFiles() = %d skills, want 0 for seen duplicate", len(skills))
	}
}

func TestParseAgentFiles_DuplicateSkipped(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "agents")
	if err := os.MkdirAll(agentDir, 0o750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}
	content := "---\nname: my-agent\ndescription: Test\n---\nContent"
	if err := os.WriteFile(filepath.Join(agentDir, "my-agent.agent.md"), []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	p := New(dir)
	seen := map[string]bool{"my-agent": true}
	skills, err := p.parseAgentFiles(seen)
	if err != nil {
		t.Fatalf("parseAgentFiles() unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("parseAgentFiles() = %d skills, want 0 for seen duplicate", len(skills))
	}
}
