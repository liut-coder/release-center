package appreleases

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type BlobStore interface {
	Save(ctx context.Context, namespace, filename string, body io.Reader) (string, int64, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
}

type AppBuildLookupStore interface {
	AppBuild(ctx context.Context, buildID string) (AppBuildJob, error)
}

const (
	qualityAlertNotifyAuditAction     = "quality_alert.notify"
	qualityAlertNotifyFailedAction    = "quality_alert.notify_failed"
	qualityAlertAutoExecuteAction     = "quality_alert.auto_execute"
	qualityAlertAutoExecuteFailAction = "quality_alert.auto_execute_failed"
)

func (s *Service) AdminOverview(ctx context.Context) (AdminOverview, error) {
	if s.store == nil {
		return AdminOverview{
			QualityPolicy:     s.cfg.QualityPolicy,
			QualityAutomation: s.qualityAutomationPolicy(),
			MessageZh:         "App 发版中心未连接数据库",
		}, nil
	}
	resp, err := s.store.AdminOverview(ctx)
	if err != nil {
		return AdminOverview{}, err
	}
	resp.QualityPolicy = s.cfg.QualityPolicy
	resp.QualityAutomation = s.qualityAutomationPolicy()
	resp.QualityMetrics = buildReleaseQualityMetrics(resp.UpgradeEvents, resp.ResourceUpdateEvents, s.cfg.QualityPolicy)
	resp.QualityAlerts = s.processQualityAlerts(ctx, &resp, buildQualityAlerts(resp.QualityMetrics))
	resp.MessageZh = "App 发版中心已读取"
	return resp, nil
}

func (s *Service) qualityAutomationPolicy() QualityAutomationPolicy {
	return QualityAutomationPolicy{
		AlertWebhookConfigured: strings.TrimSpace(s.cfg.QualityAlertWebhookURL) != "",
		AutoPauseResource:      s.cfg.QualityAutoPauseResource,
		AutoRollbackAPK:        s.cfg.QualityAutoRollbackAPK,
	}
}

func (s *Service) processQualityAlerts(ctx context.Context, overview *AdminOverview, alerts []QualityAlert) []QualityAlert {
	for i := range alerts {
		s.processQualityAlertNotification(ctx, &alerts[i])
		s.processQualityAlertAutomation(ctx, overview, &alerts[i])
	}
	return alerts
}

func (s *Service) processQualityAlertNotification(ctx context.Context, alert *QualityAlert) {
	if strings.TrimSpace(s.cfg.QualityAlertWebhookURL) == "" {
		alert.NotificationStatus = "not_configured"
		return
	}
	if s.auditExists(ctx, qualityAlertNotifyAuditAction, alert.ID) {
		alert.NotificationStatus = "sent"
		return
	}
	if err := s.sendQualityAlertWebhook(ctx, *alert); err != nil {
		alert.NotificationStatus = "failed"
		_ = s.store.InsertAudit(ctx, qualityAlertNotifyFailedAction, "quality_alert", alert.ID, "质量告警通知失败", map[string]any{
			"alert_id":           alert.ID,
			"category":           alert.Category,
			"recommended_action": alert.RecommendedAction,
			"error":              err.Error(),
		})
		return
	}
	alert.NotificationStatus = "sent"
	_ = s.store.InsertAudit(ctx, qualityAlertNotifyAuditAction, "quality_alert", alert.ID, "质量告警通知已发送", map[string]any{
		"alert_id":           alert.ID,
		"category":           alert.Category,
		"recommended_action": alert.RecommendedAction,
		"severity":           alert.Severity,
	})
}

func (s *Service) processQualityAlertAutomation(ctx context.Context, overview *AdminOverview, alert *QualityAlert) {
	switch alert.RecommendedAction {
	case "pause_resource":
		alert.AutomationAction = "pause_resource"
		if !s.cfg.QualityAutoPauseResource {
			alert.AutomationStatus = "disabled"
			return
		}
		target := qualityResourceAutomationTarget(overview)
		if target == nil {
			alert.AutomationStatus = "target_missing"
			return
		}
		alert.AutomationTargetID = target.ID
		if s.auditExists(ctx, qualityAlertAutoExecuteAction, alert.ID) {
			alert.AutomationStatus = "executed"
			return
		}
		resource, err := s.store.UpdateResourceStatus(ctx, target.ID, "paused", false)
		if err != nil {
			alert.AutomationStatus = "failed"
			_ = s.store.InsertAudit(ctx, qualityAlertAutoExecuteFailAction, "quality_alert", alert.ID, "质量告警自动执行失败", map[string]any{
				"alert_id":           alert.ID,
				"automation_action":  alert.AutomationAction,
				"target_resource_id": target.ID,
				"recommended_action": alert.RecommendedAction,
				"recommended_reason": alert.Reason,
				"automation_error":   err.Error(),
			})
			return
		}
		alert.AutomationStatus = "executed"
		if resource.ID != "" {
			alert.AutomationTargetID = resource.ID
		}
		_ = s.store.InsertAudit(ctx, qualityAlertAutoExecuteAction, "quality_alert", alert.ID, "质量告警已自动暂停资源", map[string]any{
			"alert_id":           alert.ID,
			"automation_action":  alert.AutomationAction,
			"target_resource_id": alert.AutomationTargetID,
			"resource_version":   firstNonBlank(resource.ResourceVersion, target.ResourceVersion),
			"recommended_action": alert.RecommendedAction,
			"recommended_reason": alert.Reason,
		})
	case "rollback_apk":
		alert.AutomationAction = "rollback_apk"
		if s.cfg.QualityAutoRollbackAPK {
			alert.AutomationStatus = "manual_required"
			return
		}
		alert.AutomationStatus = "disabled"
	default:
		alert.AutomationStatus = "not_applicable"
	}
}

func (s *Service) sendQualityAlertWebhook(ctx context.Context, alert QualityAlert) error {
	body, err := json.Marshal(map[string]any{
		"app_key": s.cfg.AppKey,
		"alert":   alert,
	})
	if err != nil {
		return err
	}
	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, strings.TrimSpace(s.cfg.QualityAlertWebhookURL), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("quality alert webhook returned %d", resp.StatusCode)
	}
	return nil
}

