# Klein-Harness 架构收口文档

## 0. 文档定位

本文不是“准备怎么改”的讨论稿，而是面向 `Klein-Harness` 当前方向的**架构收口文档**。

本文用于冻结以下内容：

- 运行时主链路的 authoritative 对象边界
- contract-first 的 done definition 闭环
- evaluator-gated 的验收与完成闭环
- derived summary 的可重建规则
- 渐进迁移顺序、阶段验收口径与稳定状态

本文的作用不是扩张系统，而是收紧系统。本文明确规定：

- 不引入第二调度器
- 不把 skill / methodology / prompt layer 扩写成新的 runtime
- 不把 summary、prompt、trace、markdown 继续抬升为 authority
- 不通过重构重写追求“整洁感”，而通过对象边界冻结和读取方切换完成收口

本文覆盖的 canonical runtime 仍然是：

- `cmd/harness`
- `internal/runtime`
- `internal/route`
- `internal/dispatch`
- `internal/lease`
- `internal/worker`
- `internal/verify`
- repo-local control plane `.harness/`

---

## 1. 当前架构判断

当前系统已经具备可收口的骨架，不需要再发明新的骨架。

### 1.1 已经成立的部分

当前方向已经成立的判断如下：

- `harness` 是唯一 canonical CLI
- Go runtime 已经拥有 route / dispatch / lease / burst / verify / control 主链路
- `tmux` 只是执行承载，不是调度器，不是 authority
- `codex` 只是模型执行后端，不是控制面
- `.harness/` 已经形成 repo-local state plane
- worker 只拥有 task-local execution，不拥有 completion / archive / merge 决策权
- completion 已经不再等价于“worker 说自己做完了”

这意味着系统的主要问题不是“能力缺失”，而是“对象边界和闭环定义还不够硬”。

### 1.2 当前未收紧的地方

需要继续收口的点集中在五类：

1. `accepted-packet / task-contract / worker-spec` 三者边界还不够硬
2. `verify` 仍带有 ingest 色彩，独立 evaluator 身份不够硬
3. `completion-gate` 还需要完全升级为 contract + evidence + scorecard 驱动
4. `submit -> classify -> fuse -> bind -> selective replan` 需要从“已有设计”升级为“唯一前置链路”
5. 部分 summary 仍存在偷偷承载 authority 的风险

### 1.3 当前改造方向的正确结论

当前正确方向不是：

- 重写 runtime
- 发明新的 planning runtime
- 把 spec workflow 重新外置
- 用更复杂的 orchestration 替代当前链路

当前正确方向是：

- 冻结真相对象
- 让读取方统一只读这些对象
- 让 verify 成为 evaluator
- 让 completion gate 成为最终裁决面
- 让所有 derived summary 明确降权并可重建

---

## 2. 改造总目标

本次收口的总目标是把当前系统收紧为一套**对象真相源唯一、合同优先、评估独立、完成判定可解释、summary 可重建**的 runtime。

完成后，系统必须满足以下状态：

- **task truth 唯一**：任务身份、状态、epoch、thread 绑定只由 task truth 持有
- **accepted packet 唯一**：一个 accepted epoch 只有一个 packet truth
- **dispatch contract 唯一**：一个 dispatch 只有一个当前有效的 task contract，且它是这次 dispatch 的唯一 done definition
- **verify judgment 唯一**：verify 输出的是独立 judgment，而不是 worker 结果翻译
- **completion gate 唯一**：是否 completed 只由 completion gate 决定
- **summary 全部降级**：summary 只能投影 authoritative truth，不能重新定义 truth
- **submit 前置链路唯一**：任何新输入都必须进入 classify / fuse / bind / selective replan，而不是直接偷写 task truth
- **target repo 验收闭环成立**：phase-1 必须能在目标仓中验证真实 requirement，而不是只在 body repo 自测自证

---

## 3. 设计原则

### 3.1 单一真相源原则

同一种语义只允许一个 authoritative owner。

如果两个对象都能回答同一个问题，系统迟早漂移。

### 3.2 contract-first 原则

worker 不执行“模糊目标”，只执行当前 dispatch 对应的 task contract。

verify 不验证“worker 自述”，只验证 contract fulfillment。

completion gate 不消费“passed / failed 口头结论”，只消费 contract-aware judgment。

### 3.3 evaluator-gated 原则

verify 不是 ingest 的别名。

verify 的职责是：

- 基于 contract 做独立评估
- 产出 scorecard / findings / reasons / next action
- 否决看起来像完成但证据不足、范围偏移、评审未闭环的执行结果

### 3.4 authority 与 derived 严格分层原则

authoritative truth 参与主链路决策。

derived summary 只服务读取性能、operator 视图和紧凑上下文，不参与重新定义主链路语义。

### 3.5 渐进替换原则

收口顺序必须是：

