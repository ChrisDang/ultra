package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/christopherdang/vibecloud/cli/internal/config"
	vexec "github.com/christopherdang/vibecloud/cli/internal/exec"
	"github.com/christopherdang/vibecloud/cli/internal/output"
	"github.com/spf13/cobra"
)

var loginProvider string

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with provider CLIs (vercel, supabase, expo)",
	RunE:  runLogin,
}

func init() {
	loginCmd.Flags().StringVar(&loginProvider, "provider", "", "login to a specific provider (vercel, supabase, expo)")
	rootCmd.AddCommand(loginCmd)
}

// cliForProvider maps provider names to their CLI binary names.
var cliForProvider = map[string]string{
	"vercel":   "vercel",
	"supabase": "supabase",
	"expo":     "eas",
}

func runLogin(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	providers := []string{"vercel", "supabase", "expo"}
	if loginProvider != "" {
		if _, ok := cliForProvider[loginProvider]; !ok {
			output.PrintErrorWithRecovery(
				fmt.Sprintf("unknown provider %q", loginProvider),
				output.ErrProviderUnknown,
				fmt.Sprintf("Unknown provider '%s'. Valid providers: vercel, supabase, expo.", loginProvider),
				nil,
			)
			return nil
		}
		providers = []string{loginProvider}
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	if cfg.CLIStatus == nil {
		cfg.CLIStatus = make(map[string]config.CLIInfo)
	}

	results := map[string]string{}

	for _, p := range providers {
		bin := cliForProvider[p]

		if err := vexec.EnsureCLI(bin); err != nil {
			output.Warn(p, fmt.Sprintf("Skipping %s: %s", p, err))
			results[p] = fmt.Sprintf("not installed: %s", err)
			cfg.CLIStatus[p] = config.CLIInfo{Installed: false, LoggedIn: false}
			continue
		}

		cfg.CLIStatus[p] = config.CLIInfo{Installed: true, LoggedIn: false}

		output.Progress(p, "login", fmt.Sprintf("\n— Logging in to %s...", p), 0)
		var loginErr error
		if flagYes {
			loginErr = vexec.RunNonInteractive(ctx, bin, "login")
		} else {
			loginErr = vexec.Run(ctx, bin, "login")
		}
		if loginErr != nil {
			output.Warn(p, fmt.Sprintf("%s login failed: %s", p, loginErr))
			results[p] = "login failed"
			continue
		}

		results[p] = "authenticated"
		cfg.CLIStatus[p] = config.CLIInfo{Installed: true, LoggedIn: true}
	}

	_ = config.Save(cfg)

	// Build actionable instructions.
	var authenticated, failed []string
	for p, result := range results {
		if result == "authenticated" {
			authenticated = append(authenticated, p)
		} else {
			failed = append(failed, fmt.Sprintf("%s (%s)", p, result))
		}
	}

	var instructions string
	if len(failed) == 0 {
		instructions = fmt.Sprintf("All providers authenticated: %s. Run 'vibecloud doctor' to verify project linkage, then 'vibecloud deploy' to deploy.", strings.Join(authenticated, ", "))
	} else {
		instructions = fmt.Sprintf("Authenticated: %s. Failed: %s. Re-run 'vibecloud login --provider <name>' for failed providers. The user may need to complete browser-based OAuth manually.",
			strings.Join(authenticated, ", "), strings.Join(failed, ", "))
	}

	if len(failed) > 0 && len(authenticated) == 0 {
		output.PrintPartialSuccess("Login failed", results, output.ErrAuthRequired, instructions, &output.Recovery{
			Command:         "vibecloud login",
			AutoRecoverable: false,
		})
	} else {
		output.PrintSuccess("Login complete", results, instructions)
	}

	return nil
}
