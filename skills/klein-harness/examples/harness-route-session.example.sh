#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "usage: $0 <TASK_ID> [ROOT]" >&2
  exit 1
fi

TASK_ID="$1"
ROOT="${2:-$(pwd)}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PYTHON_ROUTE=""

if [[ "${KH_ENABLE_TWIN_NODE:-0}" = "1" ]] && command -v kh-orchestrator >/dev/null 2>&1; then
  exec kh-orchestrator route --root "$ROOT" --task-id "$TASK_ID"
fi

if [ -f "$SCRIPT_DIR/../scripts/route-session.py" ]; then
  PYTHON_ROUTE="$SCRIPT_DIR/../scripts/route-session.py"
elif [ -f "$SCRIPT_DIR/route-session.example.py" ]; then
  PYTHON_ROUTE="$SCRIPT_DIR/route-session.example.py"
else
  echo "route-session script not found" >&2
  exit 1
fi

python3 "$PYTHON_ROUTE" --root "$ROOT" --task-id "$TASK_ID"
