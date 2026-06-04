package appreleases

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/liut-coder/game-helper-server/internal/platform/httpx"
)

func (h *Handler) BuildCenterProjects(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.BuildCenterOverview(r.Context())
	if err != nil {
		httpx.Error(w, r, http.StatusInternalServerError, "build_center.overview_failed", "读取构建中心失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) BuildCenterProject(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.BuildCenterProject(r.Context(), chi.URLParam(r, "project_key"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, errBuildCenterStoreUnavailable) {
			httpx.Error(w, r, http.StatusNotFound, "build_center.project_not_found", "构建项目不存在", nil)
			return
		}
		httpx.Error(w, r, http.StatusInternalServerError, "build_center.project_failed", "读取构建项目失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) DeploymentTargets(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.DeploymentTargets(r.Context())
	if err != nil {
		httpx.Error(w, r, http.StatusInternalServerError, "deploy.targets_failed", "读取部署目标失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deployment_targets": resp})
}
