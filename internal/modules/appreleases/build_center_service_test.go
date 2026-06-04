package appreleases

import (
	"context"
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
	audits    []capturedAudit
	completed chan BuildCenterRunPatch
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

func (s *buildCenterAuditStore) CreateBuildCenterRun(_ context.Context, projectKey string, req BuildCenterRunRequest) (BuildCenterRunAdmin, BuildProfileAdmin, error) {
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
		}, nil
}

func (s *buildCenterAuditStore) MarkBuildCenterRunStarted(_ context.Context, runID, logDir string) error {
	return nil
}

func (s *buildCenterAuditStore) CompleteBuildCenterRun(_ context.Context, runID string, patch BuildCenterRunPatch) (BuildCenterRunAdmin, error) {
	if s.completed != nil {
		s.completed <- patch
	}
	return BuildCenterRunAdmin{ID: runID, Status: patch.Status}, nil
}

func (s *buildCenterAuditStore) ReplaceBuildCenterRunArtifacts(_ context.Context, runID string, artifacts []BuildCenterRunArtifact) error {
	return nil
}
