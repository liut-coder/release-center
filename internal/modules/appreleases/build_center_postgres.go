package appreleases

import (
	"context"
	"encoding/json"

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
