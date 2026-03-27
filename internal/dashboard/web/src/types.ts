export type ToolStatus = {
  name: string;
  found: boolean;
  path?: string;
};

export type TokenUsage = {
  inputTokens: number;
  cachedInputTokens: number;
  outputTokens: number;
  turns: number;
  sourcePaths?: string[];
};

export type ReleaseBoard = {
  readyCount: number;
  needsReviewCount: number;
  needsReplanCount: number;
  awaitingGateCount: number;
  blockedCount: number;
  remainingSliceCount: number;
};

export type DashboardOverview = {
  totalTasks: number;
  pendingTasks: number;
  totalThreads: number;
  totalRequests: number;
  activeTmuxSessions: number;
  legacySessionCount: number;
  tokenUsage: TokenUsage;
  releaseBoard: ReleaseBoard;
};

export type RequestLanding = {
  requestId: string;
  taskId?: string;
  taskStatus?: string;
  bindingAction?: string;
  normalizedIntentClass?: string;
  frontDoorTriage?: string;
  goal?: string;
  contexts?: string[];
  createdAt?: string;
  classificationReason?: string;
};

export type DashboardThread = {
  threadKey: string;
  status?: string;
  planEpoch?: number;
  currentPlanEpoch?: number;
  latestValidPlanEpoch?: number;
  latestRequestId?: string;
  latestTaskId?: string;
  requestCount: number;
  taskCount: number;
  requestLandings?: RequestLanding[];
  taskIds?: string[];
};

export type PlannerLane = {
  id: string;
  name: string;
  focus?: string;
  taskName?: string;
  proposedFlow?: string;
  promptRef?: string;
  resultSummary?: string;
  keyMoves?: string[];
  risks?: string[];
  evidence?: string[];
  inferred: boolean;
};

export type JudgeMergeView = {
  judgeId?: string;
  judgeName?: string;
  selectedFlow?: string;
  winnerStrategy?: string;
  rationale?: string[];
  selectedDimensions?: string[];
  selectedLensIds?: string[];
  reviewRequired: boolean;
  verifyRequired: boolean;
};

export type DashboardPlanning = {
  source?: string;
  executionSliceId?: string;
  promptStages?: string[];
  plannerLanes?: PlannerLane[];
  judge?: JudgeMergeView;
  tracePreview?: string[];
};

export type DashboardModelView = {
  objective?: string;
  deliverables?: string[];
  acceptance?: string[];
  boundaries?: string[];
};

export type DashboardRuntimeView = {
  status?: string;
  releaseStatus?: string;
  dispatchId?: string;
  leaseId?: string;
  sessionName?: string;
  currentSliceId?: string;
  promptStages?: string[];
  attachCommand?: string;
  tokenTurns?: number;
};

export type DashboardOperatorView = {
  headline?: string;
  currentStep?: string;
  nextAction?: string;
  humanTaskList?: string[];
  blockers?: string[];
  notes?: string[];
};

export type ExecutionSliceView = {
  id?: string;
  title?: string;
  summary?: string;
  status?: string;
  inScope?: string[];
  doneCriteria?: string[];
  requiredEvidence?: string[];
  verificationSteps?: string[];
};

export type ChecklistView = {
  id?: string;
  title?: string;
  status?: string;
  required: boolean;
  detail?: string;
  source?: string;
};

export type ExecutionEvent = {
  at?: string;
  kind?: string;
  title?: string;
  status?: string;
  summary?: string;
  source?: string;
  taskId?: string;
  dispatchId?: string;
  workerId?: string;
  sessionName?: string;
  path?: string;
};

export type ReleaseReadiness = {
  status: string;
  ready: boolean;
  safeToArchive: boolean;
  nextAction?: string;
  blockingReasons?: string[];
};

export type DashboardTaskFlow = {
  taskId: string;
  threadKey?: string;
  name?: string;
  title?: string;
  summary?: string;
  status?: string;
  statusReason?: string;
  updatedAt?: string;
  planEpoch?: number;
  currentSliceId?: string;
  lastDispatchId?: string;
  tmuxSession?: string;
  release: ReleaseReadiness;
  planning: DashboardPlanning;
  model: DashboardModelView;
  runtime: DashboardRuntimeView;
  operator: DashboardOperatorView;
  taskList?: ExecutionSliceView[];
  checklist?: ChecklistView[];
  requestLandings?: RequestLanding[];
  executionChain?: ExecutionEvent[];
  tokenUsage: TokenUsage;
  attachCommand?: string;
  logPreview?: string[];
  dataWarnings?: string[];
};

export type Dashboard = {
  root: string;
  generatedAt: string;
  environment: {
    codexHome?: string;
    tools?: ToolStatus[];
  };
  overview: DashboardOverview;
  threads: DashboardThread[];
  taskFlows: DashboardTaskFlow[];
  recentEvents?: ExecutionEvent[];
  warnings?: string[];
};
