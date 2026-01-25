package sync

import (
	"testing"
)

func TestStrategy_IsValid(t *testing.T) {
	tests := []struct {
		strategy Strategy
		valid    bool
	}{
		{StrategyOverwrite, true},
		{StrategySkip, true},
		{StrategyNewer, true},
		{StrategyMerge, true},
		{Strategy("invalid"), false},
		{Strategy(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.strategy), func(t *testing.T) {
			if got := tt.strategy.IsValid(); got != tt.valid {
				t.Errorf("Strategy(%q).IsValid() = %v, want %v", tt.strategy, got, tt.valid)
			}
		})
	}
}

func TestAllStrategies(t *testing.T) {
	strategies := AllStrategies()

	if len(strategies) != 4 {
		t.Errorf("Expected 4 strategies, got %d", len(strategies))
	}

	// Verify all returned strategies are valid
	for _, s := range strategies {
		if !s.IsValid() {
			t.Errorf("AllStrategies() returned invalid strategy: %s", s)
		}
	}
}

func TestStrategy_String(t *testing.T) {
	tests := []struct {
		strategy Strategy
		expected string
	}{
		{StrategyOverwrite, "overwrite"},
		{StrategySkip, "skip"},
		{StrategyNewer, "newer"},
		{StrategyMerge, "merge"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.strategy.String(); got != tt.expected {
				t.Errorf("Strategy.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStrategy_Description(t *testing.T) {
	tests := []struct {
		strategy    Strategy
		notEmpty    bool
		containsKey string
	}{
		{StrategyOverwrite, true, "Replace"},
		{StrategySkip, true, "Skip"},
		{StrategyNewer, true, "newer"},
		{StrategyMerge, true, "Merge"},
		{Strategy("unknown"), true, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.strategy), func(t *testing.T) {
			desc := tt.strategy.Description()
			if tt.notEmpty && desc == "" {
				t.Error("Description should not be empty")
			}
		})
	}
}
