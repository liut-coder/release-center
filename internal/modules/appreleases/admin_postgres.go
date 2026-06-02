package appreleases

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *PostgresStore) AdminOverview(ctx context.Context) (AdminOverview, error) {
	apps, err := s.listAdminApps(ctx)
	if err != nil {
		return AdminOverview{}, err
	}
	builds, err := s.listAdminBuilds(ctx)
	if err != nil {
		return AdminOverview{}, err
	}
	releases, err := s.listAdminReleases(ctx)
	if err != nil {
		return AdminOverview{}, err
	}
	resources, err := s.listAdminResources(ctx)
	if err != nil {
		return AdminOverview{}, err
	}
	installations, err := s.listAdminInstallations(ctx)
	if err != nil {
		return AdminOverview{}, err
	}
	events, err := s.listAdminUpgradeEvents(ctx, false)
	if err != nil {
		return AdminOverview{}, err
	}
	resourceEvents, err := s.listAdminUpgradeEvents(ctx, true)
	if err != nil {
		return AdminOverview{}, err
	}
	webhookEvents, err := s.listAdminWebhookEvents(ctx)
	if err != nil {
		return AdminOverview{}, err
	}
	auditLogs, err := s.listAdminAuditLogs(ctx)
	if err != nil {
		return AdminOverview{}, err
	}
	var latest *AppReleaseAdmin
	for i := range releases {
		if releases[i].IsLatest {
			latest = &releases[i]
			break
		}
	}
	var latestResource *AppResourceVersionAdmin
	for i := range resources {
		if resources[i].Status == "released" || resources[i].Status == "rolling_out" {
			latestResource = &resources[i]
			break
		}
	}
	return AdminOverview{
		Apps:                 apps,
		Releases:             releases,
		Latest:               latest,
		BuildJobs:            builds,
		WebhookEvents:        webhookEvents,
		ResourceVersions:     resources,
		LatestResource:       latestResource,
		Installations:        installations,
		UpgradeEvents:        events,
		ResourceUpdateEvents: resourceEvents,
		QualityMetrics:       buildReleaseQualityMetrics(events, resourceEvents, normalizeQualityPolicy(QualityPolicy{})),
		AuditLogs:            auditLogs,
	}, nil
}

func buildReleaseQualityMetrics(apkEvents, resourceEvents []AppUpgradeEventAdmin, policy QualityPolicy) []ReleaseQualityMetric {
	policy = normalizeQualityPolicy(policy)
	return []ReleaseQualityMetric{
		buildReleaseQualityMetric("apk", apkEvents, policy),
		buildReleaseQualityMetric("resource", resourceEvents, policy),
	}
}

func buildQualityAlerts(metrics []ReleaseQualityMetric) []QualityAlert {
	alerts := make([]QualityAlert, 0, len(metrics))
	for _, metric := range metrics {
		if metric.RecommendedAction == "" || metric.RecommendedAction == "observe" {
			continue
		}
		alerts = append(alerts, QualityAlert{
			ID:                "quality." + metric.Category + "." + metric.RecommendedAction,
			Category:          metric.Category,
			Severity:          qualityAlertSeverity(metric),
			RecommendedAction: metric.RecommendedAction,
			Reason:            metric.ActionReason,
			Threshold:         metric.PolicyThreshold,
			FailureRate:       metric.FailureRate,
			FailureEvents:     metric.FailureEvents,
			LatestFailure:     metric.LatestFailureReason,
			CreatedAt:         time.Now().UTC(),
		})
	}
	return alerts
}

func qualityAlertSeverity(metric ReleaseQualityMetric) string {
	if metric.RecommendedAction == "rollback_apk" {
		return "critical"
	}
	return "warning"
}

func buildReleaseQualityMetric(category string, events []AppUpgradeEventAdmin, policy QualityPolicy) ReleaseQualityMetric {
	policy = normalizeQualityPolicy(policy)
	metric := ReleaseQualityMetric{Category: category, TotalEvents: len(events), RecommendedAction: "observe"}
	reasonCounts := map[string]int{}
	eventTypeCounts := map[string]int{}
	for _, event := range events {
		eventTypeCounts[event.EventType]++
		switch {
		case isSuccessfulUpgradeEvent(event.EventType):
			metric.SuccessEvents++
		case isFailedUpgradeEvent(event.EventType):
			metric.FailureEvents++
			reason := strings.TrimSpace(event.ErrorMessage)
			if reason == "" {
				reason = event.EventType
			}
			if metric.LatestFailureReason == "" {
				metric.LatestFailureReason = reason
			}
			reasonCounts[reason]++
		}
	}
	terminalEvents := metric.SuccessEvents + metric.FailureEvents
	if terminalEvents > 0 {
		metric.SuccessRate = metric.SuccessEvents * 100 / terminalEvents
		metric.FailureRate = metric.FailureEvents * 100 / terminalEvents
	} else {
		metric.SuccessRate = 100
	}
	metric.FailureReasons = sortQualityReasons(reasonCounts)
	metric.PolicyThreshold = qualityPolicyThreshold(category, policy)
	metric.RecommendedAction, metric.ActionReason = recommendedQualityAction(metric, eventTypeCounts, reasonCounts, policy)
	return metric
}

