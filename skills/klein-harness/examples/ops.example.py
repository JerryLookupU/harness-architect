#!/usr/bin/env python3
import argparse
import json
import subprocess
import sys
import time
from pathlib import Path

from runtime_common import ensure_runtime_scaffold, load_json, load_optional_json


def load_summary(path: Path, default=None):
    return load_optional_json(path, default if default is not None else {})


def collect_state(root: Path):
    files = ensure_runtime_scaffold(root, generator="harness-ops")
    state_dir = files["state_dir"]
    return {
        "files": files,
        "progress": load_summary(files["progress_state_path"], {}),
        "current": load_summary(state_dir / "current.json", {}),
        "runtime": load_summary(state_dir / "runtime.json", {}),
        "queue": load_summary(files["queue_summary_path"], {}),
        "task": load_summary(files["task_summary_path"], {}),
        "worker": load_summary(files["worker_summary_path"], {}),
        "daemon": load_summary(files["daemon_summary_path"], {}),
        "request": load_summary(files["request_summary_path"], {}),
        "lineage": load_summary(files["lineage_index_path"], {}),
        "feedback": load_summary(files["feedback_summary_path"], {}),
        "log": load_summary(files["log_index_path"], {}),
        "policy": load_summary(files["policy_summary_path"], {}),
        "research": load_summary(files["research_summary_path"], {}),
        "requestIndex": load_summary(files["request_index_path"], {}),
        "taskPool": load_summary(files["harness"] / "task-pool.json", {}),
    }


def top_view(state: dict):
    progress = state["progress"]
    queue = state["queue"]
    task = state["task"]
    worker = state["worker"]
    daemon = state["daemon"]
    runtime = state["runtime"]
    current = state["current"]
    return {
        "mode": progress.get("mode"),
        "planningStage": progress.get("planningStage"),
        "currentFocus": progress.get("currentFocus"),
        "currentRole": progress.get("currentRole"),
        "currentTaskId": current.get("currentTaskId"),
        "currentTaskTitle": current.get("currentTaskTitle"),
        "queueDepth": queue.get("queueDepth", 0),
        "activeTaskCount": runtime.get("activeTaskCount", 0),
        "workerCount": worker.get("workerCount", 0),
        "runtimeHealth": daemon.get("runtimeHealth"),
        "daemonStatus": daemon.get("status"),
        "dispatchBackendDefault": daemon.get("dispatchBackendDefault"),
        "dispatchBackendCounts": worker.get("dispatchBackendCounts", {}),
        "staleWorkerCount": worker.get("staleWorkerCount", 0),
        "blockedRouteCount": daemon.get("blockedRouteCount", 0),
        "compactLogCount": state["log"].get("compactLogCount", 0),
    }


def tasks_view(state: dict):
    task = state["task"]
    return {
        "taskStatusCounts": task.get("taskStatusCounts", {}),
        "taskKindCounts": task.get("taskKindCounts", {}),
        "roleHintCounts": task.get("roleHintCounts", {}),
        "dispatchableTaskIds": task.get("dispatchableTaskIds", []),
        "recoverableTaskIds": task.get("recoverableTaskIds", []),
        "activeTasks": task.get("activeTasks", []),
        "tasksWithRecentFailures": task.get("tasksWithRecentFailures", []),
    }


def task_view(state: dict, task_id: str):
    for task in state["taskPool"].get("tasks", []):
        if task.get("taskId") == task_id:
            return task
    raise KeyError(f"task not found: {task_id}")


def request_view(state: dict, request_id: str):
    for request in state["requestIndex"].get("requests", []):
        if request.get("requestId") == request_id:
            binding = next(
                (item for item in state["request"].get("bindings", []) if item.get("requestId") == request_id),
                None,
            )
            return {
                "request": request,
                "binding": binding,
            }
    raise KeyError(f"request not found: {request_id}")


def blockers_view(state: dict):
    return {
        "queueBlocked": state["queue"].get("recentBlockedRequests", []),
        "routeBlocked": state["task"].get("blockedRoutes", []),
        "logBlocked": state["log"].get("openBlockers", []),
    }


def logs_view(state: dict):
    return {
        "compactLogCount": state["log"].get("compactLogCount", 0),
        "recentHighSignalLogs": state["log"].get("recentHighSignalLogs", []),
        "openBlockers": state["log"].get("openBlockers", []),
        "recurringTags": state["log"].get("recurringTags", {}),
    }


