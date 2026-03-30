#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT="text"
WINDOW_DAYS=30

while [[ $# -gt 0 ]]; do
  case "$1" in
    --json)
      OUTPUT="json"
      shift
      ;;
    --window-days)
      WINDOW_DAYS="${2:-30}"
      shift 2
      ;;
    *)
      echo "usage: $0 [--json] [--window-days N]" >&2
      exit 1
      ;;
  esac
done

cd "$ROOT"

if ! command -v git >/dev/null 2>&1; then
  echo "git not found" >&2
  exit 1
fi

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "not a git repository" >&2
  exit 1
fi

current_commit="$(git rev-parse HEAD)"
tracked_files="$(git ls-files | wc -l | tr -d ' ')"
tracked_loc="$(git ls-files '*.py' '*.sh' '*.md' '*.json' | xargs cat 2>/dev/null | wc -l | tr -d ' ')"
script_entries="$(git ls-files 'scripts/*.sh' 'skills/klein-harness/examples/*.sh' | wc -l | tr -d ' ')"

baseline_commit="$(git rev-list -1 --before="${WINDOW_DAYS} days ago" HEAD || true)"
if [[ -z "$baseline_commit" ]]; then
  baseline_commit="$(git rev-list --max-parents=0 HEAD | tail -n 1)"
fi

baseline_files="$(git ls-tree -r --name-only "$baseline_commit" | wc -l | tr -d ' ')"
baseline_loc="$(
  git ls-tree -r --name-only "$baseline_commit" \
    | rg -N '\.(py|sh|md|json)$' \
    | xargs -I{} git show "${baseline_commit}:{}" 2>/dev/null \
    | wc -l \
    | tr -d ' '
)"

files_delta=$((tracked_files - baseline_files))
loc_delta=$((tracked_loc - baseline_loc))
files_per_day="$(awk -v d="$files_delta" -v w="$WINDOW_DAYS" 'BEGIN { printf "%.2f", d / w }')"
loc_per_day="$(awk -v d="$loc_delta" -v w="$WINDOW_DAYS" 'BEGIN { printf "%.2f", d / w }')"

if [[ "$OUTPUT" == "json" ]]; then
  cat <<JSON
{
  "repoRoot": "$ROOT",
  "head": "$current_commit",
  "windowDays": $WINDOW_DAYS,
  "trackedFiles": $tracked_files,
  "trackedLoc": $tracked_loc,
  "scriptEntries": $script_entries,
  "baselineCommit": "$baseline_commit",
  "baselineFiles": $baseline_files,
  "baselineLoc": $baseline_loc,
  "filesDelta": $files_delta,
  "locDelta": $loc_delta,
  "filesPerDay${WINDOW_DAYS}d": $files_per_day,
  "locPerDay${WINDOW_DAYS}d": $loc_per_day
}
JSON
else
  cat <<TEXT
repoRoot: $ROOT
head: $current_commit
windowDays: $WINDOW_DAYS
trackedFiles: $tracked_files
trackedLoc: $tracked_loc
scriptEntries: $script_entries
baselineCommit: $baseline_commit
baselineFiles: $baseline_files
baselineLoc: $baseline_loc
filesDelta: $files_delta
locDelta: $loc_delta
filesPerDay${WINDOW_DAYS}d: $files_per_day
locPerDay${WINDOW_DAYS}d: $loc_per_day
TEXT
fi
