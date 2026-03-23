package rca

import "strings"

type Allocation struct {
	Taxonomy  string `json:"taxonomy"`
	OwnerRole string `json:"ownerRole"`
	Summary   string `json:"summary"`
}

func Allocate(summary string, reasonCodes []string) Allocation {
	lower := strings.ToLower(summary + " " + strings.Join(reasonCodes, " "))
	switch {
	case strings.Contains(lower, "verify"):
		return Allocation{Taxonomy: "verification_guardrail", OwnerRole: "architect/orchestrator", Summary: summary}
	case strings.Contains(lower, "route"), strings.Contains(lower, "resume"):
		return Allocation{Taxonomy: "routing_session", OwnerRole: "runtime/orchestrator", Summary: summary}
	default:
		return Allocation{Taxonomy: "underdetermined", OwnerRole: "architect/orchestrator", Summary: summary}
	}
}
