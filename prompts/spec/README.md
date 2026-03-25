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
4. tasks.md
5. worker-spec.md
6. dispatch-ticket.md
7. worker-result.md
8. apply.md
9. verify.md
10. archive.md
11. planner-architecture.md
12. planner-delivery.md
13. planner-risk.md
14. judge.md

Compatibility note:
- `proposal.md`, `specs.md`, `design.md`, and `tasks.md` remain only as mapping shims for older mental models
- do not treat those shim files as first-class runtime stages
- behavior guardrails from dotfiles-style workflows should map into this prompt layer, route reason codes, and task-local verify/review requirements rather than a separate Hookify runtime

Usage rules:
- when a request arrives as a requirement, start from this directory
- synthesize or refresh a packet only when runtime state does not already hold an accepted packet for the active epoch
- planner and judge prompts may shape packet and worker-spec candidates, but completion still belongs to `completion-gate.json`
- prefer bounded, verifiable task slices over broad architectural narration
- when route or dispatch supplies `reasonCodes` with `policy_*` tags, treat those tags as hard guardrails that must be reflected in packet flow selection, execution tasks, verification, and review
