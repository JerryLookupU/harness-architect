package worker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"klein-harness/internal/adapter"
	"klein-harness/internal/dispatch"
	"klein-harness/internal/orchestration"
	"klein-harness/internal/state"
	"klein-harness/internal/verify"
)

type verificationManifest struct {
	Rules []verificationRule `json:"rules"`
}

type verificationRule struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Exec         string `json:"exec"`
	Timeout      int    `json:"timeout"`
	ReadOnlySafe bool   `json:"readOnlySafe"`
}

type executionTaskSpec struct {
	Title                string
	Summary              string
	TaskGroupID          string
	BatchLabel           string
	EntityBatch          []string
	OutputTargets        []string
	SharedContextSummary string
	InScope              []string
	DoneCriteria         []string
}

type corpusPlanningInfo struct {
	OutputDir        string
	OutputFile       string
	SubjectLabel     string
	SubjectCount     int
	MinChars         int
	FileExtension    string
	IndexFile        string
	RequiredSections []string
	RequiresIndex    bool
	SingleDocument   bool
}

type DispatchBundle struct {
	TicketPath         string
	WorkerSpecPath     string
	PromptPath         string
	PlanningTracePath  string
	AcceptedPacketPath string
	TaskContractPath   string
	ArtifactDir        string
	CommandBanner      string
}

