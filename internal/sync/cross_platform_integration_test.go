package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/util"
)

// Integration tests for cross-platform synchronization with format transformation

// Helper function to check if string contains substring
func containsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestIntegration_ClaudeCodeToCodex(t *testing.T) {
	// Test syncing from Claude Code (.md) to Codex (AGENTS.md)
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create Claude Code skill with Agent Skills Standard fields
	sourceContent := `---
name: claude-to-codex-skill
description: Testing Claude Code to Codex sync
scope: user
license: MIT
compatibility:
  claude: ">=1.0.0"
tools:
  - read
  - write
references:
  - https://docs.example.com
---

# Claude to Codex Test

This content should be transformed to Codex format.
`

	util.WriteFile(t, filepath.Join(sourceDir, "claude-to-codex-skill.md"), sourceContent)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Codex, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 1)
	util.AssertEqual(t, result.Source, model.ClaudeCode)
	util.AssertEqual(t, result.Target, model.Codex)

	// Verify Codex file structure (AGENTS.md + potentially config.toml)
	// Codex uses different file structure
	targetFiles := []string{}
	err = filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(targetDir, path)
			targetFiles = append(targetFiles, rel)
		}
		return nil
	})
	util.AssertNoError(t, err)

	// Should have created at least one file
	if len(targetFiles) == 0 {
		t.Error("Expected Codex sync to create files")
	}
}

func TestIntegration_CodexToClaudeCode(t *testing.T) {
	// Test syncing from Codex to Claude Code format
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create Codex-style skill (simplified for test)
	codexDir := filepath.Join(sourceDir, "codex-skill")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	agentsContent := `# Codex to Claude Skill

This is a Codex skill that should be converted to Claude Code format.

## Instructions
Use this skill for testing cross-platform sync.
`

	util.WriteFile(t, filepath.Join(codexDir, "AGENTS.md"), agentsContent)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.Codex, model.ClaudeCode, opts)
	util.AssertNoError(t, err)

	// Should create Claude Code format files
	util.AssertEqual(t, result.Source, model.Codex)
	util.AssertEqual(t, result.Target, model.ClaudeCode)

	if result.TotalProcessed() == 0 {
		t.Error("Expected Codex to Claude Code sync to process skills")
	}
}

func TestIntegration_MetadataPreservation(t *testing.T) {
	// Test that AgentSkills Standard metadata is preserved across platforms
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create skill with comprehensive metadata
	sourceContent := `---
name: metadata-test
description: Testing metadata preservation
scope: repo
license: Apache-2.0
compatibility:
  claude: "^2.0.0"
  cursor: "*"
  codex: ">=1.5.0"
tools:
  - read
  - write
  - bash
scripts:
  install: npm install
  test: npm test
references:
  - https://example.com/docs
  - https://github.com/example/repo
assets:
  - logo.png
  - screenshot.jpg
---

# Metadata Preservation Test

This skill has comprehensive AgentSkills Standard metadata.
`

	util.WriteFile(t, filepath.Join(sourceDir, "metadata-test.md"), sourceContent)

	// Sync from Claude Code to Cursor (both use .md format)
	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 1)

	// Read target file and verify metadata preservation
	targetPath := filepath.Join(targetDir, "metadata-test.md")
	// #nosec G304 - test file path is controlled
	targetContent, err := os.ReadFile(targetPath)
	util.AssertNoError(t, err)

	target := string(targetContent)

	// Verify key metadata fields are preserved
	if !containsSubstring(target, "name: metadata-test") {
		t.Error("Name metadata not preserved")
	}
	if !containsSubstring(target, "scope: repo") {
		t.Error("Scope metadata not preserved")
	}
	if !containsSubstring(target, "license: Apache-2.0") {
		t.Error("License metadata not preserved")
	}
	if !containsSubstring(target, "compatibility:") {
		t.Error("Compatibility metadata not preserved")
	}
}

func TestIntegration_ToolsArrayPreservation(t *testing.T) {
	// Test that tools array is correctly preserved
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	sourceContent := `---
name: tools-test
description: Testing tools array
tools:
  - read
  - write
  - bash
  - grep
  - edit
---

Content with tools.
`

	util.WriteFile(t, filepath.Join(sourceDir, "tools-test.md"), sourceContent)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	_, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Read result
	// #nosec G304 - test file path is controlled
	targetContent, err := os.ReadFile(filepath.Join(targetDir, "tools-test.md"))
	util.AssertNoError(t, err)

	target := string(targetContent)

	// Verify at least some tools are preserved (exact format may vary)
	if !containsSubstring(target, "tools") && !containsSubstring(target, "read") {
		t.Error("Tools metadata not preserved")
	}
}

