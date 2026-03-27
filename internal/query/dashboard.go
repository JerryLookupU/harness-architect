package query

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"klein-harness/internal/a2a"
	"klein-harness/internal/adapter"
	"klein-harness/internal/checkpoint"
	"klein-harness/internal/runtime"
	"klein-harness/internal/state"
	"klein-harness/internal/tmux"
)

type Dashboard struct {
	Root         string               `json:"root"`
	GeneratedAt  string               `json:"generatedAt"`
	Environment  DashboardEnvironment `json:"environment"`
	Overview     DashboardOverview    `json:"overview"`
	Threads      []DashboardThread    `json:"threads"`
	TaskFlows    []DashboardTaskFlow  `json:"taskFlows"`
	RecentEvents []ExecutionEvent     `json:"recentEvents,omitempty"`
	Warnings     []string             `json:"warnings,omitempty"`
}

type DashboardEnvironment struct {
	CodexHome string       `json:"codexHome,omitempty"`
	Tools     []ToolStatus `json:"tools,omitempty"`
}

type ToolStatus struct {
	Name  string `json:"name"`
	Found bool   `json:"found"`
	Path  string `json:"path,omitempty"`
}

type DashboardOverview struct {
	TotalTasks         int          `json:"totalTasks"`
	PendingTasks       int          `json:"pendingTasks"`
	TotalThreads       int          `json:"totalThreads"`
	TotalRequests      int          `json:"totalRequests"`
	ActiveTmuxSessions int          `json:"activeTmuxSessions"`
	LegacySessionCount int          `json:"legacySessionCount"`
	TokenUsage         TokenUsage   `json:"tokenUsage"`
	ReleaseBoard       ReleaseBoard `json:"releaseBoard"`
}

type DashboardThread struct {
	ThreadKey            string           `json:"threadKey"`
	Status               string           `json:"status,omitempty"`
	PlanEpoch            int              `json:"planEpoch,omitempty"`
	CurrentPlanEpoch     int              `json:"currentPlanEpoch,omitempty"`
	LatestValidPlanEpoch int              `json:"latestValidPlanEpoch,omitempty"`
	LatestRequestID      string           `json:"latestRequestId,omitempty"`
	LatestTaskID         string           `json:"latestTaskId,omitempty"`
	RequestCount         int              `json:"requestCount"`
	TaskCount            int              `json:"taskCount"`
	RequestLandings      []RequestLanding `json:"requestLandings,omitempty"`
	TaskIDs              []string         `json:"taskIds,omitempty"`
}

type RequestLanding struct {
	RequestID             string   `json:"requestId"`
	TaskID                string   `json:"taskId,omitempty"`
	TaskStatus            string   `json:"taskStatus,omitempty"`
	BindingAction         string   `json:"bindingAction,omitempty"`
	NormalizedIntentClass string   `json:"normalizedIntentClass,omitempty"`
	FrontDoorTriage       string   `json:"frontDoorTriage,omitempty"`
	Goal                  string   `json:"goal,omitempty"`
	Contexts              []string `json:"contexts,omitempty"`
	CreatedAt             string   `json:"createdAt,omitempty"`
	ClassificationReason  string   `json:"classificationReason,omitempty"`
}

type DashboardTaskFlow struct {
	TaskID          string                `json:"taskId"`
	ThreadKey       string                `json:"threadKey,omitempty"`
	Name            string                `json:"name,omitempty"`
	Title           string                `json:"title,omitempty"`
	Summary         string                `json:"summary,omitempty"`
	Status          string                `json:"status,omitempty"`
	StatusReason    string                `json:"statusReason,omitempty"`
	UpdatedAt       string                `json:"updatedAt,omitempty"`
	PlanEpoch       int                   `json:"planEpoch,omitempty"`
	CurrentSliceID  string                `json:"currentSliceId,omitempty"`
	LastDispatchID  string                `json:"lastDispatchId,omitempty"`
	TmuxSession     string                `json:"tmuxSession,omitempty"`
	Release         ReleaseReadiness      `json:"release"`
	Planning        DashboardPlanning     `json:"planning"`
	Model           DashboardModelView    `json:"model"`
	Runtime         DashboardRuntimeView  `json:"runtime"`
	Operator        DashboardOperatorView `json:"operator"`
	TaskList        []ExecutionSliceView  `json:"taskList,omitempty"`
	Checklist       []ChecklistView       `json:"checklist,omitempty"`
	RequestLandings []RequestLanding      `json:"requestLandings,omitempty"`
	ExecutionChain  []ExecutionEvent      `json:"executionChain,omitempty"`
	TokenUsage      TokenUsage            `json:"tokenUsage"`
	AttachCommand   string                `json:"attachCommand,omitempty"`
	LogPreview      []string              `json:"logPreview,omitempty"`
	DataWarnings    []string              `json:"dataWarnings,omitempty"`
}

type DashboardModelView struct {
	Objective    string   `json:"objective,omitempty"`
	Deliverables []string `json:"deliverables,omitempty"`
	Acceptance   []string `json:"acceptance,omitempty"`
	Boundaries   []string `json:"boundaries,omitempty"`
}

type DashboardRuntimeView struct {
	Status         string   `json:"status,omitempty"`
	ReleaseStatus  string   `json:"releaseStatus,omitempty"`
	DispatchID     string   `json:"dispatchId,omitempty"`
	LeaseID        string   `json:"leaseId,omitempty"`
	SessionName    string   `json:"sessionName,omitempty"`
	CurrentSliceID string   `json:"currentSliceId,omitempty"`
	PromptStages   []string `json:"promptStages,omitempty"`
	AttachCommand  string   `json:"attachCommand,omitempty"`
	TokenTurns     int      `json:"tokenTurns,omitempty"`
}

type DashboardOperatorView struct {
	Headline      string   `json:"headline,omitempty"`
	CurrentStep   string   `json:"currentStep,omitempty"`
	NextAction    string   `json:"nextAction,omitempty"`
	HumanTaskList []string `json:"humanTaskList,omitempty"`
	Blockers      []string `json:"blockers,omitempty"`
	Notes         []string `json:"notes,omitempty"`
}

type DashboardPlanning struct {
	Source           string          `json:"source,omitempty"`
	ExecutionSliceID string          `json:"executionSliceId,omitempty"`
	PromptStages     []string        `json:"promptStages,omitempty"`
	PlannerLanes     []PlannerLane   `json:"plannerLanes,omitempty"`
	Judge            *JudgeMergeView `json:"judge,omitempty"`
	TracePreview     []string        `json:"tracePreview,omitempty"`
}

type PlannerLane struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Focus         string   `json:"focus,omitempty"`
	TaskName      string   `json:"taskName,omitempty"`
	ProposedFlow  string   `json:"proposedFlow,omitempty"`
	PromptRef     string   `json:"promptRef,omitempty"`
	ResultSummary string   `json:"resultSummary,omitempty"`
	KeyMoves      []string `json:"keyMoves,omitempty"`
	Risks         []string `json:"risks,omitempty"`
	Evidence      []string `json:"evidence,omitempty"`
	Inferred      bool     `json:"inferred"`
}

type JudgeMergeView struct {
	JudgeID            string   `json:"judgeId,omitempty"`
	JudgeName          string   `json:"judgeName,omitempty"`
	SelectedFlow       string   `json:"selectedFlow,omitempty"`
	WinnerStrategy     string   `json:"winnerStrategy,omitempty"`
	Rationale          []string `json:"rationale,omitempty"`
	SelectedDimensions []string `json:"selectedDimensions,omitempty"`
	SelectedLensIDs    []string `json:"selectedLensIds,omitempty"`
	ReviewRequired     bool     `json:"reviewRequired"`
	VerifyRequired     bool     `json:"verifyRequired"`
}

type ExecutionSliceView struct {
	ID                string   `json:"id,omitempty"`
	Title             string   `json:"title,omitempty"`
	Summary           string   `json:"summary,omitempty"`
	Status            string   `json:"status,omitempty"`
	InScope           []string `json:"inScope,omitempty"`
	DoneCriteria      []string `json:"doneCriteria,omitempty"`
	RequiredEvidence  []string `json:"requiredEvidence,omitempty"`
	VerificationSteps []string `json:"verificationSteps,omitempty"`
}

