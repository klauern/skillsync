package export

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
)

// Format represents the output format for exported skills.
type Format string

const (
	// FormatJSON exports skills as JSON.
	FormatJSON Format = "json"
	// FormatYAML exports skills as YAML.
	FormatYAML Format = "yaml"
	// FormatMarkdown exports skills as Markdown.
	FormatMarkdown Format = "markdown"
)

// IsValid returns true if the format is recognized.
func (f Format) IsValid() bool {
	switch f {
	case FormatJSON, FormatYAML, FormatMarkdown:
		return true
	default:
		return false
	}
}

// String returns the string representation of the format.
func (f Format) String() string {
	return string(f)
}

// AllFormats returns all supported export formats.
func AllFormats() []Format {
	return []Format{FormatJSON, FormatYAML, FormatMarkdown}
}

// ParseFormat parses a string into a Format.
func ParseFormat(s string) (Format, error) {
	format := Format(strings.ToLower(strings.TrimSpace(s)))
	if !format.IsValid() {
		return "", fmt.Errorf("unsupported format %q (valid: json, yaml, markdown)", s)
	}
	return format, nil
}

// Options configures export behavior.
type Options struct {
	// Format specifies the output format.
	Format Format
	// Pretty enables pretty-printing for JSON/YAML.
	Pretty bool
	// IncludeMetadata includes metadata fields in the export.
	IncludeMetadata bool
	// Platform filters skills by platform (empty means all).
	Platform model.Platform
}

// DefaultOptions returns the default export options.
func DefaultOptions() Options {
	return Options{
		Format:          FormatJSON,
		Pretty:          true,
		IncludeMetadata: true,
	}
}

// Exporter handles exporting skills to different formats.
type Exporter struct {
	opts Options
}

// New creates a new Exporter with the given options.
func New(opts Options) *Exporter {
	return &Exporter{opts: opts}
}

// Export exports the given skills to the writer in the configured format.
func (e *Exporter) Export(skills []model.Skill, w io.Writer) error {
	defer logging.Timer("export")()

	logging.Debug("starting export",
		slog.String("format", string(e.opts.Format)),
		logging.Count(len(skills)),
		logging.Platform(string(e.opts.Platform)),
		logging.Operation("export"),
	)

	// Filter by platform if specified
	filtered := e.filterByPlatform(skills)

	if len(filtered) != len(skills) {
		logging.Debug("skills filtered by platform",
			logging.Count(len(filtered)),
			slog.Int("original", len(skills)),
		)
	}

	var err error
	switch e.opts.Format {
	case FormatJSON:
		err = e.exportJSON(filtered, w)
	case FormatYAML:
		err = e.exportYAML(filtered, w)
	case FormatMarkdown:
		err = e.exportMarkdown(filtered, w)
	default:
		err = fmt.Errorf("unsupported format: %s", e.opts.Format)
	}

	if err != nil {
		logging.Error("export failed",
			slog.String("format", string(e.opts.Format)),
			logging.Err(err),
		)
		return err
	}

	logging.Info("export completed successfully",
		slog.String("format", string(e.opts.Format)),
		logging.Count(len(filtered)),
	)

	return nil
}

// ExportSingle exports a single skill to the writer.
func (e *Exporter) ExportSingle(skill model.Skill, w io.Writer) error {
	return e.Export([]model.Skill{skill}, w)
}

// filterByPlatform filters skills by the configured platform.
func (e *Exporter) filterByPlatform(skills []model.Skill) []model.Skill {
	if e.opts.Platform == "" {
		return skills
	}

	var filtered []model.Skill
	for _, skill := range skills {
		if skill.Platform == e.opts.Platform {
			filtered = append(filtered, skill)
		}
	}
	return filtered
}

