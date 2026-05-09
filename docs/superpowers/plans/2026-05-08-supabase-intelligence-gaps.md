# Supabase Intelligence Gaps Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 6 Supabase integration gaps in the vibecloud CLI — dynamic pooler discovery, DATABASE_URL construction, unpause detection, CLAUDE.md guidance, and new `db` commands.

**Architecture:** New `internal/supabase/` package centralizes project info, pooler discovery, and connection string construction. Existing commands (`doctor`, `init`, `env sync`) are updated to use it. New `vibecloud db push` and `vibecloud db status` commands wrap Supabase CLI operations.

**Tech Stack:** Go 1.26, Cobra CLI framework, `supabase projects list -o json` for project metadata, TCP dial probes for pooler discovery.

---

### Task 1: Create `internal/supabase/connstring.go` — connection string builder

**Files:**
- Create: `cli/internal/supabase/connstring.go`
- Test: `cli/internal/supabase/connstring_test.go`

- [ ] **Step 1: Write the failing tests**

Create `cli/internal/supabase/connstring_test.go`:

```go
package supabase

import (
	"testing"
)

func TestBuildDatabaseURL_Basic(t *testing.T) {
	got := BuildDatabaseURL("abcdefghijklmnopqrst", "aws-1-us-east-1.pooler.supabase.com", 6543, "mypassword")
	want := "postgresql://postgres.abcdefghijklmnopqrst:mypassword@aws-1-us-east-1.pooler.supabase.com:6543/postgres"
	if got != want {
		t.Errorf("BuildDatabaseURL() = %q, want %q", got, want)
	}
}

func TestBuildDatabaseURL_SessionMode(t *testing.T) {
	got := BuildDatabaseURL("abcdefghijklmnopqrst", "aws-0-us-east-1.pooler.supabase.com", 5432, "pass123")
	want := "postgresql://postgres.abcdefghijklmnopqrst:pass123@aws-0-us-east-1.pooler.supabase.com:5432/postgres"
	if got != want {
		t.Errorf("BuildDatabaseURL() = %q, want %q", got, want)
	}
}

func TestBuildDatabaseURL_SpecialCharsInPassword(t *testing.T) {
	got := BuildDatabaseURL("abcdefghijklmnopqrst", "aws-0-us-east-1.pooler.supabase.com", 6543, "p@ss w0rd!#")
	want := "postgresql://postgres.abcdefghijklmnopqrst:p%40ss+w0rd%21%23@aws-0-us-east-1.pooler.supabase.com:6543/postgres"
	if got != want {
		t.Errorf("BuildDatabaseURL() = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd cli && go test ./internal/supabase/ -v -run TestBuildDatabaseURL`
Expected: compilation error — package doesn't exist yet.

- [ ] **Step 3: Write the implementation**

Create `cli/internal/supabase/connstring.go`:

```go
package supabase

import (
	"fmt"
	"net/url"
)

// BuildDatabaseURL constructs a Supavisor pooler connection string.
// The password is URL-encoded to handle special characters.
func BuildDatabaseURL(ref, poolerHost string, port int, password string) string {
	return fmt.Sprintf("postgresql://postgres.%s:%s@%s:%d/postgres",
		ref,
		url.QueryEscape(password),
		poolerHost,
		port,
	)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd cli && go test ./internal/supabase/ -v -run TestBuildDatabaseURL`
Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add cli/internal/supabase/connstring.go cli/internal/supabase/connstring_test.go
git commit -m "feat(cli): add Supabase connection string builder"
```

---

### Task 2: Create `internal/supabase/project.go` — project info from CLI

**Files:**
- Create: `cli/internal/supabase/project.go`
- Test: `cli/internal/supabase/project_test.go`

- [ ] **Step 1: Write the failing tests**

Create `cli/internal/supabase/project_test.go`:

```go
package supabase

import (
	"testing"
)