type ChecklistView struct {
	ID       string `json:"id,omitempty"`
	Title    string `json:"title,omitempty"`
	Status   string `json:"status,omitempty"`
	Required bool   `json:"required"`
	Detail   string `json:"detail,omitempty"`
	Source   string `json:"source,omitempty"`
}

type ExecutionEvent struct {
	At          string `json:"at,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Title       string `json:"title,omitempty"`
	Status      string `json:"status,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Source      string `json:"source,omitempty"`
	TaskID      string `json:"taskId,omitempty"`
	DispatchID  string `json:"dispatchId,omitempty"`
	WorkerID    string `json:"workerId,omitempty"`
	SessionName string `json:"sessionName,omitempty"`
	Path        string `json:"path,omitempty"`
}

type TokenUsage struct {
	InputTokens       int64    `json:"inputTokens"`
	CachedInputTokens int64    `json:"cachedInputTokens"`
	OutputTokens      int64    `json:"outputTokens"`
	Turns             int      `json:"turns"`
	SourcePaths       []string `json:"sourcePaths,omitempty"`
}

type dashboardRequestState struct {
	Summary runtime.RequestSummary
	Index   runtime.RequestIndex
	Map     runtime.RequestTaskMap
	Threads runtime.ThreadState
	ByID    map[string]runtime.RequestRecord
}

type legacyCodexSessionSummary struct {
	Sessions map[string]legacyCodexSession `json:"sessions"`
	Order    []string                      `json:"order"`
}

type legacyCodexSession struct {
	ID             string `json:"id"`
	TaskID         string `json:"taskId"`
	LastDispatchID string `json:"lastDispatchId,omitempty"`
	LastStatus     string `json:"lastStatus,omitempty"`
	LastSummary    string `json:"lastSummary,omitempty"`
	CreatedAt      string `json:"createdAt,omitempty"`
	UpdatedAt      string `json:"updatedAt,omitempty"`
	Model          string `json:"model,omitempty"`
}

type legacyDispatchManifest struct {
	DispatchID     string                   `json:"dispatchId"`
	GeneratedAt    string                   `json:"generatedAt,omitempty"`
	PromptStages   []string                 `json:"promptStages,omitempty"`
	RuntimeRefs    map[string]string        `json:"runtimeRefs,omitempty"`
	Artifacts      map[string]string        `json:"artifacts,omitempty"`
	ArtifactDir    string                   `json:"artifactDir,omitempty"`
	ExecutionCwd   string                   `json:"executionCwd,omitempty"`
	ExecutionModel string                   `json:"executionModel,omitempty"`
	ResumeStrategy string                   `json:"resumeStrategy,omitempty"`
	Verification   legacyManifestVerifyPlan `json:"verification,omitempty"`
}

type legacyManifestVerifyPlan struct {
	Commands []map[string]any `json:"commands,omitempty"`
	RuleIDs  []string         `json:"ruleIds,omitempty"`
}

type legacyOutcome struct {
	SchemaVersion string               `json:"schemaVersion,omitempty"`
	Status        string               `json:"status,omitempty"`
	Summary       string               `json:"summary,omitempty"`
	ExitCode      int                  `json:"exitCode,omitempty"`
	GeneratedAt   string               `json:"generatedAt,omitempty"`
	StartedAt     string               `json:"startedAt,omitempty"`
	FinishedAt    string               `json:"finishedAt,omitempty"`
	DurationSec   int                  `json:"durationSec,omitempty"`
	Stdout        string               `json:"stdout,omitempty"`
	Stderr        string               `json:"stderr,omitempty"`
	DiffStats     checkpoint.DiffStats `json:"diffStats,omitempty"`
	Artifacts     []string             `json:"artifacts,omitempty"`
}

func ProjectDashboard(root string) (Dashboard, error) {
	paths, err := adapter.Resolve(root)
	if err != nil {
		return Dashboard{}, err
	}
	tasks, err := ListTasks(root)
	if err != nil {
		return Dashboard{}, err
	}
	releaseBoard, err := ReleaseStatus(root)
	if err != nil {
		return Dashboard{}, err
	}
	requestState, err := loadDashboardRequestState(paths)
	if err != nil {
		return Dashboard{}, err
	}
	events, err := a2a.LoadEvents(paths.EventLogPath)
	if err != nil {
		return Dashboard{}, err
	}
	tmuxSummary, err := tmux.LoadSummary(root)
	if err != nil {
		return Dashboard{}, err
	}
	checkpointSummary, err := loadCheckpointSummary(paths.CheckpointSummaryPath)
	if err != nil {
		return Dashboard{}, err
	}
	legacySessions, err := loadLegacyCodexSessions(filepath.Join(paths.StateDir, "codex-session-summary.json"))
	if err != nil {
		return Dashboard{}, err
	}

	type taskBundle struct {
		Task     adapter.Task
		View     *TaskView
		Flow     DashboardTaskFlow
		Token    TokenUsage
		Events   []ExecutionEvent
		Warnings []string
	}
	bundles := make([]taskBundle, 0, len(tasks))
	projectWarnings := []string{}
	totalTokens := TokenUsage{}
	for _, task := range tasks {
		view, viewWarnings := loadDashboardTaskView(root, task.TaskID)
		flow, tokenUsage, flowWarnings, flowEvents := buildDashboardTaskFlow(paths, task, view, requestState, events, tmuxSummary, checkpointSummary, legacySessions)
		mergedWarnings := append(append([]string{}, viewWarnings...), flowWarnings...)
		if len(mergedWarnings) > 0 {
			projectWarnings = append(projectWarnings, mergedWarnings...)
		}
		totalTokens = mergeTokenUsage(totalTokens, tokenUsage)
		bundles = append(bundles, taskBundle{
			Task:     task,
			View:     view,
			Flow:     flow,
			Token:    tokenUsage,
			Events:   flowEvents,
			Warnings: mergedWarnings,
		})
	}
	sort.SliceStable(bundles, func(i, j int) bool {
		left := firstNonEmpty(bundles[i].Task.UpdatedAt, bundles[i].Task.TaskID)
		right := firstNonEmpty(bundles[j].Task.UpdatedAt, bundles[j].Task.TaskID)
		return left > right
	})
	taskFlows := make([]DashboardTaskFlow, 0, len(bundles))
	recentEvents := []ExecutionEvent{}
	for _, bundle := range bundles {
		taskFlows = append(taskFlows, bundle.Flow)
		recentEvents = append(recentEvents, bundle.Events...)
	}
	sortExecutionEvents(recentEvents)
	if len(recentEvents) > 18 {
		recentEvents = recentEvents[:18]
	}
	threads := buildDashboardThreads(tasks, requestState)
	activeTmuxSessions := 0
	for _, session := range tmuxSummary.Sessions {
		if strings.TrimSpace(session.Status) == "" || session.Status == "running" {
			activeTmuxSessions++
		}
	}
	pendingTasks := 0
	for _, task := range tasks {
		switch task.Status {
		case "completed", "archived", "stopped":
		default:
			pendingTasks++
		}
	}
	if len(tasks) > 0 && len(requestState.ByID) == 0 {
		projectWarnings = append(projectWarnings, "runtime request index is absent; thread/request tracking will degrade to queue or legacy state only")
	}
	if len(tasks) > 0 && len(tmuxSummary.Sessions) == 0 {
		projectWarnings = append(projectWarnings, "tmux summary is absent; dashboard is using legacy checkpoint or A2A evidence where available")
	}
	return Dashboard{
		Root:        paths.Root,
		GeneratedAt: state.NowUTC(),
		Environment: buildDashboardEnvironment(),
		Overview: DashboardOverview{
			TotalTasks:         len(tasks),
			PendingTasks:       pendingTasks,
			TotalThreads:       len(threads),
			TotalRequests:      len(requestState.ByID),
			ActiveTmuxSessions: activeTmuxSessions,
			LegacySessionCount: len(legacySessions.Sessions),
			TokenUsage:         totalTokens,
			ReleaseBoard:       releaseBoard,
		},
		Threads:      threads,
		TaskFlows:    taskFlows,
		RecentEvents: recentEvents,
		Warnings:     uniqueNonEmpty(projectWarnings),
	}, nil
}

