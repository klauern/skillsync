package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/util"
)

func TestRunDetectCommand_TableOutput(t *testing.T) {
	home := t.TempDir()
	workdir := t.TempDir()
	t.Setenv("HOME", home)

	mustMkdirAll(t, filepath.Join(home, ".claude", "skills"))
	mustMkdirAll(t, filepath.Join(home, ".codex", "skills"))

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	output := captureStdout(t, func() {
		if err := Run(context.Background(), []string{"skillsync", "detect"}); err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
	})

	if !strings.Contains(output, "Detected 2 of 6 platform(s)") {
		t.Fatalf("unexpected detect summary:\n%s", output)
	}
	if !strings.Contains(output, "claude-code") {
		t.Fatalf("expected claude-code in output:\n%s", output)
	}
	if !strings.Contains(output, "codex") {
		t.Fatalf("expected codex in output:\n%s", output)
	}
	if !strings.Contains(output, "missing") {
		t.Fatalf("expected missing platform details in output:\n%s", output)
	}
}

func TestRunDetectCommand_JSONOutput(t *testing.T) {
	home := t.TempDir()
	workdir := t.TempDir()
	t.Setenv("HOME", home)

	mustMkdirAll(t, filepath.Join(home, ".claude", "skills"))
	mustMkdirAll(t, filepath.Join(home, ".codex", "skills"))
	mustMkdirAll(t, filepath.Join(workdir, ".github"))

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	output := captureStdout(t, func() {
		if err := Run(context.Background(), []string{"skillsync", "detect", "--format", "json"}); err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
	})

	var result util.PlatformDetectionResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput:\n%s", err, output)
	}
	if len(result.Detected) != 3 {
		t.Fatalf("expected 3 detected platforms, got %d", len(result.Detected))
	}
	if !result.HasPlatform("claude-code") {
		t.Fatalf("expected claude-code to be detected: %+v", result.Detected)
	}
	if !result.HasPlatform("codex") {
		t.Fatalf("expected codex to be detected: %+v", result.Detected)
	}
	if !result.HasPlatform("copilot") {
		t.Fatalf("expected copilot to be detected: %+v", result.Detected)
	}
}
