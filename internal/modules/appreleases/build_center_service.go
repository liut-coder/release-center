package appreleases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var errBuildCenterStoreUnavailable = errors.New("build center store unavailable")

type BuildCenterStore interface {
	BuildCenterOverview(ctx context.Context) (BuildCenterOverview, error)
	BuildCenterProject(ctx context.Context, projectKey string) (BuildCenterProject, error)
	BuildCenterRun(ctx context.Context, runID string) (BuildCenterRunAdmin, error)
	DeploymentTargets(ctx context.Context) ([]DeploymentTargetAdmin, error)
}

type BuildCenterExecutionStore interface {
	BuildCenterStore
	CreateBuildCenterRun(ctx context.Context, projectKey string, req BuildCenterRunRequest) (BuildCenterRunAdmin, BuildProfileAdmin, error)
	MarkBuildCenterRunStarted(ctx context.Context, runID, logDir string) error
	CompleteBuildCenterRun(ctx context.Context, runID string, patch BuildCenterRunPatch) (BuildCenterRunAdmin, error)
	ReplaceBuildCenterRunArtifacts(ctx context.Context, runID string, artifacts []BuildCenterRunArtifact) error
}

type BuildCenterWorkerTaskStore interface {
	CreateWorkerTask(ctx context.Context, req CreateWorkerTaskRequest) (WorkerTaskAdmin, error)
}

type BuildCenterWebhookStore interface {
	MatchWebhookBuildRoutes(ctx context.Context, event WebhookEventRequest) ([]WebhookBuildRoute, error)
}

type BuildCenterConfigStore interface {
	BuildCenterStore
	CreateBuildCenterProject(ctx context.Context, req BuildCenterProjectRequest) (BuildCenterProject, error)
	UpsertCodeRepository(ctx context.Context, projectKey string, req CodeRepositoryRequest) (CodeRepositoryAdmin, error)
	UpsertBuildProfile(ctx context.Context, projectKey string, req BuildProfileRequest) (BuildProfileAdmin, error)
	UpsertWebhookRoute(ctx context.Context, projectKey string, req WebhookRouteRequest) (WebhookRouteAdmin, error)
}

type buildctlStatus struct {
	BuildID     string                   `json:"build_id"`
	Project     string                   `json:"project"`
	GitCommit   string                   `json:"git_commit"`
	VersionName string                   `json:"version_name"`
	VersionCode int                      `json:"version_code"`
	BuildNumber int                      `json:"build_number"`
	Status      string                   `json:"status"`
	ArtifactDir string                   `json:"artifact_dir"`
	LogDir      string                   `json:"log_dir"`
	Artifacts   []buildctlStatusArtifact `json:"artifacts"`
}

