#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "usage: $0 <TASK_ID> [ROOT] [--write-back]" >&2
  exit 1
fi

TASK_ID="$1"
ROOT="${2:-$(pwd)}"
WRITE_BACK_FLAG="${3:-}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if [ "$WRITE_BACK_FLAG" = "--write-back" ]; then
  python3 "$SCRIPT_DIR/diff-summary.example.py" --root "$ROOT" --task-id "$TASK_ID" --write-back
else
  python3 "$SCRIPT_DIR/diff-summary.example.py" --root "$ROOT" --task-id "$TASK_ID"
fi
