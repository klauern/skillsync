---
name: with-dependencies
description: Example skill that depends on other skills
scope: user
dependencies:
  - basic-skill
  - full-agent-skill
tools:
  - Read
  - Write
---
# Skill with Dependencies

This skill demonstrates dependency declaration in the Agent Skills Standard.

It depends on:
- `basic-skill`: Provides foundational functionality
- `full-agent-skill`: Provides advanced features

When syncing, this skill will be processed AFTER its dependencies are synced.
