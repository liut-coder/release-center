package appreleases

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresStore) BuildCenterOverview(ctx context.Context) (BuildCenterOverview, error) {
	projects, err := s.listBuildCenterProjects(ctx, "")
	if err != nil {
		return BuildCenterOverview{}, err
	}
	if err := s.attachBuildCenterRelations(ctx, projects); err != nil {
		return BuildCenterOverview{}, err
	}
	targets, err := s.DeploymentTargets(ctx)
	if err != nil {
		return BuildCenterOverview{}, err
	}
	return BuildCenterOverview{Projects: projects, DeploymentTargets: targets}, nil
}

func (s *PostgresStore) BuildCenterProject(ctx context.Context, projectKey string) (BuildCenterProject, error) {
	projects, err := s.listBuildCenterProjects(ctx, "and project_key = $1", projectKey)
	if err != nil {
		return BuildCenterProject{}, err
	}
	if len(projects) == 0 {
		return BuildCenterProject{}, pgx.ErrNoRows
	}
	if err := s.attachBuildCenterRelations(ctx, projects); err != nil {
		return BuildCenterProject{}, err
	}
	return projects[0], nil
}

func (s *PostgresStore) BuildCenterRun(ctx context.Context, runID string) (BuildCenterRunAdmin, error) {
	var run BuildCenterRunAdmin
	var metadata []byte
	err := s.db.QueryRow(ctx, `
		select id::text, project_id::text, coalesce(build_profile_id::text, ''),
		       coalesce(app_build_id::text, ''), trigger_type, trigger_source, action,
		       git_ref, git_commit, version_name, version_code, build_number, channel,
		       status, coalesce(exit_code, 0), started_by,
		       coalesce(started_at, '0001-01-01 00:00:00+00'::timestamptz),
		       coalesce(finished_at, '0001-01-01 00:00:00+00'::timestamptz),
		       duration_ms, workspace_dir, artifact_dir, log_dir, manifest_path,
		       upload_status, error_message, metadata, created_at, updated_at
		from build_center_runs
		where tenant_id = 'default' and id = $1::uuid
	`, runID).Scan(
		&run.ID, &run.ProjectID, &run.BuildProfileID,
		&run.AppBuildID, &run.TriggerType, &run.TriggerSource, &run.Action,
		&run.GitRef, &run.GitCommit, &run.VersionName, &run.VersionCode,
		&run.BuildNumber, &run.Channel, &run.Status, &run.ExitCode, &run.StartedBy,
		&run.StartedAt, &run.FinishedAt, &run.DurationMS, &run.WorkspaceDir,
		&run.ArtifactDir, &run.LogDir, &run.ManifestPath, &run.UploadStatus,
		&run.ErrorMessage, &metadata, &run.CreatedAt, &run.UpdatedAt,
	)
	if err != nil {
		return BuildCenterRunAdmin{}, err
	}
	run.Metadata = rawJSON(metadata, "{}")
	runs := []BuildCenterRunAdmin{run}
	if err := s.attachBuildCenterRunArtifacts(ctx, runs, map[string]int{run.ID: 0}); err != nil {
		return BuildCenterRunAdmin{}, err
	}
	return runs[0], nil
}

func (s *PostgresStore) DeploymentTargets(ctx context.Context) ([]DeploymentTargetAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, project_id::text, coalesce(app_id::text, ''),
		       target_key, name, provider, environment, endpoint_url,
		       cloudflare_account_id, cloudflare_project_name, cloudflare_script_name,
		       cloudflare_bucket_name, credential_ref, enabled, metadata, created_at, updated_at
		from deployment_targets
		where tenant_id = 'default'
		order by environment asc, provider asc, target_key asc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var targets []DeploymentTargetAdmin
	for rows.Next() {
		var item DeploymentTargetAdmin
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.AppID,
			&item.TargetKey, &item.Name, &item.Provider, &item.Environment, &item.EndpointURL,
			&item.CloudflareAccountID, &item.CloudflareProjectName, &item.CloudflareScriptName,
			&item.CloudflareBucketName, &item.CredentialRef, &item.Enabled, &metadata,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Metadata = rawJSON(metadata, "{}")
		targets = append(targets, item)
	}
	return targets, rows.Err()
}

