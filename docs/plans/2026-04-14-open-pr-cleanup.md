# Open PR Cleanup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task.

**Goal:** Finish repairing the remaining dirty open PR branches so each one is either green or explicitly deferred.

**Architecture:** Work one PR branch at a time in a clean worktree. For each branch, merge `origin/main`, resolve the exact conflict set reported by GitHub, run the narrowest relevant tests first, then finish with repo-wide lint and test checks before pushing the updated head back to the original PR branch.

**Tech Stack:** Go 1.25.8, `gofmt`, `go test`, `golangci-lint`, Git/GitHub PRs.

---

### Task 1: Finish the Pi.dev wiring branch

**Branch:** `ss-c72-1-pidev-platform-wiring`

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/model/platform.go`
- Modify: `internal/model/platform_test.go`
- Modify: `internal/parser/pidev/pidev.go`
- Modify: `internal/parser/pidev/pidev_test.go`
- Modify: `internal/parser/tiered/factories.go`
- Modify: `internal/util/paths.go`
- Modify: `internal/util/paths_test.go`
- Modify: `internal/validation/validation.go`
- Modify: `internal/validation/validation_test.go`
- Verify: `internal/model/skill_test.go` if the merge changes platform scope output

**Step 1: Merge `origin/main` and resolve conflicts**

Run:
```bash
git merge origin/main
```

Expected: conflict markers only in the Pi.dev/config/platform/validation files listed above.

**Step 2: Resolve Pi.dev platform wiring**

Keep the full Pi.dev implementation, including:
- platform registration in `internal/model/platform.go`
- config defaults and env vars in `internal/config/config.go`
- path resolution in `internal/util/paths.go`
- parser factory wiring in `internal/parser/tiered/factories.go`
- validation path handling in `internal/validation/validation.go`

Prefer the richer `internal/parser/pidev/pidev.go` / `pidev_test.go` implementation that discovers skills, prompts, instructions, and system prompt files.

**Step 3: Update the matching tests**

Make sure tests cover:
- Pi.dev defaults in `internal/config/config_test.go`
- `model.Platform` behavior in `internal/model/platform_test.go`
- Pi.dev path selection in `internal/util/paths_test.go`
- Pi.dev validation path overrides in `internal/validation/validation_test.go`
- Pi.dev parser behavior in `internal/parser/pidev/pidev_test.go`

**Step 4: Format and run targeted tests**

Run:
```bash
gofmt -w internal/config/config.go internal/config/config_test.go internal/model/platform.go internal/model/platform_test.go internal/parser/pidev/pidev.go internal/parser/pidev/pidev_test.go internal/parser/tiered/factories.go internal/util/paths.go internal/util/paths_test.go internal/validation/validation.go internal/validation/validation_test.go
GOCACHE=/tmp/go-build XDG_CACHE_HOME=/tmp HOME=/tmp go test ./internal/config ./internal/model ./internal/parser/pidev ./internal/parser/tiered ./internal/util ./internal/validation
GOCACHE=/tmp/go-build XDG_CACHE_HOME=/tmp HOME=/tmp golangci-lint run ./internal/config ./internal/model ./internal/parser/pidev ./internal/parser/tiered ./internal/util ./internal/validation
```

Expected: all pass with no lint findings.

**Step 5: Commit and push**

Run:
```bash
git add -A
git commit -m "merge main into pidev wiring branch"
git push origin fix/pr24-pidev-wiring:ss-c72-1-pidev-platform-wiring
```

---

### Task 2: Finish the Copilot instruction parser branch

**Branch:** `codex/ss-6a3-copilot-instructions`

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/model/platform.go`
- Modify: `internal/model/platform_test.go`
- Modify: `internal/model/skill_test.go`
- Modify: `internal/parser/copilot/copilot.go`
- Modify: `internal/parser/copilot/copilot_test.go`
- Modify: `internal/parser/integration_fixtures_test.go`
- Modify: `internal/parser/tiered/factories.go`
- Modify: `internal/util/paths.go`
- Modify: `internal/util/paths_test.go`
- Modify: `internal/validation/validation.go`
- Modify: `internal/validation/validation_test.go`

**Step 1: Merge `origin/main` and resolve conflicts**

Run:
```bash
git merge origin/main
```

Expected: conflicts around Copilot/Gemini platform registration, config defaults, and path helpers.

**Step 2: Keep both Gemini and Copilot support**

