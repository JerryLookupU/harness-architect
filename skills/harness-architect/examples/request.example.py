#!/usr/bin/env python3
import argparse
import json
import sys
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path


def now_iso():
    return datetime.now(timezone.utc).astimezone().isoformat(timespec="seconds")


def load_json(path: Path):
    return json.loads(path.read_text())


def write_json(path: Path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n")


def load_optional_json(path: Path, default):
    if path.exists():
        return load_json(path)
    return default


def ensure_request_state(harness: Path):
    requests_dir = harness / "requests"
    state_dir = harness / "state"
    queue_path = requests_dir / "queue.jsonl"
    archive_dir = requests_dir / "archive"
    index_path = state_dir / "request-index.json"
    task_map_path = state_dir / "request-task-map.json"
    project_meta_path = harness / "project-meta.json"
    timestamp = now_iso()

    requests_dir.mkdir(parents=True, exist_ok=True)
    archive_dir.mkdir(parents=True, exist_ok=True)
    state_dir.mkdir(parents=True, exist_ok=True)
    queue_path.touch(exist_ok=True)

    if not index_path.exists():
        write_json(index_path, {
            "schemaVersion": "1.0",
            "generator": "harness-request",
            "generatedAt": timestamp,
            "nextSeq": 1,
            "requests": []
        })
    if not task_map_path.exists():
        write_json(task_map_path, {
            "schemaVersion": "1.0",
            "generator": "harness-request",
            "generatedAt": timestamp,
            "bindings": []
        })
    if not project_meta_path.exists():
        write_json(project_meta_path, {
            "schemaVersion": "1.0",
            "generator": "harness-request",
            "generatedAt": timestamp,
            "projectRoot": str(harness.parent.resolve()),
            "lifecycle": "initialized",
            "bootstrapStatus": "not_started",
            "requestQueueEnabled": True
        })

    return {
        "queue_path": queue_path,
        "archive_dir": archive_dir,
        "index_path": index_path,
        "task_map_path": task_map_path,
        "project_meta_path": project_meta_path,
    }


def load_runtime_snapshot(harness: Path):
    state_dir = harness / "state"
    current = load_optional_json(state_dir / "current.json", {})
    runtime = load_optional_json(state_dir / "runtime.json", {})
    return current, runtime


def normalize_context_paths(root: Path, values: list[str]) -> list[str]:
    result = []
    for value in values:
        path = Path(value)
        if not path.is_absolute():
            path = (root / value).resolve()
        else:
            path = path.resolve()
        if not path.exists():
            raise FileNotFoundError(f"context path not found: {value}")
        result.append(str(path))
    return result


def build_request_id(seq: int) -> str:
    return f"R-{seq:04d}"


def cmd_submit(args):
    root = Path(args.root).resolve()
    harness = root / ".harness"
    files = ensure_request_state(harness)
    index = load_json(files["index_path"])
    context_paths = normalize_context_paths(root, args.context or [])
    seq = int(index.get("nextSeq", 1))
    request_id = build_request_id(seq)
    request = {
        "requestId": request_id,
        "seq": seq,
        "source": args.source,
        "kind": args.kind,
        "goal": args.goal,
        "projectRoot": str(root),
        "contextPaths": context_paths,
        "threadKey": args.thread_key,
        "priority": args.priority,
        "scope": args.scope,
        "mergePolicy": args.merge_policy,
        "replyPolicy": args.reply_policy,
        "status": "queued",
        "createdAt": now_iso(),
    }

    with files["queue_path"].open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(request, ensure_ascii=False) + "\n")

    requests = index.get("requests", [])
    requests.append({
        **request,
        "summary": args.goal[:120],
    })
    index["generatedAt"] = now_iso()
    index["nextSeq"] = seq + 1
    index["requests"] = requests
    write_json(files["index_path"], index)

    print(json.dumps({
        "ok": True,
        "requestId": request_id,
        "status": "queued",
        "queuePath": str(files["queue_path"]),
    }, ensure_ascii=False, indent=2))
    return 0


