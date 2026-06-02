package appreleases

import (
	"context"
	"fmt"
	"path"
	"strings"
)

type Service struct {
	cfg   Config
	store Store
}

func NewService(cfg Config) *Service {
	return &Service{cfg: normalizeConfig(cfg)}
}

func NewServiceWithStore(cfg Config, store Store) *Service {
	return &Service{cfg: normalizeConfig(cfg), store: store}
}

func (s *Service) SyncConfiguredRelease(ctx context.Context) error {
	if s.store == nil {
		return nil
	}
	return s.store.EnsureConfiguredRelease(ctx, s.cfg)
}

func (s *Service) Check(ctx context.Context, req CheckRequest) CheckResponse {
	req = normalizeCheckRequest(req, s.cfg)
	cfg := s.cfg
	if s.store != nil {
		if latest, err := s.store.FindLatestRelease(ctx, req, s.cfg); err == nil {
			cfg = latest
		}
		_ = s.store.UpsertInstallation(ctx, cfg, req)
	}
	return checkWithConfig(cfg, req)
}

func (s *Service) ResourceCheck(ctx context.Context, req ResourceCheckRequest) ResourceCheckResponse {
	req = normalizeResourceCheckRequest(req, s.cfg)
	if s.store == nil {
		return noResourceUpdate(req)
	}
	candidate, err := s.store.FindLatestResource(ctx, req, s.cfg)
	if err != nil || candidate.ResourceVersion == "" {
		return noResourceUpdate(req)
	}
	current := firstNonBlank(req.ResourceVersion, req.CurrentResourceVer)
	versionCode := effectiveResourceAppVersionCode(req)
	hasUpdate := current == "" || candidate.ResourceVersion != current
	force := candidate.UpdateLevel == "forced"
	if candidate.MinAppVersionCode > 0 && versionCode > 0 && versionCode < candidate.MinAppVersionCode {
		force = true
	}
	message := "运行资源可以继续使用"
	if hasUpdate {
		message = "发现新的运行资源"
	}
	if force {
		message = "需要更新运行资源后继续执行任务"
	}
	return ResourceCheckResponse{
		HasUpdate:             hasUpdate,
		HasUpdateSnake:        hasUpdate,
		UpdateLevel:           candidate.UpdateLevel,
		LatestResourceVersion: candidate.ResourceVersion,
		Title:                 candidate.Title,
		Summary:               candidate.Summary,
		ReleaseNotesMarkdown:  candidate.ReleaseNotesMarkdown,
		ManifestURL:           candidate.ManifestURL,
		TotalSize:             candidate.TotalSize,
		ForceUpdate:           force,
		BlockTaskExecution:    force,
		MessageZh:             message,
		Packages:              candidate.Packages,
	}
}

func (s *Service) SaveUpdateEvent(ctx context.Context, req UpdateEventRequest) (UpdateEventResponse, error) {
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	if s.store != nil {
		if err := s.store.SaveUpdateEvent(ctx, s.cfg, req); err != nil {
			return UpdateEventResponse{}, err
		}
	}
	return UpdateEventResponse{Status: "ok", MessageZh: "更新事件已接收"}, nil
}

func (s *Service) SaveHeartbeat(ctx context.Context, req HeartbeatRequest) (HeartbeatResponse, error) {
	req = normalizeHeartbeatRequest(req, s.cfg)
	if firstNonBlank(req.DeviceID, req.DeviceKey) == "" {
		return HeartbeatResponse{}, fmt.Errorf("deviceId is required")
	}
	if s.store != nil {
		if err := s.store.SaveHeartbeat(ctx, s.cfg, req); err != nil {
			return HeartbeatResponse{}, err
		}
	}
	return HeartbeatResponse{Status: "ok", MessageZh: "设备心跳已接收"}, nil
}

func (s *Service) TaskPreflight(ctx context.Context, req TaskPreflightRequest) TaskPreflightResponse {
	checkReq, resourceReq := normalizeTaskPreflightRequest(req, s.cfg)
	appUpdate := s.Check(ctx, checkReq)
	resourceUpdate := s.ResourceCheck(ctx, resourceReq)
	canExecute := !appUpdate.BlockTaskExecution && !resourceUpdate.BlockTaskExecution
	blockReason := ""
	message := "任务可以执行"
	if appUpdate.BlockTaskExecution {
		blockReason = "app_update_required"
		message = appUpdate.MessageZh
	}
	if blockReason == "" && resourceUpdate.BlockTaskExecution {
		blockReason = "resource_update_required"
		message = resourceUpdate.MessageZh
	}
	return TaskPreflightResponse{
		CanExecute:        canExecute,
		BlockReason:       blockReason,
		MessageZh:         message,
		AppUpdate:         appUpdate,
		ResourceUpdate:    resourceUpdate,
		AppUpdateRequired: appUpdate.BlockTaskExecution,
		ResourceRequired:  resourceUpdate.BlockTaskExecution,
	}
}

