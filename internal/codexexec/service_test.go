package codexexec

import (
	"errors"
	"strings"
	"testing"

	"klein-harness/internal/codexconfig"
	"klein-harness/internal/instructions"
)

func TestBuildCommandProfileUsesCodexProtocol(t *testing.T) {
	profile := codexconfig.Profile{
		Model:          "gpt-5.4",
		ApprovalPolicy: "never",
		SandboxMode:    "workspace-write",
	}
	command := buildCommandProfile(false, Request{SkipGitRepoCheck: true}, profile)
	for _, want := range []string{
		"codex exec",
		"--json",
		"--output-last-message <LAST_MESSAGE_PATH>",
		"--model gpt-5.4",
		"--full-auto",
		"--skip-git-repo-check",
	} {
		if !strings.Contains(command, want) {
			t.Fatalf("command missing %q: %s", want, command)
		}
	}
	resumeCommand := buildCommandProfile(true, Request{}, profile)
	if !strings.Contains(resumeCommand, "codex exec resume <SESSION_ID>") {
		t.Fatalf("resume command missing session placeholder: %s", resumeCommand)
	}
}

func TestDetectNativeSessionIDPrefersNewEntry(t *testing.T) {
	before := []sessionIndexEntry{
		{ID: "sess-1"},
	}
	after := []sessionIndexEntry{
		{ID: "sess-1"},
		{ID: "sess-2"},
	}
	if got := detectNativeSessionID(before, after, ""); got != "sess-2" {
		t.Fatalf("unexpected detected session id: %s", got)
	}
	if got := detectNativeSessionID(before, before, "fallback"); got != "fallback" {
		t.Fatalf("unexpected fallback session id: %s", got)
	}
}

func TestBuildTaskUsesDefaultPacketConvergencePrompt(t *testing.T) {
	task := buildTask("/repo", Request{Prompt: "Add orchestration convergence."}, sessionRecord{
		ID:                     "sess-1",
		TaskID:                 "task-1",
		OrchestrationSessionID: "orch-1",
	}, "orch-1", codexconfig.Profile{
		Model:          "gpt-5.4",
		ApprovalPolicy: "never",
		SandboxMode:    "workspace-write",
	}, []instructions.File{{Path: "/repo/AGENTS.md"}})
	if !strings.Contains(task.Description, "dispatch ticket") || !strings.Contains(task.Description, "worker-spec") {
		t.Fatalf("task description missing orchestration defaults: %s", task.Description)
	}
	if len(task.PromptStages) < 4 || task.PromptStages[1] != "packet_parallel_planning" {
		t.Fatalf("unexpected prompt stages: %+v", task.PromptStages)
	}
}

func TestStatusForRouteDecisionMapsReplanAndBlock(t *testing.T) {
	if got := statusForRouteDecision("replan"); got != "needs_replan" {
		t.Fatalf("expected replan to map to needs_replan, got %q", got)
	}
	if got := statusForRouteDecision("block"); got != "blocked" {
		t.Fatalf("expected block to map to blocked, got %q", got)
	}
	if got := statusForRouteDecision("dispatch"); got != "queued" {
		t.Fatalf("expected dispatch fallback to stay queued, got %q", got)
	}
}

func TestStatusForBurstMapsTerminalStates(t *testing.T) {
	if got := statusForBurst("failed"); got != "needs_replan" {
		t.Fatalf("expected failed burst to map to needs_replan, got %q", got)
	}
	if got := statusForBurst("timed_out"); got != "needs_replan" {
		t.Fatalf("expected timed_out burst to map to needs_replan, got %q", got)
	}
	if got := statusForBurst("succeeded"); got != "succeeded" {
		t.Fatalf("expected succeeded burst to stay succeeded, got %q", got)
	}
}

func TestFinalTaskStatusCompletedAfterVerifiedCompletion(t *testing.T) {
	got := finalTaskStatus("succeeded", "passed", "task.completed", nil)
	if got != "completed" {
		t.Fatalf("expected completed final status, got %q", got)
	}
}

func TestFinalTaskStatusNeedsReplanForAnalysisLoop(t *testing.T) {
	got := finalTaskStatus("succeeded", "failed", "replan.emitted", nil)
	if got != "needs_replan" {
		t.Fatalf("expected needs_replan final status, got %q", got)
	}
	if followUp := effectiveFollowUp("replan.emitted", "succeeded", "failed", nil); followUp != "analysis.required" {
		t.Fatalf("expected effective follow up to become analysis.required, got %q", followUp)
	}
}

func TestFinalTaskStatusBlockedWhenVerificationErrors(t *testing.T) {
	got := finalTaskStatus("succeeded", "passed", "", errors.New("gate open"))
	if got != "blocked" {
		t.Fatalf("expected blocked final status on verification error, got %q", got)
	}
}
