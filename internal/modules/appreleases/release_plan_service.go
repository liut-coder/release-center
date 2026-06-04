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
	return ReleasePlanActionResponse{OK: true, Plan: plan, MessageZh: "发布计划已创建"}, nil
}

func (s *Service) ReleasePlanAction(ctx context.Context, planID, action string, req ReleasePlanActionRequest) (ReleasePlanActionResponse, error) {
	store, ok := s.store.(ReleasePlanStore)
	if !ok {
		return ReleasePlanActionResponse{}, errReleasePlanStoreUnavailable
	}
	planID = strings.TrimSpace(planID)
	req.ApprovedBy = strings.TrimSpace(req.ApprovedBy)
	status := ""
	message := "发布计划状态已更新"
	switch action {
	case "publish":
		status = "released"
		message = "发布计划已发布"
	case "pause":
		status = "paused"
		message = "发布计划已暂停"
	case "rollback":
		status = "rolled_back"
		message = "发布计划已回滚"
	default:
		return ReleasePlanActionResponse{}, fmt.Errorf("unsupported release plan action: %s", action)
	}
	plan, err := store.UpdateReleasePlanStatus(ctx, planID, status, req)
	if err != nil {
		return ReleasePlanActionResponse{}, err
	}
	return ReleasePlanActionResponse{OK: true, Plan: plan, MessageZh: message}, nil
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
	if len(plan.Artifacts) == 0 {
		return ReleasePlanDeploymentResponse{}, fmt.Errorf("release plan artifacts are required")
	}
	records := make([]DeploymentRecordAdmin, 0, len(plan.Artifacts))
	tasks := make([]WorkerTaskAdmin, 0, len(plan.Artifacts))
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
		if !req.DryRun {
			task, err := s.enqueueDeploymentWorkerTask(ctx, record, deployReq)
			if err != nil {
				return ReleasePlanDeploymentResponse{}, err
			}
			tasks = append(tasks, task)
		}
	}
	return ReleasePlanDeploymentResponse{
		OK:                true,
		Plan:              plan,
		DeploymentRecords: records,
		WorkerTasks:       tasks,
		MessageZh:         releasePlanDeploymentMessage(req.DryRun),
	}, nil
}

func releasePlanDeploymentMessage(dryRun bool) string {
	if dryRun {
		return "发布计划部署记录已创建"
	}
	return "发布计划部署记录已创建并投递 Worker"
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
	case "draft", "scheduled", "queued", "released", "rolling_out", "paused", "recalled", "rolled_back", "archived":
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
