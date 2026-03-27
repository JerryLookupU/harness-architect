package verify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type ScorecardItem struct {
	ID        string `json:"id"`
	Title     string `json:"title,omitempty"`
	Score     int    `json:"score,omitempty"`
	Threshold int    `json:"threshold,omitempty"`
	Status    string `json:"status,omitempty"`
	Summary   string `json:"summary,omitempty"`
}

type Assessment struct {
	SchemaVersion         string           `json:"schemaVersion,omitempty"`
	TaskID                string           `json:"taskId,omitempty"`
	DispatchID            string           `json:"dispatchId,omitempty"`
	AcceptedPacketID      string           `json:"acceptedPacketId,omitempty"`
	AcceptedPacketPath    string           `json:"acceptedPacketPath,omitempty"`
	ContractID            string           `json:"contractId,omitempty"`
	TaskContractPath      string           `json:"taskContractPath,omitempty"`
	OverallStatus         string           `json:"overallStatus,omitempty"`
	OverallSummary        string           `json:"overallSummary,omitempty"`
	RecommendedNextAction string           `json:"recommendedNextAction,omitempty"`
	Scorecard             []ScorecardItem  `json:"scorecard,omitempty"`
	EvidenceLedger        []map[string]any `json:"evidenceLedger,omitempty"`
	Findings              []map[string]any `json:"findings,omitempty"`
	ReviewChecklist       []map[string]any `json:"reviewChecklist,omitempty"`
}

func LoadAssessment(path string) (Assessment, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return Assessment{}, err
	}
	var envelope struct {
		Assessment
		Scorecard json.RawMessage `json:"scorecard,omitempty"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return Assessment{}, err
	}
	assessment := envelope.Assessment
	scorecard, err := decodeScorecard(envelope.Scorecard)
	if err != nil {
		return Assessment{}, err
	}
	assessment.Scorecard = scorecard
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err == nil {
		if strings.TrimSpace(assessment.OverallStatus) == "" {
			assessment.OverallStatus = coalesceString(raw["overallStatus"], raw["status"])
		}
		if strings.TrimSpace(assessment.OverallSummary) == "" {
			assessment.OverallSummary = coalesceString(raw["overallSummary"], raw["summary"])
		}
		if strings.TrimSpace(assessment.SchemaVersion) == "" {
			assessment.SchemaVersion = coalesceString(raw["schemaVersion"])
		}
		if strings.TrimSpace(assessment.TaskID) == "" {
			assessment.TaskID = coalesceString(raw["taskId"])
		}
		if strings.TrimSpace(assessment.DispatchID) == "" {
			assessment.DispatchID = coalesceString(raw["dispatchId"])
		}
		if strings.TrimSpace(assessment.AcceptedPacketID) == "" {
			assessment.AcceptedPacketID = coalesceString(raw["acceptedPacketId"], raw["packetId"])
		}
		if strings.TrimSpace(assessment.AcceptedPacketPath) == "" {
			assessment.AcceptedPacketPath = coalesceString(raw["acceptedPacketPath"])
		}
		if strings.TrimSpace(assessment.ContractID) == "" {
			assessment.ContractID = coalesceString(raw["contractId"])
		}
		if strings.TrimSpace(assessment.TaskContractPath) == "" {
			assessment.TaskContractPath = coalesceString(raw["taskContractPath"])
		}
		if len(assessment.EvidenceLedger) == 0 {
			assessment.EvidenceLedger = synthesizeEvidenceLedger(raw, path)
		}
	}
	assessment.Scorecard = normalizeScorecard(assessment.Scorecard, assessment.OverallStatus, assessment.OverallSummary)
	if strings.TrimSpace(assessment.RecommendedNextAction) == "" {
		assessment.RecommendedNextAction = deriveRecommendedNextAction(assessment)
	}
	return assessment, nil
}

func decodeScorecard(payload json.RawMessage) ([]ScorecardItem, error) {
	if len(strings.TrimSpace(string(payload))) == 0 || strings.EqualFold(strings.TrimSpace(string(payload)), "null") {
		return nil, nil
	}
	var items []ScorecardItem
	if err := json.Unmarshal(payload, &items); err == nil {
		return items, nil
	}
	var container struct {
		Requirements []scorecardRequirement `json:"requirements"`
		Items        []scorecardRequirement `json:"items"`
	}
	if err := json.Unmarshal(payload, &container); err != nil {
		return nil, err
	}
	requirements := container.Requirements
	if len(requirements) == 0 {
		requirements = container.Items
	}
	out := make([]ScorecardItem, 0, len(requirements))
	for _, requirement := range requirements {
		out = append(out, requirement.toScorecardItem())
	}
	return out, nil
}

type scorecardRequirement struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Title     string `json:"title,omitempty"`
	Status    string `json:"status,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Evidence  string `json:"evidence,omitempty"`
	Note      string `json:"note,omitempty"`
	Result    string `json:"result,omitempty"`
	Score     int    `json:"score,omitempty"`
	Threshold int    `json:"threshold,omitempty"`
}

