# Klein-Harness 复盘与 Skill 设计审查

本文基于本次架构梳理过程，对仓库内 `skills/` 设计做一次面向维护者的复盘。重点不是评价文风，而是判断这些 skill 作为“可执行协议”是否仍然和当前 runtime 架构一致，是否稳定、可组合、可持续演进。

## 1. 先说复盘结论

这次把项目完整架构文档写下来之后，有三个明显结论：

### 1.1 项目主架构已经收敛，但技能层没有完全跟上

当前仓库的 canonical runtime 已经很清楚：

- 主入口是 `cmd/harness`
- 主编排在 `internal/runtime`
- 主状态面在 `.harness/requests/queue.jsonl`、`.harness/task-pool.json`、`.harness/state/{runtime,dispatch,lease,verification,tmux,completion-gate,guard-state}.json`

这条链路在 README、`docs/runtime-mvp.md`、Go 实现和测试里基本一致。

但 `skills/` 里仍有部分内容大量引用旧的 `.harness` 面，例如：

- `.harness/state/progress.json`
- `.harness/spec.json`
- `.harness/work-items.json`
- `.harness/state/current.json`
- `.harness/state/request-summary.json`
- `.harness/state/log-index.json`
- `.harness/session-init.sh`
- `.harness/bin/harness-log-search`

这说明当前问题不是“skill 质量差”，而是“runtime 已切到一代新控制面，但技能层仍处于混合代际状态”。

### 1.2 skill 组合本身是有设计感的

从 portfolio 角度看，仓库里的 skill 并不是随便堆的：

- `blueprint-architect` 负责设计收敛
- `systematic-debugging` 负责 RCA-first 调试
- `markdown-fetch` 负责网页证据获取
- `generate-contributor-guide` 负责 repo 约定提炼
- `qiushi-execution` 提供执行纪律
- `klein-harness` 负责 `.harness` 协作系统
- `harness-log-search-cskill` 负责日志检索

也就是说，skill 的方向分层其实是对的：设计、执行、调试、研究、协作、日志，都有明确分工。

### 1.3 真正的风险在于“边界漂移”，不是“技能数量太多”

问题不是 skill 多，而是部分 skill 的输入面和当前 runtime 真相不再完全一致。

这会导致 agent 在两个层面上犯错：

- 读错状态源
- 走错命令面

一旦 skill 把 agent 引到旧状态文件或旧命令入口，后续行为即使很认真，也会在错误地基上推进。

## 2. 当前 skill 组合的设计模式

| Skill | 主模式 | 辅模式 | 结论 |
| --- | --- | --- | --- |
| `blueprint-architect` | Pipeline | Inversion + Tool Wrapper | 设计最稳，结构清晰，渐进加载做得好。 |
| `generate-contributor-guide` | Reviewer | Generator + Pipeline | 很适合做 repo-local 研究稿和热规范蒸馏。 |
| `markdown-fetch` | Tool Wrapper | Generator | 聚焦、短、命令面明确。 |
| `systematic-debugging` | Pipeline | Inversion | 很干净，退出条件明确。 |
| `qiushi-execution` | Inversion | Discipline Layer | 方向对，但操作性偏弱。 |
| `harness-log-search-cskill` | Tool Wrapper | Reviewer | 太依赖旧日志面，存在明显 schema drift。 |
| `klein-harness` | Pipeline | Generator + Reviewer + Inversion | 能力最强，但也最容易变成 God Skill。 |

总体判断：

- skill portfolio 的“职责拆分方向”是好的
- 结构最稳的是 `blueprint-architect`、`generate-contributor-guide`、`markdown-fetch`、`systematic-debugging`
- 风险最大的是 `klein-harness` 和 `harness-log-search-cskill`
- 最值得补强的是 `qiushi-execution`

## 3. 做得好的地方

### 3.1 `blueprint-architect` 是当前最成熟的 skill 之一

证据：

- 明确写清“负责什么 / 不负责什么”：[`skills/blueprint-architect/SKILL.md:29`](/Users/linzhenjie/code/claw-code/harness-architect/skills/blueprint-architect/SKILL.md:29)
- 先选模式，再进入工作流：[`skills/blueprint-architect/SKILL.md:49`](/Users/linzhenjie/code/claw-code/harness-architect/skills/blueprint-architect/SKILL.md:49)
- progressive disclosure 做得很克制：[`skills/blueprint-architect/SKILL.md:96`](/Users/linzhenjie/code/claw-code/harness-architect/skills/blueprint-architect/SKILL.md:96)
- recommendation 场景先 options 再 blueprint：[`skills/blueprint-architect/SKILL.md:137`](/Users/linzhenjie/code/claw-code/harness-architect/skills/blueprint-architect/SKILL.md:137)

