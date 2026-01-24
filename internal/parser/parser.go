package parser

import "github.com/klauern/skillsync/internal/model"

// Parser defines the interface for platform-specific skill parsers
type Parser interface {
	// Parse parses skills from the platform's configuration
	Parse() ([]model.Skill, error)

	// Platform returns the platform this parser handles
	Platform() model.Platform

	// DefaultPath returns the default path to search for skills
	DefaultPath() string
}
