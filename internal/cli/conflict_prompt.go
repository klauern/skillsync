package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/klauern/skillsync/internal/sync"
)

// ConflictResolver handles interactive conflict resolution with users.
type ConflictResolver struct {
	reader *bufio.Reader
}

// NewConflictResolver creates a new interactive conflict resolver.
func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{
		reader: bufio.NewReader(os.Stdin),
	}
}

// ResolveConflicts prompts the user to resolve each conflict interactively.
// Returns a map of skill names to their resolved content.
func (cr *ConflictResolver) ResolveConflicts(conflicts []*sync.Conflict) (map[string]string, error) {
	resolved := make(map[string]string)

	fmt.Printf("\n=== Conflict Resolution ===\n")
	fmt.Printf("Found %d conflict(s) that require resolution.\n\n", len(conflicts))

	merger := sync.NewMerger()

	for i, conflict := range conflicts {
		fmt.Printf("--- Conflict %d of %d: %s ---\n", i+1, len(conflicts), conflict.SkillName)
		fmt.Printf("Type: %s\n", conflict.Type)
		fmt.Printf("Changes: %s\n\n", conflict.DiffSummary())

		// Show diff preview
		cr.showDiffPreview(conflict)

		// Prompt for resolution
		choice, err := cr.promptResolution(conflict)
		if err != nil {
			return nil, fmt.Errorf("failed to get resolution for %s: %w", conflict.SkillName, err)
		}

		// Apply resolution
		resolvedContent := merger.ResolveWithChoice(conflict, choice)
		resolved[conflict.SkillName] = resolvedContent
		conflict.Resolution = choice
		conflict.ResolvedContent = resolvedContent

		fmt.Printf("✓ Resolved %s with: %s\n\n", conflict.SkillName, choice)
	}

	return resolved, nil
}

// showDiffPreview displays a preview of the differences.
func (cr *ConflictResolver) showDiffPreview(conflict *sync.Conflict) {
	fmt.Println("Preview of changes:")
	fmt.Println(strings.Repeat("-", 50))

	maxLines := 10 // Limit preview length
	shown := 0

	for _, hunk := range conflict.Hunks {
		if shown >= maxLines {
			fmt.Printf("... (%d more hunks not shown)\n", len(conflict.Hunks)-1)
			break
		}

		fmt.Printf("@@ -%d,%d +%d,%d @@\n",
			hunk.SourceStart, hunk.SourceCount,
			hunk.TargetStart, hunk.TargetCount)

		for _, line := range hunk.Lines {
			if shown >= maxLines {
				fmt.Println("... (truncated)")
				break
			}
			fmt.Println(line.String())
			shown++
		}
	}

	fmt.Println(strings.Repeat("-", 50))
}

// promptResolution asks the user to choose how to resolve a conflict.
func (cr *ConflictResolver) promptResolution(conflict *sync.Conflict) (sync.ResolutionChoice, error) {
	fmt.Println("\nHow would you like to resolve this conflict?")
	fmt.Println("  1. Use source version (overwrite target)")
	fmt.Println("  2. Keep target version (discard source changes)")
	fmt.Println("  3. Attempt automatic merge (may have conflict markers)")
	fmt.Println("  4. Skip this skill")
	fmt.Println("  5. Show full source content")
	fmt.Println("  6. Show full target content")
	fmt.Print("\nEnter choice [1-6]: ")

	for {
		response, err := cr.reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}

		response = strings.TrimSpace(response)
		choice, err := strconv.Atoi(response)
		if err != nil || choice < 1 || choice > 6 {
			fmt.Print("Invalid choice. Enter 1-6: ")
			continue
		}

		switch choice {
		case 1:
			return sync.ResolutionUseSource, nil
		case 2:
			return sync.ResolutionUseTarget, nil
		case 3:
			return sync.ResolutionMerge, nil
		case 4:
			return sync.ResolutionSkip, nil
		case 5:
			cr.showFullContent("SOURCE", conflict.Source.Content)
			fmt.Print("\nEnter choice [1-6]: ")
		case 6:
			cr.showFullContent("TARGET", conflict.Target.Content)
			fmt.Print("\nEnter choice [1-6]: ")
		}
	}
}