1. 先冻结定义
2. 再写入对象
3. 再切换读取方
4. 再切换 gate 判定
5. 再做真实世界 target 验证

不允许跳过定义冻结，直接靠运行时行为“自然收敛”。

### 3.6 最小扩张原则

本次收口允许新增的，只能是：

- 缺失的 authoritative object
- 缺失的 evaluator output
- 缺失的 rebuild metadata
- 缺失的 rollout acceptance rule

本次收口不允许新增的，是第二套 runtime 语义层。

---

## 4. 收口目标图

### 4.1 主链路目标图

```text
submit
  -> classify
  -> fuse
  -> bind thread/epoch
  -> selective replan if needed
  -> route
  -> dispatch
  -> lease
  -> prepare accepted-packet
  -> prepare task-contract
  -> prepare worker-spec
  -> bounded burst
  -> ingest worker-result
  -> build evidence ledger
  -> verify as evaluator against contract
  -> completion gate against contract + scorecard + evidence + review
  -> emit next action / completed / blocked / needs_replan / needs_review
  -> rebuild summaries
```

### 4.2 对象层级目标图

```text
Task Truth
  -> owns task identity / thread binding / plan epoch / lifecycle

Accepted Packet Truth
  -> owns accepted-epoch intent, plan selection, acceptance markers, replan triggers

Task Contract Truth
  -> owns current dispatch done definition

Worker Spec
  -> owns execution boundary, write budget, path scope, operational constraints

Worker Result
  -> owns this burst's claimed outcome only

Verify Judgment
  -> owns evaluator scorecard / findings / reasons / next action

Completion Gate
  -> owns final completion decision

Summaries / Projections
  -> own nothing authoritative; rebuild only
```

### 4.3 闭环目标

闭环不是“worker 跑完”。

闭环必须同时成立：

- 输入进入统一 intake 前置链路
- dispatch 之前已有 accepted packet
- dispatch 之时已有 task contract
- worker 只按 contract 执行
- verify 独立评估 contract fulfillment
- completion gate 独立裁决 completed 与否
- 所有摘要都可从 truth 重建
- 真实 target repo 能复现相同闭环

---

## 5. 真相对象边界

本节是本文最重要的收口部分。每个关键对象都必须明确：谁拥有它、谁引用它、谁不能重定义它、坏了会影响什么。

### 5.1 对象边界总表

| 对象 | authoritative owner | 写入方 | 核心语义 | 谁只能引用不能重定义 | 对主链路影响 |
| --- | --- | --- | --- | --- | --- |
| task truth | runtime task ledger | `internal/runtime` / control actions | task 身份、thread、epoch、生命周期、当前状态 | query / verify / summaries / worker | 坏了会直接影响主链路 |
| accepted packet truth | runtime accepted-packet ledger | orchestration prepare / runtime accepted epoch write | accepted epoch 的目标、约束、选定方案、acceptance markers、replan triggers | route / dispatch / verify / gate / query / worker | 坏了会直接影响主链路 |
| task contract truth | current dispatch artifact | worker prepare under runtime control | 当前 dispatch 的 done definition、verification checklist、required evidence | worker / verify / gate / query | 坏了会直接影响主链路 |
| worker spec | current dispatch artifact | worker prepare under runtime control | 操作边界、写入范围、预算、blocked paths、执行约束 | worker / query | 坏了会影响执行边界，但不应重定义 done |
| worker result | current dispatch artifact | worker burst | 本次执行结果、产出声明、运行结论 | verify / query / audits | 坏了会影响该次评估输入 |
| verify judgment truth | verify artifact / latest verification ledger | `internal/verify` | evaluator scorecard、findings、reasons、recommended next action | gate / query / control | 坏了会直接影响验收链 |
| completion gate truth | completion-gate ledger | `internal/verify` or dedicated gate step within runtime | completed / incomplete / blocked / needs_review / needs_replan 的最终判定 | archive / query / control | 坏了会直接影响完成与归档 |
| summaries / projections | derived only | summary generators | operator read model、compact machine read surface | 所有读取方都不得回写 authority | 坏了可重建，不应破坏主链路 |

### 5.2 task truth

`task truth` 是任务主链路的根对象。它回答：

- 这是不是同一个 task
- 属于哪个 thread
- 当前 plan epoch 是多少
- 当前状态是什么
- 当前是否 queued / routing / dispatched / blocked / needs_replan / completed

规则：

- task truth 只能由 runtime ledger 定义
- 任何 summary 都不能重新定义 task status
- worker 不能直接回写 task status 成 completed
- verify 可以提出 `recommendedNextAction`，但不能直接替代 task truth

### 5.3 accepted packet truth

`accepted packet truth` 是 accepted epoch 的唯一战略真相。

它回答：

