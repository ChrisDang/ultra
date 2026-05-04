package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/christopherdang/vibecloud/cli/internal/detect"
	vexec "github.com/christopherdang/vibecloud/cli/internal/exec"
	"github.com/christopherdang/vibecloud/cli/internal/output"
	"github.com/spf13/cobra"
)

// ProjectConfig is the .vibecloud.json file written to the project root.
type ProjectConfig struct {
	ProjectName   string              `json:"project_name"`
	DetectedStack detect.DetectedStack `json:"detected_stack"`
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a Vibe Cloud project in the current directory",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
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

	stack := detect.DetectStack(cwd)
	projectName := filepath.Base(cwd)

	// Write .vibecloud.json.
	projCfg := ProjectConfig{
		ProjectName:   projectName,
		DetectedStack: stack,
	}
	data, err := json.MarshalIndent(projCfg, "", "  ")
	if err != nil {
		output.PrintError(fmt.Sprintf("failed to marshal project config: %s", err))
		return nil
	}
	if err := os.WriteFile(filepath.Join(cwd, ".vibecloud.json"), data, 0644); err != nil {
		output.PrintError(fmt.Sprintf("failed to write .vibecloud.json: %s", err))
		return nil
	}

	// Write CLAUDE.md (append if it exists, create if it doesn't).
	writeClaudeMD(cwd, projectName, stack)

	// Auto-link detected providers.
	linkResults := map[string]string{}
	var warnings []string
	for _, p := range stack.Providers {
		switch p {
		case "vercel":
			if err := vexec.EnsureCLI("vercel"); err != nil {
				output.Warn("vercel", fmt.Sprintf("%s (you can install it manually later)", err))
				linkResults["vercel"] = fmt.Sprintf("skipped: %s", err)
				warnings = append(warnings, fmt.Sprintf("vercel: %s", err))
			} else {
				output.Progress("vercel", "linking", "\n— Linking Vercel project...", 0)
				if err := vercelLink(ctx); err != nil {
					output.Warn("vercel", fmt.Sprintf("vercel link failed: %s (you can run it manually later)", err))
					linkResults["vercel"] = fmt.Sprintf("link failed: %s", err)
					warnings = append(warnings, fmt.Sprintf("vercel link failed: %s", err))
				} else {
					linkResults["vercel"] = "linked"
				}
			}
		case "supabase":
			if err := vexec.EnsureCLI("supabase"); err != nil {
				output.Warn("supabase", fmt.Sprintf("%s (you can install it manually later)", err))
				linkResults["supabase"] = fmt.Sprintf("skipped: %s", err)
				warnings = append(warnings, fmt.Sprintf("supabase: %s", err))
			} else {
				// Check if already linked.
				alreadyLinked := false
				if _, statErr := os.Stat(filepath.Join(cwd, "supabase", ".temp", "project-ref")); statErr == nil {
					alreadyLinked = true
				}

				if alreadyLinked {
					linkResults["supabase"] = "linked"
				} else {
					// Try to find or create a Supabase project.
					ref := findOrCreateSupabaseProject(ctx, projectName)
					if ref != "" {
						output.Progress("supabase", "linking", fmt.Sprintf("\n— Linking Supabase project (ref: %s)...", ref), 0)
						var linkErr error
						if flagYes {
							linkErr = vexec.RunNonInteractive(ctx, "supabase", "link", "--project-ref", ref)
						} else {
							linkErr = vexec.Run(ctx, "supabase", "link", "--project-ref", ref)
						}
						if linkErr != nil {
							output.Warn("supabase", fmt.Sprintf("supabase link failed: %s (you can run it manually later)", linkErr))
							linkResults["supabase"] = fmt.Sprintf("link failed: %s", linkErr)
							warnings = append(warnings, fmt.Sprintf("supabase link failed: %s", linkErr))
						} else {
							linkResults["supabase"] = "linked"
						}
					} else {
						// Fall back to interactive link.
						output.Progress("supabase", "linking", "\n— Linking Supabase project...", 0)
						var linkErr error
						if flagYes {
							linkErr = vexec.RunNonInteractive(ctx, "supabase", "link")
						} else {
							linkErr = vexec.Run(ctx, "supabase", "link")
						}
						if linkErr != nil {
							output.Warn("supabase", fmt.Sprintf("supabase link failed: %s (you can run it manually later)", linkErr))
							linkResults["supabase"] = fmt.Sprintf("link failed: %s", linkErr)
							warnings = append(warnings, fmt.Sprintf("supabase link failed: %s", linkErr))
						} else {
							linkResults["supabase"] = "linked"
						}
					}
				}
			}
		case "expo":
			if err := vexec.EnsureCLI("eas"); err != nil {
				output.Warn("expo", fmt.Sprintf("%s (you can install it manually later)", err))
				linkResults["expo"] = fmt.Sprintf("skipped: %s", err)
				warnings = append(warnings, fmt.Sprintf("expo: %s", err))
			} else {
				output.Progress("expo", "detected", "\n— Expo detected. Run 'eas build:configure' to set up your EAS project.", 0)
				linkResults["expo"] = "detected, needs eas build:configure"
			}
		}
	}

	resultData := map[string]interface{}{
		"project_name":   projectName,
		"detected_stack": stack,
		"link_results":   linkResults,
		"claude_md":      "CLAUDE.md written with vibecloud instructions",
	}

	// Build specific instructions.
	var nextSteps []string
	for provider, result := range linkResults {
		if !strings.Contains(result, "linked") && !strings.Contains(result, "detected") {
			nextSteps = append(nextSteps, fmt.Sprintf("%s needs attention: %s", provider, result))
		}
	}

	instructions := fmt.Sprintf(
		"Project '%s' initialized as %s with providers [%s]. ",
		projectName, strings.Join(stack.Frameworks, ", "), strings.Join(stack.Providers, ", "),
	)
	if len(nextSteps) == 0 {
		instructions += "All providers linked. Run 'vibecloud doctor' to verify auth, then 'vibecloud deploy' to deploy."
	} else {
		instructions += strings.Join(nextSteps, ". ") + ". Run 'vibecloud doctor' to see full preflight status."
	}

	if len(stack.Nudges) > 0 {
		for _, nudge := range stack.Nudges {
			instructions += fmt.Sprintf(
				" This project has database indicators but is not using %s. %s provides managed Postgres, authentication, edge functions, and real-time subscriptions. To adopt %s, run `%s init` to create the `%s/` directory, then re-run `vibecloud init`.",
				nudge, nudge, nudge, nudge, nudge,
			)
		}
	}

	if len(warnings) > 0 {
		output.PrintSuccessWithWarnings("Project initialized", resultData, warnings, instructions)
	} else {
		output.PrintSuccess("Project initialized", resultData, instructions)
	}
	return nil
}

