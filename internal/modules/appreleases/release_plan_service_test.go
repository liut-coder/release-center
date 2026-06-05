package appreleases

import (
	"context"
	"strings"
	"testing"
)

func TestNormalizeReleaseUnitType(t *testing.T) {
	if got := normalizeReleaseUnitType(" WEB "); got != "web" {
		t.Fatalf("expected web, got %q", got)
	}
	if got := normalizeReleaseUnitType("custom"); got != "custom" {
		t.Fatalf("expected custom passthrough, got %q", got)
	}
}

func TestNormalizeReleasePlanStatus(t *testing.T) {
	if got := normalizeReleasePlanStatus(" PAUSED "); got != "paused" {
		t.Fatalf("expected paused, got %q", got)
	}
	if got := normalizeReleasePlanStatus("unknown"); got != "draft" {
		t.Fatalf("expected draft fallback, got %q", got)
	}
}

func TestNormalizeReleasePlanArtifactsDefaultsNameAndType(t *testing.T) {
	items := normalizeReleasePlanArtifacts([]ReleasePlanArtifactRequest{{FileName: "web-dist.tar.gz"}})
	if len(items) != 1 {
		t.Fatalf("expected one artifact, got %#v", items)
	}
	if items[0].ArtifactName != "web-dist.tar.gz" || items[0].ArtifactType != "artifact" {
		t.Fatalf("unexpected artifact defaults: %#v", items[0])
	}
}

func TestDefaultReleasePlanKeyIncludesUnitEnvironmentAndVersion(t *testing.T) {
	key := defaultReleasePlanKey(CreateReleasePlanRequest{
		UnitKey:        "admin-web",
		EnvironmentKey: "prod",
		VersionName:    "1.0.0",
	})
	if !strings.HasPrefix(key, "admin-web-prod-1.0.0-") {
		t.Fatalf("unexpected key prefix: %q", key)
	}
}