Make sure the merged result still contains:
- Gemini platform handling already present in this branch
- Copilot support from `origin/main`
- both parser factories in `internal/parser/tiered/factories.go`
- both platform path implementations in `internal/util/paths.go`

**Step 3: Update tests and fixtures**

Verify or extend:
- platform behavior in `internal/model/platform_test.go`
- skill scope display in `internal/model/skill_test.go`
- config defaults and env overrides in `internal/config/config_test.go`
- path helpers in `internal/util/paths_test.go`
- parser integration fixtures in `internal/parser/integration_fixtures_test.go`

**Step 4: Format and run targeted checks**

Run:
```bash
gofmt -w internal/config/config.go internal/config/config_test.go internal/model/platform.go internal/model/platform_test.go internal/model/skill_test.go internal/parser/copilot/copilot.go internal/parser/copilot/copilot_test.go internal/parser/integration_fixtures_test.go internal/parser/tiered/factories.go internal/util/paths.go internal/util/paths_test.go internal/validation/validation.go internal/validation/validation_test.go
GOCACHE=/tmp/go-build XDG_CACHE_HOME=/tmp HOME=/tmp go test ./internal/config ./internal/model ./internal/parser/copilot ./internal/parser/tiered ./internal/util ./internal/validation
GOCACHE=/tmp/go-build XDG_CACHE_HOME=/tmp HOME=/tmp golangci-lint run ./internal/config ./internal/model ./internal/parser/copilot ./internal/parser/tiered ./internal/util ./internal/validation
```

Expected: all pass.

**Step 5: Commit and push**

Run:
```bash
git add -A
git commit -m "merge main into copilot parser branch"
git push origin fix/pr18-copilot-instructions:codex/ss-6a3-copilot-instructions
```

---

### Task 3: Finish the beads/TUI branch

**Branch:** `beads-sync`

**Files:**
- Modify: `internal/cli/commands.go`
- Modify: `internal/cli/scope_commands.go`
- Modify: `internal/model/platform.go`
- Modify: `internal/model/platform_test.go`
- Modify: `internal/parser/tiered/factories.go`
- Modify: `internal/parser/tiered/tiered.go`
- Modify: `internal/parser/tiered/tiered_test.go`
- Modify: `internal/parser/skills/skills.go`
- Modify: `internal/ui/tui/*`
- Modify: `internal/util/paths.go`
- Modify: `internal/util/paths_test.go`
- Modify: `internal/validation/validation.go`
- Modify: `internal/validation/validation_test.go`
- Review: `.beads/issues.jsonl` only if the merge requires it

**Step 1: Merge `origin/main` and resolve conflicts**

Run:
```bash
git merge origin/main
```

Expected: conflicts in the CLI, validation, and TUI files related to Pi.dev and platform plumbing.

**Step 2: Resolve the shared platform plumbing**

Make sure the branch keeps:
- the newer platform model updates
- any Pi.dev path and validation logic
- the TUI scrolling work already merged on the other branches

**Step 3: Run the most relevant tests first**

Run:
```bash
GOCACHE=/tmp/go-build XDG_CACHE_HOME=/tmp HOME=/tmp go test ./internal/cli ./internal/model ./internal/parser/tiered ./internal/ui/tui ./internal/util ./internal/validation
```

If that passes, run:
```bash
GOCACHE=/tmp/go-build XDG_CACHE_HOME=/tmp HOME=/tmp go test ./...
GOCACHE=/tmp/go-build XDG_CACHE_HOME=/tmp HOME=/tmp golangci-lint run ./...
```

**Step 4: Commit and push**

Run:
```bash
git add -A
git commit -m "merge main into beads sync branch"
git push origin beads-sync
```

---

### Task 4: Final verification

**Files:**
- None, unless a final merge conflict or lint issue appears

**Step 1: Re-check PR state**

Run:
```bash
gh pr list --state open --limit 50 --json number,title,headRefName,mergeStateStatus
```

Expected: no remaining `DIRTY` PRs, or a short list of any final holdouts.

**Step 2: Confirm repo health**

Run:
```bash
GOCACHE=/tmp/go-build XDG_CACHE_HOME=/tmp HOME=/tmp go test ./...
GOCACHE=/tmp/go-build XDG_CACHE_HOME=/tmp HOME=/tmp golangci-lint run ./...
```

**Step 3: Record any leftovers**

If any PR still needs human input, add a beads issue for it and note the exact file and conflict set.
