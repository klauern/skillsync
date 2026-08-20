# Adding a New Platform to skillsync

Adding a harness starts with one registry definition. The registry is the source
of truth for canonical identity, aliases, roots, parser selection metadata, and
artifact surfaces; callers should not introduce another platform switch.

---

## Checklist

### 1. Add the canonical platform identity

- [ ] Add a `Platform` constant:
  ```go
  MyPlatform Platform = "my-platform"
  ```
- [ ] Add it to the canonical six-or-more platform enumeration and parsing tests.
- [ ] Add deprecated spellings only when a compatibility contract requires them.

### 2. Add one `internal/harness` registry definition

- [ ] Canonical repository and user write roots.
- [ ] Ordered compatibility/discovery roots.
- [ ] Parser factory key and supported artifact surfaces.
- [ ] Alias-resolution and root-order tests.

### 3. Add configuration and environment overrides

- [ ] Add `MyPlatform PlatformConfig` field to `PlatformsConfig` with a `yaml:"my_platform"` tag.
- [ ] Populate default `SkillsPaths` in `Default()`.
- [ ] Add env-var override in `applyEnvironment()`:
  ```go
  if v := os.Getenv("SKILLSYNC_MY_PLATFORM_SKILLS_PATHS"); v != "" {
      c.Platforms.MyPlatform.SkillsPaths = splitPaths(v)
  }
  ```
- [ ] If replacing an old identifier, add deterministic load precedence and emit
      only the canonical key on new saves.

### 4. Add parser and conformance coverage

- [ ] Create `internal/parser/myplatform/` and register its factory key.
- [ ] Reuse the shared `SKILL.md` parser for directory bundles.
- [ ] Add harness-specific fixtures without weakening shared conformance rules.

### 5. Add target transformation only where necessary

- [ ] Add a `case model.MyPlatform:` arm in `transformPath()` for any target-platform-specific path rules.
- [ ] Add a `case model.MyPlatform:` arm in `transformMetadata()` if metadata keys need to be renamed or dropped when targeting this platform.

### 6. Update user-facing surfaces

- [ ] Add a `case "my-platform":` arm in `colorPlatform()` (`internal/cli/commands.go` around line 927) to assign a distinct color.
- [ ] Ensure CLI/TUI enumeration consumes the registry rather than a local list.
- [ ] Update help, configuration snapshots, and platform documentation.

### 7. Update portability references

- [ ] Add a `my_platform` entry under `platform_support` with `status`, `artifact_surfaces`, and `notes`.
- [ ] Add a `my_platform` entry under `precedence` listing the scope names in order.

### 8. Add portability warnings

- [ ] Add any platform-specific metadata keys to `lossyFieldUnsupportedBy` that should warn when syncing _to_ other platforms (e.g. `"applyTo"` for Copilot).

### 9. Tests

- [ ] Unit tests in `internal/parser/myplatform/` for the new parser.
- [ ] Registry/factory test: `NewForPlatformWithDir("my-platform", ...)` should succeed.
- [ ] Six-or-more harness bundle matrix coverage for a round trip FROM and TO the new harness.
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
