# Klein-Harness 架构补完与收口实施方案

## 0. 文档定位

本文是面向当前 `Klein-Harness` 的一份**可直接落地执行**的架构补完方案。

目标不是推翻现有设计，而是在当前已经成立的：

- `route-first-dispatch-second`
- `repo-local state plane`
- `runtime-owned completion gate`
- `worker task-local authority`

这些核心原则之上，补齐长任务 harness 最关键的三个闭环：

1. **contract-first 闭环**
2. **independent evaluator 闭环**
3. **authoritative vs derived 真相边界闭环**

本文同时满足以下要求：

- 提供**最小可执行任务**定义
- 方案按**原子任务**拆解
- 提供**谱系式 to-do / checklist**
- 提供**最终效果描述**与验收口径
- 作为后续实现、评审、迁移、验收的统一文档

---

## 1. 当前架构判断

当前架构不是缺乏骨架，而是已经具备强控制面基础，但仍存在几个尚未完全收口的关键点：

### 已成立的优势

- runtime / worker authority split 明确
- `.harness/` repo-local state plane 已成型
- route / dispatch / lease / verify / completion gate 分层清楚
- task-local artifact 与 global runtime summary 已有边界
- worker 不拥有 completion / archive / merge 决策权

### 尚未完全收口的缺口

1. `verify` 仍偏向结果摄入，尚未完全成为独立 evaluator
2. `task-contract` 语义存在，但尚未成为 dispatch 前后的绝对主合同
3. `accepted-packet / task-contract / worker-spec` 三者边界需要进一步硬化
4. `submit -> classify -> fuse -> bind -> selective replan` 已有文档定义，但还需完全硬化成 canonical 前置链路
5. `derived summaries` 必须持续证明可从 authoritative truth 重建，避免偷偷承载 authority

---

## 2. 改造总目标

把当前 harness 从“控制面已强、验收面偏弱、对象边界尚在收敛中”的状态，演进到：

- **packet-owned**：accepted epoch 只有一个 packet truth
- **contract-backed**：每次 dispatch 都有清晰、可验证的 task contract
- **evaluator-gated**：verify 不是 ingest，而是独立评估面
- **gate-explained**：completion gate 明确解释为什么未完成
- **summary-rebuildable**：所有 summary 都能从 authoritative truth 重建
- **thread-aware intake**：新输入进入 thread / epoch / selective replan 体系，而不是简单等于新 task

---

## 3. 设计原则

### 3.1 单一真相源原则

任何一个语义只允许一个主要 authoritative owner：

- task truth：`task-pool.json`
- accepted epoch truth：`accepted-packet-<task>.json`
- current dispatch contract truth：`task-contract.json`
- verification judgment truth：`verify.json` + `verify-scorecard`
- completion decision truth：`completion-gate.json`

### 3.2 合同优先原则

worker 不应直接执行“模糊目标”，而应执行当前 dispatch 对应的明确合同。

### 3.3 评估独立原则

verify 必须能否决“看起来像完成”的执行结果。

### 3.4 summary 可重建原则

所有 hot summary 都必须视为可重建投影，不能变成隐藏 authority。

### 3.5 渐进替换原则

先补 runtime object，再让现有 runtime 节点逐步消费新对象；
不先引入第二套调度器，不先大改 loop choreography。

### 3.6 最小增量原则

每一步改造都必须：

- 可单独实施
- 可单独测试
- 可单独回滚
- 不破坏 canonical surface

---

## 4. 目标架构效果

改造完成后，系统应具备以下结构：

```text
submit
  -> intake classification
  -> request fusion
  -> thread bind
  -> selective replan if needed
  -> route
  -> issue dispatch
  -> acquire lease
  -> prepare accepted packet + task contract + worker spec
  -> run bounded burst
  -> ingest outcome
  -> evaluator verify against contract
  -> refresh completion gate
  -> emit next action / complete / blocked / replan
  -> refresh summaries
```

运行期对象关系应稳定为：

```text
accepted-packet = accepted epoch 的战略真相
 task-contract  = 当前 dispatch / slice 的战术合同
   worker-spec  = 执行边界与操作约束
verify-scorecard = 独立评估结论
completion-gate  = runtime 最终完成裁决
```

---

## 5. 最小可执行任务清单

以下任务全部采用统一格式：

- **[task:任务名]**
- 任务描述
- 任务要求
- 任务目标
- 输入
- 输出
- 完成定义（DoD）
- 依赖
- 验证方式
- 回滚边界

这些任务被设计为**原子可执行**，每项任务只解决一个明确问题。

---

### [task:T01 固化 Accepted Packet Authority]

**任务描述**

将 `accepted-packet-<task>.json` 硬化为 accepted epoch 的唯一 packet truth，禁止 route / dispatch / verify / query 在其他对象中重新拼装 packet 真相。

**任务要求**

- 明确 accepted packet 的 schema
- 明确 `taskId + planEpoch` 唯一对应当前 accepted packet
- 明确 query / verify / gate 都从 accepted packet 读取 packet truth
- 明确 planning trace 仅为解释材料，不再承担 packet truth

**任务目标**

让 packet 从“存在的对象”升级为“被全系统依赖的真相对象”。

**输入**

- 当前 runtime task
- route 决策结果
- orchestration synthesis 输出

**输出**

- `.harness/state/accepted-packet-<task>.json`
- schema 约束
- 读取方引用切换

**完成定义（DoD）**

- accepted packet 包含完整关键字段
- query 展示 packet 摘要时只读此对象
- verify 与 completion gate 不再从 worker prompt / planning trace 反推 packet
- stale packet 不会覆盖新 epoch truth

**依赖**

- 无前置硬依赖

**验证方式**

- 同一 task bump epoch 后，旧 packet 不再被视为当前 truth
- query 可直接展示 packet 核心字段
- verify 日志能标识其消费的 packet identity

**回滚边界**

- 只影响 packet truth 读取路径
- 不改 canonical CLI surface

---

### [task:T02 固化 Task Contract Authority]

**任务描述**

将 `task-contract.json` 提升为当前 dispatch / execution slice 的唯一 done definition，明确它不是附属工件，而是本轮执行合同。

**任务要求**

- 明确 `dispatchId` 与 contract 的唯一绑定
- 明确 contract 必须包含 `inScope / outOfScope / doneCriteria / verificationChecklist / requiredEvidence`
- 明确 worker 执行依据与 verify 验收依据都来自同一 contract

**任务目标**

把“这轮到底算做完什么”从隐含语义变成 runtime 可检查对象。

**输入**

- accepted packet
- 当前 dispatch
- execution slice selection

**输出**

- `.harness/artifacts/<task>/<dispatch>/task-contract.json`

**完成定义（DoD）**

- 每个 dispatch 都有唯一 contract
- verify 明确引用 contract id
- completion gate 可逐项判断 done criteria 是否满足
- worker-spec 不再重复主合同定义

**依赖**

- T01

**验证方式**

- 多 dispatch 同 task 场景下，各自 contract 独立存在
- verify 失败时可精确指出未满足的 contract 条款

**回滚边界**

- 主要影响 worker prepare / verify / gate 消费关系

---

### [task:T03 收敛 Packet / Contract / Worker-Spec 边界]

**任务描述**

显式规定三类对象的语义边界，防止字段重复与 authority 漂移。

**任务要求**