func TestParseProjectList_MatchesRef(t *testing.T) {
	jsonData := `[
		{"ref":"aaabbbcccdddeeefffgg","name":"OtherProject","region":"eu-west-1","status":"ACTIVE_HEALTHY","database":{"host":"db.aaabbbcccdddeeefffgg.supabase.co"}},
		{"ref":"bwxvxzfzujkxnphedtom","name":"VibeCloudAI","region":"us-east-1","status":"ACTIVE_HEALTHY","database":{"host":"db.bwxvxzfzujkxnphedtom.supabase.co"}}
	]`
	info, err := parseProjectList(jsonData, "bwxvxzfzujkxnphedtom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Ref != "bwxvxzfzujkxnphedtom" {
		t.Errorf("Ref = %q, want %q", info.Ref, "bwxvxzfzujkxnphedtom")
	}
	if info.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", info.Region, "us-east-1")
	}
	if info.Status != "ACTIVE_HEALTHY" {
		t.Errorf("Status = %q, want %q", info.Status, "ACTIVE_HEALTHY")
	}
	if info.Name != "VibeCloudAI" {
		t.Errorf("Name = %q, want %q", info.Name, "VibeCloudAI")
	}
}

func TestParseProjectList_NotFound(t *testing.T) {
	jsonData := `[{"ref":"aaabbbcccdddeeefffgg","name":"Other","region":"eu-west-1","status":"ACTIVE_HEALTHY"}]`
	_, err := parseProjectList(jsonData, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent ref, got nil")
	}
}

func TestParseProjectList_PausedProject(t *testing.T) {
	jsonData := `[{"ref":"bwxvxzfzujkxnphedtom","name":"VibeCloudAI","region":"us-east-1","status":"INACTIVE"}]`
	info, err := parseProjectList(jsonData, "bwxvxzfzujkxnphedtom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Status != "INACTIVE" {
		t.Errorf("Status = %q, want %q", info.Status, "INACTIVE")
	}
	if !info.IsPaused() {
		t.Error("IsPaused() = false, want true")
	}
}

func TestParseProjectList_ComingUp(t *testing.T) {
	jsonData := `[{"ref":"bwxvxzfzujkxnphedtom","name":"VibeCloudAI","region":"us-east-1","status":"COMING_UP"}]`
	info, err := parseProjectList(jsonData, "bwxvxzfzujkxnphedtom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.IsStartingUp() {
		t.Error("IsStartingUp() = false, want true")
	}
}

func TestParseProjectList_InvalidJSON(t *testing.T) {
	_, err := parseProjectList("not json", "anything")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd cli && go test ./internal/supabase/ -v -run TestParseProjectList`
Expected: compilation error — `parseProjectList` doesn't exist.

- [ ] **Step 3: Write the implementation**

Create `cli/internal/supabase/project.go`:

```go
package supabase

import (
	"context"
	"encoding/json"
	"fmt"

	vexec "github.com/christopherdang/vibecloud/cli/internal/exec"
)

// ProjectInfo holds metadata about a Supabase project.
type ProjectInfo struct {
	Ref    string `json:"ref"`
	Name   string `json:"name"`
	Region string `json:"region"`
	Status string `json:"status"`
	DBHost string `json:"-"`
}

// projectJSON maps the JSON structure from `supabase projects list -o json`.
type projectJSON struct {
	Ref      string `json:"ref"`
	Name     string `json:"name"`
	Region   string `json:"region"`
	Status   string `json:"status"`
	Database struct {
		Host string `json:"host"`
	} `json:"database"`
}

// IsPaused returns true if the project is paused (INACTIVE).
func (p *ProjectInfo) IsPaused() bool {
	return p.Status == "INACTIVE"
}

// IsStartingUp returns true if the project is in the process of starting.
func (p *ProjectInfo) IsStartingUp() bool {
	return p.Status == "COMING_UP"
}

// IsHealthy returns true if the project is active and healthy.
func (p *ProjectInfo) IsHealthy() bool {
	return p.Status == "ACTIVE_HEALTHY"
}

// GetProjectInfo fetches project metadata by running
// `supabase projects list -o json` and finding the matching ref.
func GetProjectInfo(ctx context.Context, ref string) (*ProjectInfo, error) {
	stdout, _, err := vexec.RunCaptureAll(ctx, "supabase", "projects", "list", "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to list Supabase projects: %w", err)
	}
	return parseProjectList(stdout, ref)
}