def make_report_payload(root: Path, request_id: str | None):
    harness = root / ".harness"
    files = ensure_request_state(harness)
    index = load_json(files["index_path"])
    requests = index.get("requests", [])
    current, runtime = load_runtime_snapshot(harness)
    project_meta = load_optional_json(files["project_meta_path"], {})

    selected = None
    if request_id:
        for request in requests:
            if request.get("requestId") == request_id:
                selected = request
                break
        if not selected:
            raise KeyError(f"request not found: {request_id}")

    counts = dict(Counter(request.get("status", "unknown") for request in requests))
    payload = {
        "projectRoot": str(root),
        "projectLifecycle": project_meta.get("lifecycle"),
        "bootstrapStatus": project_meta.get("bootstrapStatus"),
        "requestCounts": counts,
        "totalRequests": len(requests),
        "selectedRequest": selected,
        "recentRequests": requests[-5:],
        "currentFocus": current.get("currentFocus"),
        "currentRole": current.get("currentRole"),
        "currentTaskId": current.get("currentTaskId"),
        "currentTaskTitle": current.get("currentTaskTitle"),
        "activeTaskCount": runtime.get("activeTaskCount", 0),
        "activeRunnerCount": runtime.get("activeRunnerCount", 0),
        "recoverableTaskCount": runtime.get("recoverableTaskCount", 0),
        "staleRunnerCount": runtime.get("staleRunnerCount", 0),
        "verifiedTaskCount": runtime.get("verifiedTaskCount", 0),
        "failingVerificationCount": runtime.get("failingVerificationCount", 0),
        "lastTickAt": runtime.get("lastTickAt"),
        "lastTrigger": runtime.get("lastTrigger"),
    }
    return payload


def format_report_text(payload: dict):
    lines = [
        f"project: {payload.get('projectRoot')}",
        f"lifecycle: {payload.get('projectLifecycle')}",
        f"bootstrapStatus: {payload.get('bootstrapStatus')}",
        f"totalRequests: {payload.get('totalRequests')}",
        f"requestCounts: {payload.get('requestCounts')}",
        f"currentFocus: {payload.get('currentFocus')}",
        f"currentRole: {payload.get('currentRole')}",
        f"currentTask: {payload.get('currentTaskId')} {payload.get('currentTaskTitle')}",
        f"activeTaskCount: {payload.get('activeTaskCount')}",
        f"activeRunnerCount: {payload.get('activeRunnerCount')}",
        f"recoverableTaskCount: {payload.get('recoverableTaskCount')}",
        f"staleRunnerCount: {payload.get('staleRunnerCount')}",
        f"verifiedTaskCount: {payload.get('verifiedTaskCount')}",
        f"failingVerificationCount: {payload.get('failingVerificationCount')}",
    ]
    if payload.get("selectedRequest"):
        selected = payload["selectedRequest"]
        lines.extend([
            "",
            f"selectedRequest: {selected.get('requestId')}",
            f"kind: {selected.get('kind')}",
            f"status: {selected.get('status')}",
            f"source: {selected.get('source')}",
            f"goal: {selected.get('goal')}",
        ])
    elif payload.get("recentRequests"):
        lines.append("")
        lines.append("recentRequests:")
        for request in payload["recentRequests"]:
            lines.append(
                f"- {request.get('requestId')} [{request.get('status')}] {request.get('kind')} {request.get('goal')}"
            )
    return "\n".join(lines)


def cmd_report(args):
    root = Path(args.root).resolve()
    payload = make_report_payload(root, args.request_id)
    if args.format == "text":
        print(format_report_text(payload))
    else:
        print(json.dumps(payload, ensure_ascii=False, indent=2))
    return 0


def main():
    parser = argparse.ArgumentParser(description="request intake and reporting")
    sub = parser.add_subparsers(dest="command")

    p_submit = sub.add_parser("submit", help="append a request into the project queue")
    p_submit.add_argument("--root", required=True)
    p_submit.add_argument("--kind", required=True)
    p_submit.add_argument("--goal", required=True)
    p_submit.add_argument("--source", default="shell")
    p_submit.add_argument("--context", action="append", default=[])
    p_submit.add_argument("--thread-key")
    p_submit.add_argument("--priority", default="P2")
    p_submit.add_argument("--scope", default="project")
    p_submit.add_argument("--merge-policy", default="append")
    p_submit.add_argument("--reply-policy", default="summary")

    p_report = sub.add_parser("report", help="summarize request queue and runtime state")
    p_report.add_argument("--root", required=True)
    p_report.add_argument("--request-id")
    p_report.add_argument("--format", default="text", choices=["text", "json"])

    args = parser.parse_args()
    if args.command == "submit":
        return cmd_submit(args)
    if args.command == "report":
        return cmd_report(args)
    parser.print_help()
    return 1


if __name__ == "__main__":
    try:
        sys.exit(main())
    except Exception as exc:
        print(f"request example failed: {exc}", file=sys.stderr)
        sys.exit(1)
