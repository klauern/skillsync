package model

import "testing"

func TestCustomAgentValidateAndKey(t *testing.T) {
	t.Parallel()
	a := CustomAgent{Name: "reviewer", Description: "Review changes", Platform: ClaudeCode}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := a.Key(); got != "claude-code:agent:reviewer" {
		t.Fatalf("Key() = %q", got)
	}
	a.Description = ""
	if err := a.Validate(); err == nil {
		t.Fatal("Validate() error = nil")
	}
}

func TestCustomAgentRejectsUnsafeNames(t *testing.T) {
	t.Parallel()
	for _, name := range []string{" reviewer", "reviewer ", ".", "..", "../reviewer", "reviewer\\..", "reviewer\x00agent"} {
		t.Run(name, func(t *testing.T) {
			a := CustomAgent{Name: name, Description: "Review changes", Platform: ClaudeCode}
			if err := a.Validate(); err == nil {
				t.Fatalf("Validate(%q) error = nil", name)
			}
		})
	}
}
