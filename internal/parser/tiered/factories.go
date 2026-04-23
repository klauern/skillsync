// Package tiered provides factory functions for creating tiered parsers.
package tiered

import (
	"os"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/codex"
	"github.com/klauern/skillsync/internal/parser/copilot"
	"github.com/klauern/skillsync/internal/parser/cursor"
	"github.com/klauern/skillsync/internal/parser/gemini"
	"github.com/klauern/skillsync/internal/parser/piagent"
	"github.com/klauern/skillsync/internal/parser/pidev"
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

// PiAgentParserFactory returns a ParserFactory for Pi Agent.
func PiAgentParserFactory() ParserFactory {
	return func(basePath string) parser.Parser {
		return piagent.New(basePath)
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

// ParserFactoryFor returns the appropriate ParserFactory for a platform.
func ParserFactoryFor(platform model.Platform) ParserFactory {
	switch platform {
	case model.ClaudeCode:
		return ClaudeCodeParserFactory()
	case model.Cursor:
		return CursorParserFactory()
	case model.Codex:
		return CodexParserFactory()
	case model.PiAgent:
		return PiAgentParserFactory()
	case model.Copilot:
		return CopilotParserFactory()
	case model.Gemini:
		return GeminiParserFactory()
	case model.PiDev:
		return PiDevParserFactory()
	default:
		// Return a factory that creates Claude parsers as a fallback
		return ClaudeCodeParserFactory()
	}
}

// NewForPlatform creates a TieredParser for the given platform with sensible defaults.
// It uses the current working directory for repo-level discovery.
func NewForPlatform(platform model.Platform) (*Parser, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	cfg := Config{
		Platform:      platform,
		WorkingDir:    cwd,
		ParserFactory: ParserFactoryFor(platform),
	}
	if platform == model.PiAgent {
		searchPaths, err := piagent.DiscoverSearchPaths(cwd)
		if err != nil {
			return nil, err
		}
		cfg.SearchPaths = searchPaths
	}

	return New(cfg), nil
}

// NewForPlatformWithDir creates a TieredParser for the given platform and working directory.
func NewForPlatformWithDir(platform model.Platform, workingDir string) *Parser {
	cfg := Config{
		Platform:      platform,
		WorkingDir:    workingDir,
		ParserFactory: ParserFactoryFor(platform),
	}
	if platform == model.PiAgent {
		searchPaths, err := piagent.DiscoverSearchPaths(workingDir)
		if err == nil {
			cfg.SearchPaths = searchPaths
		}
	}
	return New(cfg)
}
