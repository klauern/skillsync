---
description: Review pull requests with minimal diff philosophy, address PR comments with targeted fixes, and ensure tests pass. Handles both reviewing others' PRs and fixing your own PRs based on feedback.
name: pr-reviewer
---

# PR Reviewer Skill

This skill helps you review pull requests and address PR comments with a **minimal diff approach**—making the smallest possible changes to fix issues without scope creep or unnecessary refactoring.

## Core Principles

1. **Minimal Diff Approach**: Make the smallest possible change to address the issue
2. **Direct Fixes Only**: No refactoring, no "improvements", no "while we're here" changes
3. **Tests MUST Pass**: All tests must pass before considering the work complete
4. **Scope Discipline**: Stay strictly within the bounds of the original PR or specific comment

## Usage Scenarios

### Scenario 1: Reviewing a PR (as Reviewer)

When asked to review a PR, you should:

1. **Fetch the PR**: Use `gh pr view <number>` or `gh pr checkout <number>` to get PR details
2. **Analyze Changes**: Review the diff with focus on:
   - Does this change do ONE thing?
   - Are there unrelated changes that should be separate PRs?
   - Is the scope minimal or is there refactoring/cleanup?
   - Are tests included and passing?
3. **Provide Focused Feedback**: Comment on:
   - Scope creep or unnecessary changes
   - Missing tests or failing tests
   - Direct issues with the implementation
   - Suggest minimal fixes (not refactors)
4. **Use GitHub CLI**: Post comments using `gh pr review` or `gh pr comment`

### Scenario 2: Fixing Your PR Based on Comments (as Author)

When asked to address PR review comments:

1. **Fetch PR Comments**: Use `gh pr view <number>` to see all review comments
2. **Analyze Each Comment**: Understand what the reviewer is asking for
3. **Plan Minimal Fixes**: For each comment, identify the smallest possible fix
4. **Implement Changes**: Make only the changes necessary to address the comment
5. **Run Tests**: MUST verify all tests pass before completing
6. **Respond to Comments**: Use `gh pr comment` to reply to each addressed comment
7. **Update PR**: Push changes with clear commit message referencing the comments

### Scenario 3: Reducing Scope of an Existing Fix

When asked to reduce scope or make a fix more minimal:

1. **Identify Core Change**: What is the ONE thing this PR is trying to fix?
2. **Remove Tangential Changes**: Revert any changes not directly related to the core fix
3. **Avoid Refactoring**: If code works but is "messy", leave it unless it's the PR's purpose
4. **Split if Necessary**: Suggest creating separate PRs for unrelated improvements

## Workflow Steps

### Initial Assessment

```bash
# Get PR information
gh pr view <number>

# Get detailed diff
gh pr diff <number>

# Check PR status and comments
gh pr checks <number>
gh pr view <number> --comments
```

### Making Changes

1. **Checkout the PR branch**:
   ```bash
   gh pr checkout <number>
   ```

2. **Make minimal targeted changes**:
   - Use `Edit` tool for precise modifications
   - Avoid reformatting unrelated code
   - Keep diffs as small as possible

3. **Run tests** (REQUIRED):
   ```bash
   # Run project-specific test command
   npm test           # or
   pytest             # or
   cargo test         # or
   bundle exec rspec  # or
   bun test          # etc.
   ```

4. **Verify the fix**:
   - Tests pass ✓
   - Only necessary files changed ✓
   - No scope creep ✓

5. **Commit and push**:
   ```bash
   git add <files>
   git commit -m "fix: address review comment - <specific issue>"
   git push
   ```

6. **Respond to comments**:
   ```bash
   gh pr comment <number> --body "Fixed in <commit-sha>"
   ```

## Integration with External Systems

### GitHub CLI (gh)

Use `gh` for all GitHub interactions:
- `gh pr list` - List PRs
- `gh pr view <number>` - View PR details
- `gh pr checkout <number>` - Check out PR branch
- `gh pr diff <number>` - View PR diff
- `gh pr checks <number>` - Check CI status
- `gh pr review <number>` - Start a review
- `gh pr comment <number>` - Add comments

### Jira CLI (optional)

If the PR references Jira tickets:
- Link PR to Jira ticket
- Update ticket status when PR is ready
- Add comments to Jira with PR link

## Anti-Patterns to Avoid

❌ **Scope Creep**: "While I'm here, let me also fix this other thing..."
❌ **Refactoring**: "This code is messy, let me clean it up..."
❌ **Style Changes**: "Let me reformat this file to match my preferences..."
❌ **Over-Engineering**: "Let me add this abstraction for future flexibility..."
❌ **Skipping Tests**: "The fix looks good, I'll skip running tests..."

✅ **Good Approach**: "This PR comment asks for X. The minimal change to fix X is Y. Let me make change Y, run tests, and confirm it works."

## Checklist Before Completing

- [ ] All review comments addressed with minimal changes
- [ ] Tests run and passing (`npm test`, `pytest`, etc.)
- [ ] No unrelated changes in the diff
- [ ] No refactoring unless that's the PR's explicit purpose
- [ ] Commits reference specific comments or issues
- [ ] CI checks passing on GitHub
- [ ] Comments responded to on GitHub

## Example Usage

**User**: "Review PR #123 and let me know if the scope is too broad"

**Process**:
1. Fetch PR #123 details and diff
2. Analyze changes for scope creep
3. Identify minimal core change
4. Flag any unrelated changes
5. Provide specific feedback

**User**: "Address the comments on my PR #456"

**Process**:
1. Fetch PR #456 comments
2. Checkout the branch
3. For each comment, make minimal fix
4. Run all tests
5. Commit and push changes
6. Reply to each comment with resolution

## Notes

- Always use `gh` CLI for GitHub interactions
- Always run tests before considering work complete
- When in doubt, make the smaller change
- If a reviewer suggests a large refactor, discuss scope in comments first
- Use Jira CLI integration when tickets are referenced