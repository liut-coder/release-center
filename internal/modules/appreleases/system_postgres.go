package appreleases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *PostgresStore) SystemManagementOverview(ctx context.Context) (SystemManagementOverview, error) {
	users, err := s.listSystemUsers(ctx)
	if err != nil {
		return SystemManagementOverview{}, err
	}
	roles, err := s.listSystemRoles(ctx)
	if err != nil {
		return SystemManagementOverview{}, err
	}
	permissions, err := s.listSystemPermissions(ctx)
	if err != nil {
		return SystemManagementOverview{}, err
	}
	dictionaries, err := s.listSystemDictionaries(ctx)
	if err != nil {
		return SystemManagementOverview{}, err
	}
	menus, err := s.listSystemMenus(ctx)
	if err != nil {
		return SystemManagementOverview{}, err
	}
	return SystemManagementOverview{
		Users:        users,
		Roles:        roles,
		Permissions:  permissions,
		Dictionaries: dictionaries,
		Menus:        menus,
	}, nil
}

func (s *PostgresStore) AdminRolePermissions(ctx context.Context, account string) ([]string, bool, error) {
	var permissions []string
	var userEnabled bool
	var roleEnabled bool
	err := s.db.QueryRow(ctx, `
		select r.permissions, u.status = 'enabled', r.enabled
		from system_users u
		join system_roles r on r.tenant_id = u.tenant_id and r.code = u.role_code
		where u.tenant_id = 'default' and u.account = $1
		limit 1
	`, account).Scan(&permissions, &userEnabled, &roleEnabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return permissions, userEnabled && roleEnabled, nil
}

func (s *PostgresStore) CreateSystemUser(ctx context.Context, req CreateSystemUserRequest) (SystemUserAdmin, error) {
	var id string
	err := s.db.QueryRow(ctx, `
		insert into system_users (id, tenant_id, name, account, role_code, department, status, metadata)
		values ($1, 'default', $2, $3, $4, $5, $6, '{"source":"admin_api"}'::jsonb)
		on conflict (tenant_id, account) do update set
			name = excluded.name,
			role_code = excluded.role_code,
			department = excluded.department,
			status = excluded.status,
			metadata = system_users.metadata || excluded.metadata,
			updated_at = now()
		returning id::text
	`, uuid.NewString(), req.Name, req.Account, req.RoleCode, req.Department, req.Status).Scan(&id)
	if err != nil {
		return SystemUserAdmin{}, err
	}
	return s.getSystemUser(ctx, id)
}

func (s *PostgresStore) UpdateSystemUserStatus(ctx context.Context, id, status string) (SystemUserAdmin, error) {
	_, err := s.db.Exec(ctx, `
		update system_users
		set status = $2, updated_at = now()
		where tenant_id = 'default' and id = $1
	`, id, status)
	if err != nil {
		return SystemUserAdmin{}, err
	}
	return s.getSystemUser(ctx, id)
}

func (s *PostgresStore) CreateSystemRole(ctx context.Context, req CreateSystemRoleRequest) (SystemRoleAdmin, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var id string
	err := s.db.QueryRow(ctx, `
		insert into system_roles (id, tenant_id, name, code, data_scope, permissions, enabled, metadata)
		values ($1, 'default', $2, $3, $4, $5, $6, '{"source":"admin_api"}'::jsonb)
		on conflict (tenant_id, code) do update set
			name = excluded.name,
			data_scope = excluded.data_scope,
			permissions = excluded.permissions,
			enabled = excluded.enabled,
			metadata = system_roles.metadata || excluded.metadata,
			updated_at = now()
		returning id::text
	`, uuid.NewString(), req.Name, req.Code, req.Scope, req.Permissions, enabled).Scan(&id)
	if err != nil {
		return SystemRoleAdmin{}, err
	}
	return s.getSystemRole(ctx, id)
}

func (s *PostgresStore) UpdateSystemRoleEnabled(ctx context.Context, id string, enabled bool) (SystemRoleAdmin, error) {
	_, err := s.db.Exec(ctx, `
		update system_roles
		set enabled = $2, updated_at = now()
		where tenant_id = 'default' and id = $1
	`, id, enabled)
	if err != nil {
		return SystemRoleAdmin{}, err
	}
	return s.getSystemRole(ctx, id)
}

func (s *PostgresStore) CreateSystemPermission(ctx context.Context, req CreateSystemPermissionRequest) (SystemPermissionAdmin, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var id string
	err := s.db.QueryRow(ctx, `
		insert into system_permissions (id, tenant_id, name, code, module, permission_type, enabled, metadata)
		values ($1, 'default', $2, $3, $4, $5, $6, '{"source":"admin_api"}'::jsonb)
		on conflict (tenant_id, code) do update set
			name = excluded.name,
			module = excluded.module,
			permission_type = excluded.permission_type,
			enabled = excluded.enabled,
			metadata = system_permissions.metadata || excluded.metadata,
			updated_at = now()
		returning id::text
	`, uuid.NewString(), req.Name, req.Code, req.Module, req.Type, enabled).Scan(&id)
	if err != nil {
		return SystemPermissionAdmin{}, err
	}
	return s.getSystemPermission(ctx, id)
}

func (s *PostgresStore) UpdateSystemPermissionEnabled(ctx context.Context, id string, enabled bool) (SystemPermissionAdmin, error) {
	_, err := s.db.Exec(ctx, `
		update system_permissions
		set enabled = $2, updated_at = now()
		where tenant_id = 'default' and id = $1
	`, id, enabled)
	if err != nil {
		return SystemPermissionAdmin{}, err
	}
	return s.getSystemPermission(ctx, id)
}

func (s *PostgresStore) CreateSystemDictionary(ctx context.Context, req CreateSystemDictionaryRequest) (SystemDictionaryAdmin, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var id string
	err := s.db.QueryRow(ctx, `
		insert into system_dictionaries (id, tenant_id, group_key, item_key, label, item_value, sort_order, enabled, metadata)
		values ($1, 'default', $2, $3, $4, $5, $6, $7, '{"source":"admin_api"}'::jsonb)
		on conflict (tenant_id, group_key, item_key) do update set
			label = excluded.label,
			item_value = excluded.item_value,
			sort_order = excluded.sort_order,
			enabled = excluded.enabled,
			metadata = system_dictionaries.metadata || excluded.metadata,
			updated_at = now()
		returning id::text
	`, uuid.NewString(), req.Group, req.Key, req.Label, req.Value, req.Sort, enabled).Scan(&id)
	if err != nil {
		return SystemDictionaryAdmin{}, err
	}
	return s.getSystemDictionary(ctx, id)
}

func (s *PostgresStore) UpdateSystemDictionaryEnabled(ctx context.Context, id string, enabled bool) (SystemDictionaryAdmin, error) {
	_, err := s.db.Exec(ctx, `
		update system_dictionaries
		set enabled = $2, updated_at = now()
		where tenant_id = 'default' and id = $1
	`, id, enabled)
	if err != nil {
		return SystemDictionaryAdmin{}, err
	}
	return s.getSystemDictionary(ctx, id)
}

func (s *PostgresStore) CreateSystemMenu(ctx context.Context, req CreateSystemMenuRequest) (SystemMenuAdmin, error) {
	visible := true
	if req.Visible != nil {
		visible = *req.Visible
	}
	var id string
	err := s.db.QueryRow(ctx, `
		insert into system_menus (id, tenant_id, title, path, icon, parent_title, sort_order, visible, metadata)
		values ($1, 'default', $2, $3, $4, $5, $6, $7, '{"source":"admin_api"}'::jsonb)
		on conflict (tenant_id, path) do update set
			title = excluded.title,
			icon = excluded.icon,
			parent_title = excluded.parent_title,
			sort_order = excluded.sort_order,
			visible = excluded.visible,
			metadata = system_menus.metadata || excluded.metadata,
			updated_at = now()
		returning id::text
	`, uuid.NewString(), req.Title, req.Path, req.Icon, req.Parent, req.Sort, visible).Scan(&id)
	if err != nil {
		return SystemMenuAdmin{}, err
	}
	return s.getSystemMenu(ctx, id)
}

func (s *PostgresStore) UpdateSystemMenuVisible(ctx context.Context, id string, visible bool) (SystemMenuAdmin, error) {
	_, err := s.db.Exec(ctx, `
		update system_menus
		set visible = $2, updated_at = now()
		where tenant_id = 'default' and id = $1
	`, id, visible)
	if err != nil {
		return SystemMenuAdmin{}, err
	}
	return s.getSystemMenu(ctx, id)
}

func (s *PostgresStore) listSystemUsers(ctx context.Context) ([]SystemUserAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select u.id::text, u.name, u.account, coalesce(r.name, u.role_code), u.role_code,
		       u.department, u.status, coalesce(to_char(u.last_login_at, 'YYYY-MM-DD HH24:MI'), '-'),
		       u.created_at, u.updated_at
		from system_users u
		left join system_roles r on r.tenant_id = u.tenant_id and r.code = u.role_code
		where u.tenant_id = 'default'
		order by u.updated_at desc, u.created_at desc
		limit 200
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []SystemUserAdmin
	for rows.Next() {
		var item SystemUserAdmin
		if err := rows.Scan(&item.ID, &item.Name, &item.Account, &item.Role, &item.RoleCode, &item.Department, &item.Status, &item.LastLogin, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *PostgresStore) listSystemRoles(ctx context.Context) ([]SystemRoleAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select r.id::text, r.name, r.code, count(u.id)::int, r.data_scope, r.permissions, r.enabled, r.created_at, r.updated_at
		from system_roles r
		left join system_users u on u.tenant_id = r.tenant_id and u.role_code = r.code
		where r.tenant_id = 'default'
		group by r.id, r.name, r.code, r.data_scope, r.permissions, r.enabled, r.created_at, r.updated_at
		order by r.updated_at desc, r.created_at desc
		limit 200
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []SystemRoleAdmin
	for rows.Next() {
		var item SystemRoleAdmin
		if err := rows.Scan(&item.ID, &item.Name, &item.Code, &item.Users, &item.Scope, &item.Permissions, &item.Enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *PostgresStore) listSystemPermissions(ctx context.Context) ([]SystemPermissionAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, name, code, module, permission_type, enabled, created_at, updated_at
		from system_permissions
		where tenant_id = 'default'
		order by module, code
		limit 500
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []SystemPermissionAdmin
	for rows.Next() {
		var item SystemPermissionAdmin
		if err := rows.Scan(&item.ID, &item.Name, &item.Code, &item.Module, &item.Type, &item.Enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *PostgresStore) listSystemDictionaries(ctx context.Context) ([]SystemDictionaryAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, group_key, item_key, label, item_value, sort_order, enabled, created_at, updated_at
		from system_dictionaries
		where tenant_id = 'default'
		order by group_key, sort_order, item_key
		limit 500
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []SystemDictionaryAdmin
	for rows.Next() {
		var item SystemDictionaryAdmin
		if err := rows.Scan(&item.ID, &item.Group, &item.Key, &item.Label, &item.Value, &item.Sort, &item.Enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *PostgresStore) listSystemMenus(ctx context.Context) ([]SystemMenuAdmin, error) {
	rows, err := s.db.Query(ctx, `
		select id::text, title, path, icon, parent_title, sort_order, visible, created_at, updated_at
		from system_menus
		where tenant_id = 'default'
		order by sort_order, title
		limit 500
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []SystemMenuAdmin
	for rows.Next() {
		var item SystemMenuAdmin
		if err := rows.Scan(&item.ID, &item.Title, &item.Path, &item.Icon, &item.Parent, &item.Sort, &item.Visible, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *PostgresStore) getSystemUser(ctx context.Context, id string) (SystemUserAdmin, error) {
	var item SystemUserAdmin
	err := s.db.QueryRow(ctx, `
		select u.id::text, u.name, u.account, coalesce(r.name, u.role_code), u.role_code,
		       u.department, u.status, coalesce(to_char(u.last_login_at, 'YYYY-MM-DD HH24:MI'), '-'),
		       u.created_at, u.updated_at
		from system_users u
		left join system_roles r on r.tenant_id = u.tenant_id and r.code = u.role_code
		where u.tenant_id = 'default' and u.id = $1
	`, id).Scan(&item.ID, &item.Name, &item.Account, &item.Role, &item.RoleCode, &item.Department, &item.Status, &item.LastLogin, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *PostgresStore) getSystemRole(ctx context.Context, id string) (SystemRoleAdmin, error) {
	var item SystemRoleAdmin
	err := s.db.QueryRow(ctx, `
		select r.id::text, r.name, r.code, count(u.id)::int, r.data_scope, r.permissions, r.enabled, r.created_at, r.updated_at
		from system_roles r
		left join system_users u on u.tenant_id = r.tenant_id and u.role_code = r.code
		where r.tenant_id = 'default' and r.id = $1
		group by r.id, r.name, r.code, r.data_scope, r.permissions, r.enabled, r.created_at, r.updated_at
	`, id).Scan(&item.ID, &item.Name, &item.Code, &item.Users, &item.Scope, &item.Permissions, &item.Enabled, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *PostgresStore) getSystemPermission(ctx context.Context, id string) (SystemPermissionAdmin, error) {
	var item SystemPermissionAdmin
	err := s.db.QueryRow(ctx, `
		select id::text, name, code, module, permission_type, enabled, created_at, updated_at
		from system_permissions
		where tenant_id = 'default' and id = $1
	`, id).Scan(&item.ID, &item.Name, &item.Code, &item.Module, &item.Type, &item.Enabled, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *PostgresStore) getSystemDictionary(ctx context.Context, id string) (SystemDictionaryAdmin, error) {
	var item SystemDictionaryAdmin
	err := s.db.QueryRow(ctx, `
		select id::text, group_key, item_key, label, item_value, sort_order, enabled, created_at, updated_at
		from system_dictionaries
		where tenant_id = 'default' and id = $1
	`, id).Scan(&item.ID, &item.Group, &item.Key, &item.Label, &item.Value, &item.Sort, &item.Enabled, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *PostgresStore) getSystemMenu(ctx context.Context, id string) (SystemMenuAdmin, error) {
	var item SystemMenuAdmin
	err := s.db.QueryRow(ctx, `
		select id::text, title, path, icon, parent_title, sort_order, visible, created_at, updated_at
		from system_menus
		where tenant_id = 'default' and id = $1
	`, id).Scan(&item.ID, &item.Title, &item.Path, &item.Icon, &item.Parent, &item.Sort, &item.Visible, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}
