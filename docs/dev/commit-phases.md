# Commit Phases

This file groups the repo history into phases that are easy to reference during reviews, release retros, and operator onboarding.

## Source Commands

```bash
git log --reverse --pretty=format:'%h%x09%ad%x09%s' --date=short main
git log --reverse --pretty=format:'%h%x09%s' v0.1.0..main
git log --reverse --pretty=format:'%h%x09%s' v0.2..main
git show --stat --summary 9d86d01
git show --stat --summary f92a08c
git show --stat --summary 65e134f
```

## Phase Map

### Phase 0: Trial Repo Bootstrap

Range:

- `c28837e` to `2b3f18d`
- Date window: 2026-03-21 to 2026-03-22
- Terminal marker: `v0.1.0`

What defines the phase:

- The repo is still a trial `harness-architect` setup.
- Work is dominated by README, positioning, install flow, and shell-loop setup.
- The goal is to make the project legible and runnable, not yet to lock the full Klein runtime model.

Canonical commits:

- `c28837e` init harness-architect trial repo
- `797cbcb` docs: clarify codex-first positioning and add MIT license
- `ee3642b` docs: rewrite README in bilingual codex-first format
- `b30e6f7` docs: add bilingual ascii architecture flow to README
- `f5ce202` update install.sh
- `f3523c5` update agent shell loop
- `2b3f18d` release: v0.1.0

### Phase 1: Klein Closed-Loop Adoption

Range:

- `9d86d01` to `f92a08c`
- Date window: 2026-03-22 to 2026-03-23
- Entry anchor: `9d86d01`
- Terminal marker: `v0.2`

What defines the phase:

- The repo stops being just a trial harness shell and becomes a Klein closed-loop runtime project.
- The name, architecture docs, examples, runner behavior, and machine-first control plane all move in the same direction.
- This phase contains both the large architecture pivot and the worktree/merge-aware release cut.

Sub-steps inside the phase:

1. Closed-loop runtime lands through `9d86d01` and merge commit `0f11dfd`.
2. Mainline cleanup and renaming finish the identity move to `klein-harness`.
3. Runner daemon, blueprint skill, and RCA/runtime control-plane capabilities are layered in.
4. `c830f11` and merge `8bfd3be` add log-search and blueprint-research gating.
5. `f92a08c` cuts the worktree-first, merge-aware runtime baseline.

Why `9d86d01` is an anchor:

- It is the payload commit for the `feat-klein-closed-loop` branch.
- `git show --stat --summary 9d86d01` reports 2594 insertions across architecture and runtime example files.
- This is the clearest "old repo -> Klein runtime" transition point.

Why `f92a08c` is an anchor:

- It is tagged `v0.2`.
- `git show --stat --summary f92a08c` reports 4472 insertions and introduces worktree/merge runtime docs plus a merge smoke example.
- This is the point where local merge handling becomes part of the declared runtime design rather than an implementation detail.

### Phase 2: Post-`v0.2` Convergence And Guardrails

Range:

- `05e25a4` to `65e134f`
- Date window: 2026-03-23 to 2026-03-24
- Entry marker: first commit after `v0.2`
- Current head: `65e134f`

What defines the phase:

- The architecture baseline is already in place.
- The repo focuses on stabilization, operator discipline, prompt drift control, install guards, and request/state convergence.
- This is the phase most aligned with phase-one validation-loop hardening.

Canonical commits after `v0.2`:

```text
05e25a4 Compress public CLI for v0.2.1 cleanup
ce27ed8 提交图片
ea33695 收束版本
704fd44 Release v0.2.2 runtime degradation fixes
a25cb6e Add safe worktree tidy control
116aeeb Document harness iteration rules
cace6d4 Tighten control-plane prompt drift guardrails
26a1957 Lock phase-one operator goal
35038bc Refine harness self-driven guidance
fa12742 Tighten control-plane replan convergence
0d2510b Stabilize control-plane request focus
f6ab72e Document phase-one validation loop
44ba615 Generalize license copyright holder
7dca06d Avoid current-task raw log recursion
0a0958b Guard harness installs with runtime smoke checks
65e134f Repair request-task state convergence
```

Why `65e134f` is the current anchor:

- `git show --stat --summary 65e134f` is small compared with the earlier architecture commits, but it is sharply targeted.
- The change repairs convergence in request-task state handling, which is exactly the kind of post-architecture correction expected in this phase.

## Short Reference

- Need the bootstrap story: use Phase 0.
- Need the architecture pivot story: use Phase 1 with `9d86d01` and `f92a08c`.
- Need the current stabilization story: use Phase 2 with `65e134f`.
