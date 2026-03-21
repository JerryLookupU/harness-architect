#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 4 ]; then
  echo "usage: $0 <TASK_ID> <ROOT> <ROLE> <STAGE>" >&2
  echo "example: $0 T-002 . worker start" >&2
  exit 1
fi

TASK_ID="$1"
ROOT="$2"
ROLE="$3"
STAGE="$4"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

python3 "$SCRIPT_DIR/render-prompt.example.py" \
  --root "$ROOT" \
  --task-id "$TASK_ID" \
  --role "$ROLE" \
  --stage "$STAGE"
