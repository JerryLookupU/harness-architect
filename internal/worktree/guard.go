package worktree

import (
	"fmt"
	"strings"
)

func RequiresIsolatedWorktree(roleHint, kind, workerMode string) bool {
	if roleHint == "orchestrator" {
		return false
	}
	if kind == "audit" || workerMode == "audit" {
		return false
	}
	return true
}

func PathOverlap(left, right string) bool {
	left = normalize(left)
	right = normalize(right)
	if left == "" || right == "" {
		return false
	}
	if left == right {
		return true
	}
	if strings.HasSuffix(left, "/**") {
		return strings.HasPrefix(right, strings.TrimSuffix(left, "/**"))
	}
	if strings.HasSuffix(right, "/**") {
		return strings.HasPrefix(left, strings.TrimSuffix(right, "/**"))
	}
	return false
}

func GuardArtifacts(ownedPaths, artifacts []string) error {
	for _, artifact := range artifacts {
		allowed := false
		for _, owned := range ownedPaths {
			if PathOverlap(artifact, owned) || PathOverlap(owned, artifact) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("artifact outside owned paths: %s", artifact)
		}
	}
	return nil
}

func normalize(path string) string {
	return strings.TrimSuffix(path, "/")
}
