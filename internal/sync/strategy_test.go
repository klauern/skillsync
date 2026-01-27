package sync

import "testing"

func TestStrategyIsValid(t *testing.T) {
	tests := map[string]struct {
		strategy Strategy
		valid    bool
	}{
		"overwrite valid":    {strategy: StrategyOverwrite, valid: true},
		"skip valid":         {strategy: StrategySkip, valid: true},
		"newer valid":        {strategy: StrategyNewer, valid: true},
		"merge valid":        {strategy: StrategyMerge, valid: true},
		"three-way valid":    {strategy: StrategyThreeWay, valid: true},
		"interactive valid":  {strategy: StrategyInteractive, valid: true},
		"empty invalid":      {strategy: "", valid: false},
		"unknown invalid":    {strategy: "unknown", valid: false},
		"uppercase invalid":  {strategy: "OVERWRITE", valid: false},
		"mixed case invalid": {strategy: "Overwrite", valid: false},
		"whitespace invalid": {strategy: " overwrite", valid: false},
		"similar invalid":    {strategy: "override", valid: false},
		"suffix invalid":     {strategy: "overwrite2", valid: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.strategy.IsValid()
			if got != tt.valid {
				t.Errorf("Strategy(%q).IsValid() = %v, want %v",
					tt.strategy, got, tt.valid)
			}
		})
	}
}

func TestAllStrategies(t *testing.T) {
	strategies := AllStrategies()

	t.Run("returns correct count", func(t *testing.T) {
		if len(strategies) != 6 {
			t.Errorf("AllStrategies() returned %d strategies, want 6", len(strategies))
		}
	})

	t.Run("all strategies are valid", func(t *testing.T) {
		for _, s := range strategies {
			if !s.IsValid() {
				t.Errorf("AllStrategies() returned invalid strategy: %q", s)
			}
		}
	})

	t.Run("returns correct order", func(t *testing.T) {
		expectedOrder := []Strategy{
			StrategyOverwrite,
			StrategySkip,
			StrategyNewer,
			StrategyMerge,
			StrategyThreeWay,
			StrategyInteractive,
		}
		for i, s := range strategies {
			if s != expectedOrder[i] {
				t.Errorf("AllStrategies()[%d] = %q, want %q", i, s, expectedOrder[i])
			}
		}
	})

	t.Run("no duplicates", func(t *testing.T) {
		seen := make(map[Strategy]bool)
		for _, s := range strategies {
			if seen[s] {
				t.Errorf("AllStrategies() contains duplicate: %q", s)
			}
			seen[s] = true
		}
	})
}

func TestStrategyString(t *testing.T) {
	tests := map[string]struct {
		strategy Strategy
		want     string
	}{
		"overwrite":   {strategy: StrategyOverwrite, want: "overwrite"},
		"skip":        {strategy: StrategySkip, want: "skip"},
		"newer":       {strategy: StrategyNewer, want: "newer"},
		"merge":       {strategy: StrategyMerge, want: "merge"},
		"three-way":   {strategy: StrategyThreeWay, want: "three-way"},
		"interactive": {strategy: StrategyInteractive, want: "interactive"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.strategy.String()
			if got != tt.want {
				t.Errorf("Strategy(%q).String() = %q, want %q",
					tt.strategy, got, tt.want)
			}
		})
	}
}

func TestStrategyDescription(t *testing.T) {
	t.Run("all valid strategies have non-empty descriptions", func(t *testing.T) {
		for _, strategy := range AllStrategies() {
			desc := strategy.Description()
			if desc == "" {
				t.Errorf("Strategy(%q).Description() should not be empty", strategy)
			}
			if desc == "Unknown strategy" {
				t.Errorf("Strategy(%q).Description() should not return 'Unknown strategy'", strategy)
			}
		}
	})

	t.Run("unknown strategy returns Unknown strategy", func(t *testing.T) {
		unknown := Strategy("invalid")
		got := unknown.Description()
		if got != "Unknown strategy" {
			t.Errorf("Strategy(%q).Description() = %q, want %q",
				unknown, got, "Unknown strategy")
		}
	})

	t.Run("empty strategy returns Unknown strategy", func(t *testing.T) {
		empty := Strategy("")
		got := empty.Description()
		if got != "Unknown strategy" {
			t.Errorf("Strategy(%q).Description() = %q, want %q",
				empty, got, "Unknown strategy")
		}
	})

	t.Run("descriptions match expected content", func(t *testing.T) {
		tests := map[string]struct {
			strategy Strategy
			contains string
		}{
			"overwrite contains replace":    {strategy: StrategyOverwrite, contains: "Replace"},
			"skip contains skip":            {strategy: StrategySkip, contains: "Skip"},
			"newer contains newer":          {strategy: StrategyNewer, contains: "newer"},
			"merge contains merge":          {strategy: StrategyMerge, contains: "Merge"},
			"three-way contains merge":      {strategy: StrategyThreeWay, contains: "merge"},
			"interactive contains conflict": {strategy: StrategyInteractive, contains: "conflict"},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				desc := tt.strategy.Description()
				if !containsIgnoreCase(desc, tt.contains) {
					t.Errorf("Strategy(%q).Description() = %q, expected to contain %q",
						tt.strategy, desc, tt.contains)
				}
			})
		}
	})
}

func TestStrategyConstants(t *testing.T) {
	t.Run("constants have expected string values", func(t *testing.T) {
		tests := map[string]struct {
			strategy Strategy
			value    string
		}{
			"overwrite":   {strategy: StrategyOverwrite, value: "overwrite"},
			"skip":        {strategy: StrategySkip, value: "skip"},
			"newer":       {strategy: StrategyNewer, value: "newer"},
			"merge":       {strategy: StrategyMerge, value: "merge"},
			"three-way":   {strategy: StrategyThreeWay, value: "three-way"},
			"interactive": {strategy: StrategyInteractive, value: "interactive"},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				if string(tt.strategy) != tt.value {
					t.Errorf("strategy constant %s = %q, want %q",
						name, string(tt.strategy), tt.value)
				}
			})
		}
	})
}

func TestStrategyConsistency(t *testing.T) {
	t.Run("all AllStrategies pass IsValid", func(t *testing.T) {
		for _, s := range AllStrategies() {
			if !s.IsValid() {
				t.Errorf("Strategy(%q) from AllStrategies() failed IsValid()", s)
			}
		}
	})

	t.Run("String roundtrip matches constant", func(t *testing.T) {
		for _, s := range AllStrategies() {
			str := s.String()
			recreated := Strategy(str)
			if recreated != s {
				t.Errorf("Strategy roundtrip failed: %q -> String() -> %q -> Strategy = %q",
					s, str, recreated)
			}
		}
	})
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if equalIgnoreCase(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

// equalIgnoreCase compares two strings case-insensitively.
func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		if toLower(a[i]) != toLower(b[i]) {
			return false
		}
	}
	return true
}

// toLower converts a single byte to lowercase.
func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