- `accepted-packet`：战略层与 accepted epoch truth
- `task-contract`：当前 dispatch 的验收合同
- `worker-spec`：执行约束、操作边界、资源预算
- 同一字段不可在三者中重复定义不同版本

**任务目标**

消除对象重叠导致的未来复杂度失控风险。

**输入**

- 当前三类对象定义

**输出**

- 边界表
- schema ownership 规则
- 重复字段清理方案

**完成定义（DoD）**

- 三者 owner 清单清楚
- query / verify / worker 各自只读自己应读的对象
- 不存在需要“综合三份文档才知道 done definition”的情况

**依赖**

- T01
- T02

**验证方式**

- 抽样任意 dispatch，可一眼识别战略、战术、执行边界分别来自哪个对象

**回滚边界**

- 文档和 schema 层改动为主，逻辑切换可渐进

---

### [task:T04 建立 Verify Scorecard Schema]

**任务描述**

把 verify 从单一 passed/failed 结果，升级为多维度 scorecard。

**任务要求**

固定至少以下维度：

- `scopeCompletion`
- `behaviorCorrectness`
- `packetAlignment`
- `evidenceQuality`
- `reviewReadiness`

每个维度必须包含：

- `score`
- `threshold`
- `status`
- `summary`

**任务目标**

让 verify 可以表达“哪里不够、差多少、下一步补什么”，而不是单句总结。

**输入**

- accepted packet
- task contract
- worker-result
- verify evidence

**输出**

- 增强版 `verify.json`
- 或独立 `verify-scorecard` 结构

**完成定义（DoD）**

- verify 产物能表达多维结论
- completion gate 可读取 scorecard
- query 可展示 scorecard 摘要

**依赖**

- T01
- T02

**验证方式**

- 同一个任务可以出现“功能完成但 review 未达标”的独立失败结论
- verify 不再只输出单一总体状态

**回滚边界**

- scorecard 字段可增量加入，保留旧 status 兼容期

---

### [task:T05 建立 Evidence Ledger]

**任务描述**

为 verify 引入标准化 evidence ledger，明确“验证证据”不是自由文本，而是结构化引用集合。

**任务要求**

证据至少支持以下类别：

- command result
- file diff / changed paths
- test output
- artifact existence
- review evidence
- runtime checkpoint reference

**任务目标**

让 verify 结论可被追溯、可审计、可解释。

**输入**

- worker-result
- verify 阶段收集的证据

**输出**

- `evidenceLedger`
- 标准引用字段

**完成定义（DoD）**

- verify findings 均可追溯到 evidence ledger
- completion gate 可判断证据是否完整
- reviewRequired 场景可检查 review evidence 是否存在

**依赖**

- T04

**验证方式**

- 随机抽一个 verify finding，可定位到具体 evidence 引用

**回滚边界**

- 主要是 verify artifact 增强

---

### [task:T06 让 Verify 成为 Independent Evaluator]

**任务描述**

在不新增新 binary 的前提下，先把 `internal/verify` 的职责升级为独立 evaluator，而不仅是 ingest 模块。

**任务要求**

verify 必须做到：

- 对照 accepted packet
- 对照 task contract
- 对照 worker-result claims
- 对照 evidence ledger
- 输出 findings / scorecard / recommendedNextAction

**任务目标**

使 verify 能对“看起来完成”的执行结果做真正否决。

**输入**

- T01/T02/T04/T05 产物

**输出**

- evaluator 风格 verify 结果
- `recommendedNextAction`

**完成定义（DoD）**

- worker 成功不等于 verify 成功
- verify 成功不等于 completion gate 完成
- verify fail 时能生成可执行的 repair / replan 建议

**依赖**

- T01
- T02
- T04
- T05

**验证方式**

- 构造一个“代码修改存在但 evidence 不足”的任务，verify 应判失败
- 构造一个“功能达标但 review 未达标”的任务，verify 应输出 needs_review 倾向

**回滚边界**

- 只增强 verify 权限，不新增调度环

---

### [task:T07 让 Completion Gate 消费 Contract 与 Scorecard]

**任务描述**

将 completion gate 从主要消费 verify 总结果，升级为消费 packet / contract / scorecard / evidence bundle 的组合裁决面。

**任务要求**

gate 至少判断：

- accepted packet 是否有效
- 当前 dispatch 是否拥有有效 contract
- done criteria 是否全部满足
- evidence ledger 是否完整
- high priority findings 是否未清除
- reviewRequired 时 review evidence 是否满足阈值

**任务目标**

让 completion 决策成为可解释、可审计、可复现的 runtime 判定。

**输入**

- accepted packet
- task contract
- verify scorecard
- evidence ledger

**输出**

- 强化后的 `completion-gate.json`

**完成定义（DoD）**

- gate 输出原因清单而不是单个 bool
- gate 能区分 `completed / incomplete / blocked / needs_replan / needs_review`
- archive 仍严格依赖 gate，而不是仅依赖 verify status

**依赖**

- T02
- T04
- T05
- T06

**验证方式**

- 同一个 verify status 下，gate 能根据 evidence 完整度给出不同结论

**回滚边界**

- 主要影响 verify/gate 层，外部 CLI 保持不变

---

### [task:T08 统一 Failure Reason Taxonomy]

**任务描述**

统一 verify 与 completion gate 的 failure reason taxonomy，避免 operator 看到模糊失败。

**任务要求**

至少覆盖：

- `missing_contract`
- `missing_packet`
- `missing_evidence`
- `failed_done_criteria`
- `review_pending`
- `high_risk_finding_open`
- `stale_dispatch`
- `stale_lease`
- `needs_replan_due_to_scope_change`

**任务目标**

把“为什么没完成”从人脑推理改成结构化 runtime 输出。

**输入**

- gate / verify 当前失败场景

**输出**

- taxonomy
- reason code 使用规范

**完成定义（DoD）**

- query 能清晰展示 failure reasons
- operator 不需要读多个文件才能知道阻塞原因

**依赖**

- T06
- T07

**验证方式**

- 抽样 blocked / incomplete / replan 场景，可被 reason code 唯一解释

**回滚边界**

- reason code 可渐进扩展

---

### [task:T09 升级 Submit 为 Single-Entry Intake]

**任务描述**

把 `submit` 从“默认创建新 task”升级为 thread-aware 的 single-entry intake。

**任务要求**

`submit` 必须支持：

- append-only request
- intake classification
- request fusion
- thread correlation
- inflight impact analysis
- selective replan

**任务目标**

让新增输入进入 thread/epoch 系统，而不是永远产生新 task。

**输入**

- 新 submission
- 当前 thread state
- 当前 task pool

**输出**

- 更新后的 request / task / thread / change / todo summaries

**完成定义（DoD）**

- 新 context 不一定 bump epoch
- 只有实际影响 execution scope / acceptance 才 bump epoch
- queued 老 epoch 任务不会继续 dispatch

**依赖**

- 无硬前置，但建议在 contract/evaluator 闭环后实施

**验证方式**

- 同 thread 的 context enrichment 不会错误地产生全新独立 task
- 影响执行范围的补充输入会触发 selective replan

**回滚边界**

- 重点影响 runtime.Submit 与 summary refresh

---

### [task:T10 硬化 Thread / Epoch / Selective Replan]

**任务描述**

把 thread key、plan epoch、impact class、selective replan 的规则从文档定义变成硬 runtime 规则。

**任务要求**

支持 impact class：

