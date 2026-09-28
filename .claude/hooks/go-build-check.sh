#!/usr/bin/env bash
# PostToolUse (Edit|Write): fast compile check after every .go edit.
# Non-blocking for non-.go files; feeds a build failure back to Claude via
# decision:"block" so it gets fixed before moving on, instead of piling up.
set -u

input=$(cat)
file=$(printf '%s' "$input" | jq -r '.tool_response.filePath // .tool_input.file_path // empty')

case "$file" in
  *.go) ;;
  *) exit 0 ;;
esac

root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
out=$(cd "$root" && go build ./... 2>&1)

if [ $? -eq 0 ]; then
  exit 0
fi

jq -n --arg reason "go build failed after editing $file:
$out" '{decision: "block", reason: $reason}'
exit 0
