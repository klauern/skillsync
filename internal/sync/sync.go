package sync

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/codex"
	"github.com/klauern/skillsync/internal/parser/cursor"
	"github.com/klauern/skillsync/internal/validation"
)

// ProgressEvent represents a synchronization progress event.
type ProgressEvent struct {
	// Type of progress event
	Type ProgressEventType

	// Current skill being processed
	Skill *model.Skill

	// Current progress (0-100)
	PercentComplete int

	// Total number of skills to process
	TotalSkills int

	// Number of skills processed so far
	ProcessedSkills int

	// Action taken for current skill
	Action Action

	// Message describing the event
	Message string

	// Error if something went wrong
	Error error

	// Conflict details if applicable
	Conflict *Conflict
}

// ProgressEventType defines types of progress events.
type ProgressEventType string

const (
	// ProgressEventStart indicates sync started
	ProgressEventStart ProgressEventType = "start"

	// ProgressEventSkillStart indicates a skill started processing
	ProgressEventSkillStart ProgressEventType = "skill_start"

	// ProgressEventSkillComplete indicates a skill finished processing
	ProgressEventSkillComplete ProgressEventType = "skill_complete"

	// ProgressEventComplete indicates sync completed
	ProgressEventComplete ProgressEventType = "complete"

	// ProgressEventError indicates an error occurred
	ProgressEventError ProgressEventType = "error"
)

// ProgressCallback is called during synchronization to report progress.
// If the callback returns an error, synchronization will be aborted.
type ProgressCallback func(event ProgressEvent) error

// Options configures synchronization behavior.
type Options struct {
	// DryRun enables preview mode without making actual changes.
	DryRun bool

	// Strategy defines how to handle conflicts (default: overwrite).
	Strategy Strategy

	// SourcePath overrides the default source path.
	SourcePath string

	// TargetPath overrides the default target path.
	TargetPath string

	// TargetScope specifies the scope to write to (repo or user).
	// Defaults to user scope if not specified.
	TargetScope model.SkillScope

	// SkipValidation skips pre-sync validation.
	SkipValidation bool

	// Verbose enables detailed output.
	Verbose bool

	// Progress callback for real-time progress reporting.
	// Optional - if nil, no progress events are emitted.
	Progress ProgressCallback

	// Bidirectional enables two-way sync (both platforms can be source and target).
	// When true, syncs in both directions and reconciles conflicts.
	Bidirectional bool
}

// DefaultOptions returns the default sync options.
func DefaultOptions() Options {
	return Options{
		DryRun:   false,
		Strategy: StrategyOverwrite,
	}
}

// Syncer defines the interface for synchronization strategies.
type Syncer interface {
	// Sync performs synchronization between platforms.
	// When opts.DryRun is true, returns a preview of changes without modifying files.
	Sync(source, target model.Platform, opts Options) (*Result, error)
}

// Synchronizer implements the Syncer interface.
type Synchronizer struct {
	transformer      *Transformer
	conflictDetector *ConflictDetector
	merger           *Merger
}

// New creates a new Synchronizer.
func New() *Synchronizer {
	return &Synchronizer{
		transformer:      NewTransformer(),
		conflictDetector: NewConflictDetector(),
		merger:           NewMerger(),
	}
}

// emitProgress emits a progress event if a callback is configured.
// Returns an error if the callback fails, allowing cancellation.
func (s *Synchronizer) emitProgress(opts Options, event ProgressEvent) error {
	if opts.Progress == nil {
		return nil
	}
	return opts.Progress(event)
}