为什么好：

- 触发条件精确
- 不会抢执行层职责
- 明确要求 repo-local scan
- 外部研究有 gate，不会默认深挖

### 3.2 `generate-contributor-guide` 的“完整稿 -> 蒸馏热规范”设计非常对

证据：

- 先选输出面，不默认双写：[`skills/generate-contributor-guide/SKILL.md:35`](/Users/linzhenjie/code/claw-code/harness-architect/skills/generate-contributor-guide/SKILL.md:35)
- 先研究稿，再决定要不要写进热路径：[`skills/generate-contributor-guide/SKILL.md:94`](/Users/linzhenjie/code/claw-code/harness-architect/skills/generate-contributor-guide/SKILL.md:94)
- 明确禁止凭空编规则：[`skills/generate-contributor-guide/SKILL.md:28`](/Users/linzhenjie/code/claw-code/harness-architect/skills/generate-contributor-guide/SKILL.md:28)

为什么好：

- 避免一上来污染 `.harness/standards.md`
- 很适合和当前 runtime 的 repo-local state 结合
- 对 maintainer 风格提炼这类任务，设计边界很稳

### 3.3 `markdown-fetch` 和 `systematic-debugging` 都是“短而稳”的技能

证据：

- `markdown-fetch` 的命令面和失败回退都很清楚：[`skills/markdown-fetch/SKILL.md:28`](/Users/linzhenjie/code/claw-code/harness-architect/skills/markdown-fetch/SKILL.md:28), [`skills/markdown-fetch/SKILL.md:162`](/Users/linzhenjie/code/claw-code/harness-architect/skills/markdown-fetch/SKILL.md:162)
- `systematic-debugging` 的单一假设和最小判别测试非常明确：[`skills/systematic-debugging/SKILL.md:58`](/Users/linzhenjie/code/claw-code/harness-architect/skills/systematic-debugging/SKILL.md:58), [`skills/systematic-debugging/SKILL.md:66`](/Users/linzhenjie/code/claw-code/harness-architect/skills/systematic-debugging/SKILL.md:66)

为什么好：

- 技能足够短
- 进入条件清楚
- 不依赖庞大资产
- 失败时不容易乱跳

这类 skill 可以作为整个 skill 体系的“短协议模板”。

## 4. 主要问题

下面这几项是本次审查里最重要的 finding。

### 4.1 `klein-harness` 与当前 runtime 已出现明显 schema drift

严重度：`fail`  
模式：`Schema Drift` + `God Skill`

关键证据：

- skill 把 `.harness/state/progress.json`、`.harness/work-items.json`、`.harness/spec.json`、`.harness/session-init.sh` 当作日常主读面：[`skills/klein-harness/SKILL.md:48`](/Users/linzhenjie/code/claw-code/harness-architect/skills/klein-harness/SKILL.md:48)
- skill 把 `.harness/state/current.json`、`request-summary.json`、`lineage-index.json`、`log-index.json` 当作热状态：[`skills/klein-harness/SKILL.md:61`](/Users/linzhenjie/code/claw-code/harness-architect/skills/klein-harness/SKILL.md:61)
- skill 的生成前 example 映射仍围绕 `features.json`、`work-items.json`、`spec.json`、`state/progress.json`：[`skills/klein-harness/SKILL.md:191`](/Users/linzhenjie/code/claw-code/harness-architect/skills/klein-harness/SKILL.md:191)
- skill 文件本体长达 1109 行：`wc -l` 结果见本次检查

与当前 runtime 的冲突证据：

- 当前 README 明确把 canonical 状态面定义为 `queue.jsonl`、`task-pool.json`、`dispatch-summary.json`、`lease-summary.json`、`runtime.json`、`verification-summary.json`、`tmux-summary.json`、`completion-gate.json`、`guard-state.json`：[`README.md:115`](/Users/linzhenjie/code/claw-code/harness-architect/README.md:115)
- `adapter.Resolve` 真实暴露的路径也是这套新面：[`internal/adapter/project.go:186`](/Users/linzhenjie/code/claw-code/harness-architect/internal/adapter/project.go:186)