- 这个 epoch 到底接受了什么目标
- 接受了哪些约束和 owned paths
- 当前选定方案是什么
- acceptance markers 是什么
- 什么变化会触发 replan

规则：

- 一个 `taskId + planEpoch` 只有一个 accepted packet truth
- planning trace、prompt、中间 synthesis 产物都不是 packet truth
- route、verify、completion gate、query 一律只引用 accepted packet，不得自行拼装 packet 语义
- 旧 epoch packet 不能覆盖新 epoch packet

`accepted packet` 是 authoritative，不是 derived。坏了会影响 route / verify / gate 主链路。

### 5.4 task contract truth

`task contract truth` 是当前 dispatch 的唯一 done definition。

它回答：

- 这次 dispatch 到底要完成什么
- 哪些属于 `inScope`
- 哪些属于 `outOfScope`
- 完成标准 `doneCriteria` 是什么
- verify 要检查什么 `verificationChecklist`
- 必须提供什么 `requiredEvidence`
- 哪些 finding 会阻止判定完成

规则：

- 一个 `dispatchId` 只能绑定一个当前有效 contract
- contract 来自 accepted packet 的切片，不是 worker 自由发挥
- worker 执行时消费 contract
- verify 验收时消费同一 contract
- completion gate 判定时消费同一 contract
- repair / selective replan 如果改变 done definition，必须回到 contract 层，而不是只改 worker prompt

`task contract` 是 authoritative，不是 worker note，也不是 summary。

### 5.5 worker spec

`worker spec` 不是 done definition，它只是执行边界对象。

它回答：

- worker 这次能写哪里
- 不能碰哪里
- 预算是多少
- worktree / sandbox / tool boundary 是什么
- checkpoint / rollback / resume 的执行约束是什么

规则：

- worker spec 可以约束“怎么做”，不能重定义“做成什么算完成”
- worker spec 可以细化执行预算，不能改写 contract 的 done criteria
- verify 不应把 worker spec 当成 done truth
- completion gate 不应根据 worker spec 判定完成，只能用它辅助判断是否越界执行

### 5.6 verify judgment truth

`verify judgment truth` 是 evaluator 输出，不是 ingest 归档。

它回答：

- contract fulfillment 是否成立
- 哪些 criteria 满足，哪些未满足
- evidence 是否足够
- review readiness 是否成立
- 当前最重要的 findings 是什么
- 下一步应该 repair / replan / review / complete

规则：

- verify 必须引用 accepted packet + task contract + worker result + evidence ledger
- verify 不能只翻译 worker 的 success / fail
- verify 成功不等于 completed
- verify 必须输出结构化 reasons，而不是只输出自然语言总结

### 5.7 completion gate truth

`completion gate truth` 是最终完成裁决面。

它回答：

- 当前是否可以标记 completed
- 如果不能，阻塞原因是什么
- 是 `incomplete`、`blocked`、`needs_review` 还是 `needs_replan`
- 如果需要 review，缺的是哪类 review evidence
- 如果需要 replan，触发源是什么

规则：

- completion gate 必须消费 contract + verify scorecard + evidence ledger + review evidence
- completion gate 不能只消费 verify 的 `passed/failed`
- archive、closeout、operator 完成视图一律服从 completion gate

### 5.8 summaries / projections

summary 是派生层，不是 authority。

规则：

- 所有 summary 必须显式标记 `schemaVersion / generator / generatedAt / sourceTruths`
- summary 损坏时必须允许直接重建
- summary 不得保存只有自己知道、authoritative source 中不存在的关键字段
- markdown 只能是 JSON summary 的投影，不得成为机器依赖

坏了可以重建的是 summary；坏了影响主链路的是 truth object。这个边界必须长期保持清晰。

---

## 6. contract-first 闭环

### 6.1 为什么 task-contract 是当前 dispatch 的唯一 done definition

因为 dispatch 不是在实现“整个 packet”，而是在实现 packet 中当前被挑出的执行切片。

因此：

- `accepted-packet` 定义的是 accepted epoch 的战略真相
- `task-contract` 定义的是当前 dispatch 的战术完成标准
- `worker-spec` 定义的是这次执行的操作边界

如果没有 `task-contract` 这一层，系统就会退化成：

- worker 根据 packet 自己解释 done
- verify 根据 worker 结果反推 done
- completion gate 根据 passed / failed 近似判断 done

这不是 contract-first，而是语义漂移。

### 6.2 task-contract 与 accepted-packet 的边界

边界如下：

| 对象 | 层级 | 回答的问题 |
| --- | --- | --- |
| accepted-packet | 战略层 | 这个 epoch 接受了什么目标、方案、约束、验收标记 |
| task-contract | 战术层 | 这次 dispatch 到底要交付什么才算 done |
| worker-spec | 执行层 | worker 这次可以怎么做、不能怎么做 |

收口规则：

