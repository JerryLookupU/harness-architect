# Guard Loop

The public UX is not the scheduler.

The repo-local runtime is the guard loop:

```text
submit
-> classify
-> fuse
-> derive/update todo
-> safety checks
-> route
-> dispatch / recover / resume
-> verify
-> checkpoint / merge / archive when safe
-> refresh summaries
-> next tick
```

## Rules

- triggers only wake the guard
- the guard owns safety boundaries, not `tmux`
- the guard may converge stale repo-local tmux sessions before dispatch
- execution is blocked when the control plane is unsure
- summary state is the default operator surface
- semantic success requires gate-aligned evidence, not only exit code
- `verification.completed` is not terminal by itself; runtime only emits `task.completed` after completion-gate checks pass
- review-required work may verify successfully and still remain open until review evidence exists
- orchestration packet meaning is runtime-owned; workers only operate on task-local worker-specs and dispatch tickets

## Safety Boundary

Before non-interactive execution the guard checks:

- repo and git state are usable
- conflicting live execution is not already in progress
- unknown dirty worktree state is absent
- actionable todo still exists
- completion gate is not already satisfied
- worktree / merge state is coherent

Operator-initiated `harness-runner run` / `recover` may override automation-only dirty blockers,
but they do not override task drift, failed verification, satisfied completion gate, or superseded-task blockers.

Current implementation note:

- `lockHealthy` is currently derived from daemon/runtime health plus conflicting-execution checks
- it is not yet a dedicated on-disk lease ledger

## Planned node split

The next refactor keeps the guard semantics but sharpens the runtime into two loops:

- orchestrator big loop: `submit -> classify -> fuse -> bind -> route -> issue dispatch ticket -> ingest outcome -> verify -> emit repair/replan/complete -> refresh summaries`
- worker-supervisor small loop: `claim ticket -> ensure worker -> ensure worktree -> run bounded burst -> write heartbeat/checkpoint/outcome -> release or renew lease`

Important clarification:

- mini-agent-loop is not the outer runtime loop
- b3e 3+1 convergence exists only inside orchestration packet synthesis subunits
- workers execute only task-local scope and may not mutate global control-plane ledgers

The critical rule stays the same:

- the supervisor does not decide `fresh` vs `resume`
- the route gate does

See:

- [twin-node-prd.md](/Users/mac/code/harness-architect/docs/dev/twin-node-prd.md)
- [a2a-protocol-v1.md](/Users/mac/code/harness-architect/docs/dev/a2a-protocol-v1.md)
- [migration-plan.md](/Users/mac/code/harness-architect/docs/dev/migration-plan.md)

## Key State

- `.harness/state/guard-state.json`
- `.harness/state/todo-summary.json`
- `.harness/state/completion-gate.json`
- `.harness/state/runtime.json`

## Runtime Completion Gate

The hard gate stays inside the existing runtime surfaces.

- verification writes `verification.completed` first, then refreshes `completion-gate.json` and `guard-state.json`
- passed-like verification statuses do not imply completion on their own
- runtime requires a coherent evidence bundle before emitting `task.completed`
- the minimum acceptable bundle is:
  - a non-empty completion summary
  - a valid verification evidence path with meaningful content
  - required task artifacts that match the dispatch outcome when available
- when a task is explicitly marked `reviewRequired`, runtime also requires review evidence before completion
- archive / retire actions must respect the current completion gate instead of bypassing it

Blocked and failed flows keep their existing branches:

- `blocked` still emits `task.blocked`
- failed verification can still allocate RCA or emit replan follow-up
- the gate only prevents false completion / archive; it does not collapse failure handling into one path
