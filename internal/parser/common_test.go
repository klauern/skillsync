package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/util"
)

func TestSplitFrontmatter(t *testing.T) {
	tests := map[string]struct {
		input              string
		wantHasFrontmatter bool
		wantFrontmatter    string
		wantContent        string
	}{
		"yaml frontmatter": {
			input: `---
name: test-skill
description: A test skill
---
This is the content`,
			wantHasFrontmatter: true,
			wantFrontmatter:    "name: test-skill\ndescription: A test skill",
			wantContent:        "This is the content",
		},
		"yaml frontmatter with windows line endings": {
			input:              "---\r\nname: test\r\n---\r\nContent",
			wantHasFrontmatter: true,
			wantFrontmatter:    "name: test",
			wantContent:        "Content",
		},
		"alternative frontmatter (plus signs)": {
			input: `+++
name: test
+++
Content here`,
			wantHasFrontmatter: true,
			wantFrontmatter:    "name: test",
			wantContent:        "Content here",
		},
		"no frontmatter": {
			input:              "Just plain content",
			wantHasFrontmatter: false,
			wantFrontmatter:    "",
			wantContent:        "Just plain content",
		},
		"no closing delimiter": {
			input: `---
name: test
This looks like frontmatter but has no closing delimiter`,
			wantHasFrontmatter: false,
			wantFrontmatter:    "",
			wantContent:        "---\nname: test\nThis looks like frontmatter but has no closing delimiter",
		},
		"empty frontmatter": {
			input: `---
---
Content only`,
			wantHasFrontmatter: true,
			wantFrontmatter:    "",
			wantContent:        "Content only",
		},
		"empty content": {
			input: `---
name: test
---`,
			wantHasFrontmatter: true,
			wantFrontmatter:    "name: test",
			wantContent:        "",
		},
		"empty file": {
			input:              "",
			wantHasFrontmatter: false,
			wantFrontmatter:    "",
			wantContent:        "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := SplitFrontmatter([]byte(tt.input))

			if result.HasFrontmatter != tt.wantHasFrontmatter {
				t.Errorf("HasFrontmatter = %v, want %v", result.HasFrontmatter, tt.wantHasFrontmatter)
			}

			gotFrontmatter := string(result.Frontmatter)
			if gotFrontmatter != tt.wantFrontmatter {
				t.Errorf("Frontmatter = %q, want %q", gotFrontmatter, tt.wantFrontmatter)
			}

			if result.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", result.Content, tt.wantContent)
			}
		})
	}
}

func TestParseYAMLFrontmatter(t *testing.T) {
	tests := map[string]struct {
		input   string
		want    map[string]interface{}
		wantErr bool
	}{
		"valid yaml": {
			input: "name: test-skill\ndescription: A test",
			want: map[string]interface{}{
				"name":        "test-skill",
				"description": "A test",
			},
			wantErr: false,
		},
		"yaml with array": {
			input: "name: skill\ntools:\n  - Read\n  - Write",
			want: map[string]interface{}{
				"name":  "skill",
				"tools": []interface{}{"Read", "Write"},
			},
			wantErr: false,
		},
		"empty yaml": {
			input:   "",
			want:    map[string]interface{}{},
			wantErr: false,
		},
		"invalid yaml": {
			input:   "name: test\n  invalid: indentation",
			want:    nil,
			wantErr: true,
		},
		"malformed yaml": {
			input:   "{ this is not valid yaml",
			want:    nil,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParseYAMLFrontmatter([]byte(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseYAMLFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // Don't check value if we expected an error
			}

			// Compare keys
			if len(got) != len(tt.want) {
				t.Errorf("ParseYAMLFrontmatter() returned %d keys, want %d", len(got), len(tt.want))
			}

			for key, wantVal := range tt.want {
				gotVal, ok := got[key]
				if !ok {
					t.Errorf("ParseYAMLFrontmatter() missing key %q", key)
					continue
				}

				// For simple comparison (strings)
				if wantStr, ok := wantVal.(string); ok {
					if gotStr, ok := gotVal.(string); ok {
						if gotStr != wantStr {
							t.Errorf("ParseYAMLFrontmatter()[%q] = %q, want %q", key, gotStr, wantStr)
						}
					}
				}
			}
		})
	}
}

