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
