//go:build integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/christopherdang/vibecloud/api/internal/auth"
	authhandler "github.com/christopherdang/vibecloud/api/internal/handler"
	"github.com/christopherdang/vibecloud/api/internal/repository"
	"github.com/christopherdang/vibecloud/api/internal/response"
	"github.com/christopherdang/vibecloud/api/internal/service"
)

var (
	testServer *httptest.Server
	testPool   *pgxpool.Pool
)

func TestMain(m *testing.M) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}
	jwtSecret := os.Getenv("JWT_SIGNING_SECRET")
	if jwtSecret == "" {
		fmt.Fprintln(os.Stderr, "JWT_SIGNING_SECRET is required")
		os.Exit(1)
	}

	ctx := context.Background()

	var err error
	testPool, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create pool: %v\n", err)
		os.Exit(1)
	}

	// Run migration
	migrationSQL, err := os.ReadFile("migrations/001_create_schema.sql")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read migration: %v\n", err)
		os.Exit(1)
	}
	if _, err := testPool.Exec(ctx, string(migrationSQL)); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run migration: %v\n", err)
		os.Exit(1)
	}

	// Wire up router exactly like v1.go setup()
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	// Health check
	r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, 200, map[string]string{"status": "ok"})
	})

	userRepo := repository.NewUserRepository(testPool)
	authService := service.NewAuthService(userRepo, jwtSecret)
	authH := authhandler.NewAuthHandler(authService)
	authMiddleware := auth.NewMiddleware(jwtSecret)

	deviceCodeRepo := repository.NewDeviceCodeRepository(testPool)
	deviceCodeService := service.NewDeviceCodeService(deviceCodeRepo, authService)
	deviceCodeHandler := authhandler.NewDeviceCodeHandler(deviceCodeService)

	deployLogRepo := repository.NewDeployLogRepository(testPool)
	deployService := service.NewDeployService(deployLogRepo)
	deployHandler := authhandler.NewDeployHandler(deployService)

	// Public auth routes
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", authH.Register)
		r.Post("/login", authH.Login)
		r.Post("/refresh", authH.Refresh)
		r.Post("/device-code/exchange", deviceCodeHandler.Exchange)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Post("/device-code", deviceCodeHandler.Generate)
		})
	})

	// Protected routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Get("/me", authH.Me)
		r.Patch("/tier", authH.UpdateTier)
		r.Post("/deploys/check", deployHandler.CheckLimit)
		r.Post("/deploys/log", deployHandler.LogDeploy)
	})

	testServer = httptest.NewServer(r)

	code := m.Run()

	// Cleanup
	testPool.Exec(context.Background(), "TRUNCATE deploy_logs, device_codes, users CASCADE")
	testPool.Close()
	testServer.Close()

	os.Exit(code)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func doRequest(t *testing.T, method, path string, body interface{}, token string) (*http.Response, []byte) {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, testServer.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, respBody
}

type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func parseResponse(t *testing.T, body []byte) apiResponse {
	t.Helper()
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to parse response: %v\nbody: %s", err, string(body))
	}
	return resp
}

func signupAndGetTokens(t *testing.T, email, password string) (accessToken, refreshToken string) {
	t.Helper()
	resp, body := doRequest(t, "POST", "/api/v1/auth/register", map[string]string{
		"email": email, "password": password,
	}, "")
	if resp.StatusCode != 201 {
		t.Fatalf("signup failed: %d %s", resp.StatusCode, string(body))
	}
	apiResp := parseResponse(t, body)
	var data struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal signup data: %v", err)
	}
	return data.AccessToken, data.RefreshToken
}

// ---------------------------------------------------------------------------
// Group 1: Register
// ---------------------------------------------------------------------------

