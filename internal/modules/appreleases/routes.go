package appreleases

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type RouteOptions struct {
	AdminMiddleware   []func(http.Handler) http.Handler
	CIMiddleware      []func(http.Handler) http.Handler
	WebhookMiddleware []func(http.Handler) http.Handler
	ClientMiddleware  []func(http.Handler) http.Handler
}

func (h *Handler) Routes() chi.Router {
	return h.RoutesWithOptions(RouteOptions{})
}

func (h *Handler) RoutesWithOptions(opts RouteOptions) chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		useAll(r, opts.AdminMiddleware)
		r.Get("/admin/api/app-releases", h.AdminOverview)
		r.Get("/admin/api/apps", h.AdminOverview)
		r.Get("/admin/api/builds", h.AdminOverview)
		r.Get("/admin/api/artifacts", h.ArtifactCenterOverview)
		r.Get("/admin/api/artifacts/{artifact_id}", h.ArtifactCenterItem)
		r.Get("/admin/api/audit-logs", h.AdminOverview)
		r.Post("/admin/api/apps", h.CreateApp)
		r.Post("/admin/api/apps/{app_id}/enable", h.AppAction("enable"))
		r.Post("/admin/api/apps/{app_id}/disable", h.AppAction("disable"))

		r.Get("/admin/api/build-center/projects", h.BuildCenterProjects)
		r.Post("/admin/api/build-center/projects", h.CreateBuildCenterProject)
		r.Get("/admin/api/build-center/projects/{project_key}", h.BuildCenterProject)
		r.Post("/admin/api/build-center/projects/{project_key}/repositories", h.UpsertCodeRepository)
		r.Post("/admin/api/build-center/projects/{project_key}/profiles", h.UpsertBuildProfile)
		r.Post("/admin/api/build-center/projects/{project_key}/webhook-routes", h.UpsertWebhookRoute)
		r.Post("/admin/api/build-center/projects/{project_key}/runs", h.CreateBuildCenterRun)
		r.Get("/admin/api/build-center/runs/{run_id}", h.BuildCenterRun)
		r.Get("/admin/api/build-center/runs/{run_id}/logs", h.BuildCenterRunLogs)
		r.Get("/admin/api/deployment-targets", h.DeploymentTargets)
		r.Post("/admin/api/deployment-targets", h.CreateDeploymentTarget)
		r.Get("/admin/api/deployments", h.Deployments)
		r.Get("/admin/api/deployments/{deployment_id}", h.Deployment)
		r.Post("/admin/api/deployments", h.CreateDeployment)
		r.Post("/admin/api/deployments/{deployment_id}/complete", h.CompleteDeployment)
		r.Post("/admin/api/deployments/{deployment_id}/fail", h.FailDeployment)
		r.Get("/admin/api/workers", h.WorkerOverview)
		r.Post("/admin/api/workers/tasks", h.CreateWorkerTask)
		r.Get("/admin/api/release-plans", h.ReleasePlanOverview)
		r.Post("/admin/api/release-units", h.CreateReleaseUnit)
		r.Post("/admin/api/release-plans", h.CreateReleasePlan)
		r.Get("/admin/api/release-plans/{plan_id}", h.ReleasePlan)
		r.Post("/admin/api/release-plans/{plan_id}/publish", h.ReleasePlanAction("publish"))
		r.Post("/admin/api/release-plans/{plan_id}/pause", h.ReleasePlanAction("pause"))
		r.Post("/admin/api/release-plans/{plan_id}/rollback", h.ReleasePlanAction("rollback"))

		r.Get("/admin/api/system/overview", h.SystemManagementOverview)
		r.Get("/admin/api/system/users", h.SystemManagementOverview)
		r.Post("/admin/api/system/users", h.CreateSystemUser)
		r.Post("/admin/api/system/users/{user_id}/enable", h.SystemUserAction("enable"))
		r.Post("/admin/api/system/users/{user_id}/disable", h.SystemUserAction("disable"))
		r.Get("/admin/api/system/roles", h.SystemManagementOverview)
		r.Post("/admin/api/system/roles", h.CreateSystemRole)
		r.Post("/admin/api/system/roles/{role_id}/enable", h.SystemRoleAction("enable"))
		r.Post("/admin/api/system/roles/{role_id}/disable", h.SystemRoleAction("disable"))
		r.Get("/admin/api/system/permissions", h.SystemManagementOverview)
		r.Post("/admin/api/system/permissions", h.CreateSystemPermission)
		r.Post("/admin/api/system/permissions/{permission_id}/enable", h.SystemPermissionAction("enable"))
		r.Post("/admin/api/system/permissions/{permission_id}/disable", h.SystemPermissionAction("disable"))
		r.Get("/admin/api/system/dictionaries", h.SystemManagementOverview)
		r.Post("/admin/api/system/dictionaries", h.CreateSystemDictionary)
		r.Post("/admin/api/system/dictionaries/{dictionary_id}/enable", h.SystemDictionaryAction("enable"))
		r.Post("/admin/api/system/dictionaries/{dictionary_id}/disable", h.SystemDictionaryAction("disable"))
		r.Get("/admin/api/system/menus", h.SystemManagementOverview)
		r.Post("/admin/api/system/menus", h.CreateSystemMenu)
		r.Post("/admin/api/system/menus/{menu_id}/show", h.SystemMenuAction("show"))
		r.Post("/admin/api/system/menus/{menu_id}/hide", h.SystemMenuAction("hide"))

		r.Post("/admin/api/app-releases/build", h.CreateBuild)
		r.Post("/admin/api/app-releases", h.CreateRelease)
		r.Get("/admin/api/app-releases/builds/{job_id}", h.GetBuildJob)
		r.Get("/admin/api/app-releases/builds/{job_id}/logs", h.GetBuildLogs)
		r.Get("/admin/api/app-releases/builds/{build_id}/download", h.DownloadBuild)
		r.Get("/admin/api/app-releases/{release_id}/download", h.DownloadRelease)
		r.Post("/admin/api/app-releases/{release_id}/publish", h.ReleaseAction("publish"))
		r.Post("/admin/api/app-releases/{release_id}/pause", h.ReleaseAction("pause"))
		r.Post("/admin/api/app-releases/{release_id}/recall", h.ReleaseAction("recall"))
		r.Post("/admin/api/app-releases/{release_id}/unpublish", h.ReleaseAction("unpublish"))
		r.Post("/admin/api/app-releases/{release_id}/rollback", h.ReleaseAction("rollback"))
		r.Post("/admin/api/app-releases/{release_id}/rollout", h.UpdateReleaseRollout)
		r.Post("/admin/api/app-releases/{release_id}/notes", h.UpdateReleaseNotes)

		r.Post("/admin/api/releases", h.CreateRelease)
		r.Post("/admin/api/releases/{release_id}/publish", h.ReleaseAction("publish"))
		r.Post("/admin/api/releases/{release_id}/pause", h.ReleaseAction("pause"))
		r.Post("/admin/api/releases/{release_id}/rollback", h.ReleaseAction("rollback"))

		r.Post("/admin/api/app-resources", h.CreateResourceVersion)
		r.Post("/admin/api/app-resources/{resource_id}/publish", h.ResourceAction("publish"))
		r.Post("/admin/api/app-resources/{resource_id}/pause", h.ResourceAction("pause"))
		r.Post("/admin/api/app-resources/{resource_id}/recall", h.ResourceAction("recall"))
		r.Post("/admin/api/app-resources/{resource_id}/rollback", h.ResourceAction("rollback"))
		r.Post("/admin/api/app-resources/{resource_id}/rollout", h.UpdateResourceRollout)
		r.Post("/admin/api/app-resources/{resource_id}/notes", h.UpdateResourceNotes)
	})

	r.Group(func(r chi.Router) {
		useAll(r, opts.CIMiddleware)
		r.Post("/api/v1/ci/builds", h.CreateCIBuild)
		r.Post("/api/v1/ci/artifacts", h.CreateCIArtifact)
		r.Post("/api/v1/ci/releases", h.CreateRelease)
	})

	r.Group(func(r chi.Router) {
		useAll(r, opts.CIMiddleware)
		r.Post("/api/v1/workers/register", h.RegisterWorker)
		r.Post("/api/v1/workers/heartbeat", h.WorkerHeartbeat)
		r.Get("/api/v1/workers/tasks/next", h.NextWorkerTask)
		r.Post("/api/v1/workers/tasks/{task_id}/logs", h.AppendWorkerTaskLogs)
		r.Post("/api/v1/workers/tasks/{task_id}/artifacts", h.SaveWorkerTaskArtifacts)
		r.Post("/api/v1/workers/tasks/{task_id}/complete", h.CompleteWorkerTask)
		r.Post("/api/v1/workers/tasks/{task_id}/fail", h.FailWorkerTask)
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
		r.Get("/api/v1/app/builds/{build_id}/download", h.DownloadBuild)
		r.Get("/api/v1/app/build-artifacts/{artifact_id}/download", h.DownloadBuildArtifact)
		r.Post("/api/v1/app/heartbeat", h.Heartbeat)
		r.Post("/api/v1/app/update-event", h.UpdateEvent)

		r.Post("/api/v1/apps/update-check", h.UpdateCheck)
		r.Post("/api/v1/apps/resource-check", h.ResourceCheck)
		r.Post("/api/v1/apps/task-preflight", h.TaskPreflight)
		r.Get("/api/v1/apps/resources/{resource_id}/manifest", h.DownloadResourceManifest)
		r.Get("/api/v1/apps/resources/packages/{package_id}/download", h.DownloadResourcePackage)
		r.Get("/api/v1/apps/builds/{build_id}/download", h.DownloadBuild)
		r.Get("/api/v1/apps/build-artifacts/{artifact_id}/download", h.DownloadBuildArtifact)
		r.Post("/api/v1/apps/heartbeat", h.Heartbeat)
		r.Post("/api/v1/apps/update-events", h.UpdateEvent)
	})

	return r
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