func loadDashboardTaskView(root, taskID string) (*TaskView, []string) {
	view, err := Task(root, taskID)
	if err != nil {
		return nil, []string{fmt.Sprintf("task %s could not load full query view: %v", taskID, err)}
	}
	return &view, nil
}

func buildDashboardTaskFlow(
	paths adapter.Paths,
	task adapter.Task,
	view *TaskView,
	requestState dashboardRequestState,
	events []a2a.Envelope,
	tmuxSummary tmux.Summary,
	checkpointSummary checkpoint.Summary,
	legacySessions legacyCodexSessionSummary,
) (DashboardTaskFlow, TokenUsage, []string, []ExecutionEvent) {
	flow := DashboardTaskFlow{
		TaskID:         task.TaskID,
		ThreadKey:      task.ThreadKey,
		Title:          task.Title,
		Summary:        task.Summary,
		Status:         task.Status,
		StatusReason:   task.StatusReason,
		UpdatedAt:      task.UpdatedAt,
		PlanEpoch:      task.PlanEpoch,
		LastDispatchID: task.LastDispatchID,
		TmuxSession:    task.TmuxSession,
	}
	warnings := []string{}
	if view != nil {
		flow.Release = view.Release
		flow.AttachCommand = view.AttachCommand
		flow.LogPreview = append([]string(nil), view.LogPreview...)
	} else {
		flow.Release = ReleaseReadiness{Status: task.Status}
	}
	requestIDs := append([]string{}, requestState.Map.TaskToRequests[task.TaskID]...)
	if len(requestIDs) == 0 {
		for id, record := range requestState.ByID {
			if record.TaskID == task.TaskID {
				requestIDs = append(requestIDs, id)
			}
		}
	}
	sortRequestIDs(requestIDs)
	flow.RequestLandings = make([]RequestLanding, 0, len(requestIDs))
	for _, requestID := range requestIDs {
		record, ok := requestState.ByID[requestID]
		if !ok {
			continue
		}
		landing := requestLandingFromRecord(record, task.Status)
		if strings.TrimSpace(landing.TaskID) == "" {
			landing.TaskID = task.TaskID
		}
		flow.RequestLandings = append(flow.RequestLandings, landing)
	}
	flow.Name = deriveTaskFlowName(task, view, flow.RequestLandings)
	flow.Planning = buildPlanningView(task, view, &warnings)
	flow.TaskList = buildTaskList(view)
	flow.Checklist = buildChecklist(view)
	flow.CurrentSliceID = currentSliceID(view)
	executionChain := buildExecutionChain(paths, task, view, events, tmuxSummary, checkpointSummary, legacySessions, &warnings)
	sortExecutionEvents(executionChain)
	flow.ExecutionChain = executionChain
	tokenUsage := collectTokenUsage(paths, task, view, &warnings)
	flow.TokenUsage = tokenUsage
	flow.Model = buildModelView(task, view, flow.TaskList, flow.Checklist)
	flow.Runtime = buildRuntimeView(task, view, flow)
	flow.Operator = buildOperatorView(task, view, flow, warnings)
	flow.DataWarnings = uniqueNonEmpty(warnings)
	return flow, tokenUsage, warnings, executionChain
}

func deriveTaskFlowName(task adapter.Task, view *TaskView, landings []RequestLanding) string {
	source := firstNonEmpty(task.Title, task.Summary)
	if source == "" && view != nil && view.Request != nil {
		source = view.Request.Goal
	}
	if source == "" && len(landings) > 0 {
		source = landings[0].Goal
	}
	label := strings.TrimSpace(source)
	label = strings.TrimPrefix(label, "针对本地 harness-architect ")
	label = strings.TrimPrefix(label, "针对 harness-architect ")
	switch {
	case strings.Contains(label, "前端页面开发"):
		return "harness-architect 前端页面开发"
	case strings.Contains(label, "可视化开发"):
		return "harness-architect 可视化开发"
	case strings.Contains(strings.ToLower(label), "dashboard"):
		return "Dashboard 可视化任务"
	}
	if label == "" {
		return task.TaskID
	}
	runes := []rune(label)
	if len(runes) > 24 {
		return string(runes[:24]) + "..."
	}
	return label
}

func currentSliceID(view *TaskView) string {
	if view == nil {
		return ""
	}
	if view.TaskContract != nil && strings.TrimSpace(view.TaskContract.ExecutionSliceID) != "" {
		return view.TaskContract.ExecutionSliceID
	}
	if view.Planning != nil {
		return strings.TrimSpace(view.Planning.ExecutionSliceID)
	}
	return ""
}

func buildPlanningView(task adapter.Task, view *TaskView, warnings *[]string) DashboardPlanning {
	planning := DashboardPlanning{}
	if view == nil {
		planning.Source = "task-only fallback"
		return planning
	}
	if view.Planning != nil {
		planning.Source = "dispatch-ticket"
		planning.ExecutionSliceID = view.Planning.ExecutionSliceID
		planning.PromptStages = append([]string(nil), view.Planning.PromptStages...)
		planning.TracePreview = append([]string(nil), view.Planning.TracePreview...)
		if view.Planning.JudgeDecision.JudgeID != "" {
			planning.Judge = &JudgeMergeView{
				JudgeID:            view.Planning.JudgeDecision.JudgeID,
				JudgeName:          view.Planning.JudgeDecision.JudgeName,
				SelectedFlow:       view.Planning.JudgeDecision.SelectedFlow,
				WinnerStrategy:     view.Planning.JudgeDecision.WinnerStrategy,
				Rationale:          append([]string(nil), view.Planning.JudgeDecision.Rationale...),
				SelectedDimensions: append([]string(nil), view.Planning.JudgeDecision.SelectedDimensions...),
				SelectedLensIDs:    append([]string(nil), view.Planning.JudgeDecision.SelectedLensIDs...),
				ReviewRequired:     view.Planning.JudgeDecision.ReviewRequired,
				VerifyRequired:     view.Planning.JudgeDecision.VerifyRequired,
			}
		}
		if len(view.Planning.PacketSynthesis.Planners) > 0 {
			planning.PlannerLanes = inferPlannerLanes(view, warnings)
		}
		if len(planning.PlannerLanes) == 0 && planning.Judge == nil {
			*warnings = append(*warnings, fmt.Sprintf("task %s has planning metadata but no packet/judge surfaces were persisted", task.TaskID))
		}
		return planning
	}
	manifestPath := filepath.Join(filepath.Dir(filepath.Dir(pathsafe(view, task))), "unused")
	_ = manifestPath
	planning.Source = "task-only fallback"
	return planning
}