- `continue_safe`
- `continue_with_note`
- `checkpoint_then_replan`
- `supersede_queued`
- `inspection_only_overlay`

**任务目标**

避免长任务中“上下文到了，但执行层没同步”的漂移。

**输入**

- 新 request
- 当前 inflight/queued task 状态

**输出**

- 更新后的 `thread-state.json`
- 变更后的 `todo-summary.json`
- 需要时更新的 `planEpoch`

**完成定义（DoD）**

- active task 是否继续 / checkpoint / replan 有明确规则
- queued task 在旧 epoch 下不会误发 dispatch
- inspection overlay 不会无故阻塞无关工作

**依赖**

- T09

**验证方式**

- 构造补充需求、澄清需求、完全改目标三类输入，观察 epoch 和 todo 是否正确变化

**回滚边界**

- 不影响已存在的 dispatch / lease 机制

---

### [task:T11 强化 Query 为 Operator Truth Surface]

**任务描述**

强化 `harness task` / `harness tasks` 视图，使 operator 看到的不是散文件拼装结果，而是聚合后的 runtime truth surface。

**任务要求**

视图至少展示：

- accepted packet 摘要
- current task contract 摘要
- verify scorecard 摘要
- completion gate 原因
- recommended next action
- active lease / latest dispatch / current epoch

**任务目标**

让 operator 在不读 planning trace 的情况下掌握当前状态。

**输入**

- packet / contract / verify / gate / dispatch / lease / thread summaries

**输出**

- query 读模型增强

**完成定义（DoD）**

- operator 看 task 详情时能直接知道：现在做什么、为什么没完成、下一步是什么
- 失败/阻塞原因无需二次 grep

**依赖**

- T01
- T02
- T04
- T07
- T08

**验证方式**

- 随机抽一个任务，通过单次 task view 能读懂当前状态

**回滚边界**

- 只改 query surface

---

### [task:T12 建立 Summary Rebuildability 审计]

**任务描述**

逐个确认所有 summary 是否真可从 authoritative 层重建，并消除隐藏 authority。

**任务要求**

为每个 summary 标记：

- authoritative 来源
- generator
- rebuild 方法
- 是否允许 operator 直接依赖
- 是否存在隐藏 authority 风险

**任务目标**

防止 derived summary 逐步腐化成真正真相源。

**输入**

- 当前 `.harness/state/*.json` 列表

**输出**

- summary inventory
- rebuildability matrix
- 隐藏 authority 清理清单

**完成定义（DoD）**

- 每个 summary 都能回答“谁生成”“从哪来”“坏了怎么重建”
- 高风险 summary 被降权或重构

**依赖**

- 无硬前置，建议与 query 增强同步推进

**验证方式**

- 人工删除任意 derived summary 后，runtime 可从 authoritative 层重建

**回滚边界**

- 以审计与整理为主，低风险

---

### [task:T13 建立 Phase-1 真实目标仓验证回路]

**任务描述**

将 body repo 与 target repo 的 phase-1 验证机制固化为常规回归路径。

**任务要求**

- body repo 只修 harness 本体
- target repo 通过安装后的 harness 执行真实需求
- 每次失败必须抽象为 harness capability gap
- 不允许 operator 直接手修 target business code 来伪造成功

**任务目标**

让验证不只停留在 body repo 自测，而是对真实 target requirement 闭环负责。

**输入**

- body repo 变更
- target repo 真实需求

**输出**

- target control-plane evidence
- success/failure lineage
- harness gap 抽象记录

**完成定义（DoD）**

- 至少一条真实 requirement 能由 target repo 中的 harness 流程完成
- 失败时能清楚区分 prompt gap / system gap / target genuine work

**依赖**

- T06
- T07
- T09
- T11

**验证方式**

- 重跑同一真实需求，不会反复触发同一 control-plane bug

**回滚边界**

- 不改 body runtime 主链路，只增强验收回归方式

---

## 6. 扩展原子任务池（Atomic Task Pool）

上面的 T01 ~ T13 是**能力层任务**，用于定义改造范围。

为了便于真正排期、逐步编码、逐步验证，下面把它们进一步细化为一组**原子可执行任务**。

设计要求：

- 每个任务尽量只改一个明确点
- 每个任务都应能单独落地、单独测试、单独回滚
- 每个任务都必须能回答：改什么、为什么改、完成后如何判断完成
- 原子任务完成后，必须能挂回上层能力任务 T01 ~ T13

说明：

- `Txx`：能力层任务（epic / capability）
- `Axx`：原子执行任务（atomic implementation unit）

---

### 6.1 L1 真相对象收口：Atomic Tasks

#### [task:A01 定义 Accepted Packet Identity]

**任务描述**

定义 accepted packet 的身份模型，明确它的主键、版本边界和 accepted epoch 绑定关系。

**任务要求**

- 明确 `taskId + planEpoch` 是当前 packet truth 的主识别面
- 明确 packet 是否需要 `packetId`
- 明确 accepted / superseded / stale 的状态判定规则

**任务目标**

让 accepted packet 的身份关系先于 schema 落地，避免后续读写方各自发明 identity 语义。

**完成定义（DoD）**

- 身份字段与状态字段被明确写入文档与实现约束
- 任意读取方都能确定“当前 packet 是哪一个”

**前置**

- 无

---

#### [task:A02 冻结 Accepted Packet Schema]

**任务描述**

确定 accepted packet 的最小必要字段与可选字段。

**任务要求**

至少覆盖：

- `taskId`
- `threadKey`
- `planEpoch`
- `packetId`
- `objective`
- `constraints`
- `selectedPlan`
- `executionTasks`
- `verificationPlan`
- `acceptanceMarkers`
- `ownedPaths`
- `replanTriggers`
- `acceptedAt`
- `acceptedBy`

**任务目标**

建立稳定 packet schema，使 route / query / verify / gate 可依赖统一结构。

**完成定义（DoD）**

- schema 最小集冻结
- 可选字段和强制字段边界明确

**前置**

- A01

---

#### [task:A03 写入 Accepted Packet Artifact]

**任务描述**

在 worker prepare / orchestration 阶段写出 accepted packet 实体文件。

**任务要求**

- 输出路径固定
- 写入行为幂等
- 对同一 `taskId + planEpoch` 可稳定覆盖当前 truth

**任务目标**

让 accepted packet 从设计对象变成真实落盘对象。

**完成定义（DoD）**

- `.harness/state/accepted-packet-<task>.json` 稳定生成
- 没有 accepted packet 时，后续模块能明确感知缺失

**前置**

- A01
- A02

---

#### [task:A04 增加 Accepted Packet Stale Protection]

**任务描述**

阻止旧 epoch packet 或旧 dispatch 对新 packet truth 的覆盖。

**任务要求**

- 比较 plan epoch
- 必要时比较 accepted timestamp / revision
- 对 stale 写入给出显式拒绝或丢弃语义

**任务目标**

防止 packet truth 在 resume、replan、多轮执行中回退。

**完成定义（DoD）**

- 旧 packet 无法覆盖新 epoch truth
- 发生 stale 写入时有可观察 reason

**前置**

- A03

---

#### [task:A05 Query 迁移到 Accepted Packet 读取]

**任务描述**

把 query 视图中的 packet 摘要来源切换为 accepted packet。

**任务要求**

- 不再从 planning trace 或 worker prompt 反推 packet
- query 缺失 packet 时给出明确状态

**任务目标**

