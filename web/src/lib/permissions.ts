export type AdminRole = "system_admin" | "release_admin" | "release_viewer" | string;

const rolePermissions: Record<string, string[]> = {
  system_admin: ["system:*", "release:*", "build:*", "deploy:*", "worker:*", "integration:*"],
  release_admin: [
    "release:read",
    "release:write",
    "release:approve",
    "release:rollback",
    "release:audit",
    "build:read",
    "build:write",
    "deploy:read",
    "deploy:write",
    "deploy:approve",
    "deploy:rollback",
    "worker:read",
    "worker:write",
    "integration:read",
    "integration:write",
  ],
  release_viewer: ["release:read", "release:audit", "build:read", "deploy:read", "worker:read", "integration:read"],
};

export function permissionsForRole(role: AdminRole) {
  return rolePermissions[role] ?? rolePermissions.release_viewer;
}

export function hasPermission(role: AdminRole, permission: string) {
  return permissionsForRole(role).some((candidate) => {
    if (candidate === "*" || candidate === permission) return true;
    if (candidate.endsWith(":*")) return permission.startsWith(candidate.slice(0, -1));
    return false;
  });
}

export function missingPermissionText(permission: string) {
  return `当前角色缺少 ${permission} 权限`;
}
