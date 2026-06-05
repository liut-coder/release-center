package appreleases

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var errReleasePlanStoreUnavailable = errors.New("release plan store unavailable")

type ReleasePlanStore interface {
	ReleasePlanOverview(ctx context.Context) (ReleasePlanOverview, error)
	ReleasePlan(ctx context.Context, planID string) (ReleasePlanAdmin, error)
	CreateReleaseUnit(ctx context.Context, req CreateReleaseUnitRequest) (ReleaseUnitAdmin, error)
	CreateReleasePlan(ctx context.Context, req CreateReleasePlanRequest) (ReleasePlanAdmin, error)
	UpdateReleasePlanStatus(ctx context.Context, planID, status string, req ReleasePlanActionRequest) (ReleasePlanAdmin, error)
}

type ReleasePlanRollbackStore interface {
	PreviousReleasePlan(ctx context.Context, planID string) (ReleasePlanAdmin, error)
}

func (s *Service) ReleasePlanOverview(ctx context.Context) (ReleasePlanOverview, error) {
	store, ok := s.store.(ReleasePlanStore)
	if !ok {
		return ReleasePlanOverview{
			Environments: []ReleaseEnvironmentAdmin{},
			ReleaseUnits: []ReleaseUnitAdmin{},
			ReleasePlans: []ReleasePlanAdmin{},
			MessageZh:    "发布计划未连接数据库",
		}, nil
	}
	overview, err := store.ReleasePlanOverview(ctx)
	if err != nil {
		return ReleasePlanOverview{}, err
	}
	overview.MessageZh = "发布计划已读取"
	return overview, nil
}

func (s *Service) ReleasePlan(ctx context.Context, planID string) (ReleasePlanActionResponse, error) {
	store, ok := s.store.(ReleasePlanStore)
	if !ok {
		return ReleasePlanActionResponse{}, errReleasePlanStoreUnavailable
	}
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return ReleasePlanActionResponse{}, fmt.Errorf("plan_id is required")
	}
	plan, err := store.ReleasePlan(ctx, planID)
	if err != nil {
		return ReleasePlanActionResponse{}, err
	}
	return ReleasePlanActionResponse{OK: true, Plan: plan, MessageZh: "发布计划已读取"}, nil
}

func (s *Service) CreateReleaseUnit(ctx context.Context, req CreateReleaseUnitRequest) (map[string]any, error) {
	store, ok := s.store.(ReleasePlanStore)
	if !ok {
		return nil, errReleasePlanStoreUnavailable
	}
	req.ProjectKey = strings.TrimSpace(req.ProjectKey)
	req.UnitKey = strings.TrimSpace(req.UnitKey)
	req.Name = strings.TrimSpace(req.Name)
	req.UnitType = normalizeReleaseUnitType(req.UnitType)
	req.DefaultChannel = normalizeChannel(req.DefaultChannel, s.cfg.Channel)
	req.AppID = strings.TrimSpace(req.AppID)
	if req.ProjectKey == "" || req.UnitKey == "" || req.Name == "" || req.UnitType == "" {
		return nil, fmt.Errorf("project_key, unit_key, name and unit_type are required")
	}
	unit, err := store.CreateReleaseUnit(ctx, req)
	if err != nil {
		return nil, err
	}
	s.insertAudit(ctx, "release_unit.save", "release_unit", unit.ID, "保存发布单元", map[string]any{
		"project_key":     unit.ProjectKey,
		"project_id":      unit.ProjectID,
		"unit_key":        unit.UnitKey,
		"unit_type":       unit.UnitType,
		"default_channel": unit.DefaultChannel,
		"enabled":         unit.Enabled,
	})
	return map[string]any{"ok": true, "release_unit": unit, "message_zh": "发布单元已保存"}, nil
}