让 operator 视角首先收敛到 packet truth。

**完成定义（DoD）**

- `harness task` 可直接展示 accepted packet 摘要
- 摘要来源唯一

**前置**

- A03

---

#### [task:A06 Verify 与 Gate 迁移到 Accepted Packet 读取]

**任务描述**

让 verify 与 completion gate 都把 accepted packet 当作 packet truth 来源。

**任务要求**

- verify 不再自行拼 packet 语义
- gate 不再依赖 prompt / trace 间接推理 packet

**任务目标**

把 packet authority 全面收口。

**完成定义（DoD）**

- verify / gate 的 packet 来源唯一
- packet 缺失会成为结构化失败原因

**前置**

- A03
- A04

---

#### [task:A07 定义 Task Contract Identity]

**任务描述**

定义 task contract 的身份模型，明确它和 dispatch / execution slice 的绑定关系。

**任务要求**

- 明确 `dispatchId` 是 contract 主绑定面
- 必要时定义 `contractId`
- 明确一个 dispatch 只能有一个当前有效 contract

**任务目标**

避免 contract 变成可有可无的附属文件。

**完成定义（DoD）**

- 任意 dispatch 都能定位唯一 contract
- contract 的 superseded 规则清楚

**前置**

- 无

---

#### [task:A08 冻结 Task Contract Schema]

**任务描述**

确定 task contract 的最小 schema。

**任务要求**

至少覆盖：

- `contractId`
- `taskId`
- `dispatchId`
- `planEpoch`
- `executionSliceId`
- `objective`
- `inScope`
- `outOfScope`
- `doneCriteria`
- `verificationChecklist`
- `requiredEvidence`
- `reviewRequired`
- `acceptedAt`

**任务目标**

让 contract 具备明确的执行与验收边界。

**完成定义（DoD）**

- contract 字段最小集冻结
- worker / verify / gate 能共享理解

**前置**

- A07

---

#### [task:A09 在 Worker Prepare 输出 Task Contract]

**任务描述**

让 worker prepare 阶段输出 `task-contract.json`。

**任务要求**

- 输出路径稳定
- contract 与 dispatch 一起生成
- 当前 execution slice 写入 contract

**任务目标**

让 contract 成为 dispatch 启动前即可使用的对象。

**完成定义（DoD）**

- 每个 dispatch 都有唯一 contract 文件
- 缺失 contract 会被后续模块识别为异常

**前置**

- A08

---

#### [task:A10 明确 Contract 与 Execution Slice 绑定规则]

**任务描述**

明确当前 dispatch 对应哪个 execution slice，以及 contract 如何表达这一点。

**任务要求**

- 支持 slice identity
- 支持 slice 完成状态写回
- 不允许多个 slice 混入同一 contract

**任务目标**

让 contract 真正对应“本轮要完成什么”，而不是泛泛而谈。

**完成定义（DoD）**

- verify 可以基于 contract 精确判断本轮是否完成

**前置**

- A09

---

#### [task:A11 Verify 迁移到 Contract 读取]

**任务描述**

让 verify 的验收依据明确来自 task contract。

**任务要求**

- verify 对照 `doneCriteria`
- verify 对照 `verificationChecklist`
- verify 对照 `requiredEvidence`

**任务目标**

把“本轮 done definition”收口到 contract。

**完成定义（DoD）**

- verify failure 能指向具体 contract 条款

**前置**

- A09
- A10

---

#### [task:A12 Completion Gate 迁移到 Contract 读取]

**任务描述**

让 completion gate 明确消费 task contract，而不是只消费 verify 总结果。

**任务要求**

- gate 能逐项判断 contract 是否满足
- gate 缺失 contract 时输出 `missing_contract`

**任务目标**

让完成判定直接对齐 dispatch contract。

**完成定义（DoD）**

- gate 能解释哪个 contract 条件未满足

**前置**

- A09
- A11

---

#### [task:A13 输出 Packet / Contract / Worker-Spec 边界矩阵]

**任务描述**

整理三类对象的职责边界矩阵。

**任务要求**

- packet 管战略真相
- contract 管当前轮次合同
- worker-spec 管执行约束与预算

**任务目标**

避免对象边界继续模糊化。

**完成定义（DoD）**

- 文档中存在可直接引用的边界矩阵

**前置**

- A02
- A08

---

#### [task:A14 清理三类对象中的重复字段]

**任务描述**

识别并清理 packet / contract / worker-spec 中重复定义的字段。

**任务要求**

- 每个字段有唯一 owner
- 重复字段要删除、降级或改为引用

**任务目标**

降低长期复杂度和语义冲突风险。

**完成定义（DoD）**

- 三类对象不存在相互冲突的同名字段定义

**前置**

- A13

---

#### [task:A15 建立对象 Ownership 校验用例]

**任务描述**

为对象边界建立测试或校验规则。

**任务要求**

- 至少覆盖 packet/contract/spec 三类对象
- 校验核心字段是否落在正确 owner 上

**任务目标**

防止未来演进重新引入重复 authority。

**完成定义（DoD）**

- 新增或变更字段时可被校验

**前置**

- A14

---

### 6.2 L2 独立验收收口：Atomic Tasks

#### [task:A16 定义 Verify Scorecard Schema]

**任务描述**

定义 verify scorecard 的结构与维度。

**任务要求**

至少包含：

- `scopeCompletion`
- `behaviorCorrectness`
- `packetAlignment`
- `evidenceQuality`
- `reviewReadiness`

**任务目标**

让 verify 能表达多维结论，而不是单句通过/失败。

**完成定义（DoD）**

- 每个维度都有 `score / threshold / status / summary`

**前置**

- A06
- A11

---

#### [task:A17 定义 Evidence Ledger Schema]

**任务描述**

定义 evidence ledger 结构，规范证据的引用方式。

**任务要求**

支持：

- command result
- file diff / changed paths
- test output
- artifact existence
- checkpoint reference
- review evidence

**任务目标**

让 verify 结论可以溯源。

**完成定义（DoD）**

- 证据结构统一，支持引用与分类

**前置**

- A16

---

#### [task:A18 实现 Command / Test / File Evidence Collector]

**任务描述**

实现对命令输出、测试结果、文件变化等证据的结构化采集。

**任务要求**

- 明确 evidence type
- 引用路径或摘要而非堆大文本
- 保持 task-local

**任务目标**

把常见验证证据纳入统一 ledger。

**完成定义（DoD）**

- 至少三类常见证据可被稳定收集

**前置**

- A17

---

#### [task:A19 实现 Review Evidence Collector]

**任务描述**

对 reviewRequired 场景收集 review evidence。

**任务要求**

- 支持 review artifact 存在性校验
- 支持 review 结论阈值表达

**任务目标**

让 review 不再只是人工口头前提，而成为 gate 可消费证据。

**完成定义（DoD）**

- 无 review evidence 时可稳定阻止 completed

**前置**

- A17

---

#### [task:A20 定义 Findings Schema]

**任务描述**

定义 verify findings 的结构。

**任务要求**

至少覆盖：

- finding id
- severity
- category
- related criteria
- related evidence
- remediation hint

**任务目标**

让 verify 输出从“摘要”升级为“问题清单”。

**完成定义（DoD）**

- findings 支持和 contract/evidence 建立引用关系

**前置**

- A16
- A17

---

#### [task:A21 定义 RecommendedNextAction Schema]

**任务描述**