func recommendedQualityAction(metric ReleaseQualityMetric, eventTypeCounts, reasonCounts map[string]int, policy QualityPolicy) (string, string) {
	policy = normalizeQualityPolicy(policy)
	switch metric.Category {
	case "resource":
		activationFailures := eventTypeCounts["activation_failed"] + eventTypeCounts["resource_activation_failed"]
		for reason, count := range reasonCounts {
			if strings.Contains(reason, "activation") && reason != "activation_failed" && reason != "resource_activation_failed" {
				activationFailures += count
			}
		}
		if activationFailures >= policy.ResourceActivationFailedCount && metric.FailureRate >= policy.ResourceFailureRate {
			return "pause_resource", fmt.Sprintf("资源激活失败达到阈值：%d 次且失败率 >= %d%%", policy.ResourceActivationFailedCount, policy.ResourceFailureRate)
		}
	case "apk":
		checksumFailures := eventTypeCounts["checksum_failed"]
		installFailures := eventTypeCounts["install_failed"]
		for reason, count := range reasonCounts {
			if strings.Contains(reason, "checksum") && reason != "checksum_failed" {
				checksumFailures += count
			}
			if strings.Contains(reason, "install") && reason != "install_failed" {
				installFailures += count
			}
		}
		if checksumFailures >= policy.APKChecksumFailedCount {
			return "rollback_apk", fmt.Sprintf("APK 校验失败达到阈值：%d 次", policy.APKChecksumFailedCount)
		}
		if installFailures >= policy.APKInstallFailedCount && metric.FailureRate >= policy.APKInstallFailureRate {
			return "rollback_apk", fmt.Sprintf("APK 安装失败达到阈值：%d 次且失败率 >= %d%%", policy.APKInstallFailedCount, policy.APKInstallFailureRate)
		}
	}
	return "observe", "指标未达到自动干预阈值"
}

func qualityPolicyThreshold(category string, policy QualityPolicy) string {
	policy = normalizeQualityPolicy(policy)
	if category == "resource" {
		return fmt.Sprintf("activation_failed >= %d 且失败率 >= %d%%", policy.ResourceActivationFailedCount, policy.ResourceFailureRate)
	}
	if category == "apk" {
		return fmt.Sprintf("checksum_failed >= %d，或 install_failed >= %d 且失败率 >= %d%%", policy.APKChecksumFailedCount, policy.APKInstallFailedCount, policy.APKInstallFailureRate)
	}
	return ""
}

func sortQualityReasons(counts map[string]int) []QualityReasonCount {
	reasons := make([]QualityReasonCount, 0, len(counts))
	for reason, count := range counts {
		reasons = append(reasons, QualityReasonCount{Reason: reason, Count: count})
	}
	sort.Slice(reasons, func(i, j int) bool {
		if reasons[i].Count != reasons[j].Count {
			return reasons[i].Count > reasons[j].Count
		}
		return reasons[i].Reason < reasons[j].Reason
	})
	if len(reasons) > 5 {
		reasons = reasons[:5]
	}
	return reasons
}

func isSuccessfulUpgradeEvent(eventType string) bool {
	return strings.Contains(eventType, "success") || strings.Contains(eventType, "completed")
}

func isFailedUpgradeEvent(eventType string) bool {
	return strings.Contains(eventType, "failed") || strings.Contains(eventType, "checksum")
}

func (s *PostgresStore) CreateApp(ctx context.Context, req CreateAppRequest) (AppAdminSummary, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var id string
	err := s.db.QueryRow(ctx, `
		insert into apps (id, tenant_id, app_key, name, platform, package_name, description, enabled, metadata)
		values ($1, 'default', $2, $3, $4, $5, $6, $7, '{"source":"admin_api"}'::jsonb)
		on conflict (tenant_id, app_key, platform) do update set
			name = excluded.name,
			package_name = excluded.package_name,
			description = excluded.description,
			enabled = excluded.enabled,
			metadata = apps.metadata || excluded.metadata,
			updated_at = now()
		returning id::text
	`, uuid.NewString(), req.AppKey, req.Name, req.Platform, req.PackageName, req.Description, enabled).Scan(&id)
	if err != nil {
		return AppAdminSummary{}, err
	}
	return s.getAdminApp(ctx, id)
}

func (s *PostgresStore) UpdateAppEnabled(ctx context.Context, id string, enabled bool) (AppAdminSummary, error) {
	_, err := s.db.Exec(ctx, `
		update apps
		set enabled = $2,
		    updated_at = now()
		where id = $1 and tenant_id = 'default'
	`, id, enabled)
	if err != nil {
		return AppAdminSummary{}, err
	}
	return s.getAdminApp(ctx, id)
}