func (s *Service) auditExists(ctx context.Context, action, targetID string) bool {
	if s.store == nil {
		return false
	}
	exists, err := s.store.AuditExists(ctx, action, targetID)
	return err == nil && exists
}

func qualityResourceAutomationTarget(overview *AdminOverview) *AppResourceVersionAdmin {
	if overview == nil {
		return nil
	}
	if overview.LatestResource != nil && canAutoPauseResource(overview.LatestResource.Status) {
		return overview.LatestResource
	}
	for i := range overview.ResourceVersions {
		if canAutoPauseResource(overview.ResourceVersions[i].Status) {
			return &overview.ResourceVersions[i]
		}
	}
	return nil
}

func canAutoPauseResource(status string) bool {
	switch status {
	case "released", "rolling_out", "testing":
		return true
	default:
		return false
	}
}

func (s *Service) CreateApp(ctx context.Context, req CreateAppRequest) (AppActionResponse, error) {
	if s.store == nil {
		return AppActionResponse{}, fmt.Errorf("store is required")
	}
	req.AppKey = strings.TrimSpace(req.AppKey)
	req.Name = strings.TrimSpace(req.Name)
	req.Platform = normalizePlatform(req.Platform)
	req.PackageName = strings.TrimSpace(req.PackageName)
	req.Description = strings.TrimSpace(req.Description)
	if req.AppKey == "" || req.Name == "" || req.PackageName == "" {
		return AppActionResponse{}, fmt.Errorf("app_key, name and package_name are required")
	}
	req.AppKey = safeFilePart(req.AppKey)
	app, err := s.store.CreateApp(ctx, req)
	if err != nil {
		return AppActionResponse{}, err
	}
	_ = s.store.InsertAudit(ctx, "app.create", "app", app.ID, "创建或更新 App", map[string]any{
		"app_key":      app.AppKey,
		"platform":     app.Platform,
		"package_name": app.PackageName,
	})
	return AppActionResponse{OK: true, App: app, MessageZh: "App 已保存"}, nil
}

func (s *Service) AppAction(ctx context.Context, id, action string) (AppActionResponse, error) {
	enabled := false
	switch strings.TrimSpace(action) {
	case "enable":
		enabled = true
	case "disable":
		enabled = false
	default:
		return AppActionResponse{}, fmt.Errorf("unsupported app action: %s", action)
	}
	app, err := s.store.UpdateAppEnabled(ctx, strings.TrimSpace(id), enabled)
	if err != nil {
		return AppActionResponse{}, err
	}
	_ = s.store.InsertAudit(ctx, "app."+action, "app", app.ID, "更新 App 启用状态", map[string]any{
		"enabled": enabled,
	})
	return AppActionResponse{OK: true, App: app, MessageZh: "App 状态已更新"}, nil
}

