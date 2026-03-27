# Context

用户要的不是泛泛架构建议，而是一份**能直接指导当前代码库实施**的工程蓝图，并且必须把 **规划 / 执行 / 验证** 三个环节拆清楚。

当前代码已经具备主链路，不需要另起炉灶：

- `internal/runtime/service.go:77` 的 `Submit` 已经是 single-entry intake，并且已有 `classifySubmission`（约 `:610`）提供 `frontDoorTriage / normalizedIntentClass / fusionDecision / targetThreadKey / targetPlanEpoch`
- `internal/route/gate.go:44` 的 `Evaluate` 已经是 canonical route gate，已覆盖 `context_enrichment -> replan`、stale epoch、checkpoint block、resume/session contest、dispatch
- `internal/worker/manifest.go:41` 的 `Prepare` 已经会写出 accepted packet、task contract、worker-spec、dispatch ticket
- `internal/verify/gate.go:110` 的 completion gate 已经消费 packet / contract / scorecard / evidence / review / execution progress
- `internal/query/service.go:124` 的 `Task` 已经是 operator read model 主入口

因此，这一轮蓝图的目标不是设计新 runtime，而是**把现有 runtime 收紧成可直接执行的实现路线**：

1. 先冻结对象与路径约定
2. 再补 request/task binding 与缺失 hot state
3. 再统一 epoch / verify / query 的读取语义
4. 再补 task-scoped gate surfaces 与 revision-safe artifact writes
5. 最后用现有覆盖测试与真实使用验证封口

---

# 推荐方案

## 一、规划环节（先冻结，不直接改行为）

### 1. 冻结本轮实现边界

本轮只做以下六件事：

1. 让 `Submit` 不再“永远创建新 task”，而是先做 binding，再决定复用已有 task 还是创建新 task
2. 补齐 request hot-state：
   - `request-summary.json`
   - `request-index.json`
   - `request-task-map.json`
3. 统一 `thread-state.json` 中的 `planEpoch` 语义，消除 writer / reader 漂移
4. 让 accepted packet / task contract / packet progress 的写入具备 revision/CAS 语义
5. 让 completion gate / guard state 支持 **task-scoped latest snapshots**，同时保留现有 singleton alias 兼容路径
6. 清理 worker prompt / query 中引用了但当前并未实现的 hot summaries

明确不做：

- 不新增调度器
- 不重写 `nextRunnableTask`
- 不重写 route 逻辑
- 不引入第二套 verify / gate
- 不扩写 log/lineage 子系统超出本轮最小闭环

### 2. 冻结关键对象与路径

#### 2.1 request binding objects

在 `internal/runtime/model.go` 增加并冻结：

- `RequestSummary`
- `RequestIndex`
- `RequestTaskMap`

建议字段：

- `RequestSummary`
  - `latestRequestId`
  - `latestTaskId`
  - `latestThreadKey`
  - `frontDoorTriage`
  - `normalizedIntentClass`
  - `fusionDecision`
  - `bindingAction` (`created_new_task | reused_existing_task`)
  - `targetPlanEpoch`
  - `requestCount`
  - `reusedTaskCount`
  - `createdTaskCount`

- `RequestIndex`
  - `requestsById`
  - `latestRequestByTaskId`
  - `latestRequestByThreadKey`
  - `latestRequestByIdempotencyKey`

- `RequestTaskMap`
  - `requestToTask`
  - `requestToThread`
  - `taskToRequests`
  - `threadToRequests`
  - `threadToTasks`

#### 2.2 RequestRecord 最小扩展

在 `internal/runtime/model.go:5` 的 `RequestRecord` 仅新增：

- `bindingAction`
- `reusedTaskId,omitempty`

不再额外发明第二套 request/task truth 字段。

#### 2.3 thread-state epoch 统一字段

在 `internal/runtime/model.go:38` 的 `ThreadEntry` 增加：

- `currentPlanEpoch`
- `latestValidPlanEpoch`

并保留现有 `planEpoch` 作为兼容镜像字段。

语义冻结：

- `currentPlanEpoch`：当前 thread 头部 epoch
- `latestValidPlanEpoch`：当前已被接受、可供 route 作为最新有效 epoch 读取的 epoch
- `planEpoch`：兼容镜像，值等于 `currentPlanEpoch`

