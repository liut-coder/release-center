package appreleases

import "context"

type Store interface {
	EnsureConfiguredRelease(ctx context.Context, cfg Config) error
	FindLatestRelease(ctx context.Context, req CheckRequest, fallback Config) (Config, error)
	UpsertInstallation(ctx context.Context, cfg Config, req CheckRequest) error
	FindLatestResource(ctx context.Context, req ResourceCheckRequest, fallback Config) (ResourceCandidate, error)
	SaveUpdateEvent(ctx context.Context, cfg Config, req UpdateEventRequest) error
	SaveHeartbeat(ctx context.Context, cfg Config, req HeartbeatRequest) error
	Overview(ctx context.Context) (ReleaseOverview, error)
	AdminOverview(ctx context.Context) (AdminOverview, error)
	CreateApp(ctx context.Context, req CreateAppRequest) (AppAdminSummary, error)
	UpdateAppEnabled(ctx context.Context, id string, enabled bool) (AppAdminSummary, error)
	CreateBuild(ctx context.Context, build AppBuildJob, cfg Config) (AppBuildJob, error)
	SaveWebhookEvent(ctx context.Context, event WebhookEventRequest) (WebhookEventAdmin, error)
	CreateRelease(ctx context.Context, req CreateReleaseRequest) (AppReleaseAdmin, error)
	UpdateReleaseStatus(ctx context.Context, id, status string, publish bool) (AppReleaseAdmin, error)
	UpdateReleaseRollout(ctx context.Context, id string, rollout int) (AppReleaseAdmin, error)
	UpdateReleaseNotes(ctx context.Context, id string, req UpdateNotesRequest) (AppReleaseAdmin, error)
	CreateResourceVersion(ctx context.Context, req CreateResourceVersionRequest, packages []AppResourcePackageAdmin, manifestURL string, cfg Config) (AppResourceVersionAdmin, error)
	UpdateResourceStatus(ctx context.Context, id, status string, publish bool) (AppResourceVersionAdmin, error)
	UpdateResourceRollout(ctx context.Context, id string, rollout int) (AppResourceVersionAdmin, error)
	UpdateResourceNotes(ctx context.Context, id string, req UpdateNotesRequest) (AppResourceVersionAdmin, error)
	GetBuildStorageKey(ctx context.Context, buildID string) (string, string, error)
	GetReleaseStorageKey(ctx context.Context, releaseID string) (string, string, error)
	GetResourcePackageStorageKey(ctx context.Context, packageID string) (string, string, error)
	GetResourceManifestStorageKey(ctx context.Context, resourceID string) (string, string, error)
	InsertAudit(ctx context.Context, action, targetType, targetID, message string, metadata map[string]any) error
	AuditExists(ctx context.Context, action, targetID string) (bool, error)
}

type ResourceCandidate struct {
	ResourceVersion      string
	Channel              string
	UpdateLevel          string
	Title                string
	Summary              string
	ReleaseNotesMarkdown string
	ManifestURL          string
	TotalSize            int64
	MinAppVersionCode    int
	MaxAppVersionCode    int
	Packages             []ResourcePackage
}
