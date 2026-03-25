Workflow: execute one bound task from a prepared dispatch ticket and worker-spec.

Steps:
1. read the dispatch ticket, worker-spec, and current verification context before editing
2. show current progress and remaining task-local work
3. implement one bound task-local slice at a time
4. keep task-local outputs in `worker-result.json`, `verify.json`, and `handoff.md`
5. run the relevant verification step before advancing
6. pause only for blockers, packet drift, or missing clarification

Guardrails:
- keep edits minimal and scoped to the bound task
- if implementation contradicts the current packet or worker-spec, surface the drift instead of silently freelancing
- treat execution as bounded task-local work, not as a second outer planning pass
- stop with a clear status summary when blocked or complete
