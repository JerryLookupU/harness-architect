Legacy mapping shim: tasks

The old `tasks` meaning is now internalized into:
- orchestration packet `executionTasks`
- task-local `worker-spec.json`
- immutable dispatch tickets

Use this file only when translating older prompt stacks.
Do not treat `tasks` as a separate outer runtime stage.

Task generation rules when older mental models still ask for "tasks":
- emit task-local work that maps cleanly into `executionTasks`, `verificationPlan`, and `worker-spec.json`
- each task must have a clear objective, owned paths, acceptance markers, and explicit evidence expectations
- do not use free-text "done" as a task closure mechanism

Flow-specific task shaping:
- bug / failure / regression / error requests:
  - create a debugging task before any implementation task
  - debugging task must gather reproduction steps or failure evidence, isolate one active hypothesis, and define the minimal change boundary
  - implementation task is allowed only after the debugging task records evidence strong enough to justify a fix
- recommendation / compare / choose / best-way requests:
  - create an options / blueprint task first
  - that task must produce 2 to 3 viable options, trade-offs, and one recommendation before any implementation plan
  - only the selected option may become execution work
- continue / resume requests:
  - create a resume-inspection task first when state has not already been proven fresh
  - that task must read hot state, active tasks, session bindings, compact logs, and relevant skills / AGENTS instructions
  - only after that state read may coding or recovery tasks dispatch

Verification and review tasks:
- every execution task must carry a verification step that names expected commands, inspections, or artifacts
- a noop or already-satisfied outcome still needs concrete evidence
- when the change spans multiple files or high-risk surfaces, add a review dimension
- the review dimension may be:
  - a dedicated review task, or
  - a review checklist embedded in `verificationPlan`
- high-risk surfaces include routing, dispatch, verify, prompts, worktree, merge, auth, install, and other control-plane files
