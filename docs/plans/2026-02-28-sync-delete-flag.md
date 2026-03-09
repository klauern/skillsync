# `--delete` Flag for Sync Command Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `--delete` flag to `skillsync sync` that removes orphaned target skills (skills in target but not in source) after syncing, with interactive selection in both CLI and TUI paths.

**Architecture:** Post-sync delete phase. After the normal sync completes, detect orphaned skills by comparing target skills against source skills. In CLI mode, prompt per-orphan (interactive strategy) or bulk-confirm. In TUI mode, launch `RunDeleteList` pre-populated with orphans.

**Tech Stack:** Go 1.25.4, urfave/cli v3, BubbleTea TUI

---

## Task 1: Add `--delete` flag and `syncConfig` field

**Files:**

- Modify: `internal/cli/commands.go:809-848` (syncFlags function)
- Modify: `internal/cli/commands.go:1069-1081` (syncConfig struct)
- Modify: `internal/cli/commands.go:1122-1134` (parseSyncConfig return)

**Step 1: Add the flag to `syncFlags()`**

In `internal/cli/commands.go`, inside the `syncFlags()` function (line 809), add a new `BoolFlag` after the existing `include-prompts` flag (line 846):

```go
&cli.BoolFlag{
    Name:  "delete",
    Usage: "After sync, delete target skills not present in source (orphan cleanup)",
},
```

**Step 2: Add field to `syncConfig`**

In `internal/cli/commands.go`, add `deleteOrphans bool` to the `syncConfig` struct (line 1069):

```go
type syncConfig struct {
    sourceSpec     model.PlatformSpec
    targetSpec     model.PlatformSpec
    dryRun         bool
    strategy       sync.Strategy
    skipBackup     bool
    skipValidation bool
    yesFlag        bool
    deleteMode     bool
    deleteOrphans  bool  // NEW: --delete flag for post-sync orphan cleanup
    includePlugins bool
    typeFilter     []model.SkillType
    sourceSkills   []model.Skill
}
```

**Step 3: Wire up in `parseSyncConfig`**

In `internal/cli/commands.go`, in the return statement of `parseSyncConfig` (line 1122), add:

```go
deleteOrphans:  cmd.Bool("delete"),
```

**Step 4: Update sync command description**

In `internal/cli/commands.go`, in the `syncCommand()` Description (line 856), add an example:

```text
    skillsync sync --delete cursor claudecode          # Sync and remove orphaned skills from target
```

**Step 5: Commit**

```bash
git add internal/cli/commands.go
git commit -m "feat(sync): add --delete flag definition to sync command"
```

---

## Task 2: Implement orphan detection function

**Files:**

- Modify: `internal/cli/commands.go` (add new function near `filterDeleteCandidates` at line 1339)
- Create: `internal/cli/orphan_detection_test.go`

**Step 1: Write the failing test**

Create `internal/cli/orphan_detection_test.go`:

```go
package cli

import (
    "testing"

    "github.com/klauern/skillsync/internal/model"
)

func TestFindOrphanedSkills(t *testing.T) {
    tests := map[string]struct {
        sourceSkills []model.Skill
        targetSkills []model.Skill
        wantOrphans  int
        wantNames    []string
    }{
        "no orphans when target subset of source": {
            sourceSkills: []model.Skill{
                {Name: "skill-a"},
                {Name: "skill-b"},
            },
            targetSkills: []model.Skill{
                {Name: "skill-a"},
            },
            wantOrphans: 0,
        },
        "finds orphans in target not in source": {
            sourceSkills: []model.Skill{
                {Name: "skill-a"},
            },
            targetSkills: []model.Skill{
                {Name: "skill-a"},
                {Name: "skill-b"},
                {Name: "skill-c"},
            },
            wantOrphans: 2,
            wantNames:   []string{"skill-b", "skill-c"},
        },
        "empty source means all targets are orphans": {
            sourceSkills: []model.Skill{},
            targetSkills: []model.Skill{
                {Name: "skill-a"},
            },
            wantOrphans: 1,
            wantNames:   []string{"skill-a"},
        },
        "empty target means no orphans": {
            sourceSkills: []model.Skill{
                {Name: "skill-a"},
            },
            targetSkills: nil,
            wantOrphans:  0,
        },
        "no orphans when sets are equal": {
            sourceSkills: []model.Skill{
                {Name: "skill-a"},
                {Name: "skill-b"},
            },
            targetSkills: []model.Skill{
                {Name: "skill-a"},
                {Name: "skill-b"},
            },
            wantOrphans: 0,
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            orphans := findOrphanedSkills(tt.sourceSkills, tt.targetSkills)
            if len(orphans) != tt.wantOrphans {
                t.Errorf("got %d orphans, want %d", len(orphans), tt.wantOrphans)
            }
            if tt.wantNames != nil {
                gotNames := make(map[string]bool)
                for _, s := range orphans {
                    gotNames[s.Name] = true
                }
                for _, name := range tt.wantNames {
                    if !gotNames[name] {
                        t.Errorf("expected orphan %q not found", name)
                    }
                }
            }
        })
    }
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/nklauer/dev/go/skillsync && go test ./internal/cli/ -run TestFindOrphanedSkills -v
```

