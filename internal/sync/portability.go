package sync

import (
	"sort"

	"github.com/klauern/skillsync/internal/model"
)

// lossyFieldUnsupportedBy maps metadata key names to the platforms that do not natively support them.
// Derived from docs/platforms/portability-snapshot.yaml lossy_field_mappings.
// Fields present in a skill's Metadata that target a platform in this list will be flagged.
var lossyFieldUnsupportedBy = map[string][]model.Platform{
	"argument-hint":            {model.Codex, model.Cursor, model.Gemini, model.PiDev},
	"allowed-tools":            {model.Gemini, model.PiDev},
	"model":                    {model.Codex, model.Cursor, model.Gemini, model.PiDev},
	"disable-model-invocation": {model.Codex, model.Cursor, model.Gemini, model.PiDev},
	"user-invocable":           {model.Codex, model.Cursor, model.Gemini, model.PiDev},
	"context":                  {model.Codex, model.Copilot, model.Cursor, model.Gemini, model.PiDev},
	"hooks":                    {model.Codex, model.Copilot, model.Cursor, model.Gemini, model.PiDev},
	"applyTo":                  {model.ClaudeCode, model.Codex, model.Cursor, model.Gemini, model.PiDev},
	"globs":                    {model.ClaudeCode, model.Codex, model.Copilot, model.Gemini, model.PiDev},
}

// portabilityWarningsForSkill returns the sorted list of metadata field names that will be
// lossy or silently degraded when skill is written to target.
func portabilityWarningsForSkill(skill model.Skill, target model.Platform) []string {
	var warnings []string
	for field, unsupported := range lossyFieldUnsupportedBy {
		if _, ok := skill.Metadata[field]; !ok {
			continue
		}
		for _, p := range unsupported {
			if p == target {
				warnings = append(warnings, field)
				break
			}
		}
	}
	sort.Strings(warnings)
	return warnings
}
