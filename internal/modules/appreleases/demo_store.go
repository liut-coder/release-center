package appreleases

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type DemoStore struct {
	mu             sync.Mutex
	apps           []AppAdminSummary
	builds         []AppBuildJob
	releases       []AppReleaseAdmin
	resources      []AppResourceVersionAdmin
	installations  []AppInstallationAdmin
	apkEvents      []AppUpgradeEventAdmin
	resourceEvents []AppUpgradeEventAdmin
	webhookEvents  []WebhookEventAdmin
	auditLogs      []AppReleaseAuditLogAdmin
	system         SystemManagementOverview
}

func NewDemoStore() *DemoStore {
	now := time.Now().UTC()
	app := AppAdminSummary{
		ID:          "app_demo_game_helper",
		AppKey:      "game-helper-android",
		Name:        "游戏助手",
		Platform:    "android",
		PackageName: "com.kingdomhelper.executor",
		Description: "Demo 项目，用于本地验收发布中心闭环",
		Enabled:     true,
		CreatedAt:   now.Add(-48 * time.Hour),
		UpdatedAt:   now.Add(-2 * time.Hour),
	}
	build := AppBuildJob{
		ID:               "build_demo_001",
		Status:           "success",
		AppID:            app.ID,
		GitRef:           "main",
		GitCommit:        "demo001",
		GitBranch:        "main",
		BuildType:        "debug",
		Channel:          "dev",
		VersionName:      "0.1.0-dev.1",
		VersionCode:      1,
		BuildNumber:      1,
		BuildEnvironment: "dev",
		ArtifactType:     "apk",
		ArtifactPath:     "https://example.invalid/demo/game-helper-0.1.0-dev.1.apk",
		ArtifactSize:     42800000,
		SHA256:           strings.Repeat("a", 64),
		FileName:         "game-helper-0.1.0-dev.1.apk",
		APIBaseURL:       "http://127.0.0.1:18080/",
		ReleaseNotes:     "Demo 构建",
		StartedBy:        "demo",
		StartedAt:        now.Add(-4 * time.Hour),
		CreatedAt:        now.Add(-4 * time.Hour),
		FinishedAt:       now.Add(-4*time.Hour + time.Minute),
		DurationMS:       60000,
		LogTail:          []string{"Demo 构建已完成", "等待创建发布草稿"},
	}
	release := AppReleaseAdmin{
		ID:                   "release_demo_001",
		AppID:                app.ID,
		BuildID:              build.ID,
		PackageName:          app.PackageName,
		VersionName:          build.VersionName,
		VersionCode:          build.VersionCode,
		BuildNumber:          build.BuildNumber,
		Channel:              build.Channel,
		BuildType:            build.BuildType,
		GitRef:               build.GitRef,
		GitCommit:            build.GitCommit,
		APIBaseURL:           build.APIBaseURL,
		ArtifactPath:         build.ArtifactPath,
		FileName:             build.FileName,
		SizeBytes:            build.ArtifactSize,
		SHA256:               build.SHA256,
		Status:               "released",
		Title:                "Demo 版本更新",
		Summary:              "本地 demo 发布记录",
		ReleaseNotesMarkdown: "Demo 版本用于验证后台页面和 API 写入闭环。",
		UpdateLevel:          "normal",
		RolloutPercentage:    100,
		TargetType:           "all",
		TargetValue:          "all",
		IsLatest:             true,
		IsPublished:          true,
		PublishedAt:          now.Add(-3 * time.Hour),
		CreatedBy:            "demo",
		CreatedAt:            now.Add(-3 * time.Hour),
		UpdatedAt:            now.Add(-3 * time.Hour),
		DownloadURL:          build.ArtifactPath,
	}
	resource := AppResourceVersionAdmin{
		ID:                   "resource_demo_001",
		AppID:                app.ID,
		ResourceVersion:      "20260602.1",
		Channel:              "dev",
		Status:               "released",
		UpdateLevel:          "normal",
		Title:                "Demo 资源更新",
		Summary:              "识图模板和 OCR 词库 demo 更新",
		ReleaseNotesMarkdown: "Demo 资源用于验证增量资源发布链路。",
		RolloutPercentage:    100,
		ManifestURL:          "/api/v1/app/resources/resource_demo_001/manifest",
		TotalSize:            256,
		PublishedAt:          now.Add(-2 * time.Hour),
		CreatedBy:            "demo",
		CreatedAt:            now.Add(-2 * time.Hour),
		UpdatedAt:            now.Add(-2 * time.Hour),
		Packages: []AppResourcePackageAdmin{{
			ID:          "resource_pkg_demo_001",
			PackageKey:  "templates-common",
			PackageType: "zip",
			FileURL:     "/api/v1/app/resources/packages/resource_pkg_demo_001/download",
			FileSize:    256,
			SHA256:      strings.Repeat("b", 64),
			CreatedAt:   now.Add(-2 * time.Hour),
		}},
	}
	return &DemoStore{
		apps:      []AppAdminSummary{app},
		builds:    []AppBuildJob{build},
		releases:  []AppReleaseAdmin{release},
		resources: []AppResourceVersionAdmin{resource},
		installations: []AppInstallationAdmin{
			{ID: "install_demo_001", AppID: app.ID, UserID: "10001", DeviceID: "demo-device-1", DeviceName: "Pixel Demo", Platform: "android", OSVersion: "14", DeviceModel: "Pixel", InstalledVersion: "0.1.0-dev.1", InstalledCode: 1, BuildNumber: 1, ResourceVersion: "20260602.1", LastSeenAt: now.Add(-30 * time.Minute), LastUpgradeStatus: "install_success", CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now.Add(-30 * time.Minute)},
			{ID: "install_demo_002", AppID: app.ID, UserID: "10002", DeviceID: "demo-device-2", DeviceName: "QA Demo", Platform: "android", OSVersion: "13", DeviceModel: "Android", InstalledVersion: "0.0.9", InstalledCode: 0, BuildNumber: 0, ResourceVersion: "20260601.1", LastSeenAt: now.Add(-2 * time.Hour), LastUpgradeStatus: "download_success", CreatedAt: now.Add(-20 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
		},
		apkEvents: []AppUpgradeEventAdmin{
			{ID: "apk_event_demo_001", AppID: app.ID, ReleaseID: release.ID, UserID: "10001", DeviceID: "demo-device-1", FromVersionCode: 0, ToVersionCode: 1, FromVersion: "0.0.9", ToVersion: "0.1.0-dev.1", EventType: "install_success", CreatedAt: now.Add(-90 * time.Minute), Category: "apk"},
		},
		resourceEvents: []AppUpgradeEventAdmin{
			{ID: "resource_event_demo_001", AppID: app.ID, UserID: "10001", DeviceID: "demo-device-1", FromVersion: "20260601.1", ToVersion: "20260602.1", EventType: "activation_success", PackageKey: "templates-common", CreatedAt: now.Add(-80 * time.Minute), Category: "resource"},
		},
		auditLogs: []AppReleaseAuditLogAdmin{
			{ID: "audit_demo_001", AppID: app.ID, Operator: "demo", Action: "release.create", TargetType: "app_release", TargetID: release.ID, CreatedAt: now.Add(-3 * time.Hour)},
			{ID: "audit_demo_002", AppID: app.ID, Operator: "demo", Action: "resource.create", TargetType: "app_resource_version", TargetID: resource.ID, CreatedAt: now.Add(-2 * time.Hour)},
		},
		system: demoSystemManagementOverview(),
	}
}

func (s *DemoStore) EnsureConfiguredRelease(ctx context.Context, cfg Config) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	cfg = normalizeConfig(cfg)
	if cfg.LatestVersionCode <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	app := s.ensureAppLocked(cfg)
	build := s.ensureBuildLocked(app, cfg)
	s.ensureReleaseLocked(app, build, cfg)
	return nil
}

func (s *DemoStore) FindLatestRelease(ctx context.Context, req CheckRequest, fallback Config) (Config, error) {
	select {
	case <-ctx.Done():
		return fallback, ctx.Err()
	default:
	}
	fallback = normalizeConfig(fallback)
	req = normalizeCheckRequest(req, fallback)
	s.mu.Lock()
	defer s.mu.Unlock()
	var best *AppReleaseAdmin
	for i := range s.releases {
		release := &s.releases[i]
		if !releaseMatchesRequest(*release, req) {
			continue
		}
		if best == nil || release.VersionCode > best.VersionCode || (release.VersionCode == best.VersionCode && release.BuildNumber > best.BuildNumber) {
			best = release
		}
	}
	if best == nil {
		return fallback, nil
	}
	return normalizeConfig(Config{
		AppKey:                  fallback.AppKey,
		Name:                    fallback.Name,
		PackageName:             firstNonBlank(best.PackageName, fallback.PackageName),
		LatestVersionName:       best.VersionName,
		LatestVersionCode:       best.VersionCode,
		BuildNumber:             best.BuildNumber,
		Channel:                 best.Channel,
		BuildType:               best.BuildType,
		APKURL:                  firstNonBlank(best.DownloadURL, best.ArtifactPath, best.APKPath),
		DownloadURL:             firstNonBlank(best.DownloadURL, best.ArtifactPath, best.APKPath),
		SHA256:                  best.SHA256,
		SizeBytes:               best.SizeBytes,
		FileName:                best.FileName,
		ForceUpdate:             best.UpdateLevel == "forced",
		CurrentVersionAvailable: true,
		MinSupportedVersionCode: best.MinSupportedCode,
		ReleaseNotes:            firstNonBlank(best.ReleaseNotesMarkdown, best.ReleaseNotes),
		MessageZh:               firstNonBlank(best.UpgradeMessage, best.Summary),
	}), nil
}

func (s *DemoStore) UpsertInstallation(ctx context.Context, cfg Config, req CheckRequest) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	deviceID := firstNonBlank(req.DeviceID, req.DeviceKey)
	if deviceID == "" {
		return nil
	}
	cfg = normalizeConfig(cfg)
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	app := s.ensureAppLocked(cfg)
	for i := range s.installations {
		if s.installations[i].AppID == app.ID && s.installations[i].DeviceID == deviceID {
			s.installations[i].UserID = stringifyUserID(req.UserID)
			s.installations[i].InstalledVersion = req.VersionName
			s.installations[i].InstalledCode = effectiveCheckVersionCode(req)
			s.installations[i].BuildNumber = req.BuildNumber
			s.installations[i].LastSeenAt = now
			s.installations[i].UpdatedAt = now
			return nil
		}
	}
	s.installations = append([]AppInstallationAdmin{{
		ID:               "install_demo_" + uuid.NewString(),
		AppID:            app.ID,
		UserID:           stringifyUserID(req.UserID),
		DeviceID:         deviceID,
		DeviceName:       deviceID,
		Platform:         "android",
		InstalledVersion: req.VersionName,
		InstalledCode:    effectiveCheckVersionCode(req),
		BuildNumber:      req.BuildNumber,
		LastSeenAt:       now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}}, s.installations...)
	return nil
}