func inferPlannerLanes(view *TaskView, warnings *[]string) []PlannerLane {
	if view == nil || view.Planning == nil {
		return nil
	}
	if len(view.Planning.PlannerCandidates) > 0 {
		lanes := make([]PlannerLane, 0, len(view.Planning.PlannerCandidates))
		for _, candidate := range view.Planning.PlannerCandidates {
			lanes = append(lanes, PlannerLane{
				ID:            candidate.PlannerID,
				Name:          candidate.PlannerName,
				Focus:         candidate.Focus,
				TaskName:      candidate.TaskName,
				ProposedFlow:  candidate.ProposedFlow,
				ResultSummary: candidate.ResultSummary,
				KeyMoves:      append([]string(nil), candidate.KeyMoves...),
				Risks:         append([]string(nil), candidate.Risks...),
				Evidence:      append([]string(nil), candidate.Evidence...),
				Inferred:      false,
			})
		}
		return lanes
	}
	lanes := make([]PlannerLane, 0, len(view.Planning.PacketSynthesis.Planners))
	taskName := dashboardTaskName(view)
	for _, planner := range view.Planning.PacketSynthesis.Planners {
		lane := PlannerLane{
			ID:       planner.ID,
			Name:     planner.Name,
			Focus:    planner.Focus,
			TaskName: taskName,
			ProposedFlow: func() string {
				if view.Planning != nil && view.Planning.JudgeDecision.SelectedFlow != "" {
					return view.Planning.JudgeDecision.SelectedFlow
				}
				if view.AcceptedPacket != nil {
					return view.AcceptedPacket.FlowSelection
				}
				return ""
			}(),
			PromptRef: planner.PromptRef,
			Inferred:  true,
		}
		lowered := strings.ToLower(planner.ID + " " + planner.Name + " " + planner.Focus)
		switch {
		case strings.Contains(lowered, "architecture"):
			if view.AcceptedPacket != nil {
				lane.ResultSummary = firstNonEmpty(
					view.AcceptedPacket.FlowSelection,
					view.AcceptedPacket.SelectedPlan,
					view.AcceptedPacket.Objective,
				)
				lane.KeyMoves = append(lane.KeyMoves,
					"保持主任务命名稳定，便于追加需求继续落在同一条主线上",
					"先确认 control plane / execution plane / operator plane，再决定页面改动边界",
				)
				lane.Risks = append(lane.Risks,
					"owned paths 过宽会让主任务被错误切碎",
					"如果 worker 越过 runtime authority boundary，会直接触发 replan",
				)
				lane.Evidence = append(lane.Evidence, summarizeOwnedPaths(view.AcceptedPacket.OwnedPaths))
				lane.Evidence = append(lane.Evidence, summarizeList("constraints", view.AcceptedPacket.Constraints, 2))
			}
		case strings.Contains(lowered, "delivery"):
			if view.AcceptedPacket != nil {
				lane.ResultSummary = fmt.Sprintf("judge accepted %d execution slices", len(view.AcceptedPacket.ExecutionTasks))
				lane.KeyMoves = append(lane.KeyMoves,
					fmt.Sprintf("把主任务 `%s` 映射成可追踪的 execution tasks", taskName),
					"让当前 dispatch 只绑定一个 active execution slice",
				)
				if len(view.AcceptedPacket.ExecutionTasks) > 0 {
					lane.Evidence = append(lane.Evidence, "next slice: "+firstNonEmpty(view.NextSliceID, view.AcceptedPacket.ExecutionTasks[0].ID))
				}
			}
			if view.TaskContract != nil && view.TaskContract.ExecutionSliceID != "" {
				lane.KeyMoves = append(lane.KeyMoves, "current contract slice="+view.TaskContract.ExecutionSliceID)
				lane.Risks = append(lane.Risks,
					"delivery slice 太碎会让 operator 难以理解主任务编排",
					"缺少聚合 task 关系时，追加需求落点不直观",
				)
				lane.Evidence = append(lane.Evidence, "contract slice: "+view.TaskContract.ExecutionSliceID)
			}
		case strings.Contains(lowered, "risk"):
			if view.AcceptedPacket != nil {
				lane.ResultSummary = summarizeVerificationPlan(view.AcceptedPacket.VerificationPlan)
				lane.KeyMoves = append(lane.KeyMoves,
					"把 verify / review / rollback 结果挂到同一条执行链上",
					"verify 失败时重新进入 analysis.required -> needs_replan",
				)
				lane.Evidence = append(lane.Evidence, summarizeList("replan", view.AcceptedPacket.ReplanTriggers, 2))
				lane.Evidence = append(lane.Evidence, summarizeList("rollback", view.AcceptedPacket.RollbackHints, 2))
				lane.Risks = append(lane.Risks,
					firstNonEmpty(summarizeList("replan", view.AcceptedPacket.ReplanTriggers, 2), "verify 失败会触发 replan"),
					firstNonEmpty(summarizeList("rollback", view.AcceptedPacket.RollbackHints, 2), "rollback hint 未显式生成"),
				)
			}
			if view.TaskContract != nil {
				lane.Evidence = append(lane.Evidence, fmt.Sprintf("reviewRequired=%t", view.TaskContract.ReviewRequired))
			}
		default:
			lane.ResultSummary = "planner role is persisted, but raw planner candidate payload is not yet stored; dashboard is showing observable downstream effects only"
		}
		lane.KeyMoves = uniqueNonEmpty(lane.KeyMoves)
		lane.Risks = uniqueNonEmpty(lane.Risks)
		lane.Evidence = uniqueNonEmpty(lane.Evidence)
		if strings.TrimSpace(lane.ResultSummary) == "" {
			lane.ResultSummary = "planner lane is inferred from accepted packet / task contract outputs because raw planner candidates are not persisted yet"
		}
		lanes = append(lanes, lane)
	}
	if len(lanes) > 0 {
		*warnings = append(*warnings, "planner lanes are inferred from accepted packet / task contract outputs; raw planner-by-planner candidate payloads are not yet persisted")
	}
	return lanes
}

func dashboardTaskName(view *TaskView) string {
	if view == nil {
		return ""
	}
	candidate := ""
	switch {
	case view.Task.Title != "":
		candidate = view.Task.Title
	case view.Request != nil && strings.TrimSpace(view.Request.Goal) != "":
		candidate = view.Request.Goal
	case view.AcceptedPacket != nil:
		candidate = firstNonEmpty(view.AcceptedPacket.SelectedPlan, view.AcceptedPacket.Objective)
	}
	candidate = strings.TrimSpace(candidate)
	if strings.Contains(candidate, "前端页面开发") || strings.Contains(candidate, "可视化开发") {
		return "harness-architect 前端页面开发"
	}
	if candidate == "" {
		return view.Task.TaskID
	}
	runes := []rune(candidate)
	if len(runes) > 24 {
		return string(runes[:24]) + "..."
	}
	return candidate
}

func buildModelView(task adapter.Task, view *TaskView, taskList []ExecutionSliceView, checklist []ChecklistView) DashboardModelView {
	objective := strings.TrimSpace(firstNonEmpty(task.Summary, task.Title))
	if view != nil {
		objective = strings.TrimSpace(firstNonEmpty(pathsafeGoal(view), objective))
		if objective == "" && view.AcceptedPacket != nil {
			objective = strings.TrimSpace(firstNonEmpty(view.AcceptedPacket.Objective, view.AcceptedPacket.SelectedPlan))
		}
	}
	deliverables := parseHumanTaskLines(task, view)
	if len(deliverables) == 0 {
		for _, slice := range taskList {
			line := strings.TrimSpace(firstNonEmpty(slice.Title, slice.Summary))
			if line != "" {
				deliverables = append(deliverables, line)
			}
		}
	}
	acceptance := []string{}
	for _, slice := range taskList {
		acceptance = append(acceptance, slice.DoneCriteria...)
	}
	if len(acceptance) == 0 {
		for _, item := range checklist {
			acceptance = append(acceptance, strings.TrimSpace(item.Title))
		}
	}
	acceptance = limitStrings(uniqueNonEmpty(acceptance), 6)
	boundaries := []string{}
	if view != nil && view.AcceptedPacket != nil && shouldSurfaceBoundaryHints(view.AcceptedPacket.OwnedPaths) {
		boundaries = append(boundaries, view.AcceptedPacket.OwnedPaths...)
	}
	return DashboardModelView{
		Objective:    objective,
		Deliverables: limitStrings(uniqueNonEmpty(deliverables), 8),
		Acceptance:   acceptance,
		Boundaries:   boundaries,
	}
}

func buildRuntimeView(task adapter.Task, view *TaskView, flow DashboardTaskFlow) DashboardRuntimeView {
	runtimeView := DashboardRuntimeView{
		Status:         task.Status,
		ReleaseStatus:  flow.Release.Status,
		DispatchID:     task.LastDispatchID,
		LeaseID:        task.LastLeaseID,
		SessionName:    firstNonEmpty(task.TmuxSession, flow.TmuxSession),
		CurrentSliceID: flow.CurrentSliceID,
		AttachCommand:  flow.AttachCommand,
		TokenTurns:     flow.TokenUsage.Turns,
	}
	if view != nil && view.Planning != nil {
		runtimeView.PromptStages = append([]string(nil), view.Planning.PromptStages...)
	}
	return runtimeView
}