#### 2.4 task-scoped verify snapshots

在 `internal/adapter/project.go` 增加路径 helper，冻结两组 task-scoped 文件：

- `.harness/state/completion-gate-<taskId>.json`
- `.harness/state/guard-state-<taskId>.json`

同时保留现有 singleton alias：

- `.harness/state/completion-gate.json`
- `.harness/state/guard-state.json`

#### 2.5 revision-safe runtime objects

在 `internal/orchestration/runtime_objects.go` 中冻结 accepted packet / task contract / packet progress 的 revision 语义：

- 保留当前 domain `schemaVersion`
- 增加 `revision`
- 使用 domain-specific CAS 写入
- **不要** 直接改成 `state.Metadata` / `state.WriteSnapshot` 替代品，以免破坏现有 schema

### 3. 冻结 task reuse 规则

`Submit` 的 binding 冻结为以下规则：

#### 可复用已有 task 的条件

仅当以下全部满足时复用：

- 命中已有 thread
- 命中的 latest task 状态属于：`"" | queued | needs_replan | recoverable`
- 该 task 尚未持有 active execution ownership，具体要求：
  - `LastDispatchID == ""`
  - `LastLeaseID == ""`
  - `TmuxSession == ""`

#### 必须新建同 thread follow-up task 的条件

任一满足即新建：

- latest task 处于 `routing | running`
- latest task 已 terminal：`completed | archived | blocked`
- latest task 已有 dispatch / lease / tmux ownership
- 需要单独 follow-up，而不是把 request 合并进现有 queued work

### 4. 冻结验证口径

本轮的成功不是“代码改完”，而是三类问题都能直接回答：

- planning：这个 request 绑定到了哪个 thread / task / epoch
- execution：这次 dispatch 的 authoritative packet / contract / spec 是哪一组
- verification：这个 task 的 gate / guard / next action 是不是 task-scoped 且可独立读取

---

## 二、执行环节（按代码路径实施）

### Phase A：补模型与路径，不动主链路决策

#### 目标

先把 schema 和 path helper 补齐，让后续逻辑改造有稳定承载面。

#### 关键文件

- `internal/runtime/model.go`
- `internal/adapter/project.go`

#### 具体实施

1. 在 `internal/runtime/model.go` 添加：
   - `RequestSummary`
   - `RequestIndex`
   - `RequestTaskMap`
   - `RequestRecord.bindingAction`
   - `RequestRecord.reusedTaskId`
   - `ThreadEntry.currentPlanEpoch`
   - `ThreadEntry.latestValidPlanEpoch`

2. 在 `internal/adapter/project.go` 的 `Paths` / resolve helper 中增加：
   - `RequestSummaryPath`
   - `RequestIndexPath`
   - `RequestTaskMapPath`
   - `CompletionGateTaskPath(taskID)`
   - `GuardStateTaskPath(taskID)`

3. 保持旧路径继续可读，不在本阶段删除任何旧字段/旧文件。

#### 交付结果

- 新对象有稳定类型定义
- 新热状态有稳定路径约定
- 后续逻辑改造不需要边写边猜 schema

---

### Phase B：重构 Submit 为「classify -> bind -> materialize」

#### 目标

在不改 canonical CLI 的前提下，让 `Submit` 真正变成 single-entry intake，而不是 single-entry + always-new-task。

#### 关键文件

- `internal/runtime/service.go`
- `internal/runtime/model.go`
- 可复用函数：
  - `classifySubmission` in `internal/runtime/service.go:610`
  - `latestMatchingTask` in `internal/runtime/service.go:1005`
  - `updateIntakeState` in `internal/runtime/service.go:661`
  - `refreshThreadState` in `internal/runtime/service.go:1077`
  - `refreshTodoSummary` in `internal/runtime/service.go:1126`

#### 具体实施

1. 从 `Submit`（`internal/runtime/service.go:77`）中拆出新函数：
   - `resolveSubmissionBinding(...)`
   - `writeRequestHotState(...)`

2. 保留 `classifySubmission` 作为前置，不重写算法。

