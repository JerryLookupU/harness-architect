#!/usr/bin/env python3
import argparse
import json
import re
import sys
from collections import Counter
from pathlib import Path


def load_json(path: Path):
    return json.loads(path.read_text())


def load_progress(path: Path):
    text = path.read_text()
    match = re.search(r"```json\s*(\{[\s\S]*?\})\s*```", text)
    if not match:
        raise ValueError(f"missing json block in {path}")
    return json.loads(match.group(1))


def load_optional_json(path: Path):
    if path.exists():
        return load_json(path)
    return None


def summarize_tasks(tasks):
    return dict(Counter(task.get("status", "unknown") for task in tasks))


def summarize_features(features):
    return dict(Counter(feature.get("verificationStatus", feature.get("status", "unknown")) for feature in features))


def active_tasks(tasks):
    return [t for t in tasks if t.get("status") in {"active", "claimed", "in_progress"}]


def find_task(tasks, task_id: str):
    for task in tasks:
        if task.get("taskId") == task_id:
            return task
    raise KeyError(f"task not found: {task_id}")


def make_overview(progress, tasks, work_items, features, session_registry, runtime_state=None):
    active = active_tasks(tasks)
    return {
        "mode": progress.get("mode"),
        "planningStage": progress.get("planningStage"),
        "currentFocus": progress.get("currentFocus"),
        "currentRole": progress.get("currentRole"),
        "currentTaskId": progress.get("currentTaskId"),
        "currentTaskTitle": progress.get("currentTaskTitle"),
        "currentTaskSummary": progress.get("currentTaskSummary"),
        "blockerCount": len(progress.get("blockers", [])),
        "nextActions": progress.get("nextActions", []),
        "taskStatusCounts": summarize_tasks(tasks),
        "workItemStatusCounts": dict(Counter(item.get("status", "unknown") for item in work_items)),
        "featureStatusCounts": summarize_features(features),
        "activeTaskCount": (runtime_state or {}).get("activeTaskCount", len(active)),
        "activeRunnerCount": (runtime_state or {}).get("activeRunnerCount", 0),
        "recoverableTaskCount": (runtime_state or {}).get("recoverableTaskCount", 0),
        "staleRunnerCount": (runtime_state or {}).get("staleRunnerCount", 0),
        "orchestrationSessionId": (runtime_state or {}).get("orchestrationSessionId", session_registry.get("orchestrationSessionId")),
    }


def make_progress(progress, tasks, work_items, features):
    return {
        "mode": progress.get("mode"),
        "planningStage": progress.get("planningStage"),
        "currentFocus": progress.get("currentFocus"),
        "lastAuditStatus": progress.get("lastAuditStatus"),
        "blockers": progress.get("blockers", []),
        "taskStatusCounts": summarize_tasks(tasks),
        "workItemStatusCounts": dict(Counter(item.get("status", "unknown") for item in work_items)),
        "featureStatusCounts": summarize_features(features),
        "claimSummary": progress.get("claimSummary", {}),
    }


def make_current(progress, tasks, session_registry):
    active = active_tasks(tasks)
    current = []
    for task in active:
        current.append(
            {
                "taskId": task.get("taskId"),
                "kind": task.get("kind"),
                "roleHint": task.get("roleHint"),
                "workerMode": task.get("workerMode"),
                "title": task.get("title"),
                "summary": task.get("summary"),
                "nodeId": task.get("claim", {}).get("nodeId"),
                "boundSessionId": task.get("claim", {}).get("boundSessionId"),
                "worktreePath": task.get("worktreePath"),
                "branchName": task.get("branchName"),
            }
        )
    return {
        "currentTaskId": progress.get("currentTaskId"),
        "currentTaskTitle": progress.get("currentTaskTitle"),
        "currentTaskSummary": progress.get("currentTaskSummary"),
        "currentRole": progress.get("currentRole"),
        "activeTasks": current,
        "orchestrationSessionId": session_registry.get("orchestrationSessionId"),
    }


