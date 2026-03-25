Artifact: orchestration packet

Purpose:
- hold the runtime-owned meaning that older stacks spread across `proposal/specs/design/tasks`

Required fields:
- objective
- constraints
- selectedPlan
- rejectedAlternatives
- executionTasks
- verificationPlan
- decisionRationale
- ownedPaths
- taskBudgets
- acceptanceMarkers
- replanTriggers
- rollbackHints

Rules:
- the packet belongs to one accepted epoch
- same accepted epoch must not produce conflicting packet truth
- executionTasks must stay bounded and dispatchable
- merge, archive, and final completion remain runtime-owned
