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
	DeploymentTargets(ctx context.Context) ([]DeploymentTargetAdmin, error)
}

type BuildCenterExecutionStore interface {
	BuildCenterStore
	CreateBuildCenterRun(ctx context.Context, projectKey string, req BuildCenterRunRequest) (BuildCenterRunAdmin, BuildProfileAdmin, error)
	MarkBuildCenterRunStarted(ctx context.Context, runID, logDir string) error
	CompleteBuildCenterRun(ctx context.Context, runID string, patch BuildCenterRunPatch) (BuildCenterRunAdmin, error)
	ReplaceBuildCenterRunArtifacts(ctx context.Context, runID string, artifacts []BuildCenterRunArtifact) error
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

func (s *Service) DeploymentTargets(ctx context.Context) ([]DeploymentTargetAdmin, error) {
	store, ok := s.store.(BuildCenterStore)
	if !ok {
		return []DeploymentTargetAdmin{}, nil
	}
	return store.DeploymentTargets(ctx)
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
	go s.executeBuildCenterRun(run, profile, req)
	return BuildCenterRunResponse{Run: run}, nil
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

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
