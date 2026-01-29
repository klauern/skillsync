package dependency

import (
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestResolve_EmptySkills(t *testing.T) {
	result := Resolve([]model.Skill{})
	if result.HasErrors() {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
	if result.HasWarnings() {
		t.Errorf("expected no warnings, got: %v", result.Warnings)
	}
	if len(result.Ordered) != 0 {
		t.Errorf("expected empty ordered list, got %d skills", len(result.Ordered))
	}
}

func TestResolve_SingleSkill(t *testing.T) {
	skills := []model.Skill{
		{Name: "skill-a", Dependencies: []string{}},
	}

	result := Resolve(skills)
	if result.HasErrors() {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
	if len(result.Ordered) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result.Ordered))
	}
	if result.Ordered[0].Name != "skill-a" {
		t.Errorf("expected skill-a, got %s", result.Ordered[0].Name)
	}
}

func TestResolve_NoDependencies(t *testing.T) {
	skills := []model.Skill{
		{Name: "skill-a"},
		{Name: "skill-b"},
		{Name: "skill-c"},
	}

	result := Resolve(skills)
	if result.HasErrors() {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
	if len(result.Ordered) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(result.Ordered))
	}
	// Should be alphabetically sorted when no dependencies
	expected := []string{"skill-a", "skill-b", "skill-c"}
	for i, name := range expected {
		if result.Ordered[i].Name != name {
			t.Errorf("expected %s at position %d, got %s", name, i, result.Ordered[i].Name)
		}
	}
}

func TestResolve_SimpleDependency(t *testing.T) {
	skills := []model.Skill{
		{Name: "skill-b", Dependencies: []string{"skill-a"}},
		{Name: "skill-a", Dependencies: []string{}},
	}

	result := Resolve(skills)
	if result.HasErrors() {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
	if len(result.Ordered) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(result.Ordered))
	}
	// skill-a should come before skill-b
	if result.Ordered[0].Name != "skill-a" {
		t.Errorf("expected skill-a first, got %s", result.Ordered[0].Name)
	}
	if result.Ordered[1].Name != "skill-b" {
		t.Errorf("expected skill-b second, got %s", result.Ordered[1].Name)
	}
}

func TestResolve_ChainDependency(t *testing.T) {
	skills := []model.Skill{
		{Name: "skill-c", Dependencies: []string{"skill-b"}},
		{Name: "skill-a", Dependencies: []string{}},
		{Name: "skill-b", Dependencies: []string{"skill-a"}},
	}

	result := Resolve(skills)
	if result.HasErrors() {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
	if len(result.Ordered) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(result.Ordered))
	}
	// Expected order: skill-a -> skill-b -> skill-c
	expected := []string{"skill-a", "skill-b", "skill-c"}
	for i, name := range expected {
		if result.Ordered[i].Name != name {
			t.Errorf("expected %s at position %d, got %s", name, i, result.Ordered[i].Name)
		}
	}
}

