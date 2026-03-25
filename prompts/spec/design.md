Legacy mapping shim: design

The old `design` meaning is now internalized into packet and worker-spec fields:
- `selectedPlan`
- `rejectedAlternatives`
- `decisionRationale`
- `rollbackHints`

Use this file only when translating older prompt stacks.
Do not treat `design` as a separate outer runtime stage.