func buildOperatorView(task adapter.Task, view *TaskView, flow DashboardTaskFlow, warnings []string) DashboardOperatorView {
	humanTasks := append([]string(nil), flow.Model.Deliverables...)
	headline := humanProgressHeadline(task, flow, view)
	currentStep := humanCurrentStep(task, flow)
	nextAction := strings.TrimSpace(flow.Release.NextAction)
	if nextAction == "" {
		nextAction = humanDefaultNextAction(task, flow)
	}
	blockers := append([]string(nil), flow.Release.BlockingReasons...)
	notes := []string{}
	outputPath := extractPrimaryOutputPath(task, view)
	if outputPath != "" {
		fileCount := countVisibleFiles(outputPath)
		switch {
		case fileCount > 0:
			notes = append(notes, fmt.Sprintf("目标目录 `%s` 当前已可见 %d 个文件。", outputPath, fileCount))
		case task.Status == "running":
			notes = append(notes, fmt.Sprintf("目标目录 `%s` 还没有产出文件。", outputPath))
		}
	}
	if flow.TokenUsage.Turns == 0 && task.Status == "running" {
		notes = append(notes, "还没有可见 token 账本，说明 worker 还没走到有效 turn.completed 或业务输出阶段。")
	}
	for _, warning := range warnings {
		if strings.Contains(strings.ToLower(warning), "owned paths") {
			continue
		}
		notes = append(notes, warning)
	}
	return DashboardOperatorView{
		Headline:      headline,
		CurrentStep:   currentStep,
		NextAction:    nextAction,
		HumanTaskList: humanTasks,
		Blockers:      uniqueNonEmpty(blockers),
		Notes:         limitStrings(uniqueNonEmpty(notes), 6),
	}
}

func buildTaskList(view *TaskView) []ExecutionSliceView {
	if view == nil || view.AcceptedPacket == nil {
		return nil
	}
	completed := map[string]struct{}{}
	if view.PacketProgress != nil {
		for _, id := range view.PacketProgress.CompletedSliceIDs {
			completed[id] = struct{}{}
		}
	}
	out := make([]ExecutionSliceView, 0, len(view.AcceptedPacket.ExecutionTasks))
	for _, slice := range view.AcceptedPacket.ExecutionTasks {
		status := "pending"
		if _, ok := completed[slice.ID]; ok {
			status = "completed"
		} else if view.TaskContract != nil && view.TaskContract.ExecutionSliceID == slice.ID {
			status = "active"
		}
		out = append(out, ExecutionSliceView{
			ID:                slice.ID,
			Title:             slice.Title,
			Summary:           slice.Summary,
			Status:            status,
			InScope:           append([]string(nil), slice.InScope...),
			DoneCriteria:      append([]string(nil), slice.DoneCriteria...),
			RequiredEvidence:  append([]string(nil), slice.RequiredEvidence...),
			VerificationSteps: append([]string(nil), slice.VerificationSteps...),
		})
	}
	return out
}

func pathsafeGoal(view *TaskView) string {
	if view == nil || view.Request == nil {
		return ""
	}
	return strings.TrimSpace(view.Request.Goal)
}