定义 verify / gate 推荐下一步动作的结构。

**任务要求**

至少支持：

- `continue`
- `repair`
- `needs_review`
- `needs_replan`
- `block`

**任务目标**

把“下一步该干什么”收敛为结构化输出，而不是靠人读总结猜。

**完成定义（DoD）**

- next action 可被 query 直接展示

**前置**

- A20

---

#### [task:A22 Verify 消费 Packet + Contract]

**任务描述**

让 verify 同时以 accepted packet 和 task contract 为输入对象。

**任务要求**

- packet 用于看全局对齐
- contract 用于看当前 dispatch 达成情况

**任务目标**

建立 evaluator 的双输入面。

**完成定义（DoD）**

- verify 的输入对象明确且固定

**前置**

- A06
- A11

---

#### [task:A23 Verify 产出 Scorecard + Evidence + Findings]

**任务描述**

让 verify 真正输出 scorecard、evidence ledger、findings、recommended next action。

**任务要求**

- 产物结构稳定
- 产物可被 gate/query 消费

**任务目标**

把 verify 从 ingest 升级为 evaluator 产物生成器。

**完成定义（DoD）**

- verify 结果至少包含四类结构化输出

**前置**

- A18
- A19
- A20
- A21
- A22

---

#### [task:A24 Completion Gate 消费 Contract Satisfaction]

**任务描述**

让 gate 逐项检查 contract satisfaction。

**任务要求**

- 对 `doneCriteria` 做明确判断
- 对缺失项输出结构化原因

**任务目标**

让 gate 的完成判断对齐 contract。

**完成定义（DoD）**

- gate 可列出未满足的 contract 条目

**前置**

- A12
- A23

---

#### [task:A25 Completion Gate 消费 Evidence Completeness]

**任务描述**

让 gate 把 evidence completeness 纳入判定。

**任务要求**

- contract requiredEvidence 必须有对应证据
- reviewRequired 时 review evidence 必须被检查

**任务目标**

防止“看起来完成，但没有证据”通过 gate。

**完成定义（DoD）**

- 缺失 evidence 时 gate 不会放行 completed

**前置**

- A17
- A19
- A24

---

#### [task:A26 定义 Gate Outcome Taxonomy]

**任务描述**

统一 gate 结果分类。

**任务要求**

至少支持：

- `completed`
- `incomplete`
- `blocked`
- `needs_replan`
- `needs_review`

**任务目标**

让 gate 结果具备稳定语义。

**完成定义（DoD）**

- gate 输出不再只是 bool 或 passed/failed

**前置**

- A24
- A25

---

#### [task:A27 统一 Failure Reason Codes]

**任务描述**

统一 verify/gate 的 reason codes。

**任务要求**

至少包含：

- `missing_packet`
- `missing_contract`
- `missing_evidence`
- `failed_done_criteria`
- `review_pending`
- `high_risk_finding_open`
- `stale_dispatch`
- `stale_lease`
- `needs_replan_due_to_scope_change`

**任务目标**

让阻塞原因具备标准化输出。

**完成定义（DoD）**

- query 能直接展示统一 reason code

**前置**

- A26

---

#### [task:A28 在 Query 暴露 Verify / Gate 结构化原因]

**任务描述**

把 verify findings、gate reasons、recommended next action 接入 query。

**任务要求**

- operator 一次读取可看到主要结论
- 不要求人工 grep 多个工件

**任务目标**

提升 operator surface 的解释力。

**完成定义（DoD）**

- `harness task` 能直接展示主要失败原因与下一步动作

**前置**

- A23
- A27

---

#### [task:A29 建立 Evaluator / Gate 回归测试]

**任务描述**

围绕 verify/gate 建立回归测试样例。

**任务要求**

至少覆盖：

- worker success 但 evidence 不足
- review pending
- stale dispatch / stale lease
- contract 条款未完成

**任务目标**

保证 evaluator/gate 收口后可长期稳定演进。

**完成定义（DoD）**

- 关键验收分支有自动化测试覆盖

**前置**

- A23
- A24
- A25
- A26
- A27

---

### 6.3 L3 输入与线程收口：Atomic Tasks

#### [task:A30 定义 Intake Classification Rules]

**任务描述**

为新 submission 定义 intake 分类规则。

**任务要求**

至少支持：

- 新任务
- 上下文补充
- 目标变更
- 检查类 overlay
- 需要 replan 的变更

**任务目标**

让 submit 不再简单等于新 task。

**完成定义（DoD）**

- 任意输入都能被归到明确 intake class

**前置**

- 无

---

#### [task:A31 定义 Request Fusion Rules]

**任务描述**

定义 request fusion 规则，明确哪些输入应合入现有 thread。

**任务要求**

- 支持 append-only request record
- 支持 merged context 引用
- 支持 selective merge，而不是全文混入

**任务目标**

避免新输入被粗暴拆成新 task 或完全吞进历史。

**完成定义（DoD）**

- 同 thread 的补充信息可被稳定融合

**前置**

- A30

---

#### [task:A32 定义 Thread Correlation Rules]

**任务描述**

定义 request 与 thread 的关联规则。

**任务要求**

- threadKey 生成/匹配逻辑明确
- 支持已有 thread 复用
- 支持无法关联时创建新 thread

**任务目标**

让 thread 成为真正的工作流载体。

**完成定义（DoD）**

- 新输入可以明确说明自己属于哪个 thread

**前置**

- A31

---

#### [task:A33 定义 Impact Analysis Classes]

**任务描述**

定义 inflight impact analysis 类别。

**任务要求**

至少支持：

- `continue_safe`
- `continue_with_note`
- `checkpoint_then_replan`
- `supersede_queued`
- `inspection_only_overlay`

**任务目标**

让新输入对现有执行的影响可结构化表达。

**完成定义（DoD）**

- 每个新输入都能产生 impact class

**前置**

- A32

---

#### [task:A34 建立 Selective Replan Trigger Engine]

**任务描述**

建立 selective replan 触发规则。

**任务要求**

- 仅在影响 execution scope / acceptance 时 bump epoch
- 普通 context enrichment 不自动 bump epoch

**任务目标**

减少不必要 replan，同时避免 scope drift。

**完成定义（DoD）**

- epoch bump 有清晰触发条件

**前置**

- A33

---

#### [task:A35 Submit 写入 Intake / Thread / Change / Todo Summaries]

**任务描述**

让 submit 阶段产出四类关键 summary。

**任务要求**

- `intake-summary.json`
- `thread-state.json`
- `change-summary.json`
- `todo-summary.json`

**任务目标**

把文档里的 intake/fusion/bind 语义落成 runtime state。

**完成定义（DoD）**

- submit 后 operator 能看到输入影响摘要

**前置**

- A30
- A31
- A32
- A33
- A34

---

#### [task:A36 Route 强制检查 Epoch Freshness]

**任务描述**

让 route 以当前 epoch freshness 为硬前置。

**任务要求**

- 旧 epoch queued task 不得 dispatch
- 需要 replan 的任务要被 route 阻止

**任务目标**

防止旧计划继续推进执行。

**完成定义（DoD）**

- route decision 明确受 epoch freshness 影响

**前置**

- A34
- A35

---

#### [task:A37 抑制旧 Epoch Queued Task 的继续派发]

**任务描述**

确保 queued task 在旧 epoch 下不会被继续派发。

**任务要求**

- 支持 superseded 状态或等价阻断语义
- query 可解释为何未派发

