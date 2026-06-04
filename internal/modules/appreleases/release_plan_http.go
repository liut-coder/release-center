package appreleases

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/liut-coder/game-helper-server/internal/platform/httpx"
)

func (h *Handler) ReleasePlanOverview(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.ReleasePlanOverview(r.Context())
	if err != nil {
		releasePlanAPIError(w, r, err, "release_plan.overview_failed", "读取发布计划失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) ReleasePlan(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.ReleasePlan(r.Context(), chi.URLParam(r, "plan_id"))
	if err != nil {
		releasePlanAPIError(w, r, err, "release_plan.read_failed", "读取发布计划失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateReleaseUnit(w http.ResponseWriter, r *http.Request) {
	var req CreateReleaseUnitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CreateReleaseUnit(r.Context(), req)
	if err != nil {
		releasePlanAPIError(w, r, err, "release_unit.save_failed", "保存发布单元失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateReleasePlan(w http.ResponseWriter, r *http.Request) {
	var req CreateReleasePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CreateReleasePlan(r.Context(), req)
	if err != nil {
		releasePlanAPIError(w, r, err, "release_plan.create_failed", "创建发布计划失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) ReleasePlanAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ReleasePlanActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
			return
		}
		resp, err := h.service.ReleasePlanAction(r.Context(), chi.URLParam(r, "plan_id"), action, req)
		if err != nil {
			releasePlanAPIError(w, r, err, "release_plan.action_failed", "发布计划操作失败")
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}

func (h *Handler) CreateReleasePlanDeployment(w http.ResponseWriter, r *http.Request) {
	var req CreateReleasePlanDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CreateReleasePlanDeployment(r.Context(), chi.URLParam(r, "plan_id"), req)
	if err != nil {
		releasePlanAPIError(w, r, err, "release_plan.deployment_create_failed", "创建发布计划部署记录失败")
		return
	}
	httpx.JSON(w, http.StatusAccepted, resp)
}

func releasePlanAPIError(w http.ResponseWriter, r *http.Request, err error, code, message string) {
	status := http.StatusBadRequest
	if errors.Is(err, pgx.ErrNoRows) {
		status = http.StatusNotFound
	}
	if errors.Is(err, errReleasePlanStoreUnavailable) {
		status = http.StatusServiceUnavailable
	}
	if errors.Is(err, errDeploymentStoreUnavailable) {
		status = http.StatusServiceUnavailable
	}
	httpx.Error(w, r, status, code, message, map[string]any{"error": err.Error()})
}
