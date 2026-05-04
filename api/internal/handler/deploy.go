package handler

import (
	"encoding/json"
	"net/http"

	"github.com/christopherdang/vibecloud/api/internal/auth"
	"github.com/christopherdang/vibecloud/api/internal/response"
	"github.com/christopherdang/vibecloud/api/internal/service"
)

type DeployHandler struct {
	deployService *service.DeployService
}

func NewDeployHandler(deployService *service.DeployService) *DeployHandler {
	return &DeployHandler{deployService: deployService}
}

func (h *DeployHandler) CheckLimit(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	tier := auth.GetTier(r.Context())

	check, err := h.deployService.CheckLimit(r.Context(), userID, tier)
	if err != nil {
		response.InternalError(w, "Failed to check deploy limit")
		return
	}

	response.JSON(w, 200, check)
}

func (h *DeployHandler) LogDeploy(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var req struct {
		ProjectName string   `json:"project_name"`
		Providers   []string `json:"providers"`
		Environment string   `json:"environment"`
		Status      string   `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if err := h.deployService.LogDeploy(r.Context(), userID, req.ProjectName, req.Providers, req.Environment, req.Status); err != nil {
		response.InternalError(w, "Failed to log deploy")
		return
	}

	response.JSON(w, 201, map[string]string{"status": "logged"})
}
