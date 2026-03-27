---
name: qiushi-execution
description: |
  触发：当任务需要把“先调查、再聚焦、再验证、再复盘”作为默认工作纪律时调用。适用于规划收敛、执行闭环、验证守门、复杂任务推进。该 skill 受 qiushi-skill 启发，但已按 Klein-Harness 的 route -> b3ehive -> tmux worker 结构改写。
---

# Qiushi Execution

这个 skill 为 Klein-Harness 提供一套轻量、可执行的方法纪律，不新增新的 runtime 实体，也不替代现有的 route、b3ehive packet synthesis、dispatch、worker、verify 链路。

目标只有一个：

- 让判断更贴事实
- 让规划更聚焦
- 让执行更闭环

# Use When

适用于：

- 任务复杂，但事实不足
- 方向很多，需要先收敛主线
- 已经执行过，但 verify / closeout 还没闭环
- 任务完成状态与 evidence 不一致
- 需要把 worker 行为收得更稳

# Do Not Use When

不适用于：

- 把它当成新的 runtime
- 用它替代 route / dispatch / verify / gate
- 用它做大段方法论叙事而不进入执行

# Expected Effects

使用这个 skill 后，Codex 应该：

- 调查优先，而不是凭经验直接改
- 一次只推进一个 bounded slice
- 用命令 / 文件 / artifact 证据证明完成
- 在 closeout 里诚实写清风险和未完成项

## 核心纪律

1. 先事实，后判断

- 在证据不足、上下文不清、仓库陌生时，先调查，再下结论。
- 不把猜测、经验或偏好当成事实。

2. 先聚焦，后扩展

- 同一轮只解决一个最有杠杆的核心问题。
- 不把多个目标揉成一个模糊的大任务。

3. 先实践，后宣布完成

- 方案必须经过执行、验证或可审计证据的检验。
- 没有 artifact、verify、日志，不算完成。

4. 先复盘，后收口

- 收口前必须明确本轮做了什么、验证了什么、还剩什么风险。
- 对失败和偏差要能诚实写进 handoff 或 review。

## 与 Klein 运行时的映射

### Route

- 证据不足时优先调查，不急于分发大任务。
- 在多个方向里选一个主攻方向，避免同时散开。
- 当存在速度、风险、范围、可验证性冲突时，优先选可验证、可回滚、边界清晰的路线。

### B3Ehive Packet Synthesis

- planner A 更关注边界和结构。
- planner B 更关注交付切片和顺序。
- planner C 更关注风险、验证和回滚。
- judge 不是做平均，而是选一个最适合当前仓库和当前证据的方案。

### Worker Execution

- 先读 dispatch ticket、worker-spec、planning trace。
- 然后尽快进入受控执行，不做无休止的二次规划。
- 每次改动都要朝验证闭环推进。

### Verify / Handoff

- verify 必须记录命令、结果、证据路径。
- handoff 必须说明已完成、未完成、风险和下一步。
- 如果 evidence 不完整，宁可阻断 closeout，也不要假完成。

## 适用信号

当出现这些信号时，优先遵循这套纪律：

- 任务复杂，但事实不足
- 方向很多，但注意力有限
- 执行已经发生，但验证还没闭环
- 任务完成状态和底层 evidence 不一致
- 需要把 planning 和 worker 行为收得更稳

## 禁止事项

- 不新增平行控制面
- 不把方法论做成新的 task ledger 实体
- 不用口头总结替代 verify 证据
- 不在执行阶段无限扩读文档来逃避落地

## 使用方式

把它当成 Klein 的工作纪律，而不是单独的 runtime。

一句话记忆：

`调查优先 -> 聚焦主线 -> 小步执行 -> 证据验证 -> 诚实复盘`

# Canonical Runtime Mapping

这份 `SKILL.md` 的真正 runtime 落点是：

- `internal/route/gate.go`
  - 通过 `policy_bug_rca_first`、`policy_resume_state_first` 等 reason code 激活不同 discipline
- `internal/orchestration/defaults.go`
  - 通过 `MethodologyContract`、`ExecutionLoopContract`、`ConstraintSystem` 结构化表达这套纪律
- `internal/worker/manifest.go`
  - 把这套纪律写进 dispatch ticket、worker-spec、worker prompt

因此这份 skill 文档是 **Codex 的入口说明**，不是 runtime authority。

## tmux / Hook Matching

如果当前执行发生在 tmux worker 里，优先使用统一前缀：

- `[harness:<task-id>] <node-task-description>`

后续 hook、日志检索和 closeout 复盘都应先匹配这个前缀，再结合 dispatch / slice 证据判断是否跑偏。

# Minimal Read Order / Inputs

当 Codex 以这份 skill 为当前执行纪律时，至少应先读：

1. dispatch ticket
2. worker-spec
3. accepted packet
4. task contract
5. planning trace
6. constraint snapshot
7. 若存在最近失败，再读 feedback summary 中当前 task 的局部失败记忆

# Optimization Points

- 保持它是 discipline，不是第二 runtime
- 让 route policy tags 更清楚地驱动这套 discipline
- 让 worker prompt 以简短硬约束表达，而不是长篇说明文
- 让 closeout / verify 与 evidence-first 原则完全对齐

# Drift Risks

出现以下情况时，说明 skill 已与 runtime 漂移：

- worker prompt 不再强调 investigate -> execute -> verify -> closeout
- verify 可以在 evidence 不足时默认通过
- planning trace 看不到 methodology lenses / execution loop
- 这份文档开始长出新的 runtime 阶段或新的控制面实体