func (s *DemoStore) FindLatestResource(ctx context.Context, req ResourceCheckRequest, fallback Config) (ResourceCandidate, error) {
	select {
	case <-ctx.Done():
		return ResourceCandidate{}, ctx.Err()
	default:
	}
	req = normalizeResourceCheckRequest(req, fallback)
	appCode := effectiveResourceAppVersionCode(req)
	s.mu.Lock()
	defer s.mu.Unlock()
	var best *AppResourceVersionAdmin
	for i := range s.resources {
		resource := &s.resources[i]
		if resource.Channel != req.Channel || !isPublishedStatus(resource.Status) {
			continue
		}
		if appCode > 0 && resource.MinAppVersionCode > 0 && appCode < resource.MinAppVersionCode {
			continue
		}
		if appCode > 0 && resource.MaxAppVersionCode > 0 && appCode > resource.MaxAppVersionCode {
			continue
		}
		if best == nil || resource.PublishedAt.After(best.PublishedAt) || resource.CreatedAt.After(best.CreatedAt) {
			best = resource
		}
	}
	if best == nil {
		return ResourceCandidate{}, nil
	}
	candidate := ResourceCandidate{
		ResourceVersion:      best.ResourceVersion,
		Channel:              best.Channel,
		UpdateLevel:          best.UpdateLevel,
		Title:                best.Title,
		Summary:              best.Summary,
		ReleaseNotesMarkdown: best.ReleaseNotesMarkdown,
		ManifestURL:          best.ManifestURL,
		TotalSize:            best.TotalSize,
		MinAppVersionCode:    best.MinAppVersionCode,
		MaxAppVersionCode:    best.MaxAppVersionCode,
	}
	for _, pkg := range best.Packages {
		candidate.Packages = append(candidate.Packages, ResourcePackage{
			PackageKey:  pkg.PackageKey,
			PackageType: pkg.PackageType,
			URL:         pkg.FileURL,
			Size:        pkg.FileSize,
			SHA256:      pkg.SHA256,
		})
	}
	return candidate, nil
}

