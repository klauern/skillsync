#!/usr/bin/env bash
# next-task.sh — print the next open beads task in priority+dependency order.
#
# Usage: bash scripts/next-task.sh [--prompt]
#   --prompt  also print a suggested claude prompt for the task
#
# The script reads .beads/issues.jsonl, determines which issues are closed,
# then selects the highest-priority open issue whose dependencies are all closed.
# This lets each 5-hour session start fresh without manual triage.

set -euo pipefail

ISSUES_FILE="$(git rev-parse --show-toplevel)/.beads/issues.jsonl"
SHOW_PROMPT=false

for arg in "$@"; do
  case "$arg" in
    --prompt) SHOW_PROMPT=true ;;
  esac
done

if [[ ! -f "$ISSUES_FILE" ]]; then
  echo "ERROR: $ISSUES_FILE not found" >&2
  exit 1
fi

python3 - "$ISSUES_FILE" "$SHOW_PROMPT" << 'PYEOF'
import sys
import json
from collections import defaultdict

issues_file = sys.argv[1]
show_prompt = sys.argv[2].lower() == "true"

with open(issues_file) as f:
    issues = [json.loads(line) for line in f if line.strip()]

closed_ids = {i["id"] for i in issues if i.get("status") in ("closed", "done")}
open_issues = [i for i in issues if i.get("status") == "open"]

if not open_issues:
    print("All issues are closed. Nothing to do!")
    sys.exit(0)

def get_blockers(issue):
    """Return IDs of issues that must be closed before this one can start."""
    blockers = []
    for dep in issue.get("dependencies", []):
        dep_type = dep.get("type", "")
        dep_on = dep.get("depends_on_id", "")
        # "blocks" means this issue depends on the other (other must be done first)
        # "parent-child" means this is a child of the parent (parent doesn't block children)
        if dep_type == "blocks" and dep_on:
            blockers.append(dep_on)
    return blockers

def is_ready(issue):
    """True if all blocking dependencies are closed."""
    for blocker_id in get_blockers(issue):
        if blocker_id not in closed_ids:
            return False
    return True

ready = [i for i in open_issues if is_ready(i)]

if not ready:
    print("No ready tasks — all open tasks have unresolved dependencies.")
    print()
    print("Open issues with pending blockers:")
    for i in open_issues:
        blockers = [b for b in get_blockers(i) if b not in closed_ids]
        print(f"  {i['id']} (P{i.get('priority','?')}): {i['title']}")
        for b in blockers:
            print(f"    blocked by: {b}")
    sys.exit(1)

# Sort: lowest priority number first, then by id for stability
ready.sort(key=lambda i: (i.get("priority", 99), i["id"]))
task = ready[0]

print(f"┌─ Next Task {'─'*55}")
print(f"│  ID:       {task['id']}")
print(f"│  Priority: {task.get('priority', '?')}")
print(f"│  Type:     {task.get('issue_type', '?')}")
print(f"│  Title:    {task['title']}")
print(f"└─{'─'*65}")
print()

desc = task.get("description", "").strip()
if desc:
    print("Description:")
    for line in desc.splitlines():
        print(f"  {line}")
    print()

ac = task.get("acceptance_criteria", "").strip()
if ac:
    print("Acceptance criteria:")
    for line in ac.splitlines():
        print(f"  {line}")
    print()

# Show remaining open tasks grouped by priority
by_priority = defaultdict(list)
for i in open_issues:
    by_priority[i.get("priority", 99)].append(i)

print(f"Remaining open tasks ({len(open_issues)} total):")
for prio in sorted(by_priority.keys()):
    print(f"  Priority {prio}:")
    for i in by_priority[prio]:
        ready_marker = "✓" if is_ready(i) else "⏳"
        print(f"    {ready_marker} {i['id']}: {i['title']}")
print()

if show_prompt:
    print("─" * 67)
    print("Suggested claude prompt:")
    print()
    print(f'Implement beads task {task["id"]}: {task["title"]}')
    print()
    print(desc[:500] + ("..." if len(desc) > 500 else ""))
    if ac:
        print()
        print(f"Acceptance criteria: {ac}")
PYEOF