func (s *Service) CreateReleasePlan(ctx context.Context, req CreateReleasePlanRequest) (ReleasePlanActionResponse, error) {
	store, ok := s.store.(ReleasePlanStore)
	if !ok {
		return ReleasePlanActionResponse{}, errReleasePlanStoreUnavailable
	}
	req.ProjectKey = strings.TrimSpace(req.ProjectKey)
	req.UnitKey = strings.TrimSpace(req.UnitKey)
	req.EnvironmentKey = normalizeDeploymentEnvironment(req.EnvironmentKey)
	req.PlanKey = strings.TrimSpace(req.PlanKey)
	req.Title = strings.TrimSpace(req.Title)
	req.VersionName = strings.TrimSpace(req.VersionName)
	req.GitCommit = strings.TrimSpace(req.GitCommit)
	req.Channel = normalizeChannel(req.Channel, s.cfg.Channel)
	req.Status = normalizeReleasePlanStatus(req.Status)
	req.RolloutPercentage = normalizeRollout(req.RolloutPercentage)
	req.TargetType = normalizeTargetType(req.TargetType)
	req.TargetValue = strings.TrimSpace(req.TargetValue)
	req.ApprovedBy = strings.TrimSpace(req.ApprovedBy)
	req.CreatedBy = strings.TrimSpace(firstNonBlank(req.CreatedBy, "admin"))
	req.Artifacts = normalizeReleasePlanArtifacts(req.Artifacts)
	if req.ProjectKey == "" || req.UnitKey == "" || req.EnvironmentKey == "" || req.Title == "" {
		return ReleasePlanActionResponse{}, fmt.Errorf("project_key, unit_key, environment_key and title are required")
	}
	if req.TargetType != "all" && req.TargetValue == "" {
		return ReleasePlanActionResponse{}, fmt.Errorf("target_value is required for %s", req.TargetType)
	}
	if req.PlanKey == "" {
		req.PlanKey = defaultReleasePlanKey(req)
	}
	if req.ScheduledAt != "" {
		if _, err := time.Parse(time.RFC3339, req.ScheduledAt); err != nil {
			return ReleasePlanActionResponse{}, fmt.Errorf("scheduled_at must be RFC3339")
		}
	}
	plan, err := store.CreateReleasePlan(ctx, req)
	if err != nil {
		return ReleasePlanActionResponse{}, err
	}
	s.insertAudit(ctx, "release_plan.create", "release_plan", plan.ID, "创建发布计划", releasePlanAuditMetadata(plan, map[string]any{
		"artifact_count": len(plan.Artifacts),
	}))
	return ReleasePlanActionResponse{OK: true, Plan: plan, MessageZh: "发布计划已创建"}, nil
}

