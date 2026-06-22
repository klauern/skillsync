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
	"github.com/klauern/skillsync/internal/util"
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
	home := filepath.Join(t.TempDir(), "home")
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(home, ".agents", "skills"), 0o750); err != nil {
		t.Fatalf("failed to create home .agents skills dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, ".agents", "skills"), 0o750); err != nil {
		t.Fatalf("failed to create repo .agents skills dir: %v", err)
	}
	t.Setenv("HOME", home)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("failed to chdir to repo root: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

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
		"repo scope codex": {
			platform:   model.Codex,
			scope:      model.ScopeRepo,
			skillName:  "test-skill",
			wantSuffix: filepath.Join(".codex", "skills", "test-skill", "SKILL.md"),
			wantErr:    false,
		},
		"user scope codex prefers agents": {
			platform:   model.Codex,
			scope:      model.ScopeUser,
			skillName:  "test-skill",
			wantSuffix: filepath.Join(".agents", "skills", "test-skill", "SKILL.md"),
			wantErr:    false,
		},
		"repo scope copilot": {
			platform:   model.Copilot,
			scope:      model.ScopeRepo,
			skillName:  "test-skill",
			wantSuffix: filepath.Join(".github", "agents", "test-skill.agent.md"),
			wantErr:    false,
		},
		"user scope copilot": {
			platform:   model.Copilot,
			scope:      model.ScopeUser,
			skillName:  "test-skill",
			wantSuffix: filepath.Join(".github", "agents", "test-skill.agent.md"),
			wantErr:    false,
		},
		"repo scope gemini": {
			platform:   model.Gemini,
			scope:      model.ScopeRepo,
			skillName:  "test-skill",
			wantSuffix: filepath.Join(".gemini", "skills", "test-skill", "SKILL.md"),
			wantErr:    false,
		},
		"user scope gemini": {
			platform:   model.Gemini,
			scope:      model.ScopeUser,
			skillName:  "test-skill",
			wantSuffix: filepath.Join(".gemini", "skills", "test-skill", "SKILL.md"),
			wantErr:    false,
		},
		"repo scope pi.dev": {
			platform:   model.PiDev,
			scope:      model.ScopeRepo,
			skillName:  "test-skill",
			wantSuffix: filepath.Join(".agents", "skills", "test-skill", "SKILL.md"),
			wantErr:    false,
		},
		"user scope pi.dev": {
			platform:   model.PiDev,
			scope:      model.ScopeUser,
			skillName:  "test-skill",
			wantSuffix: filepath.Join(".agents", "skills", "test-skill", "SKILL.md"),
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

func TestPromoteDemoteCommand_AllPlatforms(t *testing.T) {
	tests := []struct {
		name        string
		platformArg string
		platform    model.Platform
	}{
		{name: "claude-code", platformArg: "claude-code", platform: model.ClaudeCode},
		{name: "cursor", platformArg: "cursor", platform: model.Cursor},
		{name: "codex", platformArg: "codex", platform: model.Codex},
		{name: "copilot", platformArg: "copilot", platform: model.Copilot},
		{name: "gemini", platformArg: "gemini", platform: model.Gemini},
		{name: "pi.dev", platformArg: "pi.dev", platform: model.PiDev},
		{name: "pi-agent alias", platformArg: "pi-agent", platform: model.PiDev},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			home := filepath.Join(t.TempDir(), "home")
			if err := os.MkdirAll(home, 0o750); err != nil {
				t.Fatalf("failed to create home dir: %v", err)
			}
			if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o750); err != nil {
				t.Fatalf("failed to create repo marker: %v", err)
			}
			if err := os.MkdirAll(filepath.Join(repoRoot, ".agents", "skills"), 0o750); err != nil {
				t.Fatalf("failed to create repo .agents skills dir: %v", err)
			}
			if err := os.MkdirAll(filepath.Join(home, ".agents", "skills"), 0o750); err != nil {
				t.Fatalf("failed to create user .agents skills dir: %v", err)
			}

			t.Setenv("HOME", home)

			oldWd, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get working dir: %v", err)
			}
			if err := os.Chdir(repoRoot); err != nil {
				t.Fatalf("failed to chdir to repo root: %v", err)
			}
			defer func() { _ = os.Chdir(oldWd) }()

			promoteName := "repo-promote"
			demoteName := "user-demote"

			promoteSource := writeScopeMoveFixture(t, tt.platform, model.ScopeRepo, repoRoot, home, promoteName)
			promoteTarget, err := getSkillPathForScope(tt.platform, model.ScopeUser, promoteName)
			if err != nil {
				t.Fatalf("failed to determine promote target: %v", err)
			}

			ctx := context.Background()
			if err := Run(ctx, []string{"skillsync", "promote", promoteName, "--platform", tt.platformArg, "--force"}); err != nil {
				t.Fatalf("promote failed for %s: %v", tt.platformArg, err)
			}

			assertFileHasContent(t, promoteTarget, promoteName)
			assertFileHasContent(t, promoteSource, promoteName)

			demoteSource := writeScopeMoveFixture(t, tt.platform, model.ScopeUser, repoRoot, home, demoteName)
			demoteTarget, err := getSkillPathForScope(tt.platform, model.ScopeRepo, demoteName)
			if err != nil {
				t.Fatalf("failed to determine demote target: %v", err)
			}

			if err := Run(ctx, []string{"skillsync", "demote", demoteName, "--platform", tt.platformArg, "--force"}); err != nil {
				t.Fatalf("demote failed for %s: %v", tt.platformArg, err)
			}

			assertFileHasContent(t, demoteTarget, demoteName)
			assertFileHasContent(t, demoteSource, demoteName)
		})
	}
}

func writeScopeMoveFixture(
	t *testing.T,
	platform model.Platform,
	scope model.SkillScope,
	repoRoot, home, skillName string,
) string {
	t.Helper()

	var targetPath string

	switch platform {
	case model.Copilot:
		basePath := filepath.Join(home, ".github")
		if scope == model.ScopeRepo {
			basePath = filepath.Join(repoRoot, ".github")
		}
		targetPath = filepath.Join(basePath, "agents", skillName+".agent.md")
	case model.Gemini:
		basePath := filepath.Join(home, ".gemini")
		if scope == model.ScopeRepo {
			basePath = filepath.Join(repoRoot, ".gemini")
		}
		targetPath = filepath.Join(basePath, "skills", skillName, "SKILL.md")
	default:
		basePath := util.PlatformSkillsPath(platform)
		if scope == model.ScopeRepo {
			basePath = util.RepoSkillsPath(platform, repoRoot)
		}
		targetPath = filepath.Join(basePath, skillName, "SKILL.md")
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
		t.Fatalf("failed to create fixture dir for %s: %v", targetPath, err)
	}

	content := "---\nname: " + skillName + "\ndescription: test fixture\n---\n\n# " + skillName + "\n"
	if err := os.WriteFile(targetPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write fixture %s: %v", targetPath, err)
	}

	return targetPath
}

func assertFileHasContent(t *testing.T, path, want string) {
	t.Helper()

	data, err := os.ReadFile(path) // #nosec G304 -- path is test-controlled fixture.
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("file %s missing expected content %q", path, want)
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
