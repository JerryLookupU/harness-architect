# Klein-Harness

Klein is a repo-local agent runtime:

- native `tmux` is the execution shell
- native `codex exec` / `codex exec resume` is the worker runner
- Go owns bootstrap, routing, dispatch, lease, burst, checkpoint, outcome, verify, query, and control
- shell scripts are compatibility wrappers, not runtime source of truth

## Runtime MVP

Canonical implementation:

- CLI: `cmd/harness`
- runtime: `internal/runtime`
- bootstrap: `internal/bootstrap`
- routing: `internal/route`
- dispatch: `internal/dispatch`
- lease: `internal/lease`
- verify: `internal/verify`
- query: `internal/query`
- executor: `internal/executor/codex` and real `internal/tmux`

Compatibility surfaces that still exist:

- `scripts/harness-*.sh`
- `cmd/kh-codex`
- `cmd/kh-orchestrator`
- `cmd/kh-worker-supervisor`

These compatibility paths delegate into the Go runtime. They are no longer the canonical path.

## Install

```bash
./install.sh --force
```

Install effects:

- installs skills into `$CODEX_HOME/skills`
- installs the canonical `harness` CLI into `$CODEX_HOME/bin` when Go is available
- installs compatibility wrappers such as `harness-submit` and `harness-control`
- updates the managed block in `$CODEX_HOME/AGENTS.md` without touching user content outside the block
- updates managed profiles in `$CODEX_HOME/config.toml` without overwriting existing user profiles

Installed skills:

- `klein-harness`
- `blueprint-architect`
- `qiushi-execution`
- `systematic-debugging`
- `harness-log-search-cskill`
- `markdown-fetch`
- `generate-contributor-guide`

## Canonical CLI

Use `harness` directly:

```bash
harness init /path/to/repo
harness submit /path/to/repo --goal "Fix failing verify regression" --context docs/prd.md
harness tasks /path/to/repo
harness task /path/to/repo T-001
harness control /path/to/repo task T-001 status
harness daemon loop /path/to/repo --interval 30s --skip-git-repo-check
harness dashboard /path/to/repo --addr 127.0.0.1:7420 --skip-git-repo-check
```

Compatibility wrappers still work:

```bash
harness-submit /path/to/repo --goal "Fix failing verify regression"
harness-tasks /path/to/repo
harness-task /path/to/repo T-001
harness-control /path/to/repo task T-001 status
```

Those wrappers are thin shells that exec the Go CLI. They do not own runtime logic anymore.

## Execution Model

Fresh worker burst:

```text
codex exec --json --output-last-message <path> ...
```

Resume burst:

```text
codex exec resume <SESSION_ID> --json --output-last-message <path> ...
```

Real execution path:

1. `harness submit`
2. `harness daemon loop` or `harness dashboard`
3. route task
4. issue dispatch
5. acquire and claim lease
6. create real tmux session
7. run native codex inside tmux
8. persist checkpoint and outcome
9. ingest verify
10. expose query/control state

`harness dashboard` now starts the operator page and a repo-local `daemon loop` together by default. Use `--no-daemon` when you only want the read surface without background scheduling.

The operator page now runs on a go-zero `rest.Server`, and when daemon mode is enabled it is started in the same go-zero `ServiceGroup` as the repo-local scheduler loop. This keeps the HTTP surface and the runtime loop under one service lifecycle while leaving the core runtime state machine in place.

`harness control /repo task <TASK_ID> attach` uses the real tmux session name. In non-interactive contexts it prints the exact attach command.

## Planning And Dispatch Model

Klein now treats planning, dispatch, and execution as three different layers:

1. planner / judge decide the shared task-group context
2. judge emits the actual dispatchable task list
3. tmux workers receive the shared context plus one current execution slice

For corpus-style or batch-style tasks such as "10 mathematicians" or "1000 scientists", the planning layer should freeze the shared decisions before any worker starts:

- who or what is in scope
- the file schema and required fields
- the length and format constraints
- the output directory and naming rules
- the source / research policy

