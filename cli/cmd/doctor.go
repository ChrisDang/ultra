package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/christopherdang/vibecloud/cli/internal/config"
	vexec "github.com/christopherdang/vibecloud/cli/internal/exec"
	"github.com/christopherdang/vibecloud/cli/internal/output"
	supa "github.com/christopherdang/vibecloud/cli/internal/supabase"
	versionpkg "github.com/christopherdang/vibecloud/cli/internal/version"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Preflight check — verify CLIs, auth, and project linkage before deploying",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

// ProviderHealth is the structured health check for a single provider.
type ProviderHealth struct {
	Installed     bool   `json:"installed"`
	Version       string `json:"version,omitempty"`
	Outdated      bool   `json:"outdated,omitempty"`
	Authenticated bool   `json:"authenticated"`
	Linked        bool   `json:"linked"`
	Issue         string `json:"issue,omitempty"`
}

func runDoctor(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cwd, err := os.Getwd()
	if err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("failed to get working directory: %s", err),
			output.ErrFilesystem,
			"Cannot determine working directory. Ensure you are running vibecloud from a valid project directory.",
			nil,
		)
		return nil
	}

	// Check for .vibecloud.json — auto-init if missing.
	projCfg, cfgErr := ensureInitialized(cmd, cwd)
	if cfgErr != nil {
		return nil
	}

	cfg, _ := config.Load()
	if cfg == nil {
		cfg = &config.Config{}
	}
	if cfg.CLIStatus == nil {
		cfg.CLIStatus = make(map[string]config.CLIInfo)
	}

	providers := map[string]*ProviderHealth{}
	var blockers []string
	var warnings []string

	for _, p := range projCfg.DetectedStack.Providers {
		health := &ProviderHealth{}
		providers[p] = health

		bin := cliForProvider[p]
		if bin == "" {
			bin = p
		}

		// 1. Installed?
		health.Installed = vexec.CheckInstalled(bin)
		if !health.Installed {
			health.Issue = fmt.Sprintf("%s CLI is not installed", bin)
			blockers = append(blockers, fmt.Sprintf("%s: CLI not installed — run 'vibecloud login --provider %s' to auto-install", p, p))
			continue
		}

		// 2. Version check — warn if outdated or untested (non-blocking).
		if vi := versionpkg.CheckProviderVersion(bin); vi != nil {
			health.Version = vi.Current
			if vi.Outdated {
				health.Outdated = true
				warnings = append(warnings, fmt.Sprintf(
					"%s CLI is outdated (v%s, minimum v%s) — run: %s",
					bin, vi.Current, vi.Minimum, vi.UpdateCommand,
				))
			}
			if vi.Untested {
				warnings = append(warnings, fmt.Sprintf(
					"%s CLI (v%s) is newer than max tested version (v%s) — may have breaking changes. Run /update-provider-clis to check.",
					bin, vi.Current, vi.MaxTested,
				))
			}
		}

		// 3. Authenticated?
		health.Authenticated = checkAuth(ctx, p, bin)
		if !health.Authenticated {
			health.Issue = "not authenticated"
			blockers = append(blockers, fmt.Sprintf("%s: not authenticated — run 'vibecloud login --provider %s'", p, p))
		}

		// 4. Linked?
		health.Linked = checkLinked(ctx, p, cwd)
		if !health.Linked {
			if health.Issue == "" {
				health.Issue = "not linked to a project"
			}
			blockers = append(blockers, fmt.Sprintf("%s: not linked — run 'vibecloud init' to link", p))
		}
	}

	// Cross-provider compatibility checks.
	var connectivity map[string]interface{}
	if contains(projCfg.DetectedStack.Providers, "vercel") && contains(projCfg.DetectedStack.Providers, "supabase") {
		// Check for Supabase IPv6 / connection pooler issue.
		if ref := getSupabaseProjectRef(cwd); ref != "" {
			connectivity = checkVercelSupabaseConnectivity(ctx, ref, &warnings)
		}
	}

	readyToDeploy := len(blockers) == 0

	data := map[string]interface{}{
		"project":         projCfg.ProjectName,
		"frameworks":      projCfg.DetectedStack.Frameworks,
		"providers":       providers,
		"blockers":        blockers,
		"ready_to_deploy": readyToDeploy,
	}
	if connectivity != nil {
		data["connectivity"] = connectivity
	}
	if len(projCfg.DetectedStack.Nudges) > 0 {
		data["nudges"] = projCfg.DetectedStack.Nudges
		for _, nudge := range projCfg.DetectedStack.Nudges {
			warnings = append(warnings, fmt.Sprintf(
				"This project has database indicators but is not using %s. Run '%s init' to adopt it, then re-run 'vibecloud init'.",
				nudge, nudge,
			))
		}
	}

	if readyToDeploy {
		if len(warnings) > 0 {
			output.PrintSuccessWithWarnings(
				"All checks passed",
				data,
				warnings,
				"All providers are installed, authenticated, and linked. Ready to deploy with 'vibecloud deploy'.",
			)
		} else {
			output.PrintSuccess(
				"All checks passed",
				data,
				"All providers are installed, authenticated, and linked. Ready to deploy with 'vibecloud deploy'.",
			)
		}
	} else {
		instructions := fmt.Sprintf(
			"Cannot deploy. %d blocker(s) found: %s",
			len(blockers),
			strings.Join(blockers, "; "),
		)

		// Suggest the most impactful recovery command.
		var recovery *output.Recovery
		for _, b := range blockers {
			if strings.Contains(b, "not authenticated") {
				// Extract provider name from blocker string (format: "provider: not authenticated ...").
				authProvider := strings.SplitN(b, ":", 2)[0]
				recovery = &output.Recovery{
					Command:         fmt.Sprintf("vibecloud login --provider %s", authProvider),
					AutoRecoverable: false,
				}
				instructions = fmt.Sprintf(
					"Authentication required — this is interactive and requires a browser. Ask the user to run: ! vibecloud login --provider %s",
					authProvider,
				)
				break
			}
			if strings.Contains(b, "not initialized") || strings.Contains(b, "not linked") {
				recovery = &output.Recovery{Command: "vibecloud init", AutoRecoverable: true}
				break
			}
		}

		if len(warnings) > 0 {
			output.PrintPartialSuccessWithWarnings("Preflight issues found", data, output.ErrAuthRequired, warnings, instructions, recovery)
		} else {
			output.PrintPartialSuccess("Preflight issues found", data, output.ErrAuthRequired, instructions, recovery)
		}
	}

	return nil
}

