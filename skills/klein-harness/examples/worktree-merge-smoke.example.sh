#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

TMP_ROOT="$(mktemp -d)"
CODEX_HOME_DIR="$TMP_ROOT/codex"
PROJECT_ROOT="$TMP_ROOT/worktree-merge-project"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT

export CODEX_HOME="$CODEX_HOME_DIR"
export PATH="$CODEX_HOME_DIR/bin:$PATH"

"$REPO_ROOT/install.sh" --dest "$CODEX_HOME_DIR/skills" --bin-dir "$CODEX_HOME_DIR/bin" --no-shell-rc --force >/dev/null
harness-init "$PROJECT_ROOT" >/dev/null

git -C "$PROJECT_ROOT" init -b main >/dev/null
git -C "$PROJECT_ROOT" config user.name "Klein Merge Smoke"
git -C "$PROJECT_ROOT" config user.email "smoke@example.com"
printf 'base\n' > "$PROJECT_ROOT/shared.txt"
printf '# tmp\n' > "$PROJECT_ROOT/README.md"
git -C "$PROJECT_ROOT" add shared.txt README.md
git -C "$PROJECT_ROOT" commit -m "baseline" >/dev/null
git -C "$PROJECT_ROOT" branch orch/spec-S-100 >/dev/null

cat > "$PROJECT_ROOT/.harness/spec.json" <<'EOF'
{"schemaVersion":"1.0","generator":"smoke","generatedAt":"2026-03-23T00:00:00+08:00","specRevision":"S-100","planningStage":"execution-ready","objective":"worktree merge smoke","blocks":[{"id":"TB-100","title":"merge block","status":"active","featureIds":["F-100"]}]}
EOF

cat > "$PROJECT_ROOT/.harness/features.json" <<'EOF'
{"schemaVersion":"1.0","generator":"smoke","generatedAt":"2026-03-23T00:00:00+08:00","features":[{"id":"F-100","title":"merge smoke","verificationStatus":"pass","priority":"P0"}]}
EOF

cat > "$PROJECT_ROOT/.harness/work-items.json" <<'EOF'
{"schemaVersion":"1.0","generator":"smoke","generatedAt":"2026-03-23T00:00:00+08:00","items":[{"id":"WI-100","kind":"feature","title":"clean merge","summary":"clean merge lane","status":"queued","priority":"P0","roleHint":"worker","featureIds":["F-100"],"dependsOn":[]},{"id":"WI-101","kind":"feature","title":"conflict merge","summary":"conflict merge lane","status":"queued","priority":"P0","roleHint":"worker","featureIds":["F-100"],"dependsOn":[]}]}
EOF

cat > "$PROJECT_ROOT/.harness/session-registry.json" <<'EOF'
{"schemaVersion":"1.0","generator":"smoke","generatedAt":"2026-03-23T00:00:00+08:00","orchestrationSessionId":"orch-session-100","orchestrationSessions":[{"sessionId":"orch-session-100","model":"gpt-5.4","role":"orchestrator","status":"active","purpose":"merge smoke","lastUsedAt":"2026-03-23T00:00:00+08:00"}],"sessions":[],"families":[],"routingDecisions":[],"activeBindings":[],"recoverableBindings":[],"lastCompletedByTask":{}}
EOF

cat > "$PROJECT_ROOT/.harness/verification-rules/manifest.json" <<'EOF'
{"schemaVersion":"1.0","generator":"smoke","generatedAt":"2026-03-23T00:00:00+08:00","rules":[{"id":"VR-100","title":"clean merge rule","type":"shell","costTier":"cheap","readOnlySafe":true,"exec":"test -f shared.txt"},{"id":"VR-101","title":"conflict merge rule","type":"shell","costTier":"cheap","readOnlySafe":true,"exec":"test -f shared.txt"}]}
EOF

