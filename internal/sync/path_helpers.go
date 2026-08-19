package sync

import (
	"path/filepath"

	"github.com/klauern/skillsync/internal/model"
)

// shouldLinkClaudeDirectorySkill reports whether a same-harness Claude
// directory skill can be linked without bypassing cross-harness frontmatter
// filtering.
//
// Returns true when:
//   - the skill originates from ClaudeCode,
//   - its path resolves to a directory (SourceTypeDirectory), and
//   - the target is Claude Code as well.
func shouldLinkClaudeDirectorySkill(skill model.Skill, target model.Platform) bool {
	if skill.Platform != model.ClaudeCode {
		return false
	}
	sourceType, _ := detectSourceType(skill.Path)
	if sourceType != SourceTypeDirectory {
		return false
	}
	return target == model.ClaudeCode
}

// needsCanonicalEntrypointCopy reports whether a directory skill has a
// case-variant entrypoint that cannot safely be linked verbatim.
func needsCanonicalEntrypointCopy(skill model.Skill, target model.Platform) bool {
	return shouldLinkClaudeDirectorySkill(skill, target) && filepath.Base(skill.Path) != "SKILL.md"
}
