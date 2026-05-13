package tiered

import (
	"os"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

// TestParserFactoryFor_PiAgent tests that ParserFactoryFor correctly handles
// the PiAgent platform, returning a factory that creates a working parser.
func TestParserFactoryFor_PiAgent(t *testing.T) {
	factory, err := ParserFactoryFor(model.PiAgent)
	if err != nil {
		t.Fatalf("ParserFactoryFor(PiAgent) unexpected error: %v", err)
	}
	if factory == nil {
		t.Fatal("ParserFactoryFor(PiAgent) returned nil factory")
	}

	p := factory("/test/pi-agent/skills")
	if p == nil {
		t.Fatal("PiAgentParserFactory returned nil parser")
	}

	if p.Platform() != model.PiAgent {
		t.Errorf("parser.Platform() = %s, want %s", p.Platform(), model.PiAgent)
	}
}

// TestParserFactoryFor_AllPlatforms verifies all supported platforms return
// a non-nil factory and parser, including PiAgent.
func TestParserFactoryFor_AllPlatforms(t *testing.T) {
	platforms := []model.Platform{
		model.ClaudeCode,
		model.Cursor,
		model.Codex,
		model.PiAgent,
		model.Copilot,
		model.Gemini,
		model.PiDev,
	}

	for _, platform := range platforms {
		t.Run(string(platform), func(t *testing.T) {
			factory, err := ParserFactoryFor(platform)
			if err != nil {
				t.Fatalf("ParserFactoryFor(%s) unexpected error: %v", platform, err)
			}
			if factory == nil {
				t.Fatalf("ParserFactoryFor(%s) returned nil factory", platform)
			}

			p := factory("/tmp/test")
			if p == nil {
				t.Fatalf("factory for %s returned nil parser", platform)
			}

			if p.Platform() != platform {
				t.Errorf("parser.Platform() = %s, want %s", p.Platform(), platform)
			}
		})
	}
}

// TestNewForPlatform_PiAgent tests that NewForPlatform correctly creates a tiered
// parser for PiAgent, including the special DiscoverSearchPaths integration.
func TestNewForPlatform_PiAgent(t *testing.T) {
	p, err := NewForPlatform(model.PiAgent)
	if err != nil {
		t.Fatalf("NewForPlatform(PiAgent) error = %v", err)
	}
	if p == nil {
		t.Fatal("NewForPlatform(PiAgent) returned nil parser")
	}
	if p.Platform() != model.PiAgent {
		t.Errorf("Platform() = %s, want %s", p.Platform(), model.PiAgent)
	}
}

// TestNewForPlatformWithDir_PiAgent tests NewForPlatformWithDir for PiAgent with
// a specific working directory.
func TestNewForPlatformWithDir_PiAgent(t *testing.T) {
	workingDir := t.TempDir()
	p, err := NewForPlatformWithDir(model.PiAgent, workingDir)
	if err != nil {
		t.Fatalf("NewForPlatformWithDir(PiAgent) unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("NewForPlatformWithDir(PiAgent) returned nil parser")
	}
	if p.Platform() != model.PiAgent {
		t.Errorf("Platform() = %s, want %s", p.Platform(), model.PiAgent)
	}
}

// TestNewForPlatformWithDir_PiAgent_MalformedSettings verifies that a malformed
// .pi/settings.json does not crash NewForPlatformWithDir — the error is logged
// and the parser is returned with an empty search-path set.
func TestNewForPlatformWithDir_PiAgent_MalformedSettings(t *testing.T) {
	workingDir := t.TempDir()

	piDir := workingDir + "/.pi"
	if err := os.MkdirAll(piDir, 0o750); err != nil {
		t.Fatalf("failed to create .pi dir: %v", err)
	}
	if err := os.WriteFile(piDir+"/settings.json", []byte("not valid json {{"), 0o600); err != nil {
		t.Fatalf("failed to write malformed settings.json: %v", err)
	}

	p, err := NewForPlatformWithDir(model.PiAgent, workingDir)
	if err != nil {
		t.Fatalf("NewForPlatformWithDir(PiAgent) unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("NewForPlatformWithDir returned nil; expected graceful degradation")
	}
	if p.Platform() != model.PiAgent {
		t.Errorf("Platform() = %s, want %s", p.Platform(), model.PiAgent)
	}
}

// TestNewForPlatformWithDir_AllPlatforms verifies that NewForPlatformWithDir works
// for every supported platform including the newer ones (PiAgent, Copilot, Gemini, PiDev).
func TestNewForPlatformWithDir_AllPlatforms(t *testing.T) {
	platforms := []model.Platform{
		model.ClaudeCode,
		model.Cursor,
		model.Codex,
		model.PiAgent,
		model.Copilot,
		model.Gemini,
		model.PiDev,
	}

	workingDir := t.TempDir()

	for _, platform := range platforms {
		t.Run(string(platform), func(t *testing.T) {
			p, err := NewForPlatformWithDir(platform, workingDir)
			if err != nil {
				t.Fatalf("NewForPlatformWithDir(%s) unexpected error: %v", platform, err)
			}
			if p == nil {
				t.Fatalf("NewForPlatformWithDir(%s) returned nil parser", platform)
			}
			if p.Platform() != platform {
				t.Errorf("Platform() = %s, want %s", p.Platform(), platform)
			}
		})
	}
}
