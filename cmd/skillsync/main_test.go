package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/cli"
)

func TestCLIInitialization(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run help command to verify CLI initializes correctly
	ctx := context.Background()
	err := cli.Run(ctx, []string{"skillsync", "--help"})

	// Restore stdout
	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe writer: %v", closeErr)
	}
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatalf("failed to read captured output: %v", copyErr)
	}
	output := buf.String()

	if err != nil {
		t.Fatalf("CLI initialization failed: %v", err)
	}

	// Verify help output contains expected content
	if !strings.Contains(output, "skillsync") {
		t.Errorf("expected help output to contain 'skillsync', got: %q", output)
	}
	if !strings.Contains(output, "USAGE") || !strings.Contains(output, "COMMANDS") {
		t.Errorf("expected help output to contain USAGE and COMMANDS sections, got: %q", output)
	}
}

func TestVersionFlag(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := cli.Run(ctx, []string{"skillsync", "--version"})

	// Restore stdout
	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe writer: %v", closeErr)
	}
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatalf("failed to read captured output: %v", copyErr)
	}
	output := buf.String()

	if err != nil {
		t.Fatalf("--version flag failed: %v", err)
	}

	// Verify version output
	if !strings.Contains(output, "skillsync") {
		t.Errorf("expected version output to contain 'skillsync', got: %q", output)
	}
}

func TestGlobalFlagsRecognized(t *testing.T) {
	tests := map[string]struct {
		args    []string
		wantErr bool
	}{
		"verbose flag": {
			args:    []string{"skillsync", "--verbose", "version"},
			wantErr: false,
		},
		"debug flag": {
			args:    []string{"skillsync", "--debug", "version"},
			wantErr: false,
		},
		"no-color flag": {
			args:    []string{"skillsync", "--no-color", "version"},
			wantErr: false,
		},
		"combined flags": {
			args:    []string{"skillsync", "--verbose", "--no-color", "version"},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			ctx := context.Background()
			err := cli.Run(ctx, tt.args)

			// Restore stdout
			if closeErr := w.Close(); closeErr != nil {
				t.Fatalf("failed to close pipe writer: %v", closeErr)
			}
			os.Stdout = old

			// Drain pipe
			var buf bytes.Buffer
			if _, copyErr := io.Copy(&buf, r); copyErr != nil {
				t.Fatalf("failed to read captured output: %v", copyErr)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAllCommandsRegistered(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := cli.Run(ctx, []string{"skillsync", "--help"})

	// Restore stdout
	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe writer: %v", closeErr)
	}
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatalf("failed to read captured output: %v", copyErr)
	}
	output := buf.String()

	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	// Verify all expected commands are registered
	expectedCommands := []string{
		"version",
		"config",
		"sync",
		"discover",
		"compare",
		"dedupe",
		"export",
		"backup",
		"promote",
		"demote",
		"scope",
	}

	for _, cmd := range expectedCommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("expected command %q to be registered, help output: %q", cmd, output)
		}
	}
}

func TestHelpSubcommand(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := cli.Run(ctx, []string{"skillsync", "help"})

	// Restore stdout
	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe writer: %v", closeErr)
	}
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatalf("failed to read captured output: %v", copyErr)
	}
	output := buf.String()

	if err != nil {
		t.Fatalf("help subcommand failed: %v", err)
	}

	// Verify help output
	if !strings.Contains(output, "skillsync") {
		t.Errorf("expected help output to contain 'skillsync', got: %q", output)
	}
}
