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
		logging.Debug("skills directory not found",
			logging.Platform(string(p.platform)),
			logging.Path(p.basePath),
		)
		return []model.Skill{}, nil
	}

	// Discover SKILL.md files
	patterns := []string{"SKILL.md", "**/SKILL.md"}
	files, err := parser.DiscoverFiles(p.basePath, patterns)
	if err != nil {
		logging.Error("failed to discover SKILL.md files",
			logging.Platform(string(p.platform)),
			logging.Path(p.basePath),
			logging.Err(err),
		)
		return nil, fmt.Errorf("failed to discover SKILL.md files in %q: %w", p.basePath, err)
	}

	logging.Debug("discovered SKILL.md files",
		logging.Platform(string(p.platform)),
		logging.Path(p.basePath),
		logging.Count(len(files)),
	)

	// Parse each skill file
	skills := make([]model.Skill, 0, len(files))
	for _, filePath := range files {
		skill, err := p.parseSkillFile(filePath)
		if err != nil {
			logging.Warn("failed to parse SKILL.md file",
				logging.Platform(string(p.platform)),
				logging.Path(filePath),
				logging.Err(err),
			)
			continue
		}
		skills = append(skills, skill)
	}

	logging.Debug("completed parsing SKILL.md files",
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
	}

	if result.HasFrontmatter {
		fm, err := parser.ParseYAMLFrontmatter(result.Frontmatter)
		if err != nil {
			return model.Skill{}, fmt.Errorf("failed to parse frontmatter in %q: %w", filePath, err)
		}

		// Extract required fields
		skill.Name = extractString(fm, "name")
		skill.Description = extractString(fm, "description")

		// Extract optional legacy fields
		skill.Tools = extractStringSlice(fm, "tools")

		// Extract Agent Skills Standard fields
		if scopeStr := extractString(fm, "scope"); scopeStr != "" {
			scope, err := model.ParseScope(scopeStr)
			if err != nil {
				logging.Warn("invalid scope in SKILL.md frontmatter",
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
			"name": true, "description": true, "tools": true,
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

	// If no name in frontmatter, derive from parent directory name
	if skill.Name == "" {
		skill.Name = deriveNameFromPath(filePath)
	}

	// Validate skill name
	if err := parser.ValidateSkillName(skill.Name); err != nil {
		return model.Skill{}, fmt.Errorf("invalid skill name %q in %q: %w", skill.Name, filePath, err)
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

	return skill, nil
}

// deriveNameFromPath extracts a skill name from the SKILL.md file path.
// Uses the parent directory name as the skill name.
func deriveNameFromPath(filePath string) string {
	dir := filepath.Dir(filePath)
	return filepath.Base(dir)
}

// detectSkillDirectoryStructure checks for standard skill subdirectories
// and populates the skill's Scripts, References, and Assets fields if found.
func detectSkillDirectoryStructure(skill *model.Skill, skillDir string) {
	// Check for scripts/ directory
	scriptsDir := filepath.Join(skillDir, "scripts")
	if entries := listFiles(scriptsDir); len(entries) > 0 {
		// Append discovered scripts to any defined in frontmatter
		for _, entry := range entries {
			relPath := filepath.Join("scripts", entry)
			if !slices.Contains(skill.Scripts, relPath) {
				skill.Scripts = append(skill.Scripts, relPath)
			}
		}
	}

	// Check for references/ directory
	refsDir := filepath.Join(skillDir, "references")
	if entries := listFiles(refsDir); len(entries) > 0 {
		for _, entry := range entries {
			relPath := filepath.Join("references", entry)
			if !slices.Contains(skill.References, relPath) {
				skill.References = append(skill.References, relPath)
			}
		}
	}

	// Check for assets/ directory
	assetsDir := filepath.Join(skillDir, "assets")
	if entries := listFiles(assetsDir); len(entries) > 0 {
		for _, entry := range entries {
			relPath := filepath.Join("assets", entry)
			if !slices.Contains(skill.Assets, relPath) {
				skill.Assets = append(skill.Assets, relPath)
			}
		}
	}
}

// listFiles returns a list of file names in a directory.
// Returns an empty slice if the directory doesn't exist or can't be read.
func listFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files
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
		skill.Tools = extractStringSlice(fm, "tools")

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
			"name": true, "description": true, "tools": true,
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
		return model.Skill{}, fmt.Errorf("invalid skill name %q: %w", skill.Name, err)
	}

	// Normalize content
	skill.Content = parser.NormalizeContent(result.Content)

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
// A valid skill directory contains a SKILL.md file.
func HasSkillDirectory(path string) bool {
	skillFile := filepath.Join(path, "SKILL.md")
	info, err := os.Stat(skillFile)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// ListSkillDirectories finds all directories containing SKILL.md files.
func ListSkillDirectories(basePath string) ([]string, error) {
	files, err := parser.DiscoverFiles(basePath, []string{"SKILL.md", "**/SKILL.md"})
	if err != nil {
		return nil, err
	}

	dirs := make([]string, 0, len(files))
	for _, f := range files {
		dirs = append(dirs, filepath.Dir(f))
	}
	return dirs, nil
}

// SkillDirectoryContents returns information about the contents of a skill directory.
type SkillDirectoryContents struct {
	SkillFile  string   // Path to SKILL.md
	Scripts    []string // Files in scripts/
	References []string // Files in references/
	Assets     []string // Files in assets/
}

// GetSkillDirectoryContents returns the contents of a skill directory.
func GetSkillDirectoryContents(skillDir string) (*SkillDirectoryContents, error) {
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		return nil, fmt.Errorf("SKILL.md not found in %q: %w", skillDir, err)
	}

	contents := &SkillDirectoryContents{
		SkillFile:  skillFile,
		Scripts:    listFiles(filepath.Join(skillDir, "scripts")),
		References: listFiles(filepath.Join(skillDir, "references")),
		Assets:     listFiles(filepath.Join(skillDir, "assets")),
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
