package sync

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
	"github.com/klauern/skillsync/internal/parser"
	"github.com/klauern/skillsync/internal/parser/claude"
	"github.com/klauern/skillsync/internal/parser/codex"
	"github.com/klauern/skillsync/internal/parser/copilot"
	"github.com/klauern/skillsync/internal/parser/cursor"
	"github.com/klauern/skillsync/internal/parser/gemini"
	"github.com/klauern/skillsync/internal/parser/pidev"
	"github.com/klauern/skillsync/internal/util"
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

	// Progress is called during sync operations to report incremental progress.
	// If nil, progress events are suppressed.
	Progress ProgressCallback

	// Bidirectional enables two-way sync (both platforms can be source and target).
	// Call SyncBidirectional to use it explicitly.
	Bidirectional bool

	// DeleteMode enables deletion sync: deletes skills from target that match source.
	// Instead of copying skills TO target, removes skills FROM target that exist in source.
	DeleteMode bool
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
	logging.Debug(
		"starting sync operation",
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
		logging.Error(
			"failed to parse source skills",
			logging.Platform(string(source)),
			logging.Operation("sync"),
			logging.Err(err),
		)
		return result, fmt.Errorf("failed to parse source skills: %w", err)
	}
	if !opts.SkipValidation {
		if err := validateSourceSkills(sourceSkills, source); err != nil {
			return result, err
		}
	}
	result.SelectedCount = len(sourceSkills)
	result.TotalAvailable = len(sourceSkills)

	logging.Debug(
		"parsed source skills",
		logging.Platform(string(source)),
		logging.Count(len(sourceSkills)),
	)

	totalSkills := len(sourceSkills)
	if err := s.emitProgress(opts, ProgressEvent{
		Type:        ProgressEventStart,
		TotalSkills: totalSkills,
		Message:     progressStartMessage(totalSkills, "skills"),
	}); err != nil {
		return result, fmt.Errorf("progress callback failed: %w", err)
	}

	if totalSkills == 0 {
		logging.Debug(
			"no skills to sync",
			logging.Platform(string(source)),
		)
		_ = s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventComplete,
			TotalSkills:     0,
			ProcessedSkills: 0,
			PercentComplete: 100,
			Message:         "No skills to sync",
		})
		return result, nil // Nothing to sync
	}

	// Skip nested skills when a parent directory/symlink skill is also present.
	// The parent copy already includes nested content, so syncing both creates
	// duplicate top-level artifacts.
	filteredSourceSkills, nestedSkipped := filterNestedDirectorySkills(sourceSkills)
	if len(nestedSkipped) > 0 {
		result.Skills = append(result.Skills, nestedSkipped...)
	}
	sourceSkills = filteredSourceSkills

	if len(sourceSkills) == 0 {
		logging.Debug("all source skills were skipped as nested duplicates")
		_ = s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventComplete,
			TotalSkills:     totalSkills,
			ProcessedSkills: 0,
			PercentComplete: 100,
			Message:         "All source skills were skipped as nested duplicates",
		})
		return result, nil
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
		logging.Debug(
			"target skills not found, starting fresh",
			logging.Platform(string(target)),
			logging.Err(err),
		)
		// Target may not exist yet, which is okay
		targetSkills = []model.Skill{}
	} else {
		logging.Debug(
			"parsed target skills",
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
			logging.Error(
				"failed to create target directory",
				logging.Path(targetPath),
				logging.Err(err),
			)
			return result, fmt.Errorf("failed to create target directory: %w", err)
		}
		logging.Debug(
			"ensured target directory exists",
			logging.Path(targetPath),
		)
	}

	// Process each source skill
	for i, sourceSkill := range sourceSkills {
		if err := s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventSkillStart,
			Skill:           &sourceSkill,
			TotalSkills:     totalSkills,
			ProcessedSkills: i,
			PercentComplete: progressPercent(i, totalSkills),
			Message:         fmt.Sprintf("Processing %s", sourceSkill.Name),
		}); err != nil {
			return result, fmt.Errorf("progress callback failed: %w", err)
		}

		skillResult := s.processSkill(sourceSkill, target, targetPath, targetSkillMap, opts)
		result.Skills = append(result.Skills, skillResult)

		processedCount := i + 1
		if err := s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventSkillComplete,
			Skill:           &sourceSkill,
			Action:          skillResult.Action,
			TotalSkills:     totalSkills,
			ProcessedSkills: processedCount,
			PercentComplete: progressPercent(processedCount, totalSkills),
			Message:         skillResult.Message,
			Error:           skillResult.Error,
			Conflict:        skillResult.Conflict,
		}); err != nil {
			return result, fmt.Errorf("progress callback failed: %w", err)
		}
	}

	logging.Debug(
		"sync operation completed",
		logging.Platform(string(source)),
		slog.String("target", string(target)),
		logging.Count(len(result.Skills)),
	)

	_ = s.emitProgress(opts, ProgressEvent{
		Type:            ProgressEventComplete,
		TotalSkills:     totalSkills,
		ProcessedSkills: len(sourceSkills),
		PercentComplete: 100,
		Message:         fmt.Sprintf("Sync completed: %d skills processed", len(sourceSkills)),
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
	case model.Copilot:
		p = copilot.New(basePath)
	case model.Gemini:
		p = gemini.New(basePath)
	case model.PiDev:
		p = pidev.New(basePath)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	skills, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse synchronized skills for %s: %w", platform, err)
	}
	return skills, nil
}

