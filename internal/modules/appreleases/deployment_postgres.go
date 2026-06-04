package appreleases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresStore) CreateDeploymentTarget(ctx context.Context, req CreateDeploymentTargetRequest) (DeploymentTargetAdmin, error) {
	enabled := false
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var target DeploymentTargetAdmin
	var metadata []byte
	err := s.db.QueryRow(ctx, `
		with project_row as (
		  select id from release_projects where tenant_id = 'default' and project_key = $1
		)
		insert into deployment_targets (
		  tenant_id, project_id, app_id, target_key, name, provider, environment,
		  endpoint_url, cloudflare_account_id, cloudflare_project_name,
		  cloudflare_script_name, cloudflare_bucket_name, credential_ref,
		  enabled, metadata
		)
		select 'default', project_row.id, nullif($2, '')::uuid, $3, $4, $5, $6,
		       $7, $8, $9, $10, $11, $12, $13, $14
		from project_row
		on conflict (project_id, target_key) do update
		set app_id = excluded.app_id,
		    name = excluded.name,
		    provider = excluded.provider,
		    environment = excluded.environment,
		    endpoint_url = excluded.endpoint_url,
		    cloudflare_account_id = excluded.cloudflare_account_id,
		    cloudflare_project_name = excluded.cloudflare_project_name,
		    cloudflare_script_name = excluded.cloudflare_script_name,
		    cloudflare_bucket_name = excluded.cloudflare_bucket_name,
		    credential_ref = excluded.credential_ref,
		    enabled = excluded.enabled,
		    metadata = deployment_targets.metadata || excluded.metadata,
		    updated_at = now()
		returning id::text, project_id::text, coalesce(app_id::text, ''),
		          target_key, name, provider, environment, endpoint_url,
		          cloudflare_account_id, cloudflare_project_name, cloudflare_script_name,
		          cloudflare_bucket_name, credential_ref, enabled, metadata, created_at, updated_at
	`, req.ProjectKey, req.AppID, req.TargetKey, req.Name, req.Provider, req.Environment,
		req.EndpointURL, req.CloudflareAccountID, req.CloudflareProjectName,
		req.CloudflareScriptName, req.CloudflareBucketName, req.CredentialRef,
		enabled, jsonb(req.Metadata)).Scan(
		&target.ID, &target.ProjectID, &target.AppID, &target.TargetKey, &target.Name,
		&target.Provider, &target.Environment, &target.EndpointURL,
		&target.CloudflareAccountID, &target.CloudflareProjectName, &target.CloudflareScriptName,
		&target.CloudflareBucketName, &target.CredentialRef, &target.Enabled, &metadata,
		&target.CreatedAt, &target.UpdatedAt,
	)
	if err != nil {
		return DeploymentTargetAdmin{}, err
	}
	target.Metadata = rawJSON(metadata, "{}")
	return target, nil
}

