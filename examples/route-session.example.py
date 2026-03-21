#!/usr/bin/env python3
import argparse
import json
import sys
from pathlib import Path


def load_json(path: Path):
    return json.loads(path.read_text())


def find_task(tasks, task_id: str):
    for task in tasks:
        if task.get("taskId") == task_id:
            return task
    raise KeyError(f"task not found: {task_id}")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", required=True, help="project root containing .harness/")
    parser.add_argument("--task-id", required=True, help="task id to route")
    args = parser.parse_args()

    root = Path(args.root).resolve()
    harness = root / ".harness"
    task_pool = load_json(harness / "task-pool.json")
    session_registry = load_json(harness / "session-registry.json")

    task = find_task(task_pool.get("tasks", []), args.task_id)
    orchestration_session_id = session_registry.get("orchestrationSessionId")
    if not orchestration_session_id:
        raise ValueError("missing orchestrationSessionId in session-registry.json")

    decision = {
        "taskId": task["taskId"],
        "routingModel": task.get("routingModel", "gpt-5.4"),
        "executionModel": task.get("executionModel", "gpt-5.3-codex"),
        "orchestrationSessionId": orchestration_session_id,
        "resumeStrategy": task.get("resumeStrategy", "fresh"),
        "preferredResumeSessionId": task.get("preferredResumeSessionId"),
        "candidateResumeSessionIds": task.get("candidateResumeSessionIds", []),
        "sessionFamilyId": task.get("sessionFamilyId"),
        "cacheAffinityKey": task.get("cacheAffinityKey"),
        "routingReason": task.get("routingReason"),
        "claimBinding": {
            "boundSessionId": task.get("claim", {}).get("boundSessionId"),
            "boundResumeStrategy": task.get("claim", {}).get("boundResumeStrategy"),
            "boundFromTaskId": task.get("claim", {}).get("boundFromTaskId"),
        },
        "commandProfile": task.get("dispatch", {}).get("commandProfile", {}),
    }

    if decision["resumeStrategy"] == "resume" and not decision["preferredResumeSessionId"]:
        raise ValueError("resumeStrategy=resume but preferredResumeSessionId is missing")

    print(json.dumps(decision, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"route-session example failed: {exc}", file=sys.stderr)
        sys.exit(1)
