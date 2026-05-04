package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	vexec "github.com/christopherdang/vibecloud/cli/internal/exec"
	"github.com/christopherdang/vibecloud/cli/internal/output"
	"github.com/spf13/cobra"
)

var explainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Show full project state across all providers in one call",
	RunE:  runExplain,
}

func init() {
	rootCmd.AddCommand(explainCmd)
}

// ProviderState is the structured state for a single provider.
type ProviderState struct {
	Status      string      `json:"status"`
	Details     interface{} `json:"details,omitempty"`
	URL         string      `json:"url,omitempty"`
	Environment string      `json:"environment,omitempty"`
	Error       string      `json:"error,omitempty"`
}

func runExplain(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cwd, err := os.Getwd()
	if err != nil {
		output.PrintError(fmt.Sprintf("failed to get working directory: %s", err))
		return nil
	}

	projCfg, err := ensureInitialized(cmd, cwd)
	if err != nil {
		return nil
	}

	providerStates := map[string]*ProviderState{}
	var summaryParts []string

	for _, p := range projCfg.DetectedStack.Providers {
		state := &ProviderState{}
		providerStates[p] = state

		bin := cliForProvider[p]
		if bin == "" {
			bin = p
		}

		if !vexec.CheckInstalled(bin) {
			state.Status = "not_installed"
			state.Error = fmt.Sprintf("%s CLI not installed", bin)
			summaryParts = append(summaryParts, fmt.Sprintf("%s: CLI not installed", p))
			continue
		}

		switch p {
		case "vercel":
			explainVercel(ctx, state)
		case "supabase":
			explainSupabase(ctx, state, cwd)
		case "expo":
			explainExpo(ctx, state)
		}

		summaryParts = append(summaryParts, fmt.Sprintf("%s: %s", p, state.Status))
	}

	data := map[string]interface{}{
		"project":    projCfg.ProjectName,
		"frameworks": projCfg.DetectedStack.Frameworks,
		"providers":  providerStates,
	}

	instructions := fmt.Sprintf(
		"Project '%s' (%s). %s. Use 'vibecloud doctor' to check deploy-readiness or 'vibecloud deploy' to deploy.",
		projCfg.ProjectName,
		strings.Join(projCfg.DetectedStack.Frameworks, ", "),
		strings.Join(summaryParts, ". "),
	)

	output.PrintSuccess("Project state", data, instructions)
	return nil
}

func explainVercel(ctx context.Context, state *ProviderState) {
	// Get project info via vercel inspect or vercel ls.
	stdout, _, err := vexec.RunCaptureAll(ctx, "vercel", "ls", "--json")
	if err != nil {
		// Check if it's an auth issue.
		if strings.Contains(stdout, "not_authorized") || strings.Contains(stdout, "forbidden") {
			state.Status = "auth_expired"
			state.Error = "Vercel authentication expired"
			return
		}
		state.Status = "error"
		state.Error = fmt.Sprintf("vercel ls failed: %s", err)
		return
	}

	// Try to parse deployment list.
	var lsResp struct {
		Deployments []struct {
			URL   string `json:"url"`
			State string `json:"state"`
			Target string `json:"target"`
		} `json:"deployments"`
	}
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &lsResp); jsonErr == nil && len(lsResp.Deployments) > 0 {
		latest := lsResp.Deployments[0]
		state.Status = "deployed"
		state.URL = latest.URL
		state.Environment = latest.Target
		state.Details = map[string]interface{}{
			"latest_state":      latest.State,
			"total_deployments": len(lsResp.Deployments),
		}
		return
	}

	// Fallback: vercel ls succeeded but output wasn't JSON.
	state.Status = "linked"
	state.Details = map[string]string{"raw": strings.TrimSpace(stdout)}
}

func explainSupabase(ctx context.Context, state *ProviderState, cwd string) {
	// Check if supabase is linked.
	if _, err := os.Stat(cwd + "/supabase/config.toml"); err != nil {
		state.Status = "not_linked"
		state.Error = "supabase directory exists but project not linked"
		return
	}

	stdout, _, err := vexec.RunCaptureAll(ctx, "supabase", "projects", "list")
	if err != nil {
		state.Status = "error"
		state.Error = fmt.Sprintf("supabase projects list failed: %s", err)
		return
	}

	state.Status = "linked"
	state.Details = map[string]string{"projects": strings.TrimSpace(stdout)}

	// Check for pending migrations.
	migrateOut, _, migrateErr := vexec.RunCaptureAll(ctx, "supabase", "db", "diff")
	if migrateErr == nil {
		diff := strings.TrimSpace(migrateOut)
		if diff == "" {
			state.Status = "deployed"
		} else {
			state.Status = "has_pending_changes"
			state.Details = map[string]string{
				"projects":        strings.TrimSpace(stdout),
				"pending_changes": diff,
			}
		}
	}
}

func explainExpo(ctx context.Context, state *ProviderState) {
	stdout, _, err := vexec.RunCaptureAll(ctx, "eas", "build:list", "--json", "--limit", "1")
	if err != nil {
		state.Status = "error"
		state.Error = fmt.Sprintf("eas build:list failed: %s", err)
		return
	}

	var builds []struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Platform string `json:"platform"`
	}
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &builds); jsonErr == nil && len(builds) > 0 {
		state.Status = builds[0].Status
		state.Details = builds[0]
		return
	}

	state.Status = "configured"
	state.Details = map[string]string{"raw": strings.TrimSpace(stdout)}
}
