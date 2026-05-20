// Package tiered provides factory functions for creating tiered parsers.
package tiered

import (
	"fmt"
	"os"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/codex"
	"github.com/klauern/skillsync/internal/parser/copilot"
	"github.com/klauern/skillsync/internal/parser/cursor"
	"github.com/klauern/skillsync/internal/parser/gemini"
	"github.com/klauern/skillsync/internal/parser/pidev"
	"github.com/klauern/skillsync/internal/util"
)

// ClaudeCodeParserFactory returns a ParserFactory for Claude Code.
func ClaudeCodeParserFactory() ParserFactory {
	return func(basePath string) parser.Parser {
		return claude.New(basePath)
	}
}

// CursorParserFactory returns a ParserFactory for Cursor.
func CursorParserFactory() ParserFactory {
	return func(basePath string) parser.Parser {
		return cursor.New(basePath)
	}
}

// CodexParserFactory returns a ParserFactory for Codex.
func CodexParserFactory() ParserFactory {
	return func(basePath string) parser.Parser {
		return codex.New(basePath)
	}
}

// CopilotParserFactory returns a ParserFactory for GitHub Copilot.
func CopilotParserFactory() ParserFactory {
	return func(basePath string) parser.Parser {
		return copilot.New(basePath)
	}
}

// GeminiParserFactory returns a ParserFactory for Gemini CLI.
func GeminiParserFactory() ParserFactory {
	return func(basePath string) parser.Parser {
		return gemini.New(basePath)
	}
}

// PiDevParserFactory returns a ParserFactory for Pi.dev.
func PiDevParserFactory() ParserFactory {
	return func(basePath string) parser.Parser {
		return pidev.New(basePath)
	}
}

// ParserFactoryFor returns the appropriate ParserFactory for a platform, or an
// error if the platform is not recognized. Callers that receive an error should
// treat the platform as unsupported rather than falling back to another parser.
func ParserFactoryFor(platform model.Platform) (ParserFactory, error) {
	switch platform {
	case model.ClaudeCode:
		return ClaudeCodeParserFactory(), nil
	case model.Cursor:
		return CursorParserFactory(), nil
	case model.Codex:
		return CodexParserFactory(), nil
	case model.Copilot:
		return CopilotParserFactory(), nil
	case model.Gemini:
		return GeminiParserFactory(), nil
	case model.PiDev:
		return PiDevParserFactory(), nil
	default:
		return nil, fmt.Errorf("no parser factory for platform %q", platform)
	}
}

// NewForPlatform creates a TieredParser for the given platform with sensible defaults.
// It uses the current working directory for repo-level discovery.
func NewForPlatform(platform model.Platform) (*Parser, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	factory, err := ParserFactoryFor(platform)
	if err != nil {
		return nil, err
	}

	cfg := Config{
		Platform:      platform,
		WorkingDir:    cwd,
		ParserFactory: factory,
	}
	if platform == model.PiDev {
		cfg.SearchPaths = util.GetAllSearchPaths(util.TieredPathConfig{WorkingDir: cwd, Platform: platform})
	}

	return New(cfg), nil
}

// NewForPlatformWithDir creates a TieredParser for the given platform and working directory.
func NewForPlatformWithDir(platform model.Platform, workingDir string) (*Parser, error) {
	factory, err := ParserFactoryFor(platform)
	if err != nil {
		return nil, err
	}

	cfg := Config{
		Platform:      platform,
		WorkingDir:    workingDir,
		ParserFactory: factory,
	}
	if platform == model.PiDev {
		cfg.SearchPaths = util.GetAllSearchPaths(util.TieredPathConfig{WorkingDir: workingDir, Platform: platform})
	}
	return New(cfg), nil
}
