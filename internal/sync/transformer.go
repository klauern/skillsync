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

func namedArtifactTargetPath(skillName string, target model.Platform) string {
	switch target {
	case model.Codex, model.Gemini, model.PiDev:
		// These platforms discover named artifacts as SKILL.md-based directories.
		return filepath.Join(skillName, "SKILL.md")
	case model.Cursor, model.ClaudeCode:
		// Cursor and Claude keep named artifacts in flat markdown files.
		return skillName + ".md"
	default:
		return ""
	}
}

// Transform converts a skill from source to target platform format.
func (t *Transformer) Transform(skill model.Skill, targetPlatform model.Platform) (model.Skill, error) {
	logging.Debug(
		"transforming skill",
		logging.Skill(skill.Name),
		logging.Platform(string(skill.Platform)),
		slog.String("target", string(targetPlatform)),
		logging.Operation("transform"),
	)

	transformed := skill
	transformed.Platform = targetPlatform

	// Update path for target platform
	transformed.Path = t.transformPath(skill, targetPlatform)
	logging.Debug(
		"transformed path",
		logging.Skill(skill.Name),
		slog.String("original_path", skill.Path),
		logging.Path(transformed.Path),
	)

	// Transform content based on target platform requirements
	content, err := t.transformContent(skill, targetPlatform, transformed.Path)
	if err != nil {
		logging.Warn(
			"content transformation failed",
			logging.Skill(skill.Name),
			logging.Err(err),
		)
		return model.Skill{}, fmt.Errorf("failed to transform content: %w", err)
	}
	transformed.Content = content

	// Transform metadata for platform-specific fields
	transformed.Metadata = t.transformMetadata(skill, targetPlatform)

	logging.Debug(
		"skill transformation completed",
		logging.Skill(skill.Name),
		slog.String("target", string(targetPlatform)),
	)

	return transformed, nil
}

// transformPath generates the appropriate file path for the target platform.
//
//nolint:gocyclo // intentional (source-platform × target-platform) dispatch table
func (t *Transformer) transformPath(skill model.Skill, target model.Platform) string {
	if isSystemPromptSkill(skill) {
		switch target {
		case model.PiDev:
			if skill.Metadata["mode"] == "append" {
				return "APPEND_SYSTEM.md"
			}
			return "SYSTEM.md"
		case model.Gemini:
			return "GEMINI.md"
		default:
			baseName := filepath.Base(skill.Path)
			if baseName == "" {
				if skill.Metadata["mode"] == "append" {
					return "append-system.md"
				}
				return "system.md"
			}
			return baseName
		}
	}

	if target == model.Copilot {
		return transformCopilotPath(skill)
	}

	if target == model.Gemini {
		if skill.Metadata["type"] == "instructions" {
			return "GEMINI.md"
		}
		if skill.Type == model.SkillTypePrompt {
			return namedArtifactTargetPath(skill.Name, target)
		}
		baseName := filepath.Base(skill.Path)
		if isSkillFile(baseName) && skill.Name != "" {
			return namedArtifactTargetPath(skill.Name, target)
		}
		if baseName == "" {
			return "SKILL.md"
		}
		return baseName
	}

	if skill.Type == model.SkillTypePrompt {
		if targetPath := namedArtifactTargetPath(skill.Name, target); targetPath != "" {
			return targetPath
		}
	}

	baseName := filepath.Base(skill.Path)
	if isSkillFile(baseName) && skill.Name != "" {
		if targetPath := namedArtifactTargetPath(skill.Name, target); targetPath != "" {
			return targetPath
		}
		return baseName
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
	case model.PiDev:
		return filepath.Join(skill.Name, "SKILL.md")
	default:
		return baseName
	}
}

// transformContent transforms skill content for the target platform.
func (t *Transformer) transformContent(skill model.Skill, target model.Platform, targetPath string) (string, error) {
	// Build frontmatter based on target platform
	var frontmatter map[string]any
	includeFrontmatter := shouldIncludeFrontmatter(target, targetPath)
	if isSystemPromptSkill(skill) && target == model.PiDev {
		includeFrontmatter = false
	}
	if includeFrontmatter {
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
		} else if applyTo, ok := skill.Metadata["applyTo"]; ok && applyTo != "" {
			fm["globs"] = applyTo
		}
		if alwaysApply, ok := skill.Metadata["alwaysApply"]; ok {
			fm["alwaysApply"] = alwaysApply
		}
	case model.Copilot:
		if len(skill.Tools) > 0 {
			fm["tools"] = skill.Tools
		}
	}

	// Include other metadata that's platform-agnostic
	for key, val := range skill.Metadata {
		// Skip fields we've already handled
		if key == "globs" || key == "alwaysApply" {
			continue
		}
		if target == model.Copilot && key == model.MetadataKeyCopilotArtifact {
			continue
		}
		// Include if not already set
		if _, exists := fm[key]; !exists {
			fm[key] = val
		}
	}

	if target == model.Copilot && copilotArtifactType(skill) == model.CopilotArtifactRepositoryInstructions {
		delete(fm, "name")
		delete(fm, "type")
		delete(fm, "trigger")
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
	if target == model.PiDev {
		return isSkillFile(targetPath)
	}
	return true
}

