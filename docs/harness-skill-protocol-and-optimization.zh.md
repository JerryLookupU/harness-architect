# Harness Skill 协议与优化说明

## 1. 文档目标

这份文档专门解释一件事：

**当 Codex / Claude 在 Klein-Harness 里“使用 skill”时，skill 到底处在什么位置，什么情况应该用，应该产生什么效果，当前优化点又是什么。**

本文不把 skill 当成第二调度器，也不把 `SKILL.md` 误认为 runtime 真相源。

本文只做三件事：

1. 说明 harness skill 的协议边界
2. 说明不同 skill 的使用时机、效果和优化点
3. 说明这些 skill 如何映射到 canonical runtime

---

## 2. 先讲清楚：skill 在当前架构里的真实位置

当前仓库里，和 skill 相关的面一共有三层：

### 2.1 安装型 skill 入口面

主要文件：

- `skills/klein-harness/SKILL.md`
- `skills/harness-log-search-cskill/SKILL.md`
- `skills/qiushi-execution/SKILL.md`

这一层的作用是：

- 给人类与原生 Codex 一个**入口说明**
- 说明这个 skill 是做什么的
- 说明什么时候应该触发
- 说明默认读什么、不要做什么

这一层**不是 runtime 真相源**。

### 2.2 runtime-internal prompt / contract 面

主要文件：

- `prompts/spec/README.md`
- `prompts/spec/orchestrator.md`
- `prompts/spec/methodology.md`
- `prompts/spec/packet.md`
- `prompts/spec/worker-spec.md`
- `prompts/spec/verify.md`

这一层的作用是：

- 把 orchestration、methodology、packet、worker-spec、verify 的合同写清楚
- 规定 route-first-dispatch-second 的 prompt contract
- 规定方法论如何附着在 runtime 上，而不是长出第二 runtime

### 2.3 canonical Go runtime 面

主要文件：

- `internal/route/gate.go`
- `internal/orchestration/defaults.go`
- `internal/worker/manifest.go`
- `internal/query/service.go`
- `internal/instructions/discovery.go`

这一层的作用是：

- 决定 route policy tags
- 决定 methodology / judge / execution loop 合同
- 决定 dispatch ticket / worker prompt / worker-spec 实际写什么
- 决定 operator 能看到什么

**真正会影响 Codex 行为的，是这一层。**

---

## 3. 当前 skill 协议的核心判断

### 3.1 `SKILL.md` 是入口，不是 authority

`skills/*/SKILL.md` 的主要职责：

- 提供使用场景
- 提供最小读集
- 提供行为纪律
- 提供 drift 提醒

但真正的 authority 仍然是：

- route 的 `policy_*` reason codes
- orchestration defaults 里的 methodology / execution loop / constraints
- worker manifest 写出来的 dispatch ticket / worker prompt / worker-spec
- query 暴露出来的可观测 runtime surface

### 3.2 当前仓库并没有“repo-local skill runtime”

从 `internal/instructions/discovery.go` 看，当前发现的是：

- `AGENTS.override.md`
- `AGENTS.md`

而不是 repo-local `skills/*/SKILL.md`。

这意味着当前 skill 协议有两个入口：

1. **原生安装型入口**
   - skill 安装到 `$CODEX_HOME/skills`
   - 原生 Codex 可以直接使用

2. **repo runtime 引用型入口**
   - route reason codes
   - execution loop skill path
   - prompt refs
   - worker manifest 注入

因此第一轮优化不应该发明新的 skill runtime，而应该先把：

- skill 使用条件
- skill 产生的 runtime 效果
- skill 和 policy tag / prompt / query 的映射

写清楚并做轻量增强。

---

## 4. 技能协议总链路

当前建议把 harness skill 理解成下面这条协议链：

```text
用户请求 / 当前任务状态
  -> route 识别 task type / risk / resume / inspection 信号
  -> route 产出 reasonCodes / policy tags
  -> orchestration defaults 选择 methodology / judge flow / execution loop / constraints
  -> worker manifest 把这些合同写进 dispatch ticket / worker-spec / prompt
  -> Codex 执行时按这些合同读状态、改代码、验证、交接
  -> verify / gate 决定是否收口
  -> query 对 operator 暴露 skill discipline 和当前状态
```

这一链路里每层职责必须清楚：

- route：决定**该走哪种 discipline**
- orchestration：决定**如何结构化表达 discipline**
- manifest：决定**Codex 实际看到什么**
- query：决定**operator 如何理解当前 discipline**

---

## 5. Skill 选择矩阵

