---
name: full-agent-skill
description: A skill with all Agent Skills Standard fields
scope: user
disable-model-invocation: true
license: MIT
compatibility:
  claude-code: ">=1.0.0"
  cursor: ">=0.5.0"
  codex: ">=0.1.0"
scripts:
  - setup.sh
  - validate.sh
references:
  - docs/guide.md
  - https://example.com/docs
assets:
  - templates/config.yaml
  - data/schema.json
tools:
  - Read
  - Write
  - Bash
---
# Full Agent Skills Standard Skill

This skill demonstrates all fields supported by the Agent Skills Standard.

## Usage

This skill is configured with:
- Custom scope (user-level)
- Model invocation disabled
- MIT license
- Platform compatibility constraints
- Associated scripts, references, and assets
