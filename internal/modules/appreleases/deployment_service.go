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
	ApproveDeploymentRecord(ctx context.Context, deploymentID string, req ApproveDeploymentRequest) (DeploymentRecordAdmin, error)
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
	s.insertDeploymentAudit(ctx, "deployment.target_save", "deployment_target", target.ID, "保存部署目标", map[string]any{
		"project_id":  target.ProjectID,
		"target_key":  target.TargetKey,
		"provider":    target.Provider,
		"environment": target.Environment,
		"enabled":     target.Enabled,
	})
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
	if record.ProviderStatus == "pending_approval" {
		resp.MessageZh = "生产部署记录已创建，等待审批"
	} else if !req.DryRun {
		task, err := s.enqueueDeploymentWorkerTask(ctx, record, req)
		if err != nil {
			return DeploymentActionResponse{}, err
		}
		resp.WorkerTask = &task
		resp.MessageZh = "部署记录已创建并投递 Worker"
	}
	auditMetadata := deploymentRecordAuditMetadata(record)
	auditMetadata["dry_run"] = req.DryRun
	auditMetadata["worker_task_id"] = ""
	if resp.WorkerTask != nil {
		auditMetadata["worker_task_id"] = resp.WorkerTask.ID
	}
	s.insertDeploymentAudit(ctx, "deployment.create", "deployment_record", record.ID, "创建部署记录", auditMetadata)
	return resp, nil
}

func (s *Service) ApproveDeployment(ctx context.Context, deploymentID string, req ApproveDeploymentRequest) (DeploymentActionResponse, error) {
	store, ok := s.store.(DeploymentStore)
	if !ok {
		return DeploymentActionResponse{}, errDeploymentStoreUnavailable
	}
	deploymentID = strings.TrimSpace(deploymentID)
	req.ApprovedBy = strings.TrimSpace(firstNonBlank(req.ApprovedBy, "admin"))
	req.Comment = strings.TrimSpace(req.Comment)
	if deploymentID == "" {
		return DeploymentActionResponse{}, fmt.Errorf("deployment_id is required")
	}
	current, err := store.DeploymentRecord(ctx, deploymentID)
	if err != nil {
		return DeploymentActionResponse{}, err
	}
	if current.ProviderStatus != "pending_approval" {
		return DeploymentActionResponse{}, fmt.Errorf("deployment is not pending approval")
	}
	record, err := store.ApproveDeploymentRecord(ctx, deploymentID, req)
	if err != nil {
		return DeploymentActionResponse{}, err
	}
	resp := DeploymentActionResponse{OK: true, Record: &record, MessageZh: "部署已批准并投递 Worker"}
	task, err := s.enqueueDeploymentWorkerTask(ctx, record, CreateDeploymentRequest{Metadata: req.Metadata})
	if err != nil {
		return DeploymentActionResponse{}, err
	}
	resp.WorkerTask = &task
	metadata := deploymentRecordAuditMetadata(record)
	metadata["approved_by"] = req.ApprovedBy
	metadata["approval_comment"] = req.Comment
	metadata["worker_task_id"] = task.ID
	s.insertDeploymentAudit(ctx, "deployment.approve", "deployment_record", record.ID, "批准生产部署", metadata)
	return resp, nil
}

func (s *Service) CompleteDeployment(ctx context.Context, deploymentID string, req UpdateDeploymentStatusRequest) (DeploymentActionResponse, error) {
	req.ProviderStatus = firstNonBlank(req.ProviderStatus, "success")
	resp, err := s.updateDeploymentStatus(ctx, deploymentID, req, "部署已完成")
	if err != nil {
		return DeploymentActionResponse{}, err
	}
	if resp.Record != nil {
		s.insertDeploymentAudit(ctx, "deployment.complete", "deployment_record", resp.Record.ID, "部署已完成", deploymentRecordAuditMetadata(*resp.Record))
	}
	return resp, nil
}

