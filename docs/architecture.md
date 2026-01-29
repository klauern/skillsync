# Architecture

## Package Dependencies

```mermaid
graph TD
    cmd[cmd/skillsync] --> cli[internal/cli]
    cli --> config[config]
    cli --> parser[parser]
    cli --> sync[sync]
    cli --> backup[backup]
    cli --> export[export]
    parser --> model[model]
    sync --> model
    export --> model
    parser --> tiered[parser/tiered]
    tiered --> claude[parser/claude]
    tiered --> cursor[parser/cursor]
    tiered --> codex[parser/codex]
```

## Core Interfaces

**Parser** (`internal/parser/parser.go`):

```go
type Parser interface {
    Parse() ([]model.Skill, error)
    Platform() model.Platform
    DefaultPath() string
}
```

**Platform**: `ClaudeCode | Cursor | Codex` (`internal/model/platform.go`)

**Strategy**: `overwrite | skip | newer | merge | three-way | interactive`
(`internal/sync/strategy.go`)

## Data Flow

1. CLI command invoked
2. Parser discovers skills from platform config
3. Sync applies strategy to merge skills
4. Export writes to target format
