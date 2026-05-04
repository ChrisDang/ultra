package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/christopherdang/vibecloud/cli/internal/api"
	"github.com/christopherdang/vibecloud/cli/internal/config"
	"github.com/christopherdang/vibecloud/cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage VibeCloud account authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with your VibeCloud account",
	RunE:  runAuthLogin,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored VibeCloud credentials",
	RunE:  runAuthLogout,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current VibeCloud auth status",
	RunE:  runAuthStatus,
}

var authUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade to Premium (alpha preview)",
	RunE:  runAuthUpgrade,
}

var authDowngradeCmd = &cobra.Command{
	Use:   "downgrade",
	Short: "Downgrade to Free tier",
	RunE:  runAuthDowngrade,
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authUpgradeCmd)
	authCmd.AddCommand(authDowngradeCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}

	if cfg.AccessToken != "" {
		output.PrintSuccess(
			"Already authenticated",
			map[string]string{"email": cfg.UserEmail, "tier": cfg.UserTier},
			fmt.Sprintf("Already logged in as %s (%s tier). Run 'vibecloud auth logout' first to switch accounts.", cfg.UserEmail, cfg.UserTier),
		)
		return nil
	}

	baseURL := cfg.APIBaseURL
	if baseURL == "" {
		baseURL = config.DefaultAPIBaseURL
	}

	// Open browser to login page.
	loginURL := baseURL + "/login?cli=true"
	fmt.Fprintf(os.Stderr, "Opening %s in your browser...\n", loginURL)
	openBrowser(loginURL)

	fmt.Fprintf(os.Stderr, "\nLog in on the website, go to Dashboard, and generate a CLI code.\n")
	fmt.Fprintf(os.Stderr, "Enter the code: ")

	var code string
	if term.IsTerminal(int(os.Stdin.Fd())) {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			code = strings.TrimSpace(strings.ToUpper(scanner.Text()))
		}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			code = strings.TrimSpace(strings.ToUpper(scanner.Text()))
		}
	}

	if code == "" {
		output.PrintErrorWithRecovery(
			"No code entered",
			output.ErrAuthRequired,
			"No code was provided. Run 'vibecloud auth login' to try again.",
			nil,
		)
		return nil
	}

	// Exchange code for tokens.
	client := api.NewClient(baseURL, "", "")
	tokens, err := client.ExchangeDeviceCode(code)
	if err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("Code exchange failed: %s", err),
			output.ErrAuthRequired,
			"Invalid or expired code. Generate a new code from the dashboard and try again.",
			nil,
		)
		return nil
	}

	// Validate tokens by calling /me.
	authedClient := api.NewClient(baseURL, tokens.AccessToken, tokens.RefreshToken)
	user, err := authedClient.GetMe()
	if err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("Token validation failed: %s", err),
			output.ErrAuthRequired,
			"Token validation failed. Try 'vibecloud auth login' again.",
			nil,
		)
		return nil
	}

	// Store credentials.
	cfg.AccessToken = tokens.AccessToken
	cfg.RefreshToken = tokens.RefreshToken
	cfg.APIBaseURL = baseURL
	cfg.UserEmail = user.Email
	cfg.UserTier = user.Tier
	if err := config.Save(cfg); err != nil {
		output.PrintError(fmt.Sprintf("Failed to save credentials: %s", err))
		return nil
	}

	output.PrintSuccess(
		fmt.Sprintf("Authenticated as %s", user.Email),
		map[string]string{"email": user.Email, "tier": user.Tier},
		fmt.Sprintf("Logged in as %s (%s tier). You can now deploy with 'vibecloud deploy'.", user.Email, user.Tier),
	)
	return nil
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}

	cfg.AccessToken = ""
	cfg.RefreshToken = ""
	cfg.UserEmail = ""
	cfg.UserTier = ""

	if err := config.Save(cfg); err != nil {
		output.PrintError(fmt.Sprintf("Failed to clear credentials: %s", err))
		return nil
	}

	output.PrintSuccess("Logged out", nil, "VibeCloud credentials removed. Run 'vibecloud auth login' to authenticate again.")
	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil || cfg.AccessToken == "" {
		output.PrintErrorWithRecovery(
			"Not authenticated with VibeCloud",
			output.ErrAuthRequired,
			"No VibeCloud account linked. Run 'vibecloud auth login' to authenticate.",
			&output.Recovery{Command: "vibecloud auth login", AutoRecoverable: false},
		)
		return nil
	}

	// Verify token is still valid.
	baseURL := cfg.APIBaseURL
	if baseURL == "" {
		baseURL = config.DefaultAPIBaseURL
	}
	client := api.NewClient(baseURL, cfg.AccessToken, cfg.RefreshToken)
	user, err := client.GetMe()
	if err != nil {
		output.PrintErrorWithRecovery(
			"Token expired or invalid",
			output.ErrAuthExpired,
			"Your session has expired. Run 'vibecloud auth login' to re-authenticate.",
			&output.Recovery{Command: "vibecloud auth login", AutoRecoverable: false},
		)
		return nil
	}

	// Update cached info.
	cfg.UserEmail = user.Email
	cfg.UserTier = user.Tier
	_ = config.Save(cfg)

	output.PrintSuccess(
		"Authenticated",
		map[string]string{"email": user.Email, "tier": user.Tier},
		fmt.Sprintf("Logged in as %s (%s tier).", user.Email, user.Tier),
	)
	return nil
}