func (s *Service) ReleasePlanAction(ctx context.Context, planID, action string, req ReleasePlanActionRequest) (ReleasePlanActionResponse, error) {
	store, ok := s.store.(ReleasePlanStore)
	if !ok {
		return ReleasePlanActionResponse{}, errReleasePlanStoreUnavailable
	}
	planID = strings.TrimSpace(planID)
	action = strings.TrimSpace(action)
	req.ApprovedBy = strings.TrimSpace(req.ApprovedBy)
	req.TargetID = strings.TrimSpace(req.TargetID)
	req.TargetKey = strings.TrimSpace(req.TargetKey)
	req.TriggeredBy = strings.TrimSpace(req.TriggeredBy)
	req.DeploymentURL = strings.TrimSpace(req.DeploymentURL)
	if action == "rollback" {
		return s.rollbackReleasePlan(ctx, store, planID, req)
	}
	if planID == "" {
		return ReleasePlanActionResponse{}, fmt.Errorf("plan_id is required")
	}
	current, err := store.ReleasePlan(ctx, planID)
	if err != nil {
		return ReleasePlanActionResponse{}, err
	}
	status := ""
	message := "发布计划状态已更新"
	auditAction := action
	metadata := map[string]any{
		"approved_by": req.ApprovedBy,
	}
	switch action {
	case "publish":
		if releasePlanRequiresApproval(current) {
			if current.Status == "pending_approval" {
				return ReleasePlanActionResponse{}, fmt.Errorf("release plan is already pending approval")
			}
			requestedBy := firstNonBlank(req.TriggeredBy, req.ApprovedBy, "admin")
			status = "pending_approval"
			message = "生产发布计划已提交审批"
			auditAction = "publish_request"
			req.Metadata = mergeMaps(req.Metadata, map[string]any{
				"approval_required": true,
				"requested_status":  "released",
				"previous_status":   current.Status,
				"requested_by":      requestedBy,
			})
			req.ApprovedBy = ""
			metadata["approval_required"] = true
			metadata["requested_status"] = "released"
			metadata["previous_status"] = current.Status
			metadata["requested_by"] = requestedBy
			metadata["approved_by"] = ""
		} else {
			status = "released"
			message = "发布计划已发布"
		}
	case "approve":
		if current.Status != "pending_approval" {
			return ReleasePlanActionResponse{}, fmt.Errorf("release plan is not pending approval")
		}
		if req.ApprovedBy == "" {
			req.ApprovedBy = firstNonBlank(req.TriggeredBy, "admin")
			metadata["approved_by"] = req.ApprovedBy
		}
		status = "released"
		message = "发布计划已审批发布"
		metadata["approval_required"] = releasePlanRequiresApproval(current)
		metadata["previous_status"] = current.Status
	case "pause":
		status = "paused"
		message = "发布计划已暂停"
	default:
		return ReleasePlanActionResponse{}, fmt.Errorf("unsupported release plan action: %s", action)
	}
	plan, err := store.UpdateReleasePlanStatus(ctx, planID, status, req)
	if err != nil {
		return ReleasePlanActionResponse{}, err
	}
	s.insertAudit(ctx, "release_plan."+auditAction, "release_plan", plan.ID, message, releasePlanAuditMetadata(plan, metadata))
	return ReleasePlanActionResponse{OK: true, Plan: plan, MessageZh: message}, nil
}