func (s *PostgresStore) CreateBuildCenterProject(ctx context.Context, req BuildCenterProjectRequest) (BuildCenterProject, error) {
	_, err := s.db.Exec(ctx, `
		insert into release_projects (
		  tenant_id, project_key, name, description, owner_account,
		  lifecycle_status, default_channel, metadata
		)
		values ('default', $1, $2, $3, $4, $5, $6, $7)
		on conflict (tenant_id, project_key) do update
		set name = excluded.name,
		    description = excluded.description,
		    owner_account = excluded.owner_account,
		    lifecycle_status = excluded.lifecycle_status,
		    default_channel = excluded.default_channel,
		    metadata = release_projects.metadata || excluded.metadata,
		    updated_at = now()
	`, req.ProjectKey, req.Name, req.Description, req.OwnerAccount,
		req.LifecycleStatus, req.DefaultChannel, jsonb(req.Metadata))
	if err != nil {
		return BuildCenterProject{}, err
	}
	return s.BuildCenterProject(ctx, req.ProjectKey)
}

func (s *PostgresStore) UpsertCodeRepository(ctx context.Context, projectKey string, req CodeRepositoryRequest) (CodeRepositoryAdmin, error) {
	var item CodeRepositoryAdmin
	var metadata []byte
	err := s.db.QueryRow(ctx, `
		with project_row as (
		  select id from release_projects where tenant_id = 'default' and project_key = $1
		)
		insert into code_repositories (
		  tenant_id, project_id, provider, repo_url, repo_full_name,
		  default_ref, credential_ref, webhook_secret_ref,
		  webhook_enabled, trigger_on_push, trigger_on_tag, metadata
		)
		select 'default', project_row.id, $2, $3, $4,
		       $5, $6, $7, $8, $9, $10, $11
		from project_row
		on conflict (tenant_id, provider, repo_url) do update
		set project_id = excluded.project_id,
		    repo_full_name = excluded.repo_full_name,
		    default_ref = excluded.default_ref,
		    credential_ref = excluded.credential_ref,
		    webhook_secret_ref = excluded.webhook_secret_ref,
		    webhook_enabled = excluded.webhook_enabled,
		    trigger_on_push = excluded.trigger_on_push,
		    trigger_on_tag = excluded.trigger_on_tag,
		    metadata = code_repositories.metadata || excluded.metadata,
		    updated_at = now()
		returning id::text, project_id::text, provider, repo_url, repo_full_name,
		          default_ref, credential_ref, webhook_secret_ref, webhook_enabled,
		          trigger_on_push, trigger_on_tag, metadata, created_at, updated_at
	`, projectKey, req.Provider, req.RepoURL, req.RepoFullName,
		req.DefaultRef, req.CredentialRef, req.WebhookSecretRef,
		boolValue(req.WebhookEnabled, false), boolValue(req.TriggerOnPush, false),
		boolValue(req.TriggerOnTag, false), jsonb(req.Metadata)).Scan(
		&item.ID, &item.ProjectID, &item.Provider, &item.RepoURL,
		&item.RepoFullName, &item.DefaultRef, &item.CredentialRef,
		&item.WebhookSecretRef, &item.WebhookEnabled, &item.TriggerOnPush,
		&item.TriggerOnTag, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return CodeRepositoryAdmin{}, err
	}
	item.Metadata = rawJSON(metadata, "{}")
	return item, nil
}

