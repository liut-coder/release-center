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