func (s *Service) FailDeployment(ctx context.Context, deploymentID string, req UpdateDeploymentStatusRequest) (DeploymentActionResponse, error) {
	req.ProviderStatus = firstNonBlank(req.ProviderStatus, "failed")
	if strings.TrimSpace(req.ErrorMessage) == "" {
		req.ErrorMessage = "deployment failed"
	}
	resp, err := s.updateDeploymentStatus(ctx, deploymentID, req, "部署失败已记录")
	if err != nil {
		return DeploymentActionResponse{}, err
	}
	if resp.Record != nil {
		metadata := deploymentRecordAuditMetadata(*resp.Record)
		metadata["error_message"] = resp.Record.ErrorMessage
		s.insertDeploymentAudit(ctx, "deployment.fail", "deployment_record", resp.Record.ID, "部署失败已记录", metadata)
	}
	return resp, nil
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
	if resp.Record != nil {
		metadata := deploymentRecordAuditMetadata(*resp.Record)
		metadata["dry_run"] = req.DryRun
		metadata["rollback_from_deployment_id"] = current.ID
		metadata["rollback_to_deployment_id"] = previous.ID
		metadata["rollback_reason"] = req.Reason
		if resp.WorkerTask != nil {
			metadata["worker_task_id"] = resp.WorkerTask.ID
		}
		s.insertDeploymentAudit(ctx, "deployment.rollback", "deployment_record", resp.Record.ID, "创建回滚部署", metadata)
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

func (s *Service) insertDeploymentAudit(ctx context.Context, action, targetType, targetID, message string, metadata map[string]any) {
	s.insertAudit(ctx, action, targetType, targetID, message, metadata)
}

func deploymentRecordAuditMetadata(record DeploymentRecordAdmin) map[string]any {
	return map[string]any{
		"target_id":              record.TargetID,
		"target_key":             record.TargetKey,
		"provider":               record.Provider,
		"environment":            record.Environment,
		"provider_status":        record.ProviderStatus,
		"version_name":           record.VersionName,
		"build_number":           record.BuildNumber,
		"git_commit":             record.GitCommit,
		"run_id":                 record.RunID,
		"app_build_id":           record.AppBuildID,
		"app_build_artifact_id":  record.AppBuildArtifactID,
		"deployment_url":         record.DeploymentURL,
		"external_deployment_id": record.ExternalDeploymentID,
		"triggered_by":           record.TriggeredBy,
	}
}

func deploymentRecordMetadataForTarget(target DeploymentTargetAdmin, req CreateDeploymentRequest) map[string]any {
	metadata := mergeMaps(req.Metadata, map[string]any{
		"provider":               target.Provider,
		"environment":            target.Environment,
		"deployment_target_id":   target.ID,
		"deployment_target_key":  target.TargetKey,
		"deployment_target_name": target.Name,
	})
	if target.CredentialRef != "" {
		metadata["credential_ref"] = target.CredentialRef
	}
	if strings.HasPrefix(target.Provider, "cloudflare_") {
		if target.CloudflareAccountID != "" {
			metadata["cloudflare_account_id"] = target.CloudflareAccountID
		}
		if target.CloudflareProjectName != "" {
			metadata["cloudflare_project_name"] = target.CloudflareProjectName
		}
		if target.CloudflareScriptName != "" {
			metadata["cloudflare_script_name"] = target.CloudflareScriptName
		}
		if target.CloudflareBucketName != "" {
			metadata["cloudflare_bucket_name"] = target.CloudflareBucketName
		}
		command := cloudflareDeploymentCommand(target, CreateDeploymentRequest{Metadata: metadata})
		if len(command) > 0 {
			metadata["prepared_command"] = command
		}
	}
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
	case "queued", "pending_approval", "running", "success", "failed", "canceled", "dry_run", "external":
		return status
	default:
		return "queued"
	}
}

func deploymentPendingApproval(record DeploymentRecordAdmin) bool {
	return record.ProviderStatus == "pending_approval"
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
