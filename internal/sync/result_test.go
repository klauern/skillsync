package sync

import (
	"errors"
	"strings"
	"testing"

	"github.com/klauern/skillsync/internal/model"
)

func TestAction_Constants(t *testing.T) {
	tests := []struct {
		name   string
		action Action
		want   string
	}{
		{"created action", ActionCreated, "created"},
		{"updated action", ActionUpdated, "updated"},
		{"skipped action", ActionSkipped, "skipped"},
		{"merged action", ActionMerged, "merged"},
		{"failed action", ActionFailed, "failed"},
		{"conflict action", ActionConflict, "conflict"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.action) != tt.want {
				t.Errorf("Action = %q, want %q", tt.action, tt.want)
			}
		})
	}
}

func TestSkillResult_Success(t *testing.T) {
	tests := []struct {
		name   string
		action Action
		want   bool
	}{
		{"created is success", ActionCreated, true},
		{"updated is success", ActionUpdated, true},
		{"skipped is success", ActionSkipped, true},
		{"merged is success", ActionMerged, true},
		{"conflict is success", ActionConflict, true},
		{"failed is not success", ActionFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := &SkillResult{Action: tt.action}
			if got := sr.Success(); got != tt.want {
				t.Errorf("SkillResult.Success() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSkillResult_Fields(t *testing.T) {
	skill := model.Skill{Name: "test-skill", Content: "test content"}
	conflict := &Conflict{SkillName: "test-skill", Type: ConflictTypeContent}
	testErr := errors.New("test error")

	sr := &SkillResult{
		Skill:      skill,
		Action:     ActionFailed,
		TargetPath: "/path/to/target",
		Error:      testErr,
		Message:    "test message",
		Conflict:   conflict,
	}

	if sr.Skill.Name != "test-skill" {
		t.Errorf("Skill.Name = %q, want %q", sr.Skill.Name, "test-skill")
	}
	if sr.Action != ActionFailed {
		t.Errorf("Action = %q, want %q", sr.Action, ActionFailed)
	}
	if sr.TargetPath != "/path/to/target" {
		t.Errorf("TargetPath = %q, want %q", sr.TargetPath, "/path/to/target")
	}
	if !errors.Is(sr.Error, testErr) {
		t.Errorf("Error = %v, want %v", sr.Error, testErr)
	}
	if sr.Message != "test message" {
		t.Errorf("Message = %q, want %q", sr.Message, "test message")
	}
	if sr.Conflict != conflict {
		t.Errorf("Conflict = %v, want %v", sr.Conflict, conflict)
	}
}

func TestResult_Created(t *testing.T) {
	result := &Result{
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "created1"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "updated1"}, Action: ActionUpdated},
			{Skill: model.Skill{Name: "created2"}, Action: ActionCreated},
		},
	}

	created := result.Created()
	if len(created) != 2 {
		t.Errorf("Created() returned %d items, want 2", len(created))
	}
	if created[0].Skill.Name != "created1" || created[1].Skill.Name != "created2" {
		t.Error("Created() returned wrong skills")
	}
}

func TestResult_Updated(t *testing.T) {
	result := &Result{
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "updated1"}, Action: ActionUpdated},
			{Skill: model.Skill{Name: "created1"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "updated2"}, Action: ActionUpdated},
		},
	}

	updated := result.Updated()
	if len(updated) != 2 {
		t.Errorf("Updated() returned %d items, want 2", len(updated))
	}
	if updated[0].Skill.Name != "updated1" || updated[1].Skill.Name != "updated2" {
		t.Error("Updated() returned wrong skills")
	}
}

func TestResult_Skipped(t *testing.T) {
	result := &Result{
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "skipped1"}, Action: ActionSkipped},
			{Skill: model.Skill{Name: "created1"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "skipped2"}, Action: ActionSkipped},
		},
	}

	skipped := result.Skipped()
	if len(skipped) != 2 {
		t.Errorf("Skipped() returned %d items, want 2", len(skipped))
	}
	if skipped[0].Skill.Name != "skipped1" || skipped[1].Skill.Name != "skipped2" {
		t.Error("Skipped() returned wrong skills")
	}
}

