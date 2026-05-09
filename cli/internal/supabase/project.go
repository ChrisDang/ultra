package supabase

import (
	"context"
	"encoding/json"
	"fmt"

	vexec "github.com/christopherdang/vibecloud/cli/internal/exec"
)

// ProjectInfo holds metadata about a Supabase project.
type ProjectInfo struct {
	Ref    string `json:"ref"`
	Name   string `json:"name"`
	Region string `json:"region"`
	Status string `json:"status"`
	DBHost string `json:"-"`
}

// projectJSON maps the JSON structure from `supabase projects list -o json`.
type projectJSON struct {
	Ref      string `json:"ref"`
	Name     string `json:"name"`
	Region   string `json:"region"`
	Status   string `json:"status"`
	Database struct {
		Host string `json:"host"`
	} `json:"database"`
}

// IsPaused returns true if the project is paused (INACTIVE).
func (p *ProjectInfo) IsPaused() bool {
	return p.Status == "INACTIVE"
}

// IsStartingUp returns true if the project is in the process of starting.
func (p *ProjectInfo) IsStartingUp() bool {
	return p.Status == "COMING_UP"
}

// IsHealthy returns true if the project is active and healthy.
func (p *ProjectInfo) IsHealthy() bool {
	return p.Status == "ACTIVE_HEALTHY"
}

// GetProjectInfo fetches project metadata by running
// `supabase projects list -o json` and finding the matching ref.
func GetProjectInfo(ctx context.Context, ref string) (*ProjectInfo, error) {
	stdout, _, err := vexec.RunCaptureAll(ctx, "supabase", "projects", "list", "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to list Supabase projects: %w", err)
	}
	return parseProjectList(stdout, ref)
}

// parseProjectList extracts a ProjectInfo from the JSON output of
// `supabase projects list -o json`.
func parseProjectList(jsonData string, ref string) (*ProjectInfo, error) {
	var projects []projectJSON
	if err := json.Unmarshal([]byte(jsonData), &projects); err != nil {
		return nil, fmt.Errorf("failed to parse project list JSON: %w", err)
	}

	for _, p := range projects {
		if p.Ref == ref {
			return &ProjectInfo{
				Ref:    p.Ref,
				Name:   p.Name,
				Region: p.Region,
				Status: p.Status,
				DBHost: p.Database.Host,
			}, nil
		}
	}

	return nil, fmt.Errorf("project with ref %q not found", ref)
}