func (s *DemoStore) SaveUpdateEvent(ctx context.Context, cfg Config, req UpdateEventRequest) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	cfg = normalizeConfig(cfg)
	now := time.Now().UTC()
	deviceID := firstNonBlank(req.DeviceID, req.DeviceKey, "unknown")
	s.mu.Lock()
	defer s.mu.Unlock()
	app := s.ensureAppLocked(cfg)
	event := AppUpgradeEventAdmin{
		ID:              "event_demo_" + uuid.NewString(),
		AppID:           app.ID,
		UserID:          stringifyUserID(req.UserID),
		DeviceID:        deviceID,
		FromVersion:     req.FromVersion,
		ToVersion:       req.ToVersion,
		FromVersionCode: req.FromVersionCode,
		ToVersionCode:   req.ToVersionCode,
		EventType:       req.EventType,
		PackageKey:      req.PackageKey,
		ErrorMessage:    req.ErrorMessage,
		CreatedAt:       now,
	}
	if isResourceEvent(req.EventType) {
		event.Category = "resource"
		s.resourceEvents = append([]AppUpgradeEventAdmin{event}, s.resourceEvents...)
	} else {
		event.Category = "apk"
		s.apkEvents = append([]AppUpgradeEventAdmin{event}, s.apkEvents...)
	}
	s.upsertInstallationAfterEventLocked(app.ID, deviceID, req, now)
	if shouldAutoPauseResource(req) {
		s.pauseResourceAfterActivationFailureLocked(app.ID, req, now)
	}
	return nil
}

func (s *DemoStore) SaveHeartbeat(ctx context.Context, cfg Config, req HeartbeatRequest) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	cfg = normalizeConfig(cfg)
	req = normalizeHeartbeatRequest(req, cfg)
	deviceID := firstNonBlank(req.DeviceID, req.DeviceKey)
	if deviceID == "" {
		return fmt.Errorf("deviceId is required")
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	app := s.ensureAppLocked(cfg)
	versionCode := req.VersionCode
	if req.AppVersionCode > 0 {
		versionCode = req.AppVersionCode
	}
	for i := range s.installations {
		if s.installations[i].AppID == app.ID && s.installations[i].DeviceID == deviceID {
			s.installations[i].UserID = stringifyUserID(req.UserID)
			s.installations[i].DeviceName = firstNonBlank(req.DeviceName, s.installations[i].DeviceName)
			s.installations[i].Platform = firstNonBlank(req.Platform, s.installations[i].Platform, "android")
			s.installations[i].OSVersion = firstNonBlank(req.OSVersion, s.installations[i].OSVersion)
			s.installations[i].DeviceModel = firstNonBlank(req.DeviceModel, s.installations[i].DeviceModel)
			s.installations[i].InstalledVersion = firstNonBlank(req.VersionName, s.installations[i].InstalledVersion)
			if versionCode > 0 {
				s.installations[i].InstalledCode = versionCode
			}
			if req.BuildNumber > 0 {
				s.installations[i].BuildNumber = req.BuildNumber
			}
			s.installations[i].ResourceVersion = firstNonBlank(req.ResourceVersion, s.installations[i].ResourceVersion)
			s.installations[i].LastSeenAt = now
			s.installations[i].UpdatedAt = now
			return nil
		}
	}
	s.installations = append([]AppInstallationAdmin{{
		ID:               "install_demo_" + uuid.NewString(),
		AppID:            app.ID,
		UserID:           stringifyUserID(req.UserID),
		DeviceID:         deviceID,
		DeviceName:       firstNonBlank(req.DeviceName, deviceID),
		Platform:         firstNonBlank(req.Platform, "android"),
		OSVersion:        req.OSVersion,
		DeviceModel:      req.DeviceModel,
		InstalledVersion: req.VersionName,
		InstalledCode:    versionCode,
		BuildNumber:      req.BuildNumber,
		ResourceVersion:  req.ResourceVersion,
		LastSeenAt:       now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}}, s.installations...)
	return nil
}

