# Merge Conflict As Runtime Signal

In Klein-Harness, a git conflict is not just a Git incident. It is a runtime event.

## Rule

If local merge preview or local integration merge detects conflict:

- do not silently fail
- do not leave dirty merge state behind
- do not wait for remote push failure to discover it

Instead the runtime records a structured conflict outcome and emits the smallest necessary follow-up.

## Conflict outputs

The runtime records:

- conflicting paths
- task / request / thread / epoch
- branch / base / integration metadata
- impact classification
- follow-up request id when emitted

Primary surfaces:

- `.harness/state/merge-queue.json`
- `.harness/state/merge-summary.json`
- `.harness/lineage.jsonl`

## Impact classes

- `non-conflicting`
- `safe-to-finish`
- `unsafe-conflict`

## Follow-up behavior

Depending on deterministic preflight and conflict scope, the runtime can emit:

- `audit`
- `replan`
- `stop`

Workers do not decide final integration on their own, and they do not merge directly into the integration branch.

## Why this matters

This keeps merge handling inside the closed loop:

- conflict becomes machine-readable
- lineage remains intact
- operator tools can see blocked integration without opening full logs
- the runtime can checkpoint, replan, or stop without pretending the work already integrated