func (s *Service) rollbackReleasePlan(ctx context.Context, store ReleasePlanStore, planID string, req ReleasePlanActionRequest) (ReleasePlanActionResponse, error) {
	if planID == "" {
		return ReleasePlanActionResponse{}, fmt.Errorf("plan_id is required")
	}
	rollbackStore, ok := s.store.(ReleasePlanRollbackStore)
	if !ok {
		return ReleasePlanActionResponse{}, fmt.Errorf("release plan rollback store unavailable")
	}
	current, err := store.ReleasePlan(ctx, planID)
	if err != nil {
		return ReleasePlanActionResponse{}, err
	}
	previous, err := rollbackStore.PreviousReleasePlan(ctx, planID)
	if err != nil {
		return ReleasePlanActionResponse{}, err
	}
	rollbackResp, err := s.CreateReleasePlan(ctx, rollbackReleasePlanRequest(current, previous, req))
	if err != nil {
		return ReleasePlanActionResponse{}, err
	}
	rollbackPlan := rollbackResp.Plan
	resp := ReleasePlanActionResponse{
		OK:           true,
		RollbackPlan: &rollbackPlan,
		MessageZh:    "发布计划已回滚，已生成回滚计划",
	}
	if releasePlanRollbackDeploymentRequested(req) {
		deployResp, err := s.CreateReleasePlanDeployment(ctx, rollbackPlan.ID, CreateReleasePlanDeploymentRequest{
			TargetID:      req.TargetID,
			TargetKey:     req.TargetKey,
			DryRun:        req.DryRun,
			TriggeredBy:   firstNonBlank(req.TriggeredBy, req.ApprovedBy, "rollback"),
			DeploymentURL: req.DeploymentURL,
			Metadata: mergeMaps(req.Metadata, map[string]any{
				"source":                    "release_plan_rollback",
				"rollback_from_plan_id":     current.ID,
				"rollback_from_plan_key":    current.PlanKey,
				"rollback_to_plan_id":       previous.ID,
				"rollback_to_plan_key":      previous.PlanKey,
				"rollback_release_plan_id":  rollbackPlan.ID,
				"rollback_release_plan_key": rollbackPlan.PlanKey,
			}),
		})
		if err != nil {
			return ReleasePlanActionResponse{}, err
		}
		resp.DeploymentRecords = deployResp.DeploymentRecords
		resp.WorkerTasks = deployResp.WorkerTasks
		resp.MessageZh = releasePlanRollbackMessage(req.DryRun, len(deployResp.WorkerTasks), len(deployResp.DeploymentRecords))
	}
	updateReq := req
	updateReq.Metadata = mergeMaps(req.Metadata, map[string]any{
		"rollback_plan_id":   rollbackPlan.ID,
		"rollback_plan_key":  rollbackPlan.PlanKey,
		"rollback_to_id":     previous.ID,
		"rollback_to_key":    previous.PlanKey,
		"rollback_to_status": previous.Status,
	})
	plan, err := store.UpdateReleasePlanStatus(ctx, planID, "rolled_back", updateReq)
	if err != nil {
		return ReleasePlanActionResponse{}, err
	}
	resp.Plan = plan
	s.insertAudit(ctx, "release_plan.rollback", "release_plan", plan.ID, "发布计划已回滚", releasePlanAuditMetadata(plan, map[string]any{
		"approved_by":       req.ApprovedBy,
		"rollback_plan_id":  rollbackPlan.ID,
		"rollback_plan_key": rollbackPlan.PlanKey,
		"rollback_to_id":    previous.ID,
		"rollback_to_key":   previous.PlanKey,
		"deployment_count":  len(resp.DeploymentRecords),
		"worker_task_count": len(resp.WorkerTasks),
	}))
	s.insertAudit(ctx, "release_plan.rollback_plan", "release_plan", rollbackPlan.ID, "生成回滚发布计划", releasePlanAuditMetadata(rollbackPlan, map[string]any{
		"rollback_from_plan_id":  current.ID,
		"rollback_from_plan_key": current.PlanKey,
		"rollback_to_plan_id":    previous.ID,
		"rollback_to_plan_key":   previous.PlanKey,
	}))
	return resp, nil
}

