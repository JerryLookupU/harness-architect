---
generator: klein-harness
generatedAt: "2026-03-19T14:30:00+08:00"
project: openclaw-brain-plugin
---

# Engineering Standards

## STD-001: Unit Test Coverage

所有导出的公共模块必须有对应的单元测试文件。

**理由**: 确保核心逻辑在重构和迭代中保持正确性。

**验证方式**: 检查每个 `src/` 下的 `.ts` 模块在 `tests/unit/` 下有对应 `.test.mjs` 文件。

<!-- @harness-lint: kind=standard id=STD-001 status=active reviewCadence=fibonacci-hours reviewInterval=1h lastReview=2026-03-19T14:30:00+08:00 nextReview=2026-03-19T15:30:00+08:00 -->

---

## STD-002: Error Recovery

涉及外部 I/O（网络、文件系统、数据库）的操作必须有明确的错误恢复路径，不允许静默吞错。

**理由**: 长时 agent 任务中，静默错误会导致不可追踪的漂移。

**验证方式**: 静态检查 — 所有 `await` 调用外部服务的代码路径必须有 `try/catch` 或 `.catch()` 且包含日志或重抛。

<!-- @harness-lint: kind=standard id=STD-002 status=active reviewCadence=fibonacci-hours reviewInterval=3h lastReview=2026-03-19T14:30:00+08:00 nextReview=2026-03-19T17:30:00+08:00 -->

---

## STD-003: Incremental Processing

批量数据处理操作必须支持增量模式，避免全量重跑。

**理由**: 全量重跑在大数据集上不可接受，且浪费 token/算力。

**验证方式**: 对应测试用例验证：给定已处理过的数据集，再次运行只处理新增/变更项。

<!-- @harness-lint: kind=standard id=STD-003 status=active reviewCadence=fibonacci-hours reviewInterval=8h lastReview=2026-03-19T14:30:00+08:00 nextReview=2026-03-19T22:30:00+08:00 -->

---

## STD-004: Graceful Degradation

当可选依赖（QMD、pgvector、OpenViking）不可用时，系统必须降级到备选方案而非崩溃。

**理由**: 不同部署环境的依赖可用性不同，插件必须在最小配置下可用。

**验证方式**: 集成测试 — 在不配置可选依赖的情况下启动插件，验证核心功能正常。

<!-- @harness-lint: kind=standard id=STD-004 status=active reviewCadence=fibonacci-hours reviewInterval=13h lastReview=2026-03-19T14:30:00+08:00 nextReview=2026-03-20T03:30:00+08:00 -->

---

## STD-005: TypeScript Strict Mode

所有源码必须在 `strict: true` 下编译通过，不允许 `@ts-ignore` 或 `any` 类型逃逸。

**理由**: 类型安全是长期可维护性的基础。

**验证方式**: `tsc --noEmit` 零错误。

<!-- @harness-lint: kind=standard id=STD-005 status=active reviewCadence=fibonacci-hours reviewInterval=34h lastReview=2026-03-19T14:30:00+08:00 nextReview=2026-03-21T00:30:00+08:00 -->

---

## STD-006: Harness Footprint Budget

`.harness` 控制面必须持续输出 footprint 指标，并将体积增长控制在可解释预算内；本体仓库也必须跟踪文件数、LOC 和脚本入口增长速度。

**理由**: phase-1 的目标是补足可复用控制面能力，而不是让 `.harness` 或本体仓库无限膨胀并拖慢并行协作。

**验证方式**:
- `refresh-state` 必须写出 `.harness/state/footprint.json`。
- `harness-status` 必须展示 footprint 关键信号。
- 每轮运行记录一次 `scripts/harness-footprint.sh --json` 或 `scripts/harness-size-tracker.sh --warn-only` 输出。

<!-- @harness-lint: kind=standard id=STD-006 status=active reviewCadence=fibonacci-hours reviewInterval=8h lastReview=2026-03-24T10:30:00+08:00 nextReview=2026-03-24T18:30:00+08:00 -->

---

## STD-007: Multi-Thread Push Conflict Discipline

当 task 允许提交或推送且存在并发 worker 时，必须先 fetch，再做一次有界 rebase 或 merge 重试；若仍冲突，升级为 merge 或 replan 请求，禁止强推覆盖。

**理由**: 多线程并发下 push 冲突是常态，不可把“强推成功”当成收敛策略。

**验证方式**: lineage 或 merge queue 中能看到冲突重试或升级记录；出现 push 失败时不得直接进入 completed。

<!-- @harness-lint: kind=standard id=STD-007 status=active reviewCadence=fibonacci-hours reviewInterval=13h lastReview=2026-03-24T10:30:00+08:00 nextReview=2026-03-24T23:30:00+08:00 -->
