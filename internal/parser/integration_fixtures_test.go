package parser_test

import (
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/codex"
	"github.com/klauern/skillsync/internal/parser/cursor"
)

func TestFixtureDiscoveryCountsByPlatform(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "testdata", "skills")

	tests := map[string]struct {
		parse func() (int, error)
		want  int
	}{
		"claude-code": {
			parse: func() (int, error) {
				skills, err := claude.New(filepath.Join(fixtureRoot, "claude")).Parse()
				return len(skills), err
			},
			want: 7,
		},
		"cursor": {
			parse: func() (int, error) {
				skills, err := cursor.New(filepath.Join(fixtureRoot, "cursor")).Parse()
				return len(skills), err
			},
			want: 2,
		},
		"codex": {
			parse: func() (int, error) {
				skills, err := codex.New(filepath.Join(fixtureRoot, "codex")).Parse()
				return len(skills), err
			},
			want: 5,
		},
	}

	for platform, tt := range tests {
		t.Run(platform, func(t *testing.T) {
			got, err := tt.parse()
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if got != tt.want {
				t.Fatalf("discovered %d skills, want %d", got, tt.want)
			}
		})
	}
}
