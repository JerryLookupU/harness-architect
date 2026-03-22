#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF' >&2
usage: harness-submit <ROOT> --kind <KIND> --goal <TEXT> [options...]

If the project has not been initialized yet, this command auto-runs harness-init first.
EOF
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

ROOT_INPUT="$1"
shift

if [[ "$ROOT_INPUT" = /* ]]; then
  ROOT="$ROOT_INPUT"
else
  ROOT="$(pwd)/$ROOT_INPUT"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INIT="$SCRIPT_DIR/harness-init"
if [[ ! -x "$INIT" && -x "$SCRIPT_DIR/harness-init.sh" ]]; then
  INIT="$SCRIPT_DIR/harness-init.sh"
fi

mkdir -p "$ROOT"
ROOT="$(cd "$ROOT" && pwd)"

LOCAL_SUBMIT="$ROOT/.harness/bin/harness-submit"

if [[ ! -x "$LOCAL_SUBMIT" ]]; then
  if [[ ! -x "$INIT" ]]; then
    echo "init helper not found: $INIT" >&2
    exit 1
  fi
  "$INIT" "$ROOT" >/dev/null
fi

if [[ ! -x "$LOCAL_SUBMIT" ]]; then
  echo "project-local submit helper missing: $LOCAL_SUBMIT" >&2
  exit 1
fi

exec "$LOCAL_SUBMIT" "$ROOT" "$@"
