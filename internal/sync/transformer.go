package sync

import (
	"bytes"
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
)

// Transformer handles skill transformation between platforms.
type Transformer struct{}

// NewTransformer creates a new transformer.
func NewTransformer() *Transformer {
	return &Transformer{}
}

// Transform converts a skill from source to target platform format.
func (t *Transformer) Transform(skill model.Skill, targetPlatform model.Platform) (model.Skill, error) {
	logging.Debug("transforming skill",
		logging.Skill(skill.Name),
		logging.Platform(string(skill.Platform)),
		slog.String("target", string(targetPlatform)),
		logging.Operation("transform"),
	)

	transformed := skill
	transformed.Platform = targetPlatform

	// Update path for target platform
	transformed.Path = t.transformPath(skill, targetPlatform)
	logging.Debug("transformed path",
		logging.Skill(skill.Name),
		slog.String("original_path", skill.Path),
		logging.Path(transformed.Path),
	)

	// Transform content based on target platform requirements
	content, err := t.transformContent(skill, targetPlatform, transformed.Path)
	if err != nil {
		logging.Warn("content transformation failed",
			logging.Skill(skill.Name),
			logging.Err(err),
		)
		return model.Skill{}, fmt.Errorf("failed to transform content: %w", err)
	}
	transformed.Content = content

	// Transform metadata for platform-specific fields
	transformed.Metadata = t.transformMetadata(skill, targetPlatform)

	logging.Debug("skill transformation completed",
		logging.Skill(skill.Name),
		slog.String("target", string(targetPlatform)),
	)

	return transformed, nil
}

// transformPath generates the appropriate file path for the target platform.
func (t *Transformer) transformPath(skill model.Skill, target model.Platform) string {
	if skill.Type == model.SkillTypePrompt {
		switch target {
		case model.Codex:
			// Codex discovery is SKILL.md-centric; store prompts as SKILL artifacts.
			return filepath.Join(skill.Name, "SKILL.md")
		case model.Cursor:
			// Cursor prompt artifacts are markdown-based; keep simple filename layout.
			return skill.Name + ".md"
		case model.ClaudeCode:
			// Claude prompt/command legacy artifacts are markdown files.
			return skill.Name + ".md"
		}
	}

	baseName := filepath.Base(skill.Path)
	if isSkillFile(baseName) && skill.Name != "" {
		switch target {
		case model.Codex:
			return filepath.Join(skill.Name, "SKILL.md")
		case model.Cursor:
			return skill.Name + ".md"
		case model.ClaudeCode:
			return skill.Name + ".md"
		default:
			return baseName
		}
	}
	nameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	switch target {
	case model.ClaudeCode:
		// Claude Code uses .md extension
		return nameWithoutExt + ".md"
	case model.Cursor:
		// Cursor uses .md or .mdc extension
		// Preserve .mdc if source was .mdc, otherwise use .md
		if strings.HasSuffix(skill.Path, ".mdc") {
			return nameWithoutExt + ".mdc"
		}
		return nameWithoutExt + ".md"
	case model.Codex:
		// Codex uses AGENTS.md for agent instructions
		if nameWithoutExt == "AGENTS" || nameWithoutExt == "agents" {
			return "AGENTS.md"
		}
		return nameWithoutExt + ".md"
	default:
		return baseName
	}
}

// transformContent transforms skill content for the target platform.
func (t *Transformer) transformContent(skill model.Skill, target model.Platform, targetPath string) (string, error) {
	// Build frontmatter based on target platform
	var frontmatter map[string]any
	if shouldIncludeFrontmatter(target, targetPath) {
		frontmatter = t.buildFrontmatter(skill, target)
	}

	var sb strings.Builder

	// Add frontmatter if present
	if len(frontmatter) > 0 {
		sb.WriteString("---\n")
		fm, err := yaml.Marshal(frontmatter)
		if err != nil {
			return "", fmt.Errorf("failed to marshal frontmatter: %w", err)
		}
		sb.Write(bytes.TrimSpace(fm))
		sb.WriteString("\n---\n\n")
	}

	// Add the main content
	sb.WriteString(skill.Content)

	return sb.String(), nil
}

// buildFrontmatter creates platform-appropriate frontmatter.
func (t *Transformer) buildFrontmatter(skill model.Skill, target model.Platform) map[string]any {
	fm := make(map[string]any)

	// Always include name if present
	if skill.Name != "" {
		fm["name"] = skill.Name
	}

	// Always include description if present
	if skill.Description != "" {
		fm["description"] = skill.Description
	}

	if skill.Type != "" {
		fm["type"] = skill.Type.String()
	}
	if skill.Trigger != "" {
		fm["trigger"] = skill.Trigger
	}

	switch target {
	case model.ClaudeCode:
		// Claude Code supports tools array
		if len(skill.Tools) > 0 {
			fm["tools"] = skill.Tools
		}

	case model.Cursor:
		// Cursor has specific fields like globs and alwaysApply
		if globs, ok := skill.Metadata["globs"]; ok {
			fm["globs"] = globs
		}
		if alwaysApply, ok := skill.Metadata["alwaysApply"]; ok {
			fm["alwaysApply"] = alwaysApply
		}
	}

	// Include other metadata that's platform-agnostic
	for key, val := range skill.Metadata {
		// Skip fields we've already handled
		if key == "globs" || key == "alwaysApply" {
			continue
		}
		// Include if not already set
		if _, exists := fm[key]; !exists {
			fm[key] = val
		}
	}

	return fm
}

func isSkillFile(path string) bool {
	return strings.EqualFold(filepath.Base(path), "SKILL.md")
}

func shouldIncludeFrontmatter(target model.Platform, targetPath string) bool {
	if target == model.Codex {
		return isSkillFile(targetPath)
	}
	return true
}

// transformMetadata transforms metadata for the target platform.
func (t *Transformer) transformMetadata(skill model.Skill, target model.Platform) map[string]string {
	metadata := make(map[string]string)

	// Copy existing metadata
	maps.Copy(metadata, skill.Metadata)

	// Add platform-specific transformations
	switch target {
	case model.ClaudeCode:
		// Remove Cursor-specific fields
		delete(metadata, "globs")
		delete(metadata, "alwaysApply")

	case model.Cursor:
		// Cursor metadata is typically preserved as-is

	case model.Codex:
		// Codex metadata handling - preserve source info
		metadata["source_platform"] = string(skill.Platform)
	}

	return metadata
}

// CanTransform returns true if transformation between platforms is supported.
func (t *Transformer) CanTransform(source, target model.Platform) bool {
	// All platform combinations are supported
	return source.IsValid() && target.IsValid()
}

// MergeContent merges source and target content with clear separation.
func (t *Transformer) MergeContent(sourceContent, targetContent string, sourceName string) string {
	logging.Debug("merging content with separator",
		logging.Skill(sourceName),
		logging.Operation("merge_content"),
	)

	var sb strings.Builder

	// Add existing target content first
	sb.WriteString(targetContent)

	// Add separator and source content
	sb.WriteString("\n\n---\n\n")
	sb.WriteString(fmt.Sprintf("## Merged from: %s\n\n", sourceName))
	sb.WriteString(sourceContent)

	return sb.String()
}
