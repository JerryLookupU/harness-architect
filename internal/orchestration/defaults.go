package orchestration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PlannerAgent struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Focus     string `json:"focus"`
	PromptRef string `json:"promptRef"`
}

type JudgeAgent struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Focus      string   `json:"focus"`
	PromptRef  string   `json:"promptRef"`
	Dimensions []string `json:"dimensions"`
}

type PacketSynthesisLoop struct {
	PlannerCount     int            `json:"plannerCount"`
	Planners         []PlannerAgent `json:"planners"`
	Judge            JudgeAgent     `json:"judge"`
	PacketFields     []string       `json:"packetFields"`
	WorkerSpecFields []string       `json:"workerSpecFields"`
}

const promptDir = "prompts/spec"

func DefaultPromptStages() []string {
	return []string{
		"context_assembly",
		"packet_parallel_planning",
		"packet_judging",
		"worker_spec_synthesis",
		"execute",
		"verify",
		"handoff",
	}
}

func PromptDir(root string) string {
	return filepath.Join(root, promptDir)
}

func PromptFiles() []string {
	return []string{
		"README.md",
		"orchestrator.md",
		"propose.md",
		"packet.md",
		"worker-spec.md",
		"dispatch-ticket.md",
		"worker-result.md",
		"apply.md",
		"verify.md",
		"archive.md",
		"planner-architecture.md",
		"planner-delivery.md",
		"planner-risk.md",
		"judge.md",
	}
}

func PromptRefs(root string) map[string]string {
	dir := PromptDir(root)
	return map[string]string{
		"promptDir":                dir,
		"runtimeReadme":            filepath.Join(dir, "README.md"),
		"orchestratorPrompt":       filepath.Join(dir, "orchestrator.md"),
		"packetWorkflow":           filepath.Join(dir, "propose.md"),
		"orchestrationPacketGuide": filepath.Join(dir, "packet.md"),
		"workerSpecGuide":          filepath.Join(dir, "worker-spec.md"),
		"dispatchTicketGuide":      filepath.Join(dir, "dispatch-ticket.md"),
		"workerResultGuide":        filepath.Join(dir, "worker-result.md"),
		"applyWorkflow":            filepath.Join(dir, "apply.md"),
		"verifyWorkflow":           filepath.Join(dir, "verify.md"),
		"archiveWorkflow":          filepath.Join(dir, "archive.md"),
		"plannerArchitecture":      filepath.Join(dir, "planner-architecture.md"),
		"plannerDelivery":          filepath.Join(dir, "planner-delivery.md"),
		"plannerRisk":              filepath.Join(dir, "planner-risk.md"),
		"judgePrompt":              filepath.Join(dir, "judge.md"),
	}
}

func DefaultPacketSynthesisLoop(root string) PacketSynthesisLoop {
	promptsDir := PromptDir(root)
	return PacketSynthesisLoop{
		PlannerCount: 3,
		Planners: []PlannerAgent{
			{
				ID:        "packet-architecture",
				Name:      "Packet Planner A",
				Focus:     "Architecture fit, authority boundaries, and bounded change shape",
				PromptRef: filepath.Join(promptsDir, "planner-architecture.md"),
			},
			{
				ID:        "packet-delivery",
				Name:      "Packet Planner B",
				Focus:     "Incremental delivery, worker-spec slicing, and dependency order",
				PromptRef: filepath.Join(promptsDir, "planner-delivery.md"),
			},
			{
				ID:        "packet-risk",
				Name:      "Packet Planner C",
				Focus:     "Risk, verification, rollback, noop-validation, and phase-1 control-plane fit",
				PromptRef: filepath.Join(promptsDir, "planner-risk.md"),
			},
		},
		Judge: JudgeAgent{
			ID:        "packet-judge",
			Name:      "Packet Judge",
			Focus:     "Choose one packet candidate and format final execution-ready worker-spec slices",
			PromptRef: filepath.Join(promptsDir, "judge.md"),
			Dimensions: []string{
				"packet_clarity",
				"repo_fit",
				"execution_feasibility",
				"verification_completeness",
				"rollback_risk",
			},
		},
		PacketFields: []string{
			"objective",
			"constraints",
			"selectedPlan",
			"rejectedAlternatives",
			"executionTasks",
			"verificationPlan",
			"decisionRationale",
			"ownedPaths",
			"taskBudgets",
			"acceptanceMarkers",
			"replanTriggers",
			"rollbackHints",
		},
		WorkerSpecFields: []string{
			"taskId",
			"objective",
			"constraints",
			"ownedPaths",
			"blockedPaths",
			"taskBudget",
			"acceptanceMarkers",
			"verificationPlan",
			"replanTriggers",
			"rollbackHints",
		},
	}
}

func DefaultTopLevelPrompt(root, userPrompt string) string {
	base := loadPromptOrFallback(
		filepath.Join(PromptDir(root), "orchestrator.md"),
		`You are the Klein orchestration agent.

The repo-local runtime owns the outer loop:
- submit -> classify -> fuse -> bind -> route -> issue dispatch ticket -> ingest outcome -> verify -> refresh summaries

When orchestration packet synthesis is needed, use the default b3e convergence subunit:
- run 3 isolated planners that each produce one orchestration packet candidate plus task-local worker-spec candidates
- have 1 judge select and format the final runtime-owned packet`,
	)
	lines := []string{
		strings.TrimSpace(base),
		"",
		"Load supporting runtime prompts from prompts/spec in this order when relevant:",
	}
	for _, file := range PromptFiles() {
		if file == "README.md" || file == "orchestrator.md" {
			continue
		}
		lines = append(lines, fmt.Sprintf("- prompts/spec/%s", file))
	}
	lines = append(lines,
		"",
		"User requirement:",
		strings.TrimSpace(userPrompt),
		"",
		"Final orchestration packet must include:",
		"- objective",
		"- constraints",
		"- selectedPlan",
		"- rejectedAlternatives",
		"- executionTasks",
		"- verificationPlan",
		"- decisionRationale",
		"- ownedPaths",
		"- taskBudgets",
		"- acceptanceMarkers",
		"- replanTriggers",
		"- rollbackHints",
	)
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func loadPromptOrFallback(path, fallback string) string {
	payload, err := os.ReadFile(path)
	if err != nil {
		return strings.TrimSpace(fallback)
	}
	text := strings.TrimSpace(string(payload))
	if text == "" {
		return strings.TrimSpace(fallback)
	}
	return text
}
