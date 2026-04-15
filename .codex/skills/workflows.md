# CI Failure Analysis Workflows

Step-by-step workflows for analyzing and fixing CI failures.

## Standard Workflow

### Phase 1: Context Detection (Haiku)
```bash
git status                          # Check clean state
git branch --show-current           # Get branch
gh pr view --json number,state      # Check if PR exists
```

### Phase 2: List Failing Checks (Haiku)
```bash
# If PR exists:
gh pr checks

# If branch-only:
gh run list --branch $(git branch --show-current) --limit 5 --json databaseId,conclusion,name
```

### Phase 3: Retrieve Logs (Haiku)
```bash
gh run view <run-id> --log-failed
```

### Phase 4: Analyze Root Causes (Sonnet)
1. Extract error messages using patterns from [log-parsing.md](log-parsing.md)
2. Categorize by type (see [failure-patterns.md](failure-patterns.md))
3. Determine fixability: auto-fix vs manual vs investigation

### Phase 5: Apply Fixes (Mixed)
**Auto-fix (Haiku)**:
```bash
npx prettier --write .      # Formatting
npx eslint --fix .          # Linting
npm install                 # Lock file
black .                     # Python formatting
gofumpt -w .                # Go formatting
```

**Manual (Sonnet)**: Explain issue, suggest approach, ask user before editing.

### Phase 6: Verify & Re-run (Haiku)
```bash
git diff --stat             # Show changes
gh run rerun <run-id>       # Optional re-trigger
```

---

## Workflow by Failure Type

### Formatter/Linter Workflow
1. Detect tool from log (prettier, black, eslint, etc.)
2. Confirm config exists
3. Announce: "Running: npx prettier --write ."
4. Apply fix
5. Show `git diff --stat`
6. Verify with `--check` mode

**Guardrails**: Warn if >25 files affected. Never edit non-formatting files.

### Dependency Workflow
```bash
# Lock mismatch
npm install                 # or yarn install, poetry lock

# Missing package (if obvious)
npm install <package>

# After fixing
git diff package-lock.json  # Show changes
npm ci                      # Verify
```

### Test Failure Workflow (Sonnet)
1. Parse test output: failing test name, expected vs received
2. Read test file and implementation
3. Check `git diff main` for recent changes
4. Determine: test wrong vs logic wrong
5. Ask user before fixing

### Secrets/Permissions Workflow
1. Detect: "Resource not accessible", "No value for required secret"
2. Identify secret name from workflow YAML
3. Provide remediation: "Define `SECRET_NAME` in Settings → Secrets"
4. **Never edit workflow files or guess secret values**

### Cache/Artifact Workflow
1. Detect: "Failed to restore cache", "tar: short read"
2. Recommend: Bump cache key suffix, clear artifact, or re-run
3. **No code changes needed**

### Matrix Workflow
1. List jobs: `gh run view <id> --json jobs`
2. Identify failing combination (e.g., Node 18 only)
3. Provide targeted repro: `nvm use 18 && npm test`
4. Rerun only failing job: `gh run rerun <id> --job "test (node-18)"`

---

## Model Strategy

| Phase | Model | Why |
|-------|-------|-----|
| Context detection | Haiku | Fast I/O |
| Log retrieval | Haiku | Command execution |
| Pattern matching | Haiku | Known signatures |
| Root cause analysis | Sonnet | Semantic reasoning |
| Fix strategy | Sonnet | Decision making |
| Tool execution | Haiku | Run commands |
| Explanation | Sonnet | Natural language |

**Decision**: Use Haiku if pattern is known and fix is mechanical. Use Sonnet if reasoning or explanation needed.

---

## Result Reporting Template

```
CI Failure Summary
------------------
Branch/PR: <name>
Failing jobs: <list>

Root causes:
1. <Category> — <description> (confidence: High/Med/Low)
   Action: <auto-fix command OR guidance>
   Status: Done / Needs approval / Blocked

Actions taken:
- [x] npx prettier --write .
- [ ] Manual fix pending

Next steps:
- Rerun via `gh run rerun <id>`
- Manual: update secret `NPM_TOKEN`
```

---

## Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| No failing checks | Wrong branch or queued | `git branch --show-current`, wait |
| Cannot retrieve logs | Run in progress | Wait for completion |
| Auto-fix didn't work | CI uses different config | Check CI config vs local |
| Too many failures | Root cause cascade | Fix highest priority first |
| Flaky test | Non-deterministic | Add mocks/retries, see [examples](examples.md) |