func TestDiscoverFiles(t *testing.T) {
	// Create temp directory with test files
	tempDir := util.CreateTempDir(t)

	// Create test file structure
	testFiles := []string{
		"skill1.md",
		"skill2.md",
		"subdir/skill3.md",
		"other.txt",
		".hidden.md",
	}

	for _, file := range testFiles {
		path := filepath.Join(tempDir, file)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		// #nosec G306 - test files don't need restrictive permissions
		if err := os.WriteFile(path, []byte("test content"), 0o600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	tests := map[string]struct {
		baseDir  string
		patterns []string
		want     []string
		wantErr  bool
	}{
		"single pattern": {
			baseDir:  tempDir,
			patterns: []string{"*.md"},
			want:     []string{".hidden.md", "skill1.md", "skill2.md"}, // glob matches hidden files too
			wantErr:  false,
		},
		"multiple patterns": {
			baseDir:  tempDir,
			patterns: []string{"*.md", "*.txt"},
			want:     []string{".hidden.md", "skill1.md", "skill2.md", "other.txt"},
			wantErr:  false,
		},
		"recursive pattern": {
			baseDir:  tempDir,
			patterns: []string{"**/*.md"},
			want:     []string{".hidden.md", "skill1.md", "skill2.md", "subdir/skill3.md"},
			wantErr:  false,
		},
		"hidden files only": {
			baseDir:  tempDir,
			patterns: []string{".*.md"},
			want:     []string{".hidden.md"},
			wantErr:  false,
		},
		"no matches": {
			baseDir:  tempDir,
			patterns: []string{"*.json"},
			want:     []string{},
			wantErr:  false,
		},
		"nonexistent directory": {
			baseDir:  filepath.Join(tempDir, "nonexistent"),
			patterns: []string{"*.md"},
			want:     []string{},
			wantErr:  false, // Not an error, just empty result
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := DiscoverFiles(tt.baseDir, tt.patterns)

			if (err != nil) != tt.wantErr {
				t.Errorf("DiscoverFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Convert absolute paths back to relative for comparison
			var gotRelative []string
			for _, absPath := range got {
				rel, err := filepath.Rel(tt.baseDir, absPath)
				if err != nil {
					t.Fatalf("failed to get relative path: %v", err)
				}
				gotRelative = append(gotRelative, rel)
			}

			// Sort both slices for comparison
			if len(gotRelative) != len(tt.want) {
				t.Errorf("DiscoverFiles() returned %d files, want %d\ngot: %v\nwant: %v",
					len(gotRelative), len(tt.want), gotRelative, tt.want)
				return
			}

			// Check each expected file is present
			gotMap := make(map[string]bool)
			for _, f := range gotRelative {
				gotMap[f] = true
			}

			for _, wantFile := range tt.want {
				if !gotMap[wantFile] {
					t.Errorf("DiscoverFiles() missing expected file %q", wantFile)
				}
			}
		})
	}
}

func TestValidateSkillName(t *testing.T) {
	tests := map[string]struct {
		name    string
		wantErr bool
	}{
		"valid simple name": {
			name:    "test-skill",
			wantErr: false,
		},
		"valid with underscores": {
			name:    "test_skill_123",
			wantErr: false,
		},
		"valid with numbers": {
			name:    "skill-v2",
			wantErr: false,
		},
		"valid with namespace": {
			name:    "namespace:skill-name",
			wantErr: false,
		},
		"valid with path": {
			name:    "namespace/category/skill",
			wantErr: false,
		},
		"empty name": {
			name:    "",
			wantErr: true,
		},
		"leading whitespace": {
			name:    " test-skill",
			wantErr: true,
		},
		"trailing whitespace": {
			name:    "test-skill ",
			wantErr: true,
		},
		"invalid characters (spaces)": {
			name:    "test skill",
			wantErr: true,
		},
		"invalid characters (special)": {
			name:    "test@skill",
			wantErr: true,
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			err := ValidateSkillName(tt.name)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSkillName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeContent(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
	}{
		"trim whitespace": {
			input: "  content  ",
			want:  "content",
		},
		"normalize line endings": {
			input: "line1\r\nline2\r\nline3",
			want:  "line1\nline2\nline3",
		},
		"trim and normalize": {
			input: "  line1\r\nline2  \r\n  ",
			want:  "line1\nline2",
		},
		"already normalized": {
			input: "content",
			want:  "content",
		},
		"empty string": {
			input: "",
			want:  "",
		},
		"only whitespace": {
			input: "   \n  \t  ",
			want:  "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := NormalizeContent(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Benchmark tests for critical performance paths

func BenchmarkDiscoverFiles(b *testing.B) {
	// Create realistic file structure with 100+ files
	tempDir := b.TempDir()

	// Create nested directory structure with multiple levels
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			dir := filepath.Join(tempDir, "level1", "level2", "level3")
			if err := os.MkdirAll(dir, 0o750); err != nil {
				b.Fatalf("failed to create directory: %v", err)
			}
			path := filepath.Join(dir, "skill-"+string(rune('a'+i))+"-"+string(rune('0'+j))+".md")
			// #nosec G306 - benchmark files don't need restrictive permissions
			if err := os.WriteFile(path, []byte("test content"), 0o600); err != nil {
				b.Fatalf("failed to create file: %v", err)
			}
		}
	}

	// Also create some files at root level
	for i := 0; i < 20; i++ {
		path := filepath.Join(tempDir, "root-skill-"+string(rune('a'+i))+".md")
		// #nosec G306 - benchmark files don't need restrictive permissions
		if err := os.WriteFile(path, []byte("test content"), 0o600); err != nil {
			b.Fatalf("failed to create file: %v", err)
		}
	}

	patterns := []string{"**/*.md"}

	b.ResetTimer()
	for b.Loop() {
		_, err := DiscoverFiles(tempDir, patterns)
		if err != nil {
			b.Fatalf("DiscoverFiles failed: %v", err)
		}
	}
}

func BenchmarkSplitFrontmatter(b *testing.B) {
	// Realistic skill file content
	content := []byte(`---
name: test-skill
description: A comprehensive test skill for benchmarking
platforms: [claude-code, cursor, codex]
author: Test Author
version: 1.0.0
tags:
  - testing
  - benchmark
  - performance
---
# Test Skill

This is a realistic skill file with multiple sections.

## Usage

Instructions for using this skill.

## Examples

Multiple examples of how to use this skill:
- Example 1
- Example 2
- Example 3

## Notes

Additional notes and considerations.
`)

	b.ResetTimer()
	for b.Loop() {
		_ = SplitFrontmatter(content)
	}
}

func BenchmarkParseYAMLFrontmatter(b *testing.B) {
	frontmatter := []byte(`name: test-skill
description: A comprehensive test skill
platforms:
  - claude-code
  - cursor
  - codex
author: Test Author
version: 1.0.0
tags:
  - testing
  - benchmark
  - performance
metadata:
  created: 2026-01-28
  updated: 2026-01-28
  category: development
`)

	b.ResetTimer()
	for b.Loop() {
		_, err := ParseYAMLFrontmatter(frontmatter)
		if err != nil {
			b.Fatalf("ParseYAMLFrontmatter failed: %v", err)
		}
	}
}
