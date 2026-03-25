Legacy mapping shim: tasks

The old `tasks` meaning is now internalized into:
- orchestration packet `executionTasks`
- task-local `worker-spec.json`
- immutable dispatch tickets

Use this file only when translating older prompt stacks.
Do not treat `tasks` as a separate outer runtime stage.