func TestCreateReleasePlanDeploymentCreatesDeploymentPerArtifact(t *testing.T) {
	store := &releasePlanDeploymentStore{
		plan: ReleasePlanAdmin{
			ID:                "plan-1",
			ProjectKey:        "release-center",
			ReleaseUnitID:     "unit-1",
			UnitKey:           "admin-web",
			EnvironmentKey:    "prod",
			PlanKey:           "admin-web-prod-1",
			VersionName:       "1.2.3",
			BuildNumber:       42,
			GitCommit:         "abc123",
			RolloutPercentage: 100,
			TargetType:        "all",
			Artifacts: []ReleasePlanArtifactAdmin{
				{
					BuildRunID:         "run-1",
					AppBuildID:         "build-1",
					AppBuildArtifactID: "artifact-1",
					ArtifactName:       "web-dist",
					ArtifactType:       "web_dist",
					FileName:           "dist.tar.gz",
					ImmutableRef:       "build_center_artifact:artifact-1",
				},
			},
		},
	}
	service := &Service{store: store}

	resp, err := service.CreateReleasePlanDeployment(context.Background(), "plan-1", CreateReleasePlanDeploymentRequest{
		TargetID:    "target-1",
		DryRun:      true,
		TriggeredBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateReleasePlanDeployment() error = %v", err)
	}
	if !resp.OK || len(resp.DeploymentRecords) != 1 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if len(store.deploymentRequests) != 1 {
		t.Fatalf("expected one deployment request, got %d", len(store.deploymentRequests))
	}
	req := store.deploymentRequests[0]
	if req.TargetID != "target-1" || req.ProjectKey != "release-center" || req.AppBuildArtifactID != "artifact-1" {
		t.Fatalf("unexpected deployment request linkage: %#v", req)
	}
	if req.VersionName != "1.2.3" || req.BuildNumber != 42 || req.GitCommit != "abc123" {
		t.Fatalf("unexpected deployment version fields: %#v", req)
	}
	if req.Metadata["release_plan_id"] != "plan-1" || req.Metadata["immutable_ref"] != "build_center_artifact:artifact-1" {
		t.Fatalf("unexpected metadata: %#v", req.Metadata)
	}
	audit, ok := findCapturedAudit(store.audits, "release_plan.deploy")
	if !ok {
		t.Fatalf("expected release plan deployment audit, got %#v", store.audits)
	}
	if audit.targetID != "plan-1" || audit.metadata["deployment_count"] != 1 || audit.metadata["dry_run"] != true {
		t.Fatalf("unexpected deployment audit: %#v", audit)
	}
}

func TestCreateReleasePlanInsertsAudit(t *testing.T) {
	store := &releasePlanAuditStore{}
	service := &Service{cfg: Config{Channel: "stable"}, store: store}

	resp, err := service.CreateReleasePlan(context.Background(), CreateReleasePlanRequest{
		ProjectKey:        "release-center",
		UnitKey:           "admin-web",
		EnvironmentKey:    "prod",
		Title:             "Web prod 1.2.3",
		VersionName:       "1.2.3",
		BuildNumber:       42,
		GitCommit:         "abc123",
		RolloutPercentage: 100,
		TargetType:        "all",
		CreatedBy:         "tester",
		Artifacts: []ReleasePlanArtifactRequest{{
			ArtifactName: "web-dist",
			ArtifactType: "web_dist",
			FileName:     "dist.tar.gz",
			ImmutableRef: "build_center_artifact:artifact-1",
		}},
	})
	if err != nil {
		t.Fatalf("CreateReleasePlan() error = %v", err)
	}
	if !resp.OK || resp.Plan.ID != "plan-1" {
		t.Fatalf("unexpected response: %#v", resp)
	}
	audit, ok := findCapturedAudit(store.audits, "release_plan.create")
	if !ok {
		t.Fatalf("expected release plan create audit, got %#v", store.audits)
	}
	if audit.metadata["plan_key"] != resp.Plan.PlanKey || audit.metadata["artifact_count"] != 1 {
		t.Fatalf("unexpected create audit metadata: %#v", audit.metadata)
	}
}

func TestReleasePlanActionInsertsAudit(t *testing.T) {
	store := &releasePlanAuditStore{}
	service := &Service{store: store}

	resp, err := service.ReleasePlanAction(context.Background(), "plan-1", "publish", ReleasePlanActionRequest{
		ApprovedBy: "approver",
	})
	if err != nil {
		t.Fatalf("ReleasePlanAction() error = %v", err)
	}
	if resp.Plan.Status != "released" {
		t.Fatalf("expected released status, got %#v", resp.Plan)
	}
	audit, ok := findCapturedAudit(store.audits, "release_plan.publish")
	if !ok {
		t.Fatalf("expected release plan publish audit, got %#v", store.audits)
	}
	if audit.metadata["status"] != "released" || audit.metadata["approved_by"] != "approver" {
		t.Fatalf("unexpected action audit metadata: %#v", audit.metadata)
	}
}

func TestReleasePlanRollbackCreatesPlanFromPreviousRelease(t *testing.T) {
	store := newReleasePlanRollbackStore()
	service := &Service{store: store}

	resp, err := service.ReleasePlanAction(context.Background(), "plan-current", "rollback", ReleasePlanActionRequest{
		ApprovedBy: "release-admin",
	})
	if err != nil {
		t.Fatalf("ReleasePlanAction(rollback) error = %v", err)
	}
	if resp.Plan.Status != "rolled_back" {
		t.Fatalf("expected current plan rolled_back, got %#v", resp.Plan)
	}
	if resp.RollbackPlan == nil {
		t.Fatalf("expected rollback plan in response: %#v", resp)
	}
	if len(store.createdPlans) != 1 {
		t.Fatalf("expected one rollback plan create request, got %#v", store.createdPlans)
	}
	req := store.createdPlans[0]
	if req.VersionName != "1.0.0" || req.BuildNumber != 10 || req.GitCommit != "old123" {
		t.Fatalf("rollback plan should target previous version: %#v", req)
	}
	if req.Status != "draft" || !strings.HasPrefix(req.PlanKey, "rollback-admin-web-prod-2-to-admin-web-prod-1") {
		t.Fatalf("unexpected rollback plan key/status: %#v", req)
	}
	if len(req.Artifacts) != 1 || req.Artifacts[0].AppBuildArtifactID != "artifact-old" {
		t.Fatalf("rollback plan should reuse previous artifacts: %#v", req.Artifacts)
	}
	if req.Metadata["rollback_from_plan_id"] != "plan-current" || req.Metadata["rollback_target_plan_id"] != "plan-previous" {
		t.Fatalf("rollback metadata missing linkage: %#v", req.Metadata)
	}
	if _, ok := findCapturedAudit(store.audits, "release_plan.rollback_plan"); !ok {
		t.Fatalf("expected rollback plan audit, got %#v", store.audits)
	}
}

func TestReleasePlanRollbackCreatesDeploymentWhenTargetProvided(t *testing.T) {
	store := newReleasePlanRollbackStore()
	service := &Service{store: store}

	resp, err := service.ReleasePlanAction(context.Background(), "plan-current", "rollback", ReleasePlanActionRequest{
		TargetID:    "target-1",
		DryRun:      true,
		TriggeredBy: "tester",
	})
	if err != nil {
		t.Fatalf("ReleasePlanAction(rollback deploy) error = %v", err)
	}
	if resp.RollbackPlan == nil || len(resp.DeploymentRecords) != 1 {
		t.Fatalf("expected rollback plan and deployment record, got %#v", resp)
	}
	if len(store.deploymentRequests) != 1 {
		t.Fatalf("expected one deployment request, got %#v", store.deploymentRequests)
	}
	req := store.deploymentRequests[0]
	if req.TargetID != "target-1" || req.AppBuildArtifactID != "artifact-old" || req.VersionName != "1.0.0" {
		t.Fatalf("unexpected rollback deployment request: %#v", req)
	}
	if req.Metadata["rollback_from_plan_id"] != "plan-current" || req.Metadata["rollback_to_plan_id"] != "plan-previous" {
		t.Fatalf("rollback deployment metadata missing linkage: %#v", req.Metadata)
	}
}

type releasePlanDeploymentStore struct {
	Store
	ReleasePlanStore
	DeploymentStore
	plan               ReleasePlanAdmin
	deploymentRequests []CreateDeploymentRequest
	audits             []capturedAudit
}

func (s *releasePlanDeploymentStore) ReleasePlan(ctx context.Context, planID string) (ReleasePlanAdmin, error) {
	return s.plan, nil
}

func (s *releasePlanDeploymentStore) CreateDeploymentRecord(ctx context.Context, req CreateDeploymentRequest) (DeploymentRecordAdmin, error) {
	s.deploymentRequests = append(s.deploymentRequests, req)
	return DeploymentRecordAdmin{
		ID:                 "deployment-1",
		TargetID:           req.TargetID,
		RunID:              req.RunID,
		AppBuildID:         req.AppBuildID,
		AppBuildArtifactID: req.AppBuildArtifactID,
		VersionName:        req.VersionName,
		BuildNumber:        req.BuildNumber,
		GitCommit:          req.GitCommit,
		ProviderStatus:     "dry_run",
	}, nil
}

func (s *releasePlanDeploymentStore) InsertAudit(_ context.Context, action, targetType, targetID, message string, metadata map[string]any) error {
	s.audits = append(s.audits, capturedAudit{
		action:     action,
		targetType: targetType,
		targetID:   targetID,
		message:    message,
		metadata:   metadata,
	})
	return nil
}

type releasePlanAuditStore struct {
	Store
	ReleasePlanStore
	audits []capturedAudit
}

func (s *releasePlanAuditStore) InsertAudit(_ context.Context, action, targetType, targetID, message string, metadata map[string]any) error {
	s.audits = append(s.audits, capturedAudit{
		action:     action,
		targetType: targetType,
		targetID:   targetID,
		message:    message,
		metadata:   metadata,
	})
	return nil
}

func (s *releasePlanAuditStore) CreateReleasePlan(_ context.Context, req CreateReleasePlanRequest) (ReleasePlanAdmin, error) {
	return ReleasePlanAdmin{
		ID:                "plan-1",
		ProjectID:         "project-1",
		ProjectKey:        req.ProjectKey,
		ReleaseUnitID:     "unit-1",
		UnitKey:           req.UnitKey,
		UnitType:          "web",
		EnvironmentID:     "env-1",
		EnvironmentKey:    req.EnvironmentKey,
		PlanKey:           req.PlanKey,
		Title:             req.Title,
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
			ID:           "plan-artifact-1",
			ArtifactName: req.Artifacts[0].ArtifactName,
			ArtifactType: req.Artifacts[0].ArtifactType,
			FileName:     req.Artifacts[0].FileName,
			ImmutableRef: req.Artifacts[0].ImmutableRef,
		}},
	}, nil
}

