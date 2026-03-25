Workflow: synthesize or refresh one orchestration packet from a requirement.

Goal:
- create a runtime-owned packet that is ready for route-first-dispatch-second execution

Steps:
1. derive or normalize the primary objective and constraints
2. decide whether the accepted epoch needs a new packet or only packet reuse
3. when synthesis is needed, shape one packet candidate and task-local worker-spec candidates together
4. stop when dispatch-ready task slices, verification intent, and rollback hints are explicit

Guardrails:
- read current runtime summaries before creating a fresh packet
- ask for clarification only when ambiguity would materially change packet ownership or acceptance
- do not copy meta-rules into packet output
- end by stating what is ready for dispatch and what still requires route/runtime judgment