// Sync performs synchronization from source to target platform.
func (s *Synchronizer) Sync(source, target model.Platform, opts Options) (*Result, error) {
	logging.Debug("starting sync operation",
		logging.Platform(string(source)),
		logging.Operation("sync"),
		slog.String("target", string(target)),
		slog.String(logging.KeyStrategy, string(opts.Strategy)),
		slog.Bool("dry_run", opts.DryRun),
	)

	result := &Result{
		Source:   source,
		Target:   target,
		Strategy: opts.Strategy,
		DryRun:   opts.DryRun,
		Skills:   make([]SkillResult, 0),
	}

	// Set default strategy if not specified
	if result.Strategy == "" {
		result.Strategy = StrategyOverwrite
	}

	// Parse source skills
	sourceSkills, err := s.parseSkills(source, opts.SourcePath)
	if err != nil {
		logging.Error("failed to parse source skills",
			logging.Platform(string(source)),
			logging.Operation("sync"),
			logging.Err(err),
		)
		return result, fmt.Errorf("failed to parse source skills: %w", err)
	}

	logging.Debug("parsed source skills",
		logging.Platform(string(source)),
		logging.Count(len(sourceSkills)),
	)

	totalSkills := len(sourceSkills)

	// Emit start event
	if err := s.emitProgress(opts, ProgressEvent{
		Type:        ProgressEventStart,
		TotalSkills: totalSkills,
		Message:     fmt.Sprintf("Starting sync of %d skills", totalSkills),
	}); err != nil {
		return result, fmt.Errorf("progress callback failed: %w", err)
	}

	if totalSkills == 0 {
		logging.Debug("no skills to sync",
			logging.Platform(string(source)),
		)
		// Emit completion event for empty sync
		_ = s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventComplete,
			TotalSkills:     0,
			ProcessedSkills: 0,
			PercentComplete: 100,
			Message:         "No skills to sync",
		})
		return result, nil // Nothing to sync
	}

	// Get target path
	targetPath := opts.TargetPath
	if targetPath == "" {
		targetPath, err = validation.GetPlatformPath(target)
		if err != nil {
			return result, fmt.Errorf("failed to get target path: %w", err)
		}
	}

	// Parse existing target skills for conflict detection
	targetSkills, err := s.parseSkills(target, opts.TargetPath)
	if err != nil {
		logging.Debug("target skills not found, starting fresh",
			logging.Platform(string(target)),
			logging.Err(err),
		)
		// Target may not exist yet, which is okay
		targetSkills = []model.Skill{}
	} else {
		logging.Debug("parsed target skills",
			logging.Platform(string(target)),
			logging.Count(len(targetSkills)),
		)
	}

	// Build a map of existing target skills by name
	targetSkillMap := make(map[string]model.Skill)
	for _, skill := range targetSkills {
		targetSkillMap[skill.Name] = skill
	}

	// Ensure target directory exists (unless dry run)
	if !opts.DryRun {
		if err := os.MkdirAll(targetPath, 0o750); err != nil {
			logging.Error("failed to create target directory",
				logging.Path(targetPath),
				logging.Err(err),
			)
			return result, fmt.Errorf("failed to create target directory: %w", err)
		}
		logging.Debug("ensured target directory exists",
			logging.Path(targetPath),
		)
	}

	// Process each source skill
	for i, sourceSkill := range sourceSkills {
		// Emit skill start event
		if err := s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventSkillStart,
			Skill:           &sourceSkill,
			TotalSkills:     totalSkills,
			ProcessedSkills: i,
			PercentComplete: (i * 100) / totalSkills,
			Message:         fmt.Sprintf("Processing %s", sourceSkill.Name),
		}); err != nil {
			return result, fmt.Errorf("progress callback failed: %w", err)
		}

		skillResult := s.processSkill(sourceSkill, target, targetPath, targetSkillMap, opts)
		result.Skills = append(result.Skills, skillResult)

		// Emit skill complete event
		processedCount := i + 1
		if err := s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventSkillComplete,
			Skill:           &sourceSkill,
			Action:          skillResult.Action,
			TotalSkills:     totalSkills,
			ProcessedSkills: processedCount,
			PercentComplete: (processedCount * 100) / totalSkills,
			Message:         skillResult.Message,
			Error:           skillResult.Error,
			Conflict:        skillResult.Conflict,
		}); err != nil {
			return result, fmt.Errorf("progress callback failed: %w", err)
		}
	}

	logging.Debug("sync operation completed",
		logging.Platform(string(source)),
		slog.String("target", string(target)),
		logging.Count(len(result.Skills)),
	)

	// Emit completion event
	_ = s.emitProgress(opts, ProgressEvent{
		Type:            ProgressEventComplete,
		TotalSkills:     totalSkills,
		ProcessedSkills: totalSkills,
		PercentComplete: 100,
		Message:         fmt.Sprintf("Sync completed: %d skills processed", totalSkills),
	})

	return result, nil
}

