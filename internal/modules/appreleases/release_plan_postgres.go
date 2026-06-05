package appreleases

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *PostgresStore) ReleasePlanOverview(ctx context.Context) (ReleasePlanOverview, error) {
	environments, err := s.listReleaseEnvironments(ctx)
	if err != nil {
		return ReleasePlanOverview{}, err
	}
	units, err := s.listReleaseUnits(ctx)
	if err != nil {
		return ReleasePlanOverview{}, err
	}
	plans, err := s.listReleasePlans(ctx, "")
	if err != nil {
		return ReleasePlanOverview{}, err
	}
	if err := s.attachReleasePlanArtifacts(ctx, plans); err != nil {
		return ReleasePlanOverview{}, err
	}
	return ReleasePlanOverview{Environments: environments, ReleaseUnits: units, ReleasePlans: plans}, nil
}

func (s *PostgresStore) ReleasePlan(ctx context.Context, planID string) (ReleasePlanAdmin, error) {
	plans, err := s.listReleasePlans(ctx, "and rp.id = $1::uuid", planID)
	if err != nil {
		return ReleasePlanAdmin{}, err
	}
	if len(plans) == 0 {
		return ReleasePlanAdmin{}, pgx.ErrNoRows
	}
	if err := s.attachReleasePlanArtifacts(ctx, plans); err != nil {
		return ReleasePlanAdmin{}, err
	}
	return plans[0], nil
}

func (s *PostgresStore) PreviousReleasePlan(ctx context.Context, planID string) (ReleasePlanAdmin, error) {
	plans, err := s.listReleasePlans(ctx, `
		and rp.id = (
		  select previous.id
		  from release_plans current
		  join release_plans previous
		    on previous.tenant_id = current.tenant_id
		   and previous.project_id = current.project_id
		   and previous.release_unit_id = current.release_unit_id
		   and previous.environment_id = current.environment_id
		  where current.tenant_id = 'default'
		    and current.id = $1::uuid
		    and previous.id <> current.id
		    and previous.status in ('released', 'rolling_out')
		    and coalesce(previous.published_at, previous.updated_at, previous.created_at)
		        < coalesce(current.published_at, current.updated_at, current.created_at)
		  order by coalesce(previous.published_at, previous.updated_at, previous.created_at) desc
		  limit 1
		)
	`, planID)
	if err != nil {
		return ReleasePlanAdmin{}, err
	}
	if len(plans) == 0 {
		return ReleasePlanAdmin{}, pgx.ErrNoRows
	}
	if err := s.attachReleasePlanArtifacts(ctx, plans); err != nil {
		return ReleasePlanAdmin{}, err
	}
	return plans[0], nil
}

func (s *PostgresStore) CreateReleaseUnit(ctx context.Context, req CreateReleaseUnitRequest) (ReleaseUnitAdmin, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var unit ReleaseUnitAdmin
	var metadata []byte
	err := s.db.QueryRow(ctx, `
		with project_row as (
		  select id, project_key from release_projects where tenant_id = 'default' and project_key = $1
		)
		insert into release_units (
		  tenant_id, project_id, app_id, unit_key, name, unit_type,
		  default_channel, enabled, metadata
		)
		select 'default', project_row.id, nullif($2, '')::uuid, $3, $4, $5,
		       $6, $7, $8
		from project_row
		on conflict (project_id, unit_key) do update
		set app_id = excluded.app_id,
		    name = excluded.name,
		    unit_type = excluded.unit_type,
		    default_channel = excluded.default_channel,
		    enabled = excluded.enabled,
		    metadata = release_units.metadata || excluded.metadata,
		    updated_at = now()
		returning id::text, project_id::text,
		          (select project_key from project_row), coalesce(app_id::text, ''),
		          unit_key, name, unit_type, default_channel, enabled,
		          metadata, created_at, updated_at
	`, req.ProjectKey, req.AppID, req.UnitKey, req.Name, req.UnitType,
		req.DefaultChannel, enabled, jsonb(req.Metadata)).Scan(
		&unit.ID, &unit.ProjectID, &unit.ProjectKey, &unit.AppID, &unit.UnitKey,
		&unit.Name, &unit.UnitType, &unit.DefaultChannel, &unit.Enabled,
		&metadata, &unit.CreatedAt, &unit.UpdatedAt,
	)
	if err != nil {
		return ReleaseUnitAdmin{}, err
	}
	unit.Metadata = rawJSON(metadata, "{}")
	return unit, nil
}

