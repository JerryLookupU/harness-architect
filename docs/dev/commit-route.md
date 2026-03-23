# Commit Route

This file freezes the Klein evolution route in three views:

- first-parent mainline
- all-graph branch topology
- tag-range phase slices

Use this file as the compact provenance entrypoint.

## View 1: First-Parent Mainline

Primary command:

```bash
git log --first-parent --reverse --decorate --oneline main
```

Canonical route:

```text
c28837e init harness-architect trial repo
797cbcb docs: clarify codex-first positioning and add MIT license
ee3642b docs: rewrite README in bilingual codex-first format
b30e6f7 docs: add bilingual ascii architecture flow to README
f5ce202 update install.sh
f3523c5 update agent shell loop
2b3f18d (tag: v0.1.0) release: v0.1.0
0f11dfd Merge pull request #2 from JerryLookupU/feat-klein-closed-loop
62f2e18 Align Klein docs and smoke cleanup
35c1b3e Rename harness-architect to klein-harness
cd4ab76 Clean up README for Klein-Harness
399ee54 Refine Klein-Harness tagline
2c3c802 Fix harness-kick empty context rerun output
c1a0278 Harden bootstrap flow and review cadence
e2b72ab Add blueprint-architect skill
b9eaa6e Add built-in runner daemon mode
d93be28 Enable runner daemon by default after bootstrap
f1002b4 Refine README positioning and quick start
9bca4ff Add Klein surface hero image to README
b513e04 Add Klein hero image and README tagline
ad11fd2 Add closed-loop RCA allocation runtime
8bfd3be Merge branch 'feat/log-search-blueprint-research'
e77dc94 Polish README layout and messaging
c52b449 Harden machine-first control plane
f92a08c (tag: v0.2) Add worktree-first merge-aware runtime
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

Reading rule:

- `v0.1.0` closes the bootstrap/docs-first trial line.
- `0f11dfd` lands the closed-loop branch on `main`.
- `f92a08c` cuts the worktree-first runtime baseline as `v0.2`.
- everything after `v0.2` is convergence, guardrail, and phase-one hardening work

## View 2: All-Graph Branch Topology

Primary command:

```bash
git log --graph --decorate --oneline --all
```

Branches that matter:

### `feat-klein-closed-loop`

- merge commit: `0f11dfd`
- payload anchor: `9d86d01`
- branch tip still visible on remote: `4e8b1de`

Why it matters:

- this is the architecture pivot from trial harness repo to Klein closed-loop runtime
- the rename to `klein-harness` and subsequent cleanup happen immediately after this merge on `main`

### `feat/log-search-blueprint-research`

- merge commit: `8bfd3be`
- payload anchor: `c830f11`

Why it matters:

- it adds compact log search and blueprint research gating
- it is the bridge between the first closed-loop line and the worktree-first `v0.2` cut

## View 3: Tag-Range Phases

Primary commands:

```bash
git log --reverse --pretty=format:'%h%x09%s' v0.1.0..main
git log --reverse --pretty=format:'%h%x09%s' v0.2..main
```

### Phase 0: bootstrap to `v0.1.0`

Range:

- `c28837e` -> `2b3f18d`

Definition:

- repo setup
- README/positioning/install loop
- no full Klein runtime split yet

### Phase 1: closed-loop adoption to `v0.2`

Range:

- `9d86d01` -> `f92a08c`

Definition:

- Klein runtime architecture lands
- orchestrator/worker semantics sharpen
- runner daemon, RCA flow, blueprint/log-search gates appear
- worktree-first merge-aware runtime becomes the released baseline

### Phase 2: post-`v0.2` convergence

Range:

- `05e25a4` -> `65e134f`

Definition:

- CLI cleanup
- degradation fixes
- worktree tidy safety
- prompt drift and replan guardrails
- phase-one operator/body-vs-target discipline
- request-task convergence repair

## Anchor Commits

### `9d86d01`

- large architecture payload inside the closed-loop branch
- adds `docs/klein-architecture.md`
- expands runtime example surfaces substantially

### `f92a08c`

- tagged `v0.2`
- introduces worktree-first merge-aware runtime docs and smoke coverage
- should be treated as the second major architecture cut

### `65e134f`

- current head
- focused convergence repair on request-task state
- marks the present stabilization phase, not another broad architecture rewrite

## Practical Use

- release/history question: start with first-parent
- branch provenance question: check the graph anchors
- migration or architecture timing question: use the tag-range phases
