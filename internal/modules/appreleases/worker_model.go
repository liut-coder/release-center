package appreleases

import (
	"encoding/json"
	"time"
)

type WorkerRegisterRequest struct {
	WorkerKey   string         `json:"worker_key"`
	Name        string         `json:"name"`
	EndpointURL string         `json:"endpoint_url"`
	Labels      []string       `json:"labels"`
	Capacity    int            `json:"capacity"`
	Metadata    map[string]any `json:"metadata"`
}

type WorkerHeartbeatRequest struct {
	WorkerKey    string         `json:"worker_key"`
	Status       string         `json:"status"`
	Labels       []string       `json:"labels"`
	Capacity     int            `json:"capacity"`
	RunningTasks int            `json:"running_tasks"`
	Metadata     map[string]any `json:"metadata"`
}

type WorkerTaskNextRequest struct {
	WorkerKey string `json:"worker_key"`
}

type WorkerTaskLogsRequest struct {
	WorkerKey  string   `json:"worker_key"`
	LeaseToken string   `json:"lease_token"`
	Lines      []string `json:"lines"`
}

type WorkerTaskArtifactsRequest struct {
	WorkerKey  string               `json:"worker_key"`
	LeaseToken string               `json:"lease_token"`
	Artifacts  []WorkerTaskArtifact `json:"artifacts"`
	Metadata   map[string]any       `json:"metadata"`
}

type WorkerTaskCompleteRequest struct {
	WorkerKey  string               `json:"worker_key"`
	LeaseToken string               `json:"lease_token"`
	Artifacts  []WorkerTaskArtifact `json:"artifacts"`
	Metadata   map[string]any       `json:"metadata"`
}

type WorkerTaskFailRequest struct {
	WorkerKey    string         `json:"worker_key"`
	LeaseToken   string         `json:"lease_token"`
	ErrorMessage string         `json:"error_message"`
	Metadata     map[string]any `json:"metadata"`
}

type CreateWorkerTaskRequest struct {
	ProjectKey     string         `json:"project_key"`
	BuildProfileID string         `json:"build_profile_id"`
	BuildRunID     string         `json:"build_run_id"`
	TaskType       string         `json:"task_type"`
	Action         string         `json:"action"`
	RequiredLabels []string       `json:"required_labels"`
	Priority       int            `json:"priority"`
	Metadata       map[string]any `json:"metadata"`
}

type BuildWorkerAdmin struct {
	ID           string          `json:"id"`
	WorkerKey    string          `json:"worker_key"`
	Name         string          `json:"name"`
	EndpointURL  string          `json:"endpoint_url"`
	Labels       []string        `json:"labels"`
	Status       string          `json:"status"`
	Capacity     int             `json:"capacity"`
	RunningTasks int             `json:"running_tasks"`
	LastSeenAt   time.Time       `json:"last_seen_at,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type WorkerTaskAdmin struct {
	ID               string          `json:"id"`
	WorkerID         string          `json:"worker_id,omitempty"`
	BuildRunID       string          `json:"build_run_id,omitempty"`
	ProjectID        string          `json:"project_id,omitempty"`
	BuildProfileID   string          `json:"build_profile_id,omitempty"`
	TaskType         string          `json:"task_type"`
	Action           string          `json:"action"`
	Status           string          `json:"status"`
	RequiredLabels   []string        `json:"required_labels"`
	Priority         int             `json:"priority"`
	LeaseToken       string          `json:"lease_token,omitempty"`
	LeasedUntil      time.Time       `json:"leased_until,omitempty"`
	Attempts         int             `json:"attempts"`
	LogTail          []string        `json:"log_tail,omitempty"`
	ArtifactManifest json.RawMessage `json:"artifact_manifest,omitempty"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	StartedAt        time.Time       `json:"started_at,omitempty"`
	FinishedAt       time.Time       `json:"finished_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type WorkerTaskArtifact struct {
	Name         string         `json:"name"`
	ArtifactType string         `json:"artifact_type"`
	FileName     string         `json:"file_name"`
	LocalPath    string         `json:"local_path"`
	SizeBytes    int64          `json:"size_bytes"`
	SHA256       string         `json:"sha256"`
	DownloadURL  string         `json:"download_url"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type WorkerActionResponse struct {
	OK        bool              `json:"ok"`
	Worker    *BuildWorkerAdmin `json:"worker,omitempty"`
	Task      *WorkerTaskAdmin  `json:"task,omitempty"`
	MessageZh string            `json:"message_zh,omitempty"`
}

type WorkerTaskNextResponse struct {
	Task      *WorkerTaskAdmin `json:"task,omitempty"`
	MessageZh string           `json:"message_zh,omitempty"`
}

type WorkerOverviewResponse struct {
	Workers   []BuildWorkerAdmin `json:"workers"`
	Tasks     []WorkerTaskAdmin  `json:"tasks"`
	MessageZh string             `json:"message_zh,omitempty"`
}
