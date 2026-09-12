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
	for _, name := range []string{"../reviewer", "reviewer/../../escape", "reviewer\\..", ".", "..", " reviewer", "reviewer "} {
		if path, err := CanonicalPath(model.ClaudeCode, name); err == nil {
			t.Fatalf("CanonicalPath(%q) = %q, want error", name, path)
		}
	}
	for _, path := range []string{".claude/agents/../reviewer.md", "/tmp/custom-agents/reviewer.md"} {
		if _, err := DecodeMarkdown(model.ClaudeCode, path, []byte("---\ndescription: review\n---\nbody")); err != nil {
			t.Fatalf("DecodeMarkdown(%q) source path error = %v", path, err)
		}
	}
}

func TestMarkdownCodecRejectsMalformedReservedFields(t *testing.T) {
	t.Parallel()
	path := ".claude/agents/reviewer.md"
	for _, frontmatter := range []string{
		"name: 42\ndescription: review",
		"name: reviewer\ndescription: 42",
		"name: reviewer\ndescription: review\nmodel: [fast]",
		"name: reviewer\ndescription: review\nmapping-key: [review]",
		"name: reviewer\ndescription: review\ntools: read",
		"name: reviewer\ndescription: review\ntools: [read, 42]",
	} {
		if _, err := DecodeMarkdown(model.ClaudeCode, path, []byte("---\n"+frontmatter+"\n---\nbody")); err == nil {
			t.Fatalf("DecodeMarkdown(%q) error = nil", frontmatter)
		}
	}
	got, err := DecodeMarkdown(model.ClaudeCode, path, []byte("---\nname: Display Name\ndescription: review\n---\nbody"))
	if err != nil {
		t.Fatalf("DecodeMarkdown() explicit name error = %v", err)
	}
	if got.Name != "Display Name" {
		t.Fatalf("DecodeMarkdown() name = %q, want explicit name", got.Name)
	}
}

func TestMarkdownEncoderRejectsReservedNativeFields(t *testing.T) {
	t.Parallel()
	for _, field := range []string{"name", "description", "tools", "model", "mapping-key"} {
		a := model.CustomAgent{Name: "reviewer", Description: "review", Platform: model.ClaudeCode, Native: map[string]any{field: "override"}}
		if _, err := EncodeMarkdown(a); err == nil {
			t.Fatalf("EncodeMarkdown() native %q error = nil", field)
		}
	}
}
