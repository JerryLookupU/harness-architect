# harness-architect

## 中文

`harness-architect` 是一套用于建立和维护 `.harness/` 协作系统的 skill 与模板仓库。

当前版本主要面向 `Codex` 工作流，默认采用以下模型与命令约定：

- `codex exec`
- `codex exec resume`
- `gpt-5.4` 用于 orchestration、pre-worker routing、prompt refinement、replan
- `gpt-5.3-codex` 用于 worker execution

仓库同时保留了对 Claude 与其他 agent 工作流的兼容思路，但命令组织、session 路由、prompt 模板与 operator CLI 以 `Codex-first` 为默认设计中心。

### 项目概述

本仓库用于将 PRD、代码仓库与 agent 执行流程组织为一套可持续推进、可并发、可恢复、可审计的 `.harness` 执行链。

适用场景：

- 需要将一份 PRD 稳定落成项目
- 需要多个 agent 接力或并行推进
- 需要降低换模型、换会话、换执行者后的上下文恢复成本
- 需要让试用者快速上手并提供结构化反馈

### 仓库内容

本仓库主要包含三类内容：

- `SKILL.md`
  主说明书，定义模式、gate、角色、默认执行链与产物约束
- `references/`
  协议、路由、worktree、prompt、query、schema 等参考文档
- `examples/`
  可直接复用的模板、脚本、JSON、Markdown 与 prompt 示例

### 解决的问题

本仓库主要处理以下问题：

1. 长时间运行任务的状态恢复
2. 多 worker 并发时的路径冲突与编排漂移
3. `gpt-5.4` 与 `gpt-5.3-codex` 的模型分工
4. `session / worktree / diff / audit` 的闭环
5. 面向人类与工具的 operator / query 界面

### 快速开始

推荐试用流程：

1. 阅读 [SKILL.md](./SKILL.md)
2. 选择一个测试项目目录
3. 使用安装脚本将最小 CLI 与脚本写入该项目的 `.harness/`
4. 运行 `query` 与 `dashboard`
5. 根据项目需要接入 routing、worktree、audit

安装最小工具集：

```bash
./examples/harness-install-tools.example.sh <PROJECT_ROOT>
```

刷新热状态：

```bash
python3 .harness/scripts/refresh-state.py .
```

查看总览：

```bash
.harness/bin/harness-dashboard .
```

查看结构化查询：

```bash
.harness/bin/harness-query overview .
```

### 核心模型

- `gpt-5.4`
  负责 orchestration、pre-worker routing、prompt refinement、replan
- `gpt-5.3-codex`
  负责 worker execution
- `orchestrationSessionId`
  单写主线 session
- `worktree`
  代码隔离层
- `diff`
  审计与 merge 的证据层
- `state/*.json`
  面向机器热路径的状态层

可以将这套结构理解为：

- `session` 管理上下文
- `worktree` 管理代码隔离
- `diff` 管理证据
- `.harness` 管理状态

### 最小安装集

推荐的最小安装集如下：

- `.harness/bin/harness-query`
- `.harness/bin/harness-dashboard`
- `.harness/scripts/query.py`
- `.harness/scripts/refresh-state.py`
- `.harness/tooling-manifest.json`

### 最小热路径

建议工具优先读取：

- `.harness/state/current.json`
- `.harness/state/runtime.json`
- `.harness/state/blueprint-index.json`

建议人工优先阅读：

- `.harness/progress.md`
- `.harness/work-items.json`
- `.harness/task-pool.json`
- `.harness/spec.json`

每轮 orchestration / daemon / session 结束后，建议刷新热状态：

```bash
python3 .harness/scripts/refresh-state.py .
```

### 常用 CLI

机器可读查询：

```bash
.harness/bin/harness-query overview .
.harness/bin/harness-query progress .
.harness/bin/harness-query current .
.harness/bin/harness-query blueprint .
.harness/bin/harness-query task . T-004
```

人类可读面板：

```bash
.harness/bin/harness-dashboard .
.harness/bin/harness-dashboard . T-004
.harness/bin/harness-dashboard . T-004 --watch 2
```

### 典型执行链

```text
session-init
-> gpt-5.4 orchestration
-> pre-worker routing
-> gpt-5.3-codex worker
-> audit worker
-> merge / replan / stop
-> refresh-state
```

### 推荐阅读

优先阅读：

- [SKILL.md](./SKILL.md)
- [TRY-IT.md](./TRY-IT.md)
- [FEEDBACK.md](./FEEDBACK.md)
- [references/schema-contracts.md](./references/schema-contracts.md)
- [references/openclaw-dispatch.md](./references/openclaw-dispatch.md)
- [references/model-routing.md](./references/model-routing.md)

阅读顺序建议：

1. `SKILL.md`
2. `references/schema-contracts.md`
3. `references/openclaw-dispatch.md`
4. `references/model-routing.md`
5. `references/git-worktree-playbook.md`
6. `examples/`

### 试用与反馈

建议在试用前先阅读：

- [TRY-IT.md](./TRY-IT.md)
- [FEEDBACK.md](./FEEDBACK.md)

建议重点反馈：

- 哪个环节最难理解
- 哪些文档过长
- 哪些字段命名不够直观
- 哪个脚本最先失效
- 弱模型最容易在哪一步偏离
- 并发、session、worktree 的心智成本是否偏高

### 仓库定位

