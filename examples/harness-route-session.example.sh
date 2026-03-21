#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "usage: $0 <TASK_ID> [ROOT]" >&2
  exit 1
fi

TASK_ID="$1"
ROOT="${2:-$(pwd)}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

python3 "$SCRIPT_DIR/route-session.example.py" --root "$ROOT" --task-id "$TASK_ID"
