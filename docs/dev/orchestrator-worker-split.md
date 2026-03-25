# Orchestrator / Worker-Supervisor Split

This note freezes the next control-plane split for Klein-Harness.

The target shape is a two-node runtime:

- `orchestrator-node`
- `worker-supervisor-node`

The split is meant to tighten authority, recovery, and auditability.
It is a system-boundary change, not a prompt polish pass.

## Node A: `orchestrator-node`

The orchestrator owns only five classes of work:

1. `intake`
   Accept `harness-submit` and related human/runtime-originated requests.
2. `reconcile`
   Perform request fusion, thread correlation, and request-task binding.
3. `route`
   Decide `dispatch`, `resume`, `replan`, `audit`, `stop`, or `block`.
4. `verify_and_rca`
   Consume worker outcomes, decide completion, replan, or RCA allocation.
5. `project_summaries`
   Refresh `current.json`, `runtime.json`, `request-summary.json`, `worker-summary.json`, and `progress.json`.

Hard boundaries:

- it does not directly write business code
- it does not directly own `tmux`
- it does not treat long chat memory as system state
- it is `deterministic-first`, with LLM help only where heuristics are insufficient
- it owns orchestration packet truth; workers do not

## Node B: `worker-supervisor-node`

The supervisor owns only six classes of work:

1. manage the `tmux` worker pool
2. prepare or reuse worktrees
3. consume dispatch tickets
4. start one bounded Codex burst via `exec` or `resume`
5. collect checkpoint, heartbeat, and outcome artifacts
6. handle `stop`, `cancel`, stale lease, and cleanup

Hard boundaries:

- it does not decide whether work should be `fresh` or `resume`
- it does not decide whether the next step is `dispatch`, `replan`, `audit`, or `stop`
- it only executes work that was already approved by the orchestrator route gate

In this model, `tmux` is a reusable worker-node backend, not task identity.

## Big Loop vs Small Loop

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

Note:

- the outer runtime loop is not a mini-agent-loop
- b3e 3+1 convergence is only an internal packet synthesis subunit when runtime needs a fresh packet

### Small loop: worker supervisor

```text
claim ticket
-> ensure worker
-> ensure worktree
-> run bounded burst
-> parse structured output
-> write heartbeat / checkpoint / outcome
-> release or renew lease
-> wait next ticket
```

## Authority Rules

- `route-first-dispatch-second` remains mandatory.
- Orchestrator writes `.harness` authority state before supervisor acts.
- Supervisor executes only against an approved dispatch ticket.
- Worktree ownership is task-scoped and lease-scoped, not session-scoped.
- Session reuse is valid only when the route gate already emitted a resumable decision.
- Worker output is task-local only and cannot mutate global control-plane ledgers.

## Why This Split Fits Klein

- GSD contributes the large-loop discipline.
- `pi-mono` contributes the small-loop boundary and node hygiene.
- Klein contributes the repo-local control-plane ledgers and summary semantics.

## Rollout Order

### Phase 1: recover authority

Goal:

- make the truth source explicit before adding more autonomy

Required surfaces:

- `.harness/state/lease-summary.json`
- `.harness/state/dispatch-summary.json`
- `.harness/state/checkpoint-summary.json`

Hard rule:

- orchestrator must persist summary state before any dispatch is considered valid

Success condition:

- operators can explain who owns the lease, which dispatch is active, and which checkpoint is latest without inspecting raw `tmux`

### Phase 2: demote `tmux session` to `worker node`

Goal:

- stop treating `tmux` lifecycle as task identity

Hard rule:

- worker-to-task binding is temporary and lease-backed

Success condition:

- the same worker node can safely execute different tasks over time without corrupting task identity

### Phase 3: enforce bounded bursts

Goal:

- ensure every execution step is resumable, inspectable, and cheap to recover

Each burst must have:

- a concrete goal
- explicit path boundaries
- a time budget
- structured output
- a checkpoint

Success condition:

- network failure, topic drift, or context rot damages at most one burst instead of an entire task

### Phase 4: make the phase-one validation loop a hard gate

Goal:

- completion means the target requirement closed through the harness loop, not merely that code changed

Success condition:

- the body-vs-target split remains intact and target success is measured through harness convergence evidence

## Non-Goals

- no permanent prompt-only scheduler
- no hidden mutable state in chat history
- no task identity encoded only in `tmux`
- no worker-side replan authority
