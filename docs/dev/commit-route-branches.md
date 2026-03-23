# Commit Route: Branches

This file freezes the branch-side view behind `main`.

Use it when first-parent history is too compressed and you need to answer "what came from which branch?"

## Source Commands

Primary graph:

```bash
git log --graph --decorate --oneline --all
```

Useful supporting slices:

```bash
git show --stat --summary 9d86d01
git show --stat --summary f92a08c
git show --stat --summary 65e134f
```

## Branch Topology

Observed graph shape:

```text
* 65e134f (HEAD -> main, origin/main, origin/HEAD) Repair request-task state convergence
* 0a0958b Guard harness installs with runtime smoke checks
* 7dca06d Avoid current-task raw log recursion
* 44ba615 Generalize license copyright holder
* f6ab72e Document phase-one validation loop
* 0d2510b Stabilize control-plane request focus
* fa12742 Tighten control-plane replan convergence
* 35038bc Refine harness self-driven guidance
* 26a1957 Lock phase-one operator goal
* cace6d4 Tighten control-plane prompt drift guardrails
* 116aeeb Document harness iteration rules
* a25cb6e Add safe worktree tidy control
* 704fd44 Release v0.2.2 runtime degradation fixes
* ea33695 收束版本
* ce27ed8 提交图片
* 05e25a4 Compress public CLI for v0.2.1 cleanup
* f92a08c (tag: v0.2) Add worktree-first merge-aware runtime
* c52b449 Harden machine-first control plane
* e77dc94 Polish README layout and messaging
*   8bfd3be Merge branch 'feat/log-search-blueprint-research'
|\  
| * c830f11 Add compact log search and blueprint research gate
|/  
* ad11fd2 Add closed-loop RCA allocation runtime
* b513e04 Add Klein hero image and README tagline
* 9bca4ff Add Klein surface hero image to README
* f1002b4 Refine README positioning and quick start
* d93be28 Enable runner daemon by default after bootstrap
* b9eaa6e Add built-in runner daemon mode
* e2b72ab Add blueprint-architect skill
* c1a0278 Harden bootstrap flow and review cadence
* 2c3c802 Fix harness-kick empty context rerun output
* 399ee54 Refine Klein-Harness tagline
* cd4ab76 Clean up README for Klein-Harness
* 35c1b3e Rename harness-architect to klein-harness
* 62f2e18 Align Klein docs and smoke cleanup
*   0f11dfd Merge pull request #2 from JerryLookupU/feat-klein-closed-loop
|\  
| | * 4e8b1de (origin/feat-klein-closed-loop) Align Klein docs and smoke cleanup
| |/  
| * 9d86d01 Upgrade harness runtime to Klein closed loop
|/  
* 2b3f18d (tag: v0.1.0) release: v0.1.0
```

## Branches That Matter

### `feat-klein-closed-loop`

Mainline entry:

- Merge commit: `0f11dfd`
- Branch payload commit: `9d86d01`
- Extra branch tip visible on remote: `4e8b1de`

What this branch changed:

- It is the large runtime pivot from the pre-Klein baseline to the Klein closed-loop model.
- `9d86d01` adds `docs/klein-architecture.md` and introduces a large `runtime-common` expansion in the examples.
- The merge is followed on main by rename and cleanup commits that finish the identity shift from `harness-architect` to `klein-harness`.

Why it matters:

- This is the branch that turns `v0.1.0` into the modern Klein line.
- When someone asks where the current runtime architecture really starts, point here first.

### `feat/log-search-blueprint-research`

Mainline entry:

- Merge commit: `8bfd3be`
- Branch payload commit: `c830f11`

What this branch changed:

- It adds compact log search and a blueprint-research gate.
- It is a focused feature branch, unlike the earlier Klein pivot branch.
- Its changes are immediately followed by control-plane hardening and then the `v0.2` worktree-first runtime cut.

Why it matters:

- This branch shows the repo moving from "closed loop exists" to "closed loop can inspect, research, and route with more machine-first discipline."

## Anchor Commits

### `9d86d01` as the architecture branch payload

`git show --stat --summary 9d86d01` shows:

- 13 files changed
- 2594 insertions, 789 deletions
- new `docs/klein-architecture.md`
- new `skills/harness-architect/examples/runtime-common.example.py`

Interpretation:

- This is not a cleanup commit.
- It is the architecture payload inside the `feat-klein-closed-loop` line.

### `f92a08c` as the `v0.2` cut

`git show --stat --summary f92a08c` shows:

- 18 files changed
- 4472 insertions, 362 deletions
- new docs for merge queue, merge conflict runtime signal, and worktree-first execution
- new `worktree-merge-smoke` example

Interpretation:

- `v0.2` is the worktree/merge-aware runtime release, not just a small tag bump.
- This commit should be treated as the second major architecture anchor after `9d86d01`.

### `65e134f` as the latest stabilization anchor

`git show --stat --summary 65e134f` shows:

- 3 files changed
- 184 insertions, 14 deletions
- focused updates in control/refresh/runtime example code

Interpretation:

- The current line is no longer broad architecture creation.
- It is convergence repair on request-task state behavior.

## Reading Rule

- If the question is "what shipped on main?", use [commit-route-mainline.md](/Users/mac/code/harness-architect/docs/dev/commit-route-mainline.md).
- If the question is "which branch injected that capability?", use this file.
- If the question is "which stage of repo evolution was this?", use [commit-phases.md](/Users/mac/code/harness-architect/docs/dev/commit-phases.md).
