This directory contains the runtime-internal orchestration prompts for Klein-Harness.

Runtime shape:
- the outer runtime is route-first-dispatch-second and remains repo-owned
- orchestration meaning is carried by a runtime-owned packet, not a visible outer `proposal/specs/design/tasks` stage
- b3e-style 3+1 convergence exists only inside packet synthesis subunits

Runtime role split:
- orchestrator runtime: submit -> classify -> fuse -> bind -> route -> issue dispatch ticket -> ingest outcome -> verify -> refresh summaries
- packet synthesis subunit: 3 planners + 1 judge produce one orchestration packet and task-local worker-spec candidates
- worker execution: read dispatch ticket + worker-spec -> execute -> verify -> handoff

Default load order:
1. orchestrator.md
2. propose.md
3. packet.md
4. worker-spec.md
5. dispatch-ticket.md
6. worker-result.md
7. apply.md
8. verify.md
9. archive.md
10. planner-architecture.md
11. planner-delivery.md
12. planner-risk.md
13. judge.md

Compatibility note:
- `proposal.md`, `specs.md`, `design.md`, and `tasks.md` remain only as mapping shims for older mental models
- do not treat those shim files as first-class runtime stages

Usage rules:
- when a request arrives as a requirement, start from this directory
- synthesize or refresh a packet only when runtime state does not already hold an accepted packet for the active epoch
- planner and judge prompts may shape packet and worker-spec candidates, but completion still belongs to `completion-gate.json`
- prefer bounded, verifiable task slices over broad architectural narration