| Skill | 什么情况使用 | 不该什么情况使用 | 期望效果 | 当前 canonical 映射 | 主要优化点 |
| --- | --- | --- | --- | --- | --- |
| `klein-harness` | bootstrap、refresh、audit、agent-entry；需要补 `.harness/` 协作面；需要明确 claim / handoff / operator surface | 普通单次代码实现任务；单纯 bug fix；不需要重整 harness 状态时 | 先看 control plane，再看 execution plane，再看 operator plane；建立/修复协作契约 | `skills/klein-harness/SKILL.md` + `internal/query/service.go` + `internal/worker/manifest.go` + `internal/route/gate.go` | 补“Use When / Do Not Use When / Expected Effects”；补 harness-state-first discipline；补 operator-surface-required hint |
| `harness-log-search-cskill` | 需要查 handoff、verify、跨 worker 线索、operator debug、RCA、resume 前状态恢复 | 正常代码任务默认不应先扫 raw logs；没有明确日志诉求时不应过早进入 | 优先热状态、compact logs、index；仅在 detail 或证据不足时回退 raw logs | `skills/harness-log-search-cskill/SKILL.md` + `internal/query/service.go` + `internal/worker/manifest.go` | 把 compact-first / raw-fallback 纪律编码到 manifest/query，而不是只写在 doc 里 |
| `qiushi-execution` | 任务复杂、事实不足、需要 bounded execution、需要 evidence-first verify、需要诚实 closeout | 不能把它当成第二 runtime；不能用它替代 route / dispatch / verify / gate | 调查优先 -> 聚焦主线 -> 小步执行 -> 证据验证 -> 诚实复盘 | `skills/qiushi-execution/SKILL.md` + `internal/orchestration/defaults.go` + `internal/worker/manifest.go` | 保持它是 discipline，不是第二 runtime；把 route policy tag 和 execution loop 的映射解释得更清楚 |
| `systematic-debugging` | bug / failure / regression / crash / RCA-first 场景 | 普通 feature 无需默认使用 | 先证据、后假设、再最小修复 | 当前主要由 `README.md` guardrail mapping + `internal/route/gate.go` 的 `policy_bug_rca_first` 驱动 | 补 skill hint surface；让 query/manifest 能看出为什么此 skill 被激活 |
| `blueprint-architect` | recommendation / compare / choose / trade-off / options before plan | 已经进入明确实现阶段时不该继续扩规划 | 先比较方案，再收敛单一路线 | 当前主要由 `README.md` guardrail mapping + `internal/route/gate.go` 的 `policy_options_before_plan` 驱动 | 补 route -> planning -> query 的链路可见性 |

---

## 6. 每个 skill 该在什么情况下使用

## 6.1 `klein-harness`

### 应使用的情况

适用于以下场景：

- 仓库还没有 `.harness/`
- `.harness/` 存在，但状态已经漂移或断裂
- 需要 bootstrap / refresh / audit / agent-entry
- 需要补：
  - task claim 规则
  - session / worktree / handoff 规则
  - operator overview / watch / metrics 入口
- 需要判断这套 harness 是否还能支撑多 agent 并行推进

### 不应使用的情况

以下情况不应默认拉起 `klein-harness`：

- 只是普通 feature 实现
- 只是单个 bug fix
- 当前 `.harness/` 已经健康，任务只是普通 worker lane
- 只是查日志、查 verify 失败、resume handoff，此时优先考虑 log-search / qiushi discipline

### 期望效果

使用它之后，理想效果应该是：

1. 先按三层理解当前 harness：
   - control plane
   - execution plane
   - operator plane
2. 明确当前模式：
   - bootstrap
   - refresh
   - audit
   - agent-entry
3. 产出最小可用的协作契约，而不是堆文档
4. 让后续 agent 不需要重新猜：
   - 现在做什么
   - 谁在占哪条路径
   - 失败后怎么回退
   - 下一位如何接力

### 当前优化点

- 把模式判断写得更显式
- 把 `control plane / execution plane / operator plane` 三层理解法变成 query / prompt 可见约束
- 把 unattended / parallel / forever 场景的 operator surface 要求下沉到 runtime hint
- 避免继续在 `SKILL.md` 内堆积过多 runtime 事实

---

## 6.2 `harness-log-search-cskill`

### 应使用的情况

适用于以下场景：

- 要查 `.harness` 的 compact handoff logs
- 要做 RCA / operator debug
- resume 前需要先恢复状态
- 需要验证跨 worker handoff 的事实链
- 需要围绕 verify 失败找证据

### 不应使用的情况