cat > "$PROJECT_ROOT/.harness/task-pool.json" <<'EOF'
{
  "schemaVersion": "1.0",
  "generator": "smoke",
  "generatedAt": "2026-03-23T00:00:00+08:00",
  "integrationBranch": "orch/spec-S-100",
  "tasks": [
    {
      "taskId": "T-100",
      "workItemId": "WI-100",
      "blockId": "TB-100",
      "kind": "feature",
      "roleHint": "worker",
      "title": "Clean merge lane",
      "summary": "Dispatch in a dedicated worktree and merge cleanly.",
      "description": "Clean local merge path.",
      "status": "queued",
      "priority": "P0",
      "dependsOn": [],
      "planningStage": "execution-ready",
      "lineagePath": ["F-100", "WI-100", "T-100"],
      "baseRef": "refs/heads/orch/spec-S-100",
      "branchName": "task/T-100-clean",
      "worktreePath": ".worktrees/T-100-clean",
      "diffBase": "refs/heads/orch/spec-S-100",
      "diffSummary": "seed diff",
      "ownedPaths": ["shared.txt"],
      "verificationRuleIds": ["VR-100"],
      "routingModel": "gpt-5.4",
      "executionModel": "gpt-5.3-codex",
      "resumeStrategy": "fresh",
      "preferredResumeSessionId": null,
      "candidateResumeSessionIds": [],
      "lastKnownSessionId": null,
      "sessionFamilyId": "SF-F100-WI100",
      "cacheAffinityKey": "feature:F-100|parent:WI-100|role:worker",
      "dispatch": {
        "runner": "codex exec",
        "targetKind": "worker-node",
        "targetSelector": "tmux:worker-smoke",
        "entryRole": "worker",
        "taskContextId": "CTX-T-100",
        "worktreePath": ".worktrees/T-100-clean",
        "branchName": "task/T-100-clean",
        "baseRef": "refs/heads/orch/spec-S-100",
        "diffBase": "refs/heads/orch/spec-S-100",
        "commandProfile": {
          "standard": "codex exec --yolo -m gpt-5.3-codex",
          "localCompat": "codex exec --yolo -m gpt-5.3-codex"
        }
      },
      "handoff": {
        "mergeRequired": true
      },
      "claim": {
        "agentId": null
      }
    },
    {
      "taskId": "T-101",
      "workItemId": "WI-101",
      "blockId": "TB-100",
      "kind": "feature",
      "roleHint": "worker",
      "title": "Conflict merge lane",
      "summary": "Prepare a second stale branch and surface merge conflict as runtime signal.",
      "description": "Conflict local merge path.",
      "status": "queued",
      "priority": "P0",
      "dependsOn": [],
      "planningStage": "execution-ready",
      "lineagePath": ["F-100", "WI-101", "T-101"],
      "baseRef": "refs/heads/orch/spec-S-100",
      "branchName": "task/T-101-conflict",
      "worktreePath": ".worktrees/T-101-conflict",
      "diffBase": "refs/heads/orch/spec-S-100",
      "diffSummary": "seed diff",
      "ownedPaths": ["shared.txt"],
      "verificationRuleIds": ["VR-101"],
      "routingModel": "gpt-5.4",
      "executionModel": "gpt-5.3-codex",
      "resumeStrategy": "fresh",
      "preferredResumeSessionId": null,
      "candidateResumeSessionIds": [],
      "lastKnownSessionId": null,
      "sessionFamilyId": "SF-F100-WI101",
      "cacheAffinityKey": "feature:F-100|parent:WI-101|role:worker",
      "dispatch": {
        "runner": "codex exec",
        "targetKind": "worker-node",
        "targetSelector": "tmux:worker-smoke",
        "entryRole": "worker",
        "taskContextId": "CTX-T-101",
        "worktreePath": ".worktrees/T-101-conflict",
        "branchName": "task/T-101-conflict",
        "baseRef": "refs/heads/orch/spec-S-100",
        "diffBase": "refs/heads/orch/spec-S-100",
        "commandProfile": {
          "standard": "codex exec --yolo -m gpt-5.3-codex",
          "localCompat": "codex exec --yolo -m gpt-5.3-codex"
        }
      },
      "handoff": {
        "mergeRequired": true
      },
      "claim": {
        "agentId": null
      }
    }
  ]
}
EOF

RUN_JSON="$TMP_ROOT/run.json"

