package appreleases

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDemoStoreCreateBuildPersistsInAdminOverview(t *testing.T) {
	service := NewServiceWithStore(Config{}, NewDemoStore())

	resp, err := service.CreateBuild(context.Background(), CreateBuildRequest{
		GitRef:      "feature/demo-smoke",
		Channel:     "dev",
		BuildType:   "debug",
		VersionName: "0.1.1-dev.1",
		VersionCode: 2,
		BuildNumber: 2,
		APIBaseURL:  "http://127.0.0.1:18080/",
	})
	if err != nil {
		t.Fatalf("create build failed: %v", err)
	}

	overview, err := service.AdminOverview(context.Background())
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}
	for _, build := range overview.BuildJobs {
		if build.ID == resp.Job.ID && build.VersionName == "0.1.1-dev.1" {
			return
		}
	}
	t.Fatalf("expected created build in overview, got %#v", overview.BuildJobs)
}

func TestDemoStoreCreateReleaseDraftPersistsInAdminOverview(t *testing.T) {
	service := NewServiceWithStore(Config{}, NewDemoStore())
	build, err := service.CreateBuild(context.Background(), CreateBuildRequest{
		GitRef:      "main",
		Channel:     "dev",
		BuildType:   "debug",
		VersionName: "0.1.2-dev.1",
		VersionCode: 3,
		BuildNumber: 3,
	})
	if err != nil {
		t.Fatalf("create build failed: %v", err)
	}

	resp, err := service.CreateRelease(context.Background(), CreateReleaseRequest{
		BuildID:           build.Job.ID,
		Channel:           "dev",
		Title:             "Demo 发布草稿",
		UpdateLevel:       "normal",
		RolloutPercentage: 100,
		TargetType:        "all",
	})
	if err != nil {
		t.Fatalf("create release failed: %v", err)
	}
	if resp.Release.Status != "draft" {
		t.Fatalf("expected draft release, got %s", resp.Release.Status)
	}

	overview, err := service.AdminOverview(context.Background())
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}
	for _, release := range overview.Releases {
		if release.ID == resp.Release.ID && release.Title == "Demo 发布草稿" {
			return
		}
	}
	t.Fatalf("expected created release in overview, got %#v", overview.Releases)
}

func TestDemoStoreCreateResourceVersionPersistsInAdminOverview(t *testing.T) {
	service := NewServiceWithStore(Config{}, NewDemoStore())
	blob := &memoryBlobStore{}
	body := demoResourceZip(t)

	resp, err := service.CreateResourceVersion(context.Background(), CreateResourceVersionRequest{
		ResourceVersion:   "20260602.smoke",
		Channel:           "dev",
		Title:             "Demo 资源版本",
		UpdateLevel:       "normal",
		RolloutPercentage: 100,
	}, []UploadedResourcePackage{{
		PackageKey: "templates-common",
		FileName:   "templates-common-20260602.smoke.zip",
		Content:    body,
	}}, blob)
	if err != nil {
		t.Fatalf("create resource failed: %v", err)
	}
	if resp.ResourceVersion.Status != "draft" {
		t.Fatalf("expected draft resource, got %s", resp.ResourceVersion.Status)
	}
	if len(blob.saved) != 2 {
		t.Fatalf("expected package and manifest blobs to be saved, got %d", len(blob.saved))
	}

	overview, err := service.AdminOverview(context.Background())
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}
	for _, resource := range overview.ResourceVersions {
		if resource.ID == resp.ResourceVersion.ID && resource.ResourceVersion == "20260602.smoke" {
			return
		}
	}
	t.Fatalf("expected created resource in overview, got %#v", overview.ResourceVersions)
}

func TestRoutesWithDemoStoreEnforcesAdminRBAC(t *testing.T) {
	service := NewServiceWithStore(Config{}, NewDemoStore())
	routes := NewHandler(service).RoutesWithOptions(RouteOptions{
		AdminMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("release-token"),
		},
		AdminPermissionMiddleware: AdminRBACMiddleware(service, map[string]string{
			"release-token": "release.admin",
		}),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/system/users", strings.NewReader(`{"name":"受限用户","account":"limited.user","role_code":"release_viewer"}`))
	req.Header.Set("Authorization", "Bearer release-token")
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload["code"] != "auth.permission_denied" {
		t.Fatalf("expected permission denied, got %#v", payload)
	}
}

type memoryBlobStore struct {
	saved map[string][]byte
}

func (s *memoryBlobStore) Save(ctx context.Context, namespace, filename string, body io.Reader) (string, int64, error) {
	select {
	case <-ctx.Done():
		return "", 0, ctx.Err()
	default:
	}
	if s.saved == nil {
		s.saved = map[string][]byte{}
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return "", 0, err
	}
	key := namespace + "/" + filename
	s.saved[key] = data
	return key, int64(len(data)), nil
}

func (s *memoryBlobStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return io.NopCloser(bytes.NewReader(s.saved[key])), nil
}

func demoResourceZip(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	file, err := writer.Create("templates/demo.txt")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := file.Write([]byte("demo-template")); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}