def make_blueprint(spec, task_pool, work_items, tasks):
    by_block = {}
    for block in spec.get("blocks", []):
        block_id = block.get("id")
        block_tasks = [t for t in tasks if t.get("blockId") == block_id]
        block_items = [w for w in work_items if set(w.get("featureIds", [])) & set(block.get("featureIds", []))]
        by_block[block_id] = {
            "title": block.get("title"),
            "goal": block.get("goal"),
            "status": block.get("status"),
            "featureIds": block.get("featureIds", []),
            "ownedPaths": block.get("ownedPaths", []),
            "verificationRuleIds": block.get("verificationRuleIds", []),
            "workItemIds": [w.get("id") for w in block_items],
            "taskIds": [t.get("taskId") for t in block_tasks],
        }
    return {
        "specRevision": spec.get("specRevision"),
        "planningStage": spec.get("planningStage"),
        "objective": spec.get("objective"),
        "integrationBranch": task_pool.get("integrationBranch"),
        "blocks": by_block,
    }


def make_task_view(task):
    return {
        "taskId": task.get("taskId"),
        "kind": task.get("kind"),
        "roleHint": task.get("roleHint"),
        "workerMode": task.get("workerMode"),
        "title": task.get("title"),
        "summary": task.get("summary"),
        "description": task.get("description"),
        "status": task.get("status"),
        "planningStage": task.get("planningStage"),
        "dependsOn": task.get("dependsOn", []),
        "ownedPaths": task.get("ownedPaths", []),
        "verificationRuleIds": task.get("verificationRuleIds", []),
        "branchName": task.get("branchName"),
        "worktreePath": task.get("worktreePath"),
        "diffBase": task.get("diffBase"),
        "diffSummary": task.get("diffSummary"),
        "resumeStrategy": task.get("resumeStrategy"),
        "preferredResumeSessionId": task.get("preferredResumeSessionId"),
        "verificationStatus": task.get("verificationStatus"),
        "verificationSummary": task.get("verificationSummary"),
        "verificationResultPath": task.get("verificationResultPath"),
        "claim": task.get("claim", {}),
        "handoff": task.get("handoff", {}),
    }


