# Pi Agent

> Platform reference for SkillSync parser development and cross-platform mapping.

## Overview

Pi Agent stores reusable skills as Agent Skills Standard bundles rooted at
`.agents/skills/` in repositories and `~/.agents/skills/` for user scope.
Additional skill roots can be declared in `.pi/settings.json` and
`~/.config/pi/settings.json` via `skillsDirectories`.

SkillSync treats Pi as a skills-only platform in v1. It reads `skillsDirectories`
for discovery, resolves relative entries from the settings file that declared
them, and writes synced skills to `.agents/skills/` rather than mutating
Pi settings.

## Directory Structure

```text
.agents/
  skills/
    <skill-name>/
      SKILL.md

.pi/
  settings.json

~/.agents/
  skills/
    <skill-name>/SKILL.md

~/.config/pi/
  settings.json
```

## Discovery Order

SkillSync resolves Pi skills in this order:

1. Nearest ancestor `.agents/skills/` up to the repo root
2. Additional directories from `<repo>/.pi/settings.json`
3. `~/.agents/skills/`
4. Additional directories from `~/.config/pi/settings.json`

Duplicate directories are deduplicated after path resolution. If two Pi skills
share a name within the same scope, the first discovered path wins.

## Artifact Type

Pi skills use `SKILL.md` bundles with standard `name` and `description`
frontmatter. SkillSync preserves additional frontmatter as metadata and writes
cross-platform sync targets as `<skill-name>/SKILL.md`.