Those shared decisions now live in packet-level `sharedContext` and are written into task-local `.harness/artifacts/<TASK>/<DISPATCH>/shared-context.json`.

That means workers are expected to:

- read `dispatch-ticket`
- read `worker-spec`
- read `shared-context.json`
- read `task-contract`
- execute only the current slice

Workers should not rediscover the full roster, rewrite the common prompt, or reopen planner / judge decisions unless the runtime artifacts contradict each other.

`ownedPaths` remains a boundary and audit surface. It is not the primary human task list.

## Worker Prompt Contract

The worker prompt is intentionally lighter than before:

- planning metadata remains in `.harness`
- shared task-group background remains in `.harness`
- the active tmux node label follows `[harness:<task-id>] <node-task-description>`
- the worker focuses on the current batch or milestone, then writes `worker-result.json`, `verify.json`, and `handoff.md`

This separation is important for large-volume task groups. The worker should act like a bounded executor, not a second outer orchestrator.

## Runtime State

Authoritative files:

- `.harness/requests/queue.jsonl`
- `.harness/task-pool.json`
- `.harness/state/dispatch-summary.json`
- `.harness/state/lease-summary.json`
- `.harness/state/session-registry.json`
- `.harness/state/runtime.json`
- `.harness/state/verification-summary.json`
- `.harness/state/tmux-summary.json`
- `.harness/checkpoints/*`
- `.harness/artifacts/*`

Derived or view-oriented state:

- `.harness/state/completion-gate.json`
- `.harness/state/guard-state.json`

Writers and readers are documented in [docs/runtime-mvp.md](/Users/linzhenjie/code/claw-code/harness-architect/docs/runtime-mvp.md).

## Guardrail Mapping

Klein does not implement a Hookify runtime. The old dotfiles intent is mapped into route/prompt/policy/runtime surfaces:

| dotfiles intent | Klein mapping |
| --- | --- |
| bug / failure / regression | `policy_bug_rca_first` + debugging-first worker guidance + `systematic-debugging` skill |
| recommendation / compare / choose | `policy_options_before_plan` + `blueprint-architect` |
| continue / resume | `policy_resume_state_first` + state/log/skill-first resume flow |
| verify-before-stop | prompt evidence rules + runtime completion gate |
| review-before-done | review-required metadata + runtime review evidence gate |
| methodology discipline | qiushi-inspired fact-first / focus-first / verify-first mapping in prompts, planning trace, and managed AGENTS guidance |

Qiushi-inspired design mapping is documented in [docs/qiushi-integration.md](/Users/linzhenjie/code/claw-code/harness-architect/docs/qiushi-integration.md).
Current role split:
- `b3e` / B3Ehive = orchestration and packet convergence
- `qiushi-execution` = execution / validation loop discipline

## Tests

Unit tests:

```bash
go test ./...
```

If your local macOS toolchain currently fails with a `libtapi.dylib` signature error during link, use:

```bash
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go build ./cmd/harness
```

That is a local linker environment issue, not a harness runtime requirement.

Coverage-oriented integration tests:

```bash
go test -tags=integration ./...
```

These tests are for regression coverage, not the final source of truth for runtime behavior.

Real validation should happen through actual harness usage in the target repo and environment:

- submit a real task
- let `harness daemon loop` or `harness dashboard` drive the runtime
- inspect the real task/thread/dispatch/tmux/verify surfaces
- iterate from those real outcomes instead of synthetic smoke flows

Klein intentionally prefers real runtime validation over synthetic smoke flows:

- use real repo state
- use real tmux workers
- use real verify / handoff artifacts
- use coverage tests for regression protection, not as a replacement for real operator feedback

## Migration Notes

- `control.py` is no longer on the canonical path
- `.harness/bin/*` is no longer the system source of truth
- `scripts/harness-*.sh` now forward into Go
- the runtime no longer pretends `/bin/sh -lc` is `tmux`

Details and old-to-new mapping are in [docs/refactor-runtime-migration.md](/Users/linzhenjie/code/claw-code/harness-architect/docs/refactor-runtime-migration.md).
