package trust

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestPolicyBlocksExecutableByDefaultAndAllowsOverride(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "run.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	skill := model.Skill{Name: "unsafe", Path: root}
	decisions, err := (Policy{}).Evaluate(skill, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 1 || decisions[0].Risk != RiskExecutable || decisions[0].Allowed {
		t.Fatalf("unexpected default decisions: %#v", decisions)
	}
	policy, err := ParseAllowed("executable")
	if err != nil {
		t.Fatal(err)
	}
	decisions, err = policy.Evaluate(skill, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 1 || !decisions[0].Allowed {
		t.Fatalf("unexpected override decisions: %#v", decisions)
	}
}

func TestPolicyBlocksExternalReferenceAndNativeConfig(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "SKILL.md")
	if err := os.WriteFile(path, []byte("safe"), 0o644); err != nil {
		t.Fatal(err)
	}
	skill := model.Skill{
		Name: "unsafe", Path: path,
		References: []string{"https://example.com/instructions"},
		Metadata:   map[string]string{"hooks": "pre-run"},
	}
	decisions, err := (Policy{}).Evaluate(skill, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 2 {
		t.Fatalf("got %d decisions: %#v", len(decisions), decisions)
	}
	for _, decision := range decisions {
		if decision.Allowed {
			t.Fatalf("default policy allowed %#v", decision)
		}
	}
}

func TestPolicyBlocksDeclaredScriptWithoutExecuteBit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "run.sh")
	if err := os.WriteFile(path, []byte("echo safe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	decisions, err := (Policy{}).Evaluate(model.Skill{Path: root, Scripts: []string{path}}, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 1 || decisions[0].Risk != RiskExecutable || decisions[0].Allowed {
		t.Fatalf("unexpected decisions: %#v", decisions)
	}
}

func TestPolicyBlocksCodexRuntimeConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("instructions = 'safe'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	skill := model.Skill{Name: "codex-config", Path: path, Metadata: map[string]string{"approval_policy": "never"}}
	decisions, err := (Policy{}).Evaluate(skill, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 1 || decisions[0].Risk != RiskNativeConfig || decisions[0].Allowed {
		t.Fatalf("unexpected decisions: %#v", decisions)
	}
}

func TestParseAllowedRejectsUnknownCategory(t *testing.T) {
	if _, err := ParseAllowed("everything"); err == nil {
		t.Fatal("expected an error")
	}
}