func TestRegisterSuccess(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	resp, body := doRequest(t, "POST", "/api/v1/auth/register", map[string]string{
		"email": email, "password": "securepassword",
	}, "")
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseResponse(t, body)
	if !apiResp.Success {
		t.Fatalf("expected success=true")
	}

	var data struct {
		User struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Tier  string `json:"tier"`
		} `json:"user"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}

	if data.User.ID == "" {
		t.Errorf("expected user.id to be non-empty")
	}
	if data.User.Email != email {
		t.Errorf("expected user.email=%q, got %q", email, data.User.Email)
	}
	if data.User.Tier != "free" {
		t.Errorf("expected user.tier=free, got %q", data.User.Tier)
	}
	if data.AccessToken == "" {
		t.Errorf("expected access_token to be non-empty")
	}
	if data.RefreshToken == "" {
		t.Errorf("expected refresh_token to be non-empty")
	}

	// Verify hashed_password is NOT in the JSON response
	var rawData map[string]json.RawMessage
	json.Unmarshal(apiResp.Data, &rawData)
	var rawUser map[string]interface{}
	json.Unmarshal(rawData["user"], &rawUser)
	if _, found := rawUser["hashed_password"]; found {
		t.Errorf("hashed_password must not be present in response")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	signupAndGetTokens(t, email, "securepassword")

	resp, body := doRequest(t, "POST", "/api/v1/auth/register", map[string]string{
		"email": email, "password": "securepassword",
	}, "")
	if resp.StatusCode != 409 {
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, string(body))
	}
	apiResp := parseResponse(t, body)
	if apiResp.Success {
		t.Errorf("expected success=false")
	}
	if apiResp.Error == nil || apiResp.Error.Message == "" {
		t.Errorf("expected error message about email exists")
	}
}

func TestRegisterWeakPassword(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	resp, body := doRequest(t, "POST", "/api/v1/auth/register", map[string]string{
		"email": email, "password": "short",
	}, "")
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestRegisterMissingFields(t *testing.T) {
	// Empty body
	resp, body := doRequest(t, "POST", "/api/v1/auth/register", map[string]string{}, "")
	if resp.StatusCode != 400 {
		t.Errorf("empty body: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// Empty email
	resp, body = doRequest(t, "POST", "/api/v1/auth/register", map[string]string{
		"email": "", "password": "securepassword",
	}, "")
	if resp.StatusCode != 400 {
		t.Errorf("empty email: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// Empty password
	resp, body = doRequest(t, "POST", "/api/v1/auth/register", map[string]string{
		"email": "nopass@test.com", "password": "",
	}, "")
	if resp.StatusCode != 400 {
		t.Errorf("empty password: expected 400, got %d: %s", resp.StatusCode, string(body))
	}
}

// ---------------------------------------------------------------------------
// Group 2: Login
// ---------------------------------------------------------------------------

func TestLoginSuccess(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	signupAndGetTokens(t, email, "securepassword")

	resp, body := doRequest(t, "POST", "/api/v1/auth/login", map[string]string{
		"email": email, "password": "securepassword",
	}, "")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseResponse(t, body)
	var data struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	json.Unmarshal(apiResp.Data, &data)

	if data.User.Email != email {
		t.Errorf("expected email=%q, got %q", email, data.User.Email)
	}
	if data.AccessToken == "" {
		t.Errorf("expected access_token to be non-empty")
	}

	// Verify access token works on /me
	meResp, meBody := doRequest(t, "GET", "/api/v1/me", nil, data.AccessToken)
	if meResp.StatusCode != 200 {
		t.Errorf("GET /me failed with login token: %d %s", meResp.StatusCode, string(meBody))
	}
}

func TestLoginWrongPassword(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	signupAndGetTokens(t, email, "securepassword")

	resp, body := doRequest(t, "POST", "/api/v1/auth/login", map[string]string{
		"email": email, "password": "wrongpassword",
	}, "")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestLoginNonexistentUser(t *testing.T) {
	resp, body := doRequest(t, "POST", "/api/v1/auth/login", map[string]string{
		"email": "nobody@test.com", "password": "somepassword",
	}, "")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, string(body))
	}

	// Should return the same generic error as wrong password (no user enumeration)
	apiResp := parseResponse(t, body)
	if apiResp.Error == nil || apiResp.Error.Message != "Invalid email or password" {
		t.Errorf("expected generic error message, got: %+v", apiResp.Error)
	}
}

// ---------------------------------------------------------------------------
// Group 3: Refresh
// ---------------------------------------------------------------------------

func TestRefreshSuccess(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	_, refreshToken := signupAndGetTokens(t, email, "securepassword")

	resp, body := doRequest(t, "POST", "/api/v1/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, "")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseResponse(t, body)
	var data struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	json.Unmarshal(apiResp.Data, &data)

	if data.AccessToken == "" {
		t.Errorf("expected new access_token to be non-empty")
	}
	if data.RefreshToken == "" {
		t.Errorf("expected new refresh_token to be non-empty")
	}

	// Verify new access token works on /me
	meResp, meBody := doRequest(t, "GET", "/api/v1/me", nil, data.AccessToken)
	if meResp.StatusCode != 200 {
		t.Errorf("GET /me with refreshed token: %d %s", meResp.StatusCode, string(meBody))
	}
}

func TestRefreshInvalidToken(t *testing.T) {
	resp, body := doRequest(t, "POST", "/api/v1/auth/refresh", map[string]string{
		"refresh_token": "garbage-token-value",
	}, "")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, string(body))
	}
}

// ---------------------------------------------------------------------------
// Group 4: Me
// ---------------------------------------------------------------------------

func TestMeSuccess(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	accessToken, _ := signupAndGetTokens(t, email, "securepassword")

	resp, body := doRequest(t, "GET", "/api/v1/me", nil, accessToken)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseResponse(t, body)
	var user struct {
		Email string `json:"email"`
		Tier  string `json:"tier"`
	}
	json.Unmarshal(apiResp.Data, &user)

	if user.Email != email {
		t.Errorf("expected email=%q, got %q", email, user.Email)
	}
	if user.Tier != "free" {
		t.Errorf("expected tier=free, got %q", user.Tier)
	}
}

func TestMeNoToken(t *testing.T) {
	resp, body := doRequest(t, "GET", "/api/v1/me", nil, "")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestMeInvalidToken(t *testing.T) {
	resp, body := doRequest(t, "GET", "/api/v1/me", nil, "garbage-token-value")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, string(body))
	}
}

// ---------------------------------------------------------------------------
// Group 5: Tier
// ---------------------------------------------------------------------------

func TestTierUpgrade(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	accessToken, _ := signupAndGetTokens(t, email, "securepassword")

	resp, body := doRequest(t, "PATCH", "/api/v1/tier", map[string]string{
		"tier": "premium",
	}, accessToken)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseResponse(t, body)
	var user struct {
		Tier string `json:"tier"`
	}
	json.Unmarshal(apiResp.Data, &user)
	if user.Tier != "premium" {
		t.Errorf("expected tier=premium, got %q", user.Tier)
	}

	// Verify with GET /me
	meResp, meBody := doRequest(t, "GET", "/api/v1/me", nil, accessToken)
	if meResp.StatusCode != 200 {
		t.Fatalf("GET /me after upgrade: %d %s", meResp.StatusCode, string(meBody))
	}
	meApiResp := parseResponse(t, meBody)
	var meUser struct {
		Tier string `json:"tier"`
	}
	json.Unmarshal(meApiResp.Data, &meUser)
	if meUser.Tier != "premium" {
		t.Errorf("GET /me: expected tier=premium, got %q", meUser.Tier)
	}
}

func TestTierDowngrade(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	accessToken, _ := signupAndGetTokens(t, email, "securepassword")

	// Upgrade first
	resp, body := doRequest(t, "PATCH", "/api/v1/tier", map[string]string{
		"tier": "premium",
	}, accessToken)
	if resp.StatusCode != 200 {
		t.Fatalf("upgrade failed: %d %s", resp.StatusCode, string(body))
	}

	// Downgrade
	resp, body = doRequest(t, "PATCH", "/api/v1/tier", map[string]string{
		"tier": "free",
	}, accessToken)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseResponse(t, body)
	var user struct {
		Tier string `json:"tier"`
	}
	json.Unmarshal(apiResp.Data, &user)
	if user.Tier != "free" {
		t.Errorf("expected tier=free, got %q", user.Tier)
	}
}

func TestTierInvalid(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	accessToken, _ := signupAndGetTokens(t, email, "securepassword")

	resp, body := doRequest(t, "PATCH", "/api/v1/tier", map[string]string{
		"tier": "gold",
	}, accessToken)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}
}

// ---------------------------------------------------------------------------
// Group 6: Device Code
// ---------------------------------------------------------------------------

func TestDeviceCodeGenerate(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	accessToken, _ := signupAndGetTokens(t, email, "securepassword")

	resp, body := doRequest(t, "POST", "/api/v1/auth/device-code", nil, accessToken)
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseResponse(t, body)
	var data struct {
		Code string `json:"code"`
	}
	json.Unmarshal(apiResp.Data, &data)

	if len(data.Code) != 8 {
		t.Errorf("expected code length=8, got %d (%q)", len(data.Code), data.Code)
	}

	// Verify uppercase alphanumeric (the codeChars set excludes I, O, 0, 1)
	matched, _ := regexp.MatchString(`^[A-Z0-9]{8}$`, data.Code)
	if !matched {
		t.Errorf("expected code to be uppercase alphanumeric, got %q", data.Code)
	}
}

func TestDeviceCodeExchange(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	accessToken, _ := signupAndGetTokens(t, email, "securepassword")

	// Generate code
	resp, body := doRequest(t, "POST", "/api/v1/auth/device-code", nil, accessToken)
	if resp.StatusCode != 201 {
		t.Fatalf("generate failed: %d %s", resp.StatusCode, string(body))
	}
	apiResp := parseResponse(t, body)
	var genData struct {
		Code string `json:"code"`
	}
	json.Unmarshal(apiResp.Data, &genData)

	// Exchange (no auth header)
	resp, body = doRequest(t, "POST", "/api/v1/auth/device-code/exchange", map[string]string{
		"code": genData.Code,
	}, "")
	if resp.StatusCode != 200 {
		t.Fatalf("exchange failed: %d %s", resp.StatusCode, string(body))
	}

	apiResp = parseResponse(t, body)
	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	json.Unmarshal(apiResp.Data, &tokens)

	if tokens.AccessToken == "" {
		t.Errorf("expected access_token to be non-empty")
	}
	if tokens.RefreshToken == "" {
		t.Errorf("expected refresh_token to be non-empty")
	}

	// Verify the exchanged token works on /me and belongs to the same user
	meResp, meBody := doRequest(t, "GET", "/api/v1/me", nil, tokens.AccessToken)
	if meResp.StatusCode != 200 {
		t.Fatalf("GET /me with device-code token: %d %s", meResp.StatusCode, string(meBody))
	}
	meApiResp := parseResponse(t, meBody)
	var meUser struct {
		Email string `json:"email"`
	}
	json.Unmarshal(meApiResp.Data, &meUser)
	if meUser.Email != email {
		t.Errorf("expected email=%q from device-code token, got %q", email, meUser.Email)
	}
}

func TestDeviceCodeExchangeInvalid(t *testing.T) {
	resp, body := doRequest(t, "POST", "/api/v1/auth/device-code/exchange", map[string]string{
		"code": "XXXXXXXX",
	}, "")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestDeviceCodeExchangeAlreadyClaimed(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	accessToken, _ := signupAndGetTokens(t, email, "securepassword")

	// Generate code
	resp, body := doRequest(t, "POST", "/api/v1/auth/device-code", nil, accessToken)
	if resp.StatusCode != 201 {
		t.Fatalf("generate failed: %d %s", resp.StatusCode, string(body))
	}
	apiResp := parseResponse(t, body)
	var genData struct {
		Code string `json:"code"`
	}
	json.Unmarshal(apiResp.Data, &genData)

	// First exchange — should succeed
	resp, body = doRequest(t, "POST", "/api/v1/auth/device-code/exchange", map[string]string{
		"code": genData.Code,
	}, "")
	if resp.StatusCode != 200 {
		t.Fatalf("first exchange failed: %d %s", resp.StatusCode, string(body))
	}

	// Second exchange — should fail
	resp, body = doRequest(t, "POST", "/api/v1/auth/device-code/exchange", map[string]string{
		"code": genData.Code,
	}, "")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401 on second exchange, got %d: %s", resp.StatusCode, string(body))
	}
}

// ---------------------------------------------------------------------------
// Group 7: Deploy Limits
// ---------------------------------------------------------------------------

func TestDeployLimitFreeUnderLimit(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	accessToken, _ := signupAndGetTokens(t, email, "securepassword")

	resp, body := doRequest(t, "POST", "/api/v1/deploys/check", nil, accessToken)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseResponse(t, body)
	var check struct {
		Allowed bool   `json:"allowed"`
		Used    int    `json:"used"`
		Limit   int    `json:"limit"`
		Tier    string `json:"tier"`
	}
	json.Unmarshal(apiResp.Data, &check)

	if !check.Allowed {
		t.Errorf("expected allowed=true, got false")
	}
	if check.Used != 0 {
		t.Errorf("expected used=0, got %d", check.Used)
	}
	if check.Limit != 15 {
		t.Errorf("expected limit=15, got %d", check.Limit)
	}
}

func TestDeployLimitFreeAtLimit(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	accessToken, _ := signupAndGetTokens(t, email, "securepassword")

	// Log 15 deploys
	for i := 0; i < 15; i++ {
		resp, body := doRequest(t, "POST", "/api/v1/deploys/log", map[string]interface{}{
			"project_name": "test",
			"providers":    []string{"vercel"},
			"environment":  "preview",
			"status":       "completed",
		}, accessToken)
		if resp.StatusCode != 201 {
			t.Fatalf("deploy log %d failed: %d %s", i+1, resp.StatusCode, string(body))
		}
	}

	// Check limit
	resp, body := doRequest(t, "POST", "/api/v1/deploys/check", nil, accessToken)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp := parseResponse(t, body)
	var check struct {
		Allowed bool `json:"allowed"`
		Used    int  `json:"used"`
		Limit   int  `json:"limit"`
	}
	json.Unmarshal(apiResp.Data, &check)

	if check.Allowed {
		t.Errorf("expected allowed=false, got true")
	}
	if check.Used != 15 {
		t.Errorf("expected used=15, got %d", check.Used)
	}
	if check.Limit != 15 {
		t.Errorf("expected limit=15, got %d", check.Limit)
	}
}

func TestDeployLimitPremium(t *testing.T) {
	email := fmt.Sprintf("test-%s@test.com", t.Name())
	accessToken, _ := signupAndGetTokens(t, email, "securepassword")

	// Upgrade to premium
	resp, body := doRequest(t, "PATCH", "/api/v1/tier", map[string]string{
		"tier": "premium",
	}, accessToken)
	if resp.StatusCode != 200 {
		t.Fatalf("upgrade failed: %d %s", resp.StatusCode, string(body))
	}

	// The JWT issued at registration still has tier=free in its claims.
	// We need a fresh token with the updated tier. Re-login to get one.
	resp, body = doRequest(t, "POST", "/api/v1/auth/login", map[string]string{
		"email": email, "password": "securepassword",
	}, "")
	if resp.StatusCode != 200 {
		t.Fatalf("re-login failed: %d %s", resp.StatusCode, string(body))
	}
	apiResp := parseResponse(t, body)
	var loginData struct {
		AccessToken string `json:"access_token"`
	}
	json.Unmarshal(apiResp.Data, &loginData)

	resp, body = doRequest(t, "POST", "/api/v1/deploys/check", nil, loginData.AccessToken)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	apiResp = parseResponse(t, body)
	var check struct {
		Allowed bool `json:"allowed"`
		Limit   int  `json:"limit"`
	}
	json.Unmarshal(apiResp.Data, &check)

	if !check.Allowed {
		t.Errorf("expected allowed=true, got false")
	}
	if check.Limit != -1 {
		t.Errorf("expected limit=-1, got %d", check.Limit)
	}
}
