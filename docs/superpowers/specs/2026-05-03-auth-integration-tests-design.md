# Auth Backend Integration Tests — Design Spec

## Goal

Verify the auth backend works end-to-end against a real Postgres database. Covers registration, login, token refresh, device code flow, tier management, and deploy limit enforcement.

## Architecture

Single test file (`api/integration_test.go`) behind a `//go:build integration` tag. Uses `httptest.NewServer` with the full chi router — same code path as production. No mocks.

## Test Infrastructure

### TestMain Setup

1. Start Postgres 16 via Docker on port 5433, or use `DATABASE_URL` env var if set
2. Read and apply `api/migrations/001_create_schema.sql` to the test database
3. Set `JWT_SIGNING_SECRET` to `"test-secret-32-chars-minimum-key"`
4. Initialize the full chi router (repositories, services, handlers, middleware) using the test database pool
5. Start `httptest.NewServer` with the router

### TestMain Teardown

1. `TRUNCATE users, device_codes, deploy_logs CASCADE`
2. Close database pool
3. Stop httptest server
4. Stop and remove Docker container (if started by test)

### Helper Functions

```go
func doRequest(method, path string, body interface{}, token string) (*http.Response, []byte)
func parseJSON[T any](body []byte) T
func signupAndGetToken(email, password string) (accessToken, refreshToken string)
```

## Test Groups

### 1. Register (4 tests)

| Test | Method | Path | Validates |
|------|--------|------|-----------|
| TestRegisterSuccess | POST | /api/v1/auth/register | 201, returns user (id, email, tier=free), access_token, refresh_token. Password hash NOT in response. |
| TestRegisterDuplicateEmail | POST | /api/v1/auth/register | 409 when email already exists |
| TestRegisterWeakPassword | POST | /api/v1/auth/register | 400 for password < 8 chars |
| TestRegisterMissingFields | POST | /api/v1/auth/register | 400 for empty email or password |

### 2. Login (3 tests)

| Test | Method | Path | Validates |
|------|--------|------|-----------|
| TestLoginSuccess | POST | /api/v1/auth/login | 200, valid JWT with correct `sub` claim, returns user + tokens |
| TestLoginWrongPassword | POST | /api/v1/auth/login | 401 "Invalid email or password" |
| TestLoginNonexistentUser | POST | /api/v1/auth/login | 401 same error message (no user enumeration) |

### 3. Refresh (2 tests)

| Test | Method | Path | Validates |
|------|--------|------|-----------|
| TestRefreshSuccess | POST | /api/v1/auth/refresh | 200, new token pair, new access token works on /me |
| TestRefreshInvalidToken | POST | /api/v1/auth/refresh | 401 for garbage/expired token |

### 4. Me (3 tests)

| Test | Method | Path | Validates |
|------|--------|------|-----------|
| TestMeSuccess | GET | /api/v1/me | 200, returns user id, email, tier |
| TestMeNoToken | GET | /api/v1/me | 401 |
| TestMeInvalidToken | GET | /api/v1/me | 401 |

### 5. Tier (3 tests)

| Test | Method | Path | Validates |
|------|--------|------|-----------|
| TestTierUpgrade | PATCH | /api/v1/tier | 200, tier changes to "premium" |
| TestTierDowngrade | PATCH | /api/v1/tier | 200, tier changes to "free" |
| TestTierInvalid | PATCH | /api/v1/tier | 400 for tier value other than "free"/"premium" |

### 6. Device Code (4 tests)

| Test | Method | Path | Validates |
|------|--------|------|-----------|
| TestDeviceCodeGenerate | POST | /api/v1/auth/device-code | 201, returns 8-char uppercase alphanumeric code |
| TestDeviceCodeExchange | POST | /api/v1/auth/device-code/exchange | 200, returns valid token pair, token works on /me |
| TestDeviceCodeExchangeInvalid | POST | /api/v1/auth/device-code/exchange | 401 for nonexistent code |
| TestDeviceCodeExchangeAlreadyClaimed | POST | /api/v1/auth/device-code/exchange | 401 when code used twice |

### 7. Deploy Limits (3 tests)

| Test | Method | Path | Validates |
|------|--------|------|-----------|
| TestDeployLimitFreeUnderLimit | POST | /api/v1/deploys/check | allowed=true, used=0, limit=15 |
| TestDeployLimitFreeAtLimit | POST | /api/v1/deploys/check | Log 15 deploys, then check → allowed=false, used=15 |
| TestDeployLimitPremium | POST | /api/v1/deploys/check | Upgrade to premium, check → allowed=true, limit=-1 |

## Running

```bash
# Start test Postgres
docker run --rm -d -p 5433:5432 \
  -e POSTGRES_PASSWORD=test \
  -e POSTGRES_DB=vibecloud_test \
  --name vc-test-db postgres:16

# Run tests
cd api
DATABASE_URL="postgres://postgres:test@localhost:5433/vibecloud_test?sslmode=disable" \
JWT_SIGNING_SECRET="test-secret-32-chars-minimum-key" \
go test -tags=integration -v -count=1 ./...

# Cleanup
docker stop vc-test-db
```

## File Structure

```
api/
├── integration_test.go    # All 22 tests + helpers + TestMain
├── v1.go                  # Production entry point (unchanged)
├── go.mod                 # May need testify added
└── migrations/
    └── 001_create_schema.sql
```

## Dependencies

- `github.com/stretchr/testify` for assertions (optional — can use stdlib `testing` if preferred)
- Docker with `postgres:16` image

## Out of Scope

- Provider integration tests (real Vercel/Supabase API calls)
- Full E2E smoke test (CLI subprocess → API → Postgres → live URL)
- Frontend tests
- Load/performance testing

These are deferred to a future testing phase.
