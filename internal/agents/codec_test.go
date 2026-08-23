package agents

import (
	"reflect"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestMarkdownRoundTripAndCanonicalPaths(t *testing.T) {
	t.Parallel()
	for _, platform := range []model.Platform{model.ClaudeCode, model.Copilot, model.Gemini} {
		platform := platform
		t.Run(string(platform), func(t *testing.T) {
			t.Parallel()
			a := model.CustomAgent{Name: "reviewer", Description: "Review changes", Platform: platform, Content: "Review the patch.\n", Tools: []string{"read", "search"}, Model: "fast", Native: map[string]any{"preview": true}}
			data, err := EncodeMarkdown(a)
			if err != nil {
				t.Fatal(err)
			}
			path, err := CanonicalPath(platform, a.Name)
			if err != nil {
				t.Fatal(err)
			}
			got, err := DecodeMarkdown(platform, path, data)
			if err != nil {
				t.Fatal(err)
			}
			if got.Name != a.Name || got.Description != a.Description || got.Content != a.Content || !reflect.DeepEqual(got.Tools, a.Tools) {
				t.Fatalf("round trip = %+v", got)
			}
		})
	}
}

func TestUnsupportedCodecAndPath(t *testing.T) {
	t.Parallel()
	if _, err := CanonicalPath(model.Codex, "reviewer"); err == nil {
		t.Fatal("CanonicalPath() error = nil")
	}
	if _, err := DecodeMarkdown(model.Copilot, ".github/agents/reviewer.md", []byte("---\ndescription: review\n---\nbody")); err == nil {
		t.Fatal("DecodeMarkdown() suffix error = nil")
	}
}
