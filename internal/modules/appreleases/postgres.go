package appreleases

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	db *pgxpool.Pool
}

func NewPostgresStore(db *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) EnsureConfiguredRelease(ctx context.Context, cfg Config) error {
	cfg = normalizeConfig(cfg)
	if cfg.LatestVersionCode <= 0 {
		return nil
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	appID, err := ensureApp(ctx, tx, cfg)
	if err != nil {
		return err
	}
	buildID, err := ensureBuild(ctx, tx, appID, cfg)
	if err != nil {
		return err
	}
	if err := ensureRelease(ctx, tx, appID, buildID, cfg); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *PostgresStore) FindLatestRelease(ctx context.Context, req CheckRequest, fallback Config) (Config, error) {
	fallback = normalizeConfig(fallback)
	req = normalizeCheckRequest(req, fallback)
	var cfg Config
	var updateLevel string
	err := s.db.QueryRow(ctx, `
		select a.app_key, a.name, a.package_name,
		       b.version_name, b.version_code, b.build_number,
		       r.channel, r.build_type,
		       b.artifact_path, b.sha256, b.artifact_size,
		       coalesce(b.metadata->>'file_name', ''),
		       r.update_level, r.current_version_available, r.min_supported_code,
		       r.release_notes_markdown, r.upgrade_message,
		       coalesce(r.metadata->>'unavailable_reason', '')
		from app_releases r
		join apps a on a.id = r.app_id
		join app_builds b on b.id = r.build_id
		where r.tenant_id = 'default'
		  and a.package_name = $1
		  and a.platform = 'android'
		  and a.enabled = true
		  and r.status in ('released', 'rolling_out', 'testing')
		  and r.channel = $2
		  and r.build_type = $3
		  and (r.published_at is null or r.published_at <= now())
		  and (
		    coalesce(nullif(r.metadata->>'target_type', ''), 'all') = 'all'
		    or (r.metadata->>'target_type' = 'device_id' and coalesce(r.metadata->>'target_value', '') = $4)
		    or (r.metadata->>'target_type' = 'user_id' and coalesce(r.metadata->>'target_value', '') = $5)
		    or exists (
		      select 1
		      from app_release_rules rr
		      where rr.release_id = r.id
		        and rr.tenant_id = r.tenant_id
		        and (
		          (rr.rule_type = 'device_id' and rr.rule_value = $4)
		          or (rr.rule_type = 'user_id' and rr.rule_value = $5)
		          or (rr.rule_type = 'all' and rr.rule_value = 'all')
		        )
		    )
		  )
		  and (
		    r.rollout_percentage >= 100
		    or mod(abs(hashtext(coalesce(nullif($4, ''), nullif($5, ''), a.package_name))), 100) < greatest(0, r.rollout_percentage)
		  )
		order by b.version_code desc, b.build_number desc, r.published_at desc nulls last, r.updated_at desc
		limit 1
	`, req.PackageName, req.Channel, req.BuildType, req.DeviceID, stringifyUserID(req.UserID)).Scan(
		&cfg.AppKey,
		&cfg.Name,
		&cfg.PackageName,
		&cfg.LatestVersionName,
		&cfg.LatestVersionCode,
		&cfg.BuildNumber,
		&cfg.Channel,
		&cfg.BuildType,
		&cfg.DownloadURL,
		&cfg.SHA256,
		&cfg.SizeBytes,
		&cfg.FileName,
		&updateLevel,
		&cfg.CurrentVersionAvailable,
		&cfg.MinSupportedVersionCode,
		&cfg.ReleaseNotes,
		&cfg.MessageZh,
		&cfg.UnavailableReason,
	)
	if err == pgx.ErrNoRows {
		return fallback, nil
	}
	if err != nil {
		return fallback, err
	}
	cfg.APKURL = cfg.DownloadURL
	cfg.ForceUpdate = updateLevel == "forced"
	if cfg.MessageZh == "" {
		cfg.MessageZh = fallback.MessageZh
	}
	if cfg.UnavailableReason == "" {
		cfg.UnavailableReason = fallback.UnavailableReason
	}
	return normalizeConfig(cfg), nil
}

func (s *PostgresStore) UpsertInstallation(ctx context.Context, cfg Config, req CheckRequest) error {
	deviceID := firstNonBlank(req.DeviceID, req.DeviceKey)
	if deviceID == "" {
		return nil
	}
	appID, err := s.findOrCreateApp(ctx, cfg)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `
		insert into app_installations (
			id, tenant_id, app_id, user_id, device_id, device_key, platform,
			os_version, device_model, installed_version, installed_code,
			build_number, resource_version, last_seen_at, metadata
		)
		values ($1, 'default', $2, $3, $4, $5, 'android',
		        $6, $7, $8, $9, $10, '', now(), $11)
		on conflict (app_id, device_id) do update set
			user_id = excluded.user_id,
			device_key = excluded.device_key,
			os_version = excluded.os_version,
			device_model = excluded.device_model,
			installed_version = excluded.installed_version,
			installed_code = excluded.installed_code,
			build_number = excluded.build_number,
			last_seen_at = now(),
			metadata = app_installations.metadata || excluded.metadata,
			updated_at = now()
	`, uuid.NewString(), appID, stringifyUserID(req.UserID), deviceID, req.DeviceKey,
		req.OSVersion, req.DeviceModel, req.VersionName, effectiveCheckVersionCode(req),
		req.BuildNumber, jsonb(map[string]any{"source": "update_check"}))
	return err
}

