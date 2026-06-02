package appreleases

import (
	"context"
	"net/http"
	"strings"

	"github.com/liut-coder/game-helper-server/internal/platform/httpx"
)

type adminAccountContextKey struct{}

func AdminAccountFromContext(ctx context.Context) string {
	account, _ := ctx.Value(adminAccountContextKey{}).(string)
	return account
}

func AdminRBACMiddleware(service *Service, tokenAccounts map[string]string) func(permission string) func(http.Handler) http.Handler {
	accountsByToken := normalizeTokenAccounts(tokenAccounts)
	return func(permission string) func(http.Handler) http.Handler {
		permission = strings.TrimSpace(permission)
		return func(next http.Handler) http.Handler {
			if permission == "" || len(accountsByToken) == 0 {
				return next
			}
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				token := bearerToken(r)
				account := accountsByToken[token]
				if account == "" {
					httpx.Error(w, r, http.StatusUnauthorized, "auth.unauthorized", "后台 Token 无效或已失效", nil)
					return
				}
				allowed, err := service.AdminHasPermission(r.Context(), account, permission)
				if err != nil {
					httpx.Error(w, r, http.StatusInternalServerError, "auth.rbac_load_failed", "读取权限数据失败", map[string]any{"error": err.Error()})
					return
				}
				if !allowed {
					httpx.Error(w, r, http.StatusForbidden, "auth.permission_denied", "当前账号没有执行该操作的权限", map[string]any{"permission": permission})
					return
				}
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), adminAccountContextKey{}, account)))
			})
		}
	}
}

func normalizeTokenAccounts(values map[string]string) map[string]string {
	result := map[string]string{}
	for token, account := range values {
		token = strings.TrimSpace(token)
		account = strings.TrimSpace(account)
		if token == "" || account == "" {
			continue
		}
		result[token] = account
	}
	return result
}

func bearerToken(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
}

func accountHasPermission(overview SystemManagementOverview, account, permission string) bool {
	permissions, active := accountPermissionsFromOverview(overview, account)
	return active && permissionsAllow(permissions, permission)
}

func accountPermissionsFromOverview(overview SystemManagementOverview, account string) ([]string, bool) {
	account = strings.TrimSpace(account)
	if account == "" {
		return nil, false
	}
	var roleCode string
	for _, user := range overview.Users {
		if strings.EqualFold(user.Account, account) {
			if user.Status != "" && user.Status != "enabled" {
				return nil, false
			}
			roleCode = user.RoleCode
			break
		}
	}
	if roleCode == "" {
		return nil, false
	}
	for _, role := range overview.Roles {
		if role.Code != roleCode || !role.Enabled {
			continue
		}
		return append([]string(nil), role.Permissions...), true
	}
	return nil, false
}

func permissionsAllow(permissions []string, required string) bool {
	required = strings.TrimSpace(required)
	if required == "" {
		return true
	}
	if required == "admin:access" {
		return true
	}
	for _, permission := range permissions {
		permission = strings.TrimSpace(permission)
		if permission == "" {
			continue
		}
		if permission == "*" || permission == required {
			return true
		}
		if strings.HasSuffix(permission, ":*") && strings.HasPrefix(required, strings.TrimSuffix(permission, "*")) {
			return true
		}
		if strings.HasSuffix(required, ":read") && strings.TrimSuffix(required, ":read")+":write" == permission {
			return true
		}
	}
	return false
}
