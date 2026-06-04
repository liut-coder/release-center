package appreleases

import (
	"context"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

const artifactCenterLimit = 200

func (s *PostgresStore) ArtifactCenterOverview(ctx context.Context) (ArtifactCenterOverview, error) {
	items := make([]ArtifactCenterItem, 0, 64)
	appArtifacts, err := s.listAppBuildArtifactCenterItems(ctx, "")
	if err != nil {
		return ArtifactCenterOverview{}, err
	}
	items = append(items, appArtifacts...)
	buildArtifacts, err := s.listBuildCenterArtifactCenterItems(ctx, "")
	if err != nil {
		return ArtifactCenterOverview{}, err
	}
	items = append(items, buildArtifacts...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > artifactCenterLimit {
		items = items[:artifactCenterLimit]
	}
	return ArtifactCenterOverview{Artifacts: items}, nil
}

func (s *PostgresStore) ArtifactCenterItem(ctx context.Context, artifactID string) (ArtifactCenterItem, error) {
	artifactID = strings.TrimSpace(artifactID)
	if artifactID == "" {
		return ArtifactCenterItem{}, pgx.ErrNoRows
	}
	items, err := s.listBuildCenterArtifactCenterItems(ctx, "and bcra.id::text = $1", artifactID)
	if err != nil {
		return ArtifactCenterItem{}, err
	}
	if len(items) > 0 {
		return items[0], nil
	}
	items, err = s.listAppBuildArtifactCenterItems(ctx, "and aba.id::text = $1", artifactID)
	if err != nil {
		return ArtifactCenterItem{}, err
	}
	if len(items) == 0 {
		return ArtifactCenterItem{}, pgx.ErrNoRows
	}
	return items[0], nil
}

func (s *PostgresStore) listAppBuildArtifactCenterItems(ctx context.Context, extraWhere string, args ...any) ([]ArtifactCenterItem, error) {
	rows, err := s.db.Query(ctx, `
		select aba.id::text, 'app_build' as source,
		       '' as project_key, '' as project_name,
		       coalesce(a.app_key, '') as app_key, coalesce(a.name, '') as app_name,
		       aba.build_id::text, '' as run_id, aba.id::text as app_build_artifact_id,
		       aba.name, aba.artifact_type, aba.file_name, aba.artifact_path as location,
		       'app_build_artifact:' || aba.id::text as immutable_ref,
		       aba.artifact_size, aba.sha256, 'uploaded' as upload_status,
		       b.version_name, b.build_number, b.git_commit, b.channel, b.build_status as status,
		       aba.metadata, aba.created_at
		from app_build_artifacts aba
		join app_builds b on b.id = aba.build_id and b.tenant_id = aba.tenant_id
		left join apps a on a.id = b.app_id and a.tenant_id = b.tenant_id
		where aba.tenant_id = 'default' `+extraWhere+`
		order by aba.created_at desc, aba.name asc
		limit 200
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArtifactCenterItems(rows)
}

func (s *PostgresStore) listBuildCenterArtifactCenterItems(ctx context.Context, extraWhere string, args ...any) ([]ArtifactCenterItem, error) {
	rows, err := s.db.Query(ctx, `
		select bcra.id::text, 'build_center' as source,
		       coalesce(p.project_key, '') as project_key, coalesce(p.name, '') as project_name,
		       coalesce(a.app_key, '') as app_key, coalesce(a.name, '') as app_name,
		       coalesce(bcr.app_build_id::text, '') as build_id, bcra.run_id::text,
		       coalesce(bcra.app_build_artifact_id::text, '') as app_build_artifact_id,
		       bcra.name, bcra.artifact_type, bcra.file_name,
		       coalesce(nullif(bcra.download_url, ''), bcra.local_path) as location,
		       'build_center_artifact:' || bcra.id::text as immutable_ref,
		       bcra.size_bytes, bcra.sha256, bcra.upload_status,
		       bcr.version_name, bcr.build_number, bcr.git_commit, bcr.channel, bcr.status,
		       bcra.metadata, bcra.created_at
		from build_center_run_artifacts bcra
		join build_center_runs bcr on bcr.id = bcra.run_id and bcr.tenant_id = bcra.tenant_id
		left join release_projects p on p.id = bcr.project_id and p.tenant_id = bcr.tenant_id
		left join app_builds b on b.id = bcr.app_build_id and b.tenant_id = bcr.tenant_id
		left join apps a on a.id = b.app_id and a.tenant_id = b.tenant_id
		where bcra.tenant_id = 'default' `+extraWhere+`
		order by bcra.created_at desc, bcra.name asc
		limit 200
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArtifactCenterItems(rows)
}

func scanArtifactCenterItems(rows pgx.Rows) ([]ArtifactCenterItem, error) {
	var items []ArtifactCenterItem
	for rows.Next() {
		var item ArtifactCenterItem
		var metadata []byte
		if err := rows.Scan(
			&item.ID,
			&item.Source,
			&item.ProjectKey,
			&item.ProjectName,
			&item.AppKey,
			&item.AppName,
			&item.BuildID,
			&item.RunID,
			&item.AppBuildArtifactID,
			&item.Name,
			&item.ArtifactType,
			&item.FileName,
			&item.Location,
			&item.ImmutableRef,
			&item.SizeBytes,
			&item.SHA256,
			&item.UploadStatus,
			&item.VersionName,
			&item.BuildNumber,
			&item.GitCommit,
			&item.Channel,
			&item.Status,
			&metadata,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.Metadata = rawJSON(metadata, "{}")
		items = append(items, item)
	}
	return items, rows.Err()
}
