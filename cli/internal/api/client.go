package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the VibeCloud API.
type Client struct {
	baseURL      string
	accessToken  string
	refreshToken string
	httpClient   *http.Client
}

// UserProfile represents the authenticated user returned by /api/v1/me.
type UserProfile struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Tier  string `json:"tier"`
}

// TokenPair holds the access and refresh tokens returned after authentication.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// apiResponse is the standard API envelope returned by VibeCloud endpoints.
type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewClient creates a new VibeCloud API client.
func NewClient(baseURL, accessToken, refreshToken string) *Client {
	return &Client{
		baseURL:      baseURL,
		accessToken:  accessToken,
		refreshToken: refreshToken,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

// GetMe validates the token and returns the user profile.
func (c *Client) GetMe() (*UserProfile, error) {
	body, err := c.doRequest("GET", "/api/v1/me", nil)
	if err != nil {
		return nil, err
	}
	var user UserProfile
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user: %w", err)
	}
	return &user, nil
}

// ExchangeDeviceCode exchanges a device code for a token pair.
func (c *Client) ExchangeDeviceCode(code string) (*TokenPair, error) {
	payload := map[string]string{"code": code}
	body, err := c.doRequestNoAuth("POST", "/api/v1/auth/device-code/exchange", payload)
	if err != nil {
		return nil, err
	}
	var tokens TokenPair
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("failed to parse tokens: %w", err)
	}
	return &tokens, nil
}

// UpdateTier changes the user's tier.
func (c *Client) UpdateTier(tier string) (*UserProfile, error) {
	payload := map[string]string{"tier": tier}
	body, err := c.doRequest("PATCH", "/api/v1/tier", payload)
	if err != nil {
		return nil, err
	}
	var user UserProfile
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user: %w", err)
	}
	return &user, nil
}

// CheckDeployLimit checks if the user can deploy.
func (c *Client) CheckDeployLimit(projectName string, providers []string, environment string) (bool, int, int, error) {
	payload := map[string]interface{}{
		"project_name": projectName,
		"providers":    providers,
		"environment":  environment,
	}
	body, err := c.doRequest("POST", "/api/v1/deploys/check", payload)
	if err != nil {
		return false, 0, 0, err
	}
	var result struct {
		Allowed bool `json:"allowed"`
		Used    int  `json:"used"`
		Limit   int  `json:"limit"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, 0, 0, err
	}
	return result.Allowed, result.Used, result.Limit, nil
}

// doRequest makes an authenticated HTTP request and unwraps the API envelope.
func (c *Client) doRequest(method, path string, payload interface{}) (json.RawMessage, error) {
	var bodyReader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("API returned %d", resp.StatusCode)
		}
		return respBody, nil
	}

	if !apiResp.Success && apiResp.Error != nil {
		return nil, fmt.Errorf("%s", apiResp.Error.Message)
	}

	if apiResp.Data != nil {
		return apiResp.Data, nil
	}
	return respBody, nil
}

// doRequestNoAuth makes a request without the Authorization header (for public endpoints).
func (c *Client) doRequestNoAuth(method, path string, payload interface{}) (json.RawMessage, error) {
	saved := c.accessToken
	c.accessToken = ""
	defer func() { c.accessToken = saved }()
	return c.doRequest(method, path, payload)
}
