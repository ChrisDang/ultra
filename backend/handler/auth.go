package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/christopherdang/vibecloud/backend/auth"
	"github.com/christopherdang/vibecloud/backend/model"
	"github.com/christopherdang/vibecloud/backend/response"
	"github.com/christopherdang/vibecloud/backend/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		response.BadRequest(w, "Email and password are required")
		return
	}

	user, tokens, err := h.authService.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, model.ErrUserExists) {
			response.Conflict(w, "An account with this email already exists")
			return
		}
		response.BadRequest(w, err.Error())
		return
	}

	response.JSON(w, 201, map[string]interface{}{
		"user":          user,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	user, tokens, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, model.ErrInvalidCredentials) {
			response.Unauthorized(w, "Invalid email or password")
			return
		}
		response.InternalError(w, "Login failed")
		return
	}

	response.JSON(w, 200, map[string]interface{}{
		"user":          user,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		response.BadRequest(w, "refresh_token is required")
		return
	}

	tokens, err := h.authService.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		response.Unauthorized(w, "Invalid refresh token")
		return
	}

	response.JSON(w, 200, tokens)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	user, err := h.authService.GetUserByID(r.Context(), userID)
	if err != nil || user == nil {
		response.Unauthorized(w, "User not found")
		return
	}
	response.JSON(w, 200, user)
}

func (h *AuthHandler) UpdateTier(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	var req struct {
		Tier string `json:"tier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}
	if req.Tier != "free" && req.Tier != "premium" {
		response.BadRequest(w, "Tier must be 'free' or 'premium'")
		return
	}

	if err := h.authService.UpdateTier(r.Context(), userID, req.Tier); err != nil {
		response.InternalError(w, "Failed to update tier")
		return
	}

	user, _ := h.authService.GetUserByID(r.Context(), userID)
	response.JSON(w, 200, user)
}