影响：

- agent 容易先读到旧状态面，再据此做错误判断
- bootstrap / refresh / agent-entry 三种模式会围绕不同代际 schema 行动
- 这会让 skill 生成的 `.harness` 与当前 Go runtime 不同构

最小修复方向：

1. 明确声明这是“legacy harness schema”还是“runtime-mvp schema”
2. 如果要对齐当前仓库，应把主读面改为：
   - `.harness/requests/queue.jsonl`
   - `.harness/task-pool.json`
   - `.harness/state/runtime.json`
   - `.harness/state/dispatch-summary.json`
   - `.harness/state/lease-summary.json`
   - `.harness/state/verification-summary.json`
   - `.harness/state/tmux-summary.json`
   - `.harness/state/completion-gate.json`
   - `.harness/state/guard-state.json`
3. 把旧面移到 `legacy surfaces` 段，明确写成“仅兼容旧 harness 仓库”

### 4.2 `harness-log-search-cskill` 目前基本绑定旧日志面和旧命令面

严重度：`fail`  
模式：`Schema Drift`

关键证据：

- 检索顺序从 `.harness/state/current.json`、`request-summary.json`、`lineage-index.json`、`log-index.json` 开始：[`skills/harness-log-search-cskill/SKILL.md:16`](/Users/linzhenjie/code/claw-code/harness-architect/skills/harness-log-search-cskill/SKILL.md:16)
- 命令面依赖 `.harness/bin/harness-log-search` 和 `.harness/bin/harness-query`：[`skills/harness-log-search-cskill/SKILL.md:30`](/Users/linzhenjie/code/claw-code/harness-architect/skills/harness-log-search-cskill/SKILL.md:30)

与当前 runtime 的冲突：

- 当前 canonical CLI 是 `harness`，不是 `.harness/bin/*`：[`docs/runtime-mvp.md:9`](/Users/linzhenjie/code/claw-code/harness-architect/docs/runtime-mvp.md:9)
- 当前 query 面主要是 `harness task` / `harness control ... status`，而不是 skill 中列出的旧脚本

影响：

- 在新 runtime 仓库里，这个 skill 很可能一上来就读不到东西
- operator 被引导到错误命令面
- 日志检索会把“热状态不存在”误当成“没有证据”

最小修复方向：

1. 用 `harness task <ROOT> <TASK_ID>` 和 `harness control <ROOT> task <TASK_ID> status` 替换旧命令面
2. 将默认检索顺序改为：
   - `task-pool.json`
   - `runtime.json`
   - `dispatch-summary.json`
   - `lease-summary.json`
   - `verification-summary.json`
   - `tmux-summary.json`
   - `completion-gate.json`
   - `artifacts/<task>/<dispatch>/verify.json`
   - `artifacts/<task>/<dispatch>/handoff.md`
   - `logs/tmux/<task>/<dispatch>.log` 或 tmux log path
3. 只有在旧仓库检测到 `current.json` / `log-index.json` 时，再走 legacy 分支

### 4.3 `qiushi-execution` 方向正确，但更像价值宣言，不像可执行 skill

严重度：`warn`  
模式：`Inversion`，但 gate 不够硬

证据：

- skill 主要描述原则、映射和禁止事项：[`skills/qiushi-execution/SKILL.md:17`](/Users/linzhenjie/code/claw-code/harness-architect/skills/qiushi-execution/SKILL.md:17)
- 缺少显式输入、读取顺序、进入 gate、退出 gate、产物协议
- front matter 也没有 `allowed-tools`

影响：

- 作为团队方法纪律很好
- 作为执行技能时，agent 很难知道“现在先读哪些文件、做到什么算完成”
- 同一个 skill 会被不同 agent 理解成不同强度的流程约束

最小修复方向：

- 保留它的“纪律层”定位
- 但补一个最小操作协议，例如：
  - `Entry Gate`
  - `Read Order`
  - `Bounded Slice Rule`
  - `Evidence Gate`
  - `Closeout Checklist`
- 如果不想让它过重，可以把它写成被 `worker/manifest` prompt 和 `klein-harness` skill 共同引用的短规约

### 4.4 `klein-harness` 已经接近 God Skill

严重度：`warn`

证据：