- packet 不能被 worker-spec 稀释
- contract 不能被 worker prompt 替代
- worker-spec 不能篡改 contract 的 scope 和 done criteria

### 6.3 task-contract 必须包含的最小字段

当前 dispatch 的 contract 至少必须包含：

- `contractId`
- `taskId`
- `dispatchId`
- `planEpoch`
- `packetRef`
- `inScope`
- `outOfScope`
- `doneCriteria`
- `verificationChecklist`
- `requiredEvidence`
- `reviewRequired`
- `failureSeverityRules`
- `replanTriggers`

这些字段的意义是：

- worker 知道必须交什么
- verify 知道必须查什么
- gate 知道必须凭什么批准完成
- query 知道必须展示什么给 operator

### 6.4 worker 如何消费它

worker 只能把 contract 视为本次执行的任务合同。

worker 的职责是：

- 按 `inScope` 执行
- 避免触碰 `outOfScope`
- 围绕 `doneCriteria` 组织产出
- 围绕 `requiredEvidence` 留下证据
- 遇到 `replanTriggers` 时停止把模糊变更强行做完

worker 不允许：

- 自行扩大 contract scope
- 用“顺手改了更多”替代 contract fulfillment
- 只留下 free-text 自述而不留下 evidence

### 6.5 verify 如何消费它

verify 不是拿 contract 做背景阅读，而是拿 contract 做主判据。

verify 至少必须逐项回答：

- `doneCriteria` 是否满足
- `verificationChecklist` 是否被验证
- `requiredEvidence` 是否齐全
- `reviewRequired` 是否仍未闭环
- worker 是否发生 scope drift

换句话说，verify 的主问题不是“worker 跑成功了吗”，而是“contract fulfillment 成立了吗”。

### 6.6 completion gate 如何消费它

completion gate 不是再次解释 packet，而是依据 contract 判定：

- contract 是否存在且是当前 dispatch 的有效 contract
- contract 条款是否全部满足
- scorecard 是否达到阈值
- required evidence 是否全部满足
- review evidence 是否满足 reviewRequired
- 是否仍存在阻断型 findings

### 6.7 repair / replan 如何回到它

repair 和 replan 的回流原则如下：

- 只修实现、不改完成定义：回到同一个 contract 下继续执行
- 完成定义变了：生成新 contract
- accepted 目标或 acceptance markers 变了：回到 accepted packet 并 bump epoch

这条规则非常关键。否则系统会把本应回到 contract / packet 层的问题，偷偷塞回 worker prompt。

---

## 7. evaluator-gated 闭环

### 7.1 为什么 verify 不是 ingest，而是 evaluator

ingest 的职责是把 task-local 结果收进 runtime。

evaluator 的职责是独立判断这些结果是否满足 contract。

`verify` 必须属于后者，原因只有一个：

如果 verify 只是 ingest，那么系统就没有独立的完成前评估层，worker 的自述会被动变成事实。

因此 verify 的输入不是“日志文件”，而是一个评估包：

- accepted packet
- task contract
- worker result
- evidence ledger
- review evidence
- 当前 route / dispatch / lease identity

verify 的输出也不是“收到了什么”，而是：

- scorecard
- findings
- reasons
- recommended next action
- reviewRequired closure state

### 7.2 verify 与 completion gate 的职责边界

二者边界必须长期保持硬分离：

| 对象 | 职责 |
| --- | --- |
| verify | 评估 contract fulfillment，给出 judgment、findings、scorecard、next action |
| completion gate | 在 verify judgment 基础上，结合 gate 规则做最终 completed / not completed 裁决 |

verify 回答的是：**质量与满足度怎么样**。

completion gate 回答的是：**现在能不能正式算完成**。

### 7.3 为什么 verify 成功不等于 completed

因为 verify 只证明“评估通过”，不证明“所有 gate 条件都已闭环”。

以下场景都可能出现“verify 成功但不应 completed”：

- `reviewRequired=true`，但 review evidence 不足
- contract fulfillment 成立，但 archive prerequisite 未满足
- 当前 dispatch 不是最新 authoritative dispatch
- evidence 足够证明功能成立，但缺少要求中的 target validation evidence
- 存在高优先级 open finding 被标记为必须在 close 前清除

因此 `verification passed` 只是 gate 输入，不是 gate 结论。

### 7.4 scorecard、evidence ledger、completion gate 的关系

三者关系如下：

- `scorecard` 负责表达多维 judgment
- `evidence ledger` 负责证明 judgment 来自什么证据
- `completion gate` 负责将 judgment 与制度规则转成最终完成裁决

任何缺一个，闭环都会塌：

- 没有 scorecard：只能得到模糊 passed / failed
- 没有 evidence ledger：无法追溯 judgment 来源
- 没有 completion gate：verify judgment 无法转成最终完成制度

### 7.5 verify scorecard 最小维度

scorecard 至少固定以下维度：

