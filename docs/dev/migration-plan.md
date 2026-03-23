# Migration Plan

## Objective

Move the current runtime into twin-node architecture in small compatible steps.

The migration must preserve:

- repo-local closed-loop semantics
- public 4-command UX
- worktree-first execution for code-bearing tasks
- deterministic routing and evidence-based completion

## Compatibility Rule

Do not replace `harness-submit`, `harness-tasks`, `harness-task`, or `harness-control`.
New logic enters behind adapters and optional wrappers first.

## Phase 1: Freeze Contracts

Deliverables:

- [commit-route.md](/Users/mac/code/harness-architect/docs/dev/commit-route.md)
- [twin-node-prd.md](/Users/mac/code/harness-architect/docs/dev/twin-node-prd.md)
- [a2a-protocol-v1.md](/Users/mac/code/harness-architect/docs/dev/a2a-protocol-v1.md)

Exit criteria:

- node boundaries are explicit
- A2A envelope and event kinds are fixed
- migration order is documented before code starts expanding

## Phase 2: Add Authority Summaries

Introduce:

- `.harness/state/lease-summary.json`
- `.harness/state/dispatch-summary.json`
- `.harness/state/checkpoint-summary.json`
- `.harness/events/a2a.jsonl`

Rules:

- new summaries use revisioned writes
- event append is idempotent and replayable
- no existing public command is removed

Compatibility path:

- legacy Python runtime remains the default
- new summaries are additive, not a replacement for legacy ledgers

## Phase 3: Add Go Twin-Node Skeleton

Introduce package boundaries:

- `cmd/kh-orchestrator`
- `cmd/kh-worker-supervisor`
- `internal/a2a`
- `internal/state`
- `internal/lease`
- `internal/route`
- `internal/dispatch`
- `internal/worktree`
- `internal/checkpoint`
- `internal/verify`
- `internal/rca`
- `internal/tmux`
- `internal/adapter`

Rules:

- standard library only where possible
- bounded functionality before clever heuristics
- tests land with each package

## Phase 4: Route Adapter

Adapter target:

- `harness-route-session`

Migration step:

- if twin-node binary is present and opt-in is enabled, `harness-route-session` delegates to `kh-orchestrator route`
- otherwise it falls back to the legacy Python route path

Reason:

- this respects Route First, Dispatch Second
- it introduces the new orchestrator boundary without changing public UX

## Phase 5: Runner/Supervisor Adapter

Adapter target:

- `harness-runner`

Migration step:

- if a twin-node dispatch ticket exists and twin-node opt-in is enabled, `harness-runner` can delegate bounded execution to `kh-worker-supervisor`
- otherwise it continues to use the legacy runner path

Rules:

- supervisor must refuse direct execution without an approved dispatch ticket
- lease state must outlive `tmux`

## Phase 6: Verification and Follow-Up Adapter

Adapter targets:

- `harness-verify-task`
- `refresh-state`

Migration step:

- verification results feed the new follow-up emission path first
- old reporting remains readable during migration

Rules:

- `task.completed` requires verification evidence
- failure emits `replan` or `rca`, never silent terminal success

## Phase 7: Target Validation Automation

Loop:

1. patch body repo
2. install harness into target repo
3. submit real requirement through `harness-submit`
4. observe target closed loop
5. feed failures back into body repo capability work

Exit criteria:

- target requirement survives interruption and can recover
- duplicate dispatch is prevented
- unsafe concurrent resume is prevented
- completion means target convergence evidence exists

## Cutover Rules

- keep legacy code paths available until twin-node summaries are proven stable
- prefer opt-in environment or binary presence checks before changing default behavior
- migrate reads to hot summaries first, raw logs last
- do not require operators to learn new public commands during transition

## Known Risk

The current legacy runtime does not yet universally persist `revision` on every snapshot.
Migration should therefore:

- enforce revision on all new twin-node summaries immediately
- treat legacy revisionless snapshots as compatibility inputs
- avoid a one-shot rewrite of every existing Python writer in the first implementation pass
