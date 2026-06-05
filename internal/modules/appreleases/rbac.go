package appreleases

import (
	"context"
	"net/http"
	"strings"

	"github.com/liut-coder/game-helper-server/internal/platform/httpx"
)

type adminContextKey string

const adminIdentityKey adminContextKey = "admin_identity"

type AdminIdentity struct {
	Account     string
	RoleCode    string
	Permissions []string
}

var defaultRolePermissions = map[string][]string{
	"system_admin":  {"system:*", "release:*", "build:*", "deploy:*", "worker:*", "integration:*"},
	"release_admin": {"release:read", "release:write", "release:audit", "build:read", "build:write", "deploy:read", "deploy:write", "worker:read", "integration:read", "integration:write"},
	"release_viewer": {"release:read", "release:audit", "build:read", "deploy:read", "worker:read", "integration:read"},
}

func AdminIdentityMiddleware(defaultRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := firstNonBlank(r.Header.Get("X-Admin-Role"), defaultRole, "system_admin")
			account := firstNonBlank(r.Header.Get("X-Admin-Account"), "admin")
			permissions := permissionsForRole(role)
			ctx := context.WithValue(r.Context(), adminIdentityKey, AdminIdentity{
				Account:     strings.TrimSpace(account),
				RoleCode:    strings.TrimSpace(role),
				Permissions: permissions,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequirePermission(permission string) func(http.Handler) http.Handler {
	permission = strings.TrimSpace(permission)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity := AdminIdentityFromContext(r.Context())
			if !identity.HasPermission(permission) {
				httpx.Error(w, r, http.StatusForbidden, "rbac.forbidden", "当前账号没有执行该操作的权限", map[string]any{
					"required_permission": permission,
					"role_code":           identity.RoleCode,
					"account":             identity.Account,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func AdminIdentityFromContext(ctx context.Context) AdminIdentity {
	identity, _ := ctx.Value(adminIdentityKey).(AdminIdentity)
	return identity
}

func (i AdminIdentity) HasPermission(permission string) bool {
	permission = strings.TrimSpace(permission)
	if permission == "" {
		return true
	}
	for _, candidate := range i.Permissions {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == permission {
			return true
		}
		if strings.HasSuffix(candidate, ":*") && strings.HasPrefix(permission, strings.TrimSuffix(candidate, "*")) {
			return true
		}
	}
	return false
}

func permissionsForRole(role string) []string {
	role = strings.TrimSpace(role)
	if permissions, ok := defaultRolePermissions[role]; ok {
		return append([]string{}, permissions...)
	}
	return []string{"release:read", "build:read", "deploy:read", "worker:read", "integration:read"}
}