// filterNestedDirectorySkills removes nested directory/symlink skills that would
// filterNestedDirectorySkills filters out skills whose source paths are nested inside
// another skill's directory or symlink and returns the remaining skills along with
// SkillResult entries for each skipped nested skill.
//
// The function prefers the deepest matching parent directory/symlink when deciding
// which parent causes a child to be skipped. For each skipped skill a SkillResult
// with ActionSkipped and a message indicating the parent skill name is returned.
func filterNestedDirectorySkills(skills []model.Skill) ([]model.Skill, []SkillResult) {
	if len(skills) < 2 {
		return skills, nil
	}

	type rootInfo struct {
		skill      model.Skill
		sourceType SourceType
		rootPath   string
	}

	roots := make([]rootInfo, 0, len(skills))
	for _, skill := range skills {
		sourceType, rootPath := detectSourceType(skill.Path)
		roots = append(roots, rootInfo{
			skill:      skill,
			sourceType: sourceType,
			rootPath:   filepath.Clean(rootPath),
		})
	}

	// Build skip map by index, preferring the deepest matching parent path.
	skipParent := make(map[int]string)
	for childIdx := range roots {
		child := roots[childIdx]
		bestParentDepth := -1
		bestParentName := ""

		for parentIdx := range roots {
			if parentIdx == childIdx {
				continue
			}

			parent := roots[parentIdx]
			if parent.sourceType != SourceTypeDirectory && parent.sourceType != SourceTypeSymlink {
				continue
			}

			if !isNestedPath(child.rootPath, parent.rootPath) {
				continue
			}

			depth := pathDepth(parent.rootPath)
			if depth > bestParentDepth {
				bestParentDepth = depth
				bestParentName = parent.skill.Name
			}
		}

		if bestParentName != "" {
			skipParent[childIdx] = bestParentName
		}
	}

	if len(skipParent) == 0 {
		return skills, nil
	}

	filtered := make([]model.Skill, 0, len(skills)-len(skipParent))
	skipped := make([]SkillResult, 0, len(skipParent))
	for idx, info := range roots {
		parentName, shouldSkip := skipParent[idx]
		if !shouldSkip {
			filtered = append(filtered, info.skill)
			continue
		}

		msg := fmt.Sprintf("nested skill already included via parent directory copy: %s", parentName)
		skipped = append(skipped, SkillResult{
			Skill:   info.skill,
			Action:  ActionSkipped,
			Message: msg,
		})
		logging.Debug(
			"skipping nested skill to avoid duplicate copy",
			logging.Skill(info.skill.Name),
			slog.String("parent_skill", parentName),
			logging.Path(info.rootPath),
		)
	}

	return filtered, skipped
}

func isNestedPath(path, parent string) bool {
	if path == parent {
		return false
	}

	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	if rel == "." || rel == ".." {
		return false
	}

	parentPrefix := ".." + string(os.PathSeparator)
	return !strings.HasPrefix(rel, parentPrefix)
}

