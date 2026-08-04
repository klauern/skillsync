// Package skills implements a shared parser for the Agent Skills Standard SKILL.md format.
// This parser extracts both legacy skill fields and Agent Skills Standard metadata.
package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
)

// Parser implements the parser.Parser interface for Agent Skills Standard SKILL.md files.
type Parser struct {
	basePath string
	platform model.Platform
}

// New creates a new SKILL.md parser.
// basePath specifies the directory to search for SKILL.md files.
// platform specifies which platform this parser is associated with.
func New(basePath string, platform model.Platform) *Parser {
	return &Parser{
		basePath: basePath,
		platform: platform,
	}
}

// Parse parses SKILL.md files from the configured directory.
func (p *Parser) Parse() ([]model.Skill, error) {
	// Check if the base path exists
	if _, err := os.Stat(p.basePath); os.IsNotExist(err) {
		logging.Debug(
			"skills directory not found",
			logging.Platform(string(p.platform)),
			logging.Path(p.basePath),
		)
		return []model.Skill{}, nil
	}

	// Discover SKILL.md files (case-insensitive)
	// The Agent Skills Standard uses SKILL.md, but some users create skill.md
	patterns := []string{"*", "**/*"}
	files, err := parser.DiscoverFiles(p.basePath, patterns)
	if err != nil {
		logging.Error(
			"failed to discover SKILL.md files",
			logging.Platform(string(p.platform)),
			logging.Path(p.basePath),
			logging.Err(err),
		)
		return nil, fmt.Errorf("failed to discover SKILL.md files in %q: %w", p.basePath, err)
	}
	files = slices.DeleteFunc(files, func(path string) bool {
		return !parser.IsSkillEntrypointName(filepath.Base(path))
	})
	files = deduplicateSkillEntrypoints(files)

	logging.Debug(
		"discovered SKILL.md files",
		logging.Platform(string(p.platform)),
		logging.Path(p.basePath),
		logging.Count(len(files)),
	)

	// Parse each skill file
	skills := make([]model.Skill, 0, len(files))
	for _, filePath := range files {
		skill, err := p.parseSkillFile(filePath)
		if err != nil {
			logging.Warn(
				"failed to parse SKILL.md file",
				logging.Platform(string(p.platform)),
				logging.Path(filePath),
				logging.Err(err),
			)
			continue
		}
		skills = append(skills, skill)
	}

	logging.Debug(
		"completed parsing SKILL.md files",
		logging.Platform(string(p.platform)),
		logging.Count(len(skills)),
	)

	return skills, nil
}