func (item scorecardRequirement) toScorecardItem() ScorecardItem {
	title := firstNonEmpty(item.Title, item.Name, item.ID)
	summary := firstNonEmpty(item.Summary, item.Evidence, item.Note, item.Result)
	score := item.Score
	threshold := item.Threshold
	if threshold == 0 {
		threshold = 3
	}
	if score == 0 {
		switch normalizeScorecardStatus(item.Status) {
		case "pass":
			score = threshold
		case "blocked", "fail":
			score = 1
		}
	}
	return ScorecardItem{
		ID:        firstNonEmpty(item.ID, title),
		Title:     title,
		Score:     score,
		Threshold: threshold,
		Status:    item.Status,
		Summary:   summary,
	}
}

func AssessmentPath(root, taskID, dispatchID string) string {
	return filepath.Join(root, ".harness", "artifacts", taskID, dispatchID, "verify.json")
}

func DefaultScorecard(status, summary string) []ScorecardItem {
	resultStatus := normalizeScorecardStatus(status)
	score := 1
	if resultStatus == "pass" {
		score = 3
	}
	return []ScorecardItem{
		{
			ID:        "scopeCompletion",
			Title:     "Scope Completion",
			Score:     score,
			Threshold: 3,
			Status:    resultStatus,
			Summary:   summary,
		},
		{
			ID:        "behaviorCorrectness",
			Title:     "Behavior Correctness",
			Score:     score,
			Threshold: 3,
			Status:    resultStatus,
			Summary:   summary,
		},
		{
			ID:        "packetAlignment",
			Title:     "Packet Alignment",
			Score:     score,
			Threshold: 3,
			Status:    resultStatus,
			Summary:   summary,
		},
		{
			ID:        "evidenceQuality",
			Title:     "Evidence Quality",
			Score:     score,
			Threshold: 3,
			Status:    resultStatus,
			Summary:   summary,
		},
		{
			ID:        "reviewReadiness",
			Title:     "Review Readiness",
			Score:     score,
			Threshold: 3,
			Status:    resultStatus,
			Summary:   summary,
		},
	}
}

func normalizeScorecard(items []ScorecardItem, status, summary string) []ScorecardItem {
	defaults := DefaultScorecard(status, summary)
	merged := make(map[string]ScorecardItem, len(defaults))
	order := make([]string, 0, len(defaults))
	for _, item := range defaults {
		merged[item.ID] = item
		order = append(order, item.ID)
	}
	for _, item := range items {
		id := normalizeScorecardID(item.ID)
		if id == "" {
			continue
		}
		base, ok := merged[id]
		if !ok {
			base = ScorecardItem{ID: id}
			order = append(order, id)
		}
		if strings.TrimSpace(item.Title) != "" {
			base.Title = item.Title
		}
		if item.Score > 0 {
			base.Score = item.Score
		}
		if item.Threshold > 0 {
			base.Threshold = item.Threshold
		}
		if strings.TrimSpace(item.Status) != "" {
			base.Status = normalizeScorecardStatus(item.Status)
		}
		if strings.TrimSpace(item.Summary) != "" {
			base.Summary = item.Summary
		}
		merged[id] = base
	}
	out := make([]ScorecardItem, 0, len(order))
	seen := map[string]struct{}{}
	for _, id := range order {
		if _, ok := seen[id]; ok {
			continue
		}
		item := merged[id]
		if item.Threshold == 0 {
			item.Threshold = 3
		}
		if strings.TrimSpace(item.Status) == "" {
			item.Status = normalizeScorecardStatus(status)
		}
		if strings.TrimSpace(item.Title) == "" {
			item.Title = scorecardTitle(id)
		}
		if strings.TrimSpace(item.Summary) == "" {
			item.Summary = summary
		}
		out = append(out, item)
		seen[id] = struct{}{}
	}
	return out
}

