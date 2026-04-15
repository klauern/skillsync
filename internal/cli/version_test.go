package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	tests := map[string]struct {
		args       []string
		wantErr    bool
		wantOutput []string
	}{
		"version command outputs version info": {
			args:    []string{"skillsync", "version"},
			wantErr: false,
			wantOutput: []string{
				"skillsync version",
				"commit:",
				"built:",
				"go:",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run command
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

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check all expected output substrings
			if !tt.wantErr {
				for _, want := range tt.wantOutput {
					if !strings.Contains(output, want) {
						t.Errorf("Run() output = %q, want substring %q", output, want)
					}
				}
			}
		})
	}
}

func TestVersionCommandOutputFormat(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run version command
	ctx := context.Background()
	err := Run(ctx, []string{"skillsync", "version"})

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

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify output format - should be 4 lines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 4 {
		t.Errorf("expected 4 lines of output, got %d: %q", len(lines), output)
	}

	// Verify first line starts with "skillsync version"
	if !strings.HasPrefix(lines[0], "skillsync version ") {
		t.Errorf("first line should start with 'skillsync version ', got %q", lines[0])
	}

	// Verify indentation of subsequent lines
	for i, line := range lines[1:] {
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("line %d should be indented with 2 spaces, got %q", i+2, line)
		}
	}

	// Verify each line contains expected label
	expectedLabels := []string{"version", "commit:", "built:", "go:"}
	for i, label := range expectedLabels {
		if !strings.Contains(lines[i], label) {
			t.Errorf("line %d should contain %q, got %q", i+1, label, lines[i])
		}
	}
}

func TestVersionCommandIncludesVariables(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run version command
	ctx := context.Background()
	err := Run(ctx, []string{"skillsync", "version"})

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

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify the actual variable values appear in output
	if !strings.Contains(output, Version) {
		t.Errorf("output should contain Version %q, got %q", Version, output)
	}
	if !strings.Contains(output, Commit) {
		t.Errorf("output should contain Commit %q, got %q", Commit, output)
	}
	if !strings.Contains(output, BuildDate) {
		t.Errorf("output should contain BuildDate %q, got %q", BuildDate, output)
	}
	if !strings.Contains(output, runtime.Version()) {
		t.Errorf("output should contain Go version %q, got %q", runtime.Version(), output)
	}
}

func TestVersionCommandReturnsNoError(t *testing.T) {
	// Capture stdout to prevent output during test
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run version command
	ctx := context.Background()
	err := Run(ctx, []string{"skillsync", "version"})

	// Restore stdout
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}
	os.Stdout = old

	// Drain pipe to prevent blocking
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}

	// Version command should never return an error
	if err != nil {
		t.Errorf("version command should return nil error, got %v", err)
	}
}

func TestVersionCommandDefinition(t *testing.T) {
	cmd := versionCommand()

	if cmd.Name != "version" {
		t.Errorf("command name = %q, want %q", cmd.Name, "version")
	}

	if cmd.Usage == "" {
		t.Error("command should have usage text")
	}

	if !strings.Contains(cmd.Usage, "version") {
		t.Errorf("usage should mention version, got %q", cmd.Usage)
	}

	if cmd.Action == nil {
		t.Error("command should have an action function")
	}
}
