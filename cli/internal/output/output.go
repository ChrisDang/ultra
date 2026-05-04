package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// upgradeNotice is prepended to claude_instructions when the CLI is outdated.
// Set by the root command's PersistentPreRun via SetUpgradeNotice.
var upgradeNotice string

// machineMode suppresses human-readable stderr output and emits structured
// NDJSON progress events instead. Set via the --machine global flag.
var machineMode bool

// SetUpgradeNotice stores an upgrade message to be prepended to all
// claude_instructions output.
func SetUpgradeNotice(notice string) {
	upgradeNotice = notice
}

// SetMachineMode enables or disables machine-readable output mode.
func SetMachineMode(enabled bool) {
	machineMode = enabled
}

// MachineMode returns whether machine-readable output is enabled.
func MachineMode() bool {
	return machineMode
}

// Error codes that Claude can branch on programmatically.
const (
	ErrNotInitialized  = "NOT_INITIALIZED"
	ErrCLIMissing      = "CLI_MISSING"
	ErrAuthExpired     = "AUTH_EXPIRED"
	ErrAuthRequired    = "AUTH_REQUIRED"
	ErrNotLinked       = "NOT_LINKED"
	ErrDeployFailed    = "DEPLOY_FAILED"
	ErrPartialDeploy   = "PARTIAL_DEPLOY"
	ErrBuildFailed     = "BUILD_FAILED"
	ErrMigrationFailed = "MIGRATION_FAILED"
	ErrProviderUnknown = "PROVIDER_UNKNOWN"
	ErrNoProvider      = "NO_PROVIDER"
	ErrFilesystem      = "FILESYSTEM_ERROR"
)

// Exit codes that let agents distinguish error categories without parsing JSON.
const (
	ExitSuccess        = 0
	ExitGeneralError   = 1
	ExitAuthError      = 2
	ExitNotInitialized = 3
	ExitBuildError     = 4
	ExitFilesystem     = 5
)

// exitCodeForError maps error codes to exit codes.
func exitCodeForError(errorCode string) int {
	switch errorCode {
	case ErrAuthExpired, ErrAuthRequired:
		return ExitAuthError
	case ErrNotInitialized, ErrNotLinked:
		return ExitNotInitialized
	case ErrDeployFailed, ErrPartialDeploy, ErrBuildFailed, ErrMigrationFailed:
		return ExitBuildError
	case ErrFilesystem:
		return ExitFilesystem
	default:
		return ExitGeneralError
	}
}

// Recovery tells Claude exactly how to fix a failure — including a command
// it can run automatically.
type Recovery struct {
	Command         string `json:"command"`
	AutoRecoverable bool   `json:"auto_recoverable"`
}

// Response is the structured JSON output format consumed by Claude and other
// AI agents.
type Response struct {
	Success            bool        `json:"success"`
	Message            string      `json:"message"`
	ErrorCode          string      `json:"error_code,omitempty"`
	Warnings           []string    `json:"warnings,omitempty"`
	Data               interface{} `json:"data,omitempty"`
	Recovery           *Recovery   `json:"recovery,omitempty"`
	ClaudeInstructions string      `json:"claude_instructions,omitempty"`
}

// Event is a structured progress event emitted as NDJSON to stderr
// when --machine mode is active.
type Event struct {
	Type     string  `json:"type"`               // "progress", "phase", "warning", "error"
	Phase    string  `json:"phase,omitempty"`     // e.g. "supabase_migrations", "vercel_deploy"
	Provider string  `json:"provider,omitempty"`  // e.g. "vercel", "supabase", "expo"
	Message  string  `json:"message"`             // human-readable description
	Progress float64 `json:"progress,omitempty"`  // 0.0–1.0 when known
	Time     string  `json:"time"`                // RFC3339
}

// Progress emits a structured progress event to stderr (in machine mode)
// or a human-readable message (in normal mode).
func Progress(provider, phase, message string, progress float64) {
	if machineMode {
		evt := Event{
			Type:     "progress",
			Phase:    phase,
			Provider: provider,
			Message:  message,
			Progress: progress,
			Time:     time.Now().UTC().Format(time.RFC3339),
		}
		data, _ := json.Marshal(evt)
		fmt.Fprintln(os.Stderr, string(data))
	} else {
		fmt.Fprintln(os.Stderr, message)
	}
}

