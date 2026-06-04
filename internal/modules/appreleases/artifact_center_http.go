package appreleases

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/liut-coder/game-helper-server/internal/platform/httpx"
)

func (h *Handler) ArtifactCenterOverview(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.ArtifactCenterOverview(r.Context())
	if err != nil {
		artifactCenterAPIError(w, r, err, "artifact_center.overview_failed", "读取制品中心失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) ArtifactCenterItem(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.ArtifactCenterItem(r.Context(), chi.URLParam(r, "artifact_id"))
	if err != nil {
		artifactCenterAPIError(w, r, err, "artifact_center.item_failed", "读取制品失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func artifactCenterAPIError(w http.ResponseWriter, r *http.Request, err error, code, message string) {
	status := http.StatusInternalServerError
	if strings.Contains(err.Error(), "required") {
		status = http.StatusBadRequest
	}
	if errors.Is(err, pgx.ErrNoRows) {
		status = http.StatusNotFound
	}
	if errors.Is(err, errArtifactCenterStoreUnavailable) {
		status = http.StatusServiceUnavailable
	}
	httpx.Error(w, r, status, code, message, map[string]any{"error": err.Error()})
}
