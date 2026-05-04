package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	vexec "github.com/christopherdang/vibecloud/cli/internal/exec"
	"github.com/christopherdang/vibecloud/cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	envAddProvider    string
	envAddEnvironment string
	envListProvider   string
	envRmProvider     string
	envRmEnvironment  string
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment variables across providers",
}

var envAddCmd = &cobra.Command{
	Use:   "add <KEY>",
	Short: "Set an environment variable (value prompted securely)",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvAdd,
}

var envSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Supabase credentials into Vercel environment variables",
	RunE:  runEnvSync,
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List environment variables for a provider",
	RunE:  runEnvList,
}

var envRmCmd = &cobra.Command{
	Use:   "rm <KEY>",
	Short: "Remove an environment variable",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvRm,
}

func init() {
	envAddCmd.Flags().StringVar(&envAddProvider, "provider", "vercel", "provider to set the variable on (vercel, supabase, expo)")
	envAddCmd.Flags().StringVar(&envAddEnvironment, "environment", "production", "target environment (production, preview, development)")

	envListCmd.Flags().StringVar(&envListProvider, "provider", "vercel", "provider to list variables for")

	envRmCmd.Flags().StringVar(&envRmProvider, "provider", "vercel", "provider to remove the variable from")
	envRmCmd.Flags().StringVar(&envRmEnvironment, "environment", "production", "target environment (production, preview, development)")

	envCmd.AddCommand(envAddCmd)
	envCmd.AddCommand(envSyncCmd)
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envRmCmd)
	rootCmd.AddCommand(envCmd)
}

// pipeToCommand runs a CLI command with the given value piped to stdin.
// Provider output is sent to stderr so stdout remains reserved for JSON.
func pipeToCommand(ctx context.Context, value string, binary string, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = strings.NewReader(value)
	cmd.Stdout = os.Stderr // provider output goes to stderr, not stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runEnvAdd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	key := args[0]

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

	if _, err := ensureInitialized(cmd, cwd); err != nil {
		return nil
	}

	bin, ok := cliForProvider[envAddProvider]
	if !ok {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("unknown provider %q", envAddProvider),
			output.ErrProviderUnknown,
			fmt.Sprintf("Unknown provider '%s'. Valid providers: vercel, supabase, expo.", envAddProvider),
			nil,
		)
		return nil
	}

	if err := vexec.EnsureCLI(bin); err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("%s CLI not available: %s", envAddProvider, err),
			output.ErrCLIMissing,
			fmt.Sprintf("The %s CLI is required. Install it and retry.", envAddProvider),
			&output.Recovery{Command: fmt.Sprintf("vibecloud doctor"), AutoRecoverable: true},
		)
		return nil
	}

	// Prompt for the secret value on stderr so it does not appear in JSON stdout.
	fmt.Fprintf(os.Stderr, "Enter value for %s: ", key)

	var value string
	if term.IsTerminal(int(os.Stdin.Fd())) {
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // newline after hidden input
		if err != nil {
			output.PrintErrorWithRecovery(
				fmt.Sprintf("failed to read secret input: %s", err),
				output.ErrFilesystem,
				"Could not read from terminal. Try piping the value: echo 'val' | vibecloud env add KEY",
				nil,
			)
			return nil
		}
		value = string(raw)
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			value = scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			output.PrintErrorWithRecovery(
				fmt.Sprintf("failed to read value from stdin: %s", err),
				output.ErrFilesystem,
				"Could not read from stdin.",
				nil,
			)
			return nil
		}
	}

	if err := pipeToCommand(ctx, value, bin, "env", "add", key, envAddEnvironment); err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("failed to set %s on %s: %s", key, envAddProvider, err),
			output.ErrDeployFailed,
			fmt.Sprintf("Could not set environment variable '%s' on %s. Check that the project is linked and you are authenticated.", key, envAddProvider),
			&output.Recovery{Command: "vibecloud doctor", AutoRecoverable: true},
		)
		return nil
	}

	output.PrintSuccess(
		fmt.Sprintf("Environment variable %s set for %s on %s", key, envAddProvider, envAddEnvironment),
		map[string]string{
			"key":         key,
			"provider":    envAddProvider,
			"environment": envAddEnvironment,
		},
		fmt.Sprintf("Environment variable '%s' has been set on %s (%s). Run 'vibecloud env list' to verify.", key, envAddProvider, envAddEnvironment),
	)
	return nil
}