func (s *PostgresStore) UpsertBuildProfile(ctx context.Context, projectKey string, req BuildProfileRequest) (BuildProfileAdmin, error) {
	enabled := boolValue(req.Enabled, true)
	var item BuildProfileAdmin
	var commands, artifactRules, metadata []byte
	err := s.db.QueryRow(ctx, `
		with project_row as (
		  select id from release_projects where tenant_id = 'default' and project_key = $1
		)
		insert into build_profiles (
		  tenant_id, project_id, app_id, profile_key, name, build_center_project,
		  stack_type, build_type, config_path, source_workdir, default_ref,
		  default_version_name, default_version_code, default_channel,
		  build_action, commands, artifact_rules, enabled, metadata
		)
		select 'default', project_row.id, $2::uuid, $3, $4, $5,
		       $6, $7, $8, $9, $10, $11, $12, $13,
		       $14, $15, $16, $17, $18
		from project_row
		on conflict (project_id, profile_key) do update
		set app_id = excluded.app_id,
		    name = excluded.name,
		    build_center_project = excluded.build_center_project,
		    stack_type = excluded.stack_type,
		    build_type = excluded.build_type,
		    config_path = excluded.config_path,
		    source_workdir = excluded.source_workdir,
		    default_ref = excluded.default_ref,
		    default_version_name = excluded.default_version_name,
		    default_version_code = excluded.default_version_code,
		    default_channel = excluded.default_channel,
		    build_action = excluded.build_action,
		    commands = excluded.commands,
		    artifact_rules = excluded.artifact_rules,
		    enabled = excluded.enabled,
		    metadata = build_profiles.metadata || excluded.metadata,
		    updated_at = now()
		returning id::text, project_id::text, coalesce(app_id::text, ''),
		          profile_key, name, build_center_project, stack_type,
		          build_type, config_path, source_workdir, default_ref,
		          default_version_name, default_version_code, default_channel,
		          build_action, commands, artifact_rules, enabled,
		          metadata, created_at, updated_at
	`, projectKey, nullableUUID(req.AppID), req.ProfileKey, req.Name,
		req.BuildCenterProject, req.StackType, req.BuildType, req.ConfigPath,
		req.SourceWorkdir, req.DefaultRef, req.DefaultVersionName,
		req.DefaultVersionCode, req.DefaultChannel, req.BuildAction,
		jsonb(req.Commands), jsonb(req.ArtifactRules), enabled, jsonb(req.Metadata)).Scan(
		&item.ID, &item.ProjectID, &item.AppID, &item.ProfileKey,
		&item.Name, &item.BuildCenterProject, &item.StackType,
		&item.BuildType, &item.ConfigPath, &item.SourceWorkdir,
		&item.DefaultRef, &item.DefaultVersionName, &item.DefaultVersionCode,
		&item.DefaultChannel, &item.BuildAction, &commands, &artifactRules,
		&item.Enabled, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return BuildProfileAdmin{}, err
	}
	item.Commands = rawJSON(commands, "{}")
	item.ArtifactRules = rawJSON(artifactRules, "[]")
	item.Metadata = rawJSON(metadata, "{}")
	return item, nil
}

func (s *PostgresStore) UpsertWebhookRoute(ctx context.Context, projectKey string, req WebhookRouteRequest) (WebhookRouteAdmin, error) {
	enabled := boolValue(req.Enabled, true)
	var item WebhookRouteAdmin
	var metadata []byte
	err := s.db.QueryRow(ctx, `
		with project_row as (
		  select id from release_projects where tenant_id = 'default' and project_key = $1
		),
		repo_row as (
		  select cr.id
		  from code_repositories cr
		  join project_row on project_row.id = cr.project_id
		  where cr.tenant_id = 'default' and cr.id = $2::uuid
		),
		profile_row as (
		  select bp.id, bp.profile_key
		  from build_profiles bp
		  join project_row on project_row.id = bp.project_id
		  where bp.tenant_id = 'default' and bp.profile_key = nullif($3, '')
		)
		insert into webhook_routes (
		  tenant_id, project_id, repository_id, build_profile_id,
		  event_type, ref_pattern, action, enabled, metadata
		)
		select 'default', project_row.id, repo_row.id,
		       (select id from profile_row), $4, $5, $6, $7, $8
		from project_row, repo_row
		on conflict (repository_id, event_type, ref_pattern, action) do update
		set build_profile_id = excluded.build_profile_id,
		    enabled = excluded.enabled,
		    metadata = webhook_routes.metadata || excluded.metadata,
		    updated_at = now()
		returning id::text, project_id::text, repository_id::text,
		          coalesce(build_profile_id::text, ''),
		          coalesce((select profile_key from profile_row), ''),
		          event_type, ref_pattern, action, enabled,
		          metadata, created_at, updated_at
	`, projectKey, req.RepositoryID, req.ProfileKey, req.EventType,
		req.RefPattern, req.Action, enabled, jsonb(req.Metadata)).Scan(
		&item.ID, &item.ProjectID, &item.RepositoryID, &item.BuildProfileID,
		&item.ProfileKey, &item.EventType, &item.RefPattern, &item.Action,
		&item.Enabled, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return WebhookRouteAdmin{}, err
	}
	item.Metadata = rawJSON(metadata, "{}")
	return item, nil
}