func (s *Service) Overview(ctx context.Context) (ReleaseOverview, error) {
	if s.store == nil {
		return ReleaseOverview{
			Apps: []AppSummary{{
				AppKey:      s.cfg.AppKey,
				Name:        s.cfg.Name,
				Platform:    "android",
				PackageName: s.cfg.PackageName,
				Enabled:     true,
			}},
			Releases: []ReleaseSummary{{
				AppKey:                  s.cfg.AppKey,
				VersionName:             s.cfg.LatestVersionName,
				VersionCode:             s.cfg.LatestVersionCode,
				BuildNumber:             effectiveBuildNumber(s.cfg),
				Channel:                 s.cfg.Channel,
				BuildType:               s.cfg.BuildType,
				Status:                  "released",
				UpdateLevel:             configuredUpdateLevel(s.cfg),
				MinSupportedVersionCode: s.cfg.MinSupportedVersionCode,
				DownloadURL:             firstNonBlank(s.cfg.DownloadURL, s.cfg.APKURL),
				SHA256:                  s.cfg.SHA256,
				SizeBytes:               s.cfg.SizeBytes,
			}},
			MessageZh: "App 发版概览已读取",
		}, nil
	}
	overview, err := s.store.Overview(ctx)
	if err != nil {
		return ReleaseOverview{}, err
	}
	overview.MessageZh = "App 发版概览已读取"
	return overview, nil
}

func (s *Service) CheckLegacy(req CheckRequest) CheckResponse {
	return checkWithConfig(s.cfg, normalizeCheckRequest(req, s.cfg))
}

func checkWithConfig(cfg Config, req CheckRequest) CheckResponse {
	latestCode := cfg.LatestVersionCode
	currentCode := effectiveCheckVersionCode(req)
	hasUpdate := latestCode > 0 && currentCode > 0 && latestCode > currentCode
	versionAvailable := cfg.CurrentVersionAvailable
	if cfg.MinSupportedVersionCode > 0 && currentCode > 0 && currentCode < cfg.MinSupportedVersionCode {
		versionAvailable = false
	}
	forceUpdate := cfg.ForceUpdate && hasUpdate
	if cfg.MinSupportedVersionCode > 0 && currentCode > 0 && currentCode < cfg.MinSupportedVersionCode {
		forceUpdate = true
	}
	if !versionAvailable {
		forceUpdate = true
	}
	updateLevel := "normal"
	if forceUpdate {
		updateLevel = "forced"
	} else if hasUpdate {
		updateLevel = "recommended"
	}
	blockTaskExecution := forceUpdate || !versionAvailable

	message := ""
	if hasUpdate || forceUpdate || !versionAvailable {
		message = cfg.MessageZh
	}
	if message == "" {
		switch {
		case !versionAvailable:
			message = "当前版本不可用，请更新后继续使用"
		case forceUpdate:
			message = "发现必须安装的新版本"
		case hasUpdate:
			message = "发现新版本"
		default:
			message = "当前版本可以继续使用"
		}
	}

	return CheckResponse{
		HasUpdate:               hasUpdate,
		UpdateAvailable:         hasUpdate,
		UpdateLevel:             updateLevel,
		BlockTaskExecution:      blockTaskExecution,
		VersionName:             cfg.LatestVersionName,
		LatestVersionName:       cfg.LatestVersionName,
		VersionCode:             latestCode,
		LatestVersionCode:       latestCode,
		BuildNumber:             effectiveBuildNumber(cfg),
		Channel:                 firstNonBlank(req.Channel, cfg.Channel),
		BuildType:               firstNonBlank(req.BuildType, cfg.BuildType),
		APKURL:                  cfg.APKURL,
		DownloadURL:             cfg.DownloadURL,
		SHA256:                  cfg.SHA256,
		SizeBytes:               cfg.SizeBytes,
		ForceUpdate:             forceUpdate,
		CurrentVersionAvailable: versionAvailable,
		MinSupportedVersionCode: cfg.MinSupportedVersionCode,
		ReleaseNotes:            cfg.ReleaseNotes,
		MessageZh:               message,
		UnavailableReason:       cfg.UnavailableReason,
		FileName:                firstNonBlank(cfg.FileName, path.Base(firstNonBlank(cfg.DownloadURL, cfg.APKURL))),
	}
}

