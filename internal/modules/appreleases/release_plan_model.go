package appreleases

import (
	"encoding/json"
	"time"
)

type ReleasePlanOverview struct {
	Environments []ReleaseEnvironmentAdmin `json:"environments"`
	ReleaseUnits []ReleaseUnitAdmin        `json:"release_units"`
	ReleasePlans []ReleasePlanAdmin        `json:"release_plans"`
	MessageZh    string                    `json:"message_zh,omitempty"`
}

type ReleaseEnvironmentAdmin struct {
	ID               string          `json:"id"`
	EnvironmentKey   string          `json:"environment_key"`
	Name             string          `json:"name"`
	SortOrder        int             `json:"sort_order"`
	RequiresApproval bool            `json:"requires_approval"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type ReleaseUnitAdmin struct {
	ID             string          `json:"id"`
	ProjectID      string          `json:"project_id"`
	ProjectKey     string          `json:"project_key,omitempty"`
	AppID          string          `json:"app_id,omitempty"`
	UnitKey        string          `json:"unit_key"`
	Name           string          `json:"name"`
	UnitType       string          `json:"unit_type"`
	DefaultChannel string          `json:"default_channel"`
	Enabled        bool            `json:"enabled"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type ReleasePlanArtifactAdmin struct {
	ID                 string          `json:"id"`
	ReleasePlanID      string          `json:"release_plan_id"`
	BuildRunID         string          `json:"build_run_id,omitempty"`
	AppBuildID         string          `json:"app_build_id,omitempty"`
	AppBuildArtifactID string          `json:"app_build_artifact_id,omitempty"`
	ArtifactName       string          `json:"artifact_name"`
	ArtifactType       string          `json:"artifact_type"`
	FileName           string          `json:"file_name"`
	ImmutableRef       string          `json:"immutable_ref"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type ReleasePlanAdmin struct {
	ID                string                     `json:"id"`
	ProjectID         string                     `json:"project_id"`
	ProjectKey        string                     `json:"project_key,omitempty"`
	ReleaseUnitID     string                     `json:"release_unit_id"`
	UnitKey           string                     `json:"unit_key,omitempty"`
	UnitType          string                     `json:"unit_type,omitempty"`
	EnvironmentID     string                     `json:"environment_id"`
	EnvironmentKey    string                     `json:"environment_key,omitempty"`
	PlanKey           string                     `json:"plan_key"`
	Title             string                     `json:"title"`
	Description       string                     `json:"description"`
	VersionName       string                     `json:"version_name"`
	BuildNumber       int                        `json:"build_number"`
	GitCommit         string                     `json:"git_commit"`
	Channel           string                     `json:"channel"`
	Status            string                     `json:"status"`
	RolloutPercentage int                        `json:"rollout_percentage"`
	TargetType        string                     `json:"target_type"`
	TargetValue       string                     `json:"target_value"`
	ScheduledAt       *time.Time                 `json:"scheduled_at,omitempty"`
	PublishedAt       *time.Time                 `json:"published_at,omitempty"`
	PausedAt          *time.Time                 `json:"paused_at,omitempty"`
	ApprovedBy        string                     `json:"approved_by,omitempty"`
	Metadata          json.RawMessage            `json:"metadata,omitempty"`
	CreatedBy         string                     `json:"created_by"`
	Artifacts         []ReleasePlanArtifactAdmin `json:"artifacts,omitempty"`
	CreatedAt         time.Time                  `json:"created_at"`
	UpdatedAt         time.Time                  `json:"updated_at"`
}

type CreateReleaseUnitRequest struct {
	ProjectKey     string         `json:"project_key"`
	AppID          string         `json:"app_id"`
	UnitKey        string         `json:"unit_key"`
	Name           string         `json:"name"`
	UnitType       string         `json:"unit_type"`
	DefaultChannel string         `json:"default_channel"`
	Enabled        *bool          `json:"enabled,omitempty"`
	Metadata       map[string]any `json:"metadata"`
}

type CreateReleasePlanRequest struct {
	ProjectKey        string                       `json:"project_key"`
	UnitKey           string                       `json:"unit_key"`
	EnvironmentKey    string                       `json:"environment_key"`
	PlanKey           string                       `json:"plan_key"`
	Title             string                       `json:"title"`
	Description       string                       `json:"description"`
	VersionName       string                       `json:"version_name"`
	BuildNumber       int                          `json:"build_number"`
	GitCommit         string                       `json:"git_commit"`
	Channel           string                       `json:"channel"`
	Status            string                       `json:"status"`
	RolloutPercentage int                          `json:"rollout_percentage"`
	TargetType        string                       `json:"target_type"`
	TargetValue       string                       `json:"target_value"`
	ScheduledAt       string                       `json:"scheduled_at"`
	ApprovedBy        string                       `json:"approved_by"`
	CreatedBy         string                       `json:"created_by"`
	Artifacts         []ReleasePlanArtifactRequest `json:"artifacts"`
	Metadata          map[string]any               `json:"metadata"`
}

type ReleasePlanArtifactRequest struct {
	BuildRunID         string         `json:"build_run_id"`
	AppBuildID         string         `json:"app_build_id"`
	AppBuildArtifactID string         `json:"app_build_artifact_id"`
	ArtifactName       string         `json:"artifact_name"`
	ArtifactType       string         `json:"artifact_type"`
	FileName           string         `json:"file_name"`
	ImmutableRef       string         `json:"immutable_ref"`
	Metadata           map[string]any `json:"metadata"`
}

type ReleasePlanActionRequest struct {
	ApprovedBy string         `json:"approved_by"`
	Metadata   map[string]any `json:"metadata"`
}

type CreateReleasePlanDeploymentRequest struct {
	TargetID      string         `json:"target_id"`
	TargetKey     string         `json:"target_key"`
	DryRun        bool           `json:"dry_run"`
	TriggeredBy   string         `json:"triggered_by"`
	DeploymentURL string         `json:"deployment_url"`
	Metadata      map[string]any `json:"metadata"`
}

type ReleasePlanActionResponse struct {
	OK        bool             `json:"ok"`
	Plan      ReleasePlanAdmin `json:"plan"`
	MessageZh string           `json:"message_zh,omitempty"`
}

type ReleasePlanDeploymentResponse struct {
	OK                bool                    `json:"ok"`
	Plan              ReleasePlanAdmin        `json:"plan"`
	DeploymentRecords []DeploymentRecordAdmin `json:"deployment_records"`
	WorkerTasks       []WorkerTaskAdmin       `json:"worker_tasks,omitempty"`
	MessageZh         string                  `json:"message_zh,omitempty"`
}
