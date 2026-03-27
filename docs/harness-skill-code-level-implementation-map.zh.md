# Harness Skill 代码级实现地图

## 1. 文档目标

这份文档不是抽象讨论 skill，而是把 **harness skill 的改进方案继续拆到代码级**。

目标是回答五个问题：

1. 当前 skill 的 runtime 落点在哪里
2. 每个关键文件分别承担什么职责
3. 如果要优化这些 skill，应该改哪些代码点
4. 改完之后用户能观察到什么效果
5. 每一步怎么验证，怎么控制风险

本文关注的不是“文档更完整”，而是：

- route 更稳定
- worker prompt 更贴任务类型
- query 更能解释当前状态
- operator 与后续 agent 接力成本更低

---

## 2. 当前代码架构基线

当前与 harness skill 最相关的 canonical 代码层有五个：

### 2.1 `internal/route/gate.go`

职责：

- 根据 task 状态与输入信号决定 `dispatch / resume / replan / block`
- 生成 `ReasonCodes`
- 通过 `policyReasonCodes(...)` 生成 `policy_*` tag

意义：

- 这是 skill / discipline 进入 runtime 的第一层信号源

### 2.2 `internal/orchestration/defaults.go`

职责：

- 定义 packet synthesis 默认合同
- 定义 methodology contract
- 定义 judge decision
- 定义 execution loop contract
- 定义 layered constraint system
- 生成 planning trace

意义：

- 这是 skill / discipline 被结构化表达成 runtime contract 的地方

### 2.3 `internal/worker/manifest.go`

职责：

- 生成 dispatch ticket
- 生成 worker-spec
- 生成 accepted packet / task contract
- 生成 worker prompt

意义：

- 这是 Codex 真正“看到 skill 影响”的地方

### 2.4 `internal/query/service.go`

职责：

- 聚合 task / planning / dispatch / gate / verify / logs / packet / contract
- 形成 operator 的 read model

意义：

- 这�� skill 改进是否可见、是否可解释的出口

### 2.5 `internal/instructions/discovery.go`

职责：

- 发现全局/项目/嵌套 `AGENTS.md`

意义：

- 这是 worker 实际 instruction surface 的上游
- 当前并不发现 repo-local `skills/*/SKILL.md`

---

## 3. 与 skill 直接相关的关键文件地图

| 文件 | 当前职责 | 与 skill 的关系 | 是否建议第一轮修改 |
| --- | --- | --- | --- |
| `internal/route/gate.go` | route + reasonCodes + policy tags | skill 激活信号源 | 是 |
| `internal/orchestration/defaults.go` | methodology / execution loop / constraints / planning trace | skill 协议组装层 | 是 |
| `internal/worker/manifest.go` | dispatch ticket / worker-spec / prompt | Codex 实际执行入口 | 是 |
| `internal/query/service.go` | task/planning/query read model | operator skill explanation surface | 是 |
| `internal/instructions/discovery.go` | 发现 AGENTS 指令文件 | instruction surface 上游 | 后评估 |
| `skills/klein-harness/SKILL.md` | harness 入口说明 | 入口说明面 | 是 |
| `skills/harness-log-search-cskill/SKILL.md` | compact log 检索说明 | 入口说明面 | 是 |
| `skills/qiushi-execution/SKILL.md` | execution discipline 说明 | 入口说明面 | 是 |
| `prompts/spec/README.md` | runtime prompt role split | canonical boundary doc | 参考对齐 |
| `README.md` | canonical runtime 和 guardrail mapping | 参考对齐 | 视需要 |
| `internal/worker/manifest_test.go` | manifest/prompt 断言测试 | 核心回归入口 | 是 |

---

## 4. 文件级拆解：`internal/route/gate.go`

## 4.1 当前职责

当前 `Evaluate(input Input) Decision` 已经能处理：

- `plan_epoch_stale`
- `checkpoint_required`
- `worktree_missing`
- `owned_paths_missing`
- `resume_session_contested`
- `resume`
- `dispatch`

同时，`policyReasonCodes(...)` 已经能识别：

- `policy_bug_rca_first`
- `policy_options_before_plan`
- `policy_read_only_intake`
- `policy_thread_reuse`
- `policy_smallest_pending_slice_first`
- `policy_resume_state_first`
- `policy_verify_evidence_required`
- `policy_review_if_multi_file_or_high_risk`

这说明 route 层已经是一个天然的 skill activation layer。

## 4.2 当前不足

对于 harness-specific skill 场景，route 还不够细：

- 还缺少 harness bootstrap / refresh / audit / agent-entry 的显式信号
- 还缺少 compact-log-first 的显式信号
- 还缺少 operator-surface-required 的显式信号
- 还缺少 worktree-preferred 之类更贴近 harness 的信号

## 4.3 建议代码级改动

