# Supabase Intelligence Gaps — Design Spec

**Date:** 2026-05-08
**Scope:** vibecloud CLI (`cli/`)
**Issues addressed:** feedback_cli_supabase_gaps.md items #0–#5

## Problem

The vibecloud CLI has six gaps in its Supabase integration that cause users to bypass the CLI and debug connectivity issues manually:

1. `init` doesn't warn agents about the pooler requirement when both Supabase and Vercel are detected
2. `doctor` hardcodes the pooler host to `aws-0-us-east-1.pooler.supabase.com`, which is wrong for projects on `aws-1` or other regions
3. `env sync` doesn't construct or sync `DATABASE_URL`
4. No detection or warning for paused/recently-unpaused projects where the pooler hasn't re-provisioned
5. Claude bypasses the CLI for `supabase db push` and `vercel env add` because vibecloud lacks wrappers
6. The Supabase MCP was needed for operations the CLI should handle natively

## Design

### New package: `internal/supabase/`

Centralizes Supabase project discovery so `doctor`, `init`, `env sync`, and new commands share one source of truth.

#### `project.go` — Project info from CLI

```go
type ProjectInfo struct {
    Ref       string // e.g. "bwxvxzfzujkxnphedtom"
    Name      string
    Region    string // e.g. "us-east-1"
    Status    string // e.g. "ACTIVE_HEALTHY", "COMING_UP", "INACTIVE"
    DBHost    string // e.g. "db.{ref}.supabase.co"
}

// GetProjectInfo runs `supabase projects list -o json`, finds the
// project matching ref, and returns its region and status.
func GetProjectInfo(ctx context.Context, ref string) (*ProjectInfo, error)
```

Source: `supabase projects list -o json` — stable, already used by the CLI, returns `region` and `status` fields.

#### `pooler.go` — Dynamic pooler host discovery

```go
type PoolerResult struct {
    Host            string // e.g. "aws-1-us-east-1.pooler.supabase.com"
    TransactionPort int    // 6543
    SessionPort     int    // 5432
    Reachable       bool
}

// DiscoverPooler does parallel TCP dials to candidate pooler hosts
// for the given region and returns the first that responds.
func DiscoverPooler(ctx context.Context, region string) (*PoolerResult, error)
```

**Candidate hosts:** `aws-0-{region}.pooler.supabase.com` and `aws-1-{region}.pooler.supabase.com`. Dial both in parallel on port 6543 with a 5-second timeout. Return the first to connect. If neither connects, return `Reachable: false` with an appropriate error message.

**Why TCP probe instead of a lookup table:** Supabase doesn't expose the cloud prefix (`aws-0` vs `aws-1`) through any API. A probe is resilient to future host additions. The cost is ~2s worst-case when neither responds.

#### `connstring.go` — DATABASE_URL construction

```go
// BuildDatabaseURL constructs the pooler connection string.
// Password must be provided by the caller (prompted from user).
func BuildDatabaseURL(ref string, poolerHost string, port int, password string) string
```

Format: `postgresql://postgres.{ref}:{password}@{poolerHost}:{port}/postgres`

Default port: 6543 (transaction mode). Transaction mode is correct for Vercel Functions (short-lived, no prepared statements).

### Changes to existing commands

#### `cmd/init.go` — CLAUDE.md pooler guidance (#0)

When `writeClaudeMD()` detects both `supabase` and `vercel` in providers, append this section:

```markdown
### Supabase + Vercel connectivity
Vercel Functions cannot connect to Supabase Postgres directly (IPv6-only).
You MUST use the connection pooler (Supavisor):
- Transaction mode: port 6543 (use for Vercel Functions)
- Session mode: port 5432 (use for migrations)
- Connection format: `postgresql://postgres.[REF]:[PASSWORD]@[POOLER_HOST]:[PORT]/postgres`
- Never use the direct connection (`db.[REF].supabase.co`) from Vercel.
Run `vibecloud env sync` to auto-configure DATABASE_URL with the correct pooler endpoint.
```

#### `cmd/doctor.go` — Dynamic pooler check and unpause detection (#1, #3)

Replace `checkVercelSupabaseConnectivity()`:

1. Call `supabase.GetProjectInfo(ctx, ref)` to get region and status
2. If status is `INACTIVE`: warn "project is paused — run `supabase projects unpause` first"
3. If status is `COMING_UP`: warn "project is starting up — pooler may take 5-15 minutes to re-provision"
4. Call `supabase.DiscoverPooler(ctx, region)` instead of hardcoded `aws-0-us-east-1`
5. Report discovered pooler host in output for debugging

#### `cmd/env.go` — DATABASE_URL in env sync (#2)

Extend `runEnvSync()`:

1. After fetching API keys, call `supabase.GetProjectInfo(ctx, ref)` for region
2. Call `supabase.DiscoverPooler(ctx, region)` for pooler host
3. If pooler is reachable, prompt user for their Supabase DB password (secure terminal input, same pattern as `env add`)
4. Call `supabase.BuildDatabaseURL(ref, poolerHost, 6543, password)`
5. Sync `DATABASE_URL` to Vercel alongside the existing 3 env vars
6. If pooler is not reachable, skip DATABASE_URL with a warning and sync the other 3 vars

The password prompt uses the existing `readSecureInput()` from env.go. If `--yes` flag is set (agent mode), read from stdin.

#### New command: `cmd/db.go` — `vibecloud db` (#4, #5)

Two subcommands that wrap the Supabase CLI:

**`vibecloud db push`**
- Wraps `supabase db push --linked`
- Checks project status first (warns if paused)
- Reports success/failure in standard JSON output format

**`vibecloud db status`**
- Calls `supabase.GetProjectInfo()` for project status
- Calls `supabase.DiscoverPooler()` for pooler reachability
- Calls `supabase projects api-keys --project-ref {ref}` for key status
- Reports: project status, region, pooler host, pooler reachability, whether DATABASE_URL is set on Vercel
- Output in standard JSON format

### What this does NOT include

- **Storing the DB password:** The password is prompted each time `env sync` is run. No credential storage.
- **`vibecloud db query`:** Direct SQL execution stays with `supabase db query`. Adding a query wrapper has security implications beyond the scope of these fixes.
- **Migration management:** `db push` wraps the existing Supabase command. No custom migration logic.
- **Multi-region project creation:** `init` still defaults to `us-east-1` for new projects. Changing this is a separate UX decision.

### File inventory

| Action | File | What changes |
|--------|------|-------------|
| Create | `internal/supabase/project.go` | `GetProjectInfo()` |
| Create | `internal/supabase/pooler.go` | `DiscoverPooler()` |
| Create | `internal/supabase/connstring.go` | `BuildDatabaseURL()` |
| Create | `cmd/db.go` | `vibecloud db push`, `vibecloud db status` |
| Edit | `cmd/init.go` | Add pooler guidance to CLAUDE.md generation |
| Edit | `cmd/doctor.go` | Replace hardcoded pooler with dynamic discovery |
| Edit | `cmd/env.go` | Add DATABASE_URL to env sync |
| Edit | `cmd/root.go` | Register `db` subcommand |

### Testing

- `internal/supabase/connstring.go`: Unit test for URL format with various refs, passwords (including special chars that need URL-encoding), ports
- `internal/supabase/pooler.go`: Unit test with mock TCP listener
- `cmd/db.go`: Integration test via existing e2e framework
- Manual verification: run `vibecloud doctor` and `vibecloud env sync` against the VibeCloudAI project to confirm correct pooler host discovery