def format_text(view, payload):
    if view == "overview":
        lines = [
            f"mode: {payload.get('mode')}",
            f"planningStage: {payload.get('planningStage')}",
            f"focus: {payload.get('currentFocus')}",
            f"role: {payload.get('currentRole')}",
            f"current: {payload.get('currentTaskId')} {payload.get('currentTaskTitle')}",
            f"summary: {payload.get('currentTaskSummary')}",
            f"activeTaskCount: {payload.get('activeTaskCount')}",
            f"activeRunners: {payload.get('activeRunnerCount', 0)}",
            f"recoverableTasks: {payload.get('recoverableTaskCount', 0)}",
        ]
        return "\n".join(lines)
    if view == "progress":
        return "\n".join(
            [
                f"mode: {payload.get('mode')}",
                f"planningStage: {payload.get('planningStage')}",
                f"lastAuditStatus: {payload.get('lastAuditStatus')}",
                f"taskStatusCounts: {payload.get('taskStatusCounts')}",
                f"workItemStatusCounts: {payload.get('workItemStatusCounts')}",
                f"featureStatusCounts: {payload.get('featureStatusCounts')}",
            ]
        )
    if view == "current":
        lines = [
            f"currentRole: {payload.get('currentRole')}",
            f"current: {payload.get('currentTaskId')} {payload.get('currentTaskTitle')}",
            f"summary: {payload.get('currentTaskSummary')}",
            "activeTasks:",
        ]
        for item in payload.get("activeTasks", []):
            lines.append(f"- {item['taskId']} [{item['kind']}] {item['title']} @ {item.get('nodeId')}")
        return "\n".join(lines)
    if view == "blueprint":
        lines = [
            f"specRevision: {payload.get('specRevision')}",
            f"planningStage: {payload.get('planningStage')}",
            f"objective: {payload.get('objective')}",
            "blocks:",
        ]
        for block_id, block in payload.get("blocks", {}).items():
            lines.append(f"- {block_id} {block['title']} status={block['status']} tasks={block['taskIds']}")
        return "\n".join(lines)
    if view == "task":
        return "\n".join(
            [
                f"taskId: {payload.get('taskId')}",
                f"kind: {payload.get('kind')}",
                f"roleHint: {payload.get('roleHint')}",
                f"workerMode: {payload.get('workerMode')}",
                f"title: {payload.get('title')}",
                f"summary: {payload.get('summary')}",
                f"status: {payload.get('status')}",
                f"planningStage: {payload.get('planningStage')}",
                f"nodeId: {payload.get('claim', {}).get('nodeId')}",
                f"boundSessionId: {payload.get('claim', {}).get('boundSessionId')}",
                f"branchName: {payload.get('branchName')}",
                f"worktreePath: {payload.get('worktreePath')}",
                f"diffBase: {payload.get('diffBase')}",
                f"diffSummary: {payload.get('diffSummary')}",
                f"verificationStatus: {payload.get('verificationStatus')}",
                f"verificationSummary: {payload.get('verificationSummary')}",
                f"verificationResultPath: {payload.get('verificationResultPath')}",
            ]
        )
    return json.dumps(payload, ensure_ascii=False, indent=2)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", required=True)
    parser.add_argument("--view", required=True, choices=["overview", "progress", "current", "blueprint", "task"])
    parser.add_argument("--task-id")
    parser.add_argument("--format", default="json", choices=["json", "text"])
    args = parser.parse_args()

    root = Path(args.root).resolve()
    harness = root / ".harness"
    state_dir = harness / "state"
    current_state = load_optional_json(state_dir / "current.json")
    runtime_state = load_optional_json(state_dir / "runtime.json")
    blueprint_state = load_optional_json(state_dir / "blueprint-index.json")

    progress = current_state or load_progress(harness / "progress.md")
    task_pool = load_json(harness / "task-pool.json")
    work_items = load_json(harness / "work-items.json")
    features = load_json(harness / "features.json")
    spec = load_json(harness / "spec.json")
    session_registry_path = harness / "session-registry.json"
    session_registry = load_json(session_registry_path) if session_registry_path.exists() else {}

    tasks = task_pool.get("tasks", [])
    items = work_items.get("items", [])
    feature_items = features.get("features", [])

    if args.view == "overview":
        if current_state and runtime_state:
            payload = make_overview(progress, tasks, items, feature_items, session_registry, runtime_state)
        else:
            payload = make_overview(progress, tasks, items, feature_items, session_registry)
    elif args.view == "progress":
        payload = make_progress(progress, tasks, items, feature_items)
    elif args.view == "current":
        if current_state and runtime_state:
            payload = {
                "currentTaskId": current_state.get("currentTaskId"),
                "currentTaskTitle": current_state.get("currentTaskTitle"),
                "currentTaskSummary": current_state.get("currentTaskSummary"),
                "currentRole": current_state.get("currentRole"),
                "activeTasks": runtime_state.get("activeTasks", []),
                "activeRuns": runtime_state.get("activeRuns", []),
                "orchestrationSessionId": runtime_state.get("orchestrationSessionId"),
            }
        else:
            payload = make_current(progress, tasks, session_registry)
    elif args.view == "task":
        if not args.task_id:
            raise ValueError("--task-id is required for view=task")
        payload = make_task_view(find_task(tasks, args.task_id))
    else:
        payload = blueprint_state or make_blueprint(spec, task_pool, items, tasks)

    if args.format == "text":
        print(format_text(args.view, payload))
    else:
        print(json.dumps(payload, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"query-harness example failed: {exc}", file=sys.stderr)
        sys.exit(1)
