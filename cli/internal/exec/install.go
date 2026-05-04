package exec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// brewFormulas maps CLI names to their Homebrew install commands.
// CLIs without a direct brew formula (vercel, eas) are installed via npm
// after ensuring Node.js is present.
var brewFormulas = map[string]string{
	"supabase": "supabase/tap/supabase",
}

// npmPackages maps CLI binary names to their npm package names.
var npmPackages = map[string]string{
	"vercel": "vercel",
	"eas":    "eas-cli",
}

// EnsureCLI checks that a CLI binary is installed. If it is missing, it
// attempts to install it via Homebrew (installing Homebrew itself first
// if necessary). Returns an error only if the install attempt fails.
func EnsureCLI(name string) error {
	if CheckInstalled(name) {
		return nil
	}

	fmt.Fprintf(os.Stderr, "  %s not found — installing automatically...\n", name)

	if err := ensureBrew(); err != nil {
		return fmt.Errorf("failed to install Homebrew: %w", err)
	}

	// Direct brew formula (supabase).
	if formula, ok := brewFormulas[name]; ok {
		return brewInstall(formula)
	}

	// npm-based CLI (vercel, eas) — need Node.js first.
	if pkg, ok := npmPackages[name]; ok {
		if err := ensureNode(); err != nil {
			return err
		}
		return npmInstallGlobal(pkg)
	}

	return fmt.Errorf("%s is not installed and no automatic installer is configured — run: %s", name, InstallHint(name))
}

// ensureBrew installs Homebrew via curl if it is not already present.
func ensureBrew() error {
	if CheckInstalled("brew") {
		return nil
	}

	fmt.Fprintln(os.Stderr, "  Homebrew not found — installing via curl...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/bash", "-c",
		`NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("homebrew install script failed: %w", err)
	}

	// Verify it landed in PATH.
	if !CheckInstalled("brew") {
		return fmt.Errorf("homebrew was installed but 'brew' is not in PATH — restart your terminal and retry")
	}

	fmt.Fprintln(os.Stderr, "  Homebrew installed successfully.")
	return nil
}

// ensureNode installs Node.js via Homebrew if it is not already present.
func ensureNode() error {
	if CheckInstalled("node") {
		return nil
	}

	fmt.Fprintln(os.Stderr, "  Node.js not found — installing via Homebrew...")
	return brewInstall("node")
}

// brewInstall runs `brew install <formula>`.
func brewInstall(formula string) error {
	fmt.Fprintf(os.Stderr, "  brew install %s\n", formula)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "brew", "install", formula)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew install %s failed: %w", formula, err)
	}
	return nil
}

// npmInstallGlobal runs `npm install -g <pkg>`.
func npmInstallGlobal(pkg string) error {
	fmt.Fprintf(os.Stderr, "  npm install -g %s\n", pkg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "npm", "install", "-g", pkg)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm install -g %s failed: %w", pkg, err)
	}
	return nil
}
