package copilot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestNew(t *testing.T) {
	tests := map[string]struct {
		basePath string
		want     string
	}{
		"empty path uses current directory": {
			basePath: "",
			want:     ".",
		},
		"custom path preserved": {
			basePath: "/custom/path",
			want:     "/custom/path",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p := New(tt.basePath)
			if p.basePath != tt.want {
				t.Fatalf("New(%q).basePath = %q, want %q", tt.basePath, p.basePath, tt.want)
			}
		})
	}
}

func TestParserPlatformAndDefaultPath(t *testing.T) {
	p := New("")
	if got := p.Platform(); got != model.Copilot {
		t.Fatalf("Platform() = %q, want %q", got, model.Copilot)
	}
	if got := p.DefaultPath(); got != "." {
		t.Fatalf("DefaultPath() = %q, want %q", got, ".")
	}
}

func TestParserParse(t *testing.T) {
	tmpDir := t.TempDir()

	files := map[string]string{
		".github/copilot-instructions.md": `# Repository defaults

Always explain repository conventions.`,
		".github/instructions/go.instructions.md": `---
description: Go file guidance
applyTo: "**/*.go"
excludeAgent: code-review
---

# Go instructions

Use table-driven tests.`,
		".github/instructions/docs.instructions.md": `---
name: docs-guidance
description: Markdown guidance
---

Prefer concise prose.`,
	}

	writeFiles(t, tmpDir, files)

	skills, err := New(tmpDir).Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 3 {
		t.Fatalf("Parse() returned %d skills, want 3", len(skills))
	}

	byName := make(map[string]model.Skill, len(skills))
	for _, skill := range skills {
		byName[skill.Name] = skill
	}

	repo := byName["copilot-instructions"]
	if repo.Platform != model.Copilot {
		t.Fatalf("repo Platform = %q, want %q", repo.Platform, model.Copilot)
	}
	if repo.Metadata["type"] != "repository-instructions" {
		t.Fatalf("repo metadata type = %q, want repository-instructions", repo.Metadata["type"])
	}

	goInstructions := byName["go"]
	if goInstructions.Description != "Go file guidance" {
		t.Fatalf("go Description = %q, want %q", goInstructions.Description, "Go file guidance")
	}
	if goInstructions.Metadata["applyTo"] != "**/*.go" {
		t.Fatalf("go applyTo = %q, want %q", goInstructions.Metadata["applyTo"], "**/*.go")
	}
	if goInstructions.Metadata["excludeAgent"] != "code-review" {
		t.Fatalf("go excludeAgent = %q, want %q", goInstructions.Metadata["excludeAgent"], "code-review")
	}

	docsInstructions := byName["docs-guidance"]
	if docsInstructions.Description != "Markdown guidance" {
		t.Fatalf("docs Description = %q, want %q", docsInstructions.Description, "Markdown guidance")
	}
}

func TestParserParseWithoutFrontmatterUsesFilename(t *testing.T) {
	tmpDir := t.TempDir()
	writeFiles(t, tmpDir, map[string]string{
		".github/instructions/python.instructions.md": "Use Ruff and mypy.\n",
	})

	skills, err := New(tmpDir).Parse()
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

func TestParserParseRepositoryAndScopedInstructionsCoexist(t *testing.T) {
	tmpDir := t.TempDir()
	writeFiles(t, tmpDir, map[string]string{
		".github/copilot-instructions.md": "# Repository instructions\n",
		".github/instructions/backend.instructions.md": `---
applyTo: "**/*.go"
---

Backend rules.`,
	})

	skills, err := New(tmpDir).Parse()
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("Parse() returned %d skills, want 2", len(skills))
	}

	foundRepo := false
	foundScoped := false
	for _, skill := range skills {
		switch skill.Name {
		case "copilot-instructions":
			foundRepo = true
		case "backend":
			foundScoped = true
		}
	}

	if !foundRepo || !foundScoped {
		t.Fatalf("expected repository and scoped instructions to coexist, got %+v", skills)
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

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()

	for relPath, content := range files {
		fullPath := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", filepath.Dir(fullPath), err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write %q: %v", fullPath, err)
		}
	}
}