func (s *releasePlanAuditStore) UpdateReleasePlanStatus(_ context.Context, planID, status string, req ReleasePlanActionRequest) (ReleasePlanAdmin, error) {
	return ReleasePlanAdmin{
		ID:                planID,
		ProjectID:         "project-1",
		ProjectKey:        "release-center",
		ReleaseUnitID:     "unit-1",
		UnitKey:           "admin-web",
		UnitType:          "web",
		EnvironmentID:     "env-1",
		EnvironmentKey:    "prod",
		PlanKey:           "admin-web-prod-1",
		VersionName:       "1.2.3",
		BuildNumber:       42,
		GitCommit:         "abc123",
		Channel:           "stable",
		Status:            status,
		RolloutPercentage: 100,
		TargetType:        "all",
		CreatedBy:         "tester",
		ApprovedBy:        req.ApprovedBy,
	}, nil
}

type releasePlanRollbackStore struct {
	Store
	ReleasePlanStore
	ReleasePlanRollbackStore
	DeploymentStore
	plans              map[string]ReleasePlanAdmin
	previous           ReleasePlanAdmin
	createdPlans       []CreateReleasePlanRequest
	deploymentRequests []CreateDeploymentRequest
	audits             []capturedAudit
}

func newReleasePlanRollbackStore() *releasePlanRollbackStore {
	current := ReleasePlanAdmin{
		ID:                "plan-current",
		ProjectID:         "project-1",
		ProjectKey:        "release-center",
		ReleaseUnitID:     "unit-1",
		UnitKey:           "admin-web",
		UnitType:          "web",
		EnvironmentID:     "env-1",
		EnvironmentKey:    "prod",
		PlanKey:           "admin-web-prod-2",
		Title:             "Admin Web 2.0.0",
		VersionName:       "2.0.0",
		BuildNumber:       20,
		GitCommit:         "new123",
		Channel:           "stable",
		Status:            "released",
		RolloutPercentage: 100,
		TargetType:        "all",
		CreatedBy:         "tester",
	}
	previous := ReleasePlanAdmin{
		ID:                "plan-previous",
		ProjectID:         "project-1",
		ProjectKey:        "release-center",
		ReleaseUnitID:     "unit-1",
		UnitKey:           "admin-web",
		UnitType:          "web",
		EnvironmentID:     "env-1",
		EnvironmentKey:    "prod",
		PlanKey:           "admin-web-prod-1",
		Title:             "Admin Web 1.0.0",
		VersionName:       "1.0.0",
		BuildNumber:       10,
		GitCommit:         "old123",
		Channel:           "stable",
		Status:            "released",
		RolloutPercentage: 100,
		TargetType:        "all",
		CreatedBy:         "tester",
		Artifacts: []ReleasePlanArtifactAdmin{{
			ID:                 "plan-artifact-old",
			ReleasePlanID:      "plan-previous",
			AppBuildID:         "build-old",
			AppBuildArtifactID: "artifact-old",
			ArtifactName:       "web-dist",
			ArtifactType:       "web_dist",
			FileName:           "dist-old.tar.gz",
			ImmutableRef:       "app_build_artifact:artifact-old",
		}},
	}
	return &releasePlanRollbackStore{
		plans: map[string]ReleasePlanAdmin{
			current.ID:  current,
			previous.ID: previous,
		},
		previous: previous,
	}
}