// parseSkillFile parses a single SKILL.md file.
func (p *Parser) parseSkillFile(filePath string) (model.Skill, error) {
	// Read file content
	// #nosec G304 - filePath is validated through directory traversal from basePath
	content, err := os.ReadFile(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	// Split frontmatter from content
	result := parser.SplitFrontmatter(content)

	// Extract metadata from frontmatter
	skill := model.Skill{
		Platform: p.platform,
		Path:     filePath,
		Metadata: make(map[string]string),
		Type:     model.SkillTypeSkill,
	}

	if result.HasFrontmatter {
		fm, err := parser.ParseYAMLFrontmatter(result.Frontmatter)
		if err != nil {
			return model.Skill{}, fmt.Errorf("failed to parse frontmatter in %q: %w", filePath, err)
		}

		// Extract required fields
		skill.Name = extractString(fm, "name")
		skill.Description = extractString(fm, "description")

		// Extract tool allowlist fields.
		// SKILL.md content can use either `tools` or `allowed-tools`.
		skill.Tools = extractTools(fm)

		// Extract skill type (skill vs prompt/slash-command)
		if typeStr := extractString(fm, "type"); typeStr != "" {
			skillType, err := model.ParseSkillType(typeStr)
			if err != nil {
				logging.Warn(
					"invalid type in SKILL.md frontmatter",
					logging.Path(filePath),
					logging.Err(err),
				)
			} else {
				skill.Type = skillType
			}
		}

		// Extract trigger for prompts/slash-commands
		skill.Trigger = extractString(fm, "trigger")

		// Extract Agent Skills Standard fields
		if scopeStr := extractString(fm, "scope"); scopeStr != "" {
			scope, err := model.ParseScope(scopeStr)
			if err != nil {
				logging.Warn(
					"invalid scope in SKILL.md frontmatter",
					logging.Path(filePath),
					logging.Err(err),
				)
			} else {
				skill.Scope = scope
			}
		}

		skill.DisableModelInvocation = extractBool(fm, "disable-model-invocation")
		skill.License = extractString(fm, "license")
		skill.Compatibility = extractStringMap(fm, "compatibility")
		skill.Scripts = extractStringSlice(fm, "scripts")
		skill.References = extractStringSlice(fm, "references")
		skill.Assets = extractStringSlice(fm, "assets")

		// Store remaining frontmatter fields in metadata
		knownFields := map[string]bool{
			"name": true, "description": true, "tools": true, "allowed-tools": true, "type": true, "trigger": true,
			"scope": true, "disable-model-invocation": true, "license": true,
			"compatibility": true, "scripts": true, "references": true, "assets": true,
		}
		for key, val := range fm {
			if !knownFields[key] {
				if strVal, ok := val.(string); ok {
					skill.Metadata[key] = strVal
				} else {
					skill.Metadata[key] = fmt.Sprintf("%v", val)
				}
			}
		}
	}

	// If no name in frontmatter, derive from parent directory name.
	if skill.Name == "" {
		skill.Name = deriveNameFromPath(filePath)
	}

	// Validate skill name
	if err := parser.ValidateSkillName(skill.Name); err != nil {
		// Codex skills often use human-readable frontmatter names, but the
		// directory basename is the canonical identifier for discovery/sync.
		// Fall back to the directory name when it is valid so discover stays quiet.
		if p.platform == model.Codex {
			fallback := deriveNameFromPath(filePath)
			if fallback != skill.Name {
				if fallbackErr := parser.ValidateSkillName(fallback); fallbackErr == nil {
					skill.Name = fallback
				}
			}
		}
		if err := parser.ValidateSkillName(skill.Name); err != nil {
			return model.Skill{}, fmt.Errorf("invalid skill name %q in %q: %w", skill.Name, filePath, err)
		}
	}

	// Detect skill directory structure
	skillDir := filepath.Dir(filePath)
	detectSkillDirectoryStructure(&skill, skillDir)

	// Get file modification time
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return model.Skill{}, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}
	skill.ModifiedAt = fileInfo.ModTime()

	// Normalize content
	skill.Content = parser.NormalizeContent(result.Content)

	if skill.Type == "" {
		skill.Type = model.SkillTypeSkill
	}

	return skill, nil
}

// deriveNameFromPath extracts a skill name from the SKILL.md file path.
// Uses the parent directory name as the skill name.
func deriveNameFromPath(filePath string) string {
	dir := filepath.Dir(filePath)
	return filepath.Base(dir)
}

func skillEntrypointPriority(name string) int {
	switch strings.ToLower(name) {
	case "skill.md":
		// Prefer canonical uppercase form when multiple variants exist in one dir.
		if name == "SKILL.md" {
			return 0
		}
		if name == "skill.md" {
			return 1
		}
		return 2
	default:
		return 3
	}
}

// deduplicateSkillEntrypoints keeps at most one SKILL.md variant per directory.
// This avoids duplicate skills on case-sensitive filesystems where SKILL.md and
// skill.md may coexist in the same skill directory.
func deduplicateSkillEntrypoints(files []string) []string {
	type selection struct {
		path     string
		priority int
	}
	byDir := make(map[string]selection, len(files))
	order := make([]string, 0, len(files))

	for _, file := range files {
		dir := filepath.Clean(filepath.Dir(file))
		prio := skillEntrypointPriority(filepath.Base(file))
		if existing, ok := byDir[dir]; ok {
			if prio < existing.priority {
				byDir[dir] = selection{path: file, priority: prio}
			}
			continue
		}
		byDir[dir] = selection{path: file, priority: prio}
		order = append(order, dir)
	}

	deduped := make([]string, 0, len(order))
	for _, dir := range order {
		deduped = append(deduped, byDir[dir].path)
	}
	return deduped
}

