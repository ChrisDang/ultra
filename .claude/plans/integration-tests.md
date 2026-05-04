# Integration Tests Plan

Three test suites, each building on the last. The end goal is **Part C**: a single E2E test an agent can run to verify the product works — CLI login through live URL, with real OAuth tokens against live Vercel, Railway, and Supabase accounts.

---

## Part A: API Integration Tests (Go test server + real Postgres)

### Goal
Test the full API request lifecycle — auth flows, CRUD operations, provider management — against a real PostgreSQL database with the actual chi router. No mocks. This validates that the control plane works before we layer the CLI and providers on top.

### Location
`api/integration_test.go` — uses `//go:build integration` tag so it doesn't run with `go test ./...` by default.

### Setup (TestMain)
1. Start a PostgreSQL instance (Docker on port 5433, or use `DATABASE_URL` env var)
2. Apply schema from `supabase/migrations/00001_initial_schema.sql`
3. Set `JWT_SIGNING_SECRET` and `ENCRYPTION_KEY` to test values
4. Start the full chi router via `httptest.NewServer`
5. After all tests: `TRUNCATE ... CASCADE` all tables, close DB, shutdown server

### Test Groups

#### 1. Auth Flow (14 tests)

| Test | Method | Endpoint | Validates |
|------|--------|----------|-----------|
| `TestAuthSignup` | POST | `/api/auth/signup` | 201, access_token, refresh_token, user with ID/email. Password hash NOT in response. |
| `TestAuthSignupDuplicateEmail` | POST | `/api/auth/signup` | 409 when email exists |
| `TestAuthSignupWeakPassword` | POST | `/api/auth/signup` | 400 for password < 8 chars |
| `TestAuthSignupMissingFields` | POST | `/api/auth/signup` | 400 for empty email or password |
| `TestAuthLogin` | POST | `/api/auth/login` | 200, valid JWT with correct `sub` claim |
| `TestAuthLoginWrongPassword` | POST | `/api/auth/login` | 401 |
| `TestAuthLoginNonexistentUser` | POST | `/api/auth/login` | 401 (same error — no user enumeration) |
| `TestAuthMe` | GET | `/api/auth/me` | Returns user ID and email |
| `TestAuthMeNoToken` | GET | `/api/auth/me` | 401 |
| `TestAuthMeInvalidToken` | GET | `/api/auth/me` | 401 |
| `TestAuthRefresh` | POST | `/api/auth/refresh` | New token pair. Old refresh token invalidated (rotation). New access token works. |
| `TestAuthRefreshInvalidToken` | POST | `/api/auth/refresh` | 401 |
| `TestAuthRefreshReuse` | POST | `/api/auth/refresh` | Same refresh token twice → 401 (rotation enforcement) |
| `TestAuthLogout` | POST | `/api/auth/logout` | 204. Refresh token invalidated. |

#### 2. Projects CRUD (5 tests)

| Test | Method | Endpoint | Validates |
|------|--------|----------|-----------|
| `TestProjectCreate` | POST | `/api/projects` | 201 with project ID, name, framework |
| `TestProjectList` | GET | `/api/projects` | Array containing created project |
| `TestProjectGet` | GET | `/api/projects/{id}` | Returns project by ID |
| `TestProjectGetNotFound` | GET | `/api/projects/{bad-id}` | 404 |
| `TestProjectIsolation` | GET | `/api/projects` | User B cannot see User A's projects |

#### 3. Provider Links (5 tests)

| Test | Method | Endpoint | Validates |
|------|--------|----------|-----------|
| `TestProviderListEmpty` | GET | `/api/providers` | Empty array for new user |
| `TestProviderConnectSupabase` | POST | `/api/providers/supabase/connect` | 200, shows as connected |
| `TestProviderListAfterConnect` | GET | `/api/providers` | Array with supabase connected |
| `TestProviderDisconnect` | DELETE | `/api/providers/supabase` | 200, no longer in list |
| `TestProviderTokenNotExposed` | GET | `/api/providers` | Encrypted token never in JSON |

#### 4. Logs (2 tests)