3. 在 `Submit` 中改为：
   - 先 `classifySubmission`
   - 再 `resolveSubmissionBinding`
   - 若 binding = reuse：
     - 不生成新的 `taskID`
     - request 直接绑定到已有 task
     - 仅最小更新 task 的 summary/description/statusReason/updatedAt（避免意外改 status）
   - 若 binding = create：
     - 走现有 task 创建逻辑

4. 无论 reuse 还是 create，都必须：
   - append 到 `queue.jsonl`
   - 写 `request-summary.json`
   - 写 `request-index.json`
   - 写 `request-task-map.json`
   - 刷新 `intake-summary.json`
   - 刷新 `thread-state.json`
   - 刷新 `change-summary.json`
   - 刷新 `todo-summary.json`

5. `RequestRecord.TaskID` 在 reuse 场景下必须写入被复用的 task id；这是 route/query 后续读取的前提。

#### 交付结果

- submit 对重复/补充请求不再无脑膨胀 task 数量
- request 到 task/thread 的 binding 有显式热状态
- route/query 不再只能回扫 `queue.jsonl`

---

### Phase C：统一 thread-state 的 epoch 读写语义

#### 目标

消除 `refreshThreadState` 与 `adapter.LoadLatestPlanEpoch` 对 epoch 字段理解不一致的问题。

#### 关键文件

- `internal/runtime/service.go`
- `internal/adapter/project.go`

#### 具体实施

1. 修改 `refreshThreadState`，写出：
   - `planEpoch`
   - `currentPlanEpoch`
   - `latestValidPlanEpoch`

2. 写入规则：
   - `currentPlanEpoch = max(existing, task.PlanEpoch)`
   - `planEpoch = currentPlanEpoch`
   - `latestValidPlanEpoch` 先按保守规则与当前 accepted planning 边界对齐；在缺少更细 event source 时，至少保证不会回退

3. 修改 `adapter.LoadLatestPlanEpoch`（当前在 `internal/adapter/project.go:311` 一带）：
   - 优先读 `latestValidPlanEpoch`
   - fallback 到 `currentPlanEpoch`
   - 再 fallback 到 `planEpoch`

#### 交付结果

- route stale epoch 判断不再依赖含糊字段
- thread-state 成为稳定的 epoch 热读面

---

### Phase D：把 accepted packet / task contract / packet progress 改成 revision-safe writes

#### 目标

让 planning/execution truth object 不再是“普通覆盖写”，而是显式具备版本安全。

#### 关键文件

- `internal/orchestration/runtime_objects.go`
- `internal/worker/manifest.go`
- 可复用调用点：
  - `Prepare` in `internal/worker/manifest.go:41`
  - `WriteAcceptedPacket` in `internal/orchestration/runtime_objects.go`
  - `WriteTaskContract` in `internal/orchestration/runtime_objects.go`
  - `WritePacketProgress` in `internal/orchestration/runtime_objects.go`

#### 具体实施

1. 在 runtime objects 中增加 `revision` 字段。
2. 新增 domain-specific CAS helper：
   - `WriteAcceptedPacketCAS`
   - `WriteTaskContractCAS`
   - `WritePacketProgressCAS`
3. 在 `Prepare` 中把 accepted packet / task contract 的写入切到 CAS helper。
4. 在 verify progress 更新处把 packet-progress 写入切到 CAS helper。
5. revision 冲突时返回显式错误，不做 silent overwrite。

#### 交付结果

- authoritative truth object 拥有最小版本安全
- stale write 不再悄悄覆盖最新 truth

---

### Phase E：把 completion gate / guard 改成 task-scoped latest snapshots

#### 目标

解决当前 `query.Task` 只在 singleton gate/guard 恰好属于该 task 时才能显示状态的问题。

#### 关键文件

- `internal/verify/gate.go`
- `internal/runtime/control.go`
- `internal/query/service.go`
- 关键读取/写入点：
  - `buildCompletionGate` in `internal/verify/gate.go:110`
  - `Task` in `internal/query/service.go:124`
  - `ArchiveTask` in `internal/runtime/control.go`

#### 具体实施

