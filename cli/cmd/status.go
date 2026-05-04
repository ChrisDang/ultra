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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show project status from each provider",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	projCfg, err := ensureInitialized(cmd, cwd)
	if err != nil {
		return nil
	}

	results := map[string]string{}

	for _, p := range projCfg.DetectedStack.Providers {
		switch p {
		case "vercel":
			if err := vexec.EnsureCLI("vercel"); err != nil {
				results["vercel"] = fmt.Sprintf("CLI not available: %s", err)
				continue
			}
			out, _, err := vexec.RunCaptureAll(ctx, "vercel", "ls")
			if err != nil {
				results["vercel"] = fmt.Sprintf("error: %s", err)
			} else {
				results["vercel"] = strings.TrimSpace(out)
			}

		case "supabase":
			if err := vexec.EnsureCLI("supabase"); err != nil {
				results["supabase"] = fmt.Sprintf("CLI not available: %s", err)
				continue
			}
			out, _, err := vexec.RunCaptureAll(ctx, "supabase", "projects", "list")
			if err != nil {
				results["supabase"] = fmt.Sprintf("error: %s", err)
			} else {
				results["supabase"] = strings.TrimSpace(out)
			}

		case "expo":
			if err := vexec.EnsureCLI("eas"); err != nil {
				results["expo"] = fmt.Sprintf("CLI not available: %s", err)
				continue
			}
			out, _, err := vexec.RunCaptureAll(ctx, "eas", "build:list")
			if err != nil {
				results["expo"] = fmt.Sprintf("error: %s", err)
			} else {
				results["expo"] = strings.TrimSpace(out)
			}
		}
	}

	// Build specific instructions.
	var errProviders []string
	for p, result := range results {
		if strings.HasPrefix(result, "error:") || strings.HasPrefix(result, "CLI not available:") {
			errProviders = append(errProviders, p)
		}
	}

	var instructions string
	if len(errProviders) == 0 {
		instructions = fmt.Sprintf("Status retrieved for %d provider(s). No issues detected.", len(results))
	} else {
		instructions = fmt.Sprintf(
			"Status retrieved but %s had errors. Run 'vibecloud doctor' to diagnose.",
			strings.Join(errProviders, ", "),
		)
	}

	output.PrintSuccess("Project status", results, instructions)
	return nil
}