func Prepare(root string, ticket dispatch.Ticket, leaseID string) (DispatchBundle, error) {
	paths, err := adapter.Resolve(root)
	if err != nil {
		return DispatchBundle{}, err
	}
	task, err := adapter.LoadTask(root, ticket.TaskID)
	if err != nil {
		return DispatchBundle{}, err
	}
	projectMeta, err := adapter.LoadProjectMeta(root)
	if err != nil {
		return DispatchBundle{}, err
	}
	verifyCommands, err := verificationCommands(paths.VerificationRulesPath, task.VerificationRuleIDs)
	if err != nil {
		return DispatchBundle{}, err
	}
	constraintSystem := orchestration.DefaultConstraintSystem(paths.Root, unique(ticket.ReasonCodes))
	constraintSystem, err = verify.EvolveConstraintSystemFromFeedback(paths.Root, task, constraintSystem)
	if err != nil {
		return DispatchBundle{}, err
	}
	constraintPath := orchestration.ConstraintSnapshotPath(paths.Root, task.TaskID)
	softRules, hardRules := orchestration.SplitConstraintRules(constraintSystem)
	if err := orchestration.WriteConstraintSnapshot(constraintPath, orchestration.ConstraintSnapshot{
		SchemaVersion:    "kh.constraint-snapshot.v1",
		Generator:        "kh-worker-supervisor",
		GeneratedAt:      nowUTC(),
		TaskID:           task.TaskID,
		DispatchID:       ticket.DispatchID,
		PlanEpoch:        task.PlanEpoch,
		ConstraintSystem: constraintSystem,
		SoftRules:        softRules,
		HardRules:        hardRules,
	}); err != nil {
		return DispatchBundle{}, err
	}
	hookPlan := verify.BuildHookPlan(root, task, ticket, verifyCommands, constraintPath, constraintSystem)
	feedbackSummary, _ := verify.LoadFeedbackSummary(root)
	taskFeedback, hasTaskFeedback := verify.CurrentTaskFeedback(feedbackSummary, task.TaskID)
	executionCwd := adapter.TaskCWD(paths, task)
	artifactDir := filepath.Join(paths.ArtifactsDir, task.TaskID, ticket.DispatchID)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return DispatchBundle{}, err
	}
	ticketPath := filepath.Join(paths.StateDir, fmt.Sprintf("dispatch-ticket-%s.json", task.TaskID))
	workerSpecPath := filepath.Join(artifactDir, "worker-spec.json")
	acceptedPacketPath := orchestration.AcceptedPacketPath(paths.Root, task.TaskID)
	taskContractPath := orchestration.TaskContractPath(artifactDir)
	sharedContextPath := filepath.Join(artifactDir, "shared-context.json")
	promptPath := filepath.Join(paths.StateDir, fmt.Sprintf("runner-prompt-%s.md", task.TaskID))
	planningTracePath := filepath.Join(paths.StateDir, fmt.Sprintf("planning-trace-%s.md", task.TaskID))
	repoRole := projectMeta.RepoRole
	if repoRole == "" {
		repoRole = "target_repo"
	}
	directTargetEditAllowed := true
	if projectMeta.DirectTargetEditAllowed != nil {
		directTargetEditAllowed = *projectMeta.DirectTargetEditAllowed
	}
	intentFingerprint := stableFingerprint(
		task.TaskID,
		fmt.Sprintf("%d", task.PlanEpoch),
		task.Title,
		task.Summary,
		strings.Join(task.OwnedPaths, "|"),
		strings.Join(task.VerificationRuleIDs, "|"),
	)
	packetSynthesis := orchestration.DefaultPacketSynthesisLoop(paths.Root)
	methodology := orchestration.DefaultMethodologyContract(paths.Root, unique(ticket.ReasonCodes))
	judgeDecision := orchestration.DefaultJudgeDecision(packetSynthesis, methodology, unique(ticket.ReasonCodes))
	executionLoop := orchestration.DefaultExecutionLoopContract(paths.Root, unique(ticket.ReasonCodes))
	activeSkills := executionLoop.ActiveSkills
	skillHints := executionLoop.SkillHints
	workerSpec := map[string]any{
		"schemaVersion":     "kh.worker-spec.v1",
		"generator":         "kh-worker-supervisor",
		"generatedAt":       nowUTC(),
		"dispatchId":        ticket.DispatchID,
		"taskId":            task.TaskID,
		"threadKey":         task.ThreadKey,
		"planEpoch":         task.PlanEpoch,
		"attempt":           ticket.Attempt,
		"reasonCodes":       unique(ticket.ReasonCodes),
		"policyTags":        policyTags(ticket.ReasonCodes),
		"activeSkills":      activeSkills,
		"skillHints":        skillHints,
		"objective":         coalesce(task.Summary, task.Title),
		"selectedPlan":      coalesce(task.Description, task.Summary, task.Title),
		"constraints":       taskConstraints(task),
		"ownedPaths":        unique(task.OwnedPaths),
		"blockedPaths":      unique(task.ForbiddenPaths),
		"taskBudget":        ticket.Budget,
		"acceptanceMarkers": hookPlan.AcceptanceMarkers,
		"verificationPlan": map[string]any{
			"ruleIds":  unique(task.VerificationRuleIDs),
			"commands": verifyCommands,
		},
		"validationHooks":    hookPlan.Hooks,
		"learningHints":      hookPlan.LearningHints,
		"outerLoopMemory":    taskFeedback,
		"constraintPath":     constraintPath,
		"acceptedPacketPath": acceptedPacketPath,
		"taskContractPath":   taskContractPath,
		"decisionRationale":  coalesce(task.Description, task.Summary),
		"replanTriggers": []string{
			"verification_failed",
			"acceptance_markers_missing",
			"owned_paths_conflict",
			"authority_boundary_conflict",
		},
		"rollbackHints": []string{
			"leave_task_local_artifacts_intact",
			"preserve_checkpoint_for_supervisor",
			"handoff_before_exit_when_blocked",
		},
	}
	acceptedPacket := buildAcceptedPacket(task, ticket, judgeDecision, hookPlan, verifyCommands)
	if acceptedPacket.SharedContext != nil {
		if err := writeJSON(sharedContextPath, acceptedPacket.SharedContext); err != nil {
			return DispatchBundle{}, err
		}
	}
	plannerCandidates := buildPlannerCandidates(task, ticket, packetSynthesis, judgeDecision, acceptedPacket, hookPlan, verifyCommands)
	acceptedPacketRevision, err := state.CurrentRevision(acceptedPacketPath)
	if err != nil {
		return DispatchBundle{}, err
	}
	if err := orchestration.WriteAcceptedPacketCAS(acceptedPacketPath, acceptedPacket, acceptedPacketRevision); err != nil {
		return DispatchBundle{}, err
	}
	taskContract := buildTaskContract(paths.Root, task, ticket, acceptedPacket, hookPlan, acceptedPacketPath)
	taskContractRevision, err := state.CurrentRevision(taskContractPath)
	if err != nil {
		return DispatchBundle{}, err
	}
	if err := orchestration.WriteTaskContractCAS(taskContractPath, taskContract, taskContractRevision); err != nil {
		return DispatchBundle{}, err
	}
	workerSpec["acceptedPacketId"] = acceptedPacket.PacketID
	workerSpec["contractId"] = taskContract.ContractID
	workerSpec["executionSliceId"] = taskContract.ExecutionSliceID
	workerSpec["sliceInScope"] = taskContract.InScope
	workerSpec["sharedContextPath"] = sharedContextPath
	workerSpec["sharedContext"] = acceptedPacket.SharedContext
	if selectedTask := selectedExecutionTaskByID(acceptedPacket.ExecutionTasks, taskContract.ExecutionSliceID); selectedTask != nil {
		workerSpec["taskGroupId"] = selectedTask.TaskGroupID
		workerSpec["batchLabel"] = selectedTask.BatchLabel
		workerSpec["entityBatch"] = selectedTask.EntityBatch
		workerSpec["outputTargets"] = selectedTask.OutputTargets
	}
	commandBanner := tmuxCommandBanner(task, acceptedPacket.ExecutionTasks, taskContract)
	workerSpec["tmuxCommandBanner"] = commandBanner
	workerSpec["tmuxCommandProtocol"] = "[harness:<task-id>] <node-task-description>"
	if err := writeJSON(workerSpecPath, workerSpec); err != nil {
		return DispatchBundle{}, err
	}
	planningTrace := orchestration.RenderPlanningTrace(
		task.TaskID,
		task.ThreadKey,
		task.PlanEpoch,
		task.ResumeStrategy,
		task.RoutingModel,
		task.ExecutionModel,
		unique(task.PromptStages),
		unique(ticket.ReasonCodes),
		packetSynthesis,
	)
	if err := os.WriteFile(planningTracePath, []byte(planningTrace), 0o644); err != nil {
		return DispatchBundle{}, err
	}
	dispatchTicket := map[string]any{
		"schemaVersion":           "kh.dispatch-ticket.v1",
		"generator":               "kh-worker-supervisor",
		"generatedAt":             nowUTC(),
		"dispatchId":              ticket.DispatchID,
		"idempotencyKey":          ticket.IdempotencyKey,
		"leaseId":                 leaseID,
		"taskId":                  task.TaskID,
		"threadKey":               task.ThreadKey,
		"planEpoch":               task.PlanEpoch,
		"attempt":                 ticket.Attempt,
		"intentFingerprint":       intentFingerprint,
		"taskKind":                task.Kind,
		"workerMode":              task.WorkerMode,
		"roleHint":                task.RoleHint,
		"repoRole":                repoRole,
		"directTargetEditAllowed": directTargetEditAllowed,
		"projectRoot":             paths.Root,
		"executionCwd":            executionCwd,
		"worktreePath":            coalesce(task.Dispatch.WorktreePath, task.WorktreePath),
		"branchName":              coalesce(task.Dispatch.BranchName, task.BranchName),
		"diffBase":                coalesce(task.Dispatch.DiffBase, task.DiffBase, task.BaseRef),
		"resumeStrategy":          task.ResumeStrategy,
		"sessionId":               ticket.ResumeSessionID,
		"routingModel":            task.RoutingModel,
		"executionModel":          task.ExecutionModel,
		"orchestrationSessionId":  task.OrchestrationSessionID,
		"promptStages":            unique(task.PromptStages),
		"reasonCodes":             unique(ticket.ReasonCodes),
		"policyTags":              policyTags(ticket.ReasonCodes),
		"activeSkills":            activeSkills,
		"skillHints":              skillHints,
		"allowedWriteGlobs":       unique(task.OwnedPaths),
		"blockedWriteGlobs":       unique(task.ForbiddenPaths),
		"artifactDir":             artifactDir,
		"planningTracePath":       planningTracePath,
		"acceptedPacketPath":      acceptedPacketPath,
		"taskContractPath":        taskContractPath,
		"sharedContextPath":       sharedContextPath,
		"executionSliceId":        taskContract.ExecutionSliceID,
		"workerSpecPath":          workerSpecPath,
		"workerSpec":              workerSpec,
		"tmuxCommandBanner":       commandBanner,
		"tmuxCommandProtocol":     "[harness:<task-id>] <node-task-description>",
		"acceptedPacket":          acceptedPacket,
		"sharedContext":           acceptedPacket.SharedContext,
		"taskContract":            taskContract,
		"artifacts": map[string]string{
			"sharedContext":  sharedContextPath,
			"acceptedPacket": acceptedPacketPath,
			"taskContract":   taskContractPath,
			"workerSpec":     workerSpecPath,
			"workerResult":   filepath.Join(artifactDir, "worker-result.json"),
			"verify":         filepath.Join(artifactDir, "verify.json"),
			"handoff":        filepath.Join(artifactDir, "handoff.md"),
		},
		"authorityBoundary": map[string]any{
			"routeFirstDispatchSecond":  true,
			"workerMayWriteGlobalState": false,
			"workerMayMergeOrArchive":   false,
			"completionOwnedByRuntime":  true,
			"completionGatePath":        filepath.Join(paths.StateDir, "completion-gate.json"),
		},
		"verification": map[string]any{
			"ruleIds":  unique(task.VerificationRuleIDs),
			"commands": verifyCommands,
		},
		"validationHooks":   hookPlan.Hooks,
		"learningHints":     hookPlan.LearningHints,
		"outerLoopMemory":   taskFeedback,
		"constraintPath":    constraintPath,
		"methodology":       methodology,
		"judgeDecision":     judgeDecision,
		"executionLoop":     executionLoop,
		"constraintSystem":  constraintSystem,
		"packetSynthesis":   packetSynthesis,
		"plannerCandidates": plannerCandidates,
		"runtimeRefs": mergeStringMaps(
			map[string]string{
				"promptRef":       ticket.PromptRef,
				"promptPath":      promptPath,
				"workerSpec":      workerSpecPath,
				"planningTrace":   planningTracePath,
				"feedbackSummary": filepath.Join(paths.StateDir, "feedback-summary.json"),
				"constraints":     constraintPath,
				"acceptedPacket":  acceptedPacketPath,
				"taskContract":    taskContractPath,
			},
			orchestration.PromptRefs(paths.Root),
		),
	}
	if err := writeJSON(ticketPath, dispatchTicket); err != nil {
		return DispatchBundle{}, err
	}
	var taskFeedbackPtr *verify.TaskFeedbackSummary
	if hasTaskFeedback {
		copy := taskFeedback
		taskFeedbackPtr = &copy
	}
	prompt := buildPrompt(ticketPath, workerSpecPath, sharedContextPath, acceptedPacketPath, taskContractPath, taskContract.ExecutionSliceID, planningTracePath, constraintPath, artifactDir, filepath.Join(paths.StateDir, "feedback-summary.json"), task, ticket, packetSynthesis, executionLoop, hookPlan, taskFeedbackPtr, constraintSystem, acceptedPacket.SharedContext, selectedExecutionTaskByID(acceptedPacket.ExecutionTasks, taskContract.ExecutionSliceID))
	if err := os.WriteFile(promptPath, []byte(prompt), 0o644); err != nil {
		return DispatchBundle{}, err
	}
	return DispatchBundle{
		TicketPath:         ticketPath,
		WorkerSpecPath:     workerSpecPath,
		PromptPath:         promptPath,
		PlanningTracePath:  planningTracePath,
		AcceptedPacketPath: acceptedPacketPath,
		TaskContractPath:   taskContractPath,
		ArtifactDir:        artifactDir,
		CommandBanner:      commandBanner,
	}, nil
}

func verificationCommands(path string, ruleIDs []string) ([]map[string]any, error) {
	if len(ruleIDs) == 0 {
		return []map[string]any{}, nil
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	var manifest verificationManifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return nil, err
	}
	index := map[string]verificationRule{}
	for _, rule := range manifest.Rules {
		if rule.ID != "" {
			index[rule.ID] = rule
		}
	}
	commands := make([]map[string]any, 0, len(ruleIDs))
	for _, ruleID := range ruleIDs {
		rule, ok := index[ruleID]
		if !ok {
			continue
		}
		commands = append(commands, map[string]any{
			"ruleId":       rule.ID,
			"title":        rule.Title,
			"exec":         rule.Exec,
			"timeout":      rule.Timeout,
			"readOnlySafe": rule.ReadOnlySafe,
		})
	}
	return commands, nil
}