func (s *Service) CreateReleasePlanDeployment(ctx context.Context, planID string, req CreateReleasePlanDeploymentRequest) (ReleasePlanDeploymentResponse, error) {
	releaseStore, ok := s.store.(ReleasePlanStore)
	if !ok {
		return ReleasePlanDeploymentResponse{}, errReleasePlanStoreUnavailable
	}
	deploymentStore, ok := s.store.(DeploymentStore)
	if !ok {
		return ReleasePlanDeploymentResponse{}, errDeploymentStoreUnavailable
	}
	planID = strings.TrimSpace(planID)
	req.TargetID = strings.TrimSpace(req.TargetID)
	req.TargetKey = strings.TrimSpace(req.TargetKey)
	req.TriggeredBy = firstNonBlank(req.TriggeredBy, "admin")
	req.DeploymentURL = strings.TrimSpace(req.DeploymentURL)
	if planID == "" {
		return ReleasePlanDeploymentResponse{}, fmt.Errorf("plan_id is required")
	}
	if req.TargetID == "" && req.TargetKey == "" {
		return ReleasePlanDeploymentResponse{}, fmt.Errorf("target_id or target_key is required")
	}
	plan, err := releaseStore.ReleasePlan(ctx, planID)
	if err != nil {
		return ReleasePlanDeploymentResponse{}, err
	}
	if err := ensureReleasePlanDeploymentApproved(plan, req.DryRun); err != nil {
		return ReleasePlanDeploymentResponse{}, err
	}
	if len(plan.Artifacts) == 0 {
		return ReleasePlanDeploymentResponse{}, fmt.Errorf("release plan artifacts are required")
	}
	records := make([]DeploymentRecordAdmin, 0, len(plan.Artifacts))
	tasks := make([]WorkerTaskAdmin, 0, len(plan.Artifacts))
	pendingApproval := 0
	for _, artifact := range plan.Artifacts {
		metadata := mergeMaps(req.Metadata, map[string]any{
			"source":            "release_plan",
			"release_plan_id":   plan.ID,
			"release_plan_key":  plan.PlanKey,
			"release_unit_id":   plan.ReleaseUnitID,
			"release_unit_key":  plan.UnitKey,
			"environment_key":   plan.EnvironmentKey,
			"artifact_name":     artifact.ArtifactName,
			"artifact_type":     artifact.ArtifactType,
			"file_name":         artifact.FileName,
			"immutable_ref":     artifact.ImmutableRef,
			"rollout_percent":   plan.RolloutPercentage,
			"release_target":    plan.TargetType,
			"release_target_id": plan.TargetValue,
		})
		if _, ok := metadata["artifact_path"]; !ok {
			metadata["artifact_path"] = firstNonBlank(artifact.ImmutableRef, artifact.FileName, artifact.ArtifactName)
		}
		if _, ok := metadata["object_key"]; !ok {
			metadata["object_key"] = firstNonBlank(artifact.FileName, artifact.ArtifactName)
		}
		deployReq := CreateDeploymentRequest{
			ProjectKey:         plan.ProjectKey,
			TargetID:           req.TargetID,
			TargetKey:          req.TargetKey,
			RunID:              artifact.BuildRunID,
			AppBuildID:         artifact.AppBuildID,
			AppBuildArtifactID: artifact.AppBuildArtifactID,
			ProviderStatus:     "queued",
			DeploymentURL:      req.DeploymentURL,
			VersionName:        plan.VersionName,
			BuildNumber:        plan.BuildNumber,
			GitCommit:          plan.GitCommit,
			TriggeredBy:        req.TriggeredBy,
			DryRun:             req.DryRun,
			Metadata:           metadata,
		}
		record, err := deploymentStore.CreateDeploymentRecord(ctx, deployReq)
		if err != nil {
			return ReleasePlanDeploymentResponse{}, err
		}
		records = append(records, record)
		if record.ProviderStatus == "pending_approval" {
			pendingApproval++
			continue
		}
		if !req.DryRun {
			task, err := s.enqueueDeploymentWorkerTask(ctx, record, deployReq)
			if err != nil {
				return ReleasePlanDeploymentResponse{}, err
			}
			tasks = append(tasks, task)
		}
	}
	s.insertAudit(ctx, "release_plan.deploy", "release_plan", plan.ID, "创建发布计划部署", releasePlanAuditMetadata(plan, map[string]any{
		"target_id":               req.TargetID,
		"target_key":              req.TargetKey,
		"dry_run":                 req.DryRun,
		"triggered_by":            req.TriggeredBy,
		"deployment_count":        len(records),
		"deployment_record_ids":   deploymentRecordIDs(records),
		"worker_task_count":       len(tasks),
		"worker_task_ids":         workerTaskIDs(tasks),
		"pending_approval_count":  pendingApproval,
		"release_plan_artifacts":  len(plan.Artifacts),
		"deployment_records_only": req.DryRun || pendingApproval > 0,
	}))
	return ReleasePlanDeploymentResponse{
		OK:                true,
		Plan:              plan,
		DeploymentRecords: records,
		WorkerTasks:       tasks,
		MessageZh:         releasePlanDeploymentMessage(req.DryRun, pendingApproval),
	}, nil
}

func releasePlanDeploymentMessage(dryRun bool, pendingApproval int) string {
	if dryRun {
		return "发布计划部署记录已创建"
	}
	if pendingApproval > 0 {
		return fmt.Sprintf("发布计划部署记录已创建，%d 条生产部署等待审批", pendingApproval)
	}
	return "发布计划部署记录已创建并投递 Worker"
}