func (s *releasePlanRollbackStore) ReleasePlan(_ context.Context, planID string) (ReleasePlanAdmin, error) {
	plan, ok := s.plans[planID]
	if !ok {
		return ReleasePlanAdmin{}, errCaptureStoreNotFound
	}
	return plan, nil
}

func (s *releasePlanRollbackStore) PreviousReleasePlan(_ context.Context, planID string) (ReleasePlanAdmin, error) {
	return s.previous, nil
}

func (s *releasePlanRollbackStore) CreateReleaseUnit(context.Context, CreateReleaseUnitRequest) (ReleaseUnitAdmin, error) {
	return ReleaseUnitAdmin{}, nil
}

func (s *releasePlanRollbackStore) CreateReleasePlan(_ context.Context, req CreateReleasePlanRequest) (ReleasePlanAdmin, error) {
	s.createdPlans = append(s.createdPlans, req)
	plan := ReleasePlanAdmin{
		ID:                "plan-rollback",
		ProjectID:         "project-1",
		ProjectKey:        req.ProjectKey,
		ReleaseUnitID:     "unit-1",
		UnitKey:           req.UnitKey,
		UnitType:          "web",
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
	}
	for _, artifact := range req.Artifacts {
		plan.Artifacts = append(plan.Artifacts, ReleasePlanArtifactAdmin{
			ID:                 "rollback-artifact-1",
			ReleasePlanID:      plan.ID,
			BuildRunID:         artifact.BuildRunID,
			AppBuildID:         artifact.AppBuildID,
			AppBuildArtifactID: artifact.AppBuildArtifactID,
			ArtifactName:       artifact.ArtifactName,
			ArtifactType:       artifact.ArtifactType,
			FileName:           artifact.FileName,
			ImmutableRef:       artifact.ImmutableRef,
		})
	}
	s.plans[plan.ID] = plan
	return plan, nil
}