// detectSkillDirectoryStructure checks all subdirectories in a skill directory.
// scripts/ and assets/ map to their dedicated fields; all other subdirectories
// (examples, resources, templates, patterns, references, and any other subdir)
// are treated as references to preserve supporting corpus artifacts.
// Uses recursive listing so nested files (e.g. references/docs/guide.md) are included.
func detectSkillDirectoryStructure(skill *model.Skill, skillDir string) {
	entries, err := os.ReadDir(skillDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirName := entry.Name()
		files := listFilesRecursive(filepath.Join(skillDir, dirName))
		for _, relFile := range files {
			relPath := filepath.ToSlash(filepath.Join(dirName, relFile))
			switch strings.ToLower(dirName) {
			case "scripts":
				if !slices.Contains(skill.Scripts, relPath) {
					skill.Scripts = append(skill.Scripts, relPath)
				}
			case "assets":
				if !slices.Contains(skill.Assets, relPath) {
					skill.Assets = append(skill.Assets, relPath)
				}
			default:
				if !slices.Contains(skill.References, relPath) {
					skill.References = append(skill.References, relPath)
				}
			}
		}
	}
}

// listFilesRecursive returns all file paths under dir, relative to dir.
// Nested structure is preserved (e.g. docs/guide.md, templates/config.yaml).
// Returns an empty slice if the directory doesn't exist or can't be read.
func listFilesRecursive(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var result []string
	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(dir, name)
		if entry.IsDir() {
			for _, nested := range listFilesRecursive(fullPath) {
				result = append(result, filepath.ToSlash(filepath.Join(name, nested)))
			}
		} else {
			result = append(result, name)
		}
	}
	return result
}

// extractString extracts a string value from a frontmatter map.
func extractString(fm map[string]any, key string) string {
	if val, ok := fm[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}

// extractBool extracts a boolean value from a frontmatter map.
func extractBool(fm map[string]any, key string) bool {
	if val, ok := fm[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return false
}

// extractStringSlice extracts a string slice from a frontmatter map.
func extractStringSlice(fm map[string]any, key string) []string {
	if val, ok := fm[key]; ok {
		if slice, ok := val.([]any); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if strVal, ok := item.(string); ok {
					result = append(result, strVal)
				}
			}
			return result
		}
	}
	return nil
}

// extractTools extracts tool allowlist values from `tools` or `allowed-tools`.
// Supports YAML arrays and string values.
func extractTools(fm map[string]any) []string {
	tools := extractToolsByKey(fm, "tools")
	if len(tools) == 0 {
		tools = extractToolsByKey(fm, "allowed-tools")
	}
	return tools
}

func extractToolsByKey(fm map[string]any, key string) []string {
	val, ok := fm[key]
	if !ok {
		return nil
	}

	switch v := val.(type) {
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			strVal, ok := item.(string)
			if !ok {
				continue
			}
			tool := strings.TrimSpace(strVal)
			if tool != "" {
				result = append(result, tool)
			}
		}
		return result
	case []string:
		result := make([]string, 0, len(v))
		for _, item := range v {
			tool := strings.TrimSpace(item)
			if tool != "" {
				result = append(result, tool)
			}
		}
		return result
	case string:
		raw := strings.TrimSpace(v)
		if raw == "" {
			return nil
		}

		var parts []string
		if strings.Contains(raw, ",") {
			parts = strings.Split(raw, ",")
		} else {
			parts = strings.Fields(raw)
		}

		result := make([]string, 0, len(parts))
		for _, part := range parts {
			tool := strings.TrimSpace(part)
			if tool != "" {
				result = append(result, tool)
			}
		}
		return result
	default:
		return nil
	}
}