以下情况不应默认使用：

- 普通 feature 实现
- 普通代码编辑任务
- 没有日志或状态恢复诉求时
- 为了省事直接扫全量 raw logs

### 期望效果

这个 skill 的效果不是“把全部 transcript 打出来”，而应该是：

1. 先读热状态：
   - `current.json`
   - `runtime.json`
   - `request-summary.json`
   - `lineage-index.json`
   - `log-index.json`
2. 再读 compact logs / handoff
3. 只有 detail 或证据不足时才回退 raw runner logs
4. 返回 one-screen summary、blockers、verification notes、evidence refs

### 当前优化点

- 把 `compact-first / raw-fallback` 规则沉到 worker prompt / query surface
- 让 query 本身更贴近这个检索顺序
- 让 route 能明确打出 `policy_log_compact_first` 一类信号
- 让 operator 能从 `harness task` 看出当前为何走 compact log discipline

---

## 6.3 `qiushi-execution`

### 应使用的情况

适用于以下场景：

- 任务复杂，但事实不足
- 方向很多，需要先聚焦主线
- 已经执行过，但 verify / closeout 没闭环
- 任务状态和 evidence 不一致
- 需要把 planning / worker 行为收紧

### 不应使用的情况

以下做法是错误的：

- 把它当成新的 runtime
- 用它替代 route / dispatch / verify / gate
- 用它做宏大叙事而不落地执行
- 在执行阶段无限扩读，逃避真正落地

### 期望效果

这个 skill 的作用不是“做更多事”，而是收紧工作纪律：

- 先事实，后判断
- 先聚焦，后扩展
- 先实践，后宣布完成
- 先复盘，后收口

在当前 runtime 里，对应映射是：

- route：证据不足先调查
- packet synthesis：planner/judge 选择 bounded packet
- worker：读 ticket/spec 后尽快进入受控执行
- verify / handoff：要求 evidence、风险、未完成项都写清楚

### 当前优化点

- 保持它是 execution discipline，不是第二控制面
- 让 route reason codes 对它的激活条件更清楚
- 让 manifest / query 更显式暴露它已被激活
- 减少 skill 文本与 prompt spec 的重复定义

---

## 7. 当前 skill 到 runtime 的 canonical 映射

## 7.1 Route 信号层

关键文件：`internal/route/gate.go`

当前已经存在的高价值映射：

- `policy_bug_rca_first`
- `policy_options_before_plan`
- `policy_resume_state_first`
- `policy_verify_evidence_required`
- `policy_review_if_multi_file_or_high_risk`

这些 reason codes 是当前 skill / discipline 激活的最佳上游来源。

### 当前缺口

对 harness-specific 场景还不够细，例如：

- harness bootstrap / refresh / audit / agent-entry
- compact log first
- operator surface required
- worktree preferred

---

## 7.2 Orchestration 合同层

关键文件：`internal/orchestration/defaults.go`

当前已经存在的结构化合同：

- `MethodologyContract`
- `JudgeDecision`
- `ExecutionLoopContract`
- `ConstraintSystem`
- `PacketSynthesisLoop`

这里已经把 `qiushi-execution` 部分收进结构化合同。

### 当前缺口

- 还缺少更强的 harness-specific methodology lenses
- 还缺少更清楚的 route reason code -> selected flow / active lens 映射
- 还缺少面向 operator / log-search / harness-state 的补充 flow

---

## 7.3 Worker 协议注入层

关键文件：`internal/worker/manifest.go`

这个文件是 skill 改进最关键的落点。

当前它已经把这些东西写进 dispatch ticket / worker prompt：

- reason codes
- policy tags
- methodology
- judge decision
- execution loop
- constraints
- validation hooks
- learned reminders

### 当前缺口

- 还没有显式的 skill hint / active skill surface
- 对 `harness-log-search-cskill` 的 compact-first 纪律还没有完全固化到 prompt
- 对 `klein-harness` 的 harness-state-first / operator-surface discipline 还没显式注入

---

## 7.4 Operator 可见层

关键文件：`internal/query/service.go`

当前 query 已经能读到：

- planning
- accepted packet
- task contract
- assessment
- completion gate
- guard state
- outer-loop memory
- log preview

### 当前缺口

- 还没有明确的 skill hint / discipline hint 字段
- operator 还看不到“为什么当前任务推荐某个 skill”
- query 与 `harness-log-search-cskill` 的 compact-first 语义还没完全对齐

---

## 8. 当前最值得优化的点

## 8.1 先统一三个 skill 文档的结构

目标文件：

