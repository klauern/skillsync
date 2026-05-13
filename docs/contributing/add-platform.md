# Adding a New Platform to skillsync

Adding a platform requires changes in at least 10 separate sites. Miss one and the platform will compile but silently fail to discover skills, display incorrectly in the TUI, or break tests.

> **Note**: Once `ss-370` (platform registry) lands, most of this checklist will collapse into a single registry entry. Until then, follow every step.

---

## Checklist

### 1. `internal/model/platform.go` — 4 places

- [ ] Add a `Platform` constant:
  ```go
  MyPlatform Platform = "my-platform"
  ```
- [ ] Add to `AllPlatforms()` return slice.
- [ ] Add to `IsValid()` switch (or verify it's covered by the loop over `AllPlatforms()`).
- [ ] Add aliases in `ParsePlatform()` switch so single-word and kebab-case inputs work.

### 2. `internal/util/paths.go` — 3+ places

Add helper functions for the platform's default directory layout:

- [ ] `MyPlatformSkillsPath() string` — user-level skills dir
- [ ] `MyPlatformRepoSkillsPath(projectDir string) string` — project-level skills dir
- [ ] Any additional path helpers (prompts, agents, etc.) following the existing patterns.

### 3. `internal/config/config.go` — 2 places

- [ ] Add `MyPlatform PlatformConfig` field to `PlatformsConfig` with a `yaml:"my_platform"` tag.
- [ ] Populate default `SkillsPaths` in `Default()`.
- [ ] Add env-var override in `applyEnvironment()`:
  ```go
  if v := os.Getenv("SKILLSYNC_MY_PLATFORM_SKILLS_PATHS"); v != "" {
      c.Platforms.MyPlatform.SkillsPaths = splitPaths(v)
  }
  ```
- [ ] Add to `warnDeprecatedYAMLFields()` platforms slice.

### 4. `internal/validation/validation.go` — 2+ places

- [ ] Add a `case model.MyPlatform:` arm in `PlatformSkillsPath()` (the function that resolves the primary skills directory for a given platform).
- [ ] Add format validation rules in `ValidateSkillFormat()` if the platform has unusual file extension requirements.

### 5. `internal/sync/transformer.go` — 2 places

- [ ] Add a `case model.MyPlatform:` arm in `transformPath()` for any target-platform-specific path rules.
- [ ] Add a `case model.MyPlatform:` arm in `transformMetadata()` if metadata keys need to be renamed or dropped when targeting this platform.

### 6. `internal/parser/tiered/factories.go` — 1 place

- [ ] Create a new parser package at `internal/parser/myplatform/`.
- [ ] Register it in `ParserFactoryFor()`:
  ```go
  case model.MyPlatform:
      return MyPlatformParserFactory(), nil
  ```
- [ ] Add `MyPlatformParserFactory()` helper following the existing factory pattern.

### 7. `internal/cli/commands.go` — 3 places

- [ ] Add a `case "my-platform":` arm in `colorPlatform()` (`internal/cli/commands.go` around line 927) to assign a distinct color.
- [ ] Add the platform's skills path to `parsePlatformSkillsFromPaths()` so it participates in discover/sync.
- [ ] Update any help strings that enumerate platform names.

### 8. `docs/platforms/portability-snapshot.yaml` — 2 places

- [ ] Add a `my_platform` entry under `platform_support` with `status`, `artifact_surfaces`, and `notes`.
- [ ] Add a `my_platform` entry under `precedence` listing the scope names in order.

### 9. `internal/sync/portability.go` — 1 place

- [ ] Add any platform-specific metadata keys to `lossyFieldUnsupportedBy` that should warn when syncing _to_ other platforms (e.g. `"applyTo"` for Copilot).

### 10. Tests

- [ ] Unit tests in `internal/parser/myplatform/` for the new parser.
- [ ] `internal/parser/tiered/factories.go` test: `NewForPlatformWithDir("my-platform", ...)` should succeed.
- [ ] E2E tests in `internal/e2e/e2e_test.go`: at minimum a round-trip sync FROM and TO the new platform.
- [ ] `internal/parser/tiered/coverage_test.go`: add a `TestParseFromScope_<MyPlatform>` case if new scope rules apply.

---

## Verification

After completing the checklist:

```bash
just audit                    # tidy, fmt, vet, lint, test
just portability-check        # confirm portability-snapshot.yaml is consistent
./bin/skillsync discover my-platform  # smoke-test discovery
```

Run the full e2e suite to catch any remaining gaps:

```bash
GOTOOLCHAIN=auto go test ./internal/e2e/... -v -run "MyPlatform"
```