func (s *releasePlanRollbackStore) UpdateReleasePlanStatus(_ context.Context, planID, status string, req ReleasePlanActionRequest) (ReleasePlanAdmin, error) {
	plan := s.plans[planID]
	plan.Status = status
	plan.ApprovedBy = req.ApprovedBy
	s.plans[planID] = plan
	return plan, nil
}

func (s *releasePlanRollbackStore) CreateDeploymentRecord(_ context.Context, req CreateDeploymentRequest) (DeploymentRecordAdmin, error) {
	s.deploymentRequests = append(s.deploymentRequests, req)
	return DeploymentRecordAdmin{
		ID:                 "deployment-rollback-1",
		TargetID:           req.TargetID,
		RunID:              req.RunID,
		AppBuildID:         req.AppBuildID,
		AppBuildArtifactID: req.AppBuildArtifactID,
		VersionName:        req.VersionName,
		BuildNumber:        req.BuildNumber,
		GitCommit:          req.GitCommit,
		ProviderStatus:     "dry_run",
	}, nil
}

func (s *releasePlanRollbackStore) InsertAudit(_ context.Context, action, targetType, targetID, message string, metadata map[string]any) error {
	s.audits = append(s.audits, capturedAudit{
		action:     action,
		targetType: targetType,
		targetID:   targetID,
		message:    message,
		metadata:   metadata,
	})
	return nil
}
