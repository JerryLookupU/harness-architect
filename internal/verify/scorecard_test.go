package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAssessmentAcceptsScorecardRequirementsObject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "verify.json")
	payload := `{
  "overallStatus": "pass",
  "overallSummary": "verification passed",
  "scorecard": {
    "requirements": [
      {"id": "R1", "name": "Requirement One", "status": "pass", "evidence": "evidence line"},
      {"id": "R2", "name": "Requirement Two", "status": "blocked", "evidence": "missing artifact"}
    ]
  }
}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write verify.json: %v", err)
	}

	assessment, err := LoadAssessment(path)
	if err != nil {
		t.Fatalf("LoadAssessment returned error: %v", err)
	}
	if assessment.OverallStatus != "pass" {
		t.Fatalf("overallStatus = %q, want pass", assessment.OverallStatus)
	}
	if len(assessment.Scorecard) < 2 {
		t.Fatalf("scorecard length = %d, want at least 2", len(assessment.Scorecard))
	}
	foundPass := false
	foundBlocked := false
	for _, item := range assessment.Scorecard {
		switch item.ID {
		case "R1":
			foundPass = item.Title == "Requirement One" && item.Status == "pass" && item.Score == 3 && item.Summary == "evidence line"
		case "R2":
			foundBlocked = item.Title == "Requirement Two" && item.Status == "blocked" && item.Score == 1 && item.Summary == "missing artifact"
		}
	}
	if !foundPass {
		t.Fatalf("scorecard missing parsed requirement item R1: %#v", assessment.Scorecard)
	}
	if !foundBlocked {
		t.Fatalf("scorecard missing parsed requirement item R2: %#v", assessment.Scorecard)
	}
}

func TestLoadAssessmentAcceptsLegacyScorecardArray(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "verify.json")
	payload := `{
  "overallStatus": "blocked",
  "overallSummary": "needs follow-up",
  "scorecard": [
    {"id": "scopeCompletion", "title": "Scope Completion", "status": "blocked", "score": 1, "threshold": 3, "summary": "missing artifact"}
  ]
}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write verify.json: %v", err)
	}

	assessment, err := LoadAssessment(path)
	if err != nil {
		t.Fatalf("LoadAssessment returned error: %v", err)
	}
	if len(assessment.Scorecard) == 0 {
		t.Fatal("scorecard is empty")
	}
	item := assessment.Scorecard[0]
	if item.ID != "scopeCompletion" || item.Status != "blocked" || item.Score != 1 {
		t.Fatalf("legacy scorecard item = %#v", item)
	}
	if assessment.RecommendedNextAction != "unblock" {
		t.Fatalf("recommendedNextAction = %q, want unblock", assessment.RecommendedNextAction)
	}
}
