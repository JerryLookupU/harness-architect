package projectspace

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ProjectSpace struct {
	ProjectSpaceID         string   `json:"projectSpaceId"`
	WorkspaceRoot          string   `json:"workspaceRoot"`
	WorktreePolicy         string   `json:"worktreePolicy,omitempty"`
	IsolationPolicy        string   `json:"isolationPolicy,omitempty"`
	AllowedRepos           []string `json:"allowedRepos,omitempty"`
	DefaultOwnedPathScopes []string `json:"defaultOwnedPathScopes,omitempty"`
}

type Project struct {
	ProjectID     string         `json:"projectId"`
	ProjectName   string         `json:"projectName,omitempty"`
	DefaultBranch string         `json:"defaultBranch,omitempty"`
	Spaces        []ProjectSpace `json:"spaces,omitempty"`
}

type Registry struct {
	SchemaVersion string    `json:"schemaVersion"`
	Generator     string    `json:"generator"`
	GeneratedAt   string    `json:"generatedAt"`
	Projects      []Project `json:"projects"`
}

func RegistryPath(root string) string {
	return filepath.Join(root, ".harness", "state", "project-registry.json")
}

func DefaultProjectID(root string) string {
	return "project-" + stableID(root)
}

func DefaultProjectSpaceID(root string) string {
	return "space-" + stableID(root)
}

func EnsureRegistry(root string) (Registry, error) {
	path := RegistryPath(root)
	registry, err := LoadRegistry(path)
	if err == nil {
		return registry, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Registry{}, err
	}
	projectID := DefaultProjectID(root)
	spaceID := DefaultProjectSpaceID(root)
	registry = Registry{
		SchemaVersion: "kh.project-registry.v1",
		Generator:     "kh-runtime",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Projects: []Project{
			{
				ProjectID:     projectID,
				ProjectName:   filepath.Base(root),
				DefaultBranch: "main",
				Spaces: []ProjectSpace{
					{
						ProjectSpaceID:         spaceID,
						WorkspaceRoot:          root,
						WorktreePolicy:         "repo-local",
						IsolationPolicy:        "workspace-bound",
						DefaultOwnedPathScopes: []string{"."},
					},
				},
			},
		},
	}
	if err := WriteRegistry(path, registry); err != nil {
		return Registry{}, err
	}
	return registry, nil
}

func ResolveProjectSpace(root, projectID, projectSpaceID string) (string, string, error) {
	registry, err := EnsureRegistry(root)
	if err != nil {
		return "", "", err
	}
	selectedProjectID := strings.TrimSpace(projectID)
	if selectedProjectID == "" {
		selectedProjectID = DefaultProjectID(root)
	}
	selectedSpaceID := strings.TrimSpace(projectSpaceID)
	if selectedSpaceID == "" {
		selectedSpaceID = DefaultProjectSpaceID(root)
	}
	for _, project := range registry.Projects {
		if project.ProjectID != selectedProjectID {
			continue
		}
		for _, space := range project.Spaces {
			if space.ProjectSpaceID == selectedSpaceID {
				return selectedProjectID, selectedSpaceID, nil
			}
		}
	}
	return "", "", errors.New("project space not found in registry")
}

func LoadRegistry(path string) (Registry, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return Registry{}, err
	}
	var registry Registry
	if err := json.Unmarshal(payload, &registry); err != nil {
		return Registry{}, err
	}
	if registry.SchemaVersion == "" {
		registry.SchemaVersion = "kh.project-registry.v1"
	}
	return registry, nil
}

func WriteRegistry(path string, registry Registry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if registry.SchemaVersion == "" {
		registry.SchemaVersion = "kh.project-registry.v1"
	}
	if strings.TrimSpace(registry.Generator) == "" {
		registry.Generator = "kh-runtime"
	}
	registry.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	payload, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

func stableID(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(filepath.Clean(text)))
	normalized = strings.NewReplacer(
		"/", "-",
		"\\", "-",
		" ", "-",
		"_", "-",
		".", "-",
	).Replace(normalized)
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		return "default"
	}
	if len(normalized) > 48 {
		return normalized[len(normalized)-48:]
	}
	return normalized
}
