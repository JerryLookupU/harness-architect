Artifact: worker-result

Purpose:
- capture the terminal outcome of one task-local worker run

Required fields:
- dispatchId
- taskId
- threadKey
- planEpoch
- status
- summary
- changedPaths
- producedArtifacts
- acceptanceEvidence
- nextSuggestedKind

Rules:
- worker-result is task-local and never a global completion claim
- free-text claims do not close the loop; `completion-gate.json` does
- noop completion is valid only when acceptance is already satisfied and verify evidence records that fact
