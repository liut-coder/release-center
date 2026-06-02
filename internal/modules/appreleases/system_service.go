package appreleases

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func (s *Service) SystemManagementOverview(ctx context.Context) (SystemManagementOverview, error) {
	if store, ok := s.store.(SystemManagementStore); ok {
		overview, err := store.SystemManagementOverview(ctx)
		if err != nil {
			return SystemManagementOverview{}, err
		}
		overview.MessageZh = "系统管理数据已读取"
		return overview, nil
	}
	overview := s.demoSystemOverview()
	overview.MessageZh = "系统管理使用后端 Demo 数据"
	return overview, nil
}

func (s *Service) AdminHasPermission(ctx context.Context, account, permission string) (bool, error) {
	account = strings.TrimSpace(account)
	permission = strings.TrimSpace(permission)
	if account == "" || permission == "" {
		return false, nil
	}
	permissions, active, err := s.AdminPermissions(ctx, account)
	if err != nil {
		return false, err
	}
	return active && permissionsAllow(permissions, permission), nil
}

func (s *Service) AdminPermissions(ctx context.Context, account string) ([]string, bool, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return nil, false, nil
	}
	if store, ok := s.store.(SystemPermissionStore); ok {
		permissions, active, err := store.AdminRolePermissions(ctx, account)
		if err != nil {
			return nil, false, err
		}
		return permissions, active, nil
	}
	overview := s.demoSystemOverview()
	permissions, active := accountPermissionsFromOverview(overview, account)
	return permissions, active, nil
}

func (s *Service) FilterSystemManagementOverview(ctx context.Context, overview SystemManagementOverview, account string) SystemManagementOverview {
	permissions, active, err := s.AdminPermissions(ctx, account)
	if err != nil || !active {
		return SystemManagementOverview{MessageZh: overview.MessageZh}
	}
	canReadRelease := permissionsAllow(permissions, "release:read")
	canReadSystem := permissionsAllow(permissions, "system:read")
	canWriteSystem := permissionsAllow(permissions, "system:write")

	var menus []SystemMenuAdmin
	for _, menu := range overview.Menus {
		path := strings.TrimSpace(menu.Path)
		switch {
		case path == "/dashboard":
			menus = append(menus, menu)
		case path == "/release-center" && canReadRelease:
			menus = append(menus, menu)
		case canReadSystem || canWriteSystem:
			menus = append(menus, menu)
		}
	}
	overview.Menus = menus
	if !canReadSystem && !canWriteSystem {
		overview.Users = nil
		overview.Roles = nil
		overview.Permissions = nil
		overview.Dictionaries = nil
	}
	return overview
}

func (s *Service) CreateSystemUser(ctx context.Context, req CreateSystemUserRequest) (SystemUserActionResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Account = strings.TrimSpace(req.Account)
	req.RoleCode = safeCode(firstNonBlank(req.RoleCode, req.Role))
	req.Department = strings.TrimSpace(req.Department)
	req.Status = normalizeSystemStatus(req.Status)
	if req.Name == "" || req.Account == "" {
		return SystemUserActionResponse{}, fmt.Errorf("name and account are required")
	}
	if store, ok := s.store.(SystemManagementStore); ok {
		user, err := store.CreateSystemUser(ctx, req)
		if err != nil {
			return SystemUserActionResponse{}, err
		}
		s.insertSystemAudit(ctx, "system.user.create", "system_user", user.ID, map[string]any{"account": user.Account, "role_code": user.RoleCode})
		return SystemUserActionResponse{OK: true, User: user, MessageZh: "用户已保存"}, nil
	}
	user := s.createDemoSystemUser(req)
	return SystemUserActionResponse{OK: true, User: user, MessageZh: "用户已保存到 Demo 响应"}, nil
}

