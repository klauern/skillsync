package sync

import (
	"strings"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
)

func TestNewTransformer(t *testing.T) {
	tr := NewTransformer()
	if tr == nil {
		t.Error("NewTransformer() returned nil")
	}
}

func TestTransformer_Transform(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "test-skill",
		Description: "A test skill",
		Platform:    model.ClaudeCode,
		Path:        "/source/test-skill.md",
		Tools:       []string{"Read", "Write"},
		Content:     "Test content",
		ModifiedAt:  time.Now(),
	}

	transformed, err := tr.Transform(skill, model.Cursor)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if transformed.Platform != model.Cursor {
		t.Errorf("Expected platform Cursor, got %s", transformed.Platform)
	}

	if transformed.Name != skill.Name {
		t.Errorf("Expected name %s, got %s", skill.Name, transformed.Name)
	}
}

func TestTransformer_TransformPath(t *testing.T) {
	tr := NewTransformer()

	tests := []struct {
		name       string
		sourcePath string
		skillName  string
		target     model.Platform
		expected   string
	}{
		{
			name:       "claude to cursor md",
			sourcePath: "/source/test.md",
			target:     model.Cursor,
			expected:   "test.md",
		},
		{
			name:       "cursor mdc preserved",
			sourcePath: "/source/test.mdc",
			target:     model.Cursor,
			expected:   "test.mdc",
		},
		{
			name:       "to claude code",
			sourcePath: "/source/test.mdc",
			target:     model.ClaudeCode,
			expected:   "test.md",
		},
		{
			name:       "agents to codex",
			sourcePath: "/source/AGENTS.md",
			target:     model.Codex,
			expected:   "AGENTS.md",
		},
		{
			name:       "skill directory to codex",
			sourcePath: "/source/my-skill/SKILL.md",
			skillName:  "my-skill",
			target:     model.Codex,
			expected:   "my-skill/SKILL.md",
		},
		{
			name:       "skill directory to claude",
			sourcePath: "/source/my-skill/SKILL.md",
			skillName:  "my-skill",
			target:     model.ClaudeCode,
			expected:   "my-skill/SKILL.md",
		},
		{
			name:       "prompt to codex skill file",
			sourcePath: "/source/review.md",
			skillName:  "review",
			target:     model.Codex,
			expected:   "review/SKILL.md",
		},
		{
			name:       "skill directory to pi agent",
			sourcePath: "/source/my-skill/SKILL.md",
			skillName:  "my-skill",
			target:     model.PiDev,
			expected:   "my-skill/SKILL.md",
		},
		{
			name:       "prompt to copilot prompt file",
			sourcePath: "/source/review.md",
			skillName:  "review",
			target:     model.Copilot,
			expected:   "prompts/review.prompt.md",
		},
		{
			name:       "agent to copilot agent file",
			sourcePath: "/source/skill.md",
			skillName:  "reviewer",
			target:     model.Copilot,
			expected:   "agents/reviewer.agent.md",
		},
		{
			name:       "standard skill to copilot skills directory",
			sourcePath: "/source/skill/SKILL.md",
			skillName:  "skill",
			target:     model.Copilot,
			expected:   "skill/SKILL.md",
		},
		{
			name:       "instructions to copilot instructions file",
			sourcePath: "/source/style.md",
			skillName:  "go-style",
			target:     model.Copilot,
			expected:   "instructions/go-style.instructions.md",
		},
		{
			name:       "claude instructions to copilot repo instructions",
			sourcePath: "/source/CLAUDE.md",
			skillName:  "claude",
			target:     model.Copilot,
			expected:   "copilot-instructions.md",
		},
		{
			name:       "system prompt to pidev append file",
			sourcePath: "/source/APPEND_SYSTEM.md",
			skillName:  "append-system",
			target:     model.PiDev,
			expected:   "APPEND_SYSTEM.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := model.Skill{Path: tt.sourcePath, Name: tt.skillName}
			if tt.name == "prompt to codex skill file" {
				skill.Type = model.SkillTypePrompt
				skill.Trigger = "/review"
			}
			if tt.name == "skill directory to claude" || tt.name == "standard skill to copilot skills directory" {
				skill.Type = model.SkillTypeSkill
			}
			if tt.name == "prompt to copilot prompt file" {
				skill.Type = model.SkillTypePrompt
				skill.Trigger = "/review"
			}
			if tt.name == "instructions to copilot instructions file" {
				skill.Metadata = map[string]string{model.MetadataKeyCopilotArtifact: model.CopilotArtifactInstructions}
			}
			if tt.name == "system prompt to pidev append file" {
				skill.Metadata = map[string]string{"type": "system-prompt", "mode": "append"}
			}
			result := tr.transformPath(skill, tt.target)
			if result != tt.expected {
				t.Errorf("transformPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNamedArtifactTargetPath(t *testing.T) {
	tests := []struct {
		name     string
		target   model.Platform
		expected string
	}{
		{name: "codex", target: model.Codex, expected: "review/SKILL.md"},
		{name: "gemini", target: model.Gemini, expected: "review/SKILL.md"},
		{name: "pidev", target: model.PiDev, expected: "review/SKILL.md"},
		{name: "cursor", target: model.Cursor, expected: "review.md"},
		{name: "claude", target: model.ClaudeCode, expected: "review.md"},
		{name: "copilot unsupported", target: model.Copilot, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := namedArtifactTargetPath("review", tt.target); got != tt.expected {
				t.Fatalf("namedArtifactTargetPath() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTransformer_TransformContent(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "test-skill",
		Description: "A test description",
		Content:     "The main content",
		Tools:       []string{"Read"},
	}

	// Transform to Claude Code (should include tools)
	content, err := tr.transformContent(skill, model.ClaudeCode, "test-skill.md")
	if err != nil {
		t.Fatalf("transformContent failed: %v", err)
	}

	if !strings.Contains(content, "name: test-skill") {
		t.Error("Content should contain name in frontmatter")
	}
	if !strings.Contains(content, "The main content") {
		t.Error("Content should contain main content")
	}
}

func TestTransformer_TransformContent_CodexSkillFile(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "test-skill",
		Description: "test description",
		Content:     "Main content",
	}

	content, err := tr.transformContent(skill, model.Codex, "test-skill/SKILL.md")
	if err != nil {
		t.Fatalf("transformContent failed: %v", err)
	}

	if !strings.HasPrefix(content, "---\n") {
		t.Error("Codex SKILL.md content should include frontmatter")
	}
	if !strings.Contains(content, "name: test-skill") {
		t.Error("Frontmatter should include name")
	}
	if !strings.Contains(content, "description: test description") {
		t.Error("Frontmatter should include description")
	}
}

func TestTransformer_TransformContent_PiAgentSkillFile(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "test-skill",
		Description: "test description",
		Content:     "Main content",
	}

	content, err := tr.transformContent(skill, model.PiDev, "test-skill/SKILL.md")
	if err != nil {
		t.Fatalf("transformContent failed: %v", err)
	}

	if !strings.HasPrefix(content, "---\n") {
		t.Error("Pi Agent SKILL.md content should include frontmatter")
	}
	if !strings.Contains(content, "name: test-skill") {
		t.Error("Frontmatter should include name")
	}
	if !strings.Contains(content, "description: test description") {
		t.Error("Frontmatter should include description")
	}
}

func TestTransformer_TransformContent_CodexAgents(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "agents",
		Description: "Agent instructions",
		Content:     "Main content",
	}

	content, err := tr.transformContent(skill, model.Codex, "AGENTS.md")
	if err != nil {
		t.Fatalf("transformContent failed: %v", err)
	}

	if strings.HasPrefix(content, "---\n") {
		t.Error("Codex AGENTS.md content should not include frontmatter")
	}
}

func TestTransformer_TransformContent_PiDevSystemPrompt(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "append-system",
		Description: "Append system instructions",
		Metadata: map[string]string{
			"type": "system-prompt",
			"mode": "append",
		},
		Content: "System prompt body",
	}

	content, err := tr.transformContent(skill, model.PiDev, "APPEND_SYSTEM.md")
	if err != nil {
		t.Fatalf("transformContent failed: %v", err)
	}

	if strings.HasPrefix(content, "---\n") {
		t.Fatal("Pi.dev system prompt content should not include frontmatter")
	}
	if content != "System prompt body" {
		t.Fatalf("content = %q, want body only", content)
	}
}

func TestTransformer_TransformContent_CopilotRepositoryInstructions(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name: "copilot-instructions",
		Metadata: map[string]string{
			model.MetadataKeyCopilotArtifact: model.CopilotArtifactRepositoryInstructions,
		},
		Content: "Always-on instructions",
	}

	content, err := tr.transformContent(skill, model.Copilot, "copilot-instructions.md")
	if err != nil {
		t.Fatalf("transformContent failed: %v", err)
	}

	if strings.HasPrefix(content, "---\n") {
		t.Fatal("Copilot repository instructions should not gain synthetic frontmatter")
	}
	if content != "Always-on instructions" {
		t.Fatalf("content = %q, want body only", content)
	}
}

func TestTransformer_BuildFrontmatter_ClaudeCode(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "test",
		Description: "desc",
		Tools:       []string{"Read", "Write"},
	}

	fm := tr.buildFrontmatter(skill, model.ClaudeCode)

	if fm["name"] != "test" {
		t.Error("Frontmatter should contain name")
	}
	if fm["description"] != "desc" {
		t.Error("Frontmatter should contain description")
	}
	if fm["tools"] == nil {
		t.Error("Claude Code frontmatter should contain tools")
	}
}

func TestTransformer_BuildFrontmatter_Cursor(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "test",
		Description: "desc",
		Metadata: map[string]string{
			"globs":       "*.ts",
			"alwaysApply": "true",
		},
	}

	fm := tr.buildFrontmatter(skill, model.Cursor)

	if fm["paths"] != "*.ts" {
		t.Error("Cursor frontmatter should map legacy globs to paths")
	}
	if _, ok := fm["globs"]; ok {
		t.Error("cross-harness Cursor frontmatter should not emit legacy globs")
	}
	if fm["alwaysApply"] != "true" {
		t.Error("Cursor frontmatter should contain alwaysApply")
	}
}

