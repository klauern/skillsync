// Package dependency provides dependency resolution and ordering for skills.
package dependency

import (
	"fmt"
	"sort"
	"strings"

	"github.com/klauern/skillsync/internal/model"
)

// ValidationError represents a dependency validation error.
type ValidationError struct {
	Type    string   // "circular", "missing", "invalid"
	Skills  []string // Skills involved in the error
	Message string   // Human-readable error message
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Result contains the outcome of dependency resolution.
type Result struct {
	Ordered  []model.Skill      // Skills in dependency-resolved order
	Warnings []ValidationError  // Non-fatal issues
	Errors   []ValidationError  // Fatal issues
}

// HasErrors returns true if there are any errors.
func (r *Result) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if there are any warnings.
func (r *Result) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// Resolve validates dependencies and returns skills in topologically sorted order.
// Returns a Result with ordered skills if successful, or errors/warnings if issues found.
// Even with errors, returns original order as best-effort fallback.
func Resolve(skills []model.Skill) Result {
	result := Result{
		Ordered:  skills, // Default to input order
		Warnings: []ValidationError{},
		Errors:   []ValidationError{},
	}

	// If no skills, nothing to do
	if len(skills) == 0 {
		return result
	}

	// Build skill name index
	skillsByName := make(map[string]*model.Skill)
	for i := range skills {
		skillsByName[skills[i].Name] = &skills[i]
	}

	// Build dependency graph
	graph := make(map[string][]string)
	for _, skill := range skills {
		graph[skill.Name] = skill.Dependencies
	}

	// Validate dependencies exist
	for _, skill := range skills {
		for _, dep := range skill.Dependencies {
			if _, exists := skillsByName[dep]; !exists {
				result.Warnings = append(result.Warnings, ValidationError{
					Type:    "missing",
					Skills:  []string{skill.Name, dep},
					Message: fmt.Sprintf("skill %q depends on %q which is not found", skill.Name, dep),
				})
			}
		}
	}

	// Detect circular dependencies
	if cycles := detectCycles(graph); len(cycles) > 0 {
		for _, cycle := range cycles {
			result.Errors = append(result.Errors, ValidationError{
				Type:    "circular",
				Skills:  cycle,
				Message: fmt.Sprintf("circular dependency detected: %s", strings.Join(cycle, " -> ")),
			})
		}
		// Return with errors, keeping original order
		return result
	}

	// Perform topological sort
	ordered, err := topologicalSort(skills, graph)
	if err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Type:    "invalid",
			Skills:  []string{},
			Message: fmt.Sprintf("failed to order skills: %v", err),
		})
		return result
	}

	result.Ordered = ordered
	return result
}

// detectCycles detects circular dependencies in the graph.
// Returns a list of cycles, where each cycle is a list of skill names forming the cycle.
func detectCycles(graph map[string][]string) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []string{}

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, dep := range graph[node] {
			if !visited[dep] {
				if dfs(dep) {
					return true
				}
			} else if recStack[dep] {
				// Found cycle - extract it from path
				cycleStart := -1
				for i, n := range path {
					if n == dep {
						cycleStart = i
						break
					}
				}
				if cycleStart != -1 {
					cycle := make([]string, len(path)-cycleStart)
					copy(cycle, path[cycleStart:])
					cycle = append(cycle, dep) // Complete the cycle
					cycles = append(cycles, cycle)
				}
				return true
			}
		}

		// Backtrack
		path = path[:len(path)-1]
		recStack[node] = false
		return false
	}

	// Check each node
	for node := range graph {
		if !visited[node] {
			path = []string{}
			dfs(node)
		}
	}

	return cycles
}

// topologicalSort performs Kahn's algorithm for topological sorting.
func topologicalSort(skills []model.Skill, graph map[string][]string) ([]model.Skill, error) {
	// Build in-degree map (count incoming edges = number of dependencies)
	inDegree := make(map[string]int)
	for _, skill := range skills {
		// Initialize with the count of dependencies this skill has
		inDegree[skill.Name] = len(skill.Dependencies)
	}

	// Find all nodes with in-degree 0 (no dependencies)
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	// Sort queue for deterministic ordering
	sort.Strings(queue)

	// Build skill lookup
	skillsByName := make(map[string]model.Skill)
	for _, skill := range skills {
		skillsByName[skill.Name] = skill
	}

	// Process queue
	var result []model.Skill
	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]

		// Add to result
		if skill, exists := skillsByName[current]; exists {
			result = append(result, skill)
		}

		// Reduce in-degree for dependents
		for _, skill := range skills {
			for _, dep := range skill.Dependencies {
				if dep == current {
					inDegree[skill.Name]--
					if inDegree[skill.Name] == 0 {
						queue = append(queue, skill.Name)
						sort.Strings(queue) // Keep sorted for determinism
					}
				}
			}
		}
	}

	// If we didn't process all skills, there's a cycle (shouldn't happen if detectCycles passed)
	if len(result) != len(skills) {
		return skills, fmt.Errorf("topological sort failed: processed %d of %d skills", len(result), len(skills))
	}

	return result, nil
}

// ValidateGraph performs validation on a dependency graph without ordering.
// Returns any circular or missing dependency errors.
func ValidateGraph(skills []model.Skill) []ValidationError {
	result := Resolve(skills)
	var allErrors []ValidationError
	allErrors = append(allErrors, result.Errors...)
	allErrors = append(allErrors, result.Warnings...)
	return allErrors
}