func TestResolve_MultipleIndependentChains(t *testing.T) {
	skills := []model.Skill{
		{Name: "skill-b", Dependencies: []string{"skill-a"}},
		{Name: "skill-d", Dependencies: []string{"skill-c"}},
		{Name: "skill-a", Dependencies: []string{}},
		{Name: "skill-c", Dependencies: []string{}},
	}

	result := Resolve(skills)
	if result.HasErrors() {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
	if len(result.Ordered) != 4 {
		t.Fatalf("expected 4 skills, got %d", len(result.Ordered))
	}

	// Find positions
	positions := make(map[string]int)
	for i, skill := range result.Ordered {
		positions[skill.Name] = i
	}

	// skill-a must come before skill-b
	if positions["skill-a"] >= positions["skill-b"] {
		t.Errorf("skill-a should come before skill-b")
	}
	// skill-c must come before skill-d
	if positions["skill-c"] >= positions["skill-d"] {
		t.Errorf("skill-c should come before skill-d")
	}
}

func TestResolve_CircularDependency(t *testing.T) {
	skills := []model.Skill{
		{Name: "skill-a", Dependencies: []string{"skill-b"}},
		{Name: "skill-b", Dependencies: []string{"skill-a"}},
	}

	result := Resolve(skills)
	if !result.HasErrors() {
		t.Fatalf("expected errors for circular dependency")
	}

	// Should have at least one circular error
	hasCircular := false
	for _, err := range result.Errors {
		if err.Type == "circular" {
			hasCircular = true
			break
		}
	}
	if !hasCircular {
		t.Errorf("expected circular dependency error, got: %v", result.Errors)
	}
}

func TestResolve_SelfDependency(t *testing.T) {
	skills := []model.Skill{
		{Name: "skill-a", Dependencies: []string{"skill-a"}},
	}

	result := Resolve(skills)
	if !result.HasErrors() {
		t.Fatalf("expected errors for self-dependency")
	}

	hasCircular := false
	for _, err := range result.Errors {
		if err.Type == "circular" {
			hasCircular = true
			break
		}
	}
	if !hasCircular {
		t.Errorf("expected circular dependency error for self-dependency")
	}
}

func TestResolve_MissingDependency(t *testing.T) {
	skills := []model.Skill{
		{Name: "skill-a", Dependencies: []string{"skill-missing"}},
	}

	result := Resolve(skills)
	if !result.HasWarnings() {
		t.Fatalf("expected warnings for missing dependency")
	}

	hasMissing := false
	for _, warn := range result.Warnings {
		if warn.Type == "missing" {
			hasMissing = true
			break
		}
	}
	if !hasMissing {
		t.Errorf("expected missing dependency warning")
	}
}

func TestResolve_ComplexGraph(t *testing.T) {
	// Create a complex dependency graph:
	//    a
	//   / \
	//  b   c
	//   \ /
	//    d
	skills := []model.Skill{
		{Name: "skill-d", Dependencies: []string{"skill-b", "skill-c"}},
		{Name: "skill-c", Dependencies: []string{"skill-a"}},
		{Name: "skill-a", Dependencies: []string{}},
		{Name: "skill-b", Dependencies: []string{"skill-a"}},
	}

	result := Resolve(skills)
	if result.HasErrors() {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
	if len(result.Ordered) != 4 {
		t.Fatalf("expected 4 skills, got %d", len(result.Ordered))
	}

	// Get positions
	positions := make(map[string]int)
	for i, skill := range result.Ordered {
		positions[skill.Name] = i
	}

	// Verify ordering constraints
	if positions["skill-a"] >= positions["skill-b"] {
		t.Errorf("skill-a should come before skill-b")
	}
	if positions["skill-a"] >= positions["skill-c"] {
		t.Errorf("skill-a should come before skill-c")
	}
	if positions["skill-b"] >= positions["skill-d"] {
		t.Errorf("skill-b should come before skill-d")
	}
	if positions["skill-c"] >= positions["skill-d"] {
		t.Errorf("skill-c should come before skill-d")
	}
}

func TestResolve_LongCircularChain(t *testing.T) {
	// Create: a -> b -> c -> a (circular)
	skills := []model.Skill{
		{Name: "skill-a", Dependencies: []string{"skill-b"}},
		{Name: "skill-b", Dependencies: []string{"skill-c"}},
		{Name: "skill-c", Dependencies: []string{"skill-a"}},
	}

	result := Resolve(skills)
	if !result.HasErrors() {
		t.Fatalf("expected errors for circular dependency")
	}

	hasCircular := false
	for _, err := range result.Errors {
		if err.Type == "circular" {
			hasCircular = true
			break
		}
	}
	if !hasCircular {
		t.Errorf("expected circular dependency error")
	}
}

func TestValidateGraph(t *testing.T) {
	t.Run("valid graph", func(t *testing.T) {
		skills := []model.Skill{
			{Name: "skill-a"},
			{Name: "skill-b", Dependencies: []string{"skill-a"}},
		}

		errors := ValidateGraph(skills)
		if len(errors) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errors), errors)
		}
	})

	t.Run("circular dependency", func(t *testing.T) {
		skills := []model.Skill{
			{Name: "skill-a", Dependencies: []string{"skill-b"}},
			{Name: "skill-b", Dependencies: []string{"skill-a"}},
		}

		errors := ValidateGraph(skills)
		if len(errors) == 0 {
			t.Errorf("expected errors for circular dependency")
		}
	})

	t.Run("missing dependency", func(t *testing.T) {
		skills := []model.Skill{
			{Name: "skill-a", Dependencies: []string{"missing"}},
		}

		errors := ValidateGraph(skills)
		if len(errors) == 0 {
			t.Errorf("expected errors for missing dependency")
		}
	})
}

func TestDetectCycles(t *testing.T) {
	t.Run("no cycles", func(t *testing.T) {
		graph := map[string][]string{
			"a": {"b"},
			"b": {"c"},
			"c": {},
		}

		cycles := detectCycles(graph)
		if len(cycles) != 0 {
			t.Errorf("expected no cycles, got: %v", cycles)
		}
	})

	t.Run("simple cycle", func(t *testing.T) {
		graph := map[string][]string{
			"a": {"b"},
			"b": {"a"},
		}

		cycles := detectCycles(graph)
		if len(cycles) == 0 {
			t.Errorf("expected cycle, got none")
		}
	})

	t.Run("self cycle", func(t *testing.T) {
		graph := map[string][]string{
			"a": {"a"},
		}

		cycles := detectCycles(graph)
		if len(cycles) == 0 {
			t.Errorf("expected cycle, got none")
		}
	})
}