func (s *Service) SystemUserAction(ctx context.Context, id, action string) (SystemUserActionResponse, error) {
	status := "disabled"
	if strings.TrimSpace(action) == "enable" {
		status = "enabled"
	} else if strings.TrimSpace(action) != "disable" {
		return SystemUserActionResponse{}, fmt.Errorf("unsupported user action: %s", action)
	}
	if store, ok := s.store.(SystemManagementStore); ok {
		user, err := store.UpdateSystemUserStatus(ctx, strings.TrimSpace(id), status)
		if err != nil {
			return SystemUserActionResponse{}, err
		}
		s.insertSystemAudit(ctx, "system.user."+action, "system_user", user.ID, map[string]any{"status": user.Status})
		return SystemUserActionResponse{OK: true, User: user, MessageZh: "用户状态已更新"}, nil
	}
	return SystemUserActionResponse{OK: true, User: s.updateDemoSystemUserStatus(strings.TrimSpace(id), status), MessageZh: "用户状态已更新到 Demo 响应"}, nil
}

func (s *Service) CreateSystemRole(ctx context.Context, req CreateSystemRoleRequest) (SystemRoleActionResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Code = safeCode(req.Code)
	req.Scope = strings.TrimSpace(req.Scope)
	req.Permissions = normalizeStringList(req.Permissions)
	if req.Name == "" || req.Code == "" {
		return SystemRoleActionResponse{}, fmt.Errorf("name and code are required")
	}
	if store, ok := s.store.(SystemManagementStore); ok {
		role, err := store.CreateSystemRole(ctx, req)
		if err != nil {
			return SystemRoleActionResponse{}, err
		}
		s.insertSystemAudit(ctx, "system.role.create", "system_role", role.ID, map[string]any{"code": role.Code})
		return SystemRoleActionResponse{OK: true, Role: role, MessageZh: "角色已保存"}, nil
	}
	role := s.createDemoSystemRole(req)
	return SystemRoleActionResponse{OK: true, Role: role, MessageZh: "角色已保存到 Demo 响应"}, nil
}

func (s *Service) SystemRoleAction(ctx context.Context, id, action string) (SystemRoleActionResponse, error) {
	enabled := action == "enable"
	if action != "enable" && action != "disable" {
		return SystemRoleActionResponse{}, fmt.Errorf("unsupported role action: %s", action)
	}
	if store, ok := s.store.(SystemManagementStore); ok {
		role, err := store.UpdateSystemRoleEnabled(ctx, strings.TrimSpace(id), enabled)
		if err != nil {
			return SystemRoleActionResponse{}, err
		}
		s.insertSystemAudit(ctx, "system.role."+action, "system_role", role.ID, map[string]any{"enabled": role.Enabled})
		return SystemRoleActionResponse{OK: true, Role: role, MessageZh: "角色状态已更新"}, nil
	}
	return SystemRoleActionResponse{OK: true, Role: s.updateDemoSystemRoleEnabled(strings.TrimSpace(id), enabled), MessageZh: "角色状态已更新到 Demo 响应"}, nil
}

func (s *Service) CreateSystemPermission(ctx context.Context, req CreateSystemPermissionRequest) (SystemPermissionActionResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.Module = strings.TrimSpace(req.Module)
	req.Type = normalizePermissionType(req.Type)
	if req.Name == "" || req.Code == "" {
		return SystemPermissionActionResponse{}, fmt.Errorf("name and code are required")
	}
	if store, ok := s.store.(SystemManagementStore); ok {
		permission, err := store.CreateSystemPermission(ctx, req)
		if err != nil {
			return SystemPermissionActionResponse{}, err
		}
		s.insertSystemAudit(ctx, "system.permission.create", "system_permission", permission.ID, map[string]any{"code": permission.Code})
		return SystemPermissionActionResponse{OK: true, Permission: permission, MessageZh: "权限已保存"}, nil
	}
	permission := s.createDemoSystemPermission(req)
	return SystemPermissionActionResponse{OK: true, Permission: permission, MessageZh: "权限已保存到 Demo 响应"}, nil
}