### [code-task:S01 扩展 policy tag taxonomy]

建议新增或预留：

- `policy_harness_state_first`
- `policy_log_compact_first`
- `policy_operator_surface_required`
- `policy_worktree_preferred`

### [code-task:S02 扩展 signal 输入维度]

可考虑为 `Input` 增补：

- `UserGoal string`
- `HasHarnessState bool`
- `NeedsOperatorSurface bool`
- `HighRiskSurface bool`

第一轮可以先不全部用，但需要预留接口空间。

### [code-task:S03 为 harness-specific intent 增加匹配逻辑]

建议在 `policyReasonCodes(...)` 里识别以下表达：

- bootstrap / init / refresh harness
- audit harness / inspect harness health
- log search / verify logs / handoff logs / runner logs
- dashboard / overview / watch / metrics / forever

## 4.4 改完后的可观察效果

- route 能区分普通实现任务和 harness 调整任务
- route 能区分普通 resume 和 log-recovery / compact evidence retrieval
- 下游 worker prompt 能更精准地加载读取顺序与执行纪律

## 4.5 风险

- 误判词表扩大
- policy tag 过多导致解释成本上升
- 多个 tag 并存时可能相互干扰

## 4.6 验证

建议补充 `internal/route/gate_test.go`：

- harness bootstrap 场景
- harness audit 场景
- log-search 场景
- dashboard/operator surface 场景
- resume + compact-log-first 场景

---

## 5. 文件级拆解：`internal/orchestration/defaults.go`

## 5.1 当前职责

当前该文件定义了：

- `PacketSynthesisLoop`
- `MethodologyContract`
- `JudgeDecision`
- `ExecutionLoopContract`
- `ConstraintSystem`
- `PromptRefs(...)`
- `RenderPlanningTrace(...)`

其中最关键的 skill 相关点：

- `DefaultMethodologyContract(...)`
- `methodologyLenses(...)`
- `DefaultJudgeDecision(...)`
- `DefaultExecutionLoopContract(...)`
- `selectedFlow(...)`

## 5.2 当前优势

已经把 `qiushi-execution` 下沉成结构化合同：

- execution loop 有 `SkillPath`
- methodology 已可根据 reason codes 启动不同 lens
- judge rationale 会受到 policy tag 影响
- planning trace 会把这些信息显式展示出来

## 5.3 当前不足

- `klein-harness` 还没有被结构化成 methodology lens
- `harness-log-search-cskill` 还没有被转成显式 log retrieval discipline
- `selectedFlow(...)` 还缺少 harness-specific flow
- skill 到 planning trace 的映射还不够完整

## 5.4 建议代码级改动

### [code-task:S04 给 methodology 增加 harness lenses]

建议在 `methodologyLenses(...)` 增加：

- `harness-state-first`
- `compact-log-first`
- `operator-surface-first`
- `claim-before-edit`

### [code-task:S05 扩展 selectedFlow]

基于新增 reason codes 增加 flow，例如：

- `harness-state-first packet`
- `compact-log-investigation packet`
- `operator-surface packet`

### [code-task:S06 在 execution loop / constraints 中加入 log discipline]

建议把以下约束写进 execution/verification constraints：

- 先 compact logs / state index
- raw logs 仅 detail fallback
- 日志证据要返回 window / refs，不回贴整段 transcript

### [code-task:S07 在 PromptRefs 或 contracts 中补 skill refs]

建议在 contract 中保留：

- `executionSkillPath`
- `harnessSkillPath`
- `logSearchSkillPath`

这样 planning trace / query 更容易解释当前 discipline。

## 5.5 改完后的可观察效果

- planning trace 能更准确解释当前任务为什么走特定 discipline
- harness-specific 场景不再强行塞进 debugging/resume/general 三种流
- operator 看 trace 时能看到 skill 与 flow 的明确对应关系

## 5.6 风险

- lens 太多会稀释主线
- contracts 变长，planning trace 变重
- 如果命名不稳定，容易 drift

## 5.7 验证

建议补 orchestration 测试：

- `methodologyLenses(...)` 输出断言
- `selectedFlow(...)` 断言
- `DefaultExecutionLoopContract(...)` 断言
- `RenderPlanningTrace(...)` 快照断言

---

## 6. 文件级拆解：`internal/worker/manifest.go`

## 6.1 当前职责

`Prepare(...)` 负责生成：

- dispatch ticket
- worker-spec
- accepted packet
- task contract
- planning trace
- runner prompt

`buildPrompt(...)` 负责把 runtime 合同真正转成 worker 可执行协议。

## 6.2 当前优势

当前 prompt 已经具备：

- required reads
- hard authority rules
- execution defaults
- visible orchestration layer
- visible execution / validation loop
- soft / hard constraints
- policy guardrails
- hookified verification flow
- required artifacts before exit