func runEnvSync(cmd *cobra.Command, args []string) error {
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

	// Verify both supabase and vercel are providers.
	providers := projCfg.DetectedStack.Providers
	if !contains(providers, "supabase") || !contains(providers, "vercel") {
		output.PrintErrorWithRecovery(
			"env sync requires both supabase and vercel providers",
			output.ErrNoProvider,
			"The 'env sync' command syncs Supabase credentials into Vercel. Both providers must be detected. Run 'vibecloud init' to re-detect your stack.",
			&output.Recovery{Command: "vibecloud init", AutoRecoverable: true},
		)
		return nil
	}

	for _, bin := range []string{"supabase", "vercel"} {
		if err := vexec.EnsureCLI(bin); err != nil {
			output.PrintErrorWithRecovery(
				fmt.Sprintf("%s CLI not available: %s", bin, err),
				output.ErrCLIMissing,
				fmt.Sprintf("The %s CLI is required for env sync.", bin),
				&output.Recovery{Command: "vibecloud doctor", AutoRecoverable: true},
			)
			return nil
		}
	}

	// Read Supabase project ref.
	refPath := filepath.Join(cwd, "supabase", ".temp", "project-ref")
	refBytes, err := os.ReadFile(refPath)
	if err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("could not read Supabase project ref: %s", err),
			output.ErrNotLinked,
			"Supabase project ref not found at supabase/.temp/project-ref. Run 'supabase link' first to link your Supabase project.",
			&output.Recovery{Command: "vibecloud login --provider supabase", AutoRecoverable: false},
		)
		return nil
	}
	ref := strings.TrimSpace(string(refBytes))

	// Get API keys from Supabase.
	stdout, _, err := vexec.RunCaptureAll(ctx, "supabase", "projects", "api-keys", "--project-ref", ref)
	if err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("failed to get Supabase API keys: %s", err),
			output.ErrDeployFailed,
			"Could not retrieve Supabase API keys. Ensure you are authenticated: run 'vibecloud login --provider supabase'.",
			&output.Recovery{Command: "vibecloud login --provider supabase", AutoRecoverable: false},
		)
		return nil
	}

	// Parse the table output: columns are name | api_key.
	anonKey, serviceRoleKey := parseSupabaseAPIKeys(stdout)
	if anonKey == "" || serviceRoleKey == "" {
		output.PrintErrorWithRecovery(
			"could not parse Supabase API keys from output",
			output.ErrDeployFailed,
			"Unexpected output from 'supabase projects api-keys'. Ensure the Supabase CLI is up to date.",
			&output.Recovery{Command: "vibecloud doctor", AutoRecoverable: true},
		)
		return nil
	}

	supabaseURL := fmt.Sprintf("https://%s.supabase.co", ref)

	// Sync each key into Vercel.
	envVars := []struct {
		key   string
		value string
	}{
		{"SUPABASE_URL", supabaseURL},
		{"SUPABASE_ANON_KEY", anonKey},
		{"SUPABASE_SERVICE_ROLE_KEY", serviceRoleKey},
	}

	results := map[string]string{}
	var failures []string

	for _, ev := range envVars {
		fmt.Fprintf(os.Stderr, "Syncing %s...\n", ev.key)
		if err := pipeToCommand(ctx, ev.value, "vercel", "env", "add", ev.key, "production"); err != nil {
			results[ev.key] = fmt.Sprintf("failed: %s", err)
			failures = append(failures, ev.key)
		} else {
			results[ev.key] = "synced"
		}
	}

	if len(failures) > 0 {
		output.PrintPartialSuccess(
			"Environment sync completed with errors",
			results,
			output.ErrPartialDeploy,
			fmt.Sprintf("Failed to sync: %s. Ensure the Vercel project is linked and you are authenticated. Run 'vibecloud doctor' to diagnose.", strings.Join(failures, ", ")),
			&output.Recovery{Command: "vibecloud doctor", AutoRecoverable: true},
		)
	} else {
		output.PrintSuccess(
			"Supabase credentials synced to Vercel",
			results,
			"All Supabase environment variables have been synced to Vercel (production). Run 'vibecloud env list' to verify, then 'vibecloud deploy --prod' to deploy.",
		)
	}

	return nil
}

