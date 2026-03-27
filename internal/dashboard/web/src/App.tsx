import { useEffect, useMemo, useState, startTransition } from "react";
import type {
  ChecklistView,
  Dashboard,
  DashboardTaskFlow,
  DashboardThread,
  ExecutionEvent,
  ExecutionSliceView,
  PlannerLane,
  RequestLanding,
  TokenUsage,
  ToolStatus,
} from "./types";

const REFRESH_MS = 5000;

type WorkerNode = {
  id: string;
  title: string;
  status: string;
  summary?: string;
  dispatchId?: string;
  sessionName?: string;
  workerId?: string;
  at?: string;
  path?: string;
  kind?: string;
};

export function App() {
  const [data, setData] = useState<Dashboard | null>(null);
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [selectedEventId, setSelectedEventId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      try {
        const response = await fetch("/api/dashboard", { cache: "no-store" });
        if (!response.ok) {
          throw new Error(await response.text());
        }
        const next = (await response.json()) as Dashboard;
        if (cancelled) return;
        startTransition(() => {
          setData(next);
          setError(null);
          setSelectedTaskId((current) => {
            const ordered = orderFlows(next);
            if (ordered.length === 0) return null;
            if (current && ordered.some((flow) => flow.taskId === current)) {
              return current;
            }
            return ordered[0].taskId;
          });
        });
      } catch (loadError) {
        if (cancelled) return;
        setError(loadError instanceof Error ? loadError.message : String(loadError));
      }
    };

    void load();
    const timer = window.setInterval(() => {
      void load();
    }, REFRESH_MS);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, []);

  const orderedFlows = useMemo(() => orderFlows(data), [data]);
  const selectedFlow = useMemo(() => {
    if (!orderedFlows.length) return null;
    return orderedFlows.find((flow) => flow.taskId === selectedTaskId) ?? orderedFlows[0];
  }, [orderedFlows, selectedTaskId]);
  const selectedThread = useMemo(
    () =>
      data?.threads.find((thread) => thread.threadKey === selectedFlow?.threadKey) ??
      null,
    [data, selectedFlow],
  );
  const siblingFlows = useMemo(
    () =>
      orderedFlows.filter((flow) => flow.threadKey === selectedFlow?.threadKey),
    [orderedFlows, selectedFlow],
  );
  const flowByTaskID = useMemo(() => {
    const entries = new Map<string, DashboardTaskFlow>();
    orderedFlows.forEach((flow) => {
      entries.set(flow.taskId, flow);
    });
    return entries;
  }, [orderedFlows]);
  const threadLandings = useMemo(
    () => selectedThread?.requestLandings || selectedFlow?.requestLandings || [],
    [selectedThread, selectedFlow],
  );
  const workerNodes = useMemo(() => buildWorkerNodes(selectedFlow), [selectedFlow]);
  const threadTokenUsage = useMemo(
    () => aggregateTokenUsage(siblingFlows.map((flow) => flow.tokenUsage)),
    [siblingFlows],
  );
  const selectedEvent = useMemo(
    () =>
      workerNodes.find((item) => item.id === selectedEventId) ??
      workerNodes[workerNodes.length - 1] ??
      null,
    [selectedEventId, workerNodes],
  );

  useEffect(() => {
    setSelectedEventId((current) => {
      if (current && workerNodes.some((item) => item.id === current)) return current;
      return workerNodes[workerNodes.length - 1]?.id ?? null;
    });
  }, [workerNodes]);

  return (
    <div className="app-shell">
      <header className="hero">
        <section className="hero__main glass">
          <div className="eyebrow">Harness Architect</div>
          <h1>React Operator Surface</h1>
          <p className="hero__copy">
            这次页面把同一条任务拆成三层来看。先看人话进度，再看模型合同，最后看程序状态和执行链，避免把 runtime 边界、模型目标和 operator 视角混在一起。
          </p>
          <div className="meta mono">
            root · {data?.root || "-"}
            <br />
            generatedAt · {data?.generatedAt || "-"}
          </div>
        </section>

        <section className="hero__stats glass">
          <div className="panel-head">
            <div>
              <h2>项目总览</h2>
              <p className="muted">核心指标和 token 花销直接从真实 dashboard API 拉取。</p>
            </div>
          </div>
          <div className="stat-grid">
            <StatCard
              label="Tasks"
              value={data?.overview.totalTasks ?? 0}
              meta={`pending ${data?.overview.pendingTasks ?? 0}`}
            />
            <StatCard
              label="Threads"
              value={data?.overview.totalThreads ?? 0}
              meta={`requests ${data?.overview.totalRequests ?? 0}`}
            />
            <StatCard
              label="Tmux"
              value={data?.overview.activeTmuxSessions ?? 0}
              meta={`legacy ${data?.overview.legacySessionCount ?? 0}`}
            />
            <StatCard
              label="Input Tokens"
              value={formatNumber(data?.overview.tokenUsage.inputTokens ?? 0)}
              meta={`cached ${formatNumber(data?.overview.tokenUsage.cachedInputTokens ?? 0)} / output ${formatNumber(data?.overview.tokenUsage.outputTokens ?? 0)}`}
            />
          </div>
        </section>
      </header>

      <main className="workspace">
        <aside className="sidebar">
          <section className="glass panel">
            <div className="panel-head">
              <div>
                <h2>主任务导航</h2>
                <p className="muted">先点主任务，再顺着编排链往下看。</p>
              </div>
            </div>
            <div className="nav-list">
              {orderedFlows.map((flow) => (
                <button
                  key={flow.taskId}
                  className={`nav-card ${selectedFlow?.taskId === flow.taskId ? "is-active" : ""}`}
                  onClick={() => {
                    startTransition(() => setSelectedTaskId(flow.taskId));
                  }}
                >
                  <div className="nav-card__title">{flow.name || flow.title || flow.taskId}</div>
                  <div className="badge-row">
                    <Badge label={flow.status || "unknown"} tone={flow.status} />
                    <Badge label={flow.release?.status || "release"} tone={flow.release?.status} />
                  </div>
                  <div className="nav-card__meta mono">
                    {flow.taskId} · {flow.threadKey || "-"}
                  </div>
                  <div className="nav-card__meta">
                    planEpoch {flow.planEpoch || 0} · req {flow.requestLandings?.length || 0} · turns {flow.tokenUsage?.turns || 0}
                  </div>
                </button>
              ))}
            </div>
          </section>

          <section className="glass panel">
            <div className="panel-head">
              <div>
                <h2>环境</h2>
                <p className="muted">本地工具链探测。</p>
              </div>
            </div>
            <div className="meta mono">CODEX_HOME {data?.environment.codexHome || "-"}</div>
            <div className="stack">
              {(data?.environment.tools || []).map((tool) => (
                <ToolCard key={tool.name} tool={tool} />
              ))}
            </div>
          </section>

          <section className="glass panel">
            <div className="panel-head">
              <div>
                <h2>控制面提示</h2>
                <p className="muted">缺口会直接暴露出来。</p>
              </div>
            </div>
            <div className="stack">
              {(error ? [error] : data?.warnings || []).map((warning) => (
                <div key={warning} className="warning-card">
                  {warning}
                </div>
              ))}
              {!error && !(data?.warnings || []).length ? <Empty text="当前没有新的全局告警。" /> : null}
            </div>
          </section>
        </aside>

        <section className="main-pane">
          {selectedFlow ? (
            <>
              <section className="glass panel focus">
                <div className="focus-head">
                  <div>
                    <div className="eyebrow">Focused Task</div>
                    <h2 className="focus-title">{selectedFlow.name || selectedFlow.title || selectedFlow.taskId}</h2>
                    <p className="muted">{selectedFlow.summary || selectedFlow.title || selectedFlow.taskId}</p>
                    <div className="meta mono">
                      task={selectedFlow.taskId} · thread={selectedFlow.threadKey || "-"} · dispatch={selectedFlow.lastDispatchId || "-"} · tmux={selectedFlow.tmuxSession || "-"}
                    </div>
                  </div>
                  <div className="badge-column">
                    <Badge label={selectedFlow.status || "unknown"} tone={selectedFlow.status} />
                    <Badge label={selectedFlow.release?.status || "release"} tone={selectedFlow.release?.status} />
                    {selectedFlow.currentSliceId ? <Badge label={selectedFlow.currentSliceId} tone="active" /> : null}
                  </div>
                </div>
                <div className="metric-row">
                  <Metric label="Task Name" value={selectedFlow.name || selectedFlow.taskId} />
                  <Metric label="Plan Epoch" value={String(selectedFlow.planEpoch || 0)} />
                  <Metric label="Current Slice" value={selectedFlow.currentSliceId || "not bound"} />
                  <Metric
                    label="Token Cost"
                    value={`${formatNumber(selectedFlow.tokenUsage.inputTokens)} / ${formatNumber(selectedFlow.tokenUsage.outputTokens)}`}
                    meta={`cached ${formatNumber(selectedFlow.tokenUsage.cachedInputTokens)} · turns ${selectedFlow.tokenUsage.turns || 0}`}
                  />
                </div>
                <div className="cards-split">
                  <OperatorCard flow={selectedFlow} />
                  <RuntimeCard flow={selectedFlow} />
                </div>
                <div className="stack">
                  <ModelCard flow={selectedFlow} />
                </div>
                <div className="token-ledger">
                  <TokenLedgerCard
                    scope="Project"
                    usage={data?.overview.tokenUsage}
                    note="全量任务累计"
                  />
                  <TokenLedgerCard
                    scope="Thread"
                    usage={threadTokenUsage}
                    note={selectedFlow.threadKey || "无 thread"}
                  />
                  <TokenLedgerCard
                    scope="Focused Task"
                    usage={selectedFlow.tokenUsage}
                    note={selectedFlow.taskId}
                  />
                </div>
              </section>

              <section className="layout-two">
                <section className="glass panel">
                  <div className="panel-head">
                    <div>
                      <h2>B3E Planner 分支</h2>
                      <p className="muted">每个 planner 的任务命名、候选流向、关键动作和风险点。</p>
                    </div>
                  </div>
                  <div className="planner-grid">
                    {(selectedFlow.planning?.plannerLanes || []).map((lane) => (
                      <PlannerCard key={lane.id} lane={lane} />
                    ))}
                  </div>
                  <JudgeCard flow={selectedFlow} />
                </section>

                <section className="glass panel">
                  <div className="panel-head">
                    <div>
                      <h2>Thread 聚合任务</h2>
                      <p className="muted">追加需求、聚合任务、以及新的落点都在这里点进去。</p>
                    </div>
                  </div>
                  <div className="cluster">
                    <div className="track-group">
                      <div className="track-label">Requests</div>
                      <div className="track-scroll">
                        {threadLandings.map((landing) => (
                          <RequestNode
                            key={landing.requestId}
                            landing={landing}
                            onClick={() => {
                              if (!landing.taskId) return;
                              startTransition(() => setSelectedTaskId(landing.taskId));
                            }}
                          />
                        ))}
                      </div>
                    </div>
                    <div className="track-group">
                      <div className="track-label">Tasks</div>
                      <div className="track-scroll">
                        {siblingFlows.map((flow) => (
                          <button
                            key={flow.taskId}
                            className={`track-node ${selectedFlow.taskId === flow.taskId ? "is-active" : ""}`}
                            onClick={() => {
                              startTransition(() => setSelectedTaskId(flow.taskId));
                            }}
                          >
                            <div className="track-node__title">{flow.name || flow.taskId}</div>
                            <div className="badge-row">
                              <Badge label={flow.status || "unknown"} tone={flow.status} />
                              <Badge label={flow.release?.status || "release"} tone={flow.release?.status} />
                            </div>
                            <div className="track-node__meta mono">{flow.taskId}</div>
                            <div className="track-node__meta">slice {flow.currentSliceId || "not bound"}</div>
                          </button>
                        ))}
                      </div>
                    </div>
                    <div className="track-group">
                      <div className="track-label">追加需求落点追踪</div>
                      <div className="stack">
                        {threadLandings.map((landing) => (
                          <LandingTraceCard
                            key={`trace-${landing.requestId}`}
                            landing={landing}
                            flow={landing.taskId ? flowByTaskID.get(landing.taskId) : undefined}
                            onSelectTask={setSelectedTaskId}
                          />
                        ))}
                        {!threadLandings.length ? <Empty text="当前 thread 还没有 request 落点记录。" /> : null}
                      </div>
                    </div>
                  </div>
                </section>
              </section>

              <section className="layout-two">
                <section className="glass panel">
                  <div className="panel-head">
                    <div>
                      <h2>Tmux Worker / Replan 节点</h2>
                      <p className="muted">可点击的执行链。点任一节点，右侧会看见详细信息。</p>
                    </div>
                  </div>
                  <div className="timeline-grid">
                    <div className="timeline-list">
                      {workerNodes.map((node) => (
                        <button
                          key={node.id}
                          className={`timeline-node ${selectedEvent?.id === node.id ? "is-active" : ""}`}
                          onClick={() => {
                            startTransition(() => setSelectedEventId(node.id));
                          }}
                        >
                          <span className={`timeline-dot ${toneClass(node.status)}`} />
                          <div className="timeline-node__body">
                            <div className="timeline-node__title">{node.title}</div>
                            <div className="timeline-node__summary">{node.summary || node.kind || ""}</div>
                            <div className="timeline-node__meta mono">
                              {node.dispatchId ? `dispatch=${node.dispatchId} ` : ""}
                              {node.sessionName ? `session=${node.sessionName}` : ""}
                            </div>
                          </div>
                          <Badge label={node.status || "info"} tone={node.status} />
                        </button>
                      ))}
                    </div>
                    <div className="event-detail">
                      {selectedEvent ? (
                        <>
                          <div className="event-detail__title">{selectedEvent.title}</div>
                          <div className="badge-row">
                            <Badge label={selectedEvent.status} tone={selectedEvent.status} />
                            {selectedEvent.kind ? <Badge label={selectedEvent.kind} tone="active" /> : null}
                          </div>
                          <div className="event-detail__body">{selectedEvent.summary || "这个节点没有额外摘要。"}</div>
                          <div className="meta mono">
                            {selectedEvent.dispatchId ? `dispatch=${selectedEvent.dispatchId}` : ""}
                            <br />
                            {selectedEvent.sessionName ? `session=${selectedEvent.sessionName}` : ""}
                            <br />
                            {selectedEvent.workerId ? `worker=${selectedEvent.workerId}` : ""}
                            <br />
                            {selectedEvent.path || selectedEvent.at || ""}
                          </div>
                        </>
                      ) : (
                        <Empty text="当前没有 worker 节点详情。" />
                      )}
                    </div>
                  </div>
                </section>

                <section className="glass panel">
                  <div className="panel-head">
                    <div>
                      <h2>Tasklist / Checklist</h2>
                      <p className="muted">一边看 execution tasks，一边看 verify checklist。</p>
                    </div>
                  </div>
                  <div className="cards-split">
                    <div className="stack">
                      <SectionLabel title="Tasklist" meta={`${selectedFlow.taskList?.length || 0} items`} />
                      {(selectedFlow.taskList || []).map((slice) => (
                        <ExecutionCard key={slice.id || slice.title} slice={slice} />
                      ))}
                      {!selectedFlow.taskList?.length ? <Empty text="当前没有 execution tasks。" /> : null}
                    </div>
                    <div className="stack">
                      <SectionLabel title="Checklist" meta={`${selectedFlow.checklist?.length || 0} items`} />
                      {(selectedFlow.checklist || []).map((check) => (
                        <ChecklistCard key={check.id || check.title} item={check} />
                      ))}
                      {!selectedFlow.checklist?.length ? <Empty text="当前没有 checklist。" /> : null}
                    </div>
                  </div>
                </section>
              </section>

              <section className="layout-two">
                <section className="glass panel">
                  <div className="panel-head">
                    <div>
                      <h2>任务日志窗口</h2>
                      <p className="muted">当前主任务的末尾日志和 token source。</p>
                    </div>
                  </div>
                  <div className="log-panel">
                    {(selectedFlow.logPreview || []).map((line, index) => (
                      <pre key={`${selectedFlow.taskId}-log-${index}`} className="log-line">
                        {line}
                      </pre>
                    ))}
                    {!selectedFlow.logPreview?.length ? <Empty text="当前没有可读日志。" /> : null}
                  </div>
                  <div className="meta mono">
                    {selectedFlow.attachCommand || "当前没有 attach 命令。"}
                    <br />
                    {(selectedFlow.tokenUsage.sourcePaths || []).join(" | ")}
                  </div>
                </section>

                <section className="glass panel">
                  <div className="panel-head">
                    <div>
                      <h2>全局事件 / Thread 视图</h2>
                      <p className="muted">最近事件与 thread 汇总一起看，方便定位追加需求的落点。</p>
                    </div>
                  </div>
                  <div className="cards-split">
                    <div className="stack">
                      <SectionLabel title="Recent Events" meta={`${data?.recentEvents?.length || 0} items`} />
                      {(data?.recentEvents || []).map((event) => (
                        <EventCard key={eventKey(event)} event={event} />
                      ))}
                    </div>
                    <div className="stack">
                      <SectionLabel title="Threads" meta={`${data?.threads.length || 0} items`} />
                      {(data?.threads || []).map((thread) => (
                        <ThreadCard key={thread.threadKey} thread={thread} onSelectTask={(taskId) => setSelectedTaskId(taskId)} />
                      ))}
                    </div>
                  </div>
                </section>
              </section>
            </>
          ) : (
            <section className="glass panel">
              <Empty text="当前没有 task flow 可供展示。" />
            </section>
          )}
        </section>
      </main>
    </div>
  );
}

