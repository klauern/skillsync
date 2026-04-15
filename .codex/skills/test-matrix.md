# CI Failure Analyzer Test Matrix

Validation scenarios for `/gh-checks` behavior.

## How to Use

1. Create feature branch (`test/ci-failure-matrix-*`)
2. Introduce one failure category per commit
3. Push and run `/gh-checks`
4. Compare output with expected behavior
5. Reset before next scenario

## Test Scenarios

| # | Scenario | How to Reproduce | Expected Behavior |
|---|----------|------------------|-------------------|
| 1 | Formatting | Break prettier/black style | Auto-detect tool, run formatter, show diff |
| 2 | Lint (auto-fix) | Add unused imports | Run `--fix`, list remaining manual issues |
| 3 | Type errors | Break TypeScript/mypy types | Summarize errors, offer options, **don't auto-edit** |
| 4 | Test failure | Change logic to fail assertion | Extract test name, expected vs received, ask before fixing |
| 5 | Lock mismatch | Edit package.json without npm install | Regenerate lock file, show diff |
| 6 | Build failure | Remove required import | Identify missing module, suggest fix, ask first |
| 7 | Missing secret | Remove `secrets.X` reference | Identify secret name, guide to Settings, **no code changes** |
| 8 | Cache corruption | Break cache key | Advise clearing cache, rerun |
| 9 | Matrix partial | Break only one Node version | Show failing axis, provide targeted repro, rerun only that axis |
| 10 | Flaky test | Add random timing | Detect history inconsistency, suggest mocking/retries |
| 11 | No runs | Use branch without CI trigger | Explain absence, suggest checking workflow filters |
| 12 | Auth failure | Revoke `gh` auth | Prompt `gh auth login` |

## Guardrails to Verify

| Scenario | Guardrail |
|----------|-----------|
| Formatting | Warn if >25 files |
| Type errors | Never auto-edit business logic |
| Secrets | Never print or guess values |
| Cache | Never delete via API without approval |
| Matrix | Never rerun entire matrix unnecessarily |

## Execution Tips

- **Batch by speed**: Format/lint/deps in one branch, tests/builds in another
- **Capture output**: Save `/gh-checks` transcript for comparison
- **Track findings**: Log deviations in gitignored notes
- **Rerun quarterly**: After major updates, re-test fast scenarios (1,2,5,7,11)