- `scopeCompletion`
- `behaviorCorrectness`
- `packetAlignment`
- `evidenceQuality`
- `reviewReadiness`
- `targetValidation`（当该 dispatch 要求真实目标验证时）

每个维度至少包含：

- `status`
- `threshold`
- `summary`
- `blocking`
- `evidenceRefs`

### 7.6 evidence ledger 最小结构

evidence ledger 至少支持：

- command result
- test output
- file diff / changed paths
- artifact existence
- checkpoint reference
- review evidence
- target repo validation evidence

规则：

- findings 必须能反向追溯到 evidence refs
- gate 拒绝理由必须能反向追溯到 evidence refs 或缺失项
- free-text 不是证据本体，只能是证据摘要

### 7.7 reviewRequired 场景如何闭环

`reviewRequired` 不是一个提示词，而是 contract/gate 规则。

闭环方式：

1. contract 声明 `reviewRequired`
2. verify 评估 `reviewReadiness`
3. evidence ledger 收集 review evidence
4. completion gate 检查 review evidence 是否达到阈值
5. 未达到时输出 `needs_review`，而不是 completed

### 7.8 findings / reasons / next action 的结构化回流

verify 和 gate 的回流必须结构化，不允许只写自然语言段落。

最小结构：

- `findings[]`
  - `code`
  - `severity`
  - `summary`
  - `evidenceRefs[]`
  - `blocking`
- `reasons[]`
  - `code`
  - `ownerObject`
  - `summary`
- `recommendedNextAction`
  - `type`：`repair | review | replan | complete | block`
  - `owner`：`worker | runtime | operator`
  - `basedOn[]`

这保证系统能自动说明：

- 为什么没完成
- 缺的是什么
- 下一步谁来补
- 应该回到哪一层闭环

---

## 8. summary rebuildability

### 8.1 rebuildability 的硬规则

所有 summary 都必须能明确回答四个问题：

1. 从哪些 authoritative truths 生成
2. 由哪个 generator 生成
3. 损坏后如何重建
4. operator 是否可以直接依赖它读取，但不能依赖它定义 truth

### 8.2 authoritative 与 derived 的分层

#### authoritative 层

以下对象属于 authoritative 层，直接参与主链路：

- request queue
- task truth
- accepted packet truth
- dispatch ticket / lease truth
- task contract truth
- worker result
- verify judgment truth
- completion gate truth

#### derived 层

以下对象属于 derived 层：

- runtime summary
- task summary
- queue summary
- todo summary
- current.json
- progress.json
- progress.md
- markdown projections
- packet progress / compact operator surfaces

### 8.3 哪些坏了可重建，哪些坏了会打断主链路

| 类别 | 损坏后果 | 处理原则 |
| --- | --- | --- |
| authoritative truth | 会直接影响 route / verify / gate / archive | 视为主链路故障，禁止用 summary 补真相 |
| derived summary | 不应阻断主链路，只影响可读性或读取性能 | 直接重建，不得手工补 authority |

### 8.4 rebuild metadata 要求

每个 summary 必须带：

- `schemaVersion`
- `generator`
- `generatedAt`
- `sourceTruths`
- `rebuildable: true`

`progress.md` 这类 markdown 投影必须明确：

- 来源是 JSON summary
- 自己不是 authority
- 删除后可以从 JSON 重渲染

### 8.5 高风险 derived summary 清单

以下 summary 最容易偷偷长成 authority，必须重点降权：

- `todo-summary.json`
- `task-summary.json`
- `current.json`
- 任何展示“当前目标 / 下一步 / 为什么阻塞”的紧凑视图
- 任何 markdown closeout / progress 文档

这些对象只能摘要，不得比 authoritative truth 多出独有语义。

---

## 9. 复杂度风险与防腐规则

### 9.1 最容易语义重叠的对象

长期最容易重叠的对象只有四组：

1. `accepted-packet` vs `task-contract` vs `worker-spec`
2. `worker-result` vs `verify judgment`
3. `verify judgment` vs `completion gate`
4. `task truth` vs `todo-summary` / `task-summary`

防腐规则：

- strategic truth 只能在 packet
- done definition 只能在 contract
- execution boundary 只能在 worker-spec
- evaluator judgment 只能在 verify
- final completion decision 只能在 gate
- operator convenience text 只能在 summary

### 9.2 最容易偷偷长成 authority 的 summary

必须长期警惕以下腐化模式：

- summary 持有 authoritative object 中没有的阻塞原因
- markdown 写出比 JSON truth 更详细的“真实状态”
- query 为了方便直接综合多个 summary，绕开 truth object
- prompt / trace 被拿来补 authoritative 缺口

禁止规则：

- 不允许用 prompt、trace、handoff 反推 packet truth
- 不允许用 worker note 反推 contract truth
- 不允许用 query 拼装结果替代 gate truth