func buildPrompt(ticketPath, workerSpecPath, sharedContextPath, acceptedPacketPath, taskContractPath, executionSliceID, planningTracePath, constraintPath, artifactDir, feedbackSummaryPath string, task adapter.Task, ticket dispatch.Ticket, packetSynthesis orchestration.PacketSynthesisLoop, executionLoop orchestration.ExecutionLoopContract, hookPlan verify.HookPlan, taskFeedback *verify.TaskFeedbackSummary, constraintSystem orchestration.ConstraintSystem, sharedContext *orchestration.SharedTaskGroupContext, selectedTask *orchestration.ExecutionTask) string {
	routePolicyTags := policyTags(ticket.ReasonCodes)
	lines := []string{
		"You are the Klein worker for exactly one bound task inside a repo-local closed-loop runtime.",
		"",
		"Dispatch summary:",
		fmt.Sprintf("- taskId: %s", task.TaskID),
		fmt.Sprintf("- executionSliceId: %s", executionSliceID),
		fmt.Sprintf("- title: %s", task.Title),
		fmt.Sprintf("- objective: %s", coalesce(task.Summary, task.Title)),
		fmt.Sprintf("- activeSkills: %s", strings.Join(executionLoop.ActiveSkills, ", ")),
		fmt.Sprintf("- orchestrationMode: metadata-backed B3Ehive (%d planners + 1 judge)", packetSynthesis.PlannerCount),
		"- skills are entry guidance for Codex; runtime-owned files inside .harness remain authoritative.",
		"",
		"Required reads before execution:",
		fmt.Sprintf("1. Read the immutable dispatch ticket: %s", ticketPath),
		fmt.Sprintf("2. Read the task-local worker spec: %s", workerSpecPath),
		fmt.Sprintf("3. Read the shared task-group context: %s", sharedContextPath),
		fmt.Sprintf("4. Read the current dispatch task contract: %s", taskContractPath),
		"4.5 When these files are large JSON, read only the fields needed for the current slice first; avoid full-file pretty prints unless a contradiction forces deeper inspection.",
		"5. If task-local artifacts already exist, read worker-result.json, verify.json, handoff.md, and referenced compact handoff logs.",
		"6. If feedback-summary exists and this task has recent failures, read only the current task's recent 3 high-severity failures before re-execution.",
		"7. After those reads, move to execution. Do not reopen planner/judge work unless the dispatch files contradict each other.",
		"",
		"Worker contract:",
		"- Use shared task-group context for roster, file schema, source policy, and other common background.",
		"- Use this dispatch slice for the current batch or milestone only.",
		"- Keep the tmux node label aligned with the current slice via `[harness:<task-id>] <node-task-description>`.",
		"- Do not rediscover the full roster or rewrite the common prompt if shared context already defines it.",
		"",
		"Hard authority rules:",
		"- Never create or mutate thread keys, request ids, task ids, plan epochs, leases, or global `.harness/state/*` ledgers.",
		"- Never edit files outside the bound worktree.",
		"- Never edit paths outside `allowedWriteGlobs`.",
		"- Never edit `blockedWriteGlobs` or move closeout artifacts outside `artifactDir`.",
		"- Never merge, rebase, push, archive, delete branches, or decide global completion.",
		"",
		"Execution defaults:",
		"- Keep work bounded to the current slice and its outputs.",
		"- Before each meaningful tool/action group, briefly state your immediate intent.",
		"- Prefer acting from shared context over re-reading every orchestration prompt file.",
		"- Prefer compact field extraction over dumping entire dispatch/state JSON files into the transcript.",
		"- If shared context is missing a required planning decision, stop and report planning drift instead of freelancing.",
		"",
		"Current slice payload:",
	}
	if selectedTask != nil {
		lines = append(lines, fmt.Sprintf("- taskTitle: %s", selectedTask.Title))
		lines = append(lines, fmt.Sprintf("- taskSummary: %s", selectedTask.Summary))
		if selectedTask.BatchLabel != "" {
			lines = append(lines, fmt.Sprintf("- batchLabel: %s", selectedTask.BatchLabel))
		}
		if len(selectedTask.EntityBatch) > 0 {
			lines = append(lines, fmt.Sprintf("- entityBatch: %s", strings.Join(selectedTask.EntityBatch, ", ")))
		}
		if len(selectedTask.OutputTargets) > 0 {
			lines = append(lines, fmt.Sprintf("- outputTargets: %s", strings.Join(selectedTask.OutputTargets, ", ")))
		}
	}
	if sharedContext != nil {
		lines = append(lines, "")
		lines = append(lines, "Shared task-group context:")
		if sharedContext.Summary != "" {
			lines = append(lines, fmt.Sprintf("- summary: %s", sharedContext.Summary))
		}
		if sharedContext.EntitySelection.SubjectLabel != "" || sharedContext.EntitySelection.TargetCount > 0 {
			lines = append(lines, fmt.Sprintf("- entitySelection: %s", summarizeEntitySelection(sharedContext.EntitySelection)))
		}
		if sharedContext.ContentContract.OutputDir != "" {
			lines = append(lines, fmt.Sprintf("- outputDir: %s", sharedContext.ContentContract.OutputDir))
		}
		if sharedContext.ContentContract.OutputFile != "" {
			lines = append(lines, fmt.Sprintf("- outputFile: %s", sharedContext.ContentContract.OutputFile))
		}
		if len(sharedContext.ContentContract.RequiredSections) > 0 {
			lines = append(lines, fmt.Sprintf("- requiredSections: %s", strings.Join(sharedContext.ContentContract.RequiredSections, ", ")))
		}
		if sharedContext.ContentContract.MinChars > 0 {
			lines = append(lines, fmt.Sprintf("- minChars: %d", sharedContext.ContentContract.MinChars))
		}
		if len(sharedContext.SourcePlan.PreferredSourceTypes) > 0 {
			lines = append(lines, fmt.Sprintf("- preferredSources: %s", strings.Join(sharedContext.SourcePlan.PreferredSourceTypes, ", ")))
		}
		lines = append(lines, "- sharedContextPath is the first place to look for roster / format / source rules.")
		lines = append(lines, "")
	}
	if taskFeedback != nil && len(taskFeedback.RecentFailures) > 0 {
		lines = append(lines,
			"Recent failure memory:",
			fmt.Sprintf("- feedbackSummaryPath: %s", feedbackSummaryPath),
			fmt.Sprintf("- latestFailureType: %s", taskFeedback.LatestFeedbackType),
			fmt.Sprintf("- latestFailureMessage: %s", taskFeedback.LatestMessage),
		)
		if taskFeedback.LatestThinkingSummary != "" {
			lines = append(lines, fmt.Sprintf("- latestThinkingSummary: %s", taskFeedback.LatestThinkingSummary))
		}
		if taskFeedback.LatestNextAction != "" {
			lines = append(lines, fmt.Sprintf("- latestNextAction: %s", taskFeedback.LatestNextAction))
		}
		lines = append(lines, "- recentFailures: read these reminders instead of scanning the full feedback log")
		for _, failure := range taskFeedback.RecentFailures {
			lines = append(lines, fmt.Sprintf("  - %s | %s | %s | %s", failure.ID, failure.Step, failure.FeedbackType, failure.Message))
			if failure.ThinkingSummary != "" {
				lines = append(lines, fmt.Sprintf("    thought: %s", failure.ThinkingSummary))
			}
			if failure.NextAction != "" {
				lines = append(lines, fmt.Sprintf("    next: %s", failure.NextAction))
			}
		}
		lines = append(lines, "")
	}
	lines = append(lines,
		"Verification and closeout:",
		"- Treat investigate -> execute -> verify -> closeout as a hard rhythm.",
		"- Record verify evidence with commands, file paths, or artifact refs.",
		"- Before exit, write worker-result.json, verify.json, and handoff.md.",
		"- If evidence is incomplete, stop with a blocked outcome instead of claiming success.",
		"",
		"Soft constraints appended after the base prompt:",
	)
	for _, rule := range constraintSystem.Rules {
		if rule.Enforcement != "soft" {
			continue
		}
		if rule.Layer != "execution" && rule.Layer != "verification" && rule.Layer != "learning" {
			continue
		}
		lines = append(lines, fmt.Sprintf("- [%s/%s/%s/%s] %s", rule.Layer, rule.Category, rule.Enforcement, rule.Level, rule.Rule))
	}
	lines = append(lines, "", "Hard constraints verified item-by-item by runtime / verify:")
	for _, rule := range constraintSystem.Rules {
		if rule.Enforcement != "hard" {
			continue
		}
		if rule.Layer != "execution" && rule.Layer != "verification" && rule.Layer != "runtime" {
			continue
		}
		mode := rule.VerificationMode
		if mode == "" {
			mode = "runtime_gate"
		}
		lines = append(lines, fmt.Sprintf("- [%s/%s/%s/%s] %s | check=%s", rule.Layer, rule.Category, rule.Enforcement, rule.Level, rule.Rule, mode))
	}
	lines = append(lines, "")
	if len(routePolicyTags) > 0 {
		lines = append(lines,
			"Route policy guardrails:",
			fmt.Sprintf("- reasonCodes: %s", strings.Join(unique(ticket.ReasonCodes), ", ")),
			fmt.Sprintf("- policyTags: %s", strings.Join(routePolicyTags, ", ")),
		)
		lines = append(lines, "")
	}
	lines = append(lines,
		"On-demand runtime refs when blocked:",
		fmt.Sprintf("- acceptedPacketPath: %s", acceptedPacketPath),
		fmt.Sprintf("- planningTracePath: %s", planningTracePath),
		fmt.Sprintf("- sharedConstraintPath: %s", constraintPath),
		fmt.Sprintf("- artifactDir: %s", artifactDir),
		fmt.Sprintf("- executionLoopSkill: %s", filepath.Join("skills", "qiushi-execution", "SKILL.md")),
		"",
		"Hookified verification flow:",
	)
	for _, hook := range hookPlan.Hooks {
		lines = append(lines, fmt.Sprintf("- %s | event=%s | action=%s | status=%s", hook.Name, hook.Event, hook.Action, hook.Status))
		for _, item := range hook.Checklist {
			lines = append(lines, fmt.Sprintf("  - %s: %s (%s)", item.ID, item.Title, item.Status))
		}
	}
	if len(hookPlan.LearningHints) > 0 {
		lines = append(lines, "", "Learned reminders:")
		for _, hint := range hookPlan.LearningHints {
			lines = append(lines, "- "+hint)
		}
	}
	lines = append(lines,
		"",
		"Verification:",
		"- Run verify commands from the dispatch ticket or worker-spec in order.",
		"- Start with the narrowest relevant validation, then broader checks when required.",
		"- Record each command, exit code, and output path in verify.json.",
		"- Write verify.json as a scorecard-oriented artifact with overallStatus, overallSummary, scorecard, evidenceLedger, findings, and reviewChecklist when applicable.",
		"- A noop completion is valid only when acceptance is already satisfied and verify.json records concrete evidence for that claim.",
		"- Do not claim completion without command or file evidence that supports the claim.",
		"- When the run changes multiple files or touches high-risk control-plane surfaces, perform a short review pass and record the findings in verify.json or handoff.md.",
		"- Before exit, if any required closeout artifact is missing, stop editing and write the missing artifact first.",
		"",
		"Required artifacts before exit:",
		fmt.Sprintf("- %s", sharedContextPath),
		fmt.Sprintf("- %s", workerSpecPath),
		fmt.Sprintf("- %s", acceptedPacketPath),
		fmt.Sprintf("- %s", taskContractPath),
		fmt.Sprintf("- %s", filepath.Join(artifactDir, "worker-result.json")),
		fmt.Sprintf("- %s", filepath.Join(artifactDir, "verify.json")),
		fmt.Sprintf("- %s", filepath.Join(artifactDir, "handoff.md")),
		"",
		"Final response:",
		"- Be brief.",
		"- Report only the terminal worker outcome and the key artifact path(s).",
		"- Do not claim global completion.",
	)
	return strings.Join(lines, "\n") + "\n"
}

