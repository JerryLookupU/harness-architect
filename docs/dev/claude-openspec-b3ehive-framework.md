# Internalized Packet Framework with b3e Packet Synthesis

Date: 2026-03-24
Mode: extension
Research mode: none

## Goal

Freeze the intended framework shape for Klein-Harness:

- the outer runtime is route-first-dispatch-second, not a mini-agent-loop
- packet synthesis uses b3e convergence to generate one runtime-owned orchestration packet plus worker-spec candidates
- the executor consumes dispatch tickets and task-local worker-specs

## Architecture

### 1. Outer runtime

The outer runtime is not a generic chat loop.
It is a repo-local control loop with deterministic ledgers and bounded synthesis calls.

Loop:

```text
requirement
-> context assembly
-> determine whether an accepted epoch already has a usable packet
-> synthesize or refresh packet only when needed
-> persist packet meaning into task/request/dispatch state
-> route
-> issue dispatch ticket
```

This is the Claude Code influence:

- explicit context assembly before action
- planner-like subagent decomposition
- judge-style convergence instead of naive single-pass planning
- planning separated from execution

### 2. Packet synthesis block

The packet synthesis block combines two ideas:

- runtime-owned packet fields carry the old artifact meaning
- b3ehive gives the convergence pattern
  - 3 isolated candidate planners
  - 1 judge/formatter

The synthesis output should look like an execution-ready orchestration packet, not a free-form essay.

Required output fields:

- objective
- constraints
- selectedPlan
- rejectedAlternatives
- executionTasks
- verificationPlan
- decisionRationale
- ownedPaths
- taskBudgets
- acceptanceMarkers
- replanTriggers
- rollbackHints

## 3. Inner executor

The executor is not a replanner.
Its job is to consume dispatch tickets and execute task-local worker-specs.

Loop:

```text
claim dispatch ticket
-> read dispatch ticket and worker-spec
-> apply one task-local slice
-> verify the result
-> write outcome / handoff / checkpoint
-> return outcome to runtime
```

The executor may report drift or blockers, but it should not replace the orchestrator's role.

## Current mapping in Klein

Already present:

- prompt-level orchestrator contract in `prompts/spec/`
- `3 + 1` packet synthesis loop in worker manifest and orchestration defaults
- Codex-style entrypoint via `kh-codex`
- task/session/dispatch/lease/checkpoint ledgers
- worker-side bounded burst execution

Not fully implemented yet:

- deterministic runtime fan-out/fan-in for the 3 planners plus 1 judge as first-class control-plane tasks
- a dedicated executor poller that consumes only dispatch tickets and task-local worker-specs as a separate runtime role
- standalone packet persistence per accepted epoch as a first-class repo-local snapshot, instead of prompt-derived references alone

## Design rule

When evolving the framework:

- do not let the executor become a second planner
- do not let packet synthesis skip bounded worker-spec shaping for vague requirements
- do not merge planner outputs by averaging; judge them and select deliberately
- keep packet and worker-spec outputs structured enough that execution can proceed without reinterpreting the original request