def doctor_view(state: dict):
    files = state["files"]
    daemon = state["daemon"]
    worker = state["worker"]
    issues = []
    warnings = []
    if not files["progress_state_path"].exists():
        issues.append("missing .harness/state/progress.json")
    if not files["progress_markdown_path"].exists():
        warnings.append("missing .harness/progress.md")
    else:
        text = files["progress_markdown_path"].read_text(encoding="utf-8", errors="ignore")
        if "rendered from `.harness/state/progress.json`" not in text:
            warnings.append("progress.md does not advertise JSON-derived rendering")
    if daemon.get("status") == "running" and daemon.get("runtimeHealth") != "healthy":
        issues.append(f"daemon runtime health is {daemon.get('runtimeHealth')}")
    if daemon.get("dispatchBackendDefault") is None:
        issues.append("daemon dispatch backend default is missing")
    if worker.get("staleWorkerCount", 0) > 0:
        warnings.append(f"stale workers detected: {worker.get('staleWorkerCount')}")
    if daemon.get("status") == "running" and daemon.get("dispatchBackendDefault") == "tmux" and daemon.get("sessionAlive") is False:
        issues.append("daemon claims tmux backend but tmux session is not alive")
    return {
        "ok": not issues,
        "issues": issues,
        "warnings": warnings,
    }


def run_runner_command(root: Path, args: list[str]):
    runner_script = root / ".harness" / "scripts" / "runner.py"
    subprocess.run(["python3", str(runner_script), *args], check=True)


def print_text(title: str, payload):
    if title == "top":
        lines = [
            "== Harness Ops Top ==",
            f"mode: {payload.get('mode')}",
            f"planningStage: {payload.get('planningStage')}",
            f"focus: {payload.get('currentFocus')}",
            f"role: {payload.get('currentRole')}",
            f"currentTask: {payload.get('currentTaskId')} {payload.get('currentTaskTitle')}",
            f"queueDepth: {payload.get('queueDepth')}",
            f"activeTaskCount: {payload.get('activeTaskCount')}",
            f"workerCount: {payload.get('workerCount')}",
            f"runtimeHealth: {payload.get('runtimeHealth')}",
            f"daemonStatus: {payload.get('daemonStatus')}",
            f"dispatchBackendDefault: {payload.get('dispatchBackendDefault')}",
            f"dispatchBackendCounts: {payload.get('dispatchBackendCounts')}",
            f"staleWorkerCount: {payload.get('staleWorkerCount')}",
            f"blockedRouteCount: {payload.get('blockedRouteCount')}",
            f"compactLogCount: {payload.get('compactLogCount')}",
        ]
        return "\n".join(lines)
    if title == "queue":
        lines = [
            "== Harness Queue ==",
            f"queueDepth: {payload.get('queueDepth')}",
            f"queuedByKind: {payload.get('queuedByKind')}",
            f"queuedByPriority: {payload.get('queuedByPriority')}",
            f"oldestQueuedAt: {payload.get('oldestQueuedAt')}",
        ]
        for item in payload.get("recentQueuedRequests", [])[:10]:
            lines.append(f"- {item.get('requestId')} [{item.get('priority')}] {item.get('kind')} {item.get('goal')}")
        return "\n".join(lines)
    if title == "tasks":
        lines = [
            "== Harness Tasks ==",
            f"taskStatusCounts: {payload.get('taskStatusCounts')}",
            f"taskKindCounts: {payload.get('taskKindCounts')}",
            f"dispatchableTaskIds: {payload.get('dispatchableTaskIds')}",
            f"recoverableTaskIds: {payload.get('recoverableTaskIds')}",
        ]
        for item in payload.get("activeTasks", [])[:10]:
            lines.append(f"- {item.get('taskId')} [{item.get('status')}] {item.get('title')} backend={item.get('dispatchBackend')}")
        return "\n".join(lines)
    if title == "workers":
        lines = [
            "== Harness Workers ==",
            f"workerCount: {payload.get('workerCount')}",
            f"healthyWorkerCount: {payload.get('healthyWorkerCount')}",
            f"staleWorkerCount: {payload.get('staleWorkerCount')}",
            f"recoverableWorkerCount: {payload.get('recoverableWorkerCount')}",
            f"dispatchBackendCounts: {payload.get('dispatchBackendCounts')}",
        ]
        for item in payload.get("workerNodes", [])[:10]:
            lines.append(
                f"- {item.get('taskId')} node={item.get('nodeId')} backend={item.get('dispatchBackend')} nodeHealth={item.get('nodeHealth')} backendHealth={item.get('backendHealth')}"
            )
        return "\n".join(lines)
    if title == "daemon":
        lines = [
            "== Harness Daemon ==",
            f"status: {payload.get('status')}",
            f"runtimeHealth: {payload.get('runtimeHealth')}",
            f"dispatchBackendDefault: {payload.get('dispatchBackendDefault')}",
            f"sessionName: {payload.get('sessionName')}",
            f"sessionAlive: {payload.get('sessionAlive')}",
            f"intervalSeconds: {payload.get('intervalSeconds')}",
            f"lastTickAt: {payload.get('lastTickAt')}",
            f"lastTickAgeSeconds: {payload.get('lastTickAgeSeconds')}",
            f"lastError: {payload.get('lastError')}",
            f"workerBackendHealth: {payload.get('workerBackendHealth')}",
        ]
        return "\n".join(lines)
    if title == "blockers":
        lines = ["== Harness Blockers =="]
        for section, items in payload.items():
            lines.append(f"{section}:")
            if items:
                for item in items[:10]:
                    lines.append(f"- {json.dumps(item, ensure_ascii=False)}")
            else:
                lines.append("- none")
        return "\n".join(lines)
    if title == "logs":
        lines = [
            "== Harness Logs ==",
            f"compactLogCount: {payload.get('compactLogCount')}",
            f"recurringTags: {payload.get('recurringTags')}",
        ]
        for item in payload.get("recentHighSignalLogs", [])[:10]:
            lines.append(f"- {item.get('taskId')} [{item.get('severity')}] {item.get('path')}")
        return "\n".join(lines)
    if title == "doctor":
        lines = ["== Harness Doctor ==", f"ok: {payload.get('ok')}"]
        lines.append("issues:")
        if payload.get("issues"):
            lines.extend(f"- {item}" for item in payload["issues"])
        else:
            lines.append("- none")
        lines.append("warnings:")
        if payload.get("warnings"):
            lines.extend(f"- {item}" for item in payload["warnings"])
        else:
            lines.append("- none")
        return "\n".join(lines)
    return json.dumps(payload, ensure_ascii=False, indent=2)