func ensureReleasePlanDeploymentApproved(plan ReleasePlanAdmin, dryRun bool) error {
	if dryRun || !releasePlanRequiresApproval(plan) {
		return nil
	}
	switch plan.Status {
	case "released", "rolling_out":
		return nil
	default:
		return fmt.Errorf("release plan must be approved before non-dry-run deployment")
	}
}

func rollbackReleasePlanRequest(current, previous ReleasePlanAdmin, req ReleasePlanActionRequest) CreateReleasePlanRequest {
	status := "draft"
	if releasePlanRollbackDeploymentRequested(req) && !req.DryRun {
		status = "queued"
	}
	artifacts := make([]ReleasePlanArtifactRequest, 0, len(previous.Artifacts))
	for _, artifact := range previous.Artifacts {
		metadata := rawJSONMap(artifact.Metadata)
		metadata = mergeMaps(metadata, map[string]any{
			"source":                  "release_plan_rollback",
			"rollback_from_plan_id":   current.ID,
			"rollback_from_plan_key":  current.PlanKey,
			"rollback_target_plan_id": previous.ID,
			"rollback_target_key":     previous.PlanKey,
		})
		artifacts = append(artifacts, ReleasePlanArtifactRequest{
			BuildRunID:         artifact.BuildRunID,
			AppBuildID:         artifact.AppBuildID,
			AppBuildArtifactID: artifact.AppBuildArtifactID,
			ArtifactName:       artifact.ArtifactName,
			ArtifactType:       artifact.ArtifactType,
			FileName:           artifact.FileName,
			ImmutableRef:       artifact.ImmutableRef,
			Metadata:           metadata,
		})
	}
	targetType := firstNonBlank(current.TargetType, previous.TargetType, "all")
	targetValue := current.TargetValue
	if targetType == "all" {
		targetValue = ""
	} else if targetValue == "" {
		targetValue = previous.TargetValue
	}
	return CreateReleasePlanRequest{
		ProjectKey:        current.ProjectKey,
		UnitKey:           current.UnitKey,
		EnvironmentKey:    current.EnvironmentKey,
		PlanKey:           rollbackReleasePlanKey(current, previous),
		Title:             fmt.Sprintf("回滚 %s 到 %s", firstNonBlank(current.Title, current.PlanKey), firstNonBlank(previous.VersionName, previous.PlanKey)),
		Description:       fmt.Sprintf("由发布计划 %s 回滚到 %s", current.PlanKey, previous.PlanKey),
		VersionName:       previous.VersionName,
		BuildNumber:       previous.BuildNumber,
		GitCommit:         previous.GitCommit,
		Channel:           firstNonBlank(previous.Channel, current.Channel),
		Status:            status,
		RolloutPercentage: normalizeRollout(previous.RolloutPercentage),
		TargetType:        targetType,
		TargetValue:       targetValue,
		ApprovedBy:        req.ApprovedBy,
		CreatedBy:         firstNonBlank(req.ApprovedBy, "rollback"),
		Artifacts:         artifacts,
		Metadata: mergeMaps(req.Metadata, map[string]any{
			"source":                  "release_plan_rollback",
			"rollback_from_plan_id":   current.ID,
			"rollback_from_plan_key":  current.PlanKey,
			"rollback_target_plan_id": previous.ID,
			"rollback_target_key":     previous.PlanKey,
			"rollback_dry_run":        req.DryRun,
		}),
	}
}

func rollbackReleasePlanKey(current, previous ReleasePlanAdmin) string {
	key := fmt.Sprintf("rollback-%s-to-%s", safeFilePart(firstNonBlank(current.PlanKey, current.ID)), safeFilePart(firstNonBlank(previous.PlanKey, previous.ID)))
	if len(key) > 128 {
		return strings.TrimRight(key[:128], "-")
	}
	return key
}

