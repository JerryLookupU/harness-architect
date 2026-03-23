# A2A Protocol v1

This file defines the structured replayable protocol between:

- `orchestrator-node`
- `worker-supervisor-node`

The default schema is fixed to `a2a.v1`.
Coordination must happen through JSON events and revisioned snapshots, not fuzzy prompt summaries.

## Envelope

Every event uses the same outer envelope:

```json
{
  "schemaVersion": "a2a.v1",
  "messageId": "msg_20260323_000001",
  "kind": "dispatch.issued",
  "idempotencyKey": "dispatch:task_123:epoch_7:attempt_2",
  "traceId": "req_456",
  "causationId": "route_789",
  "from": "orchestrator-node",
  "to": "worker-supervisor-node",
  "createdAt": "2026-03-23T10:15:30Z",
  "requestId": "req_456",
  "taskId": "task_123",
  "planEpoch": 7,
  "attempt": 2,
  "sessionId": null,
  "workerId": null,
  "leaseId": null,
  "reasonCodes": ["checkpoint_fresh", "owned_paths_valid"],
  "payload": {}
}
```

## Envelope Semantics

- `schemaVersion` must equal `a2a.v1`
- `messageId` is unique per emitted message
- `kind` is the semantic event name
- `idempotencyKey` is the dedupe key
- `traceId` follows request lineage across nodes
- `causationId` points to the decision or event that caused this message
- `planEpoch` tracks the valid plan revision for the thread
- `attempt` increments for each repeated execution of the same task/epoch
- `reasonCodes` are mandatory on decision or state-transition events

## Event Kinds

The initial set is fixed to:

- `request.submitted`
- `request.fused`
- `task.bound`
- `route.decided`
- `dispatch.issued`
- `worker.claimed`
- `worker.heartbeat`
- `worker.checkpoint`
- `worker.outcome`
- `verification.completed`
- `task.completed`
- `task.blocked`
- `task.cancelled`
- `replan.emitted`
- `audit.emitted`
- `rca.allocated`
- `merge.enqueued`
- `merge.completed`

New kinds are allowed only by explicit schema revision.

## Three Hard Guarantees

### 1. Idempotency

- dedupe key is `kind + idempotencyKey`
- re-delivery must not create duplicate dispatch, duplicate claim, or duplicate completion
- if the same idempotency key arrives again, the existing semantic result must be reused

### 2. Recoverability

- append-only A2A log plus current snapshots must rebuild runtime state
- no transition may live only in prompt text
- no transition may live only in a live `tmux` session

### 3. Auditability

- terminal and routing events must carry `causationId`
- decision-bearing events must carry `reasonCodes`
- operators must be able to answer why a route, stop, resume, replan, or completion happened

## Snapshot Discipline

New twin-node snapshots must include:

- `schemaVersion`
- `generator`
- `generatedAt`
- `revision`

Snapshot writes use CAS semantics:

1. read current snapshot revision
2. compare with expected revision
3. if mismatch, reject write with conflict
4. on success, increment `revision`

This rule applies at minimum to:

- `.harness/state/lease-summary.json`
- `.harness/state/dispatch-summary.json`
- `.harness/state/checkpoint-summary.json`

## Required Payload Schemas

### `route.decided`

```json
{
  "route": "resume",
  "dispatchReady": true,
  "reasonCodes": ["checkpoint_fresh", "owned_paths_valid"],
  "requiredSummaryVersion": "state.v12",
  "resumeSessionId": "sess_42",
  "worktreePath": "/path/to/worktree",
  "ownedPaths": ["internal/worker", "docs/runtime"]
}
```

Rules:

- only `orchestrator-node` emits this
- only route gate decides `resume` vs `dispatch`
- `dispatchReady = false` means worker supervisor must not start a burst

### `dispatch.issued`

