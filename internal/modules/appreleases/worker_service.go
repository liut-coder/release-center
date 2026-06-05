package appreleases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var errWorkerStoreUnavailable = errors.New("worker store unavailable")

type WorkerStore interface {
	WorkerOverview(ctx context.Context) (WorkerOverviewResponse, error)
	CreateWorkerTask(ctx context.Context, req CreateWorkerTaskRequest) (WorkerTaskAdmin, error)
	RegisterWorker(ctx context.Context, req WorkerRegisterRequest) (BuildWorkerAdmin, error)
	SaveWorkerHeartbeat(ctx context.Context, req WorkerHeartbeatRequest) (BuildWorkerAdmin, error)
	NextWorkerTask(ctx context.Context, req WorkerTaskNextRequest) (*WorkerTaskAdmin, error)
	AppendWorkerTaskLogs(ctx context.Context, taskID string, req WorkerTaskLogsRequest) (WorkerTaskAdmin, error)
	SaveWorkerTaskArtifacts(ctx context.Context, taskID string, req WorkerTaskArtifactsRequest) (WorkerTaskAdmin, error)
	CompleteWorkerTask(ctx context.Context, taskID string, req WorkerTaskCompleteRequest) (WorkerTaskAdmin, error)
	FailWorkerTask(ctx context.Context, taskID string, req WorkerTaskFailRequest) (WorkerTaskAdmin, error)
}

type WorkerArtifactMirrorStore interface {
	ReplaceBuildCenterRunArtifacts(ctx context.Context, runID string, artifacts []BuildCenterRunArtifact) error
}

func (s *Service) WorkerOverview(ctx context.Context) (WorkerOverviewResponse, error) {
	store, ok := s.store.(WorkerStore)
	if !ok {
		return WorkerOverviewResponse{Workers: []BuildWorkerAdmin{}, Tasks: []WorkerTaskAdmin{}, MessageZh: "Worker 未连接数据库"}, nil
	}
	overview, err := store.WorkerOverview(ctx)
	if err != nil {
		return WorkerOverviewResponse{}, err
	}
	overview.MessageZh = "Worker 接入已读取"
	return overview, nil
}

func (s *Service) CreateWorkerTask(ctx context.Context, req CreateWorkerTaskRequest) (WorkerActionResponse, error) {
	store, ok := s.store.(WorkerStore)
	if !ok {
		return WorkerActionResponse{}, errWorkerStoreUnavailable
	}
	req.ProjectKey = strings.TrimSpace(req.ProjectKey)
	req.BuildProfileID = strings.TrimSpace(req.BuildProfileID)
	req.BuildRunID = strings.TrimSpace(req.BuildRunID)
	req.TaskType = strings.ToLower(strings.TrimSpace(firstNonBlank(req.TaskType, "build")))
	req.Action = strings.ToLower(strings.TrimSpace(firstNonBlank(req.Action, "all")))
	req.RequiredLabels = normalizeWorkerLabels(req.RequiredLabels)
	if req.Priority < 0 {
		req.Priority = 0
	}
	if req.TaskType == "" || req.Action == "" {
		return WorkerActionResponse{}, fmt.Errorf("task_type and action are required")
	}
	task, err := store.CreateWorkerTask(ctx, req)
	if err != nil {
		return WorkerActionResponse{}, err
	}
	return WorkerActionResponse{OK: true, Task: &task, MessageZh: "Worker 任务已创建"}, nil
}

func (s *Service) RegisterWorker(ctx context.Context, req WorkerRegisterRequest) (WorkerActionResponse, error) {
	store, ok := s.store.(WorkerStore)
	if !ok {
		return WorkerActionResponse{}, errWorkerStoreUnavailable
	}
	req.WorkerKey = strings.TrimSpace(req.WorkerKey)
	req.Name = strings.TrimSpace(req.Name)
	req.EndpointURL = strings.TrimSpace(req.EndpointURL)
	req.Labels = normalizeWorkerLabels(req.Labels)
	if req.WorkerKey == "" {
		return WorkerActionResponse{}, fmt.Errorf("worker_key is required")
	}
	if req.Capacity <= 0 {
		req.Capacity = 1
	}
	worker, err := store.RegisterWorker(ctx, req)
	if err != nil {
		return WorkerActionResponse{}, err
	}
	return WorkerActionResponse{OK: true, Worker: &worker, MessageZh: "Worker 已注册"}, nil
}

