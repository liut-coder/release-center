package appreleases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var errDeploymentStoreUnavailable = errors.New("deployment store unavailable")

type DeploymentStore interface {
	CreateDeploymentTarget(ctx context.Context, req CreateDeploymentTargetRequest) (DeploymentTargetAdmin, error)
	DeploymentRecords(ctx context.Context) ([]DeploymentRecordAdmin, error)
	DeploymentRecord(ctx context.Context, deploymentID string) (DeploymentRecordAdmin, error)
	PreviousSuccessfulDeploymentRecord(ctx context.Context, deploymentID string) (DeploymentRecordAdmin, error)
	CreateDeploymentRecord(ctx context.Context, req CreateDeploymentRequest) (DeploymentRecordAdmin, error)
	UpdateDeploymentRecordStatus(ctx context.Context, deploymentID string, req UpdateDeploymentStatusRequest) (DeploymentRecordAdmin, error)
}

func (s *Service) CreateDeploymentTarget(ctx context.Context, req CreateDeploymentTargetRequest) (DeploymentActionResponse, error) {
	store, ok := s.store.(DeploymentStore)
	if !ok {
		return DeploymentActionResponse{}, errDeploymentStoreUnavailable
	}
	req.ProjectKey = strings.TrimSpace(req.ProjectKey)
	req.TargetKey = strings.TrimSpace(req.TargetKey)
	req.Name = strings.TrimSpace(req.Name)
	req.Provider = normalizeDeploymentProvider(req.Provider)
	req.Environment = normalizeDeploymentEnvironment(req.Environment)
	req.EndpointURL = strings.TrimSpace(req.EndpointURL)
	req.CredentialRef = strings.TrimSpace(req.CredentialRef)
	if req.ProjectKey == "" || req.TargetKey == "" || req.Provider == "" {
		return DeploymentActionResponse{}, fmt.Errorf("project_key, target_key and provider are required")
	}
	if req.Name == "" {
		req.Name = req.TargetKey
	}
	target, err := store.CreateDeploymentTarget(ctx, req)
	if err != nil {
		return DeploymentActionResponse{}, err
	}
	return DeploymentActionResponse{OK: true, Target: &target, MessageZh: "部署目标已保存"}, nil
}

func (s *Service) DeploymentRecords(ctx context.Context) (DeploymentRecordsResponse, error) {
	store, ok := s.store.(DeploymentStore)
	if !ok {
		return DeploymentRecordsResponse{DeploymentRecords: []DeploymentRecordAdmin{}, MessageZh: "部署记录未连接数据库"}, nil
	}
	records, err := store.DeploymentRecords(ctx)
	if err != nil {
		return DeploymentRecordsResponse{}, err
	}
	return DeploymentRecordsResponse{DeploymentRecords: records, MessageZh: "部署记录已读取"}, nil
}

func (s *Service) DeploymentRecord(ctx context.Context, deploymentID string) (DeploymentActionResponse, error) {
	store, ok := s.store.(DeploymentStore)
	if !ok {
		return DeploymentActionResponse{}, errDeploymentStoreUnavailable
	}
	deploymentID = strings.TrimSpace(deploymentID)
	if deploymentID == "" {
		return DeploymentActionResponse{}, fmt.Errorf("deployment_id is required")
	}
	record, err := store.DeploymentRecord(ctx, deploymentID)
	if err != nil {
		return DeploymentActionResponse{}, err
	}
	return DeploymentActionResponse{OK: true, Record: &record, MessageZh: "部署记录已读取"}, nil
}

func (s *Service) CreateDeployment(ctx context.Context, req CreateDeploymentRequest) (DeploymentActionResponse, error) {
	store, ok := s.store.(DeploymentStore)
	if !ok {
		return DeploymentActionResponse{}, errDeploymentStoreUnavailable
	}
	req.ProjectKey = strings.TrimSpace(req.ProjectKey)
	req.TargetID = strings.TrimSpace(req.TargetID)
	req.TargetKey = strings.TrimSpace(req.TargetKey)
	req.ProviderStatus = normalizeDeploymentStatus(req.ProviderStatus)
	req.TriggeredBy = strings.TrimSpace(firstNonBlank(req.TriggeredBy, "admin"))
	req.LogTail = normalizeWorkerLogLines(req.LogTail)
	if req.TargetID == "" && (req.ProjectKey == "" || req.TargetKey == "") {
		return DeploymentActionResponse{}, fmt.Errorf("target_id or project_key + target_key is required")
	}
	if req.DryRun {
		req.ProviderStatus = "dry_run"
		req.Metadata = mergeMaps(req.Metadata, map[string]any{
			"execution_mode": "dry_run",
		})
	}
	record, err := store.CreateDeploymentRecord(ctx, req)
	if err != nil {
		return DeploymentActionResponse{}, err
	}
	resp := DeploymentActionResponse{OK: true, Record: &record, MessageZh: "部署记录已创建"}
	if !req.DryRun {
		task, err := s.enqueueDeploymentWorkerTask(ctx, record, req)
		if err != nil {
			return DeploymentActionResponse{}, err
		}
		resp.WorkerTask = &task
		resp.MessageZh = "部署记录已创建并投递 Worker"
	}
	return resp, nil
}