这说明当前 `manifest.go` 是承接 skill 细化的最佳落点。

## 6.3 当前不足

- 还没有显式 `skillHints` / `activeSkills` surface
- `harness-log-search-cskill` 的 compact-first discipline 还没完全下沉
- `klein-harness` 的 harness-state-first 三层理解法还没进入 prompt
- operator surface 类诉求还没被显式注入 closeout 要求
- instruction discovery 结果还没有显式挂进 dispatch/runtimeRefs

## 6.4 建议代码级改动

### [code-task:S08 在 dispatch ticket / worker-spec 中增加 skill hints]

建议新增轻量字段：

- `activeSkills`
- `skillHints`
- `activationReasons`

来源：

- `ticket.ReasonCodes`
- `policyTags(...)`
- `executionLoop.SkillPath`
- methodology lenses

### [code-task:S09 在 buildPrompt 中补 harness-state-first 读取顺序]

当命中 `policy_harness_state_first` 时，建议在 Required reads 中增加：

- `.harness/state/progress.json`
- `.harness/task-pool.json`
- `.harness/session-registry.json`
- 相关 control-plane summaries

并明确三层理解顺序：

- control plane
- execution plane
- operator plane

### [code-task:S10 在 buildPrompt 中补 compact-log-first 规则]

当命中 `policy_log_compact_first` 或 `policy_resume_state_first` 时：

优先读：

- `.harness/state/current.json`
- `.harness/state/runtime.json`
- `.harness/state/request-summary.json`
- `.harness/state/lineage-index.json`
- `.harness/state/log-index.json`
- `.harness/log-<taskId>.md`

并明确：

- 不要默认扫全量 raw logs
- raw runner logs 仅用于 detail fallback

### [code-task:S11 在 buildPrompt 中补 operator surface closeout 要求]

当命中 `policy_operator_surface_required` 时，要求 closeout / handoff 明确说明：

- overview 入口
- watch 入口
- metrics snapshot
- README operator command 是否已更新

### [code-task:S12 在 runtimeRefs 中加入 discovered instruction refs]

建议在 dispatch ticket `runtimeRefs` 中补：

- `instructionFiles`
- `instructionScopes`
- `agentsGuide`

### [code-task:S13 控制 prompt 膨胀]

所有新增内容必须按 policy tag 条件注入：

- 不做全量默认注入
- 只注 path / minimal read order / short rule
- 保持弱模型可读性

## 6.5 改完后的可观察效果

- harness task 会优先理解 control/execution/operator 三层，而不是先乱改
- log-search / resume task 会先走 compact logs
- operator surface 需求会更稳定地产出 watch/overview/metrics 等结果
- worker 更少走偏路

## 6.6 风险

- prompt 变长
- 过多 references 导致 worker 扩读
- JSON 字段扩张后测试需要同步更新

## 6.7 验证

重点补 `internal/worker/manifest_test.go`：

- harness-state-first case
- compact-log-first case
- operator-surface-required case
- activeSkills / skillHints 字段断言
- prompt 关键句存在性断言

---

## 7. 文件级拆解：`internal/query/service.go`

## 7.1 当前职责

当前 `TaskView` / `PlanningView` 已经聚合：

- dispatch / lease / tmux
- accepted packet / packet progress / task contract
- assessment / gate / guard
- intake / thread / change / todo summaries
- outer-loop memory
- log preview
- release readiness

## 7.2 当前不足

- operator 还看不到 active skill / discipline hint
- 看不到 route policy summary
- 看不到 instruction discovery 结果
- log preview 还是单一 tail 视角，没有 compact-first 语义

## 7.3 建议代码级改动

### [code-task:S14 在 TaskView / PlanningView 增加 skill hint 字段]

建议增加：

- `ActiveSkills []string`
- `SkillHints []string`
- `RoutePolicyTags []string`
- `RouteReasonCodes []string`
- `ExecutionDiscipline string`

### [code-task:S15 暴露 instruction discovery 结果]

建议增加：

- `InstructionFiles []string`
- 或更结构化的 instruction preview

### [code-task:S16 扩展 log preview 语义]

建议优先支持：

- compact log preview
- verify preview
- handoff preview
- planning trace preview

而不是只 tail tmux raw log。

### [code-task:S17 给 operator 输出 next action + why this skill]

结合现有：

- `ReleaseReadiness.NextAction`
- route reason codes
- methodology / execution loop

让 operator 看到：

- 为什么当前推荐这个 discipline
- 下一步该做什么

## 7.4 改完后的可观察效果

- `harness task` 更像解释面，而不是状态堆叠
- operator 可以直接看出当前走什么 skill / discipline
- resume / log-search / audit 场景更容易接力

## 7.5 风险

- JSON view 结构扩张
- 读更多 artifact 增加 I/O
- preview 太多会影响可读性

## 7.6 验证

建议为 query 增加：

