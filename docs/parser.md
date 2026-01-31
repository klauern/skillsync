# Parser Development

## Overview

Platform parsers convert platform-specific skill layouts into normalized
`model.Skill` values used by sync and export. Each parser implements the
`Parser` interface (`Parse`, `Platform`, `DefaultPath`) and is registered in
the tiered parser factories so discovery can pick the right implementation.

## Adding a New Platform
- Add the platform constant in `internal/model/platform.go`
- Create a new parser in `internal/parser/<platform>/`
- Implement the Parser interface in `internal/parser/parser.go`
- Register the parser in `internal/parser/tiered/factories.go`

## Fixtures
- Add fixtures under `testdata/skills/<platform>/`
- Include a basic skill, assets, and edge cases (missing frontmatter, legacy formats)

## Example
- Claude parser: `internal/parser/claude/claude.go`