func (s *PostgresStore) CreateReleasePlan(ctx context.Context, req CreateReleasePlanRequest) (ReleasePlanAdmin, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return ReleasePlanAdmin{}, err
	}
	defer tx.Rollback(ctx)

	var planID string
	err = tx.QueryRow(ctx, `
		with project_row as (
		  select id from release_projects where tenant_id = 'default' and project_key = $1
		),
		unit_row as (
		  select ru.id
		  from release_units ru
		  join project_row on project_row.id = ru.project_id
		  where ru.tenant_id = 'default' and ru.unit_key = $2 and ru.enabled = true
		),
		env_row as (
		  select id from release_environments where tenant_id = 'default' and environment_key = $3
		)
		insert into release_plans (
		  tenant_id, project_id, release_unit_id, environment_id, plan_key,
		  title, description, version_name, build_number, git_commit,
		  channel, status, rollout_percentage, target_type, target_value,
		  scheduled_at, approved_by, metadata, created_by
		)
		select 'default', project_row.id, unit_row.id, env_row.id, $4,
		       $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
		       $15::timestamptz, $16, $17, $18
		from project_row, unit_row, env_row
		on conflict (project_id, plan_key) do update
		set release_unit_id = excluded.release_unit_id,
		    environment_id = excluded.environment_id,
		    title = excluded.title,
		    description = excluded.description,
		    version_name = excluded.version_name,
		    build_number = excluded.build_number,
		    git_commit = excluded.git_commit,
		    channel = excluded.channel,
		    status = excluded.status,
		    rollout_percentage = excluded.rollout_percentage,
		    target_type = excluded.target_type,
		    target_value = excluded.target_value,
		    scheduled_at = excluded.scheduled_at,
		    approved_by = excluded.approved_by,
		    metadata = release_plans.metadata || excluded.metadata,
		    updated_at = now()
		returning id::text
	`, req.ProjectKey, req.UnitKey, req.EnvironmentKey, req.PlanKey,
		req.Title, req.Description, req.VersionName, req.BuildNumber, req.GitCommit,
		req.Channel, req.Status, req.RolloutPercentage, req.TargetType, req.TargetValue,
		nullableTimeString(req.ScheduledAt), req.ApprovedBy, jsonb(req.Metadata),
		req.CreatedBy).Scan(&planID)
	if err != nil {
		return ReleasePlanAdmin{}, err
	}
	if _, err := tx.Exec(ctx, `delete from release_plan_artifacts where tenant_id = 'default' and release_plan_id = $1::uuid`, planID); err != nil {
		return ReleasePlanAdmin{}, err
	}
	for _, artifact := range req.Artifacts {
		if _, err := tx.Exec(ctx, `
			insert into release_plan_artifacts (
			  tenant_id, release_plan_id, build_run_id, app_build_id, app_build_artifact_id,
			  artifact_name, artifact_type, file_name, immutable_ref, metadata
			)
			values (
			  'default', $1::uuid, $2::uuid, $3::uuid, $4::uuid,
			  $5, $6, $7, $8, $9
			)
		`, planID, nullableUUID(artifact.BuildRunID), nullableUUID(artifact.AppBuildID),
			nullableUUID(artifact.AppBuildArtifactID), artifact.ArtifactName,
			artifact.ArtifactType, artifact.FileName, artifact.ImmutableRef,
			jsonb(artifact.Metadata)); err != nil {
			return ReleasePlanAdmin{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ReleasePlanAdmin{}, err
	}
	return s.ReleasePlan(ctx, planID)
}

func (s *PostgresStore) UpdateReleasePlanStatus(ctx context.Context, planID, status string, req ReleasePlanActionRequest) (ReleasePlanAdmin, error) {
	_, err := s.db.Exec(ctx, `
			update release_plans
			set status = $2::varchar(32),
			    approved_by = coalesce(nullif($3, ''), approved_by),
			    published_at = case when $2::varchar(32) = 'released' then coalesce(published_at, now()) else published_at end,
			    paused_at = case when $2::varchar(32) = 'paused' then now() else paused_at end,
			    metadata = metadata || $4::jsonb,
			    updated_at = now()
			where tenant_id = 'default' and id = $1::uuid
	`, planID, status, req.ApprovedBy, jsonb(req.Metadata))
	if err != nil {
		return ReleasePlanAdmin{}, err
	}
	return s.ReleasePlan(ctx, planID)
}

func (s *PostgresStore) listReleaseEnvironments(ctx context.Context) ([]ReleaseEnvironmentAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, environment_key, name, sort_order, requires_approval,
		       metadata, created_at, updated_at
		from release_environments
		where tenant_id = 'default'
		order by sort_order asc, environment_key asc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ReleaseEnvironmentAdmin
	for rows.Next() {
		var item ReleaseEnvironmentAdmin
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.EnvironmentKey, &item.Name,
			&item.SortOrder, &item.RequiresApproval, &metadata,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Metadata = rawJSON(metadata, "{}")
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) listReleaseUnits(ctx context.Context) ([]ReleaseUnitAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select ru.id::text, ru.project_id::text, p.project_key,
		       coalesce(ru.app_id::text, ''), ru.unit_key, ru.name,
		       ru.unit_type, ru.default_channel, ru.enabled, ru.metadata,
		       ru.created_at, ru.updated_at
		from release_units ru
		join release_projects p on p.id = ru.project_id and p.tenant_id = ru.tenant_id
		where ru.tenant_id = 'default'
		order by p.project_key asc, ru.unit_type asc, ru.unit_key asc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ReleaseUnitAdmin
	for rows.Next() {
		var item ReleaseUnitAdmin
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.ProjectKey, &item.AppID,
			&item.UnitKey, &item.Name, &item.UnitType, &item.DefaultChannel,
			&item.Enabled, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Metadata = rawJSON(metadata, "{}")
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) listReleasePlans(ctx context.Context, extraWhere string, args ...any) ([]ReleasePlanAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select rp.id::text, rp.project_id::text, p.project_key,
		       rp.release_unit_id::text, ru.unit_key, ru.unit_type,
		       rp.environment_id::text, re.environment_key, re.requires_approval,
		       rp.plan_key, rp.title, rp.description, rp.version_name,
		       rp.build_number, rp.git_commit, rp.channel, rp.status,
		       rp.rollout_percentage, rp.target_type, rp.target_value,
			       rp.scheduled_at, rp.published_at, rp.paused_at,
		       rp.approved_by, rp.metadata, rp.created_by, rp.created_at, rp.updated_at
		from release_plans rp
		join release_projects p on p.id = rp.project_id and p.tenant_id = rp.tenant_id
		join release_units ru on ru.id = rp.release_unit_id and ru.tenant_id = rp.tenant_id
		join release_environments re on re.id = rp.environment_id and re.tenant_id = rp.tenant_id
		where rp.tenant_id = 'default' `+extraWhere+`
		order by rp.updated_at desc, rp.created_at desc
		limit 100
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var plans []ReleasePlanAdmin
	for rows.Next() {
		plan, err := scanReleasePlan(rows)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, rows.Err()
}

func (s *PostgresStore) attachReleasePlanArtifacts(ctx context.Context, plans []ReleasePlanAdmin) error {
	if len(plans) == 0 {
		return nil
	}
	index := map[string]int{}
	ids := make([]string, 0, len(plans))
	for i, plan := range plans {
		index[plan.ID] = i
		ids = append(ids, plan.ID)
	}
	rows, err := s.db.Query(ctx, `
		select id::text, release_plan_id::text, coalesce(build_run_id::text, ''),
		       coalesce(app_build_id::text, ''), coalesce(app_build_artifact_id::text, ''),
		       artifact_name, artifact_type, file_name, immutable_ref,
		       metadata, created_at, updated_at
		from release_plan_artifacts
		where tenant_id = 'default' and release_plan_id::text = any($1)
		order by created_at asc, artifact_name asc
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var artifact ReleasePlanArtifactAdmin
		var metadata []byte
		if err := rows.Scan(&artifact.ID, &artifact.ReleasePlanID, &artifact.BuildRunID,
			&artifact.AppBuildID, &artifact.AppBuildArtifactID, &artifact.ArtifactName,
			&artifact.ArtifactType, &artifact.FileName, &artifact.ImmutableRef,
			&metadata, &artifact.CreatedAt, &artifact.UpdatedAt); err != nil {
			return err
		}
		artifact.Metadata = rawJSON(metadata, "{}")
		if i, ok := index[artifact.ReleasePlanID]; ok {
			plans[i].Artifacts = append(plans[i].Artifacts, artifact)
		}
	}
	return rows.Err()
}

func scanReleasePlan(row pgx.Row) (ReleasePlanAdmin, error) {
	var plan ReleasePlanAdmin
	var metadata []byte
	var scheduledAt pgtype.Timestamptz
	var publishedAt pgtype.Timestamptz
	var pausedAt pgtype.Timestamptz
	err := row.Scan(&plan.ID, &plan.ProjectID, &plan.ProjectKey,
		&plan.ReleaseUnitID, &plan.UnitKey, &plan.UnitType,
		&plan.EnvironmentID, &plan.EnvironmentKey, &plan.EnvironmentRequiresApproval, &plan.PlanKey,
		&plan.Title, &plan.Description, &plan.VersionName, &plan.BuildNumber,
		&plan.GitCommit, &plan.Channel, &plan.Status, &plan.RolloutPercentage,
		&plan.TargetType, &plan.TargetValue, &scheduledAt,
		&publishedAt, &pausedAt, &plan.ApprovedBy, &metadata,
		&plan.CreatedBy, &plan.CreatedAt, &plan.UpdatedAt)
	if err != nil {
		return ReleasePlanAdmin{}, err
	}
	plan.ScheduledAt = timestamptzPtr(scheduledAt)
	plan.PublishedAt = timestamptzPtr(publishedAt)
	plan.PausedAt = timestamptzPtr(pausedAt)
	plan.Metadata = rawJSON(metadata, "{}")
	return plan, nil
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func nullableTimeString(value string) any {
	if value == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	return value
}
