# Anthropic 启发下的 Harness 改造方案

## Background

当前 `Klein-Harness` 已经具备一条可运行的 canonical runtime 主链路：

- `submit`
- `route`
- `dispatch`
- `lease`
- `worker prepare`
- `tmux + codex`
- `checkpoint / outcome`
- `verify`
- `completion gate`

同时，仓库也已经引入了：

- packet / worker-spec / dispatch-ticket 这一组 runtime-owned 对象
- b3e 3+1 编排语义
- qiushi-inspired 的执行纪律
- repo-local 的状态面与闭环回流

最近参考 Anthropic 文章 [Harness design for long-running application development](https://www.anthropic.com/engineering/harness-design-long-running-apps) 后，可以进一步明确一个演进方向：

- planner / generator / evaluator 必须明确分工
- 长任务的核心不是长上下文，而是结构化交接
- 执行前需要一层“当前 sprint 的完成合同”
- evaluator 不应只是结果摄入器，而应成为独立验收面

这份蓝图的目标不是推翻现有 runtime，而是在现有 `route-first-dispatch-second` 主架构上，补强“合同面”和“独立评估面”。

## Goal

把当前 runtime 从：

- `route-first-dispatch-second`

演进为：

- `route-first`
- `packet-owned`
- `contract-backed`
- `evaluator-gated`

具体目标：

- 让每一轮执行前都有一个机器可读、可验收的任务合同
- 让 verify 从“结果摄入”升级为“独立 evaluator”
- 让 packet 成为真正的 accepted epoch 真相对象
- 让 intake / fusion / todo / guard 这些外环语义更接近文档定义

## Non-Goals

这次改造不做以下事情：

- 不引入第二套独立调度器
- 不把 qiushi 变成新的 runtime 控制面
- 不让 worker 获得全局控制权
- 不把系统改造成纯 prompt 编排器
- 不优先追求更多 agent 数量，而忽略对象契约

## Current State

当前仓库的优点：

- canonical CLI 已经统一到 `harness`
- runtime 主链路已经能闭环
- completion gate 已经存在
- worker artifacts 已经结构化
- packet / worker-spec / dispatch-ticket 的概念已经成立

当前最明显的缺口：

1. `submit` 仍主要是“创建 task”，还不是完整的 intake + fusion
2. packet 语义存在，但 accepted packet truth 还不够实体化
3. verify 更像“验收结果摄入”，还不是独立 evaluator loop
4. `executionTasks` 已存在，但缺少“本轮执行合同”这一层
5. guard loop 文档中的 `classify / fuse / derive todo / refresh summaries` 还没全部成为 canonical runtime 的硬前置

## Constraints

- 必须保持 `route-first-dispatch-second`
- completion 仍必须归 runtime，而不是 worker
- worker 仍只能操作 task-local scope
- 现有 `.harness` ledgers 不能被一波改造打散
- 兼容当前 `tmux + codex` 执行链
- 增量演进优先，避免一次性改穿全部 runtime

## Design

### 设计总原则

采用“对象先行，节点渐进替换”的方式。

先把缺少的 runtime 对象补出来，再让现有节点逐步消费这些对象，而不是先大改节点职责。

核心新增两个对象：

1. `accepted-packet.json`
2. `task-contract.json`

并增强两个已有对象：

1. `verify.json`
2. `completion-gate.json`

---

### 1. 新增 `accepted-packet.json`

#### 目的

把“编排完成后的 packet 真相”从 prompt 元数据或 worker-spec 派生物，提升为 accepted epoch 的一等 runtime 对象。

#### 建议结构

至少包含：

- `taskId`
- `threadKey`
- `planEpoch`
- `packetId`
- `objective`
- `constraints`
- `flowSelection`
- `policyTagsApplied`
- `selectedPlan`
- `rejectedAlternatives`
- `executionTasks`
- `verificationPlan`
- `decisionRationale`
- `ownedPaths`
- `taskBudgets`
- `acceptanceMarkers`
- `replanTriggers`
- `rollbackHints`
- `acceptedAt`
- `acceptedBy`

#### 放置位置

建议：

- `.harness/state/accepted-packet-<taskId>.json`

#### 作用

- 作为 accepted epoch 的唯一 packet truth
- route / dispatch / verify / resume 都不再各自拼 packet 语义
- query 也可以直接展示 packet truth，而不是只展示 planning trace

---

### 2. 新增 `task-contract.json`

#### 目的

把一轮 executionTasks 中“当前这次具体做什么、怎么验收、哪些不做”独立出来。

这个对象对应 Anthropic 文章里 generator 和 evaluator 在每个 sprint 前协商的 contract。

#### 建议结构

至少包含：

- `contractId`
- `taskId`
- `dispatchId`
- `planEpoch`
- `executionSliceId`
- `objective`
- `inScope`
- `outOfScope`
- `doneCriteria`
- `acceptanceMarkers`
- `verificationChecklist`
- `requiredEvidence`
- `reviewRequired`
- `contractStatus`
- `proposedBy`
- `acceptedBy`
- `acceptedAt`

#### 放置位置

建议：

- `.harness/artifacts/<taskId>/<dispatchId>/task-contract.json`

#### 作用

- 作为 worker 开工前的“当轮 done 定义”
- 作为 evaluator 验收的直接对照面
- 降低高层 packet 太抽象、worker-spec 太执行化之间的断层

---

### 3. 强化 `verify.json` 为 `verify-scorecard`

#### 目的

把 verify 从“最后给一个 passed/failed”升级为“多维度独立评估”。

#### 建议结构

保留原有 status 字段，同时增加：

- `overallStatus`
- `overallSummary`
- `scorecard`
- `evidenceLedger`
- `findings`
- `reviewChecklist`
- `recommendedNextAction`

其中 `scorecard` 建议固定维度：

- `scopeCompletion`
- `behaviorCorrectness`
- `packetAlignment`
- `evidenceQuality`
- `reviewReadiness`

每个维度至少有：

- `score`
- `threshold`
- `status`
- `summary`

#### 作用

- 让 verify 成为真正 evaluator 面
- completion gate 可以基于多个维度做判断
- follow-up 发射可以更具体，不再只依赖单句 summary

---

### 4. 强化 `completion-gate.json`

#### 目的

让 completion gate 真正消费：

- accepted packet
- task contract
- verify scorecard

而不是主要依赖 verify 的总结果。

#### 建议增加的判断维度

- packet 是否被当前 epoch 接受
- 当前 dispatch 是否对应有效 contract
- contract 的 done criteria 是否全部满足
- evidence ledger 是否完整
- reviewRequired 时 review 是否满足阈值
- 是否存在高优先级 verify finding

#### 结果分类

建议把 gate 输出收敛成：

- `completed`
- `incomplete`
- `blocked`
- `needs_replan`
- `needs_review`

---

### 5. 把 verify 提升为独立 evaluator 角色

#### 目的

保持现有 `internal/verify` 模块，但在职责上更明确：

- 不是只 ingest
- 而是独立地检查 worker 交付是否达标

#### 演进方式

第一阶段不新增新 binary，只增强 `internal/verify`

第二阶段再考虑显式化：

- `evaluator-node`

但它不应成为新的调度器，只应成为独立验收节点。

#### 主要职责

- 对照 accepted packet
- 对照 task contract
- 检查 worker-result claims
- 检查实际文件与命令证据
- 在多文件或高风险任务上执行 review checklist

---

### 6. 把 `submit` 升级为真正的 single-entry intake

#### 目的

把当前“submit 即新建 task”升级成：

- append-only request
- classify
- fusion
- thread bind
- selective replan

#### 建议新增状态面

- `.harness/state/intake-summary.json`
- `.harness/state/thread-state.json`
- `.harness/state/change-summary.json`
- `.harness/state/todo-summary.json`

#### 作用

- 让文档中的 guard loop 真正落到代码
- 让新输入不总是新 task
- 支持 `merged_as_context`、`append_requires_replan`、`accepted_existing_thread`

---

### 7. 强化 query 视图

#### 目的

让 operator 看到的不是“散文件拼出来的状态”，而是清晰的 runtime 视图。

#### 建议在 `task` 视图增加

- accepted packet 摘要
- 当前 task contract 摘要
- verify scorecard 摘要
- completion gate 决策原因
- 下一步推荐动作

#### 价值

- 降低 operator 读 planning trace 的负担
- 让系统更接近 machine-first operator surface 的设计目标

## Conflict Analysis

### 硬冲突 1：对象更多会增加复杂度

确实会增加文件和状态面复杂度。

处理方式：

- 不同时引入多个新主循环
- 先增加对象，再让现有 runtime 使用它们
- 保持 object contract 比 node choreography 更稳定

### 硬冲突 2：packet 和 task-contract 可能语义重叠

处理方式：

- packet 管总意图与全局切片
- task-contract 管当前 dispatch 的验收合同

简单说：

- packet 是战略层
- task-contract 是当前战术层

### 软冲突 3：verify 强化后可能与 completion gate 重复

处理方式：

- verify 负责独立评估和评分
- completion gate 负责最终是否允许宣布完成

verify 是 evaluator。
completion gate 是 runtime judge。

### 软冲突 4：双节点拆分会不会被进一步复杂化

处理方式：

- 现阶段不新增第二套调度器
- 仍保持 orchestrator / worker-supervisor 主边界
- evaluator 先做模块能力增强，后续再决定是否独立成节点

## Verification

改造完成后建议用以下验证路径。

### 对象层验证

- accepted packet 能唯一对应一个 `taskId + planEpoch`
- task-contract 能唯一对应一个 `dispatchId`
- verify-scorecard 能引用 packet 和 contract
- completion gate 能明确解释为什么没有完成

### 流程层验证

- 一个普通 feature task 能生成 packet -> contract -> dispatch -> verify -> complete
- 一个 bug task 能在 verify fail 后进入 replan，并保持 packet / contract 更新
- 一个 resume task 能读取当前 packet / contract / feedback 再继续

### 回归验证

- 现有 `harness submit`
- `harness daemon run-once`
- `harness task`
- `harness control task ...`

这些 canonical surface 不应被破坏。

## Rollout / Migration

### Phase 1：补对象，不改主循环

目标：

- 新增 `accepted-packet.json`
- 新增 `task-contract.json`
- 强化 `verify.json`

做法：

- 在 `worker.Prepare` 和 `verify` 周围补充对象生成与读取
- query 先做只读展示

### Phase 2：让 gate 消费新对象

目标：

- completion gate 以 packet / contract / scorecard 为主判断

做法：

- 改 `internal/verify/gate.go`
- 明确 gate 的 failure reasons

### Phase 3：升级 intake / fusion

目标：

- submit 不再总是等于新 task

做法：

- 引入 intake-summary / thread-state / change-summary / todo-summary
- 把 classify / fuse / bind 变成 canonical runtime 前置

### Phase 4：决定是否独立 evaluator node

目标：

- 视效果决定是否把 evaluator 从模块提升成节点

做法：

- 先通过 `internal/verify` 验证收益
- 不提前引入新 runtime split

## Blueprint -> Harness Mapping

- `accepted-packet.json`
  - 属于 runtime-owned state
- `task-contract.json`
  - 属于 dispatch-local artifact
- `verify-scorecard`
  - 属于 verification artifact / summary surface
- intake / thread / change / todo summaries
  - 属于 outer guard loop state

本方案不要求把这些对象直接塞进 `task-pool`。

建议原则：

- `task-pool` 继续保存 task truth
- packet / contract / scorecard 保存执行真相与验收真相
- summaries 保存 operator 快速读取视图

## Open Questions

1. accepted packet 是否需要进入统一 packet registry，而不只是按 task 单文件保存
2. task-contract 是否应支持 generator / evaluator 双方签收状态
3. verify-scorecard 的阈值应固定还是按 task kind 调整
4. evaluator 是否最终需要单独 runtime role，还是保持模块化即可