func TestTransformer_CursorSameHarnessPreservesRawLegacyGlobs(t *testing.T) {
	tr := NewTransformer()
	skill := model.Skill{
		Name:        "legacy",
		Description: "Legacy rule",
		Platform:    model.Cursor,
		Path:        "legacy.mdc",
		Content:     "Body",
		Metadata:    map[string]string{"globs": "[*.go]"},
		RawFrontmatter: map[string]any{
			"globs": []any{"*.go"},
		},
	}
	transformed, err := tr.Transform(skill, model.Cursor)
	if err != nil {
		t.Fatal(err)
	}
	result := parser.SplitFrontmatter([]byte(transformed.Content))
	fm, err := parser.ParseYAMLFrontmatter(result.Frontmatter)
	if err != nil {
		t.Fatal(err)
	}
	if fm["globs"] == nil {
		t.Fatalf("same-harness legacy globs were not preserved: %#v", fm)
	}
	if _, ok := fm["paths"]; ok {
		t.Fatalf("same-harness legacy round trip unexpectedly added paths: %#v", fm)
	}
}

func TestTransformer_BuildFrontmatter_Codex(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "test",
		Description: "desc",
		Type:        model.SkillTypePrompt,
		Trigger:     "/test",
	}

	fm := tr.buildFrontmatter(skill, model.Codex)

	// buildFrontmatter always returns a map (filtering is done via shouldIncludeFrontmatter)
	// For Codex, it includes name and description like other platforms
	if fm == nil {
		t.Error("Codex frontmatter should not be nil")
	}
	if fm["name"] != "test" {
		t.Error("Codex frontmatter should include name")
	}
	if fm["description"] != "desc" {
		t.Error("Codex frontmatter should include description")
	}
	if fm["type"] != "prompt" {
		t.Error("Codex frontmatter should include type for prompt artifacts")
	}
	if fm["trigger"] != "/test" {
		t.Error("Codex frontmatter should include trigger for prompt artifacts")
	}
}