// extractStringMap extracts a string map from a frontmatter map.
func extractStringMap(fm map[string]any, key string) map[string]string {
	if val, ok := fm[key]; ok {
		if mapVal, ok := val.(map[string]any); ok {
			result := make(map[string]string)
			for k, v := range mapVal {
				if strVal, ok := v.(string); ok {
					result[k] = strVal
				} else {
					result[k] = fmt.Sprintf("%v", v)
				}
			}
			return result
		}
	}
	return nil
}

// Platform returns the platform this parser is associated with.
func (p *Parser) Platform() model.Platform {
	return p.platform
}

// DefaultPath returns the configured base path.
func (p *Parser) DefaultPath() string {
	return p.basePath
}

// ParseSkillFile parses a single SKILL.md file from a given path.
// This is a convenience function for parsing individual files without creating a full parser.
func ParseSkillFile(filePath string, platform model.Platform) (model.Skill, error) {
	p := &Parser{
		basePath: filepath.Dir(filePath),
		platform: platform,
	}
	return p.parseSkillFile(filePath)
}

// ParseSkillContent parses SKILL.md content from bytes.
// This is useful for parsing skill content from non-file sources.
func ParseSkillContent(content []byte, name string, platform model.Platform) (model.Skill, error) {
	// Split frontmatter from content
	result := parser.SplitFrontmatter(content)

	skill := model.Skill{
		Name:     name,
		Platform: platform,
		Metadata: make(map[string]string),
	}

	if result.HasFrontmatter {
		fm, err := parser.ParseYAMLFrontmatter(result.Frontmatter)
		if err != nil {
			return model.Skill{}, fmt.Errorf("failed to parse frontmatter: %w", err)
		}

		// Override name if provided in frontmatter
		if fmName := extractString(fm, "name"); fmName != "" {
			skill.Name = fmName
		}
		skill.Description = extractString(fm, "description")
		skill.Tools = extractTools(fm)

		// Extract skill type (skill vs prompt/slash-command)
		if typeStr := extractString(fm, "type"); typeStr != "" {
			if skillType, err := model.ParseSkillType(typeStr); err == nil {
				skill.Type = skillType
			}
		}

		// Extract trigger for prompts/slash-commands
		skill.Trigger = extractString(fm, "trigger")

		// Extract Agent Skills Standard fields
		if scopeStr := extractString(fm, "scope"); scopeStr != "" {
			if scope, err := model.ParseScope(scopeStr); err == nil {
				skill.Scope = scope
			}
		}

		skill.DisableModelInvocation = extractBool(fm, "disable-model-invocation")
		skill.License = extractString(fm, "license")
		skill.Compatibility = extractStringMap(fm, "compatibility")
		skill.Scripts = extractStringSlice(fm, "scripts")
		skill.References = extractStringSlice(fm, "references")
		skill.Assets = extractStringSlice(fm, "assets")

		// Store remaining fields in metadata
		knownFields := map[string]bool{
			"name": true, "description": true, "tools": true, "allowed-tools": true, "type": true, "trigger": true,
			"scope": true, "disable-model-invocation": true, "license": true,
			"compatibility": true, "scripts": true, "references": true, "assets": true,
		}
		for key, val := range fm {
			if !knownFields[key] {
				if strVal, ok := val.(string); ok {
					skill.Metadata[key] = strVal
				} else {
					skill.Metadata[key] = fmt.Sprintf("%v", val)
				}
			}
		}
	}

	// Validate skill name
	if skill.Name == "" {
		return model.Skill{}, fmt.Errorf("skill name is required")
	}
	if err := parser.ValidateSkillName(skill.Name); err != nil {
		// Keep Codex content parsing aligned with file-backed SKILL.md parsing:
		// the provided name is the canonical identifier even when frontmatter
		// carries a human-readable display name.
		if platform == model.Codex && name != "" && name != skill.Name {
			if fallbackErr := parser.ValidateSkillName(name); fallbackErr == nil {
				skill.Name = name
			}
		}
		if err := parser.ValidateSkillName(skill.Name); err != nil {
			return model.Skill{}, fmt.Errorf("invalid skill name %q: %w", skill.Name, err)
		}
	}

	// Normalize content
	skill.Content = parser.NormalizeContent(result.Content)

	if skill.Type == "" {
		skill.Type = model.SkillTypeSkill
	}

	return skill, nil
}

