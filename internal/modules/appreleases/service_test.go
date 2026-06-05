package appreleases

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

var errCaptureStoreNotFound = errors.New("not found")

func TestCheckReturnsOptionalUpdate(t *testing.T) {
	service := NewService(Config{
		LatestVersionName:       "0.1.1",
		LatestVersionCode:       2,
		CurrentVersionAvailable: true,
		APKURL:                  "http://example.com/app.apk",
	})

	resp := service.Check(context.Background(), CheckRequest{VersionCode: 1, Channel: "dev", BuildType: "debug"})

	if !resp.HasUpdate || resp.ForceUpdate || !resp.CurrentVersionAvailable {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.VersionCode != 2 || resp.APKURL == "" {
		t.Fatalf("missing latest metadata: %+v", resp)
	}
}

func TestCheckUsesConfiguredFileName(t *testing.T) {
	service := NewService(Config{
		LatestVersionName:       "0.1.1",
		LatestVersionCode:       2,
		CurrentVersionAvailable: true,
		DownloadURL:             "/api/v1/app/builds/build-1/download",
		FileName:                "game-helper-0.1.1.apk",
	})

	resp := service.Check(context.Background(), CheckRequest{VersionCode: 1})

	if resp.FileName != "game-helper-0.1.1.apk" {
		t.Fatalf("expected configured file name, got %+v", resp)
	}
}

func TestCheckForceUpdatesBelowMinSupportedVersion(t *testing.T) {
	service := NewService(Config{
		LatestVersionCode:       4,
		MinSupportedVersionCode: 3,
		CurrentVersionAvailable: true,
	})

	resp := service.Check(context.Background(), CheckRequest{VersionCode: 2})

	if !resp.ForceUpdate || resp.CurrentVersionAvailable {
		t.Fatalf("expected blocked force update: %+v", resp)
	}
}

func TestCheckDoesNotBlockCurrentVersionForForcedRelease(t *testing.T) {
	service := NewService(Config{
		LatestVersionName:       "0.1.0-runtime-control-19",
		LatestVersionCode:       19,
		BuildNumber:             19,
		ForceUpdate:             true,
		MinSupportedVersionCode: 19,
		CurrentVersionAvailable: true,
		MessageZh:               "请更新到 build 19 后再继续真机托管验证",
	})

	resp := service.Check(context.Background(), CheckRequest{VersionCode: 19, BuildNumber: 19})

	if resp.HasUpdate || resp.UpdateAvailable || resp.ForceUpdate || resp.BlockTaskExecution || !resp.CurrentVersionAvailable {
		t.Fatalf("expected current forced release to keep running: %+v", resp)
	}
	if resp.UpdateLevel != "normal" || resp.MessageZh != "当前版本可以继续使用" {
		t.Fatalf("unexpected current release message: %+v", resp)
	}
}

func TestCheckAcceptsSnakeCaseVersionFields(t *testing.T) {
	service := NewService(Config{
		LatestVersionCode:       18,
		MinSupportedVersionCode: 18,
		CurrentVersionAvailable: true,
		APKURL:                  "http://example.com/app-18.apk",
	})
	var req CheckRequest
	if err := json.Unmarshal([]byte(`{
		"app_key": "game-helper-android",
		"device_key": "dev_1",
		"package_name": "com.kingdomhelper.executor",
		"version_code": 17,
		"app_version_code": 17,
		"version_name": "0.1.0-17",
		"build_number": 17,
		"channel": "dev",
		"build_type": "debug"
	}`), &req); err != nil {
		t.Fatal(err)
	}

	resp := service.Check(context.Background(), req)

	if !resp.HasUpdate || !resp.UpdateAvailable || !resp.ForceUpdate || !resp.BlockTaskExecution || resp.CurrentVersionAvailable {
		t.Fatalf("expected snake_case build 17 to be forced to update: %+v", resp)
	}
}

func TestHeartbeatAcceptsSnakeCaseFields(t *testing.T) {
	store := &captureBuildStore{}
	service := NewServiceWithStore(Config{PackageName: "com.kingdomhelper.executor"}, store)
	var req HeartbeatRequest
	if err := json.Unmarshal([]byte(`{
		"device_id": "device-1",
		"package_name": "com.kingdomhelper.executor",
		"version_name": "0.2.0",
		"version_code": 200,
		"build_number": 37,
		"resource_version": "20260602.1",
		"os_version": "14",
		"device_model": "Pixel 8"
	}`), &req); err != nil {
		t.Fatal(err)
	}

	resp, err := service.SaveHeartbeat(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("unexpected heartbeat response: %+v", resp)
	}
}

func TestHeartbeatRequiresDeviceID(t *testing.T) {
	service := NewService(Config{})

	if _, err := service.SaveHeartbeat(context.Background(), HeartbeatRequest{}); err == nil {
		t.Fatal("expected heartbeat without device id to fail")
	}
}

func TestTaskPreflightAllowsCurrentVersion(t *testing.T) {
	service := NewService(Config{
		LatestVersionCode:       20,
		MinSupportedVersionCode: 20,
		CurrentVersionAvailable: true,
	})

	resp := service.TaskPreflight(context.Background(), TaskPreflightRequest{VersionCode: 20})

	if !resp.CanExecute || resp.BlockReason != "" || resp.AppUpdateRequired || resp.ResourceRequired {
		t.Fatalf("expected task to be executable: %+v", resp)
	}
}

func TestTaskPreflightBlocksUnsupportedAppVersion(t *testing.T) {
	service := NewService(Config{
		LatestVersionCode:       20,
		MinSupportedVersionCode: 20,
		CurrentVersionAvailable: true,
	})

	resp := service.TaskPreflight(context.Background(), TaskPreflightRequest{VersionCode: 19})

	if resp.CanExecute || resp.BlockReason != "app_update_required" || !resp.AppUpdateRequired {
		t.Fatalf("expected app update to block task execution: %+v", resp)
	}
}

func TestResourceActionRecallMarksResourceRecalled(t *testing.T) {
	store := &captureBuildStore{}
	service := NewServiceWithStore(Config{}, store)

	if _, err := service.ResourceAction(context.Background(), "resource-1", "recall"); err != nil {
		t.Fatal(err)
	}
	if store.resourceStatus != "recalled" || store.resourcePublish {
		t.Fatalf("expected recalled resource without publish flag, status=%q publish=%v", store.resourceStatus, store.resourcePublish)
	}
}

func TestCheckMarksVersionUnavailable(t *testing.T) {
	service := NewService(Config{
		LatestVersionCode:       2,
		CurrentVersionAvailable: false,
		UnavailableReason:       "版本已下线",
	})

	resp := service.Check(context.Background(), CheckRequest{VersionCode: 2})

	if resp.CurrentVersionAvailable || !resp.ForceUpdate || resp.UnavailableReason != "版本已下线" {
		t.Fatalf("expected unavailable response: %+v", resp)
	}
}

func TestShouldAutoPauseResourceOnlyForActivationFailureWithTargetVersion(t *testing.T) {
	if !shouldAutoPauseResource(UpdateEventRequest{EventType: "activation_failed", ToVersion: "20260602.1"}) {
		t.Fatal("expected activation_failed with target resource version to auto pause")
	}
	if !shouldAutoPauseResource(UpdateEventRequest{EventType: "resource_activation_failed", ToVersion: "20260602.1"}) {
		t.Fatal("expected Android resource_activation_failed with target resource version to auto pause")
	}
	if shouldAutoPauseResource(UpdateEventRequest{EventType: "activation_failed"}) {
		t.Fatal("expected activation_failed without target version to skip auto pause")
	}
	if shouldAutoPauseResource(UpdateEventRequest{EventType: "download_failed", ToVersion: "20260602.1"}) {
		t.Fatal("expected download failure to skip auto pause")
	}
}

func TestInstallationStatusFromEventAllowsLifecycleEvents(t *testing.T) {
	for _, eventType := range []string{
		"download_started",
		"download_success",
		"download_failed",
		"install_started",
		"install_success",
		"install_failed",
		"activation_success",
		"activation_failed",
	} {
		if got := installationStatusFromEvent(eventType); got != eventType {
			t.Fatalf("expected %s to be tracked, got %q", eventType, got)
		}
	}
	if got := installationStatusFromEvent("webhook_received"); got != "" {
		t.Fatalf("expected unrelated event to be ignored, got %q", got)
	}
}

func TestCreateBuildKeepsBuildNumberSeparateFromVersionCode(t *testing.T) {
	store := &captureBuildStore{}
	service := NewServiceWithStore(Config{APKURL: "http://example.com/app.apk"}, store)

	resp, err := service.CreateBuild(context.Background(), CreateBuildRequest{
		GitRef:      "main",
		Channel:     "dev",
		BuildType:   "debug",
		VersionName: "0.2.0",
		VersionCode: 200,
		BuildNumber: 37,
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Job.VersionCode != 200 || resp.Job.BuildNumber != 37 {
		t.Fatalf("expected versionCode/buildNumber to stay separate, got %+v", resp.Job)
	}
	if store.build.VersionCode != 200 || store.build.BuildNumber != 37 {
		t.Fatalf("store received wrong build identity: %+v", store.build)
	}
}

func TestCreateBuildDefaultsBuildNumberToVersionCode(t *testing.T) {
	store := &captureBuildStore{}
	service := NewServiceWithStore(Config{APKURL: "http://example.com/app.apk"}, store)

	resp, err := service.CreateBuild(context.Background(), CreateBuildRequest{
		GitRef:      "main",
		Channel:     "dev",
		BuildType:   "debug",
		VersionName: "0.2.1",
		VersionCode: 201,
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Job.BuildNumber != 201 {
		t.Fatalf("expected buildNumber fallback to versionCode, got %+v", resp.Job)
	}
	if store.cfg.BuildNumber != 201 {
		t.Fatalf("expected config buildNumber fallback to versionCode, got %+v", store.cfg)
	}
}

func TestCreateArtifactUsesRequestedAppAndStorageKey(t *testing.T) {
	store := &captureBuildStore{}
	service := NewServiceWithStore(Config{AppKey: "default-app"}, store)

	resp, err := service.CreateArtifact(context.Background(), CreateArtifactRequest{
		AppKey:       "release-center",
		ArtifactName: "image-metadata",
		GitRef:       "main",
		GitCommit:    "abcdef123456",
		GitBranch:    "main",
		Channel:      "dev",
		BuildType:    "release",
		VersionName:  "0.1.0",
		VersionCode:  100,
		BuildNumber:  1,
		ArtifactType: "docker_image",
		FileName:     "image-metadata.json",
		SizeBytes:    1234,
		SHA256:       "sha256-value",
		Provider:     "build-center-local",
		Workflow:     "single-machine-image-build",
		RunID:        "run-1",
		StorageKey:   "app-releases/artifacts/image-metadata.json",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.Job.ArtifactType != "docker_image" || store.build.ArtifactType != "docker_image" {
		t.Fatalf("expected docker image artifact type, got resp=%+v store=%+v", resp.Job, store.build)
	}
	if resp.Artifact == nil || resp.Artifact.Name != "image-metadata" {
		t.Fatalf("expected artifact details in response, got %+v", resp)
	}
	if store.cfg.AppKey != "release-center" {
		t.Fatalf("expected artifact app_key to select release-center app, got %+v", store.cfg)
	}
	if store.artifact.StorageKey != "app-releases/artifacts/image-metadata.json" {
		t.Fatalf("expected artifact storage key to reach store, got %+v", store.artifact)
	}
	if store.artifact.ArtifactPath != "/api/v1/app/build-artifacts/"+store.artifact.ID+"/download" {
		t.Fatalf("expected storage-backed artifact download path, got %+v", store.artifact)
	}
}

func TestCreateReleaseMirrorsAndroidReleasePlan(t *testing.T) {
	store := newAndroidReleaseMirrorStore()
	service := NewServiceWithStore(Config{
		AppKey:  "game-helper-android",
		Name:    "Game Helper",
		Channel: "stable",
	}, store)

	resp, err := service.CreateRelease(context.Background(), CreateReleaseRequest{
		BuildID:           "build-1",
		Channel:           "stable",
		Title:             "Android 1.2.3",
		Summary:           "ship android",
		RolloutPercentage: 25,
		TargetType:        "device",
		TargetValue:       "device-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ReleasePlan == nil {
		t.Fatalf("expected mirrored release plan in response: %+v", resp)
	}
	if len(store.unitRequests) != 1 || store.unitRequests[0].UnitType != "android" {
		t.Fatalf("expected android release unit upsert, got %#v", store.unitRequests)
	}
	unitReq := store.unitRequests[0]
	if unitReq.ProjectKey != "release-center" || unitReq.UnitKey != "game-helper-android" || unitReq.AppID != "app-1" {
		t.Fatalf("unexpected release unit request: %#v", unitReq)
	}
	if len(store.planRequests) != 1 {
		t.Fatalf("expected one release plan request, got %#v", store.planRequests)
	}
	planReq := store.planRequests[0]
	if planReq.ProjectKey != "release-center" || planReq.UnitKey != "game-helper-android" {
		t.Fatalf("unexpected release plan linkage: %#v", planReq)
	}
	if planReq.EnvironmentKey != "prod" || planReq.Status != "draft" || planReq.RolloutPercentage != 25 {
		t.Fatalf("unexpected release plan state: %#v", planReq)
	}
	if len(planReq.Artifacts) != 1 || planReq.Artifacts[0].AppBuildArtifactID != "artifact-1" {
		t.Fatalf("expected app build artifact linkage, got %#v", planReq.Artifacts)
	}
	if planReq.Artifacts[0].ImmutableRef != "app_build_artifact:artifact-1" {
		t.Fatalf("unexpected immutable ref: %#v", planReq.Artifacts[0])
	}
}

func TestReleaseActionMirrorsAndroidReleasePlanStatus(t *testing.T) {
	store := newAndroidReleaseMirrorStore()
	service := NewServiceWithStore(Config{AppKey: "game-helper-android", Name: "Game Helper"}, store)

	resp, err := service.ReleaseAction(context.Background(), "release-1", "publish")
	if err != nil {
		t.Fatal(err)
	}
	if resp.ReleasePlan == nil {
		t.Fatalf("expected mirrored release plan in response: %+v", resp)
	}
	if len(store.planRequests) != 1 || store.planRequests[0].Status != "released" {
		t.Fatalf("expected released plan sync, got %#v", store.planRequests)
	}
}

func TestCreateReleaseMirrorFailureDoesNotBlockLegacyRelease(t *testing.T) {
	store := newAndroidReleaseMirrorStore()
	store.failCreatePlan = true
	service := NewServiceWithStore(Config{AppKey: "game-helper-android", Name: "Game Helper"}, store)

	resp, err := service.CreateRelease(context.Background(), CreateReleaseRequest{
		BuildID:           "build-1",
		Channel:           "stable",
		Title:             "Android 1.2.3",
		RolloutPercentage: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK || resp.Release.ID != "release-1" || resp.ReleasePlan != nil {
		t.Fatalf("legacy release should still succeed without mirrored plan: %+v", resp)
	}
	if _, ok := findCapturedAudit(store.auditEvents, "release_plan.android_mirror_failed"); !ok {
		t.Fatalf("expected mirror failure audit, got %#v", store.auditEvents)
	}
}

func TestAdminOverviewAppliesConfiguredQualityPolicy(t *testing.T) {
	store := &captureBuildStore{
		adminOverview: AdminOverview{
			ResourceUpdateEvents: []AppUpgradeEventAdmin{
				{EventType: "activation_failed"},
				{EventType: "activation_failed"},
				{EventType: "activation_failed"},
				{EventType: "activation_success"},
			},
		},
	}
	service := NewServiceWithStore(Config{
		QualityPolicy: QualityPolicy{
			ResourceActivationFailedCount: 4,
			ResourceFailureRate:           80,
		},
	}, store)

	overview, err := service.AdminOverview(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if overview.QualityPolicy.ResourceActivationFailedCount != 4 || overview.QualityPolicy.ResourceFailureRate != 80 {
		t.Fatalf("expected configured quality policy in overview, got %+v", overview.QualityPolicy)
	}
	var resourceMetric ReleaseQualityMetric
	for _, metric := range overview.QualityMetrics {
		if metric.Category == "resource" {
			resourceMetric = metric
			break
		}
	}
	if resourceMetric.Category == "" {
		t.Fatalf("expected resource quality metric, got %+v", overview.QualityMetrics)
	}
	if resourceMetric.RecommendedAction != "observe" {
		t.Fatalf("expected configured policy to keep resource metric observing, got %+v", resourceMetric)
	}
	if resourceMetric.PolicyThreshold != "activation_failed >= 4 且失败率 >= 80%" {
		t.Fatalf("unexpected threshold text: %+v", resourceMetric)
	}
}

type captureBuildStore struct {
	build           AppBuildJob
	artifact        AppBuildArtifact
	cfg             Config
	resourceStatus  string
	resourcePublish bool
	adminOverview   AdminOverview
}

func (s *captureBuildStore) EnsureConfiguredRelease(context.Context, Config) error {
	return nil
}

func (s *captureBuildStore) FindLatestRelease(context.Context, CheckRequest, Config) (Config, error) {
	return Config{}, errCaptureStoreNotFound
}

func (s *captureBuildStore) UpsertInstallation(context.Context, Config, CheckRequest) error {
	return nil
}

func (s *captureBuildStore) FindLatestResource(context.Context, ResourceCheckRequest, Config) (ResourceCandidate, error) {
	return ResourceCandidate{}, errCaptureStoreNotFound
}

func (s *captureBuildStore) SaveUpdateEvent(context.Context, Config, UpdateEventRequest) error {
	return nil
}

func (s *captureBuildStore) SaveHeartbeat(context.Context, Config, HeartbeatRequest) error {
	return nil
}

func (s *captureBuildStore) Overview(context.Context) (ReleaseOverview, error) {
	return ReleaseOverview{}, nil
}

func (s *captureBuildStore) AdminOverview(context.Context) (AdminOverview, error) {
	return s.adminOverview, nil
}

func (s *captureBuildStore) CreateApp(context.Context, CreateAppRequest) (AppAdminSummary, error) {
	return AppAdminSummary{}, nil
}

func (s *captureBuildStore) UpdateAppEnabled(context.Context, string, bool) (AppAdminSummary, error) {
	return AppAdminSummary{}, nil
}

func (s *captureBuildStore) CreateBuild(_ context.Context, build AppBuildJob, cfg Config) (AppBuildJob, error) {
	s.build = build
	s.cfg = cfg
	return build, nil
}

func (s *captureBuildStore) CreateBuildArtifact(_ context.Context, build AppBuildJob, artifact AppBuildArtifact, cfg Config) (AppBuildJob, AppBuildArtifact, error) {
	s.build = build
	s.artifact = artifact
	s.cfg = cfg
	build.Artifacts = []AppBuildArtifact{artifact}
	return build, artifact, nil
}

func (s *captureBuildStore) SaveWebhookEvent(context.Context, WebhookEventRequest) (WebhookEventAdmin, error) {
	return WebhookEventAdmin{}, nil
}

func (s *captureBuildStore) CreateRelease(context.Context, CreateReleaseRequest) (AppReleaseAdmin, error) {
	return AppReleaseAdmin{}, nil
}

func (s *captureBuildStore) UpdateReleaseStatus(context.Context, string, string, bool) (AppReleaseAdmin, error) {
	return AppReleaseAdmin{}, nil
}

func (s *captureBuildStore) UpdateReleaseRollout(context.Context, string, int) (AppReleaseAdmin, error) {
	return AppReleaseAdmin{}, nil
}

func (s *captureBuildStore) UpdateReleaseNotes(context.Context, string, UpdateNotesRequest) (AppReleaseAdmin, error) {
	return AppReleaseAdmin{}, nil
}

func (s *captureBuildStore) CreateResourceVersion(context.Context, CreateResourceVersionRequest, []AppResourcePackageAdmin, string, Config) (AppResourceVersionAdmin, error) {
	return AppResourceVersionAdmin{}, nil
}

func (s *captureBuildStore) UpdateResourceStatus(_ context.Context, _, status string, publish bool) (AppResourceVersionAdmin, error) {
	s.resourceStatus = status
	s.resourcePublish = publish
	return AppResourceVersionAdmin{}, nil
}

func (s *captureBuildStore) UpdateResourceRollout(context.Context, string, int) (AppResourceVersionAdmin, error) {
	return AppResourceVersionAdmin{}, nil
}

func (s *captureBuildStore) UpdateResourceNotes(context.Context, string, UpdateNotesRequest) (AppResourceVersionAdmin, error) {
	return AppResourceVersionAdmin{}, nil
}

func (s *captureBuildStore) GetBuildStorageKey(context.Context, string) (string, string, error) {
	return "", "", errCaptureStoreNotFound
}

func (s *captureBuildStore) GetBuildArtifactStorageKey(context.Context, string) (string, string, error) {
	return "", "", errCaptureStoreNotFound
}

func (s *captureBuildStore) GetReleaseStorageKey(context.Context, string) (string, string, error) {
	return "", "", errCaptureStoreNotFound
}

func (s *captureBuildStore) GetResourcePackageStorageKey(context.Context, string) (string, string, error) {
	return "", "", errCaptureStoreNotFound
}

func (s *captureBuildStore) GetResourceManifestStorageKey(context.Context, string) (string, string, error) {
	return "", "", errCaptureStoreNotFound
}

func (s *captureBuildStore) InsertAudit(context.Context, string, string, string, string, map[string]any) error {
	return nil
}

func (s *captureBuildStore) AuditExists(context.Context, string, string) (bool, error) {
	return false, nil
}

type androidReleaseMirrorStore struct {
	captureBuildStore
	release        AppReleaseAdmin
	buildLookup    AppBuildJob
	unitRequests   []CreateReleaseUnitRequest
	planRequests   []CreateReleasePlanRequest
	auditEvents    []capturedAudit
	failCreatePlan bool
}

func newAndroidReleaseMirrorStore() *androidReleaseMirrorStore {
	return &androidReleaseMirrorStore{
		release: AppReleaseAdmin{
			ID:                   "release-1",
			AppID:                "app-1",
			BuildID:              "build-1",
			PackageName:          "com.example.gamehelper",
			VersionName:          "1.2.3",
			VersionCode:          123,
			BuildNumber:          42,
			Channel:              "stable",
			BuildType:            "release",
			GitRef:               "main",
			GitCommit:            "abc123",
			ArtifactPath:         "/api/v1/app/builds/build-1/download",
			FileName:             "game-helper.apk",
			SizeBytes:            1024,
			SHA256:               "sha256-value",
			Status:               "draft",
			Title:                "Android 1.2.3",
			Summary:              "ship android",
			ReleaseNotesMarkdown: "notes",
			UpdateLevel:          "normal",
			RolloutPercentage:    25,
			TargetType:           "device",
			TargetValue:          "device-1",
			CreatedBy:            "admin",
		},
		buildLookup: AppBuildJob{
			ID:           "build-1",
			VersionName:  "1.2.3",
			VersionCode:  123,
			BuildNumber:  42,
			Channel:      "stable",
			BuildType:    "release",
			GitCommit:    "abc123",
			ArtifactType: "apk",
			ArtifactPath: "/api/v1/app/builds/build-1/download",
			FileName:     "game-helper.apk",
			ArtifactSize: 1024,
			SHA256:       "sha256-value",
			Artifacts: []AppBuildArtifact{{
				ID:           "artifact-1",
				BuildID:      "build-1",
				Name:         "release-apk",
				ArtifactType: "apk",
				ArtifactPath: "/api/v1/app/build-artifacts/artifact-1/download",
				FileName:     "game-helper.apk",
				SizeBytes:    1024,
				SHA256:       "sha256-value",
			}},
		},
	}
}

func (s *androidReleaseMirrorStore) CreateRelease(_ context.Context, req CreateReleaseRequest) (AppReleaseAdmin, error) {
	release := s.release
	release.BuildID = req.BuildID
	release.Channel = req.Channel
	release.Title = req.Title
	release.Summary = req.Summary
	release.ReleaseNotesMarkdown = req.ReleaseNotesMarkdown
	release.RolloutPercentage = normalizeRollout(req.RolloutPercentage)
	release.TargetType = normalizeTargetType(req.TargetType)
	release.TargetValue = req.TargetValue
	s.release = release
	return release, nil
}

func (s *androidReleaseMirrorStore) UpdateReleaseStatus(_ context.Context, id, status string, publish bool) (AppReleaseAdmin, error) {
	release := s.release
	release.ID = id
	release.Status = status
	s.release = release
	return release, nil
}

func (s *androidReleaseMirrorStore) AppBuild(_ context.Context, buildID string) (AppBuildJob, error) {
	if buildID != s.buildLookup.ID {
		return AppBuildJob{}, errCaptureStoreNotFound
	}
	return s.buildLookup, nil
}

func (s *androidReleaseMirrorStore) ReleasePlanOverview(context.Context) (ReleasePlanOverview, error) {
	return ReleasePlanOverview{}, nil
}

func (s *androidReleaseMirrorStore) ReleasePlan(context.Context, string) (ReleasePlanAdmin, error) {
	return ReleasePlanAdmin{}, errCaptureStoreNotFound
}

func (s *androidReleaseMirrorStore) CreateReleaseUnit(_ context.Context, req CreateReleaseUnitRequest) (ReleaseUnitAdmin, error) {
	s.unitRequests = append(s.unitRequests, req)
	return ReleaseUnitAdmin{
		ID:             "unit-1",
		ProjectID:      "project-1",
		ProjectKey:     req.ProjectKey,
		AppID:          req.AppID,
		UnitKey:        req.UnitKey,
		Name:           req.Name,
		UnitType:       req.UnitType,
		DefaultChannel: req.DefaultChannel,
		Enabled:        boolValue(req.Enabled, true),
	}, nil
}

func (s *androidReleaseMirrorStore) CreateReleasePlan(_ context.Context, req CreateReleasePlanRequest) (ReleasePlanAdmin, error) {
	s.planRequests = append(s.planRequests, req)
	if s.failCreatePlan {
		return ReleasePlanAdmin{}, errors.New("mirror failed")
	}
	return ReleasePlanAdmin{
		ID:                "plan-1",
		ProjectID:         "project-1",
		ProjectKey:        req.ProjectKey,
		ReleaseUnitID:     "unit-1",
		UnitKey:           req.UnitKey,
		UnitType:          "android",
		EnvironmentID:     "env-1",
		EnvironmentKey:    req.EnvironmentKey,
		PlanKey:           req.PlanKey,
		Title:             req.Title,
		Description:       req.Description,
		VersionName:       req.VersionName,
		BuildNumber:       req.BuildNumber,
		GitCommit:         req.GitCommit,
		Channel:           req.Channel,
		Status:            req.Status,
		RolloutPercentage: req.RolloutPercentage,
		TargetType:        req.TargetType,
		TargetValue:       req.TargetValue,
		CreatedBy:         req.CreatedBy,
		Artifacts: []ReleasePlanArtifactAdmin{{
			ID:                 "plan-artifact-1",
			AppBuildID:         req.Artifacts[0].AppBuildID,
			AppBuildArtifactID: req.Artifacts[0].AppBuildArtifactID,
			ArtifactName:       req.Artifacts[0].ArtifactName,
			ArtifactType:       req.Artifacts[0].ArtifactType,
			FileName:           req.Artifacts[0].FileName,
			ImmutableRef:       req.Artifacts[0].ImmutableRef,
		}},
	}, nil
}

func (s *androidReleaseMirrorStore) UpdateReleasePlanStatus(_ context.Context, planID, status string, req ReleasePlanActionRequest) (ReleasePlanAdmin, error) {
	return ReleasePlanAdmin{ID: planID, Status: status}, nil
}

func (s *androidReleaseMirrorStore) InsertAudit(_ context.Context, action, targetType, targetID, message string, metadata map[string]any) error {
	s.auditEvents = append(s.auditEvents, capturedAudit{
		action:     action,
		targetType: targetType,
		targetID:   targetID,
		message:    message,
		metadata:   metadata,
	})
	return nil
}
