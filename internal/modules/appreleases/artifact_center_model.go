package appreleases

import (
	"encoding/json"
	"time"
)

type ArtifactCenterItem struct {
	ID                 string          `json:"id"`
	Source             string          `json:"source"`
	ProjectKey         string          `json:"project_key,omitempty"`
	ProjectName        string          `json:"project_name,omitempty"`
	AppKey             string          `json:"app_key,omitempty"`
	AppName            string          `json:"app_name,omitempty"`
	BuildID            string          `json:"build_id,omitempty"`
	RunID              string          `json:"run_id,omitempty"`
	AppBuildArtifactID string          `json:"app_build_artifact_id,omitempty"`
	Name               string          `json:"name"`
	ArtifactType       string          `json:"artifact_type"`
	FileName           string          `json:"file_name,omitempty"`
	Location           string          `json:"location,omitempty"`
	ImmutableRef       string          `json:"immutable_ref,omitempty"`
	SizeBytes          int64           `json:"size_bytes,omitempty"`
	SHA256             string          `json:"sha256,omitempty"`
	UploadStatus       string          `json:"upload_status,omitempty"`
	VersionName        string          `json:"version_name,omitempty"`
	BuildNumber        int             `json:"build_number,omitempty"`
	GitCommit          string          `json:"git_commit,omitempty"`
	Channel            string          `json:"channel,omitempty"`
	Status             string          `json:"status,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

type ArtifactCenterOverview struct {
	Artifacts []ArtifactCenterItem `json:"artifacts"`
	MessageZh string               `json:"message_zh,omitempty"`
}

type ArtifactCenterItemResponse struct {
	OK        bool               `json:"ok"`
	Artifact  ArtifactCenterItem `json:"artifact"`
	MessageZh string             `json:"message_zh,omitempty"`
}