func (s *Service) SystemPermissionAction(ctx context.Context, id, action string) (SystemPermissionActionResponse, error) {
	enabled := action == "enable"
	if action != "enable" && action != "disable" {
		return SystemPermissionActionResponse{}, fmt.Errorf("unsupported permission action: %s", action)
	}
	if store, ok := s.store.(SystemManagementStore); ok {
		permission, err := store.UpdateSystemPermissionEnabled(ctx, strings.TrimSpace(id), enabled)
		if err != nil {
			return SystemPermissionActionResponse{}, err
		}
		s.insertSystemAudit(ctx, "system.permission."+action, "system_permission", permission.ID, map[string]any{"enabled": permission.Enabled})
		return SystemPermissionActionResponse{OK: true, Permission: permission, MessageZh: "权限状态已更新"}, nil
	}
	return SystemPermissionActionResponse{OK: true, Permission: s.updateDemoSystemPermissionEnabled(strings.TrimSpace(id), enabled), MessageZh: "权限状态已更新到 Demo 响应"}, nil
}

func (s *Service) CreateSystemDictionary(ctx context.Context, req CreateSystemDictionaryRequest) (SystemDictionaryActionResponse, error) {
	req.Group = safeCode(req.Group)
	req.Key = safeCode(req.Key)
	req.Label = strings.TrimSpace(req.Label)
	req.Value = strings.TrimSpace(req.Value)
	if req.Group == "" || req.Key == "" || req.Label == "" {
		return SystemDictionaryActionResponse{}, fmt.Errorf("group, key and label are required")
	}
	if req.Sort <= 0 {
		req.Sort = 100
	}
	if store, ok := s.store.(SystemManagementStore); ok {
		dictionary, err := store.CreateSystemDictionary(ctx, req)
		if err != nil {
			return SystemDictionaryActionResponse{}, err
		}
		s.insertSystemAudit(ctx, "system.dictionary.create", "system_dictionary", dictionary.ID, map[string]any{"group": dictionary.Group, "key": dictionary.Key})
		return SystemDictionaryActionResponse{OK: true, Dictionary: dictionary, MessageZh: "字典项已保存"}, nil
	}
	dictionary := s.createDemoSystemDictionary(req)
	return SystemDictionaryActionResponse{OK: true, Dictionary: dictionary, MessageZh: "字典项已保存到 Demo 响应"}, nil
}

func (s *Service) SystemDictionaryAction(ctx context.Context, id, action string) (SystemDictionaryActionResponse, error) {
	enabled := action == "enable"
	if action != "enable" && action != "disable" {
		return SystemDictionaryActionResponse{}, fmt.Errorf("unsupported dictionary action: %s", action)
	}
	if store, ok := s.store.(SystemManagementStore); ok {
		dictionary, err := store.UpdateSystemDictionaryEnabled(ctx, strings.TrimSpace(id), enabled)
		if err != nil {
			return SystemDictionaryActionResponse{}, err
		}
		s.insertSystemAudit(ctx, "system.dictionary."+action, "system_dictionary", dictionary.ID, map[string]any{"enabled": dictionary.Enabled})
		return SystemDictionaryActionResponse{OK: true, Dictionary: dictionary, MessageZh: "字典项状态已更新"}, nil
	}
	return SystemDictionaryActionResponse{OK: true, Dictionary: s.updateDemoSystemDictionaryEnabled(strings.TrimSpace(id), enabled), MessageZh: "字典项状态已更新到 Demo 响应"}, nil
}

func (s *Service) CreateSystemMenu(ctx context.Context, req CreateSystemMenuRequest) (SystemMenuActionResponse, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Path = strings.TrimSpace(req.Path)
	req.Icon = strings.TrimSpace(req.Icon)
	req.Parent = strings.TrimSpace(req.Parent)
	if req.Title == "" || req.Path == "" {
		return SystemMenuActionResponse{}, fmt.Errorf("title and path are required")
	}
	if req.Sort <= 0 {
		req.Sort = 100
	}
	if store, ok := s.store.(SystemManagementStore); ok {
		menu, err := store.CreateSystemMenu(ctx, req)
		if err != nil {
			return SystemMenuActionResponse{}, err
		}
		s.insertSystemAudit(ctx, "system.menu.create", "system_menu", menu.ID, map[string]any{"path": menu.Path})
		return SystemMenuActionResponse{OK: true, Menu: menu, MessageZh: "菜单已保存"}, nil
	}
	menu := s.createDemoSystemMenu(req)
	return SystemMenuActionResponse{OK: true, Menu: menu, MessageZh: "菜单已保存到 Demo 响应"}, nil
}

