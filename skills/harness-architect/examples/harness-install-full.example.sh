#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "usage: $0 <PROJECT_ROOT>" >&2
  exit 1
fi

ROOT="$(cd "$1" && pwd)"
HARNESS_DIR="$ROOT/.harness"
BIN_DIR="$HARNESS_DIR/bin"
SCRIPTS_DIR="$HARNESS_DIR/scripts"
STATE_DIR="$HARNESS_DIR/state"
TEMPLATES_DIR="$HARNESS_DIR/templates"
REQUESTS=(
  "$HARNESS_DIR/replan-requests.json"
  "$HARNESS_DIR/stop-requests.json"
)
MANIFEST="$HARNESS_DIR/tooling-manifest.json"
EXAMPLES_DIR="$(cd "$(dirname "$0")" && pwd)"

mkdir -p "$BIN_DIR" "$SCRIPTS_DIR" "$STATE_DIR" "$TEMPLATES_DIR" "$HARNESS_DIR/drift-log" "$HARNESS_DIR/verification-rules"

install_file() {
  local src="$1"
  local dst="$2"
  cp "$EXAMPLES_DIR/$src" "$dst"
}

install_file "harness-query.example.sh" "$BIN_DIR/harness-query"
install_file "harness-dashboard.example.sh" "$BIN_DIR/harness-dashboard"
install_file "harness-status.example.sh" "$BIN_DIR/harness-status"
install_file "harness-watch.example.sh" "$BIN_DIR/harness-watch"
install_file "harness-render-prompt.example.sh" "$BIN_DIR/harness-render-prompt"
install_file "harness-route-session.example.sh" "$BIN_DIR/harness-route-session"
install_file "harness-prepare-worktree.example.sh" "$BIN_DIR/harness-prepare-worktree"
install_file "harness-diff-summary.example.sh" "$BIN_DIR/harness-diff-summary"

install_file "query-harness.example.py" "$SCRIPTS_DIR/query.py"
install_file "refresh-state.example.py" "$SCRIPTS_DIR/refresh-state.py"
install_file "status.example.py" "$SCRIPTS_DIR/status.py"
install_file "render-prompt.example.py" "$SCRIPTS_DIR/render-prompt.py"
install_file "route-session.example.py" "$SCRIPTS_DIR/route-session.py"
install_file "prepare-worktree.example.py" "$SCRIPTS_DIR/prepare-worktree.py"
install_file "diff-summary.example.py" "$SCRIPTS_DIR/diff-summary.py"

install_file "session-init.example.sh" "$HARNESS_DIR/session-init.sh"
install_file "AGENTS.example.md" "$TEMPLATES_DIR/AGENTS.template.md"

chmod +x \
  "$BIN_DIR/harness-query" \
  "$BIN_DIR/harness-dashboard" \
  "$BIN_DIR/harness-status" \
  "$BIN_DIR/harness-watch" \
  "$BIN_DIR/harness-render-prompt" \
  "$BIN_DIR/harness-route-session" \
  "$BIN_DIR/harness-prepare-worktree" \
  "$BIN_DIR/harness-diff-summary" \
  "$HARNESS_DIR/session-init.sh"

for request_path in "${REQUESTS[@]}"; do
  if [ ! -f "$request_path" ]; then
    printf '{\n  "schemaVersion": "1.0",\n  "generator": "harness-architect",\n  "generatedAt": null,\n  "requests": []\n}\n' > "$request_path"
  fi
done

cat > "$MANIFEST" <<'JSON'
{
  "schemaVersion": "1.0",
  "generator": "harness-architect",
  "generatedAt": "INSTALL_TIME",
  "installed": [
    {
      "name": "harness-query",
      "target": ".harness/bin/harness-query",
      "source": "examples/harness-query.example.sh"
    },
    {
      "name": "harness-dashboard",
      "target": ".harness/bin/harness-dashboard",
      "source": "examples/harness-dashboard.example.sh"
    },
    {
      "name": "harness-status",
      "target": ".harness/bin/harness-status",
      "source": "examples/harness-status.example.sh"
    },
    {
      "name": "harness-watch",
      "target": ".harness/bin/harness-watch",
      "source": "examples/harness-watch.example.sh"
    },
    {
      "name": "harness-render-prompt",
      "target": ".harness/bin/harness-render-prompt",
      "source": "examples/harness-render-prompt.example.sh"
    },
    {
      "name": "harness-route-session",
      "target": ".harness/bin/harness-route-session",
      "source": "examples/harness-route-session.example.sh"
    },
    {
      "name": "harness-prepare-worktree",
      "target": ".harness/bin/harness-prepare-worktree",
      "source": "examples/harness-prepare-worktree.example.sh"
    },
    {
      "name": "harness-diff-summary",
      "target": ".harness/bin/harness-diff-summary",
      "source": "examples/harness-diff-summary.example.sh"
    },
    {
      "name": "query.py",
      "target": ".harness/scripts/query.py",
      "source": "examples/query-harness.example.py"
    },
    {
      "name": "refresh-state.py",
      "target": ".harness/scripts/refresh-state.py",
      "source": "examples/refresh-state.example.py"
    },
    {
      "name": "status.py",
      "target": ".harness/scripts/status.py",
      "source": "examples/status.example.py"
    },
    {
      "name": "render-prompt.py",
      "target": ".harness/scripts/render-prompt.py",
      "source": "examples/render-prompt.example.py"
    },
    {
      "name": "route-session.py",
      "target": ".harness/scripts/route-session.py",
      "source": "examples/route-session.example.py"
    },
    {
      "name": "prepare-worktree.py",
      "target": ".harness/scripts/prepare-worktree.py",
      "source": "examples/prepare-worktree.example.py"
    },
    {
      "name": "diff-summary.py",
      "target": ".harness/scripts/diff-summary.py",
      "source": "examples/diff-summary.example.py"
    },
    {
      "name": "session-init.sh",
      "target": ".harness/session-init.sh",
      "source": "examples/session-init.example.sh"
    },
    {
      "name": "AGENTS.template.md",
      "target": ".harness/templates/AGENTS.template.md",
      "source": "examples/AGENTS.example.md"
    }
  ]
}
JSON

python3 - <<'PY' "$MANIFEST" "${REQUESTS[@]}"
import json
import sys
from datetime import datetime, timezone

manifest_path = sys.argv[1]
request_paths = sys.argv[2:]
timestamp = datetime.now(timezone.utc).astimezone().isoformat(timespec="seconds")

manifest = json.load(open(manifest_path))
manifest["generatedAt"] = timestamp
json.dump(manifest, open(manifest_path, "w"), ensure_ascii=False, indent=2)
open(manifest_path, "a").write("\n")

for path in request_paths:
    data = json.load(open(path))
    data["generatedAt"] = timestamp
    json.dump(data, open(path, "w"), ensure_ascii=False, indent=2)
    open(path, "a").write("\n")
PY

echo "installed full harness operator toolset into $HARNESS_DIR"