| Test | Method | Endpoint | Validates |
|------|--------|----------|-----------|
| `TestLogsEmpty` | GET | `/api/projects/{id}/logs` | Empty array |
| `TestLogsWithFilters` | GET | `/api/projects/{id}/logs?provider=...&since=...` | Filters respected |

### Helpers

```go
func doRequest(method, path string, body interface{}, token string) (*http.Response, error)
func parseJSON[T any](resp *http.Response) (T, error)
func signupAndLogin(email, password string) (accessToken, refreshToken string, err error)
```

### Running

```bash
# Start test Postgres
docker run --rm -d -p 5433:5432 -e POSTGRES_PASSWORD=test -e POSTGRES_DB=vibecloud_test --name vc-test-db postgres:16

# Run
cd api
DATABASE_URL="postgres://postgres:test@localhost:5433/vibecloud_test" \
JWT_SIGNING_SECRET="test-secret-key-for-integration-tests" \
ENCRYPTION_KEY="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" \
go test -tags=integration -v -count=1 ./...
```

---

## Part B: Provider Integration Tests (real Vercel, Railway, Supabase APIs)

### Goal
Exercise the real provider APIs to validate our adapter code works against live services. Creates real projects/services, deploys, checks logs, then cleans up everything.

### Location
`tests/integration/` — existing directory. Needs to be updated:
- **Replace Render with Railway** (`test_render.go` → `test_railway.go`)
- Keep `test_vercel.go` (already written and working)
- Keep `test_supabase.go` (already written and working)

### Token Acquisition
Interactive — the test runner (`main.go`) prompts for tokens:
- **Vercel**: OAuth flow or paste a personal access token from https://vercel.com/account/tokens
- **Railway**: Paste an API token from https://railway.com/account/tokens (Railway uses token auth, not OAuth)
- **Supabase**: Paste a PAT from https://supabase.com/dashboard/account/tokens

Tokens are cached in `test_tokens.json` (gitignored) for re-runs.

### Changes to `main.go`
- Rename `tokens.Render` → `tokens.Railway`
- Update the Render prompt to say "Railway" with Railway's token URL
- Call `testRailway()` instead of `testRender()`

### Vercel Tests (keep as-is, 10 tests)

Already implemented in `test_vercel.go`:
1. List projects (API connectivity smoke test)
2. Create project with `nextjs` framework
3. Get project details
4. Set environment variable
5. List environment variables
6. Create deployment (static HTML)
7. Poll deployment status until READY
8. Get deployment events/logs
9. List deployments for project
10. Get project domains

Cleanup: delete the created project.

### Railway Tests (new — replaces `test_render.go`, 10 tests)

Railway uses a **GraphQL API** at `https://backboard.railway.com/graphql/v2`. Auth via `Authorization: Bearer <token>`.

| # | Test | GraphQL Mutation/Query | Validates |
|---|------|----------------------|-----------|
| 1 | API connectivity | `query { me { id email } }` | Token is valid, returns user info |
| 2 | Create project | `mutation { projectCreate(input: {name: "..."}) { id name } }` | Returns project ID and name |
| 3 | Get project details | `query { project(id: "...") { id name description } }` | Returns the created project |
| 4 | Create service | `mutation { serviceCreate(input: {projectId: "...", name: "...", source: {image: "nginx:alpine"}}) { id name } }` | Creates a service from a Docker image |
| 5 | Set env variable | `mutation { variableUpsert(input: {projectId: "...", serviceId: "...", environmentId: "...", name: "VIBECLOUD_TEST", value: "integration"}) }` | Variable is set |
| 6 | List variables | `query { variables(projectId: "...", serviceId: "...", environmentId: "...") { ...fields } }` | Returns the variable we set |
| 7 | Get environments | `query { project(id: "...") { environments { edges { node { id name } } } } }` | Returns at least one environment (production) |
| 8 | Trigger redeploy | `mutation { serviceInstanceRedeploy(serviceId: "...", environmentId: "...") }` | Triggers a deployment |
| 9 | Get deployments | `query { deployments(input: {projectId: "..."}) { edges { node { id status } } } }` | Returns deployment list |
| 10 | Get deployment logs | `query { deploymentLogs(deploymentId: "...", limit: 20) { ...fields } }` | Returns log entries |

