package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	releaseRepo    = "ChrisDang/vibecloud-releases"
	checkInterval  = 24 * time.Hour
	requestTimeout = 2 * time.Second
	cacheFile      = "version-cache.json"
)

// UpdateInfo holds the result of a version check.
type UpdateInfo struct {
	Current   string
	Latest    string
	Outdated  bool
	UpgradeBy string // human-readable upgrade instruction
}

type versionCache struct {
	LatestVersion string `json:"latest_version"`
	CheckedAt     int64  `json:"checked_at"`
}

// CheckForUpdate compares the running version against the latest GitHub release.
// It caches the result to avoid hitting the network on every invocation.
// Returns nil if the version is current or the check can't be performed.
func CheckForUpdate(currentVersion string) *UpdateInfo {
	if currentVersion == "dev" || currentVersion == "" {
		return nil
	}

	cachePath := getCachePath()

	// Try the cache first.
	if cached, ok := loadCache(cachePath); ok {
		if time.Since(time.Unix(cached.CheckedAt, 0)) < checkInterval {
			return compareVersions(currentVersion, cached.LatestVersion)
		}
	}

	// Cache miss or stale — fetch from GitHub.
	latest, err := fetchLatestVersion()
	if err != nil {
		return nil // silently skip on network errors
	}

	// Persist to cache.
	saveCache(cachePath, &versionCache{
		LatestVersion: latest,
		CheckedAt:     time.Now().Unix(),
	})

	return compareVersions(currentVersion, latest)
}

// UpgradeNotice returns the string to prepend to claude_instructions, or ""
// if no update is available.
func UpgradeNotice(info *UpdateInfo) string {
	if info == nil || !info.Outdated {
		return ""
	}
	return fmt.Sprintf(
		"IMPORTANT: vibecloud is outdated (v%s → v%s). Upgrade by running: curl -fsSL https://raw.githubusercontent.com/%s/main/install.sh | sh — then re-run the previous command.",
		info.Current, info.Latest, releaseRepo,
	)
}

// --- internal helpers --------------------------------------------------------

func fetchLatestVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", releaseRepo)

	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}

func compareVersions(current, latest string) *UpdateInfo {
	if latest == "" {
		return nil
	}
	info := &UpdateInfo{Current: current, Latest: latest}
	if isNewer(latest, current) {
		info.Outdated = true
	}
	return info
}

// isNewer returns true if a is a higher semver than b (major.minor.patch only).
func isNewer(a, b string) bool {
	ap := parseSemver(a)
	bp := parseSemver(b)
	for i := 0; i < 3; i++ {
		if ap[i] > bp[i] {
			return true
		}
		if ap[i] < bp[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		// Strip any pre-release suffix (e.g. "1-beta").
		num := strings.SplitN(parts[i], "-", 2)[0]
		result[i], _ = strconv.Atoi(num)
	}
	return result
}

func getCachePath() string {
	dir := os.Getenv("VIBECLOUD_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = filepath.Join(home, ".vibecloud")
	}
	return filepath.Join(dir, cacheFile)
}

func loadCache(path string) (*versionCache, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var c versionCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, false
	}
	return &c, true
}

func saveCache(path string, c *versionCache) {
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0700)

	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0600)
}
