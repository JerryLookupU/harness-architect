#!/usr/bin/env python3
import json
import re
import sys
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path


def load_json(path: Path):
    return json.loads(path.read_text())


def write_json(path: Path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n")


def load_progress(path: Path):
    text = path.read_text()
    match = re.search(r"```json\s*(\{[\s\S]*?\})\s*```", text)
    if not match:
        raise ValueError(f"missing json block in {path}")
    return json.loads(match.group(1))


def now_iso():
    return datetime.now(timezone.utc).astimezone().isoformat(timespec="seconds")


def active_tasks(tasks):
    return [t for t in tasks if t.get("status") in {"active", "claimed", "in_progress"}]


def main():
    if len(sys.argv) != 2:
        print(f"usage: {sys.argv[0]} <ROOT>", file=sys.stderr)
        sys.exit(1)

    root = Path(sys.argv[1]).resolve()
    harness = root / ".harness"
    state_dir = harness / "state"

    progress = load_progress(harness / "progress.md")
    task_pool = load_json(harness / "task-pool.json")
    work_items = load_json(harness / "work-items.json")
    spec = load_json(harness / "spec.json")
    session_registry_path = harness / "session-registry.json"
    session_registry = load_json(session_registry_path) if session_registry_path.exists() else {}

    tasks = task_pool.get("tasks", [])
    items = work_items.get("items", [])
    active = active_tasks(tasks)

    current_state = {
        "schemaVersion": "1.0",
        "generator": "harness-architect",
        "generatedAt": now_iso(),
        "mode": progress.get("mode"),
        "planningStage": progress.get("planningStage"),
        "currentFocus": progress.get("currentFocus"),
        "currentRole": progress.get("currentRole"),
        "currentTaskId": progress.get("currentTaskId"),
        "currentTaskTitle": progress.get("currentTaskTitle"),
        "currentTaskSummary": progress.get("currentTaskSummary"),
        "blockers": progress.get("blockers", []),
        "nextActions": progress.get("nextActions", []),
        "lastAuditStatus": progress.get("lastAuditStatus"),
    }

    runtime_state = {
        "schemaVersion": "1.0",
        "generator": "harness-architect",
        "generatedAt": now_iso(),
        "orchestrationSessionId": session_registry.get("orchestrationSessionId"),
        "activeTaskCount": len(active),
        "activeWorkerCount": sum(1 for t in active if t.get("roleHint") == "worker"),
        "activeAuditWorkerCount": sum(1 for t in active if t.get("kind") == "audit"),
        "activeOrchestratorCount": sum(1 for t in active if t.get("kind") in {"orchestration", "replan", "rollback", "merge", "lease-recovery"}),
        "activeTasks": [
            {
                "taskId": t.get("taskId"),
                "kind": t.get("kind"),
                "roleHint": t.get("roleHint"),
                "workerMode": t.get("workerMode"),
                "title": t.get("title"),
                "summary": t.get("summary"),
                "nodeId": t.get("claim", {}).get("nodeId"),
                "boundSessionId": t.get("claim", {}).get("boundSessionId"),
                "branchName": t.get("branchName"),
                "worktreePath": t.get("worktreePath"),
            }
            for t in active
        ],
    }

    blocks = {}
    for block in spec.get("blocks", []):
        block_id = block.get("id")
        block_items = [w for w in items if set(w.get("featureIds", [])) & set(block.get("featureIds", []))]
        block_tasks = [t for t in tasks if t.get("blockId") == block_id]
        blocks[block_id] = {
            "title": block.get("title"),
            "status": block.get("status"),
            "featureIds": block.get("featureIds", []),
            "workItemIds": [w.get("id") for w in block_items],
            "taskIds": [t.get("taskId") for t in block_tasks],
        }

    blueprint_index = {
        "schemaVersion": "1.0",
        "generator": "harness-architect",
        "generatedAt": now_iso(),
        "specRevision": spec.get("specRevision"),
        "planningStage": spec.get("planningStage"),
        "objective": spec.get("objective"),
        "integrationBranch": task_pool.get("integrationBranch"),
        "taskStatusCounts": dict(Counter(t.get("status", "unknown") for t in tasks)),
        "blocks": blocks,
    }

    write_json(state_dir / "current.json", current_state)
    write_json(state_dir / "runtime.json", runtime_state)
    write_json(state_dir / "blueprint-index.json", blueprint_index)

    print(json.dumps({"ok": True, "stateDir": str(state_dir)}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"refresh-state example failed: {exc}", file=sys.stderr)
        sys.exit(1)
