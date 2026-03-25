Legacy mapping shim: proposal

The old `proposal` artifact is now internalized into orchestration packet fields:
- `objective`
- `constraints`
- `decisionRationale`

Use this file only when translating an older mental model.
Do not treat `proposal` as a separate outer runtime stage.