func buildAcceptedPacket(task adapter.Task, ticket dispatch.Ticket, judgeDecision orchestration.JudgeDecision, hookPlan verify.HookPlan, verifyCommands []map[string]any) orchestration.AcceptedPacket {
	flowSelection := strings.TrimSpace(judgeDecision.SelectedFlow)
	if flowSelection == "" {
		flowSelection = "standard bounded delivery"
	}
	executionTasks := deriveExecutionTasks(task, verifyCommands)
	sharedContext := buildSharedTaskGroupContext(task, executionTasks)
	return orchestration.AcceptedPacket{
		SchemaVersion:     "kh.accepted-packet.v1",
		Generator:         "kh-worker-supervisor",
		GeneratedAt:       nowUTC(),
		TaskID:            task.TaskID,
		ThreadKey:         task.ThreadKey,
		PlanEpoch:         task.PlanEpoch,
		PacketID:          fmt.Sprintf("packet_%s_%d", task.TaskID, task.PlanEpoch),
		Objective:         coalesce(task.Summary, task.Title),
		Constraints:       taskConstraints(task),
		FlowSelection:     flowSelection,
		PolicyTagsApplied: policyTags(ticket.ReasonCodes),
		SelectedPlan:      selectedPlanText(task, sharedContext, executionTasks),
		RejectedAlternatives: []orchestration.RejectedAlternative{
			{CandidateID: "broad_unbounded_slice", Reason: "prefer one bounded slice that keeps verification and rollback explicit"},
			{CandidateID: "worker_self_complete", Reason: "completion remains runtime-owned and must stay outside worker authority"},
		},
		SharedContext:  sharedContext,
		ExecutionTasks: executionTasks,
		VerificationPlan: map[string]any{
			"ruleIds":  unique(task.VerificationRuleIDs),
			"commands": verifyCommands,
		},
		DecisionRationale: decisionRationaleText(task, sharedContext),
		OwnedPaths:        unique(task.OwnedPaths),
		TaskBudgets: map[string]any{
			"taskId":     task.TaskID,
			"dispatchId": ticket.DispatchID,
			"taskBudget": ticket.Budget,
		},
		AcceptanceMarkers: unique(hookPlan.AcceptanceMarkers),
		ReplanTriggers: []string{
			"verification_failed",
			"acceptance_markers_missing",
			"owned_paths_conflict",
			"authority_boundary_conflict",
		},
		RollbackHints: []string{
			"leave_task_local_artifacts_intact",
			"preserve_checkpoint_for_supervisor",
			"handoff_before_exit_when_blocked",
		},
		AcceptedAt: nowUTC(),
		AcceptedBy: "kh-worker-supervisor",
	}
}

