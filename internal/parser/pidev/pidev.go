// Package pidev implements the Parser interface for Pi.dev skills.
package pidev

import (
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser/skills"
	"github.com/klauern/skillsync/internal/util"
)

// Parser implements the parser.Parser interface for Pi.dev skills.
type Parser struct {
	basePath string
}

// New creates a new Pi.dev parser.
// If basePath is empty, it uses the default Pi.dev skills directory.
func New(basePath string) *Parser {
	if basePath == "" {
		basePath = util.PiDevSkillsPath()
	}
	return &Parser{basePath: basePath}
}

// Parse parses Pi.dev skills from SKILL.md files.
func (p *Parser) Parse() ([]model.Skill, error) {
	return skills.New(p.basePath, model.PiDev).Parse()
}

// Platform returns the platform this parser handles.
func (p *Parser) Platform() model.Platform {
	return model.PiDev
}

// DefaultPath returns the default user-level Pi.dev skills path.
func (p *Parser) DefaultPath() string {
	return util.PiDevSkillsPath()
}
