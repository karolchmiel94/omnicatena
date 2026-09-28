#!/usr/bin/env bash
# Stop: enforces CLAUDE.md rule 5 ("Tests Are Not Optional") mechanically.
# Skips entirely on master with no pending .go changes (pure Q&A turns).
# On a feature branch, or with uncommitted .go changes, runs build+vet+test
# and blocks Stop on failure so Claude keeps looping instead of handing back
# broken code.
set -u

root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
cd "$root"

branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
dirty_go=$(git status --porcelain -- '*.go' 2>/dev/null)

if [ "$branch" = "master" ] && [ -z "$dirty_go" ]; then
  exit 0
fi

block() {
  jq -n --arg reason "$1" '{decision: "block", reason: $reason}'
  exit 0
}

out=$(go build ./... 2>&1) || block "go build failed — fix before stopping:
$out"

out=$(go vet ./... 2>&1) || block "go vet failed — fix before stopping:
$out"

out=$(go test ./... 2>&1) || block "go test failed — fix before stopping (tests are not optional per CLAUDE.md):
$out"

exit 0