// parseSkills parses skills from the given platform.
func (s *Synchronizer) parseSkills(platform model.Platform, basePath string) ([]model.Skill, error) {
	var p parser.Parser

	// If basePath is empty, get the default path which respects env var overrides
	if basePath == "" {
		defaultPath, err := validation.GetPlatformPath(platform)
		if err != nil {
			return nil, fmt.Errorf("failed to get platform path: %w", err)
		}
		basePath = defaultPath
	}

	switch platform {
	case model.ClaudeCode:
		p = claude.New(basePath)
	case model.Cursor:
		p = cursor.New(basePath)
	case model.Codex:
		p = codex.New(basePath)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	return p.Parse()
}

// processSkill handles syncing a single skill.
func (s *Synchronizer) processSkill(
	source model.Skill,
	targetPlatform model.Platform,
	targetPath string,
	existingSkills map[string]model.Skill,
	opts Options,
) SkillResult {
	logging.Debug("processing skill",
		logging.Skill(source.Name),
		logging.Platform(string(source.Platform)),
		slog.String("target", string(targetPlatform)),
	)

	result := SkillResult{
		Skill: source,
	}

	// Transform the skill for the target platform
	transformed, err := s.transformer.Transform(source, targetPlatform)
	if err != nil {
		logging.Warn("transformation failed",
			logging.Skill(source.Name),
			logging.Err(err),
		)
		result.Action = ActionFailed
		result.Error = fmt.Errorf("transformation failed: %w", err)
		return result
	}

	logging.Debug("skill transformed",
		logging.Skill(source.Name),
		logging.Path(transformed.Path),
	)

	// Determine target file path
	targetFilePath := filepath.Join(targetPath, transformed.Path)
	result.TargetPath = targetFilePath

	// Check if skill exists in target
	existingSkill, exists := existingSkills[source.Name]

	// Determine action based on strategy
	action, message, conflict := s.determineAction(source, existingSkill, exists, opts.Strategy)
	result.Action = action
	result.Message = message
	result.Conflict = conflict

	logging.Debug("action determined",
		logging.Skill(source.Name),
		slog.String("action", string(action)),
		slog.String("message", message),
		slog.Bool("has_conflict", conflict != nil),
	)

	// If skipping or conflict (needs external resolution), we're done
	if action == ActionSkipped || action == ActionConflict {
		return result
	}

	// Get content to write
	content := transformed.Content

	// Handle merge strategy
	if action == ActionMerged && exists {
		logging.Debug("merging content",
			logging.Skill(source.Name),
		)
		content = s.transformer.MergeContent(transformed.Content, existingSkill.Content, source.Name)
	}

	// Write the file (unless dry run)
	if !opts.DryRun {
		// #nosec G306 - skill files should be readable
		if err := os.WriteFile(targetFilePath, []byte(content), 0o644); err != nil {
			logging.Error("failed to write skill file",
				logging.Skill(source.Name),
				logging.Path(targetFilePath),
				logging.Err(err),
			)
			result.Action = ActionFailed
			result.Error = fmt.Errorf("failed to write file: %w", err)
			return result
		}
		logging.Debug("wrote skill file",
			logging.Skill(source.Name),
			logging.Path(targetFilePath),
		)
	}

	return result
}

// determineAction decides what action to take based on strategy.
func (s *Synchronizer) determineAction(
	source model.Skill,
	existing model.Skill,
	exists bool,
	strategy Strategy,
) (Action, string, *Conflict) {
	logging.Debug("determining action",
		logging.Skill(source.Name),
		slog.String(logging.KeyStrategy, string(strategy)),
		slog.Bool("exists", exists),
	)

	if !exists {
		return ActionCreated, "new skill", nil
	}

	switch strategy {
	case StrategyOverwrite:
		return ActionUpdated, "overwriting existing skill", nil

	case StrategySkip:
		return ActionSkipped, "skill already exists", nil

	case StrategyNewer:
		if source.ModifiedAt.After(existing.ModifiedAt) {
			logging.Debug("source is newer",
				logging.Skill(source.Name),
				slog.Time("source_modified", source.ModifiedAt),
				slog.Time("existing_modified", existing.ModifiedAt),
			)
			return ActionUpdated, fmt.Sprintf("source is newer (%s > %s)",
				source.ModifiedAt.Format(time.RFC3339),
				existing.ModifiedAt.Format(time.RFC3339)), nil
		}
		logging.Debug("target is newer or same age",
			logging.Skill(source.Name),
			slog.Time("source_modified", source.ModifiedAt),
			slog.Time("existing_modified", existing.ModifiedAt),
		)
		return ActionSkipped, fmt.Sprintf("target is newer or same age (%s >= %s)",
			existing.ModifiedAt.Format(time.RFC3339),
			source.ModifiedAt.Format(time.RFC3339)), nil

	case StrategyMerge:
		return ActionMerged, "merging with existing content", nil

	case StrategyThreeWay:
		// Check for actual conflicts using the detector
		conflict := s.conflictDetector.DetectConflict(source, existing)
		if conflict == nil {
			// No conflict, content is identical
			logging.Debug("no conflict detected, content identical",
				logging.Skill(source.Name),
			)
			return ActionSkipped, "content is identical", nil
		}
		// Attempt three-way merge
		logging.Debug("attempting three-way merge",
			logging.Skill(source.Name),
			slog.String("conflict_type", string(conflict.Type)),
		)
		mergeResult := s.merger.TwoWayMerge(source, existing)
		if mergeResult.Success {
			logging.Debug("three-way merge successful",
				logging.Skill(source.Name),
			)
			return ActionMerged, "three-way merge successful", nil
		}
		// Has conflicts that need resolution
		logging.Debug("conflict requires manual resolution",
			logging.Skill(source.Name),
			slog.String("conflict_type", string(conflict.Type)),
		)
		return ActionConflict, "conflict detected - needs resolution", conflict

	case StrategyInteractive:
		// Always check for conflicts with interactive strategy
		conflict := s.conflictDetector.DetectConflict(source, existing)
		if conflict == nil {
			return ActionUpdated, "updating (no conflicts)", nil
		}
		logging.Debug("conflict detected for interactive resolution",
			logging.Skill(source.Name),
			slog.String("conflict_type", string(conflict.Type)),
		)
		return ActionConflict, "conflict detected - awaiting resolution", conflict

	case StrategySmart:
		// Smart strategy: intelligently choose approach based on content and timestamps
		logging.Debug("smart strategy evaluation",
			logging.Skill(source.Name),
			slog.Time("source_modified", source.ModifiedAt),
			slog.Time("existing_modified", existing.ModifiedAt),
		)

		// First check for conflicts
		conflict := s.conflictDetector.DetectConflict(source, existing)
		if conflict == nil {
			// No conflict, content is identical
			logging.Debug("smart: no conflict detected, content identical",
				logging.Skill(source.Name),
			)
			return ActionSkipped, "content is identical", nil
		}

		// Check timestamp difference (more than 1 hour is significant)
		timeDiff := source.ModifiedAt.Sub(existing.ModifiedAt)
		if timeDiff.Abs() > time.Hour {
			// Clear timestamp difference - use newer
			if source.ModifiedAt.After(existing.ModifiedAt) {
				logging.Debug("smart: source significantly newer, using source",
					logging.Skill(source.Name),
					slog.Duration("time_diff", timeDiff),
				)
				return ActionUpdated, fmt.Sprintf("source is significantly newer (%s > %s)",
					source.ModifiedAt.Format(time.RFC3339),
					existing.ModifiedAt.Format(time.RFC3339)), nil
			}
			logging.Debug("smart: target significantly newer, keeping target",
				logging.Skill(source.Name),
				slog.Duration("time_diff", timeDiff),
			)
			return ActionSkipped, fmt.Sprintf("target is significantly newer (%s > %s)",
				existing.ModifiedAt.Format(time.RFC3339),
				source.ModifiedAt.Format(time.RFC3339)), nil
		}

		// Similar timestamps with conflict - attempt three-way merge
		logging.Debug("smart: attempting three-way merge for concurrent changes",
			logging.Skill(source.Name),
			slog.String("conflict_type", string(conflict.Type)),
		)
		mergeResult := s.merger.TwoWayMerge(source, existing)
		if mergeResult.Success {
			logging.Debug("smart: three-way merge successful",
				logging.Skill(source.Name),
			)
			return ActionMerged, "intelligently merged concurrent changes", nil
		}

		// Merge failed - return conflict for manual resolution
		logging.Debug("smart: merge conflict requires manual resolution",
			logging.Skill(source.Name),
			slog.String("conflict_type", string(conflict.Type)),
		)
		return ActionConflict, "concurrent changes conflict - needs resolution", conflict

	default:
		return ActionUpdated, "updating (default strategy)", nil
	}
}

// SyncWithSkills syncs a specific set of skills to the target platform.
// This is useful when you've already parsed skills and want to sync them.
func (s *Synchronizer) SyncWithSkills(
	skills []model.Skill,
	target model.Platform,
	opts Options,
) (*Result, error) {
	logging.Debug("starting sync with pre-parsed skills",
		logging.Platform(string(target)),
		logging.Operation("sync"),
		logging.Count(len(skills)),
		slog.String(logging.KeyStrategy, string(opts.Strategy)),
		slog.Bool("dry_run", opts.DryRun),
		slog.String("target_scope", string(opts.TargetScope)),
	)

	totalSkills := len(skills)

	// Emit start event
	if err := s.emitProgress(opts, ProgressEvent{
		Type:        ProgressEventStart,
		TotalSkills: totalSkills,
		Message:     fmt.Sprintf("Starting sync of %d skills", totalSkills),
	}); err != nil {
		return nil, fmt.Errorf("progress callback failed: %w", err)
	}

	if totalSkills == 0 {
		logging.Debug("no skills provided to sync")
		// Emit completion event for empty sync
		_ = s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventComplete,
			TotalSkills:     0,
			ProcessedSkills: 0,
			PercentComplete: 100,
			Message:         "No skills to sync",
		})
		return &Result{
			Target:   target,
			Strategy: opts.Strategy,
			DryRun:   opts.DryRun,
			Skills:   make([]SkillResult, 0),
		}, nil
	}

	result := &Result{
		Source:   skills[0].Platform, // Assume all skills are from same platform
		Target:   target,
		Strategy: opts.Strategy,
		DryRun:   opts.DryRun,
		Skills:   make([]SkillResult, 0),
	}

	// Set default strategy
	if result.Strategy == "" {
		result.Strategy = StrategyOverwrite
	}

	// Get target path based on scope
	targetPath := opts.TargetPath
	if targetPath == "" {
		var err error
		if opts.TargetScope != "" {
			targetPath, err = validation.GetPlatformPathForScope(target, opts.TargetScope)
		} else {
			targetPath, err = validation.GetPlatformPath(target)
		}
		if err != nil {
			logging.Error("failed to get target path",
				logging.Platform(string(target)),
				slog.String("scope", string(opts.TargetScope)),
				logging.Err(err),
			)
			return result, fmt.Errorf("failed to get target path: %w", err)
		}
	}
	logging.Debug("determined target path",
		logging.Path(targetPath),
		slog.String("scope", string(opts.TargetScope)),
	)

	// Parse existing target skills
	targetSkills, err := s.parseSkills(target, opts.TargetPath)
	if err != nil {
		logging.Debug("target skills not found, starting fresh",
			logging.Platform(string(target)),
			logging.Err(err),
		)
		targetSkills = []model.Skill{}
	} else {
		logging.Debug("parsed existing target skills",
			logging.Platform(string(target)),
			logging.Count(len(targetSkills)),
		)
	}

	targetSkillMap := make(map[string]model.Skill)
	for _, skill := range targetSkills {
		targetSkillMap[skill.Name] = skill
	}

	// Ensure target directory exists
	if !opts.DryRun {
		if err := os.MkdirAll(targetPath, 0o750); err != nil {
			logging.Error("failed to create target directory",
				logging.Path(targetPath),
				logging.Err(err),
			)
			return result, fmt.Errorf("failed to create target directory: %w", err)
		}
	}

	// Process each skill
	for i, skill := range skills {
		// Emit skill start event
		if err := s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventSkillStart,
			Skill:           &skill,
			TotalSkills:     totalSkills,
			ProcessedSkills: i,
			PercentComplete: (i * 100) / totalSkills,
			Message:         fmt.Sprintf("Processing %s", skill.Name),
		}); err != nil {
			return result, fmt.Errorf("progress callback failed: %w", err)
		}

		skillResult := s.processSkill(skill, target, targetPath, targetSkillMap, opts)
		result.Skills = append(result.Skills, skillResult)

		// Emit skill complete event
		processedCount := i + 1
		if err := s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventSkillComplete,
			Skill:           &skill,
			Action:          skillResult.Action,
			TotalSkills:     totalSkills,
			ProcessedSkills: processedCount,
			PercentComplete: (processedCount * 100) / totalSkills,
			Message:         skillResult.Message,
			Error:           skillResult.Error,
			Conflict:        skillResult.Conflict,
		}); err != nil {
			return result, fmt.Errorf("progress callback failed: %w", err)
		}
	}

	logging.Debug("sync with skills completed",
		logging.Platform(string(target)),
		logging.Count(len(result.Skills)),
	)

	// Emit completion event
	_ = s.emitProgress(opts, ProgressEvent{
		Type:            ProgressEventComplete,
		TotalSkills:     totalSkills,
		ProcessedSkills: totalSkills,
		PercentComplete: 100,
		Message:         fmt.Sprintf("Sync completed: %d skills processed", totalSkills),
	})

	return result, nil
}