// parseSupabaseAPIKeys extracts the anon and service_role keys from
// the table output of `supabase projects api-keys`.
func parseSupabaseAPIKeys(tableOutput string) (anonKey, serviceRoleKey string) {
	for _, line := range strings.Split(tableOutput, "\n") {
		// Skip header/separator lines.
		if strings.Contains(line, "---") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		key := strings.TrimSpace(parts[1])
		switch name {
		case "anon":
			anonKey = key
		case "service_role":
			serviceRoleKey = key
		}
	}
	return
}

func runEnvList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	bin, ok := cliForProvider[envListProvider]
	if !ok {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("unknown provider %q", envListProvider),
			output.ErrProviderUnknown,
			fmt.Sprintf("Unknown provider '%s'. Valid providers: vercel, supabase, expo.", envListProvider),
			nil,
		)
		return nil
	}

	if err := vexec.EnsureCLI(bin); err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("%s CLI not available: %s", envListProvider, err),
			output.ErrCLIMissing,
			fmt.Sprintf("The %s CLI is required. Install it and retry.", envListProvider),
			&output.Recovery{Command: "vibecloud doctor", AutoRecoverable: true},
		)
		return nil
	}

	stdout, _, err := vexec.RunCaptureAll(ctx, bin, "env", "ls")
	if err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("failed to list environment variables: %s", err),
			output.ErrDeployFailed,
			fmt.Sprintf("Could not list environment variables for %s. Ensure the project is linked and you are authenticated.", envListProvider),
			&output.Recovery{Command: "vibecloud doctor", AutoRecoverable: true},
		)
		return nil
	}

	output.PrintSuccess(
		fmt.Sprintf("Environment variables for %s", envListProvider),
		map[string]string{"output": strings.TrimSpace(stdout)},
		fmt.Sprintf("Showing environment variables for %s. Use 'vibecloud env add <KEY>' to add new variables.", envListProvider),
	)
	return nil
}

func runEnvRm(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	key := args[0]

	bin, ok := cliForProvider[envRmProvider]
	if !ok {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("unknown provider %q", envRmProvider),
			output.ErrProviderUnknown,
			fmt.Sprintf("Unknown provider '%s'. Valid providers: vercel, supabase, expo.", envRmProvider),
			nil,
		)
		return nil
	}

	if err := vexec.EnsureCLI(bin); err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("%s CLI not available: %s", envRmProvider, err),
			output.ErrCLIMissing,
			fmt.Sprintf("The %s CLI is required. Install it and retry.", envRmProvider),
			&output.Recovery{Command: "vibecloud doctor", AutoRecoverable: true},
		)
		return nil
	}

	if err := vexec.RunNonInteractive(ctx, bin, "env", "rm", key, envRmEnvironment, "--yes"); err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("failed to remove %s from %s: %s", key, envRmProvider, err),
			output.ErrDeployFailed,
			fmt.Sprintf("Could not remove environment variable '%s' from %s. Check that the variable exists and the project is linked.", key, envRmProvider),
			&output.Recovery{Command: "vibecloud doctor", AutoRecoverable: true},
		)
		return nil
	}

	output.PrintSuccess(
		fmt.Sprintf("Environment variable %s removed from %s (%s)", key, envRmProvider, envRmEnvironment),
		map[string]string{
			"key":         key,
			"provider":    envRmProvider,
			"environment": envRmEnvironment,
		},
		fmt.Sprintf("Environment variable '%s' has been removed from %s (%s).", key, envRmProvider, envRmEnvironment),
	)
	return nil
}
