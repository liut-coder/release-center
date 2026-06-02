package appreleases

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type RouteOptions struct {
	AdminMiddleware           []func(http.Handler) http.Handler
	AdminPermissionMiddleware func(permission string) func(http.Handler) http.Handler
	CIMiddleware              []func(http.Handler) http.Handler
	WebhookMiddleware         []func(http.Handler) http.Handler
	ClientMiddleware          []func(http.Handler) http.Handler
}

func (h *Handler) Routes() chi.Router {
	return h.RoutesWithOptions(RouteOptions{})
}

func (h *Handler) RoutesWithOptions(opts RouteOptions) chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		useAll(r, opts.AdminMiddleware)
		r.With(adminPermission(opts, "release:read")).Get("/admin/api/app-releases", h.AdminOverview)
		r.With(adminPermission(opts, "release:read")).Get("/admin/api/apps", h.AdminOverview)
		r.With(adminPermission(opts, "release:read")).Get("/admin/api/builds", h.AdminOverview)
		r.With(adminPermission(opts, "release:read")).Get("/admin/api/artifacts", h.AdminOverview)
		r.With(adminPermission(opts, "release:audit")).Get("/admin/api/audit-logs", h.AdminOverview)
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/apps", h.CreateApp)
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/apps/{app_id}/enable", h.AppAction("enable"))
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/apps/{app_id}/disable", h.AppAction("disable"))

		r.With(adminPermission(opts, "admin:access")).Get("/admin/api/system/overview", h.SystemManagementOverview)
		r.With(adminPermission(opts, "system:read")).Get("/admin/api/system/users", h.SystemManagementOverview)
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/users", h.CreateSystemUser)
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/users/{user_id}/enable", h.SystemUserAction("enable"))
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/users/{user_id}/disable", h.SystemUserAction("disable"))
		r.With(adminPermission(opts, "system:read")).Get("/admin/api/system/roles", h.SystemManagementOverview)
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/roles", h.CreateSystemRole)
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/roles/{role_id}/enable", h.SystemRoleAction("enable"))
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/roles/{role_id}/disable", h.SystemRoleAction("disable"))
		r.With(adminPermission(opts, "system:read")).Get("/admin/api/system/permissions", h.SystemManagementOverview)
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/permissions", h.CreateSystemPermission)
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/permissions/{permission_id}/enable", h.SystemPermissionAction("enable"))
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/permissions/{permission_id}/disable", h.SystemPermissionAction("disable"))
		r.With(adminPermission(opts, "system:read")).Get("/admin/api/system/dictionaries", h.SystemManagementOverview)
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/dictionaries", h.CreateSystemDictionary)
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/dictionaries/{dictionary_id}/enable", h.SystemDictionaryAction("enable"))
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/dictionaries/{dictionary_id}/disable", h.SystemDictionaryAction("disable"))
		r.With(adminPermission(opts, "system:read")).Get("/admin/api/system/menus", h.SystemManagementOverview)
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/menus", h.CreateSystemMenu)
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/menus/{menu_id}/show", h.SystemMenuAction("show"))
		r.With(adminPermission(opts, "system:write")).Post("/admin/api/system/menus/{menu_id}/hide", h.SystemMenuAction("hide"))

		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-releases/build", h.CreateBuild)
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-releases", h.CreateRelease)
		r.With(adminPermission(opts, "release:read")).Get("/admin/api/app-releases/builds/{job_id}", h.GetBuildJob)
		r.With(adminPermission(opts, "release:read")).Get("/admin/api/app-releases/builds/{job_id}/logs", h.GetBuildLogs)
		r.With(adminPermission(opts, "release:read")).Get("/admin/api/app-releases/builds/{build_id}/download", h.DownloadBuild)
		r.With(adminPermission(opts, "release:read")).Get("/admin/api/app-releases/{release_id}/download", h.DownloadRelease)
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-releases/{release_id}/publish", h.ReleaseAction("publish"))
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-releases/{release_id}/pause", h.ReleaseAction("pause"))
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-releases/{release_id}/recall", h.ReleaseAction("recall"))
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-releases/{release_id}/unpublish", h.ReleaseAction("unpublish"))
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-releases/{release_id}/rollback", h.ReleaseAction("rollback"))
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-releases/{release_id}/rollout", h.UpdateReleaseRollout)
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-releases/{release_id}/notes", h.UpdateReleaseNotes)

		r.With(adminPermission(opts, "release:write")).Post("/admin/api/releases", h.CreateRelease)
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/releases/{release_id}/publish", h.ReleaseAction("publish"))
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/releases/{release_id}/pause", h.ReleaseAction("pause"))
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/releases/{release_id}/rollback", h.ReleaseAction("rollback"))

		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-resources", h.CreateResourceVersion)
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-resources/{resource_id}/publish", h.ResourceAction("publish"))
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-resources/{resource_id}/pause", h.ResourceAction("pause"))
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-resources/{resource_id}/recall", h.ResourceAction("recall"))
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-resources/{resource_id}/rollback", h.ResourceAction("rollback"))
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-resources/{resource_id}/rollout", h.UpdateResourceRollout)
		r.With(adminPermission(opts, "release:write")).Post("/admin/api/app-resources/{resource_id}/notes", h.UpdateResourceNotes)
	})

	r.Group(func(r chi.Router) {
		useAll(r, opts.CIMiddleware)
		r.Post("/api/v1/ci/builds", h.CreateCIBuild)
		r.Post("/api/v1/ci/artifacts", h.CreateCIArtifact)
		r.Post("/api/v1/ci/releases", h.CreateRelease)
	})

	r.Group(func(r chi.Router) {
		useAll(r, opts.WebhookMiddleware)
		r.Post("/api/v1/webhooks/github", h.Webhook("github"))
		r.Post("/api/v1/webhooks/gitea", h.Webhook("gitea"))
	})

	r.Group(func(r chi.Router) {
		useAll(r, opts.ClientMiddleware)
		r.Get("/api/v1/app/releases/check", h.Check)
		r.Post("/api/v1/app/update-check", h.UpdateCheck)
		r.Post("/api/v1/app/releases/check", h.UpdateCheck)
		r.Post("/api/v1/app/resource-check", h.ResourceCheck)
		r.Post("/api/v1/app/resources/check", h.ResourceCheck)
		r.Post("/api/v1/app/task-preflight", h.TaskPreflight)
		r.Get("/api/v1/app/resources/{resource_id}/manifest", h.DownloadResourceManifest)
		r.Get("/api/v1/app/resources/packages/{package_id}/download", h.DownloadResourcePackage)
		r.Post("/api/v1/app/heartbeat", h.Heartbeat)
		r.Post("/api/v1/app/update-event", h.UpdateEvent)

		r.Post("/api/v1/apps/update-check", h.UpdateCheck)
		r.Post("/api/v1/apps/resource-check", h.ResourceCheck)
		r.Post("/api/v1/apps/task-preflight", h.TaskPreflight)
		r.Get("/api/v1/apps/resources/{resource_id}/manifest", h.DownloadResourceManifest)
		r.Get("/api/v1/apps/resources/packages/{package_id}/download", h.DownloadResourcePackage)
		r.Post("/api/v1/apps/heartbeat", h.Heartbeat)
		r.Post("/api/v1/apps/update-events", h.UpdateEvent)
	})

	return r
}

func adminPermission(opts RouteOptions, permission string) func(http.Handler) http.Handler {
	if opts.AdminPermissionMiddleware == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return opts.AdminPermissionMiddleware(permission)
}

func useAll(r chi.Router, middlewares []func(http.Handler) http.Handler) {
	for _, middleware := range middlewares {
		if middleware != nil {
			r.Use(middleware)
		}
	}
}

func BearerTokenMiddleware(tokens ...string) func(http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token != "" {
			allowed[token] = struct{}{}
		}
	}
	return func(next http.Handler) http.Handler {
		if len(allowed) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := strings.TrimSpace(r.Header.Get("Authorization"))
			token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			if _, ok := allowed[token]; !ok || !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
