package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/christopherdang/vibecloud/cli/internal/api"
	"github.com/christopherdang/vibecloud/cli/internal/config"
	vexec "github.com/christopherdang/vibecloud/cli/internal/exec"
	"github.com/christopherdang/vibecloud/cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	deployProvider string
	deployProd     bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy the current project",
	RunE:  runDeploy,
}

func init() {
	deployCmd.Flags().StringVar(&deployProvider, "provider", "", "deploy only a specific provider (vercel, supabase, expo)")
	deployCmd.Flags().BoolVar(&deployProd, "prod", false, "deploy to production (Vercel)")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
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

	projCfg, err := ensureInitialized(cmd, cwd)
	if err != nil {
		return nil
	}

	// Load CLI config for auth state (used for pre-deploy check and post-deploy log)
	cfg, cfgErr := config.Load()

	providers := projCfg.DetectedStack.Providers
	if deployProvider != "" {
		if _, ok := cliForProvider[deployProvider]; !ok {
			output.PrintErrorWithRecovery(
				fmt.Sprintf("unknown provider %q", deployProvider),
				output.ErrProviderUnknown,
				fmt.Sprintf("Unknown provider '%s'. Valid providers: vercel, supabase, expo.", deployProvider),
				nil,
			)
			return nil
		}
		providers = []string{deployProvider}
	}

	// Pre-deploy limit check
	if cfgErr == nil && cfg.AccessToken != "" {
		baseURL := cfg.APIBaseURL
		if baseURL == "" {
			baseURL = config.DefaultAPIBaseURL
		}
		environment := "preview"
		if deployProd {
			environment = "production"
		}
		client := api.NewClient(baseURL, cfg.AccessToken, cfg.RefreshToken)
		allowed, used, limit, checkErr := client.CheckDeployLimit(
			projCfg.ProjectName,
			providers,
			environment,
		)
		if checkErr != nil {
			output.Warn("", fmt.Sprintf("Deploy limit check failed: %s (proceeding anyway)", checkErr))
		} else if !allowed {
			output.PrintErrorWithRecovery(
				fmt.Sprintf("Deploy limit reached: %d/%d deploys used this month", used, limit),
				output.ErrDeployFailed,
				fmt.Sprintf("You've used %d of %d free deploys this month. Run 'vibecloud auth upgrade' for unlimited deploys.", used, limit),
				&output.Recovery{Command: "vibecloud auth upgrade", AutoRecoverable: false},
			)
			return nil
		}
	} else {
		output.Warn("", "Deploy tracking disabled — run 'vibecloud auth login' to track usage and unlock premium features.")
	}

	results := map[string]string{}
	var warnings []string

	// Deploy in dependency order: supabase → vercel → expo.
	order := []string{"supabase", "vercel", "expo"}
	total := 0
	for _, p := range order {
		if contains(providers, p) {
			total++
		}
	}
	step := 0

	for _, p := range order {
		if !contains(providers, p) {
			continue
		}

		progress := float64(step) / float64(total)

		switch p {
		case "supabase":
			output.Progress("supabase", "starting", "\n— Deploying Supabase...", progress)
			if err := vexec.EnsureCLI("supabase"); err != nil {
				results["supabase"] = fmt.Sprintf("skipped: %s", err)
				warnings = append(warnings, fmt.Sprintf("supabase CLI not available: %s", err))
				continue
			}
			// Push DB migrations.
			output.Progress("supabase", "migrations", "  Running database migrations...", progress+0.1/float64(total))
			if err := vexec.RunNonInteractive(ctx, "supabase", "db", "push"); err != nil {
				output.Warn("supabase", fmt.Sprintf("supabase db push failed: %s", err))
				results["supabase"] = fmt.Sprintf("db push failed: %s", err)
			} else {
				results["supabase"] = "migrations applied"
			}
			// Deploy Edge Functions.
			fnDir := filepath.Join(cwd, "supabase", "functions")
			entries, _ := os.ReadDir(fnDir)
			if len(entries) == 0 {
				output.Progress("supabase", "functions", "  No edge functions found, skipping.", progress+0.5/float64(total))
				if results["supabase"] == "migrations applied" {
					results["supabase"] = "deployed"
				}
			} else {
				output.Progress("supabase", "functions", "  Deploying Edge Functions...", progress+0.5/float64(total))
				if err := vexec.RunNonInteractive(ctx, "supabase", "functions", "deploy"); err != nil {
					output.Warn("supabase", fmt.Sprintf("supabase functions deploy failed: %s", err))
					if results["supabase"] == "migrations applied" {
						results["supabase"] = "migrations applied, functions deploy failed"
						warnings = append(warnings, "supabase edge functions deploy failed but migrations succeeded")
					}
				} else if results["supabase"] == "migrations applied" {
					results["supabase"] = "deployed"
				}
			}

		case "vercel":
			output.Progress("vercel", "starting", "\n— Deploying to Vercel...", progress)
			if err := vexec.EnsureCLI("vercel"); err != nil {
				results["vercel"] = fmt.Sprintf("skipped: %s", err)
				warnings = append(warnings, fmt.Sprintf("vercel CLI not available: %s", err))
				continue
			}
			vercelArgs := []string{"deploy", "--yes"}
			if deployProd {
				vercelArgs = append(vercelArgs, "--prod")
			}
			output.Progress("vercel", "building", "  Building and deploying...", progress+0.3/float64(total))
			if err := vexec.RunNonInteractive(ctx, "vercel", vercelArgs...); err != nil {
				output.Warn("vercel", fmt.Sprintf("vercel deploy failed: %s", err))
				results["vercel"] = fmt.Sprintf("failed: %s", err)
			} else {
				results["vercel"] = "deployed"
			}

		case "expo":
			output.Progress("expo", "starting", "\n— Building with Expo (EAS)...", progress)
			if err := vexec.EnsureCLI("eas"); err != nil {
				results["expo"] = fmt.Sprintf("skipped: %s", err)
				warnings = append(warnings, fmt.Sprintf("eas CLI not available: %s", err))
				continue
			}
			output.Progress("expo", "building", "  Submitting EAS build...", progress+0.3/float64(total))
			if err := vexec.RunNonInteractive(ctx, "eas", "build", "--platform", "all", "--non-interactive"); err != nil {
				output.Warn("expo", fmt.Sprintf("eas build failed: %s", err))
				results["expo"] = fmt.Sprintf("failed: %s", err)
			} else {
				results["expo"] = "build submitted"
			}
		}

		step++
	}

	// Determine overall success.
	allOk := true
	for _, v := range results {
		if v != "deployed" && v != "build submitted" && v != "migrations applied" {
			allOk = false
			break
		}
	}

	// Log deploy (fire-and-forget)
	if cfgErr == nil && cfg.AccessToken != "" {
		baseURL := cfg.APIBaseURL
		if baseURL == "" {
			baseURL = config.DefaultAPIBaseURL
		}
		environment := "preview"
		if deployProd {
			environment = "production"
		}
		status := "completed"
		if !allOk {
			status = "failed"
		}
		logClient := api.NewClient(baseURL, cfg.AccessToken, cfg.RefreshToken)
		go func() {
			_ = logClient.LogDeploy(projCfg.ProjectName, providers, environment, status)
		}()
	}

	instructions := output.BuildDeployInstructions(results)

	output.Progress("", "complete", "Deploy finished.", 1.0)

	if allOk {
		if len(warnings) > 0 {
			output.PrintSuccessWithWarnings("Deployment complete", results, warnings, instructions)
		} else {
			output.PrintSuccess("Deployment complete", results, instructions)
		}
	} else {
		output.PrintPartialSuccessWithWarnings(
			"Deployment finished with issues",
			results,
			output.ErrPartialDeploy,
			warnings,
			instructions,
			&output.Recovery{Command: "vibecloud doctor", AutoRecoverable: true},
		)
	}

	return nil
}

func loadProjectConfig(dir string) (*ProjectConfig, error) {
	data, err := os.ReadFile(filepath.Join(dir, ".vibecloud.json"))
	if err != nil {
		return nil, err
	}
	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func contains(slice []string, val string) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
