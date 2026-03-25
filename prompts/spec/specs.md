Legacy mapping shim: specs

The old `specs` meaning is now internalized into runtime-owned packet fields:
- `verificationPlan`
- `acceptanceMarkers`
- `constraints`

Use this file only to map older prompts into the packet model.
Do not treat `specs` as a separate outer runtime stage.
