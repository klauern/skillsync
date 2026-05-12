# Adding a New Platform

Use this checklist when introducing a new first-class platform in SkillSync.
It is intentionally explicit: adding a platform usually touches code paths in
multiple packages, plus the portability docs and architecture diagrams.

## Before you start

- Read `docs/platforms/cross-platform-mapping.md` to understand the existing
  naming and artifact mappings.
- Read `docs/platforms/portability-assessment.md` and
  `docs/platforms/portability-snapshot.yaml` so the new platform matches the
  repo's portability model.
- If the work changes core platform behavior, expect `ss-370` to reduce some of
  this checklist later by replacing switch statements with a registry.

## Core code changes

### 1) Add the platform value and core helpers

Update `internal/model/platform.go`:

- `IsValid`
- `ConfigDir`
- `Short`
- `AllPlatforms`

Why: these are the canonical enum helpers used by validation, UI output, and
platform iteration.

### 2) Add path mapping support

Update `internal/util/paths.go`:

- `platformDirName`
- `PlatformSkillsPath`
- `RepoSkillsPath`

Why: these helpers determine where the platform reads and writes skills,
instructions, prompts, and other artifacts on disk.

### 3) Add validation coverage

Update `internal/validation/validation.go` in both path resolver code paths.

Why: platform paths must be accepted by both validation entry points or the CLI
may reject otherwise valid configuration.

### 4) Add sync transformation support

Update `internal/sync/transformer.go`:

- `transformPath`
- `transformContent`

Why: sync/export needs to know how to translate source artifacts into the new
platform's layout and content conventions.

### 5) Update CLI dispatch and display logic

Update `internal/cli/commands.go`:

- `platformSkillsPaths`
- platform color mapping
- any help text or usage strings that enumerate platforms

Why: the CLI is the user-facing source of truth for platform selection and
results display.

### 6) Add config defaults

Update `internal/config/config.go` defaults for the new platform.

Why: default search paths and runtime settings should work without manual
configuration.

## Docs and data files

### 7) Document environment variables and path conventions

Update the user-facing docs that list platform path overrides and defaults,
especially `README.md` and `docs/quick-start.md`.
If the new platform introduces its own overrides, add them there and keep the
platform-specific reference docs in `docs/platforms/` in sync.

Why: platform support is incomplete if the docs still describe only the older
set of platforms.

### 8) Update the portability snapshot

Update `docs/platforms/portability-snapshot.yaml`.

Why: this file is the machine-readable summary of what the repo considers
portable, partially portable, and non-portable.

### 9) Update the architecture diagram

Update the Mermaid diagram in `docs/architecture.md` if the new platform adds a
new parser or sync edge.

Why: the architecture diagram is the fastest way to see where the new platform
fits into the pipeline.

## Suggested verification

Run the smallest checks that cover the touched surface:

- `go test ./...` or a narrower package test set if the change is localized
- `golangci-lint run ./...` or a narrower lint target if you changed only one
  package
- a docs sanity check, such as confirming the new checklist file renders cleanly
  and the referenced paths exist

## Final review

Before closing the change, confirm:

- the platform is discoverable through `AllPlatforms`
- path helpers point to the correct directories
- sync/export round-trips the expected artifact types
- the docs and snapshot agree with the code
- no old platform list remains hidden in help text or comments

## Notes

Adding a platform without updating the docs usually produces silent gaps: missing
path mappings, incomplete help output, or incorrect portability claims. Treat
the docs as part of the implementation, not as a follow-up.
