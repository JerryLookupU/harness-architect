package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"klein-harness/internal/a2a"
	"klein-harness/internal/adapter"
	"klein-harness/internal/dispatch"
	"klein-harness/internal/route"
	runtimepkg "klein-harness/internal/runtime"
	"klein-harness/internal/state"
	"klein-harness/internal/verify"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	var err error
	switch os.Args[1] {
	case "route":
		err = runRoute(os.Args[2:])
	case "ingest-verification":
		err = runIngestVerification(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: kh-orchestrator <route|ingest-verification> [args...]")
}

func runRoute(args []string) error {
	fs := flag.NewFlagSet("route", flag.ContinueOnError)
	root := fs.String("root", ".", "project root")
	taskID := fs.String("task-id", "", "task id")
	requestID := fs.String("request-id", "", "request id")
	causationID := fs.String("causation-id", "", "causation id")
	attempt := fs.Int("attempt", 0, "attempt")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *taskID == "" {
		return fmt.Errorf("missing --task-id")
	}
	task, err := adapter.LoadTask(*root, *taskID)
	if err != nil {
		return err
	}
	latestPlanEpoch, err := adapter.LoadLatestPlanEpoch(*root, task)
	if err != nil {
		return err
	}
	checkpointFresh, err := adapter.LoadCheckpointFresh(*root, *taskID)
	if err != nil {
		return err
	}
	registry, err := adapter.LoadSessionRegistry(*root)
	if err != nil {
		return err
	}
	sessionContested := false
	if task.PreferredResumeSessionID != "" {
		for _, binding := range registry.ActiveBindings {
			if binding.TaskID != task.TaskID && binding.SessionID == task.PreferredResumeSessionID {
				sessionContested = true
				break
			}
		}
	}
	paths, err := adapter.Resolve(*root)
	if err != nil {
		return err
	}
	currentAttempt := *attempt
	if currentAttempt <= 0 {
		count, err := adapter.CountDispatchAttempts(*root, *taskID)
		if err != nil {
			return err
		}
		currentAttempt = count + 1
	}
	requiredSummaryVersion := runtimeSummaryVersion(paths.RuntimePath)
	routeInput, err := runtimepkg.BuildRouteInput(*root, task, latestPlanEpoch, checkpointFresh, sessionContested, requiredSummaryVersion)
	if err != nil {
		return err
	}
	decision := route.Evaluate(routeInput)
	payload, err := a2a.NewPayload(decision)
	if err != nil {
		return err
	}
	routeCausation := *causationID
	if routeCausation == "" {
		routeCausation = fmt.Sprintf("route:%s:%d:%d", task.TaskID, task.PlanEpoch, currentAttempt)
	}
	routeEvent, err := a2a.AppendEvent(paths.EventLogPath, a2a.Envelope{
		Kind:           "route.decided",
		IdempotencyKey: routeCausation,
		TraceID:        *requestID,
		CausationID:    routeCausation,
		From:           "orchestrator-node",
		To:             "worker-supervisor-node",
		RequestID:      *requestID,
		TaskID:         task.TaskID,
		PlanEpoch:      task.PlanEpoch,
		Attempt:        currentAttempt,
		SessionID:      decision.ResumeSessionID,
		ReasonCodes:    decision.ReasonCodes,
		Payload:        payload,
	})
	if err != nil {
		return err
	}
	var ticket *dispatch.Ticket
	if decision.DispatchReady {
		issued, _, err := dispatch.Issue(dispatch.IssueRequest{
			Root:                   *root,
			RequestID:              *requestID,
			TaskID:                 task.TaskID,
			ThreadKey:              task.ThreadKey,
			PlanEpoch:              task.PlanEpoch,
			Attempt:                currentAttempt,
			IdempotencyKey:         fmt.Sprintf("dispatch:%s:epoch_%d:attempt_%d", task.TaskID, task.PlanEpoch, currentAttempt),
			CausationID:            routeEvent.Event.MessageID,
			ReasonCodes:            decision.ReasonCodes,
			WorkerClass:            defaultWorkerClass(task.ExecutionModel),
			Cwd:                    adapter.TaskCWD(paths, task),
			Command:                adapter.DispatchCommand(task),
			PromptRef:              promptRefForTask(task.RoleHint),
			Budget:                 dispatch.Budget{MaxTurns: 8, MaxMinutes: 20, MaxToolCalls: 30},
			LeaseTTLSec:            1800,
			RequiredSummaryVersion: decision.RequiredSummaryVersion,
			ResumeSessionID:        decision.ResumeSessionID,
			WorktreePath:           decision.WorktreePath,
			OwnedPaths:             decision.OwnedPaths,
		})
		if err != nil {
			return err
		}
		ticket = &issued
	}
	task.Status = statusForRouteDecision(decision.Route, decision.DispatchReady)
	task.StatusReason = fmt.Sprintf("route=%s reasons=%s", decision.Route, strings.Join(decision.ReasonCodes, ", "))
	task.UpdatedAt = state.NowUTC()
	if ticket != nil {
		task.LastDispatchID = ticket.DispatchID
	}
	if err := adapter.UpsertTask(*root, task); err != nil {
		return err
	}
	if err := runtimepkg.RefreshExecutionIndexesForTask(*root, task); err != nil {
		return err
	}
	return writeStdout(map[string]any{
		"decision": decision,
		"dispatch": ticket,
	})
}

func runIngestVerification(args []string) error {
	fs := flag.NewFlagSet("ingest-verification", flag.ContinueOnError)
	root := fs.String("root", ".", "project root")
	taskID := fs.String("task-id", "", "task id")
	dispatchID := fs.String("dispatch-id", "", "dispatch id")
	requestID := fs.String("request-id", "", "request id")
	causationID := fs.String("causation-id", "", "causation id")
	status := fs.String("status", "", "verification status")
	summary := fs.String("summary", "", "verification summary")
	resultPath := fs.String("verification-result-path", "", "verification result path")
	followUp := fs.String("follow-up", "", "follow-up kind")
	planEpoch := fs.Int("plan-epoch", 0, "plan epoch")
	attempt := fs.Int("attempt", 1, "attempt")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *taskID == "" || *status == "" {
		return fmt.Errorf("missing --task-id or --status")
	}
	result, err := verify.Ingest(verify.Request{
		Root:                   *root,
		RequestID:              *requestID,
		TaskID:                 *taskID,
		DispatchID:             *dispatchID,
		PlanEpoch:              *planEpoch,
		Attempt:                *attempt,
		CausationID:            *causationID,
		Status:                 *status,
		Summary:                *summary,
		VerificationResultPath: *resultPath,
		FollowUp:               *followUp,
	})
	finalFollowUp := ""
	if err == nil {
		finalFollowUp = result.FollowUpEvent
	}
	if _, finalizeErr := runtimepkg.FinalizeTaskAfterVerification(*root, *taskID, *dispatchID, *status, *summary, *resultPath, finalFollowUp, err); finalizeErr != nil {
		return finalizeErr
	}
	if err != nil {
		return err
	}
	return writeStdout(result)
}

func runtimeSummaryVersion(path string) string {
	var payload struct {
		GeneratedAt string `json:"generatedAt"`
		Revision    int64  `json:"revision"`
	}
	if err := state.LoadJSON(path, &payload); err != nil {
		return "runtime:unknown"
	}
	if payload.Revision > 0 {
		return fmt.Sprintf("state.v%d", payload.Revision)
	}
	if payload.GeneratedAt != "" {
		return "runtime:" + payload.GeneratedAt
	}
	return "runtime:legacy"
}

func statusForRouteDecision(routeName string, dispatchReady bool) string {
	switch {
	case dispatchReady:
		return "routing"
	case routeName == "replan":
		return "needs_replan"
	default:
		return "blocked"
	}
}

func defaultWorkerClass(model string) string {
	if model == "" {
		return "codex-go"
	}
	return model
}

func promptRefForTask(roleHint string) string {
	if roleHint == "orchestrator" {
		return "prompts/orchestrator-burst.md"
	}
	return "prompts/worker-burst.md"
}

func writeStdout(value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(payload, '\n'))
	return err
}
