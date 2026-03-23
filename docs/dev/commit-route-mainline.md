# Commit Route: Mainline

This file freezes the canonical `main` story for this repo.

When someone asks "what actually shipped on main?", use the first-parent route first and treat side branches as supporting detail.

## Canonical View

Primary command:

```bash
git log --first-parent --reverse --decorate --oneline main
```

Observed first-parent route:

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

## How To Read It

- `v0.1.0` marks the end of the bootstrap/docs-first setup.
- `0f11dfd` is the first mainline merge that pulls the repo into the Klein closed-loop direction.
- `8bfd3be` is the second mainline merge and captures the log-search / blueprint-research branch as a single shipped step on `main`.
- `v0.2` at `f92a08c` is the mainline cut where worktree-first, merge-aware runtime becomes the baseline.
- Everything after `v0.2` is stabilization and operator/control-plane tightening.
- Current head is `65e134f`, which continues the post-`v0.2` stabilization line by repairing request-task convergence.

## Route By Era

### 1. Bootstrap to `v0.1.0`

`c28837e -> 2b3f18d`

- Repo starts as `harness-architect`.
- Early history is mostly positioning, README shaping, and install/shell-loop setup.
- `2b3f18d` freezes that initial surface as `v0.1.0`.

### 2. Klein Closed-Loop Expansion

`0f11dfd -> f92a08c`

- `0f11dfd` lands the closed-loop branch on main.
- `35c1b3e` renames the repo identity to `klein-harness`.
- `b9eaa6e` and `d93be28` establish runner-daemon behavior.
- `ad11fd2` and `8bfd3be` deepen the runtime and research/control-plane path.
- `f92a08c` cuts `v0.2` on the worktree-first merge-aware runtime.

### 3. Post-`v0.2` Stabilization

`05e25a4 -> 65e134f`

- CLI compression and version cleanup happen first.
- `704fd44` names the runtime degradation-fix release point.
- The rest of the line focuses on worktree hygiene, prompt drift control, phase-one operator discipline, request focus, install guards, and state convergence.

## Rule Of Thumb

For release or provenance discussions:

1. Start with the first-parent log.
2. Use branch history only to explain what each merge actually carried.
3. Use tags and the three anchor commits (`9d86d01`, `f92a08c`, `65e134f`) to explain phase changes.