python3 "$PROJECT_ROOT/.harness/scripts/prepare-worktree.py" --root "$PROJECT_ROOT" --task-id T-100 --create --write-back >/dev/null
python3 "$PROJECT_ROOT/.harness/scripts/prepare-worktree.py" --root "$PROJECT_ROOT" --task-id T-101 --create --write-back >/dev/null
"$PROJECT_ROOT/.harness/bin/harness-runner" run T-100 "$PROJECT_ROOT" --dispatch-mode print > "$RUN_JSON"

printf 'clean merge\n' > "$PROJECT_ROOT/.worktrees/T-100-clean/shared.txt"
git -C "$PROJECT_ROOT/.worktrees/T-100-clean" add shared.txt
git -C "$PROJECT_ROOT/.worktrees/T-100-clean" commit -m "clean merge branch" >/dev/null

printf 'conflict merge\n' > "$PROJECT_ROOT/.worktrees/T-101-conflict/shared.txt"
git -C "$PROJECT_ROOT/.worktrees/T-101-conflict" add shared.txt
git -C "$PROJECT_ROOT/.worktrees/T-101-conflict" commit -m "conflict merge branch" >/dev/null

python3 - <<'PY' "$PROJECT_ROOT"
import json
import sys
from pathlib import Path

root = Path(sys.argv[1])
sys.path.insert(0, str(root / ".harness" / "scripts"))
from runtime_common import find_task, load_json, upsert_merge_queue_entry  # type: ignore

task_pool_path = root / ".harness" / "task-pool.json"
task_pool = load_json(task_pool_path)
for task_id in ("T-100", "T-101"):
    task = find_task(task_pool["tasks"], task_id)
    task["status"] = "merge_queued"
    task["mergeStatus"] = "merge_queued"
    task["verificationStatus"] = "pass"
    task["verificationSummary"] = "smoke merge queue"
json.dump(task_pool, task_pool_path.open("w"), ensure_ascii=False, indent=2)

task_pool = load_json(task_pool_path)
upsert_merge_queue_entry(root, find_task(task_pool["tasks"], "T-100"), None, generator="worktree-merge-smoke", merge_status="merge_queued")
upsert_merge_queue_entry(root, find_task(task_pool["tasks"], "T-101"), None, generator="worktree-merge-smoke", merge_status="merge_queued")
PY

python3 "$PROJECT_ROOT/.harness/scripts/runner.py" finalize "$PROJECT_ROOT" T-100 --tmux-session "print:T-100" --runner-status 0 >/dev/null
python3 "$PROJECT_ROOT/.harness/scripts/runner.py" finalize "$PROJECT_ROOT" T-101 --tmux-session "print:T-101" --runner-status 0 >/dev/null
python3 "$PROJECT_ROOT/.harness/scripts/refresh-state.py" "$PROJECT_ROOT" >/dev/null

python3 - <<'PY' "$PROJECT_ROOT" "$RUN_JSON"
import json
import sys
from pathlib import Path

root = Path(sys.argv[1])
run_payload = json.load(open(sys.argv[2]))
merge_summary = json.load(open(root / ".harness" / "state" / "merge-summary.json"))
merge_queue = json.load(open(root / ".harness" / "state" / "merge-queue.json"))
worktree_registry = json.load(open(root / ".harness" / "state" / "worktree-registry.json"))
request_index = json.load(open(root / ".harness" / "state" / "request-index.json"))
ops_worktrees = json.loads((root / ".harness" / "bin" / "harness-ops").read_text()) if False else None

assert run_payload["dispatched"]["executionCwd"].endswith(".worktrees/T-100-clean")
assert any(item["taskId"] == "T-100" and item["mergeStatus"] == "merged" for item in merge_queue["items"])
assert any(item["taskId"] == "T-101" and item["mergeStatus"] == "merge_conflict" for item in merge_queue["items"])
assert merge_summary["mergedCount"] >= 1
assert merge_summary["conflictCount"] >= 1
assert any(item["taskId"] == "T-101" for item in merge_summary["openConflicts"])
assert any(item["taskId"] == "T-100" and item["mergeStatus"] == "merged" for item in worktree_registry["worktrees"])
assert any(item["taskId"] == "T-101" and item["status"] == "merge_conflict" for item in worktree_registry["worktrees"])
assert any(item["kind"] in {"audit", "replan", "stop"} and item["source"] == "runtime:merge" for item in request_index["requests"])
PY

echo "worktree merge smoke passed"
