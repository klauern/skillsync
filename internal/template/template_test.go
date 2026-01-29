package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestNew(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if gen == nil {
		t.Fatal("New() returned nil generator")
	}

	// Should have loaded built-in templates
	templates := gen.ListTemplates()
	if len(templates) != 3 {
		t.Errorf("Expected 3 built-in templates, got %d", len(templates))
	}
}

func TestGenerate_CommandWrapper(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	data := TemplateData{
		Name:        "test-command",
		Description: "A test command wrapper",
		Platform:    "claude-code",
		Scope:       "repo",
		Tools:       []string{"Bash", "Read"},
	}

	content, err := gen.Generate(CommandWrapper, data)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify content contains key elements
	if !strings.Contains(content, "name: test-command") {
		t.Error("Generated content missing skill name")
	}
	if !strings.Contains(content, "description: A test command wrapper") {
		t.Error("Generated content missing description")
	}
	if !strings.Contains(content, "scope: repo") {
		t.Error("Generated content missing scope")
	}
	if !strings.Contains(content, "- Bash") {
		t.Error("Generated content missing Bash tool")
	}
	if !strings.Contains(content, "# test-command") {
		t.Error("Generated content missing markdown header")
	}
}

func TestGenerate_Workflow(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	data := TemplateData{
		Name:        "test-workflow",
		Description: "A test workflow",
		Platform:    "cursor",
		Scope:       "user",
		Tools:       []string{"Bash", "Read", "Write"},
	}

	content, err := gen.Generate(Workflow, data)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify workflow-specific content
	if !strings.Contains(content, "orchestrates multiple steps") {
		t.Error("Generated content missing workflow description")
	}
	if !strings.Contains(content, "Step 1: Preparation") {
		t.Error("Generated content missing workflow steps")
	}
}

func TestGenerate_Utility(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	data := TemplateData{
		Name:        "test-util",
		Description: "A test utility",
		Platform:    "codex",
		Scope:       "repo",
		Tools:       []string{"Read", "Write"},
	}

	content, err := gen.Generate(Utility, data)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify utility-specific content
	if !strings.Contains(content, "utility skill") {
		t.Error("Generated content missing utility description")
	}
	if !strings.Contains(content, "Data Processing") {
		t.Error("Generated content missing utility features")
	}
}

func TestValidateGenerated(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	data := TemplateData{
		Name:        "valid-skill",
		Description: "A valid skill",
		Platform:    "claude-code",
		Scope:       "repo",
		Tools:       []string{"Bash"},
	}

	content, err := gen.Generate(CommandWrapper, data)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Should validate successfully
	if err := gen.ValidateGenerated(content); err != nil {
		t.Errorf("ValidateGenerated() failed for valid content: %v", err)
	}

	// Invalid content (missing frontmatter) should fail
	invalidContent := "# Just a heading\n\nNo frontmatter here."
	if err := gen.ValidateGenerated(invalidContent); err == nil {
		t.Error("ValidateGenerated() should fail for invalid content")
	}
}

func TestCreateSkillFile(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	gen, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	data := TemplateData{
		Name:        "test-skill",
		Description: "A test skill",
		Platform:    "claude-code",
		Scope:       "repo",
		Tools:       []string{"Bash", "Read"},
	}

	// Override the base path for testing
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Create skill file
	skillPath, err := gen.CreateSkillFile(CommandWrapper, data, model.ClaudeCode, model.ScopeRepo)
	if err != nil {
		t.Fatalf("CreateSkillFile() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Errorf("Skill file was not created at %s", skillPath)
	}

	// Verify content
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("Failed to read created skill file: %v", err)
	}

	if !strings.Contains(string(content), "name: test-skill") {
		t.Error("Created file missing skill name")
	}

	// Verify directory structure
	skillDir := filepath.Dir(skillPath)
	if filepath.Base(skillDir) != "test-skill" {
		t.Errorf("Expected skill directory name 'test-skill', got %s", filepath.Base(skillDir))
	}
}

func TestParseTemplateType(t *testing.T) {
	tests := []struct {
		input    string
		expected TemplateType
		wantErr  bool
	}{
		{"command-wrapper", CommandWrapper, false},
		{"command", CommandWrapper, false},
		{"cmd", CommandWrapper, false},
		{"workflow", Workflow, false},
		{"utility", Utility, false},
		{"util", Utility, false},
		{"WORKFLOW", Workflow, false},
		{"  command  ", CommandWrapper, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseTemplateType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTemplateType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseTemplateType(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLoadCustomTemplate(t *testing.T) {
	// Create a temporary template file
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "custom.md")

	customTemplate := `---
name: {{.Name}}
description: {{.Description}}
---

# Custom Template

This is a custom template for {{.Name}}.
`

	if err := os.WriteFile(templatePath, []byte(customTemplate), 0644); err != nil {
		t.Fatalf("Failed to create test template file: %v", err)
	}

	gen, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Load custom template
	if err := gen.LoadCustomTemplate("custom", templatePath); err != nil {
		t.Fatalf("LoadCustomTemplate() failed: %v", err)
	}

	// Verify it can be used
	data := TemplateData{
		Name:        "custom-skill",
		Description: "A custom skill",
	}

	content, err := gen.Generate(TemplateType("custom"), data)
	if err != nil {
		t.Fatalf("Generate() with custom template failed: %v", err)
	}

	if !strings.Contains(content, "custom-skill") {
		t.Error("Custom template did not render correctly")
	}
	if !strings.Contains(content, "This is a custom template") {
		t.Error("Custom template content missing")
	}
}

func TestListTemplates(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	templates := gen.ListTemplates()

	// Should have exactly 3 built-in templates
	if len(templates) != 3 {
		t.Errorf("Expected 3 templates, got %d", len(templates))
	}

	// Check that all expected templates are present
	expectedTemplates := map[string]bool{
		"command-wrapper": false,
		"workflow":        false,
		"utility":         false,
	}

	for _, tmpl := range templates {
		if _, ok := expectedTemplates[tmpl]; ok {
			expectedTemplates[tmpl] = true
		}
	}

	for name, found := range expectedTemplates {
		if !found {
			t.Errorf("Expected template %q not found in list", name)
		}
	}
}

func TestTemplateData_DefaultYear(t *testing.T) {
	gen, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	data := TemplateData{
		Name:        "test-skill",
		Description: "Test",
		Year:        0, // Not set
	}

	// Generate should set the year automatically
	_, err = gen.Generate(CommandWrapper, data)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}
	// Year is set internally during generation
}
