package model

import "testing"

func TestHookConfigValidate(t *testing.T) {
	t.Parallel()
	valid := HookConfig{Name: "audit", Platform: Codex, Event: "PreToolUse", Command: "./audit.sh"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	for name, hook := range map[string]HookConfig{
		"name":     {Platform: Codex, Event: "PreToolUse", Command: "true"},
		"platform": {Name: "audit", Event: "PreToolUse", Command: "true"},
		"event":    {Name: "audit", Platform: Codex, Command: "true"},
		"command":  {Name: "audit", Platform: Codex, Event: "PreToolUse"},
		"timeout":  {Name: "audit", Platform: Codex, Event: "PreToolUse", Command: "true", Timeout: -1},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := hook.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}