Cleanup: `mutation { projectDelete(id: "...") }` — deleting the project cascades to services.

### Supabase Tests (keep as-is, 11 tests)

Already implemented in `test_supabase.go`:
1. List organizations
2. List projects
3. Create project (wait for it to become `ACTIVE_HEALTHY`)
4. Poll project status
5. Get API keys
6. Execute SQL (create table)
7. Insert data via SQL
8. Query data via SQL
9. Drop table via SQL
10. Get PostgREST config

Cleanup: delete the created project.

### Test Log Output
All tests log to both stdout and `integration_test_run.txt` using the existing `TestLogger` (logger.go). Each test is a PASS/FAIL line with details.

### Running

```bash
cd tests/integration

# Build and run (interactive — prompts for tokens)
go run .

# Or with cached tokens
go run . # and answer "y" to reuse existing tokens
```

### Key Implementation Notes for Railway

- Railway's GraphQL endpoint: `https://backboard.railway.com/graphql/v2`
- Auth header: `Authorization: Bearer <token>`
- All mutations/queries use a single HTTP POST with `{"query": "...", "variables": {...}}`
- The existing `api/providers/railway/railway.go` has a `doGraphQL` helper — reuse the same pattern in tests
- Railway projects auto-create a "production" environment — query it to get the `environmentId` for variable/deploy operations
- Use `nginx:alpine` as the service source image (fast to deploy, no build step)
- Railway tokens come from https://railway.com/account/tokens (no OAuth needed for testing)

---

## Part C: End-to-End Smoke Test (the agent-runnable test)

### Goal
A single test that an AI agent (or human) can run to answer: **"Does Vibe Cloud work?"** It exercises the full product flow — OAuth into real provider accounts, secure token storage, project init, deploy, live URL verification, status, logs — against live Vercel, Railway, and Supabase infrastructure.

### Location
`tests/e2e/` — new directory.
- `main.go` — test runner and orchestrator
- `credentials.go` — encrypted credential store (OAuth token vault)
- `oauth.go` — OAuth flow handlers for each provider
- `verify.go` — provider setup verification (API health checks)
- `pipeline.go` — the sequential E2E test steps
- `fixtures/` — sample projects to deploy
- `fixtures/static-html/` — minimal static site (fastest deploy)
- `fixtures/nextjs-hello/` — minimal Next.js app (tests framework detection)