### 9.3 最容易 drift 的流程

以下流程最容易长期漂移：

- `submit -> classify -> fuse -> bind -> selective replan`
- `reviewRequired` 触发后的闭环
- `needs_replan` 与 `needs_review` 的分流
- 旧 epoch queued task 的抑制
- target repo phase-1 validation 的证据回流

防漂移规则：

- 每个流程节点都必须有明确 owner object
- 每个流程切换点都必须有 reason codes
- 每个流程最终状态都必须能由 gate 或 task truth 直接解释

### 9.4 禁止继续膨胀的区域

以下区域必须明确停止膨胀：

1. prompt layer
   - prompt 只负责生成或消费 authoritative object，不得变成 runtime 语义宿主
2. methodology layer
   - proposal/spec/design/tasks 的旧外显流程不得重新长回 runtime 主链路
3. summary layer
   - 不再新增承载“真实状态”的 summary 类型来回避 truth object 缺口
4. compatibility layer
   - shell wrapper、旧命令、兼容输出不得继续承载 canonical 语义

### 9.5 新增对象前必须回答的问题

以后新增任何 runtime object，必须先回答：

1. 它回答的是哪个唯一问题？
2. 这个问题现有对象是否已经回答？
3. 它是 authoritative 还是 derived？
4. 谁写它，谁读它，谁不能重定义它？
5. 它坏了是重建，还是主链路故障？

只要这五个问题答不清，就不允许新增对象。

---

## 10. 原子任务清单

下列任务不是泛泛建议，而是本次收口的最小原子实施单元。

### A. 定义冻结层

#### A01 冻结 truth object glossary

- 输出：统一术语表
- 验收：task truth / accepted packet / task contract / worker spec / verify judgment / completion gate / summary 全部只有一个定义

#### A02 冻结 accepted packet schema 与 identity

- 输出：`taskId + planEpoch` 唯一识别规则、最小 schema
- 验收：route / verify / query 可引用同一 packet identity

#### A03 冻结 task contract schema 与 identity

- 输出：`dispatchId -> contractId` 绑定、最小 schema
- 验收：每次 dispatch 都能定位唯一 contract

#### A04 冻结 verify scorecard / findings / reasons schema

- 输出：固定 scorecard 维度与 findings taxonomy
- 验收：verify 不再只输出单一 passed / failed

#### A05 冻结 completion gate reason taxonomy

- 输出：`missing_packet`、`missing_contract`、`missing_evidence`、`review_pending`、`failed_done_criteria`、`stale_dispatch`、`needs_replan_due_to_scope_change` 等 reason codes
- 验收：operator 单次读取即可知道为什么没完成

### B. 写入对象层

#### A06 写 accepted packet truth

- 输出：accepted epoch authoritative object
- 验收：旧 epoch 不能覆盖新 epoch

#### A07 写 task contract truth

- 输出：dispatch 级 authoritative contract object
- 验收：worker / verify / gate 共享同一 contract

#### A08 写 evidence ledger

- 输出：结构化证据账本
- 验收：findings 可反查证据

#### A09 写 verify judgment

- 输出：scorecard + findings + reasons + recommendedNextAction
- 验收：verify 具备 evaluator 语义

#### A10 写 completion gate truth

- 输出：结构化 completion decision
- 验收：gate 可区分 completed / incomplete / blocked / needs_review / needs_replan

### C. 读取方切换层

#### A11 query 切换到 packet / contract / gate

- 输出：operator truth surface
- 验收：单次 `task` 视图能回答现在做什么、为什么没完成、下一步是什么

#### A12 verify 切换到 contract-aware evaluator

- 输出：verify 只按 contract 判定 fulfillment
- 验收：worker success 不再自动等于 verify success

#### A13 gate 切换到 contract + scorecard + evidence 组合判定

- 输出：新的 gate 读取链路
- 验收：gate 不再只看 verify status

### D. intake 与 replan 收紧层

#### A14 submit 前置链路硬化

- 输出：classify / fuse / bind / impact analysis / selective replan 规则
- 验收：新输入不再默认生成新 task

#### A15 queued / inflight epoch 抑制规则落地

- 输出：旧 epoch queued task 不再被错误 dispatch
- 验收：superseded queued work 被正确抑制

### E. rebuild 与 target validation 层

#### A16 summary rebuildability inventory

- 输出：每个 summary 的 sourceTruths / generator / rebuild rule
- 验收：删除 derived summary 后可重建

#### A17 target-repo phase-1 validation 接入

- 输出：真实目标仓验证回路与证据模板
- 验收：至少一条真实 requirement 在 target repo 中闭环

#### A18 failure lineage 与 capability gap 回写

- 输出：target repo 失败可抽象回 harness capability gap
- 验收：同类真实失败不会无限重复出现为“偶发问题”

