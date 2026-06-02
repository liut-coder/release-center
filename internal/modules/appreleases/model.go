package appreleases

import "time"

type CheckRequest struct {
	AppKey              string `json:"appKey"`
	AppKeySnake         string `json:"app_key"`
	DeviceID            string `json:"deviceId"`
	DeviceIDSnake       string `json:"device_id"`
	DeviceKey           string `json:"deviceKey"`
	DeviceKeySnake      string `json:"device_key"`
	UserID              any    `json:"userId"`
	Platform            string `json:"platform"`
	OSVersion           string `json:"osVersion"`
	OSVersionSnake      string `json:"os_version"`
	DeviceModel         string `json:"deviceModel"`
	DeviceModelSnake    string `json:"device_model"`
	PackageName         string `json:"package_name"`
	PackageNameCamel    string `json:"packageName"`
	VersionCode         int    `json:"versionCode"`
	VersionCodeSnake    int    `json:"version_code"`
	AppVersionCode      int    `json:"appVersionCode"`
	AppVersionCodeSnake int    `json:"app_version_code"`
	VersionName         string `json:"versionName"`
	VersionNameSnake    string `json:"version_name"`
	BuildNumber         int    `json:"buildNumber"`
	BuildNumberSnake    int    `json:"build_number"`
	Channel             string `json:"channel"`
	BuildType           string `json:"build_type"`
	BuildTypeCamel      string `json:"buildType"`
}

type CheckResponse struct {
	HasUpdate               bool   `json:"has_update"`
	UpdateAvailable         bool   `json:"update_available"`
	UpdateLevel             string `json:"update_level,omitempty"`
	BlockTaskExecution      bool   `json:"block_task_execution"`
	VersionName             string `json:"version_name,omitempty"`
	LatestVersionName       string `json:"latest_version_name,omitempty"`
	VersionCode             int    `json:"version_code,omitempty"`
	LatestVersionCode       int    `json:"latest_version_code,omitempty"`
	BuildNumber             int    `json:"build_number,omitempty"`
	Channel                 string `json:"channel,omitempty"`
	BuildType               string `json:"build_type,omitempty"`
	APKURL                  string `json:"apk_url,omitempty"`
	DownloadURL             string `json:"download_url,omitempty"`
	SHA256                  string `json:"sha256,omitempty"`
	SizeBytes               int64  `json:"size_bytes,omitempty"`
	ForceUpdate             bool   `json:"force_update"`
	CurrentVersionAvailable bool   `json:"current_version_available"`
	MinSupportedVersionCode int    `json:"min_supported_version_code,omitempty"`
	ReleaseNotes            string `json:"release_notes,omitempty"`
	MessageZh               string `json:"message_zh,omitempty"`
	UnavailableReason       string `json:"unavailable_reason,omitempty"`
	FileName                string `json:"file_name,omitempty"`
	Release                 any    `json:"release,omitempty"`
}

type ResourceCheckRequest struct {
	AppKey               string `json:"appKey"`
	AppKeySnake          string `json:"app_key"`
	DeviceID             string `json:"deviceId"`
	DeviceIDSnake        string `json:"device_id"`
	DeviceKey            string `json:"deviceKey"`
	DeviceKeySnake       string `json:"device_key"`
	UserID               any    `json:"userId"`
	AppVersionCode       int    `json:"appVersionCode"`
	AppVersionCodeSnake  int    `json:"app_version_code"`
	VersionCode          int    `json:"versionCode"`
	VersionCodeSnake     int    `json:"version_code"`
	ResourceVersion      string `json:"resourceVersion"`
	ResourceVersionSnake string `json:"resource_version"`
	Channel              string `json:"channel"`
	PackageName          string `json:"package_name"`
	PackageNameCamel     string `json:"packageName"`
	CurrentResourceVer   string `json:"current_resource_version"`
}