// checkAuth verifies the provider CLI is authenticated by running a quick
// command that requires auth.
func checkAuth(ctx context.Context, provider, bin string) bool {
	switch provider {
	case "vercel":
		_, _, err := vexec.RunCaptureAll(ctx, bin, "whoami")
		return err == nil
	case "supabase":
		_, _, err := vexec.RunCaptureAll(ctx, bin, "projects", "list")
		return err == nil
	case "expo":
		_, _, err := vexec.RunCaptureAll(ctx, bin, "whoami")
		return err == nil
	}
	return false
}

// getSupabaseProjectRef attempts to read the Supabase project ref from local files.
func getSupabaseProjectRef(cwd string) string {
	// Try reading supabase/.temp/project-ref first.
	if data, err := os.ReadFile(filepath.Join(cwd, "supabase", ".temp", "project-ref")); err == nil {
		if ref := strings.TrimSpace(string(data)); ref != "" {
			return ref
		}
	}

	// Try parsing supabase/config.toml for project_id.
	if data, err := os.ReadFile(filepath.Join(cwd, "supabase", "config.toml")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "project_id") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					return strings.Trim(strings.TrimSpace(parts[1]), "\"'")
				}
			}
		}
	}

	return ""
}

// checkVercelSupabaseConnectivity checks for IPv6/pooler connectivity issues
// between Vercel Functions and Supabase Postgres.
func checkVercelSupabaseConnectivity(ctx context.Context, ref string, warnings *[]string) map[string]interface{} {
	result := map[string]interface{}{
		"ipv4_available":   false,
		"pooler_reachable": false,
	}

	// Check project status via `supabase projects list -o json`.
	info, infoErr := supa.GetProjectInfo(ctx, ref)
	if infoErr != nil {
		*warnings = append(*warnings, fmt.Sprintf("Could not fetch Supabase project info: %s. Pooler check skipped.", infoErr))
		result["recommendation"] = "Run 'vibecloud login --provider supabase' to authenticate, then re-run doctor."
		return result
	}

	result["project_status"] = info.Status
	result["region"] = info.Region

	if info.IsPaused() {
		*warnings = append(*warnings, "Supabase project is paused (INACTIVE). Run 'supabase projects unpause --project-ref "+ref+"' to resume. The connection pooler will not be available until the project is active.")
		result["recommendation"] = "Unpause the project first, then wait 5-15 minutes for the pooler to provision."
		return result
	}

	if info.IsStartingUp() {
		*warnings = append(*warnings, "Supabase project is starting up (COMING_UP). The connection pooler may take 5-15 minutes to re-provision after unpausing.")
		result["recommendation"] = "Wait for project status to become ACTIVE_HEALTHY, then re-run 'vibecloud doctor'."
		return result
	}

	// DNS lookup for direct Postgres — check IPv4 availability.
	addrs, err := net.LookupHost("db." + ref + ".supabase.co")
	if err == nil {
		for _, addr := range addrs {
			if strings.Contains(addr, ".") && !strings.Contains(addr, ":") {
				result["ipv4_available"] = true
				break
			}
		}
	}

	if !result["ipv4_available"].(bool) {
		*warnings = append(*warnings, "Supabase direct Postgres resolves to IPv6 only — Vercel Functions cannot connect via direct Postgres. Use the connection pooler or PostgREST (HTTP API) instead.")
	}

	// Dynamic pooler discovery.
	pooler, poolerErr := supa.DiscoverPooler(ctx, info.Region)
	if poolerErr != nil || !pooler.Reachable {
		*warnings = append(*warnings, fmt.Sprintf("Supabase connection pooler is not reachable in region %s (may still be provisioning — can take 5-15 minutes). Run 'vibecloud env sync' to sync PostgREST credentials as a fallback.", info.Region))
		result["recommendation"] = "Use PostgREST (vibecloud env sync) for Vercel+Supabase until pooler is ready."
	} else {
		result["pooler_reachable"] = true
		result["pooler_host"] = pooler.Host
		result["recommendation"] = fmt.Sprintf("Pooler reachable at %s. Run 'vibecloud env sync' to configure DATABASE_URL.", pooler.Host)
	}

	return result
}

// checkLinked verifies the provider is linked to a project in the current dir.
func checkLinked(ctx context.Context, provider, cwd string) bool {
	switch provider {
	case "vercel":
		// Vercel creates a .vercel directory when linked.
		info, err := os.Stat(cwd + "/.vercel")
		return err == nil && info.IsDir()
	case "supabase":
		// Supabase creates a supabase/.temp directory or config.toml when linked.
		if _, err := os.Stat(cwd + "/supabase/config.toml"); err == nil {
			return true
		}
		if _, err := os.Stat(cwd + "/supabase/.temp"); err == nil {
			return true
		}
		return false
	case "expo":
		// EAS creates an eas.json when configured.
		_, err := os.Stat(cwd + "/eas.json")
		return err == nil
	}
	return false
}