func (s *PostgresStore) CreateBuildCenterRun(ctx context.Context, projectKey string, req BuildCenterRunRequest) (BuildCenterRunAdmin, BuildProfileAdmin, error) {
	profile, err := s.buildProfileForRun(ctx, projectKey, req.ProfileKey)
	if err != nil {
		return BuildCenterRunAdmin{}, BuildProfileAdmin{}, err
	}
	action := firstNonBlank(req.Action, profile.BuildAction, "all")
	gitRef := firstNonBlank(req.GitRef, profile.DefaultRef)
	versionName := firstNonBlank(req.VersionName, profile.DefaultVersionName)
	versionCode := req.VersionCode
	if versionCode <= 0 {
		versionCode = profile.DefaultVersionCode
	}
	channel := firstNonBlank(req.Channel, profile.DefaultChannel, "dev")
	startedBy := firstNonBlank(req.StartedBy, "admin")

	var run BuildCenterRunAdmin
	err = s.db.QueryRow(ctx, `
		insert into build_center_runs (
		  tenant_id, project_id, build_profile_id, trigger_type, trigger_source,
		  action, git_ref, version_name, version_code, channel, status,
		  started_by, upload_status, metadata
		)
		values (
		  'default', $1::uuid, $2::uuid, 'manual', 'admin',
		  $3, $4, $5, $6, $7, 'queued',
		  $8, 'pending', $9
		)
		returning id::text, project_id::text, build_profile_id::text, trigger_type,
		          trigger_source, action, git_ref, git_commit, version_name, version_code,
		          build_number, channel, status, coalesce(exit_code, 0), started_by,
		          coalesce(started_at, '0001-01-01 00:00:00+00'::timestamptz),
		          coalesce(finished_at, '0001-01-01 00:00:00+00'::timestamptz),
		          duration_ms, workspace_dir, artifact_dir, log_dir, manifest_path,
		          upload_status, error_message, metadata, created_at, updated_at
	`, profile.ProjectID, profile.ID, action, gitRef, versionName, versionCode, channel, startedBy,
		jsonb(map[string]any{
			"profile_key": profile.ProfileKey,
			"source":      "admin_api",
		})).Scan(
		&run.ID, &run.ProjectID, &run.BuildProfileID, &run.TriggerType,
		&run.TriggerSource, &run.Action, &run.GitRef, &run.GitCommit, &run.VersionName,
		&run.VersionCode, &run.BuildNumber, &run.Channel, &run.Status, &run.ExitCode,
		&run.StartedBy, &run.StartedAt, &run.FinishedAt, &run.DurationMS,
		&run.WorkspaceDir, &run.ArtifactDir, &run.LogDir, &run.ManifestPath,
		&run.UploadStatus, &run.ErrorMessage, &run.Metadata, &run.CreatedAt, &run.UpdatedAt,
	)
	if err != nil {
		return BuildCenterRunAdmin{}, BuildProfileAdmin{}, err
	}
	return run, profile, nil
}

func (s *PostgresStore) MarkBuildCenterRunStarted(ctx context.Context, runID, logDir string) error {
	_, err := s.db.Exec(ctx, `
		update build_center_runs
		set status = 'running',
		    started_at = coalesce(started_at, now()),
		    log_dir = $2,
		    updated_at = now()
		where tenant_id = 'default' and id = $1::uuid
	`, runID, logDir)
	return err
}

