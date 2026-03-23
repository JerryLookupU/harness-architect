# Worktree-First Execution

Klein-Harness treats git worktrees as the default code-isolation layer for code-bearing tasks.

## Why

- concurrent code workers should not share one mutable checkout
- diff / verification / audit need a stable task-local base
- session lineage and code lineage are different dimensions
- runtime should be able to reclaim or retain worktrees explicitly

## Default rule

The runtime prepares a dedicated worktree when the task is code-bearing, merge-required, overlap-prone, or long-running enough that repeated verification is expected.

Control-plane-only work can stay in the main repo root.

## Runtime path

For worktree-backed tasks, the control loop does this:

1. keep `branchName`, `worktreePath`, and `baseRef` on the task
2. run `prepare-worktree.py --create --write-back`
3. persist `worktree_prepared` into task/worktree ledgers
4. dispatch the worker with `worktreePath` as execution cwd
5. run diff summary and verification against that same worktree binding

## Important separation

- runtime / daemon is the scheduler
- dispatch backend is below the scheduler abstraction
- tmux is only the current default worker-node backend
- worktree is the code-isolation layer, not a synonym for tmux session

## Operator surfaces

The main machine-readable surfaces are:

- `.harness/state/worktree-registry.json`
- `.harness/state/task-summary.json`
- `.harness/state/worker-summary.json`
- `.harness/state/runtime.json`

These surfaces should be enough to answer:

- which tasks currently own worktrees
- which branch/base/worktree binding belongs to a task
- whether a worktree is prepared, active, conflicted, merged, or reclaimable
