Artifact: dispatch ticket

Purpose:
- define the immutable execution authority for one actual dispatch

Required fields:
- dispatchId
- idempotencyKey
- taskId
- threadKey
- planEpoch
- attempt
- requiredSummaryVersion
- allowedWriteGlobs
- blockedWriteGlobs
- worktreePath
- budget
- leaseTtlSec
- workerSpecPath

Rules:
- dispatch is valid only after route
- stale leases or stale dispatches must not overwrite newer accepted execution
- the ticket defines authority boundaries; workers do not widen them
