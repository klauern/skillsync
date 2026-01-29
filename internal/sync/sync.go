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

// Sync performs synchronization from source to target platform.
func (s *Synchronizer) Sync(source, target model.Platform, opts Options) (*Result, error) {
	defer logging.Timer("sync")()

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

	if len(sourceSkills) == 0 {
		logging.Debug("no skills to sync",
			logging.Platform(string(source)),
		)
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
	for _, sourceSkill := range sourceSkills {
		skillResult := s.processSkill(sourceSkill, target, targetPath, targetSkillMap, opts)
		result.Skills = append(result.Skills, skillResult)
	}

	logging.Debug("sync operation completed",
		logging.Platform(string(source)),
		slog.String("target", string(target)),
		logging.Count(len(result.Skills)),
	)

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

	if len(skills) == 0 {
		logging.Debug("no skills provided to sync")
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
	for _, skill := range skills {
		skillResult := s.processSkill(skill, target, targetPath, targetSkillMap, opts)
		result.Skills = append(result.Skills, skillResult)
	}

	logging.Debug("sync with skills completed",
		logging.Platform(string(target)),
		logging.Count(len(result.Skills)),
	)

	return result, nil
}