func (s *Service) SaveWorkerHeartbeat(ctx context.Context, req WorkerHeartbeatRequest) (WorkerActionResponse, error) {
	store, ok := s.store.(WorkerStore)
	if !ok {
		return WorkerActionResponse{}, errWorkerStoreUnavailable
	}
	req.WorkerKey = strings.TrimSpace(req.WorkerKey)
	req.Status = normalizeWorkerStatus(req.Status)
	req.Labels = normalizeWorkerLabels(req.Labels)
	if req.WorkerKey == "" {
		return WorkerActionResponse{}, fmt.Errorf("worker_key is required")
	}
	if req.Capacity <= 0 {
		req.Capacity = 1
	}
	if req.RunningTasks < 0 {
		req.RunningTasks = 0
	}
	worker, err := store.SaveWorkerHeartbeat(ctx, req)
	if err != nil {
		return WorkerActionResponse{}, err
	}
	return WorkerActionResponse{OK: true, Worker: &worker, MessageZh: "Worker 心跳已接收"}, nil
}

func (s *Service) NextWorkerTask(ctx context.Context, req WorkerTaskNextRequest) (WorkerTaskNextResponse, error) {
	store, ok := s.store.(WorkerStore)
	if !ok {
		return WorkerTaskNextResponse{}, errWorkerStoreUnavailable
	}
	req.WorkerKey = strings.TrimSpace(req.WorkerKey)
	if req.WorkerKey == "" {
		return WorkerTaskNextResponse{}, fmt.Errorf("worker_key is required")
	}
	task, err := store.NextWorkerTask(ctx, req)
	if err != nil {
		return WorkerTaskNextResponse{}, err
	}
	if task == nil {
		return WorkerTaskNextResponse{MessageZh: "暂无可领取任务"}, nil
	}
	return WorkerTaskNextResponse{Task: task, MessageZh: "Worker 任务已领取"}, nil
}

func (s *Service) AppendWorkerTaskLogs(ctx context.Context, taskID string, req WorkerTaskLogsRequest) (WorkerActionResponse, error) {
	store, ok := s.store.(WorkerStore)
	if !ok {
		return WorkerActionResponse{}, errWorkerStoreUnavailable
	}
	taskID = strings.TrimSpace(taskID)
	req.WorkerKey = strings.TrimSpace(req.WorkerKey)
	req.LeaseToken = strings.TrimSpace(req.LeaseToken)
	req.Lines = normalizeWorkerLogLines(req.Lines)
	if taskID == "" || req.WorkerKey == "" || req.LeaseToken == "" {
		return WorkerActionResponse{}, fmt.Errorf("task_id, worker_key and lease_token are required")
	}
	task, err := store.AppendWorkerTaskLogs(ctx, taskID, req)
	if err != nil {
		return WorkerActionResponse{}, err
	}
	return WorkerActionResponse{OK: true, Task: &task, MessageZh: "Worker 日志已接收"}, nil
}

func (s *Service) SaveWorkerTaskArtifacts(ctx context.Context, taskID string, req WorkerTaskArtifactsRequest) (WorkerActionResponse, error) {
	store, ok := s.store.(WorkerStore)
	if !ok {
		return WorkerActionResponse{}, errWorkerStoreUnavailable
	}
	taskID = strings.TrimSpace(taskID)
	req.WorkerKey = strings.TrimSpace(req.WorkerKey)
	req.LeaseToken = strings.TrimSpace(req.LeaseToken)
	if taskID == "" || req.WorkerKey == "" || req.LeaseToken == "" {
		return WorkerActionResponse{}, fmt.Errorf("task_id, worker_key and lease_token are required")
	}
	task, err := store.SaveWorkerTaskArtifacts(ctx, taskID, req)
	if err != nil {
		return WorkerActionResponse{}, err
	}
	if err := s.mirrorWorkerTaskArtifacts(ctx, task, req.Artifacts); err != nil {
		return WorkerActionResponse{}, err
	}
	return WorkerActionResponse{OK: true, Task: &task, MessageZh: "Worker 产物已接收"}, nil
}

