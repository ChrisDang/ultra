package exec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Supported CLI names and their install instructions.
var installHints = map[string]string{
	"vercel":   "npm i -g vercel",
	"supabase": "brew install supabase/tap/supabase  (or: npm i -g supabase)",
	"eas":      "npm i -g eas-cli",
}

// CheckInstalled returns true if the given binary is available in PATH.
func CheckInstalled(binary string) bool {
	_, err := exec.LookPath(binary)
	return err == nil
}

// InstallHint returns the install command for a known CLI, or a generic message.
func InstallHint(name string) string {
	if hint, ok := installHints[name]; ok {
		return hint
	}
	return fmt.Sprintf("install %q and ensure it is in your PATH", name)
}

// RequireCLI checks that a CLI binary is installed and returns a user-friendly
// error with install instructions if it is not.
func RequireCLI(name string) error {
	if CheckInstalled(name) {
		return nil
	}
	return fmt.Errorf("%s is not installed — run: %s", name, InstallHint(name))
}

// Run executes a command with full terminal passthrough (stdin, stdout, stderr).
// Use this for interactive commands where the subprocess needs TTY detection
// (e.g. vercel link, supabase link, login flows).
func Run(ctx context.Context, binary string, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunCapture executes a command and returns its stdout as a string.
// Stderr is still sent to the terminal.
func RunCapture(ctx context.Context, binary string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return out.String(), err
}

// RunNonInteractive executes a command with stdin closed so TTY prompts
// cannot block. Stdout and stderr still go to the terminal. Use this for
// deploy/build commands that Claude triggers — they must never wait for
// human input.
func RunNonInteractive(ctx context.Context, binary string, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = nil // no stdin — forces non-interactive behaviour
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunCaptureAll executes a command and captures both stdout and stderr.
// Stdin is closed (non-interactive). Use this when you need to inspect
// error output programmatically.
func RunCaptureAll(ctx context.Context, binary string, args ...string) (stdout string, stderr string, err error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdin = nil
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}