func (s *PostgresStore) FindLatestResource(ctx context.Context, req ResourceCheckRequest, fallback Config) (ResourceCandidate, error) {
	fallback = normalizeConfig(fallback)
	req = normalizeResourceCheckRequest(req, fallback)
	appVersionCode := effectiveResourceAppVersionCode(req)
	rows, err := s.db.Query(ctx, `
		select rv.id::text, rv.resource_version, rv.channel, rv.update_level,
		       rv.title, rv.summary, rv.release_notes_markdown, rv.manifest_url,
		       rv.total_size, rv.min_app_version_code, rv.max_app_version_code
		from app_resource_versions rv
		join apps a on a.id = rv.app_id
		where rv.tenant_id = 'default'
		  and a.package_name = $1
		  and a.platform = 'android'
		  and a.enabled = true
		  and rv.status in ('released', 'rolling_out', 'testing')
		  and rv.channel = $2
		  and (rv.published_at is null or rv.published_at <= now())
		  and ($3 = 0 or rv.min_app_version_code = 0 or rv.min_app_version_code <= $3)
		  and ($3 = 0 or rv.max_app_version_code = 0 or rv.max_app_version_code >= $3)
		  and (
		    rv.rollout_percentage >= 100
		    or mod(abs(hashtext(coalesce(nullif($4, ''), a.package_name))), 100) < greatest(0, rv.rollout_percentage)
		  )
		order by rv.published_at desc nulls last, rv.created_at desc
		limit 1
	`, req.PackageName, req.Channel, appVersionCode, firstNonBlank(req.DeviceID, req.DeviceKey, stringifyUserID(req.UserID), req.PackageName))
	if err != nil {
		return ResourceCandidate{}, err
	}
	defer rows.Close()

	var id string
	var candidate ResourceCandidate
	if !rows.Next() {
		return ResourceCandidate{}, nil
	}
	if err := rows.Scan(
		&id,
		&candidate.ResourceVersion,
		&candidate.Channel,
		&candidate.UpdateLevel,
		&candidate.Title,
		&candidate.Summary,
		&candidate.ReleaseNotesMarkdown,
		&candidate.ManifestURL,
		&candidate.TotalSize,
		&candidate.MinAppVersionCode,
		&candidate.MaxAppVersionCode,
	); err != nil {
		return ResourceCandidate{}, err
	}
	packages, err := s.listResourcePackages(ctx, id)
	if err != nil {
		return ResourceCandidate{}, err
	}
	candidate.Packages = packages
	return candidate, rows.Err()
}

