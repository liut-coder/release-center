package appreleases

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/liut-coder/game-helper-server/internal/platform/httpx"
)

func (h *Handler) CreateDeploymentTarget(w http.ResponseWriter, r *http.Request) {
	var req CreateDeploymentTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CreateDeploymentTarget(r.Context(), req)
	if err != nil {
		deploymentAPIError(w, r, err, "deploy.target_save_failed", "保存部署目标失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) Deployments(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.DeploymentRecords(r.Context())
	if err != nil {
		deploymentAPIError(w, r, err, "deploy.records_failed", "读取部署记录失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) Deployment(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.DeploymentRecord(r.Context(), chi.URLParam(r, "deployment_id"))
	if err != nil {
		deploymentAPIError(w, r, err, "deploy.record_failed", "读取部署记录失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateDeployment(w http.ResponseWriter, r *http.Request) {
	var req CreateDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CreateDeployment(r.Context(), req)
	if err != nil {
		deploymentAPIError(w, r, err, "deploy.create_failed", "创建部署记录失败")
		return
	}
	httpx.JSON(w, http.StatusAccepted, resp)
}

func (h *Handler) CompleteDeployment(w http.ResponseWriter, r *http.Request) {
	var req UpdateDeploymentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CompleteDeployment(r.Context(), chi.URLParam(r, "deployment_id"), req)
	if err != nil {
		deploymentAPIError(w, r, err, "deploy.complete_failed", "完成部署失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) FailDeployment(w http.ResponseWriter, r *http.Request) {
	var req UpdateDeploymentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.FailDeployment(r.Context(), chi.URLParam(r, "deployment_id"), req)
	if err != nil {
		deploymentAPIError(w, r, err, "deploy.fail_failed", "记录部署失败失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func deploymentAPIError(w http.ResponseWriter, r *http.Request, err error, code, message string) {
	status := http.StatusBadRequest
	if errors.Is(err, pgx.ErrNoRows) {
		status = http.StatusNotFound
	}
	if errors.Is(err, errDeploymentStoreUnavailable) {
		status = http.StatusServiceUnavailable
	}
	httpx.Error(w, r, status, code, message, map[string]any{"error": err.Error()})
}
