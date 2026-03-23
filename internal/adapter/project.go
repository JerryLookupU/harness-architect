package adapter

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Paths struct {
	Root                 string
	HarnessDir           string
	StateDir             string
	EventsDir            string
	CheckpointsDir       string
	EventLogPath         string
	LeaseSummaryPath     string
	DispatchSummaryPath  string
	CheckpointSummaryPath string
	TaskPoolPath         string
	SessionRegistryPath  string
	RuntimePath          string
	ThreadStatePath      string
}

type CommandProfile struct {
	Standard   string `json:"standard"`
	LocalCompat string `json:"localCompat"`
}

type DispatchProfile struct {
	WorkspaceRoot string         `json:"workspaceRoot"`
	WorktreePath  string         `json:"worktreePath"`
	BranchName    string         `json:"branchName"`
	BaseRef       string         `json:"baseRef"`
	DiffBase      string         `json:"diffBase"`
	CommandProfile CommandProfile `json:"commandProfile"`
}

type Task struct {
	TaskID                    string          `json:"taskId"`
	ThreadKey                 string          `json:"threadKey"`
	Kind                      string          `json:"kind"`
	RoleHint                  string          `json:"roleHint"`
	WorkerMode                string          `json:"workerMode"`
	Status                    string          `json:"status"`
	PlanEpoch                 int             `json:"planEpoch"`
	WorktreePath              string          `json:"worktreePath"`
	DiffBase                  string          `json:"diffBase"`
	OwnedPaths                []string        `json:"ownedPaths"`
	ResumeStrategy            string          `json:"resumeStrategy"`
	PreferredResumeSessionID  string          `json:"preferredResumeSessionId"`
	CandidateResumeSessionIDs []string        `json:"candidateResumeSessionIds"`
	CheckpointRequired        bool            `json:"checkpointRequired"`
	CheckpointReason          string          `json:"checkpointReason"`
	ExecutionModel            string          `json:"executionModel"`
	Dispatch                  DispatchProfile `json:"dispatch"`
}

type TaskPool struct {
	Tasks []Task `json:"tasks"`
}

type ActiveBinding struct {
	TaskID    string `json:"taskId"`
	SessionID string `json:"sessionId"`
	NodeID    string `json:"nodeId"`
}

type SessionRegistry struct {
	OrchestrationSessionID string          `json:"orchestrationSessionId"`
	ActiveBindings         []ActiveBinding `json:"activeBindings"`
}

func Resolve(root string) (Paths, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Paths{}, err
	}
	harnessDir := filepath.Join(absRoot, ".harness")
	stateDir := filepath.Join(harnessDir, "state")
	eventsDir := filepath.Join(harnessDir, "events")
	checkpointsDir := filepath.Join(harnessDir, "checkpoints")
	for _, dir := range []string{harnessDir, stateDir, eventsDir, checkpointsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return Paths{}, err
		}
	}
	return Paths{
		Root:                  absRoot,
		HarnessDir:            harnessDir,
		StateDir:              stateDir,
		EventsDir:             eventsDir,
		CheckpointsDir:        checkpointsDir,
		EventLogPath:          filepath.Join(eventsDir, "a2a.jsonl"),
		LeaseSummaryPath:      filepath.Join(stateDir, "lease-summary.json"),
		DispatchSummaryPath:   filepath.Join(stateDir, "dispatch-summary.json"),
		CheckpointSummaryPath: filepath.Join(stateDir, "checkpoint-summary.json"),
		TaskPoolPath:          filepath.Join(harnessDir, "task-pool.json"),
		SessionRegistryPath:   filepath.Join(harnessDir, "session-registry.json"),
		RuntimePath:           filepath.Join(stateDir, "runtime.json"),
		ThreadStatePath:       filepath.Join(stateDir, "thread-state.json"),
	}, nil
}

func LoadTask(root, taskID string) (Task, error) {
	paths, err := Resolve(root)
	if err != nil {
		return Task{}, err
	}
	var pool TaskPool
	if err := loadJSON(paths.TaskPoolPath, &pool); err != nil {
		return Task{}, err
	}
	for _, task := range pool.Tasks {
		if task.TaskID == taskID {
			return task, nil
		}
	}
	return Task{}, errors.New("task not found")
}

func LoadSessionRegistry(root string) (SessionRegistry, error) {
	paths, err := Resolve(root)
	if err != nil {
		return SessionRegistry{}, err
	}
	var registry SessionRegistry
	if err := loadJSON(paths.SessionRegistryPath, &registry); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SessionRegistry{}, nil
		}
		return SessionRegistry{}, err
	}
	return registry, nil
}

func LoadLatestPlanEpoch(root string, task Task) (int, error) {
	if task.ThreadKey == "" {
		return task.PlanEpoch, nil
	}
	paths, err := Resolve(root)
	if err != nil {
		return 0, err
	}
	var payload struct {
		Threads map[string]struct {
			LatestValidPlanEpoch int `json:"latestValidPlanEpoch"`
			CurrentPlanEpoch     int `json:"currentPlanEpoch"`
		} `json:"threads"`
	}
	if err := loadJSON(paths.ThreadStatePath, &payload); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return task.PlanEpoch, nil
		}
		return 0, err
	}
	thread := payload.Threads[task.ThreadKey]
	if thread.LatestValidPlanEpoch > 0 {
		return thread.LatestValidPlanEpoch, nil
	}
	if thread.CurrentPlanEpoch > 0 {
		return thread.CurrentPlanEpoch, nil
	}
	return task.PlanEpoch, nil
}

func LoadCheckpointFresh(root, taskID string) (bool, error) {
	paths, err := Resolve(root)
	if err != nil {
		return false, err
	}
	var payload struct {
		Tasks map[string]struct {
			LatestCheckpoint struct {
				Status string `json:"status"`
			} `json:"latestCheckpoint"`
		} `json:"tasks"`
	}
	if err := loadJSON(paths.CheckpointSummaryPath, &payload); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	checkpoint := payload.Tasks[taskID]
	switch checkpoint.LatestCheckpoint.Status {
	case "checkpointed", "ready", "succeeded":
		return true, nil
	default:
		return false, nil
	}
}

func CountDispatchAttempts(root, taskID string) (int, error) {
	paths, err := Resolve(root)
	if err != nil {
		return 0, err
	}
	var payload struct {
		TaskIndex map[string][]string `json:"taskIndex"`
	}
	if err := loadJSON(paths.DispatchSummaryPath, &payload); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	return len(payload.TaskIndex[taskID]), nil
}

func TaskCWD(paths Paths, task Task) string {
	if task.Dispatch.WorktreePath != "" {
		return joinRoot(paths.Root, task.Dispatch.WorktreePath)
	}
	if task.WorktreePath != "" {
		return joinRoot(paths.Root, task.WorktreePath)
	}
	return paths.Root
}

func DispatchCommand(task Task) string {
	if task.Dispatch.CommandProfile.Standard != "" {
		return task.Dispatch.CommandProfile.Standard
	}
	return task.Dispatch.CommandProfile.LocalCompat
}

func joinRoot(root, path string) string {
	if path == "" {
		return root
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func loadJSON(path string, target any) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, target)
}
