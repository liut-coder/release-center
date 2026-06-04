package appreleases

import (
	"encoding/json"
	"time"
)

type CreateDeploymentTargetRequest struct {
	ProjectKey            string         `json:"project_key"`
	AppID                 string         `json:"app_id"`
	TargetKey             string         `json:"target_key"`
	Name                  string         `json:"name"`
	Provider              string         `json:"provider"`
	Environment           string         `json:"environment"`
	EndpointURL           string         `json:"endpoint_url"`
	CloudflareAccountID   string         `json:"cloudflare_account_id"`
	CloudflareProjectName string         `json:"cloudflare_project_name"`
	CloudflareScriptName  string         `json:"cloudflare_script_name"`
	CloudflareBucketName  string         `json:"cloudflare_bucket_name"`
	CredentialRef         string         `json:"credential_ref"`
	Enabled               *bool          `json:"enabled,omitempty"`
	Metadata              map[string]any `json:"metadata"`
}

type CreateDeploymentRequest struct {
	ProjectKey           string         `json:"project_key"`
	TargetID             string         `json:"target_id"`
	TargetKey            string         `json:"target_key"`
	RunID                string         `json:"run_id"`
	AppBuildID           string         `json:"app_build_id"`
	AppBuildArtifactID   string         `json:"app_build_artifact_id"`
	ExternalDeploymentID string         `json:"external_deployment_id"`
	ProviderStatus       string         `json:"provider_status"`
	DeploymentURL        string         `json:"deployment_url"`
	VersionName          string         `json:"version_name"`
	BuildNumber          int            `json:"build_number"`
	GitCommit            string         `json:"git_commit"`
	TriggeredBy          string         `json:"triggered_by"`
	LogTail              []string       `json:"log_tail"`
	ErrorMessage         string         `json:"error_message"`
	DryRun               bool           `json:"dry_run"`
	Metadata             map[string]any `json:"metadata"`
}

type UpdateDeploymentStatusRequest struct {
	ExternalDeploymentID string         `json:"external_deployment_id"`
	ProviderStatus       string         `json:"provider_status"`
	DeploymentURL        string         `json:"deployment_url"`
	LogTail              []string       `json:"log_tail"`
	ErrorMessage         string         `json:"error_message"`
	Metadata             map[string]any `json:"metadata"`
}

type RollbackDeploymentRequest struct {
	DryRun      bool           `json:"dry_run"`
	TriggeredBy string         `json:"triggered_by"`
	Reason      string         `json:"reason"`
	Metadata    map[string]any `json:"metadata"`
}

type ApproveDeploymentRequest struct {
	ApprovedBy string         `json:"approved_by"`
	Comment    string         `json:"comment"`
	Metadata   map[string]any `json:"metadata"`
}

type DeploymentRecordAdmin struct {
	ID                   string          `json:"id"`
	TargetID             string          `json:"target_id"`
	TargetKey            string          `json:"target_key,omitempty"`
	TargetName           string          `json:"target_name,omitempty"`
	ProjectID            string          `json:"project_id,omitempty"`
	RunID                string          `json:"run_id,omitempty"`
	AppBuildID           string          `json:"app_build_id,omitempty"`
	AppBuildArtifactID   string          `json:"app_build_artifact_id,omitempty"`
	Provider             string          `json:"provider"`
	Environment          string          `json:"environment,omitempty"`
	ExternalDeploymentID string          `json:"external_deployment_id,omitempty"`
	ProviderStatus       string          `json:"provider_status"`
	DeploymentURL        string          `json:"deployment_url,omitempty"`
	VersionName          string          `json:"version_name"`
	BuildNumber          int             `json:"build_number"`
	GitCommit            string          `json:"git_commit"`
	TriggeredBy          string          `json:"triggered_by"`
	StartedAt            time.Time       `json:"started_at,omitempty"`
	FinishedAt           time.Time       `json:"finished_at,omitempty"`
	DurationMS           int64           `json:"duration_ms"`
	LogTail              []string        `json:"log_tail,omitempty"`
	ErrorMessage         string          `json:"error_message,omitempty"`
	Metadata             json.RawMessage `json:"metadata,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

type DeploymentActionResponse struct {
	OK             bool                   `json:"ok"`
	Target         *DeploymentTargetAdmin `json:"target,omitempty"`
	Record         *DeploymentRecordAdmin `json:"record,omitempty"`
	RollbackSource *DeploymentRecordAdmin `json:"rollback_source,omitempty"`
	RollbackTarget *DeploymentRecordAdmin `json:"rollback_target,omitempty"`
	WorkerTask     *WorkerTaskAdmin       `json:"worker_task,omitempty"`
	MessageZh      string                 `json:"message_zh,omitempty"`
}

type DeploymentRecordsResponse struct {
	DeploymentRecords []DeploymentRecordAdmin `json:"deployment_records"`
	MessageZh         string                  `json:"message_zh,omitempty"`
}
