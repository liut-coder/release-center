package appreleases

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBearerTokenMiddlewareAllowsWhenNoTokensConfigured(t *testing.T) {
	called := false
	handler := BearerTokenMiddleware("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/ci/artifacts", nil))

	if !called || rec.Code != http.StatusNoContent {
		t.Fatalf("expected request to pass without configured tokens, code=%d called=%v", rec.Code, called)
	}
}

func TestBearerTokenMiddlewareRejectsInvalidToken(t *testing.T) {
	handler := BearerTokenMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ci/artifacts", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d", rec.Code)
	}
}

func TestBearerTokenMiddlewareAllowsValidToken(t *testing.T) {
	called := false
	handler := BearerTokenMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ci/artifacts", nil)
	req.Header.Set("Authorization", "Bearer secret")
	handler.ServeHTTP(rec, req)

	if !called || rec.Code != http.StatusNoContent {
		t.Fatalf("expected valid token to pass, code=%d called=%v", rec.Code, called)
	}
}

func TestRequirePermissionRejectsViewerWriteAction(t *testing.T) {
	handler := NewHandler(NewService(Config{}))
	routes := handler.RoutesWithOptions(RouteOptions{
		AdminMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("secret"),
			AdminIdentityMiddleware("release_viewer"),
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/release-plans", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer secret")
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected viewer write action to be forbidden, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "release:write") {
		t.Fatalf("expected forbidden response to include required permission, body=%s", rec.Body.String())
	}
}

func TestRequirePermissionAllowsReleaseAdminWriteAction(t *testing.T) {
	handler := NewHandler(NewService(Config{}))
	routes := handler.RoutesWithOptions(RouteOptions{
		AdminMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("secret"),
			AdminIdentityMiddleware("release_admin"),
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/release-plans", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer secret")
	routes.ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Fatalf("expected release admin to pass RBAC and reach business handler, body=%s", rec.Body.String())
	}
}

