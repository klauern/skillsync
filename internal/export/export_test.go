package export

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/model"
)

func TestFormat_IsValid(t *testing.T) {
	tests := []struct {
		format Format
		valid  bool
	}{
		{FormatJSON, true},
		{FormatYAML, true},
		{FormatMarkdown, true},
		{Format("invalid"), false},
		{Format(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			if got := tt.format.IsValid(); got != tt.valid {
				t.Errorf("Format(%q).IsValid() = %v, want %v", tt.format, got, tt.valid)
			}
		})
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Format
		wantErr bool
	}{
		{"json", "json", FormatJSON, false},
		{"JSON uppercase", "JSON", FormatJSON, false},
		{"yaml", "yaml", FormatYAML, false},
		{"markdown", "markdown", FormatMarkdown, false},
		{"with spaces", "  json  ", FormatJSON, false},
		{"invalid", "xml", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestAllFormats(t *testing.T) {
	formats := AllFormats()
	if len(formats) != 3 {
		t.Errorf("AllFormats() returned %d formats, want 3", len(formats))
	}

	expected := map[Format]bool{
		FormatJSON:     true,
		FormatYAML:     true,
		FormatMarkdown: true,
	}

	for _, f := range formats {
		if !expected[f] {
			t.Errorf("AllFormats() contains unexpected format %q", f)
		}
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Format != FormatJSON {
		t.Errorf("DefaultOptions().Format = %v, want %v", opts.Format, FormatJSON)
	}
	if !opts.Pretty {
		t.Error("DefaultOptions().Pretty = false, want true")
	}
	if !opts.IncludeMetadata {
		t.Error("DefaultOptions().IncludeMetadata = false, want true")
	}
}

func TestExporter_ExportJSON(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "test-skill",
			Description: "A test skill",
			Platform:    model.ClaudeCode,
			Path:        "/path/to/skill.md",
			Tools:       []string{"tool1", "tool2"},
			Content:     "Test content",
			ModifiedAt:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	opts := Options{
		Format:          FormatJSON,
		Pretty:          true,
		IncludeMetadata: true,
	}

	exporter := New(opts)
	var buf bytes.Buffer
	err := exporter.Export(skills, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	// Parse the output
	var result []exportSkill
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 skill, got %d", len(result))
	}

	if result[0].Name != "test-skill" {
		t.Errorf("Name = %q, want %q", result[0].Name, "test-skill")
	}
	if result[0].Platform != "claude-code" {
		t.Errorf("Platform = %q, want %q", result[0].Platform, "claude-code")
	}
}

func TestExporter_ExportJSON_Compact(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "test-skill",
			Platform: model.ClaudeCode,
			Content:  "Test content",
		},
	}

	opts := Options{
		Format: FormatJSON,
		Pretty: false,
	}

	exporter := New(opts)
	var buf bytes.Buffer
	err := exporter.Export(skills, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	// Compact JSON should be on one line
	output := buf.String()
	if strings.Count(output, "\n") > 1 {
		t.Errorf("Compact JSON should have minimal newlines, got: %q", output)
	}
}

func TestExporter_ExportYAML(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "yaml-skill",
			Description: "A YAML test skill",
			Platform:    model.Cursor,
			Content:     "YAML content",
		},
	}

	opts := Options{
		Format:          FormatYAML,
		Pretty:          true,
		IncludeMetadata: true,
	}

	exporter := New(opts)
	var buf bytes.Buffer
	err := exporter.Export(skills, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	// Parse the output
	var result []exportSkill
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse YAML output: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 skill, got %d", len(result))
	}

	if result[0].Name != "yaml-skill" {
		t.Errorf("Name = %q, want %q", result[0].Name, "yaml-skill")
	}
}

func TestExporter_ExportMarkdown(t *testing.T) {
	skills := []model.Skill{
		{
			Name:        "md-skill",
			Description: "A Markdown skill",
			Platform:    model.Codex,
			Content:     "Markdown content here",
		},
	}

	opts := Options{
		Format:          FormatMarkdown,
		IncludeMetadata: true,
	}

	exporter := New(opts)
	var buf bytes.Buffer
	err := exporter.Export(skills, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	output := buf.String()

	// Check expected content
	if !strings.Contains(output, "# Exported Skills") {
		t.Error("Markdown should contain title")
	}
	if !strings.Contains(output, "## md-skill") {
		t.Error("Markdown should contain skill name as heading")
	}
	if !strings.Contains(output, "A Markdown skill") {
		t.Error("Markdown should contain description")
	}
	if !strings.Contains(output, "codex") {
		t.Error("Markdown should contain platform")
	}
	if !strings.Contains(output, "Markdown content here") {
		t.Error("Markdown should contain skill content")
	}
}

func TestExporter_FilterByPlatform(t *testing.T) {
	skills := []model.Skill{
		{Name: "claude-skill", Platform: model.ClaudeCode, Content: "a"},
		{Name: "cursor-skill", Platform: model.Cursor, Content: "b"},
		{Name: "codex-skill", Platform: model.Codex, Content: "c"},
	}

	opts := Options{
		Format:   FormatJSON,
		Pretty:   false,
		Platform: model.ClaudeCode,
	}

	exporter := New(opts)
	var buf bytes.Buffer
	err := exporter.Export(skills, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	var result []exportSkill
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 skill after filtering, got %d", len(result))
	}

	if result[0].Name != "claude-skill" {
		t.Errorf("Filtered skill name = %q, want %q", result[0].Name, "claude-skill")
	}
}

func TestExporter_ExcludeMetadata(t *testing.T) {
	skills := []model.Skill{
		{
			Name:       "test-skill",
			Platform:   model.ClaudeCode,
			Path:       "/path/to/skill.md",
			Tools:      []string{"tool1"},
			Content:    "Test content",
			ModifiedAt: time.Now(),
		},
	}

	opts := Options{
		Format:          FormatJSON,
		Pretty:          true,
		IncludeMetadata: false,
	}

	exporter := New(opts)
	var buf bytes.Buffer
	err := exporter.Export(skills, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	var result []exportSkill
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if result[0].Path != "" {
		t.Error("Path should be empty when metadata excluded")
	}
	if result[0].Tools != nil {
		t.Error("Tools should be nil when metadata excluded")
	}
	if result[0].ModifiedAt != "" {
		t.Error("ModifiedAt should be empty when metadata excluded")
	}
}

func TestExporter_ExportSingle(t *testing.T) {
	skill := model.Skill{
		Name:     "single-skill",
		Platform: model.ClaudeCode,
		Content:  "Single skill content",
	}

	opts := Options{
		Format: FormatJSON,
		Pretty: false,
	}

	exporter := New(opts)
	var buf bytes.Buffer
	err := exporter.ExportSingle(skill, &buf)
	if err != nil {
		t.Fatalf("ExportSingle() error = %v", err)
	}

	var result []exportSkill
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 skill, got %d", len(result))
	}
}

func TestExporter_EmptySkills(t *testing.T) {
	opts := Options{
		Format: FormatJSON,
	}

	exporter := New(opts)
	var buf bytes.Buffer
	err := exporter.Export([]model.Skill{}, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if buf.String() != "[]\n" {
		t.Errorf("Empty skills should produce empty array, got: %q", buf.String())
	}
}

func TestExporter_MarkdownEmptyContent(t *testing.T) {
	skills := []model.Skill{
		{
			Name:     "empty-content-skill",
			Platform: model.ClaudeCode,
			Content:  "",
		},
	}

	opts := Options{
		Format: FormatMarkdown,
	}

	exporter := New(opts)
	var buf bytes.Buffer
	err := exporter.Export(skills, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if !strings.Contains(buf.String(), "*No content*") {
		t.Error("Empty content should show 'No content' message in Markdown")
	}
}
