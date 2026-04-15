// Package mock provides mock implementations of parsers for testing.
package mock

import (
	"github.com/klauern/skillsync/internal/model"
)

// Parser is a mock implementation of parser.Parser for testing.
type Parser struct {
	platform    model.Platform
	defaultPath string
	skills      []model.Skill
	parseError  error
	parseCalled int
}

// New creates a new mock parser.
func New(platform model.Platform) *Parser {
	return &Parser{
		platform:    platform,
		defaultPath: "/mock/" + string(platform),
		skills:      []model.Skill{},
	}
}

// WithSkills configures the parser to return the given skills.
func (p *Parser) WithSkills(skills []model.Skill) *Parser {
	p.skills = skills
	return p
}

// WithError configures the parser to return an error.
func (p *Parser) WithError(err error) *Parser {
	p.parseError = err
	return p
}

// WithDefaultPath sets the default path.
func (p *Parser) WithDefaultPath(path string) *Parser {
	p.defaultPath = path
	return p
}

// Parse implements parser.Parser.
func (p *Parser) Parse() ([]model.Skill, error) {
	p.parseCalled++
	if p.parseError != nil {
		return nil, p.parseError
	}
	return p.skills, nil
}

// Platform implements parser.Parser.
func (p *Parser) Platform() model.Platform {
	return p.platform
}

// DefaultPath implements parser.Parser.
func (p *Parser) DefaultPath() string {
	return p.defaultPath
}

// ParseCalled returns the number of times Parse was called.
func (p *Parser) ParseCalled() int {
	return p.parseCalled
}

// Reset resets the call counters.
func (p *Parser) Reset() {
	p.parseCalled = 0
}
