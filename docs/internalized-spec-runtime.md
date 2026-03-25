# Internalized Spec Runtime

Klein-Harness no longer treats `proposal/specs/design/tasks` as a visible outer workflow stage.
That meaning now lives inside runtime-owned objects and task-local artifacts.

## Old To New Mapping

- `proposal`
  Now lives in orchestration packet `objective`, `constraints`, and `decisionRationale`.
- `specs`
  Now lives in orchestration packet `verificationPlan` and `acceptanceMarkers`.
- `design`
  Now lives in orchestration packet `selectedPlan`, `rejectedAlternatives`, and `rollbackHints`.
- `tasks`
  Now lives in orchestration packet `executionTasks`, plus one task-local `worker-spec.json` per executable task.
- visible outer spec package
  Replaced by a runtime-owned orchestration packet tied to one accepted epoch.
- outer mini-agent-loop
  Removed. b3e 3+1 convergence exists only inside packet synthesis subunits.

## Runtime Objects

### Orchestration packet

Required fields:

- `objective`
- `constraints`
- `selectedPlan`
- `rejectedAlternatives`
- `executionTasks`
- `verificationPlan`
- `decisionRationale`
- `ownedPaths`
- `taskBudgets`
- `acceptanceMarkers`
- `replanTriggers`
- `rollbackHints`

Rules:

- one accepted epoch should converge on one packet truth
- packet meaning is runtime-owned, not worker-owned
- merge, archive, and completion stay outside the packet and stay runtime-owned

### Worker-spec

Each executable task gets one task-local `worker-spec.json`.

Required fields:

- `taskId`
- `objective`
- `constraints`
- `ownedPaths`
- `blockedPaths`
- `taskBudget`
- `acceptanceMarkers`
- `verificationPlan`
- `replanTriggers`
- `rollbackHints`

Rules:

- worker-spec may refine only task-local execution
- workers may not mutate global control-plane ledgers
- workers may not decide completion, merge, or archive

### Dispatch ticket

Each actual dispatch gets one immutable ticket / manifest.

Required identity and guard fields:

- `dispatchId`
- `idempotencyKey`
- `taskId`
- `threadKey`
- `planEpoch`
- `attempt`
- `requiredSummaryVersion`
- `allowedWriteGlobs`
- `blockedWriteGlobs`
- `worktreePath`
- `budget`
- `leaseTtlSec`
- `workerSpecPath`

Rules:

- route-first-dispatch-second remains mandatory
- stale leases must not overwrite newer accepted execution
- unknown dirty worktrees still block automation at runtime guard level

### Worker-result

Each worker burst emits task-local outputs:

- `worker-result.json`
- `verify.json`
- `handoff.md`

Rules:

- `worker-result.json` records only the terminal outcome of that worker run
- free-text worker claims do not close the loop
- `completion-gate.json` remains the completion decision surface
- noop completion is valid only when acceptance is already satisfied and verification evidence says so

## Authority Split

- orchestrator owns intake, reconcile, fusion, route, verify/RCA, summary refresh, and follow-up emission
- worker supervisor owns lease control, worktree prepare/reuse, bounded dispatch, heartbeat, checkpoint, outcome write-back, and cleanup
- worker owns only task-local execution and task-local artifacts

## Compatibility Shims

- `prompts/spec/proposal.md`
- `prompts/spec/specs.md`
- `prompts/spec/design.md`
- `prompts/spec/tasks.md`

These remain only as mapping shims for older prompt stacks.
They are not first-class runtime stages anymore.
