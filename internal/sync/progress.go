package sync

import (
	"fmt"

	"github.com/klauern/skillsync/internal/model"
)

// ProgressEvent represents a synchronization progress event.
type ProgressEvent struct {
	// Type of progress event.
	Type ProgressEventType

	// Current skill being processed.
	Skill *model.Skill

	// Current progress (0-100).
	PercentComplete int

	// Total number of skills to process.
	TotalSkills int

	// Number of skills processed so far.
	ProcessedSkills int

	// Action taken for current skill.
	Action Action

	// Message describing the event.
	Message string

	// Error if something went wrong.
	Error error

	// Conflict details if applicable.
	Conflict *Conflict
}

// ProgressEventType defines types of progress events.
type ProgressEventType string

const (
	// ProgressEventStart indicates sync started.
	ProgressEventStart ProgressEventType = "start"

	// ProgressEventSkillStart indicates a skill started processing.
	ProgressEventSkillStart ProgressEventType = "skill_start"

	// ProgressEventSkillComplete indicates a skill finished processing.
	ProgressEventSkillComplete ProgressEventType = "skill_complete"

	// ProgressEventComplete indicates sync completed.
	ProgressEventComplete ProgressEventType = "complete"

	// ProgressEventError indicates an error occurred.
	ProgressEventError ProgressEventType = "error"
)

// ProgressCallback is called during synchronization to report progress.
// If the callback returns an error, synchronization will be aborted.
type ProgressCallback func(event ProgressEvent) error

// emitProgress emits a progress event if a callback is configured.
// Returns an error if the callback fails, allowing cancellation.
func (s *Synchronizer) emitProgress(opts Options, event ProgressEvent) error {
	if opts.Progress == nil {
		return nil
	}
	return opts.Progress(event)
}

func progressPercent(processed, total int) int {
	if total <= 0 {
		return 0
	}
	return (processed * 100) / total
}

func progressStartMessage(total int, noun string) string {
	if noun == "" {
		noun = "skills"
	}
	return fmt.Sprintf("Starting sync of %d %s", total, noun)
}
