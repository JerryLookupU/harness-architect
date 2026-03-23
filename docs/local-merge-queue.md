# Local Merge Queue

Klein-Harness does not treat remote push as the primary integration signal.

Workers implement in their own task branches and worktrees. The runtime serializes local integration through an explicit merge queue and integration branch.

## Core surfaces

- `.harness/state/merge-queue.json`
- `.harness/state/merge-summary.json`
- `.harness/state/worktree-registry.json`

Tracked fields include:

- `taskId`
- `requestId`
- `threadKey`
- `planEpoch`
- `branchName`
- `worktreePath`
- `baseRef`
- `integrationBranch`
- `mergeRequired`
- `mergeStatus`
- `mergeCheckedAt`
- `conflictPaths`
- `supersededByEpoch`
- `mergedCommit`
- `cleanupStatus`

## Integration protocol

1. worker verifies in its own task worktree
2. runtime enqueues verified merge-required work
3. runtime previews merge against `integrationBranch`
4. if clean, runtime merges locally through the integration worktree
5. only after local integration succeeds should any later remote push happen

## Merge status progression

- `merge_queued`
- `merge_checked`
- `merged`
- `merge_conflict`
- `merge_resolution_requested`

`completed` remains the external closed-loop terminal state, but merge progression stays visible as its own runtime dimension.

## Supersession

The merge queue is thread/epoch aware:

- newer plan epochs can supersede queued merge candidates
- verified work is not automatically integrated if a newer epoch invalidates it
- unaffected valid work may still merge if policy allows
