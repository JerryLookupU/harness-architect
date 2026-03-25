---
name: generate-contributor-guide
description: "分析代码库真实约定、贡献模式和 maintainer 偏好，生成 repo-local contributor guide / research memo / distilled standards。适用于 onboarding、新仓库上手、对齐内部写法、提炼 Klein worker 可消费规范。"
allowed-tools: ["Bash", "Read", "Write", "Edit", "Glob", "Grep"]
---

# This Skill Is For

这个 skill 用于回答一个更具体的问题：

> 这个仓库的内部开发者到底怎么写代码、怎么组织改动、怎么验证和提交，而不是 README 表面上怎么说？

适合这些场景：

- 初次接手陌生仓库
- 外部贡献者要先对齐 maintainer 风格
- 项目已经有文档，但真实代码习惯散落在源码和提交历史里
- 需要把“贡献约定”沉淀成 Klein worker 可消费的热路径规范

# 这个 Skill 负责什么

- 识别技术栈和主要架构层
- 读取现有贡献文档、控制面文档和工程配置
- 抽样分析代表性代码文件，提炼真实约定
- 生成完整分析稿
- 在需要时把稳定规则蒸馏进 `.harness/standards.md` 或项目 `AGENTS.md`

# 这个 Skill 不负责什么

- 默认把唯一输出写死到 `.claude/contributor-guide.md`
- 把一堆泛化最佳实践伪装成“本仓库约定”
- 在证据不足时瞎编风格规则
- 无端重写整份 `AGENTS.md` 或 `.harness/standards.md`

# Preferred Output Surfaces

先选一个完整分析稿落点，不要默认双写：

- `.harness/research/contributor-guide.md`
  - 仓库已经有 `.harness/`
  - 这份分析主要给 orchestrator / worker / operator 消费
  - 更像 repo-local research memo
- `docs/contributor-guide.md`
  - 用户希望提交到仓库
  - 仓库更偏 docs-first
  - 需要给人类贡献者长期阅读

可选蒸馏输出：

- `.harness/standards.md`
  - 适合可执行、可验证、稳定的工程规则
- 项目根 `AGENTS.md` 的一个小型 managed section
  - 只适合短小的协作/执行规则
  - 不要把整份 contributor guide 倒进去

# Read Order

先读 repo-local 规则面，再读代码。

默认顺序：

1. `README.md`
2. 项目根 `AGENTS.md`（如果存在）
3. `.harness/standards.md`（如果存在）
4. `CONTRIBUTING.md` / `.github/CONTRIBUTING.md`
5. `.github/PULL_REQUEST_TEMPLATE.md` / issue templates
6. lint / format / test / build 配置
7. 最近 40 到 80 条 git history
8. 各层代表性源码和测试

如果仓库已经有 contributor guide：

- 先读它
- 再判断应该更新、补充，还是另写 research memo

# Progressive Disclosure

不要一次性把整个仓库读完。

先识别当前栈，再读取 [`references/analysis-checklist.md`](references/analysis-checklist.md) 里对应的部分。

只加载相关 section：

- repo-wide
- JS / TS backend
- React / frontend
- Python
- Go
- Ruby / Rails
- tests / CI / release

# Working Order

## 1. 先判断目标是“研究稿”还是“热规范”

默认先写完整分析稿，再决定要不要蒸馏。

不要上来就改：

- `.harness/standards.md`
- `AGENTS.md`

## 2. 检测技术栈与主要层

优先看：

- `package.json`
- `go.mod`
- `pyproject.toml`
- `Cargo.toml`
- `Gemfile`
- `pom.xml`
- `*.csproj`

然后识别主要层：

- models / entities
- handlers / controllers / API
- services / jobs / background work
- frontend components / state / styling
- tests
- CI / release / commit flow

如果某层文件很少，把它并入相邻层，不要机械分太细。

## 3. 读已有规则，不要只看源码

重点看：

- 官方贡献要求
- PR / issue 模板
- 代码格式化 / lint 配置
- 测试命令和 CI
- commit message 和 PR 粒度习惯

如果官方文档和真实代码习惯冲突，要写出来，不要强行美化成一致。

## 4. 每层抽样 4 到 8 个代表文件

目标不是覆盖所有文件，而是提炼高频模式。

每层至少回答：

- 命名和目录组织
- 入口和边界
- 常见结构模板
- 错误处理 / 状态处理 / 返回值约定
- 测试写法
- 与常见默认习惯不同的地方

需要检查什么，去读 checklist 对应段落，不要凭空发挥。

## 5. 生成完整分析稿

完整分析稿建议结构：

```markdown
# Contributor Guide

## Scope

## What Maintainers Actually Optimize For

## Existing Official Sources

## Repo-Wide Conventions

## Layer Patterns

## Testing and Verification Patterns

## Commit / PR Expectations

## Terminology Traps

## Recommended Distilled Rules
```

规则：

- 用仓库里的短代码片段或路径举例
- 重点写“这个仓库和常见默认不同的地方”
- 不要把 CONTRIBUTING 原文整段复述
- 证据不足时明确写“未观察到稳定约定”

## 6. 再决定要不要蒸馏进热路径

只有满足以下条件，才值得继续写热路径规则：

- 规则稳定
- 能指导后续 worker 决策
- 不是一次性观察
- 最好可验证

### 写入 `.harness/standards.md`

适合：

- 可执行的工程规则
- 可验证的测试/结构/接口规则
- 后续 worker 应长期遵守的开发约束

不要把纯描述性背景信息写进去。

### 写入 `AGENTS.md` managed section

适合：

- 贡献协作规则
- 术语禁忌
- “先读哪里、不要做什么”这类短规则

规则：

- 只改一个明确的 managed section
- 不覆盖用户已有其他规则
- 保持短小，可读，可复用

# What Good Output Looks Like

好的 contributor guide 不只是“总结代码结构”，而是能让后续执行者少踩坑：

- 知道 maintainer 更在意什么
- 知道哪些写法会被认为“不像这个仓库”
- 知道测试和提交粒度怎么对齐
- 知道哪些规则应该进入 `.harness/standards.md`
- 知道哪些只应保留在 research memo

# Important Rules

- 默认不要使用 `.claude/` 作为唯一落点
- 默认优先 repo-local：`.harness/research` -> `docs/` -> `.harness/standards.md` / `AGENTS.md`
- 如果项目已有 `.harness/standards.md`，优先补充和对齐，不要另造平行规范文件
- 如果项目已有 `AGENTS.md`，只做最小增量更新，不要整份重写
- 如果观察到模式分裂，先如实记录，不要替 maintainer “统一口径”
