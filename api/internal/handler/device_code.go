package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/christopherdang/vibecloud/api/internal/auth"
	"github.com/christopherdang/vibecloud/api/internal/response"
	"github.com/christopherdang/vibecloud/api/internal/service"
)

type DeviceCodeHandler struct {
	deviceCodeService *service.DeviceCodeService
}

func NewDeviceCodeHandler(deviceCodeService *service.DeviceCodeService) *DeviceCodeHandler {
	return &DeviceCodeHandler{deviceCodeService: deviceCodeService}
}

func (h *DeviceCodeHandler) Generate(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	code, err := h.deviceCodeService.Generate(r.Context(), userID)
	if err != nil {
		response.InternalError(w, "Failed to generate device code")
		return
	}

	response.JSON(w, 201, map[string]string{"code": code})
}

func (h *DeviceCodeHandler) Exchange(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		response.BadRequest(w, "Code is required")
		return
	}

	tokens, err := h.deviceCodeService.Exchange(r.Context(), code)
	if err != nil {
		response.Unauthorized(w, "Invalid or expired code")
		return
	}

	response.JSON(w, 200, tokens)
}
