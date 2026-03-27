---
name: harness-log-search-cskill
description: "搜索 Klein-Harness 的 compact handoff logs，并在需要时回落到 raw runner logs 的局部证据窗口。适用于跨 worker handoff、operator 调试、定向日志检索。"
allowed-tools: ["Bash", "Read", "Grep"]
---

# This Skill Is For

这个 skill 用于在不广播完整 worker transcript 的前提下，搜索 `.harness` 日志面。

默认目标：

- 先读热路径状态和 compact logs
- 只在 `--detail` 或证据不足时回落到 raw runner logs

# Use When

适用于：

- resume 前需要恢复 handoff / verify / recent failure 上下文
- 要做 operator debug 或 RCA
- 要围绕 verify、handoff、runtime state 做定向日志检索
- 需要给 Codex 一个“先 compact、后 raw fallback”的日志读取纪律

# Do Not Use When

不适用于：

- 普通实现任务一开始就扫日志
- 没有明确日志诉求时默认扫描 raw runner logs
- 为了省事贴整段 transcript 给下游 agent

# Expected Effects

使用这个 skill 后，Codex 应该：

- 先读热状态与 compact logs
- 把日志检索结果压缩成 one-screen summary
- 返回 blockers / risks / verification notes / evidence refs
- raw logs 仅返回相关窗口，不返回整段 transcript

# Retrieval Order

默认读取顺序：

1. `.harness/state/current.json`
2. `.harness/state/runtime.json`
3. `.harness/state/request-summary.json`
4. `.harness/state/lineage-index.json`
5. `.harness/state/log-index.json`
6. `.harness/log-<taskId>.md`
7. `.harness/state/runner-logs/<taskId>.log` 仅用于定向细节

不要默认把所有 raw logs 都扫一遍。

# Preferred Command Surface

优先使用：

```bash
.harness/bin/harness-log-search . --task-id T-003
.harness/bin/harness-log-search . --keyword verify --detail
.harness/bin/harness-query logs . --text
.harness/bin/harness-query log . T-003 --detail --text
```

# What To Return

优先返回：

- one-screen summary
- cross-worker relevant facts
- blockers / risks
- verification notes
- evidence refs

如果需要 raw evidence，只返回相关窗口，不要贴完整 transcript。

# Canonical Runtime Mapping

这个 skill 的真正落点不在这里，而在：

- `internal/route/gate.go`
  - 负责把 resume / log-search / evidence retrieval 信号变成 `policy_*` tags
- `internal/orchestration/defaults.go`
  - 负责把 compact-log-first 纪律写入 methodology / execution loop / constraints
- `internal/worker/manifest.go`
  - 负责把日志读取顺序和 raw fallback 规则注入给 Codex

这份 `SKILL.md` 是给 Codex 的入口说明，不是 runtime authority。

# Minimal Read Order / Inputs

Codex 在真正执行前，最小读取面应优先是：

1. `.harness/state/current.json`
2. `.harness/state/runtime.json`
3. `.harness/state/request-summary.json`
4. `.harness/state/lineage-index.json`
5. `.harness/state/log-index.json`
6. `.harness/log-<taskId>.md`
7. 只在 detail fallback 时读 `.harness/state/runner-logs/<taskId>.log`

# Optimization Points

- 把 compact-first / raw-fallback 纪律沉到 worker prompt，而不只留在 skill 文本里
- 让 route 能打出更清楚的 log-retrieval policy tags
- 让 query / compact preview 更贴近这个读取顺序
- 让 Codex 在 resume 场景下优先用 compact evidence 恢复上下文

# Drift Risks

出现以下情况时，说明这份 skill 文档已经与 runtime 漂移：

- worker prompt 没有强调 compact logs 优先
- query 仍默认偏 raw log tail，而不是 compact surface
- route 无法区分普通 resume 和日志恢复型 resume
- operator 依然需要手工扫大段 transcript 才能知道结论