**任务目标**

防止旧任务偷偷穿过 route。

**完成定义（DoD）**

- 被 supersede 的 queued task 不会进入 dispatch

**前置**

- A36

---

#### [task:A38 实现 Active Task 的 Checkpoint-Then-Replan 路径]

**任务描述**

实现 active task 受新输入影响时的 checkpoint-then-replan 分支。

**任务要求**

- 必须先 checkpoint 再切 replan
- 不允许正在执行的状态被直接抹掉

**任务目标**

让 inflight task 在 scope 变化时优雅收束，而不是粗暴中断。

**完成定义（DoD）**

- active task 在受影响时有明确且可恢复的迁移路径

**前置**

- A33
- A34
- A35

---

### 6.4 L4 可读性与可维护性收口：Atomic Tasks

#### [task:A39 强化 Query Task Detail Surface]

**任务描述**

增强 task 详情视图，聚合 packet/contract/verify/gate 关键信息。

**任务要求**

至少展示：

- accepted packet 摘要
- current contract 摘要
- scorecard 摘要
- gate 结果
- next action

**任务目标**

让单次 task 查询即可读懂当前状态。

**完成定义（DoD）**

- operator 不必读取多个散文件

**前置**

- A05
- A28

---

#### [task:A40 强化 Query Task List Surface]

**任务描述**

增强任务列表视图，使 tasks 列表能显示关键执行状态。

**任务要求**

至少展示：

- current epoch
- latest dispatch
- gate 状态
- next action 摘要

**任务目标**

让 operator 在列表层就能判断哪些任务需要关注。

**完成定义（DoD）**

- `harness tasks` 具备更强筛查能力

**前置**

- A39

---

#### [task:A41 暴露 Operator Next Action 与 Reason Surface]

**任务描述**

把 recommended next action 与核心 reason 以 operator-friendly 方式展示。

**任务要求**

- 面向 operator 可读
- 保留 machine-readable 值

**任务目标**

把“现在应该做什么”直接呈现出来。

**完成定义（DoD）**

- task/task list 均可看到 next action 摘要

**前置**

- A21
- A28
- A39

---

#### [task:A42 建立 Summary Inventory]

**任务描述**

列举所有 runtime state 与 summary 文件，并标注属性。

**任务要求**

至少标注：

- authoritative / derived
- generator
- consumer
- rebuildability

**任务目标**

建立 summary 治理基线。

**完成定义（DoD）**

- 存在完整 inventory 文档或结构化清单

**前置**

- 无

---

#### [task:A43 建立 Rebuildability Matrix]

**任务描述**

为所有 derived summary 建立可重建矩阵。

**任务要求**

- 明确 authoritative 来源
- 明确重建方法
- 明确高风险 hidden authority 点

**任务目标**

防止 summary 腐化成真相源。

**完成定义（DoD）**

- 每个 summary 都能回答“从哪来、怎么重建”

**前置**

- A42

---

#### [task:A44 建立 Summary Rebuild Procedure / Tooling]

**任务描述**

建立 summary 重建流程或工具化入口。

**任务要求**

- 支持人工触发重建
- 支持 runtime 自恢复或运维手册

**任务目标**

让 derived summary 真的可以重建，而不只是理论可重建。

**完成定义（DoD）**

- 至少有一条明确重建路径可演示

**前置**

- A43

---

#### [task:A45 建立 Summary Degraded Recovery Test]

**任务描述**

验证 summary 丢失或损坏时，系统可以从 authoritative 层恢复。

**任务要求**

- 至少覆盖 1~2 个关键 derived summary
- 恢复后 query 结果可用

**任务目标**

把“可重建”从口号变成真实能力。

**完成定义（DoD）**

- 人工删除或破坏 summary 后可恢复

**前置**

- A44

---

### 6.5 L5 真实世界验证收口：Atomic Tasks

#### [task:A46 选择 Phase-1 Target Requirement 集合]

**任务描述**

选定 1~3 条真实 target requirement 作为外部验证样本。

**任务要求**

- 至少覆盖一个普通 feature
- 至少覆盖一个失败后重规划或阻塞样本

**任务目标**

让架构验证不只发生在 body repo 自测里。

**完成定义（DoD）**

- 已选定 target repo 与 requirement 集合

**前置**

- 无

---

#### [task:A47 标准化 Phase-1 Validation Protocol]

**任务描述**

把 body repo -> reinstall -> target repo 验证的 protocol 固化下来。

**任务要求**

- 不允许直接手修 target business code 伪造成功
- 每次失败都要抽象为 harness gap

**任务目标**

让 phase-1 验证成为常规闭环。

**完成定义（DoD）**

- 验证 protocol 可重复执行

**前置**

- A46

---

#### [task:A48 建立 Target Evidence Capture Template]

**任务描述**

建立 target repo 控制面证据采集模板。

**任务要求**

至少包含：

- request/task state
- packet/contract/gate evidence
- failure reasons
- success proof

**任务目标**

让 target 验证结果可归档、可比较、可复盘。

**完成定义（DoD）**

- 任一 target round 都能用统一模板采集证据

**前置**

- A47

---

#### [task:A49 跑通一个 Target Success Scenario]

**任务描述**

在 target repo 中跑通至少一个真实需求的完整闭环成功样本。

**任务要求**

- 不人工手修 target 业务代码
- 通过 harness 流程完成需求

**任务目标**

证明改造后的 harness 具备真实世界成功能力。

**完成定义（DoD）**

- 存在一条真实 requirement 的成功证据

**前置**

- A23
- A25
- A35
- A39
- A48

---

#### [task:A50 跑通一个 Failure-to-Gap Abstraction Scenario]

**任务描述**

在 target repo 中验证一个失败样本，并将失败抽象为可复用 harness gap。

**任务要求**

- 区分 prompt gap / system gap / genuine target work
- 不允许模糊归因

**任务目标**

证明失败也能被 runtime 清晰解释，而不是落回人工 rescue。

**完成定义（DoD）**

- 存在一条失败 -> gap 抽象 -> 修补方向的闭环记录

**前置**

- A48

---

#### [task:A51 建立 Repeatability Regression]

**任务描述**

对同一 target requirement 执行重复验证，检查已修复问题不重复出现。

**任务要求**

- 至少重跑一次成功样本
- 至少验证一个已修复 gap 不复发

**任务目标**

证明系统不是偶然跑通，而是具备稳定性。

**完成定义（DoD）**

- 同一 requirement 重跑不复现相同 control-plane bug

**前置**

- A49
- A50

---

#### [task:A52 完成 Rollout Review 与发布前评估]

**任务描述**

在完成阶段性改造后，进行 rollout review，决定是否进入下一阶段演进。

**任务要求**

- 评估 evaluator 收益
- 评估 intake 收益
- 评估复杂度是否可控
- 判断是否需要 future evaluator node

**任务目标**

让架构演进在收益和复杂度之间保持优雅平衡。

**完成定义（DoD）**

- 有明确 review 结论与后续建议

**前置**

- A49
- A50
- A51

---

## 7. 谱系方案（Expanded Lineage Plan）

