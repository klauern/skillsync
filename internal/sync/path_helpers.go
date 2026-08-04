package sync

import (
	"path/filepath"

	"github.com/klauern/skillsync/internal/model"
)

// shouldLinkClaudeDirectorySkill reports whether a Claude-sourced directory skill
// should be linked (symlinked or copied as a directory) into the target platform,
// rather than being flat-transformed into a single file.
//
// Returns true when:
//   - the skill originates from ClaudeCode,
//   - its path resolves to a directory (SourceTypeDirectory), and
//   - the target is a platform that accepts directory-layout skills
//     (i.e. not ClaudeCode itself and not Gemini, which uses flat files).
func shouldLinkClaudeDirectorySkill(skill model.Skill, target model.Platform) bool {
	if skill.Platform != model.ClaudeCode {
		return false
	}
	sourceType, _ := detectSourceType(skill.Path)
	if sourceType != SourceTypeDirectory {
		return false
	}
	return target != model.ClaudeCode && target != model.Gemini
}

// needsCanonicalEntrypointCopy reports whether a directory skill has a
// case-variant entrypoint that cannot safely be linked verbatim.
func needsCanonicalEntrypointCopy(skill model.Skill, target model.Platform) bool {
	return shouldLinkClaudeDirectorySkill(skill, target) && filepath.Base(skill.Path) != "SKILL.md"
}