func (s *PostgresStore) CompleteBuildCenterRun(ctx context.Context, runID string, patch BuildCenterRunPatch) (BuildCenterRunAdmin, error) {
	if patch.UploadStatus == "" {
		patch.UploadStatus = "pending"
	}
	var run BuildCenterRunAdmin
	var metadata []byte
	err := s.db.QueryRow(ctx, `
		update build_center_runs
		set status = $2,
		    exit_code = $3,
		    git_commit = coalesce(nullif($4, ''), git_commit),
		    version_name = coalesce(nullif($5, ''), version_name),
		    version_code = case when $6 > 0 then $6 else version_code end,
		    build_number = case when $7 > 0 then $7 else build_number end,
		    artifact_dir = coalesce(nullif($8, ''), artifact_dir),
		    log_dir = coalesce(nullif($9, ''), log_dir),
		    manifest_path = coalesce(nullif($10, ''), manifest_path),
		    upload_status = $11,
		    error_message = $12,
		    duration_ms = $13,
		    finished_at = now(),
		    metadata = metadata || $14::jsonb,
		    updated_at = now()
		where tenant_id = 'default' and id = $1::uuid
		returning id::text, project_id::text, coalesce(build_profile_id::text, ''),
		          coalesce(app_build_id::text, ''), trigger_type, trigger_source, action,
		          git_ref, git_commit, version_name, version_code, build_number,
		          channel, status, coalesce(exit_code, 0), started_by,
		          coalesce(started_at, '0001-01-01 00:00:00+00'::timestamptz),
		          coalesce(finished_at, '0001-01-01 00:00:00+00'::timestamptz),
		          duration_ms, workspace_dir, artifact_dir, log_dir, manifest_path,
		          upload_status, error_message, metadata, created_at, updated_at
	`, runID, patch.Status, patch.ExitCode, patch.GitCommit, patch.VersionName,
		patch.VersionCode, patch.BuildNumber, patch.ArtifactDir, patch.LogDir,
		patch.ManifestPath, patch.UploadStatus, patch.ErrorMessage, patch.DurationMS,
		jsonb(patch.Metadata)).Scan(
		&run.ID, &run.ProjectID, &run.BuildProfileID, &run.AppBuildID, &run.TriggerType,
		&run.TriggerSource, &run.Action, &run.GitRef, &run.GitCommit, &run.VersionName,
		&run.VersionCode, &run.BuildNumber, &run.Channel, &run.Status, &run.ExitCode,
		&run.StartedBy, &run.StartedAt, &run.FinishedAt, &run.DurationMS,
		&run.WorkspaceDir, &run.ArtifactDir, &run.LogDir, &run.ManifestPath,
		&run.UploadStatus, &run.ErrorMessage, &metadata, &run.CreatedAt, &run.UpdatedAt,
	)
	if err != nil {
		return BuildCenterRunAdmin{}, err
	}
	run.Metadata = rawJSON(metadata, "{}")
	return run, nil
}

