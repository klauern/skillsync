package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	transformer *Transformer
}

// New creates a new Synchronizer.
func New() *Synchronizer {
	return &Synchronizer{
		transformer: NewTransformer(),
	}
}

// Sync performs synchronization from source to target platform.
func (s *Synchronizer) Sync(source, target model.Platform, opts Options) (*Result, error) {
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
		return result, fmt.Errorf("failed to parse source skills: %w", err)
	}

	if len(sourceSkills) == 0 {
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
		// Target may not exist yet, which is okay
		targetSkills = []model.Skill{}
	}

	// Build a map of existing target skills by name
	targetSkillMap := make(map[string]model.Skill)
	for _, skill := range targetSkills {
		targetSkillMap[skill.Name] = skill
	}

	// Ensure target directory exists (unless dry run)
	if !opts.DryRun {
		if err := os.MkdirAll(targetPath, 0o750); err != nil {
			return result, fmt.Errorf("failed to create target directory: %w", err)
		}
	}

	// Process each source skill
	for _, sourceSkill := range sourceSkills {
		skillResult := s.processSkill(sourceSkill, target, targetPath, targetSkillMap, opts)
		result.Skills = append(result.Skills, skillResult)
	}

	return result, nil
}

// parseSkills parses skills from the given platform.
func (s *Synchronizer) parseSkills(platform model.Platform, basePath string) ([]model.Skill, error) {
	var p parser.Parser

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
	result := SkillResult{
		Skill: source,
	}

	// Transform the skill for the target platform
	transformed, err := s.transformer.Transform(source, targetPlatform)
	if err != nil {
		result.Action = ActionFailed
		result.Error = fmt.Errorf("transformation failed: %w", err)
		return result
	}

	// Determine target file path
	targetFilePath := filepath.Join(targetPath, transformed.Path)
	result.TargetPath = targetFilePath

	// Check if skill exists in target
	existingSkill, exists := existingSkills[source.Name]

	// Determine action based on strategy
	action, message := s.determineAction(source, existingSkill, exists, opts.Strategy)
	result.Action = action
	result.Message = message

	// If skipping, we're done
	if action == ActionSkipped {
		return result
	}

	// Get content to write
	content := transformed.Content

	// Handle merge strategy
	if action == ActionMerged && exists {
		content = s.transformer.MergeContent(transformed.Content, existingSkill.Content, source.Name)
	}

	// Write the file (unless dry run)
	if !opts.DryRun {
		// #nosec G306 - skill files should be readable
		if err := os.WriteFile(targetFilePath, []byte(content), 0o644); err != nil {
			result.Action = ActionFailed
			result.Error = fmt.Errorf("failed to write file: %w", err)
			return result
		}
	}

	return result
}

// determineAction decides what action to take based on strategy.
func (s *Synchronizer) determineAction(
	source model.Skill,
	existing model.Skill,
	exists bool,
	strategy Strategy,
) (Action, string) {
	if !exists {
		return ActionCreated, "new skill"
	}

	switch strategy {
	case StrategyOverwrite:
		return ActionUpdated, "overwriting existing skill"

	case StrategySkip:
		return ActionSkipped, "skill already exists"

	case StrategyNewer:
		if source.ModifiedAt.After(existing.ModifiedAt) {
			return ActionUpdated, fmt.Sprintf("source is newer (%s > %s)",
				source.ModifiedAt.Format(time.RFC3339),
				existing.ModifiedAt.Format(time.RFC3339))
		}
		return ActionSkipped, fmt.Sprintf("target is newer or same age (%s >= %s)",
			existing.ModifiedAt.Format(time.RFC3339),
			source.ModifiedAt.Format(time.RFC3339))

	case StrategyMerge:
		return ActionMerged, "merging with existing content"

	default:
		return ActionUpdated, "updating (default strategy)"
	}
}

// SyncWithSkills syncs a specific set of skills to the target platform.
// This is useful when you've already parsed skills and want to sync them.
func (s *Synchronizer) SyncWithSkills(
	skills []model.Skill,
	target model.Platform,
	opts Options,
) (*Result, error) {
	result := &Result{
		Source:   skills[0].Platform, // Assume all skills are from same platform
		Target:   target,
		Strategy: opts.Strategy,
		DryRun:   opts.DryRun,
		Skills:   make([]SkillResult, 0),
	}

	if len(skills) == 0 {
		return result, nil
	}

	// Set default strategy
	if result.Strategy == "" {
		result.Strategy = StrategyOverwrite
	}

	// Get target path
	targetPath := opts.TargetPath
	if targetPath == "" {
		var err error
		targetPath, err = validation.GetPlatformPath(target)
		if err != nil {
			return result, fmt.Errorf("failed to get target path: %w", err)
		}
	}

	// Parse existing target skills
	targetSkills, err := s.parseSkills(target, opts.TargetPath)
	if err != nil {
		targetSkills = []model.Skill{}
	}

	targetSkillMap := make(map[string]model.Skill)
	for _, skill := range targetSkills {
		targetSkillMap[skill.Name] = skill
	}

	// Ensure target directory exists
	if !opts.DryRun {
		if err := os.MkdirAll(targetPath, 0o750); err != nil {
			return result, fmt.Errorf("failed to create target directory: %w", err)
		}
	}

	// Process each skill
	for _, skill := range skills {
		skillResult := s.processSkill(skill, target, targetPath, targetSkillMap, opts)
		result.Skills = append(result.Skills, skillResult)
	}

	return result, nil
}