type ResourceCheckResponse struct {
	HasUpdate             bool              `json:"hasUpdate"`
	HasUpdateSnake        bool              `json:"has_update"`
	UpdateLevel           string            `json:"updateLevel,omitempty"`
	LatestResourceVersion string            `json:"latestResourceVersion,omitempty"`
	Title                 string            `json:"title,omitempty"`
	Summary               string            `json:"summary,omitempty"`
	ReleaseNotesMarkdown  string            `json:"releaseNotesMarkdown,omitempty"`
	ManifestURL           string            `json:"manifestUrl,omitempty"`
	TotalSize             int64             `json:"totalSize"`
	ForceUpdate           bool              `json:"forceUpdate"`
	BlockTaskExecution    bool              `json:"blockTaskExecution"`
	MessageZh             string            `json:"message_zh,omitempty"`
	Packages              []ResourcePackage `json:"packages"`
}

type ResourcePackage struct {
	PackageKey  string `json:"packageKey"`
	PackageType string `json:"packageType,omitempty"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
}

type UpdateEventRequest struct {
	AppKey          string         `json:"appKey"`
	DeviceID        string         `json:"deviceId"`
	DeviceKey       string         `json:"deviceKey"`
	UserID          any            `json:"userId"`
	EventType       string         `json:"eventType"`
	FromVersion     string         `json:"fromVersion"`
	ToVersion       string         `json:"toVersion"`
	FromVersionCode int            `json:"fromVersionCode"`
	ToVersionCode   int            `json:"toVersionCode"`
	PackageKey      string         `json:"packageKey"`
	ErrorMessage    string         `json:"errorMessage"`
	PackageName     string         `json:"package_name"`
	Metadata        map[string]any `json:"metadata"`
}

type UpdateEventResponse struct {
	Status    string `json:"status"`
	MessageZh string `json:"message_zh"`
}

type HeartbeatRequest struct {
	AppKey               string `json:"appKey"`
	AppKeySnake          string `json:"app_key"`
	DeviceID             string `json:"deviceId"`
	DeviceIDSnake        string `json:"device_id"`
	DeviceKey            string `json:"deviceKey"`
	DeviceKeySnake       string `json:"device_key"`
	UserID               any    `json:"userId"`
	Platform             string `json:"platform"`
	OSVersion            string `json:"osVersion"`
	OSVersionSnake       string `json:"os_version"`
	DeviceModel          string `json:"deviceModel"`
	DeviceModelSnake     string `json:"device_model"`
	DeviceName           string `json:"deviceName"`
	DeviceNameSnake      string `json:"device_name"`
	PackageName          string `json:"package_name"`
	PackageNameCamel     string `json:"packageName"`
	VersionName          string `json:"versionName"`
	VersionNameSnake     string `json:"version_name"`
	VersionCode          int    `json:"versionCode"`
	VersionCodeSnake     int    `json:"version_code"`
	AppVersionCode       int    `json:"appVersionCode"`
	AppVersionCodeSnake  int    `json:"app_version_code"`
	BuildNumber          int    `json:"buildNumber"`
	BuildNumberSnake     int    `json:"build_number"`
	ResourceVersion      string `json:"resourceVersion"`
	ResourceVersionSnake string `json:"resource_version"`
}

type HeartbeatResponse struct {
	Status    string `json:"status"`
	MessageZh string `json:"message_zh"`
}

type TaskPreflightRequest struct {
	AppKey               string `json:"appKey"`
	AppKeySnake          string `json:"app_key"`
	DeviceID             string `json:"deviceId"`
	DeviceIDSnake        string `json:"device_id"`
	DeviceKey            string `json:"deviceKey"`
	DeviceKeySnake       string `json:"device_key"`
	UserID               any    `json:"userId"`
	PackageName          string `json:"package_name"`
	PackageNameCamel     string `json:"packageName"`
	VersionName          string `json:"versionName"`
	VersionNameSnake     string `json:"version_name"`
	VersionCode          int    `json:"versionCode"`
	VersionCodeSnake     int    `json:"version_code"`
	AppVersionCode       int    `json:"appVersionCode"`
	AppVersionCodeSnake  int    `json:"app_version_code"`
	BuildNumber          int    `json:"buildNumber"`
	BuildNumberSnake     int    `json:"build_number"`
	Channel              string `json:"channel"`
	BuildType            string `json:"build_type"`
	BuildTypeCamel       string `json:"buildType"`
	ResourceVersion      string `json:"resourceVersion"`
	ResourceVersionSnake string `json:"resource_version"`
	CurrentResourceVer   string `json:"current_resource_version"`
}

type TaskPreflightResponse struct {
	CanExecute        bool                  `json:"can_execute"`
	BlockReason       string                `json:"block_reason,omitempty"`
	MessageZh         string                `json:"message_zh"`
	AppUpdate         CheckResponse         `json:"app_update"`
	ResourceUpdate    ResourceCheckResponse `json:"resource_update"`
	AppUpdateRequired bool                  `json:"app_update_required"`
	ResourceRequired  bool                  `json:"resource_required"`
}

type ReleaseOverview struct {
	Apps             []AppSummary          `json:"apps"`
	Releases         []ReleaseSummary      `json:"releases"`
	ResourceVersions []ResourceSummary     `json:"resource_versions"`
	Installations    []InstallationSummary `json:"installations"`
	MessageZh        string                `json:"message_zh"`
}

type AdminOverview struct {
	Apps                 []AppAdminSummary         `json:"apps"`
	Releases             []AppReleaseAdmin         `json:"releases"`
	Latest               *AppReleaseAdmin          `json:"latest,omitempty"`
	BuildJobs            []AppBuildJob             `json:"build_jobs"`
	WebhookEvents        []WebhookEventAdmin       `json:"webhook_events"`
	ResourceVersions     []AppResourceVersionAdmin `json:"resource_versions"`
	LatestResource       *AppResourceVersionAdmin  `json:"latest_resource,omitempty"`
	Installations        []AppInstallationAdmin    `json:"installations"`
	UpgradeEvents        []AppUpgradeEventAdmin    `json:"upgrade_events"`
	ResourceUpdateEvents []AppUpgradeEventAdmin    `json:"resource_update_events"`
	QualityMetrics       []ReleaseQualityMetric    `json:"quality_metrics"`
	QualityPolicy        QualityPolicy             `json:"quality_policy"`
	QualityAutomation    QualityAutomationPolicy   `json:"quality_automation"`
	QualityAlerts        []QualityAlert            `json:"quality_alerts"`
	AuditLogs            []AppReleaseAuditLogAdmin `json:"audit_logs"`
	MessageZh            string                    `json:"message_zh"`
}

type AppAdminSummary struct {
	ID          string    `json:"id"`
	AppKey      string    `json:"app_key"`
	Name        string    `json:"name"`
	Platform    string    `json:"platform"`
	PackageName string    `json:"package_name"`
	Description string    `json:"description,omitempty"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AppBuildJob struct {
	ID               string    `json:"id"`
	Status           string    `json:"status"`
	AppID            string    `json:"app_id,omitempty"`
	GitRef           string    `json:"git_ref"`
	GitCommit        string    `json:"git_commit,omitempty"`
	GitBranch        string    `json:"git_branch,omitempty"`
	BuildType        string    `json:"build_type"`
	Channel          string    `json:"channel"`
	VersionName      string    `json:"version_name"`
	VersionCode      int       `json:"version_code"`
	BuildNumber      int       `json:"build_number"`
	BuildEnvironment string    `json:"build_environment,omitempty"`
	ArtifactType     string    `json:"artifact_type,omitempty"`
	ArtifactPath     string    `json:"artifact_path,omitempty"`
	ArtifactSize     int64     `json:"artifact_size,omitempty"`
	SHA256           string    `json:"sha256,omitempty"`
	FileName         string    `json:"file_name,omitempty"`
	StorageKey       string    `json:"-"`
	APIBaseURL       string    `json:"api_base_url,omitempty"`
	ReleaseNotes     string    `json:"release_notes,omitempty"`
	StartedBy        string    `json:"started_by,omitempty"`
	StartedAt        time.Time `json:"started_at,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	FinishedAt       time.Time `json:"finished_at,omitempty"`
	DurationMS       int       `json:"duration_ms,omitempty"`
	LogTail          []string  `json:"log_tail,omitempty"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	ReleaseID        string    `json:"release_id,omitempty"`
}

type AppReleaseAdmin struct {
	ID                   string    `json:"id"`
	ReleaseID            string    `json:"release_id,omitempty"`
	AppID                string    `json:"app_id,omitempty"`
	BuildID              string    `json:"build_id,omitempty"`
	PackageName          string    `json:"package_name"`
	VersionName          string    `json:"version_name"`
	VersionCode          int       `json:"version_code"`
	BuildNumber          int       `json:"build_number,omitempty"`
	Channel              string    `json:"channel"`
	BuildType            string    `json:"build_type"`
	Signed               bool      `json:"signed,omitempty"`
	GitRef               string    `json:"git_ref"`
	GitCommit            string    `json:"git_commit"`
	APIBaseURL           string    `json:"api_base_url,omitempty"`
	APKPath              string    `json:"apk_path,omitempty"`
	ArtifactPath         string    `json:"artifact_path,omitempty"`
	FileName             string    `json:"file_name"`
	SizeBytes            int64     `json:"size_bytes"`
	SHA256               string    `json:"sha256"`
	Status               string    `json:"status"`
	Title                string    `json:"title"`
	Summary              string    `json:"summary,omitempty"`
	ReleaseNotes         string    `json:"release_notes,omitempty"`
	ReleaseNotesMarkdown string    `json:"release_notes_markdown,omitempty"`
	UpgradeMessage       string    `json:"upgrade_message,omitempty"`
	UpdateLevel          string    `json:"update_level,omitempty"`
	RolloutPercentage    int       `json:"rollout_percentage,omitempty"`
	TargetType           string    `json:"target_type,omitempty"`
	TargetValue          string    `json:"target_value,omitempty"`
	MinSupportedCode     int       `json:"min_supported_code,omitempty"`
	BlockOldVersions     bool      `json:"block_old_versions,omitempty"`
	ScheduledAt          time.Time `json:"scheduled_at,omitempty"`
	IsLatest             bool      `json:"is_latest"`
	IsPublished          bool      `json:"is_published"`
	ForceUpdate          bool      `json:"force_update,omitempty"`
	PublishedAt          time.Time `json:"published_at,omitempty"`
	PausedAt             time.Time `json:"paused_at,omitempty"`
	CreatedBy            string    `json:"created_by,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	DownloadURL          string    `json:"download_url,omitempty"`
}

type AppResourcePackageAdmin struct {
	ID          string    `json:"id,omitempty"`
	PackageKey  string    `json:"package_key"`
	PackageType string    `json:"package_type,omitempty"`
	FileURL     string    `json:"file_url"`
	FileSize    int64     `json:"file_size"`
	SHA256      string    `json:"sha256"`
	StorageKey  string    `json:"-"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

type AppResourceVersionAdmin struct {
	ID                   string                    `json:"id"`
	AppID                string                    `json:"app_id,omitempty"`
	ResourceVersion      string                    `json:"resource_version"`
	Channel              string                    `json:"channel"`
	Status               string                    `json:"status"`
	MinAppVersionCode    int                       `json:"min_app_version_code,omitempty"`
	MaxAppVersionCode    int                       `json:"max_app_version_code,omitempty"`
	UpdateLevel          string                    `json:"update_level,omitempty"`
	Title                string                    `json:"title"`
	Summary              string                    `json:"summary,omitempty"`
	ReleaseNotesMarkdown string                    `json:"release_notes_markdown,omitempty"`
	RolloutPercentage    int                       `json:"rollout_percentage,omitempty"`
	ManifestURL          string                    `json:"manifest_url"`
	ManifestSigned       bool                      `json:"manifest_signed,omitempty"`
	SignatureAlgorithm   string                    `json:"signature_algorithm,omitempty"`
	TotalSize            int64                     `json:"total_size"`
	Packages             []AppResourcePackageAdmin `json:"packages,omitempty"`
	PublishedAt          time.Time                 `json:"published_at,omitempty"`
	PausedAt             time.Time                 `json:"paused_at,omitempty"`
	CreatedBy            string                    `json:"created_by,omitempty"`
	CreatedAt            time.Time                 `json:"created_at"`
	UpdatedAt            time.Time                 `json:"updated_at"`
}

type AppInstallationAdmin struct {
	ID                string    `json:"id"`
	AppID             string    `json:"app_id,omitempty"`
	UserID            string    `json:"user_id,omitempty"`
	DeviceID          string    `json:"device_id"`
	DeviceName        string    `json:"device_name,omitempty"`
	Platform          string    `json:"platform,omitempty"`
	OSVersion         string    `json:"os_version,omitempty"`
	DeviceModel       string    `json:"device_model,omitempty"`
	InstalledVersion  string    `json:"installed_version,omitempty"`
	InstalledCode     int       `json:"installed_code,omitempty"`
	BuildNumber       int       `json:"build_number,omitempty"`
	ResourceVersion   string    `json:"resource_version,omitempty"`
	LastSeenAt        time.Time `json:"last_seen_at,omitempty"`
	LastUpgradeAt     time.Time `json:"last_upgrade_at,omitempty"`
	LastUpgradeStatus string    `json:"last_upgrade_status,omitempty"`
	LastError         string    `json:"last_error,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type AppUpgradeEventAdmin struct {
	ID              string    `json:"id"`
	AppID           string    `json:"app_id,omitempty"`
	ReleaseID       string    `json:"release_id,omitempty"`
	UserID          string    `json:"user_id,omitempty"`
	DeviceID        string    `json:"device_id"`
	FromVersionCode int       `json:"from_version_code,omitempty"`
	ToVersionCode   int       `json:"to_version_code,omitempty"`
	FromVersion     string    `json:"from_version,omitempty"`
	ToVersion       string    `json:"to_version,omitempty"`
	EventType       string    `json:"event_type"`
	PackageKey      string    `json:"package_key,omitempty"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	Category        string    `json:"category,omitempty"`
}

type ReleaseQualityMetric struct {
	Category            string               `json:"category"`
	TotalEvents         int                  `json:"total_events"`
	SuccessEvents       int                  `json:"success_events"`
	FailureEvents       int                  `json:"failure_events"`
	SuccessRate         int                  `json:"success_rate"`
	FailureRate         int                  `json:"failure_rate"`
	LatestFailureReason string               `json:"latest_failure_reason,omitempty"`
	RecommendedAction   string               `json:"recommended_action"`
	ActionReason        string               `json:"action_reason,omitempty"`
	PolicyThreshold     string               `json:"policy_threshold,omitempty"`
	FailureReasons      []QualityReasonCount `json:"failure_reasons,omitempty"`
}

type QualityReasonCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type QualityPolicy struct {
	ResourceActivationFailedCount int `json:"resource_activation_failed_count"`
	ResourceFailureRate           int `json:"resource_failure_rate"`
	APKChecksumFailedCount        int `json:"apk_checksum_failed_count"`
	APKInstallFailedCount         int `json:"apk_install_failed_count"`
	APKInstallFailureRate         int `json:"apk_install_failure_rate"`
}

type QualityAutomationPolicy struct {
	AlertWebhookConfigured bool `json:"alert_webhook_configured"`
	AutoPauseResource      bool `json:"auto_pause_resource"`
	AutoRollbackAPK        bool `json:"auto_rollback_apk"`
}

type QualityAlert struct {
	ID                 string    `json:"id"`
	Category           string    `json:"category"`
	Severity           string    `json:"severity"`
	RecommendedAction  string    `json:"recommended_action"`
	Reason             string    `json:"reason"`
	Threshold          string    `json:"threshold"`
	FailureRate        int       `json:"failure_rate"`
	FailureEvents      int       `json:"failure_events"`
	LatestFailure      string    `json:"latest_failure,omitempty"`
	NotificationStatus string    `json:"notification_status,omitempty"`
	AutomationStatus   string    `json:"automation_status,omitempty"`
	AutomationAction   string    `json:"automation_action,omitempty"`
	AutomationTargetID string    `json:"automation_target_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

type AppReleaseAuditLogAdmin struct {
	ID         string    `json:"id"`
	AppID      string    `json:"app_id,omitempty"`
	Operator   string    `json:"operator"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id,omitempty"`
	BeforeJSON any       `json:"before_json,omitempty"`
	AfterJSON  any       `json:"after_json,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateBuildRequest struct {
	GitRef       string `json:"git_ref"`
	GitCommit    string `json:"git_commit"`
	GitBranch    string `json:"git_branch"`
	BuildType    string `json:"build_type"`
	Channel      string `json:"channel"`
	VersionName  string `json:"version_name"`
	VersionCode  int    `json:"version_code"`
	BuildNumber  int    `json:"build_number"`
	Status       string `json:"status"`
	APIBaseURL   string `json:"api_base_url"`
	ReleaseNotes string `json:"release_notes"`
	ArtifactType string `json:"artifact_type"`
	ArtifactURL  string `json:"artifact_url"`
	FileName     string `json:"file_name"`
	BuildURL     string `json:"build_url"`
	Provider     string `json:"provider"`
	Workflow     string `json:"workflow"`
	RunID        string `json:"run_id"`
	StartedBy    string `json:"started_by"`
	APKFileName  string `json:"-"`
	ArtifactPath string `json:"-"`
	StorageKey   string `json:"-"`
	ArtifactSize int64  `json:"artifact_size"`
	SHA256       string `json:"sha256"`
}

type CreateArtifactRequest struct {
	AppKey       string `json:"app_key"`
	GitRef       string `json:"git_ref"`
	GitCommit    string `json:"git_commit"`
	GitBranch    string `json:"git_branch"`
	BuildType    string `json:"build_type"`
	Channel      string `json:"channel"`
	VersionName  string `json:"version_name"`
	VersionCode  int    `json:"version_code"`
	BuildNumber  int    `json:"build_number"`
	ArtifactType string `json:"artifact_type"`
	ArtifactURL  string `json:"artifact_url"`
	FileName     string `json:"file_name"`
	SizeBytes    int64  `json:"size_bytes"`
	SHA256       string `json:"sha256"`
	Provider     string `json:"provider"`
	Workflow     string `json:"workflow"`
	RunID        string `json:"run_id"`
	BuildURL     string `json:"build_url"`
	ReleaseNotes string `json:"release_notes"`
}

type CreateAppRequest struct {
	AppKey      string `json:"app_key"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	PackageName string `json:"package_name"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

type WebhookEventRequest struct {
	Provider   string         `json:"provider"`
	EventType  string         `json:"event_type"`
	DeliveryID string         `json:"delivery_id"`
	Repository string         `json:"repository"`
	Ref        string         `json:"ref"`
	CommitSHA  string         `json:"commit_sha"`
	Sender     string         `json:"sender"`
	Action     string         `json:"action"`
	Workflow   string         `json:"workflow"`
	RunID      string         `json:"run_id"`
	RawPayload map[string]any `json:"raw_payload"`
	ReceivedAt time.Time      `json:"received_at,omitempty"`
}

type WebhookEventAdmin struct {
	ID         string    `json:"id"`
	Provider   string    `json:"provider"`
	EventType  string    `json:"event_type"`
	DeliveryID string    `json:"delivery_id,omitempty"`
	Repository string    `json:"repository,omitempty"`
	Ref        string    `json:"ref,omitempty"`
	CommitSHA  string    `json:"commit_sha,omitempty"`
	Sender     string    `json:"sender,omitempty"`
	Action     string    `json:"action,omitempty"`
	Workflow   string    `json:"workflow,omitempty"`
	RunID      string    `json:"run_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type WebhookEventResponse struct {
	OK        bool              `json:"ok"`
	Event     WebhookEventAdmin `json:"event"`
	MessageZh string            `json:"message_zh,omitempty"`
}

type CreateReleaseRequest struct {
	BuildID              string `json:"build_id"`
	Channel              string `json:"channel"`
	Title                string `json:"title"`
	Summary              string `json:"summary"`
	ReleaseNotesMarkdown string `json:"release_notes_markdown"`
	UpdateLevel          string `json:"update_level"`
	RolloutPercentage    int    `json:"rollout_percentage"`
	MinSupportedCode     int    `json:"min_supported_code"`
	BlockOldVersions     bool   `json:"block_old_versions"`
	TargetType           string `json:"target_type"`
	TargetValue          string `json:"target_value"`
	ScheduledAt          string `json:"scheduled_at"`
}

type UpdateRolloutRequest struct {
	RolloutPercentage int `json:"rollout_percentage"`
}

type UpdateNotesRequest struct {
	Title                string `json:"title"`
	Summary              string `json:"summary"`
	ReleaseNotesMarkdown string `json:"release_notes_markdown"`
}

type CreateResourceVersionRequest struct {
	ResourceVersion      string `json:"resource_version"`
	Channel              string `json:"channel"`
	Title                string `json:"title"`
	Summary              string `json:"summary"`
	ReleaseNotesMarkdown string `json:"release_notes_markdown"`
	MinAppVersionCode    int    `json:"min_app_version_code"`
	MaxAppVersionCode    int    `json:"max_app_version_code"`
	UpdateLevel          string `json:"update_level"`
	RolloutPercentage    int    `json:"rollout_percentage"`
}

type UploadedResourcePackage struct {
	PackageKey string
	FileName   string
	Content    []byte
}

type AdminActionResponse struct {
	OK        bool            `json:"ok"`
	Release   AppReleaseAdmin `json:"release"`
	MessageZh string          `json:"message_zh,omitempty"`
}

type BuildActionResponse struct {
	Job       AppBuildJob `json:"job"`
	MessageZh string      `json:"message_zh,omitempty"`
}

type AppActionResponse struct {
	OK        bool            `json:"ok"`
	App       AppAdminSummary `json:"app"`
	MessageZh string          `json:"message_zh,omitempty"`
}

type SystemManagementOverview struct {
	Users        []SystemUserAdmin       `json:"users"`
	Roles        []SystemRoleAdmin       `json:"roles"`
	Permissions  []SystemPermissionAdmin `json:"permissions"`
	Dictionaries []SystemDictionaryAdmin `json:"dictionaries"`
	Menus        []SystemMenuAdmin       `json:"menus"`
	MessageZh    string                  `json:"message_zh,omitempty"`
}

type SystemUserAdmin struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Account     string    `json:"account"`
	Role        string    `json:"role"`
	RoleCode    string    `json:"role_code,omitempty"`
	Department  string    `json:"department,omitempty"`
	Status      string    `json:"status"`
	LastLogin   string    `json:"last_login,omitempty"`
	LastLoginAt time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type SystemRoleAdmin struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Users       int       `json:"users"`
	Scope       string    `json:"scope"`
	Permissions []string  `json:"permissions"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type SystemPermissionAdmin struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Module    string    `json:"module"`
	Type      string    `json:"type"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type SystemDictionaryAdmin struct {
	ID        string    `json:"id"`
	Group     string    `json:"group"`
	Key       string    `json:"key"`
	Label     string    `json:"label"`
	Value     string    `json:"value"`
	Sort      int       `json:"sort"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type SystemMenuAdmin struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Path      string    `json:"path"`
	Icon      string    `json:"icon"`
	Parent    string    `json:"parent"`
	Sort      int       `json:"sort"`
	Visible   bool      `json:"visible"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type CreateSystemUserRequest struct {
	Name       string `json:"name"`
	Account    string `json:"account"`
	RoleCode   string `json:"role_code"`
	Role       string `json:"role"`
	Department string `json:"department"`
	Status     string `json:"status"`
}

type CreateSystemRoleRequest struct {
	Name        string   `json:"name"`
	Code        string   `json:"code"`
	Scope       string   `json:"scope"`
	Permissions []string `json:"permissions"`
	Enabled     *bool    `json:"enabled,omitempty"`
}

type CreateSystemPermissionRequest struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	Module  string `json:"module"`
	Type    string `json:"type"`
	Enabled *bool  `json:"enabled,omitempty"`
}

type CreateSystemDictionaryRequest struct {
	Group   string `json:"group"`
	Key     string `json:"key"`
	Label   string `json:"label"`
	Value   string `json:"value"`
	Sort    int    `json:"sort"`
	Enabled *bool  `json:"enabled,omitempty"`
}

type CreateSystemMenuRequest struct {
	Title   string `json:"title"`
	Path    string `json:"path"`
	Icon    string `json:"icon"`
	Parent  string `json:"parent"`
	Sort    int    `json:"sort"`
	Visible *bool  `json:"visible,omitempty"`
}

type SystemUserActionResponse struct {
	OK        bool            `json:"ok"`
	User      SystemUserAdmin `json:"user"`
	MessageZh string          `json:"message_zh,omitempty"`
}

type SystemRoleActionResponse struct {
	OK        bool            `json:"ok"`
	Role      SystemRoleAdmin `json:"role"`
	MessageZh string          `json:"message_zh,omitempty"`
}

type SystemPermissionActionResponse struct {
	OK         bool                  `json:"ok"`
	Permission SystemPermissionAdmin `json:"permission"`
	MessageZh  string                `json:"message_zh,omitempty"`
}

type SystemDictionaryActionResponse struct {
	OK         bool                  `json:"ok"`
	Dictionary SystemDictionaryAdmin `json:"dictionary"`
	MessageZh  string                `json:"message_zh,omitempty"`
}

type SystemMenuActionResponse struct {
	OK        bool            `json:"ok"`
	Menu      SystemMenuAdmin `json:"menu"`
	MessageZh string          `json:"message_zh,omitempty"`
}

type ResourceActionResponse struct {
	OK              bool                    `json:"ok"`
	ResourceVersion AppResourceVersionAdmin `json:"resource_version"`
	MessageZh       string                  `json:"message_zh,omitempty"`
}

type AppSummary struct {
	AppID       string    `json:"app_id"`
	AppKey      string    `json:"app_key"`
	Name        string    `json:"name"`
	Platform    string    `json:"platform"`
	PackageName string    `json:"package_name"`
	Enabled     bool      `json:"enabled"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ReleaseSummary struct {
	ReleaseID               string    `json:"release_id"`
	AppKey                  string    `json:"app_key"`
	VersionName             string    `json:"version_name"`
	VersionCode             int       `json:"version_code"`
	BuildNumber             int       `json:"build_number"`
	Channel                 string    `json:"channel"`
	BuildType               string    `json:"build_type"`
	Status                  string    `json:"status"`
	UpdateLevel             string    `json:"update_level"`
	MinSupportedVersionCode int       `json:"min_supported_version_code"`
	DownloadURL             string    `json:"download_url"`
	SHA256                  string    `json:"sha256"`
	SizeBytes               int64     `json:"size_bytes"`
	PublishedAt             time.Time `json:"published_at"`
}

type ResourceSummary struct {
	ResourceVersion string    `json:"resource_version"`
	Channel         string    `json:"channel"`
	Status          string    `json:"status"`
	UpdateLevel     string    `json:"update_level"`
	ManifestURL     string    `json:"manifest_url"`
	TotalSize       int64     `json:"total_size"`
	PublishedAt     time.Time `json:"published_at"`
}

type InstallationSummary struct {
	DeviceID          string    `json:"device_id"`
	DeviceKey         string    `json:"device_key"`
	DeviceName        string    `json:"device_name"`
	InstalledVersion  string    `json:"installed_version"`
	InstalledCode     int       `json:"installed_code"`
	BuildNumber       int       `json:"build_number"`
	ResourceVersion   string    `json:"resource_version"`
	LastSeenAt        time.Time `json:"last_seen_at"`
	LastUpgradeStatus string    `json:"last_upgrade_status"`
	LastError         string    `json:"last_error"`
}

type Config struct {
	AppKey                   string
	Name                     string
	PackageName              string
	LatestVersionName        string
	LatestVersionCode        int
	BuildNumber              int
	Channel                  string
	BuildType                string
	APKURL                   string
	DownloadURL              string
	SHA256                   string
	SizeBytes                int64
	FileName                 string
	ForceUpdate              bool
	CurrentVersionAvailable  bool
	MinSupportedVersionCode  int
	ReleaseNotes             string
	MessageZh                string
	UnavailableReason        string
	ManifestPrivateKey       string
	QualityPolicy            QualityPolicy
	QualityAlertWebhookURL   string
	QualityAutoPauseResource bool
	QualityAutoRollbackAPK   bool
}