func (s *PostgresStore) CreateBuild(ctx context.Context, build AppBuildJob, cfg Config) (AppBuildJob, error) {
	cfg = normalizeConfig(cfg)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return AppBuildJob{}, err
	}
	defer tx.Rollback(ctx)
	appID, err := ensureApp(ctx, tx, cfg)
	if err != nil {
		return AppBuildJob{}, err
	}
	if build.ID == "" {
		build.ID = uuid.NewString()
	}
	metadata := map[string]any{
		"git_ref":       build.GitRef,
		"api_base_url":  build.APIBaseURL,
		"release_notes": build.ReleaseNotes,
		"log_tail":      build.LogTail,
		"source":        "admin_api",
		"file_name":     build.FileName,
	}
	if build.StorageKey != "" {
		metadata["storage_key"] = build.StorageKey
	}
	err = tx.QueryRow(ctx, `
		insert into app_builds (
			id, tenant_id, app_id, version_name, version_code, build_number,
			channel, build_type, git_commit, git_branch, build_environment,
			build_status, artifact_type, artifact_path, artifact_size, sha256,
			built_by, metadata
		)
		values ($1, 'default', $2, $3, $4, $5, $6, $7, $8, $9, $10,
		        $11, $12, $13, $14, $15, $16, $17)
		on conflict (app_id, build_number) do update set
			version_name = excluded.version_name,
			version_code = excluded.version_code,
			channel = excluded.channel,
			build_type = excluded.build_type,
			git_commit = excluded.git_commit,
			git_branch = excluded.git_branch,
			build_environment = excluded.build_environment,
			build_status = excluded.build_status,
			artifact_type = excluded.artifact_type,
			artifact_path = excluded.artifact_path,
			artifact_size = excluded.artifact_size,
			sha256 = excluded.sha256,
			built_by = excluded.built_by,
			metadata = app_builds.metadata || excluded.metadata
		returning id::text, created_at
	`, build.ID, appID, build.VersionName, build.VersionCode, build.BuildNumber,
		build.Channel, build.BuildType, build.GitCommit, build.GitBranch, build.BuildEnvironment,
		build.Status, build.ArtifactType, build.ArtifactPath, build.ArtifactSize, build.SHA256,
		build.StartedBy, jsonb(metadata)).Scan(&build.ID, &build.CreatedAt)
	if err != nil {
		return AppBuildJob{}, err
	}
	if build.StorageKey != "" {
		build.ArtifactPath = "/api/v1/app/builds/" + build.ID + "/download"
		if _, err := tx.Exec(ctx, `
			update app_builds
			set artifact_path = $2
			where id = $1 and tenant_id = 'default'
		`, build.ID, build.ArtifactPath); err != nil {
			return AppBuildJob{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return AppBuildJob{}, err
	}
	return s.getAdminBuild(ctx, build.ID)
}

func (s *PostgresStore) SaveWebhookEvent(ctx context.Context, event WebhookEventRequest) (WebhookEventAdmin, error) {
	if event.ReceivedAt.IsZero() {
		event.ReceivedAt = time.Now()
	}
	var saved WebhookEventAdmin
	err := s.db.QueryRow(ctx, `
		insert into webhook_events (
			id, tenant_id, provider, event_type, delivery_id, repository,
			ref, commit_sha, sender, action, workflow, run_id, payload, created_at
		)
		values ($1, 'default', $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		returning id::text, provider, event_type, delivery_id, repository,
		          ref, commit_sha, sender, action, workflow, run_id, created_at
	`, uuid.NewString(), event.Provider, event.EventType, event.DeliveryID,
		event.Repository, event.Ref, event.CommitSHA, event.Sender, event.Action,
		event.Workflow, event.RunID, jsonb(event.RawPayload), event.ReceivedAt).Scan(
		&saved.ID, &saved.Provider, &saved.EventType, &saved.DeliveryID,
		&saved.Repository, &saved.Ref, &saved.CommitSHA, &saved.Sender,
		&saved.Action, &saved.Workflow, &saved.RunID, &saved.CreatedAt,
	)
	return saved, err
}

func (s *PostgresStore) CreateRelease(ctx context.Context, req CreateReleaseRequest) (AppReleaseAdmin, error) {
	status := "draft"
	var publishedAt any
	if req.ScheduledAt != "" {
		status = "scheduled"
		if parsed, err := time.Parse(time.RFC3339, req.ScheduledAt); err == nil {
			publishedAt = parsed
		}
	}
	metadata := map[string]any{
		"target_type":  req.TargetType,
		"target_value": req.TargetValue,
		"scheduled_at": req.ScheduledAt,
		"source":       "admin_api",
	}
	var id string
	err := s.db.QueryRow(ctx, `
		insert into app_releases (
			id, tenant_id, app_id, build_id, channel, build_type, status,
			title, summary, release_notes_markdown, upgrade_message,
			update_level, rollout_percentage, min_supported_code,
			block_old_versions, current_version_available, published_at,
			created_by, metadata
		)
		select $1, b.tenant_id, b.app_id, b.id, $2, b.build_type, $3,
		       $4, $5, $6, $5, $7, $8, $9, $10, true, $11, 'admin', $12
		from app_builds b
		where b.id = $13 and b.tenant_id = 'default'
		on conflict (app_id, build_id, channel, build_type) do update set
			status = excluded.status,
			title = excluded.title,
			summary = excluded.summary,
			release_notes_markdown = excluded.release_notes_markdown,
			upgrade_message = excluded.upgrade_message,
			update_level = excluded.update_level,
			rollout_percentage = excluded.rollout_percentage,
			min_supported_code = excluded.min_supported_code,
			block_old_versions = excluded.block_old_versions,
			published_at = excluded.published_at,
			metadata = app_releases.metadata || excluded.metadata,
			updated_at = now()
		returning id::text
	`, uuid.NewString(), req.Channel, status, req.Title, req.Summary,
		req.ReleaseNotesMarkdown, req.UpdateLevel, req.RolloutPercentage,
		req.MinSupportedCode, req.BlockOldVersions, publishedAt, jsonb(metadata), req.BuildID).Scan(&id)
	if err != nil {
		return AppReleaseAdmin{}, err
	}
	_, _ = s.db.Exec(ctx, `
		insert into app_release_notes (id, tenant_id, release_id, language, title, summary, content_markdown, edited_by)
		values ($1, 'default', $2, 'zh-CN', $3, $4, $5, 'admin')
		on conflict (release_id, language) do update set
			title = excluded.title,
			summary = excluded.summary,
			content_markdown = excluded.content_markdown,
			updated_at = now()
	`, uuid.NewString(), id, req.Title, req.Summary, req.ReleaseNotesMarkdown)
	if _, err := s.db.Exec(ctx, `delete from app_release_rules where release_id = $1 and tenant_id = 'default'`, id); err != nil {
		return AppReleaseAdmin{}, err
	}
	ruleType := req.TargetType
	ruleValue := req.TargetValue
	if ruleType == "" || ruleType == "all" {
		ruleType = "all"
		ruleValue = "all"
	}
	if _, err := s.db.Exec(ctx, `
		insert into app_release_rules (id, tenant_id, release_id, rule_type, rule_value)
		values ($1, 'default', $2, $3, $4)
	`, uuid.NewString(), id, ruleType, ruleValue); err != nil {
		return AppReleaseAdmin{}, err
	}
	return s.getAdminRelease(ctx, id)
}

func (s *PostgresStore) UpdateReleaseStatus(ctx context.Context, id, status string, publish bool) (AppReleaseAdmin, error) {
	_, err := s.db.Exec(ctx, `
		update app_releases
		set status = $2::varchar,
		    published_at = case when $3 then coalesce(published_at, now()) else published_at end,
		    paused_at = case when $2::text = 'paused' then now() else null end,
		    updated_at = now()
		where id = $1 and tenant_id = 'default'
	`, id, status, publish)
	if err != nil {
		return AppReleaseAdmin{}, err
	}
	return s.getAdminRelease(ctx, id)
}

func (s *PostgresStore) UpdateReleaseRollout(ctx context.Context, id string, rollout int) (AppReleaseAdmin, error) {
	_, err := s.db.Exec(ctx, `
		update app_releases
		set rollout_percentage = $2,
		    status = case when status = 'released' and $2 < 100 then 'rolling_out' else status end,
		    updated_at = now()
		where id = $1 and tenant_id = 'default'
	`, id, rollout)
	if err != nil {
		return AppReleaseAdmin{}, err
	}
	return s.getAdminRelease(ctx, id)
}

func (s *PostgresStore) UpdateReleaseNotes(ctx context.Context, id string, req UpdateNotesRequest) (AppReleaseAdmin, error) {
	_, err := s.db.Exec(ctx, `
		update app_releases
		set title = $2,
		    summary = $3,
		    release_notes_markdown = $4,
		    upgrade_message = $3,
		    updated_at = now()
		where id = $1 and tenant_id = 'default'
	`, id, req.Title, req.Summary, req.ReleaseNotesMarkdown)
	if err != nil {
		return AppReleaseAdmin{}, err
	}
	_, _ = s.db.Exec(ctx, `
		insert into app_release_notes (id, tenant_id, release_id, language, title, summary, content_markdown, edited_by)
		values ($1, 'default', $2, 'zh-CN', $3, $4, $5, 'admin')
		on conflict (release_id, language) do update set
			title = excluded.title,
			summary = excluded.summary,
			content_markdown = excluded.content_markdown,
			updated_at = now()
	`, uuid.NewString(), id, req.Title, req.Summary, req.ReleaseNotesMarkdown)
	return s.getAdminRelease(ctx, id)
}

func (s *PostgresStore) CreateResourceVersion(ctx context.Context, req CreateResourceVersionRequest, packages []AppResourcePackageAdmin, manifestURL string, cfg Config) (AppResourceVersionAdmin, error) {
	cfg = normalizeConfig(cfg)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return AppResourceVersionAdmin{}, err
	}
	defer tx.Rollback(ctx)
	appID, err := ensureApp(ctx, tx, cfg)
	if err != nil {
		return AppResourceVersionAdmin{}, err
	}
	totalSize := int64(0)
	for _, pkg := range packages {
		totalSize += pkg.FileSize
	}
	id := uuid.NewString()
	publicManifestURL := "/api/v1/app/resources/" + id + "/manifest"
	err = tx.QueryRow(ctx, `
		insert into app_resource_versions (
			id, tenant_id, app_id, resource_version, channel, status,
			min_app_version_code, max_app_version_code, update_level,
			title, summary, release_notes_markdown, rollout_percentage,
			manifest_url, total_size, published_at, created_by, metadata
		)
		values ($1, 'default', $2, $3, $4, 'draft', $5, $6, $7,
		        $8, $9, $10, $11, $12, $13, null, 'admin', '{"source":"admin_api"}'::jsonb)
		on conflict (app_id, resource_version, channel) do update set
			status = 'draft',
			min_app_version_code = excluded.min_app_version_code,
			max_app_version_code = excluded.max_app_version_code,
			update_level = excluded.update_level,
			title = excluded.title,
			summary = excluded.summary,
			release_notes_markdown = excluded.release_notes_markdown,
			rollout_percentage = excluded.rollout_percentage,
			manifest_url = excluded.manifest_url,
			total_size = excluded.total_size,
			metadata = app_resource_versions.metadata || excluded.metadata,
			updated_at = now()
		returning id::text
	`, id, appID, req.ResourceVersion, req.Channel, req.MinAppVersionCode,
		req.MaxAppVersionCode, req.UpdateLevel, req.Title, req.Summary,
		req.ReleaseNotesMarkdown, req.RolloutPercentage, publicManifestURL, totalSize).Scan(&id)
	if err != nil {
		return AppResourceVersionAdmin{}, err
	}
	metadata := map[string]any{
		"manifest_storage_key":         manifestURL,
		"manifest_signature_algorithm": "",
	}
	if strings.TrimSpace(cfg.ManifestPrivateKey) != "" {
		metadata["manifest_signature_algorithm"] = ManifestSignatureAlgorithmEd25519
	}
	if _, err := tx.Exec(ctx, `
		update app_resource_versions
		set manifest_url = $2,
		    metadata = metadata || $3
		where id = $1
	`, id, "/api/v1/app/resources/"+id+"/manifest", jsonb(metadata)); err != nil {
		return AppResourceVersionAdmin{}, err
	}
	if _, err := tx.Exec(ctx, `delete from app_resource_packages where resource_version_id = $1`, id); err != nil {
		return AppResourceVersionAdmin{}, err
	}
	for _, pkg := range packages {
		if pkg.ID == "" {
			pkg.ID = uuid.NewString()
		}
		if _, err := tx.Exec(ctx, `
			insert into app_resource_packages (
				id, tenant_id, resource_version_id, package_key, package_type,
				file_url, file_size, sha256, metadata
			)
			values ($1, 'default', $2, $3, $4, $5, $6, $7, $8)
		`, pkg.ID, id, pkg.PackageKey, pkg.PackageType, pkg.FileURL, pkg.FileSize, pkg.SHA256,
			jsonb(map[string]any{"storage_key": pkg.StorageKey})); err != nil {
			return AppResourceVersionAdmin{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return AppResourceVersionAdmin{}, err
	}
	return s.getAdminResource(ctx, id)
}

func (s *PostgresStore) UpdateResourceStatus(ctx context.Context, id, status string, publish bool) (AppResourceVersionAdmin, error) {
	_, err := s.db.Exec(ctx, `
		update app_resource_versions
		set status = $2::varchar,
		    published_at = case when $3 then coalesce(published_at, now()) else published_at end,
		    paused_at = case when $2::text = 'paused' then now() else null end,
		    updated_at = now()
		where id = $1 and tenant_id = 'default'
	`, id, status, publish)
	if err != nil {
		return AppResourceVersionAdmin{}, err
	}
	return s.getAdminResource(ctx, id)
}

func (s *PostgresStore) UpdateResourceRollout(ctx context.Context, id string, rollout int) (AppResourceVersionAdmin, error) {
	_, err := s.db.Exec(ctx, `
		update app_resource_versions
		set rollout_percentage = $2,
		    status = case when status = 'released' and $2 < 100 then 'rolling_out' else status end,
		    updated_at = now()
		where id = $1 and tenant_id = 'default'
	`, id, rollout)
	if err != nil {
		return AppResourceVersionAdmin{}, err
	}
	return s.getAdminResource(ctx, id)
}

func (s *PostgresStore) UpdateResourceNotes(ctx context.Context, id string, req UpdateNotesRequest) (AppResourceVersionAdmin, error) {
	_, err := s.db.Exec(ctx, `
		update app_resource_versions
		set title = $2,
		    summary = $3,
		    release_notes_markdown = $4,
		    updated_at = now()
		where id = $1 and tenant_id = 'default'
	`, id, req.Title, req.Summary, req.ReleaseNotesMarkdown)
	if err != nil {
		return AppResourceVersionAdmin{}, err
	}
	return s.getAdminResource(ctx, id)
}

func (s *PostgresStore) GetBuildStorageKey(ctx context.Context, buildID string) (string, string, error) {
	var storageKey, fileName string
	err := s.db.QueryRow(ctx, `
		select coalesce(metadata->>'storage_key', ''),
		       coalesce(nullif(metadata->>'file_name', ''), version_name || '-' || build_number::text || '.apk')
		from app_builds
		where id = $1 and tenant_id = 'default'
	`, buildID).Scan(&storageKey, &fileName)
	if err != nil {
		return "", "", err
	}
	if storageKey == "" {
		return "", "", fmt.Errorf("build storage_key is empty")
	}
	return storageKey, fileName, nil
}

func (s *PostgresStore) GetReleaseStorageKey(ctx context.Context, releaseID string) (string, string, error) {
	var storageKey, fileName string
	err := s.db.QueryRow(ctx, `
		select coalesce(b.metadata->>'storage_key', ''),
		       coalesce(nullif(b.metadata->>'file_name', ''), b.version_name || '-' || b.build_number::text || '.apk')
		from app_releases r
		join app_builds b on b.id = r.build_id
		where r.id = $1 and r.tenant_id = 'default'
	`, releaseID).Scan(&storageKey, &fileName)
	if err != nil {
		return "", "", err
	}
	if storageKey == "" {
		return "", "", fmt.Errorf("release storage_key is empty")
	}
	return storageKey, fileName, nil
}

func (s *PostgresStore) GetResourcePackageStorageKey(ctx context.Context, packageID string) (string, string, error) {
	var storageKey, packageKey string
	err := s.db.QueryRow(ctx, `
		select coalesce(metadata->>'storage_key', ''), package_key
		from app_resource_packages
		where id = $1 and tenant_id = 'default'
	`, packageID).Scan(&storageKey, &packageKey)
	if err != nil {
		return "", "", err
	}
	if storageKey == "" {
		return "", "", fmt.Errorf("resource package storage_key is empty")
	}
	return storageKey, packageKey + ".zip", nil
}

func (s *PostgresStore) GetResourceManifestStorageKey(ctx context.Context, resourceID string) (string, string, error) {
	var storageKey, resourceVersion string
	err := s.db.QueryRow(ctx, `
		select coalesce(metadata->>'manifest_storage_key', ''), resource_version
		from app_resource_versions
		where id = $1 and tenant_id = 'default'
	`, resourceID).Scan(&storageKey, &resourceVersion)
	if err != nil {
		return "", "", err
	}
	if storageKey == "" {
		return "", "", fmt.Errorf("resource manifest storage_key is empty")
	}
	return storageKey, "manifest-" + resourceVersion + ".json", nil
}

func (s *PostgresStore) InsertAudit(ctx context.Context, action, targetType, targetID, message string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	_, err := s.db.Exec(ctx, `
		insert into audit_events (
			tenant_id, actor_type, actor_id, action, target_type, target_id, message_zh, metadata
		)
		values ('default', 'admin', 'app-release-center', $1, $2, $3, $4, $5)
	`, action, targetType, targetID, message, jsonb(metadata))
	return err
}

func (s *PostgresStore) AuditExists(ctx context.Context, action, targetID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `
		select exists (
			select 1
			from audit_events
			where tenant_id = 'default'
			  and action = $1
			  and target_id = $2
		)
	`, action, targetID).Scan(&exists)
	return exists, err
}

func (s *PostgresStore) getAdminApp(ctx context.Context, id string) (AppAdminSummary, error) {
	apps, err := s.listAdminAppsWhere(ctx, "id = $1", id)
	if err != nil {
		return AppAdminSummary{}, err
	}
	if len(apps) == 0 {
		return AppAdminSummary{}, pgx.ErrNoRows
	}
	return apps[0], nil
}

func (s *PostgresStore) getAdminBuild(ctx context.Context, id string) (AppBuildJob, error) {
	builds, err := s.listAdminBuildsWhere(ctx, "b.id = $1", id)
	if err != nil {
		return AppBuildJob{}, err
	}
	if len(builds) == 0 {
		return AppBuildJob{}, pgx.ErrNoRows
	}
	return builds[0], nil
}

func (s *PostgresStore) getAdminRelease(ctx context.Context, id string) (AppReleaseAdmin, error) {
	releases, err := s.listAdminReleasesWhere(ctx, "r.id = $1", id)
	if err != nil {
		return AppReleaseAdmin{}, err
	}
	if len(releases) == 0 {
		return AppReleaseAdmin{}, pgx.ErrNoRows
	}
	return releases[0], nil
}

func (s *PostgresStore) getAdminResource(ctx context.Context, id string) (AppResourceVersionAdmin, error) {
	resources, err := s.listAdminResourcesWhere(ctx, "rv.id = $1", id)
	if err != nil {
		return AppResourceVersionAdmin{}, err
	}
	if len(resources) == 0 {
		return AppResourceVersionAdmin{}, pgx.ErrNoRows
	}
	return resources[0], nil
}

func (s *PostgresStore) listAdminApps(ctx context.Context) ([]AppAdminSummary, error) {
	return s.listAdminAppsWhere(ctx, "true")
}

func (s *PostgresStore) listAdminAppsWhere(ctx context.Context, where string, args ...any) ([]AppAdminSummary, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, app_key, name, platform, package_name, description,
		       enabled, created_at, updated_at
		from apps
		where tenant_id = 'default' and `+where+`
		order by updated_at desc
		limit 100
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []AppAdminSummary
	for rows.Next() {
		var app AppAdminSummary
		if err := rows.Scan(&app.ID, &app.AppKey, &app.Name, &app.Platform,
			&app.PackageName, &app.Description, &app.Enabled, &app.CreatedAt,
			&app.UpdatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func (s *PostgresStore) listAdminBuilds(ctx context.Context) ([]AppBuildJob, error) {
	return s.listAdminBuildsWhere(ctx, "true")
}

func (s *PostgresStore) listAdminBuildsWhere(ctx context.Context, where string, args ...any) ([]AppBuildJob, error) {
	rows, err := s.db.Query(ctx, `
		select b.id::text, b.app_id::text, b.build_status,
		       coalesce(b.metadata->>'git_ref', b.git_branch, ''),
		       b.git_commit, b.git_branch, b.build_type, b.channel,
		       b.version_name, b.version_code, b.build_number,
		       b.build_environment, b.artifact_type, b.artifact_path,
		       b.artifact_size, b.sha256, b.built_by,
		       b.created_at, b.created_at, b.created_at,
		       coalesce(b.metadata->'log_tail', '[]'::jsonb),
		       coalesce(b.metadata->>'storage_key', ''),
		       coalesce(b.metadata->>'file_name', ''),
		       coalesce(b.metadata->>'api_base_url', ''),
		       coalesce(b.metadata->>'release_notes', ''),
		       coalesce(r.id::text, '')
		from app_builds b
		left join app_releases r on r.build_id = b.id and r.tenant_id = b.tenant_id
		where b.tenant_id = 'default' and `+where+`
		order by b.version_code desc, b.build_number desc, b.created_at desc
		limit 100
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var builds []AppBuildJob
	for rows.Next() {
		var item AppBuildJob
		var logTail []byte
		if err := rows.Scan(&item.ID, &item.AppID, &item.Status, &item.GitRef,
			&item.GitCommit, &item.GitBranch, &item.BuildType, &item.Channel,
			&item.VersionName, &item.VersionCode, &item.BuildNumber,
			&item.BuildEnvironment, &item.ArtifactType, &item.ArtifactPath,
			&item.ArtifactSize, &item.SHA256, &item.StartedBy, &item.StartedAt,
			&item.CreatedAt, &item.FinishedAt, &logTail, &item.StorageKey, &item.FileName,
			&item.APIBaseURL, &item.ReleaseNotes, &item.ReleaseID); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(logTail, &item.LogTail)
		if item.FileName == "" {
			item.FileName = path.Base(item.ArtifactPath)
			if item.FileName == "." || item.FileName == "/" {
				item.FileName = ""
			}
		}
		builds = append(builds, item)
	}
	return builds, rows.Err()
}

func (s *PostgresStore) listAdminReleases(ctx context.Context) ([]AppReleaseAdmin, error) {
	return s.listAdminReleasesWhere(ctx, "true")
}

func (s *PostgresStore) listAdminReleasesWhere(ctx context.Context, where string, args ...any) ([]AppReleaseAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select r.id::text, r.id::text, r.app_id::text, r.build_id::text,
		       a.package_name, b.version_name, b.version_code, b.build_number,
		       r.channel, r.build_type, b.git_branch, b.git_commit,
		       coalesce(b.metadata->>'api_base_url', ''),
		       b.artifact_path, b.artifact_path, b.artifact_size, b.sha256,
		       coalesce(b.metadata->>'file_name', ''),
		       r.status, r.title, r.summary, r.release_notes_markdown,
		       r.upgrade_message, r.update_level, r.rollout_percentage,
		       coalesce(r.metadata->>'target_type', 'all'),
		       coalesce(r.metadata->>'target_value', ''),
		       r.min_supported_code, r.block_old_versions,
		       r.status in ('released', 'rolling_out', 'testing'),
		       r.update_level = 'forced',
		       coalesce(r.published_at, '0001-01-01 00:00:00+00'::timestamptz),
		       coalesce(r.paused_at, '0001-01-01 00:00:00+00'::timestamptz),
		       r.created_by, r.created_at, r.updated_at,
		       row_number() over (partition by r.channel, r.build_type order by
		           case when r.status in ('released', 'rolling_out', 'testing') then 0 else 1 end,
		           b.version_code desc, b.build_number desc, coalesce(r.published_at, r.created_at) desc
		       ) = 1
		from app_releases r
		join apps a on a.id = r.app_id
		join app_builds b on b.id = r.build_id
		where r.tenant_id = 'default' and `+where+`
		order by b.version_code desc, b.build_number desc, r.updated_at desc
		limit 100
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var releases []AppReleaseAdmin
	for rows.Next() {
		var item AppReleaseAdmin
		if err := rows.Scan(&item.ID, &item.ReleaseID, &item.AppID, &item.BuildID,
			&item.PackageName, &item.VersionName, &item.VersionCode, &item.BuildNumber,
			&item.Channel, &item.BuildType, &item.GitRef, &item.GitCommit,
			&item.APIBaseURL, &item.APKPath, &item.ArtifactPath, &item.SizeBytes, &item.SHA256, &item.FileName,
			&item.Status, &item.Title, &item.Summary, &item.ReleaseNotesMarkdown,
			&item.UpgradeMessage, &item.UpdateLevel, &item.RolloutPercentage,
			&item.TargetType, &item.TargetValue, &item.MinSupportedCode,
			&item.BlockOldVersions, &item.IsPublished, &item.ForceUpdate,
			&item.PublishedAt, &item.PausedAt, &item.CreatedBy, &item.CreatedAt,
			&item.UpdatedAt, &item.IsLatest); err != nil {
			return nil, err
		}
		item.ReleaseNotes = item.ReleaseNotesMarkdown
		item.DownloadURL = item.ArtifactPath
		if item.FileName == "" {
			item.FileName = path.Base(item.ArtifactPath)
		}
		releases = append(releases, item)
	}
	return releases, rows.Err()
}

func (s *PostgresStore) listAdminResources(ctx context.Context) ([]AppResourceVersionAdmin, error) {
	return s.listAdminResourcesWhere(ctx, "true")
}

func (s *PostgresStore) listAdminResourcesWhere(ctx context.Context, where string, args ...any) ([]AppResourceVersionAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select rv.id::text, rv.app_id::text, rv.resource_version, rv.channel,
		       rv.status, rv.min_app_version_code, rv.max_app_version_code,
		       rv.update_level, rv.title, rv.summary, rv.release_notes_markdown,
		       rv.rollout_percentage, rv.manifest_url,
		       coalesce(rv.metadata->>'manifest_signature_algorithm', ''),
		       rv.total_size,
		       coalesce(rv.published_at, '0001-01-01 00:00:00+00'::timestamptz),
		       coalesce(rv.paused_at, '0001-01-01 00:00:00+00'::timestamptz),
		       rv.created_by, rv.created_at, rv.updated_at
		from app_resource_versions rv
		where rv.tenant_id = 'default' and `+where+`
		order by coalesce(rv.published_at, rv.created_at) desc, rv.created_at desc
		limit 100
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var resources []AppResourceVersionAdmin
	for rows.Next() {
		var item AppResourceVersionAdmin
		if err := rows.Scan(&item.ID, &item.AppID, &item.ResourceVersion, &item.Channel,
			&item.Status, &item.MinAppVersionCode, &item.MaxAppVersionCode,
			&item.UpdateLevel, &item.Title, &item.Summary, &item.ReleaseNotesMarkdown,
			&item.RolloutPercentage, &item.ManifestURL, &item.SignatureAlgorithm, &item.TotalSize,
			&item.PublishedAt, &item.PausedAt, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.ManifestSigned = item.SignatureAlgorithm != ""
		packages, err := s.listAdminResourcePackages(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		item.Packages = packages
		resources = append(resources, item)
	}
	return resources, rows.Err()
}

func (s *PostgresStore) listAdminResourcePackages(ctx context.Context, resourceID string) ([]AppResourcePackageAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, package_key, package_type, file_url, file_size, sha256,
		       coalesce(metadata->>'storage_key', ''), created_at
		from app_resource_packages
		where tenant_id = 'default' and resource_version_id = $1
		order by package_key asc
	`, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var packages []AppResourcePackageAdmin
	for rows.Next() {
		var item AppResourcePackageAdmin
		if err := rows.Scan(&item.ID, &item.PackageKey, &item.PackageType, &item.FileURL,
			&item.FileSize, &item.SHA256, &item.StorageKey, &item.CreatedAt); err != nil {
			return nil, err
		}
		packages = append(packages, item)
	}
	return packages, rows.Err()
}

func (s *PostgresStore) listAdminInstallations(ctx context.Context) ([]AppInstallationAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, app_id::text, user_id, device_id, device_name, platform,
		       os_version, device_model, installed_version, installed_code,
		       build_number, resource_version,
		       coalesce(last_seen_at, '0001-01-01 00:00:00+00'::timestamptz),
		       coalesce(last_upgrade_at, '0001-01-01 00:00:00+00'::timestamptz),
		       last_upgrade_status, last_error, created_at, updated_at
		from app_installations
		where tenant_id = 'default'
		order by coalesce(last_seen_at, updated_at) desc
		limit 200
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []AppInstallationAdmin
	for rows.Next() {
		var item AppInstallationAdmin
		if err := rows.Scan(&item.ID, &item.AppID, &item.UserID, &item.DeviceID,
			&item.DeviceName, &item.Platform, &item.OSVersion, &item.DeviceModel,
			&item.InstalledVersion, &item.InstalledCode, &item.BuildNumber,
			&item.ResourceVersion, &item.LastSeenAt, &item.LastUpgradeAt,
			&item.LastUpgradeStatus, &item.LastError, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) listAdminUpgradeEvents(ctx context.Context, resource bool) ([]AppUpgradeEventAdmin, error) {
	query := `
		select id::text, app_id::text, coalesce(release_id::text, ''), user_id,
		       device_id, from_version_code, to_version_code, from_version,
		       to_version, event_type, '' as package_key, error_message,
		       created_at, 'apk'
		from app_upgrade_events
		where tenant_id = 'default'
		order by created_at desc
		limit 100
	`
	if resource {
		query = `
			select id::text, app_id::text, '' as release_id, user_id,
			       device_id, 0 as from_version_code, 0 as to_version_code,
			       from_version, to_version, event_type, package_key,
			       error_message, created_at, 'resource'
			from app_resource_update_events
			where tenant_id = 'default'
			order by created_at desc
			limit 100
		`
	}
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []AppUpgradeEventAdmin
	for rows.Next() {
		var item AppUpgradeEventAdmin
		if err := rows.Scan(&item.ID, &item.AppID, &item.ReleaseID, &item.UserID,
			&item.DeviceID, &item.FromVersionCode, &item.ToVersionCode,
			&item.FromVersion, &item.ToVersion, &item.EventType, &item.PackageKey,
			&item.ErrorMessage, &item.CreatedAt, &item.Category); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) listAdminWebhookEvents(ctx context.Context) ([]WebhookEventAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, provider, event_type, delivery_id, repository,
		       ref, commit_sha, sender, action, workflow, run_id, created_at
		from webhook_events
		where tenant_id = 'default'
		order by created_at desc
		limit 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []WebhookEventAdmin
	for rows.Next() {
		var item WebhookEventAdmin
		if err := rows.Scan(&item.ID, &item.Provider, &item.EventType,
			&item.DeliveryID, &item.Repository, &item.Ref, &item.CommitSHA,
			&item.Sender, &item.Action, &item.Workflow, &item.RunID,
			&item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) listAdminAuditLogs(ctx context.Context) ([]AppReleaseAuditLogAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select event_id::text, actor_id, action, target_type, target_id, metadata, event_time
		from audit_events
		where tenant_id = 'default'
		  and (action like 'app.%' or action like 'release.%' or action like 'resource.%' or action like 'webhook.%')
		order by event_time desc, id desc
		limit 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []AppReleaseAuditLogAdmin
	for rows.Next() {
		var item AppReleaseAuditLogAdmin
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.Operator, &item.Action, &item.TargetType,
			&item.TargetID, &metadata, &item.CreatedAt); err != nil {
			return nil, err
		}
		var meta map[string]any
		_ = json.Unmarshal(metadata, &meta)
		item.AfterJSON = meta
		items = append(items, item)
	}
	return items, rows.Err()
}
