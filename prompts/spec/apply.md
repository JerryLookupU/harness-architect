Workflow: execute one bound task from a prepared dispatch ticket and worker-spec.

Steps:
1. read the dispatch ticket, worker-spec, shared task-group context, and task contract before editing
2. restate the current slice in one short line: what batch or milestone this worker owns right now
3. execute only the current task-local slice while inheriting roster / format / source rules from shared context
4. keep task-local outputs in `worker-result.json`, `verify.json`, and `handoff.md`
5. run the relevant verification step before advancing
6. pause only for blockers, packet drift, or missing shared planning inputs

Guardrails:
- keep edits minimal and scoped to the bound task
- if implementation contradicts the current packet or worker-spec, surface the drift instead of silently freelancing
- treat execution as bounded task-local work, not as a second outer planning pass
- do not rediscover the full task roster, file schema, or source policy if shared context already defines them
- stop with a clear status summary when blocked or complete
