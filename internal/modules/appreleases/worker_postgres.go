package appreleases

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const workerTaskLogTailLimit = 200

func (s *PostgresStore) WorkerOverview(ctx context.Context) (WorkerOverviewResponse, error) {
	workers, err := s.listBuildWorkers(ctx)
	if err != nil {
		return WorkerOverviewResponse{}, err
	}
	tasks, err := s.listWorkerTasks(ctx)
	if err != nil {
		return WorkerOverviewResponse{}, err
	}
	return WorkerOverviewResponse{Workers: workers, Tasks: tasks}, nil
}

func (s *PostgresStore) CreateWorkerTask(ctx context.Context, req CreateWorkerTaskRequest) (WorkerTaskAdmin, error) {
	row := s.db.QueryRow(ctx, workerTaskInsertReturningSQL(),
		req.ProjectKey, nullableUUID(req.BuildProfileID), nullableUUID(req.BuildRunID),
		req.TaskType, req.Action, req.RequiredLabels, req.Priority, jsonb(req.Metadata))
	return scanWorkerTask(row)
}

func (s *PostgresStore) RegisterWorker(ctx context.Context, req WorkerRegisterRequest) (BuildWorkerAdmin, error) {
	row := s.db.QueryRow(ctx, `
		insert into build_workers (
		  tenant_id, worker_key, name, endpoint_url, labels, status,
		  capacity, running_tasks, last_seen_at, metadata
		)
		values ('default', $1, $2, $3, $4::text[], 'registered', $5, 0, now(), $6)
		on conflict (tenant_id, worker_key) do update
		set name = excluded.name,
		    endpoint_url = excluded.endpoint_url,
		    labels = excluded.labels,
		    capacity = excluded.capacity,
		    status = case when build_workers.status = 'offline' then 'registered' else build_workers.status end,
		    last_seen_at = now(),
		    metadata = build_workers.metadata || excluded.metadata,
		    updated_at = now()
		returning id::text, worker_key, name, endpoint_url, labels, status,
		          capacity, running_tasks,
		          coalesce(last_seen_at, '0001-01-01 00:00:00+00'::timestamptz),
		          metadata, created_at, updated_at
	`, req.WorkerKey, req.Name, req.EndpointURL, req.Labels, req.Capacity, jsonb(req.Metadata))
	return scanBuildWorker(row)
}

