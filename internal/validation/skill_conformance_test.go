package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestValidateSkillConformanceRules(t *testing.T) {
	valid := model.Skill{Name: "valid-skill", Description: "A skill", Path: filepath.Join("skills", "valid-skill", "SKILL.md"), Compatibility: "runtime >=1.0"}
	if issues := ValidateSkillConformance(valid); len(issues) != 0 {
		t.Fatalf("valid skill issues: %v", issues)
	}

	bad := valid
	bad.Name = "Bad__Name"
	bad.Description = strings.Repeat("d", MaxSkillDescriptionLength+1)
	bad.Compatibility = strings.Repeat("c", MaxSkillCompatibilityLength+1)
	bad.Path = filepath.Join("skills", "other", "skill.md")
	bad.StandardMetadata = map[string]any{"count": 1}
	issues := ValidateSkillConformance(bad)
	for _, code := range []string{"filename", "name.format", "description.length", "compatibility.length", "name.directory", "metadata.type"} {
		found := false
		for _, issue := range issues {
			if issue.Code == code && issue.Severity == "warning" {
				found = true
			}
		}
		if !found {
			t.Errorf("missing warning %q in %v", code, issues)
		}
	}
}

func TestInvalidConformanceBlocksWriteValidation(t *testing.T) {
	skill := model.Skill{Name: "bad name", Path: filepath.Join(t.TempDir(), "SKILL.md"), Platform: model.ClaudeCode, ConformanceIssues: []model.ConformanceIssue{{Code: "name.format", Message: "bad", Severity: "warning"}}}
	if err := validateSkill(skill, 0, DefaultOptions()); err == nil {
		t.Fatal("invalid conformance should block writes")
	}
}

func TestValidateSkillConformanceRequiredFrontmatterIsNotMaskedByDerivedFields(t *testing.T) {
	skill := model.Skill{
		Name:             "derived-name",
		Path:             filepath.Join("skills", "derived-name", "SKILL.md"),
		RawFrontmatter:   map[string]any{},
		StandardMetadata: map[string]any{"owner": "team"},
	}
	issues := ValidateSkillConformance(skill)
	for _, code := range []string{"name.required", "description.required"} {
		if !hasConformanceCode(issues, code) {
			t.Errorf("missing %q in %v", code, issues)
		}
	}
}

func TestValidateSkillConformanceNameRules(t *testing.T) {
	for _, name := range []string{"Upper", "under_score", "-leading", "trailing-", "double--hyphen", strings.Repeat("a", MaxSkillNameLength+1)} {
		t.Run(name, func(t *testing.T) {
			skill := model.Skill{Name: name, Description: "valid", Path: filepath.Join("skills", name, "SKILL.md")}
			issues := ValidateSkillConformance(skill)
			if !hasConformanceCode(issues, "name.format") && !hasConformanceCode(issues, "name.length") {
				t.Fatalf("invalid name %q has no name issue: %v", name, issues)
			}
		})
	}
}

func TestValidateSkillConformanceContentWithoutPathSkipsFilesystemRules(t *testing.T) {
	skill := model.Skill{Name: "content-only", Description: "valid"}
	if issues := ValidateSkillConformance(skill); len(issues) != 0 {
		t.Fatalf("content-only skill has filesystem issues: %v", issues)
	}
}

func TestValidateSkillsFormatBlocksRetainedConformanceIssues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skill.md")
	if err := os.WriteFile(path, []byte("invalid entrypoint"), 0o600); err != nil {
		t.Fatal(err)
	}
	skill := model.Skill{
		Name:        "invalid-skill",
		Description: "invalid entrypoint casing",
		Platform:    model.ClaudeCode,
		Path:        path,
		Content:     "invalid entrypoint",
		ConformanceIssues: []model.ConformanceIssue{
			{Code: "filename", Message: "skill entrypoint must be named exactly SKILL.md", Severity: "warning"},
		},
	}
	result, err := ValidateSkillsFormat([]model.Skill{skill}, model.ClaudeCode)
	if err != nil {
		t.Fatalf("ValidateSkillsFormat returned unexpected execution error: %v", err)
	}
	if result.Valid || result.Error() == nil {
		t.Fatalf("invalid retained conformance issue was not write-blocking: result=%+v", result)
	}
}

func hasConformanceCode(issues []model.ConformanceIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
