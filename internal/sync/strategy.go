// Package sync implements skill synchronization logic across platforms.
package sync

// Strategy defines the behavior for handling skill conflicts during sync.
type Strategy string

const (
	// StrategyOverwrite replaces target skills with source skills unconditionally.
	StrategyOverwrite Strategy = "overwrite"

	// StrategySkip skips skills that already exist in the target.
	StrategySkip Strategy = "skip"

	// StrategyNewer copies only if source skill is newer than target.
	StrategyNewer Strategy = "newer"

	// StrategyMerge attempts to merge skills (content concatenation with headers).
	StrategyMerge Strategy = "merge"
)

// IsValid returns true if the strategy is recognized.
func (s Strategy) IsValid() bool {
	switch s {
	case StrategyOverwrite, StrategySkip, StrategyNewer, StrategyMerge:
		return true
	default:
		return false
	}
}

// AllStrategies returns all supported sync strategies.
func AllStrategies() []Strategy {
	return []Strategy{StrategyOverwrite, StrategySkip, StrategyNewer, StrategyMerge}
}

// String returns the string representation of the strategy.
func (s Strategy) String() string {
	return string(s)
}

// Description returns a human-readable description of the strategy.
func (s Strategy) Description() string {
	switch s {
	case StrategyOverwrite:
		return "Replace target skills with source skills unconditionally"
	case StrategySkip:
		return "Skip skills that already exist in target"
	case StrategyNewer:
		return "Copy only if source is newer than target"
	case StrategyMerge:
		return "Merge source and target content"
	default:
		return "Unknown strategy"
	}
}