func (s *PostgresStore) DeploymentRecords(ctx context.Context) ([]DeploymentRecordAdmin, error) {
	rows, err := s.db.Query(ctx, deploymentRecordSelectSQL()+`
		where dr.tenant_id = 'default'
		order by dr.created_at desc
		limit 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []DeploymentRecordAdmin
	for rows.Next() {
		record, err := scanDeploymentRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *PostgresStore) DeploymentRecord(ctx context.Context, deploymentID string) (DeploymentRecordAdmin, error) {
	row := s.db.QueryRow(ctx, deploymentRecordSelectSQL()+`
		where dr.tenant_id = 'default' and dr.id = $1::uuid
	`, deploymentID)
	return scanDeploymentRecord(row)
}

func (s *PostgresStore) CreateDeploymentRecord(ctx context.Context, req CreateDeploymentRequest) (DeploymentRecordAdmin, error) {
	target, err := s.deploymentTargetForRequest(ctx, req.TargetID, req.ProjectKey, req.TargetKey)
	if err != nil {
		return DeploymentRecordAdmin{}, err
	}
	if !target.Enabled && !req.DryRun {
		return DeploymentRecordAdmin{}, fmt.Errorf("deployment target is disabled")
	}
	metadata := mergeMaps(req.Metadata, map[string]any{
		"provider":    target.Provider,
		"environment": target.Environment,
	})
	if strings.HasPrefix(target.Provider, "cloudflare_") {
		command := cloudflareDeploymentCommand(target, CreateDeploymentRequest{Metadata: metadata})
		if len(command) > 0 {
			metadata["prepared_command"] = command
		}
	}
	logTail := req.LogTail
	if req.DryRun && len(logTail) == 0 {
		logTail = []string{"dry-run deployment record created"}
	}
	startedAt, finishedAt := deploymentTimestamps(req.ProviderStatus)
	row := s.db.QueryRow(ctx, deploymentRecordInsertReturningSQL(), target.ID,
		nullableUUID(req.RunID), nullableUUID(req.AppBuildID), nullableUUID(req.AppBuildArtifactID),
		target.Provider, req.ExternalDeploymentID, req.ProviderStatus, req.DeploymentURL,
		req.VersionName, req.BuildNumber, req.GitCommit, req.TriggeredBy,
		startedAt, finishedAt, logTail, req.ErrorMessage, jsonb(metadata))
	return scanDeploymentRecord(row)
}

func (s *PostgresStore) UpdateDeploymentRecordStatus(ctx context.Context, deploymentID string, req UpdateDeploymentStatusRequest) (DeploymentRecordAdmin, error) {
	row := s.db.QueryRow(ctx, deploymentRecordUpdateReturningSQL(`
		set external_deployment_id = coalesce(nullif($2, ''), external_deployment_id),
		    provider_status = $3,
		    deployment_url = coalesce(nullif($4, ''), deployment_url),
		    log_tail = case when cardinality($5::text[]) > 0 then $5::text[] else log_tail end,
		    error_message = $6,
		    metadata = metadata || $7::jsonb,
		    finished_at = now(),
		    duration_ms = case when started_at is null then duration_ms else greatest(extract(epoch from (now() - started_at))::bigint * 1000, 0) end,
		    updated_at = now()
		where dr.tenant_id = 'default' and dr.id = $1::uuid
	`), deploymentID, req.ExternalDeploymentID, req.ProviderStatus, req.DeploymentURL,
		req.LogTail, req.ErrorMessage, jsonb(req.Metadata))
	return scanDeploymentRecord(row)
}

func (s *PostgresStore) deploymentTargetForRequest(ctx context.Context, targetID, projectKey, targetKey string) (DeploymentTargetAdmin, error) {
	var target DeploymentTargetAdmin
	var metadata []byte
	args := []any{}
	where := ""
	if targetID != "" {
		where = "dt.id = $1::uuid"
		args = append(args, targetID)
	} else {
		where = "p.project_key = $1 and dt.target_key = $2"
		args = append(args, projectKey, targetKey)
	}
	err := s.db.QueryRow(ctx, `
		select dt.id::text, dt.project_id::text, coalesce(dt.app_id::text, ''),
		       dt.target_key, dt.name, dt.provider, dt.environment, dt.endpoint_url,
		       dt.cloudflare_account_id, dt.cloudflare_project_name, dt.cloudflare_script_name,
		       dt.cloudflare_bucket_name, dt.credential_ref, dt.enabled, dt.metadata,
		       dt.created_at, dt.updated_at
		from deployment_targets dt
		join release_projects p on p.id = dt.project_id and p.tenant_id = dt.tenant_id
		where dt.tenant_id = 'default' and `+where, args...).Scan(
		&target.ID, &target.ProjectID, &target.AppID, &target.TargetKey, &target.Name,
		&target.Provider, &target.Environment, &target.EndpointURL,
		&target.CloudflareAccountID, &target.CloudflareProjectName, &target.CloudflareScriptName,
		&target.CloudflareBucketName, &target.CredentialRef, &target.Enabled, &metadata,
		&target.CreatedAt, &target.UpdatedAt,
	)
	if err != nil {
		return DeploymentTargetAdmin{}, err
	}
	target.Metadata = rawJSON(metadata, "{}")
	return target, nil
}

func deploymentTimestamps(status string) (any, any) {
	switch status {
	case "success", "failed", "canceled", "dry_run", "external":
		now := time.Now()
		return now, now
	case "running":
		return time.Now(), nil
	default:
		return nil, nil
	}
}

func nullableUUID(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func deploymentRecordSelectSQL() string {
	return `
		select dr.id::text, dr.target_id::text, dt.target_key, dt.name, dt.project_id::text,
		       coalesce(dr.run_id::text, ''), coalesce(dr.app_build_id::text, ''),
		       coalesce(dr.app_build_artifact_id::text, ''), dr.provider, dt.environment,
		       dr.external_deployment_id, dr.provider_status, dr.deployment_url,
		       dr.version_name, dr.build_number, dr.git_commit, dr.triggered_by,
		       coalesce(dr.started_at, '0001-01-01 00:00:00+00'::timestamptz),
		       coalesce(dr.finished_at, '0001-01-01 00:00:00+00'::timestamptz),
		       dr.duration_ms, dr.log_tail, dr.error_message, dr.metadata,
		       dr.created_at, dr.updated_at
		from deployment_records dr
		join deployment_targets dt on dt.id = dr.target_id and dt.tenant_id = dr.tenant_id
	`
}

func deploymentRecordInsertReturningSQL() string {
	return `
		with inserted as (
		  insert into deployment_records (
		    tenant_id, target_id, run_id, app_build_id, app_build_artifact_id,
		    provider, external_deployment_id, provider_status, deployment_url,
		    version_name, build_number, git_commit, triggered_by,
		    started_at, finished_at, log_tail, error_message, metadata
		  )
		  values (
		    'default', $1::uuid, $2::uuid, $3::uuid, $4::uuid,
		    $5, $6, $7, $8, $9, $10, $11, $12,
		    $13, $14, $15::text[], $16, $17
		  )
		  returning *
		)
		select dr.id::text, dr.target_id::text, dt.target_key, dt.name, dt.project_id::text,
		       coalesce(dr.run_id::text, ''), coalesce(dr.app_build_id::text, ''),
		       coalesce(dr.app_build_artifact_id::text, ''), dr.provider, dt.environment,
		       dr.external_deployment_id, dr.provider_status, dr.deployment_url,
		       dr.version_name, dr.build_number, dr.git_commit, dr.triggered_by,
		       coalesce(dr.started_at, '0001-01-01 00:00:00+00'::timestamptz),
		       coalesce(dr.finished_at, '0001-01-01 00:00:00+00'::timestamptz),
		       dr.duration_ms, dr.log_tail, dr.error_message, dr.metadata,
		       dr.created_at, dr.updated_at
		from inserted dr
		join deployment_targets dt on dt.id = dr.target_id and dt.tenant_id = dr.tenant_id
	`
}

func deploymentRecordUpdateReturningSQL(update string) string {
	return `
		update deployment_records dr
		` + update + `
		returning dr.id::text, dr.target_id::text,
		          (select target_key from deployment_targets where id = dr.target_id),
		          (select name from deployment_targets where id = dr.target_id),
		          (select project_id::text from deployment_targets where id = dr.target_id),
		          coalesce(dr.run_id::text, ''), coalesce(dr.app_build_id::text, ''),
		          coalesce(dr.app_build_artifact_id::text, ''), dr.provider,
		          (select environment from deployment_targets where id = dr.target_id),
		          dr.external_deployment_id, dr.provider_status, dr.deployment_url,
		          dr.version_name, dr.build_number, dr.git_commit, dr.triggered_by,
		          coalesce(dr.started_at, '0001-01-01 00:00:00+00'::timestamptz),
		          coalesce(dr.finished_at, '0001-01-01 00:00:00+00'::timestamptz),
		          dr.duration_ms, dr.log_tail, dr.error_message, dr.metadata,
		          dr.created_at, dr.updated_at
	`
}

func scanDeploymentRecord(row pgx.Row) (DeploymentRecordAdmin, error) {
	var record DeploymentRecordAdmin
	var metadata []byte
	err := row.Scan(&record.ID, &record.TargetID, &record.TargetKey, &record.TargetName,
		&record.ProjectID, &record.RunID, &record.AppBuildID, &record.AppBuildArtifactID,
		&record.Provider, &record.Environment, &record.ExternalDeploymentID,
		&record.ProviderStatus, &record.DeploymentURL, &record.VersionName,
		&record.BuildNumber, &record.GitCommit, &record.TriggeredBy,
		&record.StartedAt, &record.FinishedAt, &record.DurationMS,
		&record.LogTail, &record.ErrorMessage, &metadata, &record.CreatedAt,
		&record.UpdatedAt)
	if err != nil {
		return DeploymentRecordAdmin{}, err
	}
	record.Metadata = rawJSON(metadata, "{}")
	return record, nil
}