// parseProjectList extracts a ProjectInfo from the JSON output of
// `supabase projects list -o json`. Exported for testing.
func parseProjectList(jsonData string, ref string) (*ProjectInfo, error) {
	var projects []projectJSON
	if err := json.Unmarshal([]byte(jsonData), &projects); err != nil {
		return nil, fmt.Errorf("failed to parse project list JSON: %w", err)
	}

	for _, p := range projects {
		if p.Ref == ref {
			return &ProjectInfo{
				Ref:    p.Ref,
				Name:   p.Name,
				Region: p.Region,
				Status: p.Status,
				DBHost: p.Database.Host,
			}, nil
		}
	}

	return nil, fmt.Errorf("project with ref %q not found", ref)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd cli && go test ./internal/supabase/ -v -run TestParseProjectList`
Expected: all 5 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add cli/internal/supabase/project.go cli/internal/supabase/project_test.go
git commit -m "feat(cli): add Supabase project info discovery from CLI"
```

---

### Task 3: Create `internal/supabase/pooler.go` — dynamic pooler discovery

**Files:**
- Create: `cli/internal/supabase/pooler.go`
- Test: `cli/internal/supabase/pooler_test.go`

- [ ] **Step 1: Write the failing tests**

Create `cli/internal/supabase/pooler_test.go`:

```go
package supabase

import (
	"context"
	"net"
	"testing"
)

func TestCandidateHosts(t *testing.T) {
	hosts := candidateHosts("us-east-1")
	if len(hosts) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(hosts))
	}
	if hosts[0] != "aws-0-us-east-1.pooler.supabase.com" {
		t.Errorf("hosts[0] = %q, want %q", hosts[0], "aws-0-us-east-1.pooler.supabase.com")
	}
	if hosts[1] != "aws-1-us-east-1.pooler.supabase.com" {
		t.Errorf("hosts[1] = %q, want %q", hosts[1], "aws-1-us-east-1.pooler.supabase.com")
	}
}

func TestCandidateHosts_EURegion(t *testing.T) {
	hosts := candidateHosts("eu-west-1")
	if hosts[0] != "aws-0-eu-west-1.pooler.supabase.com" {
		t.Errorf("hosts[0] = %q, want %q", hosts[0], "aws-0-eu-west-1.pooler.supabase.com")
	}
}

func TestDiscoverPooler_WithLocalListener(t *testing.T) {
	// Start a local TCP listener to simulate a pooler.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	ctx := context.Background()

	// Probe the local listener directly.
	result, err := probeHost(ctx, addr)
	if err != nil {
		t.Fatalf("probeHost() error: %v", err)
	}
	if !result {
		t.Error("probeHost() = false, want true for listening address")
	}
}

func TestDiscoverPooler_UnreachableHost(t *testing.T) {
	ctx := context.Background()
	// RFC 5737 TEST-NET address — guaranteed unreachable.
	result, err := probeHost(ctx, "192.0.2.1:6543")
	if err == nil && result {
		t.Error("probeHost() should fail for unreachable address")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd cli && go test ./internal/supabase/ -v -run "TestCandidateHosts|TestDiscoverPooler"`
Expected: compilation error — `candidateHosts` and `probeHost` don't exist.

- [ ] **Step 3: Write the implementation**

Create `cli/internal/supabase/pooler.go`:

```go
package supabase

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// PoolerResult holds the discovered pooler host and its reachability.
type PoolerResult struct {
	Host            string `json:"host"`
	TransactionPort int    `json:"transaction_port"` // 6543
	SessionPort     int    `json:"session_port"`     // 5432
	Reachable       bool   `json:"reachable"`
}

// DiscoverPooler probes candidate pooler hosts for the given region
// in parallel and returns the first that responds on port 6543.
func DiscoverPooler(ctx context.Context, region string) (*PoolerResult, error) {
	hosts := candidateHosts(region)

	type probeResult struct {
		host string
		ok   bool
	}

	ch := make(chan probeResult, len(hosts))
	var wg sync.WaitGroup

	for _, h := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			addr := fmt.Sprintf("%s:6543", host)
			ok, _ := probeHost(ctx, addr)
			ch <- probeResult{host: host, ok: ok}
		}(h)
	}

	// Close channel after all probes finish.
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Return the first successful probe.
	for r := range ch {
		if r.ok {
			return &PoolerResult{
				Host:            r.host,
				TransactionPort: 6543,
				SessionPort:     5432,
				Reachable:       true,
			}, nil
		}
	}

	return &PoolerResult{
		Reachable: false,
	}, fmt.Errorf("no pooler host reachable for region %s (tried: %v)", region, hosts)
}

// candidateHosts returns the pooler hostnames to probe for a given region.
func candidateHosts(region string) []string {
	return []string{
		fmt.Sprintf("aws-0-%s.pooler.supabase.com", region),
		fmt.Sprintf("aws-1-%s.pooler.supabase.com", region),
	}
}

// probeHost does a TCP dial to the given address with a 3-second timeout.
func probeHost(ctx context.Context, addr string) (bool, error) {
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false, err
	}
	conn.Close()
	return true, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd cli && go test ./internal/supabase/ -v -run "TestCandidateHosts|TestDiscoverPooler" -timeout 10s`
Expected: all 4 tests PASS. The unreachable test may take ~3s due to dial timeout.

- [ ] **Step 5: Commit**

```bash
git add cli/internal/supabase/pooler.go cli/internal/supabase/pooler_test.go
git commit -m "feat(cli): add dynamic Supabase pooler host discovery"
```

---

### Task 4: Update `cmd/doctor.go` — dynamic pooler check and unpause detection

**Files:**
- Modify: `cli/cmd/doctor.go`

This task replaces the hardcoded pooler host in `checkVercelSupabaseConnectivity()` with dynamic discovery, and adds project status checks for paused/starting projects.

- [ ] **Step 1: Update the import block in `doctor.go`**

Add the supabase package import. The updated import block:

```go
import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/christopherdang/vibecloud/cli/internal/config"
	vexec "github.com/christopherdang/vibecloud/cli/internal/exec"
	"github.com/christopherdang/vibecloud/cli/internal/output"
	supa "github.com/christopherdang/vibecloud/cli/internal/supabase"
	versionpkg "github.com/christopherdang/vibecloud/cli/internal/version"
	"github.com/spf13/cobra"
)
```

Note: The `"time"` import is removed — it's no longer needed since `probeHost` handles timeouts internally.

- [ ] **Step 2: Update the cross-provider connectivity check in `runDoctor()`**

Replace the block at lines 123-130 (the cross-provider check) with:

```go
	// Cross-provider compatibility checks.
	var connectivity map[string]interface{}
	if contains(projCfg.DetectedStack.Providers, "vercel") && contains(projCfg.DetectedStack.Providers, "supabase") {
		if ref := getSupabaseProjectRef(cwd); ref != "" {
			connectivity = checkVercelSupabaseConnectivity(ctx, ref, &warnings)
		}
	}
```

The only change: pass `ctx` to `checkVercelSupabaseConnectivity`.

- [ ] **Step 3: Replace `checkVercelSupabaseConnectivity()` function**

Replace the entire function (lines 250-285) with:

```go
// checkVercelSupabaseConnectivity checks for IPv6/pooler connectivity issues
// between Vercel Functions and Supabase Postgres.
func checkVercelSupabaseConnectivity(ctx context.Context, ref string, warnings *[]string) map[string]interface{} {
	result := map[string]interface{}{
		"ipv4_available":   false,
		"pooler_reachable": false,
	}

	// Check project status via `supabase projects list -o json`.
	info, infoErr := supa.GetProjectInfo(ctx, ref)
	if infoErr != nil {
		*warnings = append(*warnings, fmt.Sprintf("Could not fetch Supabase project info: %s. Pooler check skipped.", infoErr))
		result["recommendation"] = "Run 'vibecloud login --provider supabase' to authenticate, then re-run doctor."
		return result
	}

	result["project_status"] = info.Status
	result["region"] = info.Region

	if info.IsPaused() {
		*warnings = append(*warnings, "Supabase project is paused (INACTIVE). Run 'supabase projects unpause --project-ref "+ref+"' to resume. The connection pooler will not be available until the project is active.")
		result["recommendation"] = "Unpause the project first, then wait 5-15 minutes for the pooler to provision."
		return result
	}

	if info.IsStartingUp() {
		*warnings = append(*warnings, "Supabase project is starting up (COMING_UP). The connection pooler may take 5-15 minutes to re-provision after unpausing.")
		result["recommendation"] = "Wait for project status to become ACTIVE_HEALTHY, then re-run 'vibecloud doctor'."
		return result
	}

	// DNS lookup for direct Postgres — check IPv4 availability.
	addrs, err := net.LookupHost("db." + ref + ".supabase.co")
	if err == nil {
		for _, addr := range addrs {
			if strings.Contains(addr, ".") && !strings.Contains(addr, ":") {
				result["ipv4_available"] = true
				break
			}
		}
	}

	if !result["ipv4_available"].(bool) {
		*warnings = append(*warnings, "Supabase direct Postgres resolves to IPv6 only — Vercel Functions cannot connect via direct Postgres. Use the connection pooler or PostgREST (HTTP API) instead.")
	}

	// Dynamic pooler discovery.
	pooler, poolerErr := supa.DiscoverPooler(ctx, info.Region)
	if poolerErr != nil || !pooler.Reachable {
		*warnings = append(*warnings, fmt.Sprintf("Supabase connection pooler is not reachable in region %s (may still be provisioning — can take 5-15 minutes). Run 'vibecloud env sync' to sync PostgREST credentials as a fallback.", info.Region))
		result["recommendation"] = "Use PostgREST (vibecloud env sync) for Vercel+Supabase until pooler is ready."
	} else {
		result["pooler_reachable"] = true
		result["pooler_host"] = pooler.Host
		result["recommendation"] = fmt.Sprintf("Pooler reachable at %s. Run 'vibecloud env sync' to configure DATABASE_URL.", pooler.Host)
	}

	return result
}
```

- [ ] **Step 4: Verify it compiles**

Run: `cd cli && go build ./...`
Expected: compiles with no errors.

- [ ] **Step 5: Commit**

```bash
git add cli/cmd/doctor.go
git commit -m "fix(cli): replace hardcoded pooler host with dynamic discovery in doctor"
```

---

### Task 5: Update `cmd/env.go` — add DATABASE_URL to env sync

**Files:**
- Modify: `cli/cmd/env.go`

This task extends `runEnvSync()` to discover the pooler, prompt for the DB password, and sync `DATABASE_URL` to Vercel alongside the existing 3 env vars.

- [ ] **Step 1: Add the supabase import to `env.go`**

Update the import block:

```go
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
	supa "github.com/christopherdang/vibecloud/cli/internal/supabase"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)
```

- [ ] **Step 2: Add DATABASE_URL sync after the existing env var sync**

In `runEnvSync()`, replace lines 259-301 (from `supabaseURL := ...` through the closing `return nil` of the function) with the following. This replaces the entire second half of the function:

```go
	supabaseURL := fmt.Sprintf("https://%s.supabase.co", ref)

	// Discover pooler for DATABASE_URL.
	var databaseURL string
	info, infoErr := supa.GetProjectInfo(ctx, ref)
	if infoErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch project info: %s. Skipping DATABASE_URL.\n", infoErr)
	} else if info.IsPaused() {
		fmt.Fprintf(os.Stderr, "Warning: project is paused. Skipping DATABASE_URL. Unpause first: supabase projects unpause --project-ref %s\n", ref)
	} else if info.IsStartingUp() {
		fmt.Fprintf(os.Stderr, "Warning: project is starting up. Pooler may not be ready. Skipping DATABASE_URL.\n")
	} else {
		pooler, poolerErr := supa.DiscoverPooler(ctx, info.Region)
		if poolerErr != nil || !pooler.Reachable {
			fmt.Fprintf(os.Stderr, "Warning: pooler not reachable in region %s. Skipping DATABASE_URL. It may still be provisioning (5-15 min).\n", info.Region)
		} else {
			// Prompt for DB password.
			fmt.Fprintf(os.Stderr, "Enter your Supabase database password (for DATABASE_URL): ")
			var dbPassword string
			if term.IsTerminal(int(os.Stdin.Fd())) {
				raw, pwErr := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Fprintln(os.Stderr)
				if pwErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not read password: %s. Skipping DATABASE_URL.\n", pwErr)
				} else {
					dbPassword = string(raw)
				}
			} else {
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					dbPassword = scanner.Text()
				}
			}

			if dbPassword != "" {
				databaseURL = supa.BuildDatabaseURL(ref, pooler.Host, pooler.TransactionPort, dbPassword)
			}
		}
	}

	// Sync each key into Vercel.
	envVars := []struct {
		key   string
		value string
	}{
		{"SUPABASE_URL", supabaseURL},
		{"SUPABASE_ANON_KEY", anonKey},
		{"SUPABASE_SERVICE_ROLE_KEY", serviceRoleKey},
	}
	if databaseURL != "" {
		envVars = append(envVars, struct {
			key   string
			value string
		}{"DATABASE_URL", databaseURL})
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

	if databaseURL == "" {
		results["DATABASE_URL"] = "skipped (see warnings above)"
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
```

- [ ] **Step 3: Verify it compiles**

Run: `cd cli && go build ./...`
Expected: compiles with no errors.

- [ ] **Step 4: Commit**

```bash
git add cli/cmd/env.go
git commit -m "feat(cli): add DATABASE_URL construction to env sync"
```

---

### Task 6: Update `cmd/init.go` — CLAUDE.md pooler guidance

**Files:**
- Modify: `cli/cmd/init.go`

This task adds a "Supabase + Vercel connectivity" section to the generated CLAUDE.md when both providers are detected.

- [ ] **Step 1: Add pooler guidance section to `writeClaudeMD()`**

In `writeClaudeMD()`, after the main `vibeSection` string (after the closing of the `Providers: %s` line and before the `// Check if CLAUDE.md exists.` comment at line 318), add a conditional section. Insert after line 316 (``, strings.Join(stack.Providers, "/"), strings.Join(stack.Providers, ", "))`):

```go
	// Add Supabase + Vercel connectivity guidance if both providers are present.
	hasSupabase := false
	hasVercel := false
	for _, p := range stack.Providers {
		if p == "supabase" {
			hasSupabase = true
		}
		if p == "vercel" {
			hasVercel = true
		}
	}
	if hasSupabase && hasVercel {
		vibeSection += `
### Supabase + Vercel connectivity
Vercel Functions cannot connect to Supabase Postgres directly (IPv6-only).
You MUST use the connection pooler (Supavisor):
- Transaction mode: port 6543 (use for Vercel Functions — short-lived, no prepared statements)
- Session mode: port 5432 (use for migrations with pg_dump/pg_restore)
- Connection format: ` + "`postgresql://postgres.[REF]:[PASSWORD]@[POOLER_HOST]:[PORT]/postgres`" + `
- Never use the direct connection (` + "`db.[REF].supabase.co`" + `) from Vercel Functions.
Run ` + "`vibecloud env sync`" + ` to auto-configure DATABASE_URL with the correct pooler endpoint.
Run ` + "`vibecloud db status`" + ` to check pooler reachability and project health.
`
	}
```

- [ ] **Step 2: Also update the "Deployment (VibeCloud)" section to include db commands**

In the `vibeSection` string, add the `db` commands to the command list. After the line:
```
- ` + "`vibecloud env rm <KEY>`" + ` — remove an environment variable
```

Add:
```
- ` + "`vibecloud db push`" + ` — push migrations to remote Supabase database
- ` + "`vibecloud db status`" + ` — show database connection info and pooler status
```

- [ ] **Step 3: Verify it compiles**

Run: `cd cli && go build ./...`
Expected: compiles with no errors.

- [ ] **Step 4: Commit**

```bash
git add cli/cmd/init.go
git commit -m "feat(cli): add pooler guidance to CLAUDE.md when Supabase+Vercel detected"
```

---

### Task 7: Create `cmd/db.go` — `vibecloud db push` and `vibecloud db status`

**Files:**
- Create: `cli/cmd/db.go`

- [ ] **Step 1: Create the `db` command with `push` and `status` subcommands**

Create `cli/cmd/db.go`:

```go
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
```

- [ ] **Step 2: Verify it compiles**

Run: `cd cli && go build ./...`
Expected: compiles with no errors.

- [ ] **Step 3: Verify the commands are registered**

Run: `cd cli && go run . db --help`
Expected output includes `push` and `status` subcommands.

Run: `cd cli && go run . db status --help`
Expected output shows usage for the status command.

- [ ] **Step 4: Commit**

```bash
git add cli/cmd/db.go
git commit -m "feat(cli): add vibecloud db push and db status commands"
```

---

### Task 8: Update project CLAUDE.md to document new commands

**Files:**
- Modify: `CLAUDE.md` (project root)

The project's own `CLAUDE.md` should reflect the new `db` commands.

- [ ] **Step 1: Add db commands to the Commands section**

In the root `CLAUDE.md`, in the `### Commands` section, add after the `vibecloud env rm` line (or after `vibecloud env sync`):

```markdown
- `vibecloud db push` — push migrations to remote Supabase database
- `vibecloud db status` — show database connection info and pooler status
```

- [ ] **Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add db push and db status to CLAUDE.md commands list"
```

---

### Task 9: Manual verification

This task verifies the full integration against the live VibeCloudAI project.

- [ ] **Step 1: Run unit tests**

Run: `cd cli && go test ./internal/supabase/ -v`
Expected: all tests pass.

- [ ] **Step 2: Build the CLI**

Run: `cd cli && go build -ldflags="-s -w" -o vibecloud .`
Expected: binary built successfully.

- [ ] **Step 3: Test `vibecloud doctor` with dynamic pooler**

Run: `cd cli && ./vibecloud doctor`
Expected: output includes `pooler_host` field with the correct host (likely `aws-0-us-east-1` or `aws-1-us-east-1`), not a hardcoded value. The `pooler_reachable` field should be `true`.

- [ ] **Step 4: Test `vibecloud db status`**

Run: `cd cli && ./vibecloud db status`
Expected: output includes project name, region, status, pooler host, and API key availability.

- [ ] **Step 5: Test `vibecloud db push`**

Run: `cd cli && ./vibecloud db push`
Expected: either succeeds (pushes migrations) or reports no pending migrations. Should not crash or show hardcoded host errors.

- [ ] **Step 6: Verify CLAUDE.md generation includes pooler guidance**

In a temp directory, simulate `vibecloud init` for a project with both Supabase and Vercel:
Run: `cd cli && ./vibecloud init` (from this repo which has both providers)
Check: the generated/updated `CLAUDE.md` should contain the "### Supabase + Vercel connectivity" section.