func (s *PostgresStore) ReplaceBuildCenterRunArtifacts(ctx context.Context, runID string, artifacts []BuildCenterRunArtifact) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `delete from build_center_run_artifacts where tenant_id = 'default' and run_id = $1::uuid`, runID); err != nil {
		return err
	}
	for _, artifact := range artifacts {
		if _, err := tx.Exec(ctx, `
			insert into build_center_run_artifacts (
			  tenant_id, run_id, name, artifact_type, file_name, local_path,
			  size_bytes, sha256, upload_status, download_url, metadata
			)
			values ('default', $1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, runID, artifact.Name, artifact.ArtifactType, artifact.FileName, artifact.LocalPath,
			artifact.SizeBytes, artifact.SHA256, artifact.UploadStatus, artifact.DownloadURL,
			jsonb(map[string]any{"source": "buildctl_status"})); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *PostgresStore) MatchWebhookBuildRoutes(ctx context.Context, event WebhookEventRequest) ([]WebhookBuildRoute, error) {
	if event.Repository == "" {
		return []WebhookBuildRoute{}, nil
	}
	rows, err := s.db.Query(ctx, `
		select wr.id::text, p.project_key, cr.repo_full_name,
		       coalesce(bp.profile_key, 'default'), wr.event_type,
		       wr.ref_pattern, wr.action, cr.trigger_on_push, cr.trigger_on_tag
		from webhook_routes wr
		join code_repositories cr on cr.id = wr.repository_id and cr.tenant_id = wr.tenant_id
		join release_projects p on p.id = wr.project_id and p.tenant_id = wr.tenant_id
		left join build_profiles bp on bp.id = wr.build_profile_id and bp.tenant_id = wr.tenant_id
		where wr.tenant_id = 'default'
		  and wr.enabled = true
		  and cr.webhook_enabled = true
		  and cr.provider = $1
		  and (wr.event_type = $2 or wr.event_type = '*')
		  and (
		    cr.repo_full_name = $3
		    or cr.repo_url = $3
		    or cr.repo_url like '%' || $3 || '%'
		  )
		order by wr.created_at asc
	`, event.Provider, event.EventType, event.Repository)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var routes []WebhookBuildRoute
	for rows.Next() {
		var route WebhookBuildRoute
		if err := rows.Scan(&route.ID, &route.ProjectKey, &route.Repository,
			&route.ProfileKey, &route.EventType, &route.RefPattern, &route.Action,
			&route.TriggerOnPush, &route.TriggerOnTag); err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	return routes, rows.Err()
}

func (s *PostgresStore) buildProfileForRun(ctx context.Context, projectKey, profileKey string) (BuildProfileAdmin, error) {
	if profileKey == "" {
		profileKey = "default"
	}
	var item BuildProfileAdmin
	var commands, artifactRules, metadata []byte
	err := s.db.QueryRow(ctx, `
		select bp.id::text, bp.project_id::text, coalesce(bp.app_id::text, ''),
		       bp.profile_key, bp.name, bp.build_center_project, bp.stack_type,
		       bp.build_type, bp.config_path, bp.source_workdir, bp.default_ref,
		       bp.default_version_name, bp.default_version_code, bp.default_channel,
		       bp.build_action, bp.commands, bp.artifact_rules, bp.enabled,
		       bp.metadata, bp.created_at, bp.updated_at
		from build_profiles bp
		join release_projects p on p.id = bp.project_id and p.tenant_id = bp.tenant_id
		where bp.tenant_id = 'default'
		  and p.project_key = $1
		  and bp.profile_key = $2
		  and bp.enabled = true
	`, projectKey, profileKey).Scan(
		&item.ID, &item.ProjectID, &item.AppID, &item.ProfileKey, &item.Name,
		&item.BuildCenterProject, &item.StackType, &item.BuildType, &item.ConfigPath,
		&item.SourceWorkdir, &item.DefaultRef, &item.DefaultVersionName,
		&item.DefaultVersionCode, &item.DefaultChannel, &item.BuildAction,
		&commands, &artifactRules, &item.Enabled, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return BuildProfileAdmin{}, fmt.Errorf("build profile not found: %w", err)
	}
	item.Commands = rawJSON(commands, "{}")
	item.ArtifactRules = rawJSON(artifactRules, "[]")
	item.Metadata = rawJSON(metadata, "{}")
	return item, nil
}

func (s *PostgresStore) listBuildCenterProjects(ctx context.Context, extraWhere string, args ...any) ([]BuildCenterProject, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, project_key, name, description, owner_account,
		       lifecycle_status, default_channel, metadata, created_at, updated_at
		from release_projects
		where tenant_id = 'default' `+extraWhere+`
		order by updated_at desc, project_key asc
		limit 100
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projects []BuildCenterProject
	for rows.Next() {
		var item BuildCenterProject
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.ProjectKey, &item.Name, &item.Description,
			&item.OwnerAccount, &item.LifecycleStatus, &item.DefaultChannel, &metadata,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Metadata = rawJSON(metadata, "{}")
		projects = append(projects, item)
	}
	return projects, rows.Err()
}

func (s *PostgresStore) attachBuildCenterRelations(ctx context.Context, projects []BuildCenterProject) error {
	if len(projects) == 0 {
		return nil
	}
	index := map[string]int{}
	ids := make([]string, 0, len(projects))
	for i, project := range projects {
		index[project.ID] = i
		ids = append(ids, project.ID)
	}
	if err := s.attachCodeRepositories(ctx, projects, index, ids); err != nil {
		return err
	}
	if err := s.attachBuildProfiles(ctx, projects, index, ids); err != nil {
		return err
	}
	if err := s.attachWebhookRoutes(ctx, projects, index, ids); err != nil {
		return err
	}
	if err := s.attachBuildCenterRuns(ctx, projects, index, ids); err != nil {
		return err
	}
	if err := s.attachDeploymentTargets(ctx, projects, index, ids); err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) attachCodeRepositories(ctx context.Context, projects []BuildCenterProject, index map[string]int, ids []string) error {
	rows, err := s.db.Query(ctx, `
		select id::text, project_id::text, provider, repo_url, repo_full_name,
		       default_ref, credential_ref, webhook_secret_ref, webhook_enabled,
		       trigger_on_push, trigger_on_tag, metadata, created_at, updated_at
		from code_repositories
		where tenant_id = 'default' and project_id::text = any($1)
		order by provider asc, repo_full_name asc
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item CodeRepositoryAdmin
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Provider, &item.RepoURL,
			&item.RepoFullName, &item.DefaultRef, &item.CredentialRef, &item.WebhookSecretRef,
			&item.WebhookEnabled, &item.TriggerOnPush, &item.TriggerOnTag, &metadata,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return err
		}
		item.Metadata = rawJSON(metadata, "{}")
		if i, ok := index[item.ProjectID]; ok {
			projects[i].Repositories = append(projects[i].Repositories, item)
		}
	}
	return rows.Err()
}

