package sync

import (
	"fmt"
	"log/slog"

	"github.com/klauern/skillsync/internal/logging"
	"github.com/klauern/skillsync/internal/model"
)

// SyncDirection represents the direction to sync a skill.
type SyncDirection int

const (
	// SyncDirectionAtoB means sync from platform A to B.
	SyncDirectionAtoB SyncDirection = iota

	// SyncDirectionBtoA means sync from platform B to A.
	SyncDirectionBtoA

	// SyncDirectionConflict means there's a conflict requiring manual resolution.
	SyncDirectionConflict
)

// BidirectionalConflict represents a conflict in bidirectional sync.
type BidirectionalConflict struct {
	Name     string
	SkillA   model.Skill
	SkillB   model.Skill
	Conflict *Conflict
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

	skillsAMap := make(map[string]model.Skill)
	for _, skill := range skillsA {
		skillsAMap[skill.Name] = skill
	}

	skillsBMap := make(map[string]model.Skill)
	for _, skill := range skillsB {
		skillsBMap[skill.Name] = skill
	}

	var syncAtoB []model.Skill
	var syncBtoA []model.Skill
	var conflicts []BidirectionalConflict

	for _, skillA := range skillsA {
		skillB, existsInB := skillsBMap[skillA.Name]
		if !existsInB {
			syncAtoB = append(syncAtoB, skillA)
			continue
		}

		conflict := s.conflictDetector.DetectConflict(skillA, skillB)
		if conflict == nil {
			continue
		}

		switch s.determineSyncDirection(skillA, skillB, opts.Strategy) {
		case SyncDirectionAtoB:
			syncAtoB = append(syncAtoB, skillA)
		case SyncDirectionBtoA:
			syncBtoA = append(syncBtoA, skillB)
		default:
			conflicts = append(conflicts, BidirectionalConflict{
				Name:     skillA.Name,
				SkillA:   skillA,
				SkillB:   skillB,
				Conflict: conflict,
			})
		}
	}

	for _, skillB := range skillsB {
		if _, existsInA := skillsAMap[skillB.Name]; !existsInA {
			syncBtoA = append(syncBtoA, skillB)
		}
	}

	logging.Debug("determined sync operations",
		slog.Int("sync_a_to_b", len(syncAtoB)),
		slog.Int("sync_b_to_a", len(syncBtoA)),
		logging.Count(len(conflicts)),
	)

	if err := s.emitProgress(opts, ProgressEvent{
		Type:        ProgressEventStart,
		TotalSkills: len(syncAtoB) + len(syncBtoA),
		Message:     fmt.Sprintf("Starting bidirectional sync between %s and %s", platformA, platformB),
	}); err != nil {
		return biResult, fmt.Errorf("progress callback failed: %w", err)
	}

	if len(syncAtoB) > 0 {
		optsAtoB := opts
		optsAtoB.SourcePath = opts.SourcePath
		optsAtoB.TargetPath = opts.TargetPath
		resultAtoB, err := s.SyncWithSkills(syncAtoB, platformB, optsAtoB)
		if err != nil {
			logging.Error("failed to sync A to B", logging.Err(err))
			return biResult, fmt.Errorf("failed to sync A to B: %w", err)
		}
		biResult.ResultAtoB = resultAtoB
	}

	if len(syncBtoA) > 0 {
		optsBtoA := opts
		optsBtoA.SourcePath = opts.TargetPath
		optsBtoA.TargetPath = opts.SourcePath
		resultBtoA, err := s.SyncWithSkills(syncBtoA, platformA, optsBtoA)
		if err != nil {
			logging.Error("failed to sync B to A", logging.Err(err))
			return biResult, fmt.Errorf("failed to sync B to A: %w", err)
		}
		biResult.ResultBtoA = resultBtoA
	}

	biResult.Conflicts = conflicts

	logging.Debug("bidirectional sync completed",
		slog.String("platform_a", string(platformA)),
		slog.String("platform_b", string(platformB)),
		slog.Int("synced_a_to_b", len(syncAtoB)),
		slog.Int("synced_b_to_a", len(syncBtoA)),
		logging.Count(len(conflicts)),
	)

	_ = s.emitProgress(opts, ProgressEvent{
		Type:            ProgressEventComplete,
		TotalSkills:     len(syncAtoB) + len(syncBtoA),
		ProcessedSkills: len(syncAtoB) + len(syncBtoA),
		PercentComplete:  100,
		Message:         fmt.Sprintf("Bidirectional sync completed: %d operations", len(syncAtoB)+len(syncBtoA)),
	})

	return biResult, nil
}

// determineSyncDirection decides which direction to sync based on strategy.
func (s *Synchronizer) determineSyncDirection(skillA, skillB model.Skill, strategy Strategy) SyncDirection {
	switch strategy {
	case StrategyNewer:
		if skillA.ModifiedAt.After(skillB.ModifiedAt) {
			return SyncDirectionAtoB
		}
		if skillB.ModifiedAt.After(skillA.ModifiedAt) {
			return SyncDirectionBtoA
		}
		return SyncDirectionConflict
	case StrategyOverwrite:
		return SyncDirectionAtoB
	case StrategyThreeWay, StrategyInteractive, StrategyMerge, StrategySkip:
		return SyncDirectionConflict
	default:
		return SyncDirectionConflict
	}
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

// TotalChanged returns the total number of skills changed in both directions.
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

// Summary generates a human-readable summary of bidirectional sync results.
func (r *BidirectionalResult) Summary() string {
	var summary string
	if r.DryRun {
		summary = fmt.Sprintf("Bidirectional sync preview: %s <-> %s (strategy: %s)\n\n", r.PlatformA, r.PlatformB, r.Strategy)
	} else {
		summary = fmt.Sprintf("Bidirectional sync: %s <-> %s (strategy: %s)\n\n", r.PlatformA, r.PlatformB, r.Strategy)
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