- TaskView JSON 断言
- planning/task snapshot
- 有 policy tags / instructions / logs / assessment 的组合场景测试

---

## 8. 文件级拆解：`internal/instructions/discovery.go`

## 8.1 当前职责

当前只发现：

- `$CODEX_HOME/AGENTS.override.md`
- `$CODEX_HOME/AGENTS.md`
- repo root 到当前目录链上的 `AGENTS.override.md` / `AGENTS.md`

## 8.2 当前不足

- 结果还没有显式进入 query / manifest / prompt 的解释层
- repo-local `skills/*/SKILL.md` 目前不是 discovery 面
- harness-managed guidance 还没有稳定接入点

## 8.3 建议代码级改动

### [code-task:S18 增加 instruction source 分类]

建议给 `File` 增加：

- `Kind string`
- `Priority int`

### [code-task:S19 补 summarize / worker-friendly 输出]

建议新增 summary helper：

- `DiscoverForWorker(...)`
- `Summarize(...)`

### [code-task:S20 是否支持 repo-local skill discovery：放到后续评估]

当前不建议第一轮就改。

原因：

- 这会改变 instruction precedence
- 风险大于当前收益
- 第一轮先做 runtime hint surface 更稳

## 8.4 改完后的可观察效果

- worker / operator 更容易理解当前 instruction 来源
- AGENTS 与 skill 协议的关系更可见
- resume / audit 调试成本更低

## 8.5 风险

- 引入过多文件到上下文
- 优先级混乱
- 如果过早引入 repo-local skill discovery，可能打乱现有 instruction model

## 8.6 验证

- discovery 表驱动测试
- summarize 输出断言
- query / manifest 集成断言

---

## 9. 文档与代码对齐表

| 文档 | 应映射到的代码层 | 对齐重点 |
| --- | --- | --- |
| `skills/klein-harness/SKILL.md` | `internal/route/gate.go`, `internal/worker/manifest.go`, `internal/query/service.go` | harness-state-first、三层理解法、operator surface |
| `skills/harness-log-search-cskill/SKILL.md` | `internal/route/gate.go`, `internal/worker/manifest.go`, `internal/query/service.go` | compact logs first、raw fallback、evidence refs |
| `skills/qiushi-execution/SKILL.md` | `internal/orchestration/defaults.go`, `internal/worker/manifest.go` | fact-first / focus-first / verify-first 的结构化合同 |
| `prompts/spec/README.md` | `internal/orchestration/defaults.go` | methodology 不是第二 runtime |
| `README.md` | `internal/route/gate.go`, `internal/worker/manifest.go`, `internal/query/service.go` | guardrail mapping、canonical runtime surface |

---

## 10. 第一轮最值得做的代码任务

### 优先级 P1：最短闭环

1. `S01` 扩展 route policy tags
2. `S08` 在 manifest 增加 activeSkills / skillHints
3. `S09` 在 prompt 注入 harness-state-first
4. `S10` 在 prompt 注入 compact-log-first
5. `S14` 在 query 暴露 active skills / policy tags
6. `S17` 在 query 暴露 why this skill / next action
7. 更新 `internal/worker/manifest_test.go`

### 优先级 P2：合同层补强

8. `S04` methodology harness lenses
9. `S05` selectedFlow 扩展
10. `S06` log discipline constraints
11. `S07` skill refs contract 化

### 优先级 P3：instruction 与 discovery 评估

12. `S18` instruction source 分类
13. `S19` summarize helper
14. `S20` repo-local skill discovery 评估

---

## 11. 分阶段验证路线

## 11.1 Route 验证

验证：

- bug / failure -> `policy_bug_rca_first`
- resume -> `policy_resume_state_first`
- harness audit -> `policy_harness_state_first`
- log search -> `policy_log_compact_first`

## 11.2 Manifest / Prompt 验证

验证：

- dispatch ticket 包含 activeSkills / skillHints
- prompt 出现正确的 required reads
- prompt 对 compact logs / raw fallback 的指引正确
- prompt 不发生无控制膨胀

## 11.3 Query 验证

验证：

- task view 可显示当前 discipline
- task view 可显示 route policy tags
- task view 可显示 why this skill / next action
- log preview 更贴近 compact-first 语义

## 11.4 End-to-end 验证

建议至少覆盖三类真实场景：

1. bug / regression
   - 预期：debugging-first + qiushi-execution
2. resume / handoff 恢复
   - 预期：harness-log-search-cskill + qiushi-execution
3. harness audit / refresh
   - 预期：klein-harness + harness-state-first discipline

---

## 12. 一句话结论

如果要把 harness skill 真正做好，关键不是继续堆 `SKILL.md`，而是：

**把 skill 的核心纪律下沉到 route 的信号、orchestration 的合同、worker manifest 的 prompt、query 的解释面里。**
