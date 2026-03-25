package worker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"klein-harness/internal/adapter"
	"klein-harness/internal/dispatch"
	"klein-harness/internal/orchestration"
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

type DispatchBundle struct {
	TicketPath     string
	WorkerSpecPath string
	PromptPath     string
	ArtifactDir    string
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
	executionCwd := adapter.TaskCWD(paths, task)
	artifactDir := filepath.Join(paths.ArtifactsDir, task.TaskID, ticket.DispatchID)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return DispatchBundle{}, err
	}
	ticketPath := filepath.Join(paths.StateDir, fmt.Sprintf("dispatch-ticket-%s.json", task.TaskID))
	workerSpecPath := filepath.Join(artifactDir, "worker-spec.json")
	promptPath := filepath.Join(paths.StateDir, fmt.Sprintf("runner-prompt-%s.md", task.TaskID))
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
	workerSpec := map[string]any{
		"schemaVersion":     "kh.worker-spec.v1",
		"generator":         "kh-worker-supervisor",
		"generatedAt":       nowUTC(),
		"dispatchId":        ticket.DispatchID,
		"taskId":            task.TaskID,
		"threadKey":         task.ThreadKey,
		"planEpoch":         task.PlanEpoch,
		"attempt":           ticket.Attempt,
		"objective":         coalesce(task.Summary, task.Title),
		"selectedPlan":      coalesce(task.Description, task.Summary, task.Title),
		"constraints":       taskConstraints(task),
		"ownedPaths":        unique(task.OwnedPaths),
		"blockedPaths":      unique(task.ForbiddenPaths),
		"taskBudget":        ticket.Budget,
		"acceptanceMarkers": unique(task.VerificationRuleIDs),
		"verificationPlan": map[string]any{
			"ruleIds":  unique(task.VerificationRuleIDs),
			"commands": verifyCommands,
		},
		"decisionRationale": coalesce(task.Description, task.Summary),
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
	if err := writeJSON(workerSpecPath, workerSpec); err != nil {
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
		"allowedWriteGlobs":       unique(task.OwnedPaths),
		"blockedWriteGlobs":       unique(task.ForbiddenPaths),
		"artifactDir":             artifactDir,
		"workerSpecPath":          workerSpecPath,
		"workerSpec":              workerSpec,
		"artifacts": map[string]string{
			"workerSpec":   workerSpecPath,
			"workerResult": filepath.Join(artifactDir, "worker-result.json"),
			"verify":       filepath.Join(artifactDir, "verify.json"),
			"handoff":      filepath.Join(artifactDir, "handoff.md"),
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
		"packetSynthesis": orchestration.DefaultPacketSynthesisLoop(paths.Root),
		"runtimeRefs": mergeStringMaps(
			map[string]string{
				"promptRef":  ticket.PromptRef,
				"promptPath": promptPath,
				"workerSpec": workerSpecPath,
			},
			orchestration.PromptRefs(paths.Root),
		),
	}
	if err := writeJSON(ticketPath, dispatchTicket); err != nil {
		return DispatchBundle{}, err
	}
	prompt := buildPrompt(ticketPath, workerSpecPath, artifactDir, task, ticket)
	if err := os.WriteFile(promptPath, []byte(prompt), 0o644); err != nil {
		return DispatchBundle{}, err
	}
	return DispatchBundle{
		TicketPath:     ticketPath,
		WorkerSpecPath: workerSpecPath,
		PromptPath:     promptPath,
		ArtifactDir:    artifactDir,
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

func buildPrompt(ticketPath, workerSpecPath, artifactDir string, task adapter.Task, ticket dispatch.Ticket) string {
	lines := []string{
		"You are the Klein worker for exactly one bound task inside a repo-local closed-loop runtime.",
		"",
		"Read order:",
		fmt.Sprintf("1. Read the immutable dispatch ticket first: %s", ticketPath),
		fmt.Sprintf("2. Read the task-local worker spec: %s", workerSpecPath),
		"3. If task-local artifacts already exist, read worker-result.json, verify.json, handoff.md, and referenced compact handoff logs.",
		"4. Read only the files explicitly referenced by the ticket before expanding your search.",
		"",
		"Hard authority rules:",
		"- Never create or mutate thread keys, request ids, task ids, plan epochs, leases, or global `.harness/state/*` ledgers.",
		"- Never edit files outside the bound worktree.",
		"- Never edit paths outside `allowedWriteGlobs`.",
		"- Never edit `blockedWriteGlobs`.",
		"- Never write task-local outputs outside `artifactDir`.",
		"- Never merge, rebase, push, archive, delete branches, or delete worktrees.",
		"- Never decide that the loop is complete. You may only decide the terminal outcome of this worker run.",
		"",
		"Execution style:",
		"- Fix root causes, not symptoms.",
		"- Keep changes minimal, focused, and consistent with the existing codebase.",
		"- Read a file before editing it.",
		"- Before each meaningful tool/action group, briefly state your immediate intent.",
		"- Follow the task-local loop in order: context assembly -> targeted research -> refine worker-spec understanding -> execute -> verify -> handoff.",
		"- Do not skip directly from the request text to edits when the referenced files have not been read yet.",
		"- The outer runtime already owns submit -> route -> dispatch. Do not recreate a second outer orchestrator inside this task.",
		"- If bounded packet synthesis is required inside this task, keep it task-local: 3 candidate worker-spec refinements, 1 judge, no new global task set.",
		"",
		"Verification:",
		"- Run verify commands from the dispatch ticket in order.",
		"- Start with the narrowest relevant validation, then broader checks when required.",
		"- Record each command, exit code, and output path in verify.json.",
		"- A noop completion is valid only when acceptance is already satisfied and verify.json records concrete evidence for that claim.",
		"",
		"Required artifacts before exit:",
		fmt.Sprintf("- %s", workerSpecPath),
		fmt.Sprintf("- %s", filepath.Join(artifactDir, "worker-result.json")),
		fmt.Sprintf("- %s", filepath.Join(artifactDir, "verify.json")),
		fmt.Sprintf("- %s", filepath.Join(artifactDir, "handoff.md")),
		"",
		"Task focus:",
		fmt.Sprintf("- taskId: %s", task.TaskID),
		fmt.Sprintf("- planEpoch: %d", task.PlanEpoch),
		fmt.Sprintf("- roleHint: %s", task.RoleHint),
		fmt.Sprintf("- taskKind: %s", task.Kind),
		fmt.Sprintf("- workerMode: %s", task.WorkerMode),
		fmt.Sprintf("- routingModel: %s", task.RoutingModel),
		fmt.Sprintf("- executionModel: %s", task.ExecutionModel),
		fmt.Sprintf("- orchestrationSessionId: %s", task.OrchestrationSessionID),
		fmt.Sprintf("- promptStages: %s", strings.Join(task.PromptStages, ", ")),
		fmt.Sprintf("- title: %s", task.Title),
		fmt.Sprintf("- summary: %s", task.Summary),
		fmt.Sprintf("- description: %s", task.Description),
		fmt.Sprintf("- ownedPaths: %s", strings.Join(task.OwnedPaths, ", ")),
		fmt.Sprintf("- verificationRuleIds: %s", strings.Join(task.VerificationRuleIDs, ", ")),
		fmt.Sprintf("- promptRef: %s", ticket.PromptRef),
		fmt.Sprintf("- promptDir: %s", filepath.Join("prompts", "spec")),
		fmt.Sprintf("- runtimeReadme: %s", filepath.Join("prompts", "spec", "README.md")),
		fmt.Sprintf("- orchestratorPrompt: %s", filepath.Join("prompts", "spec", "orchestrator.md")),
		fmt.Sprintf("- packetWorkflow: %s", filepath.Join("prompts", "spec", "propose.md")),
		fmt.Sprintf("- orchestrationPacketGuide: %s", filepath.Join("prompts", "spec", "packet.md")),
		fmt.Sprintf("- workerSpecGuide: %s", filepath.Join("prompts", "spec", "worker-spec.md")),
		fmt.Sprintf("- dispatchTicketGuide: %s", filepath.Join("prompts", "spec", "dispatch-ticket.md")),
		fmt.Sprintf("- workerResultGuide: %s", filepath.Join("prompts", "spec", "worker-result.md")),
		fmt.Sprintf("- applyWorkflow: %s", filepath.Join("prompts", "spec", "apply.md")),
		fmt.Sprintf("- verifyWorkflow: %s", filepath.Join("prompts", "spec", "verify.md")),
		fmt.Sprintf("- archiveWorkflow: %s", filepath.Join("prompts", "spec", "archive.md")),
		fmt.Sprintf("- plannerArchitecture: %s", filepath.Join("prompts", "spec", "planner-architecture.md")),
		fmt.Sprintf("- plannerDelivery: %s", filepath.Join("prompts", "spec", "planner-delivery.md")),
		fmt.Sprintf("- plannerRisk: %s", filepath.Join("prompts", "spec", "planner-risk.md")),
		fmt.Sprintf("- judgePrompt: %s", filepath.Join("prompts", "spec", "judge.md")),
		"",
		"Final response:",
		"- Be brief.",
		"- Report only the terminal worker outcome and the key artifact path(s).",
		"- Do not claim global completion.",
	}
	return strings.Join(lines, "\n") + "\n"
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