func (s *PostgresStore) attachBuildProfiles(ctx context.Context, projects []BuildCenterProject, index map[string]int, ids []string) error {
	rows, err := s.db.Query(ctx, `
		select id::text, project_id::text, coalesce(app_id::text, ''),
		       profile_key, name, build_center_project, stack_type, build_type,
		       config_path, source_workdir, default_ref, default_version_name,
		       default_version_code, default_channel, build_action,
		       commands, artifact_rules, enabled, metadata, created_at, updated_at
		from build_profiles
		where tenant_id = 'default' and project_id::text = any($1)
		order by enabled desc, profile_key asc
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item BuildProfileAdmin
		var commands, artifactRules, metadata []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.AppID,
			&item.ProfileKey, &item.Name, &item.BuildCenterProject, &item.StackType, &item.BuildType,
			&item.ConfigPath, &item.SourceWorkdir, &item.DefaultRef, &item.DefaultVersionName,
			&item.DefaultVersionCode, &item.DefaultChannel, &item.BuildAction,
			&commands, &artifactRules, &item.Enabled, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return err
		}
		item.Commands = rawJSON(commands, "{}")
		item.ArtifactRules = rawJSON(artifactRules, "[]")
		item.Metadata = rawJSON(metadata, "{}")
		if i, ok := index[item.ProjectID]; ok {
			projects[i].BuildProfiles = append(projects[i].BuildProfiles, item)
		}
	}
	return rows.Err()
}

func (s *PostgresStore) attachWebhookRoutes(ctx context.Context, projects []BuildCenterProject, index map[string]int, ids []string) error {
	rows, err := s.db.Query(ctx, `
		select wr.id::text, wr.project_id::text, wr.repository_id::text,
		       coalesce(wr.build_profile_id::text, ''),
		       coalesce(bp.profile_key, ''), wr.event_type, wr.ref_pattern,
		       wr.action, wr.enabled, wr.metadata, wr.created_at, wr.updated_at
		from webhook_routes wr
		left join build_profiles bp on bp.id = wr.build_profile_id and bp.tenant_id = wr.tenant_id
		where wr.tenant_id = 'default' and wr.project_id::text = any($1)
		order by wr.enabled desc, wr.event_type asc, wr.ref_pattern asc
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item WebhookRouteAdmin
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.RepositoryID,
			&item.BuildProfileID, &item.ProfileKey, &item.EventType,
			&item.RefPattern, &item.Action, &item.Enabled, &metadata,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return err
		}
		item.Metadata = rawJSON(metadata, "{}")
		if i, ok := index[item.ProjectID]; ok {
			projects[i].WebhookRoutes = append(projects[i].WebhookRoutes, item)
		}
	}
	return rows.Err()
}