### Prerequisites
The test assumes:
1. The API server is running (local or remote — configured via `VIBECLOUD_API_URL`, defaults to `http://localhost:8080`)
2. A PostgreSQL database is available (the API server's responsibility)
3. OAuth client credentials for Vercel and Railway are available (env vars), or the user has PATs to paste

The test runner can optionally **start everything itself** with `--self-contained`:
1. Start Postgres via Docker (port 5433)
2. Apply migrations
3. Build and start the API server as a subprocess
4. Build the CLI binary
5. Run the tests
6. Tear down everything

---

### Credential Store (encrypted token vault)

Tokens are **never stored in plaintext**. The E2E test has its own credential store that mirrors the production pattern (AES-256-GCM, same as `api/crypto/encrypt.go`).

#### Location
`~/.vibecloud/test-credentials.enc` — a single encrypted file.

#### How it works

1. **First run**: No credential file exists. The test runner generates a random 256-bit encryption key and stores it in the system keychain via `go-keyring` (macOS Keychain, Linux secret-service, Windows Credential Manager). The keychain entry is `vibecloud-test / encryption-key`.

2. **Token storage**: After each OAuth flow completes, the token is added to an in-memory map, then the entire map is serialized to JSON, encrypted with AES-256-GCM using the keychain key, and written to `~/.vibecloud/test-credentials.enc`.

3. **Subsequent runs**: The test runner reads the encryption key from the keychain, decrypts `test-credentials.enc`, and loads the tokens. It then **verifies each token is still valid** (see Provider Verification below) before proceeding.

4. **Key rotation**: `--rotate-key` generates a new key, re-encrypts all tokens, and stores the new key in the keychain.

5. **Wipe**: `--wipe-credentials` deletes `test-credentials.enc` and the keychain entry.

#### Credential file format (before encryption)

```json
{
  "version": 1,
  "created_at": "2026-04-12T10:00:00Z",
  "updated_at": "2026-04-12T10:00:00Z",
  "providers": {
    "vercel": {
      "access_token": "...",
      "team_id": "team_...",
      "obtained_via": "oauth",
      "obtained_at": "2026-04-12T10:00:00Z"
    },
    "railway": {
      "access_token": "...",
      "obtained_via": "pat",
      "obtained_at": "2026-04-12T10:00:00Z"
    },
    "supabase": {
      "access_token": "...",
      "obtained_via": "pat",
      "obtained_at": "2026-04-12T10:00:00Z"
    }
  }
}
```

#### Why not env vars?
Env vars work for CI, but this test is designed for agents running on a developer's machine across multiple sessions. Env vars are ephemeral — you lose them when the shell closes. The encrypted credential store persists across sessions so an agent doesn't need to re-auth every time.

Env vars are still supported as an **override** (see Token Acquisition below) for CI or one-off runs.

---

### Token Acquisition (OAuth-first, PAT-fallback)

The test runner acquires tokens through a multi-step process. When an agent runs this test, it tells the user to open a URL; the test process handles the rest.

#### Flow for each provider

```
┌──────────────────────────────────────────────────────────────┐
│  Token Acquisition for {provider}                            │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Check env var override                                   │
│     VERCEL_TOKEN / RAILWAY_TOKEN / SUPABASE_TOKEN set?       │
│     → Use it directly, skip to verification                  │
│                                                              │
│  2. Check credential store                                   │
│     Token in ~/.vibecloud/test-credentials.enc?              │
│     → Decrypt, verify still valid (API call), use if good    │
│     → If expired/revoked, continue to step 3                 │
│                                                              │
│  3. Acquire new token                                        │
│     ┌─ Vercel: OAuth flow (browser)                          │
│     │  - Start local callback server on random port          │
│     │  - Print URL for user/agent to open                    │
│     │  - Wait for callback with auth code                    │
│     │  - Exchange code for access_token                      │
│     │                                                        │
│     ├─ Railway: OAuth flow (browser)                         │
│     │  - Same pattern as Vercel                              │
│     │  - Uses Railway's OAuth endpoint                       │
│     │                                                        │
│     └─ Supabase: PAT (manual entry)                          │
│        - Print URL to token page                             │
│        - Read token from stdin                               │
│        (Supabase doesn't support third-party OAuth)          │
│                                                              │
│  4. Verify the token works (see Provider Verification)       │
│                                                              │
│  5. Encrypt and save to credential store                     │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

#### Agent interaction pattern

When an agent (like Claude) runs this test and a new token is needed, the test outputs a structured message:

```json
{
  "action_required": "oauth",
  "provider": "vercel",
  "url": "https://vercel.com/integrations/oauth/authorize?client_id=...&redirect_uri=http://127.0.0.1:54321/callback&response_type=code",
  "instruction": "Open this URL in a browser to authenticate with Vercel. The test will continue automatically after you authenticate.",
  "timeout_seconds": 300
}
```

The agent can relay this to the user: "Please open this URL to connect your Vercel account." The local callback server catches the redirect, exchanges the code, and the test continues.

For Supabase (PAT only):
```json
{
  "action_required": "paste_token",
  "provider": "supabase",
  "url": "https://supabase.com/dashboard/account/tokens",
  "instruction": "Create a personal access token at this URL and paste it here.",
  "timeout_seconds": 300
}
```

#### OAuth client credentials

The OAuth flows require client ID and secret for Vercel and Railway. These come from env vars:
- `VERCEL_CLIENT_ID`, `VERCEL_CLIENT_SECRET`
- `RAILWAY_CLIENT_ID`, `RAILWAY_CLIENT_SECRET`

These are the **same credentials** the API server uses (from the Vercel integration and Railway OAuth app). They can be shared because the redirect_uri for the test is `http://127.0.0.1:{port}/callback` — a localhost URL that's authorized in both OAuth app configurations.

If OAuth client credentials aren't set, the test falls back to PAT entry for that provider (same as Supabase).

---

### Provider Verification (setup validation)

After acquiring each token, the test verifies the provider account is properly set up for deployments. This catches misconfigurations before the deploy step.

#### Vercel Verification

| Check | API Call | Pass Condition |
|-------|----------|---------------|
| Token valid | `GET https://api.vercel.com/v2/user` | 200 with user ID |
| Can create projects | Check user's plan allows project creation | Not on a paused/deactivated account |
| Team access (if team token) | `GET https://api.vercel.com/v2/teams/{team_id}` | 200 with team details |

Output on success:
```json
{
  "provider": "vercel",
  "verified": true,
  "account": "user@example.com",
  "team": "my-team",
  "plan": "hobby"
}
```

#### Railway Verification

| Check | API Call | Pass Condition |
|-------|----------|---------------|
| Token valid | `query { me { id email } }` | Returns user ID and email |
| Can create projects | `query { me { projects { edges { node { id } } } } }` | Query succeeds (no permission error) |
| Has available resources | Check user isn't over project/service limits | No `RESOURCE_LIMIT` error |

Output on success:
```json
{
  "provider": "railway",
  "verified": true,
  "account": "user@example.com",
  "user_id": "..."
}
```

#### Supabase Verification

| Check | API Call | Pass Condition |
|-------|----------|---------------|
| Token valid | `GET https://api.supabase.com/v1/projects` | 200 (even if empty array) |
| Has an organization | `GET https://api.supabase.com/v1/organizations` | At least one org returned |
| Can create projects | Check org allows new projects | Org is on a plan that permits it |

Output on success:
```json
{
  "provider": "supabase",
  "verified": true,
  "account": "user@example.com",
  "organizations": ["my-org"],
  "plan": "free"
}
```

#### What happens on failure

If a provider fails verification, the test prints a diagnostic and asks whether to continue without that provider:

```json
{
  "provider": "vercel",
  "verified": false,
  "error": "token expired (401 from Vercel API)",
  "claude_instructions": "The Vercel token is expired. Run the test with --reauth=vercel to re-authenticate, or set VERCEL_TOKEN env var with a fresh token."
}
```

The test requires **at least Vercel** to be verified (it's the frontend deploy target). Railway and Supabase failures downgrade the test to a partial run.

---

### E2E Test Pipeline

After tokens are acquired and verified, the test runs the full product flow. Each step depends on the previous one. If any step fails, the test stops, logs the failure, cleans up, and exits non-zero.

```
┌─────────────────────────────────────────────────────────────┐
│  E2E Smoke Test Pipeline                                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Health Check                                            │
│     GET {api_url}/health → 200                              │
│                                                             │
│  2. Signup + Login (via API, not browser OAuth)              │
│     POST /api/auth/signup → access_token                    │
│     Write token to temp ~/.vibecloud/config.json            │
│                                                             │
│  3. Connect Providers (via API)                             │
│     For each verified provider:                             │
│       POST /api/providers/{provider}/connect/token           │
│       with decrypted token from credential store            │
│     Assert: GET /api/providers shows all connected          │
│                                                             │
│  4. CLI Init                                                │
│     Run `vibecloud init` in fixtures/static-html/           │
│     Assert: .vibecloud.json created with project_id         │
│     Assert: exit code 0                                     │
│     Assert: stdout is valid JSON with success=true          │
│                                                             │
│  5. CLI Deploy                                              │
│     Run `vibecloud deploy`                                  │
│     Assert: exit code 0                                     │
│     Assert: stdout JSON contains deployment_id              │
│     Assert: stdout JSON contains url (live URL)             │
│     Assert: status is "ready" (CLI polls internally)        │
│     Timeout: 120 seconds                                    │
│                                                             │
│  6. Verify Live URL                                         │
│     HTTP GET the returned URL                               │
│     Assert: status 200                                      │
│     Assert: body contains "Hello from Vibe Cloud"           │
│     Retries: 3 attempts, 5s apart (propagation delay)       │
│                                                             │
│  7. CLI Status                                              │
│     Run `vibecloud status`                                  │
│     Assert: exit code 0                                     │
│     Assert: stdout JSON shows project with deployment       │
│                                                             │
│  8. CLI Logs                                                │
│     Run `vibecloud logs`                                    │
│     Assert: exit code 0                                     │
│     Assert: stdout is valid JSON (may be empty array)       │
│                                                             │
│  9. Cleanup                                                 │
│     Delete deployment/project via API                       │
│     Delete provider project (Vercel/Railway) via API        │
│     Remove temp config and .vibecloud.json                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Step Details

#### Step 2: Signup + Login
Create a test user with a random email (`e2e-{timestamp}@test.vibecloud.dev`) via the API. Write the returned tokens to a temporary config directory so the CLI picks them up. Set `VIBECLOUD_CONFIG_DIR` env var to point the CLI at the temp config.

This avoids the browser-based OAuth login flow entirely, which can't be automated by an agent.

#### Step 3: Connect Providers
Uses the `POST /api/providers/{provider}/connect/token` endpoint to inject the real OAuth/PAT tokens (decrypted from the credential store) into the test user's provider links. The API encrypts them at rest with AES-256-GCM, same as the production OAuth callback flow.

**New API endpoint**: `POST /api/providers/{provider}/connect/token`
- Accepts `{ "access_token": "..." }` in the request body
- Only enabled when `ALLOW_DIRECT_TOKEN_CONNECT=true` env var is set (never in production)
- Reuses the exact same `crypto.Encrypt()` + `UpsertProviderLink()` path as the OAuth callbacks
- Returns 200 with `{ "provider": "vercel", "connected": true }`

Note: Supabase already has this pattern — `HandleSupabaseConnect` accepts `{ "access_token": "..." }` directly. This endpoint generalizes it for Vercel and Railway behind a feature flag.

#### Step 5: CLI Deploy
The CLI's `deploy` command already polls for completion and outputs structured JSON. The test just runs it as a subprocess and parses stdout. Timeout is 120s to account for provider cold starts.

#### Step 6: Verify Live URL
Simple HTTP GET with retries (3 attempts, 5s apart) to handle propagation delay. The fixture app returns known content so we can assert on the body.

### Fixtures

#### `fixtures/static-html/`
```
index.html          — "<h1>Hello from Vibe Cloud</h1>"
```
Minimal. Deploys to Vercel as a static site. No build step, fastest possible deploy (~10s).

#### `fixtures/nextjs-hello/` (optional, for deeper validation)
```
package.json        — next, react, react-dom
pages/index.tsx     — returns "Hello from Vibe Cloud"
next.config.js      — minimal config
```
Used when `--fixture=nextjs` is passed. Tests the framework detection and build pipeline.

### CLI Invocation Pattern

All CLI commands are run as subprocesses with:
```go
cmd := exec.Command(cliBinary, args...)
cmd.Env = append(os.Environ(),
    "VIBECLOUD_API_URL="+apiURL,
    "VIBECLOUD_CONFIG_DIR="+tempConfigDir,
)
cmd.Dir = fixtureDir
stdout, stderr, exitCode := run(cmd)
```

The test parses stdout as JSON using the same `output.Response` struct the CLI uses. This validates the full contract: CLI → API → Provider → live URL.

### Output Format

The test runner outputs structured JSON so an agent can parse results:

```json
{
  "suite": "e2e",
  "passed": true,
  "providers": {
    "vercel": { "verified": true, "account": "user@example.com" },
    "railway": { "verified": true, "account": "user@example.com" },
    "supabase": { "verified": true, "account": "user@example.com" }
  },
  "steps": [
    { "name": "health_check", "passed": true, "duration_ms": 45 },
    { "name": "signup_login", "passed": true, "duration_ms": 312 },
    { "name": "connect_providers", "passed": true, "duration_ms": 598, "providers_connected": ["vercel", "railway", "supabase"] },
    { "name": "cli_init", "passed": true, "duration_ms": 1402 },
    { "name": "cli_deploy", "passed": true, "duration_ms": 18493, "url": "https://static-html-abc123.vercel.app" },
    { "name": "verify_live_url", "passed": true, "duration_ms": 2100 },
    { "name": "cli_status", "passed": true, "duration_ms": 523 },
    { "name": "cli_logs", "passed": true, "duration_ms": 410 },
    { "name": "cleanup", "passed": true, "duration_ms": 1205 }
  ],
  "total_duration_ms": 24688,
  "claude_instructions": "All 9 E2E steps passed. All 3 providers verified. The deployment was live at https://static-html-abc123.vercel.app and has been cleaned up."
}
```

When a step fails:
```json
{
  "suite": "e2e",
  "passed": false,
  "failed_step": "connect_providers",
  "steps": [
    { "name": "health_check", "passed": true, "duration_ms": 45 },
    { "name": "signup_login", "passed": true, "duration_ms": 312 },
    { "name": "connect_providers", "passed": false, "duration_ms": 1200, "error": "vercel token verification failed: 401 Unauthorized", "claude_instructions": "The stored Vercel token is expired or revoked. Run with --reauth=vercel to re-authenticate via OAuth, or set VERCEL_TOKEN env var with a fresh personal access token from https://vercel.com/account/tokens." }
  ],
  "total_duration_ms": 1557,
  "claude_instructions": "E2E test failed at step 'connect_providers'. The Vercel token is invalid. Re-authenticate before retrying."
}
```

### Running

```bash
# First run: will trigger OAuth flows for Vercel/Railway, prompt for Supabase PAT
# Tokens are encrypted and saved for future runs
go run ./tests/e2e/

# Subsequent runs: uses stored encrypted tokens (re-verifies them first)
go run ./tests/e2e/

# Re-authenticate a specific provider (token expired/revoked)
go run ./tests/e2e/ --reauth=vercel

# Re-authenticate all providers
go run ./tests/e2e/ --reauth=all

# Override with env vars (CI, one-off runs)
VERCEL_TOKEN="..." RAILWAY_TOKEN="..." SUPABASE_TOKEN="..." go run ./tests/e2e/

# Self-contained: starts Postgres, API, builds CLI, runs tests, tears down
go run ./tests/e2e/ --self-contained

# With a specific fixture
go run ./tests/e2e/ --fixture=nextjs

# Against a remote API (staging, prod)
VIBECLOUD_API_URL="https://api.vibecloud.dev" go run ./tests/e2e/

# Credential management
go run ./tests/e2e/ --wipe-credentials    # delete all stored tokens
go run ./tests/e2e/ --rotate-key          # rotate encryption key
```

### Required Code Changes

1. **`POST /api/providers/{provider}/connect/token`** — Direct token injection for testing.
   - Location: `api/handlers/providers.go`
   - Gated by `ALLOW_DIRECT_TOKEN_CONNECT=true` env var
   - Reuses existing `crypto.Encrypt()` + `UpsertProviderLink()` — same path as OAuth callbacks
   - ~30 lines of code

2. **`VIBECLOUD_CONFIG_DIR` env var support in CLI** — So the test can point the CLI at a temp config directory instead of `~/.vibecloud/`.
   - Location: `cli/internal/config/config.go`
   - Change: check `os.Getenv("VIBECLOUD_CONFIG_DIR")` before defaulting to `~/.vibecloud`
   - ~3 lines of code

3. **`go-keyring` dependency in `tests/e2e/`** — For storing the encryption key in the system keychain.
   - `github.com/zalando/go-keyring`
   - Only used by the test binary, not by the CLI or API

---

## Implementation Order

1. **Part A first** — validates the API works. Fast to run, no external dependencies beyond Docker.
2. **Part B second** — validates provider adapters. Requires provider tokens but no running API.
3. **Part C last** — the E2E test. Depends on both the API (Part A) and providers (Part B) being solid.

Parts A and B are foundations. Part C is the deliverable — the test an agent runs to answer "does it work?"

---

## What's NOT Covered

- Provider OAuth redirect flows from the web frontend (require browser automation — covered by Playwright e2e)
- Load testing or performance benchmarks
- Multi-provider orchestration (deploying frontend + database in one `deploy` command) — future test when that feature ships