- `skills/klein-harness/SKILL.md`
- `skills/harness-log-search-cskill/SKILL.md`
- `skills/qiushi-execution/SKILL.md`

建议统一 section：

- This Skill Is For
- Use When
- Do Not Use When
- Expected Effects
- Canonical Runtime Mapping
- Minimal Read Order / Inputs
- Optimization Points
- Drift Risks

这样做的作用：

- 让 Codex / 人类更容易做正确选择
- 降低不同 skill 文本风格差异
- 把 drift 风险显式化

## 8.2 给 runtime 增加轻量 skill hint surface

目标文件：

- `internal/orchestration/defaults.go`
- `internal/worker/manifest.go`
- `internal/query/service.go`

目标不是新增控制面对象，而是增加：

- active skill hints
- activation reasons
- execution discipline hints

推荐来源：

- route `ReasonCodes`
- execution loop `SkillPath`
- methodology `ActiveLenses`

## 8.3 强化 route 到 skill 的映射

推荐先从已有 `policy_*` tag 出发：

- `policy_bug_rca_first` -> `systematic-debugging` + `qiushi-execution`
- `policy_resume_state_first` -> `harness-log-search-cskill` + `qiushi-execution`
- `policy_options_before_plan` -> `blueprint-architect`
- harness bootstrap / audit 类任务 -> `klein-harness`

## 8.4 把 compact-log-first 纪律从文档下沉到协议

这件事对 `harness-log-search-cskill` 最关键。

应该沉到：

- worker prompt
- query view
- route policy

而不是只留在 `SKILL.md` 里。

## 8.5 把 harness-state-first discipline 从 skill 文档下沉到 query / prompt

这件事对 `klein-harness` 最关键。

目标效果：

- harness task 先读 `.harness` 热状态
- 再读 execution plane
- 最后看 operator surface

而不是一上来就重写大量产物。

---

## 9. 什么情况下应该触发哪些效果

## 9.1 bug / failure / regression

推荐 discipline：

- `systematic-debugging`
- `qiushi-execution`

期望效果：

- route 打出 `policy_bug_rca_first`
- worker prompt 先要求 failure evidence
- verify 强调 evidence-first
- query 能看出当前走 debugging-first packet

## 9.2 resume / continue / handoff 恢复

推荐 discipline：

- `harness-log-search-cskill`
- `qiushi-execution`

期望效果：

- route 打出 `policy_resume_state_first`
- worker 先读 `AGENTS.md`、runtime/current/request-summary/session-registry、compact log
- operator 能看出当前是 state-first resume flow

## 9.3 harness bootstrap / refresh / audit / agent-entry

推荐 discipline：

- `klein-harness`

期望效果：

- 优先读 `.harness` control plane
- 先判断 harness 健康与状态漂移
- 先补 claim / handoff / operator surface，再谈大规模执行
- query 能看出当前任务走的是 harness-state-first discipline

## 9.4 compare / choose / recommendation

推荐 discipline：

- `blueprint-architect`

期望效果：

- route 打出 `policy_options_before_plan`
- planning 先给 2~3 个方案，再收敛一条主线
- 不直接跳进编码

---

## 10. 当前不建议做的事情

第一轮优化中，不建议做这些：

### 10.1 不要新增 skill runtime

不要让 skill 自己变成新的：

- scheduler
n- dispatch loop
- state ledger
- worker manager

### 10.2 不要让 repo-local `SKILL.md` 直接成为 runtime authority

短期内仍应保持：

- `SKILL.md` = 入口说明
- Go + prompts/spec = 执行真相

### 10.3 不要把所有 skill 文档全文塞进 prompt

正确做法是：

- 按需注入 path / hint / minimal read order
- 保持 prompt 收敛
- 避免 worker 无限扩读

---

## 11. 推荐优化顺序

### 第一批：文档与入口对齐

- 统一三个 skill 文档结构
- 写清使用条件、效果、优化点、drift 风险

### 第二批：最小 runtime hint

- 在 orchestration / manifest / query 中加轻量 skill hint
- 不改 runtime 拓扑

### 第三批：route -> skill 映射补强

- 为 harness-state / compact-log / operator-surface 增加更细信号

### 第四批：再评估 discovery

- 是否要让 repo-local skill 成为 discovery surface
- 这一步放到最后

---

## 12. 一句话结论

当前 harness skill 的正确优化方向不是“让 skill 长成第二 runtime”，而是：

**把 skill 从静态说明，逐步下沉成 route 的信号、orchestration 的合同、manifest 的执行提示、query 的可见解释面。**
