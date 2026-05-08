// Package gemini implements the Parser interface for Gemini CLI skills and GEMINI.md context files.
package gemini

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	coreparser "github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/parser/skills"
	"github.com/klauern/skillsync/internal/util"
)

// Parser implements parser.Parser for Gemini CLI artifacts.
type Parser struct {
	basePath string
}

// New creates a new Gemini parser rooted at the Gemini config directory.
func New(basePath string) *Parser {
	if basePath == "" {
		basePath = util.GeminiPath()
	}
	return &Parser{basePath: basePath}
}

// Parse discovers SKILL.md files under skills/ and GEMINI.md at the config root.
func (p *Parser) Parse() ([]model.Skill, error) {
	configRoot, skillsRoot := p.resolveRoots()

	if _, err := os.Stat(configRoot); os.IsNotExist(err) {
		logging.Debug(
			"config directory not found",
			logging.Platform(string(p.Platform())),
			logging.Path(configRoot),
		)
		return []model.Skill{}, nil
	}

	var allSkills []model.Skill
	seenNames := make(map[string]bool)

	skillsParser := skills.New(skillsRoot, p.Platform())
	agentSkills, err := skillsParser.Parse()
	if err != nil {
		logging.Warn(
			"failed to parse Gemini skills",
			logging.Platform(string(p.Platform())),
			logging.Path(skillsRoot),
			logging.Err(err),
		)
	} else {
		for _, skill := range agentSkills {
			seenNames[skill.Name] = true
			allSkills = append(allSkills, skill)
		}
	}

	contextSkill, err := p.parseContextFile(filepath.Join(configRoot, "GEMINI.md"))
	if err != nil {
		logging.Warn(
			"failed to parse GEMINI.md",
			logging.Platform(string(p.Platform())),
			logging.Path(filepath.Join(configRoot, "GEMINI.md")),
			logging.Err(err),
		)
	} else if contextSkill != nil && !seenNames[contextSkill.Name] {
		allSkills = append(allSkills, *contextSkill)
	}

	return allSkills, nil
}

// Platform returns the Gemini platform identifier.
func (p *Parser) Platform() model.Platform {
	return model.Gemini
}

// DefaultPath returns the default Gemini config root.
func (p *Parser) DefaultPath() string {
	return util.GeminiPath()
}

func (p *Parser) resolveRoots() (configRoot, skillsRoot string) {
	cleaned := filepath.Clean(p.basePath)
	if strings.EqualFold(filepath.Base(cleaned), "skills") {
		return filepath.Dir(cleaned), cleaned
	}
	return cleaned, filepath.Join(cleaned, "skills")
}

func (p *Parser) parseContextFile(filePath string) (*model.Skill, error) {
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	// #nosec G304 - filePath is discovered under the configured Gemini root.
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}

	return &model.Skill{
		Name:        "gemini-md",
		Description: "Gemini CLI GEMINI.md instructions",
		Platform:    model.Gemini,
		Path:        filePath,
		Metadata:    map[string]string{"type": "instructions"},
		Content:     coreparser.NormalizeContent(string(content)),
		ModifiedAt:  fileInfo.ModTime(),
		Type:        model.SkillTypeSkill,
	}, nil
}