func normalizeConfig(cfg Config) Config {
	if strings.TrimSpace(cfg.AppKey) == "" {
		cfg.AppKey = "game-helper-android"
	}
	if strings.TrimSpace(cfg.Name) == "" {
		cfg.Name = "游戏助手"
	}
	if strings.TrimSpace(cfg.PackageName) == "" {
		cfg.PackageName = "com.kingdomhelper.executor"
	}
	if strings.TrimSpace(cfg.Channel) == "" {
		cfg.Channel = "dev"
	}
	if strings.TrimSpace(cfg.BuildType) == "" {
		cfg.BuildType = "debug"
	}
	if cfg.BuildNumber <= 0 {
		cfg.BuildNumber = cfg.LatestVersionCode
	}
	if !cfg.CurrentVersionAvailable && strings.TrimSpace(cfg.UnavailableReason) == "" {
		cfg.UnavailableReason = "当前版本已被后台标记为不可用"
	}
	cfg.QualityPolicy = normalizeQualityPolicy(cfg.QualityPolicy)
	return cfg
}

func normalizeQualityPolicy(policy QualityPolicy) QualityPolicy {
	if policy.ResourceActivationFailedCount <= 0 {
		policy.ResourceActivationFailedCount = 3
	}
	if policy.ResourceFailureRate <= 0 {
		policy.ResourceFailureRate = 5
	}
	if policy.APKChecksumFailedCount <= 0 {
		policy.APKChecksumFailedCount = 3
	}
	if policy.APKInstallFailedCount <= 0 {
		policy.APKInstallFailedCount = 10
	}
	if policy.APKInstallFailureRate <= 0 {
		policy.APKInstallFailureRate = 10
	}
	return policy
}

func normalizeCheckRequest(req CheckRequest, cfg Config) CheckRequest {
	req.AppKey = firstNonBlank(req.AppKey, req.AppKeySnake, cfg.AppKey)
	req.DeviceID = firstNonBlank(req.DeviceID, req.DeviceIDSnake)
	req.DeviceKey = firstNonBlank(req.DeviceKey, req.DeviceKeySnake)
	req.OSVersion = firstNonBlank(req.OSVersion, req.OSVersionSnake)
	req.DeviceModel = firstNonBlank(req.DeviceModel, req.DeviceModelSnake)
	req.PackageName = firstNonBlank(req.PackageName, req.PackageNameCamel, cfg.PackageName)
	req.VersionName = firstNonBlank(req.VersionName, req.VersionNameSnake)
	req.Channel = firstNonBlank(req.Channel, cfg.Channel)
	req.BuildType = firstNonBlank(req.BuildType, req.BuildTypeCamel, cfg.BuildType)
	if req.VersionCode <= 0 {
		req.VersionCode = req.VersionCodeSnake
	}
	if req.AppVersionCode <= 0 {
		req.AppVersionCode = req.AppVersionCodeSnake
	}
	if req.BuildNumber <= 0 {
		req.BuildNumber = req.BuildNumberSnake
	}
	return req
}

func normalizeResourceCheckRequest(req ResourceCheckRequest, cfg Config) ResourceCheckRequest {
	req.AppKey = firstNonBlank(req.AppKey, req.AppKeySnake, cfg.AppKey)
	req.DeviceID = firstNonBlank(req.DeviceID, req.DeviceIDSnake)
	req.DeviceKey = firstNonBlank(req.DeviceKey, req.DeviceKeySnake)
	req.PackageName = firstNonBlank(req.PackageName, req.PackageNameCamel, cfg.PackageName)
	req.ResourceVersion = firstNonBlank(req.ResourceVersion, req.ResourceVersionSnake, req.CurrentResourceVer)
	req.Channel = firstNonBlank(req.Channel, cfg.Channel)
	if req.VersionCode <= 0 {
		req.VersionCode = req.VersionCodeSnake
	}
	if req.AppVersionCode <= 0 {
		req.AppVersionCode = req.AppVersionCodeSnake
	}
	return req
}