func (s *PostgresStore) SaveWorkerHeartbeat(ctx context.Context, req WorkerHeartbeatRequest) (BuildWorkerAdmin, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return BuildWorkerAdmin{}, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		update build_workers
		set status = $2,
		    labels = case when cardinality($3::text[]) > 0 then $3::text[] else labels end,
		    capacity = $4,
		    running_tasks = $5,
		    last_seen_at = now(),
		    metadata = metadata || $6::jsonb,
		    updated_at = now()
		where tenant_id = 'default' and worker_key = $1
		returning id::text, worker_key, name, endpoint_url, labels, status,
		          capacity, running_tasks,
		          coalesce(last_seen_at, '0001-01-01 00:00:00+00'::timestamptz),
		          metadata, created_at, updated_at
	`, req.WorkerKey, req.Status, req.Labels, req.Capacity, req.RunningTasks, jsonb(req.Metadata))
	worker, err := scanBuildWorker(row)
	if err != nil {
		return BuildWorkerAdmin{}, err
	}
	if _, err := tx.Exec(ctx, `
		insert into worker_heartbeats (
		  tenant_id, worker_id, status, labels, capacity, running_tasks, metadata
		)
		values ('default', $1::uuid, $2, $3::text[], $4, $5, $6)
	`, worker.ID, req.Status, effectiveHeartbeatLabels(req.Labels, worker.Labels), req.Capacity, req.RunningTasks, jsonb(req.Metadata)); err != nil {
		return BuildWorkerAdmin{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return BuildWorkerAdmin{}, err
	}
	return worker, nil
}

func (s *PostgresStore) NextWorkerTask(ctx context.Context, req WorkerTaskNextRequest) (*WorkerTaskAdmin, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var workerID string
	var labels []string
	if err := tx.QueryRow(ctx, `
		select id::text, labels
		from build_workers
		where tenant_id = 'default' and worker_key = $1 and status <> 'offline'
		for update
	`, req.WorkerKey).Scan(&workerID, &labels); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		update worker_tasks
		set status = 'queued',
		    worker_id = null,
		    lease_token = '',
		    leased_until = null,
		    updated_at = now()
		where tenant_id = 'default'
		  and status in ('leased', 'running')
		  and leased_until is not null
		  and leased_until < now()
	`); err != nil {
		return nil, err
	}

	var taskID string
	err = tx.QueryRow(ctx, `
		select id::text
		from worker_tasks
		where tenant_id = 'default'
		  and status = 'queued'
		  and required_labels <@ $1::text[]
		order by priority desc, created_at asc
		limit 1
		for update skip locked
	`, labels).Scan(&taskID)
	if err != nil {
		if err == pgx.ErrNoRows {
			if err := tx.Commit(ctx); err != nil {
				return nil, err
			}
			return nil, nil
		}
		return nil, err
	}

	leaseToken := uuid.NewString()
	row := tx.QueryRow(ctx, workerTaskSelectSQL()+`
		where id = $1::uuid
	`, taskID)
	task, err := scanWorkerTask(row)
	if err != nil {
		return nil, err
	}
	row = tx.QueryRow(ctx, workerTaskUpdateReturningSQL(`
		set worker_id = $2::uuid,
		    status = 'leased',
		    lease_token = $3,
		    leased_until = now() + interval '15 minutes',
		    attempts = attempts + 1,
		    started_at = coalesce(started_at, now()),
		    updated_at = now()
		where tenant_id = 'default' and id = $1::uuid
	`), task.ID, workerID, leaseToken)
	task, err = scanWorkerTask(row)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		update build_workers
		set running_tasks = running_tasks + 1,
		    updated_at = now()
		where tenant_id = 'default' and id = $1::uuid
	`, workerID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *PostgresStore) AppendWorkerTaskLogs(ctx context.Context, taskID string, req WorkerTaskLogsRequest) (WorkerTaskAdmin, error) {
	current, err := s.workerTaskLogTail(ctx, taskID, req.WorkerKey, req.LeaseToken)
	if err != nil {
		return WorkerTaskAdmin{}, err
	}
	next := append(current, req.Lines...)
	if len(next) > workerTaskLogTailLimit {
		next = next[len(next)-workerTaskLogTailLimit:]
	}
	row := s.db.QueryRow(ctx, workerTaskUpdateReturningSQL(`
		set status = 'running',
		    log_tail = $4::text[],
		    updated_at = now()
		where tenant_id = 'default'
		  and id = $1::uuid
		  and worker_id = (select id from build_workers where tenant_id = 'default' and worker_key = $2)
		  and lease_token = $3
		  and status in ('leased', 'running')
	`), taskID, req.WorkerKey, req.LeaseToken, next)
	task, err := scanWorkerTask(row)
	if err != nil {
		return WorkerTaskAdmin{}, err
	}
	if err := s.markDeploymentRecordRunning(ctx, task); err != nil {
		return WorkerTaskAdmin{}, err
	}
	return task, nil
}

func (s *PostgresStore) SaveWorkerTaskArtifacts(ctx context.Context, taskID string, req WorkerTaskArtifactsRequest) (WorkerTaskAdmin, error) {
	manifest := map[string]any{
		"artifacts": req.Artifacts,
		"metadata":  req.Metadata,
		"saved_at":  time.Now().UTC().Format(time.RFC3339),
	}
	row := s.db.QueryRow(ctx, workerTaskUpdateReturningSQL(`
		set status = 'running',
		    artifact_manifest = artifact_manifest || $4::jsonb,
		    metadata = metadata || jsonb_build_object('artifact_count', $5::int),
		    updated_at = now()
		where tenant_id = 'default'
		  and id = $1::uuid
		  and worker_id = (select id from build_workers where tenant_id = 'default' and worker_key = $2)
		  and lease_token = $3
		  and status in ('leased', 'running')
	`), taskID, req.WorkerKey, req.LeaseToken, jsonb(manifest), len(req.Artifacts))
	task, err := scanWorkerTask(row)
	if err != nil {
		return WorkerTaskAdmin{}, err
	}
	if err := s.markDeploymentRecordRunning(ctx, task); err != nil {
		return WorkerTaskAdmin{}, err
	}
	return task, nil
}

func (s *PostgresStore) CompleteWorkerTask(ctx context.Context, taskID string, req WorkerTaskCompleteRequest) (WorkerTaskAdmin, error) {
	manifest := map[string]any{
		"artifacts": req.Artifacts,
		"metadata":  req.Metadata,
		"saved_at":  time.Now().UTC().Format(time.RFC3339),
	}
	return s.finishWorkerTask(ctx, taskID, req.WorkerKey, req.LeaseToken, "success", "", jsonb(manifest), req.Metadata)
}

func (s *PostgresStore) FailWorkerTask(ctx context.Context, taskID string, req WorkerTaskFailRequest) (WorkerTaskAdmin, error) {
	return s.finishWorkerTask(ctx, taskID, req.WorkerKey, req.LeaseToken, "failed", req.ErrorMessage, jsonb(map[string]any{}), req.Metadata)
}

func (s *PostgresStore) workerTaskLogTail(ctx context.Context, taskID, workerKey, leaseToken string) ([]string, error) {
	var logTail []string
	err := s.db.QueryRow(ctx, `
		select log_tail
		from worker_tasks
		where tenant_id = 'default'
		  and id = $1::uuid
		  and worker_id = (select id from build_workers where tenant_id = 'default' and worker_key = $2)
		  and lease_token = $3
		  and status in ('leased', 'running')
	`, taskID, workerKey, leaseToken).Scan(&logTail)
	return logTail, err
}

func (s *PostgresStore) finishWorkerTask(ctx context.Context, taskID, workerKey, leaseToken, status, errorMessage string, artifactManifest []byte, metadata map[string]any) (WorkerTaskAdmin, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return WorkerTaskAdmin{}, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, workerTaskUpdateReturningSQL(`
		set status = $4,
		    lease_token = '',
		    leased_until = null,
		    artifact_manifest = artifact_manifest || $6::jsonb,
		    error_message = $5,
		    metadata = metadata || $7::jsonb,
		    finished_at = now(),
		    updated_at = now()
		where tenant_id = 'default'
		  and id = $1::uuid
		  and worker_id = (select id from build_workers where tenant_id = 'default' and worker_key = $2)
		  and lease_token = $3
		  and status in ('leased', 'running')
	`), taskID, workerKey, leaseToken, status, errorMessage, artifactManifest, jsonb(metadata))
	task, err := scanWorkerTask(row)
	if err != nil {
		return WorkerTaskAdmin{}, err
	}
	if _, err := tx.Exec(ctx, `
		update build_workers
		set running_tasks = greatest(running_tasks - 1, 0),
		    updated_at = now()
		where tenant_id = 'default' and worker_key = $1
	`, workerKey); err != nil {
		return WorkerTaskAdmin{}, err
	}
	if task.BuildRunID != "" {
		patch := BuildCenterRunPatch{
			Status:       status,
			UploadStatus: uploadStatusForAction(task.Action, status),
			ErrorMessage: errorMessage,
			Metadata: map[string]any{
				"worker_task_id": task.ID,
				"worker_key":     workerKey,
			},
		}
		if _, err := s.CompleteBuildCenterRun(ctx, task.BuildRunID, patch); err != nil {
			return WorkerTaskAdmin{}, err
		}
	}
	if deploymentID := workerTaskDeploymentRecordID(task.Metadata); deploymentID != "" {
		if _, err := tx.Exec(ctx, `
			update deployment_records
			set external_deployment_id = coalesce(nullif($2, ''), external_deployment_id),
			    provider_status = $3,
			    deployment_url = coalesce(nullif($4, ''), deployment_url),
			    log_tail = case when cardinality($5::text[]) > 0 then $5::text[] else log_tail end,
			    error_message = $6,
			    metadata = metadata || $7::jsonb,
			    finished_at = now(),
			    duration_ms = case when started_at is null then duration_ms else greatest(extract(epoch from (now() - started_at))::bigint * 1000, 0) end,
			    updated_at = now()
			where tenant_id = 'default' and id = $1::uuid
		`, deploymentID, workerTaskMetadataString(task.Metadata, "external_deployment_id"),
			deploymentStatusForWorkerStatus(status),
			workerTaskMetadataString(task.Metadata, "deployment_url"),
			task.LogTail, errorMessage, jsonb(map[string]any{
				"worker_task_id": task.ID,
				"worker_key":     workerKey,
				"worker_status":  status,
			})); err != nil {
			return WorkerTaskAdmin{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return WorkerTaskAdmin{}, err
	}
	return task, nil
}

func deploymentStatusForWorkerStatus(status string) string {
	if status == "success" {
		return "success"
	}
	return "failed"
}

func (s *PostgresStore) markDeploymentRecordRunning(ctx context.Context, task WorkerTaskAdmin) error {
	deploymentID := workerTaskDeploymentRecordID(task.Metadata)
	if deploymentID == "" {
		return nil
	}
	_, err := s.db.Exec(ctx, `
		update deployment_records
		set provider_status = 'running',
		    log_tail = case when cardinality($2::text[]) > 0 then $2::text[] else log_tail end,
		    metadata = metadata || $3::jsonb,
		    started_at = coalesce(started_at, now()),
		    updated_at = now()
		where tenant_id = 'default'
		  and id = $1::uuid
		  and provider_status in ('queued', 'running', 'external')
	`, deploymentID, task.LogTail, jsonb(map[string]any{
		"worker_task_id": task.ID,
		"worker_status":  task.Status,
	}))
	return err
}

func workerTaskDeploymentRecordID(metadata []byte) string {
	return workerTaskMetadataString(metadata, "deployment_record_id")
}

func workerTaskMetadataString(metadata []byte, key string) string {
	var values map[string]any
	if len(metadata) == 0 || json.Unmarshal(metadata, &values) != nil {
		return ""
	}
	return stringFromAny(values[key])
}

func scanBuildWorker(row pgx.Row) (BuildWorkerAdmin, error) {
	var worker BuildWorkerAdmin
	var metadata []byte
	err := row.Scan(&worker.ID, &worker.WorkerKey, &worker.Name, &worker.EndpointURL,
		&worker.Labels, &worker.Status, &worker.Capacity, &worker.RunningTasks,
		&worker.LastSeenAt, &metadata, &worker.CreatedAt, &worker.UpdatedAt)
	if err != nil {
		return BuildWorkerAdmin{}, err
	}
	worker.Metadata = rawJSON(metadata, "{}")
	return worker, nil
}

func scanWorkerTask(row pgx.Row) (WorkerTaskAdmin, error) {
	var task WorkerTaskAdmin
	var artifactManifest, metadata []byte
	err := row.Scan(&task.ID, &task.WorkerID, &task.BuildRunID, &task.ProjectID,
		&task.BuildProfileID, &task.TaskType, &task.Action, &task.Status,
		&task.RequiredLabels, &task.Priority, &task.LeaseToken, &task.LeasedUntil,
		&task.Attempts, &task.LogTail, &artifactManifest, &task.ErrorMessage,
		&metadata, &task.StartedAt, &task.FinishedAt, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return WorkerTaskAdmin{}, err
	}
	task.ArtifactManifest = rawJSON(artifactManifest, "{}")
	task.Metadata = rawJSON(metadata, "{}")
	return task, nil
}

func workerTaskSelectSQL() string {
	return `
		select id::text, coalesce(worker_id::text, ''), coalesce(build_run_id::text, ''),
		       coalesce(project_id::text, ''), coalesce(build_profile_id::text, ''),
		       task_type, action, status, required_labels, priority, lease_token,
		       coalesce(leased_until, '0001-01-01 00:00:00+00'::timestamptz),
		       attempts, log_tail, artifact_manifest, error_message, metadata,
		       coalesce(started_at, '0001-01-01 00:00:00+00'::timestamptz),
		       coalesce(finished_at, '0001-01-01 00:00:00+00'::timestamptz),
		       created_at, updated_at
		from worker_tasks
	`
}

func workerTaskInsertReturningSQL() string {
	return `
		with inserted as (
		  insert into worker_tasks (
		    tenant_id, build_run_id, project_id, build_profile_id, task_type,
		    action, status, required_labels, priority, metadata
		  )
		  values (
		    'default', $3::uuid,
		    (select id from release_projects where tenant_id = 'default' and project_key = nullif($1, '')),
		    $2::uuid, $4, $5, 'queued', $6::text[], $7, $8
		  )
		  returning *
		)
		select id::text, coalesce(worker_id::text, ''), coalesce(build_run_id::text, ''),
		       coalesce(project_id::text, ''), coalesce(build_profile_id::text, ''),
		       task_type, action, status, required_labels, priority, lease_token,
		       coalesce(leased_until, '0001-01-01 00:00:00+00'::timestamptz),
		       attempts, log_tail, artifact_manifest, error_message, metadata,
		       coalesce(started_at, '0001-01-01 00:00:00+00'::timestamptz),
		       coalesce(finished_at, '0001-01-01 00:00:00+00'::timestamptz),
		       created_at, updated_at
		from inserted
	`
}

func workerTaskUpdateReturningSQL(update string) string {
	return `
		update worker_tasks
		` + update + `
		returning id::text, coalesce(worker_id::text, ''), coalesce(build_run_id::text, ''),
		          coalesce(project_id::text, ''), coalesce(build_profile_id::text, ''),
		          task_type, action, status, required_labels, priority, lease_token,
		          coalesce(leased_until, '0001-01-01 00:00:00+00'::timestamptz),
		          attempts, log_tail, artifact_manifest, error_message, metadata,
		          coalesce(started_at, '0001-01-01 00:00:00+00'::timestamptz),
		          coalesce(finished_at, '0001-01-01 00:00:00+00'::timestamptz),
		          created_at, updated_at
	`
}

func (s *PostgresStore) listBuildWorkers(ctx context.Context) ([]BuildWorkerAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, worker_key, name, endpoint_url, labels, status,
		       capacity, running_tasks,
		       coalesce(last_seen_at, '0001-01-01 00:00:00+00'::timestamptz),
		       metadata, created_at, updated_at
		from build_workers
		where tenant_id = 'default'
		order by last_seen_at desc nulls last, updated_at desc, created_at desc
		limit 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var workers []BuildWorkerAdmin
	for rows.Next() {
		worker, err := scanBuildWorker(rows)
		if err != nil {
			return nil, err
		}
		workers = append(workers, worker)
	}
	return workers, rows.Err()
}

func (s *PostgresStore) listWorkerTasks(ctx context.Context) ([]WorkerTaskAdmin, error) {
	rows, err := s.db.Query(ctx, workerTaskSelectSQL()+`
		where tenant_id = 'default'
		order by updated_at desc, created_at desc
		limit 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []WorkerTaskAdmin
	for rows.Next() {
		task, err := scanWorkerTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func effectiveHeartbeatLabels(requested, stored []string) []string {
	if len(requested) > 0 {
		return requested
	}
	if stored == nil {
		return []string{}
	}
	return stored
}