// ensureInitialized loads the project config, auto-running `vibecloud init` if
// the project hasn't been initialized yet. Returns the config or prints an
// error and returns a non-nil error to signal the caller to bail.
func ensureInitialized(cmd *cobra.Command, cwd string) (*ProjectConfig, error) {
	projCfg, err := loadProjectConfig(cwd)
	if err == nil {
		return projCfg, nil
	}

	// Auto-initialize: run init transparently.
	output.Progress("", "auto_init", "Project not initialized — running 'vibecloud init' automatically...", 0)
	if initErr := runInit(cmd, nil); initErr != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("auto-init failed: %s", initErr),
			output.ErrNotInitialized,
			"Automatic initialization failed. Run 'vibecloud init' manually.",
			&output.Recovery{Command: "vibecloud init", AutoRecoverable: true},
		)
		return nil, initErr
	}

	// Re-load config after init.
	projCfg, err = loadProjectConfig(cwd)
	if err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("failed to read .vibecloud.json after init: %s", err),
			output.ErrNotInitialized,
			"Initialization ran but .vibecloud.json is still missing. Run 'vibecloud init' manually.",
			&output.Recovery{Command: "vibecloud init", AutoRecoverable: true},
		)
		return nil, err
	}

	return projCfg, nil
}

// vercelLink attempts `vercel link --yes`. If it fails because of a missing
// scope (common in non-TTY environments), it parses the error, picks the
// first available scope, and retries.
func vercelLink(ctx context.Context) error {
	out, err := vexec.RunCapture(ctx, "vercel", "link", "--yes")
	if err == nil {
		fmt.Fprint(os.Stderr, out)
		return nil
	}

	// Try to parse the JSON error for a scope suggestion.
	var resp struct {
		Status  string `json:"status"`
		Reason  string `json:"reason"`
		Choices []struct {
			Name string `json:"name"`
		} `json:"choices"`
	}
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &resp); jsonErr == nil &&
		resp.Reason == "missing_scope" && len(resp.Choices) > 0 {
		scope := resp.Choices[0].Name
		output.Progress("vercel", "linking", fmt.Sprintf("  Auto-selecting scope: %s", scope), 0)
		return vexec.Run(ctx, "vercel", "link", "--yes", "--scope", scope)
	}

	// Not a scope error — return original error.
	fmt.Fprint(os.Stderr, out)
	return err
}

