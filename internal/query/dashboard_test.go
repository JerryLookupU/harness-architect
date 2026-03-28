package query

import (
	"os"
	"path/filepath"
	"testing"

	"klein-harness/internal/adapter"
	"klein-harness/internal/bootstrap"
	"klein-harness/internal/orchestration"
	"klein-harness/internal/runtime"
)

func TestProjectDashboardSurfacesThreadLandingsPlanningAndTokenUsage(t *testing.T) {
	root := t.TempDir()
	paths, err := bootstrap.Init(root)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	first, err := runtime.Submit(runtime.SubmitRequest{
		Root: root,
		Goal: "Build dashboard visibility for harness execution",
	})
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	second, err := runtime.Submit(runtime.SubmitRequest{
		Root:     root,
		Goal:     "Build dashboard visibility for harness execution",
		Contexts: []string{"Add token heat tracking"},
	})
	if err != nil {
		t.Fatalf("second submit: %v", err)
	}
	if second.Task.TaskID != first.Task.TaskID {
		t.Fatalf("expected second submit to reuse the pending task, got first=%s second=%s", first.Task.TaskID, second.Task.TaskID)
	}

	if err := adapter.UpsertTask(root, adapter.Task{
		TaskID:         first.Task.TaskID,
		ThreadKey:      first.Task.ThreadKey,
		Title:          "Dashboard visibility",
		Summary:        "Build dashboard visibility for harness execution",
		Status:         "running",
		PlanEpoch:      1,
		LastDispatchID: "dispatch-dash",
		OwnedPaths:     []string{"internal/query/**"},
		UpdatedAt:      "2026-03-27T10:00:00Z",
	}); err != nil {
		t.Fatalf("upsert task: %v", err)
	}

	dispatchTicket := `{
		"resumeStrategy":"fresh",
		"sessionId":"sess-dash",
		"promptStages":["context_assembly","packet_parallel_planning","packet_judging","execute","verify"],
		"planningTracePath":"` + filepath.Join(paths.StateDir, "planning-trace-"+first.Task.TaskID+".md") + `",
		"constraintPath":"` + filepath.Join(paths.StateDir, "constraints-"+first.Task.TaskID+".json") + `",
		"acceptedPacketPath":"` + orchestration.AcceptedPacketPath(root, first.Task.TaskID) + `",
		"taskContractPath":"` + filepath.Join(paths.ArtifactsDir, first.Task.TaskID, "dispatch-dash", "task-contract.json") + `",
		"executionSliceId":"` + first.Task.TaskID + `.slice.1",
		"runtimeRefs":{"promptPath":"` + filepath.Join(paths.StateDir, "runner-prompt-"+first.Task.TaskID+".md") + `"},
		"methodology":{"mode":"fact-first","guidePath":"prompts/spec/methodology.md","coreRules":[],"activeLenses":[],"activeSkills":["qiushi-execution"]},
		"judgeDecision":{"judgeId":"packet-judge","judgeName":"Packet Judge","selectedFlow":"dashboard packet","winnerStrategy":"bounded winner","rationale":["prefer observable state","preserve operator clarity"],"selectedDimensions":["execution_feasibility"],"selectedLensIds":["operator_surface"],"reviewRequired":true,"verifyRequired":true},
		"plannerCandidates":[
			{"plannerId":"packet-architecture","plannerName":"Packet Planner A","focus":"Architecture fit","taskName":"Dashboard 可视化任务","proposedFlow":"dashboard packet","resultSummary":"stabilize task naming","keyMoves":["keep thread naming stable"],"risks":["authority drift"],"evidence":["policyTags: operator_surface"],"materializedBy":"test"},
			{"plannerId":"packet-delivery","plannerName":"Packet Planner B","focus":"Delivery slicing","taskName":"Dashboard 可视化任务","proposedFlow":"dashboard packet","resultSummary":"split into visible slices","keyMoves":["surface aggregate task"],"risks":["slice fragmentation"],"evidence":["acceptedPacket=packet-dashboard"],"materializedBy":"test"},
			{"plannerId":"packet-risk","plannerName":"Packet Planner C","focus":"Risk and verify","taskName":"Dashboard 可视化任务","proposedFlow":"dashboard packet","resultSummary":"keep token evidence visible","keyMoves":["keep verify visible"],"risks":["token usage missing"],"evidence":["verifySteps: go test ./..."],"materializedBy":"test"}
		],
		"executionLoop":{"mode":"qiushi execution / validation loop","owner":"worker + verify + runtime closeout","skillPath":"skills/qiushi-execution/SKILL.md","activeSkills":["qiushi-execution"],"skillHints":["keep token evidence visible"],"phases":["investigate","execute","verify"],"coreRules":[],"retryTransition":"retry"},
		"constraintSystem":{"mode":"two-level layered constraints","objective":"operator clarity","generation":"gen","rules":[]},
		"packetSynthesis":{"plannerCount":3,"planners":[
			{"id":"packet-architecture","name":"Packet Planner A","focus":"Architecture fit","promptRef":"planner-architecture.md"},
			{"id":"packet-delivery","name":"Packet Planner B","focus":"Delivery slicing","promptRef":"planner-delivery.md"},
			{"id":"packet-risk","name":"Packet Planner C","focus":"Risk and verify","promptRef":"planner-risk.md"}
		],"judge":{"id":"packet-judge","name":"Packet Judge","focus":"merge","promptRef":"judge.md","dimensions":["packet_clarity"]},"packetFields":[],"workerSpecFields":[]}
	}`
	if err := os.WriteFile(filepath.Join(paths.StateDir, "dispatch-ticket-"+first.Task.TaskID+".json"), []byte(dispatchTicket), 0o644); err != nil {
		t.Fatalf("write dispatch ticket: %v", err)
	}
	if err := os.WriteFile(filepath.Join(paths.StateDir, "planning-trace-"+first.Task.TaskID+".md"), []byte("trace line 1\ntrace line 2\n"), 0o644); err != nil {
		t.Fatalf("write planning trace: %v", err)
	}

	if err := orchestration.WriteAcceptedPacket(orchestration.AcceptedPacketPath(root, first.Task.TaskID), orchestration.AcceptedPacket{
		SchemaVersion:     "kh.accepted-packet.v1",
		Generator:         "test",
		GeneratedAt:       "2026-03-27T10:01:00Z",
		TaskID:            first.Task.TaskID,
		ThreadKey:         first.Task.ThreadKey,
		PlanEpoch:         1,
		PacketID:          "packet-dashboard",
		Objective:         "Show harness execution clearly",
		Constraints:       []string{"Preserve operator readability", "Prefer real evidence"},
		FlowSelection:     "dashboard packet",
		SelectedPlan:      "Render planner lanes, checklist, execution chain, and token ledger",
		ExecutionTasks:    []orchestration.ExecutionTask{{ID: first.Task.TaskID + ".slice.1", Title: "dashboard page", Summary: "ship the page", DoneCriteria: []string{"page renders"}, RequiredEvidence: []string{"token ledger visible"}}, {ID: first.Task.TaskID + ".slice.2", Title: "follow-up", Summary: "accept appended change"}},
		VerificationPlan:  map[string]any{"commands": []any{map[string]any{"title": "go test ./..."}}},
		DecisionRationale: "dashboard should stay evidence-first",
		OwnedPaths:        []string{"internal/query/**", "cmd/harness/**"},
		ReplanTriggers:    []string{"token_usage_missing"},
		RollbackHints:     []string{"preserve request lineage"},
		AcceptedAt:        "2026-03-27T10:01:00Z",
		AcceptedBy:        "test",
	}); err != nil {
		t.Fatalf("write accepted packet: %v", err)
	}
	contractPath := filepath.Join(paths.ArtifactsDir, first.Task.TaskID, "dispatch-dash", "task-contract.json")
	if err := os.MkdirAll(filepath.Dir(contractPath), 0o755); err != nil {
		t.Fatalf("mkdir task contract: %v", err)
	}
	if err := orchestration.WriteTaskContract(contractPath, orchestration.TaskContract{
		SchemaVersion:         "kh.task-contract.v1",
		Generator:             "test",
		GeneratedAt:           "2026-03-27T10:01:00Z",
		ContractID:            "contract-dashboard",
		TaskID:                first.Task.TaskID,
		DispatchID:            "dispatch-dash",
		ThreadKey:             first.Task.ThreadKey,
		PlanEpoch:             1,
		ExecutionSliceID:      first.Task.TaskID + ".slice.1",
		Objective:             "Show harness execution clearly",
		DoneCriteria:          []string{"task flow visible"},
		AcceptanceMarkers:     []string{"token ledger visible"},
		VerificationChecklist: []orchestration.VerificationChecklistItem{{ID: "tokens", Title: "token ledger visible", Required: true, Status: "pass", Detail: "turn.completed usage parsed"}},
		RequiredEvidence:      []string{"runner log"},
		ReviewRequired:        true,
		ContractStatus:        "accepted",
		ProposedBy:            "test",
		AcceptedBy:            "test",
		AcceptedAt:            "2026-03-27T10:01:00Z",
		AcceptedPacketPath:    orchestration.AcceptedPacketPath(root, first.Task.TaskID),
	}); err != nil {
		t.Fatalf("write task contract: %v", err)
	}

	outcomePath := filepath.Join(root, ".harness", "checkpoints", first.Task.TaskID, "outcome.json")
	if err := os.MkdirAll(filepath.Dir(outcomePath), 0o755); err != nil {
		t.Fatalf("mkdir outcome dir: %v", err)
	}
	outcome := `{
		"schemaVersion":"1.0",
		"status":"succeeded",
		"summary":"bounded burst completed",
		"generatedAt":"2026-03-27T10:02:00Z",
		"startedAt":"2026-03-27T10:01:30Z",
		"finishedAt":"2026-03-27T10:02:00Z",
		"stdout":"{\"type\":\"thread.started\",\"thread_id\":\"thread-demo\"}\n{\"type\":\"turn.started\"}\n{\"type\":\"turn.completed\",\"usage\":{\"input_tokens\":123,\"cached_input_tokens\":45,\"output_tokens\":67}}\n"
	}`
	if err := os.WriteFile(outcomePath, []byte(outcome), 0o644); err != nil {
		t.Fatalf("write outcome: %v", err)
	}
	tmuxSummary := `{
		"sessions": {
			"tmux-one": {
				"sessionName":"tmux-one",
				"taskId":"` + first.Task.TaskID + `",
				"dispatchId":"dispatch-dash-1",
				"workerId":"worker-a",
				"status":"succeeded",
				"startedAt":"2026-03-27T10:01:20Z",
				"finishedAt":"2026-03-27T10:01:40Z"
			},
			"tmux-two": {
				"sessionName":"tmux-two",
				"taskId":"` + first.Task.TaskID + `",
				"dispatchId":"dispatch-dash-2",
				"workerId":"worker-b",
				"status":"succeeded",
				"startedAt":"2026-03-27T10:01:41Z",
				"finishedAt":"2026-03-27T10:02:00Z"
			}
		},
		"latestByTask": {
			"` + first.Task.TaskID + `": "tmux-two"
		}
	}`
	if err := os.WriteFile(paths.TmuxSummaryPath, []byte(tmuxSummary), 0o644); err != nil {
		t.Fatalf("write tmux summary: %v", err)
	}

	dashboard, err := ProjectDashboard(root)
	if err != nil {
		t.Fatalf("project dashboard: %v", err)
	}
	if dashboard.Overview.TotalTasks != 1 || dashboard.Overview.TotalThreads != 1 {
		t.Fatalf("unexpected dashboard overview: %+v", dashboard.Overview)
	}
	if dashboard.Overview.TokenUsage.InputTokens != 123 || dashboard.Overview.TokenUsage.CachedInputTokens != 45 || dashboard.Overview.TokenUsage.OutputTokens != 67 {
		t.Fatalf("unexpected token usage: %+v", dashboard.Overview.TokenUsage)
	}
	if len(dashboard.Threads) != 1 || len(dashboard.Threads[0].RequestLandings) != 2 {
		t.Fatalf("expected thread to show both request landings: %+v", dashboard.Threads)
	}
	if len(dashboard.TaskFlows) != 1 {
		t.Fatalf("expected one task flow, got %+v", dashboard.TaskFlows)
	}
	flow := dashboard.TaskFlows[0]
	if flow.Name == "" {
		t.Fatalf("expected flow name, got %+v", flow)
	}
	if len(flow.Planning.PlannerLanes) != 3 {
		t.Fatalf("expected planner lanes, got %+v", flow.Planning)
	}
	if flow.Planning.PlannerLanes[0].Inferred || flow.Planning.PlannerLanes[0].TaskName != "Dashboard 可视化任务" {
		t.Fatalf("expected materialized planner candidates, got %+v", flow.Planning.PlannerLanes[0])
	}
	if flow.Planning.Judge == nil || flow.Planning.Judge.SelectedFlow != "dashboard packet" {
		t.Fatalf("expected judge merge view, got %+v", flow.Planning.Judge)
	}
	if len(flow.TaskList) != 2 || len(flow.Checklist) == 0 {
		t.Fatalf("expected tasklist/checklist in flow: %+v", flow)
	}
	if flow.Model.Objective == "" || len(flow.Model.Deliverables) == 0 {
		t.Fatalf("expected model view to surface objective and deliverables, got %+v", flow.Model)
	}
	if flow.Runtime.DispatchID != "dispatch-dash" || flow.Runtime.CurrentSliceID != first.Task.TaskID+".slice.1" {
		t.Fatalf("expected runtime view to surface dispatch and slice, got %+v", flow.Runtime)
	}
	if flow.ExecutionMode != "tmux" || flow.Runtime.ExecutionMode != "tmux" {
		t.Fatalf("expected tmux execution mode in flow/runtime, got flow=%q runtime=%q", flow.ExecutionMode, flow.Runtime.ExecutionMode)
	}
	if flow.Operator.Headline == "" || len(flow.Operator.HumanTaskList) == 0 {
		t.Fatalf("expected operator view to surface human progress, got %+v", flow.Operator)
	}
	if len(flow.Model.Boundaries) != 2 {
		t.Fatalf("expected narrow boundaries to stay optional but visible, got %+v", flow.Model.Boundaries)
	}
	if flow.CurrentSliceID != first.Task.TaskID+".slice.1" {
		t.Fatalf("expected current slice id, got %+v", flow)
	}
	if flow.TokenUsage.InputTokens != 123 || flow.TokenUsage.Turns != 1 {
		t.Fatalf("expected task flow token usage from outcome log: %+v", flow.TokenUsage)
	}
	tmuxEvents := []ExecutionEvent{}
	for _, event := range flow.ExecutionChain {
		if event.Kind == "tmux.session" {
			tmuxEvents = append(tmuxEvents, event)
		}
	}
	if len(tmuxEvents) != 2 {
		t.Fatalf("expected all tmux sessions in execution chain, got %+v", tmuxEvents)
	}
	dispatches := map[string]bool{}
	workers := map[string]bool{}
	for _, event := range tmuxEvents {
		dispatches[event.DispatchID] = true
		workers[event.WorkerID] = true
	}
	if !dispatches["dispatch-dash-1"] || !dispatches["dispatch-dash-2"] {
		t.Fatalf("expected tmux dispatch ids to be tracked, got %+v", tmuxEvents)
	}
	if !workers["worker-a"] || !workers["worker-b"] {
		t.Fatalf("expected tmux worker ids to be tracked, got %+v", tmuxEvents)
	}
}

