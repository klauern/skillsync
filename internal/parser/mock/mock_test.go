package mock

import (
	"errors"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/util"
)

func TestNew(t *testing.T) {
	p := New(model.ClaudeCode)

	util.AssertEqual(t, p.Platform(), model.ClaudeCode)
	util.AssertEqual(t, p.DefaultPath(), "/mock/claude-code")
	util.AssertEqual(t, p.ParseCalled(), 0)
}

func TestParser_WithSkills(t *testing.T) {
	skills := []model.Skill{
		{Name: "skill-1", Content: "content 1"},
		{Name: "skill-2", Content: "content 2"},
	}

	p := New(model.Cursor).WithSkills(skills)

	result, err := p.Parse()
	util.AssertNoError(t, err)
	util.AssertEqual(t, len(result), 2)
	util.AssertEqual(t, result[0].Name, "skill-1")
	util.AssertEqual(t, result[1].Name, "skill-2")
}

func TestParser_WithError(t *testing.T) {
	expectedErr := errors.New("parse failed")
	p := New(model.Codex).WithError(expectedErr)

	result, err := p.Parse()
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestParser_ParseCalled(t *testing.T) {
	p := New(model.ClaudeCode)

	util.AssertEqual(t, p.ParseCalled(), 0)

	_, _ = p.Parse()
	util.AssertEqual(t, p.ParseCalled(), 1)

	_, _ = p.Parse()
	util.AssertEqual(t, p.ParseCalled(), 2)

	p.Reset()
	util.AssertEqual(t, p.ParseCalled(), 0)
}

func TestParser_WithDefaultPath(t *testing.T) {
	p := New(model.Cursor).WithDefaultPath("/custom/path")
	util.AssertEqual(t, p.DefaultPath(), "/custom/path")
}

func TestParser_EmptySkills(t *testing.T) {
	p := New(model.ClaudeCode)

	result, err := p.Parse()
	util.AssertNoError(t, err)
	util.AssertEqual(t, len(result), 0)
}

func TestParser_FullSkillStructure(t *testing.T) {
	now := time.Now()
	skills := []model.Skill{
		{
			Name:        "full-skill",
			Description: "A fully populated skill",
			Platform:    model.ClaudeCode,
			Path:        "full-skill.md",
			Tools:       []string{"read", "write", "bash"},
			Metadata:    map[string]string{"category": "test"},
			Content:     "# Full Skill\n\nContent here.",
			ModifiedAt:  now,
		},
	}

	p := New(model.ClaudeCode).WithSkills(skills)
	result, err := p.Parse()
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result), 1)
	util.AssertEqual(t, result[0].Name, "full-skill")
	util.AssertEqual(t, result[0].Description, "A fully populated skill")
	util.AssertEqual(t, len(result[0].Tools), 3)
	util.AssertEqual(t, result[0].Metadata["category"], "test")
}
