package appreleases

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var errArtifactCenterStoreUnavailable = errors.New("artifact center store unavailable")

type ArtifactCenterStore interface {
	ArtifactCenterOverview(ctx context.Context) (ArtifactCenterOverview, error)
	ArtifactCenterItem(ctx context.Context, artifactID string) (ArtifactCenterItem, error)
}

func (s *Service) ArtifactCenterOverview(ctx context.Context) (ArtifactCenterOverview, error) {
	store, ok := s.store.(ArtifactCenterStore)
	if !ok {
		return ArtifactCenterOverview{Artifacts: []ArtifactCenterItem{}, MessageZh: "制品中心未连接数据库"}, nil
	}
	overview, err := store.ArtifactCenterOverview(ctx)
	if err != nil {
		return ArtifactCenterOverview{}, err
	}
	overview.MessageZh = "制品中心已读取"
	return overview, nil
}

func (s *Service) ArtifactCenterItem(ctx context.Context, artifactID string) (ArtifactCenterItemResponse, error) {
	store, ok := s.store.(ArtifactCenterStore)
	if !ok {
		return ArtifactCenterItemResponse{}, errArtifactCenterStoreUnavailable
	}
	artifactID = strings.TrimSpace(artifactID)
	if artifactID == "" {
		return ArtifactCenterItemResponse{}, fmt.Errorf("artifact_id is required")
	}
	artifact, err := store.ArtifactCenterItem(ctx, artifactID)
	if err != nil {
		return ArtifactCenterItemResponse{}, err
	}
	return ArtifactCenterItemResponse{OK: true, Artifact: artifact, MessageZh: "制品已读取"}, nil
}
