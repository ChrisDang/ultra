package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	vexec "github.com/christopherdang/vibecloud/cli/internal/exec"
	"github.com/christopherdang/vibecloud/cli/internal/output"
	supa "github.com/christopherdang/vibecloud/cli/internal/supabase"
	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage the Supabase database",
}

var dbPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push migrations to the remote Supabase database",
	RunE:  runDBPush,
}

var dbStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show database connection info and pooler status",
	RunE:  runDBStatus,
}

func init() {
	dbCmd.AddCommand(dbPushCmd)
	dbCmd.AddCommand(dbStatusCmd)
	rootCmd.AddCommand(dbCmd)
}

func runDBPush(cmd *cobra.Command, args []string) error {
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

	if _, err := ensureInitialized(cmd, cwd); err != nil {
		return nil
	}

	if err := vexec.EnsureCLI("supabase"); err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("supabase CLI not available: %s", err),
			output.ErrCLIMissing,
			"The supabase CLI is required for db push.",
			&output.Recovery{Command: "vibecloud doctor", AutoRecoverable: true},
		)
		return nil
	}

	// Check project status before pushing.
	ref := readProjectRef(cwd)
	if ref != "" {
		info, infoErr := supa.GetProjectInfo(ctx, ref)
		if infoErr == nil {
			if info.IsPaused() {
				output.PrintErrorWithRecovery(
					"Supabase project is paused",
					output.ErrDeployFailed,
					fmt.Sprintf("Cannot push migrations — project is paused. Run: supabase projects unpause --project-ref %s", ref),
					nil,
				)
				return nil
			}
			if info.IsStartingUp() {
				output.Warn("supabase", "Project is starting up. Migration push may fail if the database is not ready yet.")
			}
		}
	}

	// Run supabase db push --linked.
	fmt.Fprintln(os.Stderr, "Pushing migrations to remote database...")
	pushErr := vexec.RunNonInteractive(ctx, "supabase", "db", "push", "--linked")
	if pushErr != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("migration push failed: %s", pushErr),
			output.ErrMigrationFailed,
			"Failed to push migrations. Ensure the Supabase project is linked and the database is accessible. Run 'vibecloud doctor' to diagnose.",
			&output.Recovery{Command: "vibecloud doctor", AutoRecoverable: true},
		)
		return nil
	}

	output.PrintSuccess(
		"Migrations pushed successfully",
		map[string]string{"action": "db push"},
		"Migrations have been applied to the remote Supabase database. Run 'vibecloud deploy --prod' to deploy your application.",
	)
	return nil
}

func runDBStatus(cmd *cobra.Command, args []string) error {
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

	if _, err := ensureInitialized(cmd, cwd); err != nil {
		return nil
	}

	ref := readProjectRef(cwd)
	if ref == "" {
		output.PrintErrorWithRecovery(
			"Supabase project ref not found",
			output.ErrNotLinked,
			"Could not find supabase/.temp/project-ref. Run 'supabase link' to link your project.",
			&output.Recovery{Command: "vibecloud init", AutoRecoverable: true},
		)
		return nil
	}

	data := map[string]interface{}{
		"ref": ref,
	}

	// Get project info.
	info, infoErr := supa.GetProjectInfo(ctx, ref)
	if infoErr != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("could not fetch project info: %s", infoErr),
			output.ErrDeployFailed,
			"Failed to get Supabase project info. Ensure you are authenticated.",
			&output.Recovery{Command: "vibecloud login --provider supabase", AutoRecoverable: false},
		)
		return nil
	}

	data["name"] = info.Name
	data["region"] = info.Region
	data["status"] = info.Status
	data["db_host"] = fmt.Sprintf("db.%s.supabase.co", ref)

	var warnings []string

	if info.IsPaused() {
		warnings = append(warnings, fmt.Sprintf("Project is paused. Run: supabase projects unpause --project-ref %s", ref))
	} else if info.IsStartingUp() {
		warnings = append(warnings, "Project is starting up. Pooler may take 5-15 minutes to provision.")
	}

	// Discover pooler.
	if !info.IsPaused() {
		pooler, poolerErr := supa.DiscoverPooler(ctx, info.Region)
		if poolerErr != nil || !pooler.Reachable {
			data["pooler_reachable"] = false
			warnings = append(warnings, fmt.Sprintf("Connection pooler not reachable in region %s. May still be provisioning.", info.Region))
		} else {
			data["pooler_reachable"] = true
			data["pooler_host"] = pooler.Host
			data["pooler_transaction_port"] = pooler.TransactionPort
			data["pooler_session_port"] = pooler.SessionPort
		}
	}

	// Check API keys.
	if err := vexec.EnsureCLI("supabase"); err == nil {
		stdout, _, apiErr := vexec.RunCaptureAll(ctx, "supabase", "projects", "api-keys", "--project-ref", ref)
		if apiErr == nil {
			anonKey, serviceRoleKey := parseSupabaseAPIKeys(stdout)
			data["anon_key_available"] = anonKey != ""
			data["service_role_key_available"] = serviceRoleKey != ""
		}
	}

	instructions := fmt.Sprintf(
		"Supabase project '%s' in %s is %s.",
		info.Name, info.Region, strings.ToLower(info.Status),
	)
	if poolerHost, ok := data["pooler_host"].(string); ok {
		instructions += fmt.Sprintf(" Pooler reachable at %s. Run 'vibecloud env sync' to configure DATABASE_URL.", poolerHost)
	} else if !info.IsPaused() {
		instructions += " Pooler not yet reachable. Try again in a few minutes."
	}

	if len(warnings) > 0 {
		output.PrintSuccessWithWarnings("Database status", data, warnings, instructions)
	} else {
		output.PrintSuccess("Database status", data, instructions)
	}
	return nil
}

// readProjectRef reads the Supabase project ref from supabase/.temp/project-ref.
func readProjectRef(cwd string) string {
	data, err := os.ReadFile(filepath.Join(cwd, "supabase", ".temp", "project-ref"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