// SyncBidirectional performs two-way synchronization between platforms.
// It syncs changes in both directions and reconciles conflicts based on the strategy.
func (s *Synchronizer) SyncBidirectional(platformA, platformB model.Platform, opts Options) (*BidirectionalResult, error) {
	logging.Debug("starting bidirectional sync",
		slog.String("platform_a", string(platformA)),
		slog.String("platform_b", string(platformB)),
		slog.String(logging.KeyStrategy, string(opts.Strategy)),
		slog.Bool("dry_run", opts.DryRun),
	)

	biResult := &BidirectionalResult{
		PlatformA: platformA,
		PlatformB: platformB,
		Strategy:  opts.Strategy,
		DryRun:    opts.DryRun,
	}

	// Parse skills from both platforms
	skillsA, err := s.parseSkills(platformA, opts.SourcePath)
	if err != nil {
		logging.Error("failed to parse platform A skills",
			logging.Platform(string(platformA)),
			logging.Err(err),
		)
		return biResult, fmt.Errorf("failed to parse platform A skills: %w", err)
	}

	skillsB, err := s.parseSkills(platformB, opts.TargetPath)
	if err != nil {
		logging.Error("failed to parse platform B skills",
			logging.Platform(string(platformB)),
			logging.Err(err),
		)
		return biResult, fmt.Errorf("failed to parse platform B skills: %w", err)
	}

	logging.Debug("parsed skills from both platforms",
		slog.String("platform_a", string(platformA)),
		logging.Count(len(skillsA)),
		slog.String("platform_b", string(platformB)),
		slog.Int("count_b", len(skillsB)),
	)

	// Build maps for efficient lookup
	skillsAMap := make(map[string]model.Skill)
	for _, skill := range skillsA {
		skillsAMap[skill.Name] = skill
	}

	skillsBMap := make(map[string]model.Skill)
	for _, skill := range skillsB {
		skillsBMap[skill.Name] = skill
	}

	// Determine which skills need to be synced in which direction
	var syncAtoB []model.Skill // Skills to copy from A to B
	var syncBtoA []model.Skill // Skills to copy from B to A
	var conflicts []BidirectionalConflict

	// Process skills in A
	for _, skillA := range skillsA {
		skillB, existsInB := skillsBMap[skillA.Name]

		if !existsInB {
			// Skill only in A, copy to B
			syncAtoB = append(syncAtoB, skillA)
			continue
		}

		// Skill exists in both - check for conflicts
		conflict := s.conflictDetector.DetectConflict(skillA, skillB)
		if conflict == nil {
			// Content is identical, skip
			continue
		}

		// Determine sync direction based on strategy and timestamps
		direction := s.determineSyncDirection(skillA, skillB, opts.Strategy)
		switch direction {
		case SyncDirectionAtoB:
			syncAtoB = append(syncAtoB, skillA)
		case SyncDirectionBtoA:
			syncBtoA = append(syncBtoA, skillB)
		case SyncDirectionConflict:
			conflicts = append(conflicts, BidirectionalConflict{
				Name:     skillA.Name,
				SkillA:   skillA,
				SkillB:   skillB,
				Conflict: conflict,
			})
		}
	}

	// Process skills only in B
	for _, skillB := range skillsB {
		if _, existsInA := skillsAMap[skillB.Name]; !existsInA {
			// Skill only in B, copy to A
			syncBtoA = append(syncBtoA, skillB)
		}
	}

	logging.Debug("determined sync operations",
		slog.Int("sync_a_to_b", len(syncAtoB)),
		slog.Int("sync_b_to_a", len(syncBtoA)),
		logging.Count(len(conflicts)),
	)

	// Perform sync A -> B
	if len(syncAtoB) > 0 {
		optsAtoB := opts
		optsAtoB.SourcePath = opts.SourcePath
		optsAtoB.TargetPath = opts.TargetPath
		resultAtoB, err := s.SyncWithSkills(syncAtoB, platformB, optsAtoB)
		if err != nil {
			logging.Error("failed to sync A to B",
				logging.Err(err),
			)
			return biResult, fmt.Errorf("failed to sync A to B: %w", err)
		}
		biResult.ResultAtoB = resultAtoB
	}

	// Perform sync B -> A
	if len(syncBtoA) > 0 {
		optsBtoA := opts
		optsBtoA.SourcePath = opts.TargetPath
		optsBtoA.TargetPath = opts.SourcePath
		resultBtoA, err := s.SyncWithSkills(syncBtoA, platformA, optsBtoA)
		if err != nil {
			logging.Error("failed to sync B to A",
				logging.Err(err),
			)
			return biResult, fmt.Errorf("failed to sync B to A: %w", err)
		}
		biResult.ResultBtoA = resultBtoA
	}

	// Store conflicts
	biResult.Conflicts = conflicts

	logging.Debug("bidirectional sync completed",
		slog.String("platform_a", string(platformA)),
		slog.String("platform_b", string(platformB)),
		slog.Int("synced_a_to_b", len(syncAtoB)),
		slog.Int("synced_b_to_a", len(syncBtoA)),
		logging.Count(len(conflicts)),
	)

	return biResult, nil
}