// showFullContent displays the full content of a version.
func (cr *ConflictResolver) showFullContent(label, content string) {
	fmt.Printf("\n=== %s CONTENT ===\n", label)
	fmt.Println(strings.Repeat("-", 50))

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		fmt.Printf("%4d | %s\n", i+1, line)
	}

	fmt.Println(strings.Repeat("-", 50))
}

// PromptForConflictMode asks the user how they want to handle conflicts.
func (cr *ConflictResolver) PromptForConflictMode(conflictCount int) (ConflictMode, error) {
	fmt.Printf("\n%d skill(s) have conflicts with existing target files.\n", conflictCount)
	fmt.Println("\nHow would you like to handle these conflicts?")
	fmt.Println("  1. Resolve each conflict interactively")
	fmt.Println("  2. Use source for all (overwrite)")
	fmt.Println("  3. Keep target for all (skip changes)")
	fmt.Println("  4. Auto-merge all (may leave conflict markers)")
	fmt.Println("  5. Abort sync")
	fmt.Print("\nEnter choice [1-5]: ")

	response, err := cr.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(response)
	choice, err := strconv.Atoi(response)
	if err != nil || choice < 1 || choice > 5 {
		return "", fmt.Errorf("invalid choice: %s", response)
	}

	switch choice {
	case 1:
		return ConflictModeInteractive, nil
	case 2:
		return ConflictModeUseSource, nil
	case 3:
		return ConflictModeUseTarget, nil
	case 4:
		return ConflictModeAutoMerge, nil
	case 5:
		return ConflictModeAbort, nil
	default:
		return ConflictModeAbort, nil
	}
}

// ConflictMode defines how to handle conflicts during sync.
type ConflictMode string

const (
	// ConflictModeInteractive prompts for each conflict.
	ConflictModeInteractive ConflictMode = "interactive"

	// ConflictModeUseSource uses source version for all conflicts.
	ConflictModeUseSource ConflictMode = "use-source"

	// ConflictModeUseTarget keeps target version for all conflicts.
	ConflictModeUseTarget ConflictMode = "use-target"

	// ConflictModeAutoMerge attempts automatic merge for all.
	ConflictModeAutoMerge ConflictMode = "auto-merge"

	// ConflictModeAbort aborts the sync operation.
	ConflictModeAbort ConflictMode = "abort"
)

// ResolveAllWithMode resolves all conflicts using the specified mode.
func (cr *ConflictResolver) ResolveAllWithMode(conflicts []*sync.Conflict, mode ConflictMode) (map[string]string, error) {
	resolved := make(map[string]string)
	merger := sync.NewMerger()

	var choice sync.ResolutionChoice
	switch mode {
	case ConflictModeUseSource:
		choice = sync.ResolutionUseSource
	case ConflictModeUseTarget:
		choice = sync.ResolutionUseTarget
	case ConflictModeAutoMerge:
		choice = sync.ResolutionMerge
	default:
		return nil, fmt.Errorf("invalid conflict mode: %s", mode)
	}

	for _, conflict := range conflicts {
		resolvedContent := merger.ResolveWithChoice(conflict, choice)
		resolved[conflict.SkillName] = resolvedContent
		conflict.Resolution = choice
		conflict.ResolvedContent = resolvedContent
	}

	return resolved, nil
}

// DisplayConflictSummary shows a summary of all conflicts.
func (cr *ConflictResolver) DisplayConflictSummary(conflicts []*sync.Conflict) {
	fmt.Println("\n=== Conflict Summary ===")
	fmt.Printf("%-30s %-15s %-20s\n", "SKILL", "TYPE", "CHANGES")
	fmt.Printf("%-30s %-15s %-20s\n", "-----", "----", "-------")

	for _, conflict := range conflicts {
		name := conflict.SkillName
		if len(name) > 30 {
			name = name[:27] + "..."
		}
		fmt.Printf("%-30s %-15s %-20s\n", name, conflict.Type, conflict.DiffSummary())
	}
	fmt.Println()
}
