# Runtime Request Spec

## Goal

Unify all upstream entry points into one project runtime:

- `OpenClaw`
- `shell`
- `cron`
- future external callers

All of them submit requests.
Only the project runtime decides orchestration, task dispatch, recovery, verification, and reporting.

## Entry Model

Global entry commands:

- `harness-init <ROOT>`
- `harness-bootstrap <ROOT> <GOAL> [STACK_HINT] [kick options...]`
- `harness-submit <ROOT> --kind <KIND> --goal <TEXT> [options...]`
- `harness-report <ROOT> [--request-id <ID>] [--format text|json]`

Project-local entry commands under `.harness/bin`:

- `harness-submit`
- `harness-report`
- `harness-runner`
- `harness-status`
- `harness-query`
- `harness-dashboard`
- `harness-watch`
- `harness-verify-task`

## Lifecycle

### 1. Init

`harness-init` creates the minimal operator/runtime skeleton without invoking a model.

Creates:

- `.harness/bin/*`
- `.harness/scripts/*`
- `.harness/state/*`
- `.harness/requests/queue.jsonl`
- `.harness/requests/archive/`
- `.harness/state/request-index.json`
- `.harness/state/request-task-map.json`
- `.harness/project-meta.json`

### 2. Bootstrap

`harness-bootstrap` is the first model-backed orchestration round.

Responsibilities:

- inspect the repo
- create `.harness/standards.md`
- create `.harness/verification-rules/manifest.json`
- create `features/work-items/spec/task-pool/context-map/progress/session-registry`
- refresh hot state
- optionally launch daemon mode

### 3. Submit

All incremental requests go through `harness-submit`.

Supported request kinds in phase 1:

- `bootstrap`
- `analysis`
- `research`
- `implementation`
- `audit`
- `status`
- `change`
- `replan`
- `stop`

Phase 1 behavior:

- append request to queue
- update request index
- do not directly mutate task-pool
- let runtime/daemon consume later

### 4. Report

`harness-report` summarizes:

- request queue counts
- active request overview
- current runtime focus
- current active task
- active/recoverable/stale runners
- verification summary

## Data Contracts

### `.harness/requests/queue.jsonl`

Append-only queue of incoming requests.

Each request contains at least:

- `requestId`
- `seq`
- `source`
- `kind`
- `goal`
- `projectRoot`
- `contextPaths`
- `threadKey`
- `priority`
- `scope`
- `mergePolicy`
- `replyPolicy`
- `status`
- `createdAt`

### `.harness/state/request-index.json`

Machine-readable summary for request queries.

Contains:

- `schemaVersion`
- `generator`
- `generatedAt`
- `nextSeq`
- `requests`

### `.harness/state/request-task-map.json`

Reserved for orchestration binding between requests and tasks.

Phase 1 initializes the file but does not yet drive bindings automatically.

## Cache Hit Strategy

Cache-sensitive work should be merged with the orchestration/runtime stage, not with ad hoc callers.

Recommended split:

- request intake + orchestration routing:
  use one long-lived orchestration session
- worker execution:
  use task-level `fresh` / `resume`
- lint/verification:
  run in the same post-run reconcile chain as runner
- status/report:
  prefer reading hot state files, not opening new model sessions

## Phase 1 Delivery

Phase 1 implemented by this repo change:

- startup split into `init/bootstrap/submit/report`
- request queue + index initialization
- global helper wrappers
- project-local submit/report tools
- verification integrated into runner post-run reconcile

Still deferred:

- request-aware daemon consumption
- automatic request-to-task binding
- request-level completion transitions