---

## 11. 谱系方案

本次收口按谱系推进，不按模块热闹程度推进。

### 11.1 谱系主线

```text
L1 定义冻结
  -> L2 truth object 写入
    -> L3 evaluator 写入
      -> L4 query / verify / gate 读取方切换
        -> L5 intake / replan 硬化
          -> L6 summary rebuild audit
            -> L7 target repo validation
```

### 11.2 能力谱系映射

| 谱系层 | 目标 | 对应原子任务 |
| --- | --- | --- |
| L1 | 冻结对象与 reason taxonomy | A01-A05 |
| L2 | 写入 packet / contract / evidence | A06-A08 |
| L3 | 写入 verify judgment / gate truth | A09-A10 |
| L4 | 切换 query / verify / gate 读取方 | A11-A13 |
| L5 | 硬化 submit / selective replan | A14-A15 |
| L6 | 证明 summary 可重建 | A16 |
| L7 | 建立真实目标仓验收 | A17-A18 |

### 11.3 依赖规则

必须遵守以下依赖：

- 未冻结 schema，不切读取方
- 未写 truth object，不让 summary 假装已有 truth
- 未完成 verify evaluator 化，不切 gate 判定
- 未完成 gate contract 化，不宣布 closure 完成
- 未接入 target repo 验证，不宣布 phase-1 验收闭环完成

---

## 12. Todo / Checklist

以下 checklist 用于实施与评审，不是愿望清单。

### 12.1 定义冻结检查

- [ ] task truth 的 owner、输入、输出、边界已冻结
- [ ] accepted packet truth 的 identity、schema、stale rule 已冻结
- [ ] task contract truth 的 identity、schema、done definition 地位已冻结
- [ ] worker spec 的执行边界定义已冻结
- [ ] verify judgment 的 scorecard / findings / reasons / next action schema 已冻结
- [ ] completion gate 的 reason taxonomy 与 decision schema 已冻结

### 12.2 对象写入检查

- [ ] accepted packet 已成为 accepted epoch 的唯一 truth object
- [ ] task contract 已成为 dispatch 的唯一 done definition
- [ ] evidence ledger 已结构化落盘
- [ ] verify judgment 已结构化落盘
- [ ] completion gate 已结构化落盘

### 12.3 读取方切换检查

- [ ] query 已不再从 trace / prompt 反推 packet 或 contract
- [ ] verify 已只按 contract 做 fulfillment 判定
- [ ] gate 已消费 contract + scorecard + evidence + review
- [ ] archive 已只服从 completion gate

### 12.4 intake / replan 检查

- [ ] submit 已固定经过 classify -> fuse -> bind -> selective replan
- [ ] context enrichment 不再默认变成新 task
- [ ] scope change 会显式触发 replan / epoch bump
- [ ] queued 老 epoch work 不再误 dispatch

### 12.5 rebuildability 检查

- [ ] 每个 summary 都声明 sourceTruths / generator / rebuild rule
- [ ] markdown projection 已明确不是 authority
- [ ] 删除 derived summary 后能自动或受控重建

### 12.6 target validation 检查

- [ ] target repo phase-1 验证路径已存在
- [ ] 真实 requirement 有 evidence ledger 记录
- [ ] 失败能回写为 harness capability gap
- [ ] 不通过手工修 target business code 伪造成功

---

## 13. 分阶段 rollout

分阶段 rollout 的目的不是“方便排期”，而是确保每个阶段结束后系统进入新的稳定状态。

### Phase 0：冻结定义

**目标**

冻结 truth object、contract schema、verify schema、gate taxonomy。

**本阶段动作**

- 完成 A01-A05
- 冻结 packet / contract / verify / gate 最小字段与身份规则
- 冻结 authoritative 与 derived 的边界定义

**阶段验收**

- 文档和 schema 一致
- 每个关键对象都能回答“谁拥有它”
- 每个 failure reason 都有结构化 code

**阶段结束后的新稳定状态**

系统虽然仍可能沿旧读取路径运行，但对象定义不再模糊，不再允许多重解释。

### Phase 1：写 truth objects

**目标**

把 packet、contract、evidence、verify、gate 写成真实对象。

**本阶段动作**

- 完成 A06-A10
- 写 accepted packet truth
- 写 task contract truth
- 写 evidence ledger
- 写 verify judgment
- 写 completion gate truth

**阶段验收**

- dispatch 之后存在 contract truth
- verify 之后存在 judgment truth
- gate 之后存在 completion truth
- 旧 epoch 无法覆盖新 epoch packet truth

**阶段结束后的新稳定状态**

truth objects 已存在，但部分读取方还可兼容旧路径；系统进入“新真相已写入”的稳定状态。

### Phase 2：切换读取方

**目标**

让 query、verify、gate 统一读取新 truth objects。

**本阶段动作**

