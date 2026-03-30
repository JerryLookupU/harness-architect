# Harness Codebase Growth Guardrails

This document defines how phase-1 keeps `Klein-Harness` from uncontrolled size growth while still shipping system fixes.

## Why This Exists

Phase-1 focuses on fixing harness gaps exposed by real target repos. Without explicit budgets, quick fixes can accumulate into:

- duplicate command surfaces
- overlapping runtime logic
- oversized examples that shadow real control-plane intent
- binary/doc assets that bloat repo transfer and review cost

## Default Budgets

Use `scripts/harness-size-tracker.sh` to record a timeline and enforce budgets.

- tracked files: `<= 140`
- tracked LOC: `<= 70,000`
- tracked bytes: `<= 12,000,000`

Override thresholds with env vars when needed:

- `HARNESS_MAX_TRACKED_FILES`
- `HARNESS_MAX_TRACKED_LOC`
- `HARNESS_MAX_TRACKED_BYTES`

## Planning Rules

1. Prefer extension over expansion.
Keep changes inside existing command surfaces before creating new entry points.

2. Separate capability from one-off workaround.
If a target failure needs a new behavior, implement it as reusable guard/runtime policy, not target-specific conditionals.

3. Keep control-plane and business-code lanes distinct.
Unknown dirty in target business files must remain a blocker; only explicit control-plane tasks inside `.harness/` can receive guard allowances.

4. Reuse shared runtime modules.
Before adding scripts, check whether the behavior belongs in shared runtime utilities.

5. Treat large assets as exceptional.
Binary docs or screenshots should be justified in review context and periodically reviewed for archival or externalization.

## Runbook

Record one snapshot per phase-1 run:

```bash
scripts/harness-size-tracker.sh --repo /path/to/harness-architect --warn-only
```

If budget exceeds:

1. classify by growth source
2. open or append a control-plane replan to reduce duplication
3. avoid adding new top-level scripts until budget trend stabilizes