func (s *Service) CreateBuild(ctx context.Context, req CreateBuildRequest) (BuildActionResponse, error) {
	if s.store == nil {
		return BuildActionResponse{}, fmt.Errorf("store is required")
	}
	req.GitRef = strings.TrimSpace(req.GitRef)
	req.GitCommit = strings.TrimSpace(req.GitCommit)
	req.GitBranch = strings.TrimSpace(req.GitBranch)
	req.Channel = normalizeChannel(req.Channel, s.cfg.Channel)
	req.BuildType = normalizeBuildType(req.BuildType, s.cfg.BuildType)
	req.VersionName = strings.TrimSpace(req.VersionName)
	req.APIBaseURL = strings.TrimSpace(req.APIBaseURL)
	req.ReleaseNotes = strings.TrimSpace(req.ReleaseNotes)
	req.ArtifactType = normalizeArtifactType(req.ArtifactType)
	req.ArtifactURL = strings.TrimSpace(req.ArtifactURL)
	req.FileName = strings.TrimSpace(req.FileName)
	req.BuildURL = strings.TrimSpace(req.BuildURL)
	req.Provider = strings.TrimSpace(req.Provider)
	req.Workflow = strings.TrimSpace(req.Workflow)
	req.RunID = strings.TrimSpace(req.RunID)
	req.StartedBy = strings.TrimSpace(req.StartedBy)
	if req.GitRef == "" || req.VersionName == "" || req.VersionCode <= 0 {
		return BuildActionResponse{}, fmt.Errorf("git_ref, version_name and version_code are required")
	}
	buildNumber := req.BuildNumber
	if buildNumber <= 0 {
		buildNumber = req.VersionCode
	}
	cfg := s.cfg
	cfg.LatestVersionName = req.VersionName
	cfg.LatestVersionCode = req.VersionCode
	cfg.BuildNumber = buildNumber
	cfg.Channel = req.Channel
	cfg.BuildType = req.BuildType
	if cfg.DownloadURL == "" {
		cfg.DownloadURL = cfg.APKURL
	}
	job := AppBuildJob{
		ID:               uuid.NewString(),
		Status:           normalizeBuildStatus(req.Status),
		GitRef:           req.GitRef,
		GitCommit:        firstNonBlank(req.GitCommit, shortCommitFromRef(req.GitRef)),
		GitBranch:        firstNonBlank(req.GitBranch, req.GitRef),
		BuildType:        req.BuildType,
		Channel:          req.Channel,
		VersionName:      req.VersionName,
		VersionCode:      req.VersionCode,
		BuildNumber:      buildNumber,
		BuildEnvironment: firstNonBlank(req.Channel, "dev"),
		ArtifactType:     req.ArtifactType,
		ArtifactPath:     firstNonBlank(req.ArtifactPath, req.ArtifactURL, cfg.DownloadURL, cfg.APKURL),
		ArtifactSize:     firstPositiveInt64(req.ArtifactSize, cfg.SizeBytes),
		SHA256:           firstNonBlank(req.SHA256, cfg.SHA256),
		FileName:         path.Base(firstNonBlank(req.APKFileName, req.FileName, req.ArtifactPath, req.ArtifactURL, cfg.DownloadURL, cfg.APKURL, apkFileName(s.cfg.AppKey, req.VersionName, req.Channel, buildNumber))),
		StorageKey:       req.StorageKey,
		APIBaseURL:       firstNonBlank(req.APIBaseURL, req.BuildURL),
		ReleaseNotes:     req.ReleaseNotes,
		StartedBy:        firstNonBlank(req.StartedBy, req.Provider, "admin"),
		CreatedAt:        time.Now(),
		StartedAt:        time.Now(),
		FinishedAt:       time.Now(),
		DurationMS:       1,
		LogTail: []string{
			"已登记构建记录",
			"当前为管理后台同步登记，后续可接 CI 构建队列",
		},
	}
	if req.StorageKey != "" {
		job.ArtifactPath = "/api/v1/app/builds/" + job.ID + "/download"
	}
	created, err := s.store.CreateBuild(ctx, job, cfg)
	if err != nil {
		return BuildActionResponse{}, err
	}
	_ = s.store.InsertAudit(ctx, "release.build_create", "app_build", created.ID, "创建 App 构建记录", map[string]any{
		"version_name": created.VersionName,
		"version_code": created.VersionCode,
		"channel":      created.Channel,
		"provider":     req.Provider,
		"workflow":     req.Workflow,
		"run_id":       req.RunID,
	})
	return BuildActionResponse{Job: created, MessageZh: "构建记录已创建"}, nil
}

