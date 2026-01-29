# Worktree Examples

Practical workflows and scenarios for git worktree management.

## Basic Workflows

### Initialize a New Repository

```bash
# Clone with worktree support
wt init git@github.com:zendesk/identity-sync-service.git

# Result:
# identity-sync-service/
# ├── .bare/     # Git data
# └── main/      # Default branch worktree

cd identity-sync-service/main
```

### Start Work on a Jira Ticket

```bash
# Create new worktree for ticket
wt new klauern/FSEC-1234-add-user-validation

# Navigate to new worktree
cd ../klauern-FSEC-1234-add-user-validation

# Work on feature...
```

### Review Someone's PR While Keeping WIP

```bash
# You're working on feature A with uncommitted changes
# Need to review PR for feature B

# Add worktree for the PR branch (no need to stash!)
wt add teammate/feature-b

# Navigate and review
cd ../teammate-feature-b
# Review code, run tests...

# Return to your work
cd ../klauern-feature-a
# Your WIP is exactly as you left it
```

### Sync Branch Before Creating PR

```bash
# Before creating PR, sync with main
wt sync klauern-FSEC-1234-feature

# This rebases your branch on origin/main
# Fix any conflicts if needed, then push
git push --force-with-lease
```

### Clean Up After Merge

```bash
# After PR is merged, remove worktree
wt remove klauern-FSEC-1234-feature

# Verify it's gone
wt list
```

### Update All Worktrees

```bash
# Pull latest changes for all branches
wt pull

# This fetches and rebases each worktree
# Skips detached worktrees
```

## Beads Integration Workflows

### Create Worktree Per Beads Issue

```bash
# View ready work
bd ready

# Claim an issue
bd update beads-abc --status=in_progress

# Create worktree for it
wt new klauern/FSEC-1234-description

# Work in the worktree
cd ../klauern-FSEC-1234-description

# Beads state is shared - bd commands work here too
bd show beads-abc
```

### Work on Multiple Beads Issues

```bash
# Issue 1 - in progress
cd ~/dev/guardians/zig-workspaces/repo/klauern-FSEC-1234

# Issue 2 - start in separate worktree
wt new klauern/FSEC-5678-other-feature
cd ../klauern-FSEC-5678-other-feature

# Switch between them freely
# No stashing needed!
```

### Complete Work and Close Issue

```bash
# Finish feature
git add . && git commit -m "feat: add user validation"
git push -u origin klauern/FSEC-1234

# Create PR (using gh or skill)
gh pr create

# After merge, clean up
wt remove klauern-FSEC-1234
bd close beads-abc
```

## Claude Session Workflows

### Dedicated Session Per Feature

**Terminal 1 - Feature A:**
```bash
cd ~/dev/guardians/zig-workspaces/zendesk-identity-governance/klauern-FSEC-1234
claude
# Work on FSEC-1234 in this session
```

**Terminal 2 - Feature B:**
```bash
cd ~/dev/guardians/zig-workspaces/zendesk-identity-governance/klauern-FSEC-5678
claude
# Work on FSEC-5678 in this session
```

### Quick Context Switch

```bash
# Current directory shows which worktree/feature
pwd
# /Users/nklauer/dev/guardians/zig-workspaces/zendesk-identity-governance/klauern-FSEC-1234

# Open new terminal, start different context
cd ../klauern-FSEC-5678
claude
```

### Review PR in Separate Session

```bash
# Keep your session running, open new terminal
wt add coworker/feature-to-review
cd ../coworker-feature-to-review
claude
# Review in isolation, close when done
```

## Advanced Workflows

### Create Worktree from Specific Commit

```bash
# First create worktree for main
wt add main

# Then checkout specific commit
cd ../main
git checkout abc1234
# Now you have a detached worktree at that commit
```

### Migrate Old Layout

If you have worktrees under `branches/` subdirectory:

```bash
# Preview migration
wt migrate

# Execute migration
wt migrate --apply
```

### Interactive Branch Selection

```bash
# Use fzf to pick from available remote branches
wt add
# Arrow keys to navigate, Enter to select
```

### Multiple Repos in Workspace

```bash
# ~/dev/guardians/zig-workspaces/
# ├── zendesk-identity-governance/
# │   ├── .bare/
# │   ├── master/
# │   └── klauern-FSEC-1234/
# ├── identity-sync-service/
# │   ├── .bare/
# │   ├── main/
# │   └── klauern-feature/
# └── zendesk-access-operator/
#     ├── .bare/
#     └── master/

# Navigate to specific repo/worktree
cd ~/dev/guardians/zig-workspaces/identity-sync-service/klauern-feature
```

## Common Patterns

### Feature Branch Naming Convention

```bash
# Pattern: username/TICKET-description
wt new klauern/FSEC-1234-add-oauth-support
wt new klauern/FSEC-5678-fix-race-condition

# Results in directories:
# klauern-FSEC-1234-add-oauth-support/
# klauern-FSEC-5678-fix-race-condition/
```

### Daily Sync Routine

```bash
# Start of day - update all worktrees
wt pull

# Check status
wt list
```

### Before Creating PR

```bash
# Sync with main
wt sync klauern-FSEC-1234

# Run tests
task test  # or appropriate test command

# Push and create PR
git push -u origin klauern/FSEC-1234
gh pr create
```

## Troubleshooting Examples

### Branch Exists But Can't Add Worktree

```bash
# Check if local tracking branch exists
git branch -a | grep feature-name

# If it exists locally but not linked:
wt add origin/feature-name

# If truly doesn't exist:
wt new feature-name  # Creates new branch
```

### Worktree Has Uncommitted Changes

```bash
# Can't remove worktree with changes
wt remove my-feature
# Error: has changes

# Options:
# 1. Commit changes
cd ../my-feature && git add . && git commit -m "WIP"

# 2. Discard changes (careful!)
cd ../my-feature && git checkout .

# 3. Force remove (last resort)
git worktree remove --force ../my-feature
```

### Rebase Conflict During Sync

```bash
wt sync klauern-FSEC-1234
# Rebase failed. You may need to resolve conflicts...

# Navigate to worktree
cd ../klauern-FSEC-1234

# Check status
git status

# Resolve conflicts in files
# Then continue
git add .
git rebase --continue

# Or abort if needed
git rebase --abort
```