func parseHumanTaskLines(task adapter.Task, view *TaskView) []string {
	segments := []string{task.Description, task.Summary}
	if view != nil && view.Request != nil {
		segments = append(segments, view.Request.Contexts...)
		segments = append(segments, view.Request.Goal)
	}
	joined := strings.Join(uniqueNonEmpty(segments), "\n")
	joined = strings.ReplaceAll(strings.ReplaceAll(joined, "\r\n", "\n"), "要求：", "\n")
	numbered := regexp.MustCompile(`(\d+[\.\)、:：-]+\s*)`)
	joined = numbered.ReplaceAllString(joined, "\n$1")
	requirementRE := regexp.MustCompile(`^(?:\d+[\.\)、:：-]*|[-*•]+)\s*`)
	lines := strings.Split(joined, "\n")
	out := make([]string, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if lower == "要求" || lower == "requirements" || lower == "requirement" {
			continue
		}
		if !requirementRE.MatchString(line) {
			continue
		}
		line = strings.TrimSpace(requirementRE.ReplaceAllString(line, ""))
		line = strings.Trim(line, "：:;；")
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return uniqueNonEmpty(out)
}

func shouldSurfaceBoundaryHints(paths []string) bool {
	paths = uniqueNonEmpty(paths)
	if len(paths) == 0 || len(paths) > 4 {
		return false
	}
	for _, path := range paths {
		if strings.Contains(path, ".DS_Store") || path == "." {
			return false
		}
	}
	return true
}

func humanProgressHeadline(task adapter.Task, flow DashboardTaskFlow, view *TaskView) string {
	switch task.Status {
	case "queued":
		return "任务已进入 harness，但还没真正开始执行。"
	case "running":
		if flow.Runtime.SessionName != "" {
			return "任务已经被 worker 认领，当前正在 tmux 里执行。"
		}
		return "任务正在执行链路里推进。"
	case "needs_replan":
		return "这轮没有顺利收口，runtime 正在要求 replan。"
	case "blocked":
		return "任务当前被阻塞，需要先处理验证或环境问题。"
	case "completed":
		return "任务已经跑完，当前重点是看产物和验证结果。"
	default:
		return firstNonEmpty(task.StatusReason, task.Status, "任务状态待确认。")
	}
}

func humanCurrentStep(task adapter.Task, flow DashboardTaskFlow) string {
	switch task.Status {
	case "queued":
		return "当前停在 submit 之后，等待 daemon 把它推进到 dispatch。"
	case "running":
		if flow.Runtime.DispatchID != "" && flow.Runtime.SessionName != "" {
			return fmt.Sprintf("dispatch `%s` 已发出，tmux worker `%s` 正在执行 `%s`。", flow.Runtime.DispatchID, flow.Runtime.SessionName, firstNonEmpty(flow.Runtime.CurrentSliceID, "当前 slice"))
		}
		return "任务已进入运行态，但执行节点信息还没完整落盘。"
	case "needs_replan":
		return "这轮 worker/verify 没有通过，准备进入下一轮规划。"
	case "blocked":
		return "当前先别继续加任务，先处理阻塞项。"
	case "completed":
		return "主执行已完成，接下来只需要看验证与归档条件。"
	default:
		return firstNonEmpty(task.StatusReason, "当前阶段未识别。")
	}
}

func humanDefaultNextAction(task adapter.Task, flow DashboardTaskFlow) string {
	switch task.Status {
	case "queued":
		return "等 daemon 继续推进这条任务。"
	case "running":
		return "继续观察业务文件是否开始落地，再看 verify 和 token。"
	case "needs_replan":
		return "先看失败原因，再决定要不要 replan。"
	case "blocked":
		return "先消除阻塞，再恢复执行。"
	case "completed":
		return "核对产物是否满足目标和验收要求。"
	default:
		return "继续观察当前链路。"
	}
}

func extractPrimaryOutputPath(task adapter.Task, view *TaskView) string {
	sources := []string{task.Title, task.Summary, task.Description}
	if view != nil && view.Request != nil {
		sources = append(sources, view.Request.Goal)
		sources = append(sources, view.Request.Contexts...)
	}
	pathRE := regexp.MustCompile(`/Users/\S+`)
	for _, source := range sources {
		match := strings.TrimSpace(pathRE.FindString(source))
		if match == "" {
			continue
		}
		return strings.TrimRight(match, "。；;,")
	}
	return ""
}

func countVisibleFiles(root string) int {
	if strings.TrimSpace(root) == "" {
		return 0
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		count++
	}
	return count
}

func limitStrings(values []string, max int) []string {
	values = uniqueNonEmpty(values)
	if max <= 0 || len(values) <= max {
		return values
	}
	return values[:max]
}

func buildChecklist(view *TaskView) []ChecklistView {
	if view == nil {
		return nil
	}
	items := []ChecklistView{}
	if view.TaskContract != nil {
		for _, item := range view.TaskContract.VerificationChecklist {
			items = append(items, ChecklistView{
				ID:       item.ID,
				Title:    item.Title,
				Status:   normalizeChecklistStatus(item.Status),
				Required: item.Required,
				Detail:   item.Detail,
				Source:   "task-contract",
			})
		}
	}
	if view.Assessment != nil {
		for _, item := range view.Assessment.Scorecard {
			items = append(items, ChecklistView{
				ID:       item.ID,
				Title:    item.Title,
				Status:   normalizeChecklistStatus(item.Status),
				Required: true,
				Detail:   item.Summary,
				Source:   "verify-scorecard",
			})
		}
		for index, item := range view.Assessment.ReviewChecklist {
			items = append(items, ChecklistView{
				ID:       stringValue(item["id"], fmt.Sprintf("review-%d", index+1)),
				Title:    stringValue(item["title"], stringValue(item["check"], fmt.Sprintf("review-%d", index+1))),
				Status:   normalizeChecklistStatus(stringValue(item["status"], stringValue(item["result"], ""))),
				Required: boolValue(item["required"], true),
				Detail:   stringValue(item["detail"], stringValue(item["evidence"], "")),
				Source:   "verify-review",
			})
		}
	}
	return items
}

func buildExecutionChain(
	paths adapter.Paths,
	task adapter.Task,
	view *TaskView,
	events []a2a.Envelope,
	tmuxSummary tmux.Summary,
	checkpointSummary checkpoint.Summary,
	legacySessions legacyCodexSessionSummary,
	warnings *[]string,
) []ExecutionEvent {
	out := []ExecutionEvent{}
	for _, event := range events {
		if event.TaskID != task.TaskID {
			continue
		}
		out = append(out, a2aExecutionEvent(event))
	}
	for _, session := range taskTmuxSessions(tmuxSummary, task.TaskID) {
		out = append(out, ExecutionEvent{
			At:          firstNonEmpty(session.StartedAt, session.FinishedAt),
			Kind:        "tmux.session",
			Title:       "tmux worker session",
			Status:      firstNonEmpty(session.Status, "running"),
			Summary:     firstNonEmpty(session.Cwd, session.LogPath),
			Source:      "tmux-summary",
			TaskID:      task.TaskID,
			DispatchID:  session.DispatchID,
			WorkerID:    session.WorkerID,
			SessionName: session.SessionName,
			Path:        session.LogPath,
		})
	}
	if checkpointState, ok := checkpointSummary.Tasks[task.TaskID]; ok {
		if checkpointState.LatestCheckpoint.TaskID != "" {
			out = append(out, ExecutionEvent{
				At:         checkpointState.LatestCheckpoint.UpdatedAt,
				Kind:       "worker.checkpoint",
				Title:      "checkpoint persisted",
				Status:     checkpointState.LatestCheckpoint.Status,
				Summary:    checkpointState.LatestCheckpoint.Summary,
				Source:     "checkpoint-summary",
				TaskID:     task.TaskID,
				DispatchID: checkpointState.LatestCheckpoint.DispatchID,
				WorkerID:   checkpointState.LatestCheckpoint.LeaseID,
				Path:       checkpointState.LatestCheckpoint.CheckpointRef,
			})
		}
		if checkpointState.LatestOutcome.TaskID != "" {
			out = append(out, ExecutionEvent{
				At:         checkpointState.LatestOutcome.UpdatedAt,
				Kind:       "worker.outcome",
				Title:      "worker outcome",
				Status:     checkpointState.LatestOutcome.Status,
				Summary:    checkpointState.LatestOutcome.Summary,
				Source:     "checkpoint-summary",
				TaskID:     task.TaskID,
				DispatchID: checkpointState.LatestOutcome.DispatchID,
				WorkerID:   checkpointState.LatestOutcome.WorkerID,
				Path:       checkpointState.LatestOutcome.CheckpointRef,
			})
		}
	}
	if session := findLegacySession(legacySessions, task.TaskID); session != nil {
		out = append(out, ExecutionEvent{
			At:          firstNonEmpty(session.UpdatedAt, session.CreatedAt),
			Kind:        "legacy.session",
			Title:       "legacy codex session",
			Status:      firstNonEmpty(session.LastStatus, "unknown"),
			Summary:     firstNonEmpty(session.LastSummary, session.Model),
			Source:      "codex-session-summary",
			TaskID:      task.TaskID,
			DispatchID:  session.LastDispatchID,
			SessionName: session.ID,
		})
	}
	legacyOutcomePath := legacyOutcomePath(paths.Root, task.TaskID)
	if outcome, ok := loadLegacyOutcome(legacyOutcomePath); ok {
		out = append(out, ExecutionEvent{
			At:         firstNonEmpty(outcome.FinishedAt, outcome.GeneratedAt, outcome.StartedAt),
			Kind:       "legacy.outcome",
			Title:      "legacy worker burst outcome",
			Status:     outcome.Status,
			Summary:    firstNonEmpty(outcome.Summary, strings.TrimSpace(outcome.Stderr)),
			Source:     "checkpoint-outcome",
			TaskID:     task.TaskID,
			DispatchID: task.LastDispatchID,
			Path:       legacyOutcomePath,
		})
		out = append(out, parseExecutionLogEvents(legacyOutcomePath, outcome.Stdout, task.TaskID)...)
	}
	if view != nil && view.Tmux != nil && strings.TrimSpace(view.Tmux.LogPath) != "" {
		payload, err := os.ReadFile(view.Tmux.LogPath)
		if err == nil {
			out = append(out, parseExecutionLogEvents(view.Tmux.LogPath, string(payload), task.TaskID)...)
		} else {
			*warnings = append(*warnings, fmt.Sprintf("task %s tmux log could not be read: %v", task.TaskID, err))
		}
	}
	return out
}

func taskTmuxSessions(summary tmux.Summary, taskID string) []tmux.SessionState {
	if strings.TrimSpace(taskID) == "" {
		return nil
	}
	sessions := make([]tmux.SessionState, 0, len(summary.Sessions))
	for _, session := range summary.Sessions {
		if session.TaskID != taskID {
			continue
		}
		sessions = append(sessions, session)
	}
	sort.SliceStable(sessions, func(i, j int) bool {
		left := firstNonEmpty(sessions[i].StartedAt, sessions[i].FinishedAt, sessions[i].SessionName)
		right := firstNonEmpty(sessions[j].StartedAt, sessions[j].FinishedAt, sessions[j].SessionName)
		return left < right
	})
	return sessions
}

func collectTokenUsage(paths adapter.Paths, task adapter.Task, view *TaskView, warnings *[]string) TokenUsage {
	usage := TokenUsage{}
	if view != nil && view.Tmux != nil && strings.TrimSpace(view.Tmux.LogPath) != "" {
		payload, err := os.ReadFile(view.Tmux.LogPath)
		if err == nil {
			usage = mergeTokenUsage(usage, parseTokenUsage(view.Tmux.LogPath, string(payload)))
		}
	}
	outcomePath := legacyOutcomePath(paths.Root, task.TaskID)
	if outcome, ok := loadLegacyOutcome(outcomePath); ok {
		usage = mergeTokenUsage(usage, parseTokenUsage(outcomePath, outcome.Stdout))
	}
	if usage.Turns == 0 {
		*warnings = append(*warnings, fmt.Sprintf("task %s has no observable token usage yet; wait for turn.completed log lines or checkpoint outcome stdout", task.TaskID))
	}
	return usage
}

func buildDashboardThreads(tasks []adapter.Task, requestState dashboardRequestState) []DashboardThread {
	threadKeys := map[string]struct{}{}
	for key := range requestState.Threads.Threads {
		threadKeys[key] = struct{}{}
	}
	for key := range requestState.Map.ThreadToRequests {
		threadKeys[key] = struct{}{}
	}
	for key := range requestState.Map.ThreadToTasks {
		threadKeys[key] = struct{}{}
	}
	for _, task := range tasks {
		if strings.TrimSpace(task.ThreadKey) != "" {
			threadKeys[task.ThreadKey] = struct{}{}
		}
	}
	threads := make([]DashboardThread, 0, len(threadKeys))
	for threadKey := range threadKeys {
		entry := requestState.Threads.Threads[threadKey]
		taskIDs := append([]string{}, requestState.Map.ThreadToTasks[threadKey]...)
		if len(taskIDs) == 0 {
			taskIDs = append(taskIDs, entry.TaskIDs...)
		}
		if len(taskIDs) == 0 {
			for _, task := range tasks {
				if task.ThreadKey == threadKey {
					taskIDs = append(taskIDs, task.TaskID)
				}
			}
		}
		taskIDs = uniqueNonEmpty(taskIDs)
		requestIDs := append([]string{}, requestState.Map.ThreadToRequests[threadKey]...)
		requestIDs = append(requestIDs, entry.RequestIDs...)
		if len(requestIDs) == 0 {
			for id, record := range requestState.ByID {
				if firstNonEmpty(record.ThreadKey, record.TargetThreadKey) == threadKey {
					requestIDs = append(requestIDs, id)
				}
			}
		}
		requestIDs = uniqueNonEmpty(requestIDs)
		sortRequestIDs(requestIDs)
		landings := make([]RequestLanding, 0, len(requestIDs))
		for _, requestID := range requestIDs {
			record, ok := requestState.ByID[requestID]
			if !ok {
				continue
			}
			taskStatus := ""
			for _, task := range tasks {
				if task.TaskID == firstNonEmpty(requestState.Map.RequestToTask[requestID], record.TaskID) {
					taskStatus = task.Status
					break
				}
			}
			landings = append(landings, requestLandingFromRecord(record, taskStatus))
		}
		threads = append(threads, DashboardThread{
			ThreadKey:            threadKey,
			Status:               entry.Status,
			PlanEpoch:            entry.PlanEpoch,
			CurrentPlanEpoch:     entry.CurrentPlanEpoch,
			LatestValidPlanEpoch: entry.LatestValidPlanEpoch,
			LatestRequestID:      firstNonEmpty(entry.LatestRequestID, latestRequestID(landings)),
			LatestTaskID:         firstNonEmpty(entry.LatestTaskID, latestTaskID(taskIDs)),
			RequestCount:         len(requestIDs),
			TaskCount:            len(taskIDs),
			RequestLandings:      landings,
			TaskIDs:              taskIDs,
		})
	}
	sort.SliceStable(threads, func(i, j int) bool {
		left := firstNonEmpty(threads[i].LatestRequestID, threads[i].ThreadKey)
		right := firstNonEmpty(threads[j].LatestRequestID, threads[j].ThreadKey)
		return left > right
	})
	return threads
}

func loadDashboardRequestState(paths adapter.Paths) (dashboardRequestState, error) {
	stateView := dashboardRequestState{
		ByID: map[string]runtime.RequestRecord{},
	}
	if _, err := state.LoadJSONIfExists(paths.RequestSummaryPath, &stateView.Summary); err != nil {
		return dashboardRequestState{}, err
	}
	if _, ok, err := loadRequestIndex(paths.RequestIndexPath); err != nil {
		return dashboardRequestState{}, err
	} else if ok {
		index, _, err := loadRequestIndex(paths.RequestIndexPath)
		if err != nil {
			return dashboardRequestState{}, err
		}
		stateView.Index = index
	}
	mapping, err := loadDashboardRequestTaskMap(paths.RequestTaskMapPath)
	if err != nil {
		return dashboardRequestState{}, err
	}
	stateView.Map = mapping
	threadState, err := loadDashboardThreadState(paths.ThreadStatePath)
	if err != nil {
		return dashboardRequestState{}, err
	}
	stateView.Threads = threadState
	queueRecords, err := loadQueueRecords(paths.QueuePath)
	if err != nil {
		return dashboardRequestState{}, err
	}
	for id, record := range queueRecords {
		stateView.ByID[id] = record
	}
	for id, record := range stateView.Index.RequestsByID {
		stateView.ByID[id] = record
	}
	return stateView, nil
}

func loadDashboardRequestTaskMap(path string) (runtime.RequestTaskMap, error) {
	mapping := runtime.RequestTaskMap{
		RequestToTask:    map[string]string{},
		RequestToThread:  map[string]string{},
		TaskToRequests:   map[string][]string{},
		ThreadToRequests: map[string][]string{},
		ThreadToTasks:    map[string][]string{},
	}
	if _, err := state.LoadJSONIfExists(path, &mapping); err != nil {
		return runtime.RequestTaskMap{}, err
	}
	if mapping.RequestToTask == nil {
		mapping.RequestToTask = map[string]string{}
	}
	if mapping.RequestToThread == nil {
		mapping.RequestToThread = map[string]string{}
	}
	if mapping.TaskToRequests == nil {
		mapping.TaskToRequests = map[string][]string{}
	}
	if mapping.ThreadToRequests == nil {
		mapping.ThreadToRequests = map[string][]string{}
	}
	if mapping.ThreadToTasks == nil {
		mapping.ThreadToTasks = map[string][]string{}
	}
	return mapping, nil
}

func loadDashboardThreadState(path string) (runtime.ThreadState, error) {
	threadState := runtime.ThreadState{Threads: map[string]runtime.ThreadEntry{}}
	if _, err := state.LoadJSONIfExists(path, &threadState); err != nil {
		return runtime.ThreadState{}, err
	}
	if threadState.Threads == nil {
		threadState.Threads = map[string]runtime.ThreadEntry{}
	}
	return threadState, nil
}

func loadQueueRecords(path string) (map[string]runtime.RequestRecord, error) {
	records := map[string]runtime.RequestRecord{}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return records, nil
		}
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	buffer := make([]byte, 0, 1024)
	scanner.Buffer(buffer, 1024*1024)
	for scanner.Scan() {
		var record runtime.RequestRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue
		}
		if strings.TrimSpace(record.RequestID) == "" {
			continue
		}
		records[record.RequestID] = record
	}
	return records, scanner.Err()
}

