package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestPromoteCommand(t *testing.T) {
	tests := map[string]struct {
		args    []string
		wantErr bool
	}{
		"missing skill name": {
			args:    []string{"skillsync", "promote"},
			wantErr: true,
		},
		"promote non-existent skill": {
			args:    []string{"skillsync", "promote", "non-existent-skill", "--platform", "cursor"},
			wantErr: true,
		},
		"promote with invalid platform": {
			args:    []string{"skillsync", "promote", "my-skill", "--platform", "invalid"},
			wantErr: true,
		},
		"promote with invalid source scope": {
			args:    []string{"skillsync", "promote", "my-skill", "--from", "invalid"},
			wantErr: true,
		},
		"promote with invalid target scope": {
			args:    []string{"skillsync", "promote", "my-skill", "--to", "invalid"},
			wantErr: true,
		},
		"promote to non-writable scope": {
			args:    []string{"skillsync", "promote", "my-skill", "--to", "system"},
			wantErr: true,
		},
		"promote wrong direction": {
			args:    []string{"skillsync", "promote", "my-skill", "--from", "user", "--to", "repo"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			err := Run(ctx, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDemoteCommand(t *testing.T) {
	tests := map[string]struct {
		args    []string
		wantErr bool
	}{
		"missing skill name": {
			args:    []string{"skillsync", "demote"},
			wantErr: true,
		},
		"demote non-existent skill": {
			args:    []string{"skillsync", "demote", "non-existent-skill", "--platform", "cursor"},
			wantErr: true,
		},
		"demote with invalid platform": {
			args:    []string{"skillsync", "demote", "my-skill", "--platform", "invalid"},
			wantErr: true,
		},
		"demote with invalid source scope": {
			args:    []string{"skillsync", "demote", "my-skill", "--from", "invalid"},
			wantErr: true,
		},
		"demote with invalid target scope": {
			args:    []string{"skillsync", "demote", "my-skill", "--to", "invalid"},
			wantErr: true,
		},
		"demote wrong direction": {
			args:    []string{"skillsync", "demote", "my-skill", "--from", "repo", "--to", "user"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			err := Run(ctx, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScopeListCommand(t *testing.T) {
	tests := map[string]struct {
		args       []string
		wantErr    bool
		wantOutput string
	}{
		"scope list missing skill name": {
			args:    []string{"skillsync", "scope", "list"},
			wantErr: true,
		},
		"scope list non-existent skill": {
			args:       []string{"skillsync", "scope", "list", "non-existent-skill"},
			wantErr:    false,
			wantOutput: "not found",
		},
		"scope list with invalid platform": {
			args:    []string{"skillsync", "scope", "list", "my-skill", "--platform", "invalid"},
			wantErr: true,
		},
		"scope list with invalid format for nonexistent skill": {
			// When skill doesn't exist, we get "not found" before format matters
			args:       []string{"skillsync", "scope", "list", "nonexistent-skill-xyz", "--format", "invalid"},
			wantErr:    false,
			wantOutput: "not found",
		},
		"scope list --all": {
			args:       []string{"skillsync", "scope", "list", "--all"},
			wantErr:    false,
			wantOutput: "PLATFORM", // Header of the output table
		},
		"scope list --all with platform filter": {
			args:       []string{"skillsync", "scope", "list", "--all", "--platform", "cursor"},
			wantErr:    false,
			wantOutput: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			ctx := context.Background()
			err := Run(ctx, tt.args)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}
			output := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("Run() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestScopePruneCommand(t *testing.T) {
	tests := map[string]struct {
		args       []string
		wantErr    bool
		wantOutput string
	}{
		"prune missing platform": {
			args:    []string{"skillsync", "scope", "prune", "--scope", "user"},
			wantErr: true,
		},
		"prune missing scope": {
			args:    []string{"skillsync", "scope", "prune", "--platform", "cursor"},
			wantErr: true,
		},
		"prune with invalid platform": {
			args:    []string{"skillsync", "scope", "prune", "--platform", "invalid", "--scope", "user"},
			wantErr: true,
		},
		"prune with invalid scope": {
			args:    []string{"skillsync", "scope", "prune", "--platform", "cursor", "--scope", "invalid"},
			wantErr: true,
		},
		"prune builtin scope": {
			args:    []string{"skillsync", "scope", "prune", "--platform", "cursor", "--scope", "builtin"},
			wantErr: true,
		},
		"prune with conflicting keep-repo flag": {
			args:    []string{"skillsync", "scope", "prune", "--platform", "cursor", "--scope", "repo", "--keep-repo"},
			wantErr: true,
		},
		"prune with conflicting keep-user flag": {
			args:    []string{"skillsync", "scope", "prune", "--platform", "cursor", "--scope", "user", "--keep-user"},
			wantErr: true,
		},
		"prune no duplicates": {
			args:       []string{"skillsync", "scope", "prune", "--platform", "cursor", "--scope", "user", "--dry-run"},
			wantErr:    false,
			wantOutput: "No duplicate skills found",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			ctx := context.Background()
			err := Run(ctx, tt.args)

			// Restore stdout
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to read captured output: %v", err)
			}
			output := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("Run() output = %q, want substring %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestGetSkillPathForScope(t *testing.T) {
	tests := map[string]struct {
		platform   model.Platform
		scope      model.SkillScope
		skillName  string
		wantSuffix string
		wantErr    bool
	}{
		"repo scope cursor": {
			platform:   model.Cursor,
			scope:      model.ScopeRepo,
			skillName:  "my-skill",
			wantSuffix: filepath.Join(".cursor", "skills", "my-skill", "SKILL.md"),
			wantErr:    false,
		},
		"user scope cursor": {
			platform:   model.Cursor,
			scope:      model.ScopeUser,
			skillName:  "my-skill",
			wantSuffix: filepath.Join(".cursor", "skills", "my-skill", "SKILL.md"),
			wantErr:    false,
		},
		"repo scope claude-code": {
			platform:   model.ClaudeCode,
			scope:      model.ScopeRepo,
			skillName:  "test-skill",
			wantSuffix: filepath.Join(".claude", "skills", "test-skill", "SKILL.md"),
			wantErr:    false,
		},
		"admin scope not writable": {
			platform:  model.Cursor,
			scope:     model.ScopeAdmin,
			skillName: "my-skill",
			wantErr:   true,
		},
		"system scope not writable": {
			platform:  model.Cursor,
			scope:     model.ScopeSystem,
			skillName: "my-skill",
			wantErr:   true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := getSkillPathForScope(tt.platform, tt.scope, tt.skillName)

			if (err != nil) != tt.wantErr {
				t.Errorf("getSkillPathForScope() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !strings.HasSuffix(got, tt.wantSuffix) {
				t.Errorf("getSkillPathForScope() = %q, want suffix %q", got, tt.wantSuffix)
			}
		})
	}
}

func TestOutputAnyJSON(t *testing.T) {
	tests := map[string]struct {
		input   any
		wantErr bool
	}{
		"slice of strings": {
			input:   []string{"a", "b", "c"},
			wantErr: false,
		},
		"map": {
			input:   map[string]int{"one": 1, "two": 2},
			wantErr: false,
		},
		"struct": {
			input:   struct{ Name string }{"test"},
			wantErr: false,
		},
		"nil": {
			input:   nil,
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := outputAnyJSON(tt.input)

			// Restore stdout
			if closeErr := w.Close(); closeErr != nil {
				t.Fatalf("failed to close pipe writer: %v", closeErr)
			}
			os.Stdout = old

			// Drain the reader
			var buf bytes.Buffer
			if _, copyErr := io.Copy(&buf, r); copyErr != nil {
				t.Fatalf("failed to read output: %v", copyErr)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("outputAnyJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOutputAnyYAML(t *testing.T) {
	tests := map[string]struct {
		input   any
		wantErr bool
	}{
		"slice of strings": {
			input:   []string{"a", "b", "c"},
			wantErr: false,
		},
		"map": {
			input:   map[string]int{"one": 1, "two": 2},
			wantErr: false,
		},
		"struct": {
			input:   struct{ Name string }{"test"},
			wantErr: false,
		},
		"nil": {
			input:   nil,
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := outputAnyYAML(tt.input)

			// Restore stdout
			if closeErr := w.Close(); closeErr != nil {
				t.Fatalf("failed to close pipe writer: %v", closeErr)
			}
			os.Stdout = old

			// Drain the reader
			var buf bytes.Buffer
			if _, copyErr := io.Copy(&buf, r); copyErr != nil {
				t.Fatalf("failed to read output: %v", copyErr)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("outputAnyYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