func (s *Service) CompleteWorkerTask(ctx context.Context, taskID string, req WorkerTaskCompleteRequest) (WorkerActionResponse, error) {
	store, ok := s.store.(WorkerStore)
	if !ok {
		return WorkerActionResponse{}, errWorkerStoreUnavailable
	}
	taskID = strings.TrimSpace(taskID)
	req.WorkerKey = strings.TrimSpace(req.WorkerKey)
	req.LeaseToken = strings.TrimSpace(req.LeaseToken)
	if taskID == "" || req.WorkerKey == "" || req.LeaseToken == "" {
		return WorkerActionResponse{}, fmt.Errorf("task_id, worker_key and lease_token are required")
	}
	task, err := store.CompleteWorkerTask(ctx, taskID, req)
	if err != nil {
		return WorkerActionResponse{}, err
	}
	if err := s.mirrorWorkerTaskArtifacts(ctx, task, req.Artifacts); err != nil {
		return WorkerActionResponse{}, err
	}
	return WorkerActionResponse{OK: true, Task: &task, MessageZh: "Worker 任务已完成"}, nil
}

func (s *Service) FailWorkerTask(ctx context.Context, taskID string, req WorkerTaskFailRequest) (WorkerActionResponse, error) {
	store, ok := s.store.(WorkerStore)
	if !ok {
		return WorkerActionResponse{}, errWorkerStoreUnavailable
	}
	taskID = strings.TrimSpace(taskID)
	req.WorkerKey = strings.TrimSpace(req.WorkerKey)
	req.LeaseToken = strings.TrimSpace(req.LeaseToken)
	req.ErrorMessage = strings.TrimSpace(req.ErrorMessage)
	if taskID == "" || req.WorkerKey == "" || req.LeaseToken == "" {
		return WorkerActionResponse{}, fmt.Errorf("task_id, worker_key and lease_token are required")
	}
	if req.ErrorMessage == "" {
		req.ErrorMessage = "worker task failed"
	}
	task, err := store.FailWorkerTask(ctx, taskID, req)
	if err != nil {
		return WorkerActionResponse{}, err
	}
	return WorkerActionResponse{OK: true, Task: &task, MessageZh: "Worker 任务失败已记录"}, nil
}

func normalizeWorkerLabels(labels []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(labels))
	for _, label := range labels {
		label = strings.ToLower(strings.TrimSpace(label))
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		normalized = append(normalized, label)
	}
	return normalized
}

func normalizeWorkerStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "online", "busy", "draining", "offline":
		return status
	default:
		return "online"
	}
}

func normalizeWorkerLogLines(lines []string) []string {
	if len(lines) > 200 {
		lines = lines[len(lines)-200:]
	}
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r\n")
	}
	return lines
}

func (s *Service) mirrorWorkerTaskArtifacts(ctx context.Context, task WorkerTaskAdmin, artifacts []WorkerTaskArtifact) error {
	if task.BuildRunID == "" || len(artifacts) == 0 {
		return nil
	}
	store, ok := s.store.(WorkerArtifactMirrorStore)
	if !ok {
		return nil
	}
	items := make([]BuildCenterRunArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		metadata := mergeMaps(artifact.Metadata, map[string]any{
			"source":         "worker_task_artifact",
			"worker_task_id": task.ID,
			"worker_id":      task.WorkerID,
			"task_type":      task.TaskType,
			"action":         task.Action,
		})
		items = append(items, BuildCenterRunArtifact{
			RunID:        task.BuildRunID,
			Name:         firstNonBlank(strings.TrimSpace(artifact.Name), strings.TrimSpace(artifact.FileName), "artifact"),
			ArtifactType: firstNonBlank(strings.TrimSpace(artifact.ArtifactType), "artifact"),
			FileName:     strings.TrimSpace(artifact.FileName),
			LocalPath:    strings.TrimSpace(artifact.LocalPath),
			SizeBytes:    artifact.SizeBytes,
			SHA256:       strings.TrimSpace(artifact.SHA256),
			UploadStatus: workerArtifactUploadStatus(artifact),
			DownloadURL:  strings.TrimSpace(artifact.DownloadURL),
			Metadata:     json.RawMessage(jsonb(metadata)),
		})
	}
	return store.ReplaceBuildCenterRunArtifacts(ctx, task.BuildRunID, items)
}

func workerArtifactUploadStatus(artifact WorkerTaskArtifact) string {
	if strings.TrimSpace(artifact.DownloadURL) != "" {
		return "uploaded"
	}
	if strings.TrimSpace(artifact.LocalPath) != "" {
		return "local"
	}
	return "recorded"
}