func loadCheckpointSummary(path string) (checkpoint.Summary, error) {
	summary := checkpoint.Summary{
		Tasks:      map[string]checkpoint.TaskState{},
		ByDispatch: map[string]string{},
	}
	if _, err := state.LoadJSONIfExists(path, &summary); err != nil {
		return checkpoint.Summary{}, err
	}
	if summary.Tasks == nil {
		summary.Tasks = map[string]checkpoint.TaskState{}
	}
	if summary.ByDispatch == nil {
		summary.ByDispatch = map[string]string{}
	}
	return summary, nil
}

func loadLegacyCodexSessions(path string) (legacyCodexSessionSummary, error) {
	summary := legacyCodexSessionSummary{Sessions: map[string]legacyCodexSession{}}
	if _, err := state.LoadJSONIfExists(path, &summary); err != nil {
		return legacyCodexSessionSummary{}, err
	}
	if summary.Sessions == nil {
		summary.Sessions = map[string]legacyCodexSession{}
	}
	return summary, nil
}

func buildDashboardEnvironment() DashboardEnvironment {
	codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME"))
	if codexHome == "" {
		if home, err := os.UserHomeDir(); err == nil {
			codexHome = filepath.Join(home, ".codex")
		}
	}
	tools := []ToolStatus{}
	for _, name := range []string{"harness", "codex", "tmux", "jq", "rg"} {
		path, err := exec.LookPath(name)
		tools = append(tools, ToolStatus{
			Name:  name,
			Found: err == nil,
			Path:  strings.TrimSpace(path),
		})
	}
	return DashboardEnvironment{
		CodexHome: codexHome,
		Tools:     tools,
	}
}

func a2aExecutionEvent(event a2a.Envelope) ExecutionEvent {
	payload := map[string]any{}
	_ = json.Unmarshal(event.Payload, &payload)
	title := strings.ReplaceAll(event.Kind, ".", " ")
	status := stringValue(payload["status"], "")
	summary := firstNonEmpty(
		stringValue(payload["summary"], ""),
		stringValue(payload["route"], ""),
		stringValue(payload["promptRef"], ""),
		strings.Join(event.ReasonCodes, ", "),
	)
	if event.Kind == "dispatch.issued" {
		title = "dispatch issued"
		if budget, ok := payload["budget"].(map[string]any); ok {
			summary = fmt.Sprintf("budget turns=%s minutes=%s toolCalls=%s",
				trimString(fmt.Sprint(budget["maxTurns"])),
				trimString(fmt.Sprint(budget["maxMinutes"])),
				trimString(fmt.Sprint(budget["maxToolCalls"])),
			)
		}
	}
	if event.Kind == "route.decided" {
		title = "route decided"
	}
	if event.Kind == "worker.claimed" {
		title = "worker claimed"
	}
	if event.Kind == "worker.checkpoint" {
		title = "checkpoint persisted"
	}
	if event.Kind == "worker.outcome" {
		title = "worker outcome"
	}
	if event.Kind == "verification.completed" {
		title = "verification completed"
	}
	if event.Kind == "replan.emitted" {
		title = "replan emitted"
	}
	if event.Kind == "task.completed" {
		title = "task completed"
	}
	if event.Kind == "task.blocked" {
		title = "task blocked"
	}
	return ExecutionEvent{
		At:         event.CreatedAt,
		Kind:       event.Kind,
		Title:      title,
		Status:     status,
		Summary:    summary,
		Source:     "a2a",
		TaskID:     event.TaskID,
		DispatchID: stringValue(payload["dispatchId"], ""),
		WorkerID:   firstNonEmpty(event.WorkerID, stringValue(payload["workerId"], "")),
	}
}

