package appreleases

import (
	"encoding/json"
	"time"
)

type BuildCenterOverview struct {
	Projects          []BuildCenterProject    `json:"projects"`
	DeploymentTargets []DeploymentTargetAdmin `json:"deployment_targets"`
	MessageZh         string                  `json:"message_zh"`
}

type BuildCenterRunRequest struct {
	ProfileKey  string `json:"profile_key"`
	Action      string `json:"action"`
	GitRef      string `json:"git_ref"`
	VersionName string `json:"version_name"`
	VersionCode int    `json:"version_code"`
	Channel     string `json:"channel"`
	StartedBy   string `json:"started_by"`
}

type BuildCenterRunResponse struct {
	Run BuildCenterRunAdmin `json:"run"`
}

type BuildCenterProject struct {
	ID                string                  `json:"id"`
	ProjectKey        string                  `json:"project_key"`
	Name              string                  `json:"name"`
	Description       string                  `json:"description"`
	OwnerAccount      string                  `json:"owner_account"`
	LifecycleStatus   string                  `json:"lifecycle_status"`
	DefaultChannel    string                  `json:"default_channel"`
	Metadata          json.RawMessage         `json:"metadata,omitempty"`
	Repositories      []CodeRepositoryAdmin   `json:"repositories"`
	BuildProfiles     []BuildProfileAdmin     `json:"build_profiles"`
	RecentRuns        []BuildCenterRunAdmin   `json:"recent_runs"`
	DeploymentTargets []DeploymentTargetAdmin `json:"deployment_targets"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
}

type CodeRepositoryAdmin struct {
	ID               string          `json:"id"`
	ProjectID        string          `json:"project_id"`
	Provider         string          `json:"provider"`
	RepoURL          string          `json:"repo_url"`
	RepoFullName     string          `json:"repo_full_name"`
	DefaultRef       string          `json:"default_ref"`
	CredentialRef    string          `json:"credential_ref,omitempty"`
	WebhookSecretRef string          `json:"webhook_secret_ref,omitempty"`
	WebhookEnabled   bool            `json:"webhook_enabled"`
	TriggerOnPush    bool            `json:"trigger_on_push"`
	TriggerOnTag     bool            `json:"trigger_on_tag"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type BuildProfileAdmin struct {
	ID                 string          `json:"id"`
	ProjectID          string          `json:"project_id"`
	AppID              string          `json:"app_id,omitempty"`
	ProfileKey         string          `json:"profile_key"`
	Name               string          `json:"name"`
	BuildCenterProject string          `json:"build_center_project"`
	StackType          string          `json:"stack_type"`
	BuildType          string          `json:"build_type"`
	ConfigPath         string          `json:"config_path"`
	SourceWorkdir      string          `json:"source_workdir"`
	DefaultRef         string          `json:"default_ref"`
	DefaultVersionName string          `json:"default_version_name"`
	DefaultVersionCode int             `json:"default_version_code"`
	DefaultChannel     string          `json:"default_channel"`
	BuildAction        string          `json:"build_action"`
	Commands           json.RawMessage `json:"commands"`
	ArtifactRules      json.RawMessage `json:"artifact_rules"`
	Enabled            bool            `json:"enabled"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type BuildCenterRunAdmin struct {
	ID             string                   `json:"id"`
	ProjectID      string                   `json:"project_id"`
	BuildProfileID string                   `json:"build_profile_id,omitempty"`
	AppBuildID     string                   `json:"app_build_id,omitempty"`
	TriggerType    string                   `json:"trigger_type"`
	TriggerSource  string                   `json:"trigger_source"`
	Action         string                   `json:"action"`
	GitRef         string                   `json:"git_ref"`
	GitCommit      string                   `json:"git_commit"`
	VersionName    string                   `json:"version_name"`
	VersionCode    int                      `json:"version_code"`
	BuildNumber    int                      `json:"build_number"`
	Channel        string                   `json:"channel"`
	Status         string                   `json:"status"`
	ExitCode       int                      `json:"exit_code,omitempty"`
	StartedBy      string                   `json:"started_by"`
	StartedAt      time.Time                `json:"started_at,omitempty"`
	FinishedAt     time.Time                `json:"finished_at,omitempty"`
	DurationMS     int64                    `json:"duration_ms"`
	WorkspaceDir   string                   `json:"workspace_dir,omitempty"`
	ArtifactDir    string                   `json:"artifact_dir,omitempty"`
	LogDir         string                   `json:"log_dir,omitempty"`
	ManifestPath   string                   `json:"manifest_path,omitempty"`
	UploadStatus   string                   `json:"upload_status"`
	ErrorMessage   string                   `json:"error_message,omitempty"`
	Artifacts      []BuildCenterRunArtifact `json:"artifacts,omitempty"`
	Metadata       json.RawMessage          `json:"metadata,omitempty"`
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
}

type BuildCenterRunArtifact struct {
	ID                 string          `json:"id"`
	RunID              string          `json:"run_id"`
	AppBuildArtifactID string          `json:"app_build_artifact_id,omitempty"`
	Name               string          `json:"name"`
	ArtifactType       string          `json:"artifact_type"`
	FileName           string          `json:"file_name"`
	LocalPath          string          `json:"local_path"`
	SizeBytes          int64           `json:"size_bytes"`
	SHA256             string          `json:"sha256"`
	UploadStatus       string          `json:"upload_status"`
	DownloadURL        string          `json:"download_url,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type BuildCenterRunPatch struct {
	Status       string
	ExitCode     int
	GitCommit    string
	VersionName  string
	VersionCode  int
	BuildNumber  int
	ArtifactDir  string
	LogDir       string
	ManifestPath string
	UploadStatus string
	ErrorMessage string
	DurationMS   int64
	Metadata     map[string]any
}

type DeploymentTargetAdmin struct {
	ID                    string          `json:"id"`
	ProjectID             string          `json:"project_id"`
	AppID                 string          `json:"app_id,omitempty"`
	TargetKey             string          `json:"target_key"`
	Name                  string          `json:"name"`
	Provider              string          `json:"provider"`
	Environment           string          `json:"environment"`
	EndpointURL           string          `json:"endpoint_url,omitempty"`
	CloudflareAccountID   string          `json:"cloudflare_account_id,omitempty"`
	CloudflareProjectName string          `json:"cloudflare_project_name,omitempty"`
	CloudflareScriptName  string          `json:"cloudflare_script_name,omitempty"`
	CloudflareBucketName  string          `json:"cloudflare_bucket_name,omitempty"`
	CredentialRef         string          `json:"credential_ref,omitempty"`
	Enabled               bool            `json:"enabled"`
	Metadata              json.RawMessage `json:"metadata,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}