func TestRequirePermissionRejectsReleaseAdminSystemWrite(t *testing.T) {
	handler := NewHandler(NewService(Config{}))
	routes := handler.RoutesWithOptions(RouteOptions{
		AdminMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("secret"),
			AdminIdentityMiddleware("release_admin"),
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/system/users", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer secret")
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected release admin system write to be forbidden, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRequirePermissionAllowsSystemAdminWildcard(t *testing.T) {
	handler := NewHandler(NewService(Config{}))
	routes := handler.RoutesWithOptions(RouteOptions{
		AdminMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("secret"),
			AdminIdentityMiddleware("system_admin"),
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/system/users", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer secret")
	routes.ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Fatalf("expected system admin wildcard to pass RBAC and reach business handler, body=%s", rec.Body.String())
	}
}

func TestRoutesExposeClientLifecycleEndpoints(t *testing.T) {
	handler := NewHandler(NewService(Config{
		LatestVersionCode:       20,
		CurrentVersionAvailable: true,
	}))
	routes := handler.Routes()

	for _, tc := range []struct {
		name string
		path string
		body string
	}{
		{
			name: "update check",
			path: "/api/v1/app/update-check",
			body: `{"deviceId":"device-1","versionCode":20}`,
		},
		{
			name: "heartbeat",
			path: "/api/v1/app/heartbeat",
			body: `{"deviceId":"device-1","versionCode":20,"resourceVersion":"20260602.1"}`,
		},
		{
			name: "task preflight",
			path: "/api/v1/apps/task-preflight",
			body: `{"deviceId":"device-1","versionCode":20,"resourceVersion":"20260602.1"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			routes.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected %s to return 200, got %d body=%s", tc.path, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRoutesWithOptionsProtectsCIEndpoints(t *testing.T) {
	handler := NewHandler(NewService(Config{}))
	routes := handler.RoutesWithOptions(RouteOptions{
		CIMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("secret"),
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ci/artifacts", strings.NewReader(`{}`))
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected CI endpoint to require token, got %d", rec.Code)
	}
}

func TestRoutesWithOptionsProtectsWorkerEndpoints(t *testing.T) {
	handler := NewHandler(NewService(Config{}))
	routes := handler.RoutesWithOptions(RouteOptions{
		CIMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("secret"),
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workers/register", strings.NewReader(`{}`))
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected worker endpoint to require token, got %d", rec.Code)
	}
}

func TestRoutesWithOptionsProtectsAdminWorkerEndpoints(t *testing.T) {
	handler := NewHandler(NewService(Config{}))
	routes := handler.RoutesWithOptions(RouteOptions{
		AdminMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("secret"),
			AdminIdentityMiddleware("system_admin"),
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/workers/tasks", strings.NewReader(`{}`))
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected admin worker endpoint to require admin token, got %d", rec.Code)
	}
}

func TestRoutesWithOptionsProtectsArtifactCenterEndpoints(t *testing.T) {
	handler := NewHandler(NewService(Config{}))
	routes := handler.RoutesWithOptions(RouteOptions{
		AdminMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("secret"),
			AdminIdentityMiddleware("system_admin"),
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/api/artifacts/artifact-1", nil)
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected artifact center endpoint to require admin token, got %d", rec.Code)
	}
}

func TestRoutesWithOptionsProtectsBuildCenterConfigEndpoints(t *testing.T) {
	handler := NewHandler(NewService(Config{}))
	routes := handler.RoutesWithOptions(RouteOptions{
		AdminMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("secret"),
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/build-center/projects/release-center/repositories", strings.NewReader(`{}`))
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected build center config endpoint to require admin token, got %d", rec.Code)
	}
}

func TestRoutesWithOptionsProtectsDeploymentEndpoints(t *testing.T) {
	handler := NewHandler(NewService(Config{}))
	routes := handler.RoutesWithOptions(RouteOptions{
		AdminMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("secret"),
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/deployments", strings.NewReader(`{}`))
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected deployment endpoint to require admin token, got %d", rec.Code)
	}
}

func TestRoutesWithOptionsProtectsReleasePlanEndpoints(t *testing.T) {
	handler := NewHandler(NewService(Config{}))
	routes := handler.RoutesWithOptions(RouteOptions{
		AdminMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("secret"),
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/release-plans", strings.NewReader(`{}`))
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected release plan endpoint to require admin token, got %d", rec.Code)
	}
}

func TestRoutesWithOptionsProtectsReleasePlanDeploymentEndpoint(t *testing.T) {
	handler := NewHandler(NewService(Config{}))
	routes := handler.RoutesWithOptions(RouteOptions{
		AdminMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("secret"),
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/release-plans/plan-1/deployments", strings.NewReader(`{}`))
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected release plan deployment endpoint to require admin token, got %d", rec.Code)
	}
}

func TestReleasePlanActionAllowsEmptyBody(t *testing.T) {
	handler := NewHandler(NewService(Config{}))
	routes := handler.RoutesWithOptions(RouteOptions{
		AdminMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("secret"),
			AdminIdentityMiddleware("system_admin"),
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/api/release-plans/plan-1/publish", http.NoBody)
	req.Header.Set("Authorization", "Bearer secret")
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected empty action body to reach business handler, got %d", rec.Code)
	}
}

func TestRoutesExposeBuildDownloadEndpoints(t *testing.T) {
	handler := NewHandler(NewService(Config{}))
	routes := handler.Routes()

	for _, path := range []string{
		"/api/v1/app/builds/build-1/download",
		"/api/v1/apps/builds/build-1/download",
		"/api/v1/app/build-artifacts/artifact-1/download",
		"/api/v1/apps/build-artifacts/artifact-1/download",
	} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			routes.ServeHTTP(rec, req)
			if rec.Code == http.StatusNotFound {
				t.Fatalf("expected build download route to be mounted, got 404")
			}
		})
	}
}

func TestContentTypeForFile(t *testing.T) {
	for _, tc := range []struct {
		fileName string
		want     string
	}{
		{fileName: "app.apk", want: "application/vnd.android.package-archive"},
		{fileName: "web-dist.tar.gz", want: "application/gzip"},
		{fileName: "bundle.zip", want: "application/zip"},
		{fileName: "manifest.json", want: "application/json; charset=utf-8"},
	} {
		t.Run(tc.fileName, func(t *testing.T) {
			if got := contentTypeForFile(tc.fileName); got != tc.want {
				t.Fatalf("contentTypeForFile(%q) = %q, want %q", tc.fileName, got, tc.want)
			}
		})
	}
}

func TestNormalizeArtifactTypeSupportsBuildCenterArtifacts(t *testing.T) {
	for _, artifactType := range []string{
		"apk",
		"aab",
		"zip",
		"web_dist",
		"binary",
		"server_binary",
		"docker_image",
		"windows_exe",
		"windows_msi",
		"windows_installer",
		"windows_archive",
		"artifact",
	} {
		t.Run(artifactType, func(t *testing.T) {
			if got := normalizeArtifactType(artifactType); got != artifactType {
				t.Fatalf("normalizeArtifactType(%q) = %q, want %q", artifactType, got, artifactType)
			}
		})
	}
}