func (s *DemoStore) Overview(ctx context.Context) (ReleaseOverview, error) {
	select {
	case <-ctx.Done():
		return ReleaseOverview{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var overview ReleaseOverview
	for _, app := range s.apps {
		overview.Apps = append(overview.Apps, AppSummary{
			AppID:       app.ID,
			AppKey:      app.AppKey,
			Name:        app.Name,
			Platform:    app.Platform,
			PackageName: app.PackageName,
			Enabled:     app.Enabled,
			UpdatedAt:   app.UpdatedAt,
		})
	}
	for _, release := range s.releases {
		overview.Releases = append(overview.Releases, ReleaseSummary{
			ReleaseID:               release.ID,
			AppKey:                  appKeyForID(s.apps, release.AppID),
			VersionName:             release.VersionName,
			VersionCode:             release.VersionCode,
			BuildNumber:             release.BuildNumber,
			Channel:                 release.Channel,
			BuildType:               release.BuildType,
			Status:                  release.Status,
			UpdateLevel:             release.UpdateLevel,
			MinSupportedVersionCode: release.MinSupportedCode,
			DownloadURL:             firstNonBlank(release.DownloadURL, release.ArtifactPath, release.APKPath),
			SHA256:                  release.SHA256,
			SizeBytes:               release.SizeBytes,
			PublishedAt:             firstTime(release.PublishedAt, release.CreatedAt),
		})
	}
	for _, resource := range s.resources {
		overview.ResourceVersions = append(overview.ResourceVersions, ResourceSummary{
			ResourceVersion: resource.ResourceVersion,
			Channel:         resource.Channel,
			Status:          resource.Status,
			UpdateLevel:     resource.UpdateLevel,
			ManifestURL:     resource.ManifestURL,
			TotalSize:       resource.TotalSize,
			PublishedAt:     firstTime(resource.PublishedAt, resource.CreatedAt),
		})
	}
	for _, installation := range s.installations {
		overview.Installations = append(overview.Installations, InstallationSummary{
			DeviceID:          installation.DeviceID,
			DeviceKey:         installation.DeviceID,
			DeviceName:        installation.DeviceName,
			InstalledVersion:  installation.InstalledVersion,
			InstalledCode:     installation.InstalledCode,
			BuildNumber:       installation.BuildNumber,
			ResourceVersion:   installation.ResourceVersion,
			LastSeenAt:        installation.LastSeenAt,
			LastUpgradeStatus: installation.LastUpgradeStatus,
			LastError:         installation.LastError,
		})
	}
	return overview, nil
}

func (s *DemoStore) AdminOverview(ctx context.Context) (AdminOverview, error) {
	select {
	case <-ctx.Done():
		return AdminOverview{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	overview := AdminOverview{
		Apps:                 append([]AppAdminSummary(nil), s.apps...),
		Releases:             cloneReleases(s.releases),
		BuildJobs:            append([]AppBuildJob(nil), s.builds...),
		WebhookEvents:        append([]WebhookEventAdmin(nil), s.webhookEvents...),
		ResourceVersions:     cloneResources(s.resources),
		Installations:        append([]AppInstallationAdmin(nil), s.installations...),
		UpgradeEvents:        append([]AppUpgradeEventAdmin(nil), s.apkEvents...),
		ResourceUpdateEvents: append([]AppUpgradeEventAdmin(nil), s.resourceEvents...),
		AuditLogs:            append([]AppReleaseAuditLogAdmin(nil), s.auditLogs...),
	}
	sortAdminOverview(&overview)
	for i := range overview.Releases {
		if overview.Releases[i].IsLatest || isPublishedStatus(overview.Releases[i].Status) {
			overview.Latest = &overview.Releases[i]
			break
		}
	}
	for i := range overview.ResourceVersions {
		if isPublishedStatus(overview.ResourceVersions[i].Status) {
			overview.LatestResource = &overview.ResourceVersions[i]
			break
		}
	}
	overview.QualityMetrics = buildReleaseQualityMetrics(overview.UpgradeEvents, overview.ResourceUpdateEvents, normalizeQualityPolicy(QualityPolicy{}))
	return overview, nil
}

func (s *DemoStore) CreateApp(ctx context.Context, req CreateAppRequest) (AppAdminSummary, error) {
	select {
	case <-ctx.Done():
		return AppAdminSummary{}, ctx.Err()
	default:
	}
	now := time.Now().UTC()
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.apps {
		if s.apps[i].AppKey == req.AppKey && s.apps[i].Platform == req.Platform {
			s.apps[i].Name = req.Name
			s.apps[i].PackageName = req.PackageName
			s.apps[i].Description = req.Description
			s.apps[i].Enabled = enabled
			s.apps[i].UpdatedAt = now
			return s.apps[i], nil
		}
	}
	app := AppAdminSummary{
		ID:          "app_demo_" + uuid.NewString(),
		AppKey:      req.AppKey,
		Name:        req.Name,
		Platform:    req.Platform,
		PackageName: req.PackageName,
		Description: req.Description,
		Enabled:     enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.apps = append([]AppAdminSummary{app}, s.apps...)
	return app, nil
}

func (s *DemoStore) UpdateAppEnabled(ctx context.Context, id string, enabled bool) (AppAdminSummary, error) {
	select {
	case <-ctx.Done():
		return AppAdminSummary{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.apps {
		if s.apps[i].ID == id {
			s.apps[i].Enabled = enabled
			s.apps[i].UpdatedAt = time.Now().UTC()
			return s.apps[i], nil
		}
	}
	return AppAdminSummary{}, fmt.Errorf("app %s not found", id)
}

func (s *DemoStore) CreateBuild(ctx context.Context, build AppBuildJob, cfg Config) (AppBuildJob, error) {
	select {
	case <-ctx.Done():
		return AppBuildJob{}, ctx.Err()
	default:
	}
	cfg = normalizeConfig(cfg)
	s.mu.Lock()
	defer s.mu.Unlock()
	app := s.ensureAppLocked(cfg)
	build.AppID = app.ID
	if build.ID == "" {
		build.ID = "build_demo_" + uuid.NewString()
	}
	if build.CreatedAt.IsZero() {
		build.CreatedAt = time.Now().UTC()
	}
	if build.StartedAt.IsZero() {
		build.StartedAt = build.CreatedAt
	}
	if build.FinishedAt.IsZero() {
		build.FinishedAt = build.CreatedAt
	}
	if build.ArtifactType == "" {
		build.ArtifactType = "apk"
	}
	for i := range s.builds {
		if s.builds[i].AppID == build.AppID && s.builds[i].BuildNumber == build.BuildNumber {
			build.ID = s.builds[i].ID
			build.CreatedAt = s.builds[i].CreatedAt
			s.builds[i] = build
			return s.builds[i], nil
		}
	}
	s.builds = append([]AppBuildJob{build}, s.builds...)
	return build, nil
}

func (s *DemoStore) SaveWebhookEvent(ctx context.Context, event WebhookEventRequest) (WebhookEventAdmin, error) {
	select {
	case <-ctx.Done():
		return WebhookEventAdmin{}, ctx.Err()
	default:
	}
	now := firstTime(event.ReceivedAt, time.Now().UTC())
	saved := WebhookEventAdmin{
		ID:         "webhook_demo_" + uuid.NewString(),
		Provider:   event.Provider,
		EventType:  event.EventType,
		DeliveryID: event.DeliveryID,
		Repository: event.Repository,
		Ref:        event.Ref,
		CommitSHA:  event.CommitSHA,
		Sender:     event.Sender,
		Action:     event.Action,
		Workflow:   event.Workflow,
		RunID:      event.RunID,
		CreatedAt:  now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webhookEvents = append([]WebhookEventAdmin{saved}, s.webhookEvents...)
	return saved, nil
}

func (s *DemoStore) CreateRelease(ctx context.Context, req CreateReleaseRequest) (AppReleaseAdmin, error) {
	select {
	case <-ctx.Done():
		return AppReleaseAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	build, ok := s.findBuildLocked(req.BuildID)
	if !ok {
		return AppReleaseAdmin{}, fmt.Errorf("build %s not found", req.BuildID)
	}
	now := time.Now().UTC()
	status := "draft"
	var scheduled time.Time
	if req.ScheduledAt != "" {
		status = "scheduled"
		scheduled, _ = time.Parse(time.RFC3339, req.ScheduledAt)
	}
	release := releaseFromBuild(build, req, now)
	release.ID = "release_demo_" + uuid.NewString()
	release.Status = status
	release.ScheduledAt = scheduled
	for i := range s.releases {
		if s.releases[i].BuildID == req.BuildID && s.releases[i].Channel == req.Channel && s.releases[i].BuildType == build.BuildType {
			release.ID = s.releases[i].ID
			release.CreatedAt = s.releases[i].CreatedAt
			s.releases[i] = release
			return s.releases[i], nil
		}
	}
	s.releases = append([]AppReleaseAdmin{release}, s.releases...)
	return release, nil
}

func (s *DemoStore) UpdateReleaseStatus(ctx context.Context, id, status string, publish bool) (AppReleaseAdmin, error) {
	select {
	case <-ctx.Done():
		return AppReleaseAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for i := range s.releases {
		if s.releases[i].ID == id {
			s.releases[i].Status = status
			s.releases[i].IsPublished = publish || isPublishedStatus(status)
			s.releases[i].IsLatest = s.releases[i].IsPublished
			if publish && s.releases[i].PublishedAt.IsZero() {
				s.releases[i].PublishedAt = now
			}
			if status == "paused" {
				s.releases[i].PausedAt = now
			} else if status != "paused" {
				s.releases[i].PausedAt = time.Time{}
			}
			s.releases[i].UpdatedAt = now
			return s.releases[i], nil
		}
	}
	return AppReleaseAdmin{}, fmt.Errorf("release %s not found", id)
}

func (s *DemoStore) UpdateReleaseRollout(ctx context.Context, id string, rollout int) (AppReleaseAdmin, error) {
	select {
	case <-ctx.Done():
		return AppReleaseAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.releases {
		if s.releases[i].ID == id {
			s.releases[i].RolloutPercentage = rollout
			if s.releases[i].Status == "released" && rollout < 100 {
				s.releases[i].Status = "rolling_out"
			}
			s.releases[i].UpdatedAt = time.Now().UTC()
			return s.releases[i], nil
		}
	}
	return AppReleaseAdmin{}, fmt.Errorf("release %s not found", id)
}

func (s *DemoStore) UpdateReleaseNotes(ctx context.Context, id string, req UpdateNotesRequest) (AppReleaseAdmin, error) {
	select {
	case <-ctx.Done():
		return AppReleaseAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.releases {
		if s.releases[i].ID == id {
			s.releases[i].Title = req.Title
			s.releases[i].Summary = req.Summary
			s.releases[i].ReleaseNotesMarkdown = req.ReleaseNotesMarkdown
			s.releases[i].UpdatedAt = time.Now().UTC()
			return s.releases[i], nil
		}
	}
	return AppReleaseAdmin{}, fmt.Errorf("release %s not found", id)
}

func (s *DemoStore) CreateResourceVersion(ctx context.Context, req CreateResourceVersionRequest, packages []AppResourcePackageAdmin, manifestKey string, cfg Config) (AppResourceVersionAdmin, error) {
	select {
	case <-ctx.Done():
		return AppResourceVersionAdmin{}, ctx.Err()
	default:
	}
	cfg = normalizeConfig(cfg)
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	app := s.ensureAppLocked(cfg)
	totalSize := int64(0)
	for i := range packages {
		if packages[i].ID == "" {
			packages[i].ID = "resource_pkg_demo_" + uuid.NewString()
		}
		packages[i].FileURL = "/api/v1/app/resources/packages/" + packages[i].ID + "/download"
		if packages[i].CreatedAt.IsZero() {
			packages[i].CreatedAt = now
		}
		totalSize += packages[i].FileSize
	}
	resource := AppResourceVersionAdmin{
		ID:                   "resource_demo_" + uuid.NewString(),
		AppID:                app.ID,
		ResourceVersion:      req.ResourceVersion,
		Channel:              req.Channel,
		Status:               "draft",
		MinAppVersionCode:    req.MinAppVersionCode,
		MaxAppVersionCode:    req.MaxAppVersionCode,
		UpdateLevel:          req.UpdateLevel,
		Title:                req.Title,
		Summary:              req.Summary,
		ReleaseNotesMarkdown: req.ReleaseNotesMarkdown,
		RolloutPercentage:    req.RolloutPercentage,
		ManifestURL:          "",
		ManifestSigned:       strings.TrimSpace(cfg.ManifestPrivateKey) != "",
		TotalSize:            totalSize,
		Packages:             packages,
		CreatedBy:            "admin",
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	resource.ManifestURL = "/api/v1/app/resources/" + resource.ID + "/manifest"
	if resource.ManifestSigned {
		resource.SignatureAlgorithm = ManifestSignatureAlgorithmEd25519
	}
	for i := range s.resources {
		if s.resources[i].AppID == app.ID && s.resources[i].ResourceVersion == resource.ResourceVersion && s.resources[i].Channel == resource.Channel {
			resource.ID = s.resources[i].ID
			resource.CreatedAt = s.resources[i].CreatedAt
			resource.ManifestURL = "/api/v1/app/resources/" + resource.ID + "/manifest"
			s.resources[i] = resource
			_ = manifestKey
			return s.resources[i], nil
		}
	}
	_ = manifestKey
	s.resources = append([]AppResourceVersionAdmin{resource}, s.resources...)
	return resource, nil
}

func (s *DemoStore) UpdateResourceStatus(ctx context.Context, id, status string, publish bool) (AppResourceVersionAdmin, error) {
	select {
	case <-ctx.Done():
		return AppResourceVersionAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for i := range s.resources {
		if s.resources[i].ID == id {
			s.resources[i].Status = status
			if publish && s.resources[i].PublishedAt.IsZero() {
				s.resources[i].PublishedAt = now
			}
			if status == "paused" {
				s.resources[i].PausedAt = now
			} else if status != "paused" {
				s.resources[i].PausedAt = time.Time{}
			}
			s.resources[i].UpdatedAt = now
			return s.resources[i], nil
		}
	}
	return AppResourceVersionAdmin{}, fmt.Errorf("resource %s not found", id)
}

func (s *DemoStore) UpdateResourceRollout(ctx context.Context, id string, rollout int) (AppResourceVersionAdmin, error) {
	select {
	case <-ctx.Done():
		return AppResourceVersionAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.resources {
		if s.resources[i].ID == id {
			s.resources[i].RolloutPercentage = rollout
			if s.resources[i].Status == "released" && rollout < 100 {
				s.resources[i].Status = "rolling_out"
			}
			s.resources[i].UpdatedAt = time.Now().UTC()
			return s.resources[i], nil
		}
	}
	return AppResourceVersionAdmin{}, fmt.Errorf("resource %s not found", id)
}

func (s *DemoStore) UpdateResourceNotes(ctx context.Context, id string, req UpdateNotesRequest) (AppResourceVersionAdmin, error) {
	select {
	case <-ctx.Done():
		return AppResourceVersionAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.resources {
		if s.resources[i].ID == id {
			s.resources[i].Title = req.Title
			s.resources[i].Summary = req.Summary
			s.resources[i].ReleaseNotesMarkdown = req.ReleaseNotesMarkdown
			s.resources[i].UpdatedAt = time.Now().UTC()
			return s.resources[i], nil
		}
	}
	return AppResourceVersionAdmin{}, fmt.Errorf("resource %s not found", id)
}

func (s *DemoStore) GetBuildStorageKey(ctx context.Context, buildID string) (string, string, error) {
	return "", "", fmt.Errorf("demo build %s has no local storage key", buildID)
}

func (s *DemoStore) GetReleaseStorageKey(ctx context.Context, releaseID string) (string, string, error) {
	return "", "", fmt.Errorf("demo release %s has no local storage key", releaseID)
}

func (s *DemoStore) GetResourcePackageStorageKey(ctx context.Context, packageID string) (string, string, error) {
	return "", "", fmt.Errorf("demo resource package %s has no local storage key", packageID)
}

func (s *DemoStore) GetResourceManifestStorageKey(ctx context.Context, resourceID string) (string, string, error) {
	return "", "", fmt.Errorf("demo resource %s has no local storage key", resourceID)
}

func (s *DemoStore) InsertAudit(ctx context.Context, action, targetType, targetID, message string, metadata map[string]any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auditLogs = append([]AppReleaseAuditLogAdmin{{
		ID:         "audit_demo_" + uuid.NewString(),
		Operator:   "admin",
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		AfterJSON:  metadata,
		CreatedAt:  time.Now().UTC(),
	}}, s.auditLogs...)
	_ = message
	return nil
}

func (s *DemoStore) AuditExists(ctx context.Context, action, targetID string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, audit := range s.auditLogs {
		if audit.Action == action && audit.TargetID == targetID {
			return true, nil
		}
	}
	return false, nil
}

func (s *DemoStore) SystemManagementOverview(ctx context.Context) (SystemManagementOverview, error) {
	select {
	case <-ctx.Done():
		return SystemManagementOverview{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneSystemOverview(s.system), nil
}

func (s *DemoStore) AdminRolePermissions(ctx context.Context, account string) ([]string, bool, error) {
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	permissions, active := accountPermissionsFromOverview(s.system, account)
	return permissions, active, nil
}

func (s *DemoStore) CreateSystemUser(ctx context.Context, req CreateSystemUserRequest) (SystemUserAdmin, error) {
	select {
	case <-ctx.Done():
		return SystemUserAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user := demoSystemUser(req)
	s.system.Users = append([]SystemUserAdmin{user}, s.system.Users...)
	return user, nil
}

func (s *DemoStore) UpdateSystemUserStatus(ctx context.Context, id, status string) (SystemUserAdmin, error) {
	select {
	case <-ctx.Done():
		return SystemUserAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.system.Users {
		if s.system.Users[i].ID == id {
			s.system.Users[i].Status = status
			return s.system.Users[i], nil
		}
	}
	return SystemUserAdmin{}, fmt.Errorf("system user %s not found", id)
}

func (s *DemoStore) CreateSystemRole(ctx context.Context, req CreateSystemRoleRequest) (SystemRoleAdmin, error) {
	select {
	case <-ctx.Done():
		return SystemRoleAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	role := demoSystemRole(req)
	s.system.Roles = append([]SystemRoleAdmin{role}, s.system.Roles...)
	return role, nil
}

func (s *DemoStore) UpdateSystemRoleEnabled(ctx context.Context, id string, enabled bool) (SystemRoleAdmin, error) {
	select {
	case <-ctx.Done():
		return SystemRoleAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.system.Roles {
		if s.system.Roles[i].ID == id {
			s.system.Roles[i].Enabled = enabled
			return s.system.Roles[i], nil
		}
	}
	return SystemRoleAdmin{}, fmt.Errorf("system role %s not found", id)
}

func (s *DemoStore) CreateSystemPermission(ctx context.Context, req CreateSystemPermissionRequest) (SystemPermissionAdmin, error) {
	select {
	case <-ctx.Done():
		return SystemPermissionAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	permission := demoSystemPermission(req)
	s.system.Permissions = append([]SystemPermissionAdmin{permission}, s.system.Permissions...)
	return permission, nil
}

func (s *DemoStore) UpdateSystemPermissionEnabled(ctx context.Context, id string, enabled bool) (SystemPermissionAdmin, error) {
	select {
	case <-ctx.Done():
		return SystemPermissionAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.system.Permissions {
		if s.system.Permissions[i].ID == id {
			s.system.Permissions[i].Enabled = enabled
			return s.system.Permissions[i], nil
		}
	}
	return SystemPermissionAdmin{}, fmt.Errorf("system permission %s not found", id)
}

func (s *DemoStore) CreateSystemDictionary(ctx context.Context, req CreateSystemDictionaryRequest) (SystemDictionaryAdmin, error) {
	select {
	case <-ctx.Done():
		return SystemDictionaryAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dictionary := demoSystemDictionary(req)
	s.system.Dictionaries = append([]SystemDictionaryAdmin{dictionary}, s.system.Dictionaries...)
	return dictionary, nil
}

func (s *DemoStore) UpdateSystemDictionaryEnabled(ctx context.Context, id string, enabled bool) (SystemDictionaryAdmin, error) {
	select {
	case <-ctx.Done():
		return SystemDictionaryAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.system.Dictionaries {
		if s.system.Dictionaries[i].ID == id {
			s.system.Dictionaries[i].Enabled = enabled
			return s.system.Dictionaries[i], nil
		}
	}
	return SystemDictionaryAdmin{}, fmt.Errorf("system dictionary %s not found", id)
}

func (s *DemoStore) CreateSystemMenu(ctx context.Context, req CreateSystemMenuRequest) (SystemMenuAdmin, error) {
	select {
	case <-ctx.Done():
		return SystemMenuAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	menu := demoSystemMenu(req)
	s.system.Menus = append([]SystemMenuAdmin{menu}, s.system.Menus...)
	return menu, nil
}

func (s *DemoStore) UpdateSystemMenuVisible(ctx context.Context, id string, visible bool) (SystemMenuAdmin, error) {
	select {
	case <-ctx.Done():
		return SystemMenuAdmin{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.system.Menus {
		if s.system.Menus[i].ID == id {
			s.system.Menus[i].Visible = visible
			return s.system.Menus[i], nil
		}
	}
	return SystemMenuAdmin{}, fmt.Errorf("system menu %s not found", id)
}

func (s *DemoStore) ensureAppLocked(cfg Config) AppAdminSummary {
	for _, app := range s.apps {
		if app.AppKey == cfg.AppKey && app.Platform == "android" {
			return app
		}
	}
	now := time.Now().UTC()
	app := AppAdminSummary{
		ID:          "app_demo_" + uuid.NewString(),
		AppKey:      cfg.AppKey,
		Name:        cfg.Name,
		Platform:    "android",
		PackageName: cfg.PackageName,
		Description: "Configured demo app",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.apps = append([]AppAdminSummary{app}, s.apps...)
	return app
}

func (s *DemoStore) ensureBuildLocked(app AppAdminSummary, cfg Config) AppBuildJob {
	for _, build := range s.builds {
		if build.AppID == app.ID && build.BuildNumber == effectiveBuildNumber(cfg) {
			return build
		}
	}
	now := time.Now().UTC()
	build := AppBuildJob{
		ID:           "build_demo_" + uuid.NewString(),
		Status:       "success",
		AppID:        app.ID,
		GitRef:       "configured",
		BuildType:    cfg.BuildType,
		Channel:      cfg.Channel,
		VersionName:  cfg.LatestVersionName,
		VersionCode:  cfg.LatestVersionCode,
		BuildNumber:  effectiveBuildNumber(cfg),
		ArtifactType: "apk",
		ArtifactPath: firstNonBlank(cfg.DownloadURL, cfg.APKURL),
		ArtifactSize: cfg.SizeBytes,
		SHA256:       cfg.SHA256,
		FileName:     cfg.FileName,
		CreatedAt:    now,
		StartedAt:    now,
		FinishedAt:   now,
	}
	s.builds = append([]AppBuildJob{build}, s.builds...)
	return build
}

func (s *DemoStore) ensureReleaseLocked(app AppAdminSummary, build AppBuildJob, cfg Config) AppReleaseAdmin {
	for _, release := range s.releases {
		if release.AppID == app.ID && release.BuildID == build.ID && release.Channel == cfg.Channel && release.BuildType == cfg.BuildType {
			return release
		}
	}
	now := time.Now().UTC()
	release := AppReleaseAdmin{
		ID:                   "release_demo_" + uuid.NewString(),
		AppID:                app.ID,
		BuildID:              build.ID,
		PackageName:          app.PackageName,
		VersionName:          build.VersionName,
		VersionCode:          build.VersionCode,
		BuildNumber:          build.BuildNumber,
		Channel:              cfg.Channel,
		BuildType:            cfg.BuildType,
		GitRef:               build.GitRef,
		GitCommit:            build.GitCommit,
		ArtifactPath:         build.ArtifactPath,
		FileName:             build.FileName,
		SizeBytes:            build.ArtifactSize,
		SHA256:               build.SHA256,
		Status:               "released",
		Title:                "配置版本更新",
		Summary:              firstNonBlank(cfg.MessageZh, cfg.ReleaseNotes),
		ReleaseNotesMarkdown: cfg.ReleaseNotes,
		UpdateLevel:          configuredUpdateLevel(cfg),
		RolloutPercentage:    100,
		MinSupportedCode:     cfg.MinSupportedVersionCode,
		BlockOldVersions:     cfg.MinSupportedVersionCode > 0,
		IsLatest:             true,
		IsPublished:          true,
		ForceUpdate:          cfg.ForceUpdate,
		PublishedAt:          now,
		CreatedAt:            now,
		UpdatedAt:            now,
		DownloadURL:          firstNonBlank(cfg.DownloadURL, cfg.APKURL),
	}
	s.releases = append([]AppReleaseAdmin{release}, s.releases...)
	return release
}

func (s *DemoStore) findBuildLocked(id string) (AppBuildJob, bool) {
	for _, build := range s.builds {
		if build.ID == id {
			return build, true
		}
	}
	return AppBuildJob{}, false
}

func (s *DemoStore) upsertInstallationAfterEventLocked(appID, deviceID string, req UpdateEventRequest, now time.Time) {
	status := installationStatusFromEvent(req.EventType)
	if status == "" {
		return
	}
	for i := range s.installations {
		if s.installations[i].AppID == appID && s.installations[i].DeviceID == deviceID {
			s.installations[i].UserID = stringifyUserID(req.UserID)
			s.installations[i].LastSeenAt = now
			s.installations[i].LastUpgradeAt = now
			s.installations[i].LastUpgradeStatus = status
			if strings.Contains(req.EventType, "failed") {
				s.installations[i].LastError = req.ErrorMessage
			}
			if req.EventType == "install_success" {
				s.installations[i].InstalledVersion = req.ToVersion
				s.installations[i].InstalledCode = req.ToVersionCode
			}
			if req.EventType == "activation_success" || req.EventType == "resource_activation_success" {
				s.installations[i].ResourceVersion = req.ToVersion
			}
			s.installations[i].UpdatedAt = now
			return
		}
	}
	s.installations = append([]AppInstallationAdmin{{
		ID:                "install_demo_" + uuid.NewString(),
		AppID:             appID,
		UserID:            stringifyUserID(req.UserID),
		DeviceID:          deviceID,
		Platform:          "android",
		LastSeenAt:        now,
		LastUpgradeAt:     now,
		LastUpgradeStatus: status,
		LastError:         req.ErrorMessage,
		CreatedAt:         now,
		UpdatedAt:         now,
	}}, s.installations...)
}

func (s *DemoStore) pauseResourceAfterActivationFailureLocked(appID string, req UpdateEventRequest, now time.Time) {
	for i := range s.resources {
		if s.resources[i].AppID == appID && s.resources[i].ResourceVersion == strings.TrimSpace(req.ToVersion) && isPublishedStatus(s.resources[i].Status) {
			s.resources[i].Status = "paused"
			s.resources[i].PausedAt = now
			s.resources[i].UpdatedAt = now
			s.auditLogs = append([]AppReleaseAuditLogAdmin{{
				ID:         "audit_demo_" + uuid.NewString(),
				AppID:      appID,
				Operator:   "demo",
				Action:     "resource.auto_pause",
				TargetType: "app_resource_version",
				TargetID:   s.resources[i].ID,
				CreatedAt:  now,
			}}, s.auditLogs...)
			return
		}
	}
}

func releaseFromBuild(build AppBuildJob, req CreateReleaseRequest, now time.Time) AppReleaseAdmin {
	return AppReleaseAdmin{
		AppID:                build.AppID,
		BuildID:              build.ID,
		PackageName:          "com.kingdomhelper.executor",
		VersionName:          build.VersionName,
		VersionCode:          build.VersionCode,
		BuildNumber:          build.BuildNumber,
		Channel:              req.Channel,
		BuildType:            build.BuildType,
		GitRef:               build.GitRef,
		GitCommit:            build.GitCommit,
		APIBaseURL:           build.APIBaseURL,
		ArtifactPath:         build.ArtifactPath,
		FileName:             build.FileName,
		SizeBytes:            build.ArtifactSize,
		SHA256:               build.SHA256,
		Title:                req.Title,
		Summary:              req.Summary,
		ReleaseNotesMarkdown: req.ReleaseNotesMarkdown,
		UpdateLevel:          req.UpdateLevel,
		RolloutPercentage:    req.RolloutPercentage,
		TargetType:           firstNonBlank(req.TargetType, "all"),
		TargetValue:          firstNonBlank(req.TargetValue, "all"),
		MinSupportedCode:     req.MinSupportedCode,
		BlockOldVersions:     req.BlockOldVersions,
		CreatedBy:            "admin",
		CreatedAt:            now,
		UpdatedAt:            now,
		DownloadURL:          build.ArtifactPath,
	}
}

func releaseMatchesRequest(release AppReleaseAdmin, req CheckRequest) bool {
	if !isPublishedStatus(release.Status) {
		return false
	}
	if release.PackageName != "" && req.PackageName != "" && release.PackageName != req.PackageName {
		return false
	}
	if release.Channel != req.Channel || release.BuildType != req.BuildType {
		return false
	}
	if release.RolloutPercentage <= 0 {
		return false
	}
	return true
}

func isPublishedStatus(status string) bool {
	switch status {
	case "released", "rolling_out", "testing":
		return true
	default:
		return false
	}
}

func isResourceEvent(eventType string) bool {
	return strings.HasPrefix(eventType, "resource_") || strings.Contains(eventType, "manifest") || strings.Contains(eventType, "activation") || strings.Contains(eventType, "extract") || strings.Contains(eventType, "package_")
}

func sortAdminOverview(overview *AdminOverview) {
	sort.Slice(overview.BuildJobs, func(i, j int) bool {
		return overview.BuildJobs[i].CreatedAt.After(overview.BuildJobs[j].CreatedAt)
	})
	sort.Slice(overview.Releases, func(i, j int) bool {
		return overview.Releases[i].UpdatedAt.After(overview.Releases[j].UpdatedAt)
	})
	sort.Slice(overview.ResourceVersions, func(i, j int) bool {
		return overview.ResourceVersions[i].UpdatedAt.After(overview.ResourceVersions[j].UpdatedAt)
	})
	sort.Slice(overview.Installations, func(i, j int) bool {
		return overview.Installations[i].LastSeenAt.After(overview.Installations[j].LastSeenAt)
	})
	sort.Slice(overview.UpgradeEvents, func(i, j int) bool {
		return overview.UpgradeEvents[i].CreatedAt.After(overview.UpgradeEvents[j].CreatedAt)
	})
	sort.Slice(overview.ResourceUpdateEvents, func(i, j int) bool {
		return overview.ResourceUpdateEvents[i].CreatedAt.After(overview.ResourceUpdateEvents[j].CreatedAt)
	})
	sort.Slice(overview.AuditLogs, func(i, j int) bool {
		return overview.AuditLogs[i].CreatedAt.After(overview.AuditLogs[j].CreatedAt)
	})
}

func cloneReleases(releases []AppReleaseAdmin) []AppReleaseAdmin {
	out := append([]AppReleaseAdmin(nil), releases...)
	return out
}

func cloneResources(resources []AppResourceVersionAdmin) []AppResourceVersionAdmin {
	out := append([]AppResourceVersionAdmin(nil), resources...)
	for i := range out {
		out[i].Packages = append([]AppResourcePackageAdmin(nil), out[i].Packages...)
	}
	return out
}

func appKeyForID(apps []AppAdminSummary, appID string) string {
	for _, app := range apps {
		if app.ID == appID {
			return app.AppKey
		}
	}
	return ""
}

func firstTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}