// exportSkill is an internal representation for export.
type exportSkill struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Platform    string            `json:"platform" yaml:"platform"`
	Path        string            `json:"path,omitempty" yaml:"path,omitempty"`
	Tools       []string          `json:"tools,omitempty" yaml:"tools,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Content     string            `json:"content" yaml:"content"`
	ModifiedAt  string            `json:"modified_at,omitempty" yaml:"modified_at,omitempty"`
}

// toExportSkill converts a model.Skill to exportSkill.
func (e *Exporter) toExportSkill(skill model.Skill) exportSkill {
	es := exportSkill{
		Name:        skill.Name,
		Description: skill.Description,
		Platform:    string(skill.Platform),
		Content:     skill.Content,
	}

	if e.opts.IncludeMetadata {
		es.Path = skill.Path
		es.Tools = skill.Tools
		es.Metadata = skill.Metadata
		if !skill.ModifiedAt.IsZero() {
			es.ModifiedAt = skill.ModifiedAt.Format("2006-01-02T15:04:05Z07:00")
		}
	}

	return es
}

// exportJSON exports skills as JSON.
func (e *Exporter) exportJSON(skills []model.Skill, w io.Writer) error {
	logging.Debug("exporting as JSON",
		logging.Count(len(skills)),
		slog.Bool("pretty", e.opts.Pretty),
	)

	exported := make([]exportSkill, len(skills))
	for i, skill := range skills {
		exported[i] = e.toExportSkill(skill)
	}

	encoder := json.NewEncoder(w)
	if e.opts.Pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(exported)
}

// exportYAML exports skills as YAML.
func (e *Exporter) exportYAML(skills []model.Skill, w io.Writer) error {
	logging.Debug("exporting as YAML",
		logging.Count(len(skills)),
		slog.Bool("pretty", e.opts.Pretty),
	)

	exported := make([]exportSkill, len(skills))
	for i, skill := range skills {
		exported[i] = e.toExportSkill(skill)
	}

	encoder := yaml.NewEncoder(w)
	if e.opts.Pretty {
		encoder.SetIndent(2)
	}
	if err := encoder.Encode(exported); err != nil {
		_ = encoder.Close()
		return err
	}
	return encoder.Close()
}

// exportMarkdown exports skills as Markdown.
func (e *Exporter) exportMarkdown(skills []model.Skill, w io.Writer) error {
	logging.Debug("exporting as Markdown", logging.Count(len(skills)))

	var sb strings.Builder

	sb.WriteString("# Exported Skills\n\n")
	sb.WriteString(fmt.Sprintf("Total: %d skill(s)\n\n", len(skills)))

	for i, skill := range skills {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}
		sb.WriteString(e.formatMarkdownSkill(skill))
	}

	_, err := w.Write([]byte(sb.String()))
	return err
}

// formatMarkdownSkill formats a single skill as Markdown.
func (e *Exporter) formatMarkdownSkill(skill model.Skill) string {
	var sb strings.Builder

	// Title
	sb.WriteString(fmt.Sprintf("## %s\n\n", skill.Name))

	// Description
	if skill.Description != "" {
		sb.WriteString(fmt.Sprintf("*%s*\n\n", skill.Description))
	}

	// Metadata table
	sb.WriteString("| Property | Value |\n")
	sb.WriteString("|----------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Platform | %s |\n", skill.Platform))

	if e.opts.IncludeMetadata {
		if skill.Path != "" {
			sb.WriteString(fmt.Sprintf("| Path | `%s` |\n", skill.Path))
		}
		if len(skill.Tools) > 0 {
			sb.WriteString(fmt.Sprintf("| Tools | %s |\n", strings.Join(skill.Tools, ", ")))
		}
		if !skill.ModifiedAt.IsZero() {
			sb.WriteString(fmt.Sprintf("| Modified | %s |\n", skill.ModifiedAt.Format("2006-01-02 15:04:05")))
		}
	}

	sb.WriteString("\n")

	// Content
	sb.WriteString("### Content\n\n")
	if strings.TrimSpace(skill.Content) != "" {
		sb.WriteString("```\n")
		sb.WriteString(skill.Content)
		if !strings.HasSuffix(skill.Content, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("```\n")
	} else {
		sb.WriteString("*No content*\n")
	}

	sb.WriteString("\n")

	return sb.String()
}
