Workflow: verify that implementation matches the orchestration packet and task-local worker-spec.

Verify five dimensions:
- scopeCompletion
- behaviorCorrectness
- packetAlignment
- evidence quality
- review readiness when the change is multi-file or high-risk

Checks:
- compare worker-result claims against actual implementation evidence
- compare the accepted packet and task contract identities against the current dispatch
- compare requirements and scenarios against code and tests
- compare packet decisions and acceptance markers against the resulting implementation shape
- require command, diff, file, or runtime evidence for every success claim
- for bug or regression work, require failure / reproduction evidence plus proof that the chosen minimal fix addresses the confirmed cause
- for resume-sensitive work, require evidence that the worker read current state, active tasks, and relevant compact logs before continuing
- when the change spans multiple files or risky control-plane surfaces, run a short review checklist before allowing "done"

Severity rules:
- CRITICAL for missing required behavior, missing acceptance evidence, or incomplete tasks
- CRITICAL for verbal completion claims without command/file evidence
- WARNING for likely divergence from packet, worker-spec, or design intent
- WARNING for multi-file or high-risk changes that skipped review
- SUGGESTION for pattern or consistency improvements

Output:
- summary scorecard
- accepted packet / task contract references used for the evaluation
- evidence ledger with commands, inspected artifacts, and what each item proves
- prioritized findings
- review checklist findings when review was required
- recommendedNextAction for runtime follow-up (`archive`, `review`, `repair`, or `unblock`)
- concrete recommendations with file evidence where possible
