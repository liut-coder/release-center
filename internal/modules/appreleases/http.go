package appreleases

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/liut-coder/game-helper-server/internal/platform/httpx"
)

type Handler struct {
	service *Service
	blobs   BlobStore
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func NewHandlerWithBlobStore(service *Service, blobs BlobStore) *Handler {
	return &Handler{service: service, blobs: blobs}
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	versionCode, _ := strconv.Atoi(query.Get("version_code"))
	buildNumber, _ := strconv.Atoi(query.Get("build_number"))
	resp := h.service.Check(r.Context(), CheckRequest{
		AppKey:      query.Get("app_key"),
		DeviceID:    query.Get("device_id"),
		DeviceKey:   query.Get("device_key"),
		PackageName: query.Get("package_name"),
		VersionCode: versionCode,
		VersionName: query.Get("version_name"),
		BuildNumber: buildNumber,
		Channel:     query.Get("channel"),
		BuildType:   query.Get("build_type"),
	})
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) UpdateCheck(w http.ResponseWriter, r *http.Request) {
	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp := h.service.Check(r.Context(), req)
	resp.Release = map[string]any{
		"versionName":          resp.VersionName,
		"versionCode":          resp.VersionCode,
		"buildNumber":          resp.BuildNumber,
		"title":                "游戏助手更新",
		"summary":              resp.MessageZh,
		"releaseNotesMarkdown": resp.ReleaseNotes,
		"downloadUrl":          firstNonBlank(resp.DownloadURL, resp.APKURL),
		"sha256":               resp.SHA256,
		"fileSize":             resp.SizeBytes,
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) ResourceCheck(w http.ResponseWriter, r *http.Request) {
	var req ResourceCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	httpx.JSON(w, http.StatusOK, h.service.ResourceCheck(r.Context(), req))
}

func (h *Handler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	var req UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.SaveUpdateEvent(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "required") {
			status = http.StatusBadRequest
		}
		httpx.Error(w, r, status, "app_release.update_event_failed", "更新事件上报失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.SaveHeartbeat(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "required") {
			status = http.StatusBadRequest
		}
		httpx.Error(w, r, status, "app_release.heartbeat_failed", "设备心跳上报失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) TaskPreflight(w http.ResponseWriter, r *http.Request) {
	var req TaskPreflightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	httpx.JSON(w, http.StatusOK, h.service.TaskPreflight(r.Context(), req))
}

func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.Overview(r.Context())
	if err != nil {
		httpx.Error(w, r, http.StatusInternalServerError, "app_release.overview_failed", "读取 App 发版概览失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) AdminOverview(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.AdminOverview(r.Context())
	if err != nil {
		httpx.Error(w, r, http.StatusInternalServerError, "app_release.admin_overview_failed", "读取 App 发版中心失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateApp(w http.ResponseWriter, r *http.Request) {
	var req CreateAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CreateApp(r.Context(), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_release.create_app_failed", "保存 App 失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) AppAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := h.service.AppAction(r.Context(), chi.URLParam(r, "app_id"), action)
		if err != nil {
			httpx.Error(w, r, http.StatusBadRequest, "app_release.app_action_failed", "App 操作失败", map[string]any{"error": err.Error()})
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}

func (h *Handler) SystemManagementOverview(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.SystemManagementOverview(r.Context())
	if err != nil {
		httpx.Error(w, r, http.StatusInternalServerError, "system.overview_failed", "读取系统管理数据失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateSystemUser(w http.ResponseWriter, r *http.Request) {
	var req CreateSystemUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CreateSystemUser(r.Context(), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "system.user_save_failed", "保存用户失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) SystemUserAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := h.service.SystemUserAction(r.Context(), chi.URLParam(r, "user_id"), action)
		if err != nil {
			httpx.Error(w, r, http.StatusBadRequest, "system.user_action_failed", "用户操作失败", map[string]any{"error": err.Error()})
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}

func (h *Handler) CreateSystemRole(w http.ResponseWriter, r *http.Request) {
	var req CreateSystemRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CreateSystemRole(r.Context(), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "system.role_save_failed", "保存角色失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) SystemRoleAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := h.service.SystemRoleAction(r.Context(), chi.URLParam(r, "role_id"), action)
		if err != nil {
			httpx.Error(w, r, http.StatusBadRequest, "system.role_action_failed", "角色操作失败", map[string]any{"error": err.Error()})
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}

func (h *Handler) CreateSystemPermission(w http.ResponseWriter, r *http.Request) {
	var req CreateSystemPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CreateSystemPermission(r.Context(), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "system.permission_save_failed", "保存权限失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) SystemPermissionAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := h.service.SystemPermissionAction(r.Context(), chi.URLParam(r, "permission_id"), action)
		if err != nil {
			httpx.Error(w, r, http.StatusBadRequest, "system.permission_action_failed", "权限操作失败", map[string]any{"error": err.Error()})
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}

func (h *Handler) CreateSystemDictionary(w http.ResponseWriter, r *http.Request) {
	var req CreateSystemDictionaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CreateSystemDictionary(r.Context(), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "system.dictionary_save_failed", "保存字典失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) SystemDictionaryAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := h.service.SystemDictionaryAction(r.Context(), chi.URLParam(r, "dictionary_id"), action)
		if err != nil {
			httpx.Error(w, r, http.StatusBadRequest, "system.dictionary_action_failed", "字典操作失败", map[string]any{"error": err.Error()})
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}

func (h *Handler) CreateSystemMenu(w http.ResponseWriter, r *http.Request) {
	var req CreateSystemMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CreateSystemMenu(r.Context(), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "system.menu_save_failed", "保存菜单失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) SystemMenuAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := h.service.SystemMenuAction(r.Context(), chi.URLParam(r, "menu_id"), action)
		if err != nil {
			httpx.Error(w, r, http.StatusBadRequest, "system.menu_action_failed", "菜单操作失败", map[string]any{"error": err.Error()})
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}

func (h *Handler) CreateBuild(w http.ResponseWriter, r *http.Request) {
	var req CreateBuildRequest
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if h.blobs == nil {
			httpx.Error(w, r, http.StatusInternalServerError, "app_release.storage_missing", "APK 文件存储未配置", nil)
			return
		}
		if err := r.ParseMultipartForm(512 << 20); err != nil {
			httpx.Error(w, r, http.StatusBadRequest, "app_release.invalid_upload", "APK 上传格式不正确", map[string]any{"error": err.Error()})
			return
		}
		if err := json.Unmarshal([]byte(r.FormValue("metadata")), &req); err != nil {
			httpx.Error(w, r, http.StatusBadRequest, "app_release.invalid_metadata", "构建元数据不正确", map[string]any{"error": err.Error()})
			return
		}
		file, header, err := r.FormFile("apk")
		if err != nil && err != http.ErrMissingFile {
			httpx.Error(w, r, http.StatusBadRequest, "app_release.apk_open_failed", "APK 文件读取失败", map[string]any{"error": err.Error()})
			return
		}
		if file != nil {
			defer file.Close()
			content, err := io.ReadAll(file)
			if err != nil {
				httpx.Error(w, r, http.StatusBadRequest, "app_release.apk_read_failed", "APK 文件读取失败", map[string]any{"error": err.Error()})
				return
			}
			sum := sha256.Sum256(content)
			storageKey, size, err := h.blobs.Save(r.Context(), "app-releases/apks", header.Filename, bytes.NewReader(content))
			if err != nil {
				httpx.Error(w, r, http.StatusInternalServerError, "app_release.apk_save_failed", "APK 文件保存失败", map[string]any{"error": err.Error()})
				return
			}
			req.APKFileName = header.Filename
			req.StorageKey = storageKey
			req.ArtifactSize = size
			req.SHA256 = hex.EncodeToString(sum[:])
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
			return
		}
	}
	resp, err := h.service.CreateBuild(r.Context(), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_release.create_build_failed", "创建构建记录失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateCIBuild(w http.ResponseWriter, r *http.Request) {
	var req CreateBuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	if req.Provider == "" {
		req.Provider = "ci"
	}
	resp, err := h.service.CreateBuild(r.Context(), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_release.ci_build_failed", "CI 构建记录创建失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateCIArtifact(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		h.createCIArtifactMultipart(w, r)
		return
	}
	var req CreateArtifactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	if req.Provider == "" {
		req.Provider = "ci"
	}
	resp, err := h.service.CreateArtifact(r.Context(), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_release.artifact_failed", "构建产物登记失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) createCIArtifactMultipart(w http.ResponseWriter, r *http.Request) {
	if h.blobs == nil {
		httpx.Error(w, r, http.StatusInternalServerError, "app_release.storage_missing", "构建产物存储未配置", nil)
		return
	}
	if err := r.ParseMultipartForm(512 << 20); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_release.invalid_upload", "构建产物上传格式不正确", map[string]any{"error": err.Error()})
		return
	}
	var req CreateArtifactRequest
	if err := json.Unmarshal([]byte(r.FormValue("metadata")), &req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_release.invalid_metadata", "构建产物元数据不正确", map[string]any{"error": err.Error()})
		return
	}
	file, header, err := r.FormFile("file")
	if err == http.ErrMissingFile {
		file, header, err = r.FormFile("apk")
	}
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_release.artifact_open_failed", "构建产物读取失败", map[string]any{"error": err.Error()})
		return
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_release.artifact_read_failed", "构建产物读取失败", map[string]any{"error": err.Error()})
		return
	}
	sum := sha256.Sum256(content)
	artifactType := normalizeArtifactType(req.ArtifactType)
	storageKey, size, err := h.blobs.Save(r.Context(), "app-releases/artifacts", header.Filename, bytes.NewReader(content))
	if err != nil {
		httpx.Error(w, r, http.StatusInternalServerError, "app_release.artifact_save_failed", "构建产物保存失败", map[string]any{"error": err.Error()})
		return
	}
	req.ArtifactType = artifactType
	req.ArtifactURL = "/api/v1/app/builds/pending/download"
	req.FileName = header.Filename
	req.SizeBytes = size
	req.SHA256 = hex.EncodeToString(sum[:])
	req.StorageKey = storageKey
	if req.Provider == "" {
		req.Provider = "ci"
	}
	resp, err := h.service.CreateArtifact(r.Context(), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_release.artifact_failed", "构建产物登记失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) Webhook(provider string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
		if err != nil {
			httpx.Error(w, r, http.StatusBadRequest, "webhook.read_failed", "Webhook 读取失败", map[string]any{"error": err.Error()})
			return
		}
		var payload map[string]any
		if len(strings.TrimSpace(string(body))) > 0 {
			if err := json.Unmarshal(body, &payload); err != nil {
				httpx.Error(w, r, http.StatusBadRequest, "webhook.invalid_json", "Webhook JSON 格式不正确", map[string]any{"error": err.Error()})
				return
			}
		} else {
			payload = map[string]any{}
		}
		req := webhookEventFromRequest(provider, r, payload)
		resp, err := h.service.SaveWebhookEvent(r.Context(), req)
		if err != nil {
			httpx.Error(w, r, http.StatusBadRequest, "webhook.save_failed", "Webhook 事件记录失败", map[string]any{"error": err.Error()})
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}

func (h *Handler) GetBuildJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "job_id")
	overview, err := h.service.AdminOverview(r.Context())
	if err != nil {
		httpx.Error(w, r, http.StatusInternalServerError, "app_release.build_job_failed", "读取构建记录失败", map[string]any{"error": err.Error()})
		return
	}
	for _, job := range overview.BuildJobs {
		if job.ID == jobID {
			httpx.JSON(w, http.StatusOK, BuildActionResponse{Job: job})
			return
		}
	}
	httpx.Error(w, r, http.StatusNotFound, "app_release.build_job_not_found", "构建记录不存在", nil)
}

func (h *Handler) GetBuildLogs(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "job_id")
	overview, err := h.service.AdminOverview(r.Context())
	if err != nil {
		httpx.Error(w, r, http.StatusInternalServerError, "app_release.build_logs_failed", "读取构建日志失败", map[string]any{"error": err.Error()})
		return
	}
	for _, job := range overview.BuildJobs {
		if job.ID == jobID {
			httpx.JSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "logs": job.LogTail})
			return
		}
	}
	httpx.Error(w, r, http.StatusNotFound, "app_release.build_job_not_found", "构建记录不存在", nil)
}

func (h *Handler) CreateRelease(w http.ResponseWriter, r *http.Request) {
	var req CreateReleaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.CreateRelease(r.Context(), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_release.create_failed", "创建 App 发布失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) ReleaseAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := h.service.ReleaseAction(r.Context(), chi.URLParam(r, "release_id"), action)
		if err != nil {
			httpx.Error(w, r, http.StatusBadRequest, "app_release.action_failed", "App 发布操作失败", map[string]any{"error": err.Error()})
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}

func (h *Handler) UpdateReleaseRollout(w http.ResponseWriter, r *http.Request) {
	var req UpdateRolloutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.UpdateReleaseRollout(r.Context(), chi.URLParam(r, "release_id"), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_release.rollout_failed", "调整 App 发布灰度失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) UpdateReleaseNotes(w http.ResponseWriter, r *http.Request) {
	var req UpdateNotesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.UpdateReleaseNotes(r.Context(), chi.URLParam(r, "release_id"), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_release.notes_failed", "更新 App 发布说明失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) DownloadRelease(w http.ResponseWriter, r *http.Request) {
	releaseID := chi.URLParam(r, "release_id")
	if h.blobs != nil && h.service.store != nil {
		storageKey, fileName, err := h.service.store.GetReleaseStorageKey(r.Context(), releaseID)
		if err == nil {
			h.serveBlob(w, r, storageKey, fileName, "application/vnd.android.package-archive")
			return
		}
	}
	overview, err := h.service.AdminOverview(r.Context())
	if err != nil {
		httpx.Error(w, r, http.StatusInternalServerError, "app_release.download_failed", "读取下载地址失败", map[string]any{"error": err.Error()})
		return
	}
	for _, release := range overview.Releases {
		if release.ID == releaseID {
			if release.DownloadURL != "" {
				http.Redirect(w, r, release.DownloadURL, http.StatusFound)
				return
			}
			break
		}
	}
	httpx.Error(w, r, http.StatusNotFound, "app_release.download_not_found", "发布文件不存在", nil)
}

func (h *Handler) DownloadBuild(w http.ResponseWriter, r *http.Request) {
	if h.blobs == nil || h.service.store == nil {
		httpx.Error(w, r, http.StatusInternalServerError, "app_release.storage_missing", "APK 文件存储未配置", nil)
		return
	}
	storageKey, fileName, err := h.service.store.GetBuildStorageKey(r.Context(), chi.URLParam(r, "build_id"))
	if err != nil {
		httpx.Error(w, r, http.StatusNotFound, "app_release.build_file_not_found", "构建文件不存在", map[string]any{"error": err.Error()})
		return
	}
	h.serveBlob(w, r, storageKey, fileName, contentTypeForFile(fileName))
}

func (h *Handler) DownloadBuildArtifact(w http.ResponseWriter, r *http.Request) {
	if h.blobs == nil || h.service.store == nil {
		httpx.Error(w, r, http.StatusInternalServerError, "app_release.storage_missing", "构建产物存储未配置", nil)
		return
	}
	storageKey, fileName, err := h.service.store.GetBuildArtifactStorageKey(r.Context(), chi.URLParam(r, "artifact_id"))
	if err != nil {
		httpx.Error(w, r, http.StatusNotFound, "app_release.artifact_file_not_found", "构建产物不存在", map[string]any{"error": err.Error()})
		return
	}
	h.serveBlob(w, r, storageKey, fileName, contentTypeForFile(fileName))
}

func (h *Handler) CreateResourceVersion(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_resource.invalid_upload", "资源上传格式不正确", map[string]any{"error": err.Error()})
		return
	}
	var req CreateResourceVersionRequest
	if err := json.Unmarshal([]byte(r.FormValue("metadata")), &req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_resource.invalid_metadata", "资源版本元数据不正确", map[string]any{"error": err.Error()})
		return
	}
	files := r.MultipartForm.File["packages"]
	uploads := make([]UploadedResourcePackage, 0, len(files))
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			httpx.Error(w, r, http.StatusBadRequest, "app_resource.open_failed", "资源包读取失败", map[string]any{"error": err.Error()})
			return
		}
		content, readErr := io.ReadAll(file)
		_ = file.Close()
		if readErr != nil {
			httpx.Error(w, r, http.StatusBadRequest, "app_resource.read_failed", "资源包读取失败", map[string]any{"error": readErr.Error()})
			return
		}
		uploads = append(uploads, UploadedResourcePackage{FileName: header.Filename, Content: content})
	}
	resp, err := h.service.CreateResourceVersion(r.Context(), req, uploads, h.blobs)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_resource.create_failed", "创建资源版本失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) ResourceAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := h.service.ResourceAction(r.Context(), chi.URLParam(r, "resource_id"), action)
		if err != nil {
			httpx.Error(w, r, http.StatusBadRequest, "app_resource.action_failed", "资源版本操作失败", map[string]any{"error": err.Error()})
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	}
}

func (h *Handler) UpdateResourceRollout(w http.ResponseWriter, r *http.Request) {
	var req UpdateRolloutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.UpdateResourceRollout(r.Context(), chi.URLParam(r, "resource_id"), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_resource.rollout_failed", "调整资源灰度失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) UpdateResourceNotes(w http.ResponseWriter, r *http.Request) {
	var req UpdateNotesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "request.invalid_json", "请求 JSON 格式不正确", nil)
		return
	}
	resp, err := h.service.UpdateResourceNotes(r.Context(), chi.URLParam(r, "resource_id"), req)
	if err != nil {
		httpx.Error(w, r, http.StatusBadRequest, "app_resource.notes_failed", "更新资源说明失败", map[string]any{"error": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) DownloadResourcePackage(w http.ResponseWriter, r *http.Request) {
	if h.blobs == nil || h.service.store == nil {
		httpx.Error(w, r, http.StatusInternalServerError, "app_resource.storage_missing", "资源文件存储未配置", nil)
		return
	}
	storageKey, fileName, err := h.service.store.GetResourcePackageStorageKey(r.Context(), chi.URLParam(r, "package_id"))
	if err != nil {
		httpx.Error(w, r, http.StatusNotFound, "app_resource.package_not_found", "资源包不存在", map[string]any{"error": err.Error()})
		return
	}
	h.serveBlob(w, r, storageKey, fileName, "application/zip")
}

func (h *Handler) DownloadResourceManifest(w http.ResponseWriter, r *http.Request) {
	if h.blobs == nil || h.service.store == nil {
		httpx.Error(w, r, http.StatusInternalServerError, "app_resource.storage_missing", "资源文件存储未配置", nil)
		return
	}
	storageKey, fileName, err := h.service.store.GetResourceManifestStorageKey(r.Context(), chi.URLParam(r, "resource_id"))
	if err != nil {
		httpx.Error(w, r, http.StatusNotFound, "app_resource.manifest_not_found", "资源 Manifest 不存在", map[string]any{"error": err.Error()})
		return
	}
	h.serveBlob(w, r, storageKey, fileName, "application/json; charset=utf-8")
}

func (h *Handler) serveBlob(w http.ResponseWriter, r *http.Request, storageKey, fileName, contentType string) {
	body, err := h.blobs.Open(r.Context(), storageKey)
	if err != nil {
		httpx.Error(w, r, http.StatusNotFound, "app_resource.file_not_found", "资源文件不存在", map[string]any{"error": err.Error()})
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(fileName, `"`, "")+`"`)
	_, _ = io.Copy(w, body)
}

func contentTypeForFile(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".apk":
		return "application/vnd.android.package-archive"
	case ".aab":
		return "application/octet-stream"
	case ".zip":
		return "application/zip"
	case ".gz", ".tgz":
		return "application/gzip"
	case ".json":
		return "application/json; charset=utf-8"
	}
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}

func webhookEventFromRequest(provider string, r *http.Request, payload map[string]any) WebhookEventRequest {
	eventType := firstNonBlank(
		r.Header.Get("X-GitHub-Event"),
		r.Header.Get("X-Gitea-Event"),
		stringFromMap(payload, "event_type"),
		"unknown",
	)
	deliveryID := firstNonBlank(
		r.Header.Get("X-GitHub-Delivery"),
		r.Header.Get("X-Gitea-Delivery"),
		r.Header.Get("X-Gitea-Event-UUID"),
		stringFromMap(payload, "delivery_id"),
	)
	repository := firstNonBlank(
		nestedString(payload, "repository", "full_name"),
		nestedString(payload, "repository", "name"),
	)
	commitSHA := firstNonBlank(
		stringFromMap(payload, "after"),
		nestedString(payload, "head_commit", "id"),
		nestedString(payload, "workflow_run", "head_sha"),
	)
	sender := firstNonBlank(
		nestedString(payload, "sender", "login"),
		nestedString(payload, "sender", "username"),
		nestedString(payload, "pusher", "name"),
	)
	return WebhookEventRequest{
		Provider:   provider,
		EventType:  eventType,
		DeliveryID: deliveryID,
		Repository: repository,
		Ref:        stringFromMap(payload, "ref"),
		CommitSHA:  commitSHA,
		Sender:     sender,
		Action:     stringFromMap(payload, "action"),
		Workflow:   firstNonBlank(nestedString(payload, "workflow", "name"), stringFromMap(payload, "workflow")),
		RunID:      firstNonBlank(nestedString(payload, "workflow_run", "id"), stringFromMap(payload, "run_id")),
		RawPayload: payload,
	}
}

func stringFromMap(values map[string]any, key string) string {
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int:
		return strconv.Itoa(typed)
	default:
		return ""
	}
}

func nestedString(values map[string]any, firstKey, secondKey string) string {
	nested, ok := values[firstKey].(map[string]any)
	if !ok {
		return ""
	}
	return stringFromMap(nested, secondKey)
}
