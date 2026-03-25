Workflow: verify that implementation matches the orchestration packet and task-local worker-spec.

Verify three dimensions:
- completeness
- correctness
- coherence

Checks:
- compare worker-result claims against actual implementation evidence
- compare requirements and scenarios against code and tests
- compare packet decisions and acceptance markers against the resulting implementation shape

Severity rules:
- CRITICAL for missing required behavior, missing acceptance evidence, or incomplete tasks
- WARNING for likely divergence from packet, worker-spec, or design intent
- SUGGESTION for pattern or consistency improvements

Output:
- summary scorecard
- prioritized findings
- concrete recommendations with file evidence where possible
