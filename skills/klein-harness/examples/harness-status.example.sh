#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
if [ "$#" -ge 1 ]; then
  if [ "$1" = "--root" ]; then
    if [ "$#" -lt 2 ]; then
      echo "usage: $0 [ROOT|--root ROOT]" >&2
      exit 1
    fi
    ROOT="$2"
  elif [[ "$1" == --* ]]; then
    echo "usage: $0 [ROOT|--root ROOT]" >&2
    exit 1
  else
    ROOT="$1"
  fi
fi
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PYTHON_STATUS=""

if [ -f "$SCRIPT_DIR/../scripts/status.py" ]; then
  PYTHON_STATUS="$SCRIPT_DIR/../scripts/status.py"
elif [ -f "$SCRIPT_DIR/status.example.py" ]; then
  PYTHON_STATUS="$SCRIPT_DIR/status.example.py"
else
  echo "status script not found" >&2
  exit 1
fi

python3 "$PYTHON_STATUS" --root "$ROOT"