function StatCard(props: { label: string; value: number | string; meta: string }) {
  return (
    <article className="stat-card">
      <div className="stat-card__label">{props.label}</div>
      <div className="stat-card__value">{props.value}</div>
      <div className="stat-card__meta">{props.meta}</div>
    </article>
  );
}

function Metric(props: { label: string; value: string; meta?: string }) {
  return (
    <article className="metric-card">
      <div className="metric-card__label">{props.label}</div>
      <div className="metric-card__value">{props.value}</div>
      {props.meta ? <div className="metric-card__meta">{props.meta}</div> : null}
    </article>
  );
}

function TokenLedgerCard(props: { scope: string; usage?: TokenUsage; note?: string }) {
  const usage = props.usage || zeroTokenUsage();
  return (
    <article className="token-ledger-card">
      <div className="token-ledger-card__head">
        <div className="token-ledger-card__scope">{props.scope}</div>
        <div className="token-ledger-card__total">{formatNumber(tokenTotal(usage))}</div>
      </div>
      <div className="token-ledger-card__meta">
        in {formatNumber(usage.inputTokens)} · cached {formatNumber(usage.cachedInputTokens)} · out {formatNumber(usage.outputTokens)} · turns {usage.turns}
      </div>
      <div className="token-ledger-card__note mono">{props.note || "-"}</div>
    </article>
  );
}

