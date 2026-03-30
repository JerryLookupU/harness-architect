#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF' >&2
usage: harness-size-tracker [--repo <path>] [--log <path>] [--warn-only]

Track harness repo growth (files / LOC / bytes) into a JSONL timeline.
Defaults:
  --repo <current working directory>
  --log  <repo>/docs/metrics/harness-code-growth.jsonl

Exit code:
  0 => within budget
  2 => budget exceeded when not --warn-only
EOF
}

REPO="$(pwd)"
LOG_PATH=""
WARN_ONLY=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      REPO="$2"
      shift 2
      ;;
    --log)
      LOG_PATH="$2"
      shift 2
      ;;
    --warn-only)
      WARN_ONLY=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown arg: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if ! git -C "$REPO" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "repo is not a git worktree: $REPO" >&2
  exit 1
fi

if [[ -z "$LOG_PATH" ]]; then
  LOG_PATH="$REPO/docs/metrics/harness-code-growth.jsonl"
fi

mkdir -p "$(dirname "$LOG_PATH")"
touch "$LOG_PATH"

MAX_TRACKED_FILES="${HARNESS_MAX_TRACKED_FILES:-140}"
MAX_TRACKED_LOC="${HARNESS_MAX_TRACKED_LOC:-70000}"
MAX_TRACKED_BYTES="${HARNESS_MAX_TRACKED_BYTES:-12000000}"

calc_counts() {
  local root="$1"
  local files_tmp
  files_tmp="$(mktemp)"
  (
    cd "$root"
    git ls-files -z > "$files_tmp"
  )

  local file_count
  file_count="$(tr -cd '\0' < "$files_tmp" | wc -c | tr -d '[:space:]')"
  if [[ "$file_count" == "0" ]]; then
    rm -f "$files_tmp"
    echo "0 0 0"
    return 0
  fi

  local loc_count byte_count
  loc_count="$(
    xargs -0 wc -l < "$files_tmp" \
      | awk 'END{print $1+0}'
  )"
  byte_count="$(
    xargs -0 wc -c < "$files_tmp" \
      | awk 'END{print $1+0}'
  )"

  rm -f "$files_tmp"
  echo "$file_count $loc_count $byte_count"
}

read -r tracked_files tracked_loc tracked_bytes <<< "$(calc_counts "$REPO")"
timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
branch="$(
  cd "$REPO"
  git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown"
)"
if [[ "$branch" == "HEAD" ]]; then
  branch="detached"
fi
commit="$(
  cd "$REPO"
  git rev-parse --short HEAD 2>/dev/null || echo "unknown"
)"

prev_files=0
prev_loc=0
prev_bytes=0
if [[ -s "$LOG_PATH" ]]; then
  last_line="$(tail -n 1 "$LOG_PATH")"
  prev_files="$(echo "$last_line" | sed -n 's/.*"trackedFiles":[[:space:]]*\([0-9][0-9]*\).*/\1/p')"
  prev_loc="$(echo "$last_line" | sed -n 's/.*"trackedLOC":[[:space:]]*\([0-9][0-9]*\).*/\1/p')"
  prev_bytes="$(echo "$last_line" | sed -n 's/.*"trackedBytes":[[:space:]]*\([0-9][0-9]*\).*/\1/p')"
  prev_files="${prev_files:-0}"
  prev_loc="${prev_loc:-0}"
  prev_bytes="${prev_bytes:-0}"
fi

delta_files=$((tracked_files - prev_files))
delta_loc=$((tracked_loc - prev_loc))
delta_bytes=$((tracked_bytes - prev_bytes))

status="ok"
reasons=()
if (( tracked_files > MAX_TRACKED_FILES )); then
  status="budget-exceeded"
  reasons+=("trackedFiles>${MAX_TRACKED_FILES}")
fi
if (( tracked_loc > MAX_TRACKED_LOC )); then
  status="budget-exceeded"
  reasons+=("trackedLOC>${MAX_TRACKED_LOC}")
fi
if (( tracked_bytes > MAX_TRACKED_BYTES )); then
  status="budget-exceeded"
  reasons+=("trackedBytes>${MAX_TRACKED_BYTES}")
fi

reason_json="[]"
if (( ${#reasons[@]} > 0 )); then
  reason_json="["
  for idx in "${!reasons[@]}"; do
    reason_json+="\"${reasons[$idx]}\""
    if (( idx + 1 < ${#reasons[@]} )); then
      reason_json+=","
    fi
  done
  reason_json+="]"
fi

printf '{"ts":"%s","branch":"%s","commit":"%s","trackedFiles":%s,"trackedLOC":%s,"trackedBytes":%s,"deltaFiles":%s,"deltaLOC":%s,"deltaBytes":%s,"budget":{"maxTrackedFiles":%s,"maxTrackedLOC":%s,"maxTrackedBytes":%s},"status":"%s","reasons":%s}\n' \
  "$timestamp" "$branch" "$commit" \
  "$tracked_files" "$tracked_loc" "$tracked_bytes" \
  "$delta_files" "$delta_loc" "$delta_bytes" \
  "$MAX_TRACKED_FILES" "$MAX_TRACKED_LOC" "$MAX_TRACKED_BYTES" \
  "$status" "$reason_json" >> "$LOG_PATH"

echo "repo: $REPO"
echo "log: $LOG_PATH"
echo "tracked files: $tracked_files (delta $delta_files)"
echo "tracked loc:   $tracked_loc (delta $delta_loc)"
echo "tracked bytes: $tracked_bytes (delta $delta_bytes)"
echo "budget: files<=$MAX_TRACKED_FILES loc<=$MAX_TRACKED_LOC bytes<=$MAX_TRACKED_BYTES"
echo "status: $status"
if (( ${#reasons[@]} > 0 )); then
  printf 'reasons: %s\n' "${reasons[*]}"
fi

if [[ "$status" != "ok" && "$WARN_ONLY" -ne 1 ]]; then
  exit 2
fi