// SyncDirection represents the direction to sync a skill.
type SyncDirection int

const (
	// SyncDirectionAtoB means sync from platform A to B
	SyncDirectionAtoB SyncDirection = iota

	// SyncDirectionBtoA means sync from platform B to A
	SyncDirectionBtoA

	// SyncDirectionConflict means there's a conflict requiring manual resolution
	SyncDirectionConflict
)

// determineSyncDirection decides which direction to sync based on strategy.
func (s *Synchronizer) determineSyncDirection(skillA, skillB model.Skill, strategy Strategy) SyncDirection {
	switch strategy {
	case StrategyNewer:
		// Use timestamp to determine direction
		if skillA.ModifiedAt.After(skillB.ModifiedAt) {
			return SyncDirectionAtoB
		} else if skillB.ModifiedAt.After(skillA.ModifiedAt) {
			return SyncDirectionBtoA
		}
		// Same timestamp - no sync needed, but this shouldn't happen
		// as conflict detector would have caught identical content
		return SyncDirectionConflict

	case StrategyOverwrite:
		// In bidirectional mode with overwrite, prefer A -> B by default
		// This matches the unidirectional behavior
		return SyncDirectionAtoB

	case StrategyThreeWay, StrategyInteractive:
		// These strategies require manual conflict resolution
		return SyncDirectionConflict

	case StrategyMerge:
		// Merge strategy doesn't apply cleanly to bidirectional sync
		// Treat as conflict to be safe
		return SyncDirectionConflict

	case StrategySkip:
		// Skip means don't sync conflicts at all
		return SyncDirectionConflict

	default:
		// Unknown strategy, treat as conflict
		return SyncDirectionConflict
	}
}