func TestResult_Merged(t *testing.T) {
	result := &Result{
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "merged1"}, Action: ActionMerged},
			{Skill: model.Skill{Name: "created1"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "merged2"}, Action: ActionMerged},
		},
	}

	merged := result.Merged()
	if len(merged) != 2 {
		t.Errorf("Merged() returned %d items, want 2", len(merged))
	}
	if merged[0].Skill.Name != "merged1" || merged[1].Skill.Name != "merged2" {
		t.Error("Merged() returned wrong skills")
	}
}

func TestResult_Failed(t *testing.T) {
	result := &Result{
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "failed1"}, Action: ActionFailed},
			{Skill: model.Skill{Name: "created1"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "failed2"}, Action: ActionFailed},
		},
	}

	failed := result.Failed()
	if len(failed) != 2 {
		t.Errorf("Failed() returned %d items, want 2", len(failed))
	}
	if failed[0].Skill.Name != "failed1" || failed[1].Skill.Name != "failed2" {
		t.Error("Failed() returned wrong skills")
	}
}

func TestResult_Conflicts(t *testing.T) {
	result := &Result{
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "conflict1"}, Action: ActionConflict},
			{Skill: model.Skill{Name: "created1"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "conflict2"}, Action: ActionConflict},
		},
	}

	conflicts := result.Conflicts()
	if len(conflicts) != 2 {
		t.Errorf("Conflicts() returned %d items, want 2", len(conflicts))
	}
	if conflicts[0].Skill.Name != "conflict1" || conflicts[1].Skill.Name != "conflict2" {
		t.Error("Conflicts() returned wrong skills")
	}
}

func TestResult_FilterByAction_Empty(t *testing.T) {
	result := &Result{Skills: []SkillResult{}}

	if len(result.Created()) != 0 {
		t.Error("Created() on empty result should return empty slice")
	}
	if len(result.Updated()) != 0 {
		t.Error("Updated() on empty result should return empty slice")
	}
	if len(result.Skipped()) != 0 {
		t.Error("Skipped() on empty result should return empty slice")
	}
	if len(result.Merged()) != 0 {
		t.Error("Merged() on empty result should return empty slice")
	}
	if len(result.Failed()) != 0 {
		t.Error("Failed() on empty result should return empty slice")
	}
	if len(result.Conflicts()) != 0 {
		t.Error("Conflicts() on empty result should return empty slice")
	}
}

func TestResult_FilterByAction_NoMatch(t *testing.T) {
	result := &Result{
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "created1"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "created2"}, Action: ActionCreated},
		},
	}

	if len(result.Updated()) != 0 {
		t.Error("Updated() should return empty when no matches")
	}
	if len(result.Failed()) != 0 {
		t.Error("Failed() should return empty when no matches")
	}
}