func pathDepth(path string) int {
	cleaned := filepath.Clean(path)
	if cleaned == string(os.PathSeparator) || cleaned == "." {
		return 0
	}
	return strings.Count(cleaned, string(os.PathSeparator))
}

// processSkill handles syncing a single skill.
// It preserves the source structure: symlinks become symlinks, directories become directories.
//
//nolint:gocyclo // This dispatcher intentionally branches on source type, target type, and strategy.
func (s *Synchronizer) processSkill(
	source model.Skill,
	targetPlatform model.Platform,
	targetPath string,
	existingSkills map[string]model.Skill,
	opts Options,
) SkillResult {
	logging.Debug(
		"processing skill",
		logging.Skill(source.Name),
		logging.Platform(string(source.Platform)),
		slog.String("target", string(targetPlatform)),
	)

	result := SkillResult{
		Skill: source,
	}

	// Detect source type and get source root path
	sourceType, sourceRootPath := detectSourceType(source.Path)

	logging.Debug(
		"detected source type",
		logging.Skill(source.Name),
		slog.String("source_type", sourceType.String()),
		logging.Path(sourceRootPath),
	)

	// For symlinks and directories, use the skill name directly.
	// For files, use the transformed path (legacy behavior).
	var targetEntryPath string
	if sourceType == SourceTypeSymlink || sourceType == SourceTypeDirectory {
		// Preserve structure: target is just the skill name in the target directory
		targetEntryPath = filepath.Join(targetPath, source.Name)
	} else {
		// Legacy file behavior: transform path for target platform
		transformed, err := s.transformer.Transform(source, targetPlatform)
		if err != nil {
			logging.Warn(
				"transformation failed",
				logging.Skill(source.Name),
				logging.Err(err),
			)
			result.Action = ActionFailed
			result.Error = fmt.Errorf("transformation failed: %w", err)
			return result
		}
		targetEntryPath = filepath.Join(targetPath, transformed.Path)
	}

	result.TargetPath = targetEntryPath
	result.PortabilityWarnings = portabilityWarningsForSkill(source, targetPlatform)

	// Check if skill exists in target
	existingSkill, exists := existingSkills[source.Name]
	if !exists {
		// os.Lstat (not Stat) is intentional: captures the symlink's own mtime,
		// not the target's. Type/Scope/Metadata/PluginInfo are synthesized from
		// the source — determineAction only reads Name and ModifiedAt for most
		// strategies, so this is safe for current use.
		if info, err := os.Lstat(targetEntryPath); err == nil {
			existingSkill = model.Skill{
				Name:       source.Name,
				Platform:   targetPlatform,
				Path:       targetEntryPath,
				ModifiedAt: info.ModTime(),
				Type:       source.Type,
				Scope:      source.Scope,
				Metadata:   map[string]string{},
				PluginInfo: nil,
			}
			exists = true
		}
	}

	// Determine action based on strategy
	action, message, conflict := s.determineAction(source, existingSkill, exists, opts.Strategy)
	result.Action = action
	result.Message = message
	result.Conflict = conflict
	var extras []string
	if shouldLinkClaudeDirectorySkill(source, targetPlatform) && !needsCanonicalEntrypointCopy(source, targetPlatform) && action != ActionSkipped && action != ActionConflict {
		extras = append(extras, "linked Claude skill directory")
	}
	if warning := mappingWarning(source, targetPlatform); warning != "" {
		extras = append(extras, warning)
	}
	if len(extras) > 0 {
		if result.Message != "" {
			result.Message += "; "
		}
		result.Message += strings.Join(extras, "; ")
	}

	logging.Debug(
		"action determined",
		logging.Skill(source.Name),
		slog.String("action", string(action)),
		slog.String("message", message),
		slog.Bool("has_conflict", conflict != nil),
	)

	// If skipping or conflict (needs external resolution), we're done
	if action == ActionSkipped || action == ActionConflict {
		return result
	}

	// Execute the sync (unless dry run)
	if !opts.DryRun {
		// Remove any existing entry at target path to avoid duplicates
		if err := removeExisting(targetEntryPath); err != nil {
			logging.Error(
				"failed to remove existing entry",
				logging.Skill(source.Name),
				logging.Path(targetEntryPath),
				logging.Err(err),
			)
			result.Action = ActionFailed
			result.Error = fmt.Errorf("failed to remove existing entry: %w", err)
			return result
		}

		// Create based on source type
		switch sourceType {
		case SourceTypeSymlink:
			if source.Platform != targetPlatform && isSkillFile(filepath.Base(source.Path)) {
				resolvedSource, err := filepath.EvalSymlinks(sourceRootPath)
				if err != nil {
					result.Action = ActionFailed
					result.Error = fmt.Errorf("failed to resolve cross-harness skill symlink: %w", err)
					return result
				}
				if err := s.copyTransformedBundle(source, targetPlatform, resolvedSource, targetEntryPath); err != nil {
					result.Action = ActionFailed
					result.Error = err
					return result
				}
				break
			}
			// Recreate symlink with same target
			symlinkTarget := getSymlinkTarget(sourceRootPath)
			if symlinkTarget == "" {
				// Fallback: try from PluginInfo
				if source.PluginInfo != nil && source.PluginInfo.SymlinkTarget != "" {
					symlinkTarget = source.PluginInfo.SymlinkTarget
				}
			}

			if symlinkTarget == "" {
				logging.Error(
					"failed to determine symlink target",
					logging.Skill(source.Name),
					logging.Path(sourceRootPath),
				)
				result.Action = ActionFailed
				result.Error = fmt.Errorf("failed to determine symlink target for %q", sourceRootPath)
				return result
			}

			if err := os.Symlink(symlinkTarget, targetEntryPath); err != nil {
				logging.Error(
					"failed to create symlink",
					logging.Skill(source.Name),
					logging.Path(targetEntryPath),
					logging.Err(err),
				)
				result.Action = ActionFailed
				result.Error = fmt.Errorf("failed to create symlink: %w", err)
				return result
			}

			logging.Debug(
				"created symlink",
				logging.Skill(source.Name),
				logging.Path(targetEntryPath),
				slog.String("target", symlinkTarget),
			)

		case SourceTypeDirectory:
			if shouldLinkClaudeDirectorySkill(source, targetPlatform) && !needsCanonicalEntrypointCopy(source, targetPlatform) {
				if err := os.MkdirAll(filepath.Dir(targetEntryPath), 0o750); err != nil {
					result.Action = ActionFailed
					result.Error = fmt.Errorf("failed to create parent directories for symlink: %w", err)
					return result
				}
				absSource, err := filepath.Abs(sourceRootPath)
				if err != nil {
					result.Action = ActionFailed
					result.Error = fmt.Errorf("failed to resolve absolute source path for symlink: %w", err)
					return result
				}
				if err := os.Symlink(absSource, targetEntryPath); err != nil {
					logging.Error(
						"failed to create Claude skill symlink",
						logging.Skill(source.Name),
						logging.Path(targetEntryPath),
						logging.Err(err),
					)
					result.Action = ActionFailed
					result.Error = fmt.Errorf("failed to create Claude skill symlink: %w", err)
					return result
				}

				logging.Debug(
					"linked Claude skill directory",
					logging.Skill(source.Name),
					logging.Path(targetEntryPath),
					logging.Path(sourceRootPath),
				)
				break
			}
			if source.Platform != targetPlatform && isSkillFile(filepath.Base(source.Path)) {
				if err := s.copyTransformedBundle(source, targetPlatform, sourceRootPath, targetEntryPath); err != nil {
					result.Action = ActionFailed
					result.Error = err
					return result
				}
				break
			}
			if needsCanonicalEntrypointCopy(source, targetPlatform) {
				if err := copySkillDir(sourceRootPath, targetEntryPath, source.Path); err != nil {
					logging.Error(
						"failed to copy directory with canonical entrypoint",
						logging.Skill(source.Name),
						logging.Path(targetEntryPath),
						logging.Err(err),
					)
					result.Action = ActionFailed
					result.Error = fmt.Errorf("failed to copy directory with canonical entrypoint: %w", err)
					return result
				}

				logging.Debug(
					"copied directory with canonical entrypoint",
					logging.Skill(source.Name),
					logging.Path(targetEntryPath),
				)
				break
			}

			// Copy directory structure for non-Claude or non-linkable targets.
			if err := copyDir(sourceRootPath, targetEntryPath); err != nil {
				logging.Error(
					"failed to copy directory",
					logging.Skill(source.Name),
					logging.Path(targetEntryPath),
					logging.Err(err),
				)
				result.Action = ActionFailed
				result.Error = fmt.Errorf("failed to copy directory: %w", err)
				return result
			}

			logging.Debug(
				"copied directory",
				logging.Skill(source.Name),
				logging.Path(targetEntryPath),
			)

		case SourceTypeFile:
			// Legacy behavior: write transformed content
			transformed, err := s.transformer.Transform(source, targetPlatform)
			if err != nil {
				result.Action = ActionFailed
				result.Error = fmt.Errorf("transformation failed: %w", err)
				return result
			}

			content := transformed.Content

			// Handle merge strategy
			if action == ActionMerged && exists {
				logging.Debug(
					"merging content",
					logging.Skill(source.Name),
				)
				content = s.transformer.MergeContent(transformed.Content, existingSkill.Content, source.Name)
			}

			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(targetEntryPath), 0o750); err != nil {
				logging.Error(
					"failed to create target subdirectory",
					logging.Skill(source.Name),
					logging.Path(targetEntryPath),
					logging.Err(err),
				)
				result.Action = ActionFailed
				result.Error = fmt.Errorf("failed to create target subdirectory: %w", err)
				return result
			}

			// #nosec G301 G306 - skill files should be readable.
			if err := util.WriteFileWithPerms(targetEntryPath, []byte(content), 0o750, 0o644); err != nil {
				logging.Error(
					"failed to write skill file",
					logging.Skill(source.Name),
					logging.Path(targetEntryPath),
					logging.Err(err),
				)
				result.Action = ActionFailed
				result.Error = fmt.Errorf("failed to write file: %w", err)
				return result
			}

			logging.Debug(
				"wrote skill file",
				logging.Skill(source.Name),
				logging.Path(targetEntryPath),
			)
		}
	}

	return result
}