- 完成 A11-A13
- query 切到 packet / contract / gate
- verify 切到 contract-aware evaluator
- gate 切到 contract + scorecard + evidence 组合判定

**阶段验收**

- operator 单次 task view 可读懂当前状态
- verify 成功不再自动导出 completed
- gate 不再只看 verify status

**阶段结束后的新稳定状态**

读取面和写入面已经统一到同一批 authoritative objects，系统进入“同源读取”的稳定状态。

### Phase 3：切换 intake 与 replan 前置链路

**目标**

把 submit -> classify -> fuse -> bind -> selective replan 硬化为唯一入口前置链路。

**本阶段动作**

- 完成 A14-A15
- 明确 impact class
- 明确 epoch bump 与 queued supersede 规则

**阶段验收**

- context enrichment 不再错误拆成新 task
- scope change 会触发 selective replan
- 老 epoch queued work 被正确抑制

**阶段结束后的新稳定状态**

新输入不再绕过 thread / epoch 系统，系统进入“前置硬化”的稳定状态。

### Phase 4：summary rebuild 审计

**目标**

证明所有 derived surfaces 都可以重建，不再暗藏 authority。

**本阶段动作**

- 完成 A16
- 为所有 summary 标记 sourceTruths / generator / rebuild rule
- 清理高风险 summary 的隐藏 authority

**阶段验收**

- 抽样删除 derived summary 可重建
- markdown 不再被机器依赖为 truth

**阶段结束后的新稳定状态**

control plane 的 authority / projection 分层稳定，系统进入“summary 降权完成”的稳定状态。

### Phase 5：引入真实世界 target 验证

**目标**

把 target repo phase-1 validation 纳入常规验收闭环。

**本阶段动作**

- 完成 A17-A18
- 选择真实 target requirement
- 在 target repo 跑完整 harness 闭环
- 把失败抽象回 capability gap

**阶段验收**

- 至少一条真实 requirement 在 target repo 中闭环
- 同类失败不会再次以“偶发流程问题”形式重复出现
- target validation evidence 已进入 evidence ledger / lineage

**阶段结束后的新稳定状态**

系统不再只在 body repo 自测自证，进入“真实目标验收闭环成立”的稳定状态。

---

## 14. 最终效果描述

收口完成后，`Klein-Harness` 应呈现以下稳定特征：

- operator 不必读 trace / prompt 才知道当前 task 真相
- worker 不再拥有 done definition 的解释权
- verify 不再是 ingest 的包装，而是独立 evaluator
- completion gate 不再是 passed / failed 翻译器，而是最终完成裁决面
- packet / contract / worker-spec 三层边界长期稳定
- reviewRequired、needs_replan、blocked、completed 都有统一结构化闭环
- summaries 全部可重建且明确降权
- target repo phase-1 validation 成为真实验收链，而不是补充演示

此时系统的“完整”不是因为文档写得全，而是因为任何关键问题都能被唯一对象直接回答：

- 任务是谁：看 task truth
- 当前 accepted 的是什么：看 accepted packet
- 这次 dispatch 的 done definition 是什么：看 task contract
- 这次执行做了什么：看 worker result
- evaluator 怎么判断：看 verify judgment
- 为什么还不能完成：看 completion gate
- operator 快速视图怎么看：看 summaries，但知道它们可重建且非 authority

---

## 15. 成功定义

本次架构收口完成的成功定义如下。

### 15.1 对象成功定义

- 每个关键对象都能明确回答“谁拥有它”
- 每个关键对象都能明确回答“谁只能引用，不能重定义它”
- 每个 derived 状态都能明确回答“从哪重建”
- 每个 authoritative 状态都能明确回答“坏了为何会打断主链路”

### 15.2 闭环成功定义

- contract-first 闭环成立：dispatch done definition 唯一由 task contract 定义
- evaluator-gated 闭环成立：verify 独立判断 contract fulfillment
- completion-gated 闭环成立：completed 只由 completion gate 判定
- intake 前置闭环成立：submit 必须经过 classify / fuse / bind / selective replan
- target validation 闭环成立：至少一条真实 requirement 在 target repo 中完成闭环

### 15.3 长期复杂度成功定义

- 没有新的 summary 偷偷长成 authority
- 没有新的 prompt / methodology 层偷偷变成 runtime 主链路
- 没有新的对象和现有 truth object 争夺同一语义
- 后续新增对象前都必须回答唯一问题、authority 类型、重建规则、失败影响

### 15.4 最终判定

只有当以下四件事同时成立，才能说本次收口完成：

1. truth object 边界冻结并已切换到主读取路径
2. verify 与 gate 已形成 evaluator-gated completion loop
3. summaries 已证明可重建且不再承载 authority
4. target repo phase-1 validation 已形成真实验收闭环

少任一项，都只能算“局部优化”，不能算架构收口完成。
