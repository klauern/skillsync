package validation

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/klauern/skillsync/internal/model"
)

// Agent Skills limits from the official specification.
const (
	MaxSkillNameLength          = 64
	MaxSkillDescriptionLength   = 1024
	MaxSkillCompatibilityLength = 500
)

var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ValidateSkillConformance checks the portable Agent Skills Standard fields.
// Defects are warnings here because discovery must retain artifacts for users
// to inspect and repair; write validation can promote them to errors.
func ValidateSkillConformance(skill model.Skill) []model.ConformanceIssue {
	var issues []model.ConformanceIssue
	add := func(code, message string) {
		issues = append(issues, model.ConformanceIssue{Code: code, Message: message, Severity: "warning"})
	}

	if skill.Path != "" && filepath.Base(skill.Path) != "SKILL.md" {
		add("filename", "skill entrypoint must be named exactly SKILL.md")
	}
	nameMissing := skill.Name == ""
	descriptionMissing := skill.Description == ""
	if skill.RawFrontmatter != nil {
		rawName, ok := skill.RawFrontmatter["name"]
		name, scalar := rawName.(string)
		if !ok || !scalar || strings.TrimSpace(name) == "" {
			nameMissing = true
		}
		rawDescription, ok := skill.RawFrontmatter["description"]
		description, scalar := rawDescription.(string)
		if !ok || !scalar || strings.TrimSpace(description) == "" {
			descriptionMissing = true
		}
	}
	if nameMissing {
		add("name.required", "name is required")
	}
	if skill.Name != "" {
		if utf8.RuneCountInString(skill.Name) > MaxSkillNameLength {
			add("name.length", "name must be at most 64 characters")
		}
		if !skillNamePattern.MatchString(skill.Name) {
			add("name.format", "name must contain only lowercase letters, numbers, and single hyphens")
		}
	}
	if descriptionMissing {
		add("description.required", "description is required")
	}
	if utf8.RuneCountInString(skill.Description) > MaxSkillDescriptionLength {
		add("description.length", "description must be at most 1024 characters")
	}
	if utf8.RuneCountInString(skill.Compatibility) > MaxSkillCompatibilityLength {
		add("compatibility.length", "compatibility must be at most 500 characters")
	}
	if skill.RawFrontmatter != nil {
		if raw, ok := skill.RawFrontmatter["compatibility"]; ok {
			if _, scalar := raw.(string); !scalar {
				add("compatibility.type", "compatibility must be a string")
			}
		}
	}
	if skill.Path != "" && skill.Name != "" && filepath.Base(filepath.Dir(skill.Path)) != skill.Name {
		add("name.directory", "name must agree with the parent directory")
	}
	for key, value := range skill.StandardMetadata {
		if _, ok := value.(string); !ok {
			add("metadata.type", "metadata values must be strings ("+key+")")
		}
	}
	return issues
}
