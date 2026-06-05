package appreleases

import (
	"net/http"

	"github.com/liut-coder/game-helper-server/internal/platform/httpx"
)

func (h *Handler) IntegrationCredentials(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.IntegrationCredentials(r.Context())
	if err != nil {
		httpx.Error(w, r, http.StatusInternalServerError, "integration.credentials_failed", "读取凭证引用失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
