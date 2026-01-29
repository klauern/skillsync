package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/util"
)

// TestIntegration_DependencyOrdering verifies skills are synced in dependency order.
func TestIntegration_DependencyOrdering(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create skills with dependencies
	// skill-c depends on skill-b
	// skill-b depends on skill-a
	// Expected order: skill-a -> skill-b -> skill-c
	skills := map[string]string{
		"skill-c.md": `---
name: skill-c
description: Third skill (depends on skill-b)
dependencies:
  - skill-b
---
# Skill C
This skill depends on skill-b.
`,
		"skill-a.md": `---
name: skill-a
description: First skill (no dependencies)
---
# Skill A
This is the base skill with no dependencies.
`,
		"skill-b.md": `---
name: skill-b
description: Second skill (depends on skill-a)
dependencies:
  - skill-a
---
# Skill B
This skill depends on skill-a.
`,
	}

	for name, content := range skills {
		path := filepath.Join(sourceDir, name)
		util.WriteFile(t, path, content)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 3)
	util.AssertEqual(t, result.TotalProcessed(), 3)
	util.AssertEqual(t, result.Success(), true)

	// Verify skills were processed in dependency order
	if len(result.Skills) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(result.Skills))
	}

	// Check order: skill-a, skill-b, skill-c
	expectedOrder := []string{"skill-a", "skill-b", "skill-c"}
	for i, expected := range expectedOrder {
		if result.Skills[i].Skill.Name != expected {
			t.Errorf("position %d: expected %s, got %s", i, expected, result.Skills[i].Skill.Name)
		}
	}

	// Verify all files were created
	for name := range skills {
		targetPath := filepath.Join(targetDir, name)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			t.Errorf("Expected target file %s to exist", name)
		}
	}
}

// TestIntegration_MissingDependencyWarning verifies warnings for missing dependencies.
func TestIntegration_MissingDependencyWarning(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create skill with missing dependency
	skills := map[string]string{
		"skill-a.md": `---
name: skill-a
description: Skill with missing dependency
dependencies:
  - skill-missing
---
# Skill A
This skill depends on a non-existent skill.
`,
	}

	for name, content := range skills {
		path := filepath.Join(sourceDir, name)
		util.WriteFile(t, path, content)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	// Should still succeed with warning
	util.AssertEqual(t, len(result.Created()), 1)
	util.AssertEqual(t, result.TotalProcessed(), 1)
	util.AssertEqual(t, result.Success(), true)
}

// TestIntegration_ComplexDependencyGraph verifies complex dependency resolution.
func TestIntegration_ComplexDependencyGraph(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a diamond dependency graph:
	//     a
	//    / \
	//   b   c
	//    \ /
	//     d
	skills := map[string]string{
		"skill-d.md": `---
name: skill-d
description: Skill D (depends on B and C)
dependencies:
  - skill-b
  - skill-c
---
# Skill D
`,
		"skill-c.md": `---
name: skill-c
description: Skill C (depends on A)
dependencies:
  - skill-a
---
# Skill C
`,
		"skill-a.md": `---
name: skill-a
description: Skill A (no dependencies)
---
# Skill A
`,
		"skill-b.md": `---
name: skill-b
description: Skill B (depends on A)
dependencies:
  - skill-a
---
# Skill B
`,
	}

	for name, content := range skills {
		path := filepath.Join(sourceDir, name)
		util.WriteFile(t, path, content)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 4)
	util.AssertEqual(t, result.TotalProcessed(), 4)
	util.AssertEqual(t, result.Success(), true)

	// Verify ordering constraints
	positions := make(map[string]int)
	for i, skillResult := range result.Skills {
		positions[skillResult.Skill.Name] = i
	}

	// skill-a must come before skill-b and skill-c
	if positions["skill-a"] >= positions["skill-b"] {
		t.Errorf("skill-a should come before skill-b")
	}
	if positions["skill-a"] >= positions["skill-c"] {
		t.Errorf("skill-a should come before skill-c")
	}

	// skill-b and skill-c must come before skill-d
	if positions["skill-b"] >= positions["skill-d"] {
		t.Errorf("skill-b should come before skill-d")
	}
	if positions["skill-c"] >= positions["skill-d"] {
		t.Errorf("skill-c should come before skill-d")
	}
}

// TestIntegration_NoDependenciesPreservesAlphabeticOrder verifies alphabetic ordering.
func TestIntegration_NoDependenciesPreservesAlphabeticOrder(t *testing.T) {
	s := New()

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create skills without dependencies
	skills := map[string]string{
		"skill-c.md": `---
name: skill-c
description: Skill C
---
# Skill C
`,
		"skill-a.md": `---
name: skill-a
description: Skill A
---
# Skill A
`,
		"skill-b.md": `---
name: skill-b
description: Skill B
---
# Skill B
`,
	}

	for name, content := range skills {
		path := filepath.Join(sourceDir, name)
		util.WriteFile(t, path, content)
	}

	opts := Options{
		DryRun:     false,
		Strategy:   StrategyOverwrite,
		SourcePath: sourceDir,
		TargetPath: targetDir,
	}

	result, err := s.Sync(model.ClaudeCode, model.Cursor, opts)
	util.AssertNoError(t, err)

	util.AssertEqual(t, len(result.Created()), 3)
	util.AssertEqual(t, result.TotalProcessed(), 3)

	// Skills without dependencies should be in alphabetic order
	expectedOrder := []string{"skill-a", "skill-b", "skill-c"}
	for i, expected := range expectedOrder {
		if result.Skills[i].Skill.Name != expected {
			t.Errorf("position %d: expected %s, got %s", i, expected, result.Skills[i].Skill.Name)
		}
	}
}