func buildPlannerCandidates(task adapter.Task, ticket dispatch.Ticket, loop orchestration.PacketSynthesisLoop, judgeDecision orchestration.JudgeDecision, packet orchestration.AcceptedPacket, hookPlan verify.HookPlan, verifyCommands []map[string]any) []orchestration.PlannerCandidate {
	taskName := materializedTaskName(task)
	executionTasks := packet.ExecutionTasks
	verifySteps := verificationStepTitles(verifyCommands)
	verifySummary := summarizePlannerVerifyPlan(verifySteps, packet.ReplanTriggers)
	lanes := make([]orchestration.PlannerCandidate, 0, len(loop.Planners))
	for _, planner := range loop.Planners {
		lowered := strings.ToLower(strings.Join([]string{planner.ID, planner.Name, planner.Focus}, " "))
		candidate := orchestration.PlannerCandidate{
			PlannerID:      planner.ID,
			PlannerName:    planner.Name,
			Focus:          planner.Focus,
			TaskName:       taskName,
			ProposedFlow:   judgeDecision.SelectedFlow,
			MaterializedBy: "kh-worker-supervisor",
		}
		switch {
		case strings.Contains(lowered, "architecture"):
			candidate.ResultSummary = fmt.Sprintf("用 `%s` 作为主任务名，并把任务边界收敛在 owned paths 与 runtime authority boundary 内。", taskName)
			candidate.KeyMoves = []string{
				"保持 thread 级主任务名稳定，便于后续追加需求继续挂到同一主线",
				"先看 control plane，再看 execution plane，最后看 operator plane",
				fmt.Sprintf("限制写入边界到 %d 个 owned path scopes", len(unique(task.OwnedPaths))),
			}
			candidate.Risks = []string{
				"owned paths 过宽会让 execution slice 膨胀",
				"如果把 closeout / archive 权限下放给 worker，会破坏 runtime authority boundary",
			}
			candidate.Evidence = uniqueNonEmpty(
				summarizeOwnedPathEvidence(task.OwnedPaths),
				summarizeList("policyTags", policyTags(ticket.ReasonCodes), 3),
			)
		case strings.Contains(lowered, "delivery"):
			candidate.ResultSummary = fmt.Sprintf("把主任务 `%s` 编排成可追踪切片，并把当前 dispatch 绑定到单一 execution slice。", taskName)
			candidate.KeyMoves = []string{
				fmt.Sprintf("judge 当前接受了 %d 个 execution tasks", len(executionTasks)),
				fmt.Sprintf("当前 dispatch=%s", ticket.DispatchID),
				fmt.Sprintf("当前尝试 attempt=%d", ticket.Attempt),
			}
			if len(executionTasks) > 0 {
				candidate.KeyMoves = append(candidate.KeyMoves, "首个待执行 slice="+executionTasks[0].ID)
			}
			candidate.Risks = []string{
				"delivery slice 太碎会导致主任务编排图失真",
				"缺少主任务到聚合 tasks 的显式关系时，operator 很难跟踪追加需求落点",
			}
			candidate.Evidence = uniqueNonEmpty(
				fmt.Sprintf("selectedFlow=%s", packet.FlowSelection),
				fmt.Sprintf("acceptedPacket=%s", packet.PacketID),
			)
		case strings.Contains(lowered, "risk"):
			candidate.ResultSummary = verifySummary
			candidate.KeyMoves = []string{
				"把 verify / review / rollback 信号放进同一条主任务执行链",
				"失败时进入 analysis.required -> needs_replan -> next dispatch",
			}
			if len(hookPlan.AcceptanceMarkers) > 0 {
				candidate.KeyMoves = append(candidate.KeyMoves, "acceptance markers="+strings.Join(unique(hookPlan.AcceptanceMarkers), ", "))
			}
			candidate.Risks = uniqueNonEmpty(
				"closeout artifacts 缺失会直接触发 blocked / replan",
				summarizeList("replan", packet.ReplanTriggers, 3),
				summarizeList("rollback", packet.RollbackHints, 2),
			)
			candidate.Evidence = uniqueNonEmpty(
				summarizeList("verifySteps", verifySteps, 2),
				fmt.Sprintf("reviewRequired=%t verifyRequired=%t", judgeDecision.ReviewRequired, judgeDecision.VerifyRequired),
			)
		default:
			candidate.ResultSummary = "planner candidate was materialized from dispatch-time packet synthesis metadata"
		}
		lanes = append(lanes, candidate)
	}
	return lanes
}

func materializedTaskName(task adapter.Task) string {
	label := strings.TrimSpace(coalesce(task.Title, task.Summary, task.TaskID))
	if strings.Contains(label, "前端页面开发") || strings.Contains(label, "可视化开发") {
		return "harness-architect 前端页面开发"
	}
	if label == "" {
		return task.TaskID
	}
	runes := []rune(label)
	if len(runes) > 26 {
		return string(runes[:26]) + "..."
	}
	return label
}

func summarizePlannerVerifyPlan(steps, replanTriggers []string) string {
	if len(steps) == 0 {
		return "风险 lane 侧重 verify / replan；当前 verification plan 还没有命令级持久化。"
	}
	return fmt.Sprintf("风险 lane 计划先执行 %d 个验证步骤，并在 verify 失败时回到 replan。", len(steps))
}

func summarizeOwnedPathEvidence(paths []string) string {
	paths = unique(paths)
	if len(paths) == 0 {
		return "owned paths are not set"
	}
	if len(paths) == 1 {
		return "owned path=" + paths[0]
	}
	return fmt.Sprintf("owned paths=%s (+%d more)", paths[0], len(paths)-1)
}

func summarizeList(label string, values []string, maxCount int) string {
	values = unique(values)
	if len(values) == 0 {
		return ""
	}
	if len(values) > maxCount {
		return fmt.Sprintf("%s=%s (+%d more)", label, strings.Join(values[:maxCount], ", "), len(values)-maxCount)
	}
	return fmt.Sprintf("%s=%s", label, strings.Join(values, ", "))
}

func tmuxCommandBanner(task adapter.Task, executionTasks []orchestration.ExecutionTask, contract orchestration.TaskContract) string {
	description := commandNodeDescription(contract, executionTasks, task)
	if description == "" {
		return fmt.Sprintf("[harness:%s]", task.TaskID)
	}
	return fmt.Sprintf("[harness:%s] %s", task.TaskID, description)
}

func commandNodeDescription(contract orchestration.TaskContract, executionTasks []orchestration.ExecutionTask, task adapter.Task) string {
	if title := executionTaskTitle(executionTasks, contract.ExecutionSliceID); title != "" {
		return truncateLabel(title, 96)
	}
	if objective := strings.TrimSpace(contract.Objective); objective != "" {
		return truncateLabel(objective, 96)
	}
	return truncateLabel(coalesce(task.Title, task.Summary, task.TaskID), 96)
}

func executionTaskTitle(tasks []orchestration.ExecutionTask, sliceID string) string {
	for _, task := range tasks {
		if strings.TrimSpace(task.ID) != strings.TrimSpace(sliceID) {
			continue
		}
		if strings.TrimSpace(task.Title) != "" {
			return task.Title
		}
		if strings.TrimSpace(task.Summary) != "" {
			return task.Summary
		}
	}
	return ""
}

func truncateLabel(value string, limit int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" || limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "..."
}

func selectedExecutionTaskByID(tasks []orchestration.ExecutionTask, sliceID string) *orchestration.ExecutionTask {
	for _, item := range tasks {
		if strings.TrimSpace(item.ID) != strings.TrimSpace(sliceID) {
			continue
		}
		copy := item
		return &copy
	}
	return nil
}

func summarizeEntitySelection(selection orchestration.EntitySelection) string {
	parts := make([]string, 0, 3)
	if selection.TargetCount > 0 && selection.SubjectLabel != "" {
		parts = append(parts, fmt.Sprintf("%d 位%s", selection.TargetCount, selection.SubjectLabel))
	} else if selection.SubjectLabel != "" {
		parts = append(parts, selection.SubjectLabel)
	}
	if selection.SelectionMode != "" {
		parts = append(parts, "mode="+selection.SelectionMode)
	}
	if len(selection.Entities) > 0 {
		parts = append(parts, "entities="+strings.Join(selection.Entities, ", "))
	}
	return strings.Join(parts, " | ")
}

func buildSharedTaskGroupContext(task adapter.Task, executionTasks []orchestration.ExecutionTask) *orchestration.SharedTaskGroupContext {
	info := inferCorpusPlanning(task)
	operatorTaskList := make([]string, 0, len(executionTasks))
	for _, item := range executionTasks {
		if strings.TrimSpace(item.Title) != "" {
			operatorTaskList = append(operatorTaskList, item.Title)
		}
	}
	context := &orchestration.SharedTaskGroupContext{
		GroupID:          task.TaskID + ".group",
		Summary:          coalesce(task.Description, task.Summary, task.Title),
		OperatorTaskList: unique(operatorTaskList),
		VerificationFocus: uniqueNonEmpty(
			"名单、文件格式、资料来源这些共享决策应在规划阶段冻结",
			"worker 只执行当前批次或当前阶段，不重新发散成新的外层规划",
		),
	}
	if info.SubjectLabel != "" || info.SubjectCount > 0 {
		context.EntitySelection = orchestration.EntitySelection{
			SubjectLabel:      info.SubjectLabel,
			TargetCount:       info.SubjectCount,
			SelectionMode:     "planner_or_judge_frozen",
			SelectionCriteria: uniqueNonEmpty("优先在编排阶段冻结名单或名单生成规则，再交给 worker 执行当前批次"),
		}
	}
	if info.OutputDir != "" || info.OutputFile != "" || len(info.RequiredSections) > 0 || info.MinChars > 0 {
		contract := orchestration.ContentContract{
			OutputDir:        info.OutputDir,
			OutputFile:       info.OutputFile,
			FileExtension:    coalesce(info.FileExtension, ".md"),
			RequiredSections: unique(info.RequiredSections),
			MinChars:         info.MinChars,
		}
		if info.SingleDocument {
			contract.FormatConstraints = uniqueNonEmpty(
				"固定单文件交付",
				singleDocumentOutputConstraint(info),
			)
		} else {
			contract.IndexFile = info.IndexFile
			contract.FileNamingRule = "序号-名称.md"
			contract.RequiredFields = uniqueNonEmpty("基本信息", "代表成果", "核心贡献", "历史影响", "争议点", "延伸阅读")
			contract.FormatConstraints = uniqueNonEmpty("每位对象单独一个文件", "总索引与正文文件分离")
		}
		context.ContentContract = contract
	}
	context.SourcePlan = orchestration.SourcePlan{
		ResearchGoal: corpusResearchGoal(info),
		PreferredSourceTypes: uniqueNonEmpty(
			"百科类权威资料",
			"高校 / 学术机构页面",
			"传记、综述或权威出版物",
		),
		SearchHints:        uniqueNonEmpty(buildSearchHint(task, info)),
		RequiredCrossCheck: true,
		Notes:              uniqueNonEmpty("关键事实优先交叉确认，避免每个 worker 自己临时决定资料标准"),
	}
	context.SharedPrompt = uniqueNonEmpty(
		buildSharedPromptLine(task, info),
		"先使用 shared task-group context，再执行当前 dispatch slice。",
		"如果名单、字段模板或资料来源策略未冻结，先回报 planning drift，不要由 worker 临场重规划。",
	)
	if context.Summary == "" && len(context.SharedPrompt) == 0 && len(context.OperatorTaskList) == 0 {
		return nil
	}
	return context
}

