package sync

import (
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestPortabilityWarningsForSkill_NoLossyFields(t *testing.T) {
	skill := model.Skill{
		Platform: model.ClaudeCode,
		Name:     "clean-skill",
		Metadata: map[string]string{"description": "a regular skill"},
	}
	warnings := portabilityWarningsForSkill(skill, model.Codex)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

func TestPortabilityWarningsForSkill_LossyFieldDetected(t *testing.T) {
	skill := model.Skill{
		Platform: model.ClaudeCode,
		Name:     "hook-skill",
		Metadata: map[string]string{
			"hooks":   `{"PreToolUse":{"command":"echo hi"}}`,
			"context": "some-context",
			"model":   "claude-3-opus",
		},
	}

	warnings := portabilityWarningsForSkill(skill, model.Codex)

	wantFields := map[string]bool{"hooks": true, "context": true, "model": true}
	for _, w := range warnings {
		if !wantFields[w] {
			t.Errorf("unexpected warning field %q", w)
		}
		delete(wantFields, w)
	}
	for f := range wantFields {
		t.Errorf("expected warning for field %q, not found in %v", f, warnings)
	}
}

func TestPortabilityWarningsForSkill_CopilotToClaudeWarns(t *testing.T) {
	skill := model.Skill{
		Platform: model.Copilot,
		Name:     "agent",
		Metadata: map[string]string{"applyTo": "**/*.go"},
	}
	warnings := portabilityWarningsForSkill(skill, model.ClaudeCode)
	if len(warnings) != 1 || warnings[0] != "applyTo" {
		t.Errorf("expected [applyTo], got %v", warnings)
	}
}

func TestPortabilityWarningsForSkill_CopilotToCopilotNone(t *testing.T) {
	skill := model.Skill{
		Platform: model.Copilot,
		Name:     "agent",
		Metadata: map[string]string{"applyTo": "**/*.go"},
	}
	warnings := portabilityWarningsForSkill(skill, model.Copilot)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for copilot->copilot, got %v", warnings)
	}
}

func TestPortabilityWarningsForSkill_SortedOutput(t *testing.T) {
	skill := model.Skill{
		Platform: model.ClaudeCode,
		Name:     "multi-field",
		Metadata: map[string]string{
			"model":         "claude-3",
			"hooks":         "{}",
			"argument-hint": "some hint",
		},
	}
	warnings := portabilityWarningsForSkill(skill, model.Codex)
	for i := 1; i < len(warnings); i++ {
		if warnings[i] < warnings[i-1] {
			t.Errorf("warnings not sorted: %v", warnings)
			break
		}
	}
}
