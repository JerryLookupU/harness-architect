---
name: markdown-fetch
description: "抓取文档、网页和任意 URL 内容并转成干净的 markdown。适用于官方文档、API reference、博客、技术文章和需要 repo-local research memo 的场景。优先使用 Bash + curl + markdown.new，而不是依赖 Claude 插件。"
allowed-tools: ["Bash", "Read", "Write", "Edit"]
---

# This Skill Is For

这个 skill 用于把公开 URL 抓成适合阅读、引用、沉淀的 markdown。

适合这些场景：

- 拉官方文档或 API reference
- 读取博客、设计说明、迁移指南
- 原始 HTML 太脏，不适合直接读
- 需要把网页内容先沉淀成 repo-local research memo

一句话：

> 先把网页抓干净，再决定是临时阅读，还是沉淀进 `.harness/research/`.

# 这个 Skill 不依赖什么

- 不依赖 Claude 插件
- 不依赖 `.claude/` 目录
- 默认只使用 Bash、`curl`、`jq` 和 `markdown.new`

# Preferred Workflow

按这个顺序做：

1. 先确认目标 URL 是真的，不要猜地址
2. 先用 `auto`
3. 如果页面是 JS-heavy，再切 `browser`
4. 只有图片、图表、流程图确实重要时，才加 `retain_images=true`
5. 如果结果要进入 Klein 的共享研究面，优先写 `.harness/research/<slug>.md`

# Preferred Command Surface

## 1. GET: auto

普通文档先试这个：

```bash
curl -fsSL "https://markdown.new/https://docs.python.org/3/library/pathlib.html" | sed -n '1,220p'
```

适合：

- 官方文档
- 普通静态页面
- 想快速预览前几百行

## 2. GET: browser

JS-heavy 页面或 `auto` 不够干净时：

```bash
curl -fsSL "https://markdown.new/https://example.com/docs?method=browser" | sed -n '1,220p'
```

适合：

- React / SPA 文档站
- 需要浏览器渲染后才有正文的页面

## 3. GET: retain_images

只有图片本身是研究对象时才打开：

```bash
curl -fsSL "https://markdown.new/https://example.com/docs?method=browser&retain_images=true" | sed -n '1,220p'
```

适合：

- 图表、架构图、流程图本身有价值
- 想保留图片 markdown 以便后续人工查看

## 4. POST: auto

URL 含查询参数、片段、shell 特殊字符，或你想拿结构化响应时，优先 POST：

```bash
curl -fsSL "https://markdown.new/" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/docs"}' \
  | jq -r '.content'
```

## 5. POST: browser / retain_images

```bash
curl -fsSL "https://markdown.new/" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/docs","method":"browser","retain_images":true}' \
  | jq -r '.content'
```

# Method Choice

- `auto`
  - 默认首选
  - 适合大部分文档站
- `browser`
  - 页面正文依赖浏览器渲染时再用
- `retain_images=true`
  - 只有图片是有效信息时再开
  - 不要默认开启，避免噪音

# What To Do With The Result

## 临时阅读

如果只是回答当前问题：

- 直接引用必要片段
- 记录 URL
- 不要为了“一次性阅读”无端写文件

## 研究沉淀

如果结果属于研究资料、设计依据、上游行为证据，优先沉淀到：

- `.harness/research/<slug>.md`

如果当前仓库没有 `.harness/`，且用户要可共享文档，再考虑：

- `docs/<slug>.md`

不要默认把网页内容塞进 `.claude/` 或其他 Claude 专属目录。

推荐 memo 结构：

```markdown
---
schemaVersion: "1.0"
generator: "markdown-fetch"
generatedAt: "2026-03-25T00:00:00Z"
slug: "<slug>"
researchMode: "targeted"
question: "<why this page matters>"
sources:
  - "<url>"
---

## Summary

## Key Findings

## Relevant Snippets

## Open Questions
```

规则：

- 先提炼，再写 memo；不要整页原样倾倒
- memo 是 repo-local 热结论，原始 URL 仍然是冷证据
- 如果后续 blueprint / spec / repair 会消费这份资料，优先让它进入 `.harness/research/`

# Failure Handling

- `auto` 失败，再试 `browser`
- 如果只需要正文，不要为了失败就默认保留图片
- 如果 `markdown.new` 返回失败或内容明显不完整，要明确告诉用户，而不是假装抓到了完整页面
- 如果 URL 本身存疑，先验证 URL，再继续

# Return Shape

优先返回：

- 页面标题或主题
- 关键结论
- 与当前任务直接相关的片段
- 原始 URL
- 如果已写 memo，给出 `.harness/research/<slug>.md` 路径
