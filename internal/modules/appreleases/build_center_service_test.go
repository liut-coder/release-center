package appreleases

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTailLogLinesReturnsTailAndTruncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "buildctl.log")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	lines, truncated, err := tailLogLines(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated {
		t.Fatal("expected log tail to be marked truncated")
	}
	if len(lines) != 2 || lines[0] != "two" || lines[1] != "three" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}

func TestTailLogLinesHandlesEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "buildctl.log")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	lines, truncated, err := tailLogLines(path, 200)
	if err != nil {
		t.Fatal(err)
	}
	if truncated || len(lines) != 0 {
		t.Fatalf("expected empty non-truncated lines, got truncated=%v lines=%#v", truncated, lines)
	}
}

func TestWebhookRefMatchesSupportsSimpleGlob(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pattern string
		ref     string
		want    bool
	}{
		{name: "all", pattern: "*", ref: "refs/heads/main", want: true},
		{name: "exact", pattern: "refs/heads/main", ref: "refs/heads/main", want: true},
		{name: "tag glob", pattern: "refs/tags/*", ref: "refs/tags/v1.0.0", want: true},
		{name: "branch mismatch", pattern: "refs/heads/release/*", ref: "refs/heads/main", want: false},
		{name: "middle glob", pattern: "refs/heads/*/hotfix", ref: "refs/heads/prod/hotfix", want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := webhookRefMatches(tc.pattern, tc.ref); got != tc.want {
				t.Fatalf("webhookRefMatches(%q, %q) = %v, want %v", tc.pattern, tc.ref, got, tc.want)
			}
		})
	}
}

func TestWebhookRouteAllowsPushAndTagSwitches(t *testing.T) {
	route := WebhookBuildRoute{EventType: "push", TriggerOnPush: true, TriggerOnTag: false}
	if !webhookRouteAllowsEvent(route, WebhookEventRequest{EventType: "push", Ref: "refs/heads/main"}) {
		t.Fatal("expected branch push to be allowed")
	}
	if webhookRouteAllowsEvent(route, WebhookEventRequest{EventType: "push", Ref: "refs/tags/v1.0.0"}) {
		t.Fatal("expected tag push to be blocked when trigger_on_tag is false")
	}
	if webhookRouteAllowsEvent(route, WebhookEventRequest{EventType: "workflow_run", Ref: "refs/heads/main"}) {
		t.Fatal("expected different event type to be blocked")
	}
}

func TestBuildRefFromWebhookRef(t *testing.T) {
	for _, tc := range []struct {
		ref  string
		want string
	}{
		{ref: "refs/heads/main", want: "main"},
		{ref: "refs/tags/v1.0.0", want: "v1.0.0"},
		{ref: "feature/demo", want: "feature/demo"},
	} {
		if got := buildRefFromWebhookRef(tc.ref); got != tc.want {
			t.Fatalf("buildRefFromWebhookRef(%q) = %q, want %q", tc.ref, got, tc.want)
		}
	}
}