```json
{
  "workerClass": "codex-go",
  "cwd": "/path/to/worktree",
  "command": "codex exec ...",
  "promptRef": "prompts/worker-burst.md",
  "budget": {
    "maxTurns": 8,
    "maxMinutes": 20,
    "maxToolCalls": 30
  },
  "leaseTtlSec": 1800
}
```

Rules:

- must be causally downstream of `route.decided`
- `cwd` must match approved worktree binding
- `budget` makes execution bounded by construction

### `worker.claimed`

```json
{
  "dispatchId": "dispatch_task_123_7_2",
  "workerId": "worker-02",
  "leaseId": "lease_task_123_2",
  "claimedAt": "2026-03-23T10:18:00Z"
}
```

Rules:

- emitted when a worker supervisor successfully claims a dispatch
- duplicate claim for the same active task must be rejected unless it is the same lease replay

### `worker.heartbeat`

```json
{
  "dispatchId": "dispatch_task_123_7_2",
  "workerId": "worker-02",
  "leaseId": "lease_task_123_2",
  "phase": "running",
  "summary": "burst active",
  "expiresAt": "2026-03-23T10:48:00Z"
}
```

Rules:

- emitted for lease renewals and bounded execution liveness
- absence beyond TTL allows stale lease recovery

### `worker.checkpoint`

```json
{
  "dispatchId": "dispatch_task_123_7_2",
  "checkpointRef": ".harness/checkpoints/task_123/attempt_2.json",
  "status": "checkpointed",
  "summary": "safe resume boundary written"
}
```

Rules:

- emitted only when a resumable checkpoint artifact exists
- checkpoint reference must be repo-local and durable

### `worker.outcome`

```json
{
  "status": "needs_replan",
  "summary": "request focus changed after validation failure",
  "checkpointRef": ".harness/checkpoints/task_123/attempt_2.json",
  "diffStats": {
    "filesChanged": 4,
    "insertions": 120,
    "deletions": 18
  },
  "artifacts": [
    "docs/dev/a2a-protocol-v1.md",
    "internal/worker/supervisor.go"
  ],
  "nextSuggestedKind": "replan"
}
```

Rules:

- emitted after every bounded burst finishes or aborts
- `nextSuggestedKind` is advisory only
- route ownership remains with orchestrator

### `verification.completed`

```json
{
  "status": "failed",
  "summary": "verification evidence rejected current output",
  "verificationResultPath": ".harness/verification/task_123/result.json"
}
```

Rules:

- verification result is evidence, not a worker opinion
- `task.completed` may only follow evidence-compatible verification

### `replan.emitted`

```json
{
  "sourceTaskId": "task_123",
  "followUpKind": "replan",
  "summary": "verification failed after scope change"
}
```

### `rca.allocated`

```json
{
  "sourceTaskId": "task_123",
  "taxonomy": "verification_guardrail",
  "ownerRole": "architect/orchestrator",
  "summary": "failure requires RCA before repair emission"
}
```

## Storage Layout

Recommended durable paths:

- event log: `.harness/events/a2a.jsonl`
- lease snapshot: `.harness/state/lease-summary.json`
- dispatch snapshot: `.harness/state/dispatch-summary.json`
- checkpoint snapshot: `.harness/state/checkpoint-summary.json`

These do not replace legacy ledgers on day one.
They layer on top of them during migration.

## Replay Rules

To reconstruct state:

1. load current snapshots
2. replay later events from `a2a.jsonl`
3. apply idempotency and revision rules
4. ignore duplicate semantic actions

Replay must be able to recover:

- latest active dispatch per task
- active lease ownership
- latest resumable checkpoint
- latest worker outcome
- latest verification verdict and follow-up

## Decision Boundary

- orchestrator decides `dispatch`, `resume`, `replan`, `audit`, `stop`, or `block`
- worker supervisor decides only how to execute an already-approved bounded burst safely
- worker output can suggest, but never finalize, route changes

## Migration Note

During migration, legacy snapshots may still be revisionless.
New twin-node snapshots must still be revisioned from their first write onward.
