package appreleases

import (
	"encoding/json"
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

func TestRoutesWithOptionsEnforcesAdminRBAC(t *testing.T) {
	service := NewService(Config{})
	handler := NewHandler(service)
	routes := handler.RoutesWithOptions(RouteOptions{
		AdminMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("admin-token", "release-token"),
		},
		AdminPermissionMiddleware: AdminRBACMiddleware(service, map[string]string{
			"admin-token":   "system.admin",
			"release-token": "release.admin",
		}),
	})

	for _, tc := range []struct {
		name      string
		method    string
		path      string
		token     string
		body      string
		wantCode  int
		wantError string
	}{
		{
			name:     "release admin can read release center",
			method:   http.MethodGet,
			path:     "/admin/api/app-releases",
			token:    "release-token",
			wantCode: http.StatusOK,
		},
		{
			name:     "release admin can initialize admin shell",
			method:   http.MethodGet,
			path:     "/admin/api/system/overview",
			token:    "release-token",
			wantCode: http.StatusOK,
		},
		{
			name:      "release admin cannot write system management",
			method:    http.MethodPost,
			path:      "/admin/api/system/users",
			token:     "release-token",
			body:      `{"name":"受限用户","account":"limited.user","role_code":"release_viewer"}`,
			wantCode:  http.StatusForbidden,
			wantError: "auth.permission_denied",
		},
		{
			name:     "system admin can write system management",
			method:   http.MethodPost,
			path:     "/admin/api/system/users",
			token:    "admin-token",
			body:     `{"name":"系统用户","account":"system.user","role_code":"release_viewer"}`,
			wantCode: http.StatusOK,
		},
		{
			name:     "missing token is rejected before RBAC",
			method:   http.MethodGet,
			path:     "/admin/api/system/overview",
			wantCode: http.StatusUnauthorized,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			routes.ServeHTTP(rec, req)
			if rec.Code != tc.wantCode {
				t.Fatalf("expected %d, got %d body=%s", tc.wantCode, rec.Code, rec.Body.String())
			}
			if tc.wantError != "" && !strings.Contains(rec.Body.String(), tc.wantError) {
				t.Fatalf("expected error %q in body=%s", tc.wantError, rec.Body.String())
			}
		})
	}
}

func TestSystemOverviewFiltersMenusByAdminPermissions(t *testing.T) {
	service := NewService(Config{})
	handler := NewHandler(service)
	routes := handler.RoutesWithOptions(RouteOptions{
		AdminMiddleware: []func(http.Handler) http.Handler{
			BearerTokenMiddleware("release-token"),
		},
		AdminPermissionMiddleware: AdminRBACMiddleware(service, map[string]string{
			"release-token": "release.admin",
		}),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/api/system/overview", nil)
	req.Header.Set("Authorization", "Bearer release-token")
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected system overview to initialize, got %d body=%s", rec.Code, rec.Body.String())
	}
	var overview SystemManagementOverview
	if err := json.NewDecoder(rec.Body).Decode(&overview); err != nil {
		t.Fatalf("decode overview: %v", err)
	}
	paths := map[string]bool{}
	for _, menu := range overview.Menus {
		paths[menu.Path] = true
	}
	if !paths["/dashboard"] || !paths["/release-center"] {
		t.Fatalf("expected dashboard and release center menus, got %#v", paths)
	}
	if paths["/system/users"] || paths["/system/roles"] || paths["/system/menus"] {
		t.Fatalf("expected system menus to be hidden for release admin, got %#v", paths)
	}
	if len(overview.Users) != 0 || len(overview.Roles) != 0 || len(overview.Permissions) != 0 {
		t.Fatalf("expected system management records to be hidden, got users=%d roles=%d permissions=%d", len(overview.Users), len(overview.Roles), len(overview.Permissions))
	}
}