```text
L0 目标：把 Klein-Harness 收口为 contract-backed / evaluator-gated / thread-aware runtime

├─ L1 真相对象收口
│  ├─ T01 固化 Accepted Packet Authority
│  │  ├─ A01 定义 Accepted Packet Identity
│  │  ├─ A02 冻结 Accepted Packet Schema
│  │  ├─ A03 写入 Accepted Packet Artifact
│  │  ├─ A04 增加 Accepted Packet Stale Protection
│  │  ├─ A05 Query 迁移到 Accepted Packet 读取
│  │  └─ A06 Verify 与 Gate 迁移到 Accepted Packet 读取
│  ├─ T02 固化 Task Contract Authority
│  │  ├─ A07 定义 Task Contract Identity
│  │  ├─ A08 冻结 Task Contract Schema
│  │  ├─ A09 在 Worker Prepare 输出 Task Contract
│  │  ├─ A10 明确 Contract 与 Execution Slice 绑定规则
│  │  ├─ A11 Verify 迁移到 Contract 读取
│  │  └─ A12 Completion Gate 迁移到 Contract 读取
│  └─ T03 收敛 Packet / Contract / Worker-Spec 边界
│     ├─ A13 输出边界矩阵
│     ├─ A14 清理重复字段
│     └─ A15 建立对象 Ownership 校验用例
│
├─ L2 独立验收收口
│  ├─ T04 建立 Verify Scorecard Schema
│  │  └─ A16 定义 Verify Scorecard Schema
│  ├─ T05 建立 Evidence Ledger
│  │  ├─ A17 定义 Evidence Ledger Schema
│  │  ├─ A18 实现 Command/Test/File Evidence Collector
│  │  └─ A19 实现 Review Evidence Collector
│  ├─ T06 让 Verify 成为 Independent Evaluator
│  │  ├─ A20 定义 Findings Schema
│  │  ├─ A21 定义 RecommendedNextAction Schema
│  │  ├─ A22 Verify 消费 Packet + Contract
│  │  └─ A23 Verify 产出 Scorecard + Evidence + Findings
│  ├─ T07 让 Completion Gate 消费 Contract 与 Scorecard
│  │  ├─ A24 Gate 消费 Contract Satisfaction
│  │  ├─ A25 Gate 消费 Evidence Completeness
│  │  └─ A26 定义 Gate Outcome Taxonomy
│  └─ T08 统一 Failure Reason Taxonomy
│     ├─ A27 统一 Failure Reason Codes
│     ├─ A28 Query 暴露 Verify / Gate 原因
│     └─ A29 建立 Evaluator / Gate 回归测试
│
├─ L3 输入与线程收口
│  ├─ T09 升级 Submit 为 Single-Entry Intake
│  │  ├─ A30 定义 Intake Classification Rules
│  │  ├─ A31 定义 Request Fusion Rules
│  │  ├─ A32 定义 Thread Correlation Rules
│  │  ├─ A33 定义 Impact Analysis Classes
│  │  ├─ A34 建立 Selective Replan Trigger Engine
│  │  └─ A35 Submit 写入 Intake / Thread / Change / Todo Summaries
│  └─ T10 硬化 Thread / Epoch / Selective Replan
│     ├─ A36 Route 强制检查 Epoch Freshness
│     ├─ A37 抑制旧 Epoch Queued Task 的继续派发
│     └─ A38 实现 Active Task 的 Checkpoint-Then-Replan 路径
│
├─ L4 可读性与可维护性收口
│  ├─ T11 强化 Query 为 Operator Truth Surface
│  │  ├─ A39 强化 Query Task Detail Surface
│  │  ├─ A40 强化 Query Task List Surface
│  │  └─ A41 暴露 Operator Next Action 与 Reason Surface
│  └─ T12 建立 Summary Rebuildability 审计
│     ├─ A42 建立 Summary Inventory
│     ├─ A43 建立 Rebuildability Matrix
│     ├─ A44 建立 Summary Rebuild Procedure / Tooling
│     └─ A45 建立 Summary Degraded Recovery Test
│
└─ L5 真实世界验证收口
   └─ T13 建立 Phase-1 真实目标仓验证回路
      ├─ A46 选择 Phase-1 Target Requirement 集合
      ├─ A47 标准化 Phase-1 Validation Protocol
      ├─ A48 建立 Target Evidence Capture Template
      ├─ A49 跑通一个 Target Success Scenario
      ├─ A50 跑通一个 Failure-to-Gap Abstraction Scenario
      ├─ A51 建立 Repeatability Regression
      └─ A52 完成 Rollout Review 与发布前评估
```

---

## 8. ToDo List（Expanded Execution Board）

### P0：对象定义冻结

- [ ] A01 定义 Accepted Packet Identity
- [ ] A02 冻结 Accepted Packet Schema
- [ ] A07 定义 Task Contract Identity
- [ ] A08 冻结 Task Contract Schema
- [ ] A13 输出 Packet / Contract / Worker-Spec 边界矩阵
- [ ] A16 定义 Verify Scorecard Schema
- [ ] A17 定义 Evidence Ledger Schema
- [ ] A20 定义 Findings Schema
- [ ] A21 定义 RecommendedNextAction Schema
- [ ] A26 定义 Gate Outcome Taxonomy
- [ ] A27 统一 Failure Reason Codes

### P1：对象落盘与读取切换

- [ ] A03 写入 Accepted Packet Artifact
- [ ] A04 增加 Accepted Packet Stale Protection
- [ ] A05 Query 迁移到 Accepted Packet 读取
- [ ] A06 Verify 与 Gate 迁移到 Accepted Packet 读取
- [ ] A09 在 Worker Prepare 输出 Task Contract
- [ ] A10 明确 Contract 与 Execution Slice 绑定规则
- [ ] A11 Verify 迁移到 Contract 读取
- [ ] A12 Completion Gate 迁移到 Contract 读取
- [ ] A14 清理重复字段
- [ ] A15 建立对象 Ownership 校验用例

### P2：验证面增强

- [ ] A18 实现 Command / Test / File Evidence Collector
- [ ] A19 实现 Review Evidence Collector
- [ ] A22 Verify 消费 Packet + Contract
- [ ] A23 Verify 产出 Scorecard + Evidence + Findings
- [ ] A24 Completion Gate 消费 Contract Satisfaction
- [ ] A25 Completion Gate 消费 Evidence Completeness
- [ ] A28 在 Query 暴露 Verify / Gate 结构化原因
- [ ] A29 建立 Evaluator / Gate 回归测试

### P3：single-entry intake 增强

- [ ] A30 定义 Intake Classification Rules
- [ ] A31 定义 Request Fusion Rules
- [ ] A32 定义 Thread Correlation Rules
- [ ] A33 定义 Impact Analysis Classes
- [ ] A34 建立 Selective Replan Trigger Engine
- [ ] A35 Submit 写入 Intake / Thread / Change / Todo Summaries
- [ ] A36 Route 强制检查 Epoch Freshness
- [ ] A37 抑制旧 Epoch Queued Task 的继续派发
- [ ] A38 实现 Active Task 的 Checkpoint-Then-Replan 路径

### P4：query 与 summary 治理增强

- [ ] A39 强化 Query Task Detail Surface
- [ ] A40 强化 Query Task List Surface
- [ ] A41 暴露 Operator Next Action 与 Reason Surface
- [ ] A42 建立 Summary Inventory
- [ ] A43 建立 Rebuildability Matrix
- [ ] A44 建立 Summary Rebuild Procedure / Tooling
- [ ] A45 建立 Summary Degraded Recovery Test

### P5：真实目标仓验证增强

