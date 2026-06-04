package appreleases

import (
	"context"
	"testing"
)

func TestCloudflareDeploymentCommandPages(t *testing.T) {
	cmd := cloudflareDeploymentCommand(DeploymentTargetAdmin{
		Provider:              "cloudflare_pages",
		CloudflareProjectName: "release-center-admin",
	}, CreateDeploymentRequest{Metadata: map[string]any{"artifact_path": "web/dist"}})

	want := []string{"wrangler", "pages", "deploy", "web/dist", "--project-name", "release-center-admin"}
	if len(cmd) != len(want) {
		t.Fatalf("expected %d args, got %#v", len(want), cmd)
	}
	for i := range want {
		if cmd[i] != want[i] {
			t.Fatalf("cmd[%d] = %q, want %q", i, cmd[i], want[i])
		}
	}
}

func TestCloudflareDeploymentCommandR2(t *testing.T) {
	cmd := cloudflareDeploymentCommand(DeploymentTargetAdmin{
		Provider:             "cloudflare_r2",
		CloudflareBucketName: "release-artifacts",
	}, CreateDeploymentRequest{Metadata: map[string]any{
		"artifact_path": "dist/web.tar.gz",
		"object_key":    "web.tar.gz",
	}})

	want := []string{"wrangler", "r2", "object", "put", "release-artifacts/web.tar.gz", "--file", "dist/web.tar.gz"}
	if len(cmd) != len(want) {
		t.Fatalf("expected %d args, got %#v", len(want), cmd)
	}
	for i := range want {
		if cmd[i] != want[i] {
			t.Fatalf("cmd[%d] = %q, want %q", i, cmd[i], want[i])
		}
	}
}

func TestNormalizeDeploymentStatus(t *testing.T) {
	if got := normalizeDeploymentStatus(" SUCCESS "); got != "success" {
		t.Fatalf("expected success, got %q", got)
	}
	if got := normalizeDeploymentStatus("unknown"); got != "queued" {
		t.Fatalf("expected fallback queued, got %q", got)
	}
}

func TestCreateDeploymentEnqueuesWorkerForRealDeployment(t *testing.T) {
	store := &deploymentDispatchStore{
		record: DeploymentRecordAdmin{
			ID:             "deployment-1",
			TargetID:       "target-1",
			TargetKey:      "pages",
			Provider:       "cloudflare_pages",
			Environment:    "prod",
			VersionName:    "1.2.3",
			BuildNumber:    7,
			GitCommit:      "abc123",
			ProviderStatus: "queued",
			Metadata:       []byte(`{"prepared_command":["wrangler","pages","deploy","dist"]}`),
		},
	}
	service := &Service{store: store}

	resp, err := service.CreateDeployment(context.Background(), CreateDeploymentRequest{
		ProjectKey:  "release-center",
		TargetID:    "target-1",
		VersionName: "1.2.3",
		BuildNumber: 7,
		GitCommit:   "abc123",
		TriggeredBy: "tester",
		DryRun:      false,
		Metadata:    map[string]any{"artifact_path": "dist"},
	})
	if err != nil {
		t.Fatalf("CreateDeployment() error = %v", err)
	}
	if resp.WorkerTask == nil {
		t.Fatalf("expected worker task in response")
	}
	if len(store.workerRequests) != 1 {
		t.Fatalf("expected one worker request, got %d", len(store.workerRequests))
	}
	req := store.workerRequests[0]
	if req.TaskType != "deploy" || req.Action != "cloudflare_pages" {
		t.Fatalf("unexpected worker task type/action: %#v", req)
	}
	if len(req.RequiredLabels) != 1 || req.RequiredLabels[0] != "cloudflare" {
		t.Fatalf("unexpected labels: %#v", req.RequiredLabels)
	}
	if req.Metadata["deployment_record_id"] != "deployment-1" {
		t.Fatalf("deployment_record_id missing from metadata: %#v", req.Metadata)
	}
}

func TestCreateDeploymentDoesNotEnqueueWorkerForDryRun(t *testing.T) {
	store := &deploymentDispatchStore{
		record: DeploymentRecordAdmin{
			ID:             "deployment-1",
			TargetID:       "target-1",
			Provider:       "cloudflare_pages",
			ProviderStatus: "dry_run",
		},
	}
	service := &Service{store: store}

	resp, err := service.CreateDeployment(context.Background(), CreateDeploymentRequest{
		TargetID:    "target-1",
		VersionName: "1.2.3",
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("CreateDeployment() error = %v", err)
	}
	if resp.WorkerTask != nil || len(store.workerRequests) != 0 {
		t.Fatalf("dry-run should not enqueue worker task: resp=%#v requests=%#v", resp, store.workerRequests)
	}
}

type deploymentDispatchStore struct {
	Store
	DeploymentStore
	WorkerStore
	record         DeploymentRecordAdmin
	workerRequests []CreateWorkerTaskRequest
}

func (s *deploymentDispatchStore) CreateDeploymentRecord(ctx context.Context, req CreateDeploymentRequest) (DeploymentRecordAdmin, error) {
	return s.record, nil
}

func (s *deploymentDispatchStore) CreateWorkerTask(ctx context.Context, req CreateWorkerTaskRequest) (WorkerTaskAdmin, error) {
	s.workerRequests = append(s.workerRequests, req)
	return WorkerTaskAdmin{
		ID:             "worker-task-1",
		TaskType:       req.TaskType,
		Action:         req.Action,
		Status:         "queued",
		RequiredLabels: req.RequiredLabels,
		Metadata:       jsonb(req.Metadata),
	}, nil
}