func TestResult_HasConflicts(t *testing.T) {
	tests := []struct {
		name   string
		skills []SkillResult
		want   bool
	}{
		{
			name:   "no skills",
			skills: []SkillResult{},
			want:   false,
		},
		{
			name: "no conflicts",
			skills: []SkillResult{
				{Action: ActionCreated},
				{Action: ActionUpdated},
			},
			want: false,
		},
		{
			name: "has conflicts",
			skills: []SkillResult{
				{Action: ActionCreated},
				{Action: ActionConflict},
			},
			want: true,
		},
		{
			name: "multiple conflicts",
			skills: []SkillResult{
				{Action: ActionConflict},
				{Action: ActionConflict},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{Skills: tt.skills}
			if got := r.HasConflicts(); got != tt.want {
				t.Errorf("HasConflicts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResult_Success(t *testing.T) {
	tests := []struct {
		name   string
		skills []SkillResult
		want   bool
	}{
		{
			name:   "empty skills is success",
			skills: []SkillResult{},
			want:   true,
		},
		{
			name: "all success actions",
			skills: []SkillResult{
				{Action: ActionCreated},
				{Action: ActionUpdated},
				{Action: ActionSkipped},
				{Action: ActionMerged},
			},
			want: true,
		},
		{
			name: "conflicts still success",
			skills: []SkillResult{
				{Action: ActionCreated},
				{Action: ActionConflict},
			},
			want: true,
		},
		{
			name: "one failed is not success",
			skills: []SkillResult{
				{Action: ActionCreated},
				{Action: ActionFailed},
			},
			want: false,
		},
		{
			name: "all failed",
			skills: []SkillResult{
				{Action: ActionFailed},
				{Action: ActionFailed},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{Skills: tt.skills}
			if got := r.Success(); got != tt.want {
				t.Errorf("Success() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResult_TotalProcessed(t *testing.T) {
	tests := []struct {
		name   string
		skills []SkillResult
		want   int
	}{
		{
			name:   "empty",
			skills: []SkillResult{},
			want:   0,
		},
		{
			name: "single skill",
			skills: []SkillResult{
				{Action: ActionCreated},
			},
			want: 1,
		},
		{
			name: "multiple skills",
			skills: []SkillResult{
				{Action: ActionCreated},
				{Action: ActionUpdated},
				{Action: ActionSkipped},
				{Action: ActionFailed},
				{Action: ActionConflict},
			},
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{Skills: tt.skills}
			if got := r.TotalProcessed(); got != tt.want {
				t.Errorf("TotalProcessed() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResult_TotalChanged(t *testing.T) {
	tests := []struct {
		name   string
		skills []SkillResult
		want   int
	}{
		{
			name:   "empty",
			skills: []SkillResult{},
			want:   0,
		},
		{
			name: "only created",
			skills: []SkillResult{
				{Action: ActionCreated},
				{Action: ActionCreated},
			},
			want: 2,
		},
		{
			name: "only updated",
			skills: []SkillResult{
				{Action: ActionUpdated},
			},
			want: 1,
		},
		{
			name: "only merged",
			skills: []SkillResult{
				{Action: ActionMerged},
				{Action: ActionMerged},
			},
			want: 2,
		},
		{
			name: "created updated merged combined",
			skills: []SkillResult{
				{Action: ActionCreated},
				{Action: ActionUpdated},
				{Action: ActionMerged},
			},
			want: 3,
		},
		{
			name: "skipped and failed not counted",
			skills: []SkillResult{
				{Action: ActionCreated},
				{Action: ActionSkipped},
				{Action: ActionFailed},
				{Action: ActionConflict},
			},
			want: 1,
		},
		{
			name: "all non-changed actions",
			skills: []SkillResult{
				{Action: ActionSkipped},
				{Action: ActionFailed},
				{Action: ActionConflict},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{Skills: tt.skills}
			if got := r.TotalChanged(); got != tt.want {
				t.Errorf("TotalChanged() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResult_Summary_Basic(t *testing.T) {
	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Cursor,
		Strategy: StrategyOverwrite,
		DryRun:   false,
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "skill1"}, Action: ActionCreated},
		},
	}

	summary := result.Summary()

	if !strings.Contains(summary, "claude-code") {
		t.Error("Summary should contain source platform")
	}
	if !strings.Contains(summary, "cursor") {
		t.Error("Summary should contain target platform")
	}
	if !strings.Contains(summary, "overwrite") {
		t.Error("Summary should contain strategy")
	}
	if !strings.Contains(summary, "Created:   1") {
		t.Error("Summary should show created count")
	}
	if strings.Contains(summary, "Dry run") {
		t.Error("Summary should not contain dry run when DryRun is false")
	}
}

func TestResult_Summary_DryRun(t *testing.T) {
	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Cursor,
		Strategy: StrategyOverwrite,
		DryRun:   true,
		Skills:   []SkillResult{},
	}

	summary := result.Summary()

	if !strings.Contains(summary, "Dry run - no changes made") {
		t.Error("Summary should indicate dry run")
	}
}

func TestResult_Summary_WithConflicts(t *testing.T) {
	conflict := &Conflict{
		SkillName: "conflicted-skill",
		Type:      ConflictTypeContent,
	}

	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Cursor,
		Strategy: StrategyThreeWay,
		DryRun:   false,
		Skills: []SkillResult{
			{
				Skill:    model.Skill{Name: "conflicted-skill"},
				Action:   ActionConflict,
				Conflict: conflict,
			},
		},
	}

	summary := result.Summary()

	if !strings.Contains(summary, "Conflicts: 1") {
		t.Error("Summary should show conflict count")
	}
	if !strings.Contains(summary, "Conflicts requiring resolution") {
		t.Error("Summary should have conflicts section")
	}
	if !strings.Contains(summary, "conflicted-skill") {
		t.Error("Summary should list conflicted skill name")
	}
	if !strings.Contains(summary, "content differs") {
		t.Error("Summary should include conflict summary")
	}
}

func TestResult_Summary_ConflictWithoutConflictDetails(t *testing.T) {
	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Cursor,
		Strategy: StrategyThreeWay,
		DryRun:   false,
		Skills: []SkillResult{
			{
				Skill:    model.Skill{Name: "conflicted-skill"},
				Action:   ActionConflict,
				Conflict: nil, // No conflict details
			},
		},
	}

	summary := result.Summary()

	if !strings.Contains(summary, "conflicted-skill") {
		t.Error("Summary should list conflicted skill even without conflict details")
	}
}

func TestResult_Summary_WithFailures(t *testing.T) {
	testErr := errors.New("test error message")

	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Cursor,
		Strategy: StrategyOverwrite,
		DryRun:   false,
		Skills: []SkillResult{
			{
				Skill:  model.Skill{Name: "failed-skill"},
				Action: ActionFailed,
				Error:  testErr,
			},
		},
	}

	summary := result.Summary()

	if !strings.Contains(summary, "Failed:    1") {
		t.Error("Summary should show failed count")
	}
	if !strings.Contains(summary, "Errors:") {
		t.Error("Summary should have errors section")
	}
	if !strings.Contains(summary, "failed-skill") {
		t.Error("Summary should list failed skill name")
	}
	if !strings.Contains(summary, "test error message") {
		t.Error("Summary should include error message")
	}
}

func TestResult_Summary_AllActionTypes(t *testing.T) {
	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Codex,
		Strategy: StrategyMerge,
		DryRun:   false,
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "created"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "updated"}, Action: ActionUpdated},
			{Skill: model.Skill{Name: "skipped"}, Action: ActionSkipped},
			{Skill: model.Skill{Name: "merged"}, Action: ActionMerged},
			{Skill: model.Skill{Name: "conflict"}, Action: ActionConflict, Conflict: &Conflict{SkillName: "conflict", Type: ConflictTypeMetadata}},
			{Skill: model.Skill{Name: "failed"}, Action: ActionFailed, Error: errors.New("error")},
		},
	}

	summary := result.Summary()

	if !strings.Contains(summary, "Created:   1") {
		t.Error("Summary should show created count")
	}
	if !strings.Contains(summary, "Updated:   1") {
		t.Error("Summary should show updated count")
	}
	if !strings.Contains(summary, "Skipped:   1") {
		t.Error("Summary should show skipped count")
	}
	if !strings.Contains(summary, "Merged:    1") {
		t.Error("Summary should show merged count")
	}
	if !strings.Contains(summary, "Conflicts: 1") {
		t.Error("Summary should show conflict count")
	}
	if !strings.Contains(summary, "Failed:    1") {
		t.Error("Summary should show failed count")
	}
}

func TestResult_Summary_EmptyResult(t *testing.T) {
	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Cursor,
		Strategy: StrategyOverwrite,
		DryRun:   false,
		Skills:   []SkillResult{},
	}

	summary := result.Summary()

	if !strings.Contains(summary, "Created:   0") {
		t.Error("Summary should show zero counts for empty result")
	}
	if strings.Contains(summary, "Conflicts requiring resolution") {
		t.Error("Summary should not have conflicts section when none exist")
	}
	if strings.Contains(summary, "Errors:") {
		t.Error("Summary should not have errors section when none exist")
	}
}

func TestResult_Summary_AllStrategies(t *testing.T) {
	strategies := []Strategy{
		StrategyOverwrite,
		StrategySkip,
		StrategyNewer,
		StrategyMerge,
		StrategyThreeWay,
		StrategyInteractive,
	}

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			result := &Result{
				Source:   model.ClaudeCode,
				Target:   model.Cursor,
				Strategy: strategy,
				Skills:   []SkillResult{},
			}

			summary := result.Summary()

			if !strings.Contains(summary, string(strategy)) {
				t.Errorf("Summary should contain strategy %q", strategy)
			}
		})
	}
}

func TestResult_Summary_AllPlatformCombinations(t *testing.T) {
	platforms := []model.Platform{model.ClaudeCode, model.Cursor, model.Codex}

	for _, source := range platforms {
		for _, target := range platforms {
			if source == target {
				continue
			}
			t.Run(string(source)+"->"+string(target), func(t *testing.T) {
				result := &Result{
					Source:   source,
					Target:   target,
					Strategy: StrategyOverwrite,
					Skills:   []SkillResult{},
				}

				summary := result.Summary()

				if !strings.Contains(summary, string(source)) {
					t.Errorf("Summary should contain source platform %q", source)
				}
				if !strings.Contains(summary, string(target)) {
					t.Errorf("Summary should contain target platform %q", target)
				}
			})
		}
	}
}

func TestResult_Summary_MultipleConflicts(t *testing.T) {
	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Cursor,
		Strategy: StrategyThreeWay,
		Skills: []SkillResult{
			{
				Skill:    model.Skill{Name: "conflict1"},
				Action:   ActionConflict,
				Conflict: &Conflict{SkillName: "conflict1", Type: ConflictTypeContent},
			},
			{
				Skill:    model.Skill{Name: "conflict2"},
				Action:   ActionConflict,
				Conflict: &Conflict{SkillName: "conflict2", Type: ConflictTypeMetadata},
			},
			{
				Skill:    model.Skill{Name: "conflict3"},
				Action:   ActionConflict,
				Conflict: &Conflict{SkillName: "conflict3", Type: ConflictTypeBoth},
			},
		},
	}

	summary := result.Summary()

	if !strings.Contains(summary, "Conflicts: 3") {
		t.Error("Summary should show 3 conflicts")
	}
	if !strings.Contains(summary, "conflict1") {
		t.Error("Summary should list conflict1")
	}
	if !strings.Contains(summary, "conflict2") {
		t.Error("Summary should list conflict2")
	}
	if !strings.Contains(summary, "conflict3") {
		t.Error("Summary should list conflict3")
	}
	if !strings.Contains(summary, "content differs") {
		t.Error("Summary should include content differs")
	}
	if !strings.Contains(summary, "metadata differs") {
		t.Error("Summary should include metadata differs")
	}
	if !strings.Contains(summary, "content and metadata differ") {
		t.Error("Summary should include content and metadata differ")
	}
}

func TestResult_Summary_MultipleFailures(t *testing.T) {
	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Cursor,
		Strategy: StrategyOverwrite,
		Skills: []SkillResult{
			{
				Skill:  model.Skill{Name: "failed1"},
				Action: ActionFailed,
				Error:  errors.New("error one"),
			},
			{
				Skill:  model.Skill{Name: "failed2"},
				Action: ActionFailed,
				Error:  errors.New("error two"),
			},
		},
	}

	summary := result.Summary()

	if !strings.Contains(summary, "Failed:    2") {
		t.Error("Summary should show 2 failures")
	}
	if !strings.Contains(summary, "failed1") {
		t.Error("Summary should list failed1")
	}
	if !strings.Contains(summary, "failed2") {
		t.Error("Summary should list failed2")
	}
	if !strings.Contains(summary, "error one") {
		t.Error("Summary should include error one")
	}
	if !strings.Contains(summary, "error two") {
		t.Error("Summary should include error two")
	}
}

func TestResult_filterByAction(t *testing.T) {
	result := &Result{
		Skills: []SkillResult{
			{Skill: model.Skill{Name: "a"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "b"}, Action: ActionCreated},
			{Skill: model.Skill{Name: "c"}, Action: ActionUpdated},
		},
	}

	created := result.filterByAction(ActionCreated)
	if len(created) != 2 {
		t.Errorf("filterByAction(ActionCreated) = %d, want 2", len(created))
	}

	updated := result.filterByAction(ActionUpdated)
	if len(updated) != 1 {
		t.Errorf("filterByAction(ActionUpdated) = %d, want 1", len(updated))
	}

	failed := result.filterByAction(ActionFailed)
	if len(failed) != 0 {
		t.Errorf("filterByAction(ActionFailed) = %d, want 0", len(failed))
	}
}

func TestResult_NilSkillSlice(t *testing.T) {
	result := &Result{
		Source:   model.ClaudeCode,
		Target:   model.Cursor,
		Strategy: StrategyOverwrite,
		Skills:   nil,
	}

	if result.TotalProcessed() != 0 {
		t.Error("TotalProcessed() on nil skills should return 0")
	}
	if result.TotalChanged() != 0 {
		t.Error("TotalChanged() on nil skills should return 0")
	}
	if !result.Success() {
		t.Error("Success() on nil skills should return true")
	}
	if result.HasConflicts() {
		t.Error("HasConflicts() on nil skills should return false")
	}
	if len(result.Created()) != 0 {
		t.Error("Created() on nil skills should return empty slice")
	}
}