// IsAgentSkillsFormat checks if content follows the Agent Skills Standard format.
// Returns true if the content has valid SKILL.md frontmatter with required fields.
func IsAgentSkillsFormat(content []byte) bool {
	result := parser.SplitFrontmatter(content)
	if !result.HasFrontmatter {
		return false
	}

	fm, err := parser.ParseYAMLFrontmatter(result.Frontmatter)
	if err != nil {
		return false
	}

	// Agent Skills Standard requires name and description
	name := extractString(fm, "name")
	description := extractString(fm, "description")

	return name != "" && description != ""
}

// HasSkillDirectory checks if a path contains a valid skill directory structure.
// A valid skill directory contains a SKILL.md file (case-insensitive check).
func HasSkillDirectory(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && parser.IsSkillEntrypointName(entry.Name()) {
			return true
		}
	}
	return false
}

// ListSkillDirectories finds all directories containing SKILL.md files (case-insensitive).
func ListSkillDirectories(basePath string) ([]string, error) {
	patterns := []string{"*", "**/*"}
	files, err := parser.DiscoverFiles(basePath, patterns)
	if err != nil {
		return nil, fmt.Errorf("discover skill directories in %q: %w", basePath, err)
	}
	files = slices.DeleteFunc(files, func(path string) bool {
		return !parser.IsSkillEntrypointName(filepath.Base(path))
	})

	// Deduplicate directories (in case multiple case variants exist)
	seen := make(map[string]bool)
	dirs := make([]string, 0, len(files))
	for _, f := range files {
		dir := filepath.Dir(f)
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	return dirs, nil
}

// SkillDirectoryContents returns information about the contents of a skill directory.
type SkillDirectoryContents struct {
	SkillFile  string   // Path to SKILL.md
	Scripts    []string // Files in scripts/ (recursive)
	References []string // Files in references/ (recursive)
	Assets     []string // Files in assets/ (recursive)
}

// GetSkillDirectoryContents returns the contents of a skill directory.
func GetSkillDirectoryContents(skillDir string) (*SkillDirectoryContents, error) {
	entries, err := os.ReadDir(skillDir)
	if err != nil {
		return nil, fmt.Errorf("read skill directory %q: %w", skillDir, err)
	}
	var candidates []string
	for _, entry := range entries {
		if !entry.IsDir() && parser.IsSkillEntrypointName(entry.Name()) {
			candidates = append(candidates, filepath.Join(skillDir, entry.Name()))
		}
	}
	candidates = deduplicateSkillEntrypoints(candidates)
	var skillFile string
	if len(candidates) > 0 {
		skillFile = candidates[0]
	}
	if skillFile == "" {
		return nil, fmt.Errorf("SKILL.md not found in %q", skillDir)
	}

	contents := &SkillDirectoryContents{
		SkillFile: skillFile,
	}

	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			switch strings.ToLower(entry.Name()) {
			case "scripts":
				contents.Scripts = append(contents.Scripts, listFilesRecursive(filepath.Join(skillDir, entry.Name()))...)
			case "references":
				contents.References = append(contents.References, listFilesRecursive(filepath.Join(skillDir, entry.Name()))...)
			case "assets":
				contents.Assets = append(contents.Assets, listFilesRecursive(filepath.Join(skillDir, entry.Name()))...)
			}
		}
	}

	return contents, nil
}

// AlternativeKeyMappings provides mappings for common alternative key names.
var AlternativeKeyMappings = map[string]string{
	"disableModelInvocation":   "disable-model-invocation",
	"disable_model_invocation": "disable-model-invocation",
}

// NormalizeKey converts alternative frontmatter key names to standard names.
func NormalizeKey(key string) string {
	// Convert camelCase or snake_case to kebab-case for standard keys
	if mapped, ok := AlternativeKeyMappings[key]; ok {
		return mapped
	}
	// Convert camelCase to kebab-case
	return toKebabCase(key)
}

// toKebabCase converts a string from camelCase to kebab-case.
func toKebabCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('-')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