func TestProjectDashboardSurfacesDirectFallbackExecutionMode(t *testing.T) {
	root := t.TempDir()
	if _, err := bootstrap.Init(root); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := adapter.UpsertTask(root, adapter.Task{
		TaskID:         "T-direct",
		ThreadKey:      "thread-direct",
		Title:          "Direct fallback task",
		Summary:        "Run without tmux session",
		Status:         "completed",
		PlanEpoch:      1,
		LastDispatchID: "dispatch-direct",
		ExecutionMode:  "direct_fallback",
		TmuxLogPath:    filepath.Join(root, ".harness", "logs", "tmux", "T-direct", "dispatch-direct.log"),
		UpdatedAt:      "2026-03-28T10:00:00Z",
	}); err != nil {
		t.Fatalf("upsert task: %v", err)
	}

	dashboard, err := ProjectDashboard(root)
	if err != nil {
		t.Fatalf("project dashboard: %v", err)
	}
	if len(dashboard.TaskFlows) != 1 {
		t.Fatalf("expected one task flow, got %+v", dashboard.TaskFlows)
	}
	flow := dashboard.TaskFlows[0]
	if flow.ExecutionMode != "direct_fallback" || flow.Runtime.ExecutionMode != "direct_fallback" {
		t.Fatalf("expected direct fallback execution mode, got flow=%q runtime=%q", flow.ExecutionMode, flow.Runtime.ExecutionMode)
	}
	found := false
	for _, event := range flow.ExecutionChain {
		if event.Kind == "worker.direct_fallback" && event.Path == filepath.Join(root, ".harness", "logs", "tmux", "T-direct", "dispatch-direct.log") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected synthetic direct fallback worker event, got %+v", flow.ExecutionChain)
	}
}

func TestShouldSurfaceBoundaryHintsTreatsOwnedPathsAsOptional(t *testing.T) {
	if shouldSurfaceBoundaryHints(nil) {
		t.Fatal("expected empty ownedPaths to stay optional")
	}
	if shouldSurfaceBoundaryHints([]string{"a/**", "b/**", "c/**", "d/**", "e/**"}) {
		t.Fatal("expected broad ownedPaths to stay out of model view")
	}
	if !shouldSurfaceBoundaryHints([]string{"rundata/mathematicians/**"}) {
		t.Fatal("expected narrow ownedPaths to remain available as optional boundary hints")
	}
}
