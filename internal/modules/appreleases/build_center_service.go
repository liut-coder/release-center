package appreleases

import (
	"context"
	"errors"
)

var errBuildCenterStoreUnavailable = errors.New("build center store unavailable")

type BuildCenterStore interface {
	BuildCenterOverview(ctx context.Context) (BuildCenterOverview, error)
	BuildCenterProject(ctx context.Context, projectKey string) (BuildCenterProject, error)
	DeploymentTargets(ctx context.Context) ([]DeploymentTargetAdmin, error)
}

func (s *Service) BuildCenterOverview(ctx context.Context) (BuildCenterOverview, error) {
	store, ok := s.store.(BuildCenterStore)
	if !ok {
		return BuildCenterOverview{MessageZh: "构建中心未连接数据库"}, nil
	}
	resp, err := store.BuildCenterOverview(ctx)
	if err != nil {
		return BuildCenterOverview{}, err
	}
	resp.MessageZh = "构建中心已读取"
	return resp, nil
}

func (s *Service) BuildCenterProject(ctx context.Context, projectKey string) (BuildCenterProject, error) {
	store, ok := s.store.(BuildCenterStore)
	if !ok {
		return BuildCenterProject{}, errBuildCenterStoreUnavailable
	}
	return store.BuildCenterProject(ctx, projectKey)
}

func (s *Service) DeploymentTargets(ctx context.Context) ([]DeploymentTargetAdmin, error) {
	store, ok := s.store.(BuildCenterStore)
	if !ok {
		return []DeploymentTargetAdmin{}, nil
	}
	return store.DeploymentTargets(ctx)
}
