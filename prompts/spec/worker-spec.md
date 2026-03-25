Artifact: worker-spec

Purpose:
- describe one task-local execution slice without granting global control-plane authority

Required fields:
- taskId
- objective
- constraints
- ownedPaths
- blockedPaths
- taskBudget
- acceptanceMarkers
- verificationPlan
- replanTriggers
- rollbackHints

Rules:
- one executable task gets one task-local `worker-spec.json`
- worker-spec may refine task-local execution but may not create new global task sets
- workers may edit only owned task-local paths and artifacts
- workers may not mutate global ledgers, leases, route decisions, merges, or completion state
