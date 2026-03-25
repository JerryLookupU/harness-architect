Workflow: archive a completed change after implementation and verification.

Steps:
1. confirm task-local artifacts, dispatch lineage, and verification state
2. confirm `completion-gate.json` is satisfied by evidence, not worker self-claims
3. summarize any remaining warnings before archive
4. archive only after the operator or policy allows the move

Guardrails:
- do not hide incomplete artifacts or tasks
- preserve a clear record of what was verified, skipped, or left as warning
- archive should close a converged runtime change, not bypass unresolved verification