// BidirectionalResult represents the result of a bidirectional sync.
type BidirectionalResult struct {
	PlatformA  model.Platform
	PlatformB  model.Platform
	Strategy   Strategy
	DryRun     bool
	ResultAtoB *Result
	ResultBtoA *Result
	Conflicts  []BidirectionalConflict
}

// BidirectionalConflict represents a conflict in bidirectional sync.
type BidirectionalConflict struct {
	Name     string
	SkillA   model.Skill
	SkillB   model.Skill
	Conflict *Conflict
}

// Summary generates a human-readable summary of bidirectional sync results.
func (r *BidirectionalResult) Summary() string {
	var summary string
	if r.DryRun {
		summary = fmt.Sprintf("Bidirectional sync preview: %s <-> %s (strategy: %s)\n\n",
			r.PlatformA, r.PlatformB, r.Strategy)
	} else {
		summary = fmt.Sprintf("Bidirectional sync: %s <-> %s (strategy: %s)\n\n",
			r.PlatformA, r.PlatformB, r.Strategy)
	}

	if r.ResultAtoB != nil {
		summary += fmt.Sprintf("Direction %s -> %s:\n", r.PlatformA, r.PlatformB)
		summary += fmt.Sprintf("  Created:   %d\n", len(r.ResultAtoB.Created()))
		summary += fmt.Sprintf("  Updated:   %d\n", len(r.ResultAtoB.Updated()))
		summary += fmt.Sprintf("  Skipped:   %d\n", len(r.ResultAtoB.Skipped()))
		summary += fmt.Sprintf("  Failed:    %d\n\n", len(r.ResultAtoB.Failed()))
	}

	if r.ResultBtoA != nil {
		summary += fmt.Sprintf("Direction %s -> %s:\n", r.PlatformB, r.PlatformA)
		summary += fmt.Sprintf("  Created:   %d\n", len(r.ResultBtoA.Created()))
		summary += fmt.Sprintf("  Updated:   %d\n", len(r.ResultBtoA.Updated()))
		summary += fmt.Sprintf("  Skipped:   %d\n", len(r.ResultBtoA.Skipped()))
		summary += fmt.Sprintf("  Failed:    %d\n\n", len(r.ResultBtoA.Failed()))
	}

	if len(r.Conflicts) > 0 {
		summary += fmt.Sprintf("Conflicts requiring manual resolution: %d\n", len(r.Conflicts))
		for _, conflict := range r.Conflicts {
			conflictType := "unknown"
			if conflict.Conflict != nil {
				conflictType = string(conflict.Conflict.Type)
			}
			summary += fmt.Sprintf("  - %s (%s)\n", conflict.Name, conflictType)
		}
	}

	return summary
}

// HasConflicts returns true if there are any conflicts.
func (r *BidirectionalResult) HasConflicts() bool {
	return len(r.Conflicts) > 0
}

// TotalProcessed returns the total number of skills processed in both directions.
func (r *BidirectionalResult) TotalProcessed() int {
	total := 0
	if r.ResultAtoB != nil {
		total += r.ResultAtoB.TotalProcessed()
	}
	if r.ResultBtoA != nil {
		total += r.ResultBtoA.TotalProcessed()
	}
	return total
}

// TotalChanged returns the total number of skills changed (created or updated) in both directions.
func (r *BidirectionalResult) TotalChanged() int {
	total := 0
	if r.ResultAtoB != nil {
		total += r.ResultAtoB.TotalChanged()
	}
	if r.ResultBtoA != nil {
		total += r.ResultBtoA.TotalChanged()
	}
	return total
}
