# Command and Agent Model Decision

Date: 2026-03-30

## Decision

Do not add separate first-class `command` or `agent` artifact models to
SkillSync. Keep `model.Skill` as the canonical transport model and use the
existing fields to carry command-like and agent-like information where it is
safe to do so.

## Rationale

- Command-like artifacts already fit the existing `Type=prompt` + `Trigger`
  transport shape.
- Codex does not expose a first-class agent artifact surface that maps cleanly
  onto the other platforms.
- Introducing new top-level artifact types would imply runtime parity that the
  target CLIs do not provide.
- The current model already has the minimum metadata needed for lossy sync and
  export without overfitting to any one CLI.

## Supported Shape

- `model.Skill.Type`
  - `skill` for reusable skills
  - `prompt` for slash-command-like or prompt-like transport artifacts
- `model.Skill.Trigger`
  - Optional slash trigger when the source platform exposes one
- `model.Skill.Metadata`
  - Passthrough for non-portable fields, including Claude-specific agent
    controls
- `model.PluginInfo`
  - Claude plugin provenance only; not a separate artifact class

## Unsupported Shape

- A distinct top-level `command` artifact type
- A distinct top-level `agent` artifact type
- Any claim that Codex can reproduce Claude command triggers or subagent
  behavior natively
- Any claim that plugin-installed content changes the scope model outside the
  Claude platform layer

## Sync and Export Expectations

- Command-like artifacts may be synced as content and metadata, but behavior is
  lossy outside the source platform.
- Agent-like Claude fields may be preserved as metadata, but they should not be
  exported as a separate Codex runtime concept.
- Plugin-installed skills remain platform provenance data, not a distinct
  project/user/admin scope.