本仓库提供的是一套可安装的 `.harness` 协作骨架，以及 Codex-first 的任务编排、执行、审计、状态管理与查询工具组合。

### 许可证

本仓库采用 [MIT License](./LICENSE)。

---

## English

`harness-architect` is a skill and template repository for building and maintaining a `.harness/` coordination system.

The current version is primarily designed for `Codex` workflows, with the following default model and command assumptions:

- `codex exec`
- `codex exec resume`
- `gpt-5.4` for orchestration, pre-worker routing, prompt refinement, and replanning
- `gpt-5.3-codex` for worker execution

The repository also keeps a compatibility path for Claude and other agent workflows, but its command layout, session routing, prompt templates, and operator CLI are organized around a `Codex-first` model.

### Overview

This repository packages PRD-driven planning, repository state, and agent execution into a `.harness` workflow that supports long-running execution, concurrency, recovery, and auditability.

Typical use cases:

- turning a PRD into a real project plan and execution flow
- coordinating multiple agents in sequence or in parallel
- reducing context recovery cost across model, session, or operator changes
- enabling external testers to try the system and submit structured feedback

### Repository Contents

This repository is organized around three main parts:

- `SKILL.md`
  the main specification for modes, gates, roles, execution flow, and output contracts
- `references/`
  protocol, routing, worktree, prompt, query, and schema references
- `examples/`
  reusable templates, scripts, JSON files, Markdown files, and prompt examples

### Problems It Addresses

The repository focuses on these areas:

1. state recovery for long-running tasks
2. path conflicts and orchestration drift under multi-worker concurrency
3. model division between `gpt-5.4` and `gpt-5.3-codex`
4. end-to-end closure across `session / worktree / diff / audit`
5. operator and query surfaces for both humans and tools

### Quick Start

Recommended trial flow:

1. read [SKILL.md](./SKILL.md)
2. choose a test project directory
3. install the minimal CLI and scripts into that project's `.harness/`
4. run `query` and `dashboard`
5. add routing, worktree, and audit components as needed

Install the minimal toolset:

```bash
./examples/harness-install-tools.example.sh <PROJECT_ROOT>
```

Refresh hot state:

```bash
python3 .harness/scripts/refresh-state.py .
```

Open the dashboard:

```bash
.harness/bin/harness-dashboard .
```

Run a structured query:

```bash
.harness/bin/harness-query overview .
```

### Core Model

- `gpt-5.4`
  handles orchestration, pre-worker routing, prompt refinement, and replanning
- `gpt-5.3-codex`
  handles worker execution
- `orchestrationSessionId`
  the single-writer orchestration session
- `worktree`
  the code isolation layer
- `diff`
  the evidence layer for audit and merge
- `state/*.json`
  the machine-oriented hot-path state layer

This structure can be read as:

- `session` manages context
- `worktree` manages code isolation
- `diff` manages evidence
- `.harness` manages state

### Minimal Install Set

The recommended minimal install set includes:

- `.harness/bin/harness-query`
- `.harness/bin/harness-dashboard`
- `.harness/scripts/query.py`
- `.harness/scripts/refresh-state.py`
- `.harness/tooling-manifest.json`

### Minimal Hot Path

Tools should prefer:

- `.harness/state/current.json`
- `.harness/state/runtime.json`
- `.harness/state/blueprint-index.json`

Humans should usually start with:

- `.harness/progress.md`
- `.harness/work-items.json`
- `.harness/task-pool.json`
- `.harness/spec.json`

After each orchestration / daemon / session round, refresh hot state:

```bash
python3 .harness/scripts/refresh-state.py .
```

### Common CLI

Machine-readable queries:

```bash
.harness/bin/harness-query overview .
.harness/bin/harness-query progress .
.harness/bin/harness-query current .
.harness/bin/harness-query blueprint .
.harness/bin/harness-query task . T-004
```

Human-readable dashboard:

```bash
.harness/bin/harness-dashboard .
.harness/bin/harness-dashboard . T-004
.harness/bin/harness-dashboard . T-004 --watch 2
```

### Typical Execution Chain

```text
session-init
-> gpt-5.4 orchestration
-> pre-worker routing
-> gpt-5.3-codex worker
-> audit worker
-> merge / replan / stop
-> refresh-state
```

### Recommended Reading

Suggested starting points:

- [SKILL.md](./SKILL.md)
- [TRY-IT.md](./TRY-IT.md)
- [FEEDBACK.md](./FEEDBACK.md)
- [references/schema-contracts.md](./references/schema-contracts.md)
- [references/openclaw-dispatch.md](./references/openclaw-dispatch.md)
- [references/model-routing.md](./references/model-routing.md)

Suggested reading order:

1. `SKILL.md`
2. `references/schema-contracts.md`
3. `references/openclaw-dispatch.md`
4. `references/model-routing.md`
5. `references/git-worktree-playbook.md`
6. `examples/`

### Trial and Feedback

Before running a trial, review:

- [TRY-IT.md](./TRY-IT.md)
- [FEEDBACK.md](./FEEDBACK.md)

Useful feedback areas include:

- which step is hardest to understand
- which documents are too long
- which field names are unclear
- which script fails first
- where weaker models drift most often
- whether concurrency, session, and worktree management feel too heavy

### Repository Positioning

This repository provides an installable `.harness` coordination skeleton, together with a Codex-first set of patterns for planning, execution, audit, state management, and operator-facing tooling.

### License

This repository is licensed under the [MIT License](./LICENSE).