func normalizeHeartbeatRequest(req HeartbeatRequest, cfg Config) HeartbeatRequest {
	req.AppKey = firstNonBlank(req.AppKey, req.AppKeySnake, cfg.AppKey)
	req.DeviceID = firstNonBlank(req.DeviceID, req.DeviceIDSnake)
	req.DeviceKey = firstNonBlank(req.DeviceKey, req.DeviceKeySnake)
	req.OSVersion = firstNonBlank(req.OSVersion, req.OSVersionSnake)
	req.DeviceModel = firstNonBlank(req.DeviceModel, req.DeviceModelSnake)
	req.DeviceName = firstNonBlank(req.DeviceName, req.DeviceNameSnake)
	req.PackageName = firstNonBlank(req.PackageName, req.PackageNameCamel, cfg.PackageName)
	req.VersionName = firstNonBlank(req.VersionName, req.VersionNameSnake)
	req.ResourceVersion = firstNonBlank(req.ResourceVersion, req.ResourceVersionSnake)
	if strings.TrimSpace(req.Platform) == "" {
		req.Platform = "android"
	}
	if req.VersionCode <= 0 {
		req.VersionCode = req.VersionCodeSnake
	}
	if req.AppVersionCode <= 0 {
		req.AppVersionCode = req.AppVersionCodeSnake
	}
	if req.BuildNumber <= 0 {
		req.BuildNumber = req.BuildNumberSnake
	}
	return req
}

func normalizeTaskPreflightRequest(req TaskPreflightRequest, cfg Config) (CheckRequest, ResourceCheckRequest) {
	checkReq := normalizeCheckRequest(CheckRequest{
		AppKey:              req.AppKey,
		AppKeySnake:         req.AppKeySnake,
		DeviceID:            req.DeviceID,
		DeviceIDSnake:       req.DeviceIDSnake,
		DeviceKey:           req.DeviceKey,
		DeviceKeySnake:      req.DeviceKeySnake,
		UserID:              req.UserID,
		PackageName:         req.PackageName,
		PackageNameCamel:    req.PackageNameCamel,
		VersionName:         req.VersionName,
		VersionNameSnake:    req.VersionNameSnake,
		VersionCode:         req.VersionCode,
		VersionCodeSnake:    req.VersionCodeSnake,
		AppVersionCode:      req.AppVersionCode,
		AppVersionCodeSnake: req.AppVersionCodeSnake,
		BuildNumber:         req.BuildNumber,
		BuildNumberSnake:    req.BuildNumberSnake,
		Channel:             req.Channel,
		BuildType:           req.BuildType,
		BuildTypeCamel:      req.BuildTypeCamel,
	}, cfg)
	resourceReq := normalizeResourceCheckRequest(ResourceCheckRequest{
		AppKey:               req.AppKey,
		AppKeySnake:          req.AppKeySnake,
		DeviceID:             req.DeviceID,
		DeviceIDSnake:        req.DeviceIDSnake,
		DeviceKey:            req.DeviceKey,
		DeviceKeySnake:       req.DeviceKeySnake,
		UserID:               req.UserID,
		PackageName:          req.PackageName,
		PackageNameCamel:     req.PackageNameCamel,
		VersionCode:          req.VersionCode,
		VersionCodeSnake:     req.VersionCodeSnake,
		AppVersionCode:       req.AppVersionCode,
		AppVersionCodeSnake:  req.AppVersionCodeSnake,
		ResourceVersion:      req.ResourceVersion,
		ResourceVersionSnake: req.ResourceVersionSnake,
		CurrentResourceVer:   req.CurrentResourceVer,
		Channel:              req.Channel,
	}, cfg)
	return checkReq, resourceReq
}

func noResourceUpdate(req ResourceCheckRequest) ResourceCheckResponse {
	current := firstNonBlank(req.ResourceVersion, req.CurrentResourceVer)
	return ResourceCheckResponse{
		HasUpdate:             false,
		HasUpdateSnake:        false,
		UpdateLevel:           "normal",
		LatestResourceVersion: current,
		TotalSize:             0,
		MessageZh:             "运行资源可以继续使用",
		Packages:              []ResourcePackage{},
	}
}

func effectiveCheckVersionCode(req CheckRequest) int {
	if req.AppVersionCode > 0 {
		return req.AppVersionCode
	}
	return req.VersionCode
}

func effectiveResourceAppVersionCode(req ResourceCheckRequest) int {
	if req.AppVersionCode > 0 {
		return req.AppVersionCode
	}
	return req.VersionCode
}

func effectiveBuildNumber(cfg Config) int {
	if cfg.BuildNumber > 0 {
		return cfg.BuildNumber
	}
	return cfg.LatestVersionCode
}

func configuredUpdateLevel(cfg Config) string {
	if cfg.ForceUpdate || !cfg.CurrentVersionAvailable {
		return "forced"
	}
	return "normal"
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