// Warn emits a warning event to stderr (in machine mode) or a human-readable
// warning (in normal mode).
func Warn(provider, message string) {
	if machineMode {
		evt := Event{
			Type:     "warning",
			Provider: provider,
			Message:  message,
			Time:     time.Now().UTC().Format(time.RFC3339),
		}
		data, _ := json.Marshal(evt)
		fmt.Fprintln(os.Stderr, string(data))
	} else {
		fmt.Fprintf(os.Stderr, "⚠ %s\n", message)
	}
}

// Print marshals resp to JSON and writes it to stdout.
// If an upgrade notice has been set, it is prepended to claude_instructions.
func Print(resp Response) {
	if upgradeNotice != "" && resp.ClaudeInstructions != "" {
		resp.ClaudeInstructions = upgradeNotice + " " + resp.ClaudeInstructions
	} else if upgradeNotice != "" {
		resp.ClaudeInstructions = upgradeNotice
	}

	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		fmt.Printf(`{"success":false,"message":"failed to marshal response: %s"}`, err.Error())
		return
	}
	fmt.Println(string(data))
}

// PrintError prints an error response and exits with code 1.
func PrintError(msg string) {
	Print(Response{
		Success:            false,
		Message:            msg,
		ClaudeInstructions: msg,
	})
	os.Exit(ExitGeneralError)
}

// PrintErrorWithRecovery prints a structured error with an error code,
// actionable claude_instructions, and an optional recovery command.
func PrintErrorWithRecovery(msg, errorCode, instructions string, recovery *Recovery) {
	Print(Response{
		Success:            false,
		Message:            msg,
		ErrorCode:          errorCode,
		Recovery:           recovery,
		ClaudeInstructions: instructions,
	})
	os.Exit(exitCodeForError(errorCode))
}

// PrintSuccess prints a success response (exits normally).
func PrintSuccess(msg string, data interface{}, instructions string) {
	Print(Response{
		Success:            true,
		Message:            msg,
		Data:               data,
		ClaudeInstructions: instructions,
	})
}

// PrintSuccessWithWarnings prints a success response that includes non-fatal warnings.
func PrintSuccessWithWarnings(msg string, data interface{}, warnings []string, instructions string) {
	Print(Response{
		Success:            true,
		Message:            msg,
		Data:               data,
		Warnings:           warnings,
		ClaudeInstructions: instructions,
	})
}

// PrintPartialSuccess prints a response where some providers succeeded and
// others failed. Includes per-provider details in claude_instructions.
func PrintPartialSuccess(msg string, data interface{}, errorCode string, instructions string, recovery *Recovery) {
	Print(Response{
		Success:            true,
		Message:            msg,
		ErrorCode:          errorCode,
		Data:               data,
		Recovery:           recovery,
		ClaudeInstructions: instructions,
	})
}

// PrintPartialSuccessWithWarnings prints a partial success response with warnings.
func PrintPartialSuccessWithWarnings(msg string, data interface{}, errorCode string, warnings []string, instructions string, recovery *Recovery) {
	Print(Response{
		Success:            true,
		Message:            msg,
		ErrorCode:          errorCode,
		Warnings:           warnings,
		Data:               data,
		Recovery:           recovery,
		ClaudeInstructions: instructions,
	})
}

// BuildDeployInstructions builds a specific, actionable claude_instructions
// string from a map of provider results.
func BuildDeployInstructions(results map[string]string) string {
	var succeeded, failed []string
	for provider, result := range results {
		if result == "deployed" || result == "build submitted" || result == "migrations applied" {
			succeeded = append(succeeded, provider+": "+result)
		} else {
			failed = append(failed, provider+": "+result)
		}
	}

	if len(failed) == 0 {
		return "All providers deployed successfully. No action needed."
	}

	var b strings.Builder
	if len(succeeded) > 0 {
		b.WriteString("Succeeded: ")
		b.WriteString(strings.Join(succeeded, ", "))
		b.WriteString(". ")
	}
	b.WriteString("Failed: ")
	b.WriteString(strings.Join(failed, ", "))
	b.WriteString(". Run 'vibecloud doctor' to diagnose issues, then retry with 'vibecloud deploy'.")
	return b.String()
}