func (s *PostgresStore) SaveUpdateEvent(ctx context.Context, cfg Config, req UpdateEventRequest) error {
	cfg = normalizeConfig(cfg)
	if req.EventType == "" {
		return fmt.Errorf("eventType is required")
	}
	appID, err := s.findOrCreateApp(ctx, cfg)
	if err != nil {
		return err
	}
	deviceID := firstNonBlank(req.DeviceID, req.DeviceKey)
	if deviceID == "" {
		deviceID = "unknown"
	}
	if strings.HasPrefix(req.EventType, "resource_") ||
		strings.Contains(req.EventType, "manifest") ||
		strings.Contains(req.EventType, "activation") ||
		strings.Contains(req.EventType, "extract") {
		_, err = s.db.Exec(ctx, `
			insert into app_resource_update_events (
				id, tenant_id, app_id, user_id, device_id, from_version,
				to_version, event_type, package_key, error_message, metadata
			)
			values ($1, 'default', $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, uuid.NewString(), appID, stringifyUserID(req.UserID), deviceID,
			req.FromVersion, req.ToVersion, req.EventType, req.PackageKey,
			req.ErrorMessage, jsonb(req.Metadata))
		if err != nil {
			return err
		}
		if err := s.upsertInstallationAfterResourceEvent(ctx, appID, deviceID, req); err != nil {
			return err
		}
		if shouldAutoPauseResource(req) {
			return s.pauseResourceAfterActivationFailure(ctx, appID, req)
		}
		return nil
	}
	_, err = s.db.Exec(ctx, `
		insert into app_upgrade_events (
			id, tenant_id, app_id, user_id, device_id, from_version, to_version,
			from_version_code, to_version_code, event_type, error_message, metadata
		)
		values ($1, 'default', $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, uuid.NewString(), appID, stringifyUserID(req.UserID), deviceID,
		req.FromVersion, req.ToVersion, req.FromVersionCode, req.ToVersionCode,
		req.EventType, req.ErrorMessage, jsonb(req.Metadata))
	if err != nil {
		return err
	}
	return s.upsertInstallationAfterUpgradeEvent(ctx, appID, deviceID, req)
}

func (s *PostgresStore) SaveHeartbeat(ctx context.Context, cfg Config, req HeartbeatRequest) error {
	cfg = normalizeConfig(cfg)
	appID, err := s.findOrCreateApp(ctx, cfg)
	if err != nil {
		return err
	}
	deviceID := firstNonBlank(req.DeviceID, req.DeviceKey)
	if deviceID == "" {
		return fmt.Errorf("deviceId is required")
	}
	versionCode := req.VersionCode
	if req.AppVersionCode > 0 {
		versionCode = req.AppVersionCode
	}
	_, err = s.db.Exec(ctx, `
		insert into app_installations (
			id, tenant_id, app_id, user_id, device_id, device_key, device_name,
			platform, os_version, device_model, installed_version, installed_code,
			build_number, resource_version, last_seen_at, metadata
		)
		values ($1, 'default', $2, $3, $4, $5, $6,
		        $7, $8, $9, $10, $11, $12, $13, now(), $14)
		on conflict (app_id, device_id) do update set
			user_id = coalesce(nullif(excluded.user_id, ''), app_installations.user_id),
			device_key = coalesce(nullif(excluded.device_key, ''), app_installations.device_key),
			device_name = coalesce(nullif(excluded.device_name, ''), app_installations.device_name),
			platform = coalesce(nullif(excluded.platform, ''), app_installations.platform),
			os_version = coalesce(nullif(excluded.os_version, ''), app_installations.os_version),
			device_model = coalesce(nullif(excluded.device_model, ''), app_installations.device_model),
			installed_version = coalesce(nullif(excluded.installed_version, ''), app_installations.installed_version),
			installed_code = case when excluded.installed_code > 0 then excluded.installed_code else app_installations.installed_code end,
			build_number = case when excluded.build_number > 0 then excluded.build_number else app_installations.build_number end,
			resource_version = coalesce(nullif(excluded.resource_version, ''), app_installations.resource_version),
			last_seen_at = now(),
			metadata = app_installations.metadata || excluded.metadata,
			updated_at = now()
	`, uuid.NewString(), appID, stringifyUserID(req.UserID), deviceID, req.DeviceKey, req.DeviceName,
		firstNonBlank(req.Platform, "android"), req.OSVersion, req.DeviceModel, req.VersionName,
		versionCode, req.BuildNumber, req.ResourceVersion, jsonb(map[string]any{"source": "heartbeat"}))
	return err
}

func shouldAutoPauseResource(req UpdateEventRequest) bool {
	eventType := strings.TrimSpace(req.EventType)
	return (eventType == "activation_failed" || eventType == "resource_activation_failed") &&
		strings.TrimSpace(req.ToVersion) != ""
}

func (s *PostgresStore) upsertInstallationAfterUpgradeEvent(ctx context.Context, appID, deviceID string, req UpdateEventRequest) error {
	status := installationStatusFromEvent(req.EventType)
	if status == "" {
		return nil
	}
	success := strings.TrimSpace(req.EventType) == "install_success"
	installedVersion := ""
	installedCode := 0
	if success {
		installedVersion = strings.TrimSpace(req.ToVersion)
		installedCode = req.ToVersionCode
	}
	lastError := ""
	if strings.Contains(req.EventType, "failed") {
		lastError = req.ErrorMessage
	}
	_, err := s.db.Exec(ctx, `
		insert into app_installations (
			id, tenant_id, app_id, user_id, device_id, device_key, platform,
			installed_version, installed_code, build_number, resource_version,
			last_seen_at, last_upgrade_at, last_upgrade_status, last_error, metadata
		)
		values ($1, 'default', $2, $3, $4, $5, 'android',
		        $6, $7, 0, '', now(), now(), $8, $9, $10)
		on conflict (app_id, device_id) do update set
			user_id = coalesce(nullif(excluded.user_id, ''), app_installations.user_id),
			device_key = coalesce(nullif(excluded.device_key, ''), app_installations.device_key),
			installed_version = case when $11 and excluded.installed_version <> '' then excluded.installed_version else app_installations.installed_version end,
			installed_code = case when $11 and excluded.installed_code > 0 then excluded.installed_code else app_installations.installed_code end,
			last_seen_at = now(),
			last_upgrade_at = now(),
			last_upgrade_status = excluded.last_upgrade_status,
			last_error = excluded.last_error,
			metadata = app_installations.metadata || excluded.metadata,
			updated_at = now()
	`, uuid.NewString(), appID, stringifyUserID(req.UserID), deviceID, req.DeviceKey,
		installedVersion, installedCode, status, lastError,
		jsonb(map[string]any{"source": "update_event", "event_type": req.EventType}), success)
	return err
}

func (s *PostgresStore) upsertInstallationAfterResourceEvent(ctx context.Context, appID, deviceID string, req UpdateEventRequest) error {
	status := installationStatusFromEvent(req.EventType)
	if status == "" {
		return nil
	}
	eventType := strings.TrimSpace(req.EventType)
	success := eventType == "activation_success" || eventType == "resource_activation_success"
	resourceVersion := ""
	if success {
		resourceVersion = strings.TrimSpace(req.ToVersion)
	}
	lastError := ""
	if strings.Contains(req.EventType, "failed") {
		lastError = req.ErrorMessage
	}
	_, err := s.db.Exec(ctx, `
		insert into app_installations (
			id, tenant_id, app_id, user_id, device_id, device_key, platform,
			installed_version, installed_code, build_number, resource_version,
			last_seen_at, last_upgrade_at, last_upgrade_status, last_error, metadata
		)
		values ($1, 'default', $2, $3, $4, $5, 'android',
		        '', 0, 0, $6, now(), now(), $7, $8, $9)
		on conflict (app_id, device_id) do update set
			user_id = coalesce(nullif(excluded.user_id, ''), app_installations.user_id),
			device_key = coalesce(nullif(excluded.device_key, ''), app_installations.device_key),
			resource_version = case when $10 and excluded.resource_version <> '' then excluded.resource_version else app_installations.resource_version end,
			last_seen_at = now(),
			last_upgrade_at = now(),
			last_upgrade_status = excluded.last_upgrade_status,
			last_error = excluded.last_error,
			metadata = app_installations.metadata || excluded.metadata,
			updated_at = now()
	`, uuid.NewString(), appID, stringifyUserID(req.UserID), deviceID, req.DeviceKey,
		resourceVersion, status, lastError,
		jsonb(map[string]any{"source": "update_event", "event_type": req.EventType, "package_key": req.PackageKey}), success)
	return err
}

func installationStatusFromEvent(eventType string) string {
	switch strings.TrimSpace(eventType) {
	case "download_started", "download_success", "download_failed",
		"apk_download_started", "apk_download_success", "apk_download_failed",
		"install_started", "install_success", "install_failed",
		"manifest_downloaded", "manifest_failed",
		"package_downloaded", "package_failed",
		"extract_success", "extract_failed",
		"activation_success", "activation_failed",
		"resource_activation_success", "resource_activation_failed",
		"resource_rollback_success":
		return strings.TrimSpace(eventType)
	default:
		return ""
	}
}

func (s *PostgresStore) pauseResourceAfterActivationFailure(ctx context.Context, appID string, req UpdateEventRequest) error {
	var resourceID string
	err := s.db.QueryRow(ctx, `
		update app_resource_versions
		set status = 'paused',
		    paused_at = coalesce(paused_at, now()),
		    metadata = metadata || $3,
		    updated_at = now()
		where tenant_id = 'default'
		  and app_id = $1
		  and resource_version = $2
		  and status in ('released', 'rolling_out', 'testing')
		returning id::text
	`, appID, strings.TrimSpace(req.ToVersion), jsonb(map[string]any{
		"auto_paused_reason":  "activation_failed",
		"auto_paused_device":  firstNonBlank(req.DeviceID, req.DeviceKey),
		"auto_paused_package": req.PackageKey,
		"auto_paused_error":   req.ErrorMessage,
	})).Scan(&resourceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return err
	}
	return s.InsertAudit(ctx, "resource.auto_pause", "app_resource_version", resourceID, "资源激活失败，已自动暂停发布", map[string]any{
		"resource_version": req.ToVersion,
		"device_id":        firstNonBlank(req.DeviceID, req.DeviceKey),
		"package_key":      req.PackageKey,
		"error_message":    req.ErrorMessage,
	})
}

func (s *PostgresStore) Overview(ctx context.Context) (ReleaseOverview, error) {
	apps, err := s.listApps(ctx)
	if err != nil {
		return ReleaseOverview{}, err
	}
	releases, err := s.listReleases(ctx)
	if err != nil {
		return ReleaseOverview{}, err
	}
	resources, err := s.listResources(ctx)
	if err != nil {
		return ReleaseOverview{}, err
	}
	installations, err := s.listInstallations(ctx)
	if err != nil {
		return ReleaseOverview{}, err
	}
	return ReleaseOverview{
		Apps:             apps,
		Releases:         releases,
		ResourceVersions: resources,
		Installations:    installations,
	}, nil
}

func (s *PostgresStore) findOrCreateApp(ctx context.Context, cfg Config) (string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	appID, err := ensureApp(ctx, tx, cfg)
	if err != nil {
		return "", err
	}
	return appID, tx.Commit(ctx)
}

func ensureApp(ctx context.Context, tx pgx.Tx, cfg Config) (string, error) {
	var appID string
	err := tx.QueryRow(ctx, `
		insert into apps (id, tenant_id, app_key, name, platform, package_name, description, enabled, metadata)
		values ($1, 'default', $2, $3, 'android', $4, $5, true, $6)
		on conflict (tenant_id, app_key, platform) do update set
			name = excluded.name,
			package_name = excluded.package_name,
			enabled = true,
			metadata = apps.metadata || excluded.metadata,
			updated_at = now()
		returning id::text
	`, uuid.NewString(), cfg.AppKey, cfg.Name, cfg.PackageName,
		"Android Executor", jsonb(map[string]any{"source": "configured_release"})).Scan(&appID)
	return appID, err
}

func ensureBuild(ctx context.Context, tx pgx.Tx, appID string, cfg Config) (string, error) {
	var buildID string
	err := tx.QueryRow(ctx, `
		insert into app_builds (
			id, tenant_id, app_id, version_name, version_code, build_number,
			channel, build_type, build_status, artifact_type, artifact_path,
			artifact_size, sha256, metadata
		)
		values ($1, 'default', $2, $3, $4, $5, $6, $7, 'success', 'apk', $8, $9, $10, $11)
		on conflict (app_id, build_number) do update set
			version_name = excluded.version_name,
			version_code = excluded.version_code,
			channel = excluded.channel,
			build_type = excluded.build_type,
			build_status = 'success',
			artifact_path = excluded.artifact_path,
			artifact_size = excluded.artifact_size,
			sha256 = excluded.sha256,
			metadata = app_builds.metadata || excluded.metadata
		returning id::text
	`, uuid.NewString(), appID, cfg.LatestVersionName, cfg.LatestVersionCode,
		effectiveBuildNumber(cfg), cfg.Channel, cfg.BuildType, firstNonBlank(cfg.DownloadURL, cfg.APKURL),
		cfg.SizeBytes, cfg.SHA256, jsonb(map[string]any{"source": "env"})).Scan(&buildID)
	return buildID, err
}

func ensureRelease(ctx context.Context, tx pgx.Tx, appID, buildID string, cfg Config) error {
	updateLevel := configuredUpdateLevel(cfg)
	_, err := tx.Exec(ctx, `
		insert into app_releases (
			id, tenant_id, app_id, build_id, channel, build_type, status,
			title, summary, release_notes_markdown, upgrade_message,
			update_level, rollout_percentage, min_supported_code,
			block_old_versions, current_version_available, published_at, metadata
		)
		values ($1, 'default', $2, $3, $4, $5, 'released',
		        '游戏助手更新', $6, $7, $8, $9, 100, $10, $11, $12, now(), $13)
		on conflict (app_id, build_id, channel, build_type) do update set
			status = 'released',
			summary = excluded.summary,
			release_notes_markdown = excluded.release_notes_markdown,
			upgrade_message = excluded.upgrade_message,
			update_level = excluded.update_level,
			min_supported_code = excluded.min_supported_code,
			block_old_versions = excluded.block_old_versions,
			current_version_available = excluded.current_version_available,
			published_at = coalesce(app_releases.published_at, now()),
			metadata = app_releases.metadata || excluded.metadata,
			updated_at = now()
	`, uuid.NewString(), appID, buildID, cfg.Channel, cfg.BuildType,
		firstNonBlank(cfg.MessageZh, cfg.ReleaseNotes), cfg.ReleaseNotes, cfg.MessageZh,
		updateLevel, cfg.MinSupportedVersionCode, cfg.MinSupportedVersionCode > 0,
		cfg.CurrentVersionAvailable, jsonb(map[string]any{"source": "env", "unavailable_reason": cfg.UnavailableReason}))
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		insert into app_release_notes (id, tenant_id, release_id, language, title, summary, content_markdown, edited_by)
		select $1, 'default', r.id, 'zh-CN', '游戏助手更新', r.summary, r.release_notes_markdown, 'system'
		from app_releases r
		where r.app_id = $2 and r.build_id = $3 and r.channel = $4 and r.build_type = $5
		on conflict (release_id, language) do update set
			summary = excluded.summary,
			content_markdown = excluded.content_markdown,
			updated_at = now()
	`, uuid.NewString(), appID, buildID, cfg.Channel, cfg.BuildType)
	return err
}

func (s *PostgresStore) listResourcePackages(ctx context.Context, resourceVersionID string) ([]ResourcePackage, error) {
	rows, err := s.db.Query(ctx, `
		select package_key, package_type, file_url, file_size, sha256
		from app_resource_packages
		where resource_version_id = $1
		order by package_key asc
	`, resourceVersionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []ResourcePackage
	for rows.Next() {
		var pkg ResourcePackage
		if err := rows.Scan(&pkg.PackageKey, &pkg.PackageType, &pkg.URL, &pkg.Size, &pkg.SHA256); err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}
	return packages, rows.Err()
}

func (s *PostgresStore) listApps(ctx context.Context) ([]AppSummary, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, app_key, name, platform, package_name, enabled, updated_at
		from apps
		where tenant_id = 'default'
		order by updated_at desc
		limit 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []AppSummary
	for rows.Next() {
		var app AppSummary
		if err := rows.Scan(&app.AppID, &app.AppKey, &app.Name, &app.Platform, &app.PackageName, &app.Enabled, &app.UpdatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func (s *PostgresStore) listReleases(ctx context.Context) ([]ReleaseSummary, error) {
	rows, err := s.db.Query(ctx, `
		select r.id::text, a.app_key, b.version_name, b.version_code, b.build_number,
		       r.channel, r.build_type, r.status, r.update_level, r.min_supported_code,
		       b.artifact_path, b.sha256, b.artifact_size, coalesce(r.published_at, r.created_at)
		from app_releases r
		join apps a on a.id = r.app_id
		join app_builds b on b.id = r.build_id
		where r.tenant_id = 'default'
		order by b.version_code desc, b.build_number desc, r.updated_at desc
		limit 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var releases []ReleaseSummary
	for rows.Next() {
		var release ReleaseSummary
		if err := rows.Scan(
			&release.ReleaseID,
			&release.AppKey,
			&release.VersionName,
			&release.VersionCode,
			&release.BuildNumber,
			&release.Channel,
			&release.BuildType,
			&release.Status,
			&release.UpdateLevel,
			&release.MinSupportedVersionCode,
			&release.DownloadURL,
			&release.SHA256,
			&release.SizeBytes,
			&release.PublishedAt,
		); err != nil {
			return nil, err
		}
		releases = append(releases, release)
	}
	return releases, rows.Err()
}

func (s *PostgresStore) listResources(ctx context.Context) ([]ResourceSummary, error) {
	rows, err := s.db.Query(ctx, `
		select resource_version, channel, status, update_level, manifest_url, total_size,
		       coalesce(published_at, created_at)
		from app_resource_versions
		where tenant_id = 'default'
		order by coalesce(published_at, created_at) desc
		limit 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var resources []ResourceSummary
	for rows.Next() {
		var resource ResourceSummary
		if err := rows.Scan(&resource.ResourceVersion, &resource.Channel, &resource.Status,
			&resource.UpdateLevel, &resource.ManifestURL, &resource.TotalSize, &resource.PublishedAt); err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, rows.Err()
}

func (s *PostgresStore) listInstallations(ctx context.Context) ([]InstallationSummary, error) {
	rows, err := s.db.Query(ctx, `
		select device_id, device_key, device_name, installed_version, installed_code,
		       build_number, resource_version, coalesce(last_seen_at, updated_at),
		       last_upgrade_status, last_error
		from app_installations
		where tenant_id = 'default'
		order by coalesce(last_seen_at, updated_at) desc
		limit 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var installations []InstallationSummary
	for rows.Next() {
		var installation InstallationSummary
		if err := rows.Scan(&installation.DeviceID, &installation.DeviceKey, &installation.DeviceName,
			&installation.InstalledVersion, &installation.InstalledCode, &installation.BuildNumber,
			&installation.ResourceVersion, &installation.LastSeenAt, &installation.LastUpgradeStatus,
			&installation.LastError); err != nil {
			return nil, err
		}
		installations = append(installations, installation)
	}
	return installations, rows.Err()
}

func stringifyUserID(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return fmt.Sprint(v)
	}
}

func jsonb(value any) []byte {
	if value == nil {
		return []byte("{}")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return data
}