- [ ] A46 选择 Phase-1 Target Requirement 集合
- [ ] A47 标准化 Phase-1 Validation Protocol
- [ ] A48 建立 Target Evidence Capture Template
- [ ] A49 跑通一个 Target Success Scenario
- [ ] A50 跑通一个 Failure-to-Gap Abstraction Scenario
- [ ] A51 建立 Repeatability Regression
- [ ] A52 完成 Rollout Review 与发布前评估

---

## 9. Checklist（Expanded Acceptance Checklist）

### 9.1 对象真相验收

- [ ] accepted packet 有稳定 identity、schema、stale protection
- [ ] 每个 accepted epoch 恰有一个 accepted packet truth
- [ ] task contract 与 dispatch 一一绑定
- [ ] contract 明确表达 inScope / outOfScope / doneCriteria / requiredEvidence
- [ ] worker-spec 不再承担主合同语义
- [ ] packet / contract / worker-spec 字段 owner 单一且稳定

### 9.2 evaluator 验收

- [ ] verify 同时消费 packet 与 contract
- [ ] verify 产出 scorecard、evidence ledger、findings、recommended next action
- [ ] verify failure 可精确指向 contract 条款和证据缺口
- [ ] verify success 不自动等于 completed
- [ ] reviewRequired 且无 review evidence 时不会被误判为完成

### 9.3 completion gate 验收

- [ ] gate 逐项消费 contract satisfaction
- [ ] gate 判断 evidence completeness
- [ ] gate 结果为 `completed/incomplete/blocked/needs_replan/needs_review`
- [ ] gate 输出结构化 reasons
- [ ] archive 严格受 gate 控制

### 9.4 intake / thread / epoch 验收

- [ ] submit 不再默认等于新 task
- [ ] 新输入能被分类为明确 intake class
- [ ] request 可融合到现有 thread
- [ ] 只有影响 execution scope / acceptance 时才 bump epoch
- [ ] queued 老 epoch task 不再继续 dispatch
- [ ] active task 在受影响时具备 checkpoint-then-replan 路径

### 9.5 operator surface 验收

- [ ] `harness task` 可直接显示 packet / contract / scorecard / gate / next action
- [ ] `harness tasks` 可用于快速识别关注任务
- [ ] operator 无需阅读 planning trace 也能理解当前状态
- [ ] reason codes 与 operator-friendly 文案同时存在

### 9.6 summary 治理验收

- [ ] 每个 summary 都有 generator
- [ ] 每个 summary 都有 authoritative 来源
- [ ] 每个 derived summary 都有 rebuild 方法
- [ ] summary 丢失后可恢复
- [ ] 没有 summary 偷偷承担 authority

### 9.7 真实目标仓验收

- [ ] 至少一个 target success scenario 跑通
- [ ] 至少一个 failure-to-gap scenario 被正确抽象
- [ ] 重跑同一 requirement 不复现同一 control-plane bug
- [ ] body repo 与 target repo 的边界保持干净

---

## 10. 建议实施节奏（Expanded Rollout Plan）

### Wave 1：冻结对象定义

目标：先把真相对象的 identity 和 schema 固定下来。

建议完成：

- A01
- A02
- A07
- A08
- A13
- A16
- A17
- A20
- A21
- A26
- A27

**结果**

这一波完成后，系统会先拥有稳定的对象契约，后续开发不再在漂浮语义上反复返工。

### Wave 2：接入 packet / contract 真相对象

目标：让 accepted packet 与 task contract 真正写出来、读起来、挡起来。

建议完成：

- A03
- A04
- A05
- A06
- A09
- A10
- A11
- A12
- A14
- A15

**结果**

这一波完成后，packet truth 与 dispatch contract 会成为真实运行时对象，而不是文档概念。

### Wave 3：补强 evaluator 与 gate

目标：把 verify/gate 从轻验收升级为独立裁决面。

建议完成：

- A18
- A19
- A22
- A23
- A24
- A25
- A28
- A29

**结果**

这一波完成后，系统会具备否决伪完成、输出结构化失败原因与下一步动作的能力。

### Wave 4：补强 intake / thread / replan

目标：让输入不再被粗暴映射成新 task，并让 epoch/replan 具备硬规则。

建议完成：

- A30
- A31
- A32
- A33
- A34
- A35
- A36
- A37
- A38

**结果**

这一波完成后，系统会更适合长任务和不断变化的需求流。

### Wave 5：补强 query 与 summary 治理

目标：提升 operator 可读性，并压住 summary 腐化风险。

建议完成：

- A39
- A40
- A41
- A42
- A43
- A44
- A45

**结果**

这一波完成后，operator surface 会变得清晰、稳定、可恢复。

### Wave 6：跑真实 target 验证闭环

目标：证明这不是只在 body repo 内部自洽，而是真正能作用于真实项目。

建议完成：

- A46
- A47
- A48
- A49
- A50
- A51
- A52

**结果**

这一波完成后，架构改造会具备真实世界闭环证据，而不仅仅是理论完备。


当本方案全部落地后，`Klein-Harness` 应呈现如下效果：

### 11.1 对 operator 的效果

operator 不再需要在多个散文件之间拼接心智模型，而是能直接从 `harness task` 看到：

- 当前 accepted packet 是什么
- 本轮 contract 要求完成什么
- verify 认为哪里达标、哪里没达标
- completion gate 为什么还没放行
- 下一步推荐动作是什么

### 11.2 对 runtime 的效果

runtime 将从“能运行且状态较清楚”，升级到“能稳定解释长任务中为什么继续、为什么暂停、为什么重规划、为什么仍未完成”。

### 11.3 对 worker 的效果

worker 不再面对模糊目标，而是面对：

- 明确战略真相：accepted packet
- 明确本轮合同：task contract
- 明确执行边界：worker-spec

这会显著降低：

- scope drift
- 伪完成
- 任务越做越散
- 对高层意图的误解

### 11.4 对 verify 的效果

verify 会从“结果摄入器”升级成真正 evaluator：

- 能否决自我感觉良好的 worker 输出
- 能用 evidence ledger 说明问题
- 能用 scorecard 给出多维判断
- 能输出结构化 repair / replan 建议

### 11.5 对系统长期演进的效果

最重要的效果不是“对象更多”，而是：

- 对象边界更稳定
- authority 更清楚
- 复杂度更可控
- summary 不再偷偷变真相源
- future simplification 也更容易进行

最终，系统应达到一种更优雅的状态：

> runtime 负责真相与裁决，worker 负责执行，verify 负责独立评估，summary 负责投影，而不是互相越界。

---

## 12. 成功定义（Definition of Success）

如果以下条件同时成立，则可认为本轮架构补完成功：

1. accepted packet 成为 accepted epoch 的唯一 packet truth
2. task contract 成为当前 dispatch 的唯一 done definition
3. verify 能独立基于 contract 和 evidence 否决执行结果
4. completion gate 能清晰说明为什么没有 completed
5. submit 成为真正的 single-entry intake，而不总是产生新 task
6. query 成为 operator truth surface，而不是散文件入口
7. derived summaries 均可从 authoritative truth 重建
8. 至少一个真实 target requirement 通过 harness 流程独立闭环完成

---

## 13. 一句话结论

这份方案的核心不是“继续加更多节点”，而是：

**把已经存在但尚未完全硬化的正确设计，收口成一套真正 contract-backed、evaluator-gated、thread-aware、可解释、可审计、可长期演进的 harness runtime。**