1. 在 verify completion state 更新逻辑中：
   - 先写 `.harness/state/completion-gate-<taskId>.json`
   - 先写 `.harness/state/guard-state-<taskId>.json`
   - 再同步写当前 singleton alias

2. 在 `query.Task` 中调整读取顺序：
   - 优先读 task-scoped gate/guard
   - fallback 到 singleton alias

3. 在 `ArchiveTask` 中按目标 task 读取对应 task-scoped gate/guard，而不是默认只读 singleton。

#### 交付结果

- 多 task / release board / 非当前活动 task 的 gate 状态可正确查询
- singleton 仍保留兼容能力

---

### Phase F：清理 prompt/query 对未实现 hot summaries 的依赖

#### 目标

避免 worker prompt 和 operator surface 指向根本不存在的 summary 文件。

#### 关键文件

- `internal/worker/manifest.go`
- `internal/query/service.go`
- 可选同步文档：
  - `docs/control-plane-state.md`
  - `skills/klein-harness/references/schema-contracts.md`

#### 具体实施

1. worker prompt 中，强引用只保留本轮已实现的 hot summaries：
   - `runtime.json`
   - `request-summary.json`
   - accepted packet path
   - task contract path
   - task-scoped gate/guard（如需要）

2. 对 `lineage-index.json` / `log-index.json` 这类当前未稳定实现的面：
   - 若文件存在再提示读取
   - 否则不要在 prompt 中当作 mandatory truth surface

3. `query.Task` 继续以 authoritative objects + 真正存在的 summaries 组装 read model，不再暗示不存在的 surface。

#### 交付结果

- prompt/code/doc 三者不再漂移
- worker 不会被要求读取不存在的关键状态面

---

## 三、验证环节（必须可运行、可封口）

### 1. 单元测试

#### 1.1 submit / binding

文件：`internal/runtime/submit_test.go`

新增/修改测试：

1. `TestSubmitReusesQueuedTaskForSameCanonicalGoal`
   - 第一次 submit 创建 task
   - 第二次相同 canonical goal 且目标 task 仍 queued
   - 断言：
     - task pool 中仍只有一个对应 queued task
     - 新 request 绑定到旧 task
     - `bindingAction = reused_existing_task`

2. `TestSubmitCreatesNewTaskWhenMatchedTaskRunning`
   - 匹配 thread 已存在，但 latest task = running
   - 断言：第二次 submit 新建 follow-up task，threadKey 相同，taskID 不同

3. `TestSubmitWritesRequestSummaryIndexAndTaskMap`
   - 断言三个新 hot state 文件存在且内容相互一致

4. `TestRefreshThreadStateWritesCurrentAndLatestValidPlanEpoch`
   - 断言 thread-state 同时写出 `planEpoch/currentPlanEpoch/latestValidPlanEpoch`

### 2. 集成测试

#### 2.1 runtime integration

文件：`internal/runtime/runtime_integration_test.go`

新增测试：

1. 相同 goal 的双 submit（第二次发生在 queued 阶段）
   - daemon 每个 cycle 仍只推进一个绑定 task

2. 同 thread follow-up 请求（发生在 running 或 terminal task 后）
   - 断言生成新 task，但 thread 复用

3. 多 task 独立 gate surface
   - 跑两个 task
   - 断言两个 task 都有各自的 `completion-gate-<task>.json` / `guard-state-<task>.json`
   - query 老 task 时仍能读回自己的 gate

### 3. query 测试

文件：`internal/query/service_test.go`

新增测试：

1. `Task()` 优先读取 task-scoped completion/guard
2. `Task()` 能经由 request-index（必要时 fallback 到 queue）拿到最新 request
3. release board 聚合多个 task 时，不再受 singleton gate 覆盖问题影响

### 4. verify 测试

文件：`internal/verify/service_test.go`

基于现有覆盖继续补：

- task-scoped gate/guard 写出测试
- singleton alias 仍同步可读测试
- archive 读取 task-scoped gate 的测试

现有高价值测试必须保持通过，尤其是：

- `TestIngestPassedWithoutEvidenceDoesNotComplete`
- `TestIngestReviewRequiredWithoutReviewEvidenceDoesNotComplete`
- `TestIngestPassedWithEvidenceCompletes`
- `TestIngestPassedWithRemainingExecutionSlicesEmitsReplan`
- `TestIngestPassedWithBlockingFindingsDoesNotComplete`

