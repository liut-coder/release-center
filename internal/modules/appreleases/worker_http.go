package appreleases

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/liut-coder/game-helper-server/internal/platform/httpx"
)

func (h *Handler) WorkerOverview(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.WorkerOverview(r.Context())
	if err != nil {
		workerAPIError(w, r, err, "worker.overview_failed", "读取 Worker 失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateWorkerTask(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkerTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CreateWorkerTask(r.Context(), req)
	if err != nil {
		workerAPIError(w, r, err, "worker.task_create_failed", "创建 Worker 任务失败")
		return
	}
	httpx.JSON(w, http.StatusAccepted, resp)
}

func (h *Handler) RegisterWorker(w http.ResponseWriter, r *http.Request) {
	var req WorkerRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.RegisterWorker(r.Context(), req)
	if err != nil {
		workerAPIError(w, r, err, "worker.register_failed", "Worker 注册失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) WorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req WorkerHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.SaveWorkerHeartbeat(r.Context(), req)
	if err != nil {
		workerAPIError(w, r, err, "worker.heartbeat_failed", "Worker 心跳失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) NextWorkerTask(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.NextWorkerTask(r.Context(), WorkerTaskNextRequest{WorkerKey: r.URL.Query().Get("worker_key")})
	if err != nil {
		workerAPIError(w, r, err, "worker.next_task_failed", "Worker 领取任务失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) AppendWorkerTaskLogs(w http.ResponseWriter, r *http.Request) {
	var req WorkerTaskLogsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.AppendWorkerTaskLogs(r.Context(), chi.URLParam(r, "task_id"), req)
	if err != nil {
		workerAPIError(w, r, err, "worker.task_logs_failed", "Worker 日志回传失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) SaveWorkerTaskArtifacts(w http.ResponseWriter, r *http.Request) {
	var req WorkerTaskArtifactsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.SaveWorkerTaskArtifacts(r.Context(), chi.URLParam(r, "task_id"), req)
	if err != nil {
		workerAPIError(w, r, err, "worker.task_artifacts_failed", "Worker 产物回传失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) CompleteWorkerTask(w http.ResponseWriter, r *http.Request) {
	var req WorkerTaskCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CompleteWorkerTask(r.Context(), chi.URLParam(r, "task_id"), req)
	if err != nil {
		workerAPIError(w, r, err, "worker.task_complete_failed", "Worker 完成任务失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) FailWorkerTask(w http.ResponseWriter, r *http.Request) {
	var req WorkerTaskFailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.FailWorkerTask(r.Context(), chi.URLParam(r, "task_id"), req)
	if err != nil {
		workerAPIError(w, r, err, "worker.task_fail_failed", "Worker 失败回传失败")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func workerAPIError(w http.ResponseWriter, r *http.Request, err error, code, message string) {
	status := http.StatusBadRequest
	if errors.Is(err, pgx.ErrNoRows) {
		status = http.StatusNotFound
	}
	if errors.Is(err, errWorkerStoreUnavailable) {
		status = http.StatusServiceUnavailable
	}
	httpx.Error(w, r, status, code, message, map[string]any{"error": err.Error()})
}