func runAuthUpgrade(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil || cfg.AccessToken == "" {
		output.PrintErrorWithRecovery(
			"Not authenticated",
			output.ErrAuthRequired,
			"You must be logged in to change tiers. Run 'vibecloud auth login' first.",
			&output.Recovery{Command: "vibecloud auth login", AutoRecoverable: false},
		)
		return nil
	}

	if cfg.UserTier == "premium" {
		output.PrintSuccess("Already on Premium", map[string]string{"tier": "premium"}, "You are already on the Premium tier.")
		return nil
	}

	// Show alpha disclaimer.
	fmt.Fprintln(os.Stderr, "\nPremium is in alpha preview.")
	fmt.Fprintln(os.Stderr, "  When premium launches with billing, your tier will revert to free.")
	fmt.Fprintln(os.Stderr, "  Your snapshots and configuration will be preserved.")
	fmt.Fprintln(os.Stderr)

	if !flagYes {
		fmt.Fprintf(os.Stderr, "Continue? (y/N): ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if answer != "y" && answer != "yes" {
				output.PrintSuccess("Cancelled", nil, "Upgrade cancelled.")
				return nil
			}
		}
	}

	baseURL := cfg.APIBaseURL
	if baseURL == "" {
		baseURL = config.DefaultAPIBaseURL
	}
	client := api.NewClient(baseURL, cfg.AccessToken, cfg.RefreshToken)
	user, err := client.UpdateTier("premium")
	if err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("Upgrade failed: %s", err),
			output.ErrDeployFailed,
			"Failed to upgrade tier. Check 'vibecloud auth status' and try again.",
			nil,
		)
		return nil
	}

	cfg.UserTier = user.Tier
	_ = config.Save(cfg)

	output.PrintSuccess(
		"Upgraded to Premium (alpha)",
		map[string]string{"email": user.Email, "tier": user.Tier},
		"Upgraded to premium tier. You now have unlimited deploys and production deployments. Remember: this is an alpha preview — your tier will revert to free when billing launches.",
	)
	return nil
}

func runAuthDowngrade(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil || cfg.AccessToken == "" {
		output.PrintErrorWithRecovery(
			"Not authenticated",
			output.ErrAuthRequired,
			"You must be logged in to change tiers. Run 'vibecloud auth login' first.",
			&output.Recovery{Command: "vibecloud auth login", AutoRecoverable: false},
		)
		return nil
	}

	if cfg.UserTier == "free" {
		output.PrintSuccess("Already on Free", map[string]string{"tier": "free"}, "You are already on the Free tier.")
		return nil
	}

	baseURL := cfg.APIBaseURL
	if baseURL == "" {
		baseURL = config.DefaultAPIBaseURL
	}
	client := api.NewClient(baseURL, cfg.AccessToken, cfg.RefreshToken)
	user, err := client.UpdateTier("free")
	if err != nil {
		output.PrintErrorWithRecovery(
			fmt.Sprintf("Downgrade failed: %s", err),
			output.ErrDeployFailed,
			"Failed to downgrade tier. Check 'vibecloud auth status' and try again.",
			nil,
		)
		return nil
	}

	cfg.UserTier = user.Tier
	_ = config.Save(cfg)

	output.PrintSuccess(
		"Downgraded to Free",
		map[string]string{"email": user.Email, "tier": user.Tier},
		"Downgraded to free tier. Deploy limit: 15/month, preview only.",
	)
	return nil
}

// openBrowser opens a URL in the default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = cmd.Start()
}