func (s *Service) CreateArtifact(ctx context.Context, req CreateArtifactRequest) (BuildActionResponse, error) {
	if s.store == nil {
		return BuildActionResponse{}, fmt.Errorf("store is required")
	}
	cfg := s.cfg
	if strings.TrimSpace(req.AppKey) != "" {
		cfg.AppKey = strings.TrimSpace(req.AppKey)
	}
	req.GitRef = strings.TrimSpace(req.GitRef)
	req.GitCommit = strings.TrimSpace(req.GitCommit)
	req.GitBranch = strings.TrimSpace(req.GitBranch)
	req.Channel = normalizeChannel(req.Channel, s.cfg.Channel)
	req.BuildType = normalizeBuildType(req.BuildType, s.cfg.BuildType)
	req.VersionName = strings.TrimSpace(req.VersionName)
	req.ArtifactType = normalizeArtifactType(req.ArtifactType)
	req.ArtifactURL = strings.TrimSpace(req.ArtifactURL)
	req.FileName = strings.TrimSpace(req.FileName)
	req.BuildURL = strings.TrimSpace(req.BuildURL)
	req.ReleaseNotes = strings.TrimSpace(req.ReleaseNotes)
	req.Provider = strings.TrimSpace(req.Provider)
	req.Workflow = strings.TrimSpace(req.Workflow)
	req.RunID = strings.TrimSpace(req.RunID)
	artifactName := safeFilePart(firstNonBlank(req.ArtifactName, req.Name, req.ArtifactType, req.FileName, "artifact"))
	if req.GitRef == "" || req.VersionName == "" || req.VersionCode <= 0 {
		return BuildActionResponse{}, fmt.Errorf("git_ref, version_name and version_code are required")
	}
	buildNumber := req.BuildNumber
	if buildNumber <= 0 {
		buildNumber = req.VersionCode
	}
	cfg.LatestVersionName = req.VersionName
	cfg.LatestVersionCode = req.VersionCode
	cfg.BuildNumber = buildNumber
	cfg.Channel = req.Channel
	cfg.BuildType = req.BuildType
	fileName := path.Base(firstNonBlank(req.FileName, req.ArtifactURL, apkFileName(cfg.AppKey, req.VersionName, req.Channel, buildNumber)))
	artifactID := uuid.NewString()
	artifactPath := req.ArtifactURL
	if req.StorageKey != "" {
		artifactPath = "/api/v1/app/build-artifacts/" + artifactID + "/download"
	}
	build := AppBuildJob{
		ID:               uuid.NewString(),
		Status:           "success",
		GitRef:           req.GitRef,
		GitCommit:        firstNonBlank(req.GitCommit, shortCommitFromRef(req.GitRef)),
		GitBranch:        firstNonBlank(req.GitBranch, req.GitRef),
		BuildType:        req.BuildType,
		Channel:          req.Channel,
		VersionName:      req.VersionName,
		VersionCode:      req.VersionCode,
		BuildNumber:      buildNumber,
		BuildEnvironment: firstNonBlank(req.Channel, "dev"),
		ArtifactType:     req.ArtifactType,
		ArtifactPath:     req.ArtifactURL,
		ArtifactSize:     req.SizeBytes,
		SHA256:           req.SHA256,
		FileName:         fileName,
		StorageKey:       req.StorageKey,
		APIBaseURL:       req.BuildURL,
		ReleaseNotes:     req.ReleaseNotes,
		StartedBy:        firstNonBlank(req.Provider, "ci"),
		CreatedAt:        time.Now(),
		StartedAt:        time.Now(),
		FinishedAt:       time.Now(),
		DurationMS:       1,
		LogTail: []string{
			"已登记 CI 构建产物",
			"产物类型：" + req.ArtifactType,
		},
	}
	artifact := AppBuildArtifact{
		ID:           artifactID,
		Name:         artifactName,
		ArtifactType: req.ArtifactType,
		ArtifactPath: artifactPath,
		FileName:     fileName,
		SizeBytes:    req.SizeBytes,
		SHA256:       req.SHA256,
		StorageKey:   req.StorageKey,
		CreatedAt:    time.Now(),
	}
	created, createdArtifact, err := s.store.CreateBuildArtifact(ctx, build, artifact, cfg)
	if err != nil {
		return BuildActionResponse{}, err
	}
	_ = s.store.InsertAudit(ctx, "release.artifact_create", "app_build", created.ID, "登记构建产物", map[string]any{
		"version_name":  created.VersionName,
		"version_code":  created.VersionCode,
		"build_number":  created.BuildNumber,
		"artifact_name": createdArtifact.Name,
		"artifact_type": createdArtifact.ArtifactType,
		"provider":      req.Provider,
		"workflow":      req.Workflow,
		"run_id":        req.RunID,
	})
	return BuildActionResponse{Job: created, Artifact: &createdArtifact, MessageZh: "构建产物已登记"}, nil
}

func (s *Service) SaveWebhookEvent(ctx context.Context, req WebhookEventRequest) (WebhookEventResponse, error) {
	if s.store == nil {
		return WebhookEventResponse{}, fmt.Errorf("store is required")
	}
	req.Provider = strings.TrimSpace(req.Provider)
	req.EventType = strings.TrimSpace(req.EventType)
	if req.Provider == "" || req.EventType == "" {
		return WebhookEventResponse{}, fmt.Errorf("provider and event_type are required")
	}
	event, err := s.store.SaveWebhookEvent(ctx, req)
	if err != nil {
		return WebhookEventResponse{}, err
	}
	_ = s.store.InsertAudit(ctx, "webhook.received", "webhook_event", event.ID, "接收代码仓库 Webhook", map[string]any{
		"provider":    event.Provider,
		"event_type":  event.EventType,
		"delivery_id": event.DeliveryID,
		"repository":  event.Repository,
	})
	triggeredRuns, err := s.triggerWebhookBuildRoutes(ctx, req, event)
	if err != nil {
		return WebhookEventResponse{}, err
	}
	message := "Webhook 事件已记录"
	if len(triggeredRuns) > 0 {
		message = fmt.Sprintf("Webhook 事件已记录，已触发 %d 个构建任务", len(triggeredRuns))
	}
	return WebhookEventResponse{OK: true, Event: event, TriggeredRuns: triggeredRuns, MessageZh: message}, nil
}

func (s *Service) triggerWebhookBuildRoutes(ctx context.Context, req WebhookEventRequest, event WebhookEventAdmin) ([]BuildCenterRunAdmin, error) {
	if strings.TrimSpace(req.Repository) == "" {
		return nil, nil
	}
	routeStore, ok := s.store.(BuildCenterWebhookStore)
	if !ok {
		return nil, nil
	}
	routes, err := routeStore.MatchWebhookBuildRoutes(ctx, req)
	if err != nil {
		return nil, err
	}
	triggeredRuns := make([]BuildCenterRunAdmin, 0, len(routes))
	for _, route := range routes {
		if !webhookRouteAllowsEvent(route, req) || !webhookRefMatches(route.RefPattern, req.Ref) {
			continue
		}
		runReq := BuildCenterRunRequest{
			ProfileKey: route.ProfileKey,
			Action:     route.Action,
			GitRef:     buildRefFromWebhookRef(req.Ref),
			StartedBy:  firstNonBlank(req.Sender, "webhook"),
		}
		resp, err := s.CreateBuildCenterRun(ctx, route.ProjectKey, runReq)
		if err != nil {
			return triggeredRuns, err
		}
		run := resp.Run
		_ = s.store.InsertAudit(ctx, "build_center.webhook_trigger", "build_center_run", run.ID, "Webhook 触发构建任务", map[string]any{
			"webhook_event_id": event.ID,
			"route_id":         route.ID,
			"provider":         req.Provider,
			"repository":       req.Repository,
			"event_type":       req.EventType,
			"ref":              req.Ref,
			"commit_sha":       req.CommitSHA,
		})
		triggeredRuns = append(triggeredRuns, run)
	}
	return triggeredRuns, nil
}

func buildRefFromWebhookRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if strings.HasPrefix(ref, "refs/heads/") {
		return strings.TrimPrefix(ref, "refs/heads/")
	}
	if strings.HasPrefix(ref, "refs/tags/") {
		return strings.TrimPrefix(ref, "refs/tags/")
	}
	return ref
}

func (s *Service) CreateRelease(ctx context.Context, req CreateReleaseRequest) (AdminActionResponse, error) {
	if s.store == nil {
		return AdminActionResponse{}, fmt.Errorf("store is required")
	}
	req.BuildID = strings.TrimSpace(req.BuildID)
	req.Channel = normalizeChannel(req.Channel, s.cfg.Channel)
	req.Title = strings.TrimSpace(req.Title)
	req.UpdateLevel = normalizeUpdateLevel(req.UpdateLevel)
	req.RolloutPercentage = normalizeRollout(req.RolloutPercentage)
	req.TargetType = normalizeTargetType(req.TargetType)
	req.TargetValue = strings.TrimSpace(req.TargetValue)
	if req.BuildID == "" || req.Title == "" {
		return AdminActionResponse{}, fmt.Errorf("build_id and title are required")
	}
	if req.TargetType != "all" && req.TargetValue == "" {
		return AdminActionResponse{}, fmt.Errorf("target_value is required for %s", req.TargetType)
	}
	release, err := s.store.CreateRelease(ctx, req)
	if err != nil {
		return AdminActionResponse{}, err
	}
	_ = s.store.InsertAudit(ctx, "release.create", "app_release", release.ID, "创建 App 发布", map[string]any{
		"version_name": release.VersionName,
		"channel":      release.Channel,
		"status":       release.Status,
	})
	resp := AdminActionResponse{OK: true, Release: release, MessageZh: "发布已创建"}
	if plan := s.syncAndroidReleasePlanOrAudit(ctx, release, "create"); plan != nil {
		resp.ReleasePlan = plan
		resp.MessageZh = "发布已创建并同步 Android 发布计划"
	}
	return resp, nil
}

func (s *Service) ReleaseAction(ctx context.Context, id, action string) (AdminActionResponse, error) {
	status := ""
	publish := false
	switch action {
	case "publish", "rollback":
		status = "released"
		publish = true
	case "pause":
		status = "paused"
	case "recall", "unpublish":
		status = "recalled"
	default:
		return AdminActionResponse{}, fmt.Errorf("unsupported release action: %s", action)
	}
	release, err := s.store.UpdateReleaseStatus(ctx, strings.TrimSpace(id), status, publish)
	if err != nil {
		return AdminActionResponse{}, err
	}
	_ = s.store.InsertAudit(ctx, "release."+action, "app_release", release.ID, "更新 App 发布状态", map[string]any{
		"status": status,
	})
	resp := AdminActionResponse{OK: true, Release: release, MessageZh: "发布状态已更新"}
	if plan := s.syncAndroidReleasePlanOrAudit(ctx, release, action); plan != nil {
		resp.ReleasePlan = plan
		resp.MessageZh = "发布状态已更新并同步 Android 发布计划"
	}
	return resp, nil
}

func (s *Service) UpdateReleaseRollout(ctx context.Context, id string, req UpdateRolloutRequest) (AdminActionResponse, error) {
	release, err := s.store.UpdateReleaseRollout(ctx, strings.TrimSpace(id), normalizeRollout(req.RolloutPercentage))
	if err != nil {
		return AdminActionResponse{}, err
	}
	_ = s.store.InsertAudit(ctx, "release.rollout", "app_release", release.ID, "调整 App 发布灰度", map[string]any{
		"rollout_percentage": release.RolloutPercentage,
	})
	resp := AdminActionResponse{OK: true, Release: release, MessageZh: "灰度比例已更新"}
	if plan := s.syncAndroidReleasePlanOrAudit(ctx, release, "rollout"); plan != nil {
		resp.ReleasePlan = plan
		resp.MessageZh = "灰度比例已更新并同步 Android 发布计划"
	}
	return resp, nil
}

func (s *Service) UpdateReleaseNotes(ctx context.Context, id string, req UpdateNotesRequest) (AdminActionResponse, error) {
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return AdminActionResponse{}, fmt.Errorf("title is required")
	}
	release, err := s.store.UpdateReleaseNotes(ctx, strings.TrimSpace(id), req)
	if err != nil {
		return AdminActionResponse{}, err
	}
	_ = s.store.InsertAudit(ctx, "release.notes", "app_release", release.ID, "更新 App 发布说明", nil)
	resp := AdminActionResponse{OK: true, Release: release, MessageZh: "发布说明已更新"}
	if plan := s.syncAndroidReleasePlanOrAudit(ctx, release, "notes"); plan != nil {
		resp.ReleasePlan = plan
		resp.MessageZh = "发布说明已更新并同步 Android 发布计划"
	}
	return resp, nil
}