func (s *Service) SystemMenuAction(ctx context.Context, id, action string) (SystemMenuActionResponse, error) {
	visible := action == "show"
	if action != "show" && action != "hide" {
		return SystemMenuActionResponse{}, fmt.Errorf("unsupported menu action: %s", action)
	}
	if store, ok := s.store.(SystemManagementStore); ok {
		menu, err := store.UpdateSystemMenuVisible(ctx, strings.TrimSpace(id), visible)
		if err != nil {
			return SystemMenuActionResponse{}, err
		}
		s.insertSystemAudit(ctx, "system.menu."+action, "system_menu", menu.ID, map[string]any{"visible": menu.Visible})
		return SystemMenuActionResponse{OK: true, Menu: menu, MessageZh: "菜单显示状态已更新"}, nil
	}
	return SystemMenuActionResponse{OK: true, Menu: s.updateDemoSystemMenuVisible(strings.TrimSpace(id), visible), MessageZh: "菜单显示状态已更新到 Demo 响应"}, nil
}

func (s *Service) insertSystemAudit(ctx context.Context, action, targetType, targetID string, metadata map[string]any) {
	if s.store != nil {
		_ = s.store.InsertAudit(ctx, action, targetType, targetID, "系统管理操作", metadata)
	}
}

func normalizeSystemStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "disabled":
		return "disabled"
	default:
		return "enabled"
	}
}

func normalizePermissionType(value string) string {
	switch strings.TrimSpace(value) {
	case "menu", "api":
		return strings.TrimSpace(value)
	default:
		return "button"
	}
}