- 同时承担 bootstrap、refresh、audit、agent-entry、operator UX、SOUL/persona 检查、OpenClaw 调度集成等多类职责：[`skills/klein-harness/SKILL.md:29`](/Users/linzhenjie/code/claw-code/harness-architect/skills/klein-harness/SKILL.md:29)
- 还绑定大量 reference 和 example：[`skills/klein-harness/SKILL.md:121`](/Users/linzhenjie/code/claw-code/harness-architect/skills/klein-harness/SKILL.md:121), [`skills/klein-harness/SKILL.md:191`](/Users/linzhenjie/code/claw-code/harness-architect/skills/klein-harness/SKILL.md:191)

影响：

- skill 很强，但误触发成本高
- 新 agent 很难快速判断“我该用这个 skill 的哪一部分”
- 一旦 schema 漂移，它会放大问题，因为它覆盖面太广

最小修复方向：

可以继续保留一个 umbrella skill，但建议内部明确拆成 3 个子模式：

- `klein-harness-bootstrap`
- `klein-harness-audit`
- `klein-harness-agent-entry`

即便不拆文件，也至少要在主文件最前面把模式边界再收紧。

## 5. 一个更重要的发现：不只是 skill 在漂移，runtime prompt 里也还有旧面引用

这次审查里最值得注意的一点是：旧 schema 不只存在于 skill 文档，也存在于当前 runtime 的 worker prompt 生成逻辑。

证据：

- `internal/worker/manifest.go` 在 resume flow 提示中，仍要求 worker 读取 `.harness/state/current.json` 和 `.harness/state/request-summary.json`：[`internal/worker/manifest.go:447`](/Users/linzhenjie/code/claw-code/harness-architect/internal/worker/manifest.go:447)

这意味着当前仓库存在两套同时活跃的心智模型：

- README / runtime-mvp / Go control plane 的新模型
- 部分 skill / docs / prompt 的旧模型

所以更准确的结论不是“某个 skill 写坏了”，而是：

> runtime schema migration 还没有完全收口，skill 只是这个问题最明显的外化层。

## 6. 推荐的改造方向

### 6.1 先做“schema version 对齐”，不要先修辞

建议给 `skills/klein-harness`、`skills/harness-log-search-cskill`，以及相关 references 增加一段非常明确的兼容声明：

- `runtimeCompatibility: legacy-harness-v0`
或
- `runtimeCompatibility: go-runtime-mvp-v1`

没有这层版本标签，后续所有 skill 审查都会继续混。

### 6.2 给 skill portfolio 加一张“技能与 runtime 层映射表”

建议新增一份短文档，例如：

- `docs/skill-runtime-mapping.zh.md`

里面只回答三件事：

- 每个 skill 对应哪个 runtime 层
- 每个 skill 默认读取哪些 authoritative surface
- 每个 skill 不能碰哪些层

这样能把“技能职责”从“经验”变成“契约”。

### 6.3 优先修复两个高风险 skill

优先级建议：

1. `skills/klein-harness/SKILL.md`
2. `skills/harness-log-search-cskill/SKILL.md`
3. `internal/worker/manifest.go` 的 resume flow 引导
4. `skills/qiushi-execution/SKILL.md`

理由：

- 这两份 skill 最容易把 agent 带到错误状态面
- `worker/manifest` 是 runtime 内部 prompt，影响更深
- `qiushi-execution` 不会直接带错状态，但会带来执行强度不一致

### 6.4 保留好 skill 的“短协议风格”

应该继续保留并推广的写法：

- `blueprint-architect` 的 progressive disclosure
- `generate-contributor-guide` 的“先完整稿、再蒸馏”
- `systematic-debugging` 的单一假设和退出条件
- `markdown-fetch` 的简洁命令面

这几个 skill 已经可以作为后续 skill 重写的模板。

## 7. 最终判断

这套 skill 设计总体不是失控，而是“方向正确、迁移未收口”。

最准确的判断是：

- portfolio 拆分方向是成熟的
- 几个通用 skill 已经具备生产级形状
- `klein-harness` 和 `harness-log-search-cskill` 有明显 schema drift
- `qiushi-execution` 还需要从理念层补到操作层
- 当前最该做的不是增加新 skill，而是统一 skill 与 runtime 的控制面版本

一句话总结：

> 现在 skill 的最大问题不是不够聪明，而是不够“与当前 runtime 同步”。