Expected: FAIL — `findOrphanedSkills` undefined.

**Step 3: Implement `findOrphanedSkills`**

In `internal/cli/commands.go`, add near `filterDeleteCandidates` (around line 1357):

```go
// findOrphanedSkills returns target skills whose names don't appear in the source list.
// These are candidates for deletion when --delete is used with sync.
func findOrphanedSkills(sourceSkills, targetSkills []model.Skill) []model.Skill {
    if len(targetSkills) == 0 {
        return nil
    }

    sourceNames := make(map[string]bool, len(sourceSkills))
    for _, skill := range sourceSkills {
        sourceNames[skill.Name] = true
    }

    var orphans []model.Skill
    for _, skill := range targetSkills {
        if !sourceNames[skill.Name] {
            orphans = append(orphans, skill)
        }
    }

    return orphans
}
```

**Step 4: Run test to verify it passes**

```bash
cd /Users/nklauer/dev/go/skillsync && go test ./internal/cli/ -run TestFindOrphanedSkills -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/commands.go internal/cli/orphan_detection_test.go
git commit -m "feat(sync): add orphan detection for --delete flag"
```

---

## Task 3: Integrate `--delete` into CLI sync path

**Files:**

- Modify: `internal/cli/commands.go:1058-1065` (end of `runSyncCommand`, after sync completes)

**Step 1: Write the failing test**

Add to `internal/cli/orphan_detection_test.go`:

```go
func TestDeleteOrphansPrompt(t *testing.T) {
    // Test that findOrphanedSkills correctly identifies orphans
    // when used in the sync flow context
    source := []model.Skill{
        {Name: "keep-this", Platform: model.ClaudeCode},
    }
    target := []model.Skill{
        {Name: "keep-this", Platform: model.Cursor},
        {Name: "remove-this", Platform: model.Cursor, Scope: model.ScopeUser, Path: "/tmp/test"},
    }

    orphans := findOrphanedSkills(source, target)
    if len(orphans) != 1 {
        t.Fatalf("expected 1 orphan, got %d", len(orphans))
    }
    if orphans[0].Name != "remove-this" {
        t.Errorf("expected orphan name %q, got %q", "remove-this", orphans[0].Name)
    }
}
```

**Step 2: Run test to verify it passes** (it should already pass with existing implementation)

```bash
cd /Users/nklauer/dev/go/skillsync && go test ./internal/cli/ -run TestDeleteOrphansPrompt -v
```

**Step 3: Add the post-sync delete phase to `runSyncCommand`**

In `internal/cli/commands.go`, after `displaySyncResults(result)` (line 1059) and before the `result.Success()` check (line 1061), add:

```go
    // Post-sync orphan deletion (--delete flag)
    if cfg.deleteOrphans && !cfg.deleteMode {
        targetSkills, err := parsePlatformSkillsWithScope(
            cfg.targetSpec.Platform,
            []model.SkillScope{cfg.targetSpec.TargetScope()},
            cfg.includePlugins,
        )
        if err != nil {
            return fmt.Errorf("failed to parse target skills for orphan detection: %w", err)
        }

        orphans := findOrphanedSkills(cfg.sourceSkills, targetSkills)
        if len(orphans) > 0 {
            fmt.Printf("\nFound %d orphaned skill(s) in target (not in source):\n", len(orphans))
            for _, s := range orphans {
                fmt.Printf("  - %s\n", s.Name)
            }

            if cfg.dryRun {
                fmt.Println("\n(dry-run) Would delete the above orphaned skills")
            } else {
                // Build a temporary config for the delete operation
                deleteCfg := &syncConfig{
                    sourceSpec:     cfg.sourceSpec,
                    targetSpec:     cfg.targetSpec,
                    dryRun:         cfg.dryRun,
                    skipBackup:     cfg.skipBackup,
                    yesFlag:        cfg.yesFlag,
                    deleteMode:     true,
                    includePlugins: cfg.includePlugins,
                    sourceSkills:   orphans,
                }
                if err := executeDeleteForSkills(deleteCfg, orphans, false); err != nil {
                    return fmt.Errorf("orphan deletion failed: %w", err)
                }
            }
        } else {
            fmt.Println("\nNo orphaned skills found in target")
        }
    }
```

