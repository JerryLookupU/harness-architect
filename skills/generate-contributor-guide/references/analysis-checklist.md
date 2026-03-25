# Contributor Guide Analysis Checklist

按需读取，不要整份全塞进上下文。

建议顺序：

1. `Repo-Wide`
2. 对应技术栈 section
3. `Tests / CI / Release`

---

## Repo-Wide

- 项目根有哪些官方贡献文档
- 是否存在项目根 `AGENTS.md`
- 是否存在 `.harness/standards.md`
- 目录分层是按功能、按技术，还是混合
- 哪些术语在仓库里反复出现
- 哪些目录是高冲突区或高稳定区
- commit message 常见格式
- PR 粒度偏小步还是大批量

## JavaScript / TypeScript Backend

- 入口组织：route-first、resource-first、feature-first
- 模块系统：ESM / CJS
- 类型策略：strictness、`any` 使用、runtime validation
- API 层：参数校验、错误处理、返回结构
- 数据层：ORM / query builder / migration 习惯
- service / use-case：函数式还是 class-based
- 后台任务 / queue / cron 的组织方式

## React / Frontend

- 组件命名和目录组织
- props 类型写法
- 状态管理和数据获取方式
- 表单模式
- 样式体系：Tailwind / CSS Modules / styled-components / SCSS
- class utility 约定：`clsx` / `cn` / lookup map
- 前端测试组织和 fixture 方式

## Python

- 包管理：`uv` / poetry / pip
- 布局：flat / `src/`
- Web 框架：Django / FastAPI / Flask
- 数据层：ORM / migrations / repository pattern
- service 结构：function / class / result shape
- pytest fixture / factory / mocking 习惯

## Go

- 目录布局：`cmd/` / `internal/` / `pkg/`
- interface 放哪里定义
- error wrapping 和 sentinel error 习惯
- `context.Context` 传递策略
- 并发模式：goroutine / channel / worker pool
- table-driven tests 和 helper 组织

## Ruby / Rails

- model 文件组织和 concern 使用
- controller / policy / authorization 写法
- service / job / worker 结构
- presenter / serializer 模式
- factory / shared example / system test 习惯

## Tests / CI / Release

- 默认验证命令是什么
- unit / integration / e2e 的边界
- 测试数据构造方式
- mocking / fixture / snapshot 约定
- CI 会卡哪些 lint / type / test
- release / changelog / version bump 习惯

## What To Extract

每一层最后都尽量提炼出：

- 真实高频模式
- 与主流默认不同的地方
- 适合写进 contributor guide 的例子
- 适合蒸馏进 `.harness/standards.md` 的稳定规则
- 只适合留在 research memo 的观察结论