func (s *Service) CompleteDeployment(ctx context.Context, deploymentID string, req UpdateDeploymentStatusRequest) (DeploymentActionResponse, error) {
	req.ProviderStatus = firstNonBlank(req.ProviderStatus, "success")
	return s.updateDeploymentStatus(ctx, deploymentID, req, "部署已完成")
}

func (s *Service) FailDeployment(ctx context.Context, deploymentID string, req UpdateDeploymentStatusRequest) (DeploymentActionResponse, error) {
	req.ProviderStatus = firstNonBlank(req.ProviderStatus, "failed")
	if strings.TrimSpace(req.ErrorMessage) == "" {
		req.ErrorMessage = "deployment failed"
	}
	return s.updateDeploymentStatus(ctx, deploymentID, req, "部署失败已记录")
}

func (s *Service) RollbackDeployment(ctx context.Context, deploymentID string, req RollbackDeploymentRequest) (DeploymentActionResponse, error) {
	store, ok := s.store.(DeploymentStore)
	if !ok {
		return DeploymentActionResponse{}, errDeploymentStoreUnavailable
	}
	deploymentID = strings.TrimSpace(deploymentID)
	req.TriggeredBy = strings.TrimSpace(firstNonBlank(req.TriggeredBy, "admin"))
	req.Reason = strings.TrimSpace(req.Reason)
	if deploymentID == "" {
		return DeploymentActionResponse{}, fmt.Errorf("deployment_id is required")
	}
	current, err := store.DeploymentRecord(ctx, deploymentID)
	if err != nil {
		return DeploymentActionResponse{}, err
	}
	previous, err := store.PreviousSuccessfulDeploymentRecord(ctx, deploymentID)
	if err != nil {
		return DeploymentActionResponse{}, err
	}
	deployReq := CreateDeploymentRequest{
		TargetID:           current.TargetID,
		RunID:              previous.RunID,
		AppBuildID:         previous.AppBuildID,
		AppBuildArtifactID: previous.AppBuildArtifactID,
		VersionName:        previous.VersionName,
		BuildNumber:        previous.BuildNumber,
		GitCommit:          previous.GitCommit,
		TriggeredBy:        req.TriggeredBy,
		DryRun:             req.DryRun,
		LogTail: []string{
			fmt.Sprintf("rollback requested from deployment %s to previous success %s", current.ID, previous.ID),
		},
		Metadata: rollbackDeploymentMetadata(current, previous, req),
	}
	resp, err := s.CreateDeployment(ctx, deployReq)
	if err != nil {
		return DeploymentActionResponse{}, err
	}
	resp.RollbackSource = &current
	resp.RollbackTarget = &previous
	if req.DryRun {
		resp.MessageZh = "回滚 dry-run 部署记录已创建"
	} else {
		resp.MessageZh = "回滚部署记录已创建并投递 Worker"
	}
	return resp, nil
}

func (s *Service) updateDeploymentStatus(ctx context.Context, deploymentID string, req UpdateDeploymentStatusRequest, message string) (DeploymentActionResponse, error) {
	store, ok := s.store.(DeploymentStore)
	if !ok {
		return DeploymentActionResponse{}, errDeploymentStoreUnavailable
	}
	deploymentID = strings.TrimSpace(deploymentID)
	req.ProviderStatus = normalizeDeploymentStatus(req.ProviderStatus)
	req.LogTail = normalizeWorkerLogLines(req.LogTail)
	if deploymentID == "" {
		return DeploymentActionResponse{}, fmt.Errorf("deployment_id is required")
	}
	record, err := store.UpdateDeploymentRecordStatus(ctx, deploymentID, req)
	if err != nil {
		return DeploymentActionResponse{}, err
	}
	return DeploymentActionResponse{OK: true, Record: &record, MessageZh: message}, nil
}

