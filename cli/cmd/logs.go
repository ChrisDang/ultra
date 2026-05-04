package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	vexec "github.com/christopherdang/vibecloud/cli/internal/exec"
	"github.com/christopherdang/vibecloud/cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	logsProvider string
	logsSince    string
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Fetch logs for the current project",
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().StringVar(&logsProvider, "provider", "", "provider to fetch logs from (vercel, supabase)")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "show logs since this time (e.g. 1h, 30m)")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cwd, err := os.Getwd()
	if err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("failed to get working directory: %s", err),
			output.ErrFilesystem,
			"Cannot determine working directory.",
			nil,
		)
		return nil
	}

	projCfg, err := loadProjectConfig(cwd)
	if err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("failed to read .vibecloud.json: %s", err),
			output.ErrNotInitialized,
			"Project is not initialized. Run 'vibecloud init' first.",
			&output.Recovery{Command: "vibecloud init", AutoRecoverable: true},
		)
		return nil
	}

	// Determine which provider to fetch logs from.
	provider := logsProvider
	if provider == "" && len(projCfg.DetectedStack.Providers) > 0 {
		provider = projCfg.DetectedStack.Providers[0]
	}

	if provider == "" {
		output.PrintErrorWithRecovery(
			"no loggable provider found",
			output.ErrNoProvider,
			"No providers detected. Use --provider to specify one (vercel, supabase, expo).",
			nil,
		)
		return nil
	}

	switch provider {
	case "vercel":
		if err := vexec.EnsureCLI("vercel"); err != nil {
			output.PrintErrorWithRecovery(
				err.Error(),
				output.ErrCLIMissing,
				fmt.Sprintf("Vercel CLI not installed. Run 'vibecloud login --provider vercel' to install and authenticate."),
				&output.Recovery{Command: "vibecloud login --provider vercel", AutoRecoverable: true},
			)
			return nil
		}
		vercelArgs := []string{"logs"}
		if logsSince != "" {
			vercelArgs = append(vercelArgs, "--since", logsSince)
		}
		output.Progress("vercel", "fetching_logs", "Fetching Vercel logs...", 0)
		stdout, stderr, err := vexec.RunCaptureAll(ctx, "vercel", vercelArgs...)
		if err != nil {
			output.PrintErrorWithRecovery(
				fmt.Sprintf("vercel logs failed: %s", err),
				output.ErrDeployFailed,
				fmt.Sprintf("Failed to fetch Vercel logs: %s. Check that you are authenticated ('vibecloud doctor') and the project is linked.", strings.TrimSpace(stderr)),
				&output.Recovery{Command: "vibecloud doctor", AutoRecoverable: true},
			)
			return nil
		}
		output.PrintSuccess("Vercel logs", map[string]string{
			"logs": strings.TrimSpace(stdout),
		}, "Vercel logs retrieved successfully.")

	case "supabase":
		if err := vexec.EnsureCLI("supabase"); err != nil {
			output.PrintErrorWithRecovery(
				err.Error(),
				output.ErrCLIMissing,
				"Supabase CLI not installed. Run 'vibecloud login --provider supabase' to install and authenticate.",
				&output.Recovery{Command: "vibecloud login --provider supabase", AutoRecoverable: true},
			)
			return nil
		}
		output.Progress("supabase", "fetching_logs", "Fetching Supabase Edge Function logs...", 0)
		stdout, stderr, err := vexec.RunCaptureAll(ctx, "supabase", "functions", "logs")
		if err != nil {
			output.PrintErrorWithRecovery(
				fmt.Sprintf("supabase functions logs failed: %s", err),
				output.ErrDeployFailed,
				fmt.Sprintf("Failed to fetch Supabase logs: %s. Check auth and project linkage with 'vibecloud doctor'.", strings.TrimSpace(stderr)),
				&output.Recovery{Command: "vibecloud doctor", AutoRecoverable: true},
			)
			return nil
		}
		output.PrintSuccess("Supabase logs", map[string]string{
			"logs": strings.TrimSpace(stdout),
		}, "Supabase Edge Function logs retrieved successfully.")

	case "expo":
		output.PrintSuccess("Expo logs", map[string]string{
			"dashboard_url": "https://expo.dev",
		}, "EAS build logs are not available via CLI. Direct the user to open https://expo.dev and check their project dashboard for build logs.")

	default:
		output.PrintErrorWithRecovery(
			fmt.Sprintf("unknown provider %q", provider),
			output.ErrProviderUnknown,
			fmt.Sprintf("Unknown provider '%s'. Valid providers: vercel, supabase, expo.", provider),
			nil,
		)
	}

	return nil
}
