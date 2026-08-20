# Agent Skills Core and Harness Adapter Refresh

## Status

Accepted for implementation on 2026-08-18.

## Objective

Center Skillsync on one specification-correct Agent Skills bundle model with six
harness adapters: Claude Code, Codex, Cursor, Copilot, Gemini CLI, and Pi.
Runtime behavior, reference documentation, and portability validation must agree.

## Decisions

- `pi` is the canonical platform identifier. `pi.dev`, `pi-dev`, `pidev`,
  `pi-agent`, and `piagent` remain advisory deprecated aliases.
- Existing legacy files remain in place and discoverable. Only future default
  writes use corrected canonical roots.
- Invalid skills remain discoverable with structured conformance issues. Writes
  fail unless the existing `--skip-validation` option is explicit.
- Same-harness round trips retain raw extension frontmatter. Cross-harness writes
  emit shared fields plus explicit mappings and report discarded semantics.
- Native agents, hooks, plugins/packages, MCP configuration, and runtime trust
  enforcement are follow-up work, not part of this delivery.

## Implementation Order

1. Align the shared model/parser/validator, portability reference baseline, and
   harness registry in parallel with non-overlapping file ownership.
2. Correct the six harness adapters and synchronization behavior after the
   foundation contracts are integrated.
3. Wire CLI/help/config output, refresh shared documentation, run an independent
   review, and complete repository quality gates.

## Verification

- Focused Go tests after every implementation wave.
- A six-harness bundle matrix covering `SKILL.md`, scripts, references, assets,
  metadata, destinations, and portability warnings.
- `just portability-check`, `just audit`, `git diff --check`, and
  `graphify update .` before completion.

## Tracking

Beads epic `ss-2j9` and its child tasks are the implementation tracker.
