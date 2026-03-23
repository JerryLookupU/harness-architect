#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ]; then
  echo "usage: $0 <ROOT> <top|queue|tasks|task|request|workers|daemon|blockers|logs|watch|doctor> [args...]" >&2
  exit 1
fi

ROOT="$1"
shift
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PYTHON_OPS=""

if [ -f "$SCRIPT_DIR/../scripts/ops.py" ]; then
  PYTHON_OPS="$SCRIPT_DIR/../scripts/ops.py"
elif [ -f "$SCRIPT_DIR/ops.example.py" ]; then
  PYTHON_OPS="$SCRIPT_DIR/ops.example.py"
else
  echo "ops script not found" >&2
  exit 1
fi

python3 "$PYTHON_OPS" "$ROOT" "$@"
