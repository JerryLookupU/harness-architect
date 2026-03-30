# Harness Footprint Governance

## Goal

Keep `.harness/` durable but bounded. Phase-1 requires reusable control-plane capability, not unlimited file growth.

## Budgets

- Soft growth limit per run: `+5%` total `.harness` bytes.
- Hard growth limit per run: `+12%` total `.harness` bytes.
- Hot-state budget: `.harness/state/*.json` should remain bounded snapshots; archive long tails into append-only logs.

## Required Metrics

- `totalFiles`: file count under `.harness/`.
- `totalBytes`: total bytes under `.harness/`.
- `deltaBytes` and `deltaPct` versus last baseline.

## Growth Response Policy

1. If `deltaPct <= 5%`, continue.
2. If `5% < deltaPct <= 12%`, require a one-line reason in state or report.
3. If `deltaPct > 12%`, create a replan or audit follow-up before new feature expansion.

## Preferred Shrink Actions

- Remove stale runtime artifacts and old bootstrap leftovers.
- Prune stale `.harness.bak-*` backups.
- Keep hot state bounded and move narrative evidence to compact logs and indexes.
- Avoid adding new top-level ledgers when an existing ledger can hold the data.

## Multi-Thread Merge Discipline

- Worker lanes may commit locally when policy allows.
- Before push: `fetch` plus bounded rebase or merge retry against the integration branch.
- On unresolved conflict: emit merge or replan follow-up and stop; no force-push in shared automation lanes.