def emit(payload, fmt: str, title: str):
    if fmt == "json":
        print(json.dumps(payload, ensure_ascii=False, indent=2))
    else:
        print(print_text(title, payload))


def main():
    parser = argparse.ArgumentParser(description="machine-first operator facade for Klein-Harness")
    parser.add_argument("root", help="project root")
    parser.add_argument("--format", choices=["text", "json"], default="text")
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("top")
    sub.add_parser("queue")
    sub.add_parser("tasks")
    p_task = sub.add_parser("task")
    p_task.add_argument("task_id")
    p_request = sub.add_parser("request")
    p_request.add_argument("request_id")
    sub.add_parser("workers")
    p_daemon = sub.add_parser("daemon")
    p_daemon.add_argument("action", choices=["status", "start", "stop", "restart"], nargs="?", default="status")
    p_daemon.add_argument("--interval", type=int, default=60)
    p_daemon.add_argument("--dispatch-mode", choices=["tmux", "print"], default="tmux")
    sub.add_parser("blockers")
    sub.add_parser("logs")
    p_watch = sub.add_parser("watch")
    p_watch.add_argument("--view", choices=["top", "queue", "workers", "daemon", "blockers", "logs"], default="top")
    p_watch.add_argument("--interval", type=int, default=2)
    p_watch.add_argument("--count", type=int, default=0)
    sub.add_parser("doctor")

    args = parser.parse_args()
    root = Path(args.root).resolve()

    if args.command == "daemon" and args.action in {"start", "stop", "restart"}:
        if args.action in {"stop", "restart"}:
            run_runner_command(root, ["daemon-stop", str(root)])
        if args.action in {"start", "restart"}:
            run_runner_command(root, ["daemon", str(root), "--interval", str(args.interval), "--dispatch-mode", args.dispatch_mode, "--replace"])
        state = collect_state(root)
        emit(state["daemon"], args.format, "daemon")
        return 0

    if args.command == "watch":
        remaining = args.count
        while True:
            state = collect_state(root)
            if args.view == "top":
                payload = top_view(state)
            elif args.view == "queue":
                payload = state["queue"]
            elif args.view == "workers":
                payload = state["worker"]
            elif args.view == "daemon":
                payload = state["daemon"]
            elif args.view == "blockers":
                payload = blockers_view(state)
            else:
                payload = logs_view(state)
            if args.format == "text":
                print("\033[2J\033[H", end="")
            emit(payload, args.format, args.view)
            if remaining == 1:
                break
            if remaining > 1:
                remaining -= 1
            time.sleep(args.interval)
        return 0

    state = collect_state(root)
    if args.command == "top":
        emit(top_view(state), args.format, "top")
    elif args.command == "queue":
        emit(state["queue"], args.format, "queue")
    elif args.command == "tasks":
        emit(tasks_view(state), args.format, "tasks")
    elif args.command == "task":
        emit(task_view(state, args.task_id), args.format, "task")
    elif args.command == "request":
        emit(request_view(state, args.request_id), args.format, "request")
    elif args.command == "workers":
        emit(state["worker"], args.format, "workers")
    elif args.command == "daemon":
        emit(state["daemon"], args.format, "daemon")
    elif args.command == "blockers":
        emit(blockers_view(state), args.format, "blockers")
    elif args.command == "logs":
        emit(logs_view(state), args.format, "logs")
    elif args.command == "doctor":
        emit(doctor_view(state), args.format, "doctor")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main() or 0)
    except Exception as exc:
        print(f"harness-ops failed: {exc}", file=sys.stderr)
        sys.exit(1)