func (s *Service) syncAndroidReleasePlanOrAudit(ctx context.Context, release AppReleaseAdmin, sourceAction string) *ReleasePlanAdmin {
	plan, err := s.syncAndroidReleasePlan(ctx, release, sourceAction)
	if err == nil {
		return plan
	}
	s.insertAudit(ctx, "release_plan.android_mirror_failed", "app_release", release.ID, "Android 发布计划同步失败", map[string]any{
		"source_action": sourceAction,
		"build_id":      release.BuildID,
		"version_name":  release.VersionName,
		"build_number":  release.BuildNumber,
		"channel":       release.Channel,
		"error":         err.Error(),
	})
	return nil
}

func (s *Service) syncAndroidReleasePlan(ctx context.Context, release AppReleaseAdmin, sourceAction string) (*ReleasePlanAdmin, error) {
	if strings.TrimSpace(release.ID) == "" || strings.TrimSpace(release.BuildID) == "" {
		return nil, nil
	}
	if _, ok := s.store.(ReleasePlanStore); !ok {
		return nil, nil
	}
	buildStore, ok := s.store.(AppBuildLookupStore)
	if !ok {
		return nil, nil
	}
	build, err := buildStore.AppBuild(ctx, release.BuildID)
	if err != nil {
		return nil, err
	}
	projectKey := legacyAndroidReleaseProjectKey()
	unitKey := legacyAndroidReleaseUnitKey(s.cfg, release)
	enabled := true
	if _, err := s.CreateReleaseUnit(ctx, CreateReleaseUnitRequest{
		ProjectKey:     projectKey,
		AppID:          release.AppID,
		UnitKey:        unitKey,
		Name:           legacyAndroidReleaseUnitName(s.cfg, release),
		UnitType:       "android",
		DefaultChannel: firstNonBlank(release.Channel, s.cfg.Channel, "dev"),
		Enabled:        &enabled,
		Metadata: map[string]any{
			"source":       "legacy_app_release",
			"app_key":      s.cfg.AppKey,
			"package_name": release.PackageName,
		},
	}); err != nil {
		return nil, err
	}
	resp, err := s.CreateReleasePlan(ctx, CreateReleasePlanRequest{
		ProjectKey:        projectKey,
		UnitKey:           unitKey,
		EnvironmentKey:    legacyAndroidReleaseEnvironment(release.Channel),
		PlanKey:           legacyAndroidReleasePlanKey(release),
		Title:             firstNonBlank(release.Title, fmt.Sprintf("Android %s", release.VersionName)),
		Description:       release.Summary,
		VersionName:       release.VersionName,
		BuildNumber:       release.BuildNumber,
		GitCommit:         release.GitCommit,
		Channel:           firstNonBlank(release.Channel, s.cfg.Channel),
		Status:            normalizeReleasePlanStatus(release.Status),
		RolloutPercentage: normalizeRollout(release.RolloutPercentage),
		TargetType:        normalizeTargetType(release.TargetType),
		TargetValue:       release.TargetValue,
		CreatedBy:         firstNonBlank(release.CreatedBy, "legacy_app_release"),
		Artifacts:         []ReleasePlanArtifactRequest{legacyAndroidReleaseArtifact(release, build)},
		Metadata: map[string]any{
			"source":         "legacy_app_release",
			"source_action":  sourceAction,
			"app_release_id": release.ID,
			"update_level":   release.UpdateLevel,
			"version_code":   release.VersionCode,
		},
	})
	if err != nil {
		return nil, err
	}
	return &resp.Plan, nil
}

func legacyAndroidReleaseProjectKey() string {
	return "release-center"
}

func legacyAndroidReleaseUnitKey(cfg Config, release AppReleaseAdmin) string {
	return safeFilePart(firstNonBlank(cfg.AppKey, release.PackageName, "android-app"))
}

func legacyAndroidReleaseUnitName(cfg Config, release AppReleaseAdmin) string {
	return firstNonBlank(cfg.Name, release.PackageName, "Android App")
}

func legacyAndroidReleaseEnvironment(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "dev", "develop", "development", "alpha", "canary":
		return "dev"
	case "test", "testing", "qa":
		return "test"
	case "staging", "stage", "pre", "preview", "beta":
		return "staging"
	default:
		return "prod"
	}
}

func legacyAndroidReleasePlanKey(release AppReleaseAdmin) string {
	return "android-apk-" + safeFilePart(firstNonBlank(release.ID, fmt.Sprintf("%s-%d-%s", release.VersionName, release.BuildNumber, release.Channel)))
}