func TestTransformer_BuildFrontmatter_CopilotPrompt(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "review",
		Description: "Review code",
		Tools:       []string{"read", "search"},
		Type:        model.SkillTypePrompt,
		Trigger:     "/review",
		Metadata: map[string]string{
			model.MetadataKeyCopilotArtifact: model.CopilotArtifactPrompt,
			"model":                          "gpt-4o",
		},
	}

	fm := tr.buildFrontmatter(skill, model.Copilot)

	if fm["name"] != "review" {
		t.Error("Copilot frontmatter should include name")
	}
	if fm["tools"] == nil {
		t.Error("Copilot frontmatter should include tools")
	}
	if _, exists := fm[model.MetadataKeyCopilotArtifact]; exists {
		t.Error("Copilot frontmatter should not include internal artifact metadata")
	}
	if fm["model"] != "gpt-4o" {
		t.Error("Copilot frontmatter should preserve model")
	}
}

func TestTransformer_TransformMetadata(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Platform: model.Cursor,
		Metadata: map[string]string{
			"globs":       "*.ts",
			"alwaysApply": "true",
			"custom":      "value",
		},
	}

	// Transform to Claude Code - should remove Cursor-specific fields
	metadata := tr.transformMetadata(skill, model.ClaudeCode)

	if _, exists := metadata["globs"]; exists {
		t.Error("Claude Code metadata should not contain globs")
	}
	if _, exists := metadata["alwaysApply"]; exists {
		t.Error("Claude Code metadata should not contain alwaysApply")
	}
	if metadata["custom"] != "value" {
		t.Error("Custom metadata should be preserved")
	}
}

