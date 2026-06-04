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