func rollbackDeploymentMetadata(current, previous DeploymentRecordAdmin, req RollbackDeploymentRequest) map[string]any {
	metadata := rawJSONMap(previous.Metadata)
	for _, key := range []string{
		"deployment_record_id",
		"deployment_target_id",
		"deployment_provider",
		"deployment_target",
		"execution_mode",
		"external_deployment_id",
		"prepared_command",
		"provider",
		"worker_key",
		"worker_status",
		"worker_task_id",
	} {
		delete(metadata, key)
	}
	metadata = mergeMaps(metadata, req.Metadata)
	metadata = mergeMaps(metadata, map[string]any{
		"source":                      "deployment_rollback",
		"rollback_from_deployment_id": current.ID,
		"rollback_from_status":        current.ProviderStatus,
		"rollback_to_deployment_id":   previous.ID,
		"rollback_to_version_name":    previous.VersionName,
		"rollback_to_build_number":    previous.BuildNumber,
		"rollback_to_git_commit":      previous.GitCommit,
		"rollback_to_deployment_url":  previous.DeploymentURL,
		"rollback_reason":             req.Reason,
	})
	return metadata
}

func normalizeDeploymentProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch provider {
	case "cloudflare_pages", "cloudflare_worker", "cloudflare_r2", "generic_webhook", "ssh", "docker", "kubernetes":
		return provider
	default:
		return provider
	}
}

func normalizeDeploymentEnvironment(environment string) string {
	environment = strings.ToLower(strings.TrimSpace(environment))
	if environment == "" {
		return "prod"
	}
	return environment
}

func normalizeDeploymentStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "queued", "running", "success", "failed", "canceled", "dry_run", "external":
		return status
	default:
		return "queued"
	}
}

func mergeMaps(base map[string]any, extra map[string]any) map[string]any {
	merged := map[string]any{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func cloudflareDeploymentCommand(target DeploymentTargetAdmin, req CreateDeploymentRequest) []string {
	artifactPath := stringFromAny(req.Metadata["artifact_path"])
	switch target.Provider {
	case "cloudflare_pages":
		args := []string{"wrangler", "pages", "deploy", firstNonBlank(artifactPath, "dist")}
		if target.CloudflareProjectName != "" {
			args = append(args, "--project-name", target.CloudflareProjectName)
		}
		return args
	case "cloudflare_worker":
		args := []string{"wrangler", "deploy"}
		if target.CloudflareScriptName != "" {
			args = append(args, "--name", target.CloudflareScriptName)
		}
		return args
	case "cloudflare_r2":
		objectKey := firstNonBlank(stringFromAny(req.Metadata["object_key"]), "artifact")
		args := []string{"wrangler", "r2", "object", "put", target.CloudflareBucketName + "/" + objectKey}
		if artifactPath != "" {
			args = append(args, "--file", artifactPath)
		}
		return args
	default:
		return []string{}
	}
}

func (s *Service) enqueueDeploymentWorkerTask(ctx context.Context, record DeploymentRecordAdmin, req CreateDeploymentRequest) (WorkerTaskAdmin, error) {
	workerStore, ok := s.store.(WorkerStore)
	if !ok {
		return WorkerTaskAdmin{}, errWorkerStoreUnavailable
	}
	metadata := mergeMaps(rawJSONMap(record.Metadata), req.Metadata)
	metadata = mergeMaps(metadata, map[string]any{
		"source":               "deployment_record",
		"deployment_record_id": record.ID,
		"deployment_target_id": record.TargetID,
		"deployment_target":    record.TargetKey,
		"deployment_provider":  record.Provider,
		"environment":          record.Environment,
		"version_name":         record.VersionName,
		"build_number":         record.BuildNumber,
		"git_commit":           record.GitCommit,
	})
	task, err := workerStore.CreateWorkerTask(ctx, CreateWorkerTaskRequest{
		ProjectKey:     req.ProjectKey,
		BuildRunID:     record.RunID,
		TaskType:       "deploy",
		Action:         deploymentWorkerAction(record.Provider),
		RequiredLabels: deploymentWorkerLabels(record.Provider),
		Priority:       deploymentWorkerPriority(record.Environment),
		Metadata:       metadata,
	})
	if err != nil {
		return WorkerTaskAdmin{}, err
	}
	return task, nil
}

func deploymentWorkerAction(provider string) string {
	switch provider {
	case "cloudflare_pages", "cloudflare_worker", "cloudflare_r2", "docker", "kubernetes", "ssh", "generic_webhook":
		return provider
	default:
		return "deploy"
	}
}

func deploymentWorkerLabels(provider string) []string {
	switch provider {
	case "cloudflare_pages", "cloudflare_worker", "cloudflare_r2":
		return []string{"cloudflare"}
	case "docker":
		return []string{"docker"}
	case "kubernetes":
		return []string{"kubernetes"}
	case "ssh":
		return []string{"ssh"}
	default:
		return []string{}
	}
}

func deploymentWorkerPriority(environment string) int {
	switch normalizeDeploymentEnvironment(environment) {
	case "prod":
		return 30
	case "staging":
		return 20
	default:
		return 10
	}
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func rawJSONMap(data []byte) map[string]any {
	if len(data) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}
