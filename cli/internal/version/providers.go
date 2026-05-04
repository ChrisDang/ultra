package version

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	vexec "github.com/christopherdang/vibecloud/cli/internal/exec"
)

// ProviderVersionInfo holds the result of checking a provider CLI's version.
type ProviderVersionInfo struct {
	Provider      string `json:"provider"`
	Binary        string `json:"binary"`
	Current       string `json:"current_version"`
	Minimum       string `json:"minimum_version"`
	MaxTested     string `json:"max_tested_version"`
	Outdated      bool   `json:"outdated"`
	Untested      bool   `json:"untested"`
	UpdateCommand string `json:"update_command,omitempty"`
}

// providerManifest mirrors the shape of provider_versions.json.
type providerManifest struct {
	Providers map[string]providerEntry `json:"providers"`
}

type providerEntry struct {
	Binary           string `json:"binary"`
	MinVersion       string `json:"min_version"`
	MaxTestedVersion string `json:"max_tested_version"`
	UpdateCommand    string `json:"update_command"`
}

// versionArgs maps each CLI binary to the args that print its version.
var versionArgs = map[string][]string{
	"vercel":   {"--version"},
	"supabase": {"--version"},
	"eas":      {"--version"},
}

// versionRegexp extracts the first semver-like string (e.g. "39.2.1") from CLI output.
var versionRegexp = regexp.MustCompile(`(\d+\.\d+\.\d+)`)

// loadManifest reads provider_versions.json from the project root.
// It walks up from cwd looking for the file.
func loadManifest() (*providerManifest, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for {
		p := filepath.Join(dir, "provider_versions.json")
		data, err := os.ReadFile(p)
		if err == nil {
			var m providerManifest
			if err := json.Unmarshal(data, &m); err != nil {
				return nil, err
			}
			return &m, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return nil, os.ErrNotExist
}

// CheckProviderVersion checks whether a provider CLI is installed at a
// sufficient version and whether it exceeds the max tested version.
// Returns nil if the binary is not installed or the version cannot be determined.
func CheckProviderVersion(binary string) *ProviderVersionInfo {
	args, ok := versionArgs[binary]
	if !ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stdout, stderr, err := vexec.RunCaptureAll(ctx, binary, args...)
	if err != nil {
		return nil
	}

	raw := stdout + " " + stderr
	matches := versionRegexp.FindStringSubmatch(raw)
	if len(matches) < 2 {
		return nil
	}

	current := strings.TrimSpace(matches[1])

	info := &ProviderVersionInfo{
		Binary:  binary,
		Current: current,
	}

	// Load version bounds from the manifest.
	manifest, err := loadManifest()
	if err != nil {
		return info
	}

	entry, ok := manifest.Providers[binary]
	if !ok {
		return info
	}

	info.Minimum = entry.MinVersion
	info.MaxTested = entry.MaxTestedVersion
	info.UpdateCommand = entry.UpdateCommand

	// Too old — below minimum.
	if entry.MinVersion != "" && isNewer(entry.MinVersion, current) {
		info.Outdated = true
	}

	// Too new — above max tested version (potential drift).
	if entry.MaxTestedVersion != "" && isNewer(current, entry.MaxTestedVersion) {
		info.Untested = true
	}

	return info
}