func legacyAndroidReleaseArtifact(release AppReleaseAdmin, build AppBuildJob) ReleasePlanArtifactRequest {
	artifact := preferredAndroidBuildArtifact(build)
	artifactName := firstNonBlank(artifact.Name, release.FileName, build.ArtifactType, "apk")
	artifactType := firstNonBlank(artifact.ArtifactType, build.ArtifactType, "apk")
	artifactPath := firstNonBlank(artifact.ArtifactPath, release.ArtifactPath, release.APKPath, release.DownloadURL, build.ArtifactPath)
	fileName := firstNonBlank(artifact.FileName, release.FileName, build.FileName)
	if fileName == "" {
		if base := path.Base(artifactPath); base != "." && base != "/" {
			fileName = base
		}
	}
	immutableRef := "app_build:" + build.ID
	if artifact.ID != "" {
		immutableRef = "app_build_artifact:" + artifact.ID
	}
	return ReleasePlanArtifactRequest{
		AppBuildID:         build.ID,
		AppBuildArtifactID: artifact.ID,
		ArtifactName:       artifactName,
		ArtifactType:       artifactType,
		FileName:           fileName,
		ImmutableRef:       immutableRef,
		Metadata: map[string]any{
			"source":         "legacy_app_release",
			"app_release_id": release.ID,
			"artifact_path":  artifactPath,
			"download_url":   firstNonBlank(release.DownloadURL, artifactPath),
			"sha256":         firstNonBlank(artifact.SHA256, release.SHA256, build.SHA256),
			"size_bytes":     firstPositiveInt64(artifact.SizeBytes, release.SizeBytes, build.ArtifactSize),
			"version_code":   release.VersionCode,
		},
	}
}

func preferredAndroidBuildArtifact(build AppBuildJob) AppBuildArtifact {
	if len(build.Artifacts) == 0 {
		return AppBuildArtifact{}
	}
	for _, artifact := range build.Artifacts {
		switch strings.ToLower(strings.TrimSpace(artifact.ArtifactType)) {
		case "apk", "aab":
			return artifact
		}
	}
	return build.Artifacts[0]
}

func (s *Service) CreateResourceVersion(ctx context.Context, req CreateResourceVersionRequest, uploads []UploadedResourcePackage, blobs BlobStore) (ResourceActionResponse, error) {
	if s.store == nil {
		return ResourceActionResponse{}, fmt.Errorf("store is required")
	}
	if blobs == nil {
		return ResourceActionResponse{}, fmt.Errorf("blob store is required")
	}
	req.ResourceVersion = strings.TrimSpace(req.ResourceVersion)
	req.Channel = normalizeChannel(req.Channel, s.cfg.Channel)
	req.Title = strings.TrimSpace(req.Title)
	req.UpdateLevel = normalizeUpdateLevel(req.UpdateLevel)
	req.RolloutPercentage = normalizeRollout(req.RolloutPercentage)
	if req.ResourceVersion == "" || req.Title == "" {
		return ResourceActionResponse{}, fmt.Errorf("resource_version and title are required")
	}
	if len(uploads) == 0 {
		return ResourceActionResponse{}, fmt.Errorf("at least one resource package is required")
	}
	packages := make([]AppResourcePackageAdmin, 0, len(uploads))
	for _, upload := range uploads {
		packageKey := strings.TrimSpace(upload.PackageKey)
		if packageKey == "" {
			packageKey = packageKeyFromFilename(upload.FileName, req.ResourceVersion)
		}
		packageKey = safeFilePart(packageKey)
		if err := ValidateIncrementalResourceArchive(packageKey, upload.Content); err != nil {
			return ResourceActionResponse{}, err
		}
		sum := sha256.Sum256(upload.Content)
		storageKey, size, err := blobs.Save(ctx, "app-resources/packages/"+req.ResourceVersion, upload.FileName, bytes.NewReader(upload.Content))
		if err != nil {
			return ResourceActionResponse{}, err
		}
		packages = append(packages, AppResourcePackageAdmin{
			ID:          uuid.NewString(),
			PackageKey:  packageKey,
			PackageType: "zip",
			FileSize:    size,
			SHA256:      hex.EncodeToString(sum[:]),
			StorageKey:  storageKey,
			CreatedAt:   time.Now(),
		})
		packages[len(packages)-1].FileURL = "/api/v1/app/resources/packages/" + packages[len(packages)-1].ID + "/download"
	}
	manifestBytes, err := BuildResourceManifest(ResourceManifestRequest{
		AppKey:               s.cfg.AppKey,
		ResourceVersion:      req.ResourceVersion,
		Channel:              req.Channel,
		UpdateLevel:          req.UpdateLevel,
		MinAppVersionCode:    req.MinAppVersionCode,
		MaxAppVersionCode:    req.MaxAppVersionCode,
		Title:                req.Title,
		Summary:              req.Summary,
		ReleaseNotesMarkdown: req.ReleaseNotesMarkdown,
		ManifestPrivateKey:   s.cfg.ManifestPrivateKey,
	}, packages)
	if err != nil {
		return ResourceActionResponse{}, err
	}
	manifestName := "manifest-" + safeFilePart(req.ResourceVersion) + ".json"
	manifestKey, _, err := blobs.Save(ctx, "app-resources/manifests", manifestName, bytes.NewReader(manifestBytes))
	if err != nil {
		return ResourceActionResponse{}, err
	}
	resource, err := s.store.CreateResourceVersion(ctx, req, packages, manifestKey, s.cfg)
	if err != nil {
		return ResourceActionResponse{}, err
	}
	_ = s.store.InsertAudit(ctx, "resource.create", "app_resource_version", resource.ID, "创建资源增量发布", map[string]any{
		"resource_version": resource.ResourceVersion,
		"channel":          resource.Channel,
		"packages":         len(resource.Packages),
	})
	return ResourceActionResponse{OK: true, ResourceVersion: resource, MessageZh: "资源版本已创建"}, nil
}

