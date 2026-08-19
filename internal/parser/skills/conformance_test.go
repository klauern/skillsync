package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestParseRetainsInvalidArtifactAndRawExtensions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bad-name")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "skill.md")
	content := []byte("---\nname: Bad Name\ndescription: visible\ncompatibility: runtime\nmetadata:\n  team: tools\nextra:\n  nested:\n    enabled: true\n---\nBody")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	skill, err := ParseSkillFile(path, model.ClaudeCode)
	if err != nil {
		t.Fatal(err)
	}
	if skill.Name != "Bad Name" || skill.Compatibility != "runtime" {
		t.Fatalf("parsed standard fields lost: %#v", skill)
	}
	if skill.StandardMetadata["team"] != "tools" || skill.RawFrontmatter["extra"] == nil {
		t.Fatalf("nested/raw frontmatter lost: %#v", skill)
	}
	if len(skill.ConformanceIssues) == 0 {
		t.Fatal("invalid artifact should retain warning issues")
	}
}

func TestParseRetainsMalformedFrontmatterArtifact(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "broken-yaml")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "SKILL.md")
	content := []byte("---\nname: [unterminated\ndescription: broken\n---\nBody")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	skill, err := ParseSkillFile(path, model.Codex)
	if err != nil {
		t.Fatalf("malformed frontmatter should remain discoverable: %v", err)
	}
	if skill.Name != "broken-yaml" || skill.Content != "Body" {
		t.Fatalf("retained artifact lost identity or body: %#v", skill)
	}
	foundParseIssue := false
	for _, issue := range skill.ConformanceIssues {
		if issue.Code == "frontmatter.parse" {
			foundParseIssue = true
			break
		}
	}
	if !foundParseIssue {
		t.Fatalf("missing structured frontmatter parse issue: %#v", skill.ConformanceIssues)
	}
}