func (s *PostgresStore) attachBuildCenterRuns(ctx context.Context, projects []BuildCenterProject, index map[string]int, ids []string) error {
	rows, err := s.db.Query(ctx, `
		select id::text, project_id::text, coalesce(build_profile_id::text, ''),
		       coalesce(app_build_id::text, ''), trigger_type, trigger_source, action,
		       git_ref, git_commit, version_name, version_code, build_number, channel,
		       status, coalesce(exit_code, 0), started_by,
		       coalesce(started_at, '0001-01-01 00:00:00+00'::timestamptz),
		       coalesce(finished_at, '0001-01-01 00:00:00+00'::timestamptz),
		       duration_ms, workspace_dir, artifact_dir, log_dir, manifest_path,
		       upload_status, error_message, metadata, created_at, updated_at
		from build_center_runs
		where tenant_id = 'default' and project_id::text = any($1)
		order by created_at desc
		limit 100
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	runs := []BuildCenterRunAdmin{}
	runIndex := map[string]int{}
	for rows.Next() {
		var item BuildCenterRunAdmin
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.BuildProfileID,
			&item.AppBuildID, &item.TriggerType, &item.TriggerSource, &item.Action,
			&item.GitRef, &item.GitCommit, &item.VersionName, &item.VersionCode,
			&item.BuildNumber, &item.Channel, &item.Status, &item.ExitCode, &item.StartedBy,
			&item.StartedAt, &item.FinishedAt, &item.DurationMS, &item.WorkspaceDir,
			&item.ArtifactDir, &item.LogDir, &item.ManifestPath, &item.UploadStatus,
			&item.ErrorMessage, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return err
		}
		item.Metadata = rawJSON(metadata, "{}")
		runIndex[item.ID] = len(runs)
		runs = append(runs, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := s.attachBuildCenterRunArtifacts(ctx, runs, runIndex); err != nil {
		return err
	}
	for _, run := range runs {
		if i, ok := index[run.ProjectID]; ok {
			projects[i].RecentRuns = append(projects[i].RecentRuns, run)
		}
	}
	return nil
}

func (s *PostgresStore) attachBuildCenterRunArtifacts(ctx context.Context, runs []BuildCenterRunAdmin, runIndex map[string]int) error {
	if len(runs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(runs))
	for _, run := range runs {
		ids = append(ids, run.ID)
	}
	rows, err := s.db.Query(ctx, `
		select id::text, run_id::text, coalesce(app_build_artifact_id::text, ''),
		       name, artifact_type, file_name, local_path, size_bytes, sha256,
		       upload_status, download_url, metadata, created_at, updated_at
		from build_center_run_artifacts
		where tenant_id = 'default' and run_id::text = any($1)
		order by created_at asc, name asc
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item BuildCenterRunArtifact
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.RunID, &item.AppBuildArtifactID,
			&item.Name, &item.ArtifactType, &item.FileName, &item.LocalPath, &item.SizeBytes,
			&item.SHA256, &item.UploadStatus, &item.DownloadURL, &metadata,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return err
		}
		item.Metadata = rawJSON(metadata, "{}")
		if i, ok := runIndex[item.RunID]; ok {
			runs[i].Artifacts = append(runs[i].Artifacts, item)
		}
	}
	return rows.Err()
}

func (s *PostgresStore) attachDeploymentTargets(ctx context.Context, projects []BuildCenterProject, index map[string]int, ids []string) error {
	rows, err := s.db.Query(ctx, `
		select id::text, project_id::text, coalesce(app_id::text, ''),
		       target_key, name, provider, environment, endpoint_url,
		       cloudflare_account_id, cloudflare_project_name, cloudflare_script_name,
		       cloudflare_bucket_name, credential_ref, enabled, metadata, created_at, updated_at
		from deployment_targets
		where tenant_id = 'default' and project_id::text = any($1)
		order by provider asc, target_key asc
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item DeploymentTargetAdmin
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.AppID,
			&item.TargetKey, &item.Name, &item.Provider, &item.Environment, &item.EndpointURL,
			&item.CloudflareAccountID, &item.CloudflareProjectName, &item.CloudflareScriptName,
			&item.CloudflareBucketName, &item.CredentialRef, &item.Enabled, &metadata,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return err
		}
		item.Metadata = rawJSON(metadata, "{}")
		if i, ok := index[item.ProjectID]; ok {
			projects[i].DeploymentTargets = append(projects[i].DeploymentTargets, item)
		}
	}
	return rows.Err()
}

func rawJSON(data []byte, fallback string) json.RawMessage {
	if len(data) == 0 {
		return json.RawMessage(fallback)
	}
	return json.RawMessage(data)
}
