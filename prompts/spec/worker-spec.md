Artifact: worker-spec

Purpose:
- describe one task-local execution slice without granting global control-plane authority

Required fields:
- taskId
- objective
- sharedContextPath
- constraints
- ownedPaths
- blockedPaths
- taskBudget
- acceptanceMarkers
- verificationPlan
- replanTriggers
- rollbackHints

Serialization:
- the worker-spec is exactly one JSON object
- do not wrap it in markdown fences
- do not add prose before or after the JSON
- use the exact required field names above

Optional runtime-carried metadata:
- dispatchId
- threadKey
- planEpoch
- attempt
- reasonCodes
- policyTags
- selectedPlan
- decisionRationale
- acceptedPacketId
- contractId
- acceptedPacketPath
- taskContractPath
- executionSliceId
- sharedContext
- taskGroupId
- batchLabel
- entityBatch
- outputTargets

Field conventions:
- `objective` is the execution objective for one task-local slice
- `sharedContextPath` points to the runtime-owned task-group context in `.harness`
- `sharedContext` is optional inline duplication for convenience; the file at `sharedContextPath` remains authoritative
- `constraints`, `ownedPaths`, `blockedPaths`, `acceptanceMarkers`, `replanTriggers`, and `rollbackHints` are arrays of strings
- `taskBudget` and `verificationPlan` stay machine-readable JSON objects, not prose paragraphs

Rules:
- one executable task gets one task-local `worker-spec.json`
- worker-spec may refine task-local execution but may not create new global task sets
- worker-spec should reference the task contract for done definition instead of duplicating `doneCriteria` / `requiredEvidence`
- worker-spec should inherit roster / file schema / source policy from `sharedContext` instead of forcing the worker to reconstruct them
- workers may edit only owned task-local paths and artifacts
- workers may not mutate global ledgers, leases, route decisions, merges, or completion state
