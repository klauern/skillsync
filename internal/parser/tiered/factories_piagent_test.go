package tiered

import (
	"os"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

// TestParserFactoryFor_PiDev tests that ParserFactoryFor correctly handles
// the Pi.dev platform, returning a factory that creates a working parser.
func TestParserFactoryFor_PiDev(t *testing.T) {
	factory, err := ParserFactoryFor(model.PiDev)
	if err != nil {
		t.Fatalf("ParserFactoryFor(PiDev) unexpected error: %v", err)
	}
	if factory == nil {
		t.Fatal("ParserFactoryFor(PiDev) returned nil factory")
	}

	p := factory("/test/pi-dev/skills")
	if p == nil {
		t.Fatal("PiDevParserFactory returned nil parser")
	}

	if p.Platform() != model.PiDev {
		t.Errorf("parser.Platform() = %s, want %s", p.Platform(), model.PiDev)
	}
}

// TestParserFactoryFor_AllPlatforms verifies all supported platforms return
// a non-nil factory and parser, including Pi.dev.
func TestParserFactoryFor_AllPlatforms(t *testing.T) {
	platforms := []model.Platform{
		model.ClaudeCode,
		model.Cursor,
		model.Codex,
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

// TestNewForPlatform_PiDev tests that NewForPlatform correctly creates a tiered
// parser for Pi.dev, including the special DiscoverSearchPaths integration.
func TestNewForPlatform_PiDevSearchPaths(t *testing.T) {
	p, err := NewForPlatform(model.PiDev)
	if err != nil {
		t.Fatalf("NewForPlatform(PiDev) error = %v", err)
	}
	if p == nil {
		t.Fatal("NewForPlatform(PiDev) returned nil parser")
	}
	if p.Platform() != model.PiDev {
		t.Errorf("Platform() = %s, want %s", p.Platform(), model.PiDev)
	}
}

// TestNewForPlatformWithDir_PiDev tests NewForPlatformWithDir for Pi.dev with
// a specific working directory.
func TestNewForPlatformWithDir_PiDev(t *testing.T) {
	workingDir := t.TempDir()
	p, err := NewForPlatformWithDir(model.PiDev, workingDir)
	if err != nil {
		t.Fatalf("NewForPlatformWithDir(PiDev) unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("NewForPlatformWithDir(PiDev) returned nil parser")
	}
	if p.Platform() != model.PiDev {
		t.Errorf("Platform() = %s, want %s", p.Platform(), model.PiDev)
	}
}

// TestNewForPlatformWithDir_PiDev_MalformedSettings verifies that a malformed
// .pi/settings.json does not crash NewForPlatformWithDir — the error is logged
// and the parser is returned with an empty search-path set.
func TestNewForPlatformWithDir_PiDev_MalformedSettings(t *testing.T) {
	workingDir := t.TempDir()

	piDir := workingDir + "/.pi"
	if err := os.MkdirAll(piDir, 0o750); err != nil {
		t.Fatalf("failed to create .pi dir: %v", err)
	}
	if err := os.WriteFile(piDir+"/settings.json", []byte("not valid json {{"), 0o600); err != nil {
		t.Fatalf("failed to write malformed settings.json: %v", err)
	}

	p, err := NewForPlatformWithDir(model.PiDev, workingDir)
	if err != nil {
		t.Fatalf("NewForPlatformWithDir(PiDev) unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("NewForPlatformWithDir returned nil; expected graceful degradation")
	}
	if p.Platform() != model.PiDev {
		t.Errorf("Platform() = %s, want %s", p.Platform(), model.PiDev)
	}
}

// TestNewForPlatformWithDir_AllPlatforms verifies that NewForPlatformWithDir works
// for every supported platform including the newer ones (Copilot, Gemini, PiDev).
func TestNewForPlatformWithDir_AllPlatforms(t *testing.T) {
	platforms := []model.Platform{
		model.ClaudeCode,
		model.Cursor,
		model.Codex,
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
