# Pi

> Verified 2026-08-18 against the [Pi skills reference](https://pi.dev/docs/latest/skills). Observed local version: 0.84.2.

Pi's portable artifact is an Agent Skills Standard `SKILL.md` bundle. SkillSync
writes project skills to `.pi/skills/` and user skills to `~/.pi/agent/skills/`.
Discovery also checks `.agents/skills/` and `settings.json` `skills` entries;
settings paths are documented discovery surfaces, not synchronized settings.

Pi loads `AGENTS.md`, `CLAUDE.md`, and `AGENTS.override.md` instruction files
according to its current-directory chain. Trust checks can prevent a skill from
running in an untrusted directory. `/skill:name` is a Pi-native invocation and
is not a portable trigger.

## SkillSync boundary

Implemented: `SKILL.md` content and portable instruction content. Documented
only: prompts and settings skill entries. Native-only: packages, extensions,
trust state, and invocation behavior. SkillSync does not synchronize hooks,
plugins, packages, or custom agents.

## Roots

| Scope | Write | Discover |
|---|---|---|
| Project | `.pi/skills/` | `.pi/skills/`, `.agents/skills/`, `settings.json` `skills` |
| User | `~/.pi/agent/skills/` | `~/.pi/agent/skills/`, configured settings entries |

The shared `name` and `description` frontmatter fields are required. Supporting
`scripts/`, `references/`, and `assets/` remain part of the skill bundle.