func normalizeScorecardID(id string) string {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "scopecompletion", "scope_completion", "completeness":
		return "scopeCompletion"
	case "behaviorcorrectness", "behavior_correctness", "correctness":
		return "behaviorCorrectness"
	case "packetalignment", "packet_alignment", "coherence":
		return "packetAlignment"
	case "evidencequality", "evidence_quality":
		return "evidenceQuality"
	case "reviewreadiness", "review_readiness":
		return "reviewReadiness"
	default:
		return strings.TrimSpace(id)
	}
}

func scorecardTitle(id string) string {
	switch id {
	case "scopeCompletion":
		return "Scope Completion"
	case "behaviorCorrectness":
		return "Behavior Correctness"
	case "packetAlignment":
		return "Packet Alignment"
	case "evidenceQuality":
		return "Evidence Quality"
	case "reviewReadiness":
		return "Review Readiness"
	default:
		return id
	}
}

func synthesizeEvidenceLedger(raw map[string]any, path string) []map[string]any {
	ledger := make([]map[string]any, 0, 5)
	appendEntry := func(entry map[string]any) {
		if len(entry) == 0 {
			return
		}
		ledger = append(ledger, entry)
	}
	appendEntry(map[string]any{
		"kind":    "verification-result",
		"summary": "verify artifact loaded for runtime assessment",
		"path":    path,
	})
	if refs := uniqueJSONStringSlice(raw["evidenceRefs"]); len(refs) > 0 {
		appendEntry(map[string]any{
			"kind":      "evidence-refs",
			"summary":   "verify artifact declared explicit evidence references",
			"artifacts": refs,
		})
	}
	if commands, ok := raw["commands"].([]any); ok && len(commands) > 0 {
		appendEntry(map[string]any{
			"kind":    "command-results",
			"summary": "verify artifact recorded command outputs",
			"count":   len(commands),
		})
	}
	if results, ok := raw["results"].([]any); ok && len(results) > 0 {
		appendEntry(map[string]any{
			"kind":    "rule-results",
			"summary": "verify artifact recorded rule-level verification results",
			"count":   len(results),
		})
	}
	if reviewEvidence, ok := raw["reviewEvidence"].([]any); ok && len(reviewEvidence) > 0 {
		appendEntry(map[string]any{
			"kind":    "review-evidence",
			"summary": "verify artifact embedded review evidence",
			"count":   len(reviewEvidence),
		})
	}
	return ledger
}

func deriveRecommendedNextAction(assessment Assessment) string {
	if strings.EqualFold(strings.TrimSpace(assessment.OverallStatus), "blocked") {
		return "unblock"
	}
	failures := failedScorecardDimensions(assessment.Scorecard)
	if len(failures) == 0 && len(blockingFindingsFromAssessment(assessment)) == 0 {
		return "archive"
	}
	for _, failure := range failures {
		if failure == "reviewReadiness" {
			return "review"
		}
	}
	for _, finding := range blockingFindingsFromAssessment(assessment) {
		if strings.Contains(strings.ToLower(finding), "review") {
			return "review"
		}
	}
	return "repair"
}

func failedScorecardDimensions(scorecard []ScorecardItem) []string {
	failed := make([]string, 0)
	for _, item := range scorecard {
		switch strings.ToLower(strings.TrimSpace(item.Status)) {
		case "pass", "passed", "verified", "":
			if item.Threshold > 0 && item.Score > 0 && item.Score < item.Threshold {
				failed = append(failed, item.ID)
			}
		case "blocked", "fail", "failed":
			failed = append(failed, item.ID)
		default:
			if item.Threshold > 0 && item.Score > 0 && item.Score < item.Threshold {
				failed = append(failed, item.ID)
			}
		}
	}
	return failed
}

func blockingFindingsFromAssessment(assessment Assessment) []string {
	blocking := make([]string, 0)
	for _, finding := range assessment.Findings {
		if !isBlockingFinding(finding) {
			continue
		}
		blocking = append(blocking, findingSummary(finding))
	}
	return blocking
}

func normalizeScorecardStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "passed", "pass", "succeeded", "verified", "already_satisfied", "noop_verified":
		return "pass"
	case "blocked":
		return "blocked"
	default:
		return "fail"
	}
}

func coalesceString(values ...any) string {
	for _, value := range values {
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return typed
			}
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func uniqueJSONStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text := coalesceString(item)
		if text == "" {
			continue
		}
		if _, exists := seen[text]; exists {
			continue
		}
		seen[text] = struct{}{}
		out = append(out, text)
	}
	return out
}
