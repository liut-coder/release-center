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
	if got := normalizeDeploymentStatus(" pending_approval "); got != "pending_approval" {
		t.Fatalf("expected pending_approval, got %q", got)
	}
	if got := normalizeDeploymentStatus("unknown"); got != "queued" {
		t.Fatalf("expected fallback queued, got %q", got)
	}
}

func TestDeploymentTargetRequiresApproval(t *testing.T) {
	target := DeploymentTargetAdmin{Environment: "prod"}
	if !deploymentTargetRequiresApproval(target, CreateDeploymentRequest{}) {
		t.Fatal("expected prod deployment to require approval")
	}
	if deploymentTargetRequiresApproval(target, CreateDeploymentRequest{DryRun: true}) {
		t.Fatal("dry-run should not require approval")
	}
	if deploymentTargetRequiresApproval(DeploymentTargetAdmin{Environment: "staging"}, CreateDeploymentRequest{}) {
		t.Fatal("staging should not require approval")
	}
	if deploymentTargetRequiresApproval(target, CreateDeploymentRequest{Metadata: map[string]any{"approval_status": "approved"}}) {
		t.Fatal("approved deployment should not require approval again")
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
	if len(store.auditEvents) != 1 || store.auditEvents[0].Action != "deployment.create" {
		t.Fatalf("expected deployment.create audit, got %#v", store.auditEvents)
	}
	if store.auditEvents[0].Metadata["worker_task_id"] != "worker-task-1" {
		t.Fatalf("worker_task_id missing from audit metadata: %#v", store.auditEvents[0].Metadata)
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
	if len(store.auditEvents) != 1 || store.auditEvents[0].Action != "deployment.create" {
		t.Fatalf("expected dry-run deployment.create audit, got %#v", store.auditEvents)
	}
	if store.auditEvents[0].Metadata["dry_run"] != true {
		t.Fatalf("dry_run missing from audit metadata: %#v", store.auditEvents[0].Metadata)
	}
}

func TestCreateDeploymentWaitsForApproval(t *testing.T) {
	store := &deploymentDispatchStore{
		record: DeploymentRecordAdmin{
			ID:             "deployment-approval",
			TargetID:       "target-1",
			Provider:       "cloudflare_pages",
			Environment:    "prod",
			ProviderStatus: "pending_approval",
		},
	}
	service := &Service{store: store}

	resp, err := service.CreateDeployment(context.Background(), CreateDeploymentRequest{
		TargetID:    "target-1",
		VersionName: "1.2.3",
		DryRun:      false,
	})
	if err != nil {
		t.Fatalf("CreateDeployment() error = %v", err)
	}
	if resp.WorkerTask != nil || len(store.workerRequests) != 0 {
		t.Fatalf("pending approval deployment should not enqueue worker: resp=%#v requests=%#v", resp, store.workerRequests)
	}
	if resp.MessageZh != "生产部署记录已创建，等待审批" {
		t.Fatalf("unexpected message: %q", resp.MessageZh)
	}
}

func TestApproveDeploymentEnqueuesWorker(t *testing.T) {
	store := &deploymentDispatchStore{
		currentRecord: DeploymentRecordAdmin{
			ID:             "deployment-approval",
			TargetID:       "target-1",
			TargetKey:      "pages",
			Provider:       "cloudflare_pages",
			Environment:    "prod",
			ProviderStatus: "pending_approval",
			Metadata:       []byte(`{"prepared_command":["wrangler","pages","deploy","dist"]}`),
		},
		approvedRecord: DeploymentRecordAdmin{
			ID:             "deployment-approval",
			TargetID:       "target-1",
			TargetKey:      "pages",
			Provider:       "cloudflare_pages",
			Environment:    "prod",
			ProviderStatus: "queued",
			Metadata:       []byte(`{"prepared_command":["wrangler","pages","deploy","dist"],"approval_status":"approved"}`),
		},
	}
	service := &Service{store: store}

	resp, err := service.ApproveDeployment(context.Background(), "deployment-approval", ApproveDeploymentRequest{
		ApprovedBy: "release-admin",
		Comment:    "ship it",
	})
	if err != nil {
		t.Fatalf("ApproveDeployment() error = %v", err)
	}
	if resp.WorkerTask == nil {
		t.Fatalf("expected worker task after approval")
	}
	if len(store.approvalRequests) != 1 || store.approvalRequests[0].ApprovedBy != "release-admin" {
		t.Fatalf("approval request not captured: %#v", store.approvalRequests)
	}
	if len(store.workerRequests) != 1 {
		t.Fatalf("expected one worker request, got %d", len(store.workerRequests))
	}
	if len(store.auditEvents) != 1 || store.auditEvents[0].Action != "deployment.approve" {
		t.Fatalf("expected deployment.approve audit, got %#v", store.auditEvents)
	}
	if store.auditEvents[0].Metadata["approved_by"] != "release-admin" {
		t.Fatalf("approved_by missing from audit metadata: %#v", store.auditEvents[0].Metadata)
	}
}

func TestRollbackDeploymentCreatesDryRunFromPreviousSuccess(t *testing.T) {
	store := &deploymentDispatchStore{
		currentRecord: DeploymentRecordAdmin{
			ID:             "deployment-current",
			TargetID:       "target-1",
			Provider:       "cloudflare_pages",
			ProviderStatus: "failed",
		},
		previousRecord: DeploymentRecordAdmin{
			ID:                   "deployment-previous",
			TargetID:             "target-1",
			RunID:                "run-1",
			AppBuildID:           "build-1",
			AppBuildArtifactID:   "artifact-1",
			Provider:             "cloudflare_pages",
			ProviderStatus:       "success",
			DeploymentURL:        "https://previous.example",
			VersionName:          "1.0.0",
			BuildNumber:          10,
			GitCommit:            "abc123",
			ExternalDeploymentID: "cf-old",
			Metadata:             []byte(`{"artifact_path":"dist-old","deployment_record_id":"old-task-link","prepared_command":["stale"]}`),
		},
		record: DeploymentRecordAdmin{
			ID:             "deployment-rollback",
			TargetID:       "target-1",
			Provider:       "cloudflare_pages",
			ProviderStatus: "dry_run",
			VersionName:    "1.0.0",
			BuildNumber:    10,
		},
	}
	service := &Service{store: store}

	resp, err := service.RollbackDeployment(context.Background(), "deployment-current", RollbackDeploymentRequest{
		DryRun:      true,
		TriggeredBy: "tester",
		Reason:      "bad deploy",
	})
	if err != nil {
		t.Fatalf("RollbackDeployment() error = %v", err)
	}
	if resp.Record == nil || resp.Record.ID != "deployment-rollback" {
		t.Fatalf("unexpected rollback response: %#v", resp)
	}
	if resp.WorkerTask != nil || len(store.workerRequests) != 0 {
		t.Fatalf("dry-run rollback should not enqueue worker: resp=%#v requests=%#v", resp, store.workerRequests)
	}
	if len(store.deploymentRequests) != 1 {
		t.Fatalf("expected one deployment request, got %d", len(store.deploymentRequests))
	}
	req := store.deploymentRequests[0]
	if !req.DryRun || req.TargetID != "target-1" || req.VersionName != "1.0.0" || req.BuildNumber != 10 {
		t.Fatalf("unexpected rollback deployment request: %#v", req)
	}
	if req.Metadata["source"] != "deployment_rollback" || req.Metadata["rollback_to_deployment_id"] != "deployment-previous" {
		t.Fatalf("rollback metadata missing: %#v", req.Metadata)
	}
	if _, ok := req.Metadata["deployment_record_id"]; ok {
		t.Fatalf("rollback metadata should not keep previous worker linkage: %#v", req.Metadata)
	}
	if _, ok := req.Metadata["prepared_command"]; ok {
		t.Fatalf("rollback metadata should let deployment creation regenerate prepared_command: %#v", req.Metadata)
	}
	if len(store.auditEvents) != 2 {
		t.Fatalf("expected create and rollback audit events, got %#v", store.auditEvents)
	}
	if store.auditEvents[1].Action != "deployment.rollback" {
		t.Fatalf("expected deployment.rollback audit, got %#v", store.auditEvents)
	}
	if store.auditEvents[1].Metadata["rollback_to_deployment_id"] != "deployment-previous" {
		t.Fatalf("rollback audit metadata missing target: %#v", store.auditEvents[1].Metadata)
	}
}

func TestRollbackDeploymentEnqueuesWorkerForRealRollback(t *testing.T) {
	store := &deploymentDispatchStore{
		currentRecord: DeploymentRecordAdmin{
			ID:             "deployment-current",
			TargetID:       "target-1",
			Provider:       "cloudflare_pages",
			ProviderStatus: "failed",
		},
		previousRecord: DeploymentRecordAdmin{
			ID:             "deployment-previous",
			TargetID:       "target-1",
			Provider:       "cloudflare_pages",
			ProviderStatus: "success",
			VersionName:    "1.0.0",
			BuildNumber:    10,
			GitCommit:      "abc123",
			Metadata:       []byte(`{"artifact_path":"dist-old"}`),
		},
		record: DeploymentRecordAdmin{
			ID:             "deployment-rollback",
			TargetID:       "target-1",
			TargetKey:      "pages",
			Provider:       "cloudflare_pages",
			ProviderStatus: "queued",
			VersionName:    "1.0.0",
			BuildNumber:    10,
			GitCommit:      "abc123",
			Metadata:       []byte(`{"prepared_command":["wrangler","pages","deploy","dist-old"]}`),
		},
	}
	service := &Service{store: store}

	resp, err := service.RollbackDeployment(context.Background(), "deployment-current", RollbackDeploymentRequest{
		DryRun: false,
	})
	if err != nil {
		t.Fatalf("RollbackDeployment() error = %v", err)
	}
	if resp.WorkerTask == nil {
		t.Fatalf("expected worker task for real rollback")
	}
	if len(store.workerRequests) != 1 {
		t.Fatalf("expected one worker request, got %d", len(store.workerRequests))
	}
	if store.workerRequests[0].Metadata["rollback_from_deployment_id"] != "deployment-current" {
		t.Fatalf("worker metadata missing rollback source: %#v", store.workerRequests[0].Metadata)
	}
	if len(store.auditEvents) != 2 || store.auditEvents[1].Action != "deployment.rollback" {
		t.Fatalf("expected rollback audit events, got %#v", store.auditEvents)
	}
}

type deploymentAuditEvent struct {
	Action     string
	TargetType string
	TargetID   string
	Message    string
	Metadata   map[string]any
}

type deploymentDispatchStore struct {
	Store
	DeploymentStore
	WorkerStore
	record             DeploymentRecordAdmin
	currentRecord      DeploymentRecordAdmin
	previousRecord     DeploymentRecordAdmin
	approvedRecord     DeploymentRecordAdmin
	deploymentRequests []CreateDeploymentRequest
	approvalRequests   []ApproveDeploymentRequest
	workerRequests     []CreateWorkerTaskRequest
	auditEvents        []deploymentAuditEvent
}

func (s *deploymentDispatchStore) CreateDeploymentRecord(ctx context.Context, req CreateDeploymentRequest) (DeploymentRecordAdmin, error) {
	s.deploymentRequests = append(s.deploymentRequests, req)
	return s.record, nil
}

func (s *deploymentDispatchStore) DeploymentRecord(ctx context.Context, deploymentID string) (DeploymentRecordAdmin, error) {
	return s.currentRecord, nil
}

func (s *deploymentDispatchStore) PreviousSuccessfulDeploymentRecord(ctx context.Context, deploymentID string) (DeploymentRecordAdmin, error) {
	return s.previousRecord, nil
}

func (s *deploymentDispatchStore) ApproveDeploymentRecord(ctx context.Context, deploymentID string, req ApproveDeploymentRequest) (DeploymentRecordAdmin, error) {
	s.approvalRequests = append(s.approvalRequests, req)
	return s.approvedRecord, nil
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

func (s *deploymentDispatchStore) InsertAudit(ctx context.Context, action, targetType, targetID, message string, metadata map[string]any) error {
	s.auditEvents = append(s.auditEvents, deploymentAuditEvent{
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Message:    message,
		Metadata:   metadata,
	})
	return nil
}
