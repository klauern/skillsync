// Package tiered provides factory functions for creating tiered parsers.
package tiered

import (
	"os"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/codex"
	"github.com/klauern/skillsync/internal/parser/cursor"
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

// ParserFactoryFor returns the appropriate ParserFactory for a platform.
func ParserFactoryFor(platform model.Platform) ParserFactory {
	switch platform {
	case model.ClaudeCode:
		return ClaudeCodeParserFactory()
	case model.Cursor:
		return CursorParserFactory()
	case model.Codex:
		return CodexParserFactory()
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

	return New(Config{
		Platform:      platform,
		WorkingDir:    cwd,
		ParserFactory: ParserFactoryFor(platform),
	}), nil
}

// NewForPlatformWithDir creates a TieredParser for the given platform and working directory.
func NewForPlatformWithDir(platform model.Platform, workingDir string) *Parser {
	return New(Config{
		Platform:      platform,
		WorkingDir:    workingDir,
		ParserFactory: ParserFactoryFor(platform),
	})
}