type buildctlStatusArtifact struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	FileName  string `json:"file_name"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

const buildCenterLogTailLimit = 200

func (s *Service) BuildCenterOverview(ctx context.Context) (BuildCenterOverview, error) {
	store, ok := s.store.(BuildCenterStore)
	if !ok {
		return BuildCenterOverview{MessageZh: "构建中心未连接数据库"}, nil
	}
	resp, err := store.BuildCenterOverview(ctx)
	if err != nil {
		return BuildCenterOverview{}, err
	}
	resp.MessageZh = "构建中心已读取"
	return resp, nil
}

func (s *Service) BuildCenterProject(ctx context.Context, projectKey string) (BuildCenterProject, error) {
	store, ok := s.store.(BuildCenterStore)
	if !ok {
		return BuildCenterProject{}, errBuildCenterStoreUnavailable
	}
	return store.BuildCenterProject(ctx, projectKey)
}

func (s *Service) BuildCenterRun(ctx context.Context, runID string) (BuildCenterRunResponse, error) {
	store, ok := s.store.(BuildCenterStore)
	if !ok {
		return BuildCenterRunResponse{}, errBuildCenterStoreUnavailable
	}
	run, err := store.BuildCenterRun(ctx, runID)
	if err != nil {
		return BuildCenterRunResponse{}, err
	}
	return BuildCenterRunResponse{Run: run}, nil
}

func (s *Service) BuildCenterRunLogs(ctx context.Context, runID string) (BuildCenterRunLogsResponse, error) {
	store, ok := s.store.(BuildCenterStore)
	if !ok {
		return BuildCenterRunLogsResponse{}, errBuildCenterStoreUnavailable
	}
	run, err := store.BuildCenterRun(ctx, runID)
	if err != nil {
		return BuildCenterRunLogsResponse{}, err
	}
	if strings.TrimSpace(run.LogDir) == "" {
		return BuildCenterRunLogsResponse{
			RunID:     run.ID,
			Lines:     []string{},
			MessageZh: "构建日志尚未生成",
		}, nil
	}
	logPath := filepath.Join(run.LogDir, "buildctl.log")
	lines, truncated, err := tailLogLines(logPath, buildCenterLogTailLimit)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return BuildCenterRunLogsResponse{
				RunID:     run.ID,
				LogPath:   logPath,
				Lines:     []string{},
				MessageZh: "构建日志文件不存在",
			}, nil
		}
		return BuildCenterRunLogsResponse{}, err
	}
	return BuildCenterRunLogsResponse{
		RunID:     run.ID,
		LogPath:   logPath,
		Lines:     lines,
		Truncated: truncated,
		MessageZh: "构建日志已读取",
	}, nil
}

func (s *Service) DeploymentTargets(ctx context.Context) ([]DeploymentTargetAdmin, error) {
	store, ok := s.store.(BuildCenterStore)
	if !ok {
		return []DeploymentTargetAdmin{}, nil
	}
	return store.DeploymentTargets(ctx)
}

func (s *Service) CreateBuildCenterProject(ctx context.Context, req BuildCenterProjectRequest) (BuildCenterProjectActionResponse, error) {
	store, ok := s.store.(BuildCenterConfigStore)
	if !ok {
		return BuildCenterProjectActionResponse{}, errBuildCenterStoreUnavailable
	}
	req.ProjectKey = strings.TrimSpace(req.ProjectKey)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.OwnerAccount = strings.TrimSpace(req.OwnerAccount)
	req.LifecycleStatus = normalizeProjectLifecycleStatus(req.LifecycleStatus)
	req.DefaultChannel = normalizeChannel(req.DefaultChannel, s.cfg.Channel)
	if req.ProjectKey == "" || req.Name == "" {
		return BuildCenterProjectActionResponse{}, fmt.Errorf("project_key and name are required")
	}
	project, err := store.CreateBuildCenterProject(ctx, req)
	if err != nil {
		return BuildCenterProjectActionResponse{}, err
	}
	s.insertAudit(ctx, "build_center.project_save", "build_center_project", project.ID, "保存构建项目", map[string]any{
		"project_key":      project.ProjectKey,
		"default_channel":  project.DefaultChannel,
		"lifecycle_status": project.LifecycleStatus,
		"owner_account":    project.OwnerAccount,
	})
	return BuildCenterProjectActionResponse{OK: true, Project: project, MessageZh: "构建项目已保存"}, nil
}

func (s *Service) UpsertCodeRepository(ctx context.Context, projectKey string, req CodeRepositoryRequest) (CodeRepositoryActionResponse, error) {
	store, ok := s.store.(BuildCenterConfigStore)
	if !ok {
		return CodeRepositoryActionResponse{}, errBuildCenterStoreUnavailable
	}
	projectKey = strings.TrimSpace(projectKey)
	req.Provider = normalizeCodeProvider(req.Provider)
	req.RepoURL = strings.TrimSpace(req.RepoURL)
	req.RepoFullName = strings.TrimSpace(req.RepoFullName)
	req.DefaultRef = firstNonBlank(req.DefaultRef, "main")
	req.CredentialRef = strings.TrimSpace(req.CredentialRef)
	req.WebhookSecretRef = strings.TrimSpace(req.WebhookSecretRef)
	if projectKey == "" || req.Provider == "" || req.RepoURL == "" {
		return CodeRepositoryActionResponse{}, fmt.Errorf("project_key, provider and repo_url are required")
	}
	repository, err := store.UpsertCodeRepository(ctx, projectKey, req)
	if err != nil {
		return CodeRepositoryActionResponse{}, err
	}
	s.insertAudit(ctx, "build_center.repository_save", "code_repository", repository.ID, "保存代码仓库", map[string]any{
		"project_key":     projectKey,
		"project_id":      repository.ProjectID,
		"provider":        repository.Provider,
		"repo_full_name":  repository.RepoFullName,
		"repo_url":        repository.RepoURL,
		"default_ref":     repository.DefaultRef,
		"webhook_enabled": repository.WebhookEnabled,
		"trigger_on_push": repository.TriggerOnPush,
		"trigger_on_tag":  repository.TriggerOnTag,
	})
	return CodeRepositoryActionResponse{OK: true, Repository: repository, MessageZh: "代码仓库已保存"}, nil
}

func (s *Service) UpsertBuildProfile(ctx context.Context, projectKey string, req BuildProfileRequest) (BuildProfileActionResponse, error) {
	store, ok := s.store.(BuildCenterConfigStore)
	if !ok {
		return BuildProfileActionResponse{}, errBuildCenterStoreUnavailable
	}
	projectKey = strings.TrimSpace(projectKey)
	req.AppID = strings.TrimSpace(req.AppID)
	req.ProfileKey = strings.TrimSpace(req.ProfileKey)
	req.Name = strings.TrimSpace(req.Name)
	req.BuildCenterProject = strings.TrimSpace(req.BuildCenterProject)
	req.StackType = firstNonBlank(req.StackType, "generic")
	req.BuildType = firstNonBlank(req.BuildType, "release")
	req.ConfigPath = strings.TrimSpace(req.ConfigPath)
	req.SourceWorkdir = strings.TrimSpace(req.SourceWorkdir)
	req.DefaultRef = firstNonBlank(req.DefaultRef, "main")
	req.DefaultVersionName = strings.TrimSpace(req.DefaultVersionName)
	req.DefaultChannel = normalizeChannel(req.DefaultChannel, s.cfg.Channel)
	req.BuildAction = normalizeBuildCenterAction(req.BuildAction)
	req.Commands = normalizeRawMessage(req.Commands, "{}")
	req.ArtifactRules = normalizeRawMessage(req.ArtifactRules, "[]")
	if projectKey == "" || req.ProfileKey == "" || req.Name == "" || req.BuildCenterProject == "" {
		return BuildProfileActionResponse{}, fmt.Errorf("project_key, profile_key, name and build_center_project are required")
	}
	if !isAllowedBuildCenterAction(req.BuildAction) {
		return BuildProfileActionResponse{}, fmt.Errorf("unsupported build action: %s", req.BuildAction)
	}
	profile, err := store.UpsertBuildProfile(ctx, projectKey, req)
	if err != nil {
		return BuildProfileActionResponse{}, err
	}
	s.insertAudit(ctx, "build_center.profile_save", "build_profile", profile.ID, "保存构建配置", map[string]any{
		"project_key":          projectKey,
		"project_id":           profile.ProjectID,
		"app_id":               profile.AppID,
		"profile_key":          profile.ProfileKey,
		"build_center_project": profile.BuildCenterProject,
		"stack_type":           profile.StackType,
		"build_type":           profile.BuildType,
		"build_action":         profile.BuildAction,
		"enabled":              profile.Enabled,
	})
	return BuildProfileActionResponse{OK: true, BuildProfile: profile, MessageZh: "构建配置已保存"}, nil
}

func (s *Service) UpsertWebhookRoute(ctx context.Context, projectKey string, req WebhookRouteRequest) (WebhookRouteActionResponse, error) {
	store, ok := s.store.(BuildCenterConfigStore)
	if !ok {
		return WebhookRouteActionResponse{}, errBuildCenterStoreUnavailable
	}
	projectKey = strings.TrimSpace(projectKey)
	req.RepositoryID = strings.TrimSpace(req.RepositoryID)
	req.ProfileKey = strings.TrimSpace(req.ProfileKey)
	req.EventType = firstNonBlank(req.EventType, "push")
	req.RefPattern = firstNonBlank(req.RefPattern, "*")
	req.Action = normalizeBuildCenterAction(req.Action)
	if projectKey == "" || req.RepositoryID == "" || req.EventType == "" {
		return WebhookRouteActionResponse{}, fmt.Errorf("project_key, repository_id and event_type are required")
	}
	if !isAllowedBuildCenterAction(req.Action) {
		return WebhookRouteActionResponse{}, fmt.Errorf("unsupported build action: %s", req.Action)
	}
	route, err := store.UpsertWebhookRoute(ctx, projectKey, req)
	if err != nil {
		return WebhookRouteActionResponse{}, err
	}
	s.insertAudit(ctx, "build_center.webhook_route_save", "webhook_route", route.ID, "保存 Webhook 路由", map[string]any{
		"project_key":      projectKey,
		"project_id":       route.ProjectID,
		"repository_id":    route.RepositoryID,
		"build_profile_id": route.BuildProfileID,
		"profile_key":      route.ProfileKey,
		"event_type":       route.EventType,
		"ref_pattern":      route.RefPattern,
		"action":           route.Action,
		"enabled":          route.Enabled,
	})
	return WebhookRouteActionResponse{OK: true, WebhookRoute: route, MessageZh: "Webhook 路由已保存"}, nil
}

func (s *Service) DryRunWebhookRoute(ctx context.Context, req WebhookRouteDryRunRequest) (WebhookRouteDryRunResponse, error) {
	routeStore, ok := s.store.(BuildCenterWebhookStore)
	if !ok {
		return WebhookRouteDryRunResponse{}, errBuildCenterStoreUnavailable
	}
	event := WebhookEventRequest{
		Provider:   normalizeCodeProvider(req.Provider),
		EventType:  firstNonBlank(strings.TrimSpace(req.EventType), "push"),
		Repository: strings.TrimSpace(req.Repository),
		Ref:        normalizeWebhookDryRunRef(req.Ref, req.EventType),
		CommitSHA:  strings.TrimSpace(req.CommitSHA),
		Sender:     firstNonBlank(req.Sender, "admin-web"),
	}
	if event.Provider == "" || event.Repository == "" || event.Ref == "" {
		return WebhookRouteDryRunResponse{}, fmt.Errorf("provider, repository and ref are required")
	}
	routes, err := routeStore.MatchWebhookBuildRoutes(ctx, event)
	if err != nil {
		return WebhookRouteDryRunResponse{}, err
	}
	matches := make([]WebhookRouteDryRunMatch, 0, len(routes))
	matchedCount := 0
	for _, route := range routes {
		eventOK := webhookRouteAllowsEvent(route, event)
		refOK := webhookRefMatches(route.RefPattern, event.Ref)
		match := WebhookRouteDryRunMatch{
			Route:   route,
			EventOK: eventOK,
			RefOK:   refOK,
			Matched: eventOK && refOK,
		}
		if match.Matched {
			match.BuildRef = buildRefFromWebhookRef(event.Ref)
			matchedCount++
		} else if !eventOK {
			match.BlockReason = "事件类型或仓库 push/tag 开关未命中"
		} else if !refOK {
			match.BlockReason = "ref_pattern 未命中当前 ref"
		}
		matches = append(matches, match)
	}
	message := "Webhook route 试跑未命中"
	if matchedCount > 0 {
		message = fmt.Sprintf("Webhook route 试跑命中 %d 条", matchedCount)
	} else if len(routes) == 0 {
		message = "没有找到已启用的仓库 Webhook route"
	}
	return WebhookRouteDryRunResponse{OK: true, Event: event, Matches: matches, MessageZh: message}, nil
}

func normalizeWebhookDryRunRef(ref, eventType string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "refs/") {
		return ref
	}
	if strings.TrimSpace(eventType) == "tag" || strings.HasPrefix(ref, "v") {
		return "refs/tags/" + strings.TrimPrefix(ref, "refs/tags/")
	}
	return "refs/heads/" + strings.TrimPrefix(ref, "refs/heads/")
}

func (s *Service) CreateBuildCenterRun(ctx context.Context, projectKey string, req BuildCenterRunRequest) (BuildCenterRunResponse, error) {
	store, ok := s.store.(BuildCenterExecutionStore)
	if !ok {
		return BuildCenterRunResponse{}, errBuildCenterStoreUnavailable
	}
	req.Action = normalizeBuildCenterAction(req.Action)
	if !isAllowedBuildCenterAction(req.Action) {
		return BuildCenterRunResponse{}, fmt.Errorf("unsupported build action: %s", req.Action)
	}
	run, profile, err := store.CreateBuildCenterRun(ctx, projectKey, req)
	if err != nil {
		return BuildCenterRunResponse{}, err
	}
	s.insertAudit(ctx, "build_center.run_create", "build_center_run", run.ID, "创建构建任务", map[string]any{
		"project_key":          projectKey,
		"project_id":           run.ProjectID,
		"build_profile_id":     run.BuildProfileID,
		"profile_key":          profile.ProfileKey,
		"build_center_project": profile.BuildCenterProject,
		"action":               run.Action,
		"trigger_type":         run.TriggerType,
		"trigger_source":       run.TriggerSource,
		"git_ref":              run.GitRef,
		"version_name":         run.VersionName,
		"version_code":         run.VersionCode,
		"build_number":         run.BuildNumber,
		"channel":              run.Channel,
		"started_by":           run.StartedBy,
	})
	if shouldDispatchBuildCenterRunToWorker(profile) {
		if err := s.dispatchBuildCenterRunToWorker(ctx, projectKey, run, profile); err != nil {
			return BuildCenterRunResponse{}, err
		}
		run.Status = "running"
		return BuildCenterRunResponse{Run: run}, nil
	}
	go s.executeBuildCenterRun(run, profile, req)
	return BuildCenterRunResponse{Run: run}, nil
}

func shouldDispatchBuildCenterRunToWorker(profile BuildProfileAdmin) bool {
	return strings.EqualFold(strings.TrimSpace(profile.StackType), "windows") || strings.EqualFold(strings.TrimSpace(profile.BuildType), "windows")
}

func (s *Service) dispatchBuildCenterRunToWorker(ctx context.Context, projectKey string, run BuildCenterRunAdmin, profile BuildProfileAdmin) error {
	workerStore, ok := s.store.(BuildCenterWorkerTaskStore)
	if !ok {
		return errWorkerStoreUnavailable
	}
	execStore, ok := s.store.(BuildCenterExecutionStore)
	if !ok {
		return errBuildCenterStoreUnavailable
	}
	metadata := map[string]any{
		"source":               "build_center",
		"project_key":          projectKey,
		"build_run_id":         run.ID,
		"build_profile_id":     profile.ID,
		"profile_key":          profile.ProfileKey,
		"build_center_project": profile.BuildCenterProject,
		"stack_type":           profile.StackType,
		"build_type":           profile.BuildType,
		"git_ref":              run.GitRef,
		"version_name":         run.VersionName,
		"version_code":         run.VersionCode,
		"channel":              run.Channel,
		"commands":             rawMessageToAny(profile.Commands, map[string]any{}),
		"artifact_rules":       rawMessageToAny(profile.ArtifactRules, []any{}),
		"env": map[string]any{
			"BUILD_CENTER_PROJECT": profile.BuildCenterProject,
			"BUILD_RUN_ID":         run.ID,
			"GIT_REF":              run.GitRef,
			"VERSION_NAME":         run.VersionName,
			"VERSION_CODE":         run.VersionCode,
			"CHANNEL":              run.Channel,
		},
	}
	task, err := workerStore.CreateWorkerTask(ctx, CreateWorkerTaskRequest{
		ProjectKey:     projectKey,
		BuildProfileID: profile.ID,
		BuildRunID:     run.ID,
		TaskType:       "build",
		Action:         run.Action,
		RequiredLabels: []string{"windows"},
		Priority:       10,
		Metadata:       metadata,
	})
	if err != nil {
		return err
	}
	if err := execStore.MarkBuildCenterRunStarted(ctx, run.ID, ""); err != nil {
		return err
	}
	s.insertAudit(ctx, "build_center.run_worker_dispatch", "worker_task", task.ID, "构建任务已投递 Windows Worker", map[string]any{
		"project_key":      projectKey,
		"build_run_id":     run.ID,
		"build_profile_id": profile.ID,
		"profile_key":      profile.ProfileKey,
		"worker_task_id":   task.ID,
		"required_labels":  []string{"windows"},
	})
	return nil
}

func rawMessageToAny(value json.RawMessage, fallback any) any {
	if len(value) == 0 {
		return fallback
	}
	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil || decoded == nil {
		return fallback
	}
	return decoded
}

func normalizeProjectLifecycleStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "active", "paused", "archived":
		return status
	default:
		return "active"
	}
}

func normalizeCodeProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch provider {
	case "github", "gitea", "gitlab", "generic":
		return provider
	default:
		return provider
	}
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func normalizeRawMessage(value json.RawMessage, fallback string) json.RawMessage {
	if len(value) == 0 || strings.TrimSpace(string(value)) == "null" {
		return json.RawMessage(fallback)
	}
	if !json.Valid(value) {
		return json.RawMessage(fallback)
	}
	return value
}

func (s *Service) executeBuildCenterRun(run BuildCenterRunAdmin, profile BuildProfileAdmin, req BuildCenterRunRequest) {
	store, ok := s.store.(BuildCenterExecutionStore)
	if !ok {
		return
	}
	start := time.Now()
	buildCenterRoot := envString("BUILD_CENTER_ROOT", "/root/build-center")
	logDir := filepath.Join(buildCenterRoot, "logs", "build-center-runs", run.ID)
	_ = os.MkdirAll(logDir, 0o755)
	logPath := filepath.Join(logDir, "buildctl.log")
	_ = store.MarkBuildCenterRunStarted(context.Background(), run.ID, logDir)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		_, _ = store.CompleteBuildCenterRun(context.Background(), run.ID, BuildCenterRunPatch{
			Status:       "failed",
			ExitCode:     1,
			LogDir:       logDir,
			UploadStatus: "failed",
			ErrorMessage: err.Error(),
			DurationMS:   time.Since(start).Milliseconds(),
		})
		return
	}
	defer logFile.Close()

	args := []string{run.Action, profile.BuildCenterProject}
	if req.GitRef != "" {
		args = append(args, "--ref", req.GitRef)
	}
	if run.VersionName != "" {
		args = append(args, "--version-name", run.VersionName)
	}
	cmd := exec.Command(filepath.Join(buildCenterRoot, "scripts", "buildctl"), args...)
	cmd.Dir = buildCenterRoot
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = buildCenterEnv(buildCenterRoot, run, req)

	exitCode := 0
	status := "success"
	errorMessage := ""
	if err := cmd.Run(); err != nil {
		status = "failed"
		exitCode = 1
		errorMessage = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	patch := BuildCenterRunPatch{
		Status:       status,
		ExitCode:     exitCode,
		LogDir:       logDir,
		UploadStatus: uploadStatusForAction(run.Action, status),
		ErrorMessage: errorMessage,
		DurationMS:   time.Since(start).Milliseconds(),
		Metadata: map[string]any{
			"buildctl_args": args,
			"log_path":      logPath,
		},
	}
	if status == "success" {
		if buildStatus, artifacts, err := readBuildCenterStatus(buildCenterRoot, profile.BuildCenterProject, run.VersionName, run.Action); err == nil {
			patch.GitCommit = buildStatus.GitCommit
			patch.VersionName = buildStatus.VersionName
			patch.VersionCode = buildStatus.VersionCode
			patch.BuildNumber = buildStatus.BuildNumber
			patch.ArtifactDir = buildStatus.ArtifactDir
			patch.ManifestPath = filepath.Join(buildStatus.ArtifactDir, "artifact-manifest.json")
			if buildStatus.LogDir != "" {
				patch.Metadata["build_log_dir"] = buildStatus.LogDir
			}
			_ = store.ReplaceBuildCenterRunArtifacts(context.Background(), run.ID, artifacts)
		} else {
			patch.Metadata["status_read_error"] = err.Error()
		}
	}
	_, _ = store.CompleteBuildCenterRun(context.Background(), run.ID, patch)
}

func buildCenterEnv(buildCenterRoot string, run BuildCenterRunAdmin, req BuildCenterRunRequest) []string {
	env := os.Environ()
	env = append(env, "BUILD_CENTER_ROOT="+buildCenterRoot)
	if run.VersionName != "" {
		env = append(env, "VERSION_NAME="+run.VersionName)
	}
	if run.VersionCode > 0 {
		env = append(env, fmt.Sprintf("VERSION_CODE=%d", run.VersionCode))
	}
	if run.Channel != "" {
		env = append(env, "CHANNEL="+run.Channel)
	}
	if req.StartedBy != "" {
		env = append(env, "BUILD_STARTED_BY="+req.StartedBy)
	}
	if os.Getenv("RELEASE_CENTER_BASE_URL") == "" {
		env = append(env, "RELEASE_CENTER_BASE_URL="+envString("BUILD_CENTER_RELEASE_BASE_URL", "http://127.0.0.1:18085"))
	}
	if os.Getenv("RELEASE_CENTER_TOKEN") == "" {
		if token := strings.TrimSpace(os.Getenv("CI_TOKEN")); token != "" {
			env = append(env, "RELEASE_CENTER_TOKEN="+token)
		}
	}
	return env
}

func readBuildCenterStatus(buildCenterRoot, project, versionName, action string) (buildctlStatus, []BuildCenterRunArtifact, error) {
	args := []string{"status", project}
	if versionName != "" {
		args = append(args, "--version-name", versionName)
	}
	out, err := exec.Command(filepath.Join(buildCenterRoot, "scripts", "buildctl"), args...).Output()
	if err != nil {
		return buildctlStatus{}, nil, err
	}
	var status buildctlStatus
	if err := json.Unmarshal(out, &status); err != nil {
		return buildctlStatus{}, nil, err
	}
	artifacts := make([]BuildCenterRunArtifact, 0, len(status.Artifacts))
	for _, artifact := range status.Artifacts {
		artifacts = append(artifacts, BuildCenterRunArtifact{
			Name:         artifact.Name,
			ArtifactType: artifact.Type,
			FileName:     artifact.FileName,
			LocalPath:    artifact.Path,
			SizeBytes:    artifact.SizeBytes,
			SHA256:       artifact.SHA256,
			UploadStatus: uploadStatusForAction(action, "success"),
		})
	}
	return status, artifacts, nil
}

func normalizeBuildCenterAction(action string) string {
	action = strings.TrimSpace(action)
	if action == "" {
		return "all"
	}
	return action
}

func isAllowedBuildCenterAction(action string) bool {
	switch action {
	case "fetch", "prepare", "build", "image", "upload", "verify", "all", "status":
		return true
	default:
		return false
	}
}

func webhookRefMatches(pattern, ref string) bool {
	pattern = strings.TrimSpace(pattern)
	ref = strings.TrimSpace(ref)
	if pattern == "" || pattern == "*" {
		return true
	}
	if pattern == ref {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return false
	}
	parts := strings.Split(pattern, "*")
	position := 0
	for index, part := range parts {
		if part == "" {
			continue
		}
		found := strings.Index(ref[position:], part)
		if found < 0 {
			return false
		}
		if index == 0 && !strings.HasPrefix(pattern, "*") && found != 0 {
			return false
		}
		position += found + len(part)
	}
	lastPart := parts[len(parts)-1]
	if lastPart != "" && !strings.HasSuffix(pattern, "*") && !strings.HasSuffix(ref, lastPart) {
		return false
	}
	return true
}

func webhookRouteAllowsEvent(route WebhookBuildRoute, event WebhookEventRequest) bool {
	eventType := strings.TrimSpace(event.EventType)
	if route.EventType != "" && route.EventType != "*" && route.EventType != eventType {
		return false
	}
	if eventType != "push" {
		return true
	}
	if strings.HasPrefix(event.Ref, "refs/tags/") {
		return route.TriggerOnTag
	}
	return route.TriggerOnPush
}

func uploadStatusForAction(action, status string) string {
	if status != "success" {
		return "failed"
	}
	switch action {
	case "upload", "all":
		return "success"
	default:
		return "pending"
	}
}

func tailLogLines(path string, limit int) ([]string, bool, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, false, err
	}
	text := strings.TrimRight(string(data), "\n")
	if text == "" {
		return []string{}, false, nil
	}
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	if limit <= 0 || len(lines) <= limit {
		return lines, false, nil
	}
	return lines[len(lines)-limit:], true, nil
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