// mappingWarning builds a semicolon-separated warning string describing lossy mappings
// when converting a skill to the given target platform.
// It reports fields that will be preserved only as metadata, may require target-specific
// configuration, or will be dropped. The returned string is empty if no warnings apply.
func mappingWarning(skill model.Skill, target model.Platform) string {
	warnings := []string{}
	if skill.Type == model.SkillTypePrompt && target == model.Codex {
		warnings = append(warnings, "lossy mapping: prompt trigger semantics are not guaranteed on Codex")
	}
	if skill.Type == model.SkillTypePrompt && target == model.Cursor && skill.Trigger != "" {
		warnings = append(warnings, "lossy mapping: prompt trigger may require Cursor mode configuration")
	}
	if _, ok := skill.Metadata["argument-hint"]; ok && target != model.ClaudeCode {
		warnings = append(warnings, "lossy mapping: argument-hint preserved as metadata only")
	}
	if _, ok := skill.Metadata["applyTo"]; ok && target != model.Cursor {
		warnings = append(warnings, "lossy mapping: applyTo preserved as metadata only")
	}
	if _, ok := skill.Metadata["model"]; ok && target != model.ClaudeCode {
		warnings = append(warnings, "lossy mapping: model preserved as metadata only")
	}
	if len(skill.Tools) > 0 && target == model.PiDev {
		warnings = append(warnings, "lossy mapping: allowed-tools preserved as metadata only")
	}
	if skill.DisableModelInvocation && target != model.ClaudeCode {
		warnings = append(warnings, "lossy mapping: disable-model-invocation preserved as metadata only")
	}
	if _, ok := skill.Metadata["handoffs"]; ok {
		warnings = append(warnings, "lossy mapping: handoffs dropped without target equivalent")
	}
	if _, ok := skill.Metadata["target"]; ok {
		warnings = append(warnings, "lossy mapping: target dropped without target equivalent")
	}
	if _, ok := skill.Metadata["mcp-servers"]; ok {
		warnings = append(warnings, "lossy mapping: mcp-servers dropped without target equivalent")
	}

	return strings.Join(warnings, "; ")
}