func selectedPlanText(task adapter.Task, sharedContext *orchestration.SharedTaskGroupContext, executionTasks []orchestration.ExecutionTask) string {
	parts := []string{strings.TrimSpace(coalesce(task.Description, task.Summary, task.Title))}
	if sharedContext != nil {
		if summary := strings.TrimSpace(sharedContext.Summary); summary != "" && summary != parts[0] {
			parts = append(parts, summary)
		}
		if sharedContext.ContentContract.OutputDir != "" {
			parts = append(parts, "输出目录="+sharedContext.ContentContract.OutputDir)
		}
		if sharedContext.ContentContract.OutputFile != "" {
			parts = append(parts, "输出文件="+sharedContext.ContentContract.OutputFile)
		}
		if sharedContext.ContentContract.MinChars > 0 {
			parts = append(parts, minCharsPromptText(sharedContext.ContentContract.OutputFile != "", sharedContext.ContentContract.MinChars))
		}
	}
	if len(executionTasks) > 0 {
		parts = append(parts, fmt.Sprintf("judge tasklist=%d", len(executionTasks)))
	}
	return strings.Join(uniqueNonEmpty(parts...), " | ")
}

func decisionRationaleText(task adapter.Task, sharedContext *orchestration.SharedTaskGroupContext) string {
	parts := []string{strings.TrimSpace(coalesce(task.Description, task.Summary))}
	if sharedContext != nil {
		parts = append(parts, "共享名单/格式/资料策略应在规划阶段冻结，再交给 worker 执行当前 slice。")
	}
	return strings.Join(uniqueNonEmpty(parts...), " | ")
}

func deriveCorpusExecutionTaskSpecs(task adapter.Task) []executionTaskSpec {
	info := inferCorpusPlanning(task)
	if info.OutputDir == "" && info.OutputFile == "" && info.SubjectLabel == "" && info.SubjectCount == 0 {
		return nil
	}
	sharedSummary := buildSharedPromptLine(task, info)
	if info.SingleDocument {
		outputTarget := coalesce(info.OutputFile, filepath.Join(info.OutputDir, "deliverable"+coalesce(info.FileExtension, ".md")))
		return []executionTaskSpec{
			{
				Title:                "写入单文档交付件",
				Summary:              fmt.Sprintf("按已冻结的共享上下文把 %s 的资料写入 %s，并满足整体篇幅约束。", summarizeCorpusSubject(info), outputTarget),
				TaskGroupID:          task.TaskID + ".group",
				SharedContextSummary: sharedSummary,
				OutputTargets:        uniqueNonEmpty(outputTarget),
				DoneCriteria:         []string{"交付文件开始落盘", "总正文满足共享字数约束"},
			},
			{
				Title:                "完成校验与收口",
				Summary:              fmt.Sprintf("对单文档交付件完成整体校验，并补全 verify 与 handoff 收口。 | output=%s", outputTarget),
				TaskGroupID:          task.TaskID + ".group",
				SharedContextSummary: sharedSummary,
				OutputTargets:        uniqueNonEmpty(outputTarget),
				DoneCriteria:         []string{"整体校验完成", "verify evidence 完整", "closeout artifacts 完整"},
			},
		}
	}
	specs := []executionTaskSpec{}
	batchLabel := "正文批次"
	if info.SubjectCount > 0 {
		batchLabel = fmt.Sprintf("%s正文批次", summarizeCorpusSubject(info))
	}
	specs = append(specs, executionTaskSpec{
		Title:                "批量产出正文文件",
		Summary:              fmt.Sprintf("按已冻结的共享上下文批量产出 %s 的正文文件，并把当前批次的输出写入 %s。", summarizeCorpusSubject(info), coalesce(info.OutputDir, "目标目录")),
		TaskGroupID:          task.TaskID + ".group",
		BatchLabel:           batchLabel,
		SharedContextSummary: sharedSummary,
		OutputTargets:        uniqueNonEmpty(info.OutputDir),
		DoneCriteria:         []string{"正文文件开始落盘", "每个文件遵循共享字段模板和字数约束"},
	})
	closeoutSummary := "完成整体校验、verify 与 handoff 收口。"
	closeoutTargets := uniqueNonEmpty(info.OutputDir)
	doneCriteria := []string{"verify evidence 完整", "closeout artifacts 完整"}
	if info.RequiresIndex || info.IndexFile != "" {
		closeoutSummary = fmt.Sprintf("生成总索引文件并完成整体校验、verify 与 handoff 收口。 | index=%s", coalesce(info.IndexFile, "总索引"))
		closeoutTargets = uniqueNonEmpty(filepath.Join(info.OutputDir, coalesce(info.IndexFile, "00-总索引.md")))
		doneCriteria = append([]string{"总索引写入完成"}, doneCriteria...)
	}
	specs = append(specs, executionTaskSpec{
		Title:                "完成校验与收口",
		Summary:              closeoutSummary,
		TaskGroupID:          task.TaskID + ".group",
		SharedContextSummary: sharedSummary,
		OutputTargets:        closeoutTargets,
		DoneCriteria:         doneCriteria,
	})
	return specs
}

func inferCorpusPlanning(task adapter.Task) corpusPlanningInfo {
	text := strings.Join(uniqueNonEmpty(task.Title, task.Summary, task.Description), "\n")
	info := corpusPlanningInfo{
		FileExtension: ".md",
	}
	if outputFile := detectExplicitOutputFile(text); outputFile != "" {
		info.OutputFile = outputFile
		info.SingleDocument = true
		if ext := strings.TrimSpace(filepath.Ext(outputFile)); ext != "" {
			info.FileExtension = ext
		}
		if dir := strings.TrimSpace(filepath.Dir(outputFile)); dir != "" && dir != "." {
			info.OutputDir = dir
		}
	}
	if matches := regexp.MustCompile(`(?:在|到)\s+(\S+)\s+下`).FindStringSubmatch(text); len(matches) == 2 {
		if info.OutputDir == "" {
			info.OutputDir = strings.TrimSpace(matches[1])
		}
	}
	if matches := regexp.MustCompile(`(\d+)\s*位([^\s。；，、]+)`).FindStringSubmatch(text); len(matches) == 3 {
		fmt.Sscanf(matches[1], "%d", &info.SubjectCount)
		info.SubjectLabel = strings.TrimSpace(matches[2])
	}
	if matches := regexp.MustCompile(`不少于\s*(\d+)\s*字`).FindStringSubmatch(text); len(matches) == 2 {
		fmt.Sscanf(matches[1], "%d", &info.MinChars)
	}
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.Contains(line, "总索引") {
			info.RequiresIndex = true
		}
		if !strings.Contains(line, "包含") {
			continue
		}
		parts := strings.SplitN(line, "包含", 2)
		if len(parts) != 2 {
			continue
		}
		tail := strings.TrimSpace(parts[1])
		for _, token := range strings.FieldsFunc(tail, func(r rune) bool {
			return strings.ContainsRune("、，,。；;：:", r)
		}) {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			info.RequiredSections = append(info.RequiredSections, token)
		}
	}
	info.RequiredSections = unique(info.RequiredSections)
	if !info.SingleDocument && (info.OutputDir != "" || info.SubjectLabel != "" || info.SubjectCount > 0) {
		info.IndexFile = "00-总索引.md"
	}
	return info
}