func TestDryRunWebhookRouteReportsMatchedRoute(t *testing.T) {
	store := &buildCenterAuditStore{
		webhookRoutes: []WebhookBuildRoute{
			{
				ID:            "route-1",
				ProjectKey:    "release-center",
				Repository:    "liut-coder/release-center",
				ProfileKey:    "web",
				EventType:     "push",
				RefPattern:    "refs/heads/main",
				Action:        "all",
				TriggerOnPush: true,
			},
		},
	}
	service := NewServiceWithStore(Config{}, store)

	resp, err := service.DryRunWebhookRoute(context.Background(), WebhookRouteDryRunRequest{
		Provider:   "github",
		Repository: "liut-coder/release-center",
		EventType:  "push",
		Ref:        "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Matches) != 1 || !resp.Matches[0].Matched {
		t.Fatalf("expected one matched route, got %#v", resp.Matches)
	}
	if resp.Matches[0].BuildRef != "main" {
		t.Fatalf("expected build ref main, got %q", resp.Matches[0].BuildRef)
	}
}

func TestDryRunWebhookRouteReportsBlockedRoute(t *testing.T) {
	store := &buildCenterAuditStore{
		webhookRoutes: []WebhookBuildRoute{
			{
				ID:            "route-1",
				ProjectKey:    "release-center",
				Repository:    "liut-coder/release-center",
				ProfileKey:    "web",
				EventType:     "push",
				RefPattern:    "refs/heads/main",
				Action:        "all",
				TriggerOnPush: false,
				TriggerOnTag:  false,
			},
		},
	}
	service := NewServiceWithStore(Config{}, store)

	resp, err := service.DryRunWebhookRoute(context.Background(), WebhookRouteDryRunRequest{
		Provider:   "github",
		Repository: "liut-coder/release-center",
		EventType:  "push",
		Ref:        "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Matches) != 1 || resp.Matches[0].Matched || resp.Matches[0].BlockReason == "" {
		t.Fatalf("expected blocked route with reason, got %#v", resp.Matches)
	}
}

func TestBuildCenterConfigActionsInsertAudit(t *testing.T) {
	enabled := true
	store := &buildCenterAuditStore{}
	service := &Service{cfg: Config{Channel: "stable"}, store: store}
	ctx := context.Background()

	if _, err := service.CreateBuildCenterProject(ctx, BuildCenterProjectRequest{
		ProjectKey:      " release-center ",
		Name:            "Release Center",
		OwnerAccount:    "ops",
		LifecycleStatus: "paused",
		DefaultChannel:  "prod",
	}); err != nil {
		t.Fatalf("CreateBuildCenterProject() error = %v", err)
	}
	if _, err := service.UpsertCodeRepository(ctx, "release-center", CodeRepositoryRequest{
		Provider:       " GitHub ",
		RepoURL:        "https://github.com/example/release-center.git",
		RepoFullName:   "example/release-center",
		WebhookEnabled: &enabled,
		TriggerOnPush:  &enabled,
		TriggerOnTag:   &enabled,
	}); err != nil {
		t.Fatalf("UpsertCodeRepository() error = %v", err)
	}
	if _, err := service.UpsertBuildProfile(ctx, "release-center", BuildProfileRequest{
		ProfileKey:         "web",
		Name:               "Web",
		BuildCenterProject: "release-center",
		StackType:          "node",
		BuildAction:        "all",
		Enabled:            &enabled,
	}); err != nil {
		t.Fatalf("UpsertBuildProfile() error = %v", err)
	}
	if _, err := service.UpsertWebhookRoute(ctx, "release-center", WebhookRouteRequest{
		RepositoryID: "repo-1",
		ProfileKey:   "web",
		EventType:    "push",
		RefPattern:   "refs/heads/main",
		Action:       "all",
		Enabled:      &enabled,
	}); err != nil {
		t.Fatalf("UpsertWebhookRoute() error = %v", err)
	}

	for _, action := range []string{
		"build_center.project_save",
		"build_center.repository_save",
		"build_center.profile_save",
		"build_center.webhook_route_save",
	} {
		if _, ok := findCapturedAudit(store.audits, action); !ok {
			t.Fatalf("expected audit action %q, got %#v", action, store.audits)
		}
	}
	repositoryAudit, _ := findCapturedAudit(store.audits, "build_center.repository_save")
	if repositoryAudit.metadata["provider"] != "github" || repositoryAudit.metadata["repo_full_name"] != "example/release-center" {
		t.Fatalf("unexpected repository audit metadata: %#v", repositoryAudit.metadata)
	}
}

func TestCreateBuildCenterRunDispatchesWindowsProfileToWorker(t *testing.T) {
	store := &buildCenterAuditStore{}
	service := &Service{store: store}
	resp, err := service.CreateBuildCenterRun(context.Background(), "desktop-tool", BuildCenterRunRequest{
		ProfileKey:  "windows",
		Action:      "all",
		GitRef:      "main",
		VersionName: "2.0.0",
		VersionCode: 20,
		Channel:     "stable",
		StartedBy:   "tester",
	})
	if err != nil {
		t.Fatalf("CreateBuildCenterRun() error = %v", err)
	}
	if resp.Run.ID != "run-1" {
		t.Fatalf("unexpected run response: %#v", resp)
	}
	if len(store.workerRequests) != 1 {
		t.Fatalf("expected one worker task, got %#v", store.workerRequests)
	}
	req := store.workerRequests[0]
	if req.BuildRunID != "run-1" || req.BuildProfileID != "profile-1" || req.TaskType != "build" || req.Action != "all" {
		t.Fatalf("unexpected worker request: %#v", req)
	}
	if len(req.RequiredLabels) != 1 || req.RequiredLabels[0] != "windows" {
		t.Fatalf("expected windows label, got %#v", req.RequiredLabels)
	}
	env, ok := req.Metadata["env"].(map[string]any)
	if !ok || env["VERSION_NAME"] != "2.0.0" || env["CHANNEL"] != "stable" {
		t.Fatalf("expected version env metadata, got %#v", req.Metadata)
	}
	if store.startedRunID != "run-1" {
		t.Fatalf("expected run to be marked started, got %q", store.startedRunID)
	}
	if _, ok := findCapturedAudit(store.audits, "build_center.run_worker_dispatch"); !ok {
		t.Fatalf("expected worker dispatch audit, got %#v", store.audits)
	}
}

func TestWindowsBuildCenterRunWorkerLifecycleMirrorsArtifacts(t *testing.T) {
	store := &buildCenterAuditStore{}
	service := &Service{store: store}
	ctx := context.Background()

	runResp, err := service.CreateBuildCenterRun(ctx, "desktop-tool", BuildCenterRunRequest{
		ProfileKey:  "windows",
		Action:      "all",
		GitRef:      "refs/tags/v2.0.0",
		VersionName: "2.0.0",
		VersionCode: 20,
		Channel:     "stable",
		StartedBy:   "tester",
	})
	if err != nil {
		t.Fatalf("CreateBuildCenterRun() error = %v", err)
	}
	if runResp.Run.Status != "running" || store.startedRunID != "run-1" {
		t.Fatalf("expected running Windows build run, resp=%#v started=%q", runResp, store.startedRunID)
	}

	if _, err := service.RegisterWorker(ctx, WorkerRegisterRequest{WorkerKey: "linux-worker", Labels: []string{"linux", "go"}, Capacity: 1}); err != nil {
		t.Fatalf("RegisterWorker(linux) error = %v", err)
	}
	linuxNext, err := service.NextWorkerTask(ctx, WorkerTaskNextRequest{WorkerKey: "linux-worker"})
	if err != nil {
		t.Fatalf("NextWorkerTask(linux) error = %v", err)
	}
	if linuxNext.Task != nil {
		t.Fatalf("linux worker should not lease Windows task: %#v", linuxNext.Task)
	}

	if _, err := service.RegisterWorker(ctx, WorkerRegisterRequest{WorkerKey: "windows-worker", Labels: []string{"windows", "go", "amd64"}, Capacity: 1}); err != nil {
		t.Fatalf("RegisterWorker(windows) error = %v", err)
	}
	workerNext, err := service.NextWorkerTask(ctx, WorkerTaskNextRequest{WorkerKey: "windows-worker"})
	if err != nil {
		t.Fatalf("NextWorkerTask(windows) error = %v", err)
	}
	if workerNext.Task == nil {
		t.Fatal("expected windows worker to lease build task")
	}
	if workerNext.Task.BuildRunID != "run-1" || workerNext.Task.LeaseToken == "" {
		t.Fatalf("unexpected leased task: %#v", workerNext.Task)
	}

	var metadata map[string]any
	if err := json.Unmarshal(workerNext.Task.Metadata, &metadata); err != nil {
		t.Fatalf("task metadata should be json: %v", err)
	}
	if metadata["source"] != "build_center" || metadata["build_run_id"] != "run-1" {
		t.Fatalf("unexpected task metadata: %#v", metadata)
	}
	env, ok := metadata["env"].(map[string]any)
	if !ok || env["VERSION_NAME"] != "2.0.0" || env["CHANNEL"] != "stable" {
		t.Fatalf("expected version/channel env metadata, got %#v", metadata["env"])
	}
	rules, ok := metadata["artifact_rules"].([]any)
	if !ok || len(rules) != 2 {
		t.Fatalf("expected windows artifact rules in metadata, got %#v", metadata["artifact_rules"])
	}

	completeResp, err := service.CompleteWorkerTask(ctx, workerNext.Task.ID, WorkerTaskCompleteRequest{
		WorkerKey:  "windows-worker",
		LeaseToken: workerNext.Task.LeaseToken,
		Artifacts: []WorkerTaskArtifact{
			{
				Name:         "windows-exe",
				ArtifactType: "windows_exe",
				FileName:     "desktop-tool.exe",
				LocalPath:    `C:\build-worker\workspace\desktop-tool\dist\windows\desktop-tool.exe`,
				SizeBytes:    4096,
				SHA256:       "aaa111",
			},
			{
				Name:         "windows-archive",
				ArtifactType: "windows_archive",
				FileName:     "desktop-tool-2.0.0-windows-amd64.zip",
				LocalPath:    `C:\build-worker\workspace\desktop-tool\dist\windows\desktop-tool-2.0.0-windows-amd64.zip`,
				SizeBytes:    8192,
				SHA256:       "bbb222",
			},
		},
		Metadata: map[string]any{"worker_artifact_count": 2},
	})
	if err != nil {
		t.Fatalf("CompleteWorkerTask() error = %v", err)
	}
	if !completeResp.OK || completeResp.Task == nil || completeResp.Task.Status != "success" {
		t.Fatalf("unexpected complete response: %#v", completeResp)
	}
	if store.lastCompletedRunID != "run-1" || store.lastCompletedPatch.Status != "success" || store.lastCompletedPatch.UploadStatus != "uploaded" {
		t.Fatalf("expected build run success patch, run=%q patch=%#v", store.lastCompletedRunID, store.lastCompletedPatch)
	}
	if store.mirroredRunID != "run-1" || len(store.mirroredArtifacts) != 2 {
		t.Fatalf("expected mirrored artifacts for run-1, run=%q artifacts=%#v", store.mirroredRunID, store.mirroredArtifacts)
	}
	if store.mirroredArtifacts[0].ArtifactType != "windows_exe" || store.mirroredArtifacts[0].UploadStatus != "local" {
		t.Fatalf("unexpected mirrored exe: %#v", store.mirroredArtifacts[0])
	}
	if store.mirroredArtifacts[1].ArtifactType != "windows_archive" || store.mirroredArtifacts[1].SHA256 != "bbb222" {
		t.Fatalf("unexpected mirrored archive: %#v", store.mirroredArtifacts[1])
	}
}

func TestCreateBuildCenterRunInsertsAudit(t *testing.T) {
	root := t.TempDir()
	scriptsDir := filepath.Join(root, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	buildctl := filepath.Join(scriptsDir, "buildctl")
	script := `#!/bin/sh
if [ "$1" = "status" ]; then
  printf '%s\n' '{"build_id":"run-1","project":"release-center","git_commit":"abc123","version_name":"1.2.3","version_code":12,"build_number":42,"status":"success","artifact_dir":"/tmp/artifacts","artifacts":[]}'
  exit 0
fi
exit 0
`
	if err := os.WriteFile(buildctl, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BUILD_CENTER_ROOT", root)

	store := &buildCenterAuditStore{completed: make(chan BuildCenterRunPatch, 1)}
	service := &Service{store: store}
	resp, err := service.CreateBuildCenterRun(context.Background(), "release-center", BuildCenterRunRequest{
		ProfileKey:  "web",
		Action:      "all",
		GitRef:      "main",
		VersionName: "1.2.3",
		VersionCode: 12,
		Channel:     "prod",
		StartedBy:   "tester",
	})
	if err != nil {
		t.Fatalf("CreateBuildCenterRun() error = %v", err)
	}
	if resp.Run.ID != "run-1" {
		t.Fatalf("unexpected run response: %#v", resp)
	}
	audit, ok := findCapturedAudit(store.audits, "build_center.run_create")
	if !ok {
		t.Fatalf("expected build run audit, got %#v", store.audits)
	}
	if audit.targetType != "build_center_run" || audit.targetID != "run-1" {
		t.Fatalf("unexpected audit target: %#v", audit)
	}
	if audit.metadata["profile_key"] != "web" || audit.metadata["started_by"] != "tester" {
		t.Fatalf("unexpected audit metadata: %#v", audit.metadata)
	}

	select {
	case patch := <-store.completed:
		if patch.Status != "success" {
			t.Fatalf("expected async build to complete successfully, got %#v", patch)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async build to finish")
	}
}

type buildCenterAuditStore struct {
	Store
	audits             []capturedAudit
	completed          chan BuildCenterRunPatch
	webhookRoutes      []WebhookBuildRoute
	workerRequests     []CreateWorkerTaskRequest
	startedRunID       string
	workers            map[string]BuildWorkerAdmin
	workerTasks        []WorkerTaskAdmin
	mirroredRunID      string
	mirroredArtifacts  []BuildCenterRunArtifact
	lastCompletedRunID string
	lastCompletedPatch BuildCenterRunPatch
}

func (s *buildCenterAuditStore) InsertAudit(_ context.Context, action, targetType, targetID, message string, metadata map[string]any) error {
	s.audits = append(s.audits, capturedAudit{
		action:     action,
		targetType: targetType,
		targetID:   targetID,
		message:    message,
		metadata:   metadata,
	})
	return nil
}

func (s *buildCenterAuditStore) BuildCenterOverview(_ context.Context) (BuildCenterOverview, error) {
	return BuildCenterOverview{}, nil
}

func (s *buildCenterAuditStore) BuildCenterProject(_ context.Context, projectKey string) (BuildCenterProject, error) {
	return BuildCenterProject{}, nil
}

func (s *buildCenterAuditStore) BuildCenterRun(_ context.Context, runID string) (BuildCenterRunAdmin, error) {
	return BuildCenterRunAdmin{}, nil
}

func (s *buildCenterAuditStore) DeploymentTargets(_ context.Context) ([]DeploymentTargetAdmin, error) {
	return nil, nil
}

func (s *buildCenterAuditStore) CreateBuildCenterProject(_ context.Context, req BuildCenterProjectRequest) (BuildCenterProject, error) {
	return BuildCenterProject{
		ID:              "project-1",
		ProjectKey:      req.ProjectKey,
		Name:            req.Name,
		OwnerAccount:    req.OwnerAccount,
		LifecycleStatus: req.LifecycleStatus,
		DefaultChannel:  req.DefaultChannel,
	}, nil
}

func (s *buildCenterAuditStore) UpsertCodeRepository(_ context.Context, projectKey string, req CodeRepositoryRequest) (CodeRepositoryAdmin, error) {
	return CodeRepositoryAdmin{
		ID:             "repo-1",
		ProjectID:      "project-1",
		Provider:       req.Provider,
		RepoURL:        req.RepoURL,
		RepoFullName:   req.RepoFullName,
		DefaultRef:     req.DefaultRef,
		WebhookEnabled: boolValue(req.WebhookEnabled, true),
		TriggerOnPush:  boolValue(req.TriggerOnPush, true),
		TriggerOnTag:   boolValue(req.TriggerOnTag, false),
	}, nil
}

func (s *buildCenterAuditStore) UpsertBuildProfile(_ context.Context, projectKey string, req BuildProfileRequest) (BuildProfileAdmin, error) {
	return BuildProfileAdmin{
		ID:                 "profile-1",
		ProjectID:          "project-1",
		ProfileKey:         req.ProfileKey,
		Name:               req.Name,
		BuildCenterProject: req.BuildCenterProject,
		StackType:          req.StackType,
		BuildType:          req.BuildType,
		DefaultRef:         req.DefaultRef,
		DefaultChannel:     req.DefaultChannel,
		BuildAction:        req.BuildAction,
		Enabled:            boolValue(req.Enabled, true),
	}, nil
}

func (s *buildCenterAuditStore) UpsertWebhookRoute(_ context.Context, projectKey string, req WebhookRouteRequest) (WebhookRouteAdmin, error) {
	return WebhookRouteAdmin{
		ID:             "route-1",
		ProjectID:      "project-1",
		RepositoryID:   req.RepositoryID,
		BuildProfileID: "profile-1",
		ProfileKey:     req.ProfileKey,
		EventType:      req.EventType,
		RefPattern:     req.RefPattern,
		Action:         req.Action,
		Enabled:        boolValue(req.Enabled, true),
	}, nil
}

func (s *buildCenterAuditStore) MatchWebhookBuildRoutes(_ context.Context, event WebhookEventRequest) ([]WebhookBuildRoute, error) {
	if event.Repository == "" {
		return []WebhookBuildRoute{}, nil
	}
	return append([]WebhookBuildRoute{}, s.webhookRoutes...), nil
}

func (s *buildCenterAuditStore) CreateBuildCenterRun(_ context.Context, projectKey string, req BuildCenterRunRequest) (BuildCenterRunAdmin, BuildProfileAdmin, error) {
	stackType := "node"
	buildType := "web"
	if req.ProfileKey == "windows" {
		stackType = "windows"
		buildType = "windows"
	}
	return BuildCenterRunAdmin{
			ID:             "run-1",
			ProjectID:      "project-1",
			BuildProfileID: "profile-1",
			TriggerType:    "manual",
			TriggerSource:  "admin_api",
			Action:         req.Action,
			GitRef:         req.GitRef,
			VersionName:    req.VersionName,
			VersionCode:    req.VersionCode,
			BuildNumber:    42,
			Channel:        req.Channel,
			Status:         "queued",
			StartedBy:      req.StartedBy,
		}, BuildProfileAdmin{
			ID:                 "profile-1",
			ProfileKey:         req.ProfileKey,
			BuildCenterProject: "release-center",
			StackType:          stackType,
			BuildType:          buildType,
			Commands:           json.RawMessage(`{"all":"buildctl all release-center"}`),
			ArtifactRules:      json.RawMessage(`[{"name":"windows-exe","type":"windows_exe","path":"dist/windows/*.exe"},{"name":"windows-archive","type":"windows_archive","path":"dist/windows/*.zip"}]`),
		}, nil
}

func (s *buildCenterAuditStore) CreateWorkerTask(_ context.Context, req CreateWorkerTaskRequest) (WorkerTaskAdmin, error) {
	s.workerRequests = append(s.workerRequests, req)
	task := WorkerTaskAdmin{ID: "worker-task-1", BuildRunID: req.BuildRunID, BuildProfileID: req.BuildProfileID, TaskType: req.TaskType, Action: req.Action, Status: "queued", RequiredLabels: req.RequiredLabels, Priority: req.Priority, Metadata: json.RawMessage(jsonb(req.Metadata))}
	s.workerTasks = append(s.workerTasks, task)
	return task, nil
}

func (s *buildCenterAuditStore) WorkerOverview(_ context.Context) (WorkerOverviewResponse, error) {
	return WorkerOverviewResponse{}, nil
}

func (s *buildCenterAuditStore) RegisterWorker(_ context.Context, req WorkerRegisterRequest) (BuildWorkerAdmin, error) {
	if s.workers == nil {
		s.workers = map[string]BuildWorkerAdmin{}
	}
	worker := BuildWorkerAdmin{ID: "worker-" + req.WorkerKey, WorkerKey: req.WorkerKey, Labels: req.Labels, Capacity: req.Capacity, Status: "online"}
	s.workers[req.WorkerKey] = worker
	return worker, nil
}

func (s *buildCenterAuditStore) SaveWorkerHeartbeat(_ context.Context, req WorkerHeartbeatRequest) (BuildWorkerAdmin, error) {
	return BuildWorkerAdmin{WorkerKey: req.WorkerKey, Labels: req.Labels, Status: req.Status, Capacity: req.Capacity}, nil
}

func (s *buildCenterAuditStore) NextWorkerTask(_ context.Context, req WorkerTaskNextRequest) (*WorkerTaskAdmin, error) {
	worker, ok := s.workers[req.WorkerKey]
	if !ok {
		return nil, nil
	}
	for i := range s.workerTasks {
		task := &s.workerTasks[i]
		if task.Status != "queued" || !labelsContainAll(worker.Labels, task.RequiredLabels) {
			continue
		}
		task.Status = "leased"
		task.WorkerID = worker.ID
		task.LeaseToken = "lease-1"
		task.Attempts++
		return task, nil
	}
	return nil, nil
}

func (s *buildCenterAuditStore) AppendWorkerTaskLogs(_ context.Context, taskID string, _ WorkerTaskLogsRequest) (WorkerTaskAdmin, error) {
	return WorkerTaskAdmin{ID: taskID, Status: "running"}, nil
}

func (s *buildCenterAuditStore) SaveWorkerTaskArtifacts(_ context.Context, taskID string, _ WorkerTaskArtifactsRequest) (WorkerTaskAdmin, error) {
	return WorkerTaskAdmin{ID: taskID, BuildRunID: "run-1", TaskType: "build", Action: "all", Status: "running"}, nil
}

func (s *buildCenterAuditStore) CompleteWorkerTask(_ context.Context, taskID string, req WorkerTaskCompleteRequest) (WorkerTaskAdmin, error) {
	for i := range s.workerTasks {
		if s.workerTasks[i].ID == taskID {
			s.workerTasks[i].Status = "success"
			_, _ = s.CompleteBuildCenterRun(context.Background(), s.workerTasks[i].BuildRunID, BuildCenterRunPatch{Status: "success", UploadStatus: "uploaded", Metadata: map[string]any{"worker_task_id": taskID, "worker_key": req.WorkerKey}})
			return s.workerTasks[i], nil
		}
	}
	_, _ = s.CompleteBuildCenterRun(context.Background(), "run-1", BuildCenterRunPatch{Status: "success", UploadStatus: "uploaded", Metadata: map[string]any{"worker_task_id": taskID, "worker_key": req.WorkerKey}})
	return WorkerTaskAdmin{ID: taskID, BuildRunID: "run-1", TaskType: "build", Action: "all", Status: "success"}, nil
}

func (s *buildCenterAuditStore) FailWorkerTask(_ context.Context, taskID string, _ WorkerTaskFailRequest) (WorkerTaskAdmin, error) {
	return WorkerTaskAdmin{ID: taskID, Status: "failed"}, nil
}

func (s *buildCenterAuditStore) MarkBuildCenterRunStarted(_ context.Context, runID, logDir string) error {
	s.startedRunID = runID
	return nil
}

func (s *buildCenterAuditStore) CompleteBuildCenterRun(_ context.Context, runID string, patch BuildCenterRunPatch) (BuildCenterRunAdmin, error) {
	s.lastCompletedRunID = runID
	s.lastCompletedPatch = patch
	if s.completed != nil {
		s.completed <- patch
	}
	return BuildCenterRunAdmin{ID: runID, Status: patch.Status}, nil
}

func (s *buildCenterAuditStore) ReplaceBuildCenterRunArtifacts(_ context.Context, runID string, artifacts []BuildCenterRunArtifact) error {
	s.mirroredRunID = runID
	s.mirroredArtifacts = artifacts
	return nil
}

func labelsContainAll(labels, required []string) bool {
	seen := map[string]struct{}{}
	for _, label := range labels {
		seen[label] = struct{}{}
	}
	for _, label := range required {
		if _, ok := seen[label]; !ok {
			return false
		}
	}
	return true
}