func TestTransformer_TransformMetadata_Codex(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Platform: model.ClaudeCode,
		Metadata: map[string]string{
			"custom": "value",
		},
	}

	metadata := tr.transformMetadata(skill, model.Codex)

	if metadata["source_platform"] != "claude-code" {
		t.Error("Codex metadata should contain source_platform")
	}
}

func TestTransformer_Transform_CopilotInstructionToCursor(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "react-standards",
		Description: "Copilot instruction fixture",
		Platform:    model.ClaudeCode,
		Path:        "/source/react.instructions.md",
		Content:     "Use hooks and colocated tests.",
		Metadata: map[string]string{
			"applyTo": "**/*.tsx",
			"model":   "GPT-4o",
		},
	}

	transformed, err := tr.Transform(skill, model.Cursor)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if transformed.Path != "react.instructions.md" {
		t.Fatalf("Path = %q, want react.instructions.md", transformed.Path)
	}
	if _, ok := transformed.Metadata["applyTo"]; ok {
		t.Fatal("Cursor metadata should not retain applyTo after mapping to paths")
	}
	if transformed.Metadata["paths"] != "**/*.tsx" {
		t.Fatalf("paths metadata = %q, want **/*.tsx", transformed.Metadata["paths"])
	}
	if _, ok := transformed.Metadata["globs"]; ok {
		t.Fatal("cross-harness Cursor metadata should not retain legacy globs")
	}
	if transformed.Metadata["model"] != "GPT-4o" {
		t.Fatalf("model metadata = %q, want GPT-4o", transformed.Metadata["model"])
	}

	result := parser.SplitFrontmatter([]byte(transformed.Content))
	if !result.HasFrontmatter {
		t.Fatal("expected transformed content to include frontmatter")
	}
	fm, err := parser.ParseYAMLFrontmatter(result.Frontmatter)
	if err != nil {
		t.Fatalf("parse frontmatter: %v", err)
	}
	if got := fm["paths"]; got != "**/*.tsx" {
		t.Fatalf("frontmatter paths = %v, want **/*.tsx", got)
	}
	if got := fm["model"]; got != "GPT-4o" {
		t.Fatalf("frontmatter model = %v, want GPT-4o", got)
	}
}

func TestTransformer_Transform_CopilotPromptToCodexPreservesLiteralText(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Name:        "review",
		Description: "Copilot prompt fixture",
		Platform:    model.ClaudeCode,
		Path:        "/source/review.prompt.md",
		Type:        model.SkillTypePrompt,
		Trigger:     "/review",
		Content:     "Review ${file} and ask ${input:path} for more context.\n\nReference #file:docs/spec.md.",
		Metadata: map[string]string{
			"argument-hint": "<path>",
			"model":         "GPT-4o",
		},
	}

	transformed, err := tr.Transform(skill, model.Codex)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if transformed.Path != "review/SKILL.md" {
		t.Fatalf("Path = %q, want review/SKILL.md", transformed.Path)
	}
	if !strings.Contains(transformed.Content, "${input:path}") {
		t.Fatal("transformed content should preserve Copilot input interpolation literally")
	}
	if !strings.Contains(transformed.Content, "#file:docs/spec.md") {
		t.Fatal("transformed content should preserve Copilot file references literally")
	}
	if transformed.Metadata["argument-hint"] != "<path>" {
		t.Fatalf("argument-hint metadata = %q, want <path>", transformed.Metadata["argument-hint"])
	}
	if transformed.Metadata["model"] != "GPT-4o" {
		t.Fatalf("model metadata = %q, want GPT-4o", transformed.Metadata["model"])
	}
}

func TestTransformer_TransformMetadata_DropsCopilotAgentOnlyFields(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Platform: model.ClaudeCode,
		Metadata: map[string]string{
			"handoffs":    "[{\"agent\":\"implementer\"}]",
			"target":      "vscode",
			"mcp-servers": "{\"github\":{\"command\":\"gh\"}}",
			"model":       "GPT-4o",
		},
	}

	metadata := tr.transformMetadata(skill, model.Codex)

	for _, key := range []string{"handoffs", "target", "mcp-servers"} {
		if _, ok := metadata[key]; ok {
			t.Fatalf("metadata should drop %q for Codex target", key)
		}
	}
	if metadata["model"] != "GPT-4o" {
		t.Fatalf("model metadata = %q, want GPT-4o", metadata["model"])
	}
}

