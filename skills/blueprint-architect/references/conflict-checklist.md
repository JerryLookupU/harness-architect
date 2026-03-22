# Conflict Checklist

在这些场景读取本文件：

- 你要做 blueprint review
- 你怀疑方案看起来能做，但实际会撞边界
- 你需要把“风险”写成更可执行的冲突分析

## 冲突分类

### `hard conflict`

定义：

- 不解决就不能安全落地
- 会直接打破现有契约、数据、行为或发布路径

例子：

- 方案要求改接口，但兼容层不存在
- 方案要求共享 session，但现有执行模型禁止
- 迁移顺序要求先删旧结构，但读路径仍依赖旧结构

### `soft conflict`

定义：

- 可以落地，但成本、复杂度、维护性会明显变差

例子：

- 方案会引入重复状态
- 新层和旧层职责重叠
- rollout 顺序可做但操作性差

### `false conflict`

定义：

- 看起来冲突，实际可以通过增加维度或隔离面解决

例子：

- 并发看起来冲突，但可以 worktree 隔离
- 状态看起来冲突，但读写职责可分层

## 必扫冲突维度

1. 结构冲突
2. 接口冲突
3. 数据模型冲突
4. 状态机冲突
5. 并发/会话冲突
6. 验证冲突
7. 发布/迁移冲突
8. 回退冲突

## 推荐写法

每条冲突尽量写成：

- `where`
- `why`
- `severity`
- `resolution`

示例：

```text
where: recall planner vs new provenance writer
why: both want to own the same normalized record shape
severity: hard conflict
resolution: separate write-time envelope from read-time projection; keep planner read-only on provenance metadata
```

## 常见坏写法

不要只写：

- 有风险
- 可能冲突
- 需要注意兼容性

这种写法对实现没有帮助。

