# Twin-Node PRD

## Goal

Evolve Klein-Harness from a single repo-local closed loop into a twin-node architecture without regressing the existing closed-loop semantics.

Target runtime split:

- `orchestrator-node`
- `worker-supervisor-node`

Success means:

- `.harness` remains the only authority
- route decisions remain deterministic-first and explicit
- execution becomes bounded, leased, resumable, and auditable
- existing public command surface keeps working while the new layers are adopted
- phase-one completion is still measured by target requirement convergence, not body-only code churn

## Non-Goals

- no one-shot rewrite of the existing runtime
- no prompt-only state hidden in model memory
- no treating `tmux` as the scheduler or source of truth
- no worker-side authority to choose `fresh` vs `resume`
- no direct orchestrator business-code editing

## Hard Constraints

1. `.harness` state is authoritative; `tmux` is not.
2. Route First, Dispatch Second.
3. `fresh` vs `resume` is route-gate authority only.
4. Code-bearing tasks default to worktree-first.
5. Raw logs are cold evidence; hot summaries are default inputs.
6. Cross-node communication is structured JSON only.
7. Each worker execution is a bounded burst.
8. New snapshot writes must include `schemaVersion`, `generator`, `generatedAt`, and `revision`.
9. State transitions must carry `causationId`, `reasonCodes`, and append lineage evidence.
10. In phase-one body-vs-target mode, completion is target-loop convergence.

## Roles

### Node A: `orchestrator-node`

Responsibilities:

- intake
- request fusion
- thread correlation
- request-task binding
- route decision
- dispatch issuance
- replan
- verification ingestion
- RCA allocation
- summary refresh

Must not:

- directly own `tmux`
- directly edit target business code
- infer durable state from chat history

### Node B: `worker-supervisor-node`

Responsibilities:

- worker-pool lifecycle
- lease claim/renew/release
- worktree preparation/reuse
- dispatch consumption
- bounded `codex exec` or `codex exec resume` bursts
- heartbeat/checkpoint/outcome collection
- stale lease cleanup

Must not:

- decide `fresh` vs `resume`
- decide `dispatch` vs `replan` vs `audit` vs `stop`
- silently widen owned paths

## State Model

### Cold evidence

- `.harness/requests/queue.jsonl`
- `.harness/lineage.jsonl`
- `.harness/feedback-log.jsonl`
- `.harness/root-cause-log.jsonl`
- `.harness/state/runner-logs/*.log`
- `.harness/events/a2a.jsonl`

### Runtime ledgers

- `.harness/state/request-index.json`
- `.harness/state/request-task-map.json`
- `.harness/task-pool.json`
- `.harness/session-registry.json`
- `.harness/state/lease-summary.json`
- `.harness/state/dispatch-summary.json`
- `.harness/state/checkpoint-summary.json`

### Hot summaries

- `.harness/state/current.json`
- `.harness/state/runtime.json`
- `.harness/state/request-summary.json`
- `.harness/state/worker-summary.json`
- `.harness/state/progress.json`

Rules:

- hot summaries are the default read path
- mutable ledgers remain source-of-truth for scheduling and binding
- append-only events plus current snapshots must be enough to reconstruct state

## State Machine

### Big loop: orchestrator

```text
submit
-> classify
-> fuse
-> bind
-> route
-> issue dispatch ticket
-> ingest worker outcome
-> verify
-> emit repair / replan / complete
-> refresh summaries
-> next tick
```

### Small loop: worker supervisor

```text
claim dispatch
-> acquire lease
-> ensure worker node
-> ensure worktree
-> run bounded burst
-> write heartbeat / checkpoint / outcome
-> renew or release lease
-> await next ticket
```

## Failure Modes

### Duplicate dispatch

Expected handling:

- dedupe by `kind + idempotencyKey`
- existing ticket is returned, not duplicated

### Stale lease

Expected handling:

- stale lease is recovered from summary state, not `tmux` guesswork
- recovery emits machine-readable evidence

### Unsafe resume

Expected handling:

- route gate blocks or downgrades to fresh
- worker supervisor never self-selects resume

### Worktree/path drift

Expected handling:

- route gate blocks code-bearing execution without valid worktree/owned paths
- burst outcome records owned-path violations as machine-readable failure

### Verification failure

Expected handling:

- verification evidence is ingested
- orchestrator emits `replan` or `rca`
- task is not considered complete based only on exit code

### Target-loop non-convergence

Expected handling:

- treat as harness gap first
- patch body repo runtime, protocol, adapter, or install chain
- rerun target validation instead of manually rescuing business code

## Acceptance Criteria

### Phase 1: docs and contracts

- `commit-route.md`
- `twin-node-prd.md`
- `a2a-protocol-v1.md`
- `migration-plan.md`

### Phase 2: authority skeleton

- A2A event append exists
- snapshots have revisioned writes
- dispatch tickets and leases exist as repo-local summaries

### Phase 3: supervisor skeleton

- bounded bursts create machine-readable checkpoint/outcome sidecars
- stale lease recovery works from state alone

### Phase 4: verify and follow-up

- verification ingestion can emit `task.completed`, `replan.emitted`, or `rca.allocated`
- completion is evidence-based

### Phase 5: body-vs-target validation

- target repo can be driven through installed harness
- network/session interruption does not duplicate execution or lose recovery state
- same task is not resumed concurrently by multiple workers

## Rollout Principle

This migration is additive first:

- new node logic arrives as adapters and wrappers
- old 4-command UX remains stable
- ownership is pulled into explicit summaries before any broad rewrite