func summarizeCorpusSubject(info corpusPlanningInfo) string {
	if info.SubjectCount > 0 && info.SubjectLabel != "" {
		return fmt.Sprintf("%d 位%s", info.SubjectCount, info.SubjectLabel)
	}
	if info.SubjectLabel != "" {
		return info.SubjectLabel
	}
	return "对象集合"
}

func buildSharedPromptLine(task adapter.Task, info corpusPlanningInfo) string {
	parts := []string{fmt.Sprintf("先在编排阶段冻结 %s 的名单、格式和资料策略，再分发当前 worker slice。", summarizeCorpusSubject(info))}
	if info.OutputDir != "" {
		parts = append(parts, "输出目录="+info.OutputDir)
	}
	if info.OutputFile != "" {
		parts = append(parts, "输出文件="+info.OutputFile)
	}
	if len(info.RequiredSections) > 0 {
		parts = append(parts, "固定字段="+strings.Join(info.RequiredSections, " / "))
	}
	if info.MinChars > 0 {
		parts = append(parts, minCharsPromptText(info.SingleDocument, info.MinChars))
	}
	if strings.TrimSpace(task.Description) != "" {
		parts = append(parts, strings.TrimSpace(task.Description))
	}
	return strings.Join(uniqueNonEmpty(parts...), " | ")
}

func corpusResearchGoal(info corpusPlanningInfo) string {
	if info.SingleDocument {
		return "先冻结名单、结构和资料策略，再进入单文档写作与收口"
	}
	return "先冻结名单、格式和资料策略，再进入批量写作或整理"
}

func minCharsPromptText(singleDocument bool, minChars int) string {
	if minChars <= 0 {
		return ""
	}
	if singleDocument {
		return fmt.Sprintf("总正文不少于 %d 字", minChars)
	}
	return fmt.Sprintf("每个文件不少于 %d 字", minChars)
}

func singleDocumentOutputConstraint(info corpusPlanningInfo) string {
	if strings.TrimSpace(info.OutputFile) == "" {
		return ""
	}
	return "输出文件=" + strings.TrimSpace(info.OutputFile)
}

func detectExplicitOutputFile(text string) string {
	matches := regexp.MustCompile("(?:写入|写到|输出到|输出至|保存到|保存至|生成到|生成至|落到)\\s*[`\"'“”‘’]?([^\\s`\"'“”‘’，。；;：:]+?\\.[A-Za-z0-9]+)[`\"'“”‘’]?").FindStringSubmatch(text)
	if len(matches) != 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func buildSearchHint(task adapter.Task, info corpusPlanningInfo) string {
	if info.SubjectLabel != "" {
		return fmt.Sprintf("围绕“%s 资料 / 生平 / 主要贡献 / 历史影响”收集权威来源。", summarizeCorpusSubject(info))
	}
	return fmt.Sprintf("围绕任务 `%s` 收集权威来源并在规划阶段冻结资料策略。", coalesce(task.Title, task.TaskID))
}

func buildTaskContract(root string, task adapter.Task, ticket dispatch.Ticket, packet orchestration.AcceptedPacket, hookPlan verify.HookPlan, acceptedPacketPath string) orchestration.TaskContract {
	selectedTask := selectExecutionTask(root, task.TaskID, packet.ExecutionTasks, ticket.Attempt)
	executionSliceID := task.TaskID
	inScope := unique(task.OwnedPaths)
	doneCriteria := uniqueNonEmpty(append([]string{"task-local slice implemented", "verification evidence recorded", "closeout artifacts written"}, hookPlan.AcceptanceMarkers...)...)
	requiredEvidence := []string{"dispatch ticket", "worker-spec", "verify.json", "worker-result.json", "handoff.md"}
	objective := packet.Objective
	if selectedTask != nil {
		if strings.TrimSpace(selectedTask.ID) != "" {
			executionSliceID = selectedTask.ID
		}
		if len(selectedTask.InScope) > 0 {
			inScope = unique(selectedTask.InScope)
		}
		if len(selectedTask.DoneCriteria) > 0 {
			doneCriteria = unique(selectedTask.DoneCriteria)
		}
		if len(selectedTask.RequiredEvidence) > 0 {
			requiredEvidence = unique(selectedTask.RequiredEvidence)
		}
		if strings.TrimSpace(selectedTask.Summary) != "" {
			objective = selectedTask.Summary
		}
	}
	checklist := make([]orchestration.VerificationChecklistItem, 0, len(hookPlan.Hooks)+1)
	for _, hook := range hookPlan.Hooks {
		checklist = append(checklist, orchestration.VerificationChecklistItem{
			ID:       hook.Name,
			Title:    hook.Summary,
			Required: true,
			Status:   hook.Status,
			Detail:   hook.Action,
		})
	}
	checklist = append(checklist, orchestration.VerificationChecklistItem{
		ID:       "closeout_artifacts",
		Title:    "required closeout artifacts are present",
		Required: true,
		Status:   "required",
		Detail:   "worker-result.json, verify.json, handoff.md",
	})
	reviewRequired := task.ReviewRequired || contains(ticket.ReasonCodes, "policy_review_if_multi_file_or_high_risk") || contains(ticket.ReasonCodes, "review_required")
	return orchestration.TaskContract{
		SchemaVersion:         "kh.task-contract.v1",
		Generator:             "kh-worker-supervisor",
		GeneratedAt:           nowUTC(),
		ContractID:            fmt.Sprintf("contract_%s_%d_%d", task.TaskID, task.PlanEpoch, ticket.Attempt),
		TaskID:                task.TaskID,
		DispatchID:            ticket.DispatchID,
		ThreadKey:             task.ThreadKey,
		PlanEpoch:             task.PlanEpoch,
		ExecutionSliceID:      executionSliceID,
		Objective:             objective,
		InScope:               inScope,
		OutOfScope:            unique(append([]string{"global control-plane ledgers", "merge/archive/completion decisions"}, task.ForbiddenPaths...)),
		DoneCriteria:          doneCriteria,
		AcceptanceMarkers:     unique(hookPlan.AcceptanceMarkers),
		VerificationChecklist: checklist,
		RequiredEvidence:      requiredEvidence,
		ReviewRequired:        reviewRequired,
		ContractStatus:        "accepted",
		ProposedBy:            "kh-worker-supervisor",
		AcceptedBy:            "kh-worker-supervisor",
		AcceptedAt:            nowUTC(),
		AcceptedPacketPath:    acceptedPacketPath,
	}
}

func verificationStepTitles(commands []map[string]any) []string {
	steps := make([]string, 0, len(commands))
	for _, command := range commands {
		title := strings.TrimSpace(coalesce(stringValue(command["title"]), stringValue(command["ruleId"]), stringValue(command["exec"])))
		if title == "" {
			continue
		}
		steps = append(steps, title)
	}
	return steps
}

func deriveExecutionTasks(task adapter.Task, verifyCommands []map[string]any) []orchestration.ExecutionTask {
	verificationSteps := verificationStepTitles(verifyCommands)
	baseEvidence := []string{"verify.json", "worker-result.json", "handoff.md"}
	baseSummary := coalesce(task.Description, task.Summary, task.Title)
	specs := deriveExecutionTaskSpecs(task, baseSummary)
	tasks := make([]orchestration.ExecutionTask, 0, len(specs))
	for index, spec := range specs {
		title := strings.TrimSpace(spec.Title)
		if title == "" {
			title = coalesce(task.Title, task.TaskID)
		}
		doneCriteria := uniqueNonEmpty(spec.DoneCriteria...)
		if len(doneCriteria) == 0 {
			doneCriteria = []string{"bounded change applied", "verification evidence recorded", "closeout artifacts written"}
		}
		tasks = append(tasks, orchestration.ExecutionTask{
			ID:                   fmt.Sprintf("%s.slice.%d", task.TaskID, index+1),
			Title:                title,
			Summary:              strings.TrimSpace(spec.Summary),
			TaskGroupID:          strings.TrimSpace(spec.TaskGroupID),
			BatchLabel:           strings.TrimSpace(spec.BatchLabel),
			EntityBatch:          unique(spec.EntityBatch),
			OutputTargets:        unique(spec.OutputTargets),
			SharedContextSummary: strings.TrimSpace(spec.SharedContextSummary),
			InScope:              unique(spec.InScope),
			DoneCriteria:         doneCriteria,
			RequiredEvidence:     baseEvidence,
			VerificationSteps:    verificationSteps,
		})
	}
	return tasks
}

func deriveExecutionTaskSpecs(task adapter.Task, baseSummary string) []executionTaskSpec {
	if specs := deriveCorpusExecutionTaskSpecs(task); len(specs) > 0 {
		return specs
	}
	requirements := explicitRequirementLines(task)
	if len(requirements) == 0 {
		requirements = inlineRequirementLines(task)
	}
	if len(requirements) > 0 {
		specs := make([]executionTaskSpec, 0, len(requirements)+1)
		if title := strings.TrimSpace(task.Title); title != "" {
			specs = append(specs, executionTaskSpec{
				Title:   title,
				Summary: fmt.Sprintf("建立主任务骨架并稳定任务命名，确保后续规划、追加需求和执行链都挂在同一主线上。 | %s", title),
			})
		}
		for index, requirement := range requirements {
			specs = append(specs, executionTaskSpec{
				Title:        fmt.Sprintf("%s [%d]", semanticTaskTitle(task, requirement), index+1),
				Summary:      requirement,
				DoneCriteria: []string{"requirement intent is reflected in runtime artifacts", "verification evidence recorded", "closeout artifacts written"},
			})
		}
		if len(specs) > 0 {
			return specs
		}
	}
	inScope := unique(task.OwnedPaths)
	if len(inScope) == 0 {
		inScope = []string{"repo-local bounded slice"}
	}
	return []executionTaskSpec{
		{
			Title:   coalesce(task.Title, task.TaskID),
			Summary: strings.TrimSpace(baseSummary),
			InScope: inScope,
		},
	}
}

func explicitRequirementLines(task adapter.Task) []string {
	lines := strings.Split(strings.ReplaceAll(task.Summary+"\n"+task.Description, "\r\n", "\n"), "\n")
	requirementRE := regexp.MustCompile(`^(?:\d+[\.\)、:：-]*|[-*•]+)\s*`)
	requirements := make([]string, 0, len(lines))
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
		requirements = append(requirements, line)
	}
	return uniqueNonEmpty(requirements...)
}