func requestLandingFromRecord(record runtime.RequestRecord, taskStatus string) RequestLanding {
	return RequestLanding{
		RequestID:             record.RequestID,
		TaskID:                record.TaskID,
		TaskStatus:            taskStatus,
		BindingAction:         record.BindingAction,
		NormalizedIntentClass: record.NormalizedIntentClass,
		FrontDoorTriage:       record.FrontDoorTriage,
		Goal:                  record.Goal,
		Contexts:              append([]string(nil), record.Contexts...),
		CreatedAt:             record.CreatedAt,
		ClassificationReason:  record.ClassificationReason,
	}
}

func parseTokenUsage(sourcePath, raw string) TokenUsage {
	usage := TokenUsage{}
	for _, event := range parseExecutionJSONLines(raw) {
		if event.Type != "turn.completed" {
			continue
		}
		usage.InputTokens += event.Usage.InputTokens
		usage.CachedInputTokens += event.Usage.CachedInputTokens
		usage.OutputTokens += event.Usage.OutputTokens
		usage.Turns++
	}
	if usage.Turns > 0 && strings.TrimSpace(sourcePath) != "" {
		usage.SourcePaths = append(usage.SourcePaths, sourcePath)
	}
	return usage
}

func parseExecutionLogEvents(sourcePath, raw, taskID string) []ExecutionEvent {
	events := []ExecutionEvent{}
	for _, event := range parseExecutionJSONLines(raw) {
		switch event.Type {
		case "thread.started":
			events = append(events, ExecutionEvent{
				Kind:        event.Type,
				Title:       "worker thread started",
				Status:      "started",
				Summary:     firstNonEmpty(event.ThreadID, sourcePath),
				Source:      "runner-log",
				TaskID:      taskID,
				SessionName: event.ThreadID,
				Path:        sourcePath,
			})
		case "turn.started":
			events = append(events, ExecutionEvent{
				Kind:   event.Type,
				Title:  "turn started",
				Status: "running",
				Source: "runner-log",
				TaskID: taskID,
				Path:   sourcePath,
			})
		case "turn.completed":
			events = append(events, ExecutionEvent{
				Kind:    event.Type,
				Title:   "turn completed",
				Status:  "completed",
				Summary: fmt.Sprintf("input=%d cached=%d output=%d", event.Usage.InputTokens, event.Usage.CachedInputTokens, event.Usage.OutputTokens),
				Source:  "runner-log",
				TaskID:  taskID,
				Path:    sourcePath,
			})
		}
	}
	return events
}

type executionJSONEvent struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id,omitempty"`
	Usage    struct {
		InputTokens       int64 `json:"input_tokens"`
		CachedInputTokens int64 `json:"cached_input_tokens"`
		OutputTokens      int64 `json:"output_tokens"`
	} `json:"usage,omitempty"`
}

func parseExecutionJSONLines(raw string) []executionJSONEvent {
	result := []executionJSONEvent{}
	scanner := bufio.NewScanner(strings.NewReader(raw))
	buffer := make([]byte, 0, 1024)
	scanner.Buffer(buffer, 8*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		payload := extractJSONPayload(line)
		if payload == "" {
			continue
		}
		var event executionJSONEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}
		if strings.TrimSpace(event.Type) == "" {
			continue
		}
		result = append(result, event)
	}
	return result
}

func extractJSONPayload(line string) string {
	start := strings.Index(line, "{\"type\"")
	if start < 0 {
		return ""
	}
	end := strings.LastIndex(line, "}")
	if end <= start {
		return ""
	}
	return line[start : end+1]
}

func legacyOutcomePath(root, taskID string) string {
	return filepath.Join(root, ".harness", "checkpoints", taskID, "outcome.json")
}

func loadLegacyOutcome(path string) (legacyOutcome, bool) {
	var outcome legacyOutcome
	ok, err := state.LoadJSONIfExists(path, &outcome)
	if err != nil || !ok {
		return legacyOutcome{}, false
	}
	return outcome, true
}

func findLegacySession(summary legacyCodexSessionSummary, taskID string) *legacyCodexSession {
	for _, session := range summary.Sessions {
		if session.TaskID == taskID {
			copy := session
			return &copy
		}
	}
	return nil
}

func mergeTokenUsage(left, right TokenUsage) TokenUsage {
	merged := TokenUsage{
		InputTokens:       left.InputTokens + right.InputTokens,
		CachedInputTokens: left.CachedInputTokens + right.CachedInputTokens,
		OutputTokens:      left.OutputTokens + right.OutputTokens,
		Turns:             left.Turns + right.Turns,
		SourcePaths:       uniqueNonEmpty(append(append([]string{}, left.SourcePaths...), right.SourcePaths...)),
	}
	return merged
}

func sortExecutionEvents(events []ExecutionEvent) {
	sort.SliceStable(events, func(i, j int) bool {
		left := firstNonEmpty(events[i].At, events[i].Kind, events[i].Title)
		right := firstNonEmpty(events[j].At, events[j].Kind, events[j].Title)
		return left > right
	})
}

func sortRequestIDs(ids []string) {
	sort.SliceStable(ids, func(i, j int) bool { return ids[i] < ids[j] })
}

func normalizeChecklistStatus(value string) string {
	lowered := strings.ToLower(strings.TrimSpace(value))
	switch lowered {
	case "pass", "passed", "ok", "verified", "satisfied":
		return "pass"
	case "warn", "warning", "degraded":
		return "warn"
	case "fail", "failed", "blocked":
		return "fail"
	default:
		return lowered
	}
}

func summarizeOwnedPaths(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	if len(paths) == 1 {
		return "owned path: " + paths[0]
	}
	return fmt.Sprintf("owned paths: %s (+%d more)", paths[0], len(paths)-1)
}

func summarizeList(label string, values []string, maxCount int) string {
	values = uniqueNonEmpty(values)
	if len(values) == 0 {
		return ""
	}
	if len(values) > maxCount {
		return fmt.Sprintf("%s: %s (+%d more)", label, strings.Join(values[:maxCount], ", "), len(values)-maxCount)
	}
	return fmt.Sprintf("%s: %s", label, strings.Join(values, ", "))
}

func summarizeVerificationPlan(plan map[string]any) string {
	if len(plan) == 0 {
		return "verification plan is not persisted yet"
	}
	commands := 0
	if raw, ok := plan["commands"].([]any); ok {
		commands = len(raw)
	}
	if raw, ok := plan["ruleIds"].([]any); ok && len(raw) > 0 {
		return fmt.Sprintf("verification rules=%d commands=%d", len(raw), commands)
	}
	return fmt.Sprintf("verification commands=%d", commands)
}

func stringValue(values ...any) string {
	for _, value := range values {
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		}
	}
	return ""
}

func boolValue(value any, fallback bool) bool {
	typed, ok := value.(bool)
	if !ok {
		return fallback
	}
	return typed
}

func latestRequestID(landings []RequestLanding) string {
	if len(landings) == 0 {
		return ""
	}
	return landings[len(landings)-1].RequestID
}

func latestTaskID(taskIDs []string) string {
	if len(taskIDs) == 0 {
		return ""
	}
	return taskIDs[len(taskIDs)-1]
}

func trimString(value string) string {
	return strings.TrimSpace(strings.Trim(value, "<>"))
}

func pathsafe(view *TaskView, task adapter.Task) string {
	if view != nil && view.Planning != nil && strings.TrimSpace(view.Planning.DispatchTicketPath) != "" {
		return view.Planning.DispatchTicketPath
	}
	return task.TaskID
}

func uniqueNonEmpty(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