// determineAction decides what action to take based on strategy.
func (s *Synchronizer) determineAction(
	source model.Skill,
	existing model.Skill,
	exists bool,
	strategy Strategy,
) (Action, string, *Conflict) {
	logging.Debug(
		"determining action",
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
			logging.Debug(
				"source is newer",
				logging.Skill(source.Name),
				slog.Time("source_modified", source.ModifiedAt),
				slog.Time("existing_modified", existing.ModifiedAt),
			)
			return ActionUpdated, fmt.Sprintf("source is newer (%s > %s)",
				source.ModifiedAt.Format(time.RFC3339),
				existing.ModifiedAt.Format(time.RFC3339)), nil
		}
		logging.Debug(
			"target is newer or same age",
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
			logging.Debug(
				"no conflict detected, content identical",
				logging.Skill(source.Name),
			)
			return ActionSkipped, "content is identical", nil
		}
		// Attempt three-way merge
		logging.Debug(
			"attempting three-way merge",
			logging.Skill(source.Name),
			slog.String("conflict_type", string(conflict.Type)),
		)
		mergeResult := s.merger.TwoWayMerge(source, existing)
		if mergeResult.Success {
			logging.Debug(
				"three-way merge successful",
				logging.Skill(source.Name),
			)
			return ActionMerged, "three-way merge successful", nil
		}
		// Has conflicts that need resolution
		logging.Debug(
			"conflict requires manual resolution",
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
		logging.Debug(
			"conflict detected for interactive resolution",
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
	logging.Debug(
		"starting sync with pre-parsed skills",
		logging.Platform(string(target)),
		logging.Operation("sync"),
		logging.Count(len(skills)),
		slog.String(logging.KeyStrategy, string(opts.Strategy)),
		slog.Bool("dry_run", opts.DryRun),
		slog.String("target_scope", string(opts.TargetScope)),
	)

	if len(skills) == 0 {
		logging.Debug("no skills provided to sync")
		_ = s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventComplete,
			TotalSkills:     0,
			ProcessedSkills: 0,
			PercentComplete: 100,
			Message:         "No skills provided to sync",
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
	result.SelectedCount = len(skills)
	result.TotalAvailable = len(skills)
	if !opts.SkipValidation {
		if err := validateSourceSkills(skills, result.Source); err != nil {
			return result, err
		}
	}

	// Set default strategy
	if result.Strategy == "" {
		result.Strategy = StrategyOverwrite
	}

	// Skip nested skills when a parent directory/symlink skill is also present.
	// The parent copy already includes nested content, so syncing both creates
	// duplicate top-level artifacts.
	filteredSkills, nestedSkipped := filterNestedDirectorySkills(skills)
	if len(nestedSkipped) > 0 {
		result.Skills = append(result.Skills, nestedSkipped...)
	}
	skills = filteredSkills

	if len(skills) == 0 {
		logging.Debug("all pre-parsed skills were skipped as nested duplicates")
		_ = s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventComplete,
			TotalSkills:     result.TotalAvailable,
			ProcessedSkills: 0,
			PercentComplete: 100,
			Message:         "All pre-parsed skills were skipped as nested duplicates",
		})
		return result, nil
	}

	if err := s.emitProgress(opts, ProgressEvent{
		Type:        ProgressEventStart,
		TotalSkills: len(skills),
		Message:     progressStartMessage(len(skills), "skills"),
	}); err != nil {
		return result, fmt.Errorf("progress callback failed: %w", err)
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
			logging.Error(
				"failed to get target path",
				logging.Platform(string(target)),
				slog.String("scope", string(opts.TargetScope)),
				logging.Err(err),
			)
			return result, fmt.Errorf("failed to get target path: %w", err)
		}
	}
	logging.Debug(
		"determined target path",
		logging.Path(targetPath),
		slog.String("scope", string(opts.TargetScope)),
	)

	// Parse existing target skills
	targetSkills, err := s.parseSkills(target, opts.TargetPath)
	if err != nil {
		logging.Debug(
			"target skills not found, starting fresh",
			logging.Platform(string(target)),
			logging.Err(err),
		)
		targetSkills = []model.Skill{}
	} else {
		logging.Debug(
			"parsed existing target skills",
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
			logging.Error(
				"failed to create target directory",
				logging.Path(targetPath),
				logging.Err(err),
			)
			return result, fmt.Errorf("failed to create target directory: %w", err)
		}
	}

	// Process each skill
	for i, skill := range skills {
		if err := s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventSkillStart,
			Skill:           &skill,
			TotalSkills:     len(skills),
			ProcessedSkills: i,
			PercentComplete: progressPercent(i, len(skills)),
			Message:         fmt.Sprintf("Processing %s", skill.Name),
		}); err != nil {
			return result, fmt.Errorf("progress callback failed: %w", err)
		}

		skillResult := s.processSkill(skill, target, targetPath, targetSkillMap, opts)
		result.Skills = append(result.Skills, skillResult)

		processedCount := i + 1
		if err := s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventSkillComplete,
			Skill:           &skill,
			Action:          skillResult.Action,
			TotalSkills:     len(skills),
			ProcessedSkills: processedCount,
			PercentComplete: progressPercent(processedCount, len(skills)),
			Message:         skillResult.Message,
			Error:           skillResult.Error,
			Conflict:        skillResult.Conflict,
		}); err != nil {
			return result, fmt.Errorf("progress callback failed: %w", err)
		}
	}

	logging.Debug(
		"sync with skills completed",
		logging.Platform(string(target)),
		logging.Count(len(result.Skills)),
	)

	_ = s.emitProgress(opts, ProgressEvent{
		Type:            ProgressEventComplete,
		TotalSkills:     len(skills),
		ProcessedSkills: len(skills),
		PercentComplete: 100,
		Message:         fmt.Sprintf("Sync completed: %d skills processed", len(skills)),
	})

	return result, nil
}