func inlineRequirementLines(task adapter.Task) []string {
	lines := strings.Split(strings.ReplaceAll(task.Summary+"\n"+task.Description, "\r\n", "\n"), "\n")
	requirements := make([]string, 0, len(lines))
	for _, raw := range lines {
		line := normalizeRequirementLine(raw)
		if line == "" {
			continue
		}
		if parts := splitDisplayRequirements(line); len(parts) > 0 {
			requirements = append(requirements, parts...)
			continue
		}
		if part := splitNeedToDisplayRequirement(line); part != "" {
			requirements = append(requirements, part)
		}
	}
	return uniqueNonEmpty(requirements...)
}

func normalizeRequirementLine(raw string) string {
	line := strings.TrimSpace(raw)
	line = strings.TrimPrefix(line, "补充：")
	line = strings.TrimPrefix(line, "补充:")
	line = strings.TrimPrefix(line, "说明：")
	line = strings.TrimPrefix(line, "说明:")
	line = strings.TrimPrefix(line, "要求：")
	line = strings.TrimPrefix(line, "要求:")
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	lower := strings.ToLower(line)
	if lower == "要求" || lower == "requirements" || lower == "requirement" {
		return ""
	}
	return line
}

func splitDisplayRequirements(line string) []string {
	index := strings.Index(line, "展示")
	if index < 0 {
		index = strings.Index(line, "显示")
	}
	if index < 0 {
		return nil
	}
	verb := "展示"
	if strings.HasPrefix(line[index:], "显示") {
		verb = "显示"
	}
	payload := strings.TrimSpace(line[index+len(verb):])
	if payload == "" {
		return nil
	}
	clauses := strings.FieldsFunc(payload, func(r rune) bool {
		return r == '；' || r == ';'
	})
	requirements := make([]string, 0, len(clauses))
	for _, clause := range clauses {
		segment := strings.TrimSpace(clause)
		if segment == "" {
			continue
		}
		if strings.Contains(segment, "本次先只做") || strings.Contains(segment, "不修改") {
			continue
		}
		for _, item := range strings.Split(segment, "、") {
			item = sanitizeRequirementFragment(item)
			if item == "" {
				continue
			}
			requirements = append(requirements, verb+" "+item)
		}
	}
	return uniqueNonEmpty(requirements...)
}

func splitNeedToDisplayRequirement(line string) string {
	if !strings.Contains(line, "需要把") || !strings.Contains(line, "展示") {
		return ""
	}
	payload := strings.TrimSpace(strings.TrimPrefix(line, "需要把"))
	if payload == "" {
		return ""
	}
	if index := strings.Index(payload, "展示"); index >= 0 {
		payload = strings.TrimSpace(payload[:index])
	}
	payload = strings.TrimSpace(strings.TrimSuffix(payload, "也显式"))
	payload = sanitizeRequirementFragment(payload)
	if payload == "" {
		return ""
	}
	return "展示 " + payload
}

func sanitizeRequirementFragment(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "也显式")
	value = strings.TrimSuffix(value, "显式")
	value = strings.TrimSuffix(value, "也需要")
	value = strings.TrimSuffix(value, "在 dashboard 里")
	value = strings.TrimSuffix(value, "在dashboard里")
	value = strings.Trim(value, "：:;；，, ")
	return strings.TrimSpace(value)
}

func semanticTaskTitle(task adapter.Task, requirement string) string {
	base := strings.TrimSpace(task.Title)
	if base == "" {
		base = strings.TrimSpace(firstSentence(task.Summary))
	}
	if base == "" {
		base = task.TaskID
	}
	trimmed := strings.TrimSpace(requirement)
	if trimmed == "" {
		return base
	}
	runes := []rune(trimmed)
	if len(runes) > 24 {
		trimmed = string(runes[:24]) + "..."
	}
	return base + " - " + trimmed
}

func firstSentence(text string) string {
	for _, raw := range strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == '。' || r == '.' || r == '!' || r == '！' || r == '?' || r == '？'
	}) {
		line := strings.TrimSpace(raw)
		if line != "" {
			return line
		}
	}
	return ""
}

func selectExecutionTask(root, taskID string, tasks []orchestration.ExecutionTask, attempt int) *orchestration.ExecutionTask {
	if len(tasks) == 0 {
		return nil
	}
	if next := selectFirstIncompleteExecutionTask(root, taskID, tasks); next != nil {
		return next
	}
	if attempt <= 0 {
		attempt = 1
	}
	index := (attempt - 1) % len(tasks)
	selected := tasks[index]
	return &selected
}

func selectFirstIncompleteExecutionTask(root, taskID string, tasks []orchestration.ExecutionTask) *orchestration.ExecutionTask {
	progressPath := orchestration.PacketProgressPath(root, taskID)
	progress, err := orchestration.LoadPacketProgress(progressPath)
	if err != nil {
		return nil
	}
	completed := map[string]struct{}{}
	for _, id := range progress.CompletedSliceIDs {
		completed[id] = struct{}{}
	}
	for _, task := range tasks {
		if _, ok := completed[task.ID]; ok {
			continue
		}
		selected := task
		return &selected
	}
	return nil
}

func stableFingerprint(parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(hash[:16])
}

func unique(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func uniqueNonEmpty(values ...string) []string {
	return unique(values)
}

func policyTags(reasonCodes []string) []string {
	tags := make([]string, 0, len(reasonCodes))
	for _, code := range reasonCodes {
		if strings.HasPrefix(code, "policy_") {
			tags = append(tags, code)
		}
	}
	return unique(tags)
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func taskConstraints(task adapter.Task) []string {
	constraints := []string{
		"stay within task-local scope",
		"do not mutate global control-plane ledgers",
		"obey allowedWriteGlobs and blockedWriteGlobs",
		"leave merge, archive, and completion decisions to runtime",
	}
	if task.WorkerMode != "" {
		constraints = append(constraints, "workerMode="+task.WorkerMode)
	}
	if task.ResumeStrategy != "" {
		constraints = append(constraints, "resumeStrategy="+task.ResumeStrategy)
	}
	return constraints
}

func coalesce(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func writeJSON(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

func mergeStringMaps(parts ...map[string]string) map[string]string {
	merged := map[string]string{}
	for _, part := range parts {
		for key, value := range part {
			merged[key] = value
		}
	}
	return merged
}