function ToolCard(props: { tool: ToolStatus }) {
  const { tool } = props;
  return (
    <article className="simple-card">
      <div className="simple-card__head">
        <div className="simple-card__title">{tool.name}</div>
        <Badge label={tool.found ? "ready" : "missing"} tone={tool.found ? "pass" : "fail"} />
      </div>
      <div className="meta mono">{tool.path || "not found"}</div>
    </article>
  );
}

function PlannerCard(props: { lane: PlannerLane }) {
  const { lane } = props;
  return (
    <article className={`planner-card ${laneTone(lane)}`}>
      <div className="planner-card__kicker">{plannerAlias(lane)}</div>
      <div className="planner-card__title">{lane.name || lane.id}</div>
      <div className="muted">{lane.focus || ""}</div>
      <div className="planner-card__task">{lane.taskName || "未命名任务"}</div>
      {lane.proposedFlow ? <div className="meta">候选流向 · {lane.proposedFlow}</div> : null}
      <div className="planner-card__summary">{lane.resultSummary || "当前没有 planner 摘要。"}</div>
      <SectionLabel title="Key Moves" />
      {(lane.keyMoves || []).length ? (
        <ul className="bullet-list">
          {(lane.keyMoves || []).map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      ) : (
        <Empty text="没有 key moves。" />
      )}
      <SectionLabel title="Risks" />
      {(lane.risks || []).length ? (
        <ul className="bullet-list">
          {(lane.risks || []).map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      ) : (
        <Empty text="没有显式风险。" />
      )}
      {(lane.evidence || []).length ? <div className="meta">evidence · {(lane.evidence || []).join(" | ")}</div> : null}
      <div className="badge-row">{lane.inferred ? <Badge label="inferred" tone="warn" /> : <Badge label="materialized" tone="pass" />}</div>
    </article>
  );
}

function JudgeCard(props: { flow: DashboardTaskFlow }) {
  const judge = props.flow.planning?.judge;
  return (
    <article className="judge-card">
      <div className="panel-head">
        <div>
          <h3>Judge Merge</h3>
          <div className="judge-card__title">{judge?.judgeName || judge?.judgeId || "Judge 未生成"}</div>
          <p className="muted">{judge?.selectedFlow || judge?.winnerStrategy || "当前没有 judge 汇合结果。"}</p>
        </div>
        <div className="badge-column">
          {judge?.reviewRequired ? <Badge label="review" tone="warn" /> : null}
          {judge?.verifyRequired ? <Badge label="verify" tone="active" /> : null}
        </div>
      </div>
      {judge ? (
        <>
          <div className="meta">winner strategy · {judge.winnerStrategy || "-"}</div>
          <div className="meta">dimensions · {(judge.selectedDimensions || []).join(" | ") || "-"}</div>
          <ul className="bullet-list">
            {(judge.rationale || []).map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </>
      ) : null}
    </article>
  );
}

function RequestNode(props: { landing: RequestLanding; onClick: () => void }) {
  const { landing, onClick } = props;
  const clickable = Boolean(landing.taskId);
  return (
    <button className={`track-node ${clickable ? "" : "is-disabled"}`} onClick={onClick} disabled={!clickable}>
      <div className="track-node__title">{landing.requestId}</div>
      <div className="badge-row">
        <Badge label={landing.bindingAction || "binding"} tone={landing.bindingAction} />
        <Badge label={landing.taskStatus || "unknown"} tone={landing.taskStatus} />
      </div>
      <div className="track-node__body">{landing.goal || "-"}</div>
      <div className="track-node__meta">落点 {landing.taskId || "未绑定"}</div>
      <div className="track-node__meta">{landing.normalizedIntentClass || ""}</div>
    </button>
  );
}

function LandingTraceCard(props: {
  landing: RequestLanding;
  flow?: DashboardTaskFlow;
  onSelectTask: (taskID: string) => void;
}) {
  const { landing, flow, onSelectTask } = props;
  const clickable = Boolean(flow?.taskId);
  return (
    <button
      className={`landing-trace ${clickable ? "" : "is-disabled"}`}
      disabled={!clickable}
      onClick={() => {
        if (flow?.taskId) onSelectTask(flow.taskId);
      }}
    >
      <div className="simple-card__head">
        <div className="simple-card__title">{landing.requestId}</div>
        <div className="badge-row">
          <Badge label={landing.bindingAction || "binding"} tone={landing.bindingAction} />
          <Badge label={flow?.status || landing.taskStatus || "unknown"} tone={flow?.status || landing.taskStatus} />
        </div>
      </div>
      <div className="simple-card__body">{landing.goal || "未记录目标摘要"}</div>
      <div className="meta">
        落点 task · {flow?.taskId || landing.taskId || "未绑定"} · slice {flow?.currentSliceId || "not bound"}
      </div>
      <div className="meta mono">
        intent {landing.normalizedIntentClass || "-"} · ctx {(landing.contexts || []).length} · turns {flow?.tokenUsage.turns || 0}
      </div>
      <div className="meta mono">
        {landing.createdAt || "-"}
        {landing.classificationReason ? ` · ${landing.classificationReason}` : ""}
      </div>
    </button>
  );
}

function ExecutionCard(props: { slice: ExecutionSliceView }) {
  const { slice } = props;
  const showScope = (slice.inScope?.length || 0) > 0 && (slice.inScope?.length || 0) <= 4;
  return (
    <article className="simple-card">
      <div className="simple-card__head">
        <div className="simple-card__title">{slice.title || slice.id || "slice"}</div>
        <Badge label={slice.status || "pending"} tone={slice.status} />
      </div>
      <div className="simple-card__body">{slice.summary || ""}</div>
      {showScope ? <div className="meta mono">boundary · {slice.inScope?.join(" | ")}</div> : null}
      {slice.doneCriteria?.length ? <div className="meta">done · {slice.doneCriteria.join(" | ")}</div> : null}
      {slice.requiredEvidence?.length ? <div className="meta">evidence · {slice.requiredEvidence.join(" | ")}</div> : null}
    </article>
  );
}

function OperatorCard(props: { flow: DashboardTaskFlow }) {
  const operator = props.flow.operator || {};
  return (
    <article className="simple-card">
      <div className="simple-card__head">
        <div className="simple-card__title">人话进度</div>
        <Badge label={props.flow.status || "unknown"} tone={props.flow.status} />
      </div>
      <div className="simple-card__body">{operator.headline || "当前还没有人话进度摘要。"}</div>
      {operator.currentStep ? <div className="meta">{operator.currentStep}</div> : null}
      {operator.nextAction ? <div className="meta">next · {operator.nextAction}</div> : null}
      {(operator.humanTaskList || []).length ? (
        <>
          <SectionLabel title="任务拆解" />
          <ul className="bullet-list">
            {(operator.humanTaskList || []).map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </>
      ) : null}
      {(operator.blockers || []).length ? <div className="meta">blockers · {(operator.blockers || []).join(" | ")}</div> : null}
      {(operator.notes || []).length ? <div className="meta">{(operator.notes || []).join(" | ")}</div> : null}
    </article>
  );
}

function ModelCard(props: { flow: DashboardTaskFlow }) {
  const model = props.flow.model || {};
  return (
    <article className="simple-card">
      <div className="simple-card__head">
        <div className="simple-card__title">模型合同</div>
        <Badge label="model" tone="active" />
      </div>
      <div className="simple-card__body">{model.objective || props.flow.summary || "当前没有模型目标摘要。"}</div>
      {(model.deliverables || []).length ? (
        <>
          <SectionLabel title="交付目标" />
          <ul className="bullet-list">
            {(model.deliverables || []).map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </>
      ) : null}
      {(model.acceptance || []).length ? <div className="meta">acceptance · {(model.acceptance || []).join(" | ")}</div> : null}
      {(model.boundaries || []).length ? <div className="meta mono">optional boundary · {(model.boundaries || []).join(" | ")}</div> : null}
    </article>
  );
}

function RuntimeCard(props: { flow: DashboardTaskFlow }) {
  const runtime = props.flow.runtime || {};
  return (
    <article className="simple-card">
      <div className="simple-card__head">
        <div className="simple-card__title">程序状态</div>
        <Badge label={runtime.releaseStatus || runtime.status || "runtime"} tone={runtime.releaseStatus || runtime.status} />
      </div>
      <div className="meta mono">
        dispatch={runtime.dispatchId || "-"}
        <br />
        lease={runtime.leaseId || "-"}
        <br />
        session={runtime.sessionName || "-"}
        <br />
        slice={runtime.currentSliceId || "-"}
      </div>
      {(runtime.promptStages || []).length ? <div className="meta">stages · {(runtime.promptStages || []).join(" -> ")}</div> : null}
      <div className="meta">token turns · {runtime.tokenTurns || 0}</div>
      {runtime.attachCommand ? <div className="meta mono">{runtime.attachCommand}</div> : null}
    </article>
  );
}

function ChecklistCard(props: { item: ChecklistView }) {
  const { item } = props;
  return (
    <article className="simple-card">
      <div className="simple-card__head">
        <div className="simple-card__title">{item.title || item.id || "check"}</div>
        <div className="badge-row">
          <Badge label={item.status || "unknown"} tone={item.status} />
          {item.source ? <Badge label={item.source} tone="active" /> : null}
        </div>
      </div>
      <div className="simple-card__body">{item.detail || ""}</div>
    </article>
  );
}

function EventCard(props: { event: ExecutionEvent }) {
  const { event } = props;
  return (
    <article className="simple-card">
      <div className="simple-card__head">
        <div className="simple-card__title">{event.title || event.kind || "event"}</div>
        <div className="badge-row">
          <Badge label={event.status || "info"} tone={event.status} />
          {event.source ? <Badge label={event.source} tone="active" /> : null}
        </div>
      </div>
      <div className="simple-card__body">{event.summary || ""}</div>
      <div className="meta mono">
        {event.taskId ? `task=${event.taskId} ` : ""}
        {event.dispatchId ? `dispatch=${event.dispatchId}` : ""}
      </div>
    </article>
  );
}

function ThreadCard(props: { thread: DashboardThread; onSelectTask: (taskId: string) => void }) {
  const { thread, onSelectTask } = props;
  return (
    <article className="simple-card">
      <div className="simple-card__head">
        <div className="simple-card__title">{thread.threadKey}</div>
        <div className="badge-row">
          <Badge label={thread.status || "unknown"} tone={thread.status} />
          <Badge label={`${thread.requestCount} req`} tone="queued" />
          <Badge label={`${thread.taskCount} task`} tone="active" />
        </div>
      </div>
      <div className="meta">
        planEpoch {thread.planEpoch || 0} / current {thread.currentPlanEpoch || 0} / valid {thread.latestValidPlanEpoch || 0}
      </div>
      <div className="stack tight">
        {(thread.taskIds || []).map((taskId) => (
          <button key={taskId} className="thread-task" onClick={() => onSelectTask(taskId)}>
            <span className="mono">{taskId}</span>
            <span className="thread-task__arrow">open</span>
          </button>
        ))}
      </div>
    </article>
  );
}

function SectionLabel(props: { title: string; meta?: string }) {
  return (
    <div className="section-label">
      <span>{props.title}</span>
      {props.meta ? <span>{props.meta}</span> : null}
    </div>
  );
}

function Badge(props: { label: string; tone?: string }) {
  return <span className={`badge ${toneClass(props.tone)}`}>{props.label}</span>;
}

function Empty(props: { text: string }) {
  return <div className="empty">{props.text}</div>;
}

function orderFlows(data: Dashboard | null) {
  if (!data) return [];
  return [...data.taskFlows].sort((left, right) => flowScore(data, right) - flowScore(data, left) || String(right.updatedAt || right.taskId).localeCompare(String(left.updatedAt || left.taskId)));
}

function flowScore(data: Dashboard, flow: DashboardTaskFlow) {
  const thread = data.threads.find((item) => item.threadKey === flow.threadKey);
  let score = 0;
  if ((flow.planning?.plannerLanes || []).length) score += 40;
  if ((flow.requestLandings || []).length) score += 30;
  if ((thread?.requestLandings || []).length) score += 20;
  if (String(flow.taskId).startsWith("T-")) score += 10;
  if (!String(flow.taskId).startsWith("task_codexsess_")) score += 5;
  return score;
}

function buildWorkerNodes(flow: DashboardTaskFlow | null): WorkerNode[] {
  if (!flow) return [];
  const nodes: WorkerNode[] = [];
  const seen = new Set<string>();

  const pushNode = (node: WorkerNode) => {
    if (seen.has(node.id)) return;
    seen.add(node.id);
    nodes.push(node);
  };

  if (flow.lastDispatchId) {
    pushNode({
      id: `dispatch-${flow.lastDispatchId}`,
      title: "Current Dispatch",
      status: flow.status === "queued" ? "queued" : "active",
      summary: flow.lastDispatchId,
      dispatchId: flow.lastDispatchId,
      at: flow.updatedAt,
      kind: "dispatch.current",
    });
  }

  if (flow.tmuxSession) {
    pushNode({
      id: `tmux-${flow.tmuxSession}`,
      title: "Tmux Worker",
      status: flow.status === "running" ? "running" : "active",
      summary: flow.tmuxSession,
      dispatchId: flow.lastDispatchId,
      sessionName: flow.tmuxSession,
      at: flow.updatedAt,
      kind: "tmux.current",
    });
  }

  (flow.executionChain || []).forEach((event, index) => {
    const id = eventKey(event) + `-${index}`;
    pushNode({
      id,
      title: event.title || event.kind || "event",
      status: event.status || "active",
      summary: event.summary,
      dispatchId: event.dispatchId,
      sessionName: event.sessionName,
      workerId: event.workerId,
      at: event.at,
      path: event.path,
      kind: event.kind,
    });
  });

  if ((flow.release?.nextAction === "replan" || flow.status === "needs_replan") && !nodes.some((item) => item.kind === "replan.emitted")) {
    pushNode({
      id: `replan-${flow.taskId}-${flow.updatedAt || ""}`,
      title: "Replan Node",
      status: "needs_replan",
      summary: flow.statusReason || flow.release?.status || "runtime requested replan",
      dispatchId: flow.lastDispatchId,
      at: flow.updatedAt,
      kind: "replan.synthetic",
    });
  }

  return nodes.sort((left, right) => String(left.at || "").localeCompare(String(right.at || "")));
}

function eventKey(event: ExecutionEvent) {
  return [event.kind, event.dispatchId, event.sessionName, event.at, event.title].join("|");
}

function toneClass(value?: string) {
  const tone = String(value || "").toLowerCase();
  if (["pass", "passed", "verified", "succeeded", "release_ready", "completed", "ready"].includes(tone)) return "tone-pass";
  if (["warn", "warning", "awaiting_gate", "needs_review", "needs_replan", "replan"].includes(tone)) return "tone-warn";
  if (["fail", "failed", "blocked", "error"].includes(tone)) return "tone-fail";
  if (["running", "active", "started"].includes(tone)) return "tone-active";
  return "tone-pending";
}

function plannerAlias(lane: PlannerLane) {
  const text = `${lane.id} ${lane.name} ${lane.focus || ""}`.toLowerCase();
  if (text.includes("architecture")) return "Planner A · 边界与结构";
  if (text.includes("delivery")) return "Planner B · 切片与顺序";
  if (text.includes("risk")) return "Planner C · 风险与验证";
  return "Planner Lane";
}

function laneTone(lane: PlannerLane) {
  const text = `${lane.id} ${lane.name} ${lane.focus || ""}`.toLowerCase();
  if (text.includes("architecture")) return "planner-card--architecture";
  if (text.includes("delivery")) return "planner-card--delivery";
  if (text.includes("risk")) return "planner-card--risk";
  return "";
}

function formatNumber(value: number) {
  return new Intl.NumberFormat("zh-CN").format(value);
}

function tokenTotal(usage: TokenUsage) {
  return usage.inputTokens + usage.outputTokens;
}

function zeroTokenUsage(): TokenUsage {
  return {
    inputTokens: 0,
    cachedInputTokens: 0,
    outputTokens: 0,
    turns: 0,
    sourcePaths: [],
  };
}

function aggregateTokenUsage(usages: Array<TokenUsage | undefined>) {
  return usages.reduce<TokenUsage>((acc, usage) => {
    const item = usage || zeroTokenUsage();
    acc.inputTokens += item.inputTokens;
    acc.cachedInputTokens += item.cachedInputTokens;
    acc.outputTokens += item.outputTokens;
    acc.turns += item.turns;
    if (item.sourcePaths?.length) {
      acc.sourcePaths = [...new Set([...(acc.sourcePaths || []), ...item.sourcePaths])];
    }
    return acc;
  }, zeroTokenUsage());
}