func releasePlanRollbackDeploymentRequested(req ReleasePlanActionRequest) bool {
	return strings.TrimSpace(req.TargetID) != "" || strings.TrimSpace(req.TargetKey) != ""
}

func releasePlanRollbackMessage(dryRun bool, workerTasks, deployments int) string {
	if dryRun {
		return fmt.Sprintf("发布计划已回滚，已生成回滚计划和 %d 条 dry-run 部署记录", deployments)
	}
	if workerTasks > 0 {
		return fmt.Sprintf("发布计划已回滚，已生成回滚计划并投递 %d 个 Worker 任务", workerTasks)
	}
	return fmt.Sprintf("发布计划已回滚，已生成回滚计划和 %d 条部署记录", deployments)
}

func releasePlanRequiresApproval(plan ReleasePlanAdmin) bool {
	if plan.EnvironmentRequiresApproval {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(plan.EnvironmentKey)) {
	case "prod", "production":
		return true
	default:
		return false
	}
}

func releasePlanAuditMetadata(plan ReleasePlanAdmin, extra map[string]any) map[string]any {
	metadata := map[string]any{
		"project_key":                   plan.ProjectKey,
		"project_id":                    plan.ProjectID,
		"release_unit_id":               plan.ReleaseUnitID,
		"unit_key":                      plan.UnitKey,
		"unit_type":                     plan.UnitType,
		"environment_id":                plan.EnvironmentID,
		"environment_key":               plan.EnvironmentKey,
		"environment_requires_approval": plan.EnvironmentRequiresApproval,
		"plan_key":                      plan.PlanKey,
		"version_name":                  plan.VersionName,
		"build_number":                  plan.BuildNumber,
		"git_commit":                    plan.GitCommit,
		"channel":                       plan.Channel,
		"status":                        plan.Status,
		"rollout_percentage":            plan.RolloutPercentage,
		"target_type":                   plan.TargetType,
		"target_value":                  plan.TargetValue,
		"created_by":                    plan.CreatedBy,
	}
	for key, value := range extra {
		metadata[key] = value
	}
	return metadata
}

func deploymentRecordIDs(records []DeploymentRecordAdmin) []string {
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ID)
	}
	return ids
}

func workerTaskIDs(tasks []WorkerTaskAdmin) []string {
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	return ids
}

func normalizeReleaseUnitType(unitType string) string {
	unitType = strings.ToLower(strings.TrimSpace(unitType))
	switch unitType {
	case "android", "web", "docs", "worker", "server", "docker", "config":
		return unitType
	default:
		return unitType
	}
}

func normalizeReleasePlanStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "draft", "scheduled", "queued", "pending_approval", "released", "rolling_out", "paused", "recalled", "rolled_back", "archived":
		return status
	default:
		return "draft"
	}
}

func normalizeReleasePlanArtifacts(items []ReleasePlanArtifactRequest) []ReleasePlanArtifactRequest {
	normalized := make([]ReleasePlanArtifactRequest, 0, len(items))
	for _, item := range items {
		item.ArtifactName = strings.TrimSpace(item.ArtifactName)
		item.ArtifactType = strings.TrimSpace(item.ArtifactType)
		item.FileName = strings.TrimSpace(item.FileName)
		item.ImmutableRef = strings.TrimSpace(item.ImmutableRef)
		if item.ArtifactName == "" {
			item.ArtifactName = firstNonBlank(item.FileName, item.ArtifactType, "artifact")
		}
		if item.ArtifactType == "" {
			item.ArtifactType = "artifact"
		}
		normalized = append(normalized, item)
	}
	return normalized
}

func defaultReleasePlanKey(req CreateReleasePlanRequest) string {
	version := strings.NewReplacer("/", "-", " ", "-").Replace(firstNonBlank(req.VersionName, "manual"))
	return fmt.Sprintf("%s-%s-%s-%s", req.UnitKey, req.EnvironmentKey, version, time.Now().UTC().Format("20060102150405"))
}
