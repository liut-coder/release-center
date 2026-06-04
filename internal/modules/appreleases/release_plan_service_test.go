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
}

type releasePlanDeploymentStore struct {
	Store
	ReleasePlanStore
	DeploymentStore
	plan               ReleasePlanAdmin
	deploymentRequests []CreateDeploymentRequest
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
