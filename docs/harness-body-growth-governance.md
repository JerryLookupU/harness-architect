# Harness Body Growth Governance

## Goal

Keep `Klein-Harness` body evolution sustainable while running fast phase-1 loops on target repos.
The body repo must gain reusable capability, not accumulate one-off residue.

## Budget Signals

- Track repo growth through `scripts/harness-size-tracker.sh`.
- Track `.harness` durable footprint through `scripts/harness-footprint.sh`.
- Treat sustained ratio increase as a control-plane risk, not a cosmetic issue.
- Require every new state or log artifact to justify recovery value, operator visibility value, and a bounded retention policy.

## Anti-Bloat Rules

- Prefer extending existing ledgers over creating new top-level files.
- Keep hot state bounded; archive long-tail detail into append-only logs.
- Do not duplicate the same fact across multiple JSON surfaces unless one is a strict projection.
- For temporary runtime artifacts, define automatic cleanup or retention windows.
- For multi-thread runs, avoid shared-file write amplification.

## Conflict Strategy

- Worker tasks write only owned paths.
- Orchestrator tasks own shared ledgers and merge gates.
- Before push, fetch and perform a bounded rebase or merge retry.
- If conflict density stays high on shared control files, split worker-local shards from orchestrator merge surfaces.