func TestTransformer_CanTransform(t *testing.T) {
	tr := NewTransformer()

	tests := []struct {
		source   model.Platform
		target   model.Platform
		expected bool
	}{
		{model.ClaudeCode, model.Cursor, true},
		{model.Cursor, model.ClaudeCode, true},
		{model.ClaudeCode, model.Codex, true},
		{model.Platform("invalid"), model.Cursor, false},
		{model.ClaudeCode, model.Platform("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.source)+"->"+string(tt.target), func(t *testing.T) {
			result := tr.CanTransform(tt.source, tt.target)
			if result != tt.expected {
				t.Errorf("CanTransform() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTransformer_MergeContent(t *testing.T) {
	tr := NewTransformer()

	source := "Source content"
	target := "Target content"
	name := "test-skill"

	merged := tr.MergeContent(source, target, name)

	if !strings.Contains(merged, "Target content") {
		t.Error("Merged content should contain target content")
	}
	if !strings.Contains(merged, "Source content") {
		t.Error("Merged content should contain source content")
	}
	if !strings.Contains(merged, "Merged from: test-skill") {
		t.Error("Merged content should contain merge header")
	}
	if !strings.Contains(merged, "---") {
		t.Error("Merged content should contain separator")
	}
}

func TestTransformer_TransformMetadata_CopilotToCopilotPreservesFields(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Platform: model.Copilot,
		Name:     "my-agent",
		Metadata: map[string]string{
			"applyTo":     "**/*.go",
			"target":      "vscode",
			"handoffs":    "[{\"agent\":\"implementer\"}]",
			"mcp-servers": "{\"github\":{\"command\":\"gh\"}}",
			"model":       "GPT-4o",
		},
	}

	metadata := tr.transformMetadata(skill, model.Copilot)

	for _, key := range []string{"applyTo", "target", "handoffs", "mcp-servers", "model"} {
		if _, ok := metadata[key]; !ok {
			t.Errorf("copilot->copilot round-trip: metadata key %q should be preserved", key)
		}
	}
}

func TestTransformer_TransformMetadata_CopilotToClaudeDropsNothing_ButWarns(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Platform: model.Copilot,
		Name:     "my-agent",
		Metadata: map[string]string{
			"applyTo":     "**/*.go",
			"handoffs":    "[{\"agent\":\"implementer\"}]",
			"mcp-servers": "{\"github\":{\"command\":\"gh\"}}",
			"model":       "GPT-4o",
		},
	}

	// transformMetadata does not delete these keys for ClaudeCode target —
	// they pass through as unknown metadata (a warn is emitted for the lossiness).
	metadata := tr.transformMetadata(skill, model.ClaudeCode)

	// Cursor-specific keys should still be absent (unrelated to copilot fields)
	for _, key := range []string{"globs", "alwaysApply"} {
		if _, ok := metadata[key]; ok {
			t.Errorf("metadata key %q should not be injected for ClaudeCode target", key)
		}
	}
	// model should survive
	if metadata["model"] != "GPT-4o" {
		t.Errorf("model metadata = %q, want GPT-4o", metadata["model"])
	}
}

func TestTransformer_TransformMetadata_CopilotToCodexDropsCopilotFields(t *testing.T) {
	tr := NewTransformer()

	skill := model.Skill{
		Platform: model.Copilot,
		Name:     "my-agent",
		Metadata: map[string]string{
			"applyTo":     "**/*.go",
			"target":      "vscode",
			"handoffs":    "[{\"agent\":\"implementer\"}]",
			"mcp-servers": "{\"github\":{\"command\":\"gh\"}}",
			"model":       "GPT-4o",
		},
	}

	metadata := tr.transformMetadata(skill, model.Codex)

	for _, key := range []string{"handoffs", "target", "mcp-servers"} {
		if _, ok := metadata[key]; ok {
			t.Errorf("metadata key %q should be dropped for Codex target", key)
		}
	}
	// model and source_platform should survive
	if metadata["model"] != "GPT-4o" {
		t.Errorf("model = %q, want GPT-4o", metadata["model"])
	}
	if metadata["source_platform"] != string(model.Copilot) {
		t.Errorf("source_platform = %q, want %q", metadata["source_platform"], string(model.Copilot))
	}
}