func (s *Service) ResourceAction(ctx context.Context, id, action string) (ResourceActionResponse, error) {
	status := ""
	publish := false
	switch action {
	case "publish", "rollback":
		status = "released"
		publish = true
	case "pause":
		status = "paused"
	case "recall", "unpublish":
		status = "recalled"
	default:
		return ResourceActionResponse{}, fmt.Errorf("unsupported resource action: %s", action)
	}
	resource, err := s.store.UpdateResourceStatus(ctx, strings.TrimSpace(id), status, publish)
	if err != nil {
		return ResourceActionResponse{}, err
	}
	_ = s.store.InsertAudit(ctx, "resource."+action, "app_resource_version", resource.ID, "更新资源发布状态", map[string]any{
		"status": status,
	})
	return ResourceActionResponse{OK: true, ResourceVersion: resource, MessageZh: "资源状态已更新"}, nil
}

func (s *Service) UpdateResourceRollout(ctx context.Context, id string, req UpdateRolloutRequest) (ResourceActionResponse, error) {
	resource, err := s.store.UpdateResourceRollout(ctx, strings.TrimSpace(id), normalizeRollout(req.RolloutPercentage))
	if err != nil {
		return ResourceActionResponse{}, err
	}
	_ = s.store.InsertAudit(ctx, "resource.rollout", "app_resource_version", resource.ID, "调整资源灰度", map[string]any{
		"rollout_percentage": resource.RolloutPercentage,
	})
	return ResourceActionResponse{OK: true, ResourceVersion: resource, MessageZh: "资源灰度已更新"}, nil
}

func (s *Service) UpdateResourceNotes(ctx context.Context, id string, req UpdateNotesRequest) (ResourceActionResponse, error) {
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return ResourceActionResponse{}, fmt.Errorf("title is required")
	}
	resource, err := s.store.UpdateResourceNotes(ctx, strings.TrimSpace(id), req)
	if err != nil {
		return ResourceActionResponse{}, err
	}
	_ = s.store.InsertAudit(ctx, "resource.notes", "app_resource_version", resource.ID, "更新资源发布说明", nil)
	return ResourceActionResponse{OK: true, ResourceVersion: resource, MessageZh: "资源说明已更新"}, nil
}

func normalizeChannel(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	if value == "" {
		return "dev"
	}
	return value
}

func normalizeBuildType(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	if value == "" {
		return "debug"
	}
	return value
}

func normalizeBuildStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "building", "failed", "archived":
		return strings.TrimSpace(value)
	default:
		return "success"
	}
}

func normalizeArtifactType(value string) string {
	switch strings.TrimSpace(value) {
	case "apk", "aab", "zip", "web_dist", "binary", "docker_image", "artifact":
		return strings.TrimSpace(value)
	default:
		return "apk"
	}
}

func normalizeUpdateLevel(value string) string {
	switch strings.TrimSpace(value) {
	case "recommended", "forced":
		return strings.TrimSpace(value)
	default:
		return "normal"
	}
}

func normalizeTargetType(value string) string {
	switch strings.TrimSpace(value) {
	case "device_id", "user_id", "user_group":
		return strings.TrimSpace(value)
	default:
		return "all"
	}
}

func normalizeRollout(value int) int {
	if value <= 0 {
		return 100
	}
	if value > 100 {
		return 100
	}
	return value
}

func normalizePlatform(value string) string {
	switch strings.TrimSpace(value) {
	case "ios":
		return "ios"
	default:
		return "android"
	}
}

func firstPositiveInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func shortCommitFromRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if len(ref) >= 7 && regexp.MustCompile(`^[a-fA-F0-9]{7,64}$`).MatchString(ref) {
		return ref
	}
	return ""
}

func apkFileName(appKey, versionName, channel string, buildNumber int) string {
	appKey = firstNonBlank(appKey, "game-helper")
	return safeFilePart(appKey) + "-" + safeFilePart(versionName) + "-" + safeFilePart(channel) + "-build" + strconv.Itoa(buildNumber) + ".apk"
}

func packageKeyFromFilename(fileName, resourceVersion string) string {
	base := filepath.Base(fileName)
	ext := filepath.Ext(base)
	base = strings.TrimSuffix(base, ext)
	base = strings.TrimSpace(base)
	versionSuffix := "-" + safeFilePart(resourceVersion)
	if strings.HasSuffix(safeFilePart(base), versionSuffix) {
		base = strings.TrimSuffix(safeFilePart(base), versionSuffix)
	}
	if base == "" {
		return "resource-package"
	}
	candidate := safeFilePart(base)
	if _, ok := incrementalResourcePolicy(candidate); ok {
		return candidate
	}
	for _, policy := range IncrementalResourcePackageCatalog() {
		if strings.HasPrefix(candidate, policy.PackageKey+"-") || strings.HasSuffix(candidate, "-"+policy.PackageKey) {
			return policy.PackageKey
		}
	}
	return candidate
}

func safeFilePart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	re := regexp.MustCompile(`[^a-z0-9._-]+`)
	value = re.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-._")
	if value == "" {
		return "file"
	}
	return value
}
