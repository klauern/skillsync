package sync

import (
	"fmt"
	"strings"

	"github.com/klauern/skillsync/internal/model"
)

// Action represents the action taken on a skill during sync.
type Action string

const (
	// ActionCreated indicates a new skill was created in the target.
	ActionCreated Action = "created"

	// ActionUpdated indicates an existing skill was updated.
	ActionUpdated Action = "updated"

	// ActionSkipped indicates a skill was skipped (already exists or excluded).
	ActionSkipped Action = "skipped"

	// ActionMerged indicates a skill was merged with existing content.
	ActionMerged Action = "merged"

	// ActionFailed indicates an error occurred processing the skill.
	ActionFailed Action = "failed"

	// ActionConflict indicates a conflict was detected that needs resolution.
	ActionConflict Action = "conflict"

	// ActionDeleted indicates a skill was deleted from the target.
	ActionDeleted Action = "deleted"
)

// SkillResult represents the outcome of syncing a single skill.
type SkillResult struct {
	// Skill is the skill that was processed.
	Skill model.Skill

	// Action is the action that was taken.
	Action Action

	// TargetPath is the path where the skill was written (if applicable).
	TargetPath string

	// Error contains any error that occurred during processing.
	Error error

	// Message provides additional context about the action.
	Message string

	// Conflict holds conflict details when Action is ActionConflict.
	Conflict *Conflict
}

// Success returns true if the skill was successfully processed.
func (sr *SkillResult) Success() bool {
	return sr.Action != ActionFailed
}

// Result contains the complete outcome of a sync operation.
type Result struct {
	// Source is the source platform.
	Source model.Platform

	// Target is the target platform.
	Target model.Platform

	// Strategy is the sync strategy used.
	Strategy Strategy

	// Skills contains the result for each processed skill.
	Skills []SkillResult

	// DryRun indicates if this was a dry run (no changes made).
	DryRun bool
}

// Created returns skills that were created.
func (r *Result) Created() []SkillResult {
	return r.filterByAction(ActionCreated)
}

// Updated returns skills that were updated.
func (r *Result) Updated() []SkillResult {
	return r.filterByAction(ActionUpdated)
}

// Skipped returns skills that were skipped.
func (r *Result) Skipped() []SkillResult {
	return r.filterByAction(ActionSkipped)
}

// Merged returns skills that were merged.
func (r *Result) Merged() []SkillResult {
	return r.filterByAction(ActionMerged)
}

// Failed returns skills that failed to sync.
func (r *Result) Failed() []SkillResult {
	return r.filterByAction(ActionFailed)
}

// Conflicts returns skills that have unresolved conflicts.
func (r *Result) Conflicts() []SkillResult {
	return r.filterByAction(ActionConflict)
}

// Deleted returns skills that were deleted.
func (r *Result) Deleted() []SkillResult {
	return r.filterByAction(ActionDeleted)
}

// HasConflicts returns true if there are unresolved conflicts.
func (r *Result) HasConflicts() bool {
	return len(r.Conflicts()) > 0
}

// filterByAction returns skills with the given action.
func (r *Result) filterByAction(action Action) []SkillResult {
	var filtered []SkillResult
	for _, sr := range r.Skills {
		if sr.Action == action {
			filtered = append(filtered, sr)
		}
	}
	return filtered
}

// Success returns true if all skills were successfully processed.
func (r *Result) Success() bool {
	return len(r.Failed()) == 0
}

// TotalProcessed returns the total number of skills processed.
func (r *Result) TotalProcessed() int {
	return len(r.Skills)
}

// TotalChanged returns the number of skills that were created, updated, merged, or deleted.
func (r *Result) TotalChanged() int {
	return len(r.Created()) + len(r.Updated()) + len(r.Merged()) + len(r.Deleted())
}

// Summary returns a human-readable summary of the sync result.
func (r *Result) Summary() string {
	var sb strings.Builder

	if r.DryRun {
		sb.WriteString("Dry run - no changes made\n")
	}

	sb.WriteString(fmt.Sprintf("Synced %s -> %s using %s strategy\n",
		r.Source, r.Target, r.Strategy))

	sb.WriteString(fmt.Sprintf("  Created:   %d\n", len(r.Created())))
	sb.WriteString(fmt.Sprintf("  Updated:   %d\n", len(r.Updated())))
	sb.WriteString(fmt.Sprintf("  Merged:    %d\n", len(r.Merged())))
	sb.WriteString(fmt.Sprintf("  Deleted:   %d\n", len(r.Deleted())))
	sb.WriteString(fmt.Sprintf("  Skipped:   %d\n", len(r.Skipped())))
	sb.WriteString(fmt.Sprintf("  Conflicts: %d\n", len(r.Conflicts())))
	sb.WriteString(fmt.Sprintf("  Failed:    %d\n", len(r.Failed())))

	if r.HasConflicts() {
		sb.WriteString("\nConflicts requiring resolution:\n")
		for _, c := range r.Conflicts() {
			sb.WriteString(fmt.Sprintf("  - %s", c.Skill.Name))
			if c.Conflict != nil {
				sb.WriteString(fmt.Sprintf(": %s", c.Conflict.Summary()))
			}
			sb.WriteString("\n")
		}
	}

	if !r.Success() {
		sb.WriteString("\nErrors:\n")
		for _, f := range r.Failed() {
			sb.WriteString(fmt.Sprintf("  - %s: %v\n", f.Skill.Name, f.Error))
		}
	}

	return sb.String()
}
