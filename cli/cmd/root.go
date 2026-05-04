package cmd

import (
	"fmt"
	"os"

	"github.com/christopherdang/vibecloud/cli/internal/output"
	versionpkg "github.com/christopherdang/vibecloud/cli/internal/version"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"

	// Global flags
	flagMachine bool // --machine: NDJSON progress events, no ANSI
	flagYes     bool // --yes: skip interactive prompts
)

func SetVersion(v, c string) {
	version = v
	commit = c
}

var rootCmd = &cobra.Command{
	Use:   "vibecloud",
	Short: "Vibe Cloud — deploy apps across Vercel, Supabase, and Expo",
	Long: `Vibe Cloud orchestrates deployments across Vercel (frontend), Supabase (backend/DB), and Expo (mobile) from a single CLI.

Workflow:
  1. vibecloud init        — initialize the project (detects stack, writes .vibecloud.json & CLAUDE.md)
  2. vibecloud doctor      — verify CLIs, auth, and project linkage
  3. vibecloud deploy      — deploy all providers in dependency order
  4. vibecloud deploy --prod — deploy to production

Run 'vibecloud init' first in any new project. All other commands require initialization.

Output format:
  All commands output JSON to stdout. Parse the "claude_instructions" field for
  actionable next steps. If "recovery.auto_recoverable" is true, run the
  "recovery.command" automatically.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		output.SetMachineMode(flagMachine)

		if info := versionpkg.CheckForUpdate(version); info != nil {
			if notice := versionpkg.UpgradeNotice(info); notice != "" {
				output.Warn("", "Update available: v"+info.Current+" → v"+info.Latest)
				output.SetUpgradeNotice(notice)
			}
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagMachine, "machine", false, "emit NDJSON progress events to stderr, suppress ANSI formatting (for agents)")
	rootCmd.PersistentFlags().BoolVar(&flagYes, "yes", false, "skip interactive prompts, assume yes (for agents and CI)")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("vibecloud v%s\n", version)
		},
	})
}