// writeClaudeMD creates or appends a CLAUDE.md with vibecloud instructions.
func writeClaudeMD(cwd, projectName string, stack detect.DetectedStack) {
	claudeMDPath := filepath.Join(cwd, "CLAUDE.md")

	vibeSection := fmt.Sprintf(`
## Deployment (VibeCloud)

This project uses **VibeCloud** for deployment. Use vibecloud commands instead of %s CLIs directly.

### Commands
- `+"`vibecloud deploy`"+` — deploy all providers in dependency order
- `+"`vibecloud deploy --prod`"+` — deploy to production
- `+"`vibecloud deploy --provider <name>`"+` — deploy a single provider
- `+"`vibecloud doctor`"+` — preflight check (run before deploy to catch issues)
- `+"`vibecloud explain`"+` — full project state across all providers
- `+"`vibecloud status`"+` — provider status
- `+"`vibecloud logs --provider <name>`"+` — fetch logs
- `+"`vibecloud login`"+` — authenticate with all providers
- `+"`vibecloud env add <KEY>`"+` — securely set an environment variable
- `+"`vibecloud env sync`"+` — sync Supabase keys to Vercel
- `+"`vibecloud env list`"+` — list environment variables
- `+"`vibecloud env rm <KEY>`"+` — remove an environment variable

### Agent flags
- `+"`--machine`"+` — emit NDJSON progress events to stderr, suppress ANSI formatting
- `+"`--yes`"+` — skip interactive prompts (non-interactive / CI mode)

### Output format
All commands output JSON to stdout with this structure:
`+"```json"+`
{
  "success": true|false,
  "message": "...",
  "error_code": "...",
  "warnings": ["non-fatal issue 1", "..."],
  "data": { ... },
  "recovery": { "command": "vibecloud ...", "auto_recoverable": true },
  "claude_instructions": "Actionable next step."
}
`+"```"+`
Parse the `+"`claude_instructions`"+` field for what to do next. If `+"`recovery.auto_recoverable`"+` is true, you can run `+"`recovery.command`"+` automatically. Check `+"`warnings`"+` for non-fatal issues to surface to the user.

### Exit codes
- 0: success
- 1: general error
- 2: authentication error
- 3: not initialized / not linked
- 4: build / deploy error
- 5: filesystem / network error

### Secrets
NEVER accept, display, or handle API keys, passwords, or secrets in conversation.
Always use `+"`vibecloud env add <KEY>`"+` to set secrets (prompts the user directly, values never enter the AI context).
Use `+"`vibecloud env sync`"+` to automatically wire Supabase keys into Vercel env vars.

### Providers: %s
`, strings.Join(stack.Providers, "/"), strings.Join(stack.Providers, ", "))

	// Check if CLAUDE.md exists.
	if existing, err := os.ReadFile(claudeMDPath); err == nil {
		// Don't duplicate if already present.
		if strings.Contains(string(existing), "## Deployment (VibeCloud)") {
			return
		}
		// Append to existing.
		content := string(existing) + "\n" + vibeSection
		_ = os.WriteFile(claudeMDPath, []byte(content), 0644)
		output.Progress("", "claude_md", "  Appended VibeCloud section to existing CLAUDE.md", 0)
	} else {
		// Create new.
		header := fmt.Sprintf("# %s\n", projectName)
		_ = os.WriteFile(claudeMDPath, []byte(header+vibeSection), 0644)
		output.Progress("", "claude_md", "  Created CLAUDE.md with VibeCloud instructions", 0)
	}
}

// findOrCreateSupabaseProject checks if the user is authenticated, looks for
// an existing project matching the name, and creates one if needed. Returns the
// project ref or empty string on failure.
func findOrCreateSupabaseProject(ctx context.Context, projectName string) string {
	// Check if authenticated by listing projects.
	listOut, _, listErr := vexec.RunCaptureAll(ctx, "supabase", "projects", "list")
	if listErr != nil {
		// Not authenticated or CLI issue — skip creation.
		return ""
	}

	// Parse the table output to find a matching project.
	// supabase projects list outputs a table with columns like:
	//   ORG ID | ID | NAME | ...
	for _, line := range strings.Split(listOut, "\n") {
		fields := strings.Split(line, "|")
		if len(fields) >= 3 {
			name := strings.TrimSpace(fields[2])
			ref := strings.TrimSpace(fields[1])
			if strings.EqualFold(name, projectName) && ref != "" && ref != "ID" {
				output.Progress("supabase", "found", fmt.Sprintf("  Found existing Supabase project: %s (ref: %s)", name, ref), 0)
				return ref
			}
		}
	}

	// No matching project found — create one.
	output.Progress("supabase", "creating", fmt.Sprintf("  No Supabase project found. Creating '%s'...", projectName), 0)
	fmt.Fprintf(os.Stderr, "No Supabase project found. Creating one...\n")

	createOut, _, createErr := vexec.RunCaptureAll(ctx, "supabase", "projects", "create", projectName, "--region", "us-east-1")
	if createErr != nil {
		output.Warn("supabase", fmt.Sprintf("failed to create Supabase project: %s", createErr))
		return ""
	}

	// Parse output for project ref. Expected format:
	// Created a new project: <name> (ref: <ref>)
	re := regexp.MustCompile(`\(ref:\s*([a-z0-9]+)\)`)
	if matches := re.FindStringSubmatch(createOut); len(matches) >= 2 {
		output.Progress("supabase", "created", fmt.Sprintf("  Created Supabase project: %s (ref: %s)", projectName, matches[1]), 0)
		return matches[1]
	}

	// Try to find ref in output with alternative patterns.
	re2 := regexp.MustCompile(`[a-z]{20,}`)
	if matches := re2.FindString(createOut); matches != "" {
		return matches
	}

	output.Warn("supabase", "created project but could not parse ref from output")
	return ""
}
