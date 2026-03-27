Artifact: orchestration packet

Purpose:
- hold the runtime-owned meaning that older stacks spread across `proposal/specs/design/tasks`

Required fields:
- objective
- constraints
- flowSelection
- policyTagsApplied
- selectedPlan
- rejectedAlternatives
- sharedContext
- executionTasks
- verificationPlan
- decisionRationale
- ownedPaths
- taskBudgets
- acceptanceMarkers
- replanTriggers
- rollbackHints

Serialization:
- the packet is exactly one JSON object
- do not wrap it in markdown fences
- do not add prose before or after the JSON
- use the exact required field names above
- do not add `workerSpecCandidates` to the packet; worker-spec candidates are judge-side siblings, not packet fields

Field conventions:
- `objective`, `flowSelection`, `selectedPlan`, and `decisionRationale` are strings
- `constraints`, `policyTagsApplied`, `ownedPaths`, `acceptanceMarkers`, `replanTriggers`, and `rollbackHints` are arrays of strings
- `rejectedAlternatives` is an array of objects with `candidateId` and `reason`
- `sharedContext` is one machine-readable task-group object that freezes shared planning decisions before worker dispatch
- `sharedContext.entitySelection` should answer who/what is in scope for the task group
- `sharedContext.contentContract` should answer file format, fields, length, output directory, and naming rules
- `sharedContext.sourcePlan` should answer where evidence should come from and how it should be checked
- `sharedContext.sharedPrompt` should hold the shared instructions that every worker in the task group can inherit
- `executionTasks`, `verificationPlan`, and `taskBudgets` stay machine-readable JSON objects or arrays, not prose paragraphs

Rules:
- the packet belongs to one accepted epoch
- same accepted epoch must not produce conflicting packet truth
- executionTasks must stay bounded and dispatchable
- shared planning decisions must live in `sharedContext`; do not force every worker slice to rediscover roster, format, or source rules
- `executionTasks` should mostly describe dispatchable worker tasks, not restate the full task-group prompt
- merge, archive, and final completion remain runtime-owned