这些测试已经证明 verify/gate 主链路是正确骨架，不能在本轮被破坏。

### 5. runtime object 测试

新增或补充 `internal/orchestration/runtime_objects.go` 对应测试：

- 首次写 accepted packet / task contract / progress -> revision=1
- 带正确 expected revision 的第二次写入成功 -> revision 递增
- 带 stale revision 的写入失败 -> 返回冲突错误

### 6. 手动/端到端验证

依赖现有仓库说明中的命令，不发明新流程：

#### 单测

- `go test ./...`

#### 集成

- `go test -tags=integration ./...`

#### 真实使用验证

基于真实 harness 流程：

- 提交真实任务
- 使用 `harness daemon loop` 或 `harness dashboard` 推进
- 从 task / thread / dispatch / tmux / verify / dashboard 读取真实反馈

### 7. 验收矩阵

必须全部成立，才算这轮蓝图落地成功：

1. **planning 成功定义**
   - submit 后能明确回答：request 绑定到哪个 thread、哪个 task、哪个 targetPlanEpoch
   - request-summary / request-index / request-task-map 三者一致

2. **execution 成功定义**
   - dispatch 时 authoritative packet / contract / worker-spec 仍由当前主链路生成
   - accepted packet / task contract / packet progress 不再是静默覆盖写

3. **verification 成功定义**
   - 每个 task 都有独立 gate/guard latest snapshot
   - query 非当前活动 task 时仍能正确显示它自己的 completion state
   - 现有 verify/gate 证据闭环测试全部保持通过

4. **兼容成功定义**
   - 旧 singleton gate/guard 路径仍可工作
   - route/dispatch/lease 主链路保持不变
   - `nextRunnableTask` 不被这轮改造破坏

---

# 关键风险与控制

## 风险 1：task reuse 过度，破坏执行 ownership

控制：

- 只复用无 dispatch / 无 lease / 无 tmux session 的 queued 类 task
- running/routing/terminal 一律不复用

## 风险 2：直接把 runtime objects 切到 `state.WriteSnapshot`，破坏现有 domain schema

控制：

- 在 `internal/orchestration/runtime_objects.go` 内实现 domain-specific CAS helper
- 不直接套 `state.Metadata`

## 风险 3：task-scoped gate 改造破坏当前 archive/control 行为

控制：

- 双写：task-scoped latest + singleton alias
- query 先 task-scoped 再 fallback singleton
- control 以 task-scoped 为准，singleton 仅兼容

## 风险 4：prompt 继续引用未实现的 summary，造成 worker 指令漂移

控制：

- 本轮把 request-summary/index/map 先实现
- 其他未实现 surface 改为 conditional reference，不再硬编码为必读

---

# 关键文件清单

核心改动文件：

- `internal/runtime/model.go`
- `internal/runtime/service.go`
- `internal/adapter/project.go`
- `internal/orchestration/runtime_objects.go`
- `internal/worker/manifest.go`
- `internal/verify/gate.go`
- `internal/runtime/control.go`
- `internal/query/service.go`

核心验证文件：

- `internal/runtime/submit_test.go`
- `internal/runtime/runtime_integration_test.go`
- `internal/query/service_test.go`
- `internal/verify/service_test.go`
- `internal/worker/manifest_test.go`
- `internal/orchestration/runtime_objects*_test.go`（如需新增）

---

# 实施顺序（推荐照此执行）

1. 先改 `model.go` 与 `adapter/project.go`，冻结类型与路径
2. 再改 `runtime/service.go`，实现 bind-then-create/reuse
3. 再改 `refreshThreadState` 与 `LoadLatestPlanEpoch`，统一 epoch 语义
4. 再改 `runtime_objects.go` + `worker/manifest.go`，补 revision-safe writes
5. 再改 `verify/gate.go` + `runtime/control.go` + `query/service.go`，切 task-scoped gate/guard
6. 最后改 `worker/manifest.go` 的 prompt 引用与相关测试
7. 跑 unit -> integration -> 真实使用验证，按验证矩阵逐项封口