**Step 4: Verify the build compiles**

```bash
cd /Users/nklauer/dev/go/skillsync && go build ./...
```

**Step 5: Commit**

```bash
git add internal/cli/commands.go internal/cli/orphan_detection_test.go
git commit -m "feat(sync): integrate --delete orphan cleanup into CLI sync path"
```

---

## Task 4: Integrate `--delete` into TUI sync path

**Files:**

- Modify: `internal/cli/commands.go:2931-3025` (`runSyncTUI` function)

**Step 1: Add post-sync orphan detection and delete list to `runSyncTUI`**

In `internal/cli/commands.go`, after the sync results display in `runSyncTUI` (line 3022), before the final `return nil` (line 3024), add:

```go
    // Post-sync orphan deletion phase
    // Parse target skills to find orphans
    targetSkills, err := parsePlatformSkillsWithScope(
        targetPlatform,
        []model.SkillScope{targetScope},
        includePlugins,
    )
    if err != nil {
        // Non-fatal: just skip orphan detection
        ui.Warning(fmt.Sprintf("Could not check for orphaned skills: %v", err))
        return nil
    }

    orphans := findOrphanedSkills(sourceSkills, targetSkills)
    if len(orphans) == 0 {
        return nil
    }

    ui.Info(fmt.Sprintf("Found %d orphaned skill(s) in %s not present in %s", len(orphans), targetPlatform, sourcePlatform))

    // Launch delete list TUI pre-populated with orphans only
    deleteResult, err := tui.RunDeleteList(orphans)
    if err != nil {
        return fmt.Errorf("delete list error: %w", err)
    }

    if deleteResult.Action == tui.DeleteActionNone {
        return nil // User cancelled deletion
    }

    if deleteResult.Action == tui.DeleteActionDelete {
        return executeDelete(deleteResult)
    }
```

**Important note:** Keep plugin inclusion consistent with CLI by plumbing the same `includePlugins` setting into the TUI path before orphan detection. This avoids mismatched orphan sets between CLI and TUI flows.

Wait — actually, let's make this conditional. The TUI should ask the user if they want to clean up orphans, rather than always showing the delete list. We can use a simple prompt.

**Revised approach:** After showing orphan count, ask user if they want to review orphans for deletion:

```go
    orphans := findOrphanedSkills(sourceSkills, targetSkills)
    if len(orphans) == 0 {
        return nil
    }

    fmt.Printf("\nFound %d orphaned skill(s) in %s not present in %s\n", len(orphans), targetPlatform, sourcePlatform)
    confirmed, err := confirmAction(
        "Review orphaned skills for cleanup?",
        riskLevelInfo,
    )
    if err != nil || !confirmed {
        return nil
    }

    deleteResult, err := tui.RunDeleteList(orphans)
    if err != nil {
        return fmt.Errorf("delete list error: %w", err)
    }

    if deleteResult.Action == tui.DeleteActionDelete {
        return executeDelete(deleteResult)
    }

    return nil
```

**Step 2: Verify the build compiles**

```bash
cd /Users/nklauer/dev/go/skillsync && go build ./...
```

**Step 3: Commit**

```bash
git add internal/cli/commands.go
git commit -m "feat(sync): add orphan cleanup phase to TUI sync flow"
```

---

## Task 5: Quality gates and final verification

**Step 1: Run full test suite**

```bash
cd /Users/nklauer/dev/go/skillsync && just test
```

Expected: All tests pass.

**Step 2: Run linter**

```bash
cd /Users/nklauer/dev/go/skillsync && just lint
```

Expected: No new issues.

**Step 3: Run full audit**

```bash
cd /Users/nklauer/dev/go/skillsync && just audit
```

Expected: All checks pass.

**Step 4: Manual smoke test**

```bash
cd /Users/nklauer/dev/go/skillsync && just build
./bin/skillsync sync --help  # Verify --delete flag appears
./bin/skillsync sync --delete --dry-run cursor claudecode  # Dry-run orphan detection
```

**Step 5: Final commit if any fixes needed, then done**