func isSystemPromptSkill(skill model.Skill) bool {
	return skill.Metadata["type"] == "system-prompt"
}

func transformCopilotPath(skill model.Skill) string {
	switch copilotArtifactType(skill) {
	case model.CopilotArtifactRepositoryInstructions:
		return "copilot-instructions.md"
	case model.CopilotArtifactInstructions:
		return filepath.Join("instructions", skill.Name+".instructions.md")
	case model.CopilotArtifactPrompt:
		return filepath.Join("prompts", skill.Name+".prompt.md")
	default:
		return filepath.Join("agents", skill.Name+".agent.md")
	}
}

func copilotArtifactType(skill model.Skill) string {
	if artifact := skill.Metadata[model.MetadataKeyCopilotArtifact]; artifact != "" {
		return artifact
	}
	if skill.Type == model.SkillTypePrompt {
		return model.CopilotArtifactPrompt
	}

	base := strings.ToLower(filepath.Base(skill.Path))
	switch {
	case base == "copilot-instructions.md", base == "agents.md", base == "claude.md", base == "gemini.md":
		return model.CopilotArtifactRepositoryInstructions
	case strings.HasSuffix(base, ".instructions.md"):
		return model.CopilotArtifactInstructions
	case strings.HasSuffix(base, ".prompt.md"):
		return model.CopilotArtifactPrompt
	default:
		return model.CopilotArtifactAgent
	}
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
		warnLossyCopilotFields(skill, target, metadata)

	case model.Cursor:
		if applyTo, ok := metadata["applyTo"]; ok && applyTo != "" {
			metadata["globs"] = applyTo
			delete(metadata, "applyTo")
		}
		warnLossyCopilotFields(skill, target, metadata)

	case model.Codex:
		// Codex metadata handling - preserve source info
		metadata["source_platform"] = string(skill.Platform)
		delete(metadata, "handoffs")
		delete(metadata, "target")
		delete(metadata, "mcp-servers")
		warnLossyCopilotFields(skill, target, metadata)

	case model.PiDev, model.Gemini:
		warnLossyCopilotFields(skill, target, metadata)
	}

	return metadata
}

// copilotOnlyMetadataKeys are frontmatter fields that are meaningful only on Copilot targets.
// When transforming to another platform these fields become dead weight and are flagged.
var copilotOnlyMetadataKeys = []string{"applyTo", "target", "handoffs", "mcp-servers"}

// warnLossyCopilotFields emits a warning when a skill carries Copilot-specific metadata
// and is being transformed to a non-Copilot target.
func warnLossyCopilotFields(skill model.Skill, target model.Platform, metadata map[string]string) {
	var lossy []string
	for _, key := range copilotOnlyMetadataKeys {
		if _, ok := metadata[key]; ok {
			lossy = append(lossy, key)
		}
	}
	if len(lossy) > 0 {
		logging.Warn(
			"copilot-specific metadata fields are not portable to target platform",
			logging.Skill(skill.Name),
			slog.String("target", string(target)),
			slog.Any("fields", lossy),
		)
	}
}

// CanTransform returns true if transformation between platforms is supported.
func (t *Transformer) CanTransform(source, target model.Platform) bool {
	// All platform combinations are supported
	return source.IsValid() && target.IsValid()
}

// MergeContent merges source and target content with clear separation.
func (t *Transformer) MergeContent(sourceContent, targetContent string, sourceName string) string {
	logging.Debug(
		"merging content with separator",
		logging.Skill(sourceName),
		logging.Operation("merge_content"),
	)

	var sb strings.Builder

	// Add existing target content first
	sb.WriteString(targetContent)

	// Add separator and source content
	sb.WriteString("\n\n---\n\n")
	fmt.Fprintf(&sb, "## Merged from: %s\n\n", sourceName)
	sb.WriteString(sourceContent)

	return sb.String()
}