func normalizeStringList(values []string) []string {
	seen := map[string]struct{}{}
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func safeCode(value string) string {
	return safeFilePart(value)
}

func (s *Service) demoSystemOverview() SystemManagementOverview {
	if s.systemDemo == nil {
		s.systemDemo = newSystemDemoState()
	}
	s.systemDemo.mu.Lock()
	defer s.systemDemo.mu.Unlock()
	return cloneSystemOverview(s.systemDemo.overview)
}

func (s *Service) createDemoSystemUser(req CreateSystemUserRequest) SystemUserAdmin {
	if s.systemDemo == nil {
		s.systemDemo = newSystemDemoState()
	}
	s.systemDemo.mu.Lock()
	defer s.systemDemo.mu.Unlock()
	user := demoSystemUser(req)
	s.systemDemo.overview.Users = append([]SystemUserAdmin{user}, s.systemDemo.overview.Users...)
	return user
}

func (s *Service) updateDemoSystemUserStatus(id, status string) SystemUserAdmin {
	if s.systemDemo == nil {
		s.systemDemo = newSystemDemoState()
	}
	s.systemDemo.mu.Lock()
	defer s.systemDemo.mu.Unlock()
	for index, user := range s.systemDemo.overview.Users {
		if user.ID == id {
			s.systemDemo.overview.Users[index].Status = status
			return s.systemDemo.overview.Users[index]
		}
	}
	user := demoSystemUser(CreateSystemUserRequest{Name: "Demo 用户", Account: id, Status: status})
	s.systemDemo.overview.Users = append([]SystemUserAdmin{user}, s.systemDemo.overview.Users...)
	return user
}

func (s *Service) createDemoSystemRole(req CreateSystemRoleRequest) SystemRoleAdmin {
	if s.systemDemo == nil {
		s.systemDemo = newSystemDemoState()
	}
	s.systemDemo.mu.Lock()
	defer s.systemDemo.mu.Unlock()
	role := demoSystemRole(req)
	s.systemDemo.overview.Roles = append([]SystemRoleAdmin{role}, s.systemDemo.overview.Roles...)
	return role
}

func (s *Service) updateDemoSystemRoleEnabled(id string, enabled bool) SystemRoleAdmin {
	if s.systemDemo == nil {
		s.systemDemo = newSystemDemoState()
	}
	s.systemDemo.mu.Lock()
	defer s.systemDemo.mu.Unlock()
	for index, role := range s.systemDemo.overview.Roles {
		if role.ID == id {
			s.systemDemo.overview.Roles[index].Enabled = enabled
			return s.systemDemo.overview.Roles[index]
		}
	}
	role := demoSystemRole(CreateSystemRoleRequest{Name: "Demo 角色", Code: id, Enabled: &enabled})
	s.systemDemo.overview.Roles = append([]SystemRoleAdmin{role}, s.systemDemo.overview.Roles...)
	return role
}

func (s *Service) createDemoSystemPermission(req CreateSystemPermissionRequest) SystemPermissionAdmin {
	if s.systemDemo == nil {
		s.systemDemo = newSystemDemoState()
	}
	s.systemDemo.mu.Lock()
	defer s.systemDemo.mu.Unlock()
	permission := demoSystemPermission(req)
	s.systemDemo.overview.Permissions = append([]SystemPermissionAdmin{permission}, s.systemDemo.overview.Permissions...)
	return permission
}

func (s *Service) updateDemoSystemPermissionEnabled(id string, enabled bool) SystemPermissionAdmin {
	if s.systemDemo == nil {
		s.systemDemo = newSystemDemoState()
	}
	s.systemDemo.mu.Lock()
	defer s.systemDemo.mu.Unlock()
	for index, permission := range s.systemDemo.overview.Permissions {
		if permission.ID == id {
			s.systemDemo.overview.Permissions[index].Enabled = enabled
			return s.systemDemo.overview.Permissions[index]
		}
	}
	permission := demoSystemPermission(CreateSystemPermissionRequest{Name: "Demo 权限", Code: id, Enabled: &enabled})
	s.systemDemo.overview.Permissions = append([]SystemPermissionAdmin{permission}, s.systemDemo.overview.Permissions...)
	return permission
}

func (s *Service) createDemoSystemDictionary(req CreateSystemDictionaryRequest) SystemDictionaryAdmin {
	if s.systemDemo == nil {
		s.systemDemo = newSystemDemoState()
	}
	s.systemDemo.mu.Lock()
	defer s.systemDemo.mu.Unlock()
	dictionary := demoSystemDictionary(req)
	s.systemDemo.overview.Dictionaries = append([]SystemDictionaryAdmin{dictionary}, s.systemDemo.overview.Dictionaries...)
	return dictionary
}

func (s *Service) updateDemoSystemDictionaryEnabled(id string, enabled bool) SystemDictionaryAdmin {
	if s.systemDemo == nil {
		s.systemDemo = newSystemDemoState()
	}
	s.systemDemo.mu.Lock()
	defer s.systemDemo.mu.Unlock()
	for index, dictionary := range s.systemDemo.overview.Dictionaries {
		if dictionary.ID == id {
			s.systemDemo.overview.Dictionaries[index].Enabled = enabled
			return s.systemDemo.overview.Dictionaries[index]
		}
	}
	dictionary := demoSystemDictionary(CreateSystemDictionaryRequest{Group: "demo", Key: id, Label: "Demo 字典", Enabled: &enabled})
	s.systemDemo.overview.Dictionaries = append([]SystemDictionaryAdmin{dictionary}, s.systemDemo.overview.Dictionaries...)
	return dictionary
}

func (s *Service) createDemoSystemMenu(req CreateSystemMenuRequest) SystemMenuAdmin {
	if s.systemDemo == nil {
		s.systemDemo = newSystemDemoState()
	}
	s.systemDemo.mu.Lock()
	defer s.systemDemo.mu.Unlock()
	menu := demoSystemMenu(req)
	s.systemDemo.overview.Menus = append([]SystemMenuAdmin{menu}, s.systemDemo.overview.Menus...)
	return menu
}

func (s *Service) updateDemoSystemMenuVisible(id string, visible bool) SystemMenuAdmin {
	if s.systemDemo == nil {
		s.systemDemo = newSystemDemoState()
	}
	s.systemDemo.mu.Lock()
	defer s.systemDemo.mu.Unlock()
	for index, menu := range s.systemDemo.overview.Menus {
		if menu.ID == id {
			s.systemDemo.overview.Menus[index].Visible = visible
			return s.systemDemo.overview.Menus[index]
		}
	}
	menu := demoSystemMenu(CreateSystemMenuRequest{Title: "Demo 菜单", Path: "/" + safeCode(id), Visible: &visible})
	s.systemDemo.overview.Menus = append([]SystemMenuAdmin{menu}, s.systemDemo.overview.Menus...)
	return menu
}

func cloneSystemOverview(overview SystemManagementOverview) SystemManagementOverview {
	overview.Users = append([]SystemUserAdmin(nil), overview.Users...)
	overview.Roles = append([]SystemRoleAdmin(nil), overview.Roles...)
	overview.Permissions = append([]SystemPermissionAdmin(nil), overview.Permissions...)
	overview.Dictionaries = append([]SystemDictionaryAdmin(nil), overview.Dictionaries...)
	overview.Menus = append([]SystemMenuAdmin(nil), overview.Menus...)
	for index, role := range overview.Roles {
		overview.Roles[index].Permissions = append([]string(nil), role.Permissions...)
	}
	return overview
}

func demoSystemManagementOverview() SystemManagementOverview {
	return SystemManagementOverview{
		Users: []SystemUserAdmin{
			{ID: "u_demo_000", Name: "系统管理员", Account: "system.admin", Role: "系统管理员", RoleCode: "system_admin", Department: "平台工程", Status: "enabled", LastLogin: "2026-06-02 15:30"},
			{ID: "u_demo_001", Name: "发布管理员", Account: "release.admin", Role: "发版管理员", RoleCode: "release_admin", Department: "平台工程", Status: "enabled", LastLogin: "2026-06-02 13:46"},
			{ID: "u_demo_002", Name: "测试负责人", Account: "qa.lead", Role: "发版管理员", RoleCode: "release_admin", Department: "质量保障", Status: "enabled", LastLogin: "2026-06-01 18:21"},
			{ID: "u_demo_003", Name: "观察员", Account: "release.viewer", Role: "只读观察员", RoleCode: "release_viewer", Department: "运营支持", Status: "disabled", LastLogin: "2026-05-28 09:12"},
		},
		Roles: []SystemRoleAdmin{
			{ID: "r_demo_001", Name: "系统管理员", Code: "system_admin", Users: 1, Scope: "全部数据", Permissions: []string{"system:*", "release:*"}, Enabled: true},
			{ID: "r_demo_002", Name: "发版管理员", Code: "release_admin", Users: 2, Scope: "发版数据", Permissions: []string{"release:write", "release:audit"}, Enabled: true},
			{ID: "r_demo_003", Name: "只读观察员", Code: "release_viewer", Users: 1, Scope: "只读数据", Permissions: []string{"release:read"}, Enabled: true},
		},
		Permissions: []SystemPermissionAdmin{
			{ID: "p_demo_001", Name: "查看首页", Code: "dashboard:view", Module: "工作台", Type: "menu", Enabled: true},
			{ID: "p_demo_002", Name: "查看发布", Code: "release:read", Module: "发布中心", Type: "api", Enabled: true},
			{ID: "p_demo_003", Name: "管理发布", Code: "release:write", Module: "发布中心", Type: "button", Enabled: true},
			{ID: "p_demo_004", Name: "查看审计", Code: "release:audit", Module: "发布中心", Type: "api", Enabled: true},
			{ID: "p_demo_005", Name: "查看系统管理", Code: "system:read", Module: "系统管理", Type: "api", Enabled: true},
			{ID: "p_demo_006", Name: "维护系统管理", Code: "system:write", Module: "系统管理", Type: "api", Enabled: true},
			{ID: "p_demo_007", Name: "维护用户", Code: "system:user:write", Module: "系统管理", Type: "button", Enabled: true},
			{ID: "p_demo_008", Name: "维护菜单", Code: "system:menu:write", Module: "系统管理", Type: "button", Enabled: true},
		},
		Dictionaries: []SystemDictionaryAdmin{
			{ID: "d_demo_001", Group: "release_channel", Key: "stable", Label: "正式渠道", Value: "stable", Sort: 10, Enabled: true},
			{ID: "d_demo_002", Group: "release_channel", Key: "beta", Label: "Beta 渠道", Value: "beta", Sort: 20, Enabled: true},
			{ID: "d_demo_003", Group: "resource_package", Key: "templates-bear", Label: "打熊识图模板", Value: "templates-bear", Sort: 30, Enabled: true},
			{ID: "d_demo_004", Group: "audit_level", Key: "critical", Label: "关键操作", Value: "critical", Sort: 40, Enabled: true},
		},
		Menus: []SystemMenuAdmin{
			{ID: "m_demo_001", Title: "首页", Path: "/dashboard", Icon: "LayoutDashboard", Parent: "", Sort: 10, Visible: true},
			{ID: "m_demo_002", Title: "发布中心", Path: "/release-center", Icon: "Rocket", Parent: "工作台", Sort: 20, Visible: true},
			{ID: "m_demo_003", Title: "用户管理", Path: "/system/users", Icon: "Users", Parent: "系统管理", Sort: 30, Visible: true},
			{ID: "m_demo_004", Title: "角色管理", Path: "/system/roles", Icon: "Shield", Parent: "系统管理", Sort: 40, Visible: true},
			{ID: "m_demo_005", Title: "权限管理", Path: "/system/permissions", Icon: "KeyRound", Parent: "系统管理", Sort: 45, Visible: true},
			{ID: "m_demo_006", Title: "数据字典", Path: "/system/dictionaries", Icon: "Database", Parent: "系统管理", Sort: 50, Visible: true},
			{ID: "m_demo_007", Title: "菜单编辑", Path: "/system/menus", Icon: "ListTree", Parent: "系统管理", Sort: 60, Visible: true},
		},
	}
}

func demoSystemUser(req CreateSystemUserRequest) SystemUserAdmin {
	return SystemUserAdmin{ID: "u_demo_" + uuid.NewString(), Name: req.Name, Account: req.Account, Role: firstNonBlank(req.Role, req.RoleCode, "只读观察员"), RoleCode: req.RoleCode, Department: req.Department, Status: normalizeSystemStatus(req.Status), LastLogin: "-"}
}

func demoSystemRole(req CreateSystemRoleRequest) SystemRoleAdmin {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	permissions := normalizeStringList(req.Permissions)
	if permissions == nil {
		permissions = []string{}
	}
	return SystemRoleAdmin{ID: "r_demo_" + uuid.NewString(), Name: req.Name, Code: req.Code, Scope: req.Scope, Permissions: permissions, Enabled: enabled}
}

func demoSystemPermission(req CreateSystemPermissionRequest) SystemPermissionAdmin {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	return SystemPermissionAdmin{ID: "p_demo_" + uuid.NewString(), Name: req.Name, Code: req.Code, Module: req.Module, Type: normalizePermissionType(req.Type), Enabled: enabled}
}

func demoSystemDictionary(req CreateSystemDictionaryRequest) SystemDictionaryAdmin {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	return SystemDictionaryAdmin{ID: "d_demo_" + uuid.NewString(), Group: req.Group, Key: req.Key, Label: req.Label, Value: req.Value, Sort: req.Sort, Enabled: enabled}
}

func demoSystemMenu(req CreateSystemMenuRequest) SystemMenuAdmin {
	visible := true
	if req.Visible != nil {
		visible = *req.Visible
	}
	return SystemMenuAdmin{ID: "m_demo_" + uuid.NewString(), Title: req.Title, Path: req.Path, Icon: req.Icon, Parent: req.Parent, Sort: req.Sort, Visible: visible}
}
