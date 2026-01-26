package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/logging"
)

func TestNew_TextOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{
		Level:  logging.LevelInfo,
		Output: &buf,
		JSON:   false,
	})

	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("expected output to contain 'key=value', got: %s", output)
	}
}

func TestNew_JSONOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{
		Level:  logging.LevelInfo,
		Output: &buf,
		JSON:   true,
	})

	logger.Info("test message", "key", "value")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if entry["msg"] != "test message" {
		t.Errorf("expected msg='test message', got: %v", entry["msg"])
	}
	if entry["key"] != "value" {
		t.Errorf("expected key='value', got: %v", entry["key"])
	}
}

func TestNew_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{
		Level:  logging.LevelWarn,
		Output: &buf,
	})

	// These should be filtered out
	logger.Debug("debug message")
	logger.Info("info message")

	// This should appear
	logger.Warn("warn message")

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Error("debug message should be filtered at warn level")
	}
	if strings.Contains(output, "info message") {
		t.Error("info message should be filtered at warn level")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("warn message should appear at warn level")
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := logging.DefaultOptions()

	if opts.Level != logging.LevelInfo {
		t.Errorf("expected default level to be Info, got: %v", opts.Level)
	}
	if opts.JSON {
		t.Error("expected default JSON to be false")
	}
	if opts.AddSource {
		t.Error("expected default AddSource to be false")
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{
		Level:  logging.LevelInfo,
		Output: &buf,
	})
	logging.SetDefault(logger)

	childLogger := logging.With("component", "test")
	childLogger.Info("child message")

	output := buf.String()
	if !strings.Contains(output, "component=test") {
		t.Errorf("expected output to contain 'component=test', got: %s", output)
	}
}

func TestContextLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{
		Level:  logging.LevelInfo,
		Output: &buf,
	})

	ctx := logging.NewContext(context.Background(), logger)
	retrieved := logging.FromContext(ctx)

	if retrieved == nil {
		t.Fatal("expected logger from context, got nil")
	}

	retrieved.Info("context message")
	if !strings.Contains(buf.String(), "context message") {
		t.Error("expected logger from context to write to buffer")
	}
}

func TestFromContext_Nil(t *testing.T) {
	ctx := context.Background()
	logger := logging.FromContext(ctx)

	if logger != nil {
		t.Error("expected nil logger from empty context")
	}
}

func TestWithContext_FallbackToDefault(t *testing.T) {
	var buf bytes.Buffer
	defaultLogger := logging.New(logging.Options{
		Level:  logging.LevelInfo,
		Output: &buf,
	})
	logging.SetDefault(defaultLogger)

	// Context without logger should fall back to default
	ctx := context.Background()
	logger := logging.WithContext(ctx)
	logger.Info("fallback message")

	if !strings.Contains(buf.String(), "fallback message") {
		t.Error("expected WithContext to fall back to default logger")
	}
}

func TestAttributeHelpers(t *testing.T) {
	tests := []struct {
		name    string
		attr    slog.Attr
		wantKey string
		wantVal string
	}{
		{
			name:    "Platform",
			attr:    logging.Platform("claude-code"),
			wantKey: "platform",
			wantVal: "claude-code",
		},
		{
			name:    "Skill",
			attr:    logging.Skill("my-skill"),
			wantKey: "skill",
			wantVal: "my-skill",
		},
		{
			name:    "Path",
			attr:    logging.Path("/home/user/.config"),
			wantKey: "path",
			wantVal: "/home/user/.config",
		},
		{
			name:    "Operation",
			attr:    logging.Operation("sync"),
			wantKey: "operation",
			wantVal: "sync",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.attr.Key != tt.wantKey {
				t.Errorf("got key %q, want %q", tt.attr.Key, tt.wantKey)
			}
			if tt.attr.Value.String() != tt.wantVal {
				t.Errorf("got value %q, want %q", tt.attr.Value.String(), tt.wantVal)
			}
		})
	}
}

func TestErr_NilError(t *testing.T) {
	attr := logging.Err(nil)
	if attr.Key != "" {
		t.Errorf("expected empty key for nil error, got: %q", attr.Key)
	}
}

func TestErr_WithError(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(logging.Options{
		Level:  logging.LevelInfo,
		Output: &buf,
		JSON:   true,
	})

	testErr := &testError{msg: "test error"}
	logger.Info("error occurred", logging.Err(testErr))

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["error"] == nil {
		t.Error("expected error field in output")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
