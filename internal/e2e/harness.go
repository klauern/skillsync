// Package e2e provides testing infrastructure for end-to-end CLI tests.
// It includes test harness for running CLI commands, fixture management,
// and utilities for setting up isolated test environments.
package e2e

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/klauern/skillsync/internal/cli"
)

// Result contains the outcome of running a CLI command.
type Result struct {
	// Stdout contains the captured standard output.
	Stdout string
	// Stderr contains the captured standard error (currently unused, but reserved).
	Stderr string
	// Err is the error returned by the CLI command, if any.
	Err error
	// ExitCode is the inferred exit code (0 for success, 1 for error).
	ExitCode int
}

// Success returns true if the command completed without error.
func (r *Result) Success() bool {
	return r.Err == nil
}

// Harness provides a test harness for running E2E CLI tests.
// It manages environment isolation, temp directories, and output capture.
type Harness struct {
	t       *testing.T
	homeDir string
	env     map[string]string
}

// NewHarness creates a new E2E test harness.
// It sets up an isolated SKILLSYNC_HOME directory for testing and configures
// all platform paths to point to subdirectories within the test home.
func NewHarness(t *testing.T) *Harness {
	t.Helper()

	// Create isolated home directory for this test
	homeDir := t.TempDir()

	h := &Harness{
		t:       t,
		homeDir: homeDir,
		env:     make(map[string]string),
	}

	// Set default environment for isolation
	h.SetEnv("SKILLSYNC_HOME", homeDir)

	// Set platform paths to use test directories
	// Set BOTH old (deprecated) and new env vars for compatibility:
	// - Old vars (SKILLSYNC_*_PATH) are used by validation.GetPlatformPath() and sync package
	// - New vars (SKILLSYNC_*_SKILLS_PATHS) are used by config.applyEnvironment() and tiered parser
	h.SetEnv("SKILLSYNC_CLAUDE_CODE_PATH", homeDir+"/.claude/commands")
	h.SetEnv("SKILLSYNC_CURSOR_PATH", homeDir+"/.cursor/rules")
	h.SetEnv("SKILLSYNC_CODEX_PATH", homeDir+"/.codex")
	h.SetEnv("SKILLSYNC_CLAUDE_CODE_SKILLS_PATHS", homeDir+"/.claude/commands")
	h.SetEnv("SKILLSYNC_CURSOR_SKILLS_PATHS", homeDir+"/.cursor/rules")
	h.SetEnv("SKILLSYNC_CODEX_SKILLS_PATHS", homeDir+"/.codex")

	return h
}

// SetEnv sets an environment variable for CLI commands run through this harness.
// The environment will be restored after the test completes.
func (h *Harness) SetEnv(key, value string) {
	h.t.Helper()
	h.env[key] = value
	h.t.Setenv(key, value)
}

// HomeDir returns the isolated home directory for this test harness.
func (h *Harness) HomeDir() string {
	return h.homeDir
}

// Run executes a CLI command with the given arguments and captures the output.
// The command is run in an isolated environment with proper stdout capture.
func (h *Harness) Run(args ...string) *Result {
	h.t.Helper()

	// Prepend "skillsync" as the program name if not provided
	if len(args) == 0 || args[0] != "skillsync" {
		args = append([]string{"skillsync"}, args...)
	}

	// Capture stdout
	oldStdout := os.Stdout
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		h.t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = stdoutW

	// Read stdout concurrently to avoid pipe buffer deadlock.
	// If the command outputs more than the pipe buffer size (~64KB),
	// it will block waiting for the buffer to drain. We must read
	// concurrently while the command runs.
	var stdoutBuf bytes.Buffer
	var copyErr error
	copyDone := make(chan struct{})
	go func() {
		defer close(copyDone)
		_, copyErr = io.Copy(&stdoutBuf, stdoutR)
	}()

	// Run the command
	ctx := context.Background()
	cmdErr := cli.Run(ctx, args)

	// Restore stdout and close writer to signal EOF to the reader goroutine
	if err := stdoutW.Close(); err != nil {
		h.t.Fatalf("failed to close stdout pipe writer: %v", err)
	}
	os.Stdout = oldStdout

	// Wait for the reader goroutine to complete
	<-copyDone
	if copyErr != nil {
		h.t.Fatalf("failed to read captured stdout: %v", copyErr)
	}

	// Determine exit code
	exitCode := 0
	if cmdErr != nil {
		exitCode = 1
	}

	return &Result{
		Stdout:   stdoutBuf.String(),
		Err:      cmdErr,
		ExitCode: exitCode,
	}
}

// RunWithStdin executes a CLI command with stdin input and captures output.
// This is useful for testing commands that require user input.
func (h *Harness) RunWithStdin(stdin string, args ...string) *Result {
	h.t.Helper()

	// Prepend "skillsync" as the program name if not provided
	if len(args) == 0 || args[0] != "skillsync" {
		args = append([]string{"skillsync"}, args...)
	}

	// Set up stdin
	oldStdin := os.Stdin
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		h.t.Fatalf("failed to create stdin pipe: %v", err)
	}

	// Write stdin data
	go func() {
		defer func() {
			_ = stdinW.Close()
		}()
		if _, err := stdinW.WriteString(stdin); err != nil {
			// Can't use t.Fatal in goroutine, but error will be apparent from test failure
			return
		}
	}()
	os.Stdin = stdinR

	// Capture stdout
	oldStdout := os.Stdout
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		h.t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = stdoutW

	// Read stdout concurrently to avoid pipe buffer deadlock.
	// If the command outputs more than the pipe buffer size (~64KB),
	// it will block waiting for the buffer to drain. We must read
	// concurrently while the command runs.
	var stdoutBuf bytes.Buffer
	var copyErr error
	copyDone := make(chan struct{})
	go func() {
		defer close(copyDone)
		_, copyErr = io.Copy(&stdoutBuf, stdoutR)
	}()

	// Run the command
	ctx := context.Background()
	cmdErr := cli.Run(ctx, args)

	// Restore stdin and stdout, close writer to signal EOF to reader goroutine
	if err := stdoutW.Close(); err != nil {
		h.t.Fatalf("failed to close stdout pipe writer: %v", err)
	}
	os.Stdin = oldStdin
	os.Stdout = oldStdout

	// Wait for the reader goroutine to complete
	<-copyDone
	if copyErr != nil {
		h.t.Fatalf("failed to read captured stdout: %v", copyErr)
	}

	// Determine exit code
	exitCode := 0
	if cmdErr != nil {
		exitCode = 1
	}

	return &Result{
		Stdout:   stdoutBuf.String(),
		Err:      cmdErr,
		ExitCode: exitCode,
	}
}