func validateSourceSkills(skills []model.Skill, platform model.Platform) error {
	validationResult, err := validation.ValidateSkillsFormat(skills, platform)
	if err != nil {
		return fmt.Errorf("source validation failed: %w", err)
	}
	if err := validationResult.Error(); err != nil {
		return fmt.Errorf("source validation failed: %w", err)
	}
	return nil
}

func (s *Synchronizer) copyTransformedBundle(source model.Skill, target model.Platform, sourceRoot, targetRoot string) error {
	if err := copySkillDir(sourceRoot, targetRoot, source.Path); err != nil {
		return fmt.Errorf("failed to copy cross-harness skill bundle: %w", err)
	}
	transformed, err := s.transformer.Transform(source, target)
	if err != nil {
		return fmt.Errorf("failed to transform cross-harness skill entrypoint: %w", err)
	}
	entrypoint := filepath.Join(targetRoot, "SKILL.md")
	// #nosec G301 G306 -- synchronized skill entrypoints are intentionally readable.
	if err := util.WriteFileWithPerms(entrypoint, []byte(transformed.Content), 0o750, 0o644); err != nil {
		return fmt.Errorf("failed to write transformed cross-harness skill entrypoint: %w", err)
	}
	return nil
}

// DeleteWithSkills deletes skills from target that match the source skills.
// This is the inverse of sync: instead of copying skills TO target, it removes skills FROM target
// that exist in the source. Useful for cleaning up test skills or removing deprecated skills.
func (s *Synchronizer) DeleteWithSkills(
	sourceSkills []model.Skill,
	target model.Platform,
	opts Options,
) (*Result, error) {
	logging.Debug(
		"starting delete sync operation",
		logging.Platform(string(target)),
		logging.Operation("delete-sync"),
		logging.Count(len(sourceSkills)),
		slog.Bool("dry_run", opts.DryRun),
		slog.String("target_scope", string(opts.TargetScope)),
	)

	if len(sourceSkills) == 0 {
		logging.Debug("no skills provided to delete")
		_ = s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventComplete,
			TotalSkills:     0,
			ProcessedSkills: 0,
			PercentComplete: 100,
			Message:         "No skills provided to delete",
		})
		return &Result{
			Target:   target,
			Strategy: opts.Strategy,
			DryRun:   opts.DryRun,
			Skills:   make([]SkillResult, 0),
		}, nil
	}

	result := &Result{
		Source:   sourceSkills[0].Platform,
		Target:   target,
		Strategy: opts.Strategy,
		DryRun:   opts.DryRun,
		Skills:   make([]SkillResult, 0),
	}
	result.SelectedCount = len(sourceSkills)
	result.TotalAvailable = len(sourceSkills)

	if err := s.emitProgress(opts, ProgressEvent{
		Type:        ProgressEventStart,
		TotalSkills: len(sourceSkills),
		Message:     progressStartMessage(len(sourceSkills), "skills"),
	}); err != nil {
		return result, fmt.Errorf("progress callback failed: %w", err)
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
			logging.Error(
				"failed to get target path",
				logging.Platform(string(target)),
				slog.String("scope", string(opts.TargetScope)),
				logging.Err(err),
			)
			return result, fmt.Errorf("failed to get target path: %w", err)
		}
	}
	logging.Debug(
		"determined target path",
		logging.Path(targetPath),
		slog.String("scope", string(opts.TargetScope)),
	)

	// Parse existing target skills
	targetSkills, err := s.parseSkills(target, opts.TargetPath)
	if err != nil {
		logging.Debug(
			"target skills not found, nothing to delete",
			logging.Platform(string(target)),
			logging.Err(err),
		)
		return result, nil
	}

	logging.Debug(
		"parsed existing target skills",
		logging.Platform(string(target)),
		logging.Count(len(targetSkills)),
	)

	// Build a map of source skill names for quick lookup
	sourceSkillNames := make(map[string]model.Skill)
	for _, skill := range sourceSkills {
		sourceSkillNames[skill.Name] = skill
	}

	// Find target skills that match source skills and delete them
	for i, targetSkill := range targetSkills {
		sourceSkill, exists := sourceSkillNames[targetSkill.Name]
		if !exists {
			// Skill not in source list, skip it
			logging.Debug(
				"skill not in source list, skipping",
				logging.Skill(targetSkill.Name),
			)
			continue
		}

		if err := s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventSkillStart,
			Skill:           &sourceSkill,
			TotalSkills:     len(sourceSkills),
			ProcessedSkills: i,
			PercentComplete: progressPercent(i, len(sourceSkills)),
			Message:         fmt.Sprintf("Deleting %s", sourceSkill.Name),
		}); err != nil {
			return result, fmt.Errorf("progress callback failed: %w", err)
		}

		skillResult := SkillResult{
			Skill:      sourceSkill,
			TargetPath: targetSkill.Path,
		}

		// Delete the skill file
		if !opts.DryRun {
			if err := os.Remove(targetSkill.Path); err != nil {
				logging.Error(
					"failed to delete skill file",
					logging.Skill(targetSkill.Name),
					logging.Path(targetSkill.Path),
					logging.Err(err),
				)
				skillResult.Action = ActionFailed
				skillResult.Error = fmt.Errorf("failed to delete file: %w", err)
				result.Skills = append(result.Skills, skillResult)
				continue
			}

			// Try to remove parent directory if empty (for Codex SKILL.md files)
			parentDir := filepath.Dir(targetSkill.Path)
			if parentDir != targetPath {
				// Attempt to remove parent dir - will fail silently if not empty
				_ = os.Remove(parentDir)
			}

			logging.Debug(
				"deleted skill file",
				logging.Skill(targetSkill.Name),
				logging.Path(targetSkill.Path),
			)
		}

		skillResult.Action = ActionDeleted
		skillResult.Message = "deleted from target"
		result.Skills = append(result.Skills, skillResult)

		if err := s.emitProgress(opts, ProgressEvent{
			Type:            ProgressEventSkillComplete,
			Skill:           &sourceSkill,
			Action:          skillResult.Action,
			TotalSkills:     len(sourceSkills),
			ProcessedSkills: i + 1,
			PercentComplete: progressPercent(i+1, len(sourceSkills)),
			Message:         skillResult.Message,
			Error:           skillResult.Error,
		}); err != nil {
			return result, fmt.Errorf("progress callback failed: %w", err)
		}
	}

	logging.Debug(
		"delete sync operation completed",
		logging.Platform(string(target)),
		logging.Count(len(result.Skills)),
	)

	_ = s.emitProgress(opts, ProgressEvent{
		Type:            ProgressEventComplete,
		TotalSkills:     len(sourceSkills),
		ProcessedSkills: len(result.Skills),
		PercentComplete: 100,
		Message:         fmt.Sprintf("Delete sync completed: %d skills processed", len(result.Skills)),
	})

	return result, nil
}