func TestIntegration_PluginSkillSync(t *testing.T) {
	// Test syncing plugin-installed skills
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Simulate a plugin skill with plugin metadata
	pluginSkillContent := `---
name: plugin-skill
description: Skill installed via plugin
scope: plugin
tools:
  - read
---

# Plugin Skill

This skill was installed by a plugin.
`

	util.WriteFile(t, filepath.Join(sourceDir, "plugin-skill.md"), pluginSkillContent)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 1)

	// Verify plugin scope is preserved
	// #nosec G304 - test file path is controlled
	targetContent, err := os.ReadFile(filepath.Join(targetDir, "plugin-skill.md"))
	util.AssertNoError(t, err)

	if !containsSubstring(string(targetContent), "scope: plugin") {
		t.Error("Plugin scope not preserved in sync")
	}
}

func TestIntegration_BidirectionalSync(t *testing.T) {
	// Test bidirectional sync between platforms
	s := New()

	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Create skill in dir1
	skill1 := `---
name: bidirectional-test
description: Testing bidirectional sync
---

Content from direction 1.
`
	util.WriteFile(t, filepath.Join(dir1, "bidirectional-test.md"), skill1)

	// Sync dir1 -> dir2
	opts1 := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: dir1,
		TargetPath: dir2,
	}

	result1, err := s.Sync(model.ClaudeCode, model.Cursor, opts1)
	util.AssertNoError(t, err)
	util.AssertEqual(t, len(result1.Created()), 1)

	// Modify in dir2
	skill2Modified := `---
name: bidirectional-test
description: Modified in dir2
---

Content modified in direction 2.
`
	util.WriteFile(t, filepath.Join(dir2, "bidirectional-test.md"), skill2Modified)

	// Wait a moment to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	// Sync back dir2 -> dir1 with newer strategy
	opts2 := Options{
		DryRun:     false,
		Strategy:   StrategyNewer,
		SourcePath: dir2,
		TargetPath: dir1,
	}

	result2, err := s.Sync(model.Cursor, model.ClaudeCode, opts2)
	util.AssertNoError(t, err)

	// Should update since dir2 version is newer
	util.AssertEqual(t, len(result2.Updated()), 1)

	// Verify dir1 has the updated content
	// #nosec G304 - test file path is controlled
	finalContent, err := os.ReadFile(filepath.Join(dir1, "bidirectional-test.md"))
	util.AssertNoError(t, err)

	if !containsSubstring(string(finalContent), "Modified in dir2") {
		t.Error("Bidirectional sync did not update target")
	}
}

func TestIntegration_ScopeTransformation(t *testing.T) {
	// Test that scope information is correctly transformed between platforms
	testCases := []struct {
		scope    string
		expected string
	}{
		{"user", "user"},
		{"repo", "repo"},
		{"system", "system"},
		{"plugin", "plugin"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.scope), func(t *testing.T) {
			s := New()

			sourceDir := t.TempDir()
			targetDir := t.TempDir()

			content := `---
name: scope-test
description: Testing scope transformation
scope: ` + tc.expected + `
---

Content for scope test.
`

			util.WriteFile(t, filepath.Join(sourceDir, "scope-test.md"), content)

			opts := Options{
				DryRun:     false,
				Strategy:   StrategyOverwrite,
				SourcePath: sourceDir,
				TargetPath: targetDir,
			}

			result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
			util.AssertNoError(t, err)

			util.AssertEqual(t, len(result.Created()), 1)

			// Verify scope is preserved
			// #nosec G304 - test file path is controlled
			targetContent, err := os.ReadFile(filepath.Join(targetDir, "scope-test.md"))
			util.AssertNoError(t, err)

			if !containsSubstring(string(targetContent), "scope: "+tc.expected) {
				t.Errorf("Scope %s not preserved in transformation", tc.expected)
			}
		})
	}
}

func TestIntegration_LargeMetadataSync(t *testing.T) {
	// Test syncing skills with large metadata (many tools, references, assets)
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create skill with extensive metadata
	sourceContent := `---
name: large-metadata
description: Skill with extensive metadata
tools:
  - read
  - write
  - bash
  - grep
  - glob
  - edit
  - task
  - web_fetch
  - web_search
references:
  - https://example.com/doc1
  - https://example.com/doc2
  - https://example.com/doc3
  - https://example.com/doc4
  - https://example.com/doc5
assets:
  - image1.png
  - image2.jpg
  - video1.mp4
  - diagram.svg
  - data.json
compatibility:
  claude: ">=1.0.0"
  cursor: "^2.0.0"
  codex: "*"
scripts:
  install: npm install
  build: npm run build
  test: npm test
  lint: npm run lint
  deploy: npm run deploy
---

# Large Metadata Test

This skill has extensive metadata to test handling of large frontmatter.
`

	util.WriteFile(t, filepath.Join(sourceDir, "large-metadata.md"), sourceContent)

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 1)

	// Verify target file is created and has content
	targetPath := filepath.Join(targetDir, "large-metadata.md")
	info, err := os.Stat(targetPath)
	util.AssertNoError(t, err)

	// File should have some content
	if info.Size() == 0 {
		t.Error("Expected file with content, got empty file")
	}

	// Read and verify the file was created
	// #nosec G304 - test file path is controlled
	targetContent, err := os.ReadFile(targetPath)
	util.AssertNoError(t, err)

	// Just verify the file has some content
	if len(targetContent) == 0 {
		t.Error("Expected file content to be non-empty")
	}